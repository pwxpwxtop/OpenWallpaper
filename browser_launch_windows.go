//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// freePort 返回一个当前可用的本机端口。
func freePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// debuggerURLFromPort 通过 /json/version 获取浏览器调试地址。
func debuggerURLFromPort(port int) string {
	if port <= 0 {
		return ""
	}
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if json.NewDecoder(resp.Body).Decode(&payload) != nil {
		return ""
	}
	return payload.WebSocketDebuggerURL
}

// debuggerURLFromFile 读取浏览器写在配置目录里的调试地址（复用已运行的实例）。
func debuggerURLFromFile(profile string) string {
	data, err := os.ReadFile(filepath.Join(profile, "DevToolsActivePort"))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return ""
	}
	port := strings.TrimSpace(lines[0])
	path := strings.TrimSpace(lines[1])
	if port == "" || path == "" {
		return ""
	}
	return fmt.Sprintf("ws://127.0.0.1:%s%s", port, path)
}

func connectDebugger(profile string, port int) (*websocket.Conn, context.Context, context.CancelFunc, bool) {
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		urls := make([]string, 0, 2)
		if url := debuggerURLFromPort(port); url != "" {
			urls = append(urls, url)
		}
		if url := debuggerURLFromFile(profile); url != "" {
			urls = append(urls, url)
		}
		for _, url := range urls {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			conn, _, err := websocket.Dial(ctx, url, nil)
			if err == nil {
				return conn, ctx, cancel, true
			}
			cancel()
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, nil, nil, false
}

// cdpRequest 发送一条 CDP 命令并等待同 id 的响应，返回结果与错误信息。
func cdpRequest(ctx context.Context, conn *websocket.Conn, id int, method string, params any, sessionID string) (json.RawMessage, string) {
	message := map[string]any{
		"id":     id,
		"method": method,
	}
	if params != nil {
		message["params"] = params
	}
	if sessionID != "" {
		message["sessionId"] = sessionID
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return nil, err.Error()
	}
	if err := conn.Write(ctx, websocket.MessageText, payload); err != nil {
		return nil, err.Error()
	}
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil, err.Error()
		}
		var reply struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(data, &reply) != nil {
			continue
		}
		if reply.ID != id {
			continue
		}
		if reply.Error != nil {
			return nil, reply.Error.Message
		}
		return reply.Result, ""
	}
}

// cdpCall 是 cdpRequest 的简化版本，仅返回错误信息。
func cdpCall(ctx context.Context, conn *websocket.Conn, id int, method string, params any) string {
	_, errText := cdpRequest(ctx, conn, id, method, params, "")
	return errText
}

// maximizeWindow 把指定标签页所在的浏览器窗口最大化。
func maximizeWindow(ctx context.Context, conn *websocket.Conn, targetID string, idBase int) {
	result, errText := cdpRequest(ctx, conn, idBase, "Browser.getWindowForTarget", map[string]any{
		"targetId": targetID,
	}, "")
	if errText != "" {
		return
	}
	var info struct {
		WindowID int `json:"windowId"`
	}
	if json.Unmarshal(result, &info) != nil {
		return
	}
	cdpCall(ctx, conn, idBase+1, "Browser.setWindowBounds", map[string]any{
		"windowId": info.WindowID,
		"bounds":   map[string]any{"windowState": "maximized"},
	})
}

// openNewTab 在已有浏览器窗口里新开一个标签页并最大化窗口。
func openNewTab(ctx context.Context, conn *websocket.Conn, startURL string) bool {
	result, errText := cdpRequest(ctx, conn, 20, "Target.createTarget", map[string]any{
		"url": startURL,
	}, "")
	if errText != "" {
		return false
	}
	var created struct {
		TargetID string `json:"targetId"`
	}
	if json.Unmarshal(result, &created) != nil || created.TargetID == "" {
		return false
	}
	cdpCall(ctx, conn, 21, "Target.activateTarget", map[string]any{"targetId": created.TargetID})
	maximizeWindow(ctx, conn, created.TargetID, 22)
	return true
}

// navigateExisting 复用浏览器里已有的标签页导航到目标网址（仅用于首次创建时清理空白页）。
func navigateExisting(ctx context.Context, conn *websocket.Conn, startURL string) bool {
	if startURL == "" {
		return false
	}
	result, errText := cdpRequest(ctx, conn, 10, "Target.getTargets", nil, "")
	if errText != "" {
		return false
	}
	var targets struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	if json.Unmarshal(result, &targets) != nil {
		return false
	}

	targetID := ""
	for _, target := range targets.TargetInfos {
		if target.Type == "page" {
			targetID = target.TargetID
			break
		}
	}
	if targetID == "" {
		return false
	}

	attachResult, errText := cdpRequest(ctx, conn, 11, "Target.attachToTarget", map[string]any{
		"targetId": targetID,
		"flatten":  true,
	}, "")
	if errText != "" {
		return false
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal(attachResult, &attached) != nil || attached.SessionID == "" {
		return false
	}

	if _, errText = cdpRequest(ctx, conn, 12, "Page.navigate", map[string]any{
		"url": startURL,
	}, attached.SessionID); errText != "" {
		return false
	}
	cdpCall(ctx, conn, 13, "Target.activateTarget", map[string]any{"targetId": targetID})
	maximizeWindow(ctx, conn, targetID, 14)
	return true
}

// browserPortFile 记录当前独立浏览器实例的调试端口，便于下次复用它。
func browserPortFile() string {
	base := os.Getenv("AppData")
	if base == "" {
		return ""
	}
	return filepath.Join(base, "OpenWallpaper", "browser-port")
}

func saveBrowserPort(port int) {
	file := browserPortFile()
	if file == "" || port <= 0 {
		return
	}
	_ = os.MkdirAll(filepath.Dir(file), 0o755)
	_ = os.WriteFile(file, []byte(strconv.Itoa(port)), 0o644)
}

func loadBrowserPort() int {
	file := browserPortFile()
	if file == "" {
		return 0
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return port
}

// saveBrowserName 记录当前独立浏览器使用的内核名称。
func saveBrowserName(name string) {
	file := filepath.Join(filepath.Dir(browserPortFile()), "browser-name")
	if browserPortFile() == "" || name == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(file), 0o755)
	_ = os.WriteFile(file, []byte(name), 0o644)
}

func loadBrowserName() string {
	if browserPortFile() == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(browserPortFile()), "browser-name"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// portFromDevToolsFile 从浏览器的 DevToolsActivePort 文件读取实际调试端口。
func portFromDevToolsFile(profile string) int {
	data, err := os.ReadFile(filepath.Join(profile, "DevToolsActivePort"))
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0
	}
	return port
}

// connectRunningBrowser 尝试连接已经在运行的独立浏览器。
func connectRunningBrowser(profile string) (*websocket.Conn, context.Context, context.CancelFunc, bool) {
	urls := make([]string, 0, 2)
	if port := loadBrowserPort(); port > 0 {
		if url := debuggerURLFromPort(port); url != "" {
			urls = append(urls, url)
		}
	}
	if url := debuggerURLFromFile(profile); url != "" {
		urls = append(urls, url)
	}
	for _, url := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		conn, _, err := websocket.Dial(ctx, url, nil)
		if err == nil {
			return conn, ctx, cancel, true
		}
		cancel()
	}
	return nil, nil, nil, false
}

// reuseRunningBrowser 复用已运行的浏览器，并在新标签页打开目标网址。
func reuseRunningBrowser(profile, startURL string) bool {
	conn, ctx, cancel, ok := connectRunningBrowser(profile)
	if !ok {
		return false
	}
	defer cancel()
	defer conn.Close(websocket.StatusNormalClosure, "")
	return openNewTab(ctx, conn, startURL)
}

// loadExtensionOverCDP 通过 Chrome DevTools 协议加载未打包插件，并打开目标网页。
func loadExtensionOverCDP(profile string, port int, extensionDir, startURL string) (bool, string) {
	conn, ctx, cancel, ok := connectDebugger(profile, port)
	if !ok {
		return false, "debugger 连接失败"
	}
	defer cancel()
	defer conn.Close(websocket.StatusNormalClosure, "")

	errText := cdpCall(ctx, conn, 1, "Extensions.loadUnpacked", map[string]any{
		"path": extensionDir,
	})
	loaded := errText == "" || strings.Contains(strings.ToLower(errText), "already")
	detail := errText
	if detail == "" {
		detail = "ok"
	}

	if startURL != "" {
		// 首次创建时复用启动时的空白标签页导航，避免多开标签；没有可用标签时再新建。
		if !navigateExisting(ctx, conn, startURL) {
			openNewTab(ctx, conn, startURL)
		}
	}
	return loaded, detail
}

// appendBrowserLog 把浏览器启动 / 插件加载结果写入日志，便于排查问题。
func appendBrowserLog(line string) {
	base := os.Getenv("AppData")
	if base == "" {
		return
	}
	file := filepath.Join(base, "OpenWallpaper", "browser.log")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return
	}
	handle, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer handle.Close()
	_, _ = fmt.Fprintf(handle, "%s %s\n", time.Now().Format(time.RFC3339), line)
}

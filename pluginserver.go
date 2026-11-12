package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// 插件使用的本地 WebSocket 端口。
const pluginPort = 48921

//go:embed all:plugin
var pluginAssets embed.FS

// pluginHeader 是插件通过文本帧发送的文件头。
type pluginHeader struct {
	Type string `json:"type"` // file | eof | abort | ping
	ID   string `json:"id"`
	Name string `json:"name"`
	Dir  string `json:"dir"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

// pluginReply 是服务端回给插件的文本帧。
type pluginReply struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	OK          bool   `json:"ok"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Error       string `json:"error"`
	Port        int    `json:"port"`
	DownloadDir string `json:"downloadDir"`
}

type pluginUpload struct {
	file    *os.File
	tmpPath string
	final   string
	id      string
	written int64
}

// startPluginServer 启动仅监听本机回环地址的 WebSocket 服务，供浏览器插件推送媒体文件。
func (s *WallpaperService) startPluginServer() {
	if _, err := extractPlugin(); err != nil {
		log.Printf("插件文件导出失败: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.servePluginConn)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", pluginPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("插件服务启动失败: %v", err)
		}
	}()
}

func (s *WallpaperService) servePluginConn(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionDisabled,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(4 << 30)

	ctx := r.Context()

	_ = writePluginMessage(ctx, conn, pluginReply{
		Type:        "hello",
		OK:          true,
		Port:        pluginPort,
		DownloadDir: s.downloadDir(),
	})

	var upload *pluginUpload
	defer func() {
		if upload != nil {
			upload.file.Close()
			os.Remove(upload.tmpPath)
		}
	}()

	for {
		kind, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		if kind == websocket.MessageBinary {
			if upload == nil {
				continue
			}
			if _, err := upload.file.Write(data); err != nil {
				_ = writePluginMessage(ctx, conn, pluginReply{Type: "error", ID: upload.id, Error: err.Error()})
				upload.file.Close()
				os.Remove(upload.tmpPath)
				upload = nil
				continue
			}
			upload.written += int64(len(data))
			continue
		}

		var msg pluginHeader
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "ping":
			_ = writePluginMessage(ctx, conn, pluginReply{
				Type:        "pong",
				ID:          msg.ID,
				OK:          true,
				Port:        pluginPort,
				DownloadDir: s.downloadDir(),
			})

		case "abort":
			if upload != nil && upload.id == msg.ID {
				upload.file.Close()
				os.Remove(upload.tmpPath)
				upload = nil
			}

		case "file":
			if upload != nil {
				upload.file.Close()
				os.Remove(upload.tmpPath)
				upload = nil
			}
			final, err := s.allocateDownloadPath(msg.Dir, msg.Name)
			if err != nil {
				_ = writePluginMessage(ctx, conn, pluginReply{Type: "error", ID: msg.ID, Error: err.Error()})
				continue
			}
			tmp := final + ".part"
			file, err := os.Create(tmp)
			if err != nil {
				_ = writePluginMessage(ctx, conn, pluginReply{Type: "error", ID: msg.ID, Error: err.Error()})
				continue
			}
			upload = &pluginUpload{file: file, tmpPath: tmp, final: final, id: msg.ID}
			_ = writePluginMessage(ctx, conn, pluginReply{
				Type: "start",
				ID:   msg.ID,
				OK:   true,
				Name: filepath.Base(final),
				Path: final,
			})

		case "eof":
			if upload == nil || upload.id != msg.ID {
				_ = writePluginMessage(ctx, conn, pluginReply{
					Type:  "error",
					ID:    msg.ID,
					Error: "没有待写入的文件",
				})
				continue
			}
			done := upload
			upload = nil
			if err := done.file.Close(); err != nil {
				os.Remove(done.tmpPath)
				_ = writePluginMessage(ctx, conn, pluginReply{Type: "error", ID: done.id, Error: err.Error()})
				continue
			}
			if err := os.Rename(done.tmpPath, done.final); err != nil {
				os.Remove(done.tmpPath)
				_ = writePluginMessage(ctx, conn, pluginReply{Type: "error", ID: done.id, Error: err.Error()})
				continue
			}
			_ = writePluginMessage(ctx, conn, pluginReply{
				Type: "saved",
				ID:   done.id,
				OK:   true,
				Path: done.final,
				Name: filepath.Base(done.final),
			})
		}
	}
}

func writePluginMessage(ctx context.Context, conn *websocket.Conn, reply pluginReply) error {
	payload, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, payload)
}

// allocateDownloadPath 把插件传来的目录与文件名清洗成下载目录内的安全路径。
func (s *WallpaperService) allocateDownloadPath(dir, name string) (string, error) {
	root := s.downloadDir()
	segments := []string{root}
	for _, part := range strings.FieldsFunc(dir, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == "" || part == "." || part == ".." {
			continue
		}
		segments = append(segments, sanitizeSegment(part))
	}
	dirPath := filepath.Join(segments...)

	base := sanitizeSegment(name)
	if base == "" || base == "file" {
		base = "media-" + time.Now().Format("20060102-150405")
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	target := filepath.Join(dirPath, base)
	for i := 1; i <= 500; i++ {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			break
		}
		target = filepath.Join(dirPath, fmt.Sprintf("%s-%d%s", stem, i, ext))
	}

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return "", err
	}
	return target, nil
}

// sanitizeSegment 去掉文件名中的非法字符，避免路径穿越与 Windows 保留名。
func sanitizeSegment(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '-'
		}
		if r < 0x20 {
			return -1
		}
		return r
	}, name)
	name = strings.Trim(name, ". ")
	name = strings.TrimLeft(name, "-")
	if name == "" || name == "file" {
		return "file"
	}
	runes := []rune(name)
	if len(runes) > 120 {
		name = string(runes[:120])
	}
	if isReservedName(name) {
		name = "_" + name
	}
	return name
}

func isReservedName(name string) bool {
	base := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	}
	return false
}

// downloadDir 返回当前生效的下载 / 缓存目录。
func (s *WallpaperService) downloadDir() string {
	s.mu.RLock()
	dir := s.downloadDirPath
	s.mu.RUnlock()
	if dir != "" {
		return dir
	}
	return defaultDownloadDir()
}

func defaultDownloadDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Pictures", "OpenWallpaper")
	}
	return filepath.Join(os.TempDir(), "OpenWallpaper")
}

// ensureDownloadFolder 创建下载目录并把它加入壁纸库，下载的图片会直接出现在壁纸库中。
func (s *WallpaperService) ensureDownloadFolder() {
	dir := s.downloadDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	clean := filepath.Clean(dir)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, folder := range s.folders {
		if strings.EqualFold(folder, clean) {
			return
		}
	}
	s.folders = append(s.folders, clean)
	_ = s.save()
}

// pluginDir 返回插件被导出到的本地目录。
func pluginDir() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "OpenWallpaper", "plugin"), nil
}

// extractPlugin 把内嵌的插件文件写入本地目录，随应用安装包一起分发。
func extractPlugin() (string, error) {
	dir, err := pluginDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	err = fs.WalkDir(pluginAssets, "plugin", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel("plugin", path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := pluginAssets.ReadFile(path)
		if err != nil {
			return err
		}
		// 浏览器可能正在使用已导出的插件文件，内容一致时跳过写入，避免访问被拒绝。
		if existing, readErr := os.ReadFile(target); readErr == nil && bytes.Equal(existing, data) {
			return nil
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// OpenDownloadFolder 在文件管理器中打开下载目录。
func (s *WallpaperService) OpenDownloadFolder() error {
	dir := s.downloadDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return openInExplorer(dir)
}

// OpenPluginFolder 导出插件文件并在文件管理器中打开。
func (s *WallpaperService) OpenPluginFolder() error {
	dir, err := extractPlugin()
	if err != nil {
		return err
	}
	return openInExplorer(dir)
}

// ExtensionInstall 描述插件导出与扩展管理页的打开结果。
type ExtensionInstall struct {
	Dir     string `json:"dir"`
	Browser string `json:"browser"`
	Opened  bool   `json:"opened"`
}

// OpenURL 用系统默认浏览器打开网页。
func (s *WallpaperService) OpenURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("无效的网址")
	}
	return openInBrowser(parsed.String())
}

// InstallExtension 导出插件并尝试打开 Chrome / Edge 的扩展管理页。
func (s *WallpaperService) InstallExtension() (ExtensionInstall, error) {
	dir, err := extractPlugin()
	if err != nil {
		return ExtensionInstall{}, err
	}
	browser, opened := openExtensionManager()
	return ExtensionInstall{Dir: dir, Browser: browser, Opened: opened}, nil
}

// BrowserLaunch 描述带插件启动浏览器的结果。
type BrowserLaunch struct {
	Browser string `json:"browser"`
	Dir     string `json:"dir"`
	Profile string `json:"profile"`
	Opened  bool   `json:"opened"`
	Loaded  bool   `json:"loaded"`
}

// LaunchBrowserWithExtension 导出插件，并用独立配置启动浏览器自动加载插件。
func (s *WallpaperService) LaunchBrowserWithExtension(startURL string) (BrowserLaunch, error) {
	dir, err := extractPlugin()
	if err != nil {
		return BrowserLaunch{}, err
	}
	page := strings.TrimSpace(startURL)
	if parsed, parseErr := url.Parse(page); parseErr != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		page = ""
	}
	browser, profile, opened, loaded := launchBrowserWithExtension(dir, page)
	return BrowserLaunch{
		Browser: browser,
		Dir:     dir,
		Profile: profile,
		Opened:  opened,
		Loaded:  loaded,
	}, nil
}

// SelectDownloadDir 打开目录选择框并更新下载 / 缓存目录。
func (s *WallpaperService) SelectDownloadDir() (Settings, error) {
	app := application.Get()
	if app == nil {
		return Settings{}, errors.New("应用尚未初始化")
	}
	path, err := app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("选择缓存目录").
		PromptForSingleSelection()
	if err != nil {
		return Settings{}, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.settings(), nil
	}

	s.mu.Lock()
	s.downloadDirPath = filepath.Clean(path)
	err = s.save()
	s.mu.Unlock()
	if err != nil {
		return Settings{}, err
	}
	// 新的缓存目录自动加入壁纸库。
	s.ensureDownloadFolder()

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings(), nil
}

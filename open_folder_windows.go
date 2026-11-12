//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// openInExplorer 用资源管理器打开指定目录。
func openInExplorer(path string) error {
	return exec.Command("explorer.exe", path).Start()
}

// openInBrowser 用系统默认浏览器打开网址。
func openInBrowser(rawURL string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
}

type browserCandidate struct {
	name string
	path string
	page string
}

func envPath(base string, parts ...string) string {
	if base == "" {
		return ""
	}
	return filepath.Join(append([]string{base}, parts...)...)
}

func browserCandidates() []browserCandidate {
	return []browserCandidate{
		{"Edge", envPath(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"), "edge://extensions"},
		{"Edge", envPath(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"), "edge://extensions"},
		{"Edge", envPath(os.Getenv("LocalAppData"), "Microsoft", "Edge", "Application", "msedge.exe"), "edge://extensions"},
		{"Chrome", envPath(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"), "chrome://extensions"},
		{"Chrome", envPath(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"), "chrome://extensions"},
		{"Chrome", envPath(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"), "chrome://extensions"},
	}
}

// openExtensionManager 尝试打开 Chrome / Edge 的扩展管理页，返回浏览器名称与是否成功。
func openExtensionManager() (string, bool) {
	for _, candidate := range browserCandidates() {
		if candidate.path == "" {
			continue
		}
		info, err := os.Stat(candidate.path)
		if err != nil || info.IsDir() {
			continue
		}
		if err := exec.Command(candidate.path, candidate.page).Start(); err == nil {
			return candidate.name, true
		}
	}
	return "", false
}

// launchBrowserWithExtension 用独立配置目录启动浏览器，并通过 CDP 自动加载未打包插件。
// 优先使用应用下载的便携版 Chromium；不可用时退回本机 Chrome / Edge。
func launchBrowserWithExtension(extensionDir, startURL string) (string, string, bool, bool) {
	profile := envPath(os.Getenv("AppData"), "OpenWallpaper", "app-browser")
	if profile == "" {
		return "", "", false, false
	}
	if err := os.MkdirAll(profile, 0o755); err != nil {
		return "", "", false, false
	}

	// 只复用便携版 Chromium 创建的浏览器；其它内核可能没有装插件。
	if loadBrowserName() == "Chromium" && reuseRunningBrowser(profile, startURL) {
		return "Chromium", profile, true, true
	}

	// 只有便携版 Chromium（Chrome for Testing）支持程序化加载未打包插件。
	// 已下载过则直接复用，不再重复安装。
	if exe, err := ensureChromium(); err == nil {
		if launched, loaded := tryLaunchBrowser(exe, profile, extensionDir, startURL); launched && loaded {
			saveBrowserName("Chromium")
			return "Chromium", profile, true, true
		}
	}

	// 便携版不可用时，退回本机 Chrome / Edge（可能无法自动安装插件）。
	all := browserCandidates()
	for _, candidate := range all {
		if candidate.path == "" {
			continue
		}
		info, err := os.Stat(candidate.path)
		if err != nil || info.IsDir() {
			continue
		}
		if launched, loaded := tryLaunchBrowser(candidate.path, profile, extensionDir, startURL); launched && loaded {
			saveBrowserName(candidate.name)
			return candidate.name, profile, true, true
		}
	}
	return "", "", false, false
}

// tryLaunchBrowser 启动指定浏览器并尝试通过 CDP 加载插件。
func tryLaunchBrowser(exePath, profile, extensionDir, startURL string) (bool, bool) {
	port := freePort()
	args := []string{
		"--user-data-dir=" + profile,
		"--no-first-run",
		"--no-default-browser-check",
		"--start-maximized",
		"--disable-extensions-except=" + extensionDir,
		"--load-extension=" + extensionDir,
		"--enable-unsafe-extension-debugging",
	}
	if port > 0 {
		args = append(args, fmt.Sprintf("--remote-debugging-port=%d", port))
	} else {
		args = append(args, "--remote-debugging-port=0")
	}
	cmd := exec.Command(exePath, args...)
	if err := cmd.Start(); err != nil {
		return false, false
	}

	// 加载插件的过程放到后台并加超时保护，避免界面一直等待。
	type loadResult struct {
		loaded bool
		detail string
	}
	result := make(chan loadResult, 1)
	go func() {
		loaded, detail := loadExtensionOverCDP(profile, port, extensionDir, startURL)
		result <- loadResult{loaded: loaded, detail: detail}
	}()

	loaded := false
	detail := ""
	select {
	case state := <-result:
		loaded = state.loaded
		detail = state.detail
	case <-time.After(20 * time.Second):
		loaded = false
		detail = "加载超时"
	}
	appendBrowserLog(fmt.Sprintf("exe=%s port=%d loaded=%v detail=%s", exePath, port, loaded, detail))
	if loaded {
		// 记录调试端口，下次点击推荐网站时直接复用这个浏览器实例。
		actualPort := port
		if actualPort <= 0 {
			actualPort = portFromDevToolsFile(profile)
		}
		saveBrowserPort(actualPort)
		return true, true
	}

	// 该浏览器没有成功加载插件，关闭它再尝试下一个内核。
	_ = cmd.Process.Kill()
	time.Sleep(400 * time.Millisecond)
	return true, false
}

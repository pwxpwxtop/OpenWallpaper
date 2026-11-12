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
	profile := envPath(os.Getenv("AppData"), "OpenWallpaper", "browser-session")
	if profile == "" {
		return "", "", false, false
	}
	if err := os.MkdirAll(profile, 0o755); err != nil {
		return "", "", false, false
	}

	// 已有独立浏览器在运行时直接复用，导航到目标网址，不重复创建。
	if reuseRunningBrowser(profile, startURL) {
		name := loadBrowserName()
		if name == "" {
			name = "Chromium"
		}
		return name, profile, true, true
	}

	// 优先使用本机已安装的 Chrome 内核；已安装就不再下载便携版。
	all := browserCandidates()
	chromeInstalled := false
	for _, candidate := range all {
		if candidate.name != "Chrome" {
			continue
		}
		if candidate.path == "" {
			continue
		}
		info, err := os.Stat(candidate.path)
		if err != nil || info.IsDir() {
			continue
		}
		chromeInstalled = true
		if launched, loaded := tryLaunchBrowser(candidate.path, profile, extensionDir, startURL); launched && loaded {
			saveBrowserName(candidate.name)
			return candidate.name, profile, true, true
		}
	}

	// 没有检测到 Chrome 时才下载便携版 Chromium。
	if !chromeInstalled {
		if exe, err := ensureChromium(); err == nil {
			if launched, loaded := tryLaunchBrowser(exe, profile, extensionDir, startURL); launched && loaded {
				saveBrowserName("Chromium")
				return "Chromium", profile, true, true
			}
		}
	}

	// 最后尝试 Edge 等其它 Chromium 浏览器。
	for _, candidate := range all {
		if candidate.name == "Chrome" {
			continue
		}
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
	result := make(chan bool, 1)
	go func() {
		result <- loadExtensionOverCDP(profile, port, extensionDir, startURL)
	}()

	loaded := false
	select {
	case loaded = <-result:
	case <-time.After(20 * time.Second):
		loaded = false
	}
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

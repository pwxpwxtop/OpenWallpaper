//go:build !windows

package main

import (
	"os/exec"
	"runtime"
)

// openInExplorer 用系统文件管理器打开指定目录。
func openInExplorer(path string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	return exec.Command(command, path).Start()
}

// openInBrowser 用系统默认浏览器打开网址。
func openInBrowser(rawURL string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	return exec.Command(command, rawURL).Start()
}

// openExtensionManager 在非 Windows 平台不做自动处理。
func openExtensionManager() (string, bool) {
	return "", false
}

// launchBrowserWithExtension 在非 Windows 平台不做自动处理。
func launchBrowserWithExtension(extensionDir, startURL string) (string, string, bool, bool) {
	return "", "", false, false
}

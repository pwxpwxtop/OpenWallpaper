//go:build !windows

package main

import "errors"

// ensureChromium 在非 Windows 平台不下载便携版内核。
func ensureChromium() (string, error) {
	return "", errors.New("当前平台不支持便携版 Chromium")
}

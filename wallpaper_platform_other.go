//go:build !windows

package main

import "errors"

var errPlatformUnsupported = errors.New("当前系统暂不支持设置桌面壁纸")

func applyImageWallpaper(string) error {
	return errPlatformUnsupported
}

func applyDynamicWallpaper(Wallpaper) error {
	return errPlatformUnsupported
}

func stopDynamicWallpaper() error {
	return nil
}

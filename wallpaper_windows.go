//go:build windows

package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const dynamicWindowName = "openwallpaper-layer"

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
	procFindWindowW           = user32.NewProc("FindWindowW")
	procFindWindowExW         = user32.NewProc("FindWindowExW")
	procSendMessageTimeoutW   = user32.NewProc("SendMessageTimeoutW")
	procEnumWindows           = user32.NewProc("EnumWindows")
	procSetParent             = user32.NewProc("SetParent")
	procGetWindowLongPtrW     = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW     = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos          = user32.NewProc("SetWindowPos")
	procShowWindow            = user32.NewProc("ShowWindow")
	procGetSystemMetrics      = user32.NewProc("GetSystemMetrics")
)

const (
	spiSetDesktopWallpaper = 0x0014
	spifUpdateIniFile      = 0x0001
	spifSendChange         = 0x0002

	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79

	gwlStyle     = ^uintptr(15) // -16
	gwlExStyle   = ^uintptr(19) // -20
	wsChild      = 0x40000000
	wsPopup      = 0x80000000
	wsCaption    = 0x00C00000
	wsThickFrame = 0x00040000
	wsSysMenu    = 0x00080000
	wsVisible    = 0x10000000

	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000

	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
	swpShowWindow   = 0x0040
	swShow          = 5
	// SW_SHOWNOACTIVATE 显示窗口但不激活，避免 WebView2 在初始化完成前收到焦点消息。
	swShowNoActivate = 4
)

func applyImageWallpaper(path string) error {
	if err := stopDynamicWallpaper(); err != nil {
		return err
	}

	target := path
	// Windows 桌面壁纸只稳定支持 BMP / JPG / PNG，WebP、GIF 需要先转成 BMP。
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".webp" || ext == ".gif" {
		converted, err := convertToBMP(path)
		if err != nil {
			return fmt.Errorf("转换壁纸格式失败: %w", err)
		}
		target = converted
	}

	p, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	ret, _, callErr := procSystemParametersInfoW.Call(
		spiSetDesktopWallpaper,
		0,
		uintptr(unsafe.Pointer(p)),
		spifUpdateIniFile|spifSendChange,
	)
	if ret == 0 {
		return fmt.Errorf("设置桌面壁纸失败: %w", callErr)
	}
	return nil
}

// convertToBMP 把 WebP / GIF 等格式转成桌面壁纸可直接使用的 BMP。
func convertToBMP(source string) (string, error) {
	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	base := os.Getenv("AppData")
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "OpenWallpaper", "wallpaper-cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	target := filepath.Join(dir, fmt.Sprintf("wallpaper-%d.bmp", time.Now().UnixNano()))
	out, err := os.Create(target)
	if err != nil {
		return "", err
	}
	encodeErr := bmp.Encode(out, img)
	closeErr := out.Close()
	if encodeErr != nil {
		os.Remove(target)
		return "", encodeErr
	}
	if closeErr != nil {
		os.Remove(target)
		return "", closeErr
	}

	// 清理之前的转换缓存，只保留当前这张。
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bmp") {
				continue
			}
			old := filepath.Join(dir, entry.Name())
			if old != target {
				_ = os.Remove(old)
			}
		}
	}
	return target, nil
}

func applyDynamicWallpaper(w Wallpaper) error {
	app := application.Get()
	if app == nil {
		return errors.New("应用尚未初始化")
	}
	if old, ok := app.Window.GetByName(dynamicWindowName); ok {
		old.Close()
	}

	target := w.MediaURL
	if w.Kind == "video" {
		target = "/wallpaper.html?src=" + url.QueryEscape(w.MediaURL)
	}

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             dynamicWindowName,
		Title:            "OpenWallpaper Layer",
		URL:              target,
		Width:            1280,
		Height:           720,
		Frameless:        true,
		DisableResize:    true,
		Hidden:           true,
		BackgroundColour: application.NewRGB(0, 0, 0),
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
		},
	})

	go func() {
		var hwnd uintptr
		for i := 0; i < 200; i++ {
			if handle := win.NativeWindow(); handle != nil {
				hwnd = uintptr(handle)
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if hwnd == 0 {
			return
		}
		// 越早禁用激活越好：窗口一旦被激活，Wails 会调用 WebView2 的 MoveFocus，
		// 控制器尚未就绪时该调用返回 E_INVALIDARG，go-webview2 会直接结束进程。
		stopActivation(hwnd)
		// 等 WebView2 控制器初始化完成，再贴到桌面图层；找不到 WorkerW 时重试几次。
		time.Sleep(1500 * time.Millisecond)
		for attempt := 0; attempt < 8; attempt++ {
			if err := attachToDesktop(hwnd); err == nil {
				return
			}
			time.Sleep(700 * time.Millisecond)
		}
	}()

	return nil
}

func stopDynamicWallpaper() error {
	app := application.Get()
	if app == nil {
		return nil
	}
	if win, ok := app.Window.GetByName(dynamicWindowName); ok {
		win.Close()
	}
	return nil
}

func attachToDesktop(hwnd uintptr) error {
	worker := findWorkerW()
	if worker == 0 {
		worker = progmanWindow()
	}
	if worker == 0 {
		return errors.New("未找到桌面窗口")
	}

	// 顺序很关键：先禁用激活、改成子窗口样式，再 SetParent。
	// SetParent 会触发焦点切换，若此时窗口仍可激活，就会触达 WebView2 的 MoveFocus。
	stopActivation(hwnd)

	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	style &^= wsPopup | wsCaption | wsThickFrame | wsSysMenu
	style |= wsChild
	procSetWindowLongPtrW.Call(hwnd, gwlStyle, style)

	procSetParent.Call(hwnd, worker)

	metric := func(index uintptr) int32 {
		value, _, _ := procGetSystemMetrics.Call(index)
		return int32(value)
	}
	x := metric(smXVirtualScreen)
	y := metric(smYVirtualScreen)
	width := metric(smCXVirtualScreen)
	height := metric(smCYVirtualScreen)

	procSetWindowPos.Call(
		hwnd,
		0,
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		swpNoActivate|swpFrameChanged|swpShowWindow,
	)
	procShowWindow.Call(hwnd, swShowNoActivate)
	return nil
}

// stopActivation 把窗口标记为工具窗口且永不激活，避免抢焦点。
func stopActivation(hwnd uintptr) {
	exStyle, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)
	exStyle |= wsExToolWindow | wsExNoActivate
	procSetWindowLongPtrW.Call(hwnd, gwlExStyle, exStyle)
}

func progmanWindow() uintptr {
	hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(utf16Ptr("Progman"))), 0)
	return hwnd
}

func findWorkerW() uintptr {
	progman := progmanWindow()
	if progman != 0 {
		// 让 Explorer 生成承载壁纸的 WorkerW 窗口。
		procSendMessageTimeoutW.Call(progman, 0x052C, 0xD, 0, 0, 1000, 0)
		procSendMessageTimeoutW.Call(progman, 0x052C, 0xD, 1, 0, 1000, 0)
	}

	foundWorkerW = 0
	procEnumWindows.Call(enumWindowsCallback, 0)
	return foundWorkerW
}

var foundWorkerW uintptr

var enumWindowsCallback = syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
	shellView, _, _ := procFindWindowExW.Call(hwnd, 0, uintptr(unsafe.Pointer(utf16Ptr("SHELLDLL_DefView"))), 0)
	if shellView == 0 {
		return 1
	}
	worker, _, _ := procFindWindowExW.Call(0, hwnd, uintptr(unsafe.Pointer(utf16Ptr("WorkerW"))), 0)
	if worker != 0 {
		foundWorkerW = worker
		return 0
	}
	return 1
})

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

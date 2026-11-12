//go:build windows

package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const chromiumMetaURL = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"

var chromiumMu sync.Mutex

// chromiumRoot 返回便携版 Chromium 的安装目录。
func chromiumRoot() string {
	base := os.Getenv("AppData")
	if base == "" {
		return ""
	}
	return filepath.Join(base, "OpenWallpaper", "chromium")
}

func chromiumExe() string {
	root := chromiumRoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "chrome.exe")
}

// ensureChromium 返回便携版 Chromium 的可执行文件，不存在时自动下载并解压。
func ensureChromium() (string, error) {
	chromiumMu.Lock()
	defer chromiumMu.Unlock()

	exe := chromiumExe()
	if exe == "" {
		return "", errors.New("无法获取应用数据目录")
	}
	if info, err := os.Stat(exe); err == nil && !info.IsDir() {
		return exe, nil
	}

	url, err := chromiumDownloadURL()
	if err != nil {
		return "", err
	}

	zipPath := filepath.Join(os.TempDir(), "openwallpaper-chromium.zip")
	defer os.Remove(zipPath)
	if err := downloadFile(url, zipPath); err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "openwallpaper-chromium-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)
	if err := unzip(zipPath, tempDir); err != nil {
		return "", err
	}

	source := filepath.Join(tempDir, "chrome-win64")
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		found, findErr := findChromeExe(tempDir)
		if findErr != nil {
			return "", findErr
		}
		source = filepath.Dir(found)
	}

	root := chromiumRoot()
	if err := os.RemoveAll(root); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(source, root); err != nil {
		// 跨磁盘时无法直接重命名，退化为复制。
		if copyErr := copyTree(source, root); copyErr != nil {
			return "", copyErr
		}
	}

	if info, err := os.Stat(exe); err != nil || info.IsDir() {
		return "", errors.New("Chromium 解压后缺少 chrome.exe")
	}
	return exe, nil
}

func chromiumDownloadURL() (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(chromiumMetaURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取 Chromium 版本信息失败: %s", resp.Status)
	}

	var meta struct {
		Channels struct {
			Stable struct {
				Version   string `json:"version"`
				Downloads struct {
					Chrome []struct {
						Platform string `json:"platform"`
						URL      string `json:"url"`
					} `json:"chrome"`
				} `json:"downloads"`
			} `json:"Stable"`
		} `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}
	for _, item := range meta.Channels.Stable.Downloads.Chrome {
		if item.Platform == "win64" && item.URL != "" {
			return item.URL, nil
		}
	}
	if meta.Channels.Stable.Version != "" {
		return fmt.Sprintf(
			"https://storage.googleapis.com/chrome-for-testing-public/%s/win64/chrome-win64.zip",
			meta.Channels.Stable.Version,
		), nil
	}
	return "", errors.New("没有找到可用的 Chromium 下载地址")
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 Chromium 失败: %s", resp.Status)
	}

	temp := dest + ".part"
	file, err := os.Create(temp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(temp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(temp)
		return closeErr
	}
	return os.Rename(temp, dest)
}

func unzip(zipPath, dest string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	cleanDest := filepath.Clean(dest) + string(os.PathSeparator)
	for _, entry := range reader.File {
		target := filepath.Join(dest, entry.Name)
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), cleanDest) {
			return fmt.Errorf("压缩包包含非法路径: %s", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(out, source)
		source.Close()
		out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func findChromeExe(root string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(entry.Name(), "chrome.exe") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", err
	}
	if found == "" {
		return "", errors.New("压缩包中未找到 chrome.exe")
	}
	return found, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

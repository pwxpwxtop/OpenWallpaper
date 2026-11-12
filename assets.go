package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const thumbnailWidth = 480

func encodeToken(p string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(p))
}

func decodeToken(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func mediaURL(p string) string {
	return mediaBase() + "/media/" + encodeToken(p)
}

// mediaBase 在开发模式下把资源指向 Wails 资源服务器，生产模式使用相对路径。
func mediaBase() string {
	if os.Getenv("FRONTEND_DEVSERVER_URL") != "" {
		return "http://wails.localhost"
	}
	return ""
}

// assetMiddleware 拦截本地资源请求，把壁纸库中的文件安全地提供给 WebView。
func (s *WallpaperService) assetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/media/"):
			s.serveMedia(w, r, strings.TrimPrefix(r.URL.Path, "/media/"))
			return
		case strings.HasPrefix(r.URL.Path, "/thumb/"):
			s.serveThumb(w, r, strings.TrimPrefix(r.URL.Path, "/thumb/"))
			return
		case strings.HasPrefix(r.URL.Path, "/site/"):
			s.serveSite(w, r, strings.TrimPrefix(r.URL.Path, "/site/"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *WallpaperService) serveMedia(w http.ResponseWriter, r *http.Request, token string) {
	p, err := decodeToken(token)
	if err != nil || !s.isAllowed(p) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, p)
}

func (s *WallpaperService) serveSite(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.SplitN(rest, "/", 2)
	dir, err := decodeToken(parts[0])
	if err != nil || !s.isAllowed(dir) {
		http.NotFound(w, r)
		return
	}
	rel := "index.html"
	if len(parts) == 2 && parts[1] != "" {
		rel = path.Clean("/" + parts[1])[1:]
	}
	target := filepath.Join(dir, filepath.FromSlash(rel))
	check, err := filepath.Rel(dir, target)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, target)
}

func (s *WallpaperService) serveThumb(w http.ResponseWriter, r *http.Request, token string) {
	src, err := decodeToken(token)
	if err != nil || !s.isAllowed(src) {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(src)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cacheKey := fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d", src, info.Size(), info.ModTime().UnixNano()))))
	cachePath := filepath.Join(os.TempDir(), "OpenWallpaper", "thumbs", cacheKey+".jpg")
	if _, err := os.Stat(cachePath); err != nil {
		if err := createThumbnail(src, cachePath); err != nil {
			http.ServeFile(w, r, src)
			return
		}
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, cachePath)
}

func createThumbnail(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return fmt.Errorf("图片尺寸无效")
	}

	width := thumbnailWidth
	if bounds.Dx() < width {
		width = bounds.Dx()
	}
	height := bounds.Dy() * width / bounds.Dx()
	if height < 1 {
		height = 1
	}

	scaled := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, draw.Over, nil)

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), "thumb-*.jpg")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := jpeg.Encode(tmp, scaled, &jpeg.Options{Quality: 82}); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Close()
	return os.Rename(tmpName, dst)
}

package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wallpaper 描述一个可用的壁纸资源。
type Wallpaper struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Kind     string   `json:"kind"` // image | video | html
	Path     string   `json:"path"`
	Folder   string   `json:"folder"`
	MediaURL string   `json:"mediaUrl"`
	ThumbURL string   `json:"thumbUrl"`
	Size     int64    `json:"size"`
	Modified int64    `json:"modified"`
	Tags     []string `json:"tags"`
	Adult    bool     `json:"adult"`
}

// Folder 是壁纸库中的一个扫描目录。
type Folder struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// Settings 是持久化的应用设置。
type Settings struct {
	Folders     []Folder `json:"folders"`
	Theme       string   `json:"theme"`
	Active      string   `json:"active"`
	DownloadDir string   `json:"downloadDir"`
	PluginDir   string   `json:"pluginDir"`
	PluginPort  int      `json:"pluginPort"`
}

type storedSettings struct {
	Folders     []string            `json:"folders"`
	Theme       string              `json:"theme"`
	Active      string              `json:"active"`
	DownloadDir string              `json:"downloadDir"`
	Items       map[string]itemMeta `json:"items"`
}

// itemMeta 保存单个壁纸的标签与成人内容标记。
type itemMeta struct {
	Tags  []string `json:"tags"`
	Adult bool     `json:"adult"`
}

// WallpaperService 提供壁纸扫描、应用与设置管理能力。
type WallpaperService struct {
	mu              sync.RWMutex
	folders         []string
	theme           string
	active          string
	downloadDirPath string
	items           map[string]itemMeta
	configPath      string
}

func NewWallpaperService() *WallpaperService {
	s := &WallpaperService{theme: "dark", items: make(map[string]itemMeta)}
	if dir, err := os.UserConfigDir(); err == nil {
		s.configPath = filepath.Join(dir, "OpenWallpaper", "config.json")
		s.load()
	}
	// 下载目录默认作为壁纸库来源，插件抓取的图片可直接在壁纸库中看到。
	s.ensureDownloadFolder()
	return s
}

func (s *WallpaperService) load() {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return
	}
	var stored storedSettings
	if err := json.Unmarshal(data, &stored); err != nil {
		return
	}
	s.folders = stored.Folders
	s.active = stored.Active
	s.downloadDirPath = stored.DownloadDir
	if stored.Items != nil {
		s.items = stored.Items
	}
	if stored.Theme == "light" || stored.Theme == "dark" {
		s.theme = stored.Theme
	}
}

func (s *WallpaperService) save() error {
	if s.configPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.configPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(storedSettings{
		Folders:     s.folders,
		Theme:       s.theme,
		Active:      s.active,
		DownloadDir: s.downloadDirPath,
		Items:       s.items,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath, data, 0o644)
}

func (s *WallpaperService) settings() Settings {
	downloadDir := s.downloadDirPath
	if downloadDir == "" {
		downloadDir = defaultDownloadDir()
	}
	pluginPath, _ := pluginDir()
	out := Settings{
		Theme:       s.theme,
		Active:      s.active,
		Folders:     make([]Folder, 0, len(s.folders)),
		DownloadDir: downloadDir,
		PluginDir:   pluginPath,
		PluginPort:  pluginPort,
	}
	for _, f := range s.folders {
		info, err := os.Stat(f)
		out.Folders = append(out.Folders, Folder{Path: f, Exists: err == nil && info.IsDir()})
	}
	return out
}

// GetSettings 返回当前设置。
func (s *WallpaperService) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings()
}

// SelectFolder 打开系统目录选择框并返回用户选择的目录（取消时返回空字符串）。
func (s *WallpaperService) SelectFolder() (string, error) {
	app := application.Get()
	if app == nil {
		return "", errors.New("应用尚未初始化")
	}
	path, err := app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("选择壁纸文件夹").
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

// AddFolder 把一个目录加入壁纸库。
func (s *WallpaperService) AddFolder(path string) (Settings, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.settings(), nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return Settings{}, fmt.Errorf("目录不存在: %s", path)
	}
	clean := filepath.Clean(path)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.folders {
		if strings.EqualFold(f, clean) {
			return s.settings(), nil
		}
	}
	s.folders = append(s.folders, clean)
	if err := s.save(); err != nil {
		return Settings{}, err
	}
	return s.settings(), nil
}

// RemoveFolder 从壁纸库移除一个目录。
func (s *WallpaperService) RemoveFolder(path string) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]string, 0, len(s.folders))
	for _, f := range s.folders {
		if !strings.EqualFold(f, path) {
			kept = append(kept, f)
		}
	}
	s.folders = kept
	if err := s.save(); err != nil {
		return Settings{}, err
	}
	return s.settings(), nil
}

// SetTheme 持久化界面主题。
func (s *WallpaperService) SetTheme(theme string) error {
	if theme != "dark" && theme != "light" {
		return fmt.Errorf("不支持的主题: %s", theme)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.theme = theme
	return s.save()
}

var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".bmp": true, ".gif": true,
}

var videoExtensions = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".mkv": true, ".m4v": true, ".avi": true,
}

// ScanWallpapers 扫描所有目录，返回图片、视频和 HTML 壁纸。
func (s *WallpaperService) ScanWallpapers() ([]Wallpaper, error) {
	s.mu.RLock()
	folders := append([]string(nil), s.folders...)
	metas := make(map[string]itemMeta, len(s.items))
	for id, meta := range s.items {
		metas[id] = meta
	}
	s.mu.RUnlock()

	items := make([]Wallpaper, 0, 64)
	seen := make(map[string]bool)

	for _, folder := range folders {
		info, err := os.Stat(folder)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if path != folder && (strings.HasPrefix(name, ".") || name == "node_modules") {
					return filepath.SkipDir
				}
				if depth(folder, path) > 3 {
					return filepath.SkipDir
				}
				index := filepath.Join(path, "index.html")
				if fi, err := os.Stat(index); err == nil && !fi.IsDir() {
					item := withItemMeta(newHTMLWallpaper(path, index, folder), metas)
					if !seen[item.ID] {
						seen[item.ID] = true
						items = append(items, item)
					}
					return filepath.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			kind := ""
			switch {
			case imageExtensions[ext]:
				kind = "image"
			case videoExtensions[ext]:
				kind = "video"
			case ext == ".html" || ext == ".htm":
				kind = "html"
			default:
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			media := mediaURL(path)
			if kind == "html" {
				media = mediaBase() + "/site/" + encodeToken(filepath.Dir(path)) + "/" + url.PathEscape(d.Name())
			}
			item := Wallpaper{
				ID:       wallpaperID(kind, path),
				Name:     strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
				Kind:     kind,
				Path:     path,
				Folder:   folder,
				MediaURL: media,
				Size:     info.Size(),
				Modified: info.ModTime().Unix(),
			}
			if kind == "image" {
				item.ThumbURL = mediaBase() + "/thumb/" + encodeToken(path)
			}
			item = withItemMeta(item, metas)
			if !seen[item.ID] {
				seen[item.ID] = true
				items = append(items, item)
			}
			return nil
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Modified == items[j].Modified {
			return items[i].Name < items[j].Name
		}
		return items[i].Modified > items[j].Modified
	})
	return items, nil
}

// SetWallpaperMeta 保存单个壁纸的标签与成人内容标记。
func (s *WallpaperService) SetWallpaperMeta(id string, tags []string, adult bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("无效的壁纸")
	}
	tags = normalizeTags(tags)

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(tags) == 0 && !adult {
		delete(s.items, id)
	} else {
		s.items[id] = itemMeta{Tags: tags, Adult: adult}
	}
	return s.save()
}

// normalizeTags 去空白、去重，并限制标签数量与长度。
func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if runes := []rune(tag); len(runes) > 20 {
			tag = string(runes[:20])
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
		if len(out) >= 20 {
			break
		}
	}
	return out
}

func withItemMeta(item Wallpaper, metas map[string]itemMeta) Wallpaper {
	if meta, ok := metas[item.ID]; ok {
		item.Tags = meta.Tags
		item.Adult = meta.Adult
	}
	return item
}

// ApplyWallpaper 将指定壁纸设置为桌面壁纸。
func (s *WallpaperService) ApplyWallpaper(w Wallpaper) error {
	if w.Path == "" {
		return errors.New("无效的壁纸")
	}
	if !s.isAllowed(w.Path) {
		return errors.New("壁纸不在壁纸库目录中")
	}

	switch w.Kind {
	case "image":
		if err := applyImageWallpaper(w.Path); err != nil {
			return err
		}
	case "video", "html":
		if err := applyDynamicWallpaper(w); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的壁纸类型: %s", w.Kind)
	}

	s.mu.Lock()
	s.active = w.Path
	err := s.save()
	s.mu.Unlock()
	return err
}

// StopDynamicWallpaper 关闭视频 / HTML 壁纸图层，恢复静态壁纸。
func (s *WallpaperService) StopDynamicWallpaper() error {
	if err := stopDynamicWallpaper(); err != nil {
		return err
	}
	s.mu.Lock()
	s.active = ""
	err := s.save()
	s.mu.Unlock()
	return err
}

// DeleteWallpaper 删除壁纸文件；目录形式的 HTML 壁纸会连同整个目录一起删除。
func (s *WallpaperService) DeleteWallpaper(w Wallpaper) error {
	if w.Path == "" {
		return errors.New("无效的壁纸")
	}
	if !s.isAllowed(w.Path) {
		return errors.New("壁纸不在壁纸库目录中")
	}

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active != "" && strings.EqualFold(active, w.Path) {
		if err := stopDynamicWallpaper(); err != nil {
			return err
		}
		s.mu.Lock()
		s.active = ""
		s.mu.Unlock()
	}

	target := w.Path
	if w.Kind == "html" && strings.EqualFold(filepath.Base(w.Path), "index.html") {
		target = filepath.Dir(w.Path)
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.items, w.ID)
	err := s.save()
	s.mu.Unlock()
	return err
}

func (s *WallpaperService) isAllowed(path string) bool {
	s.mu.RLock()
	folders := append([]string(nil), s.folders...)
	s.mu.RUnlock()

	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = resolved
	}
	for _, folder := range folders {
		root, err := filepath.Abs(folder)
		if err != nil {
			continue
		}
		if r, err := filepath.EvalSymlinks(root); err == nil {
			root = r
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

func newHTMLWallpaper(dir, index, folder string) Wallpaper {
	info, _ := os.Stat(index)
	var size int64
	var modified int64
	if info != nil {
		size = info.Size()
		modified = info.ModTime().Unix()
	}
	return Wallpaper{
		ID:       wallpaperID("html", dir),
		Name:     filepath.Base(dir),
		Kind:     "html",
		Path:     index,
		Folder:   folder,
		MediaURL: mediaBase() + "/site/" + encodeToken(dir) + "/index.html",
		Size:     size,
		Modified: modified,
	}
}

func wallpaperID(kind, path string) string {
	sum := md5.Sum([]byte(kind + "|" + path))
	return hex.EncodeToString(sum[:8])
}

func depth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(os.PathSeparator)))
}

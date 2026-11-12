# OpenWallpaper

本地桌面壁纸管理工具，支持图片、视频与 HTML 壁纸，并配套一个浏览器扩展，用于从网页快速采集图片 / 视频到壁纸库。

技术栈：Wails v3（alpha.74）+ Go 1.25 + Vue 3 + Vite + PrimeVue 4 + Pinia。

## 功能

- **壁纸库**：扫描本地文件夹中的图片、视频与 HTML 壁纸，支持瀑布流展示、类型切换、搜索、刷新。
- **应用壁纸**：图片通过系统接口设置；视频与 HTML 壁纸以独立窗口贴到桌面图标下方（WorkerW 层）。
- **标签与筛选**：每张壁纸可设置类型、多个标签与成人内容标记；筛选弹窗支持类型、预设标签、自定义标签与 18+ 过滤。
- **删除壁纸**：可直接删除文件；HTML 壁纸会连同整个目录一起删除。
- **托盘**：点击关闭按钮隐藏到右下角托盘，只有托盘右键菜单中的“退出”才会真正关闭应用。
- **设置**：壁纸文件夹、深浅主题、下载缓存目录、插件导出、支持格式说明。
- **推荐网站**：内置 Pexels、Unsplash、Pixabay、Wallhaven 等免费素材站。
- **浏览器插件**：给网页图片加下载角标、右侧可拖拽悬浮球、图片 / 视频列表、低像素筛选、整页 HTML 下载，通过本地 WebSocket 以二进制回传并保存到缓存目录。
- **多语言**：应用与插件均支持 14 种语言（简繁中文、英、日、韩、德、俄、越、西、法、葡、意、泰、印尼）。

## 推荐网站的独立浏览器

点击“推荐网站”中的任意站点时，应用会：

1. 优先复用已经由应用创建的独立浏览器窗口（新开标签页，不覆盖已有页面）；
2. 若未运行，使用应用自己的便携版 Chromium（Chrome for Testing）；已下载过则直接复用，不再重复下载；
3. 本机 Chrome / Edge 会被 Chrome 官方限制挡住程序化加载未打包插件，仅在便携版不可用时作为兜底；
4. 通过 DevTools 协议自动安装插件、最大化窗口并打开目标网址。

> 浏览器窗口使用独立配置目录 `%AppData%\OpenWallpaper\app-browser`，不会影响日常浏览的 Chrome / Edge 配置。

## 环境要求

- Windows 10 / 11（依赖 WebView2 Runtime）
- Go 1.25+
- Node.js 18+
- Wails v3 CLI（`wails3`，alpha.74）
- Chrome 或 Edge（可选；没有 Chrome 时会自动下载便携版内核）

## 开发

```bash
# 启动开发模式（前后端热更新）
wails3 dev

# 仅构建前端
cd frontend && npm install && npm run build

# 构建 Windows 可执行文件
go build .
```

重新生成前后端绑定：

```bash
wails3 generate bindings -ts -i -clean -d frontend/bindings
```

## 目录结构

```
.
├── main.go                  # 应用入口：窗口、托盘、服务注册
├── wallpaper_service.go     # 壁纸扫描、应用、设置、标签与删除
├── wallpaper_windows.go     # Windows 桌面图层（WorkerW）实现
├── pluginserver.go          # 本地 WebSocket 服务、插件导出、推荐页后端接口
├── browser_launch_windows.go# 通过 CDP 启动 / 复用浏览器并安装插件
├── chromium_windows.go      # 便携版 Chromium 自动下载与解压
├── open_folder_windows.go   # 打开目录 / 浏览器、浏览器路径探测
├── plugin/                  # 浏览器扩展源码（会被内嵌进应用）
│   ├── manifest.json
│   ├── background.js
│   ├── content.js
│   ├── page-hook.js
│   └── _locales/            # 插件语言包（14 种）
└── frontend/
    └── src/
        ├── views/           # 壁纸库 / 推荐网站 / 设置
        ├── components/      # 壁纸卡片、筛选、标签、预览弹窗
        └── stores/          # Pinia：壁纸、设置、国际化
```

## 数据目录

| 内容 | 路径 |
| --- | --- |
| 应用配置 | `%AppData%\OpenWallpaper\config.json` |
| 导出的插件 | `%AppData%\OpenWallpaper\plugin` |
| 独立浏览器配置 | `%AppData%\OpenWallpaper\app-browser` |
| 便携版 Chromium | `%AppData%\OpenWallpaper\chromium` |
| 默认下载缓存 | `%UserProfile%\Pictures\OpenWallpaper` |

## 浏览器插件

- 点击推荐网站时会自动创建浏览器并安装插件。
- 也可以手动安装：设置页点击“导出并打开插件目录”，然后在 Chrome / Edge 扩展管理页开启开发者模式，选择“加载已解压的扩展程序”并指向导出的目录。
- 插件默认连接本地 `ws://127.0.0.1:48921/ws`，需要 OpenWallpaper 正在运行。

## 多语言

应用语言包位于 [`frontend/src/stores/i18n.js`](frontend/src/stores/i18n.js)。新增语言只需在 `locales` 中追加一项：

```js
ja: {
  label: '日本語',
  messages: {
    'app.library': 'ライブラリ',
    // 未提供的 key 会自动回退到简体中文
  },
},
```

插件语言包位于 `plugin/_locales/<locale>/messages.json`，遵循 Chrome 扩展 i18n 规范。

## 注意事项

- Chrome 137 及以上已收紧 `--load-extension` 命令行加载，本项目改用 DevTools 协议的 `Extensions.loadUnpacked` 安装插件。
- 首次下载便携版 Chromium 需要网络（约 150 MB），之后会直接复用。
- 关闭主窗口不会退出应用，请使用托盘菜单退出。
- 构建时若出现 `The requested operation cannot be performed on a file with a user-mapped section open`，说明旧版应用仍在运行（关闭按钮只是隐藏到托盘）。请右键托盘图标选择“退出”，或结束 `bin/OpenWallpaper.exe` 进程后重新构建。

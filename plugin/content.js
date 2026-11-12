// OpenWallpaper 媒体采集 - 页面 UI
// 1) 给页面里的每张图片左上角加一个缓存按钮
// 2) 网页右侧可拖拽的悬浮按钮，点开后可浏览并批量下载该页图片 / 视频
// 3) 支持把整个网页的 HTML 保存到本地

(() => {
  if (window.top !== window) {
    return
  }
  if (window.__openwallpaperContentReady) {
    return
  }
  window.__openwallpaperContentReady = true

  const MAX_BADGES = 120
  const ICON_DOWNLOAD =
    '<svg viewBox="0 0 24 24"><path d="M12 3v11"/><path d="m7.5 10.5 4.5 4.5 4.5-4.5"/><path d="M4 18.5h16"/></svg>'
  const ICON_BOLT =
    '<svg viewBox="0 0 24 24"><path d="M13 2 4.5 13.5H11l-1 8.5 8.5-11.5H12l1-8.5Z"/></svg>'

  // 界面文案跟随浏览器语言，使用扩展的 _locales 语言包；缺失时回退到中文默认值。
  function t(key, fallback = '') {
    try {
      const message = chrome.i18n.getMessage(key)
      if (message) {
        return message
      }
    } catch (err) {
      /* 非扩展环境 */
    }
    return fallback || key
  }

  function tf(key, fallback, params) {
    let text = t(key, fallback)
    for (const [name, value] of Object.entries(params || {})) {
      text = text.split(`{${name}}`).join(String(value))
    }
    return text
  }

  const captured = new Map()
  const statusMap = new Map()
  const badgeMap = new Map()
  // url -> { width, height }：记录探测到的真实像素尺寸，供低像素筛选使用。
  const sizeMap = new Map()
  const current = { image: [], video: [] }

  let activeTab = 'image'
  let minPixels = 0
  let panelOpen = false
  let connected = false
  let pingTimer = 0
  let badgeFrame = 0
  let dragged = false
  let dragStartX = 0
  let dragStartY = 0
  let dragStartLeft = 0
  let dragStartTop = 0

  const host = document.createElement('div')
  host.id = '__openwallpaper_media_host__'
  host.style.cssText =
    'all: initial; position: fixed; inset: 0 auto auto 0; width: 0; height: 0; z-index: 2147483647;'
  const root = host.attachShadow({ mode: 'open' })
  root.innerHTML = `
    <style>
      :host { all: initial; }
      * { box-sizing: border-box; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Microsoft YaHei', sans-serif; }
      .fab {
        position: fixed; left: 0; top: 0; width: 44px; height: 44px; border-radius: 14px;
        z-index: 2;
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        box-shadow: 0 10px 26px rgba(79, 70, 229, 0.45);
        display: grid; place-items: center; cursor: grab; color: #fff;
        touch-action: none; transition: transform 0.15s ease, box-shadow 0.15s ease;
      }
      .fab:hover { transform: scale(1.06); box-shadow: 0 14px 32px rgba(79, 70, 229, 0.55); }
      .fab:active { cursor: grabbing; }
      .fab svg { width: 22px; height: 22px; stroke: currentColor; fill: none; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .panel {
        position: fixed; left: 50%; top: 50%; width: min(900px, 94vw); max-height: min(820px, 88vh);
        z-index: 3;
        display: none; flex-direction: column; overflow: hidden;
        background: rgba(13, 14, 21, 0.97); color: #eef1f8;
        border: 1px solid rgba(255, 255, 255, 0.1); border-radius: 16px;
        box-shadow: 0 26px 64px rgba(0, 0, 0, 0.6);
        backdrop-filter: blur(18px); transform: translate(-50%, -50%);
      }
      .panel.open { display: flex; }
      .panel header {
        display: flex; align-items: center; gap: 10px; padding: 13px 14px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.08);
        cursor: grab; touch-action: none; user-select: none;
      }
      .panel header.dragging { cursor: grabbing; }
      .panel header .title { display: flex; align-items: center; gap: 7px; flex: 1; min-width: 0; }
      .panel header strong { font-size: 13px; font-weight: 600; letter-spacing: 0.01em; }
      .panel header small { font-size: 11px; color: #8b93a7; }
      .dot { width: 7px; height: 7px; border-radius: 50%; background: #f43f5e; flex-shrink: 0; }
      .dot.on { background: #22c55e; box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.18); }
      .icon-btn {
        width: 26px; height: 26px; border-radius: 8px; border: 1px solid rgba(255, 255, 255, 0.1);
        background: rgba(255, 255, 255, 0.04); color: #c8cfe0; cursor: pointer;
        display: grid; place-items: center; font-size: 14px; line-height: 1; padding: 0;
      }
      .icon-btn:hover { background: rgba(255, 255, 255, 0.1); color: #fff; }
      .tabs { display: flex; gap: 6px; padding: 10px 12px 0; }
      .tabs button {
        flex: 1; padding: 7px 0; border-radius: 9px; border: 1px solid transparent;
        background: transparent; color: #9aa3b8; font-size: 12px; cursor: pointer;
      }
      .tabs button.active { background: rgba(99, 102, 241, 0.18); border-color: rgba(129, 140, 248, 0.4); color: #e0e7ff; }
      .tabs em { font-style: normal; opacity: 0.7; margin-left: 2px; }
      .toolbar-min {
        display: flex; align-items: center; justify-content: space-between; gap: 8px;
        padding: 8px 14px 4px; font-size: 11px; color: #9aa3b8;
      }
      .min-res {
        appearance: none; border: 1px solid rgba(255, 255, 255, 0.12); border-radius: 8px;
        background: rgba(255, 255, 255, 0.05); color: #e5e9f4; font-size: 11px;
        padding: 4px 8px; cursor: pointer; outline: none;
      }
      .min-res:hover { border-color: rgba(129, 140, 248, 0.5); }
      .list {
        flex: 1; overflow-y: auto; overflow-x: hidden; padding: 12px; min-height: 160px;
      }
      .list::-webkit-scrollbar { width: 6px; }
      .list::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.16); border-radius: 6px; }
      .masonry { column-width: 168px; column-gap: 10px; }
      .card {
        display: block; width: 100%; margin-bottom: 10px; break-inside: avoid;
        border-radius: 12px; overflow: hidden;
        border: 1px solid rgba(255, 255, 255, 0.07);
        background: rgba(255, 255, 255, 0.035);
      }
      .pic {
        position: relative; width: 100%; overflow: hidden;
        background: #1b1e2b; display: grid; place-items: center;
        color: #6f7891; font-size: 22px; cursor: zoom-in;
      }
      .card.video .pic { min-height: 100px; }
      .card.image .pic { min-height: 70px; }
      .pic img { width: 100%; height: auto; display: block; transition: transform 0.2s ease; }
      .card:hover .pic img { transform: scale(1.04); }
      .card .dl {
        position: absolute; right: 7px; bottom: 7px; padding: 4px 10px; border-radius: 7px;
        border: 1px solid rgba(129, 140, 248, 0.45); background: rgba(15, 17, 28, 0.74);
        color: #c7d2fe; font-size: 11px; cursor: pointer; backdrop-filter: blur(4px);
      }
      .card .dl:hover { background: rgba(99, 102, 241, 0.62); color: #fff; }
      .card .dl[disabled] { cursor: default; border-color: rgba(255, 255, 255, 0.12); background: rgba(15, 17, 28, 0.62); color: #8b93a7; }
      .card .dl.saved { border-color: rgba(34, 197, 94, 0.4); color: #86efac; }
      .card .dl.error { border-color: rgba(244, 63, 94, 0.4); color: #fda4af; }
      .cap { padding: 7px 9px 9px; display: flex; flex-direction: column; gap: 2px; min-width: 0; }
      .cap span { font-size: 11.5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
      .cap small { font-size: 10px; color: #7f8798; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
      .empty { column-span: all; padding: 22px 12px; text-align: center; font-size: 12px; color: #9aa3b8; display: flex; flex-direction: column; gap: 6px; }
      .empty small { font-size: 10.5px; color: #7f8798; line-height: 1.5; }
      .panel footer { display: flex; gap: 8px; padding: 10px 12px; border-top: 1px solid rgba(255, 255, 255, 0.08); }
      .panel footer button {
        flex: 1; padding: 8px 0; border-radius: 9px; font-size: 12px; cursor: pointer;
        border: 1px solid rgba(255, 255, 255, 0.14); background: rgba(255, 255, 255, 0.05); color: #dbe1ee;
      }
      .panel footer button:hover { background: rgba(255, 255, 255, 0.1); }
      .panel footer button.primary { border: none; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; }
      .panel footer button.primary:hover { filter: brightness(1.08); }
      .viewer {
        position: fixed; inset: 0; display: none; place-items: center;
        z-index: 4;
        background: rgba(4, 5, 9, 0.88); backdrop-filter: blur(6px);
      }
      .viewer.open { display: grid; }
      .viewer img, .viewer video {
        max-width: 92vw; max-height: 88vh; border-radius: 12px; background: #000;
        box-shadow: 0 24px 64px rgba(0, 0, 0, 0.65);
      }
      .viewer-close {
        position: absolute; top: 20px; right: 24px; width: 40px; height: 40px; border-radius: 50%;
        border: 1px solid rgba(255, 255, 255, 0.18); background: rgba(20, 22, 34, 0.9);
        color: #e5e7eb; font-size: 17px; cursor: pointer;
      }
      .viewer-close:hover { background: rgba(99, 102, 241, 0.72); color: #fff; }
      .toast {
        position: fixed; left: 18px; bottom: 18px; padding: 9px 13px; border-radius: 10px;
        z-index: 5;
        background: rgba(13, 15, 24, 0.95); color: #eef1f8; border: 1px solid rgba(255, 255, 255, 0.12);
        font-size: 12px; opacity: 0; transform: translateY(8px); transition: opacity 0.2s ease, transform 0.2s ease;
        pointer-events: none; max-width: 320px;
      }
      .toast.show { opacity: 1; transform: translateY(0); }
      .badge {
        position: fixed; left: 0; top: 0; width: 28px; height: 28px;
        z-index: 1;
        display: none; align-items: center; justify-content: center;
        margin: 0; padding: 0; border-radius: 9px;
        border: 1px solid rgba(255, 255, 255, 0.35);
        background: rgba(13, 15, 24, 0.78); color: #e0e7ff;
        cursor: pointer; box-shadow: 0 4px 14px rgba(0, 0, 0, 0.35);
        transition: background 0.15s ease, border-color 0.15s ease;
      }
      .badge:hover { background: linear-gradient(135deg, #6366f1, #8b5cf6); border-color: transparent; color: #fff; }
      .badge svg { width: 16px; height: 16px; stroke: currentColor; fill: none; stroke-width: 1.9; stroke-linecap: round; stroke-linejoin: round; }
    </style>
    <div class="fab" part="fab">${ICON_BOLT}</div>
    <div class="panel">
      <header>
        <div class="title">
          <span class="dot"></span>
          <strong>${t('panelTitle', '媒体采集')}</strong>
          <small>${t('disconnected', '未连接，请启动应用')}</small>
        </div>
        <button class="icon-btn" data-action="refresh" title="${t('refreshList', '刷新列表')}">↻</button>
        <button class="icon-btn" data-action="close" title="${t('close', '关闭')}">✕</button>
      </header>
      <nav class="tabs">
        <button data-tab="image" class="active">${t('tabImage', '图片')} <em>0</em></button>
        <button data-tab="video">${t('tabVideo', '视频')} <em>0</em></button>
      </nav>
      <div class="toolbar-min">
        <span>${t('excludeLowRes', '排除低像素图片')}</span>
        <select class="min-res" title="${t('filterByLongEdge', '按图片长边过滤')}">
          <option value="0">${t('filterNone', '不限')}</option>
          <option value="1280">${t('filter1280', '长边 ≥ 1280')}</option>
          <option value="1920">${t('filter1920', '长边 ≥ 1920')}</option>
          <option value="2560">${t('filter2560', '长边 ≥ 2560')}</option>
          <option value="3840">${t('filter3840', '长边 ≥ 3840')}</option>
        </select>
      </div>
      <div class="list"><div class="masonry"></div></div>
      <footer>
        <button class="primary" data-action="all">${t('downloadAll', '全部下载')}</button>
        <button data-action="page">${t('downloadPage', '下载整个网页')}</button>
      </footer>
    </div>
    <div class="viewer">
      <button class="viewer-close" title="${t('close', '关闭')}">✕</button>
      <img class="viewer-img" alt="" />
      <video class="viewer-video" controls playsinline></video>
    </div>
    <div class="toast"></div>
  `

  const fabEl = root.querySelector('.fab')
  const panelEl = root.querySelector('.panel')
  const listEl = root.querySelector('.list')
  const masonryEl = root.querySelector('.masonry')
  const toastEl = root.querySelector('.toast')
  const dotEl = root.querySelector('.dot')
  const viewerEl = root.querySelector('.viewer')
  const viewerImg = root.querySelector('.viewer-img')
  const viewerVideo = root.querySelector('.viewer-video')
  const statusEl = root.querySelector('.panel header small')
  const countEls = {
    image: root.querySelector('[data-tab="image"] em'),
    video: root.querySelector('[data-tab="video"] em'),
  }
  const minResEl = root.querySelector('.min-res')

  // 点击卡片预览：图片放大查看，视频直接播放。
  function openViewer(item) {
    viewerImg.style.display = 'none'
    viewerImg.removeAttribute('src')
    viewerVideo.style.display = 'none'
    viewerVideo.pause()
    viewerVideo.removeAttribute('src')
    viewerVideo.load()
    if (item.kind === 'video') {
      viewerVideo.src = item.url
      viewerVideo.style.display = 'block'
      const played = viewerVideo.play()
      if (played && typeof played.catch === 'function') {
        played.catch(() => {})
      }
    } else {
      viewerImg.src = item.url
      viewerImg.style.display = 'block'
    }
    viewerEl.classList.add('open')
  }

  function closeViewer() {
    viewerEl.classList.remove('open')
    viewerVideo.pause()
    viewerVideo.removeAttribute('src')
    viewerVideo.load()
    viewerImg.removeAttribute('src')
  }

  viewerEl.addEventListener('click', (event) => {
    if (event.target === viewerEl) {
      closeViewer()
    }
  })
  root.querySelector('.viewer-close').addEventListener('click', closeViewer)
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && viewerEl.classList.contains('open')) {
      closeViewer()
    }
  })

  function mount() {
    if (!document.documentElement.contains(host)) {
      document.documentElement.appendChild(host)
    }
  }
  mount()
  fabEl.style.left = 'calc(100vw - 58px)'
  fabEl.style.top = '42%'

  // ---------- 工具函数 ----------

  function hash(text) {
    let value = 5381
    for (let i = 0; i < text.length; i += 1) {
      value = ((value << 5) + value + text.charCodeAt(i)) >>> 0
    }
    return value.toString(36)
  }

  function absolute(url) {
    if (!url) {
      return ''
    }
    try {
      return new URL(url, location.href).href
    } catch (err) {
      return ''
    }
  }

  function usable(url) {
    return /^https?:/i.test(url)
  }

  function guessKind(url) {
    if (/\.(mp4|m4v|webm|mov|mkv|flv|m3u8|mpd|ts)(?:$|[?#])/i.test(url)) {
      return 'video'
    }
    if (/\.(jpe?g|png|webp|gif|avif|bmp|svg)(?:$|[?#])/i.test(url)) {
      return 'image'
    }
    return ''
  }

  function nameFromURL(url, fallback) {
    try {
      const parsed = new URL(url)
      const parts = parsed.pathname.split('/').filter(Boolean)
      let name = parts.length ? decodeURIComponent(parts[parts.length - 1]) : ''
      name = name.replace(/[\\/:*?"<>|]/g, '-').trim()
      if (!name) {
        return `${fallback}-${Date.now().toString(36)}`
      }
      if (!/\.[a-z0-9]{1,8}$/i.test(name)) {
        const ext = guessKind(url) === 'video' ? 'mp4' : 'jpg'
        name = `${name}.${ext}`
      }
      return name.slice(0, 120)
    } catch (err) {
      return `${fallback}-${Date.now().toString(36)}`
    }
  }

  function hostDir() {
    return (location.hostname || 'web').replace(/[\\/:*?"<>|]/g, '-').slice(0, 60)
  }

  function stamp() {
    const now = new Date()
    const pad = (value) => String(value).padStart(2, '0')
    return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(
      now.getMinutes(),
    )}${pad(now.getSeconds())}`
  }

  function sendMessage(message, callback) {
    try {
      chrome.runtime.sendMessage(message, (response) => {
        if (chrome.runtime.lastError) {
          if (callback) {
            callback(null)
          }
          return
        }
        if (callback) {
          callback(response)
        }
      })
    } catch (err) {
      if (callback) {
        callback(null)
      }
    }
  }

  let toastTimer = 0
  function showToast(text) {
    toastEl.textContent = text
    toastEl.classList.add('show')
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => toastEl.classList.remove('show'), 2400)
  }

  // ---------- 媒体收集 ----------

  const MIN_IMAGE_SIDE = 96
  const ICON_URL_RE = /(?:favicon|sprite|spacer|placeholder|1x1|pixel\.(?:gif|png))/i

  // 小图标（站点 logo、图标、占位图、像素点）不参与采集。
  function looksLikeIcon(url) {
    return ICON_URL_RE.test(url)
  }

  function isTinyImage(img) {
    const width = img.naturalWidth
    const height = img.naturalHeight
    if (!width || !height) {
      return false
    }
    return Math.max(width, height) < MIN_IMAGE_SIDE
  }

  function collect() {
    const images = new Map()
    const videos = new Map()

    const put = (map, raw, kind, extra) => {
      const url = absolute(raw)
      if (!url || !usable(url) || map.has(url)) {
        return
      }
      const known = kind === 'image' ? sizeMap.get(url) : null
      map.set(url, {
        id: hash(url),
        url,
        kind,
        name: nameFromURL(url, kind),
        thumb: '',
        ...(kind === 'image' && known ? { width: known.width, height: known.height } : null),
        ...(extra || {}),
      })
    }

    for (const img of document.images) {
      if (isTinyImage(img)) {
        continue
      }
      const src = img.currentSrc || img.src
      put(images, src, 'image', {
        thumb: src,
        width: img.naturalWidth || 0,
        height: img.naturalHeight || 0,
      })
    }

    for (const el of document.querySelectorAll('[style*="background-image"]')) {
      const style = el.getAttribute('style') || ''
      const regex = /url\((['"]?)(.*?)\1\)/g
      let match = regex.exec(style)
      while (match) {
        if (!looksLikeIcon(match[2])) {
          put(images, match[2], 'image')
        }
        match = regex.exec(style)
      }
    }

    for (const video of document.querySelectorAll('video')) {
      const poster = video.poster || ''
      put(videos, video.currentSrc || video.src, 'video', { thumb: poster })
      for (const source of video.querySelectorAll('source')) {
        put(videos, source.src, 'video', { thumb: poster })
      }
    }

    for (const [url, kind] of captured) {
      if (kind === 'video') {
        put(videos, url, 'video')
      } else if (!looksLikeIcon(url)) {
        put(images, url, 'image')
      }
    }

    current.image = Array.from(images.values())
    current.video = Array.from(videos.values())
    countEls.image.textContent = String(current.image.length)
    countEls.video.textContent = String(current.video.length)
  }

  // ---------- 列表渲染 ----------

  // 尺寸未知的图片（背景图 / 接口抓取）保留，避免误排除。
  function passMinPixels(item) {
    if (!minPixels || item.kind !== 'image') {
      return true
    }
    const edge = Math.max(item.width || 0, item.height || 0)
    return !edge || edge >= minPixels
  }

  function items() {
    const list = activeTab === 'video' ? current.video : current.image
    return minPixels ? list.filter(passMinPixels) : list
  }

  function updateCounts() {
    countEls.image.textContent = String(
      minPixels ? current.image.filter(passMinPixels).length : current.image.length,
    )
    countEls.video.textContent = String(current.video.length)
  }

  function updateRow(id) {
    const button = listEl.querySelector(`button[data-dl="${id}"]`)
    if (!button) {
      return
    }
    const state = statusMap.get(id)
    button.classList.toggle('saved', state === 'saved')
    button.classList.toggle('error', state === 'error')
    if (state === 'pending') {
      button.textContent = t('sending', '发送中')
      button.disabled = true
    } else if (state === 'saved') {
      button.textContent = t('saved', '已保存')
      button.disabled = true
    } else if (state === 'error') {
      button.textContent = t('retry', '重试')
      button.disabled = false
    } else {
      button.textContent = t('download', '下载')
      button.disabled = false
    }
  }

  function render() {
    collect()
    updateCounts()
    masonryEl.textContent = ''
    const list = items()
    if (!list.length) {
      const empty = document.createElement('div')
      empty.className = 'empty'
      empty.textContent =
        activeTab === 'video' ? t('emptyVideoTitle', '暂未发现视频') : t('emptyImageTitle', '暂未发现图片')
      const hint = document.createElement('small')
      hint.textContent =
        activeTab === 'video'
          ? t('emptyVideoHint', '视频地址常藏在接口 JSON 中，播放或滚动页面后会自动捕获。')
          : t('emptyImageHint', '有些图片由样式或脚本加载，滚动页面后再刷新试试。')
      empty.appendChild(hint)
      masonryEl.appendChild(empty)
      return
    }

    for (const item of list) {
      const card = document.createElement('div')
      card.className = item.kind === 'video' ? 'card video' : 'card image'

      const pic = document.createElement('div')
      pic.className = 'pic'
      const src = item.thumb || (item.kind === 'image' ? item.url : '')
      if (src) {
        const img = document.createElement('img')
        // 开启低像素筛选时改为立即加载，以便尽快拿到真实尺寸。
        img.loading = minPixels ? 'eager' : 'lazy'
        img.decoding = 'async'
        img.alt = item.name
        img.addEventListener('error', () => {
          img.remove()
          pic.textContent = item.kind === 'video' ? '▶' : '□'
        })
        if (item.kind === 'image') {
          img.addEventListener('load', () => {
            const width = img.naturalWidth || 0
            const height = img.naturalHeight || 0
            if (!width || !height) {
              return
            }
            item.width = width
            item.height = height
            sizeMap.set(item.url, { width, height })
            if (minPixels && Math.max(width, height) < minPixels) {
              card.remove()
              updateCounts()
            }
          })
        }
        img.src = src
        pic.appendChild(img)
      } else {
        pic.textContent = item.kind === 'video' ? '▶' : '□'
      }

      pic.title = item.kind === 'video' ? t('clickPlay', '点击播放') : t('clickZoom', '点击放大查看')
      pic.addEventListener('click', () => openViewer(item))

      const button = document.createElement('button')
      button.className = 'dl'
      button.dataset.dl = item.id
      button.addEventListener('click', (event) => {
        event.stopPropagation()
        statusMap.set(item.id, 'pending')
        updateRow(item.id)
        queue([item])
      })
      pic.appendChild(button)

      const cap = document.createElement('div')
      cap.className = 'cap'
      const name = document.createElement('span')
      name.textContent = item.name
      name.title = item.url
      const info = document.createElement('small')
      try {
        info.textContent = new URL(item.url).host
      } catch (err) {
        info.textContent = item.url.slice(0, 60)
      }
      cap.append(name, info)

      card.append(pic, cap)
      masonryEl.appendChild(card)
      updateRow(item.id)
    }
  }

  function refreshCountsOnly() {
    updateCounts()
  }

  // ---------- 下载 ----------

  function queue(list, successText) {
    if (!list.length) {
      showToast(t('noMedia', '没有可下载的内容'))
      return
    }
    const payload = list.map((item) => ({
      clientId: item.id,
      url: item.url,
      name: item.name,
      dir: item.dir || hostDir(),
      html: item.html,
      referer: location.href,
    }))
    for (const item of list) {
      statusMap.set(item.id, 'pending')
      updateRow(item.id)
    }
    sendMessage({ type: 'ow-download', items: payload }, (response) => {
      if (!response || !response.queued) {
        for (const item of list) {
          statusMap.set(item.id, 'error')
          updateRow(item.id)
        }
        showToast(t('sendFailed', '发送失败，请确认 OpenWallpaper 已启动'))
      } else {
        showToast(
          successText || tf('queued', '已加入队列 {count} 个文件', { count: response.queued }),
        )
      }
    })
  }

  function downloadOne(url, kind) {
    const id = hash(url)
    const item = {
      id,
      url,
      kind,
      name: nameFromURL(url, kind),
    }
    queue([item])
  }

  function pageItem() {
    const dir = `${hostDir()}-${stamp()}`
    return {
      id: `page-${stamp()}-${Math.random().toString(36).slice(2, 6)}`,
      name: 'index.html',
      dir,
      html: `<!DOCTYPE html>\n${document.documentElement.outerHTML}`,
    }
  }

  // 全部：当前页所有图片（排除小图标）+ 所有视频 + 整个网页 HTML。
  function downloadAll() {
    collect()
    const list = [...current.image, ...current.video].filter(
      (item) => statusMap.get(item.id) !== 'saved' && passMinPixels(item),
    )
    const page = pageItem()
    queue(
      [...list, page],
      tf('batchSaved', '已开始保存 {count} 个媒体文件 + 整个网页 HTML', { count: list.length }),
    )
  }

  function downloadPage() {
    queue([pageItem()], t('savingPage', '正在保存整个网页的 HTML'))
  }

  // ---------- 连接状态 ----------

  function ping() {
    sendMessage({ type: 'ow-ping' }, (response) => {
      connected = Boolean(response && response.connected)
      dotEl.classList.toggle('on', connected)
      statusEl.textContent = connected
        ? t('connected', '已连接')
        : t('disconnected', '未连接，请启动应用')
    })
  }

  function startPing() {
    ping()
    clearInterval(pingTimer)
    pingTimer = setInterval(ping, 5000)
  }

  function stopPing() {
    clearInterval(pingTimer)
    pingTimer = 0
  }

  // ---------- 面板 ----------

  function setTab(tab) {
    activeTab = tab === 'video' ? 'video' : 'image'
    for (const button of root.querySelectorAll('.tabs button')) {
      button.classList.toggle('active', button.dataset.tab === activeTab)
    }
    render()
  }

  function openPanel() {
    panelOpen = true
    // 每次打开都回到屏幕中央，之后再拖动标题栏自由移动。
    panelEl.style.transform = ''
    panelEl.style.left = ''
    panelEl.style.top = ''
    panelEl.classList.add('open')
    try {
      window.postMessage({ source: 'openwallpaper-scan' }, '*')
    } catch (err) {
      /* ignore */
    }
    render()
    startPing()
  }

  function closePanel() {
    panelOpen = false
    panelEl.classList.remove('open')
    stopPing()
  }

  function togglePanel() {
    if (panelOpen) {
      closePanel()
    } else {
      openPanel()
    }
  }

  root.addEventListener('click', (event) => {
    const target = event.target
    if (!(target instanceof Element)) {
      return
    }
    const action = target.closest('[data-action]')
    if (action) {
      const name = action.getAttribute('data-action')
      if (name === 'close') {
        closePanel()
      } else if (name === 'refresh') {
        try {
          window.postMessage({ source: 'openwallpaper-scan' }, '*')
        } catch (err) {
          /* ignore */
        }
        render()
        showToast(t('listRefreshed', '列表已刷新'))
      } else if (name === 'all') {
        downloadAll()
      } else if (name === 'page') {
        downloadPage()
      }
      return
    }
    const tab = target.closest('[data-tab]')
    if (tab) {
      setTab(tab.getAttribute('data-tab'))
    }
  })

  // ---------- 悬浮按钮 / 弹窗拖拽 ----------

  function clamp(value, min, max) {
    return Math.min(Math.max(value, min), Math.max(min, max))
  }

  function fabSize() {
    const rect = fabEl.getBoundingClientRect()
    return { width: rect.width || 44, height: rect.height || 44 }
  }

  // 拖动位置以视口比例保存，窗口尺寸变化后仍保持相对位置。
  function positionFab(xRatio, yRatio) {
    const size = fabSize()
    const left = clamp(xRatio * window.innerWidth, 8, window.innerWidth - size.width - 8)
    const top = clamp(yRatio * window.innerHeight, 8, window.innerHeight - size.height - 8)
    fabEl.style.left = `${left}px`
    fabEl.style.top = `${top}px`
  }

  try {
    chrome.storage.local.get(['owFabPos', 'owFabRatio', 'owMinPixels'], (result) => {
      const savedMin = Number(result && result.owMinPixels)
      if (Number.isFinite(savedMin) && savedMin > 0) {
        minPixels = savedMin
        minResEl.value = String(savedMin)
      }
      const pos = result && result.owFabPos
      if (pos && Number.isFinite(pos.x) && Number.isFinite(pos.y)) {
        positionFab(pos.x, pos.y)
        return
      }
      const legacy = Number(result && result.owFabRatio)
      positionFab(0.96, Number.isFinite(legacy) && legacy > 0 ? legacy : 0.42)
    })
  } catch (err) {
    positionFab(0.96, 0.42)
  }

  minResEl.addEventListener('change', () => {
    minPixels = Number(minResEl.value) || 0
    try {
      chrome.storage.local.set({ owMinPixels: minPixels })
    } catch (err) {
      /* 存储不可用时仅影响持久化 */
    }
    render()
    scheduleBadges()
  })

  fabEl.addEventListener('pointerdown', (event) => {
    dragged = false
    dragStartX = event.clientX
    dragStartY = event.clientY
    const rect = fabEl.getBoundingClientRect()
    dragStartLeft = rect.left
    dragStartTop = rect.top
    try {
      fabEl.setPointerCapture(event.pointerId)
    } catch (err) {
      /* ignore */
    }
  })

  fabEl.addEventListener('pointermove', (event) => {
    if (!fabEl.hasPointerCapture || !fabEl.hasPointerCapture(event.pointerId)) {
      return
    }
    const dx = event.clientX - dragStartX
    const dy = event.clientY - dragStartY
    if (!dragged && (Math.abs(dx) > 4 || Math.abs(dy) > 4)) {
      dragged = true
    }
    if (!dragged) {
      return
    }
    const size = fabSize()
    const left = clamp(dragStartLeft + dx, 0, window.innerWidth - size.width)
    const top = clamp(dragStartTop + dy, 0, window.innerHeight - size.height)
    fabEl.style.left = `${left}px`
    fabEl.style.top = `${top}px`
  })

  fabEl.addEventListener('pointerup', (event) => {
    try {
      fabEl.releasePointerCapture(event.pointerId)
    } catch (err) {
      /* ignore */
    }
    if (!dragged) {
      togglePanel()
      return
    }
    const rect = fabEl.getBoundingClientRect()
    try {
      chrome.storage.local.set({
        owFabPos: {
          x: rect.left / window.innerWidth,
          y: rect.top / window.innerHeight,
        },
      })
    } catch (err) {
      /* ignore */
    }
  })

  window.addEventListener('resize', () => {
    const rect = fabEl.getBoundingClientRect()
    if (!rect.width) {
      return
    }
    positionFab(rect.left / window.innerWidth, rect.top / window.innerHeight)
  })

  // 面板默认居中显示，拖动标题栏可自由移动。
  const panelHeaderEl = root.querySelector('.panel header')

  function placePanel(left, top) {
    const rect = panelEl.getBoundingClientRect()
    panelEl.style.transform = 'none'
    panelEl.style.left = `${clamp(left, 0, window.innerWidth - rect.width)}px`
    panelEl.style.top = `${clamp(top, 0, window.innerHeight - rect.height)}px`
  }

  panelHeaderEl.addEventListener('pointerdown', (event) => {
    if (event.target.closest('button')) {
      return
    }
    const rect = panelEl.getBoundingClientRect()
    dragStartX = event.clientX
    dragStartY = event.clientY
    dragStartLeft = rect.left
    dragStartTop = rect.top
    panelHeaderEl.classList.add('dragging')
    try {
      panelHeaderEl.setPointerCapture(event.pointerId)
    } catch (err) {
      /* ignore */
    }
  })

  panelHeaderEl.addEventListener('pointermove', (event) => {
    if (!panelHeaderEl.hasPointerCapture || !panelHeaderEl.hasPointerCapture(event.pointerId)) {
      return
    }
    placePanel(
      dragStartLeft + (event.clientX - dragStartX),
      dragStartTop + (event.clientY - dragStartY),
    )
  })

  panelHeaderEl.addEventListener('pointerup', (event) => {
    try {
      panelHeaderEl.releasePointerCapture(event.pointerId)
    } catch (err) {
      /* ignore */
    }
    panelHeaderEl.classList.remove('dragging')
  })

  // ---------- 图片角标 ----------
  // 角标放在独立的 Shadow DOM 浮层里，用 fixed 定位跟随图片，避免被站点样式屏蔽。

  function placeBadge(entry, rect) {
    const { el } = entry
    const visible =
      rect.width > 0 &&
      rect.height > 0 &&
      rect.bottom > 0 &&
      rect.top < window.innerHeight &&
      rect.right > 0 &&
      rect.left < window.innerWidth
    if (!visible) {
      el.style.display = 'none'
      return
    }
    const size = el.offsetWidth || 28
    const left = Math.min(Math.max(4, rect.left + 6), window.innerWidth - size - 4)
    const top = Math.min(Math.max(4, rect.top + 6), window.innerHeight - size - 4)
    el.style.display = 'flex'
    el.style.left = `${left}px`
    el.style.top = `${top}px`
  }

  function positionBadges() {
    for (const entry of badgeMap.values()) {
      if (!entry.img.isConnected) {
        continue
      }
      placeBadge(entry, entry.img.getBoundingClientRect())
    }
  }

  function createBadge(img, rect) {
    const badge = document.createElement('button')
    badge.type = 'button'
    badge.className = 'badge'
    badge.innerHTML = ICON_DOWNLOAD
    badge.title = t('badgeTitle', '缓存到 OpenWallpaper')

    const stop = (event) => {
      event.preventDefault()
      event.stopPropagation()
    }
    badge.addEventListener('mousedown', stop)
    badge.addEventListener('pointerdown', stop)
    badge.addEventListener('click', (event) => {
      stop(event)
      downloadOne(img.currentSrc || img.src, 'image')
    })

    root.appendChild(badge)
    const entry = { el: badge, img }
    badgeMap.set(img, entry)
    placeBadge(entry, rect)
  }

  function updateBadges() {
    const width = window.innerWidth
    const height = window.innerHeight

    for (const img of document.images) {
      if (badgeMap.size >= MAX_BADGES) {
        break
      }
      if (badgeMap.has(img)) {
        continue
      }
      const rect = img.getBoundingClientRect()
      if (rect.width < 64 || rect.height < 64) {
        continue
      }
      if (rect.bottom < -160 || rect.top > height + 160 || rect.right < 0 || rect.left > width) {
        continue
      }
      if (!usable(img.currentSrc || img.src)) {
        continue
      }
      if (minPixels) {
        const edge = Math.max(img.naturalWidth || 0, img.naturalHeight || 0)
        if (edge && edge < minPixels) {
          continue
        }
      }
      createBadge(img, rect)
    }

    for (const [img, entry] of badgeMap) {
      const edge = Math.max(img.naturalWidth || 0, img.naturalHeight || 0)
      const tooSmall = minPixels > 0 && edge > 0 && edge < minPixels
      if (img.isConnected && !tooSmall) {
        continue
      }
      entry.el.remove()
      badgeMap.delete(img)
    }

    positionBadges()
  }

  function scheduleBadges() {
    if (badgeFrame) {
      return
    }
    badgeFrame = requestAnimationFrame(() => {
      badgeFrame = 0
      updateBadges()
    })
  }

  window.addEventListener('scroll', scheduleBadges, true)
  window.addEventListener('resize', scheduleBadges)
  try {
    new MutationObserver(scheduleBadges).observe(document.documentElement, {
      childList: true,
      subtree: true,
    })
  } catch (err) {
    /* ignore */
  }
  // 轮播、动画等场景里图片会在没有事件的情况下移动，定时同步角标位置。
  setInterval(() => {
    if (badgeMap.size) {
      positionBadges()
    }
  }, 250)
  setTimeout(scheduleBadges, 400)

  // ---------- 与注入脚本 / 后台通信 ----------

  window.addEventListener('message', (event) => {
    if (event.source !== window || !event.data || event.data.source !== 'openwallpaper-hook') {
      return
    }
    const urls = Array.isArray(event.data.urls) ? event.data.urls : []
    let added = 0
    for (const raw of urls) {
      const url = absolute(raw)
      if (!url || !usable(url) || captured.has(url)) {
        continue
      }
      const kind = guessKind(url)
      if (!kind) {
        continue
      }
      captured.set(url, kind)
      added += 1
    }
    if (!added) {
      return
    }
    if (panelOpen) {
      render()
    } else {
      collect()
      refreshCountsOnly()
    }
    scheduleBadges()
  })

  try {
    chrome.runtime.onMessage.addListener((message) => {
      if (!message || typeof message !== 'object') {
        return
      }
      if (message.type === 'ow-toggle-panel') {
        togglePanel()
        return
      }
      if (message.type === 'ow-result') {
        const state = message.ok ? 'saved' : 'error'
        statusMap.set(message.id, state)
        updateRow(message.id)
        if (message.ok) {
          showToast(
            tf('savedFile', '已保存 · {name}', {
              name: message.name || t('fileName', '文件'),
            }),
          )
        } else {
          showToast(
            tf('saveFailed', '保存失败 · {error}', {
              error: message.error || t('unknownError', '未知错误'),
            }),
          )
        }
      }
    })
  } catch (err) {
    /* ignore */
  }

  collect()
})()

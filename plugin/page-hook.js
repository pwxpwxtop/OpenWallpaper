// OpenWallpaper 媒体采集 - 页面注入脚本（MAIN world）
// 监听页面 fetch / XHR 返回的 JSON 文本，把其中隐藏的图片、视频 URL 提取出来。
// 很多站点的视频地址只出现在接口返回的 JSON 里，所以必须从网络层拦。

(() => {
  if (window.__openwallpaperHooked) {
    return
  }
  window.__openwallpaperHooked = true

  const MEDIA_RE =
    /https?:\/\/[^\s"'<>\\`]+?\.(?:mp4|m4v|webm|mov|mkv|flv|m3u8|mpd|ts|jpg|jpeg|png|webp|gif|avif|bmp|svg)(?:\?[^\s"'<>\\`]*)?/gi
  const EXT_RE = /\.(mp4|m4v|webm|mov|mkv|flv|m3u8|mpd|ts|jpg|jpeg|png|webp|gif|avif|bmp|svg)(?:$|[?#])/i
  const VIDEO_EXT = new Set(['mp4', 'm4v', 'webm', 'mov', 'mkv', 'flv', 'm3u8', 'mpd', 'ts'])

  const pending = new Set()
  let timer = null

  function kindOf(url) {
    const match = EXT_RE.exec(url)
    if (!match) {
      return ''
    }
    return VIDEO_EXT.has(match[1].toLowerCase()) ? 'video' : 'image'
  }

  function schedule() {
    if (timer) {
      return
    }
    timer = setTimeout(flush, 200)
  }

  function flush() {
    timer = null
    if (!pending.size) {
      return
    }
    const urls = Array.from(pending).slice(0, 300)
    pending.clear()
    try {
      window.postMessage({ source: 'openwallpaper-hook', urls }, '*')
    } catch (err) {
      /* ignore */
    }
  }

  function push(url) {
    if (!url || typeof url !== 'string' || url.length > 4096) {
      return
    }
    if (!/^https?:/i.test(url)) {
      return
    }
    const clean = url.replace(/\\\//g, '/').replace(/\\u0026/gi, '&')
    if (!kindOf(clean)) {
      return
    }
    pending.add(clean)
    schedule()
  }

  function extract(text) {
    if (!text || text.length > 4000000 || text.indexOf('http') === -1) {
      return
    }
    const matches = text.match(MEDIA_RE)
    if (!matches) {
      return
    }
    for (const match of matches) {
      push(match)
    }
  }

  function scan(value, depth) {
    if (depth > 8 || value == null) {
      return
    }
    if (typeof value === 'string') {
      extract(value)
      return
    }
    if (typeof value !== 'object') {
      return
    }
    if (Array.isArray(value)) {
      const limit = Math.min(value.length, 400)
      for (let i = 0; i < limit; i += 1) {
        scan(value[i], depth + 1)
      }
      return
    }
    for (const key of Object.keys(value)) {
      if (key.length > 200) {
        continue
      }
      scan(value[key], depth + 1)
    }
  }

  function inspectResponse(resp) {
    if (!resp || !resp.ok) {
      return
    }
    const type = (resp.headers.get('content-type') || '').toLowerCase()
    if (type && !/json|text|javascript|xml/.test(type)) {
      return
    }
    const length = Number(resp.headers.get('content-length') || 0)
    if (length > 4000000) {
      return
    }
    resp
      .clone()
      .text()
      .then((text) => {
        if (!text || text.length > 4000000 || text.indexOf('http') === -1) {
          return
        }
        if (/json/.test(type)) {
          try {
            scan(JSON.parse(text), 0)
            return
          } catch (err) {
            /* 不是合法 JSON，退化成文本扫描 */
          }
        }
        extract(text)
      })
      .catch(() => {})
  }

  const originalFetch = window.fetch
  if (typeof originalFetch === 'function') {
    window.fetch = function patchedFetch(...args) {
      const input = args[0]
      try {
        push(typeof input === 'string' ? input : input && input.url)
      } catch (err) {
        /* ignore */
      }
      const result = originalFetch.apply(this, args)
      result.then(inspectResponse).catch(() => {})
      return result
    }
  }

  const XHR = window.XMLHttpRequest
  if (XHR && XHR.prototype) {
    const originalOpen = XHR.prototype.open
    const originalSend = XHR.prototype.send

    XHR.prototype.open = function patchedOpen(method, url, ...rest) {
      try {
        this.__openwallpaperUrl = String(url)
        push(this.__openwallpaperUrl)
      } catch (err) {
        /* ignore */
      }
      return originalOpen.call(this, method, url, ...rest)
    }

    XHR.prototype.send = function patchedSend(...args) {
      try {
        this.addEventListener('load', () => {
          try {
            push(this.responseURL || '')
            const type = String(this.getResponseHeader('content-type') || '').toLowerCase()
            if (type && !/json|text|javascript|xml/.test(type)) {
              return
            }
            if (this.responseType === 'json') {
              scan(this.response, 0)
              return
            }
            if (this.responseType === '' || this.responseType === 'text') {
              const text = this.responseText
              if (!text) {
                return
              }
              try {
                scan(JSON.parse(text), 0)
              } catch (err) {
                extract(text)
              }
            }
          } catch (err) {
            /* ignore */
          }
        })
        push(this.__openwallpaperUrl || '')
      } catch (err) {
        /* ignore */
      }
      return originalSend.apply(this, args)
    }
  }

  function scanPerformance() {
    try {
      for (const entry of performance.getEntriesByType('resource')) {
        push(entry.name)
      }
    } catch (err) {
      /* ignore */
    }
  }

  window.addEventListener('load', () => {
    scanPerformance()
    setTimeout(scanPerformance, 1500)
    setTimeout(scanPerformance, 4000)
  })

  window.addEventListener('message', (event) => {
    if (event.source !== window || !event.data || event.data.source !== 'openwallpaper-scan') {
      return
    }
    scanPerformance()
  })
})()

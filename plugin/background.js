// OpenWallpaper 媒体采集 - 后台服务
// 负责与本地 OpenWallpaper 客户端的 WebSocket 通信，把媒体文件以二进制分块发送过去。

const WS_URL = 'ws://127.0.0.1:48921/ws'
const CHUNK = 1 << 20 // 1MB
const MAX_BUFFERED = 8 << 20 // 8MB 背压阈值

// 错误提示跟随浏览器语言，使用扩展的 _locales 语言包；缺失时回退到中文默认值。
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

const VIDEO_EXT = ['mp4', 'm4v', 'webm', 'mov', 'mkv', 'flv', 'm3u8', 'mpd', 'ts']
const IMAGE_EXT = ['jpg', 'jpeg', 'png', 'webp', 'gif', 'avif', 'bmp', 'svg']

const MIME_EXT = {
  'image/jpeg': 'jpg',
  'image/png': 'png',
  'image/webp': 'webp',
  'image/gif': 'gif',
  'image/avif': 'avif',
  'image/bmp': 'bmp',
  'image/svg+xml': 'svg',
  'video/mp4': 'mp4',
  'video/webm': 'webm',
  'video/quicktime': 'mov',
  'video/x-matroska': 'mkv',
  'video/x-flv': 'flv',
  'application/vnd.apple.mpegurl': 'm3u8',
}

let socket = null
let connecting = null
let queue = []
let running = false
let counter = 0
const pending = new Map()
const serverInfo = { connected: false, downloadDir: '' }

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

function nextId() {
  counter += 1
  return `ow-${Date.now().toString(36)}-${counter}`
}

function connect() {
  if (socket && socket.readyState === WebSocket.OPEN) {
    return Promise.resolve(socket)
  }
  if (connecting) {
    return connecting
  }
  connecting = new Promise((resolve, reject) => {
    let ws
    try {
      ws = new WebSocket(WS_URL)
    } catch (err) {
      connecting = null
      reject(err)
      return
    }
    const timer = setTimeout(() => {
      connecting = null
      try {
        ws.close()
      } catch (e) {
        /* ignore */
      }
      reject(new Error(t('bgTimeout', '连接超时，请确认 OpenWallpaper 正在运行')))
    }, 4000)

    ws.onopen = () => {
      clearTimeout(timer)
      socket = ws
      connecting = null
      serverInfo.connected = true
      ws.onmessage = onSocketMessage
      ws.onclose = () => {
        if (socket === ws) {
          socket = null
        }
        serverInfo.connected = false
        failPending(new Error(t('bgDisconnected', '与 OpenWallpaper 的连接已断开')))
      }
      resolve(ws)
    }
    ws.onerror = () => {
      clearTimeout(timer)
      connecting = null
      reject(new Error(t('bgCannotConnect', '无法连接 OpenWallpaper 本地服务，请先启动应用')))
    }
  })
  return connecting
}

function onSocketMessage(event) {
  if (typeof event.data !== 'string') {
    return
  }
  let msg
  try {
    msg = JSON.parse(event.data)
  } catch (err) {
    return
  }
  if (msg && msg.type === 'hello') {
    serverInfo.connected = true
    serverInfo.downloadDir = msg.downloadDir || ''
    return
  }
  if (msg && msg.id && pending.has(msg.id)) {
    const resolve = pending.get(msg.id)
    pending.delete(msg.id)
    resolve(msg)
  }
}

function failPending(err) {
  for (const [id, resolve] of pending) {
    pending.delete(id)
    resolve({ ok: false, error: err.message })
  }
}

function waitAck(id, timeout = 180000) {
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      pending.delete(id)
      resolve({ ok: false, error: t('bgWriteTimeout', '等待客户端写入超时') })
    }, timeout)
    pending.set(id, (msg) => {
      clearTimeout(timer)
      resolve(msg)
    })
  })
}

async function sendBytes(ws, bytes) {
  for (let offset = 0; offset < bytes.byteLength; offset += CHUNK) {
    const slice = bytes.subarray(offset, Math.min(offset + CHUNK, bytes.byteLength))
    while (ws.readyState === WebSocket.OPEN && ws.bufferedAmount > MAX_BUFFERED) {
      await sleep(30)
    }
    if (ws.readyState !== WebSocket.OPEN) {
      throw new Error(t('bgConnectionLost', '连接已断开'))
    }
    ws.send(slice)
  }
}

async function pump(ws, reader) {
  for (;;) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    if (!value || !value.byteLength) {
      continue
    }
    await sendBytes(ws, value)
  }
}

function extensionOf(url) {
  try {
    const path = new URL(url).pathname
    const match = /\.([a-z0-9]{1,8})$/i.exec(path)
    return match ? match[1].toLowerCase() : ''
  } catch (err) {
    return ''
  }
}

function nameFromURL(url, fallback = 'media') {
  try {
    const parsed = new URL(url)
    const segments = parsed.pathname.split('/').filter(Boolean)
    let name = segments.length ? decodeURIComponent(segments[segments.length - 1]) : ''
    name = name.replace(/[\\/:*?"<>|]/g, '-').trim()
    if (!name) {
      return `${fallback}-${Date.now().toString(36)}`
    }
    if (!/\.[a-z0-9]{1,8}$/i.test(name)) {
      const ext = extensionOf(url)
      if (ext) {
        name = `${name}.${ext}`
      }
    }
    return name.slice(0, 120)
  } catch (err) {
    return `${fallback}-${Date.now().toString(36)}`
  }
}

function nameFromResponse(url, contentType) {
  const name = nameFromURL(url, 'media')
  if (/\.[a-z0-9]{1,8}$/i.test(name) && !/^media-/.test(name)) {
    return name
  }
  const base = (contentType || '').split(';')[0].trim().toLowerCase()
  const ext = MIME_EXT[base]
  return ext ? `${name.replace(/\.[a-z0-9]{1,8}$/i, '')}.${ext}` : name
}

async function applyReferer(url, referer) {
  if (!referer) {
    return 0
  }
  let host = ''
  try {
    host = new URL(url).hostname
  } catch (err) {
    return 0
  }
  if (!host) {
    return 0
  }
  const id = Math.floor(Math.random() * 1e9) + 1
  try {
    await chrome.declarativeNetRequest.updateSessionRules({
      removeRuleIds: [id],
      addRules: [
        {
          id,
          priority: 1,
          action: {
            type: 'modifyHeaders',
            requestHeaders: [{ header: 'Referer', operation: 'set', value: referer }],
          },
          condition: {
            urlFilter: host,
            resourceTypes: ['xmlhttprequest', 'media', 'image', 'other'],
          },
        },
      ],
    })
    return id
  } catch (err) {
    return 0
  }
}

async function clearReferer(id) {
  if (!id) {
    return
  }
  try {
    await chrome.declarativeNetRequest.updateSessionRules({ removeRuleIds: [id] })
  } catch (err) {
    /* ignore */
  }
}

function report(task, ack) {
  if (!task.tabId || !task.clientId) {
    return
  }
  try {
    chrome.tabs
      .sendMessage(task.tabId, {
        type: 'ow-result',
        id: task.clientId,
        ok: Boolean(ack && ack.ok),
        path: (ack && ack.path) || '',
        name: (ack && ack.name) || task.name || '',
        error: (ack && ack.error) || '',
      })
      .catch(() => {})
  } catch (err) {
    /* tab 已关闭 */
  }
}

async function finish(ws, id) {
  const ack = waitAck(id)
  try {
    ws.send(JSON.stringify({ type: 'eof', id }))
  } catch (err) {
    return { ok: false, error: err.message }
  }
  return ack
}

async function runTask(task) {
  const ws = await connect()
  const id = task.wsId
  const dir = task.dir || ''

  if (typeof task.html === 'string') {
    const bytes = new TextEncoder().encode(task.html)
    ws.send(
      JSON.stringify({
        type: 'file',
        id,
        name: task.name || 'index.html',
        dir,
        size: bytes.byteLength,
      }),
    )
    await sendBytes(ws, bytes)
    report(task, await finish(ws, id))
    return
  }

  const ruleId = await applyReferer(task.url, task.referer)
  try {
    const resp = await fetch(task.url, { credentials: 'include', cache: 'no-store' })
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}`)
    }
    const name = task.name || nameFromResponse(task.url, resp.headers.get('content-type'))
    const size = Number(resp.headers.get('content-length') || 0)
    ws.send(JSON.stringify({ type: 'file', id, name, dir, size, url: task.url }))
    if (resp.body && resp.body.getReader) {
      await pump(ws, resp.body.getReader())
    } else {
      await sendBytes(ws, new Uint8Array(await resp.arrayBuffer()))
    }
    report(task, await finish(ws, id))
  } finally {
    await clearReferer(ruleId)
  }
}

async function processQueue() {
  if (running) {
    return
  }
  running = true
  try {
    while (queue.length) {
      const task = queue.shift()
      try {
        await runTask(task)
      } catch (err) {
        report(task, { ok: false, error: String((err && err.message) || err) })
      }
    }
  } finally {
    running = false
  }
}

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (!msg || typeof msg !== 'object') {
    return undefined
  }

  if (msg.type === 'ow-download') {
    const tabId = (sender.tab && sender.tab.id) || msg.tabId
    const tasks = (msg.items || []).map((item) => ({
      ...item,
      tabId,
      wsId: nextId(),
    }))
    if (tasks.length) {
      queue.push(...tasks)
      processQueue()
    }
    sendResponse({ queued: tasks.length })
    return undefined
  }

  if (msg.type === 'ow-ping') {
    connect()
      .then(() => sendResponse({ connected: true, downloadDir: serverInfo.downloadDir }))
      .catch(() => sendResponse({ connected: false, downloadDir: '' }))
    return true
  }

  return undefined
})

chrome.action.onClicked.addListener((tab) => {
  if (!tab || !tab.id) {
    return
  }
  chrome.tabs.sendMessage(tab.id, { type: 'ow-toggle-panel' }).catch(() => {})
})

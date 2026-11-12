import { defineStore } from 'pinia'
import { ref } from 'vue'
import { WallpaperService } from '../../bindings/changeme'

export const useSettingsStore = defineStore('settings', () => {
  const theme = ref('dark')
  const folders = ref([])
  const active = ref('')
  const downloadDir = ref('')
  const pluginDir = ref('')
  const pluginPort = ref(0)
  const loading = ref(false)

  let loaded = false

  function applyTheme(value) {
    const next = value === 'light' ? 'light' : 'dark'
    theme.value = next
    document.documentElement.classList.toggle('app-dark', next === 'dark')
  }

  function applySettings(settings) {
    folders.value = settings.folders ?? []
    active.value = settings.active ?? ''
    downloadDir.value = settings.downloadDir ?? ''
    pluginDir.value = settings.pluginDir ?? ''
    pluginPort.value = settings.pluginPort ?? 0
    applyTheme(settings.theme)
  }

  async function load(force = false) {
    if (loaded && !force) {
      return
    }
    loading.value = true
    try {
      const settings = await WallpaperService.GetSettings()
      applySettings(settings)
      loaded = true
    } finally {
      loading.value = false
    }
  }

  async function setTheme(value) {
    applyTheme(value)
    await WallpaperService.SetTheme(theme.value)
  }

  async function addFolder() {
    const path = await WallpaperService.SelectFolder()
    if (!path) {
      return false
    }
    const settings = await WallpaperService.AddFolder(path)
    folders.value = settings.folders ?? []
    return true
  }

  async function removeFolder(path) {
    const settings = await WallpaperService.RemoveFolder(path)
    folders.value = settings.folders ?? []
  }

  async function chooseDownloadDir() {
    const settings = await WallpaperService.SelectDownloadDir()
    applySettings(settings)
    return settings.downloadDir
  }

  async function openDownloadDir() {
    await WallpaperService.OpenDownloadFolder()
  }

  async function openPluginDir() {
    await WallpaperService.OpenPluginFolder()
  }

  async function openUrl(url) {
    await WallpaperService.OpenURL(url)
  }

  async function installExtension() {
    return await WallpaperService.InstallExtension()
  }

  async function launchBrowserExtension(url) {
    return await WallpaperService.LaunchBrowserWithExtension(url)
  }

  return {
    theme,
    folders,
    active,
    downloadDir,
    pluginDir,
    pluginPort,
    loading,
    load,
    setTheme,
    addFolder,
    removeFolder,
    chooseDownloadDir,
    openDownloadDir,
    openPluginDir,
    openUrl,
    installExtension,
    launchBrowserExtension,
  }
})

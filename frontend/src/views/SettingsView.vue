<script setup>
import { computed, onMounted } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useSettingsStore } from '../stores/settings'
import { useWallpapersStore } from '../stores/wallpapers'
import { useI18nStore } from '../stores/i18n'

const settings = useSettingsStore()
const wallpapers = useWallpapersStore()
const i18n = useI18nStore()
const toast = useToast()

const themes = computed(() => [
  {
    value: 'dark',
    label: i18n.t('settings.themeDark'),
    desc: i18n.t('settings.themeDarkDesc'),
    icon: 'pi pi-moon',
  },
  {
    value: 'light',
    label: i18n.t('settings.themeLight'),
    desc: i18n.t('settings.themeLightDesc'),
    icon: 'pi pi-sun',
  },
])

const formats = computed(() => [
  {
    kind: i18n.t('settings.formatImage'),
    icon: 'pi pi-image',
    detail: 'jpg · jpeg · png · webp · bmp · gif',
  },
  {
    kind: i18n.t('settings.formatVideo'),
    icon: 'pi pi-video',
    detail: 'mp4 · webm · mov · mkv · m4v · avi',
  },
  {
    kind: i18n.t('settings.formatHtml'),
    icon: 'pi pi-code',
    detail: i18n.t('settings.formatHtmlDetail'),
  },
])

onMounted(() => settings.load(true))

async function addFolder() {
  try {
    const added = await settings.addFolder()
    if (added) {
      await wallpapers.scan()
      toast.add({ severity: 'success', summary: i18n.t('toast.folderAdded'), life: 2200 })
    }
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.addFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  }
}

async function removeFolder(path) {
  try {
    await settings.removeFolder(path)
    await wallpapers.scan()
    toast.add({ severity: 'success', summary: i18n.t('toast.folderRemoved'), life: 2200 })
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.removeFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  }
}

async function changeTheme(value) {
  if (!value || value === settings.theme) {
    return
  }
  try {
    await settings.setTheme(value)
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.saveFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  }
}

async function stopDynamic() {
  try {
    await wallpapers.stopDynamic()
    toast.add({ severity: 'success', summary: i18n.t('toast.stopped'), life: 2200 })
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.stopFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  }
}

async function chooseDownloadDir() {
  try {
    const path = await settings.chooseDownloadDir()
    if (path) {
      await wallpapers.scan()
      toast.add({ severity: 'success', summary: i18n.t('toast.downloadDirChanged'), life: 2600 })
    }
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.downloadDirFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  }
}

async function openDownloadDir() {
  try {
    await settings.openDownloadDir()
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.openFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  }
}

async function openPluginDir() {
  try {
    await settings.openPluginDir()
    toast.add({ severity: 'success', summary: i18n.t('toast.pluginExported'), life: 2600 })
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.exportFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  }
}
</script>

<template>
  <section class="view">
    <header class="page-head">
      <span class="eyebrow"><i class="pi pi-sliders-h" /> {{ i18n.t('settings.eyebrow') }}</span>
      <h1>{{ i18n.t('settings.title') }}</h1>
      <p>{{ i18n.t('settings.subtitle') }}</p>
    </header>

    <div class="panel-grid">
      <div class="panel span-2">
        <div class="panel-head">
          <span class="panel-icon"><i class="pi pi-folder" /></span>
          <div class="panel-copy">
            <h2>{{ i18n.t('settings.folders') }}</h2>
            <p>{{ i18n.t('settings.foldersDesc') }}</p>
          </div>
          <Button
            :label="i18n.t('gallery.addFolder')"
            icon="pi pi-plus"
            class="brand-btn"
            @click="addFolder"
          />
        </div>

        <div v-if="!settings.folders.length" class="folder-empty">
          <i class="pi pi-folder-open" />
          <span>{{ i18n.t('settings.foldersEmpty') }}</span>
        </div>

        <ul v-else class="folder-list">
          <li v-for="folder in settings.folders" :key="folder.path" class="folder-item">
            <span class="folder-icon"><i class="pi pi-folder" /></span>
            <div class="folder-info">
              <span class="folder-path" :title="folder.path">{{ folder.path }}</span>
              <Tag
                :value="folder.exists ? i18n.t('settings.available') : i18n.t('settings.missing')"
                :severity="folder.exists ? 'success' : 'danger'"
              />
            </div>
            <Button
              icon="pi pi-trash"
              severity="danger"
              text
              rounded
              v-tooltip.left="i18n.t('settings.removeFolder')"
              @click="removeFolder(folder.path)"
            />
          </li>
        </ul>
      </div>

      <div class="panel">
        <div class="panel-head">
          <span class="panel-icon"><i class="pi pi-palette" /></span>
          <div class="panel-copy">
            <h2>{{ i18n.t('settings.appearance') }}</h2>
            <p>{{ i18n.t('settings.appearanceDesc') }}</p>
          </div>
        </div>

        <div class="theme-cards">
          <button
            v-for="theme in themes"
            :key="theme.value"
            type="button"
            class="theme-card"
            :class="{ active: settings.theme === theme.value }"
            @click="changeTheme(theme.value)"
          >
            <span class="swatch" :class="theme.value"><i :class="theme.icon" /></span>
            <span class="theme-copy">
              <strong>{{ theme.label }}</strong>
              <small>{{ theme.desc }}</small>
            </span>
            <i v-if="settings.theme === theme.value" class="pi pi-check-circle check" />
          </button>
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <span class="panel-icon"><i class="pi pi-desktop" /></span>
          <div class="panel-copy">
            <h2>{{ i18n.t('settings.dynamic') }}</h2>
            <p>{{ i18n.t('settings.dynamicDesc') }}</p>
          </div>
        </div>

        <div v-if="wallpapers.appliedPath" class="active-wallpaper">
          <i class="pi pi-play-circle" />
          <span :title="wallpapers.appliedPath">{{ wallpapers.appliedPath }}</span>
        </div>
        <p v-else class="hint">{{ i18n.t('settings.dynamicNone') }}</p>

        <Button
          :label="i18n.t('settings.dynamicStop')"
          icon="pi pi-stop-circle"
          class="ghost-btn stop-btn"
          outlined
          @click="stopDynamic"
        />
      </div>

      <div class="panel">
        <div class="panel-head">
          <span class="panel-icon"><i class="pi pi-cloud-download" /></span>
          <div class="panel-copy">
            <h2>{{ i18n.t('settings.download') }}</h2>
            <p>{{ i18n.t('settings.downloadDesc') }}</p>
          </div>
        </div>

        <div class="folder-item">
          <span class="folder-icon"><i class="pi pi-folder" /></span>
          <div class="folder-info">
            <span class="folder-path" :title="settings.downloadDir">{{ settings.downloadDir }}</span>
          </div>
        </div>

        <div class="btn-row">
          <Button
            :label="i18n.t('settings.changeFolder')"
            icon="pi pi-folder-open"
            class="ghost-btn"
            outlined
            @click="chooseDownloadDir"
          />
          <Button
            :label="i18n.t('settings.openFolder')"
            icon="pi pi-external-link"
            class="ghost-btn"
            outlined
            @click="openDownloadDir"
          />
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <span class="panel-icon"><i class="pi pi-globe" /></span>
          <div class="panel-copy">
            <h2>{{ i18n.t('settings.plugin') }}</h2>
            <p>{{ i18n.t('settings.pluginDesc') }}</p>
          </div>
        </div>

        <div class="ws-box">
          <span>{{ i18n.t('settings.pluginService') }}</span>
          <code>ws://127.0.0.1:{{ settings.pluginPort || 48921 }}/ws</code>
        </div>

        <p class="hint">
          {{ i18n.t('settings.pluginHint') }}
        </p>

        <div class="btn-row">
          <Button
            :label="i18n.t('settings.pluginExport')"
            icon="pi pi-upload"
            class="brand-btn"
            @click="openPluginDir"
          />
        </div>
      </div>

      <div class="panel span-2">
        <div class="panel-head">
          <span class="panel-icon"><i class="pi pi-verified" /></span>
          <div class="panel-copy">
            <h2>{{ i18n.t('settings.formats') }}</h2>
            <p>{{ i18n.t('settings.formatsDesc') }}</p>
          </div>
        </div>

        <div class="format-grid">
          <div v-for="format in formats" :key="format.kind" class="format-tile">
            <span class="format-icon"><i :class="format.icon" /></span>
            <div>
              <strong>{{ format.kind }}</strong>
              <span>{{ format.detail }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: 22px;
  max-width: 1040px;
  margin: 0 auto;
}

.page-head {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.page-head h1 {
  font-size: 1.85rem;
  font-weight: 700;
  letter-spacing: -0.025em;
}

.page-head p {
  font-size: 0.84rem;
  color: var(--p-text-muted-color);
}

.panel-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  border: 1px solid var(--app-border);
  border-radius: 18px;
  background: var(--app-panel);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.02) inset;
}

.panel.span-2 {
  grid-column: span 2;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.panel-icon {
  width: 38px;
  height: 38px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(139, 92, 246, 0.1));
  color: var(--p-primary-color);
  font-size: 0.95rem;
}

.panel-copy {
  flex: 1;
  min-width: 0;
}

.panel-copy h2 {
  font-size: 0.98rem;
  font-weight: 600;
}

.panel-copy p {
  margin-top: 3px;
  font-size: 0.78rem;
  color: var(--p-text-muted-color);
}

.panel-head > .p-button {
  flex-shrink: 0;
}

.folder-empty {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 26px;
  border: 1px dashed var(--app-border-strong);
  border-radius: 12px;
  color: var(--p-text-muted-color);
  font-size: 0.82rem;
}

.folder-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.folder-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 14px;
  border: 1px solid var(--app-border);
  border-radius: 12px;
  background: var(--app-panel-2);
  transition: border-color 0.16s ease;
}

.folder-item:hover {
  border-color: rgba(129, 140, 248, 0.4);
}

.folder-icon {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 9px;
  background: var(--app-highlight);
  color: var(--p-primary-color);
  font-size: 0.85rem;
}

.folder-info {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.folder-path {
  font-size: 0.8rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.theme-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.theme-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: 13px;
  background: var(--app-panel-2);
  color: var(--p-text-color);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease;
}

.theme-card:hover {
  border-color: rgba(129, 140, 248, 0.45);
}

.theme-card.active {
  border-color: rgba(99, 102, 241, 0.65);
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.14), rgba(139, 92, 246, 0.06));
}

.swatch {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 10px;
  font-size: 0.85rem;
}

.swatch.dark {
  background: linear-gradient(135deg, #1e1b4b, #312e81);
  color: #c7d2fe;
}

.swatch.light {
  background: linear-gradient(135deg, #fef3c7, #fde68a);
  color: #b45309;
}

.theme-copy {
  flex: 1;
  display: flex;
  flex-direction: column;
  line-height: 1.35;
}

.theme-copy strong {
  font-size: 0.84rem;
}

.theme-copy small {
  font-size: 0.72rem;
  color: var(--p-text-muted-color);
}

.check {
  color: var(--p-primary-color);
}

.active-wallpaper {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid rgba(99, 102, 241, 0.32);
  border-radius: 11px;
  background: var(--app-highlight);
  font-size: 0.76rem;
  color: var(--app-nav-active-text);
  overflow: hidden;
}

.active-wallpaper i {
  flex-shrink: 0;
}

.active-wallpaper span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hint {
  font-size: 0.78rem;
  color: var(--p-text-muted-color);
}

.stop-btn {
  align-self: flex-start;
}

.btn-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: auto;
}

.btn-row > .p-button {
  flex: 1;
  min-width: 120px;
}

.ws-box {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 11px 13px;
  border: 1px solid var(--app-border);
  border-radius: 12px;
  background: var(--app-panel-2);
}

.ws-box span {
  font-size: 0.7rem;
  color: var(--p-text-muted-color);
}

.ws-box code {
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 0.78rem;
  color: var(--p-primary-color);
  word-break: break-all;
}

.format-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.format-tile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 13px;
  background: var(--app-panel-2);
}

.format-icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: var(--app-highlight);
  color: var(--p-primary-color);
  font-size: 0.9rem;
}

.format-tile div {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.format-tile strong {
  font-size: 0.82rem;
}

.format-tile span {
  font-size: 0.7rem;
  color: var(--p-text-muted-color);
}

@media (max-width: 900px) {
  .panel-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .panel.span-2 {
    grid-column: span 1;
  }

  .format-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

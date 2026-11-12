<script setup>
import { computed, onMounted, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { useWallpapersStore } from '../stores/wallpapers'
import { useSettingsStore } from '../stores/settings'
import { useI18nStore } from '../stores/i18n'
import WallpaperCard from '../components/WallpaperCard.vue'
import PreviewDialog from '../components/PreviewDialog.vue'
import FilterDialog from '../components/FilterDialog.vue'
import TagDialog from '../components/TagDialog.vue'

const wallpapers = useWallpapersStore()
const settings = useSettingsStore()
const i18n = useI18nStore()
const toast = useToast()
const confirm = useConfirm()

const previewVisible = ref(false)
const previewItem = ref(null)
const filterVisible = ref(false)
const tagVisible = ref(false)
const tagItem = ref(null)

const tabs = computed(() => [
  { label: i18n.t('gallery.tabAll'), value: 'all', icon: 'pi pi-th-large', count: wallpapers.counts.all },
  { label: i18n.t('kind.image'), value: 'image', icon: 'pi pi-image', count: wallpapers.counts.image },
  { label: i18n.t('kind.video'), value: 'video', icon: 'pi pi-video', count: wallpapers.counts.video },
  { label: i18n.t('kind.html'), value: 'html', icon: 'pi pi-code', count: wallpapers.counts.html },
])

onMounted(async () => {
  await settings.load()
  await wallpapers.scan()
})

function openPreview(item) {
  previewItem.value = item
  previewVisible.value = true
}

function openTagDialog(item) {
  tagItem.value = item
  tagVisible.value = true
}

function tagged() {
  toast.add({ severity: 'success', summary: i18n.t('tag.saved'), life: 2200 })
}

async function applyWallpaper(item) {
  try {
    await wallpapers.apply(item)
    toast.add({
      severity: 'success',
      summary: i18n.t('toast.applied'),
      detail: i18n.t('toast.appliedDesc', { name: item.name }),
      life: 2600,
    })
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.applyFailed'),
      detail: String(err?.message ?? err),
      life: 4200,
    })
  }
}

function removeWallpaper(item) {
  const isFolderHtml = item.kind === 'html' && item.path.toLowerCase().endsWith('index.html')
  confirm.require({
    header: i18n.t('confirm.deleteTitle'),
    message: isFolderHtml
      ? i18n.t('confirm.deleteFolder', { name: item.name })
      : i18n.t('confirm.deleteFile', { name: item.name }),
    icon: 'pi pi-exclamation-triangle',
    rejectProps: { label: i18n.t('common.cancel'), severity: 'secondary', outlined: true },
    acceptProps: { label: i18n.t('common.delete'), severity: 'danger' },
    accept: async () => {
      try {
        await wallpapers.remove(item)
        toast.add({
          severity: 'success',
          summary: i18n.t('toast.deleted'),
          detail: item.name,
          life: 2400,
        })
      } catch (err) {
        toast.add({
          severity: 'error',
          summary: i18n.t('toast.deleteFailed'),
          detail: String(err?.message ?? err),
          life: 4200,
        })
      }
    },
  })
}

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
      life: 4200,
    })
  }
}

async function refresh() {
  await wallpapers.scan()
}
</script>

<template>
  <section class="view">
    <header class="page-head">
      <div class="head-copy">
        <span class="eyebrow"><i class="pi pi-sparkles" /> WALLPAPER LIBRARY</span>
        <h1>{{ i18n.t('gallery.title') }}</h1>
        <p>{{ i18n.t('gallery.subtitle') }}</p>
      </div>
      <div class="head-actions">
        <Button
          :label="i18n.t('gallery.addFolder')"
          icon="pi pi-plus"
          class="brand-btn"
          @click="addFolder"
        />
      </div>
    </header>

    <div class="summary-row">
      <span class="chip">
        <i class="pi pi-folder" />
        {{ i18n.t('gallery.folderCount', { count: settings.folders.length }) }}
      </span>
      <span class="chip">
        <i class="pi pi-images" />
        {{ i18n.t('gallery.wallpaperCount', { count: wallpapers.counts.all }) }}
      </span>
      <span v-if="wallpapers.appliedPath" class="chip live">
        <i class="pi pi-desktop" /> {{ i18n.t('gallery.dynamicRunning') }}
      </span>
    </div>

    <div class="toolbar">
      <div class="segmented">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          class="seg"
          :class="{ active: wallpapers.filter === tab.value }"
          @click="wallpapers.filter = tab.value"
        >
          <i :class="tab.icon" />
          <span>{{ tab.label }}</span>
          <em>{{ tab.count }}</em>
        </button>
      </div>

      <div class="toolbar-right">
        <div class="search">
          <i class="pi pi-search" />
          <InputText
            v-model="wallpapers.search"
            :placeholder="i18n.t('gallery.search')"
            class="search-input"
          />
          <button
            v-if="wallpapers.search"
            type="button"
            class="search-clear"
            @click="wallpapers.search = ''"
          >
            <i class="pi pi-times" />
          </button>
        </div>
        <button
          type="button"
          class="filter-btn"
          v-tooltip.bottom="i18n.t('gallery.refresh')"
          @click="refresh"
        >
          <i class="pi pi-refresh" :class="{ 'pi-spin': wallpapers.loading }" />
        </button>
        <button
          type="button"
          class="filter-btn"
          :class="{ active: wallpapers.activeFilterCount > 0 }"
          v-tooltip.bottom="i18n.t('gallery.filter')"
          @click="filterVisible = true"
        >
          <i class="pi pi-filter" />
          <em v-if="wallpapers.activeFilterCount">{{ wallpapers.activeFilterCount }}</em>
        </button>
      </div>
    </div>

    <div v-if="wallpapers.error" class="error-banner">
      <i class="pi pi-exclamation-triangle" />
      <span>{{ wallpapers.error }}</span>
      <Button :label="i18n.t('gallery.retry')" size="small" text @click="refresh" />
    </div>

    <div v-if="wallpapers.loading" class="grid">
      <div v-for="n in 8" :key="n" class="skeleton" />
    </div>

    <div v-else-if="!settings.folders.length" class="empty">
      <span class="empty-icon"><i class="pi pi-folder-open" /></span>
      <h2>{{ i18n.t('gallery.emptyFoldersTitle') }}</h2>
      <p>{{ i18n.t('gallery.emptyFoldersDesc') }}</p>
      <Button
        :label="i18n.t('gallery.chooseFolder')"
        icon="pi pi-plus"
        class="brand-btn"
        @click="addFolder"
      />
    </div>

    <div v-else-if="!wallpapers.filtered.length" class="empty">
      <span class="empty-icon"><i class="pi pi-inbox" /></span>
      <h2>{{ i18n.t('gallery.emptyMatchTitle') }}</h2>
      <p>{{ i18n.t('gallery.emptyMatchDesc') }}</p>
      <Button
        :label="i18n.t('gallery.rescan')"
        icon="pi pi-refresh"
        class="ghost-btn"
        outlined
        @click="refresh"
      />
    </div>

    <div v-else class="grid">
      <WallpaperCard
        v-for="item in wallpapers.filtered"
        :key="item.id"
        :item="item"
        :applied="wallpapers.appliedPath === item.path"
        :applying="wallpapers.applyingId === item.id"
        @preview="openPreview"
        @apply="applyWallpaper"
        @tag="openTagDialog"
        @remove="removeWallpaper"
      />
    </div>

    <PreviewDialog
      v-model:visible="previewVisible"
      :item="previewItem"
      @apply="applyWallpaper"
    />

    <FilterDialog v-model:visible="filterVisible" />

    <TagDialog v-model:visible="tagVisible" :item="tagItem" @saved="tagged" />
  </section>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: 22px;
  max-width: 1440px;
  margin: 0 auto;
}

.page-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
}

.head-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.head-copy h1 {
  font-size: 1.85rem;
  font-weight: 700;
  letter-spacing: -0.025em;
  line-height: 1.15;
}

.head-copy p {
  font-size: 0.84rem;
  color: var(--p-text-muted-color);
}

.head-actions {
  display: flex;
  gap: 10px;
}

.summary-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 6px 12px;
  border: 1px solid var(--app-border);
  border-radius: 999px;
  background: var(--app-panel);
  font-size: 0.74rem;
  color: var(--p-text-muted-color);
}

.chip i {
  font-size: 0.72rem;
  color: var(--p-primary-color);
}

.chip.live {
  border-color: rgba(99, 102, 241, 0.42);
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.16), rgba(139, 92, 246, 0.1));
  color: var(--app-nav-active-text);
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
}

.segmented {
  display: inline-flex;
  gap: 4px;
  padding: 5px;
  border: 1px solid var(--app-border);
  border-radius: 14px;
  background: var(--app-panel);
}

.seg {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 13px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--p-text-muted-color);
  font-size: 0.78rem;
  cursor: pointer;
  transition: background 0.16s ease, color 0.16s ease;
}

.seg:hover {
  color: var(--p-text-color);
  background: var(--app-highlight);
}

.seg.active {
  color: #fff;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  box-shadow: 0 8px 18px rgba(99, 102, 241, 0.32);
}

.seg em {
  font-style: normal;
  font-size: 0.65rem;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--app-highlight);
}

.seg.active em {
  background: rgba(255, 255, 255, 0.2);
}

.search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  min-width: 300px;
  padding: 0 12px;
  border: 1px solid var(--app-border);
  border-radius: 13px;
  background: var(--app-panel);
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-btn {
  position: relative;
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border: 1px solid var(--app-border);
  border-radius: 13px;
  background: var(--app-panel);
  color: var(--p-text-muted-color);
  font-size: 0.9rem;
  cursor: pointer;
  transition: border-color 0.16s ease, color 0.16s ease, background 0.16s ease;
}

.filter-btn:hover {
  border-color: rgba(129, 140, 248, 0.5);
  color: var(--p-primary-color);
}

.filter-btn.active {
  border-color: rgba(99, 102, 241, 0.65);
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.16), rgba(139, 92, 246, 0.06));
  color: var(--p-primary-color);
}

.filter-btn em {
  position: absolute;
  top: -6px;
  right: -6px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
  font-size: 0.62rem;
  font-style: normal;
  font-weight: 600;
  display: grid;
  place-items: center;
}

.search:focus-within {
  border-color: var(--p-primary-color);
  box-shadow: 0 0 0 3px var(--app-highlight);
}

.search > i {
  font-size: 0.8rem;
  color: var(--p-text-muted-color);
}

.search-input {
  flex: 1;
  padding: 0;
  border: none;
  background: transparent;
  box-shadow: none;
  font-size: 0.82rem;
}

.search-input:enabled:focus {
  border: none;
  box-shadow: none;
}

.search-clear {
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: var(--app-highlight);
  color: var(--p-text-muted-color);
  font-size: 0.6rem;
  cursor: pointer;
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border: 1px solid rgba(239, 68, 68, 0.35);
  border-radius: 12px;
  background: rgba(239, 68, 68, 0.08);
  color: #f87171;
  font-size: 0.82rem;
}

.error-banner span {
  flex: 1;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(270px, 1fr));
  gap: 18px;
  align-content: start;
}

.skeleton {
  aspect-ratio: 16 / 10;
  border-radius: 18px;
  border: 1px solid var(--app-border);
  background: linear-gradient(
    100deg,
    var(--app-panel) 20%,
    var(--app-panel-2) 40%,
    var(--app-panel) 60%
  );
  background-size: 300% 100%;
  animation: shimmer 1.4s ease-in-out infinite;
}

@keyframes shimmer {
  0% {
    background-position: 100% 0;
  }
  100% {
    background-position: -100% 0;
  }
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 72px 24px;
  border: 1px dashed var(--app-border-strong);
  border-radius: 20px;
  background: linear-gradient(180deg, var(--app-panel), transparent);
  text-align: center;
}

.empty-icon {
  width: 58px;
  height: 58px;
  display: grid;
  place-items: center;
  margin-bottom: 4px;
  border-radius: 18px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.22), rgba(139, 92, 246, 0.12));
  color: var(--p-primary-color);
  font-size: 1.5rem;
}

.empty h2 {
  font-size: 1.05rem;
  font-weight: 600;
}

.empty p {
  max-width: 460px;
  font-size: 0.82rem;
  color: var(--p-text-muted-color);
}

.empty > .p-button {
  margin-top: 8px;
}
</style>

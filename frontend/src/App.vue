<script setup>
import { computed, onMounted, ref } from 'vue'
import { useSettingsStore } from './stores/settings'
import { useI18nStore } from './stores/i18n'

const settings = useSettingsStore()
const i18n = useI18nStore()

const langOpen = ref(false)

onMounted(() => settings.load())

const themeLabel = computed(() =>
  settings.theme === 'dark' ? i18n.t('app.themeDark') : i18n.t('app.themeLight'),
)

const currentLocale = computed(
  () =>
    i18n.localeOptions.find((option) => option.code === i18n.locale) ?? {
      code: i18n.locale,
      label: i18n.locale,
    },
)

function selectLocale(code) {
  i18n.setLocale(code)
  langOpen.value = false
}

function toggleTheme() {
  settings.setTheme(settings.theme === 'dark' ? 'light' : 'dark')
}
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark"><i class="pi pi-bolt" /></span>
        <div class="brand-text">
          <strong>OpenWallpaper</strong>
          <small>{{ i18n.t('app.tagline') }}</small>
        </div>
      </div>

      <nav class="nav">
        <span class="nav-label">{{ i18n.t('app.browse') }}</span>
        <RouterLink to="/" class="nav-item" active-class="active" exact-active-class="active">
          <i class="pi pi-th-large" />
          <span>{{ i18n.t('app.library') }}</span>
        </RouterLink>
        <RouterLink to="/recommended" class="nav-item" active-class="active">
          <i class="pi pi-compass" />
          <span>{{ i18n.t('app.recommended') }}</span>
        </RouterLink>
        <RouterLink to="/settings" class="nav-item" active-class="active">
          <i class="pi pi-sliders-h" />
          <span>{{ i18n.t('app.settings') }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <div class="theme-row">
          <span class="theme-label">
            <i :class="settings.theme === 'dark' ? 'pi pi-moon' : 'pi pi-sun'" />
            {{ themeLabel }}
          </span>
          <button type="button" class="theme-switch" @click="toggleTheme">
            <span class="thumb" :class="settings.theme" />
          </button>
        </div>
        <div class="lang-row">
          <span class="theme-label">{{ i18n.t('settings.language') }}</span>
          <button
            type="button"
            class="lang-trigger"
            :title="i18n.t('settings.language')"
            @click="langOpen = true"
          >
            <i class="pi pi-globe" />
            <span>{{ currentLocale.label }}</span>
          </button>
        </div>
        <p class="version">{{ i18n.t('app.footer') }}</p>
      </div>
    </aside>

    <main class="workspace">
      <div class="workspace-inner">
        <RouterView />
      </div>
    </main>

    <Toast position="bottom-right" />
    <ConfirmDialog />

    <Dialog
      v-model:visible="langOpen"
      modal
      :header="i18n.t('settings.language')"
      :style="{ width: '320px' }"
    >
      <div class="lang-options">
        <button
          v-for="option in i18n.localeOptions"
          :key="option.code"
          type="button"
          class="lang-option"
          :class="{ active: i18n.locale === option.code }"
          @click="selectLocale(option.code)"
        >
          <span>{{ option.label }}</span>
          <i v-if="i18n.locale === option.code" class="pi pi-check" />
        </button>
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  height: 100vh;
  background: var(--app-bg);
}

.sidebar {
  position: relative;
  width: 248px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 26px;
  padding: 24px 16px 18px;
  border-right: 1px solid var(--app-border);
  background: linear-gradient(180deg, var(--app-highlight), transparent 42%),
    var(--app-sidebar);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 8px;
}

.brand-mark {
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  border-radius: 13px;
  background: linear-gradient(135deg, #6366f1, #a855f7);
  color: #fff;
  font-size: 1.05rem;
  box-shadow: 0 10px 24px rgba(99, 102, 241, 0.38);
}

.brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}

.brand-text strong {
  font-size: 0.95rem;
  letter-spacing: -0.01em;
}

.brand-text small {
  font-size: 0.7rem;
  color: var(--p-text-muted-color);
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.nav-label {
  padding: 0 12px;
  margin-bottom: 4px;
  font-size: 0.66rem;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--p-text-muted-color);
}

.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 13px;
  border-radius: 12px;
  color: var(--p-text-muted-color);
  text-decoration: none;
  font-size: 0.88rem;
  transition: background 0.16s ease, color 0.16s ease, transform 0.16s ease;
}

.nav-item i {
  font-size: 0.95rem;
}

.nav-item:hover {
  background: var(--app-highlight);
  color: var(--p-text-color);
}

.nav-item.active {
  background: linear-gradient(90deg, var(--app-nav-active-bg), transparent);
  color: var(--app-nav-active-text);
  font-weight: 600;
}

.nav-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 9px;
  bottom: 9px;
  width: 3px;
  border-radius: 3px;
  background: linear-gradient(180deg, #818cf8, #a855f7);
  box-shadow: 0 0 12px rgba(129, 140, 248, 0.7);
}

.sidebar-footer {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 4px 0;
  border-top: 1px solid var(--app-border);
}

.theme-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 8px;
}

.theme-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8rem;
  color: var(--p-text-muted-color);
}

.theme-switch {
  position: relative;
  width: 42px;
  height: 24px;
  padding: 0;
  border: 1px solid var(--app-border-strong);
  border-radius: 999px;
  background: var(--app-panel-2);
  cursor: pointer;
  transition: background 0.18s ease;
}

.theme-switch .thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #a855f7);
  transition: transform 0.18s ease;
}

.theme-switch .thumb.light {
  transform: translateX(18px);
}

.lang-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 8px;
}

.lang-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 128px;
  padding: 5px 10px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-panel-2);
  color: var(--p-text-muted-color);
  font-size: 0.7rem;
  cursor: pointer;
  transition: border-color 0.16s ease, color 0.16s ease;
}

.lang-trigger span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lang-trigger:hover {
  border-color: rgba(129, 140, 248, 0.5);
  color: var(--p-text-color);
}

.lang-options {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 56vh;
  overflow-y: auto;
  padding-right: 2px;
}

.lang-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 14px;
  border: 1px solid var(--app-border);
  border-radius: 11px;
  background: var(--app-panel-2);
  color: var(--p-text-color);
  font-size: 0.84rem;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease;
}

.lang-option:hover {
  border-color: rgba(129, 140, 248, 0.5);
}

.lang-option.active {
  border-color: rgba(99, 102, 241, 0.65);
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.16), rgba(139, 92, 246, 0.08));
  color: var(--app-nav-active-text);
}

.version {
  padding: 0 8px;
  font-size: 0.66rem;
  color: var(--p-text-muted-color);
  opacity: 0.8;
}

.workspace {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  background-image: var(--app-glow);
  background-color: var(--app-bg);
  background-repeat: no-repeat;
}

.workspace-inner {
  padding: 34px 40px 48px;
}
</style>

<script setup>
import { ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useI18nStore } from '../stores/i18n'
import { useSettingsStore } from '../stores/settings'

const i18n = useI18nStore()
const settings = useSettingsStore()
const toast = useToast()

const opening = ref('')

const sites = [
  {
    name: 'Pexels',
    url: 'https://www.pexels.com/zh-cn/',
    domain: 'pexels.com',
    icon: 'pi pi-camera',
    gradient: 'linear-gradient(135deg, #0ba37f, #0f766e)',
    cats: ['recommended.catPhoto', 'recommended.catVideo'],
  },
  {
    name: 'Unsplash',
    url: 'https://unsplash.com/',
    domain: 'unsplash.com',
    icon: 'pi pi-image',
    gradient: 'linear-gradient(135deg, #475569, #111827)',
    cats: ['recommended.catPhoto'],
  },
  {
    name: 'Pixabay',
    url: 'https://pixabay.com/zh/',
    domain: 'pixabay.com',
    icon: 'pi pi-images',
    gradient: 'linear-gradient(135deg, #65a30d, #166534)',
    cats: ['recommended.catPhoto', 'recommended.catVideo', 'recommended.catIllustration'],
  },
  {
    name: 'Wallhaven',
    url: 'https://wallhaven.cc/',
    domain: 'wallhaven.cc',
    icon: 'pi pi-desktop',
    gradient: 'linear-gradient(135deg, #0ea5e9, #1d4ed8)',
    cats: ['recommended.catPhoto', 'recommended.catIllustration'],
  },
  {
    name: 'Wikimedia Commons',
    url: 'https://commons.wikimedia.org/wiki/Main_Page',
    domain: 'commons.wikimedia.org',
    icon: 'pi pi-globe',
    gradient: 'linear-gradient(135deg, #64748b, #1e293b)',
    cats: ['recommended.catPhoto', 'recommended.catPublic'],
  },
  {
    name: 'NASA Image Library',
    url: 'https://images.nasa.gov/',
    domain: 'images.nasa.gov',
    icon: 'pi pi-star',
    gradient: 'linear-gradient(135deg, #1e3a8a, #7f1d1d)',
    cats: ['recommended.catPhoto', 'recommended.catVideo', 'recommended.catPublic'],
  },
  {
    name: 'Openverse',
    url: 'https://openverse.org/',
    domain: 'openverse.org',
    icon: 'pi pi-search',
    gradient: 'linear-gradient(135deg, #f59e0b, #b45309)',
    cats: ['recommended.catPhoto', 'recommended.catIllustration', 'recommended.catPublic'],
  },
  {
    name: 'Wallpaper Abyss',
    url: 'https://wall.alphacoders.com/',
    domain: 'wall.alphacoders.com',
    icon: 'pi pi-th-large',
    gradient: 'linear-gradient(135deg, #7c3aed, #a21caf)',
    cats: ['recommended.catPhoto', 'recommended.catIllustration'],
  },
]

async function openSite(site) {
  if (opening.value) {
    return
  }
  opening.value = site.domain
  try {
    // 优先使用应用自己的浏览器（首次会自动下载内核并安装插件），失败时退回系统默认浏览器。
    let launched = null
    try {
      launched = await settings.launchBrowserExtension(site.url)
    } catch (err) {
      /* 忽略，走系统默认浏览器 */
    }
    if (launched?.opened) {
      toast.add({
        severity: launched.loaded ? 'success' : 'warn',
        summary: launched.loaded
          ? i18n.t('recommended.autoToast', { browser: launched.browser })
          : i18n.t('recommended.autoStep3'),
        life: 3200,
      })
      return
    }
    await settings.openUrl(site.url)
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: i18n.t('toast.openFailed'),
      detail: String(err?.message ?? err),
      life: 4000,
    })
  } finally {
    opening.value = ''
  }
}
</script>

<template>
  <section class="view">
    <header class="page-head">
      <span class="eyebrow"><i class="pi pi-compass" /> {{ i18n.t('app.recommended') }}</span>
      <h1>{{ i18n.t('recommended.title') }}</h1>
      <p>{{ i18n.t('recommended.subtitle') }}</p>
    </header>

    <p v-if="opening" class="preparing">
      <i class="pi pi-spin pi-spinner" />
      {{ i18n.t('recommended.preparing') }}
    </p>

    <div class="site-grid">
      <button
        v-for="site in sites"
        :key="site.domain"
        type="button"
        class="site-card"
        :disabled="Boolean(opening)"
        @click="openSite(site)"
      >
        <span class="site-icon" :style="{ background: site.gradient }">
          <i :class="site.icon" />
        </span>
        <span class="site-copy">
          <strong>{{ site.name }}</strong>
          <small>{{ site.domain }}</small>
          <span class="site-tags">
            <em v-for="cat in site.cats" :key="cat">{{ i18n.t(cat) }}</em>
          </span>
        </span>
        <i
          v-if="opening === site.domain"
          class="pi pi-spin pi-spinner launch"
        />
        <i v-else class="pi pi-external-link launch" />
      </button>
    </div>
  </section>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: 20px;
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

.preparing {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: -6px;
  font-size: 0.78rem;
  color: var(--p-primary-color);
}

.site-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 12px;
}

.site-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 14px;
  background: var(--app-panel);
  color: var(--p-text-color);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, transform 0.16s ease;
}

.site-card:hover {
  border-color: rgba(129, 140, 248, 0.55);
  transform: translateY(-1px);
}

.site-card:disabled {
  cursor: default;
  opacity: 0.65;
  transform: none;
}

.site-icon {
  width: 42px;
  height: 42px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 12px;
  color: #fff;
  font-size: 1rem;
}

.site-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.site-copy strong {
  font-size: 0.86rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-copy small {
  font-size: 0.7rem;
  color: var(--p-text-muted-color);
}

.site-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 4px;
}

.site-tags em {
  font-style: normal;
  font-size: 0.64rem;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--app-highlight);
  color: var(--p-text-muted-color);
}

.launch {
  flex-shrink: 0;
  font-size: 0.72rem;
  color: var(--p-text-muted-color);
  opacity: 0.7;
}
</style>

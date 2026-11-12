<script setup>
import { computed, ref } from 'vue'
import { useI18nStore } from '../stores/i18n'

const props = defineProps({
  item: { type: Object, required: true },
  applied: { type: Boolean, default: false },
  applying: { type: Boolean, default: false },
})

const emit = defineEmits(['preview', 'apply', 'tag', 'remove'])

const i18n = useI18nStore()
const videoEl = ref(null)

const tagLabel = computed(() => {
  const tags = props.item.tags ?? []
  if (!tags.length) {
    return ''
  }
  const head = tags.slice(0, 2).map((tag) => i18n.tagLabel(tag)).join(' · ')
  return tags.length > 2 ? `${head} +${tags.length - 2}` : head
})

const kindMeta = computed(() => {
  switch (props.item.kind) {
    case 'video':
      return { label: i18n.t('kind.video'), icon: 'pi pi-video' }
    case 'html':
      return { label: i18n.t('kind.html'), icon: 'pi pi-code' }
    default:
      return { label: i18n.t('kind.image'), icon: 'pi pi-image' }
  }
})

const sizeLabel = computed(() => {
  if (props.item.kind === 'html') {
    return 'HTML / CSS / JS'
  }
  const size = props.item.size ?? 0
  if (size >= 1024 * 1024 * 1024) {
    return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`
  }
  if (size >= 1024 * 1024) {
    return `${(size / 1024 / 1024).toFixed(1)} MB`
  }
  return `${Math.max(1, Math.round(size / 1024))} KB`
})

const actionLabel = computed(() => {
  if (props.applied) {
    return i18n.t('card.inUse')
  }
  return props.applying ? i18n.t('card.applying') : i18n.t('card.apply')
})

const actionIcon = computed(() => {
  if (props.applied) {
    return 'pi pi-check'
  }
  return props.applying ? 'pi pi-spin pi-spinner' : 'pi pi-desktop'
})

function playPreview() {
  if (videoEl.value) {
    videoEl.value.play().catch(() => {})
  }
}

function stopPreview() {
  if (videoEl.value) {
    videoEl.value.pause()
    videoEl.value.currentTime = 0
  }
}
</script>

<template>
  <article class="card" :class="{ applied }">
    <div
      class="media"
      role="button"
      tabindex="0"
      @click="emit('preview', item)"
      @keydown.enter="emit('preview', item)"
      @mouseenter="playPreview"
      @mouseleave="stopPreview"
    >
      <img
        v-if="item.kind === 'image'"
        class="media-content"
        :src="item.thumbUrl || item.mediaUrl"
        :alt="item.name"
        loading="lazy"
      />
      <video
        v-else-if="item.kind === 'video'"
        ref="videoEl"
        class="media-content"
        :src="item.mediaUrl"
        muted
        loop
        playsinline
        preload="metadata"
      />
      <div v-else class="html-preview">
        <i class="pi pi-code" />
        <span>HTML WALLPAPER</span>
      </div>

      <span class="type-pill">
        <i :class="kindMeta.icon" />
        {{ kindMeta.label }}
      </span>
      <span v-if="applied" class="applied-pill">
        <i class="pi pi-check" />
        {{ i18n.t('card.inUse') }}
      </span>
      <span v-if="item.kind === 'video'" class="play-pill">
        <i class="pi pi-play" />
      </span>
      <span v-if="item.adult" class="adult-pill">18+</span>

      <div class="caption">
        <span class="name" :title="item.name">{{ item.name }}</span>
        <span class="meta">{{ kindMeta.label }} · {{ sizeLabel }}</span>
        <span v-if="tagLabel" class="tags" :title="item.tags.map((tag) => i18n.tagLabel(tag)).join(' · ')">
          {{ tagLabel }}
        </span>
      </div>
    </div>

    <div class="overlay">
      <button type="button" class="action ghost" @click.stop="emit('preview', item)">
        <i class="pi pi-expand" />
        {{ i18n.t('card.preview') }}
      </button>
      <button type="button" class="action ghost" @click.stop="emit('tag', item)">
        <i class="pi pi-tag" />
        {{ i18n.t('card.tag') }}
      </button>
      <button type="button" class="action danger" @click.stop="emit('remove', item)">
        <i class="pi pi-trash" />
        {{ i18n.t('card.delete') }}
      </button>
      <button
        type="button"
        class="action solid"
        :disabled="applied || applying"
        @click.stop="emit('apply', item)"
      >
        <i :class="actionIcon" />
        {{ actionLabel }}
      </button>
    </div>
  </article>
</template>

<style scoped>
.card {
  position: relative;
  border-radius: 18px;
  overflow: hidden;
  border: 1px solid var(--app-border);
  background: var(--app-panel);
  transition: transform 0.22s ease, border-color 0.22s ease, box-shadow 0.22s ease;
}

.card:hover {
  transform: translateY(-4px);
  border-color: rgba(129, 140, 248, 0.55);
  box-shadow: var(--app-card-shadow);
}

.card.applied {
  border-color: rgba(99, 102, 241, 0.6);
  box-shadow: 0 0 0 1px rgba(99, 102, 241, 0.35), var(--app-card-shadow);
}

.media {
  position: relative;
  aspect-ratio: 16 / 10;
  overflow: hidden;
  background: #0b0c12;
  cursor: pointer;
}

.media-content {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
  transition: transform 0.4s ease;
}

.card:hover .media-content {
  transform: scale(1.05);
}

.html-preview {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: rgba(255, 255, 255, 0.92);
  background: radial-gradient(120% 120% at 0% 0%, rgba(255, 255, 255, 0.28), transparent 55%),
    linear-gradient(135deg, #4f46e5, #7c3aed 52%, #0ea5e9);
}

.html-preview i {
  font-size: 1.9rem;
  opacity: 0.95;
}

.html-preview span {
  font-size: 0.64rem;
  font-weight: 600;
  letter-spacing: 0.18em;
  opacity: 0.85;
}

.type-pill,
.applied-pill,
.play-pill {
  position: absolute;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 9px;
  border-radius: 999px;
  font-size: 0.67rem;
  font-weight: 500;
  line-height: 1.4;
  color: #f8fafc;
  background: rgba(9, 9, 11, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.14);
  backdrop-filter: blur(8px);
}

.type-pill {
  left: 10px;
  top: 10px;
}

.applied-pill {
  right: 10px;
  top: 10px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  border-color: transparent;
  box-shadow: 0 6px 16px rgba(99, 102, 241, 0.45);
}

.play-pill {
  right: 10px;
  bottom: 62px;
  width: 26px;
  height: 26px;
  padding: 0;
  justify-content: center;
  font-size: 0.6rem;
}

.adult-pill {
  position: absolute;
  top: 42px;
  left: 10px;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid rgba(244, 63, 94, 0.5);
  background: rgba(159, 18, 57, 0.72);
  color: #ffe4e6;
  font-size: 0.62rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  backdrop-filter: blur(8px);
}

.tags {
  margin-top: 2px;
  font-size: 0.66rem;
  color: rgba(199, 210, 254, 0.9);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.caption {
  position: absolute;
  inset: auto 0 0 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: 38px 14px 12px;
  background: var(--app-scrim);
  pointer-events: none;
}

.name {
  font-size: 0.86rem;
  font-weight: 600;
  color: #fff;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.62);
}

.overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 0 14px;
  background: rgba(4, 5, 9, 0.52);
  backdrop-filter: blur(4px);
  opacity: 0;
  transition: opacity 0.18s ease;
}

.card:hover .overlay,
.card:focus-within .overlay {
  opacity: 1;
}

.action {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 9px 15px;
  border-radius: 11px;
  border: 1px solid transparent;
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.16s ease, transform 0.16s ease;
}

.action:active {
  transform: scale(0.97);
}

.action.ghost {
  color: #fff;
  background: rgba(255, 255, 255, 0.14);
  border-color: rgba(255, 255, 255, 0.3);
  backdrop-filter: blur(8px);
}

.action.ghost:hover {
  background: rgba(255, 255, 255, 0.24);
}

.action.solid {
  color: #fff;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  box-shadow: 0 10px 22px rgba(99, 102, 241, 0.4);
}

.action.danger {
  color: #ffe4e6;
  background: rgba(159, 18, 57, 0.55);
  border-color: rgba(244, 63, 94, 0.55);
  backdrop-filter: blur(8px);
}

.action.danger:hover {
  background: rgba(190, 18, 60, 0.75);
}

.action.solid:hover {
  background: linear-gradient(135deg, #7175f5, #9569f8);
}

.action.solid:disabled {
  opacity: 0.75;
  cursor: default;
  box-shadow: none;
}
</style>

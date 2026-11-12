<script setup>
import { computed } from 'vue'
import { useI18nStore } from '../stores/i18n'

const i18n = useI18nStore()

const props = defineProps({
  visible: { type: Boolean, default: false },
  item: { type: Object, default: null },
})

const emit = defineEmits(['update:visible', 'apply'])

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

const kindMeta = computed(() => {
  switch (props.item?.kind) {
    case 'video':
      return { label: i18n.t('kind.videoWallpaper'), icon: 'pi pi-video' }
    case 'html':
      return { label: i18n.t('kind.htmlWallpaper'), icon: 'pi pi-code' }
    default:
      return { label: i18n.t('kind.imageWallpaper'), icon: 'pi pi-image' }
  }
})
</script>

<template>
  <Dialog
    v-model:visible="dialogVisible"
    modal
    :dismissableMask="true"
    :style="{ width: 'min(1180px, 94vw)' }"
    class="preview-dialog"
  >
    <template #header>
      <div class="dialog-head">
        <span class="dialog-icon"><i :class="kindMeta.icon" /></span>
        <div class="dialog-title">
          <strong>{{ item?.name || i18n.t('preview.title') }}</strong>
          <small>{{ kindMeta.label }}</small>
        </div>
      </div>
    </template>

    <div v-if="item" class="preview-body">
      <img v-if="item.kind === 'image'" :src="item.mediaUrl" :alt="item.name" />
      <video
        v-else-if="item.kind === 'video'"
        :src="item.mediaUrl"
        controls
        autoplay
        loop
        playsinline
      />
      <iframe v-else :src="item.mediaUrl" :title="i18n.t('preview.htmlFrame')" />
    </div>

    <template #footer>
      <span class="preview-path" :title="item?.path">
        <i class="pi pi-map-marker" />
        {{ item?.path }}
      </span>
      <Button
        :label="i18n.t('common.close')"
        severity="secondary"
        text
        @click="dialogVisible = false"
      />
      <Button
        :label="i18n.t('card.apply')"
        icon="pi pi-desktop"
        class="brand-btn"
        @click="item && emit('apply', item)"
      />
    </template>
  </Dialog>
</template>

<style scoped>
.dialog-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.dialog-icon {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border-radius: 11px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.24), rgba(139, 92, 246, 0.12));
  color: var(--p-primary-color);
}

.dialog-title {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
}

.dialog-title strong {
  font-size: 1rem;
  font-weight: 600;
}

.dialog-title small {
  font-size: 0.72rem;
  color: var(--p-text-muted-color);
}

.preview-body {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 220px;
  border-radius: 14px;
  overflow: hidden;
  background: radial-gradient(120% 120% at 50% 0%, rgba(99, 102, 241, 0.14), transparent 60%),
    #05060a;
}

.preview-body img,
.preview-body video {
  display: block;
  max-width: 100%;
  max-height: 68vh;
  object-fit: contain;
}

.preview-body iframe {
  width: 100%;
  height: 64vh;
  border: none;
  background: #fff;
}

.preview-path {
  flex: 1;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  text-align: left;
  font-size: 0.72rem;
  color: var(--p-text-muted-color);
  overflow: hidden;
  white-space: nowrap;
}

.preview-path i {
  flex-shrink: 0;
}

:deep(.p-dialog) {
  border-radius: 20px;
  border: 1px solid var(--app-border);
  box-shadow: var(--app-shadow);
  overflow: hidden;
}

:deep(.p-dialog-header) {
  padding: 18px 22px;
}

:deep(.p-dialog-content) {
  padding: 0 22px 18px;
  background: transparent;
}

:deep(.p-dialog-footer) {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 22px 20px;
}
</style>

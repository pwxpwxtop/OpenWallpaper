<script setup>
import { computed, ref, watch } from 'vue'
import { useWallpapersStore } from '../stores/wallpapers'
import { useI18nStore } from '../stores/i18n'
import { TAG_GROUPS } from '../data/tagPresets'

const props = defineProps({
  visible: { type: Boolean, default: false },
  item: { type: Object, default: null },
})

const emit = defineEmits(['update:visible', 'saved'])

const wallpapers = useWallpapersStore()
const i18n = useI18nStore()

const draft = ref('')
const tags = ref([])
const adult = ref(false)
const saving = ref(false)
const error = ref('')

const shown = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

watch(
  () => props.visible,
  (open) => {
    if (!open || !props.item) {
      return
    }
    tags.value = [...(props.item.tags ?? [])]
    adult.value = Boolean(props.item.adult)
    draft.value = ''
    error.value = ''
  },
)

const suggestionGroups = computed(() =>
  TAG_GROUPS.map((group) => ({
    key: group.key,
    label: i18n.t(group.labelKey),
    tags: group.tags.filter((name) => !tags.value.includes(name)),
  })).filter((group) => group.tags.length),
)

const kindLabel = computed(() => {
  switch (props.item?.kind) {
    case 'video':
      return i18n.t('kind.video')
    case 'html':
      return i18n.t('kind.htmlWallpaper')
    default:
      return i18n.t('kind.image')
  }
})

function addTag(value) {
  const name = String(value ?? draft.value).trim()
  if (!name) {
    return
  }
  if (!tags.value.includes(name) && tags.value.length < 20) {
    tags.value = [...tags.value, name]
  }
  draft.value = ''
}

function removeTag(name) {
  tags.value = tags.value.filter((tag) => tag !== name)
}

async function save() {
  if (!props.item) {
    return
  }
  saving.value = true
  error.value = ''
  try {
    await wallpapers.setMeta(props.item, tags.value, adult.value)
    emit('saved')
    shown.value = false
  } catch (err) {
    error.value = String(err?.message ?? err)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Dialog
    v-model:visible="shown"
    modal
    :header="i18n.t('tag.title')"
    :style="{ width: '480px' }"
  >
    <div v-if="item" class="tag-editor">
      <div class="subject">
        <div class="thumb">
          <img
            v-if="item.kind === 'image'"
            :src="item.thumbUrl || item.mediaUrl"
            :alt="item.name"
          />
          <video v-else-if="item.kind === 'video'" :src="item.mediaUrl" muted preload="metadata" />
          <i v-else class="pi pi-code" />
        </div>
        <div class="subject-copy">
          <strong :title="item.name">{{ item.name }}</strong>
          <span>{{ kindLabel }}</span>
        </div>
      </div>

      <div class="field">
        <label>{{ i18n.t('tag.selected') }}</label>
        <div v-if="!tags.length" class="hint">{{ i18n.t('tag.selectedEmpty') }}</div>
        <div v-else class="chips">
          <span v-for="tag in tags" :key="tag" class="tag-chip">
            {{ i18n.tagLabel(tag) }}
            <button type="button" :title="i18n.t('tag.remove')" @click="removeTag(tag)">
              <i class="pi pi-times" />
            </button>
          </span>
        </div>
      </div>

      <div class="field">
        <label>{{ i18n.t('tag.addLabel') }}</label>
        <div class="adder">
          <InputText
            v-model="draft"
            :placeholder="i18n.t('tag.placeholder')"
            class="tag-input"
            @keyup.enter="addTag()"
          />
          <Button
            :label="i18n.t('common.add')"
            icon="pi pi-plus"
            class="ghost-btn"
            outlined
            @click="addTag()"
          />
        </div>
        <div v-if="suggestionGroups.length" class="suggestion-groups">
          <div v-for="group in suggestionGroups" :key="group.key" class="suggestion-group">
            <span class="group-label">{{ group.label }}</span>
            <div class="chips">
              <button
                v-for="name in group.tags"
                :key="name"
                type="button"
                class="suggest"
                @click="addTag(name)"
              >
                + {{ i18n.tagLabel(name) }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <button type="button" class="adult-toggle" :class="{ on: adult }" @click="adult = !adult">
        <span class="box"><i v-if="adult" class="pi pi-check" /></span>
        <span class="adult-copy">
          <strong>{{ i18n.t('tag.adult') }}</strong>
          <small>{{ i18n.t('tag.adultDesc') }}</small>
        </span>
      </button>

      <p v-if="error" class="error">{{ error }}</p>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <Button
          :label="i18n.t('common.cancel')"
          class="ghost-btn"
          outlined
          @click="shown = false"
        />
        <Button
          :label="i18n.t('common.save')"
          icon="pi pi-check"
          class="brand-btn"
          :loading="saving"
          @click="save"
        />
      </div>
    </template>
  </Dialog>
</template>

<style scoped>
.tag-editor {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding-top: 2px;
}

.subject {
  display: flex;
  align-items: center;
  gap: 12px;
}

.thumb {
  width: 72px;
  height: 46px;
  flex-shrink: 0;
  overflow: hidden;
  border-radius: 10px;
  background: var(--app-panel-2);
  display: grid;
  place-items: center;
  color: var(--p-text-muted-color);
}

.thumb img,
.thumb video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.subject-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.subject-copy strong {
  font-size: 0.86rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.subject-copy span {
  font-size: 0.72rem;
  color: var(--p-text-muted-color);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field > label {
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--p-text-muted-color);
}

.hint {
  font-size: 0.76rem;
  color: var(--p-text-muted-color);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px 5px 11px;
  border: 1px solid rgba(99, 102, 241, 0.55);
  border-radius: 999px;
  background: rgba(99, 102, 241, 0.14);
  color: var(--app-nav-active-text);
  font-size: 0.76rem;
}

.tag-chip button {
  display: grid;
  place-items: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: rgba(99, 102, 241, 0.25);
  color: inherit;
  font-size: 0.55rem;
  cursor: pointer;
}

.adder {
  display: flex;
  gap: 8px;
}

.tag-input {
  flex: 1;
}

.suggest {
  padding: 5px 10px;
  border: 1px dashed var(--app-border-strong);
  border-radius: 999px;
  background: transparent;
  color: var(--p-text-muted-color);
  font-size: 0.74rem;
  cursor: pointer;
}

.suggestion-groups {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 210px;
  overflow-y: auto;
  padding-right: 2px;
}

.suggestion-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.group-label {
  font-size: 0.68rem;
  color: var(--p-text-muted-color);
}

.suggest:hover {
  border-color: var(--p-primary-color);
  color: var(--p-primary-color);
}

.adult-toggle {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: 13px;
  background: var(--app-panel-2);
  text-align: left;
  cursor: pointer;
  color: var(--p-text-color);
}

.adult-toggle.on {
  border-color: rgba(244, 63, 94, 0.55);
  background: rgba(244, 63, 94, 0.08);
}

.box {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  margin-top: 1px;
  border: 1px solid var(--app-border-strong);
  border-radius: 6px;
  font-size: 0.6rem;
}

.adult-toggle.on .box {
  border-color: transparent;
  background: linear-gradient(135deg, #f43f5e, #fb7185);
  color: #fff;
}

.adult-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.adult-copy strong {
  font-size: 0.8rem;
}

.adult-copy small {
  font-size: 0.72rem;
  color: var(--p-text-muted-color);
}

.error {
  font-size: 0.76rem;
  color: #f43f5e;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
  width: 100%;
}
</style>

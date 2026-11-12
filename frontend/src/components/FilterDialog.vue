<script setup>
import { computed } from 'vue'
import { useWallpapersStore } from '../stores/wallpapers'
import { useI18nStore } from '../stores/i18n'
import { TAG_GROUPS, PRESET_TAGS } from '../data/tagPresets'

const props = defineProps({
  visible: { type: Boolean, default: false },
})

const emit = defineEmits(['update:visible'])

const wallpapers = useWallpapersStore()
const i18n = useI18nStore()

const types = computed(() => [
  { value: 'image', label: i18n.t('kind.image'), icon: 'pi pi-image' },
  { value: 'video', label: i18n.t('kind.video'), icon: 'pi pi-video' },
  { value: 'html', label: i18n.t('kind.html'), icon: 'pi pi-code' },
])

const adults = computed(() => [
  { value: 'all', label: i18n.t('filter.adultAll') },
  { value: 'hide', label: i18n.t('filter.adultHide') },
  { value: 'only', label: i18n.t('filter.adultOnly') },
])

const myTags = computed(() => wallpapers.tags.filter((tag) => !PRESET_TAGS.has(tag.name)))

function tagCount(name) {
  return wallpapers.tags.find((tag) => tag.name === name)?.count ?? 0
}

const shown = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})
</script>

<template>
  <Dialog v-model:visible="shown" modal :header="i18n.t('filter.title')" :style="{ width: '520px' }">
    <div class="filters">
      <section class="group">
        <h3>{{ i18n.t('filter.type') }}</h3>
        <div class="chips">
          <button
            v-for="type in types"
            :key="type.value"
            type="button"
            class="chip"
            :class="{ active: wallpapers.typeFilters.includes(type.value) }"
            @click="wallpapers.toggleType(type.value)"
          >
            <i :class="type.icon" />
            {{ type.label }}
          </button>
        </div>
      </section>

      <section v-for="group in TAG_GROUPS" :key="group.key" class="group">
        <h3>{{ i18n.t(group.labelKey) }}</h3>
        <div class="chips">
          <button
            v-for="name in group.tags"
            :key="name"
            type="button"
            class="chip"
            :class="{ active: wallpapers.tagFilters.includes(name) }"
            @click="wallpapers.toggleTag(name)"
          >
            {{ i18n.tagLabel(name) }}
            <em v-if="tagCount(name)">{{ tagCount(name) }}</em>
          </button>
        </div>
      </section>

      <section class="group">
        <h3>{{ i18n.t('filter.myTags') }}</h3>
        <p v-if="!myTags.length" class="hint">
          {{ i18n.t('filter.myTagsEmpty') }}
        </p>
        <div v-else class="chips">
          <button
            v-for="tag in myTags"
            :key="tag.name"
            type="button"
            class="chip"
            :class="{ active: wallpapers.tagFilters.includes(tag.name) }"
            @click="wallpapers.toggleTag(tag.name)"
          >
            {{ i18n.tagLabel(tag.name) }}
            <em>{{ tag.count }}</em>
          </button>
        </div>
      </section>

      <section class="group">
        <h3>{{ i18n.t('filter.adult') }}</h3>
        <div class="seg">
          <button
            v-for="option in adults"
            :key="option.value"
            type="button"
            :class="{ active: wallpapers.adultFilter === option.value }"
            @click="wallpapers.adultFilter = option.value"
          >
            {{ option.label }}
          </button>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <span class="result">
          {{ i18n.t('filter.result', { count: wallpapers.filtered.length }) }}
        </span>
        <div class="footer-btns">
          <Button
            :label="i18n.t('filter.reset')"
            icon="pi pi-filter-slash"
            class="ghost-btn"
            outlined
            @click="wallpapers.resetFilters()"
          />
          <Button :label="i18n.t('common.done')" class="brand-btn" @click="shown = false" />
        </div>
      </div>
    </template>
  </Dialog>
</template>

<style scoped>
.filters {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 4px 4px 2px 0;
  max-height: 62vh;
  overflow-y: auto;
}

.group {
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.group h3 {
  font-size: 0.74rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--p-text-muted-color);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: var(--app-panel-2);
  color: var(--p-text-color);
  font-size: 0.78rem;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease;
}

.chip:hover {
  border-color: rgba(129, 140, 248, 0.5);
}

.chip.active {
  border-color: rgba(99, 102, 241, 0.7);
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(139, 92, 246, 0.1));
  color: var(--app-nav-active-text);
}

.chip em {
  font-style: normal;
  font-size: 0.64rem;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--app-highlight);
  color: var(--p-text-muted-color);
}

.chip.active em {
  background: rgba(99, 102, 241, 0.28);
  color: inherit;
}

.hint {
  font-size: 0.76rem;
  color: var(--p-text-muted-color);
}

.seg {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border: 1px solid var(--app-border);
  border-radius: 12px;
  background: var(--app-panel-2);
}

.seg button {
  padding: 7px 14px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--p-text-muted-color);
  font-size: 0.77rem;
  cursor: pointer;
}

.seg button.active {
  color: #fff;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}

.result {
  font-size: 0.76rem;
  color: var(--p-text-muted-color);
}

.footer-btns {
  display: flex;
  gap: 9px;
}
</style>

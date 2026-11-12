import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { WallpaperService } from '../../bindings/changeme'

export const useWallpapersStore = defineStore('wallpapers', () => {
  const items = ref([])
  const loading = ref(false)
  const error = ref('')
  const filter = ref('all')
  const search = ref('')
  const applyingId = ref('')
  const appliedPath = ref('')
  const typeFilters = ref([])
  const tagFilters = ref([])
  const adultFilter = ref('all') // all | hide | only

  const filtered = computed(() => {
    const keyword = search.value.trim().toLowerCase()
    return items.value.filter((item) => {
      if (filter.value !== 'all' && item.kind !== filter.value) {
        return false
      }
      if (typeFilters.value.length && !typeFilters.value.includes(item.kind)) {
        return false
      }
      if (adultFilter.value === 'hide' && item.adult) {
        return false
      }
      if (adultFilter.value === 'only' && !item.adult) {
        return false
      }
      if (tagFilters.value.length) {
        const itemTags = item.tags ?? []
        // 同时选中多个标签时，壁纸需全部包含。
        if (!tagFilters.value.every((tag) => itemTags.includes(tag))) {
          return false
        }
      }
      return !keyword || item.name.toLowerCase().includes(keyword)
    })
  })

  // 所有已使用过的标签及其数量，供筛选与编辑时选择。
  const tags = computed(() => {
    const map = new Map()
    for (const item of items.value) {
      for (const tag of item.tags ?? []) {
        map.set(tag, (map.get(tag) ?? 0) + 1)
      }
    }
    return Array.from(map, ([name, count]) => ({ name, count })).sort(
      (a, b) => b.count - a.count || a.name.localeCompare(b.name),
    )
  })

  const activeFilterCount = computed(
    () =>
      typeFilters.value.length +
      tagFilters.value.length +
      (adultFilter.value === 'all' ? 0 : 1),
  )

  function toggleType(kind) {
    const set = new Set(typeFilters.value)
    if (set.has(kind)) {
      set.delete(kind)
    } else {
      set.add(kind)
    }
    typeFilters.value = Array.from(set)
  }

  function toggleTag(tag) {
    const set = new Set(tagFilters.value)
    if (set.has(tag)) {
      set.delete(tag)
    } else {
      set.add(tag)
    }
    tagFilters.value = Array.from(set)
  }

  function resetFilters() {
    typeFilters.value = []
    tagFilters.value = []
    adultFilter.value = 'all'
  }

  async function setMeta(item, tags, adult) {
    await WallpaperService.SetWallpaperMeta(item.id, tags, adult)
    item.tags = tags
    item.adult = adult
  }

  const counts = computed(() => ({
    all: items.value.length,
    image: items.value.filter((item) => item.kind === 'image').length,
    video: items.value.filter((item) => item.kind === 'video').length,
    html: items.value.filter((item) => item.kind === 'html').length,
  }))

  async function scan() {
    loading.value = true
    error.value = ''
    try {
      const result = await WallpaperService.ScanWallpapers()
      items.value = result ?? []
    } catch (err) {
      error.value = String(err?.message ?? err)
      items.value = []
    } finally {
      loading.value = false
    }
  }

  async function apply(item) {
    applyingId.value = item.id
    try {
      await WallpaperService.ApplyWallpaper({ ...item })
      appliedPath.value = item.path
    } finally {
      applyingId.value = ''
    }
  }

  async function stopDynamic() {
    await WallpaperService.StopDynamicWallpaper()
    appliedPath.value = ''
  }

  async function remove(item) {
    await WallpaperService.DeleteWallpaper({ ...item })
    if (appliedPath.value === item.path) {
      appliedPath.value = ''
    }
    await scan()
  }

  return {
    items,
    loading,
    error,
    filter,
    search,
    applyingId,
    appliedPath,
    typeFilters,
    tagFilters,
    adultFilter,
    filtered,
    counts,
    tags,
    activeFilterCount,
    toggleType,
    toggleTag,
    resetFilters,
    setMeta,
    scan,
    apply,
    stopDynamic,
    remove,
  }
})

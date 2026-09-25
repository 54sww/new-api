import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { RouteLocationNormalized } from 'vue-router'

export interface TabItem {
  path: string
  fullPath: string
  title: string
  closable: boolean
}

export const useTabsStore = defineStore('tabs', () => {
  const openedTabs = ref<TabItem[]>([])
  const activeTab = ref('')
  const cacheVersions = ref<Record<string, number>>({})

  function getCacheKey(path: string) {
    const version = cacheVersions.value[path] || 0
    return `${path}@${version}`
  }

  function invalidateCache(path: string) {
    cacheVersions.value[path] = (cacheVersions.value[path] || 0) + 1
  }

  function invalidateCaches(paths: string[]) {
    paths.forEach(invalidateCache)
  }

  function addTab(route: RouteLocationNormalized) {
    if (route.meta.noAuth || route.meta.hidden) return
    const title = (route.meta.title as string) || (route.name as string) || ''
    const existing = openedTabs.value.find((t) => t.path === route.path)
    if (!existing) {
      openedTabs.value.push({
        path: route.path,
        fullPath: route.fullPath,
        title,
        closable: route.meta.closable !== false,
      })
    } else {
      existing.fullPath = route.fullPath
      existing.title = title
    }
    activeTab.value = route.path
  }

  function removeTab(path: string) {
    const idx = openedTabs.value.findIndex((t) => t.path === path)
    if (idx === -1) return null
    openedTabs.value.splice(idx, 1)
    invalidateCache(path)
    if (activeTab.value === path) {
      const next = openedTabs.value[idx] || openedTabs.value[idx - 1]
      if (next) {
        activeTab.value = next.path
        return next.fullPath
      }
    }
    return null
  }

  function closeOtherTabs(path: string) {
    const removed = openedTabs.value
      .filter((t) => t.path !== path && t.closable)
      .map((t) => t.path)
    openedTabs.value = openedTabs.value.filter((t) => t.path === path || !t.closable)
    invalidateCaches(removed)
    activeTab.value = path
  }

  function closeAllTabs() {
    const removed = openedTabs.value.filter((t) => t.closable).map((t) => t.path)
    openedTabs.value = openedTabs.value.filter((t) => !t.closable)
    invalidateCaches(removed)
    const first = openedTabs.value[0]
    if (first) {
      activeTab.value = first.path
      return first.fullPath
    }
    return null
  }

  function closeLeftTabs(path: string) {
    const idx = openedTabs.value.findIndex((t) => t.path === path)
    if (idx > 0) {
      const removed = openedTabs.value.slice(0, idx).map((t) => t.path)
      openedTabs.value = openedTabs.value.slice(idx)
      invalidateCaches(removed)
    }
  }

  function closeRightTabs(path: string) {
    const idx = openedTabs.value.findIndex((t) => t.path === path)
    if (idx >= 0) {
      const removed = openedTabs.value.slice(idx + 1).map((t) => t.path)
      openedTabs.value = openedTabs.value.slice(0, idx + 1)
      invalidateCaches(removed)
    }
  }

  return {
    openedTabs,
    activeTab,
    getCacheKey,
    addTab,
    removeTab,
    closeOtherTabs,
    closeAllTabs,
    closeLeftTabs,
    closeRightTabs,
  }
})

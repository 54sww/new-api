import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export const usePermissionStore = defineStore('permission', () => {
  const menuPaths = ref<string[]>([])
  const permissions = ref<string[]>([])
  const superAdmin = ref(false)
  const loaded = ref(false)

  const isSuperAdmin = computed(() => superAdmin.value)

  function setFromUserInfo(info: {
    menu_paths?: string[]
    permissions?: string[]
    super_admin?: boolean
  }) {
    menuPaths.value = info.menu_paths || []
    permissions.value = info.permissions || []
    superAdmin.value = !!info.super_admin
    loaded.value = true
  }

  function hasPermission(code: string): boolean {
    if (superAdmin.value) return true
    return permissions.value.includes(code)
  }

  function canAccessRoute(path: string, permission?: string): boolean {
    if (superAdmin.value) return true
    if (permission && permissions.value.includes(permission)) return true
    if (menuPaths.value.includes(path)) return true
    return menuPaths.value.some((p) => path === p || path.startsWith(p + '/'))
  }

  function reset() {
    menuPaths.value = []
    permissions.value = []
    superAdmin.value = false
    loaded.value = false
  }

  return {
    menuPaths,
    permissions,
    superAdmin,
    loaded,
    isSuperAdmin,
    setFromUserInfo,
    hasPermission,
    canAccessRoute,
    reset,
  }
})

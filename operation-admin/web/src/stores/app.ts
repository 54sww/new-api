import { defineStore } from 'pinia'
import { ref } from 'vue'

/** Top-level product modules. Add `ops` routes later. */
export type Subsystem = 'finance' | 'ops' | 'system'

export const useAppStore = defineStore('app', () => {
  const sidebarCollapsed = ref(false)
  const activeSubsystem = ref<Subsystem>('finance')

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function switchSubsystem(subsystem: Subsystem) {
    activeSubsystem.value = subsystem
  }

  return {
    sidebarCollapsed,
    activeSubsystem,
    toggleSidebar,
    switchSubsystem,
  }
})

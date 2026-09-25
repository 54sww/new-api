<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { useAppStore, type Subsystem } from '@/stores/app'
import { useTabsStore } from '@/stores/tabs'
import { useUserStore } from '@/stores/user'
import { usePermissionStore } from '@/stores/permission'
import { displayName } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const tabsStore = useTabsStore()
const userStore = useUserStore()
const permStore = usePermissionStore()

const subsystems = computed(() => {
  const all: { key: Subsystem; label: string }[] = [
    { key: 'finance', label: '财务管理' },
    { key: 'ops', label: '运维管理' },
    { key: 'system', label: '系统管理' },
  ]
  return all.filter((sys) =>
    router.getRoutes().some((r) => {
      if (r.meta.hidden || r.meta.subsystem !== sys.key) return false
      if (!r.meta.title || !r.path) return false
      const path = r.path.startsWith('/') ? r.path : `/${r.path}`
      return permStore.canAccessRoute(path, r.meta.permission as string | undefined)
    })
  )
})

const activeSubsystem = computed(() => appStore.activeSubsystem)
const isCollapsed = computed(() => appStore.sidebarCollapsed)

function switchSubsystem(key: Subsystem) {
  appStore.switchSubsystem(key)
  const target = findSubsystemDefaultPath(key)
  router.push(target)
}

function findSubsystemDefaultPath(key: Subsystem) {
  const routes = router.getRoutes()
  const first = routes.find((r) => {
    if (r.meta.subsystem !== key || r.meta.hidden || !r.meta.title || !r.path) return false
    if (r.path.includes(':')) return false
    const path = r.path.startsWith('/') ? r.path : `/${r.path}`
    return permStore.canAccessRoute(path, r.meta.permission as string | undefined)
  })
  if (first) {
    return first.path.startsWith('/') ? first.path : `/${first.path}`
  }
  return '/dashboard'
}

interface MenuItem {
  id: string
  label: string
  icon?: string
  path?: string
  children?: MenuItem[]
}

const menuList = computed<MenuItem[]>(() => {
  const routes = router.getRoutes()
  const parentMap = new Map<string, MenuItem>()
  const result: MenuItem[] = []

  if (permStore.canAccessRoute('/dashboard', 'common:home')) {
    result.push({ id: 'Dashboard', label: '首页', icon: 'HomeFilled', path: '/dashboard' })
  }

  routes.forEach((r) => {
    const sub = r.meta.subsystem as string | undefined
    if (r.meta.hidden || !r.meta.title || !r.path) return
    if (sub !== activeSubsystem.value && sub !== 'common') return
    if (r.name === 'Dashboard') return
    if (sub === 'common' && !r.meta.parent) return

    const path = r.path.startsWith('/') ? r.path : `/${r.path}`
    if (!permStore.canAccessRoute(path, r.meta.permission as string | undefined)) return

    if (r.meta.parent) {
      const parentKey = r.meta.parent as string
      if (!parentMap.has(parentKey)) {
        parentMap.set(parentKey, {
          id: parentKey,
          label: parentKey,
          icon: (r.meta.parentIcon as string) || 'Folder',
          children: [],
        })
      }
      const parent = parentMap.get(parentKey)!
      if (r.meta.parentIcon) parent.icon = r.meta.parentIcon as string
      parent.children!.push({
        id: r.name as string,
        label: r.meta.title as string,
        icon: r.meta.icon as string | undefined,
        path,
      })
      return
    }

    result.push({
      id: r.name as string,
      label: r.meta.title as string,
      icon: r.meta.icon as string | undefined,
      path,
    })
  })

  for (const [, menu] of parentMap) {
    if (menu.children && menu.children.length > 0) result.push(menu)
  }
  return result
})

const activeMenu = ref(route.path)
watch(
  () => route.path,
  (p) => {
    activeMenu.value = p
  },
  { immediate: true }
)

function onMenuSelect(path: string) {
  router.push(path)
}

watch(
  () => route.fullPath,
  () => {
    if (!route.meta.noAuth) {
      tabsStore.addTab(route)
    }
  },
  { immediate: true }
)

const pageCacheKey = computed(() => tabsStore.getCacheKey(route.path))

function onTabClick(path: string) {
  const tab = tabsStore.openedTabs.find((t) => t.path === path)
  if (tab && route.path !== path) router.push(tab.fullPath)
}

function onTabRemove(path: string) {
  const navigateTo = tabsStore.removeTab(path)
  if (navigateTo) router.push(navigateTo)
}

const contextMenu = reactive({ visible: false, x: 0, y: 0, path: '' })

function onTabContextMenu(e: MouseEvent, path: string) {
  e.preventDefault()
  contextMenu.visible = true
  contextMenu.x = e.clientX
  contextMenu.y = e.clientY
  contextMenu.path = path
}

function closeContextMenu() {
  contextMenu.visible = false
}

function handleContextAction(action: string) {
  const path = contextMenu.path
  closeContextMenu()
  if (action === 'close') {
    const nav = tabsStore.removeTab(path)
    if (nav) router.push(nav)
  } else if (action === 'closeOthers') {
    tabsStore.closeOtherTabs(path)
    if (route.path !== path) router.push(path)
  } else if (action === 'closeLeft') {
    tabsStore.closeLeftTabs(path)
  } else if (action === 'closeRight') {
    tabsStore.closeRightTabs(path)
  } else if (action === 'closeAll') {
    const nav = tabsStore.closeAllTabs()
    if (nav) router.push(nav)
  }
}

onMounted(() => {
  if (userStore.isLoggedIn() && !userStore.user) {
    void userStore.fetchMe().catch(() => undefined)
  }
})

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确定退出登录?', '提示', { type: 'warning' })
    userStore.logout()
    router.push('/login')
  } catch {
    /* cancelled */
  }
}
</script>

<template>
  <div class="main-layout" @click="closeContextMenu">
    <div class="top-header">
      <div class="header-left">
        <span class="logo-icon">⚙️</span>
        <span class="logo-text">Operation Admin</span>
        <span class="header-divider">|</span>
        <div v-if="subsystems.length > 0" class="subsystem-tabs">
          <span
            v-for="sys in subsystems"
            :key="sys.key"
            :class="['subsystem-item', { active: activeSubsystem === sys.key }]"
            @click="switchSubsystem(sys.key)"
          >
            {{ sys.label }}
          </span>
        </div>
      </div>
      <div class="header-right">
        <el-dropdown trigger="click">
          <span class="user-info">
            <el-icon><UserFilled /></el-icon>
            <span>{{ displayName(userStore.user) }}</span>
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item divided @click="handleLogout">
                <el-icon><SwitchButton /></el-icon> 退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <div class="main-body">
      <div :class="['sidebar', { collapsed: isCollapsed }]">
        <el-menu
          :default-active="activeMenu"
          :collapse="isCollapsed"
          :collapse-transition="false"
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409eff"
          @select="onMenuSelect"
        >
          <template v-for="menu in menuList" :key="menu.id">
            <el-sub-menu v-if="menu.children && menu.children.length > 0" :index="menu.id">
              <template #title>
                <el-icon><component :is="menu.icon || 'Folder'" /></el-icon>
                <span>{{ menu.label }}</span>
              </template>
              <el-menu-item v-for="child in menu.children" :key="child.id" :index="child.path!">
                <el-icon><component :is="child.icon || 'Document'" /></el-icon>
                <span>{{ child.label }}</span>
              </el-menu-item>
            </el-sub-menu>
            <el-menu-item v-else :index="menu.path!">
              <el-icon v-if="menu.icon"><component :is="menu.icon" /></el-icon>
              <span>{{ menu.label }}</span>
            </el-menu-item>
          </template>
        </el-menu>

        <div class="collapse-btn" @click="appStore.toggleSidebar()">
          <el-icon :size="16">
            <Fold v-if="!isCollapsed" />
            <Expand v-else />
          </el-icon>
        </div>
      </div>

      <div class="content-area">
        <div class="tab-bar">
          <div class="tab-list">
            <div
              v-for="tab in tabsStore.openedTabs"
              :key="tab.path"
              :class="['tab-item', { active: tab.path === route.path }]"
              @click="onTabClick(tab.path)"
              @contextmenu="(e: MouseEvent) => onTabContextMenu(e, tab.path)"
            >
              <span class="tab-title">{{ tab.title }}</span>
              <el-icon
                v-if="tab.closable !== false"
                class="tab-close"
                :size="12"
                @click.stop="onTabRemove(tab.path)"
              >
                <Close />
              </el-icon>
            </div>
          </div>
          <span
            class="tab-close-all"
            @click="
              () => {
                const nav = tabsStore.closeAllTabs()
                if (nav) router.push(nav)
              }
            "
          >
            <el-icon :size="14"><CloseBold /></el-icon>
          </span>
        </div>

        <teleport to="body">
          <div
            v-if="contextMenu.visible"
            class="context-menu"
            :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
          >
            <div class="menu-item" @click="handleContextAction('close')">关闭当前</div>
            <div class="menu-item" @click="handleContextAction('closeOthers')">关闭其他</div>
            <div class="menu-item" @click="handleContextAction('closeLeft')">关闭左侧</div>
            <div class="menu-item" @click="handleContextAction('closeRight')">关闭右侧</div>
            <div class="menu-item" @click="handleContextAction('closeAll')">关闭全部</div>
          </div>
        </teleport>

        <div class="page-content">
          <router-view v-slot="{ Component }">
            <keep-alive :max="15">
              <component :is="Component" v-if="Component" :key="pageCacheKey" />
            </keep-alive>
          </router-view>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
$top-header-height: 48px;
$sidebar-width: 220px;
$sidebar-collapsed: 64px;
$tab-bar-height: 36px;
$sidebar-bg: #304156;

.main-layout {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.top-header {
  height: $top-header-height;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  border-bottom: 1px solid #e4e7ed;
  z-index: 100;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  flex-shrink: 0;

  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;

    .logo-icon {
      font-size: 20px;
    }
    .logo-text {
      font-size: 16px;
      font-weight: 700;
      color: #303133;
    }
    .header-divider {
      color: #dcdfe6;
      margin: 0 8px;
      font-size: 18px;
    }
  }

  .subsystem-tabs {
    display: flex;
    gap: 4px;

    .subsystem-item {
      padding: 6px 18px;
      font-size: 14px;
      color: #606266;
      cursor: pointer;
      border-radius: 4px;
      transition: all 0.2s;
      user-select: none;

      &:hover {
        background: #ecf5ff;
        color: #409eff;
      }

      &.active {
        background: #409eff;
        color: #fff;
      }
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;

    .user-info {
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 13px;
      color: #606266;
      padding: 4px 8px;
      border-radius: 4px;
      &:hover {
        background: #f5f7fa;
        color: #409eff;
      }
    }
  }
}

.main-body {
  display: flex;
  flex: 1;
  overflow: hidden;
  height: calc(100vh - #{$top-header-height});
}

.sidebar {
  width: $sidebar-width;
  background: $sidebar-bg;
  display: flex;
  flex-direction: column;
  transition: width 0.3s;
  flex-shrink: 0;
  overflow: hidden;

  &.collapsed {
    width: $sidebar-collapsed;
  }

  :deep(.el-menu) {
    border-right: none;
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .collapse-btn {
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #bfcbd9;
    cursor: pointer;
    border-top: 1px solid rgba(255, 255, 255, 0.08);

    &:hover {
      color: #fff;
      background: rgba(255, 255, 255, 0.05);
    }
  }
}

.content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tab-bar {
  height: $tab-bar-height;
  background: #fff;
  display: flex;
  align-items: stretch;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;

  .tab-list {
    flex: 1;
    display: flex;
    overflow-x: auto;
    overflow-y: hidden;
    &::-webkit-scrollbar {
      height: 2px;
    }
  }

  .tab-item {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 12px;
    font-size: 12px;
    color: #606266;
    cursor: pointer;
    border-right: 1px solid #ebeef5;
    white-space: nowrap;
    min-width: 80px;
    max-width: 160px;
    background: #fafafa;

    &:hover {
      background: #ecf5ff;
      .tab-close {
        opacity: 1;
      }
    }

    &.active {
      background: #fff;
      color: #409eff;
      border-bottom: 2px solid #409eff;
      font-weight: 500;
    }

    .tab-title {
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .tab-close {
      opacity: 0;
      border-radius: 50%;
      padding: 1px;
      flex-shrink: 0;
      &:hover {
        background: #c0c4cc;
        color: #fff;
      }
    }
  }

  .tab-close-all {
    display: flex;
    align-items: center;
    padding: 0 10px;
    cursor: pointer;
    color: #909399;
    flex-shrink: 0;
    &:hover {
      color: #f56c6c;
    }
  }
}

.page-content {
  flex: 1;
  overflow: auto;
  background: #f0f2f5;
}

.context-menu {
  position: fixed;
  z-index: 9999;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
  padding: 4px 0;
  min-width: 120px;

  .menu-item {
    padding: 8px 16px;
    font-size: 13px;
    color: #606266;
    cursor: pointer;
    &:hover {
      background: #ecf5ff;
      color: #409eff;
    }
  }
}
</style>

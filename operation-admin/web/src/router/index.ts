import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useAppStore, type Subsystem } from '@/stores/app'
import { usePermissionStore } from '@/stores/permission'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    icon?: string
    parent?: string
    parentIcon?: string
    subsystem?: Subsystem | 'common'
    permission?: string
    hidden?: boolean
    noAuth?: boolean
    closable?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', noAuth: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/Index.vue'),
        meta: {
          title: '首页',
          icon: 'HomeFilled',
          subsystem: 'common',
          permission: 'common:home',
          closable: false,
        },
      },
      {
        path: 'finance/reconcile',
        name: 'FinanceReconcile',
        component: () => import('@/views/reconcile/ReconcilePage.vue'),
        meta: {
          title: '实时对账',
          icon: 'DataAnalysis',
          parent: '财务对账',
          parentIcon: 'Wallet',
          subsystem: 'finance',
          permission: 'finance:reconcile:view',
        },
      },
      {
        path: 'finance/daily',
        name: 'FinanceDaily',
        component: () => import('@/views/finance/DailyReportPage.vue'),
        meta: {
          title: '消耗日报',
          icon: 'Calendar',
          parent: '财务对账',
          parentIcon: 'Wallet',
          subsystem: 'finance',
          permission: 'finance:daily:view',
        },
      },
      {
        path: 'finance/sync-settings',
        name: 'FinanceSyncSettings',
        component: () => import('@/views/finance/SyncSettingsPage.vue'),
        meta: {
          title: '自动同步',
          icon: 'Timer',
          parent: '财务对账',
          parentIcon: 'Wallet',
          subsystem: 'finance',
          permission: 'finance:sync-settings:view',
        },
      },
      {
        path: 'ops/overview',
        name: 'OpsOverview',
        component: () => import('@/views/ops/Overview.vue'),
        meta: {
          title: '运维总览',
          icon: 'Monitor',
          parent: '运维监控',
          parentIcon: 'SetUp',
          subsystem: 'ops',
          permission: 'ops:overview:view',
        },
      },
      {
        path: 'system/users',
        name: 'SystemUsers',
        component: () => import('@/views/system/UserManagement.vue'),
        meta: {
          title: '用户管理',
          icon: 'User',
          parent: '权限管理',
          parentIcon: 'Lock',
          subsystem: 'system',
          permission: 'system:user:list',
        },
      },
      {
        path: 'system/roles',
        name: 'SystemRoles',
        component: () => import('@/views/system/RoleManagement.vue'),
        meta: {
          title: '角色管理',
          icon: 'UserFilled',
          parent: '权限管理',
          parentIcon: 'Lock',
          subsystem: 'system',
          permission: 'system:role:list',
        },
      },
      {
        path: 'system/menus',
        name: 'SystemMenus',
        component: () => import('@/views/system/MenuManagement.vue'),
        meta: {
          title: '菜单管理',
          icon: 'Menu',
          parent: '权限管理',
          parentIcon: 'Lock',
          subsystem: 'system',
          permission: 'system:menu:list',
        },
      },
      {
        path: 'system/notify-channels',
        name: 'NotifyChannels',
        component: () => import('@/views/system/NotifyChannelManagement.vue'),
        meta: {
          title: '通知渠道',
          icon: 'Message',
          parent: '通知管理',
          parentIcon: 'Bell',
          subsystem: 'system',
          permission: 'system:notify-channel:list',
        },
      },
      {
        path: 'system/notify-groups',
        name: 'NotifyGroups',
        component: () => import('@/views/system/NotifyGroupManagement.vue'),
        meta: {
          title: '通知组',
          icon: 'Collection',
          parent: '通知管理',
          parentIcon: 'Bell',
          subsystem: 'system',
          permission: 'system:notify-group:list',
        },
      },
      {
        path: 'system/notify-rules',
        name: 'NotifyRules',
        component: () => import('@/views/system/NotifyRuleManagement.vue'),
        meta: {
          title: '通知规则',
          icon: 'AlarmClock',
          parent: '通知管理',
          parentIcon: 'Bell',
          subsystem: 'system',
          permission: 'system:notify-rule:list',
        },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const userStore = useUserStore()
  const permStore = usePermissionStore()

  if (to.meta.noAuth) {
    if (userStore.isLoggedIn() && to.path === '/login') {
      return '/'
    }
    return true
  }
  if (!userStore.isLoggedIn()) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (!permStore.loaded && userStore.token) {
    try {
      await userStore.fetchMe()
    } catch {
      userStore.logout()
      return { path: '/login' }
    }
  }

  const sub = to.meta.subsystem as Subsystem | 'common' | undefined
  if (sub && sub !== 'common') {
    useAppStore().switchSubsystem(sub)
  }

  if (!permStore.canAccessRoute(to.path, to.meta.permission as string | undefined)) {
    return '/dashboard'
  }
  return true
})

router.afterEach((to) => {
  document.title = `${(to.meta.title as string) || 'Operation Admin'} — 运营管理`
})

export default router

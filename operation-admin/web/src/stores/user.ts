import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { AuthUser, LoginResult } from '@/types'
import { loginApi, login2faApi, meApi } from '@/api/finance'
import { usePermissionStore } from '@/stores/permission'

const TOKEN_KEY = 'ops_token'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const user = ref<AuthUser | null>(null)

  const isLoggedIn = () => !!token.value

  function applyUser(nextUser: AuthUser | null) {
    user.value = nextUser
    usePermissionStore().setFromUserInfo({
      menu_paths: nextUser?.menu_paths,
      permissions: nextUser?.permissions,
      super_admin: nextUser?.super_admin,
    })
  }

  function setSession(nextToken: string, nextUser: AuthUser | null) {
    token.value = nextToken
    localStorage.setItem(TOKEN_KEY, nextToken)
    applyUser(nextUser)
  }

  async function login(params: { username?: string; password?: string; pat?: string }) {
    const data = await loginApi(params)
    if (data.require_2fa) {
      return data
    }
    if (!data.token) {
      throw new Error('missing token')
    }
    setSession(data.token, data.user || null)
    return data
  }

  async function login2fa(flowToken: string, code: string) {
    const data = await login2faApi({ flow_token: flowToken, code })
    if (!data.token) {
      throw new Error('missing token')
    }
    setSession(data.token, data.user || null)
    return data
  }

  async function fetchMe() {
    if (!token.value) return
    const me = await meApi()
    applyUser(me)
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    usePermissionStore().reset()
  }

  return {
    token,
    user,
    isLoggedIn,
    login,
    login2fa,
    fetchMe,
    logout,
  }
})

export type { LoginResult }

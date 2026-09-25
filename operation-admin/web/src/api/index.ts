import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'

export type ApiEnvelope<T = unknown> = {
  success: boolean
  message?: string
  data: T
}

const service: AxiosInstance = axios.create({
  baseURL: '',
  timeout: 60000,
})

service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('ops_token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

service.interceptors.response.use(
  (response: AxiosResponse) => {
    const body = response.data as ApiEnvelope
    if (body && typeof body === 'object' && 'success' in body && !body.success) {
      const msg = body.message || '请求失败'
      ElMessage.error(msg)
      return Promise.reject(new Error(msg))
    }
    return response
  },
  (error) => {
    const status = error.response?.status
    const msg = error.response?.data?.message || error.message || '网络请求失败'
    if (status === 401) {
      localStorage.removeItem('ops_token')
      if (!router.currentRoute.value.path.includes('login')) {
        ElMessage.error('登录已过期，请重新登录')
        router.push('/login')
      }
    } else if (status === 403) {
      ElMessage.error(msg || '没有权限')
    } else {
      ElMessage.error(msg)
    }
    return Promise.reject(new Error(msg))
  }
)

export async function apiData<T>(promise: Promise<AxiosResponse<ApiEnvelope<T>>>) {
  const res = await promise
  return res.data.data
}

export default service

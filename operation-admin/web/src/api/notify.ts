import service, { apiData } from './index'

export type NotifyChannel = {
  id: number
  name: string
  type: 'email' | 'dingtalk' | string
  status?: string
  remark?: string
  config?: Record<string, unknown>
  created_at?: string
  updated_at?: string
}

export type NotifyGroup = {
  id: number
  name: string
  code?: string
  status?: string
  remark?: string
  channel_ids?: number[]
  channel_names?: string[]
  created_at?: string
  updated_at?: string
}

export type NotifyRule = {
  id: number
  name: string
  status?: string
  remark?: string
  event_type?: string
  threshold_usd?: number
  notify_group_id?: number
  notify_group_name?: string
  scope?: string
  channel_ids?: number[]
  cooldown_minutes?: number
  created_at?: string
  updated_at?: string
}

export function listNotifyChannelsApi(params?: { type?: string; status?: string; name?: string }) {
  return apiData<NotifyChannel[]>(service.get('/api/system/notify-channels', { params }))
}

export function createNotifyChannelApi(data: Partial<NotifyChannel>) {
  return apiData(service.post('/api/system/notify-channels', data))
}

export function updateNotifyChannelApi(id: number, data: Partial<NotifyChannel>) {
  return apiData(service.put(`/api/system/notify-channels/${id}`, data))
}

export function deleteNotifyChannelApi(id: number) {
  return apiData(service.delete(`/api/system/notify-channels/${id}`))
}

export function listNotifyGroupsApi(params?: { name?: string }) {
  return apiData<NotifyGroup[]>(service.get('/api/system/notify-groups', { params }))
}

export function createNotifyGroupApi(data: Partial<NotifyGroup>) {
  return apiData(service.post('/api/system/notify-groups', data))
}

export function updateNotifyGroupApi(id: number, data: Partial<NotifyGroup>) {
  return apiData(service.put(`/api/system/notify-groups/${id}`, data))
}

export function deleteNotifyGroupApi(id: number) {
  return apiData(service.delete(`/api/system/notify-groups/${id}`))
}

export function listNotifyRulesApi(params?: { name?: string; status?: string }) {
  return apiData<NotifyRule[]>(service.get('/api/system/notify-rules', { params }))
}

export function createNotifyRuleApi(data: Partial<NotifyRule>) {
  return apiData(service.post('/api/system/notify-rules', data))
}

export function updateNotifyRuleApi(id: number, data: Partial<NotifyRule>) {
  return apiData(service.put(`/api/system/notify-rules/${id}`, data))
}

export function deleteNotifyRuleApi(id: number) {
  return apiData(service.delete(`/api/system/notify-rules/${id}`))
}

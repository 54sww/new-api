import service, { apiData } from './index'

export type SysMenu = {
  id: number
  parent_id?: number | null
  name: string
  path?: string
  permission?: string
  icon?: string
  parent_icon?: string
  subsystem?: string
  menu_type?: 'M' | 'C' | 'F'
  menu_group?: string
  sort?: number
  status?: string
  hidden?: boolean
  children?: SysMenu[]
}

export type SysRole = {
  id: number
  code: string
  name: string
  status?: string
  remark?: string
  menu_ids?: number[]
}

export type SysUserVO = {
  id: number
  username: string
  real_name?: string
  phone?: string
  email?: string
  status?: string
  role_ids?: number[]
  role_names?: string[]
  created_at?: string
}

export function getMenuTreeApi(subsystem?: string) {
  return apiData<SysMenu[]>(service.get('/api/system/menus/tree', { params: { subsystem } }))
}

export function createMenuApi(data: Partial<SysMenu>) {
  return apiData(service.post('/api/system/menus', data))
}

export function updateMenuApi(id: number, data: Partial<SysMenu>) {
  return apiData(service.put(`/api/system/menus/${id}`, data))
}

export function deleteMenuApi(id: number) {
  return apiData(service.delete(`/api/system/menus/${id}`))
}

export function getRolesApi(name?: string) {
  return apiData<SysRole[]>(service.get('/api/system/roles', { params: { name } }))
}

export function getRoleDetailApi(id: number) {
  return apiData<SysRole>(service.get(`/api/system/roles/${id}`))
}

export function createRoleApi(data: Partial<SysRole>) {
  return apiData(service.post('/api/system/roles', data))
}

export function updateRoleApi(id: number, data: Partial<SysRole>) {
  return apiData(service.put(`/api/system/roles/${id}`, data))
}

export function deleteRoleApi(id: number) {
  return apiData(service.delete(`/api/system/roles/${id}`))
}

export function assignRoleMenusApi(id: number, menuIds: number[]) {
  return apiData(service.put(`/api/system/roles/${id}/menus`, { menu_ids: menuIds }))
}

export function getUsersApi(params: {
  page?: number
  page_size?: number
  username?: string
  status?: string
}) {
  return apiData<{ records: SysUserVO[]; total: number }>(
    service.get('/api/system/users', { params })
  )
}

export function createUserApi(data: Record<string, unknown>) {
  return apiData(service.post('/api/system/users', data))
}

export function updateUserApi(id: number, data: Record<string, unknown>) {
  return apiData(service.put(`/api/system/users/${id}`, data))
}

export function deleteUserApi(id: number) {
  return apiData(service.delete(`/api/system/users/${id}`))
}

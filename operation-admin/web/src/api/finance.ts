import service, { apiData } from './index'
import type { AuthUser, BalanceQueryConfig, LoginResult, RechargeRow, ReconcileResult } from '@/types'

export function loginApi(body: { username?: string; password?: string; pat?: string }) {
  return apiData<LoginResult>(service.post('/api/auth/login', body))
}

export function login2faApi(body: { flow_token: string; code: string }) {
  return apiData<LoginResult>(service.post('/api/auth/login/2fa', body))
}

export function meApi() {
  return apiData<AuthUser>(service.get('/api/me'))
}

export function liveReconcileApi() {
  return apiData<ReconcileResult>(service.get('/api/reconcile/live'))
}

export function listRechargesApi() {
  return apiData<RechargeRow[]>(service.get('/api/recharges'))
}

export function createRechargeApi(body: {
  channel_id: number
  amount_usd: number
  voucher?: string
  note?: string
}) {
  return apiData(service.post('/api/recharges', body))
}

export function deleteRechargeApi(id: number) {
  return apiData(service.delete(`/api/recharges/${id}`))
}

export function syncChannelsApi() {
  return apiData(service.post('/api/sync/channels'))
}

export function syncBalancesApi() {
  return apiData(service.post('/api/sync/balances'))
}

export function syncStatusApi() {
  return apiData(service.get('/api/sync/status'))
}

export function setOpeningBalanceApi(id: number, opening_balance: number) {
  return apiData(service.put(`/api/channels/${id}/opening-balance`, { opening_balance }))
}

export function refreshBalanceApi(id: number) {
  return apiData<{
    balance: number
    balance_raw: number
    balance_currency?: string
  }>(service.post(`/api/channels/${id}/balance-refresh`))
}

export function getBalanceQueryApi(id: number) {
  return apiData<{
    config: BalanceQueryConfig
    channel_base_url?: string
    has_api_key?: boolean
    sync_balance?: boolean
    notify_enabled?: boolean
    alias?: string
  }>(service.get(`/api/channels/${id}/balance-query`))
}

export function putBalanceQueryApi(
  id: number,
  body: {
    sync_balance: boolean
    notify_enabled: boolean
    alias?: string
    config: BalanceQueryConfig
  }
) {
  return apiData(service.put(`/api/channels/${id}/balance-query`, body))
}

export function testBalanceQueryApi(id: number, cfg: BalanceQueryConfig) {
  return apiData<{
    amount_raw: number
    currency: string
    amount_usd: number
    raw_body?: string
  }>(service.post(`/api/channels/${id}/balance-query/test`, cfg))
}

export function balanceQueryPresetsApi() {
  return apiData<Array<{ id: string; config: BalanceQueryConfig }>>(
    service.get('/api/balance-query/presets')
  )
}

export type BalanceSyncSettings = {
  enabled: boolean
  interval_minutes: number
  last_sync_at?: number
  last_sync_count?: number
  last_sync_message?: string
  updated_at?: number
}

export function getBalanceSyncSettingsApi() {
  return apiData<BalanceSyncSettings>(service.get('/api/settings/balance-sync'))
}

export function putBalanceSyncSettingsApi(body: { enabled?: boolean; interval_minutes?: number }) {
  return apiData<BalanceSyncSettings>(service.put('/api/settings/balance-sync', body))
}

export type DailyReportRow = {
  channel_id: number
  channel_name: string
  sync_balance?: boolean
  day_key: string
  start_balance: number
  start_at: number
  end_balance: number
  end_at: number
  recharge_usd: number
  consume_usd: number
  snapshot_cnt: number
  status: 'ok' | 'partial' | 'in_progress' | 'no_data'
  status_note: string
}

export type DailySummaryRow = {
  day_key: string
  channel_count: number
  ok_count: number
  partial_count: number
  in_progress_count: number
  consume_total: number
  recharge_total: number
  start_total: number
  end_total: number
}

export type DailyReportResult = {
  from: string
  to: string
  by_day: DailySummaryRow[]
  items: DailyReportRow[]
  summary: {
    channel_count: number
    day_count: number
    row_count: number
    ok_count: number
    partial_count: number
    in_progress_count: number
    consume_total: number
    recharge_total: number
    start_total: number
    end_total: number
  }
}

export type SnapshotPoint = {
  id: number
  balance: number
  balance_raw: number
  currency: string
  source: string
  synced_at: number
  day_key: string
}

export function dailyReportApi(params?: {
  from?: string
  to?: string
  day?: string
  channel_id?: number
  only_sync?: boolean
}) {
  return apiData<DailyReportResult>(
    service.get('/api/reports/daily', {
      params: {
        from: params?.from,
        to: params?.to,
        day: params?.day,
        channel_id: params?.channel_id,
        only_sync: params?.only_sync === false ? '0' : '1',
      },
    })
  )
}

export function dailySnapshotsApi(params: { channel_id: number; day: string }) {
  return apiData<SnapshotPoint[]>(service.get('/api/reports/daily/snapshots', { params }))
}

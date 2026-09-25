export type AuthUser = {
  id?: number
  username?: string
  display_name?: string
  super_admin?: boolean
  auth_source?: string
  permissions?: string[]
  menu_paths?: string[]
}

export type LoginResult = {
  require_2fa?: boolean
  flow_token?: string
  token?: string
  user?: AuthUser
}

export type ReconcileRow = {
  channel_id: number
  name: string
  upstream_balance: number
  balance_raw?: number
  balance_currency?: string
  opening_balance: number
  recharge_total: number
  consume_usd: number
  theory_balance: number
  diff_usd: number
  has_balance_query?: boolean
  sync_balance?: boolean
  notify_enabled?: boolean
}

export type ReconcileResult = {
  summary: {
    channel_count: number
    upstream_total: number
    recharge_total: number
    consume_usd: number
    theory_total: number
    diff_total: number
  }
  items: ReconcileRow[]
}

export type RechargeRow = {
  id: number
  channel_id: number
  amount_usd: number
  recharged_at: number
  voucher?: string
  note?: string
  created_by?: string
}

export type BalanceQueryConfig = {
  enabled?: boolean
  base_url?: string
  url?: string
  method?: string
  amount_path?: string
  currency_path?: string
  currency?: string
  fx_to_usd?: number
  headers?: Record<string, string>
  body?: string
  api_key?: string
  timeout_sec?: number
}

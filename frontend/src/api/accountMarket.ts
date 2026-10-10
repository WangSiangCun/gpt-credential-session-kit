import { apiClient } from './client'

export interface AccountMarketConfig {
  enabled: boolean
  target_group_id?: number | null
  target_group_name?: string
  commission_percent: number
  minimum_withdrawal: number
  withdrawal_review_required: boolean
}

export interface AccountMarketAccount {
  market_id: number
  account_id: number
  uploader_user_id?: number
  uploader_email?: string
  name: string
  platform: string
  type: string
  email?: string
  market_status: 'active' | 'paused' | 'pending' | 'rejected'
  account_status: string
  schedulable: boolean
  error_message?: string
  concurrency: number
  rate_multiplier: number
  group_id: number
  group_name: string
  group_rate_multiplier: number
  proxy_id?: number
  proxy_name?: string
  proxy_protocol?: string
  quota?: unknown
  reset_at?: unknown
  last_used_at?: string
  created_at: string
  total_earned: number
}

export interface AccountMarketProxy {
  id: number
  name: string
  protocol: string
  expires_at?: string
}

export interface AccountMarketSummary {
  available: number
  locked: number
  withdrawn: number
  total: number
  account_count: number
}

export interface AccountMarketWithdrawal {
  id: number
  user_id: number
  user_email?: string
  gross_amount: number
  commission_percent: number
  commission_amount: number
  net_amount: number
  payout_method: string
  payout_account: string
  status: 'pending' | 'approved' | 'paid' | 'rejected' | 'cancelled'
  review_note?: string
  reviewed_at?: string
  paid_at?: string
  created_at: string
}

export interface UploadResult {
  index: number
  success: boolean
  account_id?: number
  name: string
  email?: string
  group_name?: string
  status?: string
  error?: string
}

export interface AccountMarketOAuthURL {
  auth_url: string
  session_id: string
}

export interface AccountMarketOAuthToken {
  access_token: string
  refresh_token?: string
  id_token?: string
  expires_at?: number
  email?: string
  name?: string
  chatgpt_account_id?: string
  chatgpt_user_id?: string
  organization_id?: string
  plan_type?: string
  client_id?: string
}

export interface AccountMarketQuotaWindow {
  used_percent: number
  limit_window_seconds: number
  reset_after_seconds: number
  reset_at: number
}

export interface AccountMarketQuotaUsage {
  rate_limit?: { primary_window?: AccountMarketQuotaWindow | null; secondary_window?: AccountMarketQuotaWindow | null } | null
  rate_limit_reset_credits?: { available_count: number; credits?: Array<{ expires_at?: string }> } | null
  credits?: { has_credits: boolean; unlimited: boolean; balance: string | null } | null
  fetched_at: number
}

export interface AccountMarketQuotaRefreshResult extends AccountMarketQuotaUsage {
  cache_persisted: boolean
  credits_cache_persisted: boolean
}

export interface AccountMarketQuotaResetResult {
  code: string
  windows_reset: number
  quota?: AccountMarketQuotaUsage | null
  cache_refreshed: boolean
  account_state_recovered: boolean
  warning_code?: string
}

export async function getConfig(): Promise<AccountMarketConfig> {
  const { data } = await apiClient.get<AccountMarketConfig>('/account-market/config')
  return data
}

export async function listProxies(): Promise<AccountMarketProxy[]> {
  const { data } = await apiClient.get<{ items: AccountMarketProxy[] }>('/account-market/proxies')
  return data.items || []
}

export async function upload(payload: unknown, proxyId: number | null = null): Promise<UploadResult[]> {
  const { data } = await apiClient.post<{ items: UploadResult[] }>('/account-market/upload', payload, {
    params: proxyId ? { proxy_id: proxyId } : undefined,
  })
  return data.items || []
}

export async function generateOAuthURL(proxyId: number | null = null): Promise<AccountMarketOAuthURL> {
  const { data } = await apiClient.post<AccountMarketOAuthURL>('/account-market/oauth/generate-auth-url', {
    proxy_id: proxyId || undefined,
  })
  return data
}

export async function exchangeOAuthCode(payload: {
  session_id: string
  code: string
  state: string
  proxy_id?: number
}): Promise<AccountMarketOAuthToken> {
  const { data } = await apiClient.post<AccountMarketOAuthToken>('/account-market/oauth/exchange-code', payload)
  return data
}

export async function listAccounts(): Promise<AccountMarketAccount[]> {
  const { data } = await apiClient.get<{ items: AccountMarketAccount[] }>('/account-market/accounts')
  return data.items || []
}

export async function updateAccount(id: number, concurrency: number): Promise<AccountMarketAccount> {
  const { data } = await apiClient.put<AccountMarketAccount>(`/account-market/accounts/${id}`, { concurrency })
  return data
}

export async function refreshQuota(id: number): Promise<AccountMarketQuotaRefreshResult> {
  const { data } = await apiClient.post<AccountMarketQuotaRefreshResult>(`/account-market/accounts/${id}/quota/refresh`)
  return data
}

export async function resetQuota(id: number): Promise<AccountMarketQuotaResetResult> {
  const { data } = await apiClient.post<AccountMarketQuotaResetResult>(`/account-market/accounts/${id}/reset-quota`, undefined, { timeout: 90_000 })
  return data
}

export async function setSchedulable(id: number, enabled: boolean): Promise<void> {
  await apiClient.put(`/account-market/accounts/${id}/schedulable`, { enabled })
}

export async function removeAccount(id: number): Promise<void> {
  await apiClient.delete(`/account-market/accounts/${id}`)
}

export async function exportAccount(id: number): Promise<Record<string, unknown>> {
  const { data } = await apiClient.get<Record<string, unknown>>(`/account-market/accounts/${id}/export`)
  return data
}

export async function getSummary(): Promise<AccountMarketSummary> {
  const { data } = await apiClient.get<AccountMarketSummary>('/account-market/summary')
  return data
}

export async function listWithdrawals(): Promise<AccountMarketWithdrawal[]> {
  const { data } = await apiClient.get<{ items: AccountMarketWithdrawal[] }>('/account-market/withdrawals')
  return data.items || []
}

export async function requestWithdrawal(payload: { payout_method: string; payout_account: string }): Promise<AccountMarketWithdrawal> {
  const { data } = await apiClient.post<AccountMarketWithdrawal>('/account-market/withdrawals', payload)
  return data
}

export async function cancelWithdrawal(id: number): Promise<void> {
  await apiClient.post(`/account-market/withdrawals/${id}/cancel`)
}

export const accountMarketAPI = {
  getConfig,
  listProxies,
  upload,
  generateOAuthURL,
  exchangeOAuthCode,
  listAccounts,
  updateAccount,
  refreshQuota,
  resetQuota,
  setSchedulable,
  removeAccount,
  exportAccount,
  getSummary,
  listWithdrawals,
  requestWithdrawal,
  cancelWithdrawal,
}

export default accountMarketAPI

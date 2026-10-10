import { apiClient } from '../client'
import type { AccountMarketAccount, AccountMarketConfig, AccountMarketWithdrawal } from '../accountMarket'

export async function getConfig(): Promise<AccountMarketConfig> {
  const { data } = await apiClient.get<AccountMarketConfig>('/admin/account-market/config')
  return data
}

export async function updateConfig(payload: AccountMarketConfig): Promise<AccountMarketConfig> {
  const { data } = await apiClient.put<AccountMarketConfig>('/admin/account-market/config', payload)
  return data
}

export async function listAccounts(): Promise<AccountMarketAccount[]> {
  const { data } = await apiClient.get<{ items: AccountMarketAccount[] }>('/admin/account-market/accounts')
  return data.items || []
}

export async function listWithdrawals(): Promise<AccountMarketWithdrawal[]> {
  const { data } = await apiClient.get<{ items: AccountMarketWithdrawal[] }>('/admin/account-market/withdrawals')
  return data.items || []
}

export async function reviewWithdrawal(id: number, status: 'approved' | 'paid' | 'rejected', note = ''): Promise<void> {
  await apiClient.put(`/admin/account-market/withdrawals/${id}`, { status, note })
}

export const adminAccountMarketAPI = { getConfig, updateConfig, listAccounts, listWithdrawals, reviewWithdrawal }
export default adminAccountMarketAPI

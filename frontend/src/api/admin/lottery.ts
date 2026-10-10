import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export interface LotteryPrize {
  id: number
  name: string
  min_amount: number
  max_amount: number
  total_quantity: number
  remaining_quantity: number
  weight: number
}

export interface LotteryCampaign {
  id: number
  token: string
  name: string
  description: string
  activity_type: 'instant_lottery' | 'registration_lottery'
  status: 'active' | 'scheduled' | 'stopped' | 'exhausted' | 'expired'
  win_rate: number
  min_total_recharged: number
  starts_at: string
  expires_at: string
  registration_deadline?: string | null
  draw_at?: string | null
  winner_limit?: number
  winner_count: number
  draw_count: number
  prizes: LotteryPrize[]
}

export interface CreateLotteryCampaign {
  name: string
  description: string
  activity_type: 'instant_lottery' | 'registration_lottery'
  win_rate: number
  min_total_recharged: number
  starts_at?: string
  expires_at?: string
  registration_deadline?: string
  draw_at?: string
  winner_limit?: number
  prizes?: Array<{
    name: string
    min_amount: number
    max_amount: number
    quantity: number
    weight: number
  }>
}

export interface LotteryWinner {
  id: number
  user_id: number
  prize_name: string
  amount: number
  created_at: string
  user_email: string
  username: string
  user_balance: number
}

export const lotteryAPI = {
  async list(page = 1, pageSize = 20): Promise<BasePaginationResponse<LotteryCampaign>> {
    const { data } = await apiClient.get('/admin/lottery-campaigns', {
      params: { page, page_size: pageSize }
    })
    return data
  },
  async create(input: CreateLotteryCampaign): Promise<LotteryCampaign> {
    const { data } = await apiClient.post('/admin/lottery-campaigns', input)
    return data
  },
  async get(id: number): Promise<LotteryCampaign> {
    const { data } = await apiClient.get(`/admin/lottery-campaigns/${id}`)
    return data
  },
  async stop(id: number): Promise<void> {
    await apiClient.post(`/admin/lottery-campaigns/${id}/stop`)
  },
  async remove(id: number): Promise<void> {
    await apiClient.delete(`/admin/lottery-campaigns/${id}`)
  },
  async winners(id: number, page = 1, pageSize = 20): Promise<BasePaginationResponse<LotteryWinner>> {
    const { data } = await apiClient.get(`/admin/lottery-campaigns/${id}/winners`, {
      params: { page, page_size: pageSize }
    })
    return data
  }
}

export default lotteryAPI

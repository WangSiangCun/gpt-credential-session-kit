import { apiClient } from './client'
import type { BasePaginationResponse } from '@/types'

export interface LotteryPublicPrize {
  id: number
  name: string
  min_amount: number
  max_amount: number
  total_quantity: number
  remaining_quantity: number
}

export interface LotteryPublicCampaign {
  id: number
  name: string
  description: string
  activity_type: 'instant_lottery' | 'registration_lottery'
  status: 'active' | 'scheduled' | 'stopped' | 'exhausted' | 'expired'
  win_rate: number
  min_total_recharged?: number
  starts_at: string
  expires_at: string
  registration_deadline?: string | null
  draw_at?: string | null
  winner_limit?: number
  winner_count: number
  prizes: LotteryPublicPrize[]
}

export interface LotteryDraw {
  id: number
  status: 'won' | 'lost' | 'registered'
  prize_name?: string
  amount: number
  created_at: string
  new_balance?: number
}

export interface LotteryPublicState {
  campaign: LotteryPublicCampaign
  my_draw: LotteryDraw | null
}

export interface LotteryPublicWinner {
  id: number
  masked_account: string
  prize_name: string
  amount: number
  created_at: string
}

export async function listLotteries(page = 1, pageSize = 50): Promise<BasePaginationResponse<LotteryPublicCampaign>> {
  const { data } = await apiClient.get('/lotteries', { params: { page, page_size: pageSize } })
  return data
}

export async function getLottery(token: string): Promise<LotteryPublicState> {
  const { data } = await apiClient.get(`/lotteries/${token}`)
  return data
}

export async function drawLottery(token: string): Promise<LotteryDraw> {
  const { data } = await apiClient.post(`/lotteries/${token}/draw`)
  return data
}

export async function listLotteryWinners(
  campaignId: number,
  page = 1,
  pageSize = 100,
): Promise<BasePaginationResponse<LotteryPublicWinner>> {
	const { data } = await apiClient.get(`/lotteries/campaigns/${campaignId}/winners`, {
    params: { page, page_size: pageSize },
  })
  return data
}

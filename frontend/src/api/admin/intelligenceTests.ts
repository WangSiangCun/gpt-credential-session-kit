import { apiClient } from '../client'

export interface IntelligenceTestResult {
  group_id: number
  model?: string
  reasoning_effort?: string
  svg: string
  status: 'success' | 'error'
  error?: string
  tested_at: string
  account_id?: number
}

export interface IntelligenceTestsState {
  prompt: string
  frequency_mins: number
  timeout_seconds: number
  proxy_id?: number | null
  public_group_filter_enabled: boolean
  public_group_ids: number[]
  proxies: IntelligenceTestProxy[]
  groups: IntelligenceTestGroup[]
  results: Record<string, IntelligenceTestResult>
}

export interface IntelligenceTestProxy {
  id: number
  name: string
  protocol: string
  host: string
  port: number
}

export type IntelligenceGroupConfigs = Record<string, { model?: string; reasoning_effort?: string }>

export interface IntelligenceTestGroup {
  id: number
  name: string
  platform: string
  status: string
  test_model: string
  test_reasoning_effort: string
  available_test_models: string[]
  available_reasoning_efforts: string[]
}

export async function getIntelligenceTests(): Promise<IntelligenceTestsState> {
  const { data } = await apiClient.get<IntelligenceTestsState>('/admin/intelligence-tests')
  return data
}

export async function getPublicIntelligenceTests(): Promise<Pick<IntelligenceTestsState, 'groups' | 'results'>> {
  const { data } = await apiClient.get<Pick<IntelligenceTestsState, 'groups' | 'results'>>('/intelligence-tests')
  return data
}

export async function updateIntelligenceTests(payload: { prompt: string; frequency_mins: number; timeout_seconds: number; proxy_id?: number | null; public_group_filter_enabled: boolean; public_group_ids: number[]; group_configs?: IntelligenceGroupConfigs }) {
  const { data } = await apiClient.put<IntelligenceTestsState & { group_configs: IntelligenceGroupConfigs }>('/admin/intelligence-tests', payload)
  return data
}

export async function runIntelligenceTest(groupId: number): Promise<IntelligenceTestResult> {
  // A configured test may legitimately take several minutes; the global UI timeout is 30s.
  const { data } = await apiClient.post<IntelligenceTestResult>(`/admin/intelligence-tests/${groupId}/run`, undefined, { timeout: 1_860_000 })
  return data
}

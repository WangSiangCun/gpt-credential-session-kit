import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { drawLottery, getLottery } from '@/api/lottery'
import LotteryView from '../LotteryView.vue'

vi.mock('@/api/lottery', () => ({
  drawLottery: vi.fn(),
  getLottery: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { token: 'test-token' } }),
}))

const AppLayoutStub = { template: '<div><slot /></div>' }

function mountView() {
  return mount(LotteryView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        'router-link': { template: '<a><slot /></a>' },
      },
    },
  })
}

const registrationCampaign = {
  id: 8,
  name: '报名开奖',
  description: '',
  activity_type: 'registration_lottery' as const,
  status: 'active' as const,
  win_rate: 0,
  starts_at: '2026-08-06T00:00:00Z',
  expires_at: '2026-08-08T00:00:00Z',
  registration_deadline: '2026-08-06T12:00:00Z',
  draw_at: '2026-08-07T12:00:00Z',
  winner_limit: 3,
  winner_count: 0,
  prizes: [],
}

describe('LotteryView interaction guards', () => {
  beforeEach(() => {
    vi.mocked(drawLottery).mockReset()
    vi.mocked(getLottery).mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('disables registration after the deadline and explains the waiting state', async () => {
    vi.mocked(getLottery).mockResolvedValue({ campaign: registrationCampaign, my_draw: null })
    const wrapper = mountView()
    await flushPromises()

    const button = wrapper.get('.draw-button')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.text()).toContain('报名已截止')
    expect(wrapper.get('.registration-copy strong').text()).toBe('报名已截止')
    expect(vi.mocked(drawLottery)).not.toHaveBeenCalled()
  })

  it('shows the server error after a failed draw instead of leaving the machine silent', async () => {
    vi.useFakeTimers()
    vi.mocked(getLottery).mockResolvedValue({
      campaign: {
        ...registrationCampaign,
        activity_type: 'instant_lottery',
        status: 'active',
        registration_deadline: null,
        draw_at: null,
        expires_at: '2026-08-08T00:00:00Z',
        prizes: [{ id: 1, name: '余额', min_amount: 1, max_amount: 1, total_quantity: 1, remaining_quantity: 1 }],
      },
      my_draw: null,
    })
    vi.mocked(drawLottery).mockRejectedValue(new Error('活动已结束'))
    const wrapper = mountView()
    await flushPromises()

    const trigger = wrapper.get('.draw-button')
    const triggerPromise = trigger.trigger('click')
    await vi.advanceTimersByTimeAsync(1900)
    await triggerPromise
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('活动已结束')
    expect(wrapper.get('.draw-button').text()).toContain('拉下拉杆')
  })
})

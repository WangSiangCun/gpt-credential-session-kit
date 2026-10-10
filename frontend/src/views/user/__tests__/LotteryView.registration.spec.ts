import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { drawLottery, getLottery } from '@/api/lottery'
import LotteryView from '../LotteryView.vue'

vi.mock('@/api/lottery', () => ({
  drawLottery: vi.fn(),
  getLottery: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { token: 'registration-token' } }),
}))

const campaign = {
  id: 8,
  name: 'Registration campaign',
  description: 'Register once and wait for the scheduled draw.',
  activity_type: 'registration_lottery' as const,
  status: 'active' as const,
  win_rate: 0,
  starts_at: '2026-08-06T00:00:00Z',
  expires_at: '2026-08-10T00:00:00Z',
  registration_deadline: '2026-08-08T12:00:00Z',
  draw_at: '2026-08-09T12:00:00Z',
  winner_limit: 3,
  winner_count: 0,
  prizes: [],
}

describe('LotteryView registration activity', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T00:00:00Z'))
    vi.mocked(getLottery).mockResolvedValue({ campaign, my_draw: null })
    vi.mocked(drawLottery).mockResolvedValue({
      id: 10,
      status: 'registered',
      amount: 0,
      created_at: '2026-08-07T00:00:00Z',
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('uses a confirmation animation without the lottery machine', async () => {
    const wrapper = mount(LotteryView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          'router-link': { template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('.registration-entry').exists()).toBe(true)
    expect(wrapper.find('.reel-track').exists()).toBe(false)
    expect(wrapper.find('.lever').exists()).toBe(false)

    const submitPromise = wrapper.get('.draw-button').trigger('click')
    await vi.advanceTimersByTimeAsync(650)
    await submitPromise
    await flushPromises()

    expect(drawLottery).toHaveBeenCalledWith('registration-token')
    expect(wrapper.find('.registration-entry.registered').exists()).toBe(true)
    expect(wrapper.find('.pixel-fireworks').exists()).toBe(false)
  })

  it('shows a clear winning result after the scheduled draw', async () => {
    vi.mocked(getLottery).mockResolvedValue({
      campaign,
      my_draw: {
        id: 11,
        status: 'won',
        amount: 0,
        created_at: '2026-08-09T12:00:00Z',
      },
    })

    const wrapper = mount(LotteryView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          'router-link': { template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('.registration-entry.won').exists()).toBe(true)
    expect(wrapper.text()).toContain('恭喜你，中奖啦')
    expect(wrapper.get('.draw-button').text()).toContain('已中奖')
    expect(wrapper.text()).not.toContain('报名参加活动')
  })

  it('shows a clear non-winning result after the scheduled draw', async () => {
    vi.mocked(getLottery).mockResolvedValue({
      campaign,
      my_draw: {
        id: 12,
        status: 'lost',
        amount: 0,
        created_at: '2026-08-09T12:00:00Z',
      },
    })

    const wrapper = mount(LotteryView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          'router-link': { template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('.registration-entry.lost').exists()).toBe(true)
    expect(wrapper.text()).toContain('本次未中奖')
    expect(wrapper.get('.draw-button').text()).toContain('未中奖')
    expect(wrapper.text()).not.toContain('报名参加活动')
  })
})

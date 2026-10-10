import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { listLotteries, listLotteryWinners } from '@/api/lottery'
import ActivitiesView from '../ActivitiesView.vue'

vi.mock('@/api/lottery', () => ({
  listLotteries: vi.fn(),
  listLotteryWinners: vi.fn(),
}))

const PixelDialogStub = {
  props: ['show', 'title'],
  template: `
    <section v-if="show" data-testid="pixel-dialog">
      <h2>{{ title }}</h2>
      <slot />
      <slot name="footer" />
    </section>
  `,
}

const QQGroupButtonStub = {
  data: () => ({ open: false }),
  methods: {
    openDialog() {
      this.open = true
    },
  },
  template: `
    <div>
      <slot name="trigger" :open="openDialog" />
      <section v-if="open" data-testid="qq-group-dialog">QQ群弹窗</section>
    </div>
  `,
}

function mountView() {
  return mount(ActivitiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        PixelDialog: PixelDialogStub,
        QQGroupButton: QQGroupButtonStub,
      },
    },
  })
}

describe('user ActivitiesView participation action', () => {
  beforeEach(() => {
    vi.mocked(listLotteries).mockReset()
    vi.mocked(listLotteryWinners).mockReset()
  })

  it('keeps the header action enabled and opens the shared QQ group dialog', async () => {
    vi.mocked(listLotteries).mockResolvedValue({ items: [] } as never)
    const wrapper = mountView()
    await flushPromises()

    const participateButton = wrapper.get('.activity-hero .pixel-action--primary')
    expect(participateButton.attributes('disabled')).toBeUndefined()
    expect(participateButton.text()).toContain('参加活动')
    expect(participateButton.text()).not.toContain('参加抽奖')

    await participateButton.trigger('click')
    expect(wrapper.get('[data-testid="qq-group-dialog"]').text()).toContain('QQ群弹窗')
  })

  it('keeps campaign cards read-only and loads winners by campaign id', async () => {
    vi.mocked(listLotteries).mockResolvedValue({
      items: [{
        id: 1,
        name: '测试抽奖',
        description: '',
        status: 'active',
        win_rate: 20,
        starts_at: '2026-08-04T00:00:00Z',
        expires_at: '2026-08-05T00:00:00Z',
        winner_count: 0,
        prizes: [],
      }],
    } as never)
    vi.mocked(listLotteryWinners).mockResolvedValue({ items: [] } as never)
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.panel-actions .pixel-action--primary').exists()).toBe(false)
    expect(wrapper.find('.activity-panel').text()).not.toContain('参加抽奖')
    await wrapper.get('.panel-actions .pixel-action').trigger('click')
    expect(listLotteryWinners).toHaveBeenCalledWith(1)
  })

  it('shows registration schedule and a non-zero recharge gate without a participation action', async () => {
    vi.mocked(listLotteries).mockResolvedValue({
      items: [{
        id: 2,
        name: '报名开奖',
        description: '统一开奖活动',
        activity_type: 'registration_lottery',
        status: 'active',
        win_rate: 0,
        min_total_recharged: 20,
        starts_at: '2026-08-07T00:00:00Z',
        expires_at: '2026-08-09T00:00:00Z',
        registration_deadline: '2026-08-07T12:00:00Z',
        draw_at: '2026-08-08T12:00:00Z',
        winner_limit: 10,
        winner_count: 0,
        prizes: [],
      }],
    } as never)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('[role="tab"]')[1].trigger('click')

    expect(wrapper.text()).toContain('活动开始')
    expect(wrapper.text()).toContain('报名截止')
    expect(wrapper.text()).toContain('开奖时间')
    expect(wrapper.text()).toContain('报名门槛：累计充值满 ¥20.00')
    expect(wrapper.find('.panel-actions').text()).not.toContain('参加抽奖')
  })

  it('does not render a recharge gate when the threshold is zero', async () => {
    vi.mocked(listLotteries).mockResolvedValue({
      items: [{
        id: 3,
        name: '无门槛报名开奖',
        description: '',
        activity_type: 'registration_lottery',
        status: 'scheduled',
        win_rate: 0,
        min_total_recharged: 0,
        starts_at: '2026-08-07T00:00:00Z',
        expires_at: '2026-08-09T00:00:00Z',
        registration_deadline: '2026-08-07T12:00:00Z',
        draw_at: '2026-08-08T12:00:00Z',
        winner_limit: 2,
        winner_count: 0,
        prizes: [],
      }],
    } as never)
    const wrapper = mountView()
    await flushPromises()
    await wrapper.findAll('[role="tab"]')[1].trigger('click')

    expect(wrapper.find('.gate-note').exists()).toBe(false)
  })
})

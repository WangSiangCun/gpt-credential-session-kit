import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AvailableChannelsTable from '../AvailableChannelsTable.vue'
import type { UserAvailableChannel } from '@/api/channels'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AvailableChannelsTable.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const timelinePath = resolve(dirname(fileURLToPath(import.meta.url)), '../ChannelRequestTimeline.vue')
const timelineSource = readFileSync(timelinePath, 'utf8')
const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../views/user/AvailableChannelsView.vue')
const viewSource = readFileSync(viewPath, 'utf8')

describe('AvailableChannelsTable panel integration', () => {
  // #4555：根元素必须是 TablePageLayout 滚动链约定的 .table-wrapper，
  // 否则内容超出视口高度时被外层 overflow-hidden 裁剪且没有滚动条。
  it('mounts panels on the .table-wrapper scroll hook', () => {
    expect(componentSource).toMatch(/<div class="table-wrapper available-channels-panels">/)
    expect(componentSource).toMatch(/class="channel-panel"/)
    expect(componentSource).not.toMatch(/<table/)
  })

  it('does not clip content with its own overflow-hidden card wrapper', () => {
    expect(componentSource).not.toMatch(/<div class="card overflow-hidden">/)
  })

  it('renders public accounts as read-only detail panels without edit controls', () => {
    expect(componentSource).toMatch(/publicAccountDetailsVisible/)
    expect(componentSource).toMatch(/v-if="publicAccountDetailsVisible && channel\.accounts\?\.length"/)
    expect(componentSource).toMatch(/channel\.accounts\?\.length/)
    expect(componentSource).toMatch(/class="account-panel"/)
    expect(componentSource).toMatch(/account\.available/)
    expect(componentSource).toMatch(/account\.plan_type/)
    expect(componentSource).toMatch(/account\.quota_remaining_percent/)
    expect(componentSource).toMatch(/account\.email/)
    expect(componentSource).toMatch(/官方直冲/)
    expect(componentSource).toMatch(/v-if="account\.official_direct"/)
    expect(componentSource).not.toMatch(/account\.groups/)
    expect(componentSource).not.toMatch(/account\.rate_multiplier/)
    expect(componentSource).not.toMatch(/editAccount/)
    expect(componentSource).not.toMatch(/deleteAccount/)
  })

  it('keeps supported models collapsed until the user expands a platform', () => {
    expect(componentSource).toMatch(/expandedModels = ref\(new Set<string>\(\)\)/)
    expect(componentSource).toMatch(/toggleModels\(channel, section\)/)
    expect(componentSource).toMatch(/section\.supported_models\.length && isModelsExpanded/)
  })

  it('keeps an accessible explanation button for the request timeline', () => {
    expect(timelineSource).toMatch(/class="timeline-help"/)
    expect(timelineSource).toMatch(/:aria-expanded="helpOpen"/)
    expect(timelineSource).toMatch(/role="tooltip"/)
    expect(timelineSource).toMatch(/availableChannels\.timeline\.help/)
  })

  it('joins timelines by stable row position instead of channel name', () => {
    expect(componentSource).toMatch(/requestTimelines\[channel\.timelineIndex \?\? channelIndex\]/)
    expect(componentSource).not.toMatch(/requestTimelines\[channel\.name\]/)
    expect(componentSource).toMatch(/overflow: visible/)
    expect(componentSource).not.toMatch(/clip-path:/)
  })

  it('stops request polling when both timelines and cache statistics are disabled', () => {
    expect(componentSource).toMatch(/v-if="timelineEnabled"/)
    expect(viewSource).toMatch(/available_channels_timeline_enabled !== false/)
    expect(viewSource).toMatch(/timelineEnabled\.value \|\| cacheHitRateVisible\.value/)
    expect(viewSource).toMatch(/if \(!requestStatsEnabled\.value \|\| !timelineActive/)
  })
})

const rows: UserAvailableChannel[] = [
  {
    name: 'Primary channel',
    description: 'Fast and reliable access',
    platforms: [
      {
        platform: 'anthropic',
        groups: [
          {
            id: 1,
            name: 'Exclusive Pro',
            platform: 'anthropic',
            subscription_type: 'standard',
            rate_multiplier: 1.2,
            peak_rate_enabled: true,
            peak_start: '08:00',
            peak_end: '10:00',
            peak_rate_multiplier: 1.5,
            is_exclusive: true,
          },
          {
            id: 2,
            name: 'Public',
            platform: 'anthropic',
            subscription_type: 'standard',
            rate_multiplier: 1,
            peak_rate_enabled: false,
            peak_start: '',
            peak_end: '',
            peak_rate_multiplier: 1,
            is_exclusive: false,
          },
        ],
        supported_models: [{ name: 'claude-test', platform: 'anthropic', pricing: null }],
      },
    ],
  },
]

const baseProps = {
  columns: {
    name: 'Channel',
    description: 'Description',
    platform: 'Platform',
    groups: 'availableChannels.columns.groups',
    supportedModels: 'availableChannels.columns.supportedModels',
  },
  rows,
  loading: false,
  pricingKeyPrefix: 'availableChannels.pricing',
  noPricingLabel: 'No pricing',
  noModelsLabel: 'No models',
  emptyLabel: 'No channels',
  userGroupRates: { 1: 0.8 },
}

function mountTable(props = {}) {
  return mount(AvailableChannelsTable, {
    props: { ...baseProps, ...props },
    global: {
      plugins: [createPinia()],
      stubs: {
        Icon: { props: ['name'], template: '<i :data-icon="name" />' },
        PlatformIcon: { template: '<i data-platform-icon />' },
        GroupBadge: {
          props: ['name', 'rateMultiplier', 'userRateMultiplier'],
          template:
            '<span data-group-badge>{{ name }}:{{ rateMultiplier }}:{{ userRateMultiplier }}</span>',
        },
        SupportedModelChip: {
          props: ['model', 'noPricingLabel'],
          template: '<span data-model-chip>{{ model.name }}:{{ noPricingLabel }}</span>',
        },
      },
    },
  })
}

describe('AvailableChannelsTable responsive surfaces', () => {
  it('renders channel metadata and lazily expands model pricing in the responsive panel', async () => {
    const wrapper = mountTable()
    const desktop = wrapper.get('.channel-panel')

    expect(wrapper.classes()).toContain('table-wrapper')
    expect(desktop.find('[data-model-chip]').exists()).toBe(false)
    await desktop.get('.model-toggle').trigger('click')
    expect(desktop.get('.model-toggle').attributes('aria-expanded')).toBe('true')
    expect(desktop.text()).toContain('Primary channel')
    expect(desktop.text()).toContain('Fast and reliable access')
    expect(desktop.findAll('[data-group-badge]')).toHaveLength(2)
    expect(desktop.get('[data-model-chip]').text()).toContain('claude-test:No pricing')
  })

  it('preserves groups, user rates, peak windows and model pricing in the panel layout', async () => {
    const wrapper = mountTable()
    const mobile = wrapper.get('.channel-panel')

    expect(mobile.text()).toContain('Primary channel')
    expect(mobile.text()).toContain('Fast and reliable access')
    expect(mobile.text()).toContain('availableChannels.columns.groups')
    expect(mobile.text()).toContain('availableChannels.columns.supportedModels')
    expect(mobile.text()).toContain('availableChannels.exclusive')
    expect(mobile.text()).toContain('availableChannels.public')
    expect(mobile.get('[data-group-badge]').text()).toBe('Exclusive Pro:1.2:0.8')
    expect(mobile.findAll('[data-group-badge]')).toHaveLength(2)
    expect(mobile.get('[data-icon="clock"]')).toBeTruthy()
    expect(mobile.text()).toContain('08:00')
    expect(mobile.text()).toContain('10:00')
    expect(mobile.text()).toContain('×1.5')
    await mobile.get('.model-toggle').trigger('click')
    expect(mobile.get('[data-model-chip]').text()).toBe('claude-test:No pricing')
    await mobile.get('.model-toggle').trigger('click')
    expect(mobile.find('[data-model-chip]').exists()).toBe(false)
  })

  it('keeps the mobile placeholders when a platform has no groups or models', () => {
    const wrapper = mountTable({
      rows: [
        {
          name: 'Fallback channel',
          description: '',
          platforms: [{ platform: 'openai', groups: [], supported_models: [] }],
        },
      ],
    })
    const mobile = wrapper.get('.channel-panel')

    expect(mobile.text()).toContain('Fallback channel')
    expect(mobile.text()).toContain('openai')
    expect(mobile.text()).toContain('No models')
    expect(mobile.findAll('.empty-data')[0].text()).toBe('-')
  })

  it('provides loading and empty states in the shared responsive surface', async () => {
    const wrapper = mountTable({ loading: true, rows: [] })

    expect(wrapper.get('.channel-state [data-icon="refresh"]')).toBeTruthy()
    expect(wrapper.find('.channel-panel').exists()).toBe(false)

    await wrapper.setProps({ loading: false })

    expect(wrapper.get('.channel-empty-state').text()).toContain('No channels')
    expect(wrapper.find('[data-icon="refresh"]').exists()).toBe(false)
  })
})

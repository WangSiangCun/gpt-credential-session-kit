import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const copyToClipboard = vi.fn().mockResolvedValue(true)
const messages: Record<string, string> = {
  'keys.endpoints.title': 'API endpoints',
  'keys.endpoints.count': '{count} endpoints',
  'keys.endpoints.default': 'Default',
  'keys.endpoints.copied': 'Copied',
  'keys.endpoints.copiedHint': 'Copied to clipboard',
  'keys.endpoints.clickToCopy': 'Click to copy this endpoint',
  'keys.endpoints.speedTest': 'Speed test',
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const value = messages[key] ?? key
      return params?.count === undefined ? value : value.replace('{count}', String(params.count))
    },
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard }),
}))

import EndpointPopover from '../EndpointPopover.vue'

describe('EndpointPopover', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renders every endpoint with its administrator-provided copy', () => {
    const wrapper = mount(EndpointPopover, {
      props: {
        apiBaseUrl: 'https://default.example.com/v1',
        customEndpoints: [{
          name: 'Backup route',
          endpoint: 'https://backup.example.com/v1',
          description: 'Optimized for a second region',
        }],
      },
    })

    expect(wrapper.text()).toContain('2 endpoints')
    expect(wrapper.text()).toContain('Backup route')
    expect(wrapper.text()).toContain('https://backup.example.com/v1')
    expect(wrapper.text()).toContain('Optimized for a second region')
    expect(wrapper.findAll('[data-test="endpoint-url"]')).toHaveLength(2)
  })

  it('copies the selected endpoint and exposes copied feedback', async () => {
    const wrapper = mount(EndpointPopover, {
      props: {
        apiBaseUrl: 'https://default.example.com/v1',
        customEndpoints: [],
      },
    })

    await wrapper.get('[data-test="endpoint-url"]').trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenCalledWith('https://default.example.com/v1', 'Copied')
    expect(wrapper.find('button[aria-label="Copied to clipboard"]').exists()).toBe(true)
  })
})

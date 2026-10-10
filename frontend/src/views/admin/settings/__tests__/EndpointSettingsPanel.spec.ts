import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

import EndpointSettingsPanel from '../EndpointSettingsPanel.vue'

const endpoint = (index: number) => ({
  name: `Route ${index}`,
  endpoint: `https://api-${index}.example.com/v1`,
  description: `Region ${index}`,
})

describe('EndpointSettingsPanel', () => {
  it('emits immutable endpoint edits, additions, and removals', async () => {
    const initial = [endpoint(1)]
    const wrapper = mount(EndpointSettingsPanel, {
      props: { apiBaseUrl: 'https://default.example.com/v1', customEndpoints: initial },
    })

    await wrapper.get('input[placeholder="admin.settings.site.customEndpoints.namePlaceholder"]').setValue('Primary backup')
    const edited = wrapper.emitted('update:customEndpoints')?.at(-1)?.[0] as typeof initial
    expect(edited[0].name).toBe('Primary backup')
    expect(initial[0].name).toBe('Route 1')

    await wrapper.get('button:not([title])').trigger('click')
    const added = wrapper.emitted('update:customEndpoints')?.at(-1)?.[0] as typeof initial
    expect(added).toHaveLength(2)
    expect(added[1]).toEqual({ name: '', endpoint: '', description: '' })

    await wrapper.get('button[title="common.delete"]').trigger('click')
    const removed = wrapper.emitted('update:customEndpoints')?.at(-1)?.[0] as typeof initial
    expect(removed).toEqual([])
  })

  it('enforces the backend maximum of ten custom endpoints', () => {
    const wrapper = mount(EndpointSettingsPanel, {
      props: {
        apiBaseUrl: '',
        customEndpoints: Array.from({ length: 10 }, (_, index) => endpoint(index + 1)),
      },
    })

    const addButton = wrapper.findAll('button').at(-1)
    expect(addButton?.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('10/10')
  })
})

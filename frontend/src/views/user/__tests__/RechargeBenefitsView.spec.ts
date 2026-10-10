import { describe, expect, it, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import RechargeBenefitsView from '../RechargeBenefitsView.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('RechargeBenefitsView', () => {
  it('exposes native payments and the existing CloudCat flow as top-level sources', async () => {
    const wrapper = shallowMount(RechargeBenefitsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          PaymentView: {
            name: 'PaymentView',
            props: {
              embedded: Boolean,
              balanceOnly: Boolean,
            },
            template: '<div data-test="native-payment-view" />',
          },
          CloudCatPurchasePanel: { template: '<div data-test="cloudcat-purchase-view" />' },
        },
      },
    })

    expect(wrapper.get('[data-test="payment-source-native"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-test="payment-source-cloudcat"]').attributes('aria-selected')).toBe('false')
    expect(wrapper.get('[data-test="native-payment-view"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="cloudcat-purchase-view"]').exists()).toBe(true)
    expect(wrapper.getComponent({ name: 'PaymentView' }).props()).toMatchObject({
      embedded: true,
      balanceOnly: true,
    })

    await wrapper.get('[data-test="payment-source-cloudcat"]').trigger('click')
    expect(wrapper.get('[data-test="payment-source-native"]').attributes('aria-selected')).toBe('false')
    expect(wrapper.get('[data-test="payment-source-cloudcat"]').attributes('aria-selected')).toBe('true')
  })
})

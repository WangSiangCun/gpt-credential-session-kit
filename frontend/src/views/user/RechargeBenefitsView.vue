<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="flex space-x-1 rounded-xl bg-gray-100 p-1 dark:bg-dark-800" role="tablist" :aria-label="t('payment.sourceTabs.label')">
        <button
          type="button"
          role="tab"
          data-test="payment-source-native"
          :aria-selected="activeSource === 'native'"
          class="flex-1 rounded-lg px-4 py-2.5 text-sm font-medium transition-all"
          :class="activeSource === 'native' ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'"
          @click="activeSource = 'native'"
        >
          {{ t('payment.sourceTabs.native') }}
        </button>
        <button
          type="button"
          role="tab"
          data-test="payment-source-cloudcat"
          :aria-selected="activeSource === 'cloudcat'"
          class="flex-1 rounded-lg px-4 py-2.5 text-sm font-medium transition-all"
          :class="activeSource === 'cloudcat' ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'"
          @click="activeSource = 'cloudcat'"
        >
          {{ t('payment.sourceTabs.cloudcat') }}
        </button>
      </div>

      <section v-show="activeSource === 'native'" role="tabpanel" data-test="payment-source-native-panel">
        <PaymentView embedded balance-only />
      </section>
      <section v-show="activeSource === 'cloudcat'" role="tabpanel" data-test="payment-source-cloudcat-panel">
        <CloudCatPurchasePanel />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import CloudCatPurchasePanel from '@/components/payment/CloudCatPurchasePanel.vue'
import PaymentView from './PaymentView.vue'

const { t } = useI18n()
const activeSource = ref<'native' | 'cloudcat'>('native')
</script>

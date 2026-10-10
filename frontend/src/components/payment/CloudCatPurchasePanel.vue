<template>
  <div class="space-y-6">
    <section class="card overflow-hidden">
      <div class="border-b border-gray-200 bg-gradient-to-r from-amber-50 via-white to-violet-50 px-5 py-5 dark:border-dark-700 dark:from-amber-950/30 dark:via-dark-900 dark:to-violet-950/30 sm:px-7">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.18em] text-amber-700 dark:text-amber-300">充值与权益</p>
            <h1 class="mt-2 text-2xl font-bold text-gray-950 dark:text-white">云猫寄售</h1>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
              购买对应额度的兑换卡，支付完成后请妥善保管云猫订单中的卡密。
            </p>
          </div>
          <a :href="cloudCatUrl" target="_blank" rel="noreferrer" class="btn btn-secondary inline-flex shrink-0 items-center justify-center gap-2">
            <Icon name="externalLink" size="sm" />
            打开云猫店铺
          </a>
        </div>
      </div>

      <div class="px-5 py-5 sm:px-7">
        <div class="mb-5 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-gray-950 dark:text-white">选择购买额度</h2>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-300">点击金额卡片查看对应二维码，支持支付宝或微信扫码。</p>
          </div>
          <div class="text-right text-xs text-gray-600 dark:text-gray-300">
            <span>当前余额</span>
            <strong class="ml-2 text-base font-bold text-violet-700 dark:text-violet-300">${{ balance }}</strong>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <button v-for="plan in plans" :key="plan.amount" type="button" class="group relative min-h-[154px] border-2 border-gray-300 bg-amber-50 px-4 py-5 text-left transition hover:-translate-y-0.5 hover:border-violet-500 hover:bg-violet-50 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-violet-400 dark:hover:bg-violet-950/30" @click="selectedPlan = plan">
            <span class="text-xs font-semibold uppercase tracking-[0.14em] text-gray-600 dark:text-gray-300">兑换卡</span>
            <div class="mt-3 text-3xl font-black tracking-tight text-gray-950 dark:text-white">¥{{ plan.amount }}</div>
            <div class="mt-2 text-sm font-semibold text-gray-700 dark:text-gray-200">到账 ¥{{ plan.amount.toFixed(2) }}</div>
            <div v-if="plan.payAmount < plan.amount" class="mt-1 text-xs font-bold text-emerald-700 dark:text-emerald-300">实付 ¥{{ plan.payAmount.toFixed(2) }} · 省{{ discount(plan) }}%</div>
            <div v-else class="mt-1 text-xs font-semibold text-violet-700 dark:text-violet-300">实付 ¥{{ plan.payAmount.toFixed(2) }}</div>
            <div class="mt-4 text-xs font-semibold text-gray-600 group-hover:text-violet-700 dark:text-gray-300 dark:group-hover:text-violet-300">点击查看二维码 →</div>
          </button>
        </div>
      </div>
    </section>

    <p class="px-1 text-xs leading-5 text-gray-600 dark:text-gray-300">
      每个二维码对应一个固定到账额度。完成支付后请在云猫订单中查看卡密；如有购买问题，请联系云猫店铺客服。
    </p>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="selectedPlan" class="fixed inset-0 z-[80] flex items-center justify-center bg-black/70 p-4" @click.self="selectedPlan = null">
          <div class="relative w-full max-w-md border-2 border-gray-900 bg-amber-50 p-5 shadow-2xl dark:border-white/30 dark:bg-dark-900 sm:p-7">
            <button type="button" class="absolute right-3 top-3 p-2 text-gray-600 hover:text-gray-950 dark:text-gray-300 dark:hover:text-white" aria-label="关闭二维码" @click="selectedPlan = null">
              <Icon name="x" size="sm" />
            </button>
            <div class="text-center">
              <p class="text-xs font-bold uppercase tracking-[0.16em] text-gray-600 dark:text-gray-300">扫码购买兑换卡</p>
              <h2 class="mt-2 text-4xl font-black text-violet-800 dark:text-violet-300">¥{{ selectedPlan.amount }}</h2>
              <div class="mt-2 flex justify-center gap-4 text-sm font-semibold text-gray-800 dark:text-gray-100">
                <span>到账 ¥{{ selectedPlan.amount.toFixed(2) }}</span>
                <span class="text-violet-700 dark:text-violet-300">实付 ¥{{ selectedPlan.payAmount.toFixed(2) }}</span>
              </div>
            </div>
            <div class="mx-auto mt-5 flex aspect-square max-w-[300px] items-center justify-center border-2 border-gray-300 bg-white p-4">
              <img :src="selectedPlan.image" :alt="`¥${selectedPlan.amount} 兑换卡二维码`" class="h-full w-full object-contain" />
            </div>
            <p class="mt-4 text-center text-sm font-bold text-gray-950 dark:text-white">请使用支付宝或微信扫码支付</p>
            <p class="mt-2 text-center text-xs leading-5 text-gray-700 dark:text-gray-300">支付完成后，请前往云猫订单查看卡密。</p>
            <a :href="cloudCatUrl" target="_blank" rel="noreferrer" class="btn btn-primary mt-5 inline-flex w-full items-center justify-center gap-2">
              <Icon name="externalLink" size="sm" />
              打开云猫店铺
            </a>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'

interface CloudCatPlan {
  amount: number
  payAmount: number
  image: string
}

const defaultCloudCatUrl = 'https://catfk.com/shop/5AUO36V9'
const defaultPlans: CloudCatPlan[] = [
  { amount: 10, payAmount: 10, image: '/payment/10.png' },
  { amount: 20, payAmount: 20, image: '/payment/20.png' },
  { amount: 50, payAmount: 50, image: '/payment/50.png' },
  { amount: 100, payAmount: 100, image: '/payment/100.png' }
]

const authStore = useAuthStore()
const appStore = useAppStore()
const balance = computed(() => Number(authStore.user?.balance || 0).toFixed(2))
const cloudCatUrl = computed(() => appStore.cachedPublicSettings?.cloudcat_store_url || defaultCloudCatUrl)
const plans = computed<CloudCatPlan[]>(() => {
  const configured = appStore.cachedPublicSettings?.cloudcat_plans
  if (!configured?.length) return defaultPlans
  return configured.map(plan => ({
    amount: Number(plan.amount),
    payAmount: Number(plan.pay_amount),
    image: plan.image,
  }))
})
const selectedPlan = ref<CloudCatPlan | null>(null)

onMounted(() => {
  void appStore.fetchPublicSettings()
})

function discount(plan: CloudCatPlan): string {
  return Math.max(0, Math.round((1 - plan.payAmount / plan.amount) * 100)).toString()
}
</script>

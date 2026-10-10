<template>
  <div class="quota-cell">
    <div class="quota-windows">
      <span v-for="item in windows" :key="item.label" class="quota-window">
        <b>{{ item.label }}</b>
        <i><em :style="{ width: `${item.percent}%` }" /></i>
        <small>{{ item.percent }}% {{ item.reset }}</small>
      </span>
    </div>
    <div class="quota-actions">
      <button type="button" class="quota-query" :disabled="loading || resetting" @click="query">↻ 查询<span v-if="usage"> {{ resetCount }}</span></button>
      <button type="button" class="quota-reset" :disabled="loading || resetting || resetCount <= 0" @click="reset">↻ 重置</button>
      <button type="button" class="quota-points" :disabled="loading || resetting" @click="query">额度 {{ points }}</button>
    </div>
    <span v-if="error" class="quota-error" :title="error">{{ error }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import accountMarketAPI, { type AccountMarketQuotaUsage } from '@/api/accountMarket'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ accountId: number }>()
const appStore = useAppStore()
const usage = ref<AccountMarketQuotaUsage | null>(null)
const loading = ref(false)
const resetting = ref(false)
const error = ref('')

type QuotaRateLimit = NonNullable<AccountMarketQuotaUsage['rate_limit']>
const windowValue = (window: QuotaRateLimit | null | undefined, key: 'primary_window' | 'secondary_window') => {
  const item = window?.[key]
  if (!item) return { percent: 0, reset: '-' }
  return { percent: Math.max(0, Math.min(100, Math.round(item.used_percent || 0))), reset: formatReset(item.reset_at) }
}
const formatReset = (timestamp: number) => timestamp ? new Date(timestamp * 1000).toLocaleString() : '-'
const windows = computed(() => {
  const rate = usage.value?.rate_limit
  const five = windowValue(rate, 'primary_window')
  const seven = windowValue(rate, 'secondary_window')
  return [{ label: '5h', ...five }, { label: '7d', ...seven }]
})
const resetCount = computed(() => usage.value?.rate_limit_reset_credits?.available_count || 0)
const points = computed(() => {
  const credits = usage.value?.credits
  if (!credits) return '-'
  if (credits.unlimited) return '无限'
  return credits.balance || (credits.has_credits ? '可用' : '0')
})

async function query() {
  if (loading.value || resetting.value) return
  loading.value = true; error.value = ''
  try { usage.value = await accountMarketAPI.refreshQuota(props.accountId) }
  catch (e) { error.value = extractApiErrorMessage(e, '查询失败') }
  finally { loading.value = false }
}

async function reset() {
  if (resetCount.value <= 0 || resetting.value) return
  if (!window.confirm('确认消耗一次重置次数并重置账号额度？')) return
  resetting.value = true; error.value = ''
  try {
    const result = await accountMarketAPI.resetQuota(props.accountId)
    if (result.quota) usage.value = result.quota
    appStore.showSuccess(`额度已重置，恢复 ${result.windows_reset || 0} 个窗口`)
  } catch (e) { error.value = extractApiErrorMessage(e, '重置失败') }
  finally { resetting.value = false }
}
</script>

<style scoped>
.quota-cell{display:flex;flex-direction:column;gap:5px;min-width:250px}.quota-windows{display:flex;flex-direction:column;gap:3px}.quota-window{display:grid;grid-template-columns:28px 62px minmax(100px,1fr);align-items:center;gap:6px;color:rgb(156 163 175);font-size:11px}.quota-window b{font-weight:600;color:rgb(156 163 175)}.quota-window i{height:8px;background:rgb(55 65 81);display:block;overflow:hidden}.quota-window em{display:block;height:100%;background:rgb(79 206 128)}.quota-window small{white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.quota-actions{display:flex;align-items:center;gap:5px;flex-wrap:wrap}.quota-actions button{border:1px solid rgb(71 85 115);background:rgb(31 41 55);padding:4px 8px;color:rgb(229 231 235);font-size:12px;font-weight:600}.quota-actions button:disabled{opacity:.5;cursor:not-allowed}.quota-query{color:rgb(147 197 253)!important}.quota-reset{color:rgb(253 186 116)!important}.quota-points{color:rgb(110 231 183)!important}.quota-error{display:block;max-width:260px;color:rgb(248 113 113);font-size:11px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.dark .quota-window i{background:rgb(55 65 81)}
</style>

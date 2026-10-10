<template>
  <AppLayout>
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-500 to-blue-600 text-white shadow-sm">
          <Icon name="brain" size="md" />
        </div>
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">智力测试</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">查看各分组最近一次生成结果</p>
        </div>
      </div>
      <button class="btn btn-secondary" :disabled="loading" @click="load">
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />刷新
      </button>
    </div>

    <div v-if="error" class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300">{{ error }}</div>
    <div v-if="loading" class="py-16 text-center text-sm text-gray-500">加载中…</div>
    <div v-else-if="groups.length === 0" class="rounded-2xl border border-gray-200 bg-white px-6 py-16 text-center text-sm text-gray-500 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400">暂无可用分组</div>
    <div v-else class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
      <article v-for="group in groups" :key="group.id" class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <h2 class="truncate font-medium text-gray-900 dark:text-white">{{ group.name }}</h2>
          <div class="mt-0.5 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ group.platform }} · 分组 #{{ group.id }}</span>
            <span v-if="result(group.id)?.status === 'error'" class="text-red-500 dark:text-red-300">异常</span>
          </div>
        </div>
        <div class="flex aspect-[4/3] items-center justify-center bg-slate-50 p-4 dark:bg-dark-900/70">
          <img v-if="result(group.id)?.svg" :src="svgDataUrl(result(group.id)!.svg)" class="h-full w-full rounded-lg border border-gray-200 bg-white object-contain dark:border-dark-700" alt="SVG 测试结果" />
          <div v-else class="w-full px-4 text-center text-sm text-gray-400"><Icon name="beaker" size="lg" class="mx-auto mb-2 opacity-50" /><p class="max-h-32 overflow-auto whitespace-pre-wrap break-words text-left leading-5">{{ result(group.id)?.error || '暂无测试结果' }}</p></div>
        </div>
        <div class="px-4 py-3 text-xs text-gray-500 dark:text-dark-400">{{ result(group.id)?.tested_at ? formatTime(result(group.id)!.tested_at) : '尚未执行' }}<template v-if="result(group.id)?.model"> · {{ result(group.id)!.model }}</template><template v-if="result(group.id)?.reasoning_effort"> · {{ result(group.id)!.reasoning_effort }}</template></div>
      </article>
    </div>
  </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getPublicIntelligenceTests, type IntelligenceTestResult } from '@/api/admin/intelligenceTests'

const loading = ref(true)
const error = ref('')
const groups = ref<Array<{ id: number; name: string; platform: string; status: string }>>([])
const results = ref<Record<string, IntelligenceTestResult>>({})
const result = (id: number) => results.value[String(id)]
const formatTime = (value: string) => new Date(value).toLocaleString()
const svgDataUrl = (svg: string) => `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`

async function load() {
  loading.value = true
  error.value = ''
  try {
    const state = await getPublicIntelligenceTests()
    groups.value = state.groups
    results.value = state.results || {}
  } catch (e: any) {
    error.value = e?.response?.data?.detail || '加载智力测试失败'
  } finally {
    loading.value = false
  }
}

load()
</script>

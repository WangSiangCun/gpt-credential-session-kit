<template>
  <AppLayout>
  <div class="space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-500 to-blue-600 text-white shadow-sm">
            <Icon name="sparkles" size="md" :stroke-width="2" />
          </div>
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">智力测试</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">按分组验证文本模型是否能生成有效 SVG</p>
          </div>
        </div>
      </div>
      <button class="btn btn-secondary" :disabled="loading" @click="load">
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </div>

    <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="mb-4 flex items-center justify-between gap-3">
        <div>
          <h2 class="font-medium text-gray-900 dark:text-white">全局配置</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">频率填 0 表示关闭自动测试，仍可手动测试分组。超时时间用于单次上游测试请求。</p>
        </div>
        <button class="btn btn-primary" :disabled="saving" @click="save">
          <Icon name="check" size="sm" />
          {{ saving ? '保存中…' : '保存配置' }}
        </button>
      </div>
      <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px_220px_260px]">
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-dark-200">测试提示词</span>
          <textarea v-model="prompt" rows="3" class="input w-full resize-y font-mono text-sm" />
          <span class="mt-1 block text-xs" :class="promptLength > 4000 ? 'text-red-600' : 'text-gray-500 dark:text-dark-400'">{{ promptLength }} / 4000 字符</span>
        </label>
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-dark-200">测试频率（分钟）</span>
          <input v-model.number="frequency" type="number" min="0" max="1440" class="input w-full" />
          <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">建议 30～120 分钟</span>
        </label>
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-dark-200">测试超时（秒）</span>
          <input v-model.number="timeoutSeconds" type="number" min="30" max="1800" step="1" class="input w-full" />
          <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">范围 30～1800 秒（最长 30 分钟），默认 120 秒</span>
        </label>
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-dark-200">测试代理</span>
          <select v-model="proxyId" class="input w-full">
            <option :value="null">不使用代理</option>
            <option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">{{ proxy.name || `${proxy.protocol}://${proxy.host}:${proxy.port}` }}</option>
          </select>
          <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">只用于智力测试请求，不改变账号业务代理</span>
        </label>
      </div>
    </section>

    <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="font-medium text-gray-900 dark:text-white">用户端展示分组</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">关闭时展示全部分组，开启后只展示勾选的分组。</p>
        </div>
        <label class="inline-flex cursor-pointer items-center gap-2 text-sm text-gray-700 dark:text-dark-200">
          <input v-model="publicGroupFilterEnabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-cyan-600 focus:ring-cyan-500" />
          启用分组筛选
        </label>
      </div>
      <div v-if="publicGroupFilterEnabled" class="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        <label v-for="group in groups" :key="group.id" class="flex min-w-0 cursor-pointer items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-700">
          <input :checked="publicGroupIds.includes(group.id)" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-cyan-600 focus:ring-cyan-500" @change="togglePublicGroup(group.id)" />
          <span class="truncate text-gray-700 dark:text-dark-200">{{ group.name }}</span>
          <span class="ml-auto shrink-0 text-xs text-gray-400">#{{ group.id }}</span>
        </label>
      </div>
      <p v-if="publicGroupFilterEnabled && groups.length === 0" class="mt-4 text-sm text-gray-500 dark:text-dark-400">当前没有可配置的活动分组。</p>
    </section>

    <div v-if="error" class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300">{{ error }}</div>
    <div v-if="loading" class="py-16 text-center text-sm text-gray-500">加载中…</div>
    <div v-else class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
      <article v-for="group in groups" :key="group.id" class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <div class="min-w-0">
            <h3 class="truncate font-medium text-gray-900 dark:text-white">{{ group.name }}</h3>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ group.platform }} · 分组 #{{ group.id }}</p>
          </div>
          <span class="rounded-full px-2 py-1 text-xs" :class="result(group.id)?.status === 'success' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300' : result(group.id)?.status === 'error' ? 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300'">
            {{ result(group.id)?.status === 'success' ? '正常' : result(group.id)?.status === 'error' ? '异常' : '未测试' }}
          </span>
        </div>
        <div class="grid gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:grid-cols-2">
          <label class="min-w-0 text-xs text-gray-500 dark:text-dark-400">
            <span class="mb-1 block">测试模型</span>
            <select v-model="group.test_model" class="input w-full text-sm" :disabled="!group.available_test_models?.length" @change="onModelChange(group); saveGroup(group)">
              <option value="">自动选择</option>
              <option v-for="model in group.available_test_models" :key="model" :value="model">{{ model }}</option>
            </select>
          </label>
          <label class="min-w-0 text-xs text-gray-500 dark:text-dark-400">
            <span class="mb-1 block">思考强度</span>
            <select v-model="group.test_reasoning_effort" class="input w-full text-sm" :disabled="reasoningOptions(group).length === 0" @change="saveGroup(group)">
              <option value="">不设置</option>
              <option v-for="effort in reasoningOptions(group)" :key="effort" :value="effort">{{ effort }}</option>
            </select>
          </label>
        </div>
        <div class="flex aspect-[4/3] items-center justify-center bg-slate-50 p-4 dark:bg-dark-900/70">
          <img v-if="result(group.id)?.svg" :src="svgDataUrl(result(group.id)!.svg)" class="h-full w-full rounded-lg border border-gray-200 bg-white object-contain dark:border-dark-700" alt="SVG 测试结果" />
          <div v-else class="w-full px-3 text-center text-sm text-gray-400"><Icon name="beaker" size="lg" class="mx-auto mb-2 opacity-50" /><p class="max-h-32 overflow-auto whitespace-pre-wrap break-words text-left leading-5">{{ result(group.id)?.error || '暂无测试结果' }}</p></div>
        </div>
        <div class="flex items-center justify-between gap-3 px-4 py-3 text-xs text-gray-500 dark:text-dark-400">
          <span class="min-w-0 truncate">{{ result(group.id)?.tested_at ? formatTime(result(group.id)!.tested_at) : '尚未执行' }}<template v-if="result(group.id)?.model"> · {{ result(group.id)!.model }}</template><template v-if="result(group.id)?.reasoning_effort"> · {{ result(group.id)!.reasoning_effort }}</template></span>
          <div class="flex shrink-0 items-center gap-2">
            <button class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="saving" @click="saveGroup(group)"><Icon name="check" size="xs" />保存设置</button>
            <button class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="running !== null || saving" @click="run(group.id)"><Icon name="play" size="xs" />{{ running === group.id ? '测试中…' : '立即测试' }}</button>
          </div>
        </div>
      </article>
    </div>
  </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getIntelligenceTests, runIntelligenceTest, updateIntelligenceTests, type IntelligenceTestResult, type IntelligenceTestProxy } from '@/api/admin/intelligenceTests'

const loading = ref(true), saving = ref(false), running = ref<number | null>(null), error = ref('')
const prompt = ref('Generate an SVG of a pelican riding a bicycle.'), frequency = ref(60), timeoutSeconds = ref(120)
const promptLength = computed(() => Array.from(prompt.value.trim()).length)
const proxyId = ref<number | null>(null)
const publicGroupFilterEnabled = ref(false)
const publicGroupIds = ref<number[]>([])
const proxies = ref<IntelligenceTestProxy[]>([])
const groups = ref<Array<{ id: number; name: string; platform: string; status: string; test_model: string; test_reasoning_effort: string; available_test_models: string[]; available_reasoning_efforts: string[] }>>([])
const results = ref<Record<string, IntelligenceTestResult>>({})
const result = (id: number) => results.value[String(id)]
const formatTime = (value: string) => new Date(value).toLocaleString()
const svgDataUrl = (svg: string) => `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
const reasoningOptions = (group: { platform: string; test_model: string; available_reasoning_efforts: string[] }) => {
  if (group.platform !== 'openai' || (group.test_model || '').toLowerCase().includes('image')) return []
  return group.available_reasoning_efforts ?? []
}
const onModelChange = (group: { platform: string; test_model: string; test_reasoning_effort: string; available_reasoning_efforts: string[] }) => {
  if (reasoningOptions(group).length === 0) group.test_reasoning_effort = ''
}

function togglePublicGroup(id: number) { publicGroupIds.value = publicGroupIds.value.includes(id) ? publicGroupIds.value.filter(item => item !== id) : [...publicGroupIds.value, id] }
const errorMessage = (e: any, fallback: string) => e?.response?.data?.message || e?.response?.data?.detail || e?.message || fallback
function validatePrompt() {
  if (promptLength.value <= 4000) return true
  error.value = '测试提示词最多支持 4000 个字符'
  return false
}
async function load() { loading.value = true; error.value = ''; try { const state = await getIntelligenceTests(); prompt.value = state.prompt; frequency.value = state.frequency_mins; timeoutSeconds.value = state.timeout_seconds || 120; proxyId.value = state.proxy_id ?? null; publicGroupFilterEnabled.value = state.public_group_filter_enabled ?? false; groups.value = state.groups || []; publicGroupIds.value = state.public_group_ids ? [...state.public_group_ids] : groups.value.map(group => group.id); proxies.value = state.proxies || []; results.value = state.results || {} } catch (e: any) { error.value = errorMessage(e, '加载智力测试失败') } finally { loading.value = false } }
async function save() { if (!validatePrompt()) return; saving.value = true; error.value = ''; try { const state = await updateIntelligenceTests({ prompt: prompt.value, frequency_mins: frequency.value, timeout_seconds: timeoutSeconds.value, proxy_id: proxyId.value, public_group_filter_enabled: publicGroupFilterEnabled.value, public_group_ids: publicGroupIds.value }); prompt.value = state.prompt; frequency.value = state.frequency_mins; timeoutSeconds.value = state.timeout_seconds; proxyId.value = state.proxy_id ?? null; await load() } catch (e: any) { error.value = errorMessage(e, '保存配置失败') } finally { saving.value = false } }
async function saveGroup(group: (typeof groups.value)[number]) {
  if (!validatePrompt()) return
  saving.value = true
  error.value = ''
  try {
    const state = await updateIntelligenceTests({
      prompt: prompt.value,
      frequency_mins: frequency.value,
      timeout_seconds: timeoutSeconds.value,
      proxy_id: proxyId.value,
      public_group_filter_enabled: publicGroupFilterEnabled.value,
      public_group_ids: publicGroupIds.value,
      group_configs: { [String(group.id)]: { model: group.test_model, reasoning_effort: group.test_reasoning_effort } }
    })
    prompt.value = state.prompt
    frequency.value = state.frequency_mins
    timeoutSeconds.value = state.timeout_seconds
    proxyId.value = state.proxy_id ?? null
  } catch (e: any) {
    error.value = errorMessage(e, `分组 ${group.name} 保存失败`)
  } finally {
    saving.value = false
  }
}
async function run(id: number) { running.value = id; error.value = ''; try { results.value[String(id)] = await runIntelligenceTest(id) } catch (e: any) { error.value = errorMessage(e, '测试失败') } finally { running.value = null } }
load()
</script>

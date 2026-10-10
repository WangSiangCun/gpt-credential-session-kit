<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { CustomEndpoint } from '@/types'

const props = defineProps<{
  apiBaseUrl: string
  customEndpoints: CustomEndpoint[]
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const copiedEndpoint = ref<string | null>(null)
let copiedResetTimer: number | undefined

const allEndpoints = computed(() => {
  const items: Array<CustomEndpoint & { isDefault: boolean }> = []
  if (props.apiBaseUrl) {
    items.push({
      name: t('keys.endpoints.title'),
      endpoint: props.apiBaseUrl,
      description: '',
      isDefault: true,
    })
  }
  for (const endpoint of props.customEndpoints) {
    items.push({ ...endpoint, isDefault: false })
  }
  return items
})

async function copy(url: string) {
  const success = await copyToClipboard(url, t('keys.endpoints.copied'))
  if (!success) return

  copiedEndpoint.value = url
  if (copiedResetTimer !== undefined) window.clearTimeout(copiedResetTimer)
  copiedResetTimer = window.setTimeout(() => {
    if (copiedEndpoint.value === url) copiedEndpoint.value = null
  }, 1800)
}

function copyHint(endpoint: string): string {
  return copiedEndpoint.value === endpoint
    ? t('keys.endpoints.copiedHint')
    : t('keys.endpoints.clickToCopy')
}

function speedTestUrl(endpoint: string): string {
  return `https://www.tcptest.cn/http/${encodeURIComponent(endpoint)}`
}

onBeforeUnmount(() => {
  if (copiedResetTimer !== undefined) window.clearTimeout(copiedResetTimer)
})
</script>

<template>
  <section
    v-if="allEndpoints.length > 0"
    :aria-label="t('keys.endpoints.title')"
    class="endpoint-panel min-w-0 border-y border-gray-200 bg-gray-50/80 px-3 py-3 dark:border-dark-600 dark:bg-dark-900/75"
  >
    <div class="endpoint-panel-header mb-2.5 flex items-center gap-2.5">
      <span class="endpoint-panel-icon flex h-7 w-7 shrink-0 items-center justify-center border border-primary-300 bg-primary-50 text-primary-600 dark:border-primary-700 dark:bg-primary-950/50 dark:text-primary-300">
        <Icon name="link" size="sm" :stroke-width="2" />
      </span>
      <div class="flex min-w-0 flex-1 items-center gap-2">
        <span class="text-xs font-semibold uppercase text-gray-700 dark:text-gray-200">
          {{ t('keys.endpoints.title') }}
        </span>
        <span class="border border-gray-300 bg-white px-1.5 py-0.5 font-mono text-[10px] text-gray-500 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300">
          {{ t('keys.endpoints.count', { count: allEndpoints.length }) }}
        </span>
      </div>
    </div>

    <div class="endpoint-list grid min-w-0 grid-cols-1 gap-2 lg:grid-cols-2 2xl:grid-cols-3">
      <article
        v-for="(item, index) in allEndpoints"
        :key="`${item.endpoint}-${index}`"
        class="endpoint-card flex min-w-0 items-center gap-3 border border-gray-300 border-l-4 border-l-primary-500 bg-white px-3 py-2.5 transition-colors hover:border-primary-400 hover:border-l-primary-500 dark:border-dark-600 dark:border-l-primary-500 dark:bg-dark-800 dark:hover:border-primary-600"
      >
        <div class="min-w-0 flex-1">
          <div class="flex min-w-0 items-center gap-1.5">
            <span
              v-if="item.isDefault"
              class="shrink-0 border border-primary-200 bg-primary-50 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-primary-600 dark:border-primary-800 dark:bg-primary-950/40 dark:text-primary-300"
            >{{ t('keys.endpoints.default') }}</span>
            <span class="truncate text-xs font-semibold text-gray-700 dark:text-gray-200" :title="item.name">
              {{ item.name }}
            </span>
          </div>
          <button
            type="button"
            data-test="endpoint-url"
            class="mt-1 block max-w-full text-left"
            :title="copyHint(item.endpoint)"
            @click="copy(item.endpoint)"
          >
            <code class="block truncate font-mono text-xs text-primary-600 hover:underline dark:text-primary-300">
              {{ item.endpoint }}
            </code>
          </button>
          <p v-if="item.description" class="endpoint-description mt-1 line-clamp-2 text-[11px] leading-4 text-gray-500 dark:text-dark-300">
            {{ item.description }}
          </p>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <button
            type="button"
            data-test="copy-endpoint"
            class="btn btn-ghost h-8 w-8 p-0"
            :class="copiedEndpoint === item.endpoint ? 'text-emerald-500' : 'text-gray-400 hover:text-primary-500'"
            :title="copyHint(item.endpoint)"
            :aria-label="copyHint(item.endpoint)"
            @click="copy(item.endpoint)"
          >
            <Icon :name="copiedEndpoint === item.endpoint ? 'check' : 'copy'" size="sm" />
          </button>
          <a
            :href="speedTestUrl(item.endpoint)"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-ghost flex h-8 w-8 items-center justify-center p-0 text-gray-400 hover:text-amber-500"
            :title="t('keys.endpoints.speedTest')"
            :aria-label="t('keys.endpoints.speedTest')"
          >
            <Icon name="bolt" size="sm" />
          </a>
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.endpoint-panel {
  position: relative;
}

.endpoint-panel::after {
  position: absolute;
  top: 10px;
  right: 12px;
  width: 4px;
  height: 4px;
  content: '';
  background: rgb(156 163 175 / 45%);
  box-shadow: 8px 0 0 rgb(156 163 175 / 30%), 16px 0 0 rgb(139 92 246 / 50%);
  pointer-events: none;
}

.endpoint-card {
  box-shadow: 2px 2px 0 rgb(156 163 175 / 18%);
}

.dark .endpoint-card {
  box-shadow: 2px 2px 0 rgb(0 0 0 / 28%);
}

@media (min-width: 768px) and (max-height: 820px) {
  .endpoint-panel {
    padding-top: 0.5rem;
    padding-bottom: 0.5rem;
  }

  .endpoint-panel-header {
    margin-bottom: 0.375rem;
  }

  .endpoint-panel-icon {
    width: 1.5rem;
    height: 1.5rem;
  }

  .endpoint-card {
    padding-top: 0.5rem;
    padding-bottom: 0.5rem;
  }

  .endpoint-description {
    -webkit-line-clamp: 1;
  }
}
</style>

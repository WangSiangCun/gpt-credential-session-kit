<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { CustomEndpoint } from '@/types'

const props = defineProps<{
  apiBaseUrl: string
  customEndpoints: CustomEndpoint[]
}>()

const emit = defineEmits<{
  (event: 'update:apiBaseUrl', value: string): void
  (event: 'update:customEndpoints', value: CustomEndpoint[]): void
}>()

const { t } = useI18n()
const maxEndpoints = 10
const canAddEndpoint = computed(() => props.customEndpoints.length < maxEndpoints)

function updateEndpoint(index: number, field: keyof CustomEndpoint, value: string) {
  emit('update:customEndpoints', props.customEndpoints.map((endpoint, itemIndex) => (
    itemIndex === index ? { ...endpoint, [field]: value } : endpoint
  )))
}

function addEndpoint() {
  if (!canAddEndpoint.value) return
  emit('update:customEndpoints', [
    ...props.customEndpoints,
    { name: '', endpoint: '', description: '' },
  ])
}

function removeEndpoint(index: number) {
  emit('update:customEndpoints', props.customEndpoints.filter((_, itemIndex) => itemIndex !== index))
}
</script>

<template>
  <div class="card overflow-hidden">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div class="flex items-start gap-3">
        <div class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-primary-200 bg-primary-50 text-primary-600 dark:border-primary-800 dark:bg-primary-950/40 dark:text-primary-300">
          <Icon name="globe" size="md" />
        </div>
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('admin.settings.endpoints.title') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.endpoints.description') }}
          </p>
        </div>
      </div>
    </div>

    <div class="space-y-6 p-6">
      <section>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.site.apiBaseUrl') }}
        </label>
        <input
          :value="apiBaseUrl"
          type="url"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.site.apiBaseUrlPlaceholder')"
          @input="emit('update:apiBaseUrl', ($event.target as HTMLInputElement).value)"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.apiBaseUrlHint') }}
        </p>
      </section>

      <section class="border-t border-gray-100 pt-6 dark:border-dark-700">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.site.customEndpoints.title') }}
            </h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.site.customEndpoints.description') }}
            </p>
          </div>
          <span class="font-mono text-xs text-gray-400 dark:text-dark-400">
            {{ customEndpoints.length }}/{{ maxEndpoints }}
          </span>
        </div>

        <div v-if="customEndpoints.length" class="mt-4 space-y-3">
          <div
            v-for="(endpoint, index) in customEndpoints"
            :key="index"
            class="border-l-4 border-primary-400 bg-gray-50 p-4 dark:border-primary-600 dark:bg-dark-800/60"
          >
            <div class="mb-3 flex items-center justify-between gap-3">
              <span class="text-sm font-medium text-gray-700 dark:text-gray-200">
                {{ t('admin.settings.site.customEndpoints.itemLabel', { n: index + 1 }) }}
              </span>
              <button
                type="button"
                class="btn btn-ghost h-8 w-8 p-0 text-red-500 hover:text-red-600"
                :title="t('common.delete')"
                @click="removeEndpoint(index)"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>

            <div class="grid grid-cols-1 gap-3 md:grid-cols-[minmax(10rem,0.7fr)_minmax(16rem,1.3fr)]">
              <label class="block">
                <span class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                  {{ t('admin.settings.site.customEndpoints.name') }}
                </span>
                <input
                  :value="endpoint.name"
                  type="text"
                  required
                  class="input text-sm"
                  maxlength="50"
                  :placeholder="t('admin.settings.site.customEndpoints.namePlaceholder')"
                  @input="updateEndpoint(index, 'name', ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label class="block">
                <span class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                  {{ t('admin.settings.site.customEndpoints.endpointUrl') }}
                </span>
                <input
                  :value="endpoint.endpoint"
                  type="url"
                  required
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.site.customEndpoints.endpointUrlPlaceholder')"
                  @input="updateEndpoint(index, 'endpoint', ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label class="block md:col-span-2">
                <span class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                  {{ t('admin.settings.site.customEndpoints.descriptionLabel') }}
                </span>
                <input
                  :value="endpoint.description"
                  type="text"
                  class="input text-sm"
                  maxlength="200"
                  :placeholder="t('admin.settings.site.customEndpoints.descriptionPlaceholder')"
                  @input="updateEndpoint(index, 'description', ($event.target as HTMLInputElement).value)"
                />
              </label>
            </div>
          </div>
        </div>

        <button
          type="button"
          class="mt-4 flex w-full items-center justify-center gap-2 border-2 border-dashed border-gray-300 px-4 py-2.5 text-sm text-gray-500 transition-colors hover:border-primary-400 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-gray-400 dark:hover:border-primary-500 dark:hover:text-primary-400"
          :disabled="!canAddEndpoint"
          @click="addEndpoint"
        >
          <Icon name="plus" size="sm" />
          {{ t('admin.settings.site.customEndpoints.add') }}
        </button>
      </section>
    </div>
  </div>
</template>

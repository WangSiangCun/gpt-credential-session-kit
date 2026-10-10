<template>
  <AppLayout>
    <TablePageLayout class="available-channels-layout">
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-80">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('availableChannels.searchPlaceholder')"
                class="input pl-10"
              />
            </div>

            <div class="channel-platform-filter" :aria-label="t('availableChannels.platformFilterLabel')">
              <span class="channel-platform-filter-label">{{ t('availableChannels.platformFilterLabel') }}</span>
              <div class="channel-platform-filter-options" role="group">
                <button
                  v-for="option in platformFilterOptions"
                  :key="option.value"
                  type="button"
                  :class="[
                    'channel-platform-filter-option',
                    `channel-platform-filter-option-${option.value}`,
                    { 'is-active': activePlatformFilter === option.value }
                  ]"
                  :aria-pressed="activePlatformFilter === option.value"
                  @click="activePlatformFilter = option.value"
                >
                  <Icon v-if="option.value === 'all'" name="grid" size="xs" />
                  <PlatformIcon v-else :platform="option.value" size="xs" />
                  <span>{{ t(option.labelKey) }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              @click="loadChannels"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh', 'Refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <AvailableChannelsTable
          :rows="filteredChannels"
          :loading="loading"
          :user-group-rates="userGroupRates"
          pricing-key-prefix="availableChannels.pricing"
          :no-pricing-label="t('availableChannels.noPricing')"
          :no-models-label="t('availableChannels.noModels')"
          :empty-label="t('availableChannels.empty')"
          :request-timelines="requestTimelines"
          :timeline-request-count="timelineRequestCount"
          :timeline-refresh-seconds="timelineRefreshSeconds"
          :timeline-loading="timelineLoading"
          :timeline-enabled="timelineEnabled"
          :cache-hit-rate-visible="cacheHitRateVisible"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type ChannelRequestTimelineItem, type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

type AvailableChannelRow = UserAvailableChannel & { timelineIndex: number }

const channels = ref<AvailableChannelRow[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
type PlatformFilter = 'all' | 'openai' | 'anthropic'
const activePlatformFilter = ref<PlatformFilter>('all')
const platformFilterOptions: Array<{ value: PlatformFilter; labelKey: string }> = [
  { value: 'all', labelKey: 'availableChannels.platformFilterAll' },
  { value: 'openai', labelKey: 'availableChannels.platformFilterOpenAI' },
  { value: 'anthropic', labelKey: 'availableChannels.platformFilterClaude' },
]
const requestTimelines = ref<ChannelRequestTimelineItem[]>([])
const timelineRequestCount = ref(0)
const timelineRefreshSeconds = ref(0)
const timelineLoading = ref(false)
let timelineTimer: ReturnType<typeof setTimeout> | null = null
let timelineController: AbortController | null = null
let timelineActive = false

// Missing settings are treated as enabled for backward compatibility. Once
// public settings explicitly return false, the page stops both rendering and
// polling the request timeline.
const timelineEnabled = computed(() => appStore.cachedPublicSettings?.available_channels_timeline_enabled !== false)
const cacheHitRateVisible = computed(() => appStore.cachedPublicSettings?.available_channels_cache_hit_rate_visible === true)
const requestStatsEnabled = computed(() => timelineEnabled.value || cacheHitRateVisible.value)

/**
 * 搜索过滤：
 * - 命中渠道名/描述 → 整个渠道（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，渠道本身被过滤掉
 */
const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  const platform = activePlatformFilter.value
  return channels.value
    .map((ch) => {
      const platformSections = platform === 'all'
        ? ch.platforms
        : ch.platforms.filter((section) => section.platform === platform)
      if (platformSections.length === 0) return null

      const platformFilteredChannel = platformSections === ch.platforms
        ? ch
        : { ...ch, platforms: platformSections }
      if (!q) return platformFilteredChannel

      const nameHit = ch.name.toLowerCase().includes(q)
      const descHit = (ch.description || '').toLowerCase().includes(q)
      const accountHit = (ch.accounts || []).some((account) => account.name.toLowerCase().includes(q))
      if (nameHit || descHit || accountHit) return platformFilteredChannel
      const matchingSections = platformSections.filter(
        (p) =>
          p.platform.toLowerCase().includes(q) ||
          p.groups.some((g) => g.name.toLowerCase().includes(q)) ||
          p.supported_models.some((m) => m.name.toLowerCase().includes(q)),
      )
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is AvailableChannelRow => ch !== null)
})

async function loadChannels() {
  loading.value = true
  try {
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞渠道展示——
    // 失败时只是无法渲染专属倍率角标，降级为仅显示默认倍率。
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    // The timeline endpoint uses the same visibility and ordering rules.
    // Keep the response position as a private UI join key so duplicate names
    // cannot make one timeline overwrite another.
    channels.value = list.map((channel, timelineIndex) => ({ ...channel, timelineIndex }))
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

function clearTimelineTimer() {
  if (timelineTimer != null) {
    clearTimeout(timelineTimer)
    timelineTimer = null
  }
}

function scheduleTimelineRefresh() {
  clearTimelineTimer()
  if (!requestStatsEnabled.value || !timelineActive || document.hidden || timelineRequestCount.value <= 0 || timelineRefreshSeconds.value <= 0) return
  timelineTimer = setTimeout(loadRequestTimeline, timelineRefreshSeconds.value * 1000)
}

async function loadRequestTimeline() {
  if (!requestStatsEnabled.value || !timelineActive || document.hidden) return
  timelineController?.abort()
  timelineController = new AbortController()
  timelineLoading.value = requestTimelines.value.length === 0
  try {
    const snapshot = await userChannelsAPI.getRequestTimeline({ signal: timelineController.signal })
    timelineRequestCount.value = snapshot.request_count
    timelineRefreshSeconds.value = snapshot.refresh_seconds
    requestTimelines.value = snapshot.channels
  } catch (err: unknown) {
    if (!(err instanceof DOMException && err.name === 'AbortError')) {
      console.error('Failed to load channel request timeline:', err)
    }
  } finally {
    timelineLoading.value = false
    scheduleTimelineRefresh()
  }
}

function handleVisibilityChange() {
  if (document.hidden) {
    clearTimelineTimer()
    timelineController?.abort()
    return
  }
  void loadRequestTimeline()
}

onMounted(() => {
  timelineActive = true
  void loadChannels()
  void (async () => {
    await appStore.fetchPublicSettings()
    if (requestStatsEnabled.value) {
      await loadRequestTimeline()
    }
  })()
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onBeforeUnmount(() => {
  timelineActive = false
  clearTimelineTimer()
  timelineController?.abort()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<style scoped>
.channel-platform-filter {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 42px;
  padding: 5px 7px 5px 9px;
  border: 3px solid rgba(126, 105, 177, 0.62);
  background: #141824;
  box-shadow: 4px 4px 0 #090b10, inset 0 1px 0 rgba(255, 255, 255, 0.06);
}

.channel-platform-filter-label {
  color: #9791ac;
  font-family: var(--frost-pixel, 'PoolPixelCN', monospace);
  font-size: 11px;
  white-space: nowrap;
}

.channel-platform-filter-options {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.channel-platform-filter-option {
  display: inline-flex;
  min-height: 28px;
  align-items: center;
  gap: 5px;
  border: 2px solid #43465d;
  border-radius: 0;
  padding: 4px 7px;
  color: #aeb5c4;
  background: #202534;
  box-shadow: 2px 2px 0 #090b10;
  font-family: var(--frost-pixel, 'PoolPixelCN', monospace);
  font-size: 11px;
  line-height: 1;
  transition: color 120ms steps(2, end), background-color 120ms steps(2, end), border-color 120ms steps(2, end), transform 120ms steps(2, end), box-shadow 120ms steps(2, end);
}

.channel-platform-filter-option:hover {
  color: #f1eff8;
  border-color: #8173b2;
  transform: translate(-1px, -1px);
}

.channel-platform-filter-option:active {
  box-shadow: 1px 1px 0 #090b10;
  transform: translate(1px, 1px);
}

.channel-platform-filter-option.is-active {
  animation: channel-filter-select 180ms steps(3, end);
  color: #ece8ff;
  border-color: #c2b4ff;
  background: #3a3159;
  box-shadow: 3px 3px 0 #090b10, inset 0 0 0 1px rgba(255, 255, 255, 0.13);
}

.channel-platform-filter-option-openai.is-active {
  color: #9af2c3;
  border-color: #49bb7d;
  background: #153a2b;
}

.channel-platform-filter-option-anthropic.is-active {
  color: #ffd0ad;
  border-color: #d88452;
  background: #482b24;
}

@keyframes channel-filter-select {
  0% { transform: translate(2px, 2px); box-shadow: 1px 1px 0 #090b10; }
  100% { transform: translate(0, 0); }
}

@media (max-width: 640px) {
  .channel-platform-filter {
    width: 100%;
    justify-content: space-between;
  }

  .channel-platform-filter-options {
    flex: 1;
    justify-content: flex-end;
  }
}
</style>

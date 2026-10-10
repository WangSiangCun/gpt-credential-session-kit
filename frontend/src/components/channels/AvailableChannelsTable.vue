<template>
  <div class="table-wrapper available-channels-panels">
    <div v-if="loading" class="channel-state">
      <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
    </div>

    <div v-else-if="rows.length === 0" class="channel-state channel-empty-state">
      <Icon name="inbox" size="xl" class="h-12 w-12 text-gray-400" />
      <p>{{ emptyLabel }}</p>
    </div>

    <div v-else class="channel-panel-list">
      <article
        v-for="(channel, channelIndex) in rows"
        :key="`${channel.name}-${channelIndex}`"
        class="channel-panel"
      >
        <header class="channel-panel-header">
          <div class="channel-heading">
            <span class="channel-marker" aria-hidden="true">
              <Icon name="cube" size="sm" />
            </span>
            <div class="min-w-0">
              <h2>{{ channel.name }}</h2>
              <p v-if="channel.description">{{ channel.description }}</p>
            </div>
          </div>

          <div class="channel-header-meta">
            <div
              v-if="cacheHitRateVisible"
              class="cache-hit-rate"
              :title="t('availableChannels.cacheHitRateHelp')"
            >
              <span>{{ t('availableChannels.cacheHitRate') }}</span>
              <strong>{{ formatCacheHitRate(requestTimelines[channel.timelineIndex ?? channelIndex]?.average_cache_hit_rate) }}</strong>
              <i aria-hidden="true">?</i>
            </div>
            <ChannelQuotaBar
              v-if="channel.quota_remaining_percent != null"
              :percent="channel.quota_remaining_percent"
            />
            <div class="channel-summary" aria-label="Channel summary">
              <span><b>{{ channel.platforms.length }}</b> {{ t('availableChannels.columns.platform') }}</span>
              <span><b>{{ groupCount(channel) }}</b> {{ t('availableChannels.columns.groups') }}</span>
              <span><b>{{ modelCount(channel) }}</b> {{ t('availableChannels.columns.supportedModels') }}</span>
            </div>
          </div>
        </header>

        <div class="channel-platform-grid">
          <section
            v-for="section in channel.platforms"
            :key="`${channel.name}-${section.platform}`"
            class="channel-platform-panel"
          >
            <div class="platform-identity">
              <span :class="['platform-badge', platformBadgeClass(section.platform)]">
                <PlatformIcon :platform="section.platform as GroupPlatform" size="xs" />
                {{ section.platform }}
              </span>
              <span class="platform-count">
                {{ section.groups.length }} {{ t('availableChannels.columns.groups') }}
              </span>
            </div>

            <div class="channel-platform-content">
              <div class="channel-data-block">
                <div class="data-block-label">
                  <Icon name="users" size="xs" />
                  <span>{{ t('availableChannels.columns.groups') }}</span>
                </div>

                <div v-if="section.groups.length" class="group-stream">
                  <div v-if="exclusiveGroups(section).length" class="group-kind group-kind-exclusive">
                    <span class="group-kind-label" :title="t('availableChannels.exclusiveTooltip')">
                      <Icon name="shield" size="xs" />
                      {{ t('availableChannels.exclusive') }}
                    </span>
                    <div class="group-badges">
                      <div v-for="group in exclusiveGroups(section)" :key="`ex-${group.id}`" class="group-badge-wrap">
                        <GroupBadge
                          :name="group.name"
                          :platform="group.platform as GroupPlatform"
                          :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
                          :rate-multiplier="group.rate_multiplier"
                          :user-rate-multiplier="userGroupRates[group.id] ?? null"
                          always-show-rate
                        />
                        <span v-if="hasPeakRate(group)" class="peak-rate" :title="peakRateTitle(group)">
                          <Icon name="clock" size="xs" />
                          {{ peakRateLabel(group) }}
                        </span>
                      </div>
                    </div>
                  </div>

                  <div v-if="publicGroups(section).length" class="group-kind">
                    <span class="group-kind-label" :title="t('availableChannels.publicTooltip')">
                      <Icon name="globe" size="xs" />
                      {{ t('availableChannels.public') }}
                    </span>
                    <div class="group-badges">
                      <div v-for="group in publicGroups(section)" :key="`pub-${group.id}`" class="group-badge-wrap">
                        <GroupBadge
                          :name="group.name"
                          :platform="group.platform as GroupPlatform"
                          :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
                          :rate-multiplier="group.rate_multiplier"
                          :user-rate-multiplier="userGroupRates[group.id] ?? null"
                          always-show-rate
                        />
                        <span v-if="hasPeakRate(group)" class="peak-rate" :title="peakRateTitle(group)">
                          <Icon name="clock" size="xs" />
                          {{ peakRateLabel(group) }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
                <span v-else class="empty-data">-</span>
              </div>

              <div class="channel-data-block channel-model-block">
                <button
                  v-if="section.supported_models.length"
                  type="button"
                  class="model-toggle"
                  :aria-expanded="isModelsExpanded(channel, section)"
                  @click="toggleModels(channel, section)"
                >
                  <span class="data-block-label">
                    <Icon name="cpu" size="xs" />
                    <span>{{ t('availableChannels.columns.supportedModels') }}</span>
                    <b>{{ section.supported_models.length }}</b>
                  </span>
                  <Icon :name="isModelsExpanded(channel, section) ? 'chevronDown' : 'chevronRight'" size="xs" />
                </button>
                <div v-if="section.supported_models.length && isModelsExpanded(channel, section)" class="model-stream">
                  <SupportedModelChip
                    v-for="model in section.supported_models"
                    :key="`${section.platform}-${model.name}`"
                    :model="model"
                    :pricing-key-prefix="pricingKeyPrefix"
                    :no-pricing-label="noPricingLabel"
                    :show-platform="false"
                    :platform-hint="section.platform"
                  />
                </div>
                <span v-else-if="!section.supported_models.length" class="empty-data">{{ noModelsLabel }}</span>
              </div>
            </div>

          </section>
        </div>

        <section
          v-if="publicAccountDetailsVisible && channel.accounts?.length"
          class="channel-account-block"
          :aria-label="`${channel.name} ${t('availableChannels.account.channelAccounts')}`"
        >
          <header class="channel-section-header">
            <div class="data-block-label">
              <Icon name="users" size="xs" />
              <span>{{ t('availableChannels.account.channelAccounts') }}</span>
            </div>
            <span class="channel-section-hint">{{ t('availableChannels.account.channelAccountsHint') }}</span>
          </header>
          <div class="account-stream">
            <article
              v-for="(account, accountIndex) in channel.accounts"
              :key="`${channel.name}-${account.name}-${accountIndex}`"
              class="account-panel"
              :class="account.available ? 'account-panel-available' : 'account-panel-unavailable'"
            >
              <header class="account-panel-header">
                <div class="account-leading">
                  <span v-if="account.official_direct" class="account-official-tag" title="官方直冲">
                    <Icon name="badge" size="xs" />
                    官方直冲
                  </span>
                  <span v-if="account.plan_type" class="account-plan-tag">
                    {{ planTypeDisplayLabel(account.plan_type) }}
                  </span>
                  <span class="account-state">{{ account.available ? t('availableChannels.accountAvailable') : t('availableChannels.accountUnavailable') }}</span>
                  <div class="account-name-wrap">
                    <i aria-hidden="true"></i>
                    <div>
                      <strong :aria-label="account.name" :title="account.name">{{ account.name }}</strong>
                      <small v-if="account.email" class="account-email" :title="maskedEmail(account.email)">{{ maskedEmail(account.email) }}</small>
                    </div>
                  </div>
                </div>
                <div class="account-compact-meta">
                  <div v-if="hasRemainingQuota(account)" class="account-quota-compact" :title="t('availableChannels.account.remainingQuota')">
                    <span>{{ t('availableChannels.account.remainingQuota') }}</span>
                    <strong>{{ formatRemainingQuota(account) }}</strong>
                    <i class="account-quota-meter" aria-hidden="true">
                      <b :style="{ width: `${quotaPercent(account)}%` }"></b>
                    </i>
                  </div>
                  <span class="account-status-compact" :title="t('availableChannels.account.status')">{{ account.status || '-' }}</span>
                </div>
              </header>

            </article>
          </div>
        </section>

        <ChannelRequestTimeline
          v-if="timelineEnabled"
          :channel-name="channel.name"
          :timeline="requestTimelines[channel.timelineIndex ?? channelIndex]"
          :request-count="timelineRequestCount"
          :refresh-seconds="timelineRefreshSeconds"
          :loading="timelineLoading"
        />
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import SupportedModelChip from './SupportedModelChip.vue'
import ChannelQuotaBar from './ChannelQuotaBar.vue'
import ChannelRequestTimeline from './ChannelRequestTimeline.vue'
import { planTypeDisplayLabel } from '@/components/account/credentialsBuilder'
import type { ChannelRequestTimelineItem, UserAvailableChannel, UserAvailableGroup, UserChannelPlatformSection, UserPublicChannelAccount } from '@/api/channels'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformBadgeClass } from '@/utils/platformColors'
import { useAppStore } from '@/stores/app'
import { hasPeakRate as groupHasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'

type AvailableChannelRow = UserAvailableChannel & { timelineIndex?: number }

defineProps<{
  rows: AvailableChannelRow[]
  loading: boolean
  pricingKeyPrefix: string
  noPricingLabel: string
  noModelsLabel: string
  emptyLabel: string
  userGroupRates: Record<number, number>
  requestTimelines: ChannelRequestTimelineItem[]
  timelineRequestCount: number
  timelineRefreshSeconds: number
  timelineLoading: boolean
  timelineEnabled: boolean
  cacheHitRateVisible: boolean
}>()

const { t } = useI18n()
const appStore = useAppStore()
const expandedModels = ref(new Set<string>())

// The API is the authoritative privacy boundary. Keep a matching UI guard so
// stale responses cannot resurrect the account section after quota disclosure
// has been disabled in settings.
const publicAccountDetailsVisible = computed(() => {
  const settings = appStore.cachedPublicSettings
  return settings?.available_channels_quota_visible === true
    && settings?.available_channels_accounts_visible === true
})

function modelSectionKey(channel: AvailableChannelRow, section: UserChannelPlatformSection): string {
  return `${channel.timelineIndex ?? channel.name}\u0000${section.platform}`
}

function isModelsExpanded(channel: UserAvailableChannel, section: UserChannelPlatformSection): boolean {
  return expandedModels.value.has(modelSectionKey(channel, section))
}

function toggleModels(channel: UserAvailableChannel, section: UserChannelPlatformSection): void {
  const key = modelSectionKey(channel, section)
  const next = new Set(expandedModels.value)
  next.has(key) ? next.delete(key) : next.add(key)
  expandedModels.value = next
}

function formatRemainingQuota(account: UserPublicChannelAccount): string {
  if (account.quota_remaining_percent == null || !Number.isFinite(account.quota_remaining_percent)) return '-'
  const value = Math.max(0, Math.min(100, account.quota_remaining_percent))
  return `${Number.isInteger(value) ? value.toFixed(0) : value.toFixed(1)}%`
}

function formatCacheHitRate(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return '--'
  const percent = Math.max(0, Math.min(1, value)) * 100
  return `${percent.toFixed(percent >= 10 ? 1 : 2)}%`
}

function quotaPercent(account: UserPublicChannelAccount): number {
  const value = account.quota_remaining_percent
  return value == null || !Number.isFinite(value) ? 0 : Math.max(0, Math.min(100, value))
}

/**
 * Public channel cards must not expose the upstream account's full email.
 * Keep the domain and a small part of the local name for recognition while
 * masking the middle; admin account management continues to use the raw value.
 */
function maskedEmail(email: string): string {
  const value = email.trim()
  const at = value.lastIndexOf('@')
  if (at <= 0 || at === value.length - 1) return value

  const local = value.slice(0, at)
  const domain = value.slice(at)
  if (local.length <= 2) return `${local[0] ?? ''}****${domain}`

  const prefixLength = Math.min(2, local.length - 1)
  const suffixLength = local.length >= 6 ? 1 : 0
  const suffix = suffixLength ? local.slice(-suffixLength) : ''
  return `${local.slice(0, prefixLength)}****${suffix}${domain}`
}

function hasRemainingQuota(account: UserPublicChannelAccount): boolean {
  return account.quota_remaining_percent != null && Number.isFinite(account.quota_remaining_percent)
}

function exclusiveGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((group) => group.is_exclusive)
}

function publicGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((group) => !group.is_exclusive)
}

function groupCount(channel: UserAvailableChannel): number {
  return channel.platforms.reduce((count, section) => count + section.groups.length, 0)
}

function modelCount(channel: UserAvailableChannel): number {
  return channel.platforms.reduce((count, section) => count + section.supported_models.length, 0)
}

function hasPeakRate(group: UserAvailableGroup): boolean {
  return groupHasPeakRate(group)
}

function peakRateLabel(group: UserAvailableGroup): string {
  return formatPeakRateWindow(group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

function peakRateTitle(group: UserAvailableGroup): string {
  return t('common.peakRateTooltip', { window: peakRateLabel(group) }) + t('common.peakRateImageNote')
}
</script>

<style scoped>
.available-channels-panels {
  min-width: 0;
  padding: 12px;
  background: transparent;
  font-family: var(--frost-pixel, 'PoolPixelCN', monospace) !important;
  -webkit-font-smoothing: none !important;
  font-smooth: never;
}

.available-channels-panels * {
  font-family: inherit !important;
  -webkit-font-smoothing: none !important;
  font-smooth: never;
}

.channel-panel-list { display: grid; gap: 12px; }

.channel-panel {
  overflow: visible;
  border: 3px solid rgba(126, 105, 177, 0.78);
  border-radius: 0;
  background: #1b202d;
  box-shadow: 6px 6px 0 #07090e, inset 0 0 0 1px rgba(255, 255, 255, 0.06);
  transition: border-color 120ms steps(2, end), transform 120ms steps(2, end);
}

.channel-panel:hover { border-color: #c9baff; box-shadow: 6px 6px 0 #07090e, inset 0 0 0 1px rgba(255, 255, 255, 0.09); transform: translate(-1px, -1px); }

.channel-panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-bottom: 1px solid rgba(127, 110, 179, 0.3);
  background: linear-gradient(90deg, rgba(98, 71, 155, 0.24), rgba(27, 32, 45, 0.12));
}

.channel-heading { display: flex; min-width: 0; align-items: flex-start; gap: 10px; }
.channel-marker { display: inline-flex; margin-top: 2px; padding: 5px; color: #d8ccff; border: 1px solid #8067ba; background: #30274a; box-shadow: 2px 2px 0 #11131a; }
.channel-heading h2 { margin: 0; color: #f5f3ff; font-family: var(--frost-pixel, 'PoolPixelCN', monospace) !important; font-size: 16px; line-height: 1.5; -webkit-font-smoothing: none !important; font-smooth: never; }
.channel-heading p { margin: 5px 0 0; color: #aeb5c4; font-size: 13px; line-height: 1.65; }

.channel-header-meta { display: flex; min-width: 0; align-items: center; justify-content: flex-end; gap: 10px; }
.cache-hit-rate { display: inline-flex; align-items: baseline; gap: 6px; padding: 4px 7px; border: 2px solid #4d8e78; color: #9ca3af; background: #11251f; box-shadow: 2px 2px 0 rgba(5, 6, 10, 0.85); font-size: 10px; white-space: nowrap; }
.cache-hit-rate strong { color: #77e0ae; font-size: 13px; }
.cache-hit-rate i { display: inline-grid; width: 15px; height: 15px; place-items: center; border: 1px solid #4d8e78; color: #9fd8c1; font-style: normal; line-height: 1; }
.channel-summary { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 6px; color: #9ca3af; font-size: 11px; text-transform: uppercase; }
.channel-summary span { padding: 4px 6px; border: 1px solid rgba(126, 105, 177, 0.54); background: #141824; white-space: nowrap; box-shadow: 2px 2px 0 rgba(5, 6, 10, 0.85); }
.channel-summary b { color: #ddd6fe; font-weight: 600; }

.channel-platform-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 520px), 1fr)); gap: 10px; padding: 12px; }
.channel-platform-panel { min-width: 0; border: 3px solid rgba(126, 105, 177, 0.62); background: #171c28; box-shadow: 4px 4px 0 #090b10; }
.platform-identity { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 10px 12px; border-bottom: 3px solid rgba(126, 105, 177, 0.42); background: #24233a; }
.platform-badge { display: inline-flex; align-items: center; gap: 5px; padding: 5px 7px; border: 3px solid currentColor; border-radius: 0 !important; font-family: var(--frost-pixel, 'PoolPixelCN', monospace) !important; font-size: 11px; line-height: 1; text-transform: uppercase; box-shadow: 4px 4px 0 rgba(5, 6, 10, 0.82); -webkit-font-smoothing: none !important; }
.platform-count { display: block; color: #aeb5c4; font-size: 11px; }

.channel-platform-content { display: grid; grid-template-columns: minmax(0, 0.88fr) minmax(0, 1.12fr); min-width: 0; }
.channel-data-block { min-width: 0; padding: 12px 14px; }
.channel-model-block { min-width: 0; border-left: 3px solid rgba(113, 113, 122, 0.3); }
.channel-account-block { min-width: 0; padding: 0 14px 14px; border-top: 4px solid rgba(126, 105, 177, 0.62); background: #141925; }
.channel-section-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 0 9px; }
.channel-section-header .data-block-label { margin: 0; color: #d4c9f1; }
.channel-section-hint { color: #7f899b; font-size: 10px; }
.data-block-label { display: flex; align-items: center; gap: 5px; margin-bottom: 8px; color: #b8b2d1; font-size: 12px; font-weight: 600; text-transform: uppercase; }
.data-block-label b { margin-left: auto; color: #ddd6fe; font-size: 11px; }

.group-stream { display: grid; gap: 8px; }
.group-kind { display: grid; gap: 5px; }
.group-kind-label { display: inline-flex; align-items: center; gap: 4px; width: fit-content; color: #a1a1aa; font-size: 11px; text-transform: uppercase; }
.group-kind-exclusive .group-kind-label { color: #c4b5fd; }
.group-badges, .model-stream { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; min-width: 0; }
.model-toggle { display: flex; width: 100%; align-items: center; justify-content: space-between; gap: 8px; border: 0; padding: 0; color: #c8c0dd; background: transparent; cursor: pointer; }
.model-toggle .data-block-label { margin: 0; }
.model-toggle .data-block-label b { margin-left: 4px; }
.model-toggle + .model-stream { margin-top: 10px; }
.account-stream { display: grid; grid-template-columns: minmax(0, 1fr); gap: 8px; min-width: 0; }
.account-panel { min-width: 0; border: 3px solid #4b5563; color: #d5d9e2; background: #1b202d; box-shadow: 4px 4px 0 #080a0f; }
.account-panel-header { display: grid; grid-template-columns: minmax(0, 1fr) auto; min-width: 0; align-items: center; gap: 12px; padding: 7px 9px; border-bottom: 3px solid rgba(126, 105, 177, 0.38); background: #24233a; }
.account-leading { display: flex; min-width: 0; align-items: center; flex-wrap: wrap; gap: 6px; overflow: hidden; }
.account-name-wrap { display: flex; min-width: 0; flex: 1 1 auto; align-items: center; gap: 7px; }
.account-name-wrap i { width: 9px; height: 9px; flex: 0 0 9px; margin-top: 3px; background: #8d96a6; box-shadow: 2px 2px 0 #080a0f; }
.account-name-wrap div { min-width: 0; }
.account-name-wrap strong { display: block; overflow: hidden; color: #f5f3ff; font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
.account-name-wrap small { display: block; overflow-wrap: anywhere; margin-top: 4px; color: #9ca3af; font-size: 10px; }
.account-name-wrap .account-email { display: block; overflow: hidden; max-width: 240px; margin-top: 3px; color: #aeb9c9; font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.account-state { flex: 0 0 auto; border: 2px solid currentColor; padding: 3px 5px; font-size: 10px; line-height: 1; white-space: nowrap; }
.account-official-tag { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 4px; border: 2px solid #d4b45e; padding: 4px 6px; color: #ffe79b; background: #3b301b; box-shadow: 2px 2px 0 #11131a; font-size: 9px; font-weight: 700; line-height: 1; white-space: nowrap; }
.account-plan-tag { flex: 0 0 auto; border: 2px solid #9f8bea; padding: 3px 5px; color: #eee9ff; background: #45366b; box-shadow: 2px 2px 0 #11131a; font-size: 10px; font-weight: 700; line-height: 1; white-space: nowrap; }
.account-panel-available { border-color: rgba(110, 193, 119, 0.66); }
.account-panel-available .account-name-wrap i { background: #6ec177; }
.account-panel-available .account-state { color: #a7f3b2; }
.account-panel-unavailable { border-color: rgba(148, 163, 184, 0.42); }
.account-panel-unavailable .account-state { color: #aeb5c4; }
.account-compact-meta { display: flex; min-width: max-content; align-items: center; gap: 9px; color: #9690a9; font-size: 10px; white-space: nowrap; }
.account-quota-compact { display: inline-flex; align-items: center; gap: 5px; }
.account-quota-compact strong { color: #e3e0eb; font-size: 10px; white-space: nowrap; }
.account-quota-meter { display: block; width: 54px; height: 6px; border: 2px solid #46516b; background: #0c111b; box-shadow: 2px 2px 0 #080a0f; }
.account-quota-meter b { display: block; height: 100%; min-width: 0; background: #6ec177; box-shadow: inset 0 1px 0 rgba(255,255,255,.3); transition: width 500ms steps(6, end); }
.account-status-compact { border-left: 2px solid rgba(126, 105, 177, 0.42); padding-left: 9px; color: #cbd5e1; font-size: 10px; text-transform: uppercase; }
.account-panel-available .account-status-compact { color: #a7f3b2; }
.account-panel-unavailable .account-status-compact { color: #aeb5c4; }
.account-details, .account-message { display: none !important; }
.group-badge-wrap { display: inline-flex; flex-wrap: wrap; align-items: center; gap: 4px; min-width: 0; }
.peak-rate { display: inline-flex; align-items: center; gap: 3px; padding: 3px 5px; border: 3px solid rgba(251, 191, 36, 0.52); border-radius: 0; color: #fcd34d; background: #3a2917; font-size: 10px; box-shadow: 3px 3px 0 rgba(5, 6, 10, 0.78); }
.empty-data { color: #8d96a6; font-size: 12px; }

.channel-data-block :deep(.rounded-md),
.channel-data-block :deep(.rounded),
.channel-data-block :deep(.rounded-lg) {
  border-radius: 0 !important;
}

.channel-data-block :deep(.inline-flex.rounded-md),
.channel-data-block :deep(.inline-flex.rounded) {
  border-width: 3px !important;
  border-color: rgba(126, 105, 177, 0.52) !important;
  box-shadow: 3px 3px 0 rgba(5, 6, 10, 0.78) !important;
}

.channel-data-block :deep(.inline-flex.rounded-md:hover),
.channel-data-block :deep(.inline-flex.rounded:hover) {
  transform: translate(-1px, -1px);
  border-color: rgba(201, 186, 255, 0.78) !important;
}

.channel-data-block :deep(.inline-flex) {
  font-family: inherit !important;
}

.channel-data-block :deep(.group-badge-wrap),
.channel-data-block :deep([role='tooltip']) {
  font-family: inherit !important;
}

.channel-state { display: grid; min-height: 240px; place-content: center; justify-items: center; gap: 12px; color: #9ca3af; font-size: 13px; }

@media (max-width: 980px) {
  .channel-panel-header { align-items: stretch; }
  .channel-header-meta { align-items: flex-end; flex-direction: column; }
  .channel-platform-content { grid-template-columns: 1fr; }
  .channel-model-block { border-top: 2px solid rgba(113, 113, 122, 0.3); border-left: 0; }
}

@media (max-width: 680px) {
  .available-channels-panels { padding: 8px; }
  .channel-panel-header { flex-direction: column; gap: 10px; padding: 12px; }
  .channel-header-meta { width: 100%; align-items: stretch; }
  .channel-summary { justify-content: flex-start; }
  .channel-data-block { padding: 11px 12px; }
  .account-panel-header { grid-template-columns: minmax(0, 1fr); align-items: flex-start; gap: 7px; }
  .account-compact-meta { width: 100%; justify-content: space-between; }
}

@media (max-width: 420px) {
  .account-stream { grid-template-columns: minmax(0, 1fr); }
}

:global(.available-channels-layout .table-scroll-container) {
  overflow: hidden !important;
  border: 0 !important;
  border-radius: 0 !important;
  background: transparent !important;
  box-shadow: none !important;
  backdrop-filter: none !important;
  -webkit-backdrop-filter: none !important;
}

:global(.available-channels-layout .table-scroll-container > .table-wrapper) {
  overflow-x: hidden !important;
  overflow-y: auto !important;
}

:global(.available-channels-layout .input),
:global(.available-channels-layout .btn) {
  border: 3px solid rgba(126, 105, 177, 0.62) !important;
  border-radius: 0 !important;
  box-shadow: 4px 4px 0 #090b10, inset 0 1px 0 rgba(255, 255, 255, 0.06) !important;
  -webkit-font-smoothing: none !important;
}

:global(.available-channels-layout .input:focus) {
  border-color: #b9a5ff !important;
  box-shadow: 4px 4px 0 #090b10, inset 0 1px 0 rgba(255, 255, 255, 0.08) !important;
}

:global(.available-channels-layout .btn:hover) {
  border-color: #c9baff !important;
  box-shadow: 4px 4px 0 #090b10, inset 0 1px 0 rgba(255, 255, 255, 0.08) !important;
  transform: translate(-1px, -1px);
}

:global(.available-channels-layout .btn:active) {
  box-shadow: 1px 1px 0 #090b10 !important;
  transform: translate(2px, 2px);
}
</style>

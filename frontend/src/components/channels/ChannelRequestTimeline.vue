<template>
  <section class="request-timeline" aria-live="polite">
    <div class="request-timeline-head">
      <div class="timeline-title-wrap">
        <span class="timeline-scope">{{ t('availableChannels.timeline.channelScope') }}</span>
        <span class="timeline-title">{{ t('availableChannels.timeline.title') }}</span>
        <span v-if="channelName" class="timeline-channel" :title="channelName">{{ channelName }}</span>
        <button
          type="button"
          class="timeline-help"
          :aria-expanded="helpOpen"
          aria-controls="channel-timeline-help"
          :aria-label="t('availableChannels.timeline.helpLabel')"
          @click="helpOpen = !helpOpen"
          @keydown.esc="helpOpen = false"
        >
          ?
        </button>
        <span id="channel-timeline-help" class="timeline-tooltip" role="tooltip" :hidden="!helpOpen">
          {{ t('availableChannels.timeline.help', { count: requestCount }) }}
        </span>
      </div>
      <span class="timeline-refresh">
        <i :class="loading ? 'timeline-refreshing' : ''" aria-hidden="true"></i>
        {{ t('availableChannels.timeline.refresh', { seconds: refreshSeconds }) }}
      </span>
    </div>

    <div class="timeline-metrics">
      <div>
        <span>{{ t('availableChannels.timeline.averageFirstToken') }}</span>
        <strong>{{ formatDuration(timeline?.average_first_token_ms) }}</strong>
      </div>
      <div>
        <span>{{ t('availableChannels.timeline.averageCost') }}</span>
        <strong>{{ formatCost(timeline?.average_actual_cost) }}</strong>
      </div>
      <div>
        <span>{{ t('availableChannels.timeline.samples') }}</span>
        <strong>{{ timeline?.total_requests ?? 0 }} / {{ requestCount }}</strong>
      </div>
    </div>

    <div v-if="timeline?.segments.length" ref="viewport" class="timeline-viewport">
      <div class="timeline-track" :aria-label="t('availableChannels.timeline.ariaLabel')">
        <i
          v-for="segment in timeline.segments"
          :key="segment.index"
          class="timeline-segment"
          :class="segment.status === 'success' ? 'timeline-success' : 'timeline-failed'"
          :title="segmentTitle(segment)"
        ></i>
      </div>
    </div>
    <div v-else class="timeline-empty">
      {{ loading ? t('common.loading') : t('availableChannels.timeline.empty') }}
    </div>

    <div class="timeline-axis" aria-hidden="true">
      <span>{{ t('availableChannels.timeline.past') }}</span>
      <span>{{ t('availableChannels.timeline.now') }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChannelRequestTimelineItem, ChannelRequestTimelineSegment } from '@/api/channels'

const props = defineProps<{
  channelName?: string
  timeline?: ChannelRequestTimelineItem
  requestCount: number
  refreshSeconds: number
  loading: boolean
}>()

const { t } = useI18n()
const viewport = ref<HTMLElement | null>(null)
const helpOpen = ref(false)

watch(
  () => props.timeline?.segments.length,
  async () => {
    await nextTick()
    if (viewport.value) viewport.value.scrollLeft = viewport.value.scrollWidth
  },
  { immediate: true },
)

function formatDuration(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  if (value < 1000) return `${Math.round(value)} ms`
  return `${(value / 1000).toFixed(value < 10000 ? 2 : 1)} s`
}

function formatCost(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  if (value === 0) return '$0.000000'
  return `$${value.toFixed(6)}`
}

function segmentTitle(segment: ChannelRequestTimelineSegment): string {
  const state = segment.status === 'success'
    ? t('availableChannels.timeline.success')
    : t('availableChannels.timeline.failed')
  const ttft = segment.first_token_ms == null ? '-' : formatDuration(segment.first_token_ms)
  return t('availableChannels.timeline.segmentTitle', {
    index: segment.index + 1,
    state,
    ttft,
  })
}
</script>

<style scoped>
.request-timeline {
  position: relative;
  margin: 0 12px 14px;
  padding: 12px 14px 10px;
  border: 3px solid rgba(82, 97, 124, 0.92);
  background: #151a25;
  box-shadow: 4px 4px 0 #080a0f, inset 0 0 0 1px rgba(255, 255, 255, 0.04);
  color: #d8deea;
}

.request-timeline-head,
.timeline-metrics,
.timeline-axis { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.request-timeline-head { margin-bottom: 10px; }
.timeline-title-wrap { position: relative; display: inline-flex; min-width: 0; align-items: center; gap: 7px; }
.timeline-scope { flex: 0 0 auto; padding: 3px 5px; border: 2px solid #9b83d8; color: #efeaff; background: #45366b; box-shadow: 2px 2px 0 #080a0f; font-size: 9px; font-weight: 700; line-height: 1; letter-spacing: .04em; }
.timeline-title { color: #f5f3ff; font-size: 13px; font-weight: 700; }
.timeline-channel { overflow: hidden; max-width: min(34vw, 300px); padding-left: 8px; border-left: 2px solid rgba(155, 131, 216, .5); color: #b9f0c4; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.timeline-help {
  display: inline-grid;
  width: 18px;
  height: 18px;
  place-items: center;
  border: 2px solid #7663a7;
  background: #26233a;
  color: #e5ddff;
  font-size: 11px;
  line-height: 1;
  cursor: help;
}
.timeline-tooltip {
  position: absolute;
  z-index: 30;
  top: 26px;
  left: 0;
  display: none;
  width: min(360px, calc(100vw - 64px));
  padding: 10px 12px;
  border: 2px solid #8d79c5;
  background: #11151f;
  box-shadow: 4px 4px 0 #06080c;
  color: #d8deea;
  font-size: 12px;
  line-height: 1.7;
}
.timeline-help:hover + .timeline-tooltip,
.timeline-help:focus-visible + .timeline-tooltip,
.timeline-tooltip:not([hidden]) { display: block; }
.timeline-refresh { display: inline-flex; align-items: center; gap: 6px; color: #8f99aa; font-size: 11px; }
.timeline-refresh i { width: 7px; height: 7px; background: #38d996; box-shadow: 0 0 0 2px rgba(56, 217, 150, 0.15); }
.timeline-refresh i.timeline-refreshing { animation: timeline-blink 500ms steps(2, end) infinite; }

.timeline-metrics { justify-content: flex-start; margin-bottom: 10px; }
.timeline-metrics > div { display: flex; align-items: baseline; gap: 7px; padding-right: 18px; border-right: 1px solid #30394a; }
.timeline-metrics > div:last-child { border-right: 0; }
.timeline-metrics span { color: #8f99aa; font-size: 11px; }
.timeline-metrics strong { color: #f7f8fb; font-size: 13px; }

.timeline-viewport { width: 100%; overflow: hidden; padding: 3px 0 5px; }
.timeline-track {
  display: grid;
  grid-template-columns: repeat(auto-fill, 10px);
  grid-auto-rows: 10px;
  gap: 3px;
  width: 100%;
  min-width: 0;
  align-content: start;
}
.timeline-segment {
  display: block;
  width: 10px;
  height: 10px;
  border: 1px solid rgba(5, 6, 10, 0.72);
  box-shadow: 2px 2px 0 rgba(5, 6, 10, 0.7), inset 1px 1px 0 rgba(255, 255, 255, 0.28);
}
.timeline-success { background: #25c987; }
.timeline-failed { background: #ff535d; }
.timeline-empty { display: grid; min-height: 36px; place-items: center; border: 1px dashed #384154; color: #778196; font-size: 12px; }
.timeline-axis { margin-top: 3px; color: #717b8d; font-size: 10px; text-transform: uppercase; }

@keyframes timeline-blink { 50% { opacity: 0.25; } }

@media (max-width: 640px) {
  .request-timeline-head { align-items: flex-start; flex-direction: column; }
  .timeline-channel { max-width: 48vw; }
  .timeline-metrics { align-items: stretch; flex-direction: column; gap: 5px; }
  .timeline-metrics > div { justify-content: space-between; padding-right: 0; border-right: 0; }
  .request-timeline { margin-inline: 8px; padding-inline: 10px; }
  .timeline-track { grid-template-columns: repeat(auto-fill, 8px); grid-auto-rows: 8px; gap: 2px; }
  .timeline-segment { width: 8px; height: 8px; box-shadow: 1px 1px 0 rgba(5, 6, 10, 0.7), inset 1px 1px 0 rgba(255, 255, 255, 0.24); }
}
</style>

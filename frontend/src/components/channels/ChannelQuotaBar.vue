<template>
  <div
    class="channel-quota"
    :class="toneClass"
    :title="t('availableChannels.remainingQuota')"
  >
    <div class="channel-quota-copy">
      <span>{{ t('availableChannels.remainingQuota') }}</span>
      <strong>{{ formattedPercent }}</strong>
    </div>

    <div
      class="channel-quota-track"
      role="progressbar"
      :aria-label="t('availableChannels.remainingQuota')"
      aria-valuemin="0"
      aria-valuemax="100"
      :aria-valuenow="normalizedPercent ?? undefined"
      :aria-valuetext="formattedPercent"
    >
      <span
        v-if="normalizedPercent !== null"
        class="channel-quota-fill"
        :style="{ width: `${animatedPercent}%` }"
      />
      <span v-else class="channel-quota-unknown" aria-hidden="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  percent?: number | null
}>()

const { t } = useI18n()
const animatedPercent = ref(0)
let animationFrame: number | null = null

const normalizedPercent = computed(() => {
  if (props.percent == null || !Number.isFinite(props.percent)) return null
  return Math.min(100, Math.max(0, props.percent))
})

const formattedPercent = computed(() => {
  if (normalizedPercent.value === null) return '--'
  const value = Math.round(animatedPercent.value * 10) / 10
  return `${Number.isInteger(value) ? value.toFixed(0) : value.toFixed(1)}%`
})

const toneClass = computed(() => {
  const value = normalizedPercent.value
  if (value === null) return 'quota-unknown'
  if (value >= 60) return 'quota-healthy'
  if (value >= 25) return 'quota-warning'
  return 'quota-critical'
})

function stopAnimation() {
  if (animationFrame !== null && typeof cancelAnimationFrame !== 'undefined') {
    cancelAnimationFrame(animationFrame)
  }
  animationFrame = null
}

function animateTo(target: number | null) {
  stopAnimation()
  animatedPercent.value = 0
  if (target === null) return

  const reducedMotion =
    typeof window !== 'undefined' &&
    window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  if (reducedMotion || typeof requestAnimationFrame === 'undefined') {
    animatedPercent.value = target
    return
  }

  const duration = 680
  const steps = 14
  let startedAt: number | null = null
  const tick = (timestamp: number) => {
    if (startedAt === null) startedAt = timestamp
    const progress = Math.min(1, (timestamp - startedAt) / duration)
    const steppedProgress = Math.floor(progress * steps) / steps
    animatedPercent.value = progress === 1 ? target : target * steppedProgress
    if (progress < 1) {
      animationFrame = requestAnimationFrame(tick)
    } else {
      animationFrame = null
    }
  }
  animationFrame = requestAnimationFrame(tick)
}

watch(normalizedPercent, animateTo, { immediate: true, flush: 'post' })
onBeforeUnmount(stopAnimation)
</script>

<style scoped>
.channel-quota {
  --quota-color: #8d96a6;
  display: grid;
  gap: 5px;
  min-width: 170px;
  padding: 7px 9px;
  border: 3px solid #465066;
  background: #101521;
  box-shadow: 3px 3px 0 #07090e;
}

.channel-quota-copy {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  color: #bdc5d2;
  font-size: 11px;
  line-height: 1.2;
}

.channel-quota-copy strong {
  color: var(--quota-color);
  font-size: 14px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.channel-quota-track {
  position: relative;
  align-self: center;
  height: 10px;
  overflow: hidden;
  border: 2px solid #4a5367;
  background:
    repeating-linear-gradient(
      90deg,
      #262d3b 0,
      #262d3b 12px,
      #1b202c 12px,
      #1b202c 15px
    );
  box-shadow: inset 2px 2px 0 #0a0d14;
}

.channel-quota-fill {
  display: block;
  height: 100%;
  background:
    repeating-linear-gradient(
      90deg,
      var(--quota-color) 0,
      var(--quota-color) 12px,
      color-mix(in srgb, var(--quota-color) 62%, #080a0f) 12px,
      color-mix(in srgb, var(--quota-color) 62%, #080a0f) 15px
    );
}

.channel-quota-unknown {
  display: block;
  width: 100%;
  height: 100%;
  background: repeating-linear-gradient(135deg, #303746 0 4px, #202633 4px 8px);
}

.quota-healthy { --quota-color: #4fce80; }
.quota-warning { --quota-color: #f0b84a; }
.quota-critical { --quota-color: #ef6464; }

@media (max-width: 680px) {
  .channel-quota {
    width: 100%;
    min-width: 0;
  }
}
</style>

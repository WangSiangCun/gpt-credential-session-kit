<template>
  <div v-if="enabled">
    <slot name="trigger" :open="openDialog">
      <button
        type="button"
        class="qq-group-trigger"
        :title="t('qqGroup.open')"
        :aria-label="t('qqGroup.open')"
        @click="openDialog"
      >
        <span class="qq-penguin qq-penguin--small" aria-hidden="true">
          <i class="qq-penguin__body"></i>
          <i class="qq-penguin__belly"></i>
          <i class="qq-penguin__eye qq-penguin__eye--left"></i>
          <i class="qq-penguin__eye qq-penguin__eye--right"></i>
          <i class="qq-penguin__beak"></i>
          <i class="qq-penguin__foot qq-penguin__foot--left"></i>
          <i class="qq-penguin__foot qq-penguin__foot--right"></i>
        </span>
      </button>
    </slot>

    <PixelDialog
      :show="dialogOpen"
      :title="t('qqGroup.title')"
      width="narrow"
      @close="dialogOpen = false"
    >
      <div class="qq-group-dialog-body">
        <div class="qq-group-dialog-heading">
          <span class="qq-penguin qq-penguin--large" aria-hidden="true">
            <i class="qq-penguin__body"></i>
            <i class="qq-penguin__belly"></i>
            <i class="qq-penguin__eye qq-penguin__eye--left"></i>
            <i class="qq-penguin__eye qq-penguin__eye--right"></i>
            <i class="qq-penguin__beak"></i>
            <i class="qq-penguin__foot qq-penguin__foot--left"></i>
            <i class="qq-penguin__foot qq-penguin__foot--right"></i>
          </span>
          <div>
            <strong>{{ t('qqGroup.scanTitle') }}</strong>
            <span>{{ t('qqGroup.scanHint') }}</span>
          </div>
        </div>

        <div class="qq-group-qr-frame">
          <img v-if="qrCode" :src="qrCode" :alt="t('qqGroup.qrAlt')" />
          <div v-else class="qq-group-qr-empty">{{ t('qqGroup.notConfigured') }}</div>
        </div>

        <p v-if="description" class="qq-group-description">{{ description }}</p>
      </div>

      <template #footer>
        <button type="button" class="qq-group-close" @click="dialogOpen = false">
          {{ t('common.close') }}
        </button>
      </template>
    </PixelDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import PixelDialog from './PixelDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()
const dialogOpen = ref(false)

const openDialog = () => {
  dialogOpen.value = true
}

const settings = computed(() => appStore.cachedPublicSettings)
const enabled = computed(() => settings.value?.qq_group_enabled !== false)
const qrCode = computed(() => sanitizeUrl(
  settings.value?.qq_group_qr_code || '/qq-group-default.jpg',
  { allowRelative: true, allowDataUrl: true },
))
const description = computed(() => (
  settings.value?.qq_group_description
  || '扫码加入用户交流群，获取使用帮助与最新动态'
).trim())
</script>

<style scoped>
.qq-group-trigger {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border: 3px solid #46515f;
  border-radius: 0;
  background: #20262f;
  box-shadow: inset 2px 2px 0 rgba(255, 255, 255, 0.06), 3px 3px 0 #080a0d;
  transition: transform 100ms steps(2), border-color 100ms linear;
}

.qq-group-trigger:hover {
  border-color: #79c96c;
  transform: translate(-1px, -1px);
}

.qq-group-trigger:focus-visible {
  outline: 3px solid #a78bfa;
  outline-offset: 3px;
}

.qq-penguin {
  position: relative;
  display: inline-block;
  flex: 0 0 auto;
}

.qq-penguin--small {
  width: 20px;
  height: 24px;
}

.qq-penguin--large {
  width: 44px;
  height: 52px;
  image-rendering: pixelated;
}

.qq-penguin i {
  position: absolute;
  display: block;
}

.qq-penguin__body {
  inset: 0 2px 3px;
  background: #242a34;
  box-shadow: inset 3px 0 #3d4652, inset -3px 0 #101319;
}

.qq-penguin__belly {
  right: 5px;
  bottom: 4px;
  left: 5px;
  height: 13px;
  background: #f2f0df;
}

.qq-penguin__eye {
  top: 6px;
  width: 3px;
  height: 4px;
  background: #f7f4df;
  box-shadow: inset 1px -2px #11151b;
}

.qq-penguin__eye--left { left: 6px; }
.qq-penguin__eye--right { right: 6px; }

.qq-penguin__beak {
  top: 11px;
  left: 8px;
  width: 5px;
  height: 3px;
  background: #e7a946;
}

.qq-penguin__foot {
  bottom: 0;
  width: 7px;
  height: 3px;
  background: #e7a946;
}

.qq-penguin__foot--left { left: 1px; }
.qq-penguin__foot--right { right: 1px; }

.qq-penguin--large .qq-penguin__body { inset: 0 4px 6px; box-shadow: inset 6px 0 #3d4652, inset -6px 0 #101319; }
.qq-penguin--large .qq-penguin__belly { right: 10px; bottom: 8px; left: 10px; height: 28px; }
.qq-penguin--large .qq-penguin__eye { top: 12px; width: 6px; height: 8px; box-shadow: inset 2px -4px #11151b; }
.qq-penguin--large .qq-penguin__eye--left { left: 12px; }
.qq-penguin--large .qq-penguin__eye--right { right: 12px; }
.qq-penguin--large .qq-penguin__beak { top: 23px; left: 18px; width: 10px; height: 6px; }
.qq-penguin--large .qq-penguin__foot { width: 14px; height: 6px; }
.qq-penguin--large .qq-penguin__foot--left { left: 2px; }
.qq-penguin--large .qq-penguin__foot--right { right: 2px; }

.qq-group-dialog-body {
  display: grid;
  gap: 18px;
}

.qq-group-dialog-heading {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 68px;
  border-bottom: 3px solid #343e4b;
  padding-bottom: 16px;
}

.qq-group-dialog-heading div {
  display: grid;
  gap: 5px;
}

.qq-group-dialog-heading strong,
.qq-group-dialog-heading span,
.qq-group-description,
.qq-group-qr-empty,
.qq-group-close {
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  letter-spacing: 0;
  -webkit-font-smoothing: none;
  font-smooth: never;
}

.qq-group-dialog-heading strong { color: #f4f7fb; font-size: 15px; }
.qq-group-dialog-heading span { color: #9ca8b8; font-size: 12px; line-height: 1.6; }

.qq-group-qr-frame {
  display: grid;
  min-height: 220px;
  place-items: center;
  overflow: hidden;
  border: 4px solid #5b6573;
  background: #f3f4f6;
  box-shadow: inset 4px 4px 0 #cfd4da, 5px 5px 0 #080a0d;
}

.qq-group-qr-frame img {
  display: block;
  width: 100%;
  max-height: min(55vh, 520px);
  object-fit: contain;
}

.qq-group-qr-empty { color: #5e6671; font-size: 12px; }

.qq-group-description {
  margin: 0;
  white-space: pre-line;
  border-left: 5px solid #79c96c;
  background: #1c222b;
  padding: 13px 15px;
  color: #d9dfe7;
  font-size: 12px;
  line-height: 1.8;
}

.qq-group-close {
  min-width: 104px;
  border: 3px solid #65518d;
  border-radius: 0;
  background: #7854bd;
  padding: 9px 16px;
  color: #fff;
  font-size: 13px;
  box-shadow: 3px 3px 0 #080a0d;
}

.qq-group-close:hover { background: #8a65cf; }
</style>

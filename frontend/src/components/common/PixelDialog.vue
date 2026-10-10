<template>
  <BaseDialog
    :show="show"
    :title="title"
    :width="width"
    :z-index="zIndex"
    :close-on-click-outside="closeOnClickOutside"
    overlay-class="pixel-dialog-overlay"
    panel-class="pixel-dialog-panel"
    @close="$emit('close')"
  >
    <slot />

    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from './BaseDialog.vue'

withDefaults(defineProps<{
  show: boolean
  title: string
  width?: 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'
  zIndex?: number
  closeOnClickOutside?: boolean
}>(), {
  width: 'normal',
  zIndex: 80,
  closeOnClickOutside: true,
})

defineEmits<{
  close: []
}>()
</script>

<style>
.pixel-dialog-overlay {
  background:
    linear-gradient(rgba(9, 11, 15, 0.82), rgba(9, 11, 15, 0.82)),
    repeating-linear-gradient(0deg, transparent 0 15px, rgba(139, 92, 246, 0.08) 15px 16px),
    repeating-linear-gradient(90deg, transparent 0 15px, rgba(139, 92, 246, 0.08) 15px 16px) !important;
  backdrop-filter: none !important;
}

.pixel-dialog-panel {
  position: relative;
  overflow: visible !important;
  border: 4px solid #515b6b !important;
  border-radius: 0 !important;
  background: #151920 !important;
  box-shadow:
    inset 3px 3px 0 rgba(255, 255, 255, 0.05),
    inset -3px -3px 0 rgba(0, 0, 0, 0.42),
    7px 7px 0 #06080b !important;
}

.pixel-dialog-panel::before,
.pixel-dialog-panel::after {
  position: absolute;
  z-index: 2;
  width: 10px;
  height: 10px;
  background: #79c96c;
  box-shadow: 2px 2px 0 #31552d;
  content: '';
}

.pixel-dialog-panel::before {
  top: -5px;
  left: 22px;
}

.pixel-dialog-panel::after {
  right: 20px;
  bottom: -5px;
  background: #9b78df;
  box-shadow: 2px 2px 0 #49366d;
}

.pixel-dialog-panel .modal-header {
  min-height: 60px;
  border-bottom: 3px solid #3c4654 !important;
  background: #1c222b !important;
  padding: 15px 18px !important;
}

.pixel-dialog-panel .modal-title {
  color: #f7f8fb !important;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace !important;
  font-size: 18px !important;
  font-weight: 700 !important;
  letter-spacing: 0 !important;
  text-shadow: 2px 2px 0 #080a0d;
  -webkit-font-smoothing: none;
  font-smooth: never;
}

.pixel-dialog-panel .modal-header button {
  border: 2px solid #4a5565;
  border-radius: 0 !important;
  background: #252c36;
  color: #d7dce5;
  box-shadow: 2px 2px 0 #080a0d;
}

.pixel-dialog-panel .modal-body {
  max-height: min(76vh, 760px);
  padding: 22px !important;
}

.pixel-dialog-panel .modal-footer {
  border-top: 3px solid #3c4654 !important;
  background: #10141a !important;
  padding: 14px 18px 17px !important;
}
</style>

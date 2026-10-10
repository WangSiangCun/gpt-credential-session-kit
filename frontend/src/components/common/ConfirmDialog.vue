<template>
  <BaseDialog
    :show="show"
    :title="title"
    width="narrow"
    :show-close-button="variant !== 'pixel'"
    :overlay-class="variant === 'pixel' ? 'pixel-confirm-overlay' : undefined"
    :panel-class="variant === 'pixel' ? 'pixel-confirm-dialog' : undefined"
    @close="handleCancel"
  >
    <div :class="variant === 'pixel' ? 'pixel-confirm-body' : 'space-y-4'">
      <div v-if="variant === 'pixel'" class="pixel-confirm-warning" aria-hidden="true">!</div>
      <div>
        <p :class="variant === 'pixel' ? 'pixel-confirm-message' : 'text-sm text-gray-600 dark:text-gray-400'">
          {{ message }}
        </p>
        <p v-if="variant === 'pixel' && danger" class="pixel-confirm-note">
          此操作立即生效，且无法撤销
        </p>
      </div>
      <slot></slot>
    </div>

    <template #footer>
      <div :class="variant === 'pixel' ? 'pixel-confirm-actions' : 'flex justify-end space-x-3'">
        <button
          @click="handleCancel"
          type="button"
          :disabled="busy"
          :class="
            variant === 'pixel'
              ? 'pixel-confirm-button pixel-confirm-cancel'
              : 'rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600 dark:focus:ring-offset-dark-800'
          "
        >
          {{ cancelText }}
        </button>
        <button
          @click="handleConfirm"
          type="button"
          :disabled="busy"
          :class="[
            variant === 'pixel'
              ? 'pixel-confirm-button pixel-confirm-submit'
              : 'rounded-md px-4 py-2 text-sm font-medium text-white focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-dark-800',
            variant !== 'pixel' && danger
              ? 'bg-red-600 hover:bg-red-700 focus:ring-red-500'
              : variant !== 'pixel'
                ? 'bg-primary-600 hover:bg-primary-700 focus:ring-primary-500'
                : '',
            variant === 'pixel' && danger ? 'is-danger' : ''
          ]"
        >
          {{ busy ? busyText : confirmText }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from './BaseDialog.vue'

const { t } = useI18n()

interface Props {
  show: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
  variant?: 'default' | 'pixel'
  busy?: boolean
  busyText?: string
}

interface Emits {
  (e: 'confirm'): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  danger: false,
  variant: 'default',
  busy: false,
  busyText: '处理中...'
})

const confirmText = computed(() => props.confirmText || t('common.confirm'))
const cancelText = computed(() => props.cancelText || t('common.cancel'))

const emit = defineEmits<Emits>()

const handleConfirm = () => {
  if (props.busy) return
  emit('confirm')
}

const handleCancel = () => {
  if (props.busy) return
  emit('cancel')
}
</script>

<style>
.pixel-confirm-overlay {
  background: rgba(4, 6, 9, 0.82) !important;
  backdrop-filter: none !important;
  -webkit-backdrop-filter: none !important;
}

.pixel-confirm-dialog {
  overflow: visible !important;
  border: 3px solid #59677a !important;
  border-radius: 0 !important;
  background:
    linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px),
    linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
    #151920 !important;
  background-size: 16px 16px !important;
  box-shadow:
    inset 3px 3px 0 rgba(255, 255, 255, 0.06),
    inset -3px -3px 0 rgba(0, 0, 0, 0.42),
    6px 6px 0 #05070a !important;
  color: #e8edf4;
}

.pixel-confirm-dialog::before,
.pixel-confirm-dialog::after {
  position: absolute;
  width: 8px;
  height: 8px;
  background: #72c776;
  content: "";
}

.pixel-confirm-dialog::before {
  top: -3px;
  left: 22px;
}

.pixel-confirm-dialog::after {
  right: 20px;
  bottom: -3px;
}

.pixel-confirm-dialog .modal-header {
  min-height: 58px;
  border-bottom: 3px solid #3a4554 !important;
  background: #1d232d;
  padding: 16px 20px !important;
}

.pixel-confirm-dialog .modal-title {
  color: #fff !important;
  font-family: "PoolPixelCN", "PoolPixelEN", monospace !important;
  font-size: 18px !important;
  font-weight: 700 !important;
  letter-spacing: 0 !important;
  line-height: 1.4;
  text-shadow: 2px 2px 0 #080a0e;
  -webkit-font-smoothing: none;
  font-smooth: never;
}

.pixel-confirm-dialog .modal-body {
  overflow: visible;
  padding: 22px 20px !important;
}

.pixel-confirm-dialog .modal-footer {
  border-top: 3px solid #3a4554 !important;
  background: #10141a;
  padding: 15px 20px 18px !important;
}

.pixel-confirm-body {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr);
  align-items: start;
  gap: 16px;
}

.pixel-confirm-warning {
  display: grid;
  width: 48px;
  height: 48px;
  place-items: center;
  border: 3px solid #9a5559;
  background: #432327;
  box-shadow: inset 3px 3px 0 rgba(255, 255, 255, 0.08), 4px 4px 0 #080a0e;
  color: #ffb7b9;
  font-family: "PoolPixelEN", monospace;
  font-size: 24px;
  font-weight: 800;
  line-height: 1;
}

.pixel-confirm-message {
  margin: 1px 0 0;
  color: #e7edf5;
  font-family: "PoolPixelCN", "PoolPixelEN", monospace;
  font-size: 14px;
  line-height: 1.9;
  -webkit-font-smoothing: none;
  font-smooth: never;
}

.pixel-confirm-note {
  margin: 10px 0 0;
  color: #e68f93;
  font-family: "PoolPixelCN", "PoolPixelEN", monospace;
  font-size: 12px;
  line-height: 1.6;
  -webkit-font-smoothing: none;
  font-smooth: never;
}

.pixel-confirm-actions {
  display: flex;
  width: 100%;
  justify-content: flex-end;
  gap: 12px;
}

.pixel-confirm-button {
  min-width: 104px;
  border: 3px solid #526074;
  border-radius: 0 !important;
  padding: 10px 15px;
  background: #252c36;
  box-shadow: inset 2px 2px 0 rgba(255, 255, 255, 0.08), 4px 4px 0 #06080b;
  color: #e8edf4;
  font-family: "PoolPixelCN", "PoolPixelEN", monospace;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.2;
  -webkit-font-smoothing: none;
  font-smooth: never;
}

.pixel-confirm-button:hover:not(:disabled) {
  transform: translate(-1px, -1px);
  box-shadow: inset 2px 2px 0 rgba(255, 255, 255, 0.1), 5px 5px 0 #06080b;
}

.pixel-confirm-button:active:not(:disabled) {
  transform: translate(3px, 3px);
  box-shadow: inset 2px 2px 0 rgba(0, 0, 0, 0.28), 1px 1px 0 #06080b;
}

.pixel-confirm-button:focus-visible {
  outline: 3px solid #8bcf7b;
  outline-offset: 3px;
}

.pixel-confirm-button.is-danger {
  border-color: #b55f64;
  background: #71383d;
  color: #fff3f3;
}

.pixel-confirm-button:disabled {
  cursor: wait;
  opacity: 0.58;
}

@media (max-width: 520px) {
  .pixel-confirm-body {
    grid-template-columns: 1fr;
  }

  .pixel-confirm-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }

  .pixel-confirm-button {
    min-width: 0;
  }
}
</style>

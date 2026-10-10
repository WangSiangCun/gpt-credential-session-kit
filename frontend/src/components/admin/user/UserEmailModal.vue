<template>
  <BaseDialog :show="show" :title="t('admin.users.sendEmail')" width="narrow" @close="$emit('close')">
    <form v-if="user" id="user-email-form" class="space-y-4" @submit.prevent="submit">
      <div class="rounded-lg bg-gray-50 px-3 py-2 text-sm dark:bg-dark-700">
        <span class="text-gray-500 dark:text-dark-400">{{ t('admin.users.recipient') }}:</span>
        <span class="ml-2 font-medium text-gray-900 dark:text-gray-100">{{ user.email }}</span>
      </div>
      <div>
        <label class="input-label" for="user-email-subject">{{ t('admin.users.emailSubject') }}</label>
        <input id="user-email-subject" v-model="form.subject" class="input" maxlength="200" required />
      </div>
      <div>
        <label class="input-label" for="user-email-body">{{ t('admin.users.emailBody') }}</label>
        <textarea id="user-email-body" v-model="form.body" class="input min-h-40 resize-y" maxlength="20000" required />
        <p class="mt-1 text-xs text-gray-400">{{ form.body.length }}/20000</p>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" form="user-email-form" class="btn btn-primary" :disabled="submitting || !form.subject.trim() || !form.body.trim()">
          {{ submitting ? t('admin.users.sendingEmail') : t('admin.users.sendEmail') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits<{ close: []; success: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const submitting = ref(false)
const form = reactive({ subject: '', body: '' })

watch(() => props.show, (visible) => {
  if (visible) {
    form.subject = ''
    form.body = ''
  }
})

async function submit() {
  if (!props.user || !form.subject.trim() || !form.body.trim()) return
  submitting.value = true
  try {
    await adminAPI.users.sendEmail(props.user.id, form.subject.trim(), form.body.trim())
    appStore.showSuccess(t('admin.users.emailSent'))
    emit('success')
    emit('close')
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.users.emailSendFailed'))
  } finally {
    submitting.value = false
  }
}
</script>

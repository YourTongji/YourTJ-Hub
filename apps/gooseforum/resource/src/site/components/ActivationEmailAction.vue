<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { getActivationStatus, resendActivationEmail, type ActivationStatus } from '@/runtime/api'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
const props = defineProps<{ accountId: number }>()
const status = ref<ActivationStatus | null>(null)
const busy = ref(false)
const feedback = ref('')
const failed = ref(false)
let statusFailed = false
const seconds = ref(0)
let generation = 0
let controller: AbortController | undefined
let deadline = 0
let timer: ReturnType<typeof setInterval> | undefined
const canSend = computed(() => status.value?.state === 'ready' && !busy.value)
async function refresh(keepFeedback = false) {
  if (!keepFeedback) feedback.value = ''
  const id = ++generation
  controller?.abort()
  controller = new AbortController()
  status.value = null
  busy.value = true
  try {
    const next = await getActivationStatus(controller.signal)
    if (id !== generation) return
    status.value = next
    if (statusFailed) feedback.value = ''
    statusFailed = false
    seconds.value = next.retryAfterSeconds
    deadline = performance.now() + next.retryAfterSeconds * 1000
  } catch (error) {
    if (id !== generation) return
    statusFailed = true
    failed.value = true
    feedback.value = error instanceof Error ? error.message : t('api.activationEmailSendFailed')
  } finally { if (id === generation) busy.value = false }
}
async function resend() {
  if (!canSend.value) return
  const id = generation
  busy.value = true
  feedback.value = ''
  try {
    const message = await resendActivationEmail()
    if (id !== generation) return
    failed.value = false
    feedback.value = message
  } catch (error) {
    if (id !== generation) return
    failed.value = true
    feedback.value = error instanceof Error ? error.message : t('api.activationEmailSendFailed')
  } finally { if (id === generation) await refresh(true) }
}
watch(() => props.accountId, () => { feedback.value = ''; void refresh() }, { immediate: true })
onMounted(() => {
  window.addEventListener('focus', refreshOnFocus)
  timer = setInterval(() => {
    if (!status.value || !seconds.value) return
    seconds.value = Math.max(0, Math.ceil((deadline - performance.now()) / 1000))
    if (!seconds.value) void refresh()
  }, 1000)
})
function refreshOnFocus() { if (!busy.value) void refresh() }
onUnmounted(() => { generation++; controller?.abort(); clearInterval(timer); window.removeEventListener('focus', refreshOnFocus) })
</script>
<template>
  <div class="space-y-2 text-sm" data-testid="activation-email-action">
    <template v-if="status">
      <p v-if="status.targetEmail" class="break-all">{{ t('settings.activationStatus.target', { email: status.targetEmail }) }}</p>
      <p>{{ t('settings.activationStatus.remaining', { count: status.remainingToday, limit: status.dailyLimit }) }}</p>
      <p v-if="status.state !== 'ready'" role="status">{{ t(`settings.activationStatus.${status.state}`, { seconds }) }}</p>
    </template>
    <div class="flex flex-wrap gap-2">
      <button type="button" class="gf-button gf-button-md gf-button-outline" :disabled="!canSend" @click="resend">{{ t('settings.emailVerification.action') }}</button>
      <button type="button" class="gf-button gf-button-md gf-button-secondary" :disabled="busy" @click="refresh()">{{ t('settings.activationStatus.refresh') }}</button>
    </div>
    <p v-if="feedback" role="status" :class="failed ? 'text-error' : 'text-base-content/75'">{{ feedback }}</p>
  </div>
</template>

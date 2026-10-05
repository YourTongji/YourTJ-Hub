<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { updateTopicAgentReplies } from '@/runtime/api'

const props = defineProps<{ topicId: number; disabled: boolean; canManage: boolean }>()
const emit = defineEmits<{ changed: [disabled: boolean] }>()
const { t } = useI18n()
const saving = ref(false)
const error = ref('')
async function toggle(event: Event) {
  // Keep the native checkbox aligned with the confirmed server state.
  ;(event.target as HTMLInputElement).checked = props.disabled
  if (saving.value) return
  const topicId = props.topicId
  const next = !props.disabled
  saving.value = true
  error.value = ''
  try {
    await updateTopicAgentReplies(topicId, next)
    if (topicId === props.topicId) emit('changed', next)
  } catch (err) {
    if (topicId === props.topicId) error.value = err instanceof Error ? err.message : t('publish.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div v-if="canManage || disabled" class="mb-4 rounded-xl border border-base-300 p-3 text-sm">
    <label v-if="canManage" class="flex cursor-pointer items-center gap-2">
      <input type="checkbox" :checked="disabled" :disabled="saving" @change="toggle" />
      <span>{{ t('agentReplies.disable') }}</span>
    </label>
    <p v-else>{{ t('agentReplies.disabled') }}</p>
    <p class="mt-1 text-xs text-base-content/60">{{ t('agentReplies.help') }}</p>
    <p v-if="error" role="alert" class="mt-2 text-error">{{ error }}</p>
  </div>
</template>

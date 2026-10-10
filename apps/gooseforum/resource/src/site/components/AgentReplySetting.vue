<script setup lang="ts">
import { ref } from 'vue'
import { Bot, BotOff } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { updateTopicAgentReplies } from '@/runtime/api'
import GfSwitch from './GfSwitch.vue'

const props = defineProps<{ topicId: number; disabled: boolean; canManage: boolean }>()
const emit = defineEmits<{ changed: [disabled: boolean] }>()
const { t } = useI18n()
const saving = ref(false)
const error = ref('')
// The switch stays bound to the confirmed server state; it flips only after a successful save.
async function setAllowed(allowed: boolean) {
  if (saving.value) return
  const topicId = props.topicId
  const next = !allowed
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
  <div v-if="canManage" class="mb-4 rounded-box bg-base-200/60 px-3 py-2.5 text-sm" data-test="agent-replies-manage">
    <div class="flex items-center gap-3">
      <Bot class="h-4 w-4 shrink-0 text-icon-muted" aria-hidden="true" />
      <div class="min-w-0 flex-1">
        <p :id="`agent-replies-label-${topicId}`" class="font-medium text-base-content/85">{{ t('agentReplies.allow') }}</p>
        <p :id="`agent-replies-help-${topicId}`" class="mt-0.5 text-xs leading-5 text-base-content/55">{{ t('agentReplies.help') }}</p>
      </div>
      <GfSwitch
        :model-value="!disabled"
        :disabled="saving"
        :labelledby="`agent-replies-label-${topicId}`"
        :describedby="`agent-replies-help-${topicId}`"
        @update:model-value="setAllowed"
      />
    </div>
    <p v-if="error" role="alert" class="mt-2 ps-7 text-xs text-error">{{ error }}</p>
  </div>
  <p v-else-if="disabled" class="mb-4 flex items-center gap-1.5 text-xs text-base-content/55" data-test="agent-replies-notice">
    <BotOff class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />{{ t('agentReplies.disabled') }}
  </p>
</template>

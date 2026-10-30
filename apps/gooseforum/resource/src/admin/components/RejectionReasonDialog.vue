<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/admin/components/ui/dialog'
import { Button } from '@/admin/components/ui/button'
import { emptyRejectionDraft, rejectionReason, rejectionError } from '@/runtime/review-rejection'
import RejectionReasonFields from '@/site/components/RejectionReasonFields.vue'
const props = defineProps<{ open: boolean; busy?: boolean; error?: string }>()
const emit = defineEmits<{ confirm: [reason: string]; 'update:open': [open: boolean] }>()
const { t } = useI18n()
const draft = ref(emptyRejectionDraft())
const validation = ref('')
watch(() => props.open, open => { if (open) { draft.value = emptyRejectionDraft(); validation.value = '' } }, { immediate: true })
function confirm() {
  if (props.busy || !props.open) return
  const reason = rejectionReason(t, draft.value)
  validation.value = rejectionError(t, reason)
  if (validation.value) { document.getElementById('review-rejection-preset')?.focus(); return }
  emit('confirm', reason)
}
</script>
<template>
  <Dialog :open="open" @update:open="value => !busy && emit('update:open', value)">
    <DialogContent class="max-h-[calc(100dvh_-_2rem)] overflow-y-auto sm:max-w-md" :show-close-button="!busy" @escape-key-down="event => busy && event.preventDefault()" @interact-outside="event => busy && event.preventDefault()">
      <DialogHeader>
        <DialogTitle>{{ t('reviewRejection.title') }}</DialogTitle>
        <DialogDescription>{{ t('reviewRejection.description') }}</DialogDescription>
      </DialogHeader>
      <form class="space-y-4" @submit.prevent="confirm">
        <RejectionReasonFields v-model="draft" id-prefix="review-rejection" :disabled="busy" :error="validation || error" />
        <DialogFooter>
          <Button type="button" variant="outline" :disabled="busy" @click="emit('update:open', false)">{{ t('common.cancel') }}</Button>
          <Button type="submit" data-test="confirm-rejection" variant="destructive" :disabled="busy">{{ busy ? t('common.loading') : t('moderation.review.confirmReject') }}</Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

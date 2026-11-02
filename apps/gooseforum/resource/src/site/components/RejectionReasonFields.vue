<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { rejectionPresets, rejectionReason, type RejectionDraft, type RejectionPreset } from '@/runtime/review-rejection'
const props = defineProps<{ modelValue: RejectionDraft; idPrefix: string; disabled?: boolean; error?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: RejectionDraft] }>()
const { t } = useI18n()
const length = computed(() => Array.from(rejectionReason(t, props.modelValue)).length)
function choose(event: Event) {
  emit('update:modelValue', { ...props.modelValue, preset: (event.target as HTMLSelectElement).value as RejectionPreset | '' })
}
</script>
<template>
  <div data-test="rejection-reason-fields" class="min-w-0 space-y-3 [overflow-wrap:anywhere]">
    <div class="space-y-1">
      <label :for="`${idPrefix}-preset`" class="block text-sm font-medium">{{ t('reviewRejection.reasonLabel') }}</label>
      <select :id="`${idPrefix}-preset`" :value="modelValue.preset" :disabled="disabled" :aria-describedby="`${idPrefix}-hint ${idPrefix}-error`" :aria-invalid="Boolean(error)" class="min-h-11 w-full min-w-0 rounded-md border bg-transparent px-3 py-2 text-sm" @change="choose">
        <option value="" disabled>{{ t('reviewRejection.choose') }}</option>
        <option v-for="preset in rejectionPresets" :key="preset" :value="preset">{{ t(`reviewRejection.options.${preset}`) }}</option>
      </select>
    </div>
    <div class="space-y-1">
      <label :for="`${idPrefix}-note`" class="block text-sm font-medium">{{ t(modelValue.preset === 'other' ? 'reviewRejection.noteRequired' : 'reviewRejection.noteLabel') }}</label>
      <textarea :id="`${idPrefix}-note`" :value="modelValue.note" :disabled="disabled" :aria-describedby="`${idPrefix}-hint ${idPrefix}-length ${idPrefix}-error`" :aria-invalid="Boolean(error)" rows="3" class="w-full min-w-0 rounded-md border bg-transparent px-3 py-2 text-sm" @input="emit('update:modelValue', { ...modelValue, note: ($event.target as HTMLTextAreaElement).value })" />
      <p :id="`${idPrefix}-length`" class="text-right text-xs opacity-65">{{ length }}/512</p>
    </div>
    <p :id="`${idPrefix}-hint`" class="text-xs leading-relaxed opacity-70">{{ t('reviewRejection.hint') }}</p>
    <p :id="`${idPrefix}-error`" :role="error ? 'alert' : undefined" class="text-sm" style="color: var(--color-error, var(--destructive))">{{ error }}</p>
  </div>
</template>

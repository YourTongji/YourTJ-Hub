<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { privateNotes } from '@/runtime/private-notes'
const props = defineProps<{ userId: number; username: string }>()
const { t } = useI18n()
const open = ref(false), busy = ref(false), error = ref(''), value = ref('')
const existing = computed(() => privateNotes.state.notes.get(props.userId)?.note ?? '')
watch(() => props.userId, () => { open.value = false; error.value = '' })
function edit() { value.value = existing.value; error.value = ''; open.value = true }
async function save() {
  if (busy.value || Array.from(value.value.trim()).length > 64) return
  busy.value = true; error.value = ''
  try { await privateNotes.update(props.userId, props.username, value.value); open.value = false }
  catch (err) { error.value = err instanceof Error ? err.message : t('api.operationFailed') }
  finally { busy.value = false }
}
</script>
<template>
  <div v-if="privateNotes.state.ownerId && privateNotes.state.ownerId !== userId" class="mt-2 min-w-0">
    <button v-if="!open" type="button" class="gf-button min-h-11" @click="edit">{{ t('privateNote.edit') }}</button>
    <form v-else class="space-y-2" @submit.prevent="save">
      <label class="block text-sm">{{ t('privateNote.label') }}<input v-model="value" type="text" class="gf-input mt-1 w-full" :disabled="busy" :aria-label="t('privateNote.label')" /></label>
      <p class="text-xs text-base-content/60">{{ t('privateNote.hint') }} ({{ Array.from(value.trim()).length }}/64)</p>
      <div class="flex flex-wrap gap-2">
        <button type="submit" class="gf-button gf-button-primary min-h-11" :disabled="busy || Array.from(value.trim()).length > 64">{{ t('common.save') }}</button>
        <button type="button" class="gf-button min-h-11" :disabled="busy" @click="open = false">{{ t('common.cancel') }}</button>
      </div>
      <p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getIdentityState, type IdentityState } from '@/runtime/anonymous-identity'
const identity = defineModel<'member' | 'persona'>({default:'member'})
const props=defineProps<{disabled?: boolean;viewer?:{username:string;avatarUrl:string}}>()
const state = ref<IdentityState>()
const error = ref('')
const {t} = useI18n()
onMounted(async () => { try { state.value = await getIdentityState() } catch(e) { error.value = e instanceof Error ? e.message : String(e) } })
</script>
<template>
 <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm">
  <img v-if="identity === 'member' && props.viewer?.avatarUrl" :src="props.viewer.avatarUrl" :alt="props.viewer.username" class="h-7 w-7 rounded-full" />
  <img v-if="identity === 'persona' && state?.persona" :src="state.persona.avatarUrl" :alt="state.persona.name" class="h-7 w-7 rounded-full" />
  <label class="flex min-w-0 items-center gap-2">{{ t('anonymous.publishAs') }}
   <select v-model="identity" :disabled="disabled" class="select select-sm max-w-64" :aria-label="t('anonymous.publishAs')">
    <option value="member">{{ t('anonymous.member') }} {{ props.viewer?.username }}</option>
    <option value="persona" :disabled="!state?.persona || state.disabled || state.governanceDisabled">{{ state?.persona ? t('anonymous.personaLabel', {name:state.persona.name}) : t('anonymous.identity') }}</option>
   </select>
  </label>
  <a v-if="!disabled" href="/settings#anonymous-identity" class="text-primary underline">{{ t('anonymous.setup') }}</a>
  <span v-if="identity==='persona' && state?.persona" class="break-all" :title="state.persona.name">{{ state.persona.name }}</span>
  <span v-if="error" role="status" class="text-error">{{ error }}</span>
 </div>
</template>

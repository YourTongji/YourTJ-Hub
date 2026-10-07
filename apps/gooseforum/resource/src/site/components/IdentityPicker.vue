<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RefreshCw } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { getIdentityState, type IdentityState } from '@/runtime/anonymous-identity'

const identity = defineModel<'member' | 'persona'>({ default: 'member' })
const props = defineProps<{ disabled?: boolean; viewer?: { username: string; avatarUrl: string } }>()
const state = ref<IdentityState>()
const error = ref('')
const loading = ref(false)
const { t } = useI18n()
const name = computed(() => identity.value === 'persona' ? state.value?.persona?.name : props.viewer?.username)

async function loadState() {
  loading.value = true
  error.value = ''
  try {
    state.value = await getIdentityState()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}
onMounted(loadState)
</script>
<template>
  <!-- Keep one row through long names and async errors, preserving editor space. -->
  <div class="flex min-h-[44px] min-w-0 items-center gap-2 text-sm">
    <img v-if="identity === 'member' && props.viewer?.avatarUrl" :src="props.viewer.avatarUrl" :alt="props.viewer.username" class="h-7 w-7 shrink-0 rounded-full" />
    <img v-if="identity === 'persona' && state?.persona" :src="state.persona.avatarUrl" :alt="state.persona.name" class="h-7 w-7 shrink-0 rounded-full" />
    <label class="min-w-0 flex-1">
      <span class="sr-only">{{ t('anonymous.publishAs') }}</span>
      <select v-model="identity" :disabled="disabled" class="select select-sm w-full min-w-0" :aria-label="t('anonymous.publishAs')" :title="name">
        <option value="member">{{ t('anonymous.member') }} {{ props.viewer?.username }}</option>
        <option value="persona" :disabled="!state?.persona || state.disabled || state.governanceDisabled">{{ state?.persona ? t('anonymous.personaLabel', { name: state.persona.name }) : t('anonymous.identity') }}</option>
      </select>
    </label>
    <a v-if="!disabled" href="/settings#anonymous-identity" class="shrink-0 text-primary underline">{{ t('anonymous.setup') }}</a>
    <button type="button" class="inline-flex min-h-[44px] min-w-[44px] shrink-0 items-center justify-center text-error" :class="{ invisible: !error }" :disabled="loading || !error" :title="error" :aria-label="t('anonymous.retry')" @click="loadState">
      <RefreshCw class="h-5 w-5" />
    </button>
    <span v-if="error" role="status" class="sr-only">{{ error }}</span>
  </div>
</template>

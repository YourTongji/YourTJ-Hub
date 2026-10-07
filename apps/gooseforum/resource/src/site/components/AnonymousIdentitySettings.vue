<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ChevronRight, EyeOff, Loader2 } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { getIdentityState, type IdentityState } from '@/runtime/anonymous-identity'
import AnonymousIdentityDialog from './AnonymousIdentityDialog.vue'
const { t } = useI18n()
const state = ref<IdentityState>()
const open = ref(false)
const trigger = ref<HTMLButtonElement>()
const loading = ref(false)
const error = ref('')
async function load() {
  loading.value = true
  try {
    state.value = await getIdentityState()
    error.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)
</script>
<template>
  <div id="anonymous-identity" class="py-4">
    <button
      ref="trigger"
      type="button"
      class="flex w-full min-w-0 items-center gap-3 rounded-lg text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
      @click="open = true"
    >
      <img
        v-if="state?.persona"
        :src="state.persona.avatarUrl"
        :alt="state.persona.name"
        class="h-10 w-10 shrink-0 rounded-full"
      />
      <span
        v-else
        class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-base-200 text-base-content/60"
        ><EyeOff class="h-5 w-5"
      /></span>
      <span class="min-w-0 flex-1"
        ><span class="block text-sm font-semibold">{{ t('anonymous.identity') }}</span
        ><span class="mt-0.5 block text-sm text-base-content/60">{{ t('anonymous.settingsHint') }}</span
        ><span v-if="state?.persona" class="mt-1 block break-all text-xs text-base-content/60"
          >{{ state.persona.name }} ·
          {{
            state.governanceDisabled
              ? t('anonymous.unavailable')
              : state.disabled
                ? t('anonymous.inactive')
                : t('anonymous.ready')
          }}</span
        ></span
      >
      <Loader2 v-if="loading" class="h-4 w-4 shrink-0 animate-spin text-base-content/60" /><span
        v-else
        class="shrink-0 text-xs text-base-content/60"
        >{{ state?.persona ? t('anonymous.manageShort') : t('anonymous.notSet') }}</span
      ><ChevronRight class="h-4 w-4 shrink-0 text-base-content/50" />
    </button>
    <p v-if="error" role="alert" class="mt-2 text-xs text-error">{{ error }}</p>
    <AnonymousIdentityDialog v-model:open="open" :return-focus="trigger" @updated="state = $event" @confirmed="load" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Check, ChevronDown, EyeOff, RefreshCw, UserRound } from '@lucide/vue'
import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import { getIdentityState, type IdentityState, type Persona } from '@/runtime/anonymous-identity'
import AnonymousIdentityDialog from './AnonymousIdentityDialog.vue'
const identity = defineModel<'member' | 'persona'>({ default: 'member' })
const props = defineProps<{
  disabled?: boolean
  viewer?: { username: string; avatarUrl: string }
}>()
const state = ref<IdentityState>()
const error = ref('')
const loading = ref(false)
const menuOpen = ref(false)
const setupOpen = ref(false)
const trigger = ref<HTMLButtonElement>()
const { t } = useI18n()
const name = computed(() =>
  identity.value === 'persona' ? state.value?.persona?.name || t('anonymous.identity') : props.viewer?.username,
)
const avatar = computed(() =>
  identity.value === 'persona' ? state.value?.persona?.avatarUrl : props.viewer?.avatarUrl,
)
const usable = computed(() => !!state.value?.persona && !state.value.disabled && !state.value.governanceDisabled)
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
function choose(value: 'member' | 'persona') {
  if (props.disabled) return
  identity.value = value
  menuOpen.value = false
}
function setup() {
  menuOpen.value = false
  setupOpen.value = true
}
function confirmed(persona: Persona) {
  if (state.value) state.value = { ...state.value, persona, disabled: false }
  if (!props.disabled) identity.value = 'persona'
}
onMounted(loadState)
</script>
<template>
  <!-- Identity setup stays inside the composer so its draft and attribution survive. -->
  <div class="flex min-h-11 min-w-0 items-center gap-2 text-sm">
    <PopoverRoot v-model:open="menuOpen">
      <PopoverTrigger as-child>
        <button
          ref="trigger"
          type="button"
          :disabled="disabled"
          :aria-label="`${t('anonymous.publishAs')}：${name}`"
          :title="name"
          class="inline-flex min-h-11 min-w-0 max-w-full items-center gap-2 rounded-lg px-2 text-base-content/80 transition hover:bg-base-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary disabled:cursor-default"
        >
          <img v-if="avatar" :src="avatar" :alt="name" class="h-7 w-7 shrink-0 rounded-full" /><EyeOff
            v-else-if="identity === 'persona'"
            class="h-5 w-5 shrink-0"
          /><UserRound v-else class="h-5 w-5 shrink-0" /> <span class="min-w-0 truncate font-medium">{{ name }}</span
          ><span class="shrink-0 rounded bg-base-200 px-1.5 py-0.5 text-[11px] text-base-content/60">{{
            identity === 'persona' ? t('anonymous.identity') : t('anonymous.member')
          }}</span
          ><ChevronDown v-if="!disabled" class="h-3.5 w-3.5 shrink-0 text-base-content/50" />
        </button>
      </PopoverTrigger>
      <PopoverPortal>
        <PopoverContent
          align="start"
          :side-offset="6"
          @close-auto-focus="
            (event) => {
              if (setupOpen) event.preventDefault()
            }
          "
          class="z-[2200] w-72 max-w-[calc(100vw_-_2rem)] rounded-lg border border-line bg-base-100 p-1.5 text-base-content shadow-lg outline-none"
        >
          <p class="px-2.5 py-2 text-xs font-medium text-base-content/55">
            {{ t('anonymous.publishAs') }}
          </p>
          <button
            type="button"
            :aria-pressed="identity === 'member'"
            class="flex w-full min-w-0 items-center gap-3 rounded-md p-2.5 text-left hover:bg-base-200 focus-visible:outline focus-visible:outline-primary"
            @click="choose('member')"
          >
            <UserRound class="h-5 w-5 shrink-0 text-base-content/60" /><span class="min-w-0 flex-1"
              ><span class="block text-sm font-medium">{{ t('anonymous.member') }}</span
              ><span class="block truncate text-xs text-base-content/60">{{ viewer?.username }}</span></span
            ><Check v-if="identity === 'member'" class="h-4 w-4 shrink-0 text-primary" />
          </button>
          <button
            v-if="state && !state.persona"
            type="button"
            class="flex w-full items-center gap-3 rounded-md p-2.5 text-left hover:bg-base-200 focus-visible:outline focus-visible:outline-primary"
            @click="setup"
          >
            <EyeOff class="h-5 w-5 shrink-0 text-base-content/60" /><span class="text-sm font-medium">{{
              t('anonymous.setup')
            }}</span>
          </button>
          <button
            v-else
            type="button"
            :disabled="!usable || loading"
            :aria-pressed="identity === 'persona'"
            class="flex w-full min-w-0 items-center gap-3 rounded-md p-2.5 text-left hover:bg-base-200 focus-visible:outline focus-visible:outline-primary disabled:opacity-50"
            @click="choose('persona')"
          >
            <EyeOff class="h-5 w-5 shrink-0 text-base-content/60" /><span class="min-w-0 flex-1"
              ><span class="block text-sm font-medium">{{ t('anonymous.identity') }}</span
              ><span class="block truncate text-xs text-base-content/60">{{
                state?.persona?.name || t('anonymous.unavailable')
              }}</span></span
            ><Check v-if="identity === 'persona'" class="h-4 w-4 shrink-0 text-primary" />
          </button>
          <button
            v-if="state?.persona"
            type="button"
            class="mt-1 w-full border-t border-line px-2.5 py-2.5 text-left text-xs text-base-content/60 hover:text-primary"
            @click="setup"
          >
            {{ t('anonymous.manage') }}
          </button>
        </PopoverContent>
      </PopoverPortal>
    </PopoverRoot>
    <button
      v-if="error"
      type="button"
      class="gf-icon-button h-11 w-11 shrink-0 text-error"
      :disabled="loading"
      :title="error"
      :aria-label="t('anonymous.retry')"
      @click="loadState"
    >
      <RefreshCw class="h-4 w-4" /></button
    ><span v-if="error" role="status" class="sr-only">{{ error }}</span>
    <AnonymousIdentityDialog
      v-model:open="setupOpen"
      :return-focus="trigger"
      @updated="state = $event"
      @confirmed="confirmed"
    />
  </div>
</template>

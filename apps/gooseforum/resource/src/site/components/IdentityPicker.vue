<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Check, ChevronDown, EyeOff, RefreshCw, Settings, UserRound } from '@lucide/vue'
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
// Why the persona cannot be chosen right now; shown in place of its name.
const personaNote = computed(() => {
  if (!state.value?.persona) return ''
  if (state.value.governanceDisabled) return t('anonymous.statusRestricted')
  return state.value.disabled ? t('anonymous.statusDisabled') : state.value.persona.name
})
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
  <div class="flex min-h-11 min-w-0 items-center gap-1 text-sm">
    <PopoverRoot v-model:open="menuOpen">
      <PopoverTrigger as-child>
        <button
          ref="trigger"
          type="button"
          :disabled="disabled"
          :aria-label="`${t('anonymous.publishAs')}：${name}`"
          :title="name"
          class="inline-flex min-h-11 min-w-0 max-w-full items-center gap-2 rounded-full ps-1.5 pe-2.5 text-base-content/80 transition-colors duration-150 hover:bg-base-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary disabled:cursor-default disabled:hover:bg-transparent"
        >
          <img v-if="avatar" :src="avatar" alt="" class="h-7 w-7 shrink-0 rounded-full outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10" /><span
            v-else
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-base-200 text-base-content/55"
            ><EyeOff v-if="identity === 'persona'" class="h-4 w-4" /><UserRound v-else class="h-4 w-4"
          /></span>
          <span class="min-w-0 truncate font-medium">{{ name }}</span>
          <span
            class="shrink-0 rounded-full px-1.5 py-0.5 text-[11px] font-medium"
            :class="identity === 'persona' ? 'bg-base-content/10 text-base-content/70' : 'bg-base-200 text-base-content/55'"
            >{{ identity === 'persona' ? t('anonymous.tag') : t('anonymous.member') }}</span
          ><ChevronDown v-if="!disabled" class="h-3.5 w-3.5 shrink-0 text-base-content/45" />
        </button>
      </PopoverTrigger>
      <PopoverPortal>
        <PopoverContent
          align="start"
          :side-offset="6"
          :collision-padding="12"
          @close-auto-focus="
            (event) => {
              if (setupOpen) event.preventDefault()
            }
          "
          class="z-[2200] w-72 max-w-[calc(100vw_-_1.5rem)] rounded-2xl bg-base-100 p-1.5 text-base-content shadow-[0_12px_32px_-8px_oklch(0_0_0/0.25),0_0_0_1px_var(--gf-color-line)] outline-none"
        >
          <p class="px-2.5 pb-1 pt-1.5 text-xs font-medium text-base-content/55">
            {{ t('anonymous.publishAs') }}
          </p>
          <button
            type="button"
            :aria-pressed="identity === 'member'"
            class="flex w-full min-w-0 items-center gap-3 rounded-xl p-2.5 text-start transition-colors duration-150 hover:bg-base-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
            @click="choose('member')"
          >
            <img v-if="viewer?.avatarUrl" :src="viewer.avatarUrl" alt="" class="h-8 w-8 shrink-0 rounded-full" /><UserRound
              v-else
              class="h-5 w-5 shrink-0 text-base-content/60"
            /><span class="min-w-0 flex-1"
              ><span class="block text-sm font-medium">{{ t('anonymous.member') }}</span
              ><span class="block truncate text-xs text-base-content/60">@{{ viewer?.username }}</span></span
            ><Check v-if="identity === 'member'" class="h-4 w-4 shrink-0 text-primary" />
          </button>
          <button
            v-if="state && !state.persona"
            type="button"
            class="flex w-full items-center gap-3 rounded-xl p-2.5 text-start transition-colors duration-150 hover:bg-base-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
            @click="setup"
          >
            <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-base-200 text-base-content/60"><EyeOff class="h-4 w-4" /></span
            ><span class="min-w-0 flex-1"
              ><span class="block text-sm font-medium">{{ t('anonymous.identity') }}</span
              ><span class="block text-xs text-primary">{{ t('anonymous.setup') }}</span></span
            >
          </button>
          <button
            v-else
            type="button"
            :disabled="!usable || loading"
            :aria-pressed="identity === 'persona'"
            class="flex w-full min-w-0 items-center gap-3 rounded-xl p-2.5 text-start transition-colors duration-150 hover:bg-base-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary disabled:cursor-not-allowed disabled:hover:bg-transparent"
            @click="choose('persona')"
          >
            <img v-if="state?.persona?.avatarUrl" :src="state.persona.avatarUrl" alt="" class="h-8 w-8 shrink-0 rounded-full" :class="{ 'opacity-50': !usable }" /><span
              v-else
              class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-base-200 text-base-content/60"
              ><EyeOff class="h-4 w-4"
            /></span><span class="min-w-0 flex-1"
              ><span class="block text-sm font-medium" :class="{ 'text-base-content/55': !usable }">{{ t('anonymous.identity') }}</span
              ><span class="block truncate text-xs" :class="usable ? 'text-base-content/60' : 'text-warning'">{{ personaNote || t('common.loadingShort') }}</span></span
            ><Check v-if="identity === 'persona'" class="h-4 w-4 shrink-0 text-primary" />
          </button>
          <template v-if="state?.persona">
            <div class="mx-2.5 my-1 h-px bg-line/70" aria-hidden="true" />
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-xl px-2.5 py-2.5 text-start text-xs text-base-content/65 transition-colors duration-150 hover:bg-base-200 hover:text-base-content focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
              @click="setup"
            >
              <Settings class="h-3.5 w-3.5 shrink-0" />{{ t('anonymous.manage') }}
            </button>
          </template>
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

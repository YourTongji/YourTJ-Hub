<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, ChevronDown, EyeOff, History, Loader2, Shuffle, X } from '@lucide/vue'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  PopoverContent,
  PopoverPortal,
  PopoverRoot,
  PopoverTrigger,
} from 'reka-ui'
import { useI18n } from 'vue-i18n'
import {
  confirmName,
  disableIdentity,
  generateNames,
  getIdentityState,
  setProfileContent,
  type IdentityState,
  type Persona,
} from '@/runtime/anonymous-identity'
import GfSwitch from './GfSwitch.vue'
import PersonaTag from './PersonaTag.vue'
const props = defineProps<{ open: boolean; returnFocus?: HTMLButtonElement }>()
const emit = defineEmits<{
  'update:open': [open: boolean]
  updated: [state: IdentityState]
  confirmed: [persona: Persona]
}>()
const { t, locale } = useI18n()
const state = ref<IdentityState>()
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const activeBatchId = ref('')
const rulesOpen = ref(false)
const historyOpen = ref(false)
const choice = ref<{ batch: string; index: number; word: string }>()
// A transport failure may hide a successful draw; retry the same request key.
let pendingKey: string | undefined
const locked = computed(
  () => !!state.value?.nameChangeAvailableAt && Date.now() < Date.parse(state.value.nameChangeAvailableAt),
)
const canChoose = computed(
  () => state.value && !locked.value && !state.value.disabled && !state.value.governanceDisabled,
)
const activeBatch = computed(() => state.value?.batches.find((batch) => batch.id === activeBatchId.value))
const activeIndex = computed(() => state.value?.batches.findIndex((batch) => batch.id === activeBatchId.value) ?? -1)
// Newest batch first; the list scrolls inside a fixed-height menu however many draws there are.
const historyBatches = computed(() => [...(state.value?.batches ?? [])].reverse())
const status = computed(() => {
  if (!state.value?.persona) return undefined
  if (state.value.governanceDisabled) return { label: t('anonymous.restricted'), tone: 'bg-error/10 text-error' }
  if (state.value.disabled) return { label: t('anonymous.inactive'), tone: 'bg-base-content/10 text-base-content/65' }
  return { label: t('anonymous.ready'), tone: 'bg-success/12 text-success' }
})
const date = (value: string) =>
  new Intl.DateTimeFormat(String(locale.value), {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
function restoreFocus(event: Event) {
  if (props.returnFocus) {
    event.preventDefault()
    props.returnFocus.focus()
  }
}
async function load() {
  loading.value = true
  error.value = ''
  try {
    const next = await getIdentityState()
    if (state.value?.day !== next.day) {
      pendingKey = undefined
      choice.value = undefined
    }
    state.value = next
    if (!next.batches.some((batch) => batch.id === activeBatchId.value))
      activeBatchId.value = next.batches.at(-1)?.id ?? ''
    emit('updated', next)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}
watch(
  () => props.open,
  (open) => {
    if (open) void load()
    else {
      rulesOpen.value = false
      historyOpen.value = false
    }
  },
  { immediate: true },
)
function updateOpen(open: boolean) {
  if (!busy.value) {
    choice.value = undefined
    emit('update:open', open)
  }
}
async function draw() {
  if (!state.value || busy.value || loading.value) return
  pendingKey ??= crypto.randomUUID()
  busy.value = true
  error.value = ''
  try {
    const batch = await generateNames(state.value.day, pendingKey)
    pendingKey = undefined
    choice.value = undefined
    activeBatchId.value = batch.id
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
async function confirm() {
  if (!choice.value || busy.value || loading.value) return
  busy.value = true
  error.value = ''
  try {
    const persona = await confirmName(choice.value.batch, choice.value.index)
    emit('confirmed', persona)
    choice.value = undefined
    emit('update:open', false)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
async function setActive(active: boolean) {
  if (!state.value || busy.value || loading.value || active === !state.value.disabled) return
  busy.value = true
  error.value = ''
  try {
    await disableIdentity(!active)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
async function setShowContent(showContent: boolean) {
  if (!state.value?.persona || busy.value || loading.value) return
  busy.value = true
  error.value = ''
  try {
    await setProfileContent(showContent)
    // Commit the switch only after success. A subsequent refresh failure
    // cannot revert a preference already saved by the server.
    state.value = { ...state.value, showContent }
    emit('updated', state.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
function pickBatch(id: string) {
  historyOpen.value = false
  if (id === activeBatchId.value) return
  activeBatchId.value = id
  choice.value = undefined
}
</script>
<template>
  <DialogRoot :open="open" @update:open="updateOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2200] bg-black/40 backdrop-blur-[2px]" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2201] flex max-h-[90dvh] w-[calc(100%_-_2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl bg-base-100 text-base-content shadow-[0_24px_64px_-16px_oklch(0_0_0/0.35),0_0_0_1px_var(--gf-color-line)] outline-none"
        :aria-busy="busy || loading"
        @close-auto-focus="restoreFocus"
      >
        <header class="flex shrink-0 items-start justify-between gap-4 px-5 pb-3 pt-5">
          <div class="min-w-0">
            <DialogTitle class="text-lg font-semibold">{{
              state?.persona ? t('anonymous.manage') : t('anonymous.setup')
            }}</DialogTitle>
            <DialogDescription class="mt-1 text-sm leading-6 text-base-content/65">{{
              t('anonymous.purpose')
            }}</DialogDescription>
          </div>
          <button
            type="button"
            class="gf-icon-button -me-2 -mt-1 h-10 w-10 shrink-0"
            :disabled="busy"
            :aria-label="t('anonymous.close')"
            @click="updateOpen(false)"
          >
            <X class="h-4 w-4" />
          </button>
        </header>
        <div class="min-h-0 space-y-6 overflow-y-auto overscroll-contain px-5 pb-5">
          <div
            v-if="loading && !state"
            role="status"
            class="flex items-center justify-center gap-2 py-10 text-sm text-base-content/65"
          >
            <Loader2 class="h-4 w-4 animate-spin" />{{ t('common.loading') }}
          </div>
          <div
            v-if="error"
            role="alert"
            class="flex items-start justify-between gap-3 rounded-xl bg-error/10 px-3 py-2.5 text-sm text-error"
          >
            <span class="min-w-0 break-words">{{ error }}</span
            ><button type="button" class="shrink-0 font-medium underline-offset-2 hover:underline" :disabled="busy || loading" @click="load">
              {{ t('anonymous.retry') }}
            </button>
          </div>
          <template v-if="state">
            <!-- Current identity and its owner-only settings -->
            <section v-if="state.persona" class="space-y-1">
              <div class="flex min-w-0 items-center gap-3 rounded-xl bg-base-200/70 p-3">
                <img :src="state.persona.avatarUrl" alt="" class="h-11 w-11 shrink-0 rounded-full outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10" />
                <div class="min-w-0 flex-1">
                  <p class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                    <span class="break-all font-semibold">{{ state.persona.name }}</span>
                    <span v-if="status" class="rounded-full px-2 py-0.5 text-[11px] font-medium" :class="status.tone">{{ status.label }}</span>
                  </p>
                  <a :href="state.persona.profileUrl" class="mt-0.5 inline-block text-xs text-base-content/60 hover:text-primary">{{ t('anonymous.viewProfile') }}</a>
                </div>
              </div>
              <p v-if="locked" class="px-1 pt-1 text-xs leading-5 text-base-content/55">
                {{ t('anonymous.lockedUntil', { date: date(state.nameChangeAvailableAt!) }) }}
              </p>
              <div class="divide-y divide-line/70 pt-2">
                <div class="flex items-start justify-between gap-4 py-3">
                  <div class="min-w-0">
                    <p id="anonymous-show-content-label" class="text-sm font-medium">{{ t('anonymous.showContent') }}</p>
                    <p id="anonymous-show-content-help" class="mt-0.5 text-xs leading-5 text-base-content/55">{{ t('anonymous.showContentDescription') }}</p>
                  </div>
                  <GfSwitch
                    class="mt-0.5"
                    :model-value="state.showContent"
                    :disabled="busy || loading"
                    labelledby="anonymous-show-content-label"
                    describedby="anonymous-show-content-help"
                    @update:model-value="setShowContent"
                  />
                </div>
                <div class="flex items-start justify-between gap-4 py-3">
                  <div class="min-w-0">
                    <p id="anonymous-active-label" class="text-sm font-medium">{{ t('anonymous.activeLabel') }}</p>
                    <p id="anonymous-active-help" class="mt-0.5 text-xs leading-5" :class="state.governanceDisabled ? 'text-error' : 'text-base-content/55'">
                      {{ state.governanceDisabled ? t('anonymous.statusRestricted') : t('anonymous.disabledHint') }}
                    </p>
                  </div>
                  <GfSwitch
                    class="mt-0.5"
                    :model-value="!state.disabled && !state.governanceDisabled"
                    :disabled="busy || loading || state.governanceDisabled"
                    labelledby="anonymous-active-label"
                    describedby="anonymous-active-help"
                    @update:model-value="setActive"
                  />
                </div>
              </div>
            </section>

            <!-- Name selection: a name-tag preview on top, today's slips below -->
            <section v-if="canChoose" class="space-y-3">
              <div class="gf-name-stage relative overflow-hidden rounded-2xl px-4 py-4">
                <p class="text-[11px] font-medium tracking-wide text-base-content/65">{{ t('anonymous.previewCaption') }}</p>
                <div class="mt-2 flex min-w-0 items-center gap-2.5" aria-live="polite">
                  <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-base-100 text-base-content/55 shadow-[0_1px_2px_oklch(0_0_0/0.08)]">
                    <EyeOff class="h-4 w-4" />
                  </span>
                  <Transition name="gf-name-swap" mode="out-in">
                    <span
                      :key="choice?.word ?? ''"
                      class="min-w-0 truncate text-lg font-semibold"
                      :class="choice ? 'text-base-content' : 'text-base-content/60'"
                    >{{ choice?.word ?? t('anonymous.previewPlaceholder') }}</span>
                  </Transition>
                  <PersonaTag class="shrink-0" />
                </div>
              </div>

              <template v-if="activeBatch">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold">{{ t('anonymous.chooseName') }}</h3>
                  <PopoverRoot v-if="state.batches.length > 1" v-model:open="historyOpen">
                    <PopoverTrigger as-child>
                      <button
                        type="button"
                        :disabled="busy || loading"
                        class="inline-flex h-8 items-center gap-1.5 rounded-full bg-base-200 ps-2.5 pe-2 text-xs font-medium text-base-content/75 transition-colors duration-150 hover:bg-base-300/70 hover:text-base-content focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                        :aria-label="`${t('anonymous.previousBatches')}：${t('anonymous.batchOf', { number: activeIndex + 1, total: state.batches.length })}`"
                      >
                        <History class="h-3.5 w-3.5" />{{ t('anonymous.batchOf', { number: activeIndex + 1, total: state.batches.length }) }}
                        <ChevronDown class="h-3.5 w-3.5 transition-transform duration-150" :class="{ 'rotate-180': historyOpen }" />
                      </button>
                    </PopoverTrigger>
                    <PopoverPortal>
                      <PopoverContent
                        align="end"
                        :side-offset="6"
                        :collision-padding="12"
                        class="z-[2300] w-72 max-w-[calc(100vw_-_1.5rem)] rounded-2xl bg-base-100 p-1.5 text-base-content shadow-[0_12px_32px_-8px_oklch(0_0_0/0.25),0_0_0_1px_var(--gf-color-line)] outline-none"
                      >
                        <p class="px-2.5 pb-1 pt-1.5 text-xs font-medium text-base-content/55">{{ t('anonymous.previousBatches') }}</p>
                        <ul class="max-h-60 overflow-y-auto overscroll-contain" role="list">
                          <li v-for="(batch, index) in historyBatches" :key="batch.id">
                            <button
                              type="button"
                              :aria-current="batch.id === activeBatchId ? 'true' : undefined"
                              class="flex w-full min-w-0 items-start gap-2.5 rounded-xl px-2.5 py-2 text-start transition-colors duration-150 hover:bg-base-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                              @click="pickBatch(batch.id)"
                            >
                              <span class="mt-0.5 w-12 shrink-0 text-xs font-semibold tabular-nums" :class="batch.id === activeBatchId ? 'text-primary' : 'text-base-content/55'">
                                {{ t('anonymous.batchNumber', { number: state.batches.length - index }) }}
                              </span>
                              <span class="min-w-0 flex-1 text-sm leading-5 text-base-content/80 [overflow-wrap:anywhere]">{{ batch.words.join(' · ') }}</span>
                              <Check v-if="batch.id === activeBatchId" class="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                            </button>
                          </li>
                        </ul>
                      </PopoverContent>
                    </PopoverPortal>
                  </PopoverRoot>
                </div>
                <div :key="activeBatch.id" class="grid grid-cols-2 gap-2.5 p-0.5" role="group" :aria-label="t('anonymous.chooseName')">
                  <button
                    v-for="(word, index) in activeBatch.words"
                    :key="index"
                    type="button"
                    :title="word"
                    :aria-pressed="choice?.batch === activeBatch.id && choice.index === index"
                    :disabled="busy || loading"
                    class="gf-name-slip group relative min-h-12 min-w-0 rounded-xl px-3 py-3 text-[15px] [overflow-wrap:anywhere] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
                    :class="
                      choice?.batch === activeBatch.id && choice.index === index
                        ? 'gf-name-slip-selected bg-primary/10 font-semibold text-base-content'
                        : 'bg-base-100 text-base-content/85 hover:text-base-content'
                    "
                    :style="{ '--i': index, '--tilt': index % 2 ? '0.6deg' : '-0.6deg' }"
                    @click="choice = { batch: activeBatch.id, index, word }"
                  >
                    <span class="pointer-events-none absolute end-2 top-1.5 text-[10px] font-medium tabular-nums text-base-content/60" aria-hidden="true">{{ String(index + 1).padStart(2, '0') }}</span>
                    {{ word }}
                    <Check
                      v-if="choice?.batch === activeBatch.id && choice.index === index"
                      class="absolute -end-1.5 -top-1.5 h-5 w-5 rounded-full bg-primary p-1 text-primary-content shadow-sm"
                      aria-hidden="true"
                    />
                  </button>
                </div>
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <button
                    type="button"
                    :disabled="busy || loading || state.remaining === 0"
                    class="gf-button gf-button-md gf-button-secondary"
                    @click="draw"
                  >
                    <Loader2 v-if="busy" class="h-4 w-4 animate-spin" /><Shuffle v-else class="h-4 w-4" />{{ t('anonymous.refreshNames') }}
                  </button>
                  <span class="text-xs tabular-nums text-base-content/55">{{
                    t('anonymous.drawsRemaining', { remaining: state.remaining })
                  }}</span>
                </div>
              </template>
              <div v-else class="space-y-4">
                <div class="grid grid-cols-3 gap-2.5" aria-hidden="true">
                  <span v-for="index in 3" :key="index" class="gf-name-slip-back flex h-14 items-center justify-center rounded-xl text-lg font-semibold text-base-content/25" :style="{ '--i': index - 1, '--tilt': ['-1.5deg', '1deg', '-0.5deg'][index - 1] }">?</span>
                </div>
                <div>
                  <p class="text-sm font-semibold">{{ t('anonymous.introTitle') }}</p>
                  <p class="mt-1 text-sm leading-6 text-base-content/65">{{ t('anonymous.introBody') }}</p>
                </div>
                <div class="flex flex-wrap items-center gap-3">
                  <button
                    type="button"
                    :disabled="busy || loading || state.remaining === 0"
                    class="gf-button gf-button-md gf-button-primary"
                    @click="draw"
                  >
                    <Loader2 v-if="busy" class="h-4 w-4 animate-spin" /><Shuffle v-else class="h-4 w-4" />{{ t('anonymous.generateNames') }}
                  </button>
                  <span class="text-xs tabular-nums text-base-content/55">{{
                    t('anonymous.drawsRemaining', { remaining: state.remaining })
                  }}</span>
                </div>
              </div>
              <p v-if="state.remaining === 0" role="status" class="text-xs leading-5 text-base-content/60">
                {{ t('anonymous.quota', { date: date(state.resetsAt) }) }}
              </p>
            </section>

            <!-- Privacy boundary: summary first, full rules on demand -->
            <section class="space-y-2 text-xs leading-5 text-base-content/60">
              <p>{{ t('anonymous.privacySummary') }}</p>
              <button
                type="button"
                class="inline-flex items-center gap-1 font-medium text-base-content/70 hover:text-primary"
                :aria-expanded="rulesOpen"
                aria-controls="anonymous-rules"
                @click="rulesOpen = !rulesOpen"
              >
                {{ t('anonymous.rules') }}
                <ChevronDown class="h-3.5 w-3.5 transition-transform duration-150" :class="{ 'rotate-180': rulesOpen }" />
              </button>
              <div v-show="rulesOpen" id="anonymous-rules" class="space-y-2">
                <p>{{ t('anonymous.boundary') }}</p>
                <p>{{ t('anonymous.linkabilityHint') }}</p>
              </div>
            </section>
          </template>
        </div>
        <footer
          v-if="canChoose && activeBatch"
          class="flex shrink-0 flex-wrap items-center justify-between gap-x-4 gap-y-3 border-t border-line/70 px-5 py-4"
        >
          <p class="min-w-0 flex-1 basis-56 text-xs leading-5 text-base-content/65">{{ t('anonymous.confirmHint') }}</p>
          <div class="ms-auto flex items-center gap-2">
            <button
              type="button"
              :disabled="busy"
              class="gf-button gf-button-md gf-button-secondary"
              @click="updateOpen(false)"
            >
              {{ t('anonymous.cancel') }}</button
            ><button
              type="button"
              :disabled="!choice || busy || loading"
              class="gf-button gf-button-md gf-button-primary"
              @click="confirm"
            >
              <Loader2 v-if="busy" class="h-4 w-4 animate-spin" />{{ t('anonymous.confirmName') }}
            </button>
          </div>
        </footer>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
/* Name slips: a soft paper stage, slips that fan in when a batch is drawn, and a lift on pick. */
.gf-name-stage {
  background:
    radial-gradient(120% 140% at 100% 0%, color-mix(in oklab, var(--gf-color-primary) 14%, transparent), transparent 60%),
    color-mix(in oklab, var(--gf-color-base-200) 80%, transparent);
}
.gf-name-slip,
.gf-name-slip-back {
  box-shadow: 0 1px 2px oklch(0 0 0 / 0.06), 0 0 0 1px var(--gf-color-line);
  rotate: var(--tilt, 0deg);
  transition-property: rotate, translate, box-shadow, background-color, color;
  transition-duration: 150ms;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}
.gf-name-slip-back {
  background: repeating-linear-gradient(135deg, var(--gf-color-base-200) 0 6px, var(--gf-color-base-100) 6px 12px);
}
.gf-name-slip:hover:not(:disabled) {
  rotate: 0deg;
  translate: 0 -1px;
}
.gf-name-slip-selected {
  rotate: 0deg;
  translate: 0 -2px;
  box-shadow: 0 6px 16px -6px color-mix(in oklab, var(--gf-color-primary) 45%, transparent), 0 0 0 1.5px var(--gf-color-primary);
}
.gf-name-slip:active:not(:disabled) {
  scale: 0.96;
}
@media (prefers-reduced-motion: no-preference) {
  .gf-name-slip {
    animation: gf-name-slip-in 280ms cubic-bezier(0.2, 0, 0, 1) backwards;
    animation-delay: calc(var(--i) * 45ms);
  }
}
@keyframes gf-name-slip-in {
  from {
    opacity: 0;
    translate: 0 8px;
    rotate: 0deg;
  }
}
.gf-name-swap-enter-active,
.gf-name-swap-leave-active {
  transition: opacity 150ms ease-out, translate 150ms ease-out;
}
.gf-name-swap-enter-from {
  opacity: 0;
  translate: 0 4px;
}
.gf-name-swap-leave-to {
  opacity: 0;
  translate: 0 -4px;
}
</style>

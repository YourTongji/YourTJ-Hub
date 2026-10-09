<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, EyeOff, Loader2, RefreshCw, X } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import { createUuidV4 } from '@/runtime/uuid'
import {
  confirmName,
  disableIdentity,
  generateNames,
  getIdentityState,
  setProfileContent,
  type IdentityState,
  type Persona,
} from '@/runtime/anonymous-identity'
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
  pendingKey ??= createUuidV4()
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
async function toggle() {
  if (!state.value || busy.value || loading.value) return
  busy.value = true
  error.value = ''
  try {
    await disableIdentity(!state.value.disabled)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
async function toggleContent() {
  if (!state.value?.persona || busy.value || loading.value) return
  busy.value = true
  error.value = ''
  try {
    const showContent = !state.value.showContent
    await setProfileContent(showContent)
    // Commit the checkbox only after success. A subsequent refresh failure
    // cannot revert a preference already saved by the server.
    state.value = { ...state.value, showContent }
    emit('updated', state.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <DialogRoot :open="open" @update:open="updateOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2200] bg-black/40 backdrop-blur-[2px]" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2201] flex max-h-[90dvh] w-[calc(100%_-_2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-box border border-line bg-base-100 text-base-content shadow-2xl outline-none"
        :aria-busy="busy || loading"
        @close-auto-focus="restoreFocus"
      >
        <header class="flex shrink-0 items-start justify-between gap-4 border-b border-line p-5">
          <div class="min-w-0">
            <DialogTitle class="text-lg font-semibold">{{
              state?.persona ? t('anonymous.manage') : t('anonymous.setup')
            }}</DialogTitle>
            <DialogDescription class="mt-1.5 text-sm leading-6 text-base-content/65">{{
              t('anonymous.purpose')
            }}</DialogDescription>
          </div>
          <button
            type="button"
            class="gf-icon-button h-9 w-9 shrink-0"
            :disabled="busy"
            :aria-label="t('anonymous.close')"
            @click="updateOpen(false)"
          >
            <X class="h-4 w-4" />
          </button>
        </header>
        <div class="min-h-0 space-y-5 overflow-y-auto overscroll-contain p-5">
          <div
            v-if="loading && !state"
            role="status"
            class="flex items-center justify-center gap-2 py-8 text-sm text-base-content/65"
          >
            <Loader2 class="h-4 w-4 animate-spin" />{{ t('common.loading') }}
          </div>
          <div
            v-if="error"
            role="alert"
            class="flex items-start justify-between gap-3 rounded-lg bg-error/10 p-3 text-sm text-error"
          >
            <span class="min-w-0 break-words">{{ error }}</span
            ><button type="button" class="shrink-0 underline" :disabled="busy || loading" @click="load">
              {{ t('anonymous.retry') }}
            </button>
          </div>
          <template v-if="state">
            <div v-if="state.persona" class="flex min-w-0 items-center gap-3">
              <img :src="state.persona.avatarUrl" :alt="state.persona.name" class="h-12 w-12 shrink-0 rounded-full" />
              <div class="min-w-0 flex-1">
                <a :href="state.persona.profileUrl" class="break-all font-semibold hover:text-primary">{{
                  state.persona.name
                }}</a>
                <p class="mt-1 text-xs text-base-content/60">
                  {{
                    state.governanceDisabled
                      ? t('anonymous.unavailable')
                      : state.disabled
                        ? t('anonymous.inactive')
                        : t('anonymous.ready')
                  }}
                </p>
              </div>
            </div>
            <p v-if="locked" class="text-sm leading-6 text-base-content/65">
              {{
                t('anonymous.lockedUntil', {
                  date: date(state.nameChangeAvailableAt!),
                })
              }}
            </p>
            <label v-if="state.persona" class="flex items-start justify-between gap-4 rounded-lg border border-line p-4">
              <span class="min-w-0">
                <span class="block text-sm font-semibold">{{ t('anonymous.showContent') }}</span>
                <span class="mt-1 block text-xs leading-5 text-base-content/60">{{
                  t('anonymous.showContentDescription')
                }}</span>
              </span>
              <input
                type="checkbox"
                :checked="state.showContent"
                :disabled="busy || loading"
                :aria-label="t('anonymous.showContent')"
                class="mt-0.5 h-5 w-5 shrink-0 rounded border-line text-primary"
                @click.prevent="toggleContent"
              />
            </label>
            <p v-if="state.governanceDisabled" role="status" class="text-sm text-error">
              {{ t('anonymous.unavailable') }}
            </p>
            <button
              v-if="state.persona && !state.governanceDisabled"
              type="button"
              :disabled="busy || loading"
              class="gf-button gf-button-xl gf-button-secondary"
              @click="toggle"
            >
              {{ state.disabled ? t('anonymous.enable') : t('anonymous.disable') }}
            </button>
            <template v-if="canChoose">
              <div v-if="activeBatch" class="space-y-3">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold">
                    {{ t('anonymous.chooseName') }}
                  </h3>
                  <select
                    v-if="state.batches.length > 1"
                    v-model="activeBatchId"
                    :disabled="busy || loading"
                    class="gf-input w-auto max-w-full text-xs"
                    :aria-label="t('anonymous.previousBatches')"
                    @change="choice = undefined"
                  >
                    <option v-for="(batch, index) in state.batches" :key="batch.id" :value="batch.id">
                      {{ t('anonymous.batchNumber', { number: index + 1 }) }}
                    </option>
                  </select>
                </div>
                <div class="grid grid-cols-2 gap-2" role="group" :aria-label="t('anonymous.chooseName')">
                  <button
                    v-for="(word, index) in activeBatch.words"
                    :key="index"
                    type="button"
                    :title="word"
                    :aria-pressed="choice?.batch === activeBatch.id && choice.index === index"
                    :disabled="busy || loading"
                    class="relative min-h-11 min-w-0 break-all rounded-lg border px-3 py-2.5 text-sm transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                    :class="
                      choice?.batch === activeBatch.id && choice.index === index
                        ? 'border-primary bg-primary/5 font-semibold text-primary'
                        : 'border-line bg-base-100 hover:border-primary/50 hover:bg-base-200'
                    "
                    @click="choice = { batch: activeBatch.id, index, word }"
                  >
                    {{ word }}
                  </button>
                </div>
              </div>
              <div v-else class="rounded-lg bg-base-200/70 px-4 py-5">
                <EyeOff class="mb-3 h-7 w-7 text-base-content/60" />
                <p class="text-sm font-semibold">
                  {{ t('anonymous.introTitle') }}
                </p>
                <p class="mt-1.5 text-sm leading-6 text-base-content/65">
                  {{ t('anonymous.historyHint') }}
                </p>
              </div>
              <div class="flex flex-wrap items-center justify-between gap-3">
                <button
                  type="button"
                  :disabled="busy || loading || state.remaining === 0"
                  class="gf-button gf-button-xl"
                  :class="activeBatch ? 'gf-button-secondary' : 'gf-button-primary'"
                  @click="draw"
                >
                  <Loader2 v-if="busy" class="h-4 w-4 animate-spin" /><RefreshCw
                    v-else-if="activeBatch"
                    class="h-4 w-4"
                  />{{ activeBatch ? t('anonymous.refreshNames') : t('anonymous.chooseName') }}
                </button>
                <span class="text-xs text-base-content/60">{{
                  t('anonymous.drawsRemaining', { remaining: state.remaining })
                }}</span>
              </div>
              <p v-if="state.remaining === 0" role="status" class="text-xs leading-5 text-base-content/65">
                {{
                  t('anonymous.quota', {
                    remaining: 0,
                    date: date(state.resetsAt),
                  })
                }}
              </p>
              <div
                v-if="choice"
                class="flex min-w-0 items-center gap-3 rounded-lg border border-primary/20 bg-primary/5 p-3"
              >
                <EyeOff class="h-5 w-5 shrink-0 text-primary" />
                <div class="min-w-0">
                  <p class="text-xs text-base-content/60">
                    {{ t('anonymous.namePreview') }}
                  </p>
                  <p class="mt-0.5 break-all text-sm font-semibold">
                    {{ choice.word }}
                  </p>
                </div>
                <Check class="ml-auto h-4 w-4 shrink-0 text-primary" />
              </div>
            </template>
            <div class="space-y-2 border-t border-line pt-4 text-xs leading-5 text-base-content/65">
              <p v-if="canChoose && !activeBatch" class="font-medium text-base-content/80">
                {{ t('anonymous.confirmHint') }}
              </p>
              <p>{{ t('anonymous.privacySummary') }}</p>
              <details>
                <summary class="cursor-pointer text-base-content/70 hover:text-primary">
                  {{ t('anonymous.rules') }}
                </summary>
                <p class="mt-2 leading-6">{{ t('anonymous.boundary') }}</p>
                <p class="mt-2 leading-6">{{ t('anonymous.historyHint') }}</p>
              </details>
            </div>
          </template>
        </div>
        <footer
          v-if="canChoose && activeBatch"
          class="shrink-0 space-y-3 border-t border-line bg-base-200/40 px-5 py-4"
        >
          <p class="text-xs leading-5 text-base-content/80">{{ t('anonymous.confirmHint') }}</p>
          <div class="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              :disabled="busy"
              class="gf-button gf-button-xl gf-button-secondary"
              @click="updateOpen(false)"
            >
              {{ t('anonymous.cancel') }}</button
            ><button
              type="button"
              :disabled="!choice || busy || loading"
              class="gf-button gf-button-xl gf-button-primary"
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

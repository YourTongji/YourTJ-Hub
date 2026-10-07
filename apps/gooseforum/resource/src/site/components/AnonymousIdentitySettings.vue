<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { confirmName, disableIdentity, generateNames, getIdentityState, type IdentityState } from '@/runtime/anonymous-identity'
const { t } = useI18n()
const state = ref<IdentityState>()
const busy = ref(false)
const error = ref('')
const choice = ref<{ batch: string; index: number; word: string }>()
// Keep the key after transport failure; a lost success response must not consume another batch.
let pendingKey: string | undefined
const locked = computed(() =>
  !!state.value?.nameChangeAvailableAt && Date.now() < Date.parse(state.value.nameChangeAvailableAt),
)
async function load() {
  try {
    const next = await getIdentityState()
    if (state.value?.day !== next.day) {
      pendingKey = undefined
      choice.value = undefined
    }
    state.value = next
    error.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}
async function run(action: () => Promise<unknown>) {
  busy.value = true
  error.value = ''
  try {
    await action()
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
function draw() {
  if (!state.value) return
  const day = state.value.day
  pendingKey ??= crypto.randomUUID()
  const key = pendingKey
  void run(async () => {
    await generateNames(day, key)
    pendingKey = undefined
  })
}
function select(batch: string, index: number) {
  void run(async () => {
    await confirmName(batch, index)
    choice.value = undefined
  })
}
onMounted(load)
</script>
<template>
 <section id="anonymous-identity" class="my-6 rounded-box border border-line p-4" :aria-busy="busy">
  <h2 class="text-lg font-semibold">{{ t('anonymous.identity') }}</h2>
  <p class="my-2 text-sm text-base-content/70">{{ t('anonymous.boundary') }}</p>
  <p v-if="error" role="alert" class="text-error">{{ error }}</p>
  <button v-if="!state || error" class="gf-button gf-button-secondary" @click="load">{{ t('anonymous.retry') }}</button>
  <template v-if="state">
   <div v-if="state.persona" class="flex min-w-0 items-center gap-3 py-3">
    <img :src="state.persona.avatarUrl" :alt="state.persona.name" class="h-12 w-12 rounded-full" />
    <a :href="state.persona.profileUrl" class="min-w-0 break-all font-semibold">{{ state.persona.name }}</a>
   </div>
   <p v-if="locked" class="text-sm">{{ t('anonymous.lockedUntil', {date: new Date(state.nameChangeAvailableAt!).toLocaleString()}) }}</p>
   <p v-if="state.governanceDisabled" role="status">{{ t('anonymous.unavailable') }}</p>
   <button v-if="state.persona && !state.governanceDisabled" :disabled="busy" class="gf-button gf-button-secondary my-2" @click="run(() => disableIdentity(!state!.disabled))">{{ state.disabled ? t('anonymous.enable') : t('anonymous.disable') }}</button>
   <template v-if="!locked && !state.disabled && !state.governanceDisabled">
    <p class="my-2 text-sm">{{ t('anonymous.quota', {remaining: state.remaining, date: new Date(state.resetsAt).toLocaleString()}) }}</p>
    <button :disabled="busy || state.remaining === 0" class="gf-button gf-button-primary" @click="draw">{{ t('anonymous.randomize') }}</button>
    <fieldset v-for="(batch, batchIndex) in state.batches" :key="batch.id" class="my-4" :disabled="busy">
     <legend class="mb-2 text-sm">{{ t('anonymous.batch', {number: batchIndex + 1}) }}</legend>
     <div class="grid grid-cols-2 gap-2 sm:grid-cols-5">
      <button v-for="(word,index) in batch.words" :key="index" type="button" :title="word" class="min-w-0 break-all rounded border border-line p-2 text-sm hover:border-primary focus-visible:outline focus-visible:outline-primary" @click="choice = {batch: batch.id,index,word}">{{ word }}</button>
     </div>
    </fieldset>
   <div v-if="choice" class="my-3 space-y-2"><p class="break-all">{{ choice.word }}</p><button :disabled="busy" class="gf-button gf-button-primary" @click="select(choice.batch,choice.index)">{{ t('anonymous.confirm') }}</button><button :disabled="busy" class="gf-button gf-button-secondary" @click="choice = undefined">{{ t('anonymous.cancel') }}</button></div>
   </template>
  </template>
 </section>
</template>

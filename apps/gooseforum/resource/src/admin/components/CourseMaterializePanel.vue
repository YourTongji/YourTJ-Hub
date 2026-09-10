<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Loader2, RefreshCw } from '@lucide/vue'
import { Button } from './ui/button'
import { materializePkCalendar } from '../runtime/api'
import { adminText } from '../runtime/i18n-text'
import type { PkMaterializeResult, PkSyncStatusItem } from '../types'

const props = defineProps<{ calendars: PkSyncStatusItem[], syncing?: boolean }>()
const term = ref('')
const busy = ref(false)
const error = ref('')
const result = ref<PkMaterializeResult | null>(null)
const calendars = computed(() => props.calendars.filter(item => item.calendarName))
const disabled = computed(() => busy.value || props.syncing || !term.value
  || props.calendars.some(item => String(item.calendarId) === term.value && item.status === 'running'))
watch(calendars, items => {
  if (!term.value && items.length) term.value = String(items[0].calendarId)
}, { immediate: true })
watch(term, () => { result.value = null; error.value = '' })

async function materialize() {
  if (disabled.value) return
  busy.value = true
  error.value = ''
  result.value = null
  try {
    result.value = await materializePkCalendar(term.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('materializeFailed')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="space-y-4 rounded-lg border border-border bg-card p-5" :aria-busy="busy">
    <h2 class="text-base font-medium">{{ adminText('materializeTitle') }}</h2>
    <p class="text-sm text-muted-foreground">{{ adminText('materializeHint') }}</p>
    <div class="flex flex-wrap items-end gap-3">
      <label class="grid min-w-0 flex-1 gap-2 text-sm font-medium">
        {{ adminText('k00th') }}
        <select v-model="term" :disabled="busy || calendars.length === 0" class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm">
          <option v-if="calendars.length === 0" value="">{{ adminText('materializeNoTerms') }}</option>
          <option v-for="item in calendars" :key="item.calendarId" :value="String(item.calendarId)">
            {{ item.calendarName }} ({{ item.calendarId }})
          </option>
        </select>
      </label>
      <Button type="button" class="h-auto max-w-full whitespace-normal text-left" :disabled="disabled" @click="materialize">
        <Loader2 v-if="busy" class="size-4 animate-spin" />
        <RefreshCw v-else class="size-4" />
        {{ adminText(busy ? 'materializeRunning' : 'materializeTitle') }}
      </Button>
    </div>
    <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
    <div v-if="result" role="status" class="space-y-2 rounded-md bg-muted p-3 text-sm">
      <p class="font-medium">{{ adminText('materializeDone', { term: result.calendarId }) }}</p>
      <p>{{ adminText('materializeCourses', { added: result.coursesInserted, updated: result.coursesUpdated }) }}</p>
      <p>{{ adminText('materializeOfferings', { added: result.offeringsInserted, updated: result.offeringsUpdated }) }}</p>
      <p class="text-muted-foreground">{{ adminText('materializeProjections') }}</p>
    </div>
  </section>
</template>

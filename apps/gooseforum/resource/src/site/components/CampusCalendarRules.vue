<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { onBeforeUnmount, ref } from 'vue'
import type { CampusCalendarRules } from '@gooseforum/client'
import { calendarRulesAPI, teachingDateLabel } from '@/runtime/calendar-rules-api'
const { t, locale } = useI18n()

const rules = ref<CampusCalendarRules | null>(null), error = ref(''), loading = ref(false)
const controller = new AbortController()
async function load(event: Event) {
  if (!(event.target as HTMLDetailsElement).open || loading.value) return
  loading.value = true; error.value = ''; rules.value = null
  try { rules.value = (await calendarRulesAPI.read(false, controller.signal)).rules }
  catch (e) { if (!controller.signal.aborted) error.value = e instanceof Error ? e.message : t('campus.errorRules') }
  finally { loading.value = false }
}
onBeforeUnmount(() => controller.abort())
</script>
<template>
  <details class="border-b border-line px-4 py-3 text-xs leading-6" @toggle="load">
    <summary class="cursor-pointer text-base-content/60">{{ t('campus.rulesTitle') }}</summary>
    <p v-if="loading" class="mt-2 text-base-content/55">{{ t('campus.rulesLoading') }}</p>
    <p v-if="error" role="alert" class="mt-2 text-error">{{ error }}</p>
    <div v-if="rules" class="mt-2 space-y-2">
      <p class="text-base-content/55">{{ t('campus.rulesHint') }}</p>
      <p v-if="!rules.holidays.length && !rules.moves.length" class="text-base-content/55">{{ t('campus.rulesEmpty') }}</p>
      <p v-for="h in rules.holidays" :key="h.startDate"><span class="gf-badge mr-2">{{ t('campus.holiday') }}</span>{{ h.name }} · {{ t('campus.dateRange', { start: teachingDateLabel(h.startDate, locale), end: teachingDateLabel(h.endDate, locale) }) }}</p>
      <p v-for="m in rules.moves" :key="m.fromDate"><span class="gf-badge mr-2">{{ t('campus.makeup') }}</span>{{ m.name }} · {{ t('campus.moveDescription', { to: teachingDateLabel(m.toDate, locale), from: teachingDateLabel(m.fromDate, locale) }) }}</p>
    </div>
  </details>
</template>

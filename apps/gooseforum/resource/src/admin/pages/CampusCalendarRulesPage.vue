<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import type { CampusCalendarRules, CampusCalendarSettings } from '@gooseforum/client'
import { CalendarDays, Sparkles, Plus, Trash2, ArrowRight, Save } from '@lucide/vue'
import { BasicPage } from '@/admin/components/global-layout'
import AdminSection from '@/admin/components/AdminSection.vue'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Textarea } from '@/admin/components/ui/textarea'
import { calendarRulesAPI, teachingDateLabel } from '@/runtime/calendar-rules-api'

const { t, locale } = useI18n()

const rules = ref<CampusCalendarRules>({ holidays: [], moves: [] })
const revision = ref(''), year = ref(new Date().getFullYear()), notice = ref('')
const loading = ref(true), busy = ref(false), error = ref(''), success = ref(''), warnings = ref<string[]>([])
const controller = new AbortController()
const dirty = ref(false)
const count = computed(() => rules.value.holidays.length + rules.value.moves.length)
function changed() { dirty.value = true; success.value = '' }
function accept(settings: CampusCalendarSettings) { rules.value = settings.rules; revision.value = settings.revision; dirty.value = false }
async function load() {
  loading.value = true; error.value = ''
  try { accept(await calendarRulesAPI.read(true, controller.signal)); warnings.value = [] }
  catch (e) { if (!controller.signal.aborted) error.value = String(e instanceof Error ? e.message : e) }
  finally { loading.value = false }
}
async function parse() {
  busy.value = true; error.value = ''; success.value = ''
  try {
    const draft = await calendarRulesAPI.parse({ year: Number(year.value), text: notice.value }, controller.signal)
    // Add to the draft without deleting previously published dates. Exact duplicates
    // are ignored; other conflicts remain visible for editing and server validation.
    const merge = <T,>(existing: T[], incoming: T[]) => [...existing, ...incoming.filter(v => !existing.some(e => JSON.stringify(e) === JSON.stringify(v)))]
    rules.value = { holidays: merge(rules.value.holidays, draft.rules.holidays), moves: merge(rules.value.moves, draft.rules.moves) }
    warnings.value = draft.warnings; changed()
    success.value = t('campus.adminDraftAdded')
  } catch (e) { if (!controller.signal.aborted) error.value = e instanceof Error ? e.message : t('campus.adminParseFailed') }
  finally { busy.value = false }
}
async function save() {
  busy.value = true; error.value = ''; success.value = ''
  try { accept(await calendarRulesAPI.save({ revision: revision.value, rules: rules.value }, controller.signal)); success.value = t('campus.adminApplied') }
  catch (e) { if (!controller.signal.aborted) error.value = e instanceof Error ? e.message : t('campus.adminSaveFailed') }
  finally { busy.value = false }
}
onMounted(load)
onBeforeUnmount(() => controller.abort())
</script>

<template>
  <BasicPage :title="t('campus.adminTitle')" :description="t('campus.adminDescription')">
    <div class="space-y-5">
      <div v-if="error" role="alert" class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">{{ error }}</div>
      <div v-if="success" role="status" class="rounded-lg border bg-muted/30 p-4 text-sm">{{ success }}</div>
      <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
        <span class="text-muted-foreground">{{ loading ? t('campus.adminLoading') : t('campus.adminCount', { count, state: dirty ? t('campus.adminDirty') : t('campus.adminPublished') }) }}</span>
        <Button variant="outline" :disabled="busy || loading" @click="load">{{ t('campus.adminReload') }}</Button>
      </div>
      <fieldset :disabled="busy || loading || !revision" class="min-w-0 space-y-5 disabled:opacity-70">
        <AdminSection body-class="space-y-4 p-5">
          <h2 class="flex items-center gap-2 font-semibold"><Sparkles class="size-4" /> {{ t('campus.adminGenerateTitle') }}</h2>
          <p class="text-sm leading-6 text-muted-foreground">{{ t('campus.adminAIHint') }} <a href="/admin/settings/ai-summary" class="underline underline-offset-4">{{ t('campus.adminAISettings') }}</a></p>
          <label class="block max-w-40 space-y-2 text-sm">{{ t('campus.adminYear') }}<Input v-model.number="year" :aria-label="t('campus.adminYear')" type="number" min="2000" max="2100" /></label>
          <label class="block space-y-2 text-sm">{{ t('campus.adminNoticeLabel') }}<Textarea v-model="notice" :aria-label="t('campus.adminNotice')" :maxlength="12000" rows="5" :placeholder="t('campus.adminExample')" /></label>
          <Button :disabled="!notice.trim() || busy || loading" @click="parse"><Sparkles class="size-4" />{{ busy ? t('campus.adminProcessing') : t('campus.adminGenerate') }}</Button>
          <p class="text-xs text-muted-foreground">{{ t('campus.adminReviewHint') }}</p>
        </AdminSection>
        <div v-if="warnings.length" class="rounded-lg border bg-muted/30 p-4 text-sm" role="alert">
          <p class="font-medium">{{ t('campus.adminWarnings') }}</p><ul class="mt-2 list-inside list-disc space-y-1"><li v-for="(warning, i) in warnings" :key="i">{{ warning }}</li></ul>
        </div>
        <AdminSection body-class="space-y-4 p-5">
          <div class="flex flex-wrap items-center justify-between gap-2"><h2 class="flex items-center gap-2 font-semibold"><CalendarDays class="size-4" /> {{ t('campus.adminHolidays') }} <span class="text-muted-foreground">{{ rules.holidays.length }}</span></h2><Button variant="outline" @click="rules.holidays.push({ name: '', startDate: '', endDate: '' }); changed()"><Plus class="size-4" />{{ t('campus.adminAddHoliday') }}</Button></div>
          <p class="text-sm text-muted-foreground">{{ t('campus.adminHolidayHint') }}</p>
          <p v-if="!rules.holidays.length" class="py-4 text-center text-sm text-muted-foreground">{{ t('campus.adminNoHolidays') }}</p>
          <div v-for="(holiday, i) in rules.holidays" :key="i" class="grid gap-3 rounded-lg border bg-muted/10 p-3 md:grid-cols-[1fr_1fr_1fr_auto]">
            <label class="space-y-1 text-xs">{{ t('campus.adminHolidayName') }}<Input v-model="holiday.name" :aria-label="t('campus.adminHolidayNameLabel', { index: i + 1 })" maxlength="80" @update:model-value="changed" /></label>
            <label class="space-y-1 text-xs">{{ t('campus.fieldStartDate') }}<Input v-model="holiday.startDate" :aria-label="t('campus.adminHolidayStartLabel', { index: i + 1 })" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(holiday.startDate, locale) }}</span></label>
            <label class="space-y-1 text-xs">{{ t('campus.fieldEndDate') }}<Input v-model="holiday.endDate" :aria-label="t('campus.adminHolidayEndLabel', { index: i + 1 })" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(holiday.endDate, locale) }}</span></label>
            <Button variant="ghost" :aria-label="t('campus.adminDeleteHoliday', { index: i + 1 })" class="self-center" @click="rules.holidays.splice(i, 1); changed()"><Trash2 class="size-4" /></Button>
          </div>
        </AdminSection>
        <AdminSection body-class="space-y-4 p-5">
          <div class="flex flex-wrap items-center justify-between gap-2"><h2 class="font-semibold">{{ t('campus.adminMoves') }} <span class="text-muted-foreground">{{ rules.moves.length }}</span></h2><Button variant="outline" @click="rules.moves.push({ name: '', fromDate: '', toDate: '' }); changed()"><Plus class="size-4" />{{ t('campus.adminAddMove') }}</Button></div>
          <p class="text-sm leading-6 text-muted-foreground">{{ t('campus.adminMoveHint') }}</p>
          <p v-if="!rules.moves.length" class="py-4 text-center text-sm text-muted-foreground">{{ t('campus.adminNoMoves') }}</p>
          <div v-for="(move, i) in rules.moves" :key="i" class="grid items-start gap-3 rounded-lg border bg-muted/10 p-3 md:grid-cols-[1fr_1fr_auto_1fr_auto]">
            <label class="space-y-1 text-xs">{{ t('campus.adminMoveName') }}<Input v-model="move.name" :aria-label="t('campus.adminMoveNameLabel', { index: i + 1 })" maxlength="80" @update:model-value="changed" /></label>
            <label class="space-y-1 text-xs">{{ t('campus.adminFrom') }}<Input v-model="move.fromDate" :aria-label="t('campus.adminFromLabel', { index: i + 1 })" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(move.fromDate, locale) }}</span></label>
            <ArrowRight class="hidden size-4 self-center text-muted-foreground md:block" />
            <label class="space-y-1 text-xs">{{ t('campus.adminTo') }}<Input v-model="move.toDate" :aria-label="t('campus.adminToLabel', { index: i + 1 })" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(move.toDate, locale) }}</span></label>
            <Button variant="ghost" :aria-label="t('campus.adminDeleteMove', { index: i + 1 })" class="self-center" @click="rules.moves.splice(i, 1); changed()"><Trash2 class="size-4" /></Button>
          </div>
        </AdminSection>
        <div class="flex flex-wrap items-center gap-4"><Button :disabled="!dirty || busy || loading" @click="save"><Save class="size-4" />{{ t('campus.adminApply') }}</Button><span class="text-xs text-muted-foreground">{{ t('campus.adminApplyHint') }}</span></div>
      </fieldset>
    </div>
  </BasicPage>
</template>

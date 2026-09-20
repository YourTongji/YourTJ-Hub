<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { campusFieldKey } from '@/site/utils/campusFields'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { CampusStatus, CampusDataset, CampusDatasetKey, CampusMessageSummary, CampusMessageDetail, LayoutPayload } from '@gooseforum/client'
import { ArrowUpRight, Bell, BookOpen, CalendarDays, ChartNoAxesCombined, Check, ChevronLeft, ChevronRight, Download, GraduationCap, Link2, Loader2, RefreshCw, ShieldCheck, Shuffle, Unplug } from '@lucide/vue'
import { setBaseDocumentTitle } from '@/runtime/document-title'
import { campusAPI, CampusError } from '@/runtime/campus-api'
import PageHeader from '@/site/components/PageHeader.vue'
import SectionHeader from '@/site/components/SectionHeader.vue'
import EmptyState from '@/site/components/EmptyState.vue'
import CampusCalendarRules from '@/site/components/CampusCalendarRules.vue'
import CampusTimetable from '@/site/components/CampusTimetable.vue'
import CampusMessageDialog from '@/site/components/CampusMessageDialog.vue'
import { timetableCourseStyle } from '@/site/utils/timetableCourseStyle'
import { saveCampusMessageReturn, takeCampusMessageReturn } from '@/site/utils/campusMessageReturn'
import { campusClock, randomCampusWish } from '@/site/utils/campusGreeting'

const { t, locale } = useI18n()
const fieldLabel = (value: string) => { const key = campusFieldKey(value); return key ? t(key) : value }
const page = defineProps<{ layout: LayoutPayload; props: Record<string, never> }>()
const status = ref<CampusStatus | null>(null)
const loading = ref(false), busy = ref(false), error = ref(''), unbindOpen = ref(false)
const tab = ref('overview'), week = ref(1), filter = ref('')
const data = ref<Partial<Record<CampusDatasetKey, CampusDataset>>>({})
const failures = ref<Partial<Record<CampusDatasetKey, string>>>({})
const now = ref(new Date())
const clock = computed(() => campusClock(now.value, locale.value))
const wish = ref(randomCampusWish())
const applyAdjustments = ref(true)
const exporting = ref(false), exportMessage = ref(''), exportError = ref('')
let exportController: AbortController | null = null
let exportRequest = 0
const exportURLs = new Set<string>()
function cancelExport() {
  exportRequest++
  exportController?.abort()
  exporting.value = false
  exportMessage.value = ''
  exportError.value = ''
  for (const url of exportURLs) URL.revokeObjectURL(url)
  exportURLs.clear()
}
async function exportCalendar() {
  if (exporting.value) return
  const request = ++exportRequest, epoch = generation
  exportController = new AbortController()
  exporting.value = true
  exportMessage.value = ''
  exportError.value = ''
  try {
    const result = await campusAPI.exportCalendar(exportController.signal, applyAdjustments.value)
    if (epoch !== generation || request !== exportRequest) return
    const url = URL.createObjectURL(new Blob([result.content], { type: 'text/calendar;charset=utf-8' }))
    exportURLs.add(url)
    const link = document.createElement('a')
    link.href = url
    link.download = result.filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => { URL.revokeObjectURL(url); exportURLs.delete(url) }, 1000)
    exportMessage.value = t('campus.exportSuccess', { count: result.eventCount })
  } catch (e) {
    if (epoch !== generation || request !== exportRequest) return
    exportError.value = e instanceof Error ? e.message : t('campus.exportFailed')
    if (e instanceof CampusError && e.code === 'campus.authorizationRequired' && status.value?.binding) status.value.binding.needsAuthorization = true
  } finally { if (request === exportRequest) exporting.value = false }
}
const dateLabel = computed(() => `${clock.value.date} ${clock.value.weekday}`)
const displayName = computed(() => data.value.profile?.metrics.find(m => m.label === '姓名')?.value || t('campus.student'))
let clockTimer: ReturnType<typeof setInterval> | undefined
let controller = new AbortController()
let generation = 0
const pendingKeys = new Set<CampusDatasetKey>()
const tabs = computed(() => [{ key: 'overview', label: t('campus.overview') }, { key: 'timetable', label: t('campus.timetable') }, { key: 'grades', label: t('campus.grades') }, { key: 'cet', label: t('campus.cet') }, { key: 'messages', label: t('campus.messages') }, { key: 'terms', label: t('campus.terms') }, { key: 'services', label: t('campus.services') }])
const keys: CampusDatasetKey[] = ['profile', 'calendar', 'today', 'timetable', 'grades', 'summary', 'cet', 'terms', 'messages', 'sports', 'health', 'arrangements']
const names = computed<Record<CampusDatasetKey, string>>(() => ({ today: t('campus.todayTimetable'), profile: t('campus.dataProfile'), calendar: t('campus.dataCalendar'), timetable: t('campus.dataTimetable'), grades: t('campus.dataGrades'), summary: t('campus.dataSummary'), cet: t('campus.dataCet'), terms: t('campus.dataTerms'), messages: t('campus.dataMessages'), sports: t('campus.dataSports'), health: t('campus.dataHealth'), arrangements: t('campus.dataArrangements') }))
const current = computed(() => data.value[tab.value as CampusDatasetKey])
const rows = computed(() => (current.value?.rows || []).filter(r => !filter.value || r.some(c => c.toLowerCase().includes(filter.value.toLowerCase()))))
const summary = computed(() => data.value.summary?.metrics || [])
const calendar = computed(() => data.value.calendar?.metrics || [])
const termWeeks = computed(() => Number(calendar.value.find(m => m.label === '学期周数')?.value) || 20)
const currentWeek = computed(() => {
  const value = Number(calendar.value.find(m => m.label === '教学周')?.value)
  return Number.isFinite(value) && value > 0 ? value : null
})
const courses = computed(() => data.value.timetable?.events || [])
const weekCourses = computed(() => courses.value.filter(c => !c.weeks.length || c.weeks.includes(week.value)))
const today = computed(() => clock.value.day)
const teachingDay = computed(() => data.value.today?.teachingDay)
const todayIsCurrent = computed(() => teachingDay.value?.date === clock.value.isoDate)
const todayCourses = computed(() => todayIsCurrent.value ? data.value.today?.events || [] : [])
const todayTitle = computed(() => failures.value.today || data.value.today?.status === 'unavailable' ? t('campus.timetableUnavailable') : !todayIsCurrent.value ? t('campus.todayLoading') : t('campus.noTodayCourses'))
const todayAdjustment = computed(() => {
  const day = teachingDay.value
  if (!todayIsCurrent.value || !day || day.kind === 'none') return ''
  return t(`campus.today${day.kind === 'makeup' ? 'Makeup' : day.kind === 'holiday' ? 'Holiday' : 'Moved'}`, { name: day.label, date: day.sourceDate })
})
const totalCredits = computed(() => Number(summary.value.find(m => m.label === '要求学分')?.value) || 0)
const completedCredits = computed(() => Number(summary.value.find(m => m.label === '已修学分')?.value) || 0)
const creditProgress = computed(() => totalCredits.value ? Math.max(0, Math.min(100, completedCredits.value / totalCredits.value * 100)) : 0)
const messages = computed(() => data.value.messages?.messages || [])
const recentMessages = computed(() => messages.value.slice(0, 5))
const filteredMessages = computed(() => messages.value.filter(m => `${m.title} ${m.publisher}`.toLowerCase().includes(filter.value.toLowerCase())))

const messageOpen = ref(false), messageLoading = ref(false), messageError = ref(''), messageNeedsAuthorization = ref(false)
const selectedMessage = ref<CampusMessageSummary | null>(null)
const messageDetail = ref<CampusMessageDetail | null>(null)
let messageController: AbortController | null = null
let messageRequest = 0
let messageTrigger: HTMLElement | null = null
let returningMessageId: string | null = null
function closeMessage() {
  messageRequest++
  messageController?.abort()
  messageOpen.value = false
  selectedMessage.value = null
  messageDetail.value = null
  messageLoading.value = false
  messageError.value = ''
  messageNeedsAuthorization.value = false
  const trigger = messageTrigger
  messageTrigger = null
  void nextTick(() => { if (trigger?.isConnected) trigger.focus() })
}
async function loadMessage() {
  const selected = selectedMessage.value
  if (!selected) return
  messageController?.abort()
  messageController = new AbortController()
  const request = ++messageRequest, epoch = generation
  messageLoading.value = true
  messageDetail.value = null
  messageError.value = ''
  messageNeedsAuthorization.value = false
  try {
    const value = await campusAPI.message(selected.id, messageController.signal)
    if (epoch === generation && request === messageRequest) messageDetail.value = value
  } catch (e) {
    if (epoch !== generation || request !== messageRequest) return
    messageError.value = e instanceof Error ? e.message : t('campus.messageLoadFailed')
    messageNeedsAuthorization.value = e instanceof CampusError && ['campus.messageAuthorizationRequired', 'campus.authorizationRequired'].includes(e.code)
  } finally { if (request === messageRequest) messageLoading.value = false }
}
function openMessage(message: CampusMessageSummary, event: Event) {
  if (!message.id) return
  messageTrigger = event.currentTarget as HTMLElement
  selectedMessage.value = message
  messageOpen.value = true
  void loadMessage()
}
function messageDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(locale.value, { timeZone: 'Asia/Shanghai', month: 'numeric', day: 'numeric' }).format(date)
}
function resetData() {
  cancelExport()
  returningMessageId = null
  generation++
  closeMessage()
  loading.value = false
  pendingKeys.clear()
  controller.abort()
  controller = new AbortController()
  data.value = {}
  failures.value = {}
  filter.value = ''
}
async function loadStatus() {
  const next = await campusAPI.status(controller.signal)
  if (next.binding?.revision !== status.value?.binding?.revision) resetData()
  status.value = next
}
function activeKeys(): CampusDatasetKey[] {
  if (tab.value === 'overview') return ['profile', 'calendar', 'messages', 'today']
  if (tab.value === 'grades') return ['summary', 'grades']
  if (tab.value === 'timetable') return ['calendar', 'timetable']
  if (tab.value === 'services') return keys
  return [tab.value as CampusDatasetKey]
}
async function loadData(targets = activeKeys(), force = false) {
  if (!status.value?.binding || status.value.binding.needsAuthorization) return
  const requested = targets.filter(key => !pendingKeys.has(key) && (force || !data.value[key] || (key === 'today' && !todayIsCurrent.value)))
  if (!requested.length) return
  const epoch = generation
  if (requested.includes('today')) delete data.value.today
  requested.forEach(key => pendingKeys.add(key))
  loading.value = true
  await Promise.all(requested.map(async key => {
    try {
      const value = await campusAPI.dataset(key, controller.signal)
      if (epoch !== generation) return
      data.value[key] = value
      delete failures.value[key]
      if (key === 'calendar' && currentWeek.value !== null) week.value = currentWeek.value
    } catch (e) {
      if (epoch !== generation || controller.signal.aborted) return
      failures.value[key] = e instanceof Error ? e.message : t('campus.loadFailed')
      if (e instanceof CampusError && e.code === 'campus.authorizationRequired' && status.value?.binding) status.value.binding.needsAuthorization = true
    } finally {
      if (epoch === generation) { pendingKeys.delete(key); loading.value = pendingKeys.size > 0 }
    }
  }))
}
async function refresh() {
  cancelExport()
  error.value = ''
  try { await loadStatus(); await loadData(activeKeys(), true) }
  catch (e) { error.value = e instanceof Error ? e.message : t('campus.loadFailed') }
}
async function authorize(mode: 'bind' | 'replace' | 'reauthorize') {
  cancelExport()
  busy.value = true
  error.value = ''
  try {
    const { url } = await campusAPI.start(mode)
    saveCampusMessageReturn(mode === 'reauthorize' && selectedMessage.value && status.value?.binding ? {
      userId: page.layout.viewer.id, revision: status.value.binding.revision,
      messageId: selectedMessage.value.id, tab: tab.value === 'messages' ? 'messages' : 'overview',
    } : null)
    window.location.assign(url)
  }
  catch (e) { error.value = (e as Error).message; busy.value = false }
}
async function confirm(resumeMessage = false) {
  cancelExport()
  const selected = resumeMessage ? selectedMessage.value : null
  const returning = returningMessageId
  const request = messageRequest
  const trigger = messageTrigger
  busy.value = true
  error.value = ''
  try {
    await campusAPI.confirm()
    const reopenId = returning || (selected && messageOpen.value && request === messageRequest ? selected.id : null)
    resetData()
    await refresh()
    // Recheck the refreshed list, rather than carrying private metadata across
    // a credential change. A close while confirmation was pending stays closed.
    const message = reopenId && messages.value.find(m => m.id === reopenId)
    if (message) {
      messageTrigger = trigger
      selectedMessage.value = message
      messageOpen.value = true
      void loadMessage()
    }
  } catch (e) {
    error.value = (e as Error).message
    if (messageOpen.value) { messageError.value = error.value; messageNeedsAuthorization.value = false }
    await loadStatus()
  } finally { busy.value = false }
}
async function unbind() {
  cancelExport()
  const binding = status.value?.binding
  if (!binding) return
  busy.value = true
  error.value = ''
  try { await campusAPI.unbind(binding.revision); resetData(); unbindOpen.value = false; await loadStatus() }
  catch (e) { error.value = (e as Error).message }
  finally { busy.value = false }
}
function setTab(key: string) { cancelExport(); tab.value = key; filter.value = ''; void loadData() }
function displayMetric(value: string) {
  return /^-?\d+\.\d+$/.test(value) ? value.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '') : value
}
function courseStyle(name: string) {
  return timetableCourseStyle(name)
}
function serviceState(key: CampusDatasetKey) {
  if (failures.value[key]) return t('campus.loadFailed')
  const state = data.value[key]?.status
  return state === 'ready' ? t('campus.connected') : state === 'empty' ? t('campus.empty') : state === 'unavailable' ? t('campus.unavailable') : loading.value ? t('campus.syncing') : t('campus.notSynced')
}
onMounted(async () => {
  watch(() => t('campus.title'), title => {
    setBaseDocumentTitle([title, page.layout.site?.name].filter(Boolean).join(' - '))
  }, { immediate: true })
  clockTimer = setInterval(() => {
    const previous = clock.value.date
    now.value = new Date()
    if (previous !== clock.value.date) {
      delete data.value.today
      delete data.value.calendar
      if (tab.value === 'overview') void loadData(['calendar', 'today'], true)
      else if (activeKeys().includes('calendar')) void loadData(['calendar'], true)
    }
  }, 60_000)
  const returning = takeCampusMessageReturn()
  const authorization = new URLSearchParams(location.search).get('authorization')
  if (authorization === 'failed') error.value = t('campus.authorizationFailed')
  if (page.layout.viewer.isAuthenticated) {
    try {
      await loadStatus()
      if (authorization === 'ready' && returning && returning.userId === page.layout.viewer.id &&
        returning.revision === status.value?.binding?.revision && status.value?.candidate?.mode === 'reauthorize') {
        returningMessageId = returning.messageId
        tab.value = returning.tab
      }
      await loadData()
    }
    catch (e) { error.value = (e as Error).message }
  }
})
onBeforeUnmount(() => { clearInterval(clockTimer); resetData() })
</script>

<template>
  <main class="campus-page min-w-0 pb-8">
    <PageHeader :title="t('campus.title')" :description="t('campus.description')" compact>
      <template #actions>
        <span class="mr-1 hidden text-xs text-base-content/55 sm:inline">{{ dateLabel }}</span>
        <button
          v-if="status?.binding"
          class="gf-button gf-button-sm gf-button-secondary text-xs"
          :disabled="loading || busy"
          :aria-label="t('campus.refreshData')"
          @click="refresh"
        >
          <RefreshCw class="h-3.5 w-3.5" :class="{ spinning: loading }" />
          {{ loading ? t('campus.syncing') : t('campus.refresh') }}
        </button>
      </template>
    </PageHeader>

    <div v-if="error" class="gf-status-message gf-status-message-error mx-4 mb-4 sm:mx-0" role="alert">{{ error }}</div>

    <section v-if="!page.layout.viewer.isAuthenticated" class="gf-card overflow-hidden py-6">
      <EmptyState :icon="GraduationCap" :title="t('campus.connectTitle')" :description="t('campus.connectDescription')">
        <a class="gf-button gf-button-md gf-button-primary" href="/login?redirect=%2Fcampus">{{ t('campus.login') }} <ArrowUpRight class="h-4 w-4" /></a>
      </EmptyState>
    </section>
    <section v-else-if="!status" class="gf-card overflow-hidden" aria-live="polite">
      <EmptyState :loading="!error" :icon="ShieldCheck" :title="error ? t('campus.statusUnavailable') : t('campus.statusLoading')">
        <button v-if="error" class="gf-button gf-button-sm gf-button-secondary" @click="refresh">{{ t('campus.retry') }}</button>
      </EmptyState>
    </section>
    <template v-else>
      <section class="gf-card mb-4 flex flex-wrap items-center gap-3 p-4">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-box bg-info/10 text-primary">
          <ShieldCheck class="h-5 w-5" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="text-sm font-semibold">{{ t('campus.identity') }}</h2>
            <span v-if="status.binding" class="gf-badge gf-badge-info">{{ t('campus.verified') }}</span>
          </div>
          <p class="mt-1 text-xs text-base-content/55">
            <span v-if="status.binding" class="tabular-nums">{{ status.binding.maskedId }} · {{ t('campus.privateData') }}</span>
            <span v-else>{{ t('campus.bindHint') }}</span>
          </p>
        </div>
        <div class="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
          <template v-if="status.binding">
            <button v-if="status.binding.needsAuthorization" class="gf-button gf-button-sm gf-button-primary text-xs" :disabled="busy" @click="authorize('reauthorize')">{{ t('campus.reauthorize') }}</button>
            <button class="gf-button gf-button-sm gf-button-secondary text-xs" :disabled="busy" @click="authorize('replace')">{{ t('campus.replace') }}</button>
            <button class="gf-button gf-button-sm gf-button-muted text-xs" :disabled="busy" @click="unbindOpen = true">{{ t('campus.unbind') }}</button>
          </template>
          <button v-else class="gf-button gf-button-sm gf-button-primary text-xs" :disabled="busy || !status.enabled" @click="authorize('bind')">
            <Loader2 v-if="busy" class="h-3.5 w-3.5 spinning" />
            <Link2 v-else class="h-3.5 w-3.5" />
            {{ status.enabled ? t('campus.connect') : t('campus.disabled') }}
          </button>
        </div>
      </section>

      <p v-if="status.binding?.needsAuthorization" class="gf-status-message gf-status-message-info mx-4 mb-4 sm:mx-0">
        {{ t('campus.authorizationHint') }}
      </p>
      <section v-if="status.candidate" class="gf-card mb-4 flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <h2 class="flex items-center gap-2 text-sm font-semibold"><ShieldCheck class="h-4 w-4 shrink-0 text-primary" />{{ status.candidate.mode === 'reauthorize' ? t('campus.pendingUpdate') : t('campus.confirmConnection', { id: status.candidate.maskedId }) }}</h2>
          <p class="mt-1 text-xs leading-5 text-base-content/55">
            {{ status.candidate.mode === 'replace' ? t('campus.candidateReplace') : status.candidate.mode === 'reauthorize' ? t('campus.candidateUpdate') : t('campus.candidateBind') }}
          </p>
        </div>
        <button class="gf-button gf-button-sm gf-button-primary self-end whitespace-nowrap text-xs sm:self-auto" :disabled="busy" @click="confirm()">
          <Loader2 v-if="busy" class="h-3.5 w-3.5 spinning" /><Check v-else class="h-3.5 w-3.5" />
          {{ status.candidate.mode === 'replace' ? t('campus.confirmReplace') : status.candidate.mode === 'reauthorize' ? t('campus.confirmUpdate') : t('campus.confirmBind') }}
        </button>
      </section>
      <div v-if="unbindOpen" class="gf-card mb-4 flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between" role="alertdialog" :aria-label="t('campus.confirmUnbind')">
        <div>
          <h2 class="text-sm font-semibold">{{ t('campus.unbindTitle') }}</h2>
          <p class="mt-1 text-xs leading-5 text-base-content/55">{{ t('campus.unbindDescription') }}</p>
        </div>
        <div class="flex shrink-0 justify-end gap-2">
          <button class="gf-button gf-button-sm gf-button-secondary text-xs" :disabled="busy" @click="unbindOpen = false">{{ t('campus.cancel') }}</button>
          <button class="gf-button gf-button-sm gf-button-danger text-xs" :disabled="busy" @click="unbind"><Unplug class="h-3.5 w-3.5" />{{ t('campus.confirmUnbind') }}</button>
        </div>
      </div>

      <section v-if="!status.binding" class="gf-card overflow-hidden">
        <EmptyState :icon="GraduationCap" :title="t('campus.welcomeTitle')" :description="t('campus.welcomeDescription')" />
        <div class="grid grid-cols-3 divide-x divide-line border-y border-line bg-base-200/50 py-4 text-center text-xs text-base-content/75">
          <div><CalendarDays class="mx-auto mb-2 h-5 w-5 text-primary" />{{ t('campus.weeklyTimetable') }}</div>
          <div><ChartNoAxesCombined class="mx-auto mb-2 h-5 w-5 text-primary" />{{ t('campus.grades') }}</div>
          <div><ShieldCheck class="mx-auto mb-2 h-5 w-5 text-primary" />{{ t('campus.privateConnection') }}</div>
        </div>
        <p class="px-4 py-3 text-xs leading-5 text-base-content/55">{{ t('campus.passwordHint') }}</p>
      </section>
      <template v-else>
        <nav class="gf-card mb-4 flex gap-1 overflow-x-auto bg-base-200/60 p-2" :aria-label="t('campus.content')">
          <button
            v-for="item in tabs"
            :key="item.key"
            class="gf-tab"
            :class="tab === item.key ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
            :aria-current="tab === item.key ? 'page' : undefined"
            @click="setTab(item.key)"
          >{{ item.label }}</button>
        </nav>

        <div v-if="tab === 'overview'" class="space-y-4">
          <section class="gf-card p-4 sm:p-5">
            <div class="flex flex-wrap items-center gap-2 text-xs text-base-content/55">
              <CalendarDays class="h-4 w-4 text-primary" />
              <span v-if="currentWeek !== null" class="font-medium text-primary">{{ t('campus.week', { week: currentWeek }) }}</span>
              <span v-else>{{ failures.calendar ? t('campus.calendarUnavailable') : t('campus.weekPending') }}</span>
              <span aria-hidden="true">·</span><span>{{ clock.weekday }}</span>
              <span class="ml-auto">{{ clock.date }}</span>
            </div>
            <h2 class="mt-4 text-xl font-semibold sm:text-2xl">{{ t(clock.greeting, { name: displayName }) }}</h2>
            <div class="mt-3 flex items-start gap-2">
              <p class="min-w-0 flex-1 text-sm leading-6 text-base-content/65">{{ t(wish) }}</p>
              <button class="gf-icon-button h-7 w-7 shrink-0" :aria-label="t('campus.anotherWish')" :title="t('campus.shuffle')" @click="wish = randomCampusWish(wish)"><Shuffle class="h-3.5 w-3.5" /></button>
            </div>
          </section>
          <section class="gf-card overflow-hidden">
            <SectionHeader :title="t('campus.messages')" :icon="Bell">
              <template #actions><button class="gf-button gf-button-xs gf-button-ghost text-xs" @click="setTab('messages')">{{ t('campus.viewAll') }} <ArrowUpRight class="h-3.5 w-3.5" /></button></template>
            </SectionHeader>
            <div v-if="recentMessages.length" class="divide-y divide-line">
              <button v-for="message in recentMessages" :key="message.id" class="flex w-full items-start gap-3 px-4 py-3 text-left hover:bg-base-200/60 disabled:cursor-default" :disabled="!message.id" @click="openMessage(message, $event)">
                <div class="min-w-0 flex-1"><h3 class="line-clamp-2 text-sm font-medium leading-6">{{ message.title }}</h3><p class="mt-1 text-xs text-base-content/55">{{ message.publisher }}</p></div>
                <time class="mt-1 shrink-0 text-xs text-base-content/45" :datetime="message.publishedAt">{{ messageDate(message.publishedAt) }}</time>
                <ChevronRight class="mt-1 h-4 w-4 shrink-0 text-base-content/40" />
              </button>
            </div>
            <EmptyState v-else :icon="Bell" :loading="!data.messages && !failures.messages" :title="failures.messages || data.messages?.status === 'unavailable' ? t('campus.messagesUnavailable') : data.messages ? t('campus.noMessages') : t('campus.messagesLoading')" />
          </section>
          <section class="gf-card overflow-hidden">
            <SectionHeader :title="t('campus.todayTimetable')" :icon="CalendarDays">
              <template #actions><button class="gf-button gf-button-xs gf-button-ghost text-xs" @click="setTab('timetable')">{{ t('campus.viewWeek') }} <ArrowUpRight class="h-3.5 w-3.5" /></button></template>
            </SectionHeader>
            <p v-if="todayAdjustment" class="border-b border-line px-4 py-3 text-sm text-base-content/65" role="status">{{ todayAdjustment }}</p>
            <EmptyState v-if="!todayCourses.length" :icon="BookOpen" :title="todayTitle" :description="failures.today || t('campus.todayHint')" />
            <div v-else class="space-y-2 p-4">
              <article v-for="(course,index) in todayCourses" :key="index" class="flex items-start gap-3 rounded-xl border p-3" :style="courseStyle(course.name)">
                <span class="w-16 shrink-0 rounded-field bg-base-100/60 py-2 text-center text-xs font-semibold tabular-nums text-[var(--card-title)]">{{ t('campus.periods', { start: course.start, end: course.end }) }}</span>
                <div class="min-w-0"><h3 class="text-sm font-semibold">{{ course.name }}</h3><p class="mt-1 text-xs leading-5 text-base-content/55">{{ course.room || t('campus.roomPending') }}<template v-if="course.teacher"> · {{ course.teacher }}</template></p></div>
              </article>
            </div>
          </section>
        </div>

        <section v-else-if="tab === 'timetable'" class="gf-card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
            <h2 class="text-sm font-semibold">{{ t('campus.weeklyTimetable') }}</h2>
            <div class="flex items-center gap-1 text-xs">
              <button class="gf-icon-button h-8 w-8 disabled:opacity-40" :aria-label="t('campus.previousWeek')" :disabled="week <= 1" @click="week--"><ChevronLeft class="h-4 w-4" /></button>
              <span class="min-w-16 text-center font-medium tabular-nums">{{ t('campus.week', { week }) }}</span>
              <button class="gf-icon-button h-8 w-8 disabled:opacity-40" :aria-label="t('campus.nextWeek')" :disabled="week >= termWeeks" @click="week++"><ChevronRight class="h-4 w-4" /></button>
              <button class="gf-button gf-button-xs gf-button-secondary ml-2 text-xs" :disabled="currentWeek === null" @click="week = currentWeek ?? 1">{{ t('campus.thisWeek') }}</button>
            </div>
          </div>
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
            <p id="campus-export-hint" class="min-w-0 flex-1 text-xs leading-5 text-base-content/55">{{ t('campus.exportHint') }}</p>
            <label class="flex shrink-0 cursor-pointer items-center gap-2 text-xs">
              <input v-model="applyAdjustments" type="checkbox" role="switch" class="peer sr-only" :disabled="exporting" />
              <span aria-hidden="true" class="relative h-5 w-9 rounded-full bg-base-content/20 transition-colors after:absolute after:left-0.5 after:top-0.5 after:h-4 after:w-4 after:rounded-full after:bg-base-100 after:shadow-sm after:transition-transform peer-checked:bg-primary peer-checked:after:translate-x-4 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary peer-disabled:opacity-50"></span>
              {{ t('campus.applyAdjustments') }}
            </label>
            <button class="gf-button gf-button-sm gf-button-secondary shrink-0 text-xs" :disabled="exporting || busy || !courses.length || status.binding.needsAuthorization" aria-describedby="campus-export-hint" @click="exportCalendar">
              <Loader2 v-if="exporting" class="h-3.5 w-3.5 spinning" /><Download v-else class="h-3.5 w-3.5" />{{ exporting ? t('campus.exporting') : t('campus.exportCalendar') }}
            </button>
          </div>
          <CampusCalendarRules />
          <p v-if="exportError" role="alert" class="gf-status-message gf-status-message-error m-4">{{ exportError }}</p>
          <p v-else-if="exportMessage" role="status" class="gf-status-message gf-status-message-info m-4">{{ exportMessage }}</p>
          <p v-if="failures.timetable" class="gf-status-message gf-status-message-error m-4">{{ failures.timetable }}</p>
          <EmptyState v-if="!data.timetable && !failures.timetable" loading :title="t('campus.timetableLoading')" />
          <EmptyState v-else-if="failures.timetable || data.timetable?.status === 'unavailable'" :icon="BookOpen" :title="t('campus.timetableUnavailable')" :description="t('campus.schoolRetry')" />
          <template v-else>
            <p v-if="!weekCourses.length" class="px-4 py-3 text-sm text-base-content/55">{{ t('campus.noWeekCourses') }}</p>
            <div class="p-2 sm:p-4"><CampusTimetable :courses="courses" :week="week" :today="today" /></div>
          </template>
          <p class="border-t border-line px-4 py-3 text-xs leading-5 text-base-content/55">{{ t('campus.weekSummary', { week, count: weekCourses.length }) }}</p>
        </section>

        <section v-else-if="tab === 'services'" class="gf-card overflow-hidden">
          <SectionHeader :title="t('campus.dataConnection')" :description="t('campus.platform')" :icon="Link2" />
          <div class="divide-y divide-line px-4">
            <div v-for="key in keys" :key="key" class="flex items-center justify-between gap-3 py-3.5">
              <h3 class="text-sm font-medium">{{ names[key] }}</h3>
              <span class="gf-badge shrink-0" :class="data[key]?.status === 'ready' ? 'gf-badge-info' : 'gf-badge-muted'">{{ serviceState(key) }}</span>
            </div>
          </div>
          <p class="border-t border-line bg-base-200/50 px-4 py-3 text-xs leading-6 text-base-content/55">{{ t('campus.serviceHint') }}</p>
        </section>

        <section v-else-if="tab === 'messages'" class="gf-card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
            <h2 class="text-sm font-semibold">{{ t('campus.messages') }}</h2>
            <label class="w-full sm:w-64"><span class="sr-only">{{ t('campus.searchMessages') }}</span><input v-model="filter" class="gf-input h-9 text-xs" type="search" :placeholder="t('campus.searchMessagesPlaceholder')" /></label>
          </div>
          <div v-if="filteredMessages.length" class="divide-y divide-line">
            <button v-for="message in filteredMessages" :key="message.id" class="flex w-full items-start gap-3 px-4 py-3 text-left hover:bg-base-200/60" :disabled="!message.id" @click="openMessage(message, $event)">
              <div class="min-w-0 flex-1"><h3 class="text-sm font-medium leading-6">{{ message.title }}</h3><p class="mt-1 text-xs text-base-content/55">{{ message.publisher }}</p></div>
              <time class="mt-1 shrink-0 text-xs text-base-content/45" :datetime="message.publishedAt">{{ messageDate(message.publishedAt) }}</time><ChevronRight class="mt-1 h-4 w-4 shrink-0 text-base-content/40" />
            </button>
          </div>
          <EmptyState v-else :icon="Bell" :loading="!data.messages && !failures.messages" :title="failures.messages || data.messages?.status === 'unavailable' ? t('campus.messagesUnavailable') : filter ? t('campus.noMatchingMessages') : t('campus.noMessages')" />
        </section>

        <section v-else class="gf-card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
            <h2 class="text-sm font-semibold">{{ names[tab as CampusDatasetKey] }}</h2>
            <label class="w-full sm:w-64"><span class="sr-only">{{ t('campus.searchRecords') }}</span><input v-model="filter" class="gf-input h-9 text-xs" type="search" :placeholder="t('campus.searchRecordsPlaceholder')" /></label>
          </div>
          <div v-if="tab === 'grades' && summary.length" class="border-b border-line">
            <dl class="grid grid-cols-2 gap-px bg-line sm:grid-cols-4">
              <div v-for="metric in summary" :key="metric.label" class="bg-base-100 p-4"><dt class="text-xs text-base-content/55">{{ fieldLabel(metric.label) }}</dt><dd class="mt-2 text-2xl font-semibold tabular-nums">{{ displayMetric(metric.value) }}<span class="ml-1 text-xs font-normal text-base-content/55">{{ fieldLabel(metric.unit) }}</span></dd></div>
            </dl>
            <div v-if="totalCredits" class="border-t border-line px-4 py-3"><div class="mb-2 flex justify-between text-xs text-base-content/55"><span>{{ t('campus.creditProgress') }}</span><span>{{ t('campus.creditsProgress', { completed: completedCredits, total: totalCredits }) }}</span></div><div class="h-2 overflow-hidden rounded-full bg-base-300"><div class="h-full rounded-full bg-primary/75" :style="{ width: `${creditProgress}%` }"></div></div></div>
          </div>
          <p v-if="tab === 'grades' && failures.summary" class="gf-status-message gf-status-message-error m-4">{{ failures.summary }}</p>
          <dl v-if="current?.metrics.length" class="flex flex-wrap gap-x-8 gap-y-4 border-b border-line p-4">
            <div v-for="metric in current.metrics" :key="metric.label">
              <dt class="text-xs text-base-content/55">{{ fieldLabel(metric.label) }}</dt>
              <dd class="mt-1 text-xl font-semibold tabular-nums">{{ displayMetric(metric.value) }}<span class="ml-1 text-xs font-normal text-base-content/55">{{ fieldLabel(metric.unit) }}</span></dd>
            </div>
          </dl>
          <div v-if="current?.series.length" class="border-b border-line bg-base-200/40 p-4 sm:p-5">
            <h3 class="mb-4 text-sm font-semibold">{{ tab === 'grades' ? t('campus.gradeChart') : t('campus.examChart') }}</h3>
            <div v-for="(point,index) in current.series" :key="index" class="campus-chart-row mb-3">
              <span class="text-xs leading-5 text-base-content/75">{{ point.label }}</span>
              <div class="h-2 overflow-hidden rounded-full bg-base-300" aria-hidden="true"><div class="h-full rounded-full bg-primary/75" :style="{ width: `${Math.max(0, Math.min(100, point.value / (tab === 'cet' ? 710 : 5) * 100))}%` }"></div></div>
              <strong class="text-right text-xs font-semibold tabular-nums">{{ point.value }}</strong>
            </div>
            <p class="mt-4 text-xs text-base-content/55">{{ tab === 'grades' ? t('campus.gradeChartHint') : t('campus.examChartHint') }}</p>
          </div>
          <EmptyState v-if="failures[tab as CampusDatasetKey] || current?.status === 'unavailable'" :icon="BookOpen" :title="t('campus.recordsUnavailable')" :description="failures[tab as CampusDatasetKey] || t('campus.errorUpstream')" />
          <EmptyState v-else-if="!current" loading :title="t('campus.recordsLoading')" />
          <EmptyState v-else-if="!rows.length" :icon="BookOpen" :title="filter ? t('campus.noMatchingRecords') : t('campus.noRecords')" />
          <div v-else class="overflow-x-auto">
            <table class="campus-records w-full text-left text-sm">
              <thead class="bg-base-200/60 text-xs text-base-content/55"><tr><th v-for="column in current.columns" :key="column" scope="col">{{ fieldLabel(column) }}</th></tr></thead>
              <tbody class="divide-y divide-line"><tr v-for="(row,index) in rows" :key="index" class="hover:bg-base-200/60"><td v-for="(cell,i) in row" :key="i">{{ cell || '—' }}</td></tr></tbody>
            </table>
          </div>
          <p v-if="current" class="border-t border-line px-4 py-3 text-xs text-base-content/55">{{ t('campus.recordsSynced', { count: rows.length, time: new Date(current.updatedAt).toLocaleTimeString(locale, { timeZone: 'Asia/Shanghai' }) }) }}</p>
        </section>
      </template>
    </template>

    <CampusMessageDialog :open="messageOpen" :summary="selectedMessage" :detail="messageDetail" :loading="messageLoading" :error="messageError" :needs-authorization="messageNeedsAuthorization" :pending-authorization="status?.candidate?.mode === 'reauthorize'" :busy="busy" @close="closeMessage" @retry="loadMessage" @confirm-authorization="confirm(true)" @authorize="authorize('reauthorize')" />

    <footer class="mt-4 flex flex-wrap items-center gap-x-2 gap-y-2 px-4 py-3 text-xs text-base-content/45 sm:px-0">
      <ShieldCheck class="h-3.5 w-3.5 shrink-0" /><span>{{ t('campus.privacyFooter') }}</span>
      <a class="ml-auto inline-flex items-center gap-1 hover:text-primary" href="https://github.com/oierxjn/OneTJ" target="_blank" rel="noopener noreferrer">{{ t('campus.thanks') }} <ArrowUpRight class="h-3 w-3" /></a>
    </footer>
  </main>
</template>

<style scoped>
/* Shared forum components own surfaces, controls and typography. Only data
   visualization geometry is local to the campus page. Colors follow tokens. */
.campus-chart-row {
  display: grid;
  grid-template-columns: minmax(7rem, 12rem) minmax(2rem, 1fr) 2.5rem;
  align-items: center;
  gap: 0.75rem;
}

.campus-records th,
.campus-records td {
  padding: 0.75rem 1rem;
  line-height: 1.6;
}

.campus-records th {
  font-weight: 600;
  white-space: nowrap;
}

.campus-records td {
  min-width: 6rem;
  vertical-align: top;
}

.campus-records td:first-child {
  min-width: 10rem;
}

.campus-records td:nth-child(2) {
  min-width: 11rem;
}

.campus-page :is(button, a):focus-visible {
  outline: 2px solid var(--gf-color-primary);
  outline-offset: 2px;
}

.spinning {
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 639.98px) {
  .campus-chart-row {
    grid-template-columns: minmax(0, 1fr) 2.5rem;
    gap: 0.375rem 0.75rem;
  }

  .campus-chart-row > span {
    grid-column: 1 / -1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinning { animation: none; }
}
</style>

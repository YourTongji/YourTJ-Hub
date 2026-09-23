<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CampusDataset, CampusEvent } from '@gooseforum/client'
import { campusAPI } from '@/runtime/campus-api'
import { officialLocationTarget, parseOfficialLocation } from '@/site/campus-map/official-location'

const props = defineProps<{ authenticated: boolean }>()
const emit = defineEmits<{ select: [target: ReturnType<typeof officialLocationTarget> | null] }>()
const { t } = useI18n()
const state = ref<'loading' | 'login' | 'unbound' | 'reauthorize' | 'ready' | 'error'>('loading')
const today = ref<CampusDataset | null>(null)
const calendar = ref<CampusDataset | null>(null)
const timetable = ref<CampusDataset | null>(null)
const scope = ref<'today' | 'week'>('today')
const week = ref(1)
const search = ref('')
const selected = ref<CampusEvent | null>(null)
let controller: AbortController | undefined
let requestVersion = 0
let clockTimer: number | undefined
let activeBindingRevision: string | undefined
let attemptedTodayRefresh = ''

const courses = computed(() => {
  const source = scope.value === 'today' ? todayCourses.value : timetable.value?.events ?? []
  return source.filter(course =>
    (scope.value === 'today' || (course.weeks.length > 0 && validWeek.value && course.weeks.includes(week.value))) &&
    (!search.value.trim() || `${course.name} ${course.room} ${course.campus}`.toLowerCase().includes(search.value.trim().toLowerCase())),
  )
})
const unavailable = computed(() => (scope.value === 'today' ? today.value : timetable.value)?.status === 'unavailable')
const termWeeks = computed(() => Number(calendar.value?.metrics.find(metric => metric.label === '学期周数')?.value) || 0)
const validWeek = computed(() => Number.isInteger(week.value) && week.value > 0 && week.value <= termWeeks.value)
const todayIsCurrent = computed(() => today.value?.teachingDay?.date === shanghaiDate())
const todayCourses = computed(() => todayIsCurrent.value ? today.value?.events ?? [] : [])
const hasUnknownWeekCourses = computed(() => scope.value === 'week' && (timetable.value?.events ?? []).some(course => course.weeks.length === 0))

function shanghaiDate() {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
  }).formatToParts(new Date())
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(item => item.type === type)?.value ?? ''
  return `${part('year')}-${part('month')}-${part('day')}`
}

function displayLocation(campus: string, room: string) {
  const parsed = parseOfficialLocation(room)
  return [campus, parsed?.building, parsed?.room || (!parsed ? room : '') || t('campus.roomPending')]
    .filter(Boolean)
    .join(' · ')
}

function chooseCourse(course: CampusEvent) {
  selected.value = selected.value === course ? null : course
  emit('select', selected.value ? officialLocationTarget(course.campus, course.room) ?? null : null)
}

function clearData() {
  activeBindingRevision = undefined
  today.value = null
  calendar.value = null
  timetable.value = null
  if (selected.value) emit('select', null)
  selected.value = null
  search.value = ''
}

function changeScope(nextScope: 'today' | 'week') {
  scope.value = nextScope
  selected.value = null
  emit('select', null)
}

function suspend(stateWhenCleared: 'loading' | 'login') {
  requestVersion++
  controller?.abort()
  clearData()
  state.value = stateWhenCleared
}

function clearTo(nextState: 'unbound' | 'reauthorize' | 'error') {
  requestVersion++
  controller?.abort()
  clearData()
  state.value = nextState
}

async function load() {
  controller?.abort()
  const current = new AbortController()
  controller = current
  const version = ++requestVersion
  const isCurrent = () => requestVersion === version && !current.signal.aborted
  clearData()
  if (!props.authenticated) {
    state.value = 'login'
    return
  }
  state.value = 'loading'
  try {
    const status = await campusAPI.status(current.signal)
    if (!isCurrent()) return
    if (!status.binding) {
      state.value = 'unbound'
      return
    }
    if (status.binding.needsAuthorization) {
      state.value = 'reauthorize'
      return
    }
    const bindingRevision = status.binding.revision
    const [todayData, calendarData, timetableData] = await Promise.all([
      campusAPI.dataset('today', current.signal),
      campusAPI.dataset('calendar', current.signal),
      campusAPI.dataset('timetable', current.signal),
    ])
    if (!isCurrent()) return
    const confirmedStatus = await campusAPI.status(current.signal)
    if (!isCurrent()) return
    if (!confirmedStatus.binding) {
      state.value = 'unbound'
      return
    }
    if (confirmedStatus.binding.needsAuthorization) {
      state.value = 'reauthorize'
      return
    }
    if (confirmedStatus.binding.revision !== bindingRevision) throw new Error('campus binding changed')
    activeBindingRevision = bindingRevision
    today.value = todayData
    calendar.value = calendarData
    timetable.value = timetableData
    const currentWeek = Number(calendarData.metrics.find(metric => metric.label === '教学周')?.value)
    if (Number.isInteger(currentWeek) && currentWeek > 0) week.value = currentWeek
    state.value = 'ready'
  } catch {
    if (!isCurrent()) return
    clearData()
    state.value = 'error'
  }
}

function recheckOnReturn() {
  if (document.visibilityState !== 'hidden') void load()
}
function handlePageHide() {
  suspend('loading')
}
function handleVisibilityChange() {
  if (document.visibilityState === 'hidden') suspend('loading')
  else void load()
}
function handleSessionCleared() {
  suspend('login')
}
async function checkBindingAndDate() {
  if (state.value !== 'ready' || document.visibilityState === 'hidden') return
  const expectedRevision = activeBindingRevision
  if (!expectedRevision) return
  const version = requestVersion
  const monitor = new AbortController()
  try {
    const status = await campusAPI.status(monitor.signal)
    if (version !== requestVersion || state.value !== 'ready') return
    if (!status.binding) {
      clearTo('unbound')
      return
    }
    if (status.binding.needsAuthorization) {
      clearTo('reauthorize')
      return
    }
    if (status.binding.revision !== expectedRevision) {
      clearTo('error')
      return
    }
    const date = shanghaiDate()
    if (today.value?.teachingDay?.date !== date && attemptedTodayRefresh !== date) {
      attemptedTodayRefresh = date
      void load()
    }
  } catch {
    if (version === requestVersion && state.value === 'ready') clearTo('error')
  }
}
onMounted(() => {
  void load()
  window.addEventListener('focus', recheckOnReturn)
  window.addEventListener('pagehide', handlePageHide)
  window.addEventListener('pageshow', recheckOnReturn)
  window.addEventListener('goose:session-cleared', handleSessionCleared)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  clockTimer = window.setInterval(() => { void checkBindingAndDate() }, 60_000)
})
onBeforeUnmount(() => {
  suspend('loading')
  if (clockTimer !== undefined) window.clearInterval(clockTimer)
  window.removeEventListener('focus', recheckOnReturn)
  window.removeEventListener('pagehide', handlePageHide)
  window.removeEventListener('pageshow', recheckOnReturn)
  window.removeEventListener('goose:session-cleared', handleSessionCleared)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<template>
  <section class="atlas-mine" aria-labelledby="atlas-mine-title">
    <p class="atlas-mine__eyebrow">{{ t('campusMap.mine.source') }}</p>
    <h2 id="atlas-mine-title">{{ t('campusMap.mine.title') }}</h2>

    <div v-if="state === 'loading'" class="atlas-mine__status" role="status">
      {{ t('campusMap.mine.loading') }}
    </div>
    <div v-else-if="state === 'login'" class="atlas-mine__status">
      <p>{{ t('campusMap.mine.loginRequired') }}</p>
      <a class="atlas-mine__action" href="/login?redirect=%2Fmap%3Fmine%3D1">{{ t('campus.login') }}</a>
    </div>
    <div v-else-if="state === 'unbound'" class="atlas-mine__status">
      <p>{{ t('campusMap.mine.bindRequired') }}</p>
      <a class="atlas-mine__action" href="/campus">{{ t('campusMap.mine.openCampus') }}</a>
    </div>
    <div v-else-if="state === 'reauthorize'" class="atlas-mine__status">
      <p>{{ t('campusMap.mine.reauthorizeRequired') }}</p>
      <a class="atlas-mine__action" href="/campus">{{ t('campusMap.mine.openCampus') }}</a>
    </div>
    <div v-else-if="state === 'error'" class="atlas-mine__status" role="alert">
      <p>{{ t('campusMap.mine.unavailable') }}</p>
      <button class="atlas-mine__action" type="button" @click="load">{{ t('campusMap.retry') }}</button>
    </div>
    <template v-else>
      <p class="atlas-mine__note">{{ t('campusMap.mine.limits') }}</p>
      <div class="atlas-mine__tabs" role="group" :aria-label="t('campusMap.mine.scheduleRange')">
        <button type="button" :aria-pressed="scope === 'today'" @click="changeScope('today')">{{ t('campus.todayTimetable') }}</button>
        <button type="button" :aria-pressed="scope === 'week'" @click="changeScope('week')">{{ t('campus.timetable') }}</button>
      </div>
      <label v-if="scope === 'week'" class="atlas-mine__week">
        <span>{{ t('campusMap.mine.week') }}</span>
        <input v-model.number="week" type="number" min="1" :max="termWeeks || undefined" inputmode="numeric" />
      </label>
      <label class="atlas-mine__search">
        <span class="sr-only">{{ t('campusMap.mine.search') }}</span>
        <input v-model="search" type="search" :placeholder="t('campusMap.mine.search')" />
      </label>
      <p v-if="hasUnknownWeekCourses" class="atlas-mine__note">{{ t('campusMap.mine.unknownWeeks') }}</p>
      <p v-if="unavailable" class="atlas-mine__status" role="status">{{ t('campusMap.mine.unavailable') }}</p>
      <p v-else-if="scope === 'week' && !validWeek" class="atlas-mine__status" role="status">{{ t('campusMap.mine.invalidWeek', { count: termWeeks || '—' }) }}</p>
      <p v-else-if="scope === 'today' && !todayIsCurrent" class="atlas-mine__status" role="status">{{ t('campus.todayLoading') }}</p>
      <p v-else-if="!courses.length" class="atlas-mine__status" role="status">
        {{ search ? t('campusMap.mine.noMatches') : t(scope === 'today' ? 'campus.noTodayCourses' : 'campusMap.mine.noCourses') }}
      </p>
      <div v-else class="atlas-mine__list" aria-live="polite">
        <button
          v-for="(course, index) in courses"
          :key="`${course.name}-${course.day}-${course.start}-${course.room}-${index}`"
          type="button"
          class="atlas-mine__course"
          :aria-pressed="selected === course"
          @click="chooseCourse(course)"
        >
          <span class="atlas-mine__period">{{ t('campus.periods', { start: course.start, end: course.end }) }}</span>
          <strong>{{ course.name }}</strong>
          <small>{{ displayLocation(course.campus, course.room) }}</small>
        </button>
      </div>
      <div v-if="selected" class="atlas-mine__selected" role="status">
        <strong>{{ selected.name }}</strong>
        <p>{{ displayLocation(selected.campus, selected.room) }}</p>
        <p v-if="!officialLocationTarget(selected.campus, selected.room)" class="atlas-mine__unverified">{{ t('campusMap.mine.locationUnverified') }}</p>
      </div>
    </template>
  </section>
</template>

<style scoped>
.atlas-mine { display: flex; min-height: 100%; flex-direction: column; gap: 12px; padding: 20px; color: #24342f; }
.atlas-mine h2 { margin: 0; font-size: 21px; font-weight: 650; }
.atlas-mine__eyebrow { margin: 0; color: #5d7469; font-size: 11px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
.atlas-mine__note, .atlas-mine__status { margin: 0; color: #62746d; font-size: 13px; line-height: 1.55; }
.atlas-mine__status { display: grid; gap: 10px; }
.atlas-mine__action { width: fit-content; border: 0; border-radius: 10px; background: #e7f1eb; padding: 9px 12px; color: #24543d; font: inherit; font-size: 13px; text-decoration: none; cursor: pointer; }
.atlas-mine__tabs { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; border-radius: 12px; background: #eef2ef; padding: 4px; }
.atlas-mine__tabs button { border: 0; border-radius: 9px; background: transparent; padding: 8px 6px; color: #607069; font: inherit; font-size: 12px; cursor: pointer; }
.atlas-mine__tabs button[aria-pressed="true"] { background: white; color: #264b39; box-shadow: 0 1px 4px #20342a18; }
.atlas-mine__search input, .atlas-mine__week input { width: 100%; border: 1px solid #dce4de; border-radius: 10px; background: white; padding: 10px 12px; color: inherit; font: inherit; font-size: 13px; }
.atlas-mine__week { display: flex; align-items: center; gap: 12px; color: #62746d; font-size: 12px; }
.atlas-mine__week input { max-width: 92px; }
.atlas-mine__list { display: grid; gap: 8px; overflow: auto; }
.atlas-mine__course { display: grid; gap: 4px; border: 1px solid #e1e8e3; border-radius: 12px; background: white; padding: 12px; color: inherit; text-align: left; cursor: pointer; }
.atlas-mine__course[aria-pressed="true"] { border-color: #58846b; box-shadow: 0 0 0 2px #58846b22; }
.atlas-mine__period, .atlas-mine__course small { color: #6c7e74; font-size: 11px; }
.atlas-mine__course strong, .atlas-mine__selected strong { font-size: 14px; }
.atlas-mine__selected { display: grid; gap: 6px; border-top: 1px solid #e1e8e3; padding-top: 12px; font-size: 13px; }
.atlas-mine__selected p { margin: 0; color: #62746d; line-height: 1.5; }
.atlas-mine__selected .atlas-mine__unverified { color: #8a5522; font-weight: 600; }
</style>

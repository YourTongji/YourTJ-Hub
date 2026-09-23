<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getPkCalendars,
  getPkCourseDetails,
  getPkCoursesByTime,
} from '@/runtime/pk-api'
import type { PkCalendar, PkCourse } from '@/site/types/pk'
import type { CampusMapTarget } from '@/site/campus-map/official-location'

interface ScheduleEntry {
  key: string
  courseName: string
  courseCode: string
  campus: string
  room: string
  teacher: string
  arrangementText: string
}

const props = defineProps<{
  resolveLocation: (campus: string, room: string) => Promise<CampusMapTarget | undefined>
}>()
const emit = defineEmits<{ select: [target: CampusMapTarget | null] }>()
const { t } = useI18n()
const calendars = ref<PkCalendar[]>([])
const calendarId = ref<number>()
const day = ref(1)
const section = ref(1)
const week = ref(1)
const search = ref('')
const state = ref<'loading-calendar' | 'ready' | 'loading' | 'error'>('loading-calendar')
const entries = ref<ScheduleEntry[]>([])
const selected = ref<ScheduleEntry | null>(null)
const locationMapped = ref(false)
const locationResolved = ref(false)
let requestVersion = 0
let selectionVersion = 0

const filteredEntries = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  if (!query) return entries.value
  return entries.value.filter((entry) =>
    `${entry.courseName} ${entry.courseCode} ${entry.campus} ${entry.room} ${entry.teacher}`
      .toLocaleLowerCase().includes(query),
  )
})
const periods = computed(() => {
  const legacy = (calendarId.value ?? 0) < 120
  return [
    [1, 2], [3, 4], [5, 6], [7, 8], [9, 9], [10, legacy ? 12 : 11],
  ]
})

async function loadCalendars() {
  state.value = 'loading-calendar'
  try {
    calendars.value = await getPkCalendars()
    calendarId.value = calendars.value[0]?.calendarId
    state.value = calendars.value.length ? 'ready' : 'error'
  } catch {
    state.value = 'error'
  }
}

async function searchSchedule() {
  if (!calendarId.value || !Number.isInteger(week.value) || week.value < 1 || week.value > 16) return
  const version = ++requestVersion
  selected.value = null
  locationMapped.value = false
  locationResolved.value = false
  emit('select', null)
  state.value = 'loading'
  try {
    const result = await getPkCoursesByTime(calendarId.value, day.value, section.value, true)
    const courses = result.courses
    const codes = [...new Set(courses.map((course) => course.courseCode).filter(Boolean))]
    const detailMap: Record<string, Awaited<ReturnType<typeof getPkCourseDetails>>[string]> = {}
    for (let i = 0; i < codes.length; i += 500) {
      Object.assign(detailMap, await getPkCourseDetails(calendarId.value, codes.slice(i, i + 500)))
    }
    if (version !== requestVersion) return
    const slots = section.value === 6
      ? [10, 11, ...((calendarId.value ?? 0) < 120 ? [12] : [])]
      : periods.value[section.value - 1] ?? []
    entries.value = courses.flatMap((course: PkCourse) =>
      (detailMap[course.courseCode] ?? []).flatMap((detail) =>
        (detail.arrangementInfo ?? [])
          .filter((arrangement) => arrangement.occupyDay === day.value &&
            arrangement.occupyWeek?.includes(week.value) &&
            arrangement.occupyTime?.some((period) => slots.includes(period)))
          .map((arrangement, index) => ({
            key: `${course.courseCode}:${detail.teachingClassId}:${arrangement.arrangementText}:${index}`,
            courseName: course.courseName,
            courseCode: course.courseCode,
            campus: detail.campus || course.campus?.[0] || '',
            room: arrangement.occupyRoom ?? '',
            teacher: arrangement.teacherAndCode ?? '',
            arrangementText: arrangement.arrangementText,
          })),
      ),
    )
    state.value = 'ready'
  } catch {
    if (version === requestVersion) state.value = 'error'
  }
}

async function locate(entry: ScheduleEntry) {
  if (selected.value === entry) {
    selected.value = null
    locationMapped.value = false
    locationResolved.value = false
    emit('select', null)
    return
  }
  const version = ++selectionVersion
  selected.value = entry
  locationMapped.value = false
  locationResolved.value = false
  const target = await props.resolveLocation(entry.campus, entry.room)
  if (version !== selectionVersion || selected.value !== entry) return
  locationMapped.value = Boolean(target)
  locationResolved.value = true
  emit('select', target ?? null)
}

onMounted(() => { void loadCalendars() })
onBeforeUnmount(() => {
  requestVersion++
  selectionVersion++
})
</script>

<template>
  <div class="atlas-schedule">
    <p class="atlas-schedule__note">{{ t('campusMap.schedule.note') }}</p>
    <label>
      <span>{{ t('campusMap.schedule.term') }}</span>
      <select v-model.number="calendarId" :disabled="state === 'loading-calendar'">
        <option v-for="calendar in calendars" :key="calendar.calendarId" :value="calendar.calendarId">{{ calendar.calendarName }}</option>
      </select>
    </label>
    <div class="atlas-schedule__filters">
      <label>
        <span>{{ t('campusMap.schedule.day') }}</span>
        <select v-model.number="day">
          <option v-for="(key, index) in ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']" :key="key" :value="index + 1">{{ t(`campus.weekdays.${key}`) }}</option>
        </select>
      </label>
      <label>
        <span>{{ t('campusMap.schedule.period') }}</span>
        <select v-model.number="section">
          <option v-for="(period, index) in periods" :key="index" :value="index + 1">{{ t('campusMap.schedule.periodRange', { start: period[0], end: period[1] }) }}</option>
        </select>
      </label>
      <label>
        <span>{{ t('campusMap.schedule.week') }}</span>
        <input v-model.number="week" type="number" min="1" max="16" inputmode="numeric" />
      </label>
    </div>
    <button class="atlas-schedule__submit" type="button" :disabled="state === 'loading' || state === 'loading-calendar' || !calendarId" @click="searchSchedule">
      {{ t('campusMap.schedule.search') }}
    </button>
    <label class="atlas-schedule__search">
      <span class="sr-only">{{ t('campusMap.schedule.filter') }}</span>
      <input v-model="search" type="search" :placeholder="t('campusMap.schedule.filter')" />
    </label>
    <p v-if="state === 'loading-calendar' || state === 'loading'" class="atlas-schedule__status" role="status">{{ t('campusMap.schedule.loading') }}</p>
    <p v-else-if="state === 'error'" class="atlas-schedule__status" role="alert">{{ t('campusMap.schedule.unavailable') }} <button type="button" @click="calendars.length ? searchSchedule() : loadCalendars()">{{ t('campus.retry') }}</button></p>
    <p v-else-if="!filteredEntries.length" class="atlas-schedule__status" role="status">{{ search ? t('campusMap.schedule.noMatches') : t('campusMap.schedule.noCourses') }}</p>
    <div v-else class="atlas-schedule__list" aria-live="polite">
      <button v-for="entry in filteredEntries" :key="entry.key" type="button" :aria-pressed="selected === entry" @click="locate(entry)">
        <strong>{{ entry.courseName }}</strong>
        <span>{{ entry.courseCode }} · {{ entry.campus }} · {{ entry.room || t('campus.roomPending') }}</span>
        <small>{{ entry.teacher || entry.arrangementText }}</small>
      </button>
    </div>
    <p v-if="selected && locationResolved && !locationMapped" class="atlas-schedule__unmapped" role="status">{{ t('campusMap.mine.locationUnverified') }}</p>
  </div>
</template>

<style scoped>
.atlas-schedule { display: grid; gap: 12px; min-height: 0; overflow: auto; padding-bottom: 18px; color: #24342f; font-size: 12px; }
.atlas-schedule label { display: grid; gap: 5px; color: #62746d; }
.atlas-schedule select, .atlas-schedule input { min-width: 0; width: 100%; border: 1px solid #dce4de; border-radius: 9px; background: white; padding: 9px 10px; color: inherit; font: inherit; }
.atlas-schedule__filters { display: grid; grid-template-columns: 1fr 1.15fr .7fr; gap: 7px; }
.atlas-schedule__submit { min-height: 36px; border: 0; border-radius: 9px; background: #e7f1eb; color: #24543d; font: inherit; font-weight: 650; cursor: pointer; }
.atlas-schedule__submit:disabled { opacity: .55; cursor: wait; }
.atlas-schedule__note, .atlas-schedule__status, .atlas-schedule__unmapped { margin: 0; color: #62746d; line-height: 1.5; }
.atlas-schedule__list { display: grid; gap: 7px; min-height: 0; overflow: auto; }
.atlas-schedule__list button { display: grid; gap: 4px; border: 1px solid #e1e8e3; border-radius: 10px; background: white; padding: 10px; color: inherit; text-align: left; cursor: pointer; }
.atlas-schedule__list button[aria-pressed='true'] { border-color: #58846b; box-shadow: 0 0 0 2px #58846b22; }
.atlas-schedule__list span, .atlas-schedule__list small { color: #62746d; line-height: 1.4; }
.atlas-schedule__unmapped { color: #8a5522; font-weight: 600; }
</style>

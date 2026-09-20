<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import type { CampusEvent } from '@gooseforum/client'
import { getPkSectionTimes } from '@/runtime/pk-api'
import { buildTimetableGrid } from '@/site/utils/timetableGrid'
import { sectionTimesFor, type SectionTime } from '@/site/utils/sectionTimes'
import ScheduleTimeGrid from './schedule/ScheduleTimeGrid.vue'

const props = defineProps<{ courses: CampusEvent[]; week: number; today: number }>()
const isMobile = useMediaQuery('(max-width: 767px)')
const overrides = ref<SectionTime[]>([])
// Official records are adapted only for rendering. Never initialize the planner
// store: its localStorage draft is a different user-owned data set.
const rows = computed(() => props.courses.some(course => course.end === 12) ? 12 : 11)
const visibleCourses = computed(() => props.courses.filter(course =>
  (!course.weeks.length || course.weeks.includes(props.week)) &&
  Number.isInteger(course.start) && Number.isInteger(course.end) &&
  course.start >= 1 && course.end >= course.start && course.end <= rows.value,
))
const grid = computed(() => buildTimetableGrid(visibleCourses.value.map(course => ({
  code: '', courseName: course.name, showText: course.name,
  occupyDay: course.day,
  occupyTime: Array.from({ length: course.end - course.start + 1 }, (_, i) => course.start + i),
  occupyWeek: course.weeks, occupyRoom: course.room, teacherAndCode: course.teacher,
})), rows.value, false))
const times = computed(() => sectionTimesFor(rows.value, overrides.value))
let active = true
onMounted(async () => {
  try {
    const result = await getPkSectionTimes()
    if (active) overrides.value = result.sectionTimes
  } catch { /* Same built-in school timetable as the scheduler when offline. */ }
})
onBeforeUnmount(() => { active = false })
</script>

<template>
  <div class="overflow-x-auto rounded-2xl border border-line/70 bg-base-100 shadow-sm" data-testid="campus-timetable">
    <ScheduleTimeGrid :grid="grid" :section-times="times" :is-mobile="isMobile" :interactive="false" :show-course-codes="false" :active-day="today" />
  </div>
</template>

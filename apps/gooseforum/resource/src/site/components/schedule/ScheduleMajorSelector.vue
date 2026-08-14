<script setup lang="ts">
// 学期→年级→专业 选择器。任何一级变更都会清空已选/备选课程（防跨学期污染），
// 对齐上游 MajorInfo 交互语义。
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SiteSelect from '@/site/components/SiteSelect.vue'
import { getPkCalendars, getPkGrades, getPkMajors } from '@/runtime/pk-api'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkCalendar, PkMajor } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const calendars = ref<PkCalendar[]>([])
const grades = ref<number[]>([])
const majors = ref<PkMajor[]>([])
const loading = ref(false)
const error = ref('')
/** 初始化恢复期：抑制 watch 触发清空（避免误清刷新恢复的已选课程）。 */
let isRestoring = true

// SiteSelect 的 modelValue 为 string；选择值本地持有字符串，变更时写回 store。
const calendarValue = ref('')
const gradeValue = ref('')
const majorValue = ref('')

const calendarOptions = computed(() =>
  calendars.value.map((c) => ({ value: String(c.calendarId), label: c.calendarName })),
)
const gradeOptions = computed(() => grades.value.map((g) => ({ value: String(g), label: String(g) })))
const majorOptions = computed(() => majors.value.map((m) => ({ value: m.code, label: m.name })))

async function loadCalendars() {
  loading.value = true
  error.value = ''
  try {
    calendars.value = await getPkCalendars()
    // 恢复持久化的学期选择（无则自动选第一个）。
    const restored = store.state.majorSelected.calendarId
    if (restored !== undefined && calendars.value.some((c) => c.calendarId === restored)) {
      calendarValue.value = String(restored)
    } else if (calendars.value.length > 0) {
      calendarValue.value = String(calendars.value[0].calendarId)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    loading.value = false
  }
}

async function loadGrades(calendarId: number) {
  grades.value = []
  majors.value = []
  try {
    grades.value = await getPkGrades(calendarId)
    const restored = store.state.majorSelected.grade
    if (restored !== undefined && grades.value.includes(restored)) {
      gradeValue.value = String(restored)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  }
}

async function loadMajors(grade: number, calendarId: number) {
  majors.value = []
  try {
    majors.value = await getPkMajors(grade, calendarId)
    const restored = store.state.majorSelected.major
    if (restored && majors.value.some((m) => m.code === restored)) {
      majorValue.value = restored
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  }
}

function resetSelection() {
  store.clearStagedAndSelectedCourses()
}

/**
 * 初始化恢复序列：从 localStorage 恢复学期→年级→专业选择。
 * 恢复期间抑制 watch（不触发清空已选课程），否则刷新后恢复的已选课程会被误清
 * （验收标准 5：刷新后已选课程从 localStorage 恢复）。
 */
async function restoreSelection() {
  isRestoring = true
  await loadCalendars()
  const selection = store.state.majorSelected
  if (selection.calendarId !== undefined && selection.grade !== undefined) {
    await loadGrades(selection.calendarId)
    if (selection.major) await loadMajors(selection.grade, selection.calendarId)
  }
  isRestoring = false
}

// 学期变更：拉年级，清空年级/专业与课程缓存。
watch(calendarValue, (value) => {
  if (isRestoring) return
  store.setMajorInfo({ calendarId: value ? Number(value) : undefined, grade: undefined, major: undefined })
  resetSelection()
  if (value) void loadGrades(Number(value))
})

// 年级变更：拉专业，清空专业与课程缓存。
watch(gradeValue, (value) => {
  if (isRestoring) return
  const calendarId = store.state.majorSelected.calendarId
  store.setMajorInfo({ calendarId, grade: value ? Number(value) : undefined, major: undefined })
  resetSelection()
  if (value && calendarId !== undefined) void loadMajors(Number(value), calendarId)
})

// 专业变更：写回 store（课程缓存已由上游清空）。
watch(majorValue, (value) => {
  if (isRestoring) return
  const calendarId = store.state.majorSelected.calendarId
  const grade = store.state.majorSelected.grade
  store.setMajorInfo({ calendarId, grade, major: value || undefined })
  resetSelection()
})

onMounted(() => {
  void restoreSelection()
})
</script>

<template>
  <div class="gf-panel p-4">
    <div class="flex flex-col gap-3 md:flex-row md:items-end">
      <label class="block min-w-0 flex-1">
        <span class="mb-1.5 block text-sm font-medium text-base-content/70">{{ t('schedule.calendar') }}</span>
        <SiteSelect v-model="calendarValue" :options="calendarOptions" :placeholder="t('schedule.selectPlaceholder')" />
      </label>
      <label class="block min-w-0 flex-1">
        <span class="mb-1.5 block text-sm font-medium text-base-content/70">{{ t('schedule.grade') }}</span>
        <SiteSelect v-model="gradeValue" :options="gradeOptions" :placeholder="t('schedule.selectPlaceholder')" />
      </label>
      <label class="block min-w-0 flex-1">
        <span class="mb-1.5 block text-sm font-medium text-base-content/70">{{ t('schedule.major') }}</span>
        <SiteSelect v-model="majorValue" :options="majorOptions" :placeholder="t('schedule.selectPlaceholder')" />
      </label>
    </div>
    <p v-if="error" class="mt-2 rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
      {{ error }}
    </p>
    <p class="mt-2 text-xs text-base-content/55">{{ t('schedule.majorHint') }}</p>
    <p v-if="loading" class="mt-2 text-xs text-base-content/45">{{ t('schedule.loading') }}</p>
  </div>
</template>

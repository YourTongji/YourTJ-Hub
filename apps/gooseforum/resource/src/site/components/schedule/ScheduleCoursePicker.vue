<script setup lang="ts">
// 选课弹窗：计划内课程（必修）/ 通识选修 / 高级检索 3 tab。
// 勾选 key 编码对齐上游：必_{grade}_{code} / 选_{label}_{code} / 查_{code}。
// 提交时分类：必修直接从 compulsoryCourses 构造，选修与搜索批量取 course-details 构造
// stagedCourse 进备选池（验收标准 1：必修来自 courses-by-major、选修来自 course-details、搜索来自 course-search）。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search, X } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import SiteSelect from '@/site/components/SiteSelect.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import {
  getPkCampuses,
  getPkCourseDetails,
  getPkCoursesByMajor,
  getPkCoursesByNature,
  getPkFaculties,
  getPkOptionalTypes,
  searchPkCourses,
} from '@/runtime/pk-api'
import type { PkCourse, PkDictItem, PkStagedCourse } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

type TabKey = 'required' | 'optional' | 'search'
const activeTab = ref<TabKey>('required')
const selectedKeys = ref<Set<string>>(new Set())
const submitting = ref(false)
const error = ref('')

// ---- 必修：按年级分组 ----
const requiredGroups = computed(() => {
  const groups = new Map<number, PkCourse[]>()
  for (const course of store.state.commonLists.compulsoryCourses) {
    const grade = course.grade ?? 0
    if (!groups.has(grade)) groups.set(grade, [])
    groups.get(grade)!.push(course)
  }
  return [...groups.entries()]
    .map(([grade, courses]) => ({
      grade,
      courses: [...courses].sort((a, b) => a.courseCode.localeCompare(b.courseCode)),
    }))
    .sort((a, b) => b.grade - a.grade)
})

// ---- 通识：按类型分组 ----
const optionalGroups = computed(() => {
  const groups = new Map<string, PkCourse[]>()
  for (const course of store.state.commonLists.optionalCourses) {
    const label = (course.courseNature?.[0] ?? '') || 'default'
    if (!groups.has(label)) groups.set(label, [])
    groups.get(label)!.push(course)
  }
  return [...groups.entries()].map(([label, courses]) => ({
    label,
    courses: [...courses].sort((a, b) => a.courseCode.localeCompare(b.courseCode)),
  }))
})

// ---- 搜索 ----
const searchForm = ref({
  courseName: '',
  courseCode: '',
  teacherName: '',
})
const searchResults = ref<PkCourse[]>([])
const searchLoading = ref(false)
const campuses = ref<PkDictItem[]>([])
const faculties = ref<PkDictItem[]>([])
const campusValue = ref('')
const facultyValue = ref('')

// ---- 加载 ----
async function ensureRequiredAndOptional() {
  error.value = ''
  if (store.state.flags.majorNotChanged) return
  const calendarId = store.state.majorSelected.calendarId
  if (calendarId === undefined) return
  const grade = store.state.majorSelected.grade
  const major = store.state.majorSelected.major
  if (grade === undefined || !major) return

  try {
    const [compulsory, optionalTypes] = await Promise.all([
      getPkCoursesByMajor(grade, major, calendarId),
      getPkOptionalTypes(calendarId),
    ])
    store.setCompulsoryCourses(compulsory)
    store.setOptionalTypes(optionalTypes)
    if (optionalTypes.length > 0) {
      const natureCourses = await getPkCoursesByNature(
        calendarId,
        optionalTypes.map((type) => type.courseLabelId),
      )
      store.setOptionalCourses(natureCourses)
    }
    store.solidify()
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  }
}

async function loadSearchDicts() {
  try {
    const [campusList, facultyList] = await Promise.all([getPkCampuses(), getPkFaculties()])
    campuses.value = campusList
    faculties.value = facultyList
  } catch {
    // 字典加载失败不阻塞搜索表单。
  }
}

watch(
  () => props.open,
  (open) => {
    if (!open) return
    selectedKeys.value = new Set()
    activeTab.value = 'required'
    void ensureRequiredAndOptional()
    void loadSearchDicts()
  },
)

// ---- 勾选 ----
function toggleKey(key: string) {
  const next = new Set(selectedKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  selectedKeys.value = next
}

function isChecked(key: string): boolean {
  return selectedKeys.value.has(key)
}

function isAlreadyStaged(courseCode: string): boolean {
  // 两端都是基础课号（无班号后缀），直接比较；再套 getCourseBaseCode 会二次剥尾，
  // 导致 122004 / 122005 这类仅后两位不同的课程被误判为同一门。
  return store.state.commonLists.stagedCourses.some((course) => course.courseCode === courseCode)
}

async function runSearch() {
  const calendarId = store.state.majorSelected.calendarId
  if (calendarId === undefined) return
  searchLoading.value = true
  error.value = ''
  try {
    searchResults.value = await searchPkCourses({
      calendarId,
      courseName: searchForm.value.courseName || undefined,
      courseCode: searchForm.value.courseCode || undefined,
      teacherName: searchForm.value.teacherName || undefined,
      campus: campusValue.value || undefined,
      faculty: facultyValue.value || undefined,
    })
    // 结果写入 store，提交时从 searchCourses 还原课程基本信息（否则添加会静默失败）。
    store.setSearchedCourses(searchResults.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    searchLoading.value = false
  }
}

// ---- 提交 ----
function buildStagedCourse(course: PkCourse, courseType: string): PkStagedCourse {
  return {
    courseCode: course.courseCode,
    courseName: `${course.courseName}(${course.courseCode})`,
    courseNameReserved: course.courseName,
    credit: course.credit,
    courseType,
    courseNature: course.courseNature,
    teacher: [],
    status: 0,
    courseDetail: course.courseDetail.map((detail) => ({
      ...detail,
      status: detail.status ?? 0,
    })),
  }
}

async function submit() {
  const calendarId = store.state.majorSelected.calendarId
  if (calendarId === undefined) {
    emit('close')
    return
  }
  submitting.value = true
  error.value = ''

  const requiredCodes: Array<{ grade: number; courseCode: string }> = []
  const detailCodes: string[] = []

  for (const key of selectedKeys.value) {
    if (key.startsWith('必_')) {
      const parts = key.split('_')
      const grade = Number(parts[1])
      const courseCode = parts.slice(2).join('_')
      requiredCodes.push({ grade, courseCode })
    } else if (key.startsWith('选_')) {
      detailCodes.push(key.split('_').slice(2).join('_'))
    } else if (key.startsWith('查_')) {
      detailCodes.push(key.slice(2))
    }
  }

  try {
    // 1) 必修：直接从 compulsoryCourses 构造（验收标准 1 数据来源）。
    for (const { grade, courseCode } of requiredCodes) {
      const course = store.state.commonLists.compulsoryCourses.find(
        (c) => c.courseCode === courseCode && (c.grade ?? 0) === grade,
      )
      if (course) store.pushStagedCourse(buildStagedCourse(course, '必'))
    }

    // 2) 选修 + 搜索：批量取 course-details 构造（验收标准 1 数据来源）。
    if (detailCodes.length > 0) {
      const detailMap = await getPkCourseDetails(calendarId, detailCodes)
      for (const courseCode of detailCodes) {
        const details = detailMap[courseCode] ?? []
        if (details.length === 0) continue
        const rough = [
          ...store.state.commonLists.optionalCourses,
          ...store.state.commonLists.searchCourses,
        ].find((course) => course.courseCode === courseCode)
        if (!rough) continue
        store.pushStagedCourse({
          ...buildStagedCourse(rough, '选'),
          courseDetail: details.map((detail) => ({ ...detail, status: detail.status ?? 0 })),
        })
      }
    }

    store.solidify()
    selectedKeys.value = new Set()
    emit('close')
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="gf-fade">
      <div v-if="open" class="fixed inset-0 z-[2000]">
        <div class="absolute inset-0 bg-black/40" @click="emit('close')"></div>
        <div class="absolute left-1/2 top-1/2 flex max-h-[88vh] w-[92vw] max-w-3xl -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl">
          <div class="flex items-center justify-between border-b border-line/60 px-4 py-3">
            <h2 class="text-sm font-bold text-base-content">{{ t('schedule.openPicker') }}</h2>
            <button type="button" class="gf-icon-button" :aria-label="t('common.close')" @click="emit('close')">
              <X class="h-4 w-4" />
            </button>
          </div>

          <!-- tabs -->
          <div class="flex gap-1 border-b border-line/60 px-3 pt-2">
            <button
              v-for="tab in ([
                { key: 'required', label: t('schedule.tabRequired') },
                { key: 'optional', label: t('schedule.tabOptional') },
                { key: 'search', label: t('schedule.tabSearch') },
              ] as const)"
              :key="tab.key"
              type="button"
              class="gf-tab"
              :class="activeTab === tab.key ? 'gf-tab-active' : 'gf-tab-idle'"
              @click="activeTab = tab.key"
            >
              {{ tab.label }}
            </button>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto p-3">
            <p v-if="error" class="mb-2 rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
              {{ error }}
            </p>

            <!-- Tab1 必修：按年级分组 -->
            <div v-if="activeTab === 'required'" class="space-y-4">
              <EmptyState
                v-if="!requiredGroups.length"
                :icon="Search"
                :title="t('schedule.empty')"
              />
              <section v-for="group in requiredGroups" :key="group.grade">
                <h3 class="mb-1.5 text-sm font-bold text-base-content/80">{{ t('schedule.gradeUnit', { grade: group.grade }) }}</h3>
                <ul class="divide-y divide-line/60 rounded-lg border border-line/60">
                  <li v-for="course in group.courses" :key="course.courseCode">
                    <label class="flex cursor-pointer items-center gap-2 px-3 py-2" :class="isAlreadyStaged(course.courseCode) ? 'opacity-40' : ''">
                      <input
                        type="checkbox"
                        class="checkbox checkbox-sm"
                        :checked="isChecked(`必_${group.grade}_${course.courseCode}`)"
                        :disabled="isAlreadyStaged(course.courseCode)"
                        @change="toggleKey(`必_${group.grade}_${course.courseCode}`)"
                      />
                      <span class="min-w-0 flex-1">
                        <span class="block truncate text-sm text-base-content">{{ course.courseName }}</span>
                        <span class="block text-xs text-base-content/50">
                          {{ course.courseCode }} · {{ course.faculty }} · {{ t('schedule.credit', { credit: course.credit }) }}
                        </span>
                      </span>
                      <span v-if="course.courseNature?.length" class="gf-badge gf-badge-ghost text-xs">
                        {{ course.courseNature[0] }}
                      </span>
                    </label>
                  </li>
                </ul>
              </section>
            </div>

            <!-- Tab2 通识：按类型分组 -->
            <div v-else-if="activeTab === 'optional'" class="space-y-4">
              <EmptyState
                v-if="!optionalGroups.length"
                :icon="Search"
                :title="t('schedule.empty')"
              />
              <section v-for="group in optionalGroups" :key="group.label">
                <h3 class="mb-1.5 text-sm font-bold text-base-content/80">{{ group.label }}</h3>
                <ul class="divide-y divide-line/60 rounded-lg border border-line/60">
                  <li v-for="course in group.courses" :key="course.courseCode">
                    <label class="flex cursor-pointer items-center gap-2 px-3 py-2" :class="isAlreadyStaged(course.courseCode) ? 'opacity-40' : ''">
                      <input
                        type="checkbox"
                        class="checkbox checkbox-sm"
                        :checked="isChecked(`选_${group.label}_${course.courseCode}`)"
                        :disabled="isAlreadyStaged(course.courseCode)"
                        @change="toggleKey(`选_${group.label}_${course.courseCode}`)"
                      />
                      <span class="min-w-0 flex-1">
                        <span class="block truncate text-sm text-base-content">{{ course.courseName }}</span>
                        <span class="block text-xs text-base-content/50">
                          {{ course.courseCode }} · {{ course.campus?.join('、') }} · {{ t('schedule.credit', { credit: course.credit }) }}
                        </span>
                      </span>
                    </label>
                  </li>
                </ul>
              </section>
            </div>

            <!-- Tab3 高级检索 -->
            <div v-else class="space-y-3">
              <form class="grid gap-2 sm:grid-cols-2" @submit.prevent="runSearch">
                <label class="block">
                  <span class="mb-1 block text-xs text-base-content/70">{{ t('schedule.courseName') }}</span>
                  <input v-model="searchForm.courseName" type="text" class="gf-input gf-input-md w-full" :placeholder="t('schedule.searchPlaceholder')" />
                </label>
                <label class="block">
                  <span class="mb-1 block text-xs text-base-content/70">{{ t('schedule.courseCode') }}</span>
                  <input v-model="searchForm.courseCode" type="text" class="gf-input gf-input-md w-full" />
                </label>
                <label class="block">
                  <span class="mb-1 block text-xs text-base-content/70">{{ t('schedule.teacher') }}</span>
                  <input v-model="searchForm.teacherName" type="text" class="gf-input gf-input-md w-full" />
                </label>
                <label class="block">
                  <span class="mb-1 block text-xs text-base-content/70">{{ t('schedule.campus') }}</span>
                  <SiteSelect v-model="campusValue" :options="campuses.map((c) => ({ value: c.code, label: c.name }))" :placeholder="t('schedule.selectPlaceholder')" />
                </label>
                <label class="block sm:col-span-2">
                  <span class="mb-1 block text-xs text-base-content/70">{{ t('schedule.faculty') }}</span>
                  <SiteSelect v-model="facultyValue" :options="faculties.map((f) => ({ value: f.code, label: f.name }))" :placeholder="t('schedule.selectPlaceholder')" />
                </label>
                <button type="submit" class="gf-button gf-button-md gf-button-primary sm:col-span-2" :disabled="searchLoading">
                  <Search class="h-4 w-4" />
                  {{ t('schedule.searchButton') }}
                </button>
              </form>

              <ul v-if="searchResults.length" class="divide-y divide-line/60 rounded-lg border border-line/60">
                <li v-for="course in searchResults" :key="course.courseCode">
                  <label class="flex cursor-pointer items-center gap-2 px-3 py-2" :class="isAlreadyStaged(course.courseCode) ? 'opacity-40' : ''">
                    <input
                      type="checkbox"
                      class="checkbox checkbox-sm"
                      :checked="isChecked(`查_${course.courseCode}`)"
                      :disabled="isAlreadyStaged(course.courseCode)"
                      @change="toggleKey(`查_${course.courseCode}`)"
                    />
                    <span class="min-w-0 flex-1">
                      <span class="block truncate text-sm text-base-content">{{ course.courseName }}</span>
                      <span class="block text-xs text-base-content/50">
                        {{ course.courseCode }} · {{ course.faculty }} · {{ t('schedule.credit', { credit: course.credit }) }}
                      </span>
                    </span>
                  </label>
                </li>
              </ul>
              <EmptyState v-else-if="searchLoading" :icon="Search" :title="t('schedule.loading')" loading />
            </div>
          </div>

          <div class="flex justify-end gap-2 border-t border-line/60 px-4 py-3">
            <button type="button" class="gf-button gf-button-md gf-button-ghost" @click="emit('close')">
              {{ t('schedule.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-primary" :disabled="submitting || selectedKeys.size === 0" @click="submit">
              {{ t('schedule.submit') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

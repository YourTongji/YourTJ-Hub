<script setup lang="ts">
// 选课弹窗：计划内课程（必修）/ 通识选修 / 高级检索 3 tab。
// 勾选 key 编码对齐上游：必_{grade}_{code} / 选_{label}_{code} / 查_{code}。
// 提交时分类：必修直接从 compulsoryCourses 构造，选修与搜索批量取 course-details 构造
// stagedCourse 进备选课程（验收标准 1：必修来自 courses-by-major、选修来自 course-details、搜索来自 course-search）。
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import { useI18n } from 'vue-i18n'
import { ChevronLeft, ChevronRight, Loader2, RotateCcw, Search, X } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import SiteSelect from '@/site/components/SiteSelect.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { sortPlannedCoursesFirst } from '@/site/utils/pkCourseOrder'
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

// 弹窗无障碍（reka-ui Dialog）：焦点移入/Tab 圈禁/Esc 关闭/焦点恢复/滚动锁定由 Dialog 内置处理。
const pickerDialogOpen = computed({
  get: () => props.open,
  set: (open: boolean) => {
    if (!open) emit('close')
  },
})

type TabKey = 'required' | 'optional' | 'search'
const activeTab = ref<TabKey>('required')

const TAB_KEYS: TabKey[] = ['required', 'optional', 'search']

/** tablist 方向键切换（WAI-ARIA APG Tabs，issue #227）。 */
function handleTabKeydown(event: KeyboardEvent) {
  const current = TAB_KEYS.indexOf(activeTab.value)
  let next: number | undefined
  if (event.key === 'ArrowRight') next = (current + 1) % TAB_KEYS.length
  else if (event.key === 'ArrowLeft') next = (current - 1 + TAB_KEYS.length) % TAB_KEYS.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = TAB_KEYS.length - 1
  if (next === undefined) return
  event.preventDefault()
  activeTab.value = TAB_KEYS[next]
  const buttons = (event.currentTarget as HTMLElement).parentElement?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
  buttons?.[next]?.focus()
}

/** 通识选修分类 tab 方向键切换 */
function handleCategoryTabKeydown(event: KeyboardEvent) {
  const tabs = optionalCategoryTabs.value
  if (!tabs.length) return
  const current = tabs.findIndex((t) => t.key === activeOptionalCategory.value)
  let next: number | undefined
  if (event.key === 'ArrowRight') next = (current + 1) % tabs.length
  else if (event.key === 'ArrowLeft') next = (current - 1 + tabs.length) % tabs.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = tabs.length - 1
  if (next === undefined) return
  event.preventDefault()
  activeOptionalCategory.value = tabs[next].key
  const buttons = categoryScrollRef.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
  const targetBtn = buttons?.[next]
  if (targetBtn) {
    targetBtn.focus()
    targetBtn.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
  }
}

/** 鼠标滚轮横向滚动支持（优化桌面端交互） */
function handleCategoryWheel(event: WheelEvent) {
  const container = categoryScrollRef.value || (event.currentTarget as HTMLElement)
  if (!container) return
  if (container.scrollWidth <= container.clientWidth) return
  if (Math.abs(event.deltaY) > Math.abs(event.deltaX)) {
    container.scrollLeft += event.deltaY
    event.preventDefault()
    updateCategoryScrollState()
  }
}

const selectedKeys = ref<Set<string>>(new Set())
const submitting = ref(false)
const error = ref('')

// ---- 实时快捷搜索（按名称或代码过滤）----
const quickSearchQuery = ref('')
const normalizedQuery = computed(() => quickSearchQuery.value.trim().toLowerCase())

function matchesSearch(course: PkCourse): boolean {
  if (!normalizedQuery.value) return true
  const name = (course.courseName ?? '').toLowerCase()
  const code = (course.courseCode ?? '').toLowerCase()
  const faculty = (course.faculty ?? '').toLowerCase()
  return name.includes(normalizedQuery.value) || code.includes(normalizedQuery.value) || faculty.includes(normalizedQuery.value)
}

// ---- 弹窗上下文副标题（学期·专业）----
const headerContextText = computed(() => {
  const parts: string[] = []
  const sel = store.state.majorSelected
  if (sel.calendarId !== undefined) {
    const calendar = store.state.calendars.find((c) => c.calendarId === sel.calendarId)
    if (calendar?.calendarName) parts.push(calendar.calendarName)
  }
  if (sel.majorName) {
    parts.push(sel.majorName)
  } else if (sel.major) {
    parts.push(sel.major)
  }
  return parts.join(' · ')
})

interface MainTabItem {
  key: TabKey
  label: string
  count?: number
}

const mainTabs = computed<MainTabItem[]>(() => [
  { key: 'required', label: t('schedule.tabRequired'), count: store.state.commonLists.compulsoryCourses.length },
  { key: 'optional', label: t('schedule.tabOptional'), count: store.state.commonLists.optionalCourses.length },
  { key: 'search', label: t('schedule.tabSearch') },
])

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

const filteredRequiredGroups = computed(() => {
  return requiredGroups.value
    .map((group) => ({
      grade: group.grade,
      courses: group.courses.filter(matchesSearch),
    }))
    .filter((group) => group.courses.length > 0)
})

// ---- 通识：按类型横向 Tab 切分（对齐 YourTJCourse-Serverless a-tabs 体验）----
const activeOptionalCategory = ref<string>('')

interface OptionalCategoryTab {
  key: string
  label: string
  count: number
  selectedCount: number
  courses: PkCourse[]
}

const optionalCategoryTabs = computed<OptionalCategoryTab[]>(() => {
  const rawCourses = store.state.commonLists.optionalCourses || []
  const rawTypes = store.state.commonLists.optionalTypes || []

  // 1. 按课程性质分组
  const categoryMap = new Map<string, PkCourse[]>()
  for (const course of rawCourses) {
    const label = (course.courseNature?.[0] ?? '').trim() || 'default'
    if (!categoryMap.has(label)) categoryMap.set(label, [])
    categoryMap.get(label)!.push(course)
  }

  // 2. 按 optionalTypes 顺序排序分类名，补充剩余分类
  const orderedLabels: string[] = []
  for (const typeItem of rawTypes) {
    const name = (typeItem.courseLabelName || '').trim()
    if (name && !orderedLabels.includes(name) && categoryMap.has(name)) {
      orderedLabels.push(name)
    }
  }
  for (const label of categoryMap.keys()) {
    if (!orderedLabels.includes(label)) {
      orderedLabels.push(label)
    }
  }

  const tabs: OptionalCategoryTab[] = []

  // 3. 各分类独立的 Tab（对齐 a-tab-pane）
  for (const label of orderedLabels) {
    const courses = categoryMap.get(label) ?? []
    const sorted = [...courses].sort((a, b) => a.courseCode.localeCompare(b.courseCode))
    const selCount = sorted.filter((c) => isChecked(`选_${label}_${c.courseCode}`)).length
    tabs.push({
      key: label,
      label,
      count: sorted.length,
      selectedCount: selCount,
      courses: sorted,
    })
  }

  // 4. 全部 Tab（位于末尾，提供全局查阅兜底）
  if (tabs.length > 1) {
    let totalSelected = 0
    for (const tab of tabs) {
      totalSelected += tab.selectedCount
    }
    tabs.push({
      key: '__ALL__',
      label: t('schedule.allCategories'),
      count: rawCourses.length,
      selectedCount: totalSelected,
      courses: [...rawCourses].sort((a, b) => a.courseCode.localeCompare(b.courseCode)),
    })
  }

  return tabs
})

// 监听 optionalCategoryTabs 变化，确保 activeOptionalCategory 总是指向有效分类（默认首个具体分类）
watch(
  optionalCategoryTabs,
  (tabs) => {
    if (!tabs.length) {
      activeOptionalCategory.value = ''
      return
    }
    if (!tabs.some((t) => t.key === activeOptionalCategory.value)) {
      activeOptionalCategory.value = tabs[0].key
    }
    void nextTick(() => {
      updateCategoryScrollState()
    })
  },
  { immediate: true },
)

const categoryScrollRef = ref<HTMLElement | null>(null)
const canScrollLeft = ref(false)
const canScrollRight = ref(false)
let categoryResizeObserver: ResizeObserver | null = null

function updateCategoryScrollState() {
  const el = categoryScrollRef.value
  if (!el) {
    canScrollLeft.value = false
    canScrollRight.value = false
    return
  }
  canScrollLeft.value = el.scrollLeft > 2
  canScrollRight.value =
    el.scrollWidth > el.clientWidth + 2 &&
    el.scrollLeft + el.clientWidth < el.scrollWidth - 2
}

function scrollCategories(direction: 'left' | 'right') {
  const el = categoryScrollRef.value
  if (!el) return
  const delta = Math.max(el.clientWidth * 0.65, 180)
  el.scrollBy({ left: direction === 'left' ? -delta : delta, behavior: 'smooth' })
}

watch(categoryScrollRef, (newEl, oldEl) => {
  if (oldEl && categoryResizeObserver) {
    categoryResizeObserver.unobserve(oldEl)
  }
  if (newEl) {
    if (!categoryResizeObserver && typeof ResizeObserver !== 'undefined') {
      categoryResizeObserver = new ResizeObserver(() => {
        updateCategoryScrollState()
      })
    }
    categoryResizeObserver?.observe(newEl)
    void nextTick(() => {
      updateCategoryScrollState()
    })
  }
})

onBeforeUnmount(() => {
  categoryResizeObserver?.disconnect()
  categoryResizeObserver = null
})

watch(activeOptionalCategory, () => {
  void nextTick(() => {
    const activeBtn = categoryScrollRef.value?.querySelector<HTMLButtonElement>('[role="tab"][aria-selected="true"]')
    activeBtn?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
    updateCategoryScrollState()
  })
})

const currentCategoryLabel = computed(() => {
  const tab = optionalCategoryTabs.value.find((t) => t.key === activeOptionalCategory.value)
  return tab?.label || t('schedule.allCategories')
})

const currentCategoryCourses = computed(() => {
  if (!optionalCategoryTabs.value.length) return []
  const tab = optionalCategoryTabs.value.find((t) => t.key === activeOptionalCategory.value)
  const courses = tab ? tab.courses : optionalCategoryTabs.value[0].courses
  return courses.filter(matchesSearch)
})

function getOptionalCourseKey(course: PkCourse): string {
  const label = (course.courseNature?.[0] ?? '').trim() || 'default'
  return `选_${label}_${course.courseCode}`
}

// ---- 搜索 ----
const searchForm = ref({
  courseName: '',
  courseCode: '',
  teacherName: '',
  teacherCode: '',
})
const searchResults = ref<PkCourse[]>([])
const searchLoading = ref(false)
const searchFilterCollapsed = ref(false)
const campuses = ref<PkDictItem[]>([])
const faculties = ref<PkDictItem[]>([])
const campusValue = ref('')
const facultyValue = ref('')

const filteredSearchResults = computed(() => {
  return sortPlannedCoursesFirst(
    searchResults.value.filter(matchesSearch),
    store.state.commonLists.compulsoryCourses,
  )
})

const searchSummaryText = computed(() => {
  const parts: string[] = []
  if (searchForm.value.courseName?.trim()) parts.push(searchForm.value.courseName.trim())
  if (searchForm.value.courseCode?.trim()) parts.push(searchForm.value.courseCode.trim())
  if (searchForm.value.teacherName?.trim()) parts.push(searchForm.value.teacherName.trim())
  if (searchForm.value.teacherCode?.trim()) parts.push(`${t('schedule.teacherCode')}: ${searchForm.value.teacherCode.trim()}`)
  if (campusValue.value) {
    const campus = campuses.value.find((c) => c.code === campusValue.value)
    if (campus) parts.push(campus.name)
  }
  if (facultyValue.value) {
    const fac = faculties.value.find((f) => f.code === facultyValue.value)
    if (fac) parts.push(fac.name)
  }
  return parts.join(' · ')
})

function resetSearch() {
  searchForm.value = {
    courseName: '',
    courseCode: '',
    teacherName: '',
    teacherCode: '',
  }
  campusValue.value = ''
  facultyValue.value = ''
  searchResults.value = []
  searchFilterCollapsed.value = false
  error.value = ''
}

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
    quickSearchQuery.value = ''
    searchFilterCollapsed.value = false
    if (optionalCategoryTabs.value.length > 0) {
      activeOptionalCategory.value = optionalCategoryTabs.value[0].key
    }
    void ensureRequiredAndOptional()
    void loadSearchDicts()
  },
)

// ---- 勾选与计算 ----
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
  return store.state.commonLists.stagedCourses.some((course) => course.courseCode === courseCode)
}

const selectedTotalCredits = computed(() => {
  let total = 0
  for (const key of selectedKeys.value) {
    if (key.startsWith('必_')) {
      const parts = key.split('_')
      const grade = Number(parts[1])
      const courseCode = parts.slice(2).join('_')
      const course = store.state.commonLists.compulsoryCourses.find(
        (c) => c.courseCode === courseCode && (c.grade ?? 0) === grade,
      )
      if (course) total += course.credit
    } else if (key.startsWith('选_')) {
      const courseCode = key.split('_').slice(2).join('_')
      const course = store.state.commonLists.optionalCourses.find((c) => c.courseCode === courseCode)
      if (course) total += course.credit
    } else if (key.startsWith('查_')) {
      const courseCode = key.slice(2)
      const course = [
        ...searchResults.value,
        ...store.state.commonLists.searchCourses,
      ].find((c) => c.courseCode === courseCode)
      if (course) total += course.credit
    }
  }
  return Math.round(total * 10) / 10
})

// ---- 计划内课程批量选择逻辑 ----
interface RequiredCourseItemWithKey {
  grade: number
  course: PkCourse
  key: string
}

/** 当前可见（响应快捷搜索）的计划内课程 */
const currentVisibleRequiredItems = computed<RequiredCourseItemWithKey[]>(() => {
  return filteredRequiredGroups.value.flatMap((group) =>
    group.courses.map((course) => ({
      grade: group.grade,
      course,
      key: `必_${group.grade}_${course.courseCode}`,
    })),
  )
})

/** 当前可见且尚未加入备选课程的可勾选计划内课程 */
const selectableRequiredItems = computed<RequiredCourseItemWithKey[]>(() => {
  return currentVisibleRequiredItems.value.filter((item) => !isAlreadyStaged(item.course.courseCode))
})

/** 可勾选计划内课程总数 */
const totalSelectableRequiredCount = computed(() => selectableRequiredItems.value.length)

/** 当前已勾选的计划内课程数 */
const requiredSelectedCount = computed(() => {
  return selectableRequiredItems.value.filter((item) => selectedKeys.value.has(item.key)).length
})

/** 是否处于全选状态 */
const isAllRequiredSelected = computed(() => {
  return (
    totalSelectableRequiredCount.value > 0 &&
    requiredSelectedCount.value === totalSelectableRequiredCount.value
  )
})

/** 是否处于半选状态（部分选中） */
const isRequiredIndeterminate = computed(() => {
  return (
    requiredSelectedCount.value > 0 &&
    requiredSelectedCount.value < totalSelectableRequiredCount.value
  )
})

/** 当前所有计划课程是否均已加入备选课程 */
const allRequiredCoursesAlreadyStaged = computed(() => {
  return currentVisibleRequiredItems.value.length > 0 && totalSelectableRequiredCount.value === 0
})

/** 一键全选/反选计划内课程 */
function toggleAllRequired() {
  const next = new Set(selectedKeys.value)
  if (isAllRequiredSelected.value) {
    for (const item of selectableRequiredItems.value) {
      next.delete(item.key)
    }
  } else {
    for (const item of selectableRequiredItems.value) {
      next.add(item.key)
    }
  }
  selectedKeys.value = next
}

/** 清空当前筛选的所有计划课程勾选 */
function clearAllRequired() {
  const next = new Set(selectedKeys.value)
  for (const item of currentVisibleRequiredItems.value) {
    next.delete(item.key)
  }
  selectedKeys.value = next
}

/** 获取单年级分组内尚未在备选课程中的课程 */
function getGroupSelectableCourses(group: { grade: number; courses: PkCourse[] }) {
  return group.courses.filter((course) => !isAlreadyStaged(course.courseCode))
}

/** 判断单年级分组是否全部勾选 */
function isGroupAllSelected(group: { grade: number; courses: PkCourse[] }): boolean {
  const selectable = getGroupSelectableCourses(group)
  if (!selectable.length) return false
  return selectable.every((course) => selectedKeys.value.has(`必_${group.grade}_${course.courseCode}`))
}

/** 切换单年级分组全选/反选 */
function toggleGroupSelection(group: { grade: number; courses: PkCourse[] }) {
  const selectable = getGroupSelectableCourses(group)
  if (!selectable.length) return
  const next = new Set(selectedKeys.value)
  const allSelected = isGroupAllSelected(group)
  for (const course of selectable) {
    const key = `必_${group.grade}_${course.courseCode}`
    if (allSelected) {
      next.delete(key)
    } else {
      next.add(key)
    }
  }
  selectedKeys.value = next
}

async function runSearch() {
  const calendarId = store.state.majorSelected.calendarId
  if (calendarId === undefined) return
  searchLoading.value = true
  error.value = ''
  try {
    searchResults.value = await searchPkCourses({
      calendarId,
      courseName: searchForm.value.courseName?.trim() || undefined,
      courseCode: searchForm.value.courseCode?.trim() || undefined,
      teacherName: searchForm.value.teacherName?.trim() || undefined,
      teacherCode: searchForm.value.teacherCode?.trim() || undefined,
      campus: campusValue.value || undefined,
      faculty: facultyValue.value || undefined,
    })
    // 结果写入 store，提交时从 searchCourses 还原课程基本信息（否则添加会静默失败）。
    store.setSearchedCourses(searchResults.value)
    if (searchResults.value.length > 0) {
      searchFilterCollapsed.value = true
    }
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
  <DialogRoot v-model:open="pickerDialogOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2000] bg-black/50 backdrop-blur-xs transition-opacity duration-200" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2000] flex h-[88vh] max-h-[88vh] w-[94vw] max-w-4xl -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl outline-none"
      >
        <!-- 弹窗顶栏 -->
        <div class="shrink-0 flex items-center justify-between border-b border-line/60 px-4 py-3 bg-base-100">
          <div class="flex items-center gap-2.5 min-w-0">
            <DialogTitle class="text-sm sm:text-base font-bold text-base-content shrink-0">
              {{ t('schedule.openPicker') }}
            </DialogTitle>
            <span
              v-if="headerContextText"
              class="hidden sm:inline-block rounded-full bg-base-200/80 px-2.5 py-0.5 text-xs text-base-content/70 font-medium truncate max-w-md"
              :title="headerContextText"
            >
              {{ headerContextText }}
            </span>
          </div>
          <DialogDescription class="sr-only">{{ t('schedule.pickerDescription') }}</DialogDescription>
          <button
            type="button"
            class="gf-icon-button transition-transform active:scale-90"
            :aria-label="t('common.close')"
            :title="t('common.close')"
            @click="emit('close')"
          >
            <X class="h-4 w-4" />
          </button>
        </div>

        <!-- 主 Tab 与快捷搜索工具条（Sticky Header） -->
        <div class="shrink-0 border-b border-line/60 bg-base-100/90 px-3 py-2 sm:px-4 backdrop-blur-xs">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
            <!-- 3 主标签（WAI-ARIA APG Tabs） -->
            <div role="tablist" aria-label="course picker tabs" class="flex gap-1">
              <button
                v-for="tab in mainTabs"
                :key="tab.key"
                type="button"
                role="tab"
                :aria-selected="activeTab === tab.key"
                class="gf-tab group relative flex items-center gap-1.5"
                :class="activeTab === tab.key ? 'gf-tab-active font-semibold' : 'gf-tab-idle'"
                @click="activeTab = tab.key"
                @keydown="handleTabKeydown"
              >
                <span>{{ tab.label }}</span>
                <span
                  v-if="tab.count !== undefined && tab.count > 0"
                  class="inline-flex items-center justify-center min-w-[18px] rounded-full px-1.5 py-0.5 text-[10px] font-mono font-bold tabular-nums leading-none transition-colors"
                  :class="activeTab === tab.key
                    ? 'bg-neutral-content/20 text-neutral-content ring-1 ring-white/25'
                    : 'bg-base-200 text-base-content/70 group-hover:bg-base-300/80'"
                >
                  {{ tab.count }}
                </span>
              </button>
            </div>

            <!-- 快捷搜索输入框 -->
            <div class="relative flex items-center min-w-[210px] sm:w-64">
              <Search class="pointer-events-none absolute left-2.5 h-3.5 w-3.5 text-base-content/40" />
              <input
                v-model="quickSearchQuery"
                type="text"
                class="gf-input gf-input-sm w-full pl-8 pr-7 text-xs bg-base-200/50 focus:bg-base-100 transition-colors placeholder:text-base-content/40"
                :placeholder="t('schedule.quickSearchPlaceholder')"
                :aria-label="t('schedule.quickSearchPlaceholder')"
                @keydown.esc="quickSearchQuery = ''"
              />
              <button
                v-if="quickSearchQuery"
                type="button"
                class="absolute right-2 flex h-4 w-4 items-center justify-center rounded-full text-base-content/40 hover:text-base-content transition-colors"
                title="清空搜索"
                aria-label="清空搜索"
                @click="quickSearchQuery = ''"
              >
                <X class="h-3 w-3" />
              </button>
            </div>
          </div>
        </div>

        <!-- 通识选修课：横向分类切分 Tab（对齐 a-tabs 体验） -->
        <div
          v-if="activeTab === 'optional' && optionalCategoryTabs.length > 0"
          class="relative shrink-0 border-b border-line/60 bg-base-200/35 min-h-[52px] group/cat-rail"
        >
          <!-- 左侧渐变遮罩 -->
          <div
            v-if="canScrollLeft"
            class="pointer-events-none absolute inset-y-0 left-0 w-12 bg-gradient-to-r from-base-100 via-base-100/70 to-transparent z-10 transition-opacity duration-200"
            aria-hidden="true"
          />

          <!-- 左侧滚动按钮 -->
          <button
            v-if="canScrollLeft"
            type="button"
            class="absolute left-1.5 top-1/2 -translate-y-1/2 z-20 flex h-7 w-7 items-center justify-center rounded-full border border-line/80 bg-base-100/95 text-base-content/80 shadow-md backdrop-blur-xs transition-all duration-150 hover:bg-base-100 hover:text-primary hover:scale-105 active:scale-95 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-primary cursor-pointer"
            :aria-label="t('schedule.scrollCategoriesLeft')"
            @click="scrollCategories('left')"
          >
            <ChevronLeft class="h-4 w-4" />
          </button>

          <!-- 滚动轨道 -->
          <div
            ref="categoryScrollRef"
            role="tablist"
            aria-label="通识选修分类"
            class="flex items-center gap-2 overflow-x-auto gf-scrollbar-none [scrollbar-width:none] [&::-webkit-scrollbar]:hidden px-3 py-2.5 sm:px-4 scroll-smooth scroll-px-3 sm:scroll-px-4"
            @scroll.passive="updateCategoryScrollState"
            @wheel="handleCategoryWheel"
          >
            <button
              v-for="tab in optionalCategoryTabs"
              :key="tab.key"
              type="button"
              role="tab"
              :aria-selected="activeOptionalCategory === tab.key"
              :tabindex="activeOptionalCategory === tab.key ? 0 : -1"
              class="group relative inline-flex h-9 shrink-0 items-center gap-2 rounded-xl px-3.5 text-xs font-medium transition-all duration-150 select-none cursor-pointer focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-primary active:scale-[0.96]"
              :class="activeOptionalCategory === tab.key
                ? 'bg-primary text-primary-content shadow-xs font-semibold'
                : 'bg-base-100 border border-line/70 text-base-content/75 hover:bg-base-200 hover:text-base-content'"
              @click="activeOptionalCategory = tab.key"
              @keydown="handleCategoryTabKeydown"
            >
              <span class="whitespace-nowrap">{{ tab.label }}</span>
              <span
                class="inline-flex items-center justify-center min-w-[18px] rounded-full px-1.5 py-0.5 text-[10px] font-mono font-bold tabular-nums leading-none transition-colors"
                :class="activeOptionalCategory === tab.key
                  ? 'bg-primary-content/20 text-primary-content ring-1 ring-white/25'
                  : 'bg-base-200 text-base-content/70 group-hover:bg-base-300/80'"
              >
                {{ tab.count }}
              </span>
              <span
                v-if="tab.selectedCount > 0"
                class="flex items-center justify-center rounded-full px-1.5 py-0.5 text-[9px] font-bold tabular-nums leading-none"
                :class="activeOptionalCategory === tab.key ? 'bg-accent text-accent-content ring-1 ring-white/30' : 'bg-accent/20 text-accent ring-1 ring-accent/30'"
                :title="`该分类已勾选 ${tab.selectedCount} 门`"
              >
                ✓ {{ tab.selectedCount }}
              </span>
            </button>
          </div>

          <!-- 右侧渐变遮罩 -->
          <div
            v-if="canScrollRight"
            class="pointer-events-none absolute inset-y-0 right-0 w-12 bg-gradient-to-l from-base-100 via-base-100/70 to-transparent z-10 transition-opacity duration-200"
            aria-hidden="true"
          />

          <!-- 右侧滚动按钮 -->
          <button
            v-if="canScrollRight"
            type="button"
            class="absolute right-1.5 top-1/2 -translate-y-1/2 z-20 flex h-7 w-7 items-center justify-center rounded-full border border-line/80 bg-base-100/95 text-base-content/80 shadow-md backdrop-blur-xs transition-all duration-150 hover:bg-base-100 hover:text-primary hover:scale-105 active:scale-95 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-primary cursor-pointer"
            :aria-label="t('schedule.scrollCategoriesRight')"
            @click="scrollCategories('right')"
          >
            <ChevronRight class="h-4 w-4" />
          </button>
        </div>

        <!-- 弹窗可滚动主体 -->
        <div class="min-h-0 flex-1 overflow-y-auto p-3.5 sm:p-4 overscroll-contain">
          <p v-if="error" class="mb-3 rounded-xl border border-error/25 bg-error/10 px-3.5 py-2 text-xs text-error">
            {{ error }}
          </p>

          <!-- Tab1 计划内课程（必修）：按年级分组 -->
          <div v-if="activeTab === 'required'" class="space-y-4">
            <EmptyState
              v-if="!requiredGroups.length"
              :icon="Search"
              :title="t('schedule.emptyRequired')"
            />
            <EmptyState
              v-else-if="!filteredRequiredGroups.length"
              :icon="Search"
              :title="t('schedule.noMatchingCourses')"
            />
            <template v-else>
              <!-- 一键全选学期计划内课程常驻控制条 -->
              <div
                class="flex flex-wrap items-center justify-between gap-2.5 rounded-xl border border-line/60 bg-base-200/40 px-3.5 py-2.5 transition-colors select-none"
              >
                <label
                  class="group flex items-center gap-2.5 cursor-pointer min-w-0"
                  :class="{ 'cursor-not-allowed opacity-60': allRequiredCoursesAlreadyStaged }"
                >
                  <input
                    type="checkbox"
                    class="checkbox checkbox-sm checkbox-primary rounded-md"
                    :checked="isAllRequiredSelected"
                    :indeterminate.prop="isRequiredIndeterminate"
                    :disabled="allRequiredCoursesAlreadyStaged"
                    :aria-label="quickSearchQuery.trim() ? t('schedule.selectAllFilteredCourses') : t('schedule.selectAllSemesterCourses')"
                    @change="toggleAllRequired"
                  />
                  <span class="text-xs font-semibold text-base-content/90 group-hover:text-primary transition-colors">
                    {{ quickSearchQuery.trim() ? t('schedule.selectAllFilteredCourses') : t('schedule.selectAllSemesterCourses') }}
                  </span>
                </label>

                <div class="flex items-center gap-2.5 shrink-0 text-xs">
                  <span
                    class="rounded-full bg-base-200 px-2 py-0.5 text-[11px] font-mono tabular-nums text-base-content/70"
                    :class="{ 'bg-primary/10 text-primary font-bold': requiredSelectedCount > 0 }"
                  >
                    {{ t('schedule.selectedRatio', { selected: requiredSelectedCount, total: totalSelectableRequiredCount }) }}
                  </span>
                  <button
                    v-if="requiredSelectedCount > 0"
                    type="button"
                    class="text-[11px] font-medium text-base-content/60 hover:text-error transition-colors cursor-pointer"
                    @click="clearAllRequired"
                  >
                    {{ t('schedule.clear') }}
                  </button>
                </div>
              </div>

              <section v-for="group in filteredRequiredGroups" :key="group.grade">
                <div class="mb-2 flex items-center justify-between">
                  <h3 class="text-xs font-bold text-base-content/80">
                    {{ t('schedule.gradeUnit', { grade: group.grade }) }}
                  </h3>
                  <div class="flex items-center gap-2.5">
                    <button
                      v-if="filteredRequiredGroups.length > 1 && getGroupSelectableCourses(group).length > 0"
                      type="button"
                      class="text-[11px] font-medium text-primary hover:underline cursor-pointer"
                      @click="toggleGroupSelection(group)"
                    >
                      {{ isGroupAllSelected(group) ? t('schedule.unselectGrade') : t('schedule.selectGrade') }}
                    </button>
                    <span class="text-[11px] text-base-content/45 tabular-nums">
                      {{ t('schedule.searchResultCount', { count: group.courses.length }) }}
                    </span>
                  </div>
                </div>
                <ul class="space-y-2">
                  <li
                    v-for="course in group.courses"
                    :key="course.courseCode"
                    class="group relative flex items-start gap-3 rounded-xl border p-3 transition-all select-none"
                    :class="[
                      isAlreadyStaged(course.courseCode)
                        ? 'cursor-not-allowed bg-base-200/30 border-line/40 opacity-50'
                        : isChecked(`必_${group.grade}_${course.courseCode}`)
                          ? 'cursor-pointer border-primary/60 bg-primary/[0.04] shadow-xs ring-1 ring-primary/20'
                          : 'cursor-pointer border-line/60 bg-base-100 hover:border-line hover:bg-base-200/40 active:scale-[0.995]',
                    ]"
                    @click="!isAlreadyStaged(course.courseCode) && toggleKey(`必_${group.grade}_${course.courseCode}`)"
                  >
                    <div class="pt-0.5 shrink-0">
                      <input
                        type="checkbox"
                        class="checkbox checkbox-sm checkbox-primary"
                        :checked="isChecked(`必_${group.grade}_${course.courseCode}`)"
                        :disabled="isAlreadyStaged(course.courseCode)"
                        :aria-label="course.courseName + (isAlreadyStaged(course.courseCode) ? '（' + t('schedule.alreadyStaged') + '）' : '')"
                        @click.stop
                        @change="toggleKey(`必_${group.grade}_${course.courseCode}`)"
                      />
                    </div>
                    <div class="min-w-0 flex-1 space-y-1.5">
                      <div class="flex items-center gap-2">
                        <span
                          class="text-[13.5px] font-semibold text-base-content group-hover:text-primary transition-colors line-clamp-1"
                          :title="course.courseName"
                        >
                          {{ course.courseName }}
                        </span>
                        <span
                          v-if="isAlreadyStaged(course.courseCode)"
                          class="rounded-full bg-base-content/10 px-2 py-0.2 text-[10px] font-medium text-base-content/60 shrink-0"
                        >
                          {{ t('schedule.alreadyStaged') }}
                        </span>
                      </div>
                      <div class="flex flex-wrap items-center gap-1.5 text-[11px]">
                        <span class="rounded-md bg-base-200 px-1.5 py-0.5 font-mono text-base-content/75">
                          {{ course.courseCode }}
                        </span>
                        <span class="rounded-md bg-primary/10 px-1.5 py-0.5 font-semibold text-primary">
                          {{ t('schedule.credit', { credit: course.credit }) }}
                        </span>
                        <span
                          v-if="course.faculty"
                          class="rounded-md bg-base-200/80 px-1.5 py-0.5 text-base-content/70"
                        >
                          {{ course.faculty }}
                        </span>
                        <span
                          v-if="course.campus && course.campus.length"
                          class="rounded-md border border-info/30 bg-info/10 px-1.5 py-0.5 font-medium text-info"
                        >
                          {{ course.campus.join('、') }}
                        </span>
                        <span
                          v-if="course.courseNature && course.courseNature.length"
                          class="rounded-md bg-base-200/60 px-1.5 py-0.5 text-base-content/60"
                        >
                          {{ course.courseNature[0] }}
                        </span>
                      </div>
                    </div>
                  </li>
                </ul>
              </section>
            </template>
          </div>

          <!-- Tab2 通识选修课：分类浏览（仅展示当前选中分类的课程） -->
          <div v-else-if="activeTab === 'optional'" class="space-y-3">
            <EmptyState
              v-if="!optionalCategoryTabs.length"
              :icon="Search"
              :title="t('schedule.emptyOptional')"
            />
            <EmptyState
              v-else-if="!currentCategoryCourses.length"
              :icon="Search"
              :title="t('schedule.noMatchingCourses')"
            />
            <template v-else>
              <!-- 当前分类信息栏 -->
              <div class="flex items-center justify-between px-1">
                <div class="flex items-center gap-2">
                  <h3 class="text-xs font-bold text-base-content/85">
                    {{ currentCategoryLabel }}
                  </h3>
                  <span class="rounded-full bg-base-200/90 px-2 py-0.5 text-[10px] text-base-content/60 font-medium tabular-nums">
                    {{ t('schedule.searchResultCount', { count: currentCategoryCourses.length }) }}
                  </span>
                </div>
                <div v-if="quickSearchQuery" class="flex items-center gap-1 text-[11px] text-base-content/50">
                  <span>筛选关键词：<span class="font-medium text-base-content/80">“{{ quickSearchQuery }}”</span></span>
                  <button
                    type="button"
                    class="ml-1 text-primary hover:underline cursor-pointer"
                    @click="quickSearchQuery = ''"
                  >
                    {{ t('schedule.searchReset') }}
                  </button>
                </div>
              </div>

              <!-- 当前分类课程卡片列表 -->
              <ul class="space-y-2">
                <li
                  v-for="course in currentCategoryCourses"
                  :key="getOptionalCourseKey(course)"
                  class="group relative flex items-start gap-3 rounded-xl border p-3 transition-all select-none"
                  :class="[
                    isAlreadyStaged(course.courseCode)
                      ? 'cursor-not-allowed bg-base-200/30 border-line/40 opacity-50'
                      : isChecked(getOptionalCourseKey(course))
                        ? 'cursor-pointer border-primary/60 bg-primary/[0.04] shadow-xs ring-1 ring-primary/20'
                        : 'cursor-pointer border-line/60 bg-base-100 hover:border-line hover:bg-base-200/40 active:scale-[0.995]',
                  ]"
                  @click="!isAlreadyStaged(course.courseCode) && toggleKey(getOptionalCourseKey(course))"
                >
                  <div class="pt-0.5 shrink-0">
                    <input
                      type="checkbox"
                      class="checkbox checkbox-sm checkbox-primary"
                      :checked="isChecked(getOptionalCourseKey(course))"
                      :disabled="isAlreadyStaged(course.courseCode)"
                      :aria-label="course.courseName + (isAlreadyStaged(course.courseCode) ? '（' + t('schedule.alreadyStaged') + '）' : '')"
                      @click.stop
                      @change="toggleKey(getOptionalCourseKey(course))"
                    />
                  </div>
                  <div class="min-w-0 flex-1 space-y-1.5">
                    <div class="flex items-center gap-2">
                      <span
                        class="text-[13.5px] font-semibold text-base-content group-hover:text-primary transition-colors line-clamp-1"
                        :title="course.courseName"
                      >
                        {{ course.courseName }}
                      </span>
                      <span
                        v-if="isAlreadyStaged(course.courseCode)"
                        class="rounded-full bg-base-content/10 px-2 py-0.2 text-[10px] font-medium text-base-content/60 shrink-0"
                      >
                        {{ t('schedule.alreadyStaged') }}
                      </span>
                    </div>
                    <div class="flex flex-wrap items-center gap-1.5 text-[11px]">
                      <span class="rounded-md bg-base-200 px-1.5 py-0.5 font-mono text-base-content/75">
                        {{ course.courseCode }}
                      </span>
                      <span class="rounded-md bg-primary/10 px-1.5 py-0.5 font-semibold text-primary">
                        {{ t('schedule.credit', { credit: course.credit }) }}
                      </span>
                      <span
                        v-if="course.faculty"
                        class="rounded-md bg-base-200/80 px-1.5 py-0.5 text-base-content/70"
                      >
                        {{ course.faculty }}
                      </span>
                      <span
                        v-if="course.campus && course.campus.length"
                        class="rounded-md border border-info/30 bg-info/10 px-1.5 py-0.5 font-medium text-info"
                      >
                        {{ course.campus.join('、') }}
                      </span>
                      <span
                        v-if="course.courseNature && course.courseNature.length"
                        class="rounded-md bg-base-200/60 px-1.5 py-0.5 text-base-content/60"
                      >
                        {{ course.courseNature[0] }}
                      </span>
                    </div>
                  </div>
                </li>
              </ul>
            </template>
          </div>

          <!-- Tab3 高级检索：重塑的卡片表单与可收起机制 -->
          <div v-else class="space-y-4">
            <!-- 检索表单卡片 -->
            <div class="rounded-2xl border border-line/70 bg-base-200/30 p-3.5 sm:p-4 transition-all">
              <div v-if="!searchFilterCollapsed" class="space-y-3">
                <form class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3" @submit.prevent="runSearch">
                  <!-- 课程名称 -->
                  <label class="block">
                    <span class="mb-1 block text-[12px] font-medium text-base-content/70">{{ t('schedule.courseName') }}</span>
                    <div class="relative flex items-center">
                      <input
                        v-model="searchForm.courseName"
                        type="text"
                        class="gf-input gf-input-sm w-full pr-7 text-xs"
                        :placeholder="t('schedule.searchPlaceholder')"
                      />
                      <button
                        v-if="searchForm.courseName"
                        type="button"
                        class="absolute right-2 text-base-content/40 hover:text-base-content"
                        @click="searchForm.courseName = ''"
                      >
                        <X class="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </label>

                  <!-- 课程代码 -->
                  <label class="block">
                    <span class="mb-1 block text-[12px] font-medium text-base-content/70">{{ t('schedule.courseCode') }}</span>
                    <div class="relative flex items-center">
                      <input
                        v-model="searchForm.courseCode"
                        type="text"
                        class="gf-input gf-input-sm w-full pr-7 font-mono text-xs"
                        placeholder="如 122004"
                      />
                      <button
                        v-if="searchForm.courseCode"
                        type="button"
                        class="absolute right-2 text-base-content/40 hover:text-base-content"
                        @click="searchForm.courseCode = ''"
                      >
                        <X class="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </label>

                  <!-- 教师姓名 -->
                  <label class="block">
                    <span class="mb-1 block text-[12px] font-medium text-base-content/70">{{ t('schedule.teacher') }}</span>
                    <div class="relative flex items-center">
                      <input
                        v-model="searchForm.teacherName"
                        type="text"
                        class="gf-input gf-input-sm w-full pr-7 text-xs"
                        placeholder="如 教师姓名"
                      />
                      <button
                        v-if="searchForm.teacherName"
                        type="button"
                        class="absolute right-2 text-base-content/40 hover:text-base-content"
                        @click="searchForm.teacherName = ''"
                      >
                        <X class="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </label>

                  <!-- 教师工号 -->
                  <label class="block">
                    <span class="mb-1 block text-[12px] font-medium text-base-content/70">{{ t('schedule.teacherCode') }}</span>
                    <div class="relative flex items-center">
                      <input
                        v-model="searchForm.teacherCode"
                        type="text"
                        class="gf-input gf-input-sm w-full pr-7 font-mono text-xs"
                        :placeholder="t('schedule.teacherCodePlaceholder')"
                      />
                      <button
                        v-if="searchForm.teacherCode"
                        type="button"
                        class="absolute right-2 text-base-content/40 hover:text-base-content cursor-pointer"
                        @click="searchForm.teacherCode = ''"
                      >
                        <X class="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </label>

                  <!-- 校区 -->
                  <label class="block">
                    <span class="mb-1 block text-[12px] font-medium text-base-content/70">{{ t('schedule.campus') }}</span>
                    <SiteSelect
                      v-model="campusValue"
                      :options="campuses.map((c) => ({ value: c.code, label: c.name }))"
                      :placeholder="t('schedule.selectPlaceholder')"
                      :label="t('schedule.campus')"
                      clearable
                      searchable
                      :search-placeholder="t('schedule.selectSearchPlaceholder')"
                      :empty-text="t('schedule.selectSearchEmpty')"
                    />
                  </label>

                  <!-- 开课学院 -->
                  <label class="block">
                    <span class="mb-1 block text-[12px] font-medium text-base-content/70">{{ t('schedule.faculty') }}</span>
                    <SiteSelect
                      v-model="facultyValue"
                      :options="faculties.map((f) => ({ value: f.code, label: f.name }))"
                      :placeholder="t('schedule.selectPlaceholder')"
                      :label="t('schedule.faculty')"
                      clearable
                      searchable
                      :search-placeholder="t('schedule.selectSearchPlaceholder')"
                      :empty-text="t('schedule.selectSearchEmpty')"
                    />
                  </label>

                  <button type="submit" class="hidden" aria-hidden="true" tabindex="-1" />
                </form>

                <!-- 表单动作行 -->
                <div class="flex items-center justify-between pt-2 border-t border-line/40">
                  <button
                    type="button"
                    class="gf-button gf-button-sm gf-button-ghost text-xs text-base-content/70 hover:text-base-content"
                    @click="resetSearch"
                  >
                    <RotateCcw class="h-3.5 w-3.5 mr-1" />
                    {{ t('schedule.searchReset') }}
                  </button>
                  <div class="flex items-center gap-2">
                    <button
                      v-if="searchResults.length > 0"
                      type="button"
                      class="gf-button gf-button-sm gf-button-ghost text-xs"
                      @click="searchFilterCollapsed = true"
                    >
                      {{ t('schedule.searchFilterToggleCollapse') }}
                    </button>
                    <button
                      type="button"
                      class="gf-button gf-button-sm gf-button-primary text-xs"
                      :disabled="searchLoading"
                      @click="runSearch"
                    >
                      <Loader2 v-if="searchLoading" class="h-3.5 w-3.5 animate-spin mr-1" />
                      <Search v-else class="h-3.5 w-3.5 mr-1" />
                      {{ t('schedule.searchButton') }}
                    </button>
                  </div>
                </div>
              </div>

              <!-- 折叠状态摘要条 -->
              <div v-else class="flex flex-wrap items-center justify-between gap-2 text-xs py-1">
                <div class="flex items-center gap-2 text-base-content/75 min-w-0">
                  <Search class="h-3.5 w-3.5 text-primary shrink-0" />
                  <span class="font-semibold text-base-content/90 shrink-0">
                    {{ t('schedule.searchResultCount', { count: searchResults.length }) }}
                  </span>
                  <span
                    v-if="searchSummaryText"
                    class="text-base-content/55 text-[11px] truncate max-w-[200px] sm:max-w-xs"
                    :title="searchSummaryText"
                  >
                    （{{ searchSummaryText }}）
                  </span>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <button
                    type="button"
                    class="gf-button gf-button-xs gf-button-ghost text-xs text-primary hover:underline cursor-pointer"
                    @click="searchFilterCollapsed = false"
                  >
                    {{ t('schedule.searchFilterToggleExpand') }}
                  </button>
                  <button
                    type="button"
                    class="gf-button gf-button-xs gf-button-ghost text-xs text-base-content/60 cursor-pointer"
                    @click="resetSearch"
                  >
                    {{ t('schedule.searchReset') }}
                  </button>
                </div>
              </div>
            </div>

            <!-- 检索结果列表 -->
            <div v-if="searchResults.length" class="space-y-2">
              <div class="flex items-center justify-between px-1">
                <span class="text-xs font-semibold text-base-content/80">
                  {{ t('schedule.searchResultCount', { count: filteredSearchResults.length }) }}
                </span>
              </div>
              <ul class="space-y-2">
                <li
                  v-for="course in filteredSearchResults"
                  :key="course.courseCode"
                  class="group relative flex items-start gap-3 rounded-xl border p-3 transition-all select-none"
                  :class="[
                    isAlreadyStaged(course.courseCode)
                      ? 'cursor-not-allowed bg-base-200/30 border-line/40 opacity-50'
                      : isChecked(`查_${course.courseCode}`)
                        ? 'cursor-pointer border-primary/60 bg-primary/[0.04] shadow-xs ring-1 ring-primary/20'
                        : 'cursor-pointer border-line/60 bg-base-100 hover:border-line hover:bg-base-200/40 active:scale-[0.995]',
                  ]"
                  @click="!isAlreadyStaged(course.courseCode) && toggleKey(`查_${course.courseCode}`)"
                >
                  <div class="pt-0.5 shrink-0">
                    <input
                      type="checkbox"
                      class="checkbox checkbox-sm checkbox-primary"
                      :checked="isChecked(`查_${course.courseCode}`)"
                      :disabled="isAlreadyStaged(course.courseCode)"
                      :aria-label="course.courseName + (isAlreadyStaged(course.courseCode) ? '（' + t('schedule.alreadyStaged') + '）' : '')"
                      @click.stop
                      @change="toggleKey(`查_${course.courseCode}`)"
                    />
                  </div>
                  <div class="min-w-0 flex-1 space-y-1.5">
                    <div class="flex items-center gap-2">
                      <span
                        class="text-[13.5px] font-semibold text-base-content group-hover:text-primary transition-colors line-clamp-1"
                        :title="course.courseName"
                      >
                        {{ course.courseName }}
                      </span>
                      <span
                        v-if="isAlreadyStaged(course.courseCode)"
                        class="rounded-full bg-base-content/10 px-2 py-0.2 text-[10px] font-medium text-base-content/60 shrink-0"
                      >
                        {{ t('schedule.alreadyStaged') }}
                      </span>
                    </div>
                    <div class="flex flex-wrap items-center gap-1.5 text-[11px]">
                      <span class="rounded-md bg-base-200 px-1.5 py-0.5 font-mono text-base-content/75">
                        {{ course.courseCode }}
                      </span>
                      <span class="rounded-md bg-primary/10 px-1.5 py-0.5 font-semibold text-primary">
                        {{ t('schedule.credit', { credit: course.credit }) }}
                      </span>
                      <span
                        v-if="course.faculty"
                        class="rounded-md bg-base-200/80 px-1.5 py-0.5 text-base-content/70"
                      >
                        {{ course.faculty }}
                      </span>
                      <span
                        v-if="course.campus && course.campus.length"
                        class="rounded-md border border-info/30 bg-info/10 px-1.5 py-0.5 font-medium text-info"
                      >
                        {{ course.campus.join('、') }}
                      </span>
                      <span
                        v-if="course.courseNature && course.courseNature.length"
                        class="rounded-md bg-base-200/60 px-1.5 py-0.5 text-base-content/60"
                      >
                        {{ course.courseNature[0] }}
                      </span>
                    </div>
                  </div>
                </li>
              </ul>
            </div>
            <EmptyState v-else-if="searchLoading" :icon="Search" :title="t('schedule.loading')" loading />
            <EmptyState v-else :icon="Search" :title="t('schedule.emptySearch')" />
          </div>
        </div>

        <!-- 弹窗固定底栏 -->
        <div class="shrink-0 flex flex-col sm:flex-row items-center justify-between gap-3 border-t border-line/60 px-4 py-3 bg-base-100">
          <!-- 左侧：已选统计与清空操作 -->
          <div class="flex items-center gap-2 text-xs text-base-content/75 w-full sm:w-auto">
            <template v-if="selectedKeys.size > 0">
              <span class="font-medium text-base-content">
                {{ t('schedule.selectedCountSummary', { count: selectedKeys.size }) }}
              </span>
              <span class="text-base-content/40">·</span>
              <span class="text-primary font-semibold">
                {{ t('schedule.selectedCreditsSummary', { credits: selectedTotalCredits }) }}
              </span>
              <button
                type="button"
                class="ml-2 text-xs text-error hover:underline cursor-pointer"
                @click="selectedKeys.clear()"
              >
                {{ t('schedule.clearSelected') }}
              </button>
            </template>
            <template v-else>
              <span class="text-base-content/45">暂未勾选课程</span>
            </template>
          </div>

          <!-- 右侧：取消与提交按钮 -->
          <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
            <button
              type="button"
              class="gf-button gf-button-md gf-button-ghost flex-1 sm:flex-initial"
              @click="emit('close')"
            >
              {{ t('schedule.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-md gf-button-primary flex-1 sm:flex-initial active:scale-[0.96]"
              :disabled="submitting || selectedKeys.size === 0"
              @click="submit"
            >
              <Loader2 v-if="submitting" class="h-4 w-4 animate-spin mr-1" />
              {{ selectedKeys.size > 0 ? t('schedule.addToStagedWithCount', { count: selectedKeys.size }) : t('schedule.submit') }}
            </button>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

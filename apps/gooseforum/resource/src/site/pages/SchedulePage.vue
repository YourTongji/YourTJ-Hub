<script setup lang="ts">
// 排课器主页面（/schedule，course.schedule）—— v2 布局。
// 桌面左右两栏：左栏（方案条 / 学期·年级·专业 / 统计卡 / 已选列表含搜索），
// 右栏课表为主视觉；移动端保持三 tab（课表/选课/详情）。
// 数据全部走 /api/pk/* JSON API 异步加载（SSR 空壳）；localStorage 持久化由 store 负责。
// 数据过期提示 + 「同步最新」：P11 latest-update 对比本地 updateTime，P12 course-info-sync
// 以全方案课程并集请求，applySyncToAllPlans 各方案保留排课状态。
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download, RefreshCw } from '@lucide/vue'
import PageHeader from '@/site/components/PageHeader.vue'
import ScheduleMajorSelector from '@/site/components/schedule/ScheduleMajorSelector.vue'
import SchedulePlanBar from '@/site/components/schedule/SchedulePlanBar.vue'
import ScheduleStatsCard from '@/site/components/schedule/ScheduleStatsCard.vue'
import ScheduleRoughList from '@/site/components/schedule/ScheduleRoughList.vue'
import ScheduleDetailList from '@/site/components/schedule/ScheduleDetailList.vue'
import ScheduleTimeTable from '@/site/components/schedule/ScheduleTimeTable.vue'
import ScheduleCoursePicker from '@/site/components/schedule/ScheduleCoursePicker.vue'
import ScheduleCellPicker from '@/site/components/schedule/ScheduleCellPicker.vue'
import ScheduleCustomEventDialog from '@/site/components/schedule/ScheduleCustomEventDialog.vue'
import ScheduleDetailCard from '@/site/components/schedule/ScheduleDetailCard.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { getPkLatestUpdate, syncPkCourseInfo } from '@/runtime/pk-api'
import { queueFlashMessage } from '@/runtime/flash-message'
import { codesToCsvRows, codesToXlsRows, downloadCsv, downloadXls, jsonToCsv, xlsRowsToXml } from '@/site/utils/pkExport'
import type { LayoutPayload, SchedulePageProps } from '@gooseforum/client'
import type { PkConflictItem } from '@/site/utils/pkConflict'
import type { PkCourseDetail, PkCourseOnTable } from '@/site/types/pk'

defineProps<{
  layout: LayoutPayload
  props: SchedulePageProps
}>()

const { t } = useI18n()
const store = useScheduleStore()

const isMobile = ref(false)
const mobileTab = ref<'timetable' | 'list' | 'detail'>('timetable')

const MOBILE_TABS: Array<{ key: 'timetable' | 'list' | 'detail'; label: string }> = [
  { key: 'timetable', label: t('schedule.timetable') },
  { key: 'list', label: t('schedule.pickCourses') },
  { key: 'detail', label: t('schedule.detail') },
]

/** 移动端 tab 方向键切换（WAI-ARIA APG Tabs，issue #227）。 */
function handleMobileTabKeydown(event: KeyboardEvent) {
  const current = MOBILE_TABS.findIndex((tab) => tab.key === mobileTab.value)
  let next: number | undefined
  if (event.key === 'ArrowRight') next = (current + 1) % MOBILE_TABS.length
  else if (event.key === 'ArrowLeft') next = (current - 1 + MOBILE_TABS.length) % MOBILE_TABS.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = MOBILE_TABS.length - 1
  if (next === undefined) return
  event.preventDefault()
  mobileTab.value = MOBILE_TABS[next].key
  const buttons = (event.currentTarget as HTMLElement).parentElement?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
  buttons?.[next]?.focus()
}

const pickerOpen = ref(false)
const customizeOpen = ref(false)
const detailCourse = ref<PkCourseOnTable | null>(null)
/** 点击课表空白格 → 该时段备选课程选择框（day/section 为点击位置）。 */
const cellPick = ref<{ day: number; section: number } | null>(null)

const dataOutdated = computed(() => store.state.flags.isDataOutdated)

function flash(message: string, type: 'success' | 'error' | 'warning' | 'info' = 'info') {
  queueFlashMessage(message, type)
}

/** 学期字典由 MajorSelector 的 loadCalendars 回填 store（P1 含起止日期），
 * 此处不重复请求。 */
// ---- 数据过期检查与同步 ----
async function checkDataOutdated() {
  store.loadSolidifyTime()
  try {
    const latest = await getPkLatestUpdate()
    const latestSyncAt = latest.latestSyncAt ?? ''
    store.setLatestUpdateTime(latestSyncAt)
    if (store.state.updateTime === '') {
      // 首次进入且已有已选课程：不清空用户数据，仅记录同步时间（防数据丢失）。
      const hasSelection = store.state.plans.some((plan) => plan.selectedCourses.length > 0)
      if (!hasSelection) {
        store.syncLatestData()
      } else {
        store.setUpdateTime(latestSyncAt)
      }
      return
    }
    store.setDataOutdated(store.state.updateTime !== latestSyncAt)
  } catch {
    // P11 不可用（#187 未合入）：静默，不提示过期。
    store.setDataOutdated(false)
  }
}

const syncing = ref(false)

/** 全方案课程并集（同步请求用）。 */
function allPlansCourseCodes(): { majorCodes: string[]; otherCodes: string[] } {
  const major = new Set<string>()
  const other = new Set<string>()
  for (const plan of store.state.plans) {
    for (const course of plan.stagedCourses) {
      const isExclusive = course.courseDetail.some((detail) => detail.isExclusive === true)
      if (isExclusive) major.add(course.courseCode)
      else other.add(course.courseCode)
    }
  }
  return { majorCodes: [...major], otherCodes: [...other] }
}

async function syncLatest() {
  if (syncing.value) return // 防重入
  const calendarId = store.state.majorSelected.calendarId
  const { majorCodes, otherCodes } = allPlansCourseCodes()
  if (calendarId === undefined || (majorCodes.length === 0 && otherCodes.length === 0)) {
    store.syncLatestData()
    flash(t('schedule.syncSuccess'), 'success')
    return
  }

  const grade = store.state.majorSelected.grade
  const major = store.state.majorSelected.major

  syncing.value = true
  try {
    const result = await syncPkCourseInfo({
      calendarId,
      majorCourseCodes: majorCodes,
      otherCourseCodes: otherCodes,
      majorInfo: { grade: grade ?? 0, code: major ?? '' },
    })

    // 各方案按课号命中替换详情，保留各方案自己的排课状态。
    store.applySyncToAllPlans(result)
    flash(t('schedule.syncSuccess'), 'success')
  } catch (err) {
    flash(err instanceof Error ? err.message : t('schedule.loadFailed'), 'error')
  } finally {
    syncing.value = false
  }
}

// ---- 冲突处理（容忍式：仅 flash 提示，不弹窗）----
function handleConflict(_detail: PkCourseDetail, conflicts: PkConflictItem[]) {
  const first = conflicts[0]?.courseName ?? ''
  flash(
    conflicts.length > 1
      ? t('schedule.stagedWithConflicts', { course: first, count: conflicts.length })
      : t('schedule.stagedWithConflict', { course: first }),
    'warning',
  )
}

// ---- 课表点击 ----
function handleOpenDetail(course: PkCourseOnTable) {
  detailCourse.value = course
}

function handleCellClick(day: number, section: number) {
  // 点击课表空格：弹出该时段可选课程（来自备选池 stagedCourses）。
  cellPick.value = { day, section }
}

// ---- 导出（CSV/XLS 菜单保留在页头；PNG 在课表工具条内）----
const exportOpen = ref(false)
const exportRoot = ref<HTMLElement | null>(null)
const exportButton = ref<HTMLButtonElement | null>(null)
const exportMenu = ref<HTMLElement | null>(null)

function openExportMenu() {
  exportOpen.value = true
  // 打开后聚焦首项，保证键盘用户可直接继续导航（无需再 Tab 到菜单内）。
  nextTick(() => {
    exportMenu.value?.querySelector<HTMLButtonElement>('button')?.focus()
  })
}

function closeExportMenu() {
  exportOpen.value = false
  // 菜单项激活 / Esc 关闭后，焦点还原到触发按钮。
  exportButton.value?.focus()
}

function toggleExportMenu() {
  if (exportOpen.value) {
    closeExportMenu()
  } else {
    openExportMenu()
  }
}

function handleExportOutsidePointerDown(event: PointerEvent) {
  const target = event.target
  if (target instanceof Node && exportRoot.value?.contains(target)) return
  // 外部点击关闭时不抢焦点，焦点留给用户点击的目标。
  exportOpen.value = false
}

function handleExportKeydown(event: KeyboardEvent) {
  // 监听 document，菜单打开时无论焦点在按钮还是菜单项内，Esc 都能关闭。
  if (event.key !== 'Escape' || !exportOpen.value) return
  event.preventDefault()
  closeExportMenu()
}

function exportableClassCodes(): string[] {
  const codes: string[] = []
  for (const plan of store.state.plans) {
    if (plan.id !== store.state.activePlanId) continue
    for (const course of plan.stagedCourses) {
      for (const detail of course.courseDetail) {
        if (detail.status === 1 || detail.status === 2) codes.push(detail.code)
      }
    }
  }
  return codes
}

function exportCsv() {
  closeExportMenu()
  const codes = exportableClassCodes()
  if (codes.length === 0) {
    flash(t('schedule.exportEmpty'), 'warning')
    return
  }
  const rows = codesToCsvRows(codes, store.state.commonLists.stagedCourses)
  downloadCsv('yourtj-schedule.csv', jsonToCsv(rows))
}

function exportXls() {
  closeExportMenu()
  const codes = exportableClassCodes()
  if (codes.length === 0) {
    flash(t('schedule.exportEmpty'), 'warning')
    return
  }
  const rows = codesToXlsRows(codes, store.state.commonLists.stagedCourses)
  downloadXls('yourtj-schedule.xls', xlsRowsToXml(rows))
}

onMounted(() => {
  store.loadSolidify()
  const query = window.matchMedia('(max-width: 767px)')
  const apply = () => {
    isMobile.value = query.matches
  }
  apply()
  query.addEventListener('change', apply)
  document.addEventListener('pointerdown', handleExportOutsidePointerDown)
  document.addEventListener('keydown', handleExportKeydown)
  void checkDataOutdated()
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleExportOutsidePointerDown)
  document.removeEventListener('keydown', handleExportKeydown)
})
</script>

<template>
  <div class="pb-12">
    <PageHeader :title="t('schedule.title')" :description="t('schedule.subtitle')">
      <template #actions>
        <div class="flex flex-wrap items-center gap-2">
          <button
            v-if="dataOutdated"
            type="button"
            class="gf-button gf-button-md gf-button-primary"
            @click="syncLatest"
          >
            <RefreshCw class="h-4 w-4" />
            {{ t('schedule.syncLatest') }}
          </button>
          <div ref="exportRoot" class="relative">
            <button
              ref="exportButton"
              type="button"
              class="gf-button gf-button-md gf-button-outline"
              :aria-expanded="exportOpen"
              aria-haspopup="menu"
              @click="toggleExportMenu"
            >
              <Download class="h-4 w-4" />
              {{ t('schedule.export') }}
            </button>
            <Transition name="gf-menu">
              <div
                v-if="exportOpen"
                ref="exportMenu"
                class="gf-menu-surface absolute right-0 top-[calc(100%+0.375rem)] z-30 w-48 p-1"
              >
                <button type="button" class="gf-menu-item w-full" @click="exportCsv">
                  {{ t('schedule.exportCsv') }}
                </button>
                <button type="button" class="gf-menu-item w-full" @click="exportXls">
                  {{ t('schedule.exportXls') }}
                </button>
              </div>
            </Transition>
          </div>
        </div>
      </template>
    </PageHeader>

    <!-- 移动端：三 tab（课表/选课/详情），方案条置顶 -->
    <div v-if="isMobile" class="mt-4 space-y-3">
      <SchedulePlanBar />
      <ScheduleMajorSelector />
      <ScheduleStatsCard />

      <div role="tablist" aria-label="schedule tabs" class="flex gap-1 rounded-lg border border-line/60 bg-base-200/40 p-1">
        <button
          v-for="tab in ([
            { key: 'timetable', label: t('schedule.timetable') },
            { key: 'list', label: t('schedule.pickCourses') },
            { key: 'detail', label: t('schedule.detail') },
          ] as const)"
          :key="tab.key"
          type="button"
          role="tab"
          :aria-selected="mobileTab === tab.key"
          class="gf-tab flex-1"
          :class="mobileTab === tab.key ? 'gf-tab-active' : 'gf-tab-idle'"
          @click="mobileTab = tab.key"
          @keydown="handleMobileTabKeydown"
        >
          {{ tab.label }}
        </button>
      </div>

      <ScheduleTimeTable
        v-if="mobileTab === 'timetable'"
        @open-detail="handleOpenDetail"
        @cell-click="handleCellClick"
        @customize="customizeOpen = true"
      />
      <ScheduleRoughList v-if="mobileTab === 'list'" @open-picker="pickerOpen = true" />
      <ScheduleDetailList
        v-if="mobileTab === 'detail'"
        @conflict="handleConflict"
        @staged="flash(t('schedule.stagedSuccess'), 'success')"
      />
    </div>

    <!-- 桌面：左右两栏（左：方案/专业/统计/已选列表；右：课表） -->
    <div v-else class="mt-4 grid items-start gap-4 lg:grid-cols-[minmax(320px,360px)_1fr]">
      <div class="space-y-3">
        <SchedulePlanBar />
        <ScheduleMajorSelector />
        <ScheduleStatsCard />
        <ScheduleRoughList @open-picker="pickerOpen = true" />
        <ScheduleDetailList @conflict="handleConflict" />
      </div>
      <ScheduleTimeTable @open-detail="handleOpenDetail" @cell-click="handleCellClick" @customize="customizeOpen = true" />
    </div>

    <ScheduleCoursePicker :open="pickerOpen" @close="pickerOpen = false" />
    <ScheduleCellPicker
      :open="cellPick !== null"
      :day="cellPick?.day ?? null"
      :section="cellPick?.section ?? null"
      @close="cellPick = null"
      @conflict="handleConflict"
      @staged="flash(t('schedule.stagedSuccess'), 'success')"
    />
    <ScheduleCustomEventDialog :open="customizeOpen" @close="customizeOpen = false" />
    <ScheduleDetailCard :course="detailCourse" @close="detailCourse = null" />
  </div>
</template>

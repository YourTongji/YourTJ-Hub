<script setup lang="ts">
// 排课器主页面（/schedule，course.schedule）—— v2 布局。
// 桌面左右两栏同高（grid stretch）：左栏（方案条 / 学期·年级·专业 / 统计卡 /
// 已选课程列表）内部滚动；右栏课表为主视觉。点击课程弹出浮动「选择教学班」
// 弹窗（取代内联班级列），避免左右栏高度互相牵制。移动端双 tab（课表/选课）。
// 数据全部走 /api/pk/* JSON API 异步加载（SSR 空壳）；localStorage 持久化由 store 负责。
// 数据过期提示 + 「同步最新」：P11 latest-update 对比本地 updateTime，P12 course-info-sync
// 以全方案课程并集请求，applySyncToAllPlans 各方案保留排课状态。
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { Download, RefreshCw, X } from '@lucide/vue'
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
import { fetchPage } from '@/runtime/router'
import { createSectionTimesRefresher } from '@/site/utils/sectionTimesRefresh'
import type { LayoutPayload, SchedulePageProps } from '@gooseforum/client'
import type { PkConflictItem } from '@/site/utils/pkConflict'
import type { PkCourseDetail, PkCourseOnTable } from '@/site/types/pk'

const pageProps = defineProps<{
  layout: LayoutPayload
  props: SchedulePageProps
}>()

const { t } = useI18n()
const store = useScheduleStore()

const isMobile = ref(false)
const mobileTab = ref<'timetable' | 'list'>('timetable')

const MOBILE_TABS = [
  { key: 'timetable', labelKey: 'schedule.timetable' },
  { key: 'list', labelKey: 'schedule.pickCourses' },
] as const

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
/** 点击课程 → 浮动「选择教学班」弹窗（内容 = ScheduleDetailList）。 */
const classPickOpen = ref(false)
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

/** 作息表静默刷新器：管理端保存新节次作息后，回到本页（bfcache/切前台）自动同步。 */
const sectionTimesRefresher = createSectionTimesRefresher(
  {
    fetchPayload: () => fetchPage(new URL(window.location.href)),
    apply: (times) => store.setSectionTimeOverrides(times),
  },
  { initial: pageProps.props.sectionTimes ?? [] },
)

/** bfcache 恢复（pageshow.persisted）→ 静默刷新作息表。 */
function handlePageShow(event: PageTransitionEvent) {
  if (event.persisted) void sectionTimesRefresher.refresh()
}

/** 切回前台（visibilitychange visible）→ 静默刷新作息表。 */
function handleVisibilityChange() {
  if (document.visibilityState === 'visible') void sectionTimesRefresher.refresh()
}

onMounted(() => {
  store.loadSolidify()
  // 后台作息覆盖（SSR props 注入；未配置为 undefined → 空数组走默认表）。
  store.setSectionTimeOverrides(pageProps.props.sectionTimes ?? [])
  const query = window.matchMedia('(max-width: 767px)')
  const apply = () => {
    isMobile.value = query.matches
  }
  apply()
  query.addEventListener('change', apply)
  document.addEventListener('pointerdown', handleExportOutsidePointerDown)
  document.addEventListener('keydown', handleExportKeydown)
  window.addEventListener('pageshow', handlePageShow)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  void checkDataOutdated()
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleExportOutsidePointerDown)
  document.removeEventListener('keydown', handleExportKeydown)
  window.removeEventListener('pageshow', handlePageShow)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  sectionTimesRefresher.dispose()
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

    <!-- 移动端：双 tab（课表/选课），方案条置顶；教学班选择走弹窗 -->
    <div v-if="isMobile" class="mt-4 space-y-3">
      <SchedulePlanBar />
      <ScheduleMajorSelector />
      <ScheduleStatsCard />

      <div role="tablist" aria-label="schedule tabs" class="flex gap-1 rounded-lg border border-line/60 bg-base-200/40 p-1">
        <button
          v-for="tab in MOBILE_TABS"
          :key="tab.key"
          type="button"
          role="tab"
          :aria-selected="mobileTab === tab.key"
          class="gf-tab flex-1"
          :class="mobileTab === tab.key ? 'gf-tab-active' : 'gf-tab-idle'"
          @click="mobileTab = tab.key"
          @keydown="handleMobileTabKeydown"
        >
          {{ t(tab.labelKey) }}
        </button>
      </div>

      <ScheduleTimeTable
        v-if="mobileTab === 'timetable'"
        @open-detail="handleOpenDetail"
        @cell-click="handleCellClick"
        @customize="customizeOpen = true"
      />
      <ScheduleRoughList
        v-if="mobileTab === 'list'"
        @open-picker="pickerOpen = true"
        @open-detail="classPickOpen = true"
      />
    </div>

    <!-- 桌面：左右两栏同高——右栏课表自然高度定容器高，左栏绝对定位拉满并内部滚动 -->
    <div v-else class="mt-4 lg:relative">
      <div class="flex flex-col gap-3 lg:absolute lg:inset-y-0 lg:left-0 lg:w-[352px] lg:min-h-0">
        <SchedulePlanBar />
        <ScheduleMajorSelector />
        <ScheduleStatsCard />
        <ScheduleRoughList class="min-h-0 lg:flex-1" @open-picker="pickerOpen = true" @open-detail="classPickOpen = true" />
      </div>
      <!-- 右栏保底高度：空课表（未选课/全退）时仍保留整张课表的高度，
           避免绝对定位的左栏（方案/选择器/统计/列表）被压扁。 -->
      <div class="min-w-0 lg:min-h-[720px] lg:pl-[368px]">
        <ScheduleTimeTable @open-detail="handleOpenDetail" @cell-click="handleCellClick" @customize="customizeOpen = true" />
      </div>
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

    <!-- 选择教学班弹窗：点击已选课程弹出（浮动，取代内联班级列） -->
    <DialogRoot :open="classPickOpen" @update:open="classPickOpen = $event">
      <DialogPortal>
        <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
        <DialogContent
          class="fixed left-1/2 top-1/2 z-[2100] max-h-[85vh] w-[92vw] max-w-[520px] -translate-x-1/2 -translate-y-1/2 overflow-y-auto outline-none lg:max-w-[880px]"
        >
          <div class="overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl">
            <div class="flex items-start justify-between gap-2 border-b border-line/60 px-4 py-3">
              <div class="min-w-0">
                <DialogTitle class="truncate text-sm font-bold text-base-content">
                  {{ store.state.clickedCourseInfo.courseName || t('schedule.classPickerTitle') }}
                </DialogTitle>
                <DialogDescription class="text-[11px] text-base-content/55">{{ t('schedule.classPickerHint') }}</DialogDescription>
              </div>
              <button type="button" class="gf-icon-button shrink-0" :aria-label="t('common.close')" @click="classPickOpen = false">
                <X class="h-4 w-4" />
              </button>
            </div>
            <div class="max-h-[calc(85vh-64px)] overflow-y-auto overscroll-contain">
              <ScheduleDetailList @conflict="handleConflict" @staged="flash(t('schedule.stagedSuccess'), 'success')" />
            </div>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </div>
</template>

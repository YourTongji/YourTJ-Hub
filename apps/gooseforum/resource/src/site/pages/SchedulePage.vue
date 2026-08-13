<script setup lang="ts">
// 排课器主页面（/schedule，course.schedule）。
// 桌面双栏（选课列表 + 课程班级）+ 下方课表；移动端三 tab（课表/选课/详情）。
// 数据全部走 /api/pk/* JSON API 异步加载（SSR 空壳）；localStorage 持久化由 store 负责。
// 数据过期提示 + 「同步最新」：P11 latest-update 对比本地 updateTime，P12 course-info-sync 增量保留已选。
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download, RefreshCw } from '@lucide/vue'
import PageHeader from '@/site/components/PageHeader.vue'
import ScheduleMajorSelector from '@/site/components/schedule/ScheduleMajorSelector.vue'
import ScheduleRoughList from '@/site/components/schedule/ScheduleRoughList.vue'
import ScheduleDetailList from '@/site/components/schedule/ScheduleDetailList.vue'
import ScheduleTimeTable from '@/site/components/schedule/ScheduleTimeTable.vue'
import ScheduleCoursePicker from '@/site/components/schedule/ScheduleCoursePicker.vue'
import ScheduleConflictDialog from '@/site/components/schedule/ScheduleConflictDialog.vue'
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

const pickerOpen = ref(false)
const conflictDetail = ref<PkCourseDetail | null>(null)
const conflictList = ref<PkConflictItem[]>([])
const detailCourse = ref<PkCourseOnTable | null>(null)

const dataOutdated = computed(() => store.state.flags.isDataOutdated)

function flash(message: string, type: 'success' | 'error' | 'warning' | 'info' = 'info') {
  queueFlashMessage(message, type)
}

// ---- 数据过期检查与同步 ----
async function checkDataOutdated() {
  store.loadSolidifyTime()
  try {
    const latest = await getPkLatestUpdate()
    const latestSyncAt = latest.latestSyncAt ?? ''
    store.setLatestUpdateTime(latestSyncAt)
    if (store.state.updateTime === '') {
      // 首次进入且已有已选课程：不清空用户数据，仅记录同步时间（防数据丢失）。
      if (store.state.commonLists.selectedCourses.length === 0) {
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

async function syncLatest() {
  if (syncing.value) return // 防重入
  const calendarId = store.state.majorSelected.calendarId
  const staged = store.state.commonLists.stagedCourses
  if (calendarId === undefined || staged.length === 0) {
    store.syncLatestData()
    flash(t('schedule.syncSuccess'), 'success')
    return
  }

  const isExclusiveCourse = (course: (typeof staged)[number]) =>
    course.courseDetail.some((detail) => detail.isExclusive === true)
  const grade = store.state.majorSelected.grade
  const major = store.state.majorSelected.major

  syncing.value = true
  try {
    const result = await syncPkCourseInfo({
      calendarId,
      majorCourseCodes: staged.filter(isExclusiveCourse).map((c) => c.courseCode),
      otherCourseCodes: staged.filter((c) => !isExclusiveCourse(c)).map((c) => c.courseCode),
      majorInfo: { grade: grade ?? 0, code: major ?? '' },
    })

    // 用最新详情替换，保留用户选择的班级状态（增量保留已选）。
    const newStaged = staged.map((course) => {
      const details = result[course.courseCode]
      if (!details || details.length === 0) return course
      return {
        ...course,
        courseDetail: details.map((detail) => {
          const old = course.courseDetail.find((o) => o.code === detail.code)
          return { ...detail, status: old?.status ?? 0 }
        }),
      }
    })
    store.applySyncedCourses(newStaged)
    flash(t('schedule.syncSuccess'), 'success')
  } catch (err) {
    flash(err instanceof Error ? err.message : t('schedule.loadFailed'), 'error')
  } finally {
    syncing.value = false
  }
}

// ---- 冲突处理 ----
function handleConflict(detail: PkCourseDetail, conflicts: PkConflictItem[]) {
  conflictDetail.value = detail
  conflictList.value = conflicts
}

function handleReplaced() {
  flash(t('schedule.replaced'), 'success')
}

// ---- 课表点击 ----
function handleOpenDetail(course: PkCourseOnTable) {
  detailCourse.value = course
}

function handleCellClick(_day: number, _section: number) {
  // 点击课表空格：引导用户通过「选择课程」添加课程（时段查课端点 P10 属 #187）。
  flash(t('schedule.empty'), 'info')
}

// ---- 导出（与课表一致：含已排入课表的所有班级）----
const exportOpen = ref(false)
const exportRoot = ref<HTMLElement | null>(null)

function closeExportMenu() {
  exportOpen.value = false
}

function handleExportOutsidePointerDown(event: PointerEvent) {
  const target = event.target
  if (target instanceof Node && exportRoot.value?.contains(target)) return
  exportOpen.value = false
}

function handleExportKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') exportOpen.value = false
}

function exportableClassCodes(): string[] {
  const codes: string[] = []
  for (const course of store.state.commonLists.stagedCourses) {
    for (const detail of course.courseDetail) {
      if (detail.status === 1 || detail.status === 2) codes.push(detail.code)
    }
  }
  return codes
}

function exportCsv() {
  closeExportMenu()
  const codes = exportableClassCodes()
  if (codes.length === 0) {
    flash(t('schedule.empty'), 'warning')
    return
  }
  const rows = codesToCsvRows(codes, store.state.commonLists.stagedCourses)
  downloadCsv('yourtj-schedule.csv', jsonToCsv(rows))
}

function exportXls() {
  closeExportMenu()
  const codes = exportableClassCodes()
  if (codes.length === 0) {
    flash(t('schedule.empty'), 'warning')
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
  void checkDataOutdated()
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleExportOutsidePointerDown)
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
            class="gf-button gf-button-md gf-button-danger"
            @click="syncLatest"
          >
            <RefreshCw class="h-4 w-4" />
            {{ t('schedule.syncLatest') }}
          </button>
          <div ref="exportRoot" class="relative">
            <button
              type="button"
              class="gf-button gf-button-md gf-button-outline"
              :aria-expanded="exportOpen"
              @click="exportOpen = !exportOpen"
              @keydown="handleExportKeydown"
            >
              <Download class="h-4 w-4" />
              {{ t('schedule.export') }}
            </button>
            <Transition name="gf-menu">
              <div
                v-if="exportOpen"
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

    <ScheduleMajorSelector />

    <!-- 移动端：三 tab（课表/选课/详情） -->
    <div v-if="isMobile" class="mt-4 space-y-3">
      <div class="flex gap-1 rounded-lg border border-line/60 bg-base-200/40 p-1">
        <button
          v-for="tab in ([
            { key: 'timetable', label: t('schedule.timetable') },
            { key: 'list', label: t('schedule.pickCourses') },
            { key: 'detail', label: t('schedule.detail') },
          ] as const)"
          :key="tab.key"
          type="button"
          class="gf-tab flex-1"
          :class="mobileTab === tab.key ? 'gf-tab-active' : 'gf-tab-idle'"
          @click="mobileTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <ScheduleTimeTable
        v-if="mobileTab === 'timetable'"
        @open-detail="handleOpenDetail"
        @cell-click="handleCellClick"
      />
      <ScheduleRoughList v-if="mobileTab === 'list'" @open-picker="pickerOpen = true" />
      <ScheduleDetailList
        v-if="mobileTab === 'detail'"
        @conflict="handleConflict"
        @staged="flash(t('schedule.syncSuccess'), 'success')"
      />
    </div>

    <!-- 桌面：双栏 + 下方课表 -->
    <div v-else class="mt-4 space-y-4">
      <div class="grid gap-4 lg:grid-cols-5">
        <div class="lg:col-span-2">
          <ScheduleRoughList @open-picker="pickerOpen = true" />
        </div>
        <div class="lg:col-span-3">
          <ScheduleDetailList @conflict="handleConflict" />
        </div>
      </div>
      <ScheduleTimeTable @open-detail="handleOpenDetail" @cell-click="handleCellClick" />
    </div>

    <ScheduleCoursePicker :open="pickerOpen" @close="pickerOpen = false" />
    <ScheduleConflictDialog
      :detail="conflictDetail"
      :conflicts="conflictList"
      @close="conflictDetail = null"
      @replaced="handleReplaced"
    />
    <ScheduleDetailCard :course="detailCourse" @close="detailCourse = null" />
  </div>
</template>

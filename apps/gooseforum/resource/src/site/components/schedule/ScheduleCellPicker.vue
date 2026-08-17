<script setup lang="ts">
// 课表空白格点击 → 弹出该时段可选课程选择框：
// 从备选池（stagedCourses）筛选所有「该天该节次有课」的教学班，点击直接尝试加入。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, X } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import { useDialogAccessibility } from '@/site/composables/useDialogAccessibility'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkConflictItem } from '@/site/utils/pkConflict'
import type { PkCourseDetail } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const props = defineProps<{
  open: boolean
  day: number | null
  section: number | null
}>()

const emit = defineEmits<{
  close: []
  conflict: [detail: PkCourseDetail, conflicts: PkConflictItem[]]
  staged: []
}>()

const { panelRef } = useDialogAccessibility(computed(() => props.open), {
  onClose: () => emit('close'),
})

const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const

const slotTitle = computed(() => {
  if (props.day === null || props.section === null) return ''
  const weekday = t(`schedule.weekdays.${WEEKDAY_KEYS[props.day - 1]}`)
  return `${weekday} ${t('schedule.sectionLabel', { section: props.section })}`
})

interface CellCandidate {
  courseName: string
  courseCode: string
  credit: number
  detail: PkCourseDetail
}

/** 该时段可排的教学班：教学班任一安排落在点击的「天 + 节次」。 */
const candidates = computed<CellCandidate[]>(() => {
  if (props.day === null || props.section === null) return []
  const list: CellCandidate[] = []
  for (const course of store.state.commonLists.stagedCourses) {
    for (const detail of course.courseDetail) {
      const hits = detail.arrangementInfo.some(
        (arr) => arr.occupyDay === props.day && arr.occupyTime.includes(props.section!),
      )
      if (hits) {
        list.push({
          courseName: course.courseNameReserved || course.courseName || course.courseCode,
          courseCode: course.courseCode,
          credit: course.credit,
          detail,
        })
      }
    }
  }
  return list
})

function arrangementText(detail: PkCourseDetail): string {
  return detail.arrangementInfo.map((arr) => arr.arrangementText).join('；')
}

function teacherText(detail: PkCourseDetail): string {
  return detail.teachers.map((teacher) => teacher.teacherName).filter(Boolean).join('、')
}

function tryStage(candidate: CellCandidate) {
  // 用候选课程填充点击上下文：stageCourse/forceReplaceCourse 依赖
  // clickedCourseInfo 生成课表行与占用格标签，避免暂存后课程名/代码
  // 串成上次打开的课程（review P1）。
  store.setClickedCourseInfo({ courseCode: candidate.courseCode, courseName: candidate.courseName })
  const result = store.stageCourse(candidate.detail)
  if (result.added) {
    store.solidify()
    emit('staged')
    emit('close')
    return
  }
  emit('conflict', candidate.detail, result.conflicts ?? [])
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="gf-fade">
      <div
        v-if="open"
        ref="panelRef"
        role="dialog"
        aria-modal="true"
        aria-labelledby="schedule-cell-pick-title"
        class="fixed inset-0 z-[2100]"
      >
        <div class="absolute inset-0 bg-black/40" @click="emit('close')"></div>
        <div class="absolute left-1/2 top-1/2 max-h-[80vh] w-[88vw] max-w-[440px] -translate-x-1/2 -translate-y-1/2 overflow-y-auto">
          <div class="overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl" @click.stop>
            <div class="flex items-start justify-between gap-2 border-b border-line/60 px-4 py-3">
              <div class="min-w-0">
                <div id="schedule-cell-pick-title" class="text-sm font-bold text-base-content">{{ t('schedule.cellPickTitle') }}</div>
                <div class="text-[11px] text-base-content/55">{{ slotTitle }}</div>
              </div>
              <button type="button" class="gf-icon-button shrink-0" :aria-label="t('common.close')" @click="emit('close')">
                <X class="h-4 w-4" />
              </button>
            </div>

            <EmptyState
              v-if="candidates.length === 0"
              class="p-6"
              :icon="BookOpen"
              :title="t('schedule.cellPickEmpty')"
            />

            <ul v-else class="gf-scrollbar-thin max-h-[50vh] divide-y divide-line/60 overflow-y-auto overscroll-contain">
              <li v-for="candidate in candidates" :key="candidate.detail.code" class="px-4 py-2.5">
                <button type="button" class="w-full text-left" @click="tryStage(candidate)">
                  <span class="block truncate text-[13px] font-medium text-base-content">{{ candidate.courseName }}</span>
                  <span class="mt-0.5 block text-[11px] text-base-content/50">{{ candidate.detail.code }} · {{ t('schedule.credit', { credit: candidate.credit }) }}</span>
                  <p v-if="teacherText(candidate.detail)" class="mt-0.5 text-[12px] text-base-content/60">
                    {{ t('schedule.teacherWith', { value: teacherText(candidate.detail) }) }}
                  </p>
                  <p class="mt-0.5 line-clamp-2 text-[12px] text-base-content/60">
                    {{ arrangementText(candidate.detail) }}
                  </p>
                </button>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
// 课程班级列表：展示 clickedCourseInfo 对应课程的全部教学班，点击班级尝试加入课表。
// 无冲突直接加入（status → 备选）；有冲突 emit('conflict') 由父级弹窗决定「强制替换/放弃」。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkConflictItem } from '@/site/utils/pkConflict'
import type { PkCourseDetail } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const emit = defineEmits<{
  conflict: [detail: PkCourseDetail, conflicts: PkConflictItem[]]
  staged: []
}>()

const currentCourse = computed(() =>
  store.state.commonLists.stagedCourses.find(
    (course) => course.courseCode === store.state.clickedCourseInfo.courseCode,
  ),
)

function statusLabel(status: number | undefined): string {
  if (status === 2) return t('schedule.statusSelected')
  if (status === 1) return t('schedule.statusStaged')
  return t('schedule.statusUnselected')
}

function statusClass(status: number | undefined): string {
  if (status === 2) return 'gf-badge gf-badge-success'
  if (status === 1) return 'gf-badge gf-badge-warning'
  return 'gf-badge gf-badge-muted'
}

function arrangementText(detail: PkCourseDetail): string {
  return detail.arrangementInfo.map((arr) => arr.arrangementText).join('；')
}

function teacherText(detail: PkCourseDetail): string {
  return detail.teachers.map((teacher) => teacher.teacherName).filter(Boolean).join('、')
}

function tryStage(detail: PkCourseDetail) {
  const result = store.stageCourse(detail)
  if (result.added) {
    store.solidify()
    emit('staged')
    return
  }
  emit('conflict', detail, result.conflicts ?? [])
}
</script>

<template>
  <div class="space-y-3">
    <EmptyState
      v-if="!currentCourse"
      class="gf-panel"
      :icon="BookOpen"
      :title="t('schedule.empty')"
      :description="t('schedule.majorHint')"
    />

    <div v-else class="gf-panel">
      <div class="border-b border-line/60 px-3 py-2">
        <h3 class="truncate text-[13px] font-bold text-base-content">
          {{ currentCourse.courseNameReserved }}
        </h3>
        <p class="text-[11px] text-base-content/50">{{ currentCourse.courseCode }} · {{ t('schedule.credit', { credit: currentCourse.credit }) }}</p>
      </div>
      <ul v-if="currentCourse.courseDetail.length" class="divide-y divide-line/60">
        <li
          v-for="detail in currentCourse.courseDetail"
          :key="detail.code"
          class="px-3 py-2"
        >
          <button
            type="button"
            class="w-full text-left"
            @click="tryStage(detail)"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="min-w-0 truncate text-[13px] font-medium text-base-content">{{ detail.code }}</span>
              <span :class="statusClass(detail.status)">{{ statusLabel(detail.status) }}</span>
            </div>
            <p v-if="teacherText(detail)" class="mt-0.5 text-[12px] text-base-content/60">
              {{ t('schedule.teacherWith', { value: teacherText(detail) }) }}
            </p>
            <p class="mt-0.5 line-clamp-2 text-[12px] text-base-content/60">
              {{ arrangementText(detail) }}
            </p>
            <p class="mt-0.5 text-[11px] text-base-content/45">
              {{ detail.campus }} · {{ detail.teachingLanguage }}
              <template v-if="detail.isExclusive"> · {{ t('schedule.tabRequired') }}</template>
            </p>
          </button>
        </li>
      </ul>
      <p v-else class="px-3 py-3 text-[12px] text-base-content/50">{{ t('schedule.empty') }}</p>
    </div>
  </div>
</template>

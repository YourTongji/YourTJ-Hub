<script setup lang="ts">
// 选课列表/备选池：展示 stagedCourses，提供「选择课程」「保存课表」与退课/清除操作。
// 点击课程行会把该课设为 clickedCourseInfo，右侧/详情 tab 展示其班级。
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, Save } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import { useDialog } from '@/site/composables/useDialog'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkStagedCourse } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const pendingDrop = ref<PkStagedCourse | null>(null)

const emit = defineEmits<{
  openPicker: []
}>()

const dropDialogOpen = computed(() => pendingDrop.value !== null)
const { dialogRef: dropDialogRef, closeDialog: closeDropDialog } = useDialog({ visible: dropDialogOpen })

function statusLabel(course: PkStagedCourse): string {
  if (course.status === 2) return t('schedule.statusSelected')
  if (course.status === 1) return t('schedule.statusStaged')
  return t('schedule.statusUnselected')
}

function statusClass(course: PkStagedCourse): string {
  if (course.status === 2) return 'gf-badge gf-badge-success'
  if (course.status === 1) return 'gf-badge gf-badge-warning'
  return 'gf-badge gf-badge-muted'
}

function selectCourse(course: PkStagedCourse) {
  store.setClickedCourseInfo({
    courseCode: course.courseCode,
    courseName: course.courseNameReserved,
  })
}

function dropCourse(course: PkStagedCourse) {
  pendingDrop.value = course
}

function confirmDrop() {
  if (!pendingDrop.value) return
  store.popStagedCourse(pendingDrop.value.courseCode)
  closeDropDialog()
}

function saveTimetable() {
  store.saveSelectedCourses()
  store.solidify()
}

/** 课表里已排入的班级数（用于展示）。 */
function arrangedClassCount(course: PkStagedCourse): number {
  return course.courseDetail.filter((d) => d.status === 2).length
}
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center gap-2">
      <button type="button" class="gf-button gf-button-md gf-button-primary" @click="emit('openPicker')">
        {{ t('schedule.openPicker') }}
      </button>
      <button type="button" class="gf-button gf-button-md gf-button-outline" @click="saveTimetable">
        <Save class="h-4 w-4" />
        {{ t('schedule.saveTimetable') }}
      </button>
    </div>

    <EmptyState
      v-if="!store.state.commonLists.stagedCourses.length"
      class="gf-panel"
      :icon="BookOpen"
      :title="t('schedule.empty')"
    />

    <ul v-else class="gf-panel divide-y divide-line/60">
      <li
        v-for="course in store.state.commonLists.stagedCourses"
        :key="course.courseCode"
        class="flex items-center gap-2 px-3 py-2"
      >
        <button
          type="button"
          class="min-w-0 flex-1 text-left"
          @click="selectCourse(course)"
        >
          <span class="block truncate text-[13px] font-medium text-base-content">
            {{ course.courseNameReserved }}
          </span>
          <span class="block truncate text-[11px] text-base-content/50">
            {{ course.courseCode }} · {{ t('schedule.credit', { credit: course.credit }) }}
            <template v-if="arrangedClassCount(course) > 0"> · {{ t('schedule.arrangedCount', { count: arrangedClassCount(course) }) }}</template>
          </span>
        </button>
        <span :class="statusClass(course)">{{ statusLabel(course) }}</span>
        <button
          type="button"
          class="gf-button gf-button-sm gf-button-ghost shrink-0"
          @click="dropCourse(course)"
        >
          {{ course.status === 2 ? t('schedule.dropCourse') : t('schedule.clear') }}
        </button>
      </li>
    </ul>

    <!-- 退课确认弹窗 -->
    <Teleport to="body">
      <Transition name="gf-modal">
        <div
          v-if="pendingDrop"
          ref="dropDialogRef"
          class="fixed inset-0 z-[2100] overflow-y-auto bg-black/40 p-2 sm:p-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="schedule-drop-title"
          @click.self="closeDropDialog"
        >
          <div class="mx-auto flex min-h-full w-full max-w-[360px] items-center justify-center">
            <div class="w-full rounded-2xl border border-line/70 bg-base-100 p-5 shadow-lg" @click.stop>
              <h3 id="schedule-drop-title" class="text-sm font-bold text-base-content">{{ t('schedule.dropCourse') }}</h3>
              <p class="mt-2 text-[13px] text-base-content/70">
                {{ pendingDrop.courseNameReserved }}（{{ pendingDrop.courseCode }}）
              </p>
              <div class="mt-4 flex justify-end gap-2">
                <button type="button" class="gf-button gf-button-md gf-button-ghost" @click="closeDropDialog">
                  {{ t('schedule.cancel') }}
                </button>
                <button type="button" class="gf-button gf-button-md gf-button-danger" @click="confirmDrop">
                  {{ t('schedule.dropCourse') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

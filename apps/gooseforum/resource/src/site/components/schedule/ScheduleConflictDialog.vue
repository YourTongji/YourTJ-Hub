<script setup lang="ts">
// 冲突处理弹窗（验收标准 2）：加入课表遇冲突时，展示冲突课程并让用户选择
// 「强制替换」（移除冲突课再加入）或「放弃」。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { AlertTriangle } from '@lucide/vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkConflictItem } from '@/site/utils/pkConflict'
import type { PkCourseDetail } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const props = defineProps<{
  detail: PkCourseDetail | null
  conflicts: PkConflictItem[]
}>()

const emit = defineEmits<{
  close: []
  replaced: []
}>()

const conflictDialogOpen = computed({
  get: () => props.detail !== null,
  set: (open: boolean) => {
    if (!open) emit('close')
  },
})

function forceReplace() {
  if (!props.detail) return
  store.forceReplaceCourse(props.detail)
  store.solidify()
  emit('replaced')
  emit('close')
}
</script>

<template>
  <DialogRoot v-model:open="conflictDialogOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2100] w-[88vw] max-w-[400px] -translate-x-1/2 -translate-y-1/2 outline-none"
      >
        <div class="rounded-2xl border border-line/70 bg-base-100 p-5 shadow-lg">
          <div class="flex items-center gap-2">
            <AlertTriangle class="h-5 w-5 text-warning" />
            <DialogTitle class="text-sm font-bold text-base-content">{{ t('schedule.conflictTitle') }}</DialogTitle>
          </div>

          <DialogDescription class="mt-3 text-[13px] text-base-content/70">
            {{ detail?.code }}
          </DialogDescription>

          <ul class="mt-2 max-h-56 space-y-1.5 overflow-y-auto rounded-lg border border-warning/25 bg-warning/10 p-3">
            <li v-for="conflict in conflicts" :key="conflict.code" class="text-[13px] text-base-content/80">
              <span class="font-medium">{{ conflict.courseName }}</span>
              <span class="text-base-content/50">（{{ conflict.code }}）</span>
            </li>
          </ul>

          <p class="mt-3 text-[12px] text-base-content/55">
            {{ conflicts.length > 1
      ? t('schedule.conflictWithMany', { course: conflicts[0]?.courseName ?? '', count: conflicts.length })
      : t('schedule.conflictWith', { course: conflicts[0]?.courseName ?? '' }) }}
          </p>

          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-ghost" @click="emit('close')">
              {{ t('schedule.abandon') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-danger" @click="forceReplace">
              {{ t('schedule.forceReplace') }}
            </button>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

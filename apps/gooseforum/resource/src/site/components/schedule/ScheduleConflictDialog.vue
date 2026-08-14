<script setup lang="ts">
// 冲突处理弹窗（验收标准 2）：加入课表遇冲突时，展示冲突课程并让用户选择
// 「强制替换」（移除冲突课再加入）或「放弃」。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogAccessibility } from '@/site/composables/useDialogAccessibility'
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

const { panelRef } = useDialogAccessibility(computed(() => props.detail !== null), {
  onClose: () => emit('close'),
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
  <Teleport to="body">
    <Transition name="gf-fade">
      <div
        v-if="detail"
        ref="panelRef"
        role="dialog"
        aria-modal="true"
        aria-labelledby="schedule-conflict-title"
        class="fixed inset-0 z-[2100]"
      >
        <div class="absolute inset-0 bg-black/40" @click="emit('close')"></div>
        <div class="absolute left-1/2 top-1/2 w-[88vw] max-w-[400px] -translate-x-1/2 -translate-y-1/2">
          <div class="rounded-2xl border border-line/70 bg-base-100 p-5 shadow-lg" @click.stop>
            <div class="flex items-center gap-2">
              <AlertTriangle class="h-5 w-5 text-warning" />
              <h3 id="schedule-conflict-title" class="text-sm font-bold text-base-content">{{ t('schedule.conflictTitle') }}</h3>
            </div>

            <p class="mt-3 text-[13px] text-base-content/70">
              {{ detail.code }}
            </p>

            <ul class="mt-2 max-h-56 space-y-1.5 overflow-y-auto rounded-lg border border-warning/25 bg-warning/10 p-3">
              <li v-for="conflict in conflicts" :key="conflict.code" class="text-[13px] text-base-content/80">
                <span class="font-medium">{{ conflict.courseName }}</span>
                <span class="text-base-content/50">（{{ conflict.code }}）</span>
              </li>
            </ul>

            <p class="mt-3 text-[12px] text-base-content/55">
              {{ t('schedule.conflictWith', { course: conflicts[0]?.courseName ?? '' }) }}
            </p>

            <div class="mt-4 flex justify-end gap-2">
              <button type="button" class="gf-button gf-button-md gf-button-ghost" @click="emit('close')">
                {{ t('schedule.abandon') }}
              </button>
              <button type="button" class="gf-button gf-button-md gf-button-primary" @click="forceReplace">
                {{ t('schedule.forceReplace') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

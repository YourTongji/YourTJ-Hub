<script setup lang="ts">
// 课表冲突提示弹窗：导出前拦截并展示时间冲突课程，引导用户先处理冲突，亦可确认继续导出。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, ArrowRight, CalendarX2, X } from '@lucide/vue'
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { conflictBaseOf, type PkConflictItem } from '@/site/utils/pkConflict'

const { t } = useI18n()
const store = useScheduleStore()

const props = defineProps<{
  open: boolean
  conflicts: Map<string, PkConflictItem[]>
}>()

const emit = defineEmits<{
  close: []
  resolve: []
  continueExport: []
}>()

const dialogOpen = computed({
  get: () => props.open,
  set: (val: boolean) => {
    if (!val) emit('close')
  },
})

function resolveCourseName(baseCode: string): string {
  const plan = store.state.plans.find((p) => p.id === store.state.activePlanId)
  const staged = plan?.stagedCourses.find((c) => c.courseCode === baseCode)
  if (staged?.courseNameReserved || staged?.courseName) {
    return staged.courseNameReserved || staged.courseName
  }
  const onTable = store.state.timeTableData.find((c) => conflictBaseOf(c.code) === baseCode)
  if (onTable?.courseName) {
    return onTable.courseName
  }
  return baseCode
}

/** 提取去重后的冲突摘要列表 */
const conflictSummaryList = computed(() => {
  const items: Array<{ baseCode: string; mainName: string; clashesWith: string[] }> = []
  for (const [baseCode, clashItems] of props.conflicts.entries()) {
    if (!clashItems || clashItems.length === 0) continue
    const mainName = resolveCourseName(baseCode)
    const clashesWith = clashItems.map((c) => c.courseName).filter(Boolean)
    items.push({
      baseCode,
      mainName,
      clashesWith: [...new Set(clashesWith)],
    })
  }
  return items
})

const totalConflictCount = computed(() => conflictSummaryList.value.length)
</script>

<template>
  <DialogRoot v-model:open="dialogOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2000] bg-black/45 backdrop-blur-xs animate-in fade-in duration-200" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2001] w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-line/80 bg-base-100 p-5 shadow-2xl outline-none animate-in fade-in zoom-in-95 duration-200"
      >
        <!-- 弹窗头部 -->
        <div class="flex items-start justify-between gap-3">
          <div class="flex items-center gap-2.5">
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-warning/15 text-warning border border-warning/25">
              <AlertTriangle class="h-5 w-5" />
            </div>
            <div>
              <DialogTitle class="text-base font-bold text-base-content tracking-tight">
                {{ t('schedule.exportConflictWarningTitle') }}
              </DialogTitle>
              <DialogDescription class="mt-0.5 text-xs text-base-content/65 leading-relaxed">
                {{ t('schedule.exportConflictWarningDesc', { count: totalConflictCount }) }}
              </DialogDescription>
            </div>
          </div>
          <DialogClose
            class="rounded-lg p-1 text-base-content/45 hover:bg-base-200 hover:text-base-content transition-colors"
            :aria-label="t('common.close')"
          >
            <X class="h-4 w-4" />
          </DialogClose>
        </div>

        <!-- 冲突条目列表 -->
        <div class="mt-4 max-h-48 overflow-y-auto space-y-2 pr-1 scrollbar-thin">
          <div
            v-for="item in conflictSummaryList"
            :key="item.baseCode"
            class="rounded-xl border border-warning/25 bg-warning/5 p-2.5 text-xs"
          >
            <div class="flex items-center gap-1.5 font-bold text-base-content">
              <CalendarX2 class="h-3.5 w-3.5 text-warning shrink-0" />
              <span class="truncate">{{ item.mainName }}</span>
            </div>
            <div class="mt-1 flex items-start gap-1.5 text-[11px] text-base-content/75 pl-5">
              <span class="opacity-60 shrink-0">{{ t('schedule.cardConflictHint') }}</span>
              <span class="font-medium text-warning-content break-words">{{ item.clashesWith.join('、') }}</span>
            </div>
          </div>
        </div>

        <!-- 底部操作按钮 -->
        <div class="mt-5 flex items-center justify-end gap-2.5 pt-2 border-t border-line/50">
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-ghost text-xs text-base-content/65 hover:text-base-content"
            @click="emit('continueExport')"
          >
            {{ t('schedule.exportConflictContinue') }}
          </button>
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-primary text-xs shadow-sm font-semibold transition-transform active:scale-[0.96]"
            @click="emit('resolve')"
          >
            <ArrowRight class="h-3.5 w-3.5" />
            {{ t('schedule.exportConflictResolve') }}
          </button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

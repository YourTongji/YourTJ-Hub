<script setup lang="ts">
// 云同步冲突弹窗（#537）：进页发现本地与云端排课方案分歧时弹出，
// 二选一「使用云端」（整包采用，不回灌上传）或「保留本地」（立即 PUT 覆盖云端）。
// 一次性决策；关闭弹窗 = 暂缓（下次进页仍分歧时再提示）。状态源为 scheduleSync
// 单例（对齐 ScheduleConfigSection 直接消费 store 的惯例），页面零 props 挂载。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CloudDownload, Laptop, X } from '@lucide/vue'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import { scheduleSync } from '@/site/composables/useScheduleSync'
import { formatDateTime } from '@/runtime/format'
import { useScheduleStore } from '@/site/composables/useScheduleStore'

const { t } = useI18n()
const store = useScheduleStore()

const open = computed(() => scheduleSync.conflict.value !== null)

/** 云端快照摘要（弹窗打开时恒非空）。 */
const cloudSummary = computed(() => {
  const snapshot = scheduleSync.conflict.value
  if (!snapshot) return null
  return {
    planCount: Array.isArray(snapshot.plans) ? snapshot.plans.length : 0,
    updatedAt: formatDateTime(snapshot.updatedAt),
  }
})

/** 本地方案数摘要。 */
const localPlanCount = computed(() => store.state.plans.length)

function handleOpenChange(value: boolean): void {
  if (!value) scheduleSync.dismissConflict()
}
</script>

<template>
  <DialogRoot :open="open" @update:open="handleOpenChange">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2200] bg-black/45 backdrop-blur-xs animate-in fade-in duration-200" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2201] w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-line/80 bg-base-100 p-5 shadow-2xl outline-none animate-in fade-in zoom-in-95 duration-200"
      >
        <!-- 弹窗头部 -->
        <div class="flex items-start justify-between gap-3">
          <div>
            <DialogTitle class="text-base font-bold text-base-content tracking-tight">
              {{ t('schedule.syncConflictTitle') }}
            </DialogTitle>
            <DialogDescription class="mt-0.5 text-xs text-base-content/65 leading-relaxed">
              {{ t('schedule.syncConflictDesc') }}
            </DialogDescription>
          </div>
          <button
            type="button"
            class="rounded-lg p-1 text-base-content/45 hover:bg-base-200 hover:text-base-content transition-colors"
            :aria-label="t('common.close')"
            @click="scheduleSync.dismissConflict()"
          >
            <X class="h-4 w-4" />
          </button>
        </div>

        <!-- 两份快照对比卡片 -->
        <div class="mt-4 grid grid-cols-1 gap-2.5">
          <div class="rounded-xl border border-primary/25 bg-primary/5 p-3 text-xs">
            <div class="flex items-center gap-1.5 font-bold text-base-content">
              <CloudDownload class="h-4 w-4 text-primary shrink-0" />
              <span>{{ t('schedule.syncCloudOption') }}</span>
            </div>
            <p v-if="cloudSummary" class="mt-1 pl-5 text-[11px] text-base-content/65">
              {{ t('schedule.syncCloudMeta', { count: cloudSummary.planCount, time: cloudSummary.updatedAt }) }}
            </p>
          </div>
          <div class="rounded-xl border border-line/70 bg-base-200/40 p-3 text-xs">
            <div class="flex items-center gap-1.5 font-bold text-base-content">
              <Laptop class="h-4 w-4 text-base-content/60 shrink-0" />
              <span>{{ t('schedule.syncLocalOption') }}</span>
            </div>
            <p class="mt-1 pl-5 text-[11px] text-base-content/65">
              {{ t('schedule.syncLocalMeta', { count: localPlanCount }) }}
            </p>
          </div>
        </div>

        <!-- 底部二选一按钮 -->
        <div class="mt-5 flex items-center justify-end gap-2.5 pt-2 border-t border-line/50">
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-outline text-xs"
            @click="scheduleSync.keepLocal()"
          >
            {{ t('schedule.syncKeepLocal') }}
          </button>
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-primary text-xs shadow-sm font-semibold transition-transform active:scale-[0.96]"
            @click="scheduleSync.useCloud()"
          >
            {{ t('schedule.syncUseCloud') }}
          </button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

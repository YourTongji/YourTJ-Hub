<script setup lang="ts">
// 统计卡（v2）：当前方案的已选门数 / 总学分 / 总学时 / 冲突门数。
// 数据来自 store.stats()（deriveConflicts 同判据，仅统计真实课程冲突）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle } from '@lucide/vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'

const { t } = useI18n()
const store = useScheduleStore()

const stats = computed(() => store.stats())
</script>

<template>
  <div class="gf-panel rounded-2xl border border-line/70 p-2 sm:p-2.5 shadow-[0_2px_10px_-4px_rgba(0,0,0,0.05)] grid grid-cols-4 gap-1.5 sm:gap-2">
    <!-- 已选课程 -->
    <div
      class="group relative flex flex-col items-center justify-center rounded-xl border p-2 text-center transition-colors duration-150"
      :class="stats.courseCount > 0 ? 'border-primary/25 bg-primary/[0.04]' : 'border-line/40 bg-base-200/40 hover:bg-base-200/60'"
    >
      <span class="text-[10px] sm:text-[11px] font-medium text-base-content/65 leading-none">
        {{ t('schedule.statsCourses') }}
      </span>
      <div class="mt-1 flex items-baseline justify-center gap-0.5">
        <span
          class="text-base sm:text-lg font-extrabold tabular-nums tracking-tight leading-none"
          :class="stats.courseCount > 0 ? 'text-primary' : 'text-base-content'"
        >
          {{ stats.courseCount }}
        </span>
        <span v-if="t('schedule.statsUnitCourse')" class="text-[10px] font-normal text-base-content/50">
          {{ t('schedule.statsUnitCourse') }}
        </span>
      </div>
    </div>

    <!-- 总学分 -->
    <div class="group relative flex flex-col items-center justify-center rounded-xl border border-line/40 bg-base-200/40 p-2 text-center transition-colors duration-150 hover:bg-base-200/60">
      <span class="text-[10px] sm:text-[11px] font-medium text-base-content/65 leading-none">
        {{ t('schedule.statsCredit') }}
      </span>
      <div class="mt-1 flex items-baseline justify-center gap-0.5">
        <span class="text-base sm:text-lg font-extrabold tabular-nums tracking-tight leading-none text-base-content">
          {{ stats.totalCredit.toFixed(1) }}
        </span>
        <span v-if="t('schedule.statsUnitCredit')" class="text-[10px] font-normal text-base-content/50">
          {{ t('schedule.statsUnitCredit') }}
        </span>
      </div>
    </div>

    <!-- 总学时 -->
    <div class="group relative flex flex-col items-center justify-center rounded-xl border border-line/40 bg-base-200/40 p-2 text-center transition-colors duration-150 hover:bg-base-200/60">
      <span class="text-[10px] sm:text-[11px] font-medium text-base-content/65 leading-none">
        {{ t('schedule.statsHours') }}
      </span>
      <div class="mt-1 flex items-baseline justify-center gap-0.5">
        <span class="text-base sm:text-lg font-extrabold tabular-nums tracking-tight leading-none text-base-content">
          {{ stats.totalHours }}
        </span>
        <span v-if="t('schedule.statsUnitHour')" class="text-[10px] font-normal text-base-content/50">
          {{ t('schedule.statsUnitHour') }}
        </span>
      </div>
    </div>

    <!-- 冲突 -->
    <div
      class="group relative flex flex-col items-center justify-center rounded-xl border p-2 text-center transition-colors duration-150"
      :class="stats.conflictCount > 0 ? 'border-error/35 bg-error/10 text-error shadow-xs' : 'border-line/40 bg-base-200/40 hover:bg-base-200/60'"
    >
      <span
        class="text-[10px] sm:text-[11px] font-medium leading-none"
        :class="stats.conflictCount > 0 ? 'text-error font-semibold' : 'text-base-content/65'"
      >
        {{ t('schedule.statsConflicts') }}
      </span>
      <div class="mt-1 flex items-baseline justify-center gap-0.5">
        <AlertTriangle v-if="stats.conflictCount > 0" class="h-3 w-3 shrink-0 text-error self-center mr-0.5" />
        <span
          class="text-base sm:text-lg font-extrabold tabular-nums tracking-tight leading-none"
          :class="stats.conflictCount > 0 ? 'text-error' : 'text-base-content'"
        >
          {{ stats.conflictCount }}
        </span>
        <span
          v-if="t('schedule.statsUnitConflict')"
          class="text-[10px] font-normal"
          :class="stats.conflictCount > 0 ? 'text-error/70' : 'text-base-content/50'"
        >
          {{ t('schedule.statsUnitConflict') }}
        </span>
      </div>
    </div>
  </div>
</template>

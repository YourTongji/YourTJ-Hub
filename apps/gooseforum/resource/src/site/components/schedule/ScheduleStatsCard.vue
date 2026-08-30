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
  <div class="gf-panel grid grid-cols-2 gap-2 p-3 sm:grid-cols-4">
    <div class="min-w-0">
      <p class="text-[11px] text-base-content/55">{{ t('schedule.statsCourses') }}</p>
      <p class="mt-0.5 truncate text-lg font-bold leading-tight text-base-content">
        {{ t('schedule.statsCourseCount', { count: stats.courseCount }) }}
      </p>
    </div>
    <div class="min-w-0">
      <p class="text-[11px] text-base-content/55">{{ t('schedule.statsCredit') }}</p>
      <p class="mt-0.5 truncate text-lg font-bold leading-tight text-base-content">
        {{ stats.totalCredit.toFixed(1) }}
      </p>
    </div>
    <div class="min-w-0">
      <p class="text-[11px] text-base-content/55">{{ t('schedule.statsHours') }}</p>
      <p class="mt-0.5 truncate text-lg font-bold leading-tight text-base-content">
        {{ stats.totalHours }}
      </p>
    </div>
    <div class="min-w-0">
      <p class="text-[11px] text-base-content/55">{{ t('schedule.statsConflicts') }}</p>
      <p
        class="mt-0.5 flex items-center gap-1 truncate text-lg font-bold leading-tight"
        :class="stats.conflictCount > 0 ? 'text-error' : 'text-base-content'"
      >
        <AlertTriangle v-if="stats.conflictCount > 0" class="h-4 w-4 shrink-0" />
        {{ t('schedule.statsConflictCount', { count: stats.conflictCount }) }}
      </p>
    </div>
  </div>
</template>

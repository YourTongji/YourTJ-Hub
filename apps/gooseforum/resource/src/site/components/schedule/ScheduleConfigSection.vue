<script setup lang="ts">
// 排课器侧栏配置区域：整合方案管理条（SchedulePlanBar）与学期/年级/专业选择器（ScheduleMajorSelector）。
// 支持折叠为单行精致摘要卡片，释放 ~265px 纵向空间，使桌面端备选课程池（ScheduleRoughList）
// 能同时完整呈现 6-8 门课程，提升排课选课效率。
// 状态与 useScheduleStore 及 localStorage（goose:scheduleConfigCollapsed）深度协同，
// 页面切换与刷新严格保持折叠记忆，绝不自动展开。
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown } from '@lucide/vue'
import SchedulePlanBar from '@/site/components/schedule/SchedulePlanBar.vue'
import ScheduleMajorSelector from '@/site/components/schedule/ScheduleMajorSelector.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'

const { t } = useI18n()
const store = useScheduleStore()

const COLLAPSED_STORAGE_KEY = 'goose:scheduleConfigCollapsed'

const isCollapsed = computed({
  get: () => store.state.isConfigCollapsed,
  set: (val: boolean) => store.setConfigCollapsed(val),
})

onMounted(() => {
  try {
    const saved = localStorage.getItem(COLLAPSED_STORAGE_KEY)
    if (saved === '1') {
      store.setConfigCollapsed(true)
    } else if (saved === '0') {
      store.setConfigCollapsed(false)
    }
  } catch {
    // ignore
  }
})

function collapse() {
  store.setConfigCollapsed(true)
}

function expand() {
  store.setConfigCollapsed(false)
}

function toggle() {
  store.setConfigCollapsed(!store.state.isConfigCollapsed)
}

defineExpose({
  isCollapsed,
  collapse,
  expand,
  toggle,
})

const activePlanName = computed(() => {
  const plan = store.state.plans.find((p) => p.id === store.state.activePlanId)
  return plan?.name ?? t('schedule.planDefaultName', { n: 1 })
})

const activeCalendarName = computed(() => {
  const calId = store.state.majorSelected.calendarId
  if (!calId) return ''
  const cal = store.state.calendars.find((c) => c.calendarId === calId)
  return cal?.calendarName ?? ''
})

const activeMajorText = computed(() => {
  const grade = store.state.majorSelected.grade
  const majorName = store.state.majorSelected.majorName || store.state.majorSelected.major
  if (grade && majorName) {
    return `${grade} · ${majorName}`
  }
  if (majorName) {
    return majorName
  }
  if (grade) {
    return `${grade} · ${t('schedule.unselectedMajor')}`
  }
  return t('schedule.unselectedMajor')
})

const summaryTooltip = computed(() => {
  const parts: string[] = [activePlanName.value]
  if (activeCalendarName.value) parts.push(activeCalendarName.value)
  if (store.state.majorSelected.grade) parts.push(String(store.state.majorSelected.grade))
  if (store.state.majorSelected.majorName || store.state.majorSelected.major) {
    parts.push(store.state.majorSelected.majorName || store.state.majorSelected.major || '')
  }
  return `${parts.join(' · ')} (${t('schedule.expandSettingsHint')})`
})
</script>

<template>
  <div class="schedule-config-section">
    <Transition name="schedule-config" mode="out-in">
      <!-- 收起态：精致紧凑摘要条，整条可点击/键盘回车展开 -->
      <div
        v-if="isCollapsed"
        key="collapsed"
        role="button"
        tabindex="0"
        :aria-expanded="false"
        :aria-label="t('schedule.expandSettingsHint')"
        :title="summaryTooltip"
        class="gf-panel rounded-2xl border border-line/70 p-3 shadow-[0_2px_10px_-4px_rgba(0,0,0,0.05)] flex items-center justify-between gap-2.5 transition-all hover:border-line hover:shadow-md cursor-pointer select-none focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-primary/40 group"
        data-testid="schedule-config-collapsed-bar"
        @click="expand"
        @keydown.enter.prevent="expand"
        @keydown.space.prevent="expand"
      >
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <!-- 方案名徽章 -->
          <span class="inline-flex items-center gap-1.5 rounded-md bg-primary/10 px-2 py-0.5 text-xs font-semibold text-primary shrink-0">
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-500 shrink-0" aria-hidden="true" />
            {{ activePlanName }}
          </span>

          <span class="text-base-content/30 text-xs shrink-0 select-none">·</span>

          <!-- 学期 · 年级 · 专业 文本（单一内联行截断，避免各分段相互挤压） -->
          <div class="min-w-0 flex-1 truncate text-xs text-base-content/75 font-medium">
            <span v-if="activeCalendarName" class="text-base-content/60">{{ activeCalendarName }}</span>
            <span v-if="activeCalendarName && activeMajorText" class="mx-1.5 text-base-content/30 select-none">·</span>
            <span class="font-medium text-base-content">{{ activeMajorText }}</span>
          </div>
        </div>

        <!-- 展开按钮指示 -->
        <div class="shrink-0 flex items-center gap-1.5 text-xs font-medium text-base-content/60 group-hover:text-primary transition-colors">
          <span class="hidden sm:inline">{{ t('schedule.expandSettings') }}</span>
          <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-base-200/70 text-base-content/60 group-hover:bg-primary/10 group-hover:text-primary group-hover:translate-y-0.5 transition-all active:scale-[0.94]">
            <ChevronDown class="h-4 w-4" />
          </span>
        </div>
      </div>

      <!-- 展开态：包含完整方案条与专业选择器 -->
      <div v-else key="expanded" class="flex flex-col gap-3" data-testid="schedule-config-expanded-wrap">
        <SchedulePlanBar :collapsible="true" @toggle-collapse="collapse" />
        <ScheduleMajorSelector :collapsible="true" @toggle-collapse="collapse" />
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.schedule-config-enter-active,
.schedule-config-leave-active {
  transition: opacity 0.18s cubic-bezier(0.16, 1, 0.3, 1), transform 0.18s cubic-bezier(0.16, 1, 0.3, 1);
}

.schedule-config-enter-from,
.schedule-config-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

@media (prefers-reduced-motion: reduce) {
  .schedule-config-enter-active,
  .schedule-config-leave-active {
    transition: none !important;
  }
}
</style>

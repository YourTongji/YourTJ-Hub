<script setup lang="ts">
// 自定义占位事件弹窗（v2）：管理当前方案的「有事」类灰色占位块。
// 字段：标签（默认「有事」）+ 星期 + 节次多选 + 周次；参与冲突派生（custom: 伪课号）。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Trash2, X } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { formatWeeksText, MAX_WEEK } from '@/site/utils/pkArrange'
import type { PkCustomEvent } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const dialogOpen = computed({
  get: () => props.open,
  set: (open: boolean) => {
    if (!open) emit('close')
  },
})

const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const
const ALL_SECTIONS = Array.from({ length: 12 }, (_, i) => i + 1)
const ALL_WEEKS = Array.from({ length: MAX_WEEK }, (_, i) => i + 1)

const events = computed<PkCustomEvent[]>(
  () => store.state.plans.find((plan) => plan.id === store.state.activePlanId)?.customEvents ?? [],
)

// ---- 新建表单 ----
const form = ref({
  label: '',
  day: 0,
  sections: new Set<number>(),
  weeks: new Set<number>(),
})

watch(
  () => props.open,
  (open) => {
    if (open) resetForm()
  },
)

function resetForm() {
  form.value = {
    label: '',
    day: 0,
    sections: new Set<number>(),
    weeks: new Set<number>(),
  }
}

function toggleSection(section: number) {
  const next = new Set(form.value.sections)
  if (next.has(section)) next.delete(section)
  else next.add(section)
  form.value.sections = next
}

function toggleWeek(week: number) {
  const next = new Set(form.value.weeks)
  if (next.has(week)) next.delete(week)
  else next.add(week)
  form.value.weeks = next
}

function addEvent() {
  const created = store.addCustomEvent({
    label: form.value.label.trim() || t('schedule.customEventDefaultLabel'),
    day: form.value.day,
    sections: [...form.value.sections],
    weeks: [...form.value.weeks],
  })
  if (created) resetForm()
}

function eventSummary(event: PkCustomEvent): string {
  const weekday = t(`schedule.weekdays.${WEEKDAY_KEYS[event.day - 1]}`)
  const sections = event.sections.length === 1
    ? `${event.sections[0]}`
    : `${event.sections[0]}-${event.sections[event.sections.length - 1]}`
  return `${weekday} ${t('schedule.sectionsN', { range: sections })} · ${t('schedule.weeksN', { range: formatWeeksText(event.weeks) || '-' })}`
}
</script>

<template>
  <DialogRoot v-model:open="dialogOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2100] max-h-[85vh] w-[92vw] max-w-[480px] -translate-x-1/2 -translate-y-1/2 overflow-y-auto outline-none"
      >
        <div class="overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl">
          <div class="flex items-start justify-between gap-2 border-b border-line/60 px-4 py-3">
            <div class="min-w-0">
              <DialogTitle class="text-sm font-bold text-base-content">{{ t('schedule.customEventTitle') }}</DialogTitle>
              <DialogDescription class="text-[11px] text-base-content/55">{{ t('schedule.customEventHint') }}</DialogDescription>
            </div>
            <button type="button" class="gf-icon-button shrink-0" :aria-label="t('common.close')" @click="emit('close')">
              <X class="h-4 w-4" />
            </button>
          </div>

          <div class="space-y-4 p-4">
            <!-- 已有事件列表 -->
            <section v-if="events.length > 0">
              <h4 class="mb-1.5 text-[12px] font-semibold text-base-content/70">{{ t('schedule.customEventListed', { count: events.length }) }}</h4>
              <ul class="divide-y divide-line/60 rounded-lg border border-line/60">
                <li v-for="event in events" :key="event.id" class="flex items-center gap-2 px-3 py-2">
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-[13px] font-medium text-base-content">{{ event.label }}</p>
                    <p class="truncate text-[11px] text-base-content/50">{{ eventSummary(event) }}</p>
                  </div>
                  <button
                    type="button"
                    class="gf-icon-button shrink-0 text-error/80"
                    :aria-label="t('schedule.customEventRemove')"
                    @click="store.removeCustomEvent(event.id)"
                  >
                    <Trash2 class="h-4 w-4" />
                  </button>
                </li>
              </ul>
            </section>

            <!-- 新建表单 -->
            <section class="space-y-3">
              <h4 class="text-[12px] font-semibold text-base-content/70">{{ t('schedule.customEventNew') }}</h4>
              <label class="block">
                <span class="mb-1 block text-[12px] text-base-content/70">{{ t('schedule.customEventLabel') }}</span>
                <input
                  v-model="form.label"
                  type="text"
                  class="gf-input gf-input-md w-full"
                  :placeholder="t('schedule.customEventDefaultLabel')"
                />
              </label>
              <div>
                <span class="mb-1 block text-[12px] text-base-content/70">{{ t('schedule.customEventDay') }}</span>
                <div class="flex flex-wrap gap-1">
                  <button
                    v-for="(day, index) in WEEKDAY_KEYS"
                    :key="day"
                    type="button"
                    class="gf-button gf-button-sm"
                    :class="form.day === index + 1 ? 'gf-button-primary' : 'gf-button-outline'"
                    :aria-pressed="form.day === index + 1"
                    @click="form.day = index + 1"
                  >
                    {{ t(`schedule.weekdays.${day}`) }}
                  </button>
                </div>
              </div>
              <div>
                <span class="mb-1 block text-[12px] text-base-content/70">{{ t('schedule.customEventSections') }}</span>
                <div class="flex flex-wrap gap-1">
                  <button
                    v-for="section in ALL_SECTIONS"
                    :key="section"
                    type="button"
                    class="gf-button gf-button-sm h-7 min-w-7 px-1.5"
                    :class="form.sections.has(section) ? 'gf-button-primary' : 'gf-button-outline'"
                    :aria-pressed="form.sections.has(section)"
                    @click="toggleSection(section)"
                  >
                    {{ section }}
                  </button>
                </div>
              </div>
              <div>
                <span class="mb-1 block text-[12px] text-base-content/70">{{ t('schedule.customEventWeeks') }}</span>
                <div class="flex flex-wrap gap-1">
                  <button
                    v-for="week in ALL_WEEKS"
                    :key="week"
                    type="button"
                    class="gf-button gf-button-sm h-7 min-w-7 px-1.5"
                    :class="form.weeks.has(week) ? 'gf-button-primary' : 'gf-button-outline'"
                    :aria-pressed="form.weeks.has(week)"
                    @click="toggleWeek(week)"
                  >
                    {{ week }}
                  </button>
                </div>
              </div>
              <button
                type="button"
                class="gf-button gf-button-md gf-button-primary w-full"
                :disabled="form.day < 1 || form.sections.size === 0 || form.weeks.size === 0"
                @click="addEvent"
              >
                {{ t('schedule.customEventAdd') }}
              </button>
            </section>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

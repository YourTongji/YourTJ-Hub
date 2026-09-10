<script setup lang="ts">
// 方案对比弹窗：多选方案 → 课程/占位对照表，差异行高亮（bg-warning/10 + 左侧警示条）。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { X } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { comparePlans } from '@/site/utils/pkPlanCompare'

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

// 选中方案 id（保持 store.state.plans 顺序，与 classesByPlan/presentByPlan 列序一致）。
const selectedPlanIds = ref<string[]>([])

watch(
  () => props.open,
  (open) => {
    if (open) selectedPlanIds.value = store.state.plans.map((plan) => plan.id)
  },
)

const selectedPlans = computed(() => store.state.plans.filter((plan) => selectedPlanIds.value.includes(plan.id)))

const comparison = computed(() => comparePlans(selectedPlans.value))

function togglePlan(planId: string) {
  if (selectedPlanIds.value.includes(planId)) {
    selectedPlanIds.value = selectedPlanIds.value.filter((id) => id !== planId)
  } else {
    selectedPlanIds.value = [...selectedPlanIds.value, planId]
  }
}
</script>

<template>
  <DialogRoot v-model:open="dialogOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2100] max-h-[85vh] w-[92vw] max-w-[520px] -translate-x-1/2 -translate-y-1/2 overflow-y-auto outline-none"
      >
        <div class="overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl">
          <div class="flex items-start justify-between gap-2 border-b border-line/60 px-4 py-3">
            <div class="min-w-0">
              <DialogTitle class="text-sm font-bold text-base-content">{{ t('schedule.planCompareTitle') }}</DialogTitle>
              <DialogDescription class="text-[11px] text-base-content/55">{{ t('schedule.planCompareHint') }}</DialogDescription>
            </div>
            <button type="button" class="gf-icon-button shrink-0" :aria-label="t('common.close')" @click="emit('close')">
              <X class="h-4 w-4" />
            </button>
          </div>

          <div class="max-h-[60vh] overflow-y-auto p-4">
            <!-- 方案多选 -->
            <div class="flex flex-wrap gap-1.5">
              <label
                v-for="plan in store.state.plans"
                :key="plan.id"
                class="flex cursor-pointer items-center gap-1.5 rounded-lg border border-line/60 px-2.5 py-1.5 text-[12px] transition-colors select-none"
                :class="selectedPlanIds.includes(plan.id) ? 'border-primary/60 bg-primary/5 text-primary' : 'text-base-content/70 hover:bg-base-200/60'"
              >
                <input
                  type="checkbox"
                  class="checkbox checkbox-sm checkbox-primary rounded-md"
                  :checked="selectedPlanIds.includes(plan.id)"
                  :aria-label="plan.name"
                  @change="togglePlan(plan.id)"
                />
                <span class="truncate">{{ plan.name }}</span>
              </label>
            </div>

            <!-- 不足两套方案：仅提示 -->
            <p v-if="selectedPlans.length < 2" class="mt-3 text-[12px] text-base-content/60">
              {{ t('schedule.planCompareSelectHint') }}
            </p>

            <template v-else>
              <!-- 汇总 -->
              <p
                class="mt-3 text-[13px] font-medium"
                :class="comparison.diffCount === 0 ? 'text-base-content/70' : 'text-warning'"
              >
                {{ comparison.diffCount === 0 ? t('schedule.planCompareSame') : t('schedule.planCompareDiffCount', { count: comparison.diffCount }) }}
              </p>

              <!-- 课程对照表 -->
              <div v-if="comparison.courses.length > 0" class="mt-3 overflow-x-auto rounded-lg border border-line/60">
                <table class="w-full text-left">
                  <thead>
                    <tr class="border-b border-line/60 bg-base-200/40">
                      <th class="px-3 py-2 text-[12px] font-semibold text-base-content/70">{{ t('schedule.planCompareCourse') }}</th>
                      <th v-for="plan in selectedPlans" :key="plan.id" class="px-3 py-2 text-[12px] font-semibold text-base-content/70">
                        {{ plan.name }}
                      </th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-line/60">
                    <tr
                      v-for="row in comparison.courses"
                      :key="row.courseCode"
                      class="border-l-2"
                      :class="row.different ? 'bg-warning/10 border-warning' : 'border-transparent'"
                    >
                      <td class="px-3 py-2 text-[13px] text-base-content">{{ row.courseName }}（{{ row.courseCode }}）</td>
                      <td v-for="(classes, i) in row.classesByPlan" :key="i" class="px-3 py-2 text-[12px] text-base-content/80">
                        {{ classes.join('、') || t('schedule.planCompareNoEntry') }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <!-- 占位对照表 -->
              <div v-if="comparison.placeholders.length > 0" class="mt-3 overflow-x-auto rounded-lg border border-line/60">
                <table class="w-full text-left">
                  <thead>
                    <tr class="border-b border-line/60 bg-base-200/40">
                      <th class="px-3 py-2 text-[12px] font-semibold text-base-content/70">{{ t('schedule.planComparePlaceholder') }}</th>
                      <th v-for="plan in selectedPlans" :key="plan.id" class="px-3 py-2 text-[12px] font-semibold text-base-content/70">
                        {{ plan.name }}
                      </th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-line/60">
                    <tr
                      v-for="row in comparison.placeholders"
                      :key="row.signature"
                      class="border-l-2"
                      :class="row.different ? 'bg-warning/10 border-warning' : 'border-transparent'"
                    >
                      <td class="px-3 py-2 text-[13px] text-base-content">{{ row.label }}</td>
                      <td v-for="(present, i) in row.presentByPlan" :key="i" class="px-3 py-2 text-[12px] text-base-content/80">
                        {{ present ? row.label : t('schedule.planCompareNoEntry') }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </template>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
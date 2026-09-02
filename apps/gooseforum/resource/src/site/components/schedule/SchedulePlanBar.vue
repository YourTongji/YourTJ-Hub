<script setup lang="ts">
// 方案管理条（v2 多方案）：方案下拉切换 + 新增 + 删除（确认）+ 清空当前方案。
// 对齐 USTC 排课器「我的方案」交互：每套方案独立持有课程与自定义占位。
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MoreHorizontal, Plus, Trash2 } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import SiteSelect from '@/site/components/SiteSelect.vue'
import { MAX_PLANS, useScheduleStore } from '@/site/composables/useScheduleStore'

const { t } = useI18n()
const store = useScheduleStore()

const planOptions = computed(() =>
  store.state.plans.map((plan) => ({ value: plan.id, label: plan.name })),
)

/** 达到方案数量上限：禁用「新增方案」并给出提示。 */
const planLimitReached = computed(() => store.state.plans.length >= MAX_PLANS)

const planValue = computed({
  get: () => store.state.activePlanId,
  set: (value: string) => store.switchPlan(value),
})

// ---- 删除确认 ----
const pendingDelete = ref(false)

// ---- 「…」更多菜单 ----
const menuOpen = ref(false)
const confirmClear = ref(false)

function handleAdd() {
  const plan = store.createPlan()
  if (!plan) return
  store.switchPlan(plan.id)
}
</script>

<template>
  <div class="gf-panel flex flex-wrap items-center gap-2 p-3">
    <span class="text-[12px] font-semibold text-base-content/70">{{ t('schedule.myPlans') }}</span>
    <div class="min-w-0 flex-1">
      <SiteSelect
        v-model="planValue"
        :options="planOptions"
        :label="t('schedule.myPlans')"
        :aria-label="t('schedule.myPlans')"
      />
    </div>
    <button
      type="button"
      class="gf-icon-button shrink-0 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="planLimitReached"
      :aria-label="planLimitReached ? t('schedule.planLimitReached', { n: MAX_PLANS }) : t('schedule.planAdd')"
      :title="planLimitReached ? t('schedule.planLimitReached', { n: MAX_PLANS }) : t('schedule.planAdd')"
      @click="handleAdd"
    >
      <Plus class="h-4 w-4" />
    </button>
    <button
      type="button"
      class="gf-icon-button shrink-0 text-error/80"
      :aria-label="t('schedule.planDelete')"
      :title="t('schedule.planDelete')"
      @click="pendingDelete = true"
    >
      <Trash2 class="h-4 w-4" />
    </button>
    <div class="relative shrink-0">
      <button
        type="button"
        class="gf-icon-button"
        :aria-label="t('schedule.planMore')"
        :title="t('schedule.planMore')"
        :aria-expanded="menuOpen"
        @click="menuOpen = !menuOpen"
      >
        <MoreHorizontal class="h-4 w-4" />
      </button>
      <Transition name="gf-menu">
        <div v-if="menuOpen" class="gf-menu-surface absolute right-0 top-[calc(100%+0.375rem)] z-30 w-44 p-1">
          <button
            type="button"
            class="gf-menu-item w-full"
            @click="menuOpen = false; confirmClear = true"
          >
            {{ t('schedule.planClearCurrent') }}
          </button>
        </div>
      </Transition>
    </div>

    <!-- 删除确认 -->
    <DialogRoot :open="pendingDelete" @update:open="(open: boolean) => { if (!open) pendingDelete = false }">
      <DialogPortal>
        <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
        <DialogContent class="fixed left-1/2 top-1/2 z-[2100] w-[88vw] max-w-[360px] -translate-x-1/2 -translate-y-1/2 outline-none">
          <div class="rounded-2xl border border-line/70 bg-base-100 p-5 shadow-lg">
            <DialogTitle class="text-sm font-bold text-base-content">{{ t('schedule.planDelete') }}</DialogTitle>
            <DialogDescription class="mt-2 text-[13px] text-base-content/70">
              {{ t('schedule.planDeleteConfirm', { name: store.state.plans.find((p) => p.id === store.state.activePlanId)?.name ?? '' }) }}
            </DialogDescription>
            <div class="mt-4 flex justify-end gap-2">
              <button type="button" class="gf-button gf-button-md gf-button-ghost" @click="pendingDelete = false">
                {{ t('schedule.cancel') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-md gf-button-danger"
                @click="store.deletePlan(store.state.activePlanId); pendingDelete = false"
              >
                {{ t('schedule.planDelete') }}
              </button>
            </div>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>

    <!-- 清空当前方案确认 -->
    <DialogRoot :open="confirmClear" @update:open="(open: boolean) => { if (!open) confirmClear = false }">
      <DialogPortal>
        <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
        <DialogContent class="fixed left-1/2 top-1/2 z-[2100] w-[88vw] max-w-[360px] -translate-x-1/2 -translate-y-1/2 outline-none">
          <div class="rounded-2xl border border-line/70 bg-base-100 p-5 shadow-lg">
            <DialogTitle class="text-sm font-bold text-base-content">{{ t('schedule.planClearCurrent') }}</DialogTitle>
            <DialogDescription class="mt-2 text-[13px] text-base-content/70">
              {{ t('schedule.planClearConfirm') }}
            </DialogDescription>
            <div class="mt-4 flex justify-end gap-2">
              <button type="button" class="gf-button gf-button-md gf-button-ghost" @click="confirmClear = false">
                {{ t('schedule.cancel') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-md gf-button-danger"
                @click="store.clearActivePlan(); confirmClear = false"
              >
                {{ t('schedule.planClearCurrent') }}
              </button>
            </div>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </div>
</template>

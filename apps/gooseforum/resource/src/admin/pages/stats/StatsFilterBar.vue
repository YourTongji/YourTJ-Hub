<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CalendarRange } from '@lucide/vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import AdminSelect from '@/admin/components/AdminSelect.vue'
import { Button } from '@/admin/components/ui/button'
import DateRangePicker from '@/admin/pages/stats/DateRangePicker.vue'

// 统一筛选栏（issue #1145）：时间区间与板块选择同时作用于板块统计、趋势、用户排行与用户明细。
// 只负责展示与派发，日期计算与校验由页面负责（页面持有唯一一份筛选状态）。
const props = defineProps<{
  presets: { key: string, label: string }[]
  activePreset: string
  startDate: string
  endDate: string
  categoryId: number
  categoryOptions: { value: number, label: string }[]
  timezone?: string
  maxRangeDays: number
  invalidMessage?: string
}>()

const emit = defineEmits<{
  'update:startDate': [value: string]
  'update:endDate': [value: string]
  'update:categoryId': [value: number]
  selectPreset: [key: string]
}>()

const { t } = useI18n()

const customOpen = computed(() => props.activePreset === 'custom')
</script>

<template>
  <AdminSection body-class="grid gap-3 p-3 sm:p-4">
    <div class="grid gap-3 lg:flex lg:flex-wrap lg:items-center lg:gap-4">
      <div class="flex items-center gap-2 text-sm font-semibold">
        <CalendarRange class="size-4 text-muted-foreground" />
        <span>{{ t('statsAdmin.filterTitle') }}</span>
      </div>

      <div class="flex flex-wrap items-center gap-2" role="group" :aria-label="t('statsAdmin.rangeLabel')">
        <Button
          v-for="preset in presets"
          :key="preset.key"
          type="button"
          size="sm"
          :variant="activePreset === preset.key ? 'default' : 'outline'"
          @click="emit('selectPreset', preset.key)"
        >
          {{ preset.label }}
        </Button>
        <Button
          type="button"
          size="sm"
          :variant="customOpen ? 'default' : 'outline'"
          @click="emit('selectPreset', 'custom')"
        >
          {{ t('statsAdmin.rangeCustom') }}
        </Button>
        <span class="ml-1 text-xs tabular-nums text-muted-foreground">{{ startDate }} ~ {{ endDate }}</span>
      </div>

      <div v-if="customOpen" class="min-w-0">
        <DateRangePicker
          :start-date="startDate"
          :end-date="endDate"
          @update:start-date="emit('update:startDate', $event)"
          @update:end-date="emit('update:endDate', $event)"
        />
      </div>

      <div class="flex items-center gap-2 lg:ml-auto">
        <span class="shrink-0 text-sm text-muted-foreground">{{ t('statsAdmin.categoryLabel') }}</span>
        <AdminSelect
          :model-value="categoryId"
          :options="categoryOptions"
          class="h-9 w-full sm:w-56"
          :aria-label="t('statsAdmin.categoryLabel')"
          @update:model-value="emit('update:categoryId', $event)"
        />
      </div>
    </div>

    <p class="text-xs leading-relaxed text-muted-foreground">
      <span v-if="timezone">{{ t('statsAdmin.timezoneHint', { zone: timezone }) }} · </span>
      <span>{{ t('statsAdmin.rangeLimitHint', { days: maxRangeDays }) }}</span>
      <br />
      <span>{{ t('statsAdmin.scopeHint') }}</span>
    </p>

    <p
      v-if="invalidMessage"
      data-testid="stats-filter-invalid"
      class="flex flex-wrap items-center gap-x-1 text-xs font-medium text-destructive"
    >
      <span>{{ invalidMessage }}</span>
      <span class="font-normal">{{ t('statsAdmin.invalidKept') }}</span>
    </p>
  </AdminSection>
</template>

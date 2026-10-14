<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VisArea, VisAxis, VisLine, VisXYContainer } from '@unovis/vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import AdminToolbar from '@/admin/components/AdminToolbar.vue'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import type { ChartConfig } from '@/admin/components/ui/chart'
import {
  ChartContainer,
  ChartCrosshair,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
} from '@/admin/components/ui/chart'
import ManagementTable from '@/admin/pages/management/ManagementTable.vue'
import { getAdminStatsCategoryTrend } from '@/admin/runtime/api'
import type {
  AdminStatsCategorySummaryResult,
  AdminStatsCategoryTrendResult,
  AdminStatsFilter,
} from '@/admin/runtime/api'

const props = defineProps<{
  filter: AdminStatsFilter
  summary?: AdminStatsCategorySummaryResult
  summaryLoading: boolean
  summaryError: string
}>()

const emit = defineEmits<{
  retry: []
  'select-category': [categoryId: number]
}>()

const { t } = useI18n()

const trend = ref<AdminStatsCategoryTrendResult>()
const trendLoading = ref(false)
const trendError = ref('')
// 请求序号守卫：快速切换筛选时，旧响应不得覆盖新结果（AC-8）。
let trendSeq = 0

const categories = computed(() => props.summary?.categories ?? [])
const totals = computed(() => props.summary?.totals)
const summaryEstimated = computed(() => props.summary?.coverage?.estimated === true)
const selectedName = computed(
  () => categories.value.find(item => Number(item.categoryId) === props.filter.categoryId)?.name || '',
)
const columns = computed(() => [
  t('statsAdmin.colCategory'),
  t('statsAdmin.colTopics'),
  t('statsAdmin.colReplies'),
  t('statsAdmin.colContributors'),
  t('statsAdmin.colAgents'),
])

const chartData = computed(() =>
  (trend.value?.days ?? []).map(day => ({
    // 本地零点解析：new Date('YYYY-MM-DD') 按 UTC 解析，负时区浏览器会把日期前移一天。
    date: new Date(`${day.date}T00:00:00`),
    topicCount: day.topicCount,
    replyCount: day.replyCount,
  })),
)
const estimatedDays = computed(() => (trend.value?.days ?? []).filter(day => day.estimated).length)

const chartConfig = computed<ChartConfig>(() => ({
  topicCount: { label: t('statsAdmin.colTopics'), color: 'var(--chart-2)' },
  replyCount: { label: t('statsAdmin.colReplies'), color: 'var(--chart-4)' },
}))

const svgDefs = `
  <linearGradient id="fillCategoryTopicCount" x1="0" y1="0" x2="0" y2="1">
    <stop offset="5%" stop-color="var(--color-topicCount)" stop-opacity="0.8" />
    <stop offset="95%" stop-color="var(--color-topicCount)" stop-opacity="0.1" />
  </linearGradient>
  <linearGradient id="fillCategoryReplyCount" x1="0" y1="0" x2="0" y2="1">
    <stop offset="5%" stop-color="var(--color-replyCount)" stop-opacity="0.8" />
    <stop offset="95%" stop-color="var(--color-replyCount)" stop-opacity="0.1" />
  </linearGradient>
`

function errorText(error: unknown) {
  return error instanceof Error && error.message ? error.message : t('statsAdmin.loadFailed')
}

async function loadTrend() {
  const seq = ++trendSeq
  const categoryId = props.filter.categoryId
  if (!categoryId) {
    trend.value = undefined
    trendError.value = ''
    trendLoading.value = false
    return
  }
  trendLoading.value = true
  trendError.value = ''
  try {
    const result = await getAdminStatsCategoryTrend({
      startDate: props.filter.startDate,
      endDate: props.filter.endDate,
      categoryId,
    })
    if (seq === trendSeq) trend.value = result
  } catch (error) {
    if (seq !== trendSeq) return
    trend.value = undefined
    trendError.value = errorText(error)
  } finally {
    if (seq === trendSeq) trendLoading.value = false
  }
}

watch(() => [props.filter.startDate, props.filter.endDate, props.filter.categoryId], loadTrend, { immediate: true })
</script>

<template>
  <ManagementTable
    :columns="columns"
    :loading="summaryLoading && categories.length === 0"
    :error="summaryError"
    :show-pagination="false"
    @retry="emit('retry')"
  >
    <template #header>
      <AdminToolbar class="-mx-3 -my-2 border-b-0">
        <div class="min-w-0">
          <h2 class="text-sm font-semibold">{{ t('statsAdmin.categoryTitle') }}</h2>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ t('statsAdmin.categoryHint') }}</p>
        </div>
        <Badge v-if="summaryEstimated" variant="secondary" class="shrink-0">
          {{ t('statsAdmin.estimatedBadge') }}
        </Badge>
      </AdminToolbar>
    </template>

    <tr v-if="!summaryLoading && !summaryError && categories.length === 0">
      <td :colspan="columns.length" class="h-24 px-4 text-center text-muted-foreground">
        {{ t('statsAdmin.categoryEmpty') }}
      </td>
    </tr>
    <tr
      v-for="category in categories"
      :key="category.categoryId"
      class="cursor-pointer transition-colors hover:bg-muted/35"
      :class="Number(category.categoryId) === filter.categoryId ? 'bg-primary/5' : ''"
      @click="emit('select-category', Number(category.categoryId))"
    >
      <td class="px-4 py-2 align-middle font-medium">{{ category.name }}</td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ category.topicCount }}</td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ category.replyCount }}</td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ category.contributorCount }}</td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ category.agentContributorCount }}</td>
    </tr>
    <tr v-if="totals" class="border-t bg-muted/40 font-medium" data-testid="stats-category-totals">
      <td class="px-4 py-2 align-middle">
        {{ t('statsAdmin.totalsRow') }}
        <span class="block text-[11px] font-normal text-muted-foreground">{{ t('statsAdmin.totalsHint') }}</span>
      </td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ totals.topicCount }}</td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ totals.replyCount }}</td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ totals.contributorCount }}</td>
      <td class="px-4 py-2 align-middle tabular-nums">{{ totals.agentContributorCount }}</td>
    </tr>
  </ManagementTable>

  <AdminSection class="mt-4" body-class="px-3 pb-3 pt-4 sm:px-5">
    <template #header>
      <div class="grid gap-2 sm:flex sm:items-center sm:justify-between">
        <div class="min-w-0">
          <h2 class="text-sm font-semibold">{{ t('statsAdmin.trendTitle') }}</h2>
          <p class="mt-0.5 text-xs text-muted-foreground">
            <span v-if="selectedName">{{ t('statsAdmin.trendFor', { name: selectedName }) }} · </span>
            <span>{{ t('statsAdmin.trendHint') }}</span>
          </p>
        </div>
      </div>
    </template>

    <p v-if="!filter.categoryId" class="flex h-[200px] items-center justify-center text-sm text-muted-foreground">
      {{ t('statsAdmin.trendEmpty') }}
    </p>
    <p v-else-if="trendLoading" class="flex h-[200px] items-center justify-center text-sm text-muted-foreground">
      {{ t('common.loadingShort') }}
    </p>
    <div v-else-if="trendError" class="flex h-[200px] flex-col items-center justify-center gap-2 text-sm text-destructive">
      <span>{{ trendError }}</span>
      <Button variant="link" size="sm" class="h-auto px-0 text-destructive" type="button" @click="loadTrend">
        {{ t('statsAdmin.detailRetry') }}
      </Button>
    </div>
    <ChartContainer
      v-else
      :config="chartConfig"
      class="aspect-auto h-[200px] w-full sm:h-[240px]"
      :cursor="false"
    >
      <VisXYContainer
        :data="chartData"
        :svg-defs="svgDefs"
        :margin="{ top: 18, right: 18, left: -24, bottom: 28 }"
      >
        <VisArea
          :x="(d: { date: Date }) => d.date"
          :y="[(d: { topicCount: number }) => d.topicCount, (d: { replyCount: number }) => d.replyCount]"
          :color="(_d: unknown, i: number) => ['url(#fillCategoryTopicCount)', 'url(#fillCategoryReplyCount)'][i]"
          :opacity="1"
        />
        <VisLine
          :x="(d: { date: Date }) => d.date"
          :y="[(d: { topicCount: number }) => d.topicCount, (d: { replyCount: number }) => d.replyCount]"
          :color="(_d: unknown, i: number) => [chartConfig.topicCount.color, chartConfig.replyCount.color][i]"
          :line-width="1"
        />
        <VisAxis
          type="x"
          :x="(d: { date: Date }) => d.date"
          :tick-line="false"
          :domain-line="false"
          :grid-line="false"
          :num-ticks="7"
          :tick-format="(d: number) => new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })"
        />
        <VisAxis type="y" :num-ticks="5" :tick-line="false" :domain-line="false" />
        <ChartTooltip />
        <ChartCrosshair
          :template="componentToString(chartConfig, ChartTooltipContent, {
            labelFormatter: (d: number | Date) => new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }),
          })"
          :color="(_d: unknown, i: number) => [chartConfig.topicCount.color, chartConfig.replyCount.color][i % 2]"
        />
      </VisXYContainer>
      <ChartLegendContent />
    </ChartContainer>

    <p
      v-if="filter.categoryId && !trendLoading"
      data-testid="stats-trend-estimated"
      class="mt-2 text-xs text-muted-foreground"
    >
      <span v-if="estimatedDays > 0" class="font-medium text-amber-600 dark:text-amber-500">
        {{ t('statsAdmin.estimatedDays', { count: estimatedDays }) }}
      </span>
      <span v-else>{{ t('statsAdmin.estimatedNone') }}</span>
      <span> · {{ t('statsAdmin.estimatedNote') }}</span>
    </p>
  </AdminSection>
</template>

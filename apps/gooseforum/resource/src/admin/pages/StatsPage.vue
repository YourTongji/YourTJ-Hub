<script setup lang="ts">import { adminText } from '@/admin/runtime/i18n-text'

import { FileText, Link as LinkIcon, MessageSquare, Users } from '@lucide/vue'
import { computed, defineAsyncComponent, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AdminSection from '@/admin/components/AdminSection.vue'
import { BasicPage } from '@/admin/components/global-layout'
import CategoryStatsSection from '@/admin/pages/stats/CategoryStatsSection.vue'
import StatsFilterBar from '@/admin/pages/stats/StatsFilterBar.vue'
import UserStatsSection from '@/admin/pages/stats/UserStatsSection.vue'
import {
  getAdminStatsCategorySummary,
  getGithubReleases,
  getServerVersion,
  getSiteStatistics,
  getTrafficOverview,
} from '@/admin/runtime/api'
import type { AdminStatsCategorySummaryResult, AdminStatsFilter } from '@/admin/runtime/api'
import type {
  AdminPayload,
  DailyTraffic,
  GithubRelease,
  ManageHomeProps,
  ServerVersion,
  SiteStatistics,
} from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

const ProjectVersion = defineAsyncComponent(() => import('@/admin/pages/stats/ProjectVersion.vue'))
const TrafficOverview = defineAsyncComponent(() => import('@/admin/pages/stats/TrafficOverview.vue'))

const { t } = useI18n()

// 贡献统计（issue #1145）区间上限，与后端校验一致；服务端返回 coverage.maxRangeDays 时以后者为准展示。
const MAX_RANGE_DAYS = 366
const DAY_MS = 86_400_000

const stats = ref<SiteStatistics>()
const statsLoading = ref(true)
const traffic = ref<DailyTraffic[]>([])
const trafficLoading = ref(true)
const serverVersion = ref<ServerVersion>()
const serverVersionLoading = ref(true)
const releases = ref<GithubRelease[]>([])
const releasesLoading = ref(true)
const releasesError = ref('')

const today = new Date()
const presets = computed(() => [
  { key: '7', label: t('statsAdmin.range7'), startDate: formatDate(addDays(today, -6)), endDate: formatDate(today) },
  { key: '30', label: t('statsAdmin.range30'), startDate: formatDate(addDays(today, -29)), endDate: formatDate(today) },
])

// filter 是筛选栏的即时输入，activeFilter 是唯一被各区块消费的有效筛选：
// 区间非法时不派发请求，页面继续展示上一次有效筛选的结果（AC-8）。
const filter = reactive({ startDate: formatDate(addDays(today, -6)), endDate: formatDate(today), categoryId: 0 })
const activePreset = ref('7')
const activeFilter = ref<AdminStatsFilter>({ ...filter })
const invalidMessage = ref('')

const categorySummary = ref<AdminStatsCategorySummaryResult>()
const categorySummaryLoading = ref(true)
const categorySummaryError = ref('')
// 请求序号守卫：快速切换筛选时旧响应不得覆盖新结果（AC-8）。
let summarySeq = 0
let trafficSeq = 0

const categoryOptions = computed(() => [
  { value: 0, label: t('statsAdmin.categoryAll') },
  ...(categorySummary.value?.categories ?? []).map(item => ({ value: Number(item.categoryId), label: item.name })),
])
const timezone = computed(() => categorySummary.value?.coverage?.timezone)
const maxRangeDays = computed(() => Number(categorySummary.value?.coverage?.maxRangeDays) || MAX_RANGE_DAYS)
const rules = computed(() => [
  t('statsAdmin.ruleTime'),
  t('statsAdmin.ruleLikes'),
  t('statsAdmin.ruleCategory'),
  t('statsAdmin.ruleUsers'),
])

const summaryItems = computed(() => [
  {
    label: adminText('k002g'),
    value: stats.value?.userCount,
    delta: `+${stats.value?.userMonthCount ?? 0}`,
    icon: Users,
    visible: true,
  },
  {
    label: adminText('k002h'),
    value: stats.value?.topicMaxId,
    delta: `+${stats.value?.topicMonthCount ?? 0}`,
    icon: FileText,
    visible: true,
  },
  {
    label: adminText('k002i'),
    value: stats.value?.postMaxId,
    icon: MessageSquare,
    visible: true,
  },
  {
    label: adminText('k002j'),
    value: stats.value?.linksCount,
    icon: LinkIcon,
    visible: true,
  },
])

function addDays(date: Date, days: number) {
  const next = new Date(date)
  next.setDate(next.getDate() + days)
  return next
}

function formatDate(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function rangeDays(startDate: string, endDate: string) {
  const start = new Date(`${startDate}T00:00:00`).getTime()
  const end = new Date(`${endDate}T00:00:00`).getTime()
  if (Number.isNaN(start) || Number.isNaN(end)) return Number.NaN
  return Math.round((end - start) / DAY_MS) + 1
}

function validateRange(startDate: string, endDate: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(startDate) || !/^\d{4}-\d{2}-\d{2}$/.test(endDate)) {
    return t('statsAdmin.invalidFormat')
  }
  const days = rangeDays(startDate, endDate)
  if (!Number.isFinite(days) || days < 1) return t('statsAdmin.invalidOrder')
  if (days > MAX_RANGE_DAYS) return t('statsAdmin.invalidSpan', { days, max: MAX_RANGE_DAYS })
  return ''
}

function errorText(error: unknown) {
  return error instanceof Error && error.message ? error.message : t('statsAdmin.loadFailed')
}

function selectPreset(key: string) {
  activePreset.value = key
  const preset = presets.value.find(item => item.key === key)
  if (!preset) return
  filter.startDate = preset.startDate
  filter.endDate = preset.endDate
}

function formatNumber(value?: number) {
  return statsLoading.value || value === undefined ? '...' : value.toLocaleString()
}

function modeLabel(mode?: string) {
  switch (mode) {
    case 'release':
      return adminText('k002k')
    case 'snapshot':
      return adminText('k002l')
    case 'development':
      return adminText('k002m')
    case 'custom':
      return adminText('k002n')
    default:
      return adminText('k002o')
  }
}

function shortCommit(commit?: string) {
  return commit ? commit.slice(0, 7) : ''
}

async function loadStats() {
  statsLoading.value = true
  try {
    stats.value = await getSiteStatistics()
  } finally {
    statsLoading.value = false
  }
}

async function loadTraffic() {
  const seq = ++trafficSeq
  trafficLoading.value = true
  try {
    const result = await getTrafficOverview(activeFilter.value.startDate, activeFilter.value.endDate)
    if (seq === trafficSeq) traffic.value = result
  } finally {
    if (seq === trafficSeq) trafficLoading.value = false
  }
}

async function loadCategorySummary() {
  const seq = ++summarySeq
  categorySummaryLoading.value = true
  categorySummaryError.value = ''
  try {
    const result = await getAdminStatsCategorySummary({
      startDate: activeFilter.value.startDate,
      endDate: activeFilter.value.endDate,
    })
    if (seq === summarySeq) categorySummary.value = result
  } catch (error) {
    if (seq !== summarySeq) return
    // 保留上一次结果：筛选栏的板块下拉仍需要上一份板块清单。
    categorySummaryError.value = errorText(error)
  } finally {
    if (seq === summarySeq) categorySummaryLoading.value = false
  }
}

async function loadServerVersion() {
  serverVersionLoading.value = true
  try {
    serverVersion.value = await getServerVersion()
  } finally {
    serverVersionLoading.value = false
  }
}

async function loadReleases() {
  releasesLoading.value = true
  releasesError.value = ''
  try {
    releases.value = await getGithubReleases()
  } catch (error) {
    console.error('Failed to fetch releases:', error)
    releasesError.value = error instanceof Error ? error.message : adminText('k002p')
  } finally {
    releasesLoading.value = false
  }
}

function applyFilter() {
  const invalid = validateRange(filter.startDate, filter.endDate)
  invalidMessage.value = invalid
  if (invalid) return
  // 同值不重派发由 Vue 响应式保证：重复选中同一预设/区间不会触发本 watcher。
  activeFilter.value = { startDate: filter.startDate, endDate: filter.endDate, categoryId: filter.categoryId }
}

watch(filter, applyFilter)

// 逐字段监听：只改板块时区间字段不变，不应重取按时间切片的接口（AC-1/AC-8）。
watch(
  [() => activeFilter.value.startDate, () => activeFilter.value.endDate],
  () => {
    void loadCategorySummary()
    void loadTraffic()
  },
  { immediate: true },
)

onMounted(() => {
  void loadStats()
  void loadServerVersion()
  void loadReleases()
})
</script>

<template>
  <BasicPage
    :title="adminText('k004c')"
    :description="adminText('k004d')"
    sticky
  >
    <template #actions>
      <div class="flex flex-wrap items-center gap-3">
        <div class="inline-flex max-w-full items-center gap-2 rounded-md border bg-muted/35 px-2.5 py-1 text-xs text-muted-foreground">
          <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500" />
          <span class="shrink-0">{{ adminText('k002q') }}</span>
          <span class="truncate font-semibold text-foreground">{{ serverVersionLoading ? adminText('k004e') : serverVersion?.version || 'dev' }}</span>
          <span
            v-if="!serverVersionLoading"
            class="inline-flex h-5 shrink-0 items-center rounded-md bg-secondary px-1.5 text-[10px] font-medium text-secondary-foreground"
          >
            {{ modeLabel(serverVersion?.mode) }}
          </span>
          <span v-if="!serverVersionLoading && shortCommit(serverVersion?.commit)" class="hidden text-muted-foreground sm:inline">
            #{{ shortCommit(serverVersion?.commit) }}
          </span>
        </div>
      </div>
    </template>

      <StatsFilterBar
        class="mb-4"
        :presets="presets"
        :active-preset="activePreset"
        :start-date="filter.startDate"
        :end-date="filter.endDate"
        :category-id="filter.categoryId"
        :category-options="categoryOptions"
        :timezone="timezone"
        :max-range-days="maxRangeDays"
        :invalid-message="invalidMessage"
        @select-preset="selectPreset"
        @update:start-date="filter.startDate = $event"
        @update:end-date="filter.endDate = $event"
        @update:category-id="filter.categoryId = $event"
      />

      <AdminSection class="mb-4">
        <p class="border-b bg-muted/10 px-4 py-2 text-xs text-muted-foreground">
          {{ t('statsAdmin.overviewNote') }}
        </p>
        <div class="grid sm:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="(item, index) in summaryItems"
          :key="item.label"
          class="flex min-h-20 items-center justify-between border-b px-4 py-3 text-card-foreground sm:[&:nth-child(2n+1)]:border-r xl:border-b-0 xl:border-r xl:last:border-r-0"
          :class="index >= 2 ? 'sm:border-b-0' : ''"
        >
          <div class="min-w-0">
            <div class="mb-2 flex items-center gap-1.5 text-[13px] font-medium text-muted-foreground">
              <component :is="item.icon" class="h-4 w-4" />
              <span>{{ item.label }}</span>
            </div>
            <div class="flex items-baseline gap-1.5">
              <span class="text-2xl font-bold leading-none tracking-tight">{{ formatNumber(item.value) }}</span>
              <span v-if="item.delta" class="text-xs font-medium text-muted-foreground">{{ item.delta }}</span>
            </div>
          </div>
        </div>
        </div>
      </AdminSection>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div class="min-w-0">
 <TrafficOverview :data="traffic" :loading="trafficLoading">
            <template #headerAction>
              <p class="max-w-[320px] text-xs text-muted-foreground sm:text-right">{{ t('statsAdmin.trafficNote') }}</p>
            </template>
          </TrafficOverview>
        </div>
        <div class="min-w-0">
          <ProjectVersion
            :releases="releases"
            :loading="releasesLoading"
            :error="releasesError"
          />
        </div>
      </div>

      <div class="mt-4">
        <CategoryStatsSection
          :filter="activeFilter"
          :summary="categorySummary"
          :summary-loading="categorySummaryLoading"
          :summary-error="categorySummaryError"
          @retry="loadCategorySummary"
          @select-category="filter.categoryId = $event"
        />
      </div>

      <div class="mt-4">
        <UserStatsSection :filter="activeFilter" />
      </div>

      <AdminSection class="mt-4" body-class="px-4 py-3">
        <template #header>
          <h2 class="text-sm font-semibold">{{ t('statsAdmin.rulesTitle') }}</h2>
        </template>
        <ul class="grid gap-1.5 text-xs leading-relaxed text-muted-foreground">
          <li v-for="rule in rules" :key="rule" class="flex gap-2">
            <span aria-hidden="true">·</span>
            <span>{{ rule }}</span>
          </li>
        </ul>
      </AdminSection>
    </BasicPage>
</template>

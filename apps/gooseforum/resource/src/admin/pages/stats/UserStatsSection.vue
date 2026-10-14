<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import AdminSelect from '@/admin/components/AdminSelect.vue'
import AdminToolbar from '@/admin/components/AdminToolbar.vue'
import ManagementTable from '@/admin/pages/management/ManagementTable.vue'
import UserDetailPanel from '@/admin/pages/stats/UserDetailPanel.vue'
import { getAdminStatsUserRanking } from '@/admin/runtime/api'
import type {
  AdminStatsActorType,
  AdminStatsFilter,
  AdminStatsUserRankingResult,
  AdminStatsUserSort,
} from '@/admin/runtime/api'

const props = defineProps<{ filter: AdminStatsFilter }>()

const { t } = useI18n()

type RankingRow = AdminStatsUserRankingResult['items'][number]

const rows = ref<RankingRow[]>([])
const total = ref(0)
const loading = ref(false)
const error = ref('')
const actorType = ref<AdminStatsActorType>(0)
const sortBy = ref<AdminStatsUserSort>('topics')
const page = ref(1)
const pageSize = ref(20)
const query = ref('')
const searchError = ref('')
const target = ref<{ userId?: number, username?: string } | null>(null)
// 请求序号守卫：快速切换筛选/排序/分页时，旧响应不得覆盖新结果（AC-8）。
let rankSeq = 0

const columns = computed(() => [
  t('statsAdmin.rankLabel'),
  t('statsAdmin.colUser'),
  t('statsAdmin.colTopics'),
  t('statsAdmin.colReplies'),
  t('statsAdmin.colLikes'),
  t('statsAdmin.colActions'),
])
const actorOptions = computed(() => [
  { value: 0, label: t('statsAdmin.actorHuman') },
  { value: 1, label: t('statsAdmin.actorAgent') },
])
const sortOptions = computed(() => [
  { value: 'topics', label: t('statsAdmin.sortTopics') },
  { value: 'replies', label: t('statsAdmin.sortReplies') },
  { value: 'likes', label: t('statsAdmin.sortLikes') },
])

function errorText(error: unknown) {
  return error instanceof Error && error.message ? error.message : t('statsAdmin.loadFailed')
}

function rankOf(index: number) {
  return (page.value - 1) * pageSize.value + index + 1
}

async function loadRanking() {
  const seq = ++rankSeq
  loading.value = true
  error.value = ''
  try {
    const result = await getAdminStatsUserRanking({
      startDate: props.filter.startDate,
      endDate: props.filter.endDate,
      categoryId: props.filter.categoryId,
      sortBy: sortBy.value,
      page: page.value,
      pageSize: pageSize.value,
      actorType: actorType.value,
    })
    if (seq !== rankSeq) return
    // 排序由后端决定（同值按 user_id 升序），前端保持返回顺序不再重排（AC-3）。
    rows.value = result.items ?? []
    total.value = result.total ?? 0
  } catch (fetchError) {
    if (seq !== rankSeq) return
    rows.value = []
    total.value = 0
    error.value = errorText(fetchError)
  } finally {
    if (seq === rankSeq) loading.value = false
  }
}

function changeActorType(value: number) {
  actorType.value = value === 1 ? 1 : 0
  page.value = 1
}

function changeSort(value: string) {
  sortBy.value = value === 'replies' || value === 'likes' ? value : 'topics'
  page.value = 1
}

function changePageSize(value: number) {
  pageSize.value = value
  page.value = 1
}

function openDetail(userId: number) {
  target.value = { userId }
}

function submitSearch() {
  const value = query.value.trim()
  if (!value) {
    searchError.value = t('statsAdmin.searchRequired')
    return
  }
  searchError.value = ''
  // 契约要求 userId 与 username 恰给一个：纯数字按用户 ID，其余按用户名查询。
  target.value = /^\d+$/.test(value) ? { userId: Number(value) } : { username: value }
}

watch(
  () => [props.filter.startDate, props.filter.endDate, props.filter.categoryId, actorType.value, sortBy.value, page.value, pageSize.value],
  loadRanking,
  { immediate: true },
)
</script>

<template>
  <div class="grid gap-4" :class="target ? 'xl:grid-cols-[minmax(0,1fr)_minmax(0,360px)]' : ''">
    <div class="min-w-0">
      <ManagementTable
        :columns="columns"
        :loading="loading && rows.length === 0"
        :error="error"
        :empty-text="t('statsAdmin.userEmpty')"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @retry="loadRanking"
        @update:page="page = $event"
        @update:page-size="changePageSize"
      >
        <template #header>
          <AdminToolbar class="-mx-3 -my-2 border-b-0">
            <div class="min-w-0">
              <h2 class="text-sm font-semibold">{{ t('statsAdmin.userTitle') }}</h2>
              <p class="mt-0.5 text-xs text-muted-foreground">{{ t('statsAdmin.userHint') }}</p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <AdminSelect
                :model-value="actorType"
                :options="actorOptions"
                size="sm"
                :aria-label="t('statsAdmin.actorTypeLabel')"
                @update:model-value="changeActorType($event)"
              />
              <AdminSelect
                :model-value="sortBy"
                :options="sortOptions"
                size="sm"
                :aria-label="t('statsAdmin.sortLabel')"
                @update:model-value="changeSort($event)"
              />
              <form class="flex items-center gap-2" @submit.prevent="submitSearch">
                <input
                  v-model="query"
                  class="h-8 w-full min-w-40 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  :placeholder="t('statsAdmin.searchPlaceholder')"
                  :aria-label="t('statsAdmin.searchPlaceholder')"
                />
                <Button type="submit" size="sm">{{ t('statsAdmin.searchAction') }}</Button>
              </form>
            </div>
          </AdminToolbar>
        </template>

        <tr v-if="!loading && !error && rows.length === 0">
          <td :colspan="columns.length" class="h-24 px-4 text-center text-muted-foreground">
            {{ t('statsAdmin.userEmpty') }}
          </td>
        </tr>
        <tr v-for="(row, index) in rows" :key="row.userId" class="hover:bg-muted/35">
          <td class="px-4 py-3 align-middle text-xs tabular-nums text-muted-foreground">{{ rankOf(index) }}</td>
          <td class="px-4 py-3 align-middle">
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ row.nickname || row.username }}</span>
              <Badge v-if="Number(row.actorType) === 1" variant="secondary">{{ t('statsAdmin.actorAgent') }}</Badge>
            </div>
            <div class="text-xs text-muted-foreground">@{{ row.username }} · ID {{ row.userId }}</div>
          </td>
          <td class="px-4 py-3 align-middle tabular-nums">{{ row.topicCount }}</td>
          <td class="px-4 py-3 align-middle tabular-nums">{{ row.replyCount }}</td>
          <td class="px-4 py-3 align-middle tabular-nums">{{ row.likeCount }}</td>
          <td class="px-4 py-3 align-middle">
            <Button variant="outline" size="sm" class="h-8 text-xs" type="button" @click="openDetail(Number(row.userId))">
              {{ t('statsAdmin.detailAction') }}
            </Button>
          </td>
        </tr>
      </ManagementTable>

      <p v-if="searchError" data-testid="stats-search-error" class="mt-2 text-xs font-medium text-destructive">
        {{ searchError }}
      </p>
    </div>

    <UserDetailPanel
      v-if="target"
      :filter="filter"
      :target="target"
      class="min-w-0"
      @close="target = null"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshCw, X } from '@lucide/vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import { getAdminStatsUserDetail } from '@/admin/runtime/api'
import type { AdminStatsFilter, AdminStatsUserDetailResult } from '@/admin/runtime/api'

const props = defineProps<{
  filter: AdminStatsFilter
  /** userId 与 username 恰给一个（与契约一致）。 */
  target: { userId?: number, username?: string }
}>()

const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()

const detail = ref<AdminStatsUserDetailResult>()
const loading = ref(false)
const error = ref('')
// 请求序号守卫：连续查询不同用户时，旧响应不得覆盖新结果（AC-8）。
let detailSeq = 0

const summaryItems = computed(() => {
  const summary = detail.value?.summary
  return [
    { label: t('statsAdmin.colTopics'), value: summary?.topicCount ?? 0 },
    { label: t('statsAdmin.colReplies'), value: summary?.replyCount ?? 0 },
    { label: t('statsAdmin.colLikes'), value: summary?.likeCount ?? 0 },
  ]
})

const days = computed(() => detail.value?.days ?? [])
const estimatedDays = computed(() => days.value.filter(day => day.estimated).length)
const isAgent = computed(() => Number(detail.value?.user?.actorType) === 1)

function errorText(error: unknown) {
  return error instanceof Error && error.message ? error.message : t('statsAdmin.loadFailed')
}

async function loadDetail() {
  const seq = ++detailSeq
  loading.value = true
  error.value = ''
  try {
    const result = await getAdminStatsUserDetail({
      startDate: props.filter.startDate,
      endDate: props.filter.endDate,
      categoryId: props.filter.categoryId,
      ...(props.target.userId ? { userId: props.target.userId } : { username: props.target.username }),
    })
    if (seq !== detailSeq) return
    detail.value = result
  } catch (fetchError) {
    if (seq !== detailSeq) return
    detail.value = undefined
    error.value = errorText(fetchError)
  } finally {
    if (seq === detailSeq) loading.value = false
  }
}

watch(() => [props.target, props.filter.startDate, props.filter.endDate, props.filter.categoryId], loadDetail, { immediate: true })
</script>

<template>
  <AdminSection body-class="px-4 pb-4 pt-3">
    <template #header>
      <div class="flex items-start justify-between gap-2">
        <div class="min-w-0">
          <h2 class="text-sm font-semibold">{{ t('statsAdmin.detailTitle') }}</h2>
          <p v-if="detail?.user" class="mt-0.5 truncate text-xs text-muted-foreground">
            <span class="font-medium text-foreground">{{ detail.user.nickname || detail.user.username }}</span>
            <span> · @{{ detail.user.username }} · ID {{ detail.user.userId }}</span>
          </p>
        </div>
        <div class="flex shrink-0 items-center gap-1">
          <Badge v-if="isAgent" variant="secondary">{{ t('statsAdmin.actorAgent') }}</Badge>
          <Button variant="ghost" size="icon-sm" type="button" :title="t('statsAdmin.detailClose')" @click="emit('close')">
            <X class="size-4" />
          </Button>
        </div>
      </div>
    </template>

    <p v-if="loading" class="flex h-32 items-center justify-center text-sm text-muted-foreground">
      {{ t('common.loadingShort') }}
    </p>
    <div v-else-if="error" class="flex h-32 flex-col items-center justify-center gap-2 text-sm text-destructive">
      <span class="text-center">{{ error }}</span>
      <Button variant="outline" size="sm" type="button" @click="loadDetail">
        <RefreshCw class="size-3.5" />
        {{ t('statsAdmin.detailRetry') }}
      </Button>
    </div>
    <div v-else class="grid gap-3">
      <div class="grid grid-cols-3 gap-2">
        <div v-for="item in summaryItems" :key="item.label" class="rounded-md border bg-muted/25 px-3 py-2">
          <div class="text-xs text-muted-foreground">{{ item.label }}</div>
          <div class="text-lg font-semibold leading-tight tabular-nums">{{ item.value }}</div>
        </div>
      </div>

      <div class="flex items-center justify-between gap-2">
        <h3 class="text-xs font-semibold text-muted-foreground">{{ t('statsAdmin.detailDaysTitle') }}</h3>
        <Badge v-if="estimatedDays > 0" variant="secondary">{{ t('statsAdmin.estimatedBadge') }}</Badge>
      </div>

      <p v-if="days.length === 0" class="py-6 text-center text-sm text-muted-foreground">
        {{ t('statsAdmin.detailEmpty') }}
      </p>
      <div v-else class="max-h-[320px] overflow-auto rounded-md border">
        <table class="w-full text-xs tabular-nums">
          <thead class="sticky top-0 z-10 border-b bg-muted/60 text-muted-foreground">
            <tr>
              <th class="h-8 px-3 text-left font-medium">{{ t('statsAdmin.detailColDate') }}</th>
              <th class="h-8 px-2 text-right font-medium">{{ t('statsAdmin.colTopics') }}</th>
              <th class="h-8 px-2 text-right font-medium">{{ t('statsAdmin.colReplies') }}</th>
              <th class="h-8 px-3 text-right font-medium">{{ t('statsAdmin.detailSource') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="day in days" :key="day.date" :data-estimated="day.estimated ? 'true' : 'false'">
              <td class="whitespace-nowrap px-3 py-1 text-muted-foreground">{{ day.date }}</td>
              <td class="px-2 py-1 text-right">{{ day.topicCount }}</td>
              <td class="px-2 py-1 text-right">{{ day.replyCount }}</td>
              <td class="px-3 py-1 text-right">
                <span
                  :class="day.estimated ? 'font-medium text-amber-600 dark:text-amber-500' : 'text-muted-foreground'"
                >
                  {{ day.estimated ? t('statsAdmin.detailEstimated') : t('statsAdmin.detailExact') }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <p v-if="estimatedDays > 0" class="text-xs leading-relaxed text-muted-foreground">
        {{ t('statsAdmin.estimatedNote') }}
      </p>
    </div>
  </AdminSection>
</template>

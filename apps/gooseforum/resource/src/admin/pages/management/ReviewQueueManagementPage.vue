<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onMounted, ref } from 'vue'
import { Check, ChevronLeft, ChevronRight, RefreshCw, X } from '@lucide/vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/admin/components/ui/table'
import { getReviewQueue, reviewAction } from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AdminPayload, ManageHomeProps, ReviewQueueItem } from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

type ReviewKind = 'topic' | 'post'

const kind = ref<ReviewKind>('topic')
const rows = ref<ReviewQueueItem[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const actionRow = ref<{ item: ReviewQueueItem, approve: boolean } | null>(null)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

async function loadQueue() {
  loading.value = true
  error.value = ''
  try {
    const result = await getReviewQueue(kind.value, page.value, pageSize.value)
    rows.value = result.items || []
    total.value = result.total || 0
    page.value = result.page || page.value
    pageSize.value = result.pageSize || pageSize.value
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('k00gd')
  } finally {
    loading.value = false
  }
}

function switchKind(next: ReviewKind) {
  if (kind.value === next) return
  kind.value = next
  page.value = 1
  void loadQueue()
}

function updatePage(value: number) {
  page.value = value
  void loadQueue()
}

function updatePageSize(value: number) {
  pageSize.value = value
  page.value = 1
  void loadQueue()
}

async function confirmAction() {
  if (!actionRow.value) return
  const { item, approve } = actionRow.value
  saving.value = true
  try {
    await reviewAction(kind.value, item.id, approve)
    actionRow.value = null
    await loadQueue()
    adminToast.success(approve ? adminText('k00gj') : adminText('k00gk'))
  } catch (err) {
    adminToast.error(err, adminText('k00gn'))
  } finally {
    saving.value = false
  }
}

onMounted(loadQueue)
</script>

<template>
  <BasicPage :title="adminText('k00ge')" :description="adminText('k00gf')" sticky>
    <template #actions>
      <Button variant="outline" size="sm" type="button" :disabled="loading" @click="loadQueue">
        <RefreshCw class="size-4" :class="loading ? 'animate-spin' : ''" />
        {{ adminText('k004q') }}
      </Button>
    </template>

    <AdminSection>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3 px-1 py-1">
          <div class="inline-flex rounded-lg border bg-muted/20 p-1">
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="kind === 'topic' ? 'bg-background shadow-xs' : 'text-muted-foreground hover:text-foreground'"
              @click="switchKind('topic')"
            >
              {{ adminText('k00gg') }}
            </button>
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="kind === 'post' ? 'bg-background shadow-xs' : 'text-muted-foreground hover:text-foreground'"
              @click="switchKind('post')"
            >
              {{ adminText('k00gh') }}
            </button>
          </div>
          <div class="flex items-center gap-2">
            <select
              class="h-9 rounded-md border bg-background px-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              :value="pageSize"
              @change="updatePageSize(Number(($event.target as HTMLSelectElement).value))"
            >
              <option :value="10">{{ adminText('k002x') }}</option>
              <option :value="20">{{ adminText('k002y') }}</option>
              <option :value="30">{{ adminText('k002z') }}</option>
              <option :value="50">{{ adminText('k0030') }}</option>
            </select>
            <Button
              variant="outline"
              size="icon"
              type="button"
              :disabled="page <= 1"
              @click="updatePage(page - 1)"
            >
              <ChevronLeft class="size-4" />
            </Button>
            <span class="min-w-16 text-center text-sm text-muted-foreground">{{ adminText('k0056') }} {{ page }} / {{ totalPages }} {{ adminText('k0057') }}</span>
            <Button
              variant="outline"
              size="icon"
              type="button"
              :disabled="page >= totalPages"
              @click="updatePage(page + 1)"
            >
              <ChevronRight class="size-4" />
            </Button>
          </div>
        </div>
      </template>

      <Table class="hidden md:table">
        <TableHeader class="bg-muted/30">
          <TableRow>
            <TableHead class="w-16 px-3">ID</TableHead>
            <TableHead class="w-[180px]">{{ adminText('k00i5') }}</TableHead>
            <TableHead>{{ adminText('k00gi') }}</TableHead>
            <TableHead class="w-[140px]">{{ adminText('k00i6') }}</TableHead>
            <TableHead class="w-[150px]">{{ adminText('k00i7') }}</TableHead>
            <TableHead class="w-[150px] text-right pr-3">{{ adminText('k007m') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-if="loading && rows.length === 0">
            <TableCell colspan="6" class="h-28 text-center text-muted-foreground">{{ adminText('k0046') }}</TableCell>
          </TableRow>
          <TableRow v-else-if="error">
            <TableCell colspan="6" class="h-28 text-center text-destructive">{{ error }}</TableCell>
          </TableRow>
          <TableRow v-else-if="rows.length === 0">
            <TableCell colspan="6" class="h-28 text-center text-muted-foreground">{{ adminText('k00i8') }}</TableCell>
          </TableRow>
          <template v-else>
            <TableRow v-for="item in rows" :key="item.id" class="hover:bg-muted/20">
              <TableCell class="px-3 py-2 font-mono text-xs text-muted-foreground">{{ item.id }}</TableCell>
              <TableCell class="max-w-0 py-2">
                <div class="min-w-0 space-y-1">
                  <div class="truncate text-sm font-medium">{{ item.title || '-' }}</div>
                  <Badge v-if="item.postNo" variant="secondary" class="px-1.5 py-0 text-[10px]">#{{ item.postNo }}</Badge>
                </div>
              </TableCell>
              <TableCell class="max-w-0 py-2">
                <p class="line-clamp-2 text-xs leading-4 text-muted-foreground">{{ item.excerpt || '-' }}</p>
              </TableCell>
              <TableCell class="py-2 text-sm">{{ item.username || `#${item.userId}` }}</TableCell>
              <TableCell class="py-2 text-xs text-muted-foreground">{{ formatTime(item.createdAt) }}</TableCell>
              <TableCell class="py-2 pr-3">
                <div class="flex justify-end gap-1.5">
                  <Button type="button" size="sm" variant="outline" class="h-8 text-xs" @click="actionRow = { item, approve: true }">
                    <Check class="size-3.5" />
                    {{ adminText('k00gj') }}
                  </Button>
                  <Button type="button" size="sm" variant="outline" class="h-8 text-xs text-destructive hover:text-destructive" @click="actionRow = { item, approve: false }">
                    <X class="size-3.5" />
                    {{ adminText('k00gk') }}
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>

      <div class="md:hidden">
        <div v-if="loading && rows.length === 0" class="px-3 py-10 text-center text-sm text-muted-foreground">{{ adminText('k0046') }}</div>
        <div v-else-if="error" class="px-3 py-10 text-center text-sm text-destructive">{{ error }}</div>
        <div v-else-if="rows.length === 0" class="px-3 py-10 text-center text-sm text-muted-foreground">{{ adminText('k00i8') }}</div>
        <div v-else class="divide-y">
          <article v-for="item in rows" :key="item.id" class="space-y-2 px-3 py-3">
            <div class="flex min-w-0 items-start justify-between gap-3">
              <div class="min-w-0 flex-1 space-y-1">
                <div class="flex min-w-0 items-center gap-1.5">
                  <span class="font-mono text-xs text-muted-foreground">#{{ item.id }}</span>
                  <span class="min-w-0 truncate text-[15px] font-semibold leading-5">{{ item.title || '-' }}</span>
                  <Badge v-if="item.postNo" variant="secondary" class="h-5 shrink-0 rounded-full px-1.5 text-[10px]">#{{ item.postNo }}</Badge>
                </div>
                <p class="line-clamp-2 break-words text-[12px] leading-5 text-muted-foreground">{{ item.excerpt || '-' }}</p>
              </div>
            </div>
            <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground">
              <span>{{ item.username || `#${item.userId}` }} · {{ formatTime(item.createdAt) }}</span>
              <div class="flex shrink-0 items-center gap-1.5">
                <Button type="button" size="sm" variant="outline" class="h-7 text-xs" @click="actionRow = { item, approve: true }">
                  <Check class="size-3.5" />
                  {{ adminText('k00gj') }}
                </Button>
                <Button type="button" size="sm" variant="outline" class="h-7 text-xs text-destructive hover:text-destructive" @click="actionRow = { item, approve: false }">
                  <X class="size-3.5" />
                  {{ adminText('k00gk') }}
                </Button>
              </div>
            </div>
          </article>
        </div>
      </div>
    </AdminSection>

    <AdminConfirmDialog
      :open="actionRow !== null"
      :title="actionRow?.approve ? adminText('k00gj') : adminText('k00gk')"
      :description="actionRow?.approve ? adminText('k00gl') : adminText('k00gm')"
      :confirm-text="actionRow?.approve ? adminText('k00gj') : adminText('k00gk')"
      :loading="saving"
      @update:open="(open) => !open && (actionRow = null)"
      @confirm="confirmAction"
    />
  </BasicPage>
</template>

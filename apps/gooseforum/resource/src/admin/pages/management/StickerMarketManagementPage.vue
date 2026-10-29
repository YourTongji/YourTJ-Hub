<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Eye, Loader2, RefreshCw, Search } from '@lucide/vue'
import AdminActionButton from '@/admin/components/AdminActionButton.vue'
import AdminSelect from '@/admin/components/AdminSelect.vue'
import AdminToolbar from '@/admin/components/AdminToolbar.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Switch } from '@/admin/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/admin/components/ui/dialog'
import ManagementTable from './ManagementTable.vue'
import StickerMarketImportProgress from './StickerMarketImportProgress.vue'
import {
  getStickerMarket,
  getStickerMarketPack,
  importStickerMarketPack,
  saveStickerMarketItem,
  saveStickerMarketPack,
  syncStickerMarket,
} from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type {
  AdminPayload,
  AdminStickerMarketItem,
  AdminStickerMarketPack,
  AdminStickerMarketPackDetail,
  AdminStickerMarketPage,
  AdminStickerMarketStatus,
  ManageHomeProps,
} from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

// Imports run in the task queue; poll while any visible pack is still waiting or running.
const POLL_INTERVAL_MS = 1500

const loading = ref(false)
const syncing = ref(false)
const error = ref('')
const market = ref<AdminStickerMarketPage | null>(null)
const keyword = ref('')
const appliedKeyword = ref('')
const topic = ref('')
const status = ref('')
const page = ref(1)
const pageSize = ref(20)
const busyPackId = ref<number | null>(null)
const detail = ref<AdminStickerMarketPackDetail | null>(null)
const detailLoading = ref(false)
const busyItemId = ref<number | null>(null)
let pollTimer: ReturnType<typeof setTimeout> | undefined
// Cover URLs that failed to load; the cell falls back to the next candidate.
const brokenCovers = ref(new Set<string>())

const packs = computed(() => market.value?.packs ?? [])
const topicLabels = computed(() => new Map((market.value?.topics ?? []).map((item) => [item.id, item.label])))

const tableColumns = computed(() => [
  adminText('stickerMarketColumnCover'),
  adminText('stickerMarketColumnPack'),
  adminText('stickerMarketColumnTopic'),
  adminText('stickerMarketColumnStickers'),
  adminText('k007j'),
  adminText('stickerMarketColumnPublished'),
  adminText('k00bf'),
  adminText('k007m'),
])

const topicOptions = computed(() => [
  { value: '', label: adminText('stickerMarketAllTopics') },
  ...(market.value?.topics ?? []).map((item) => ({ value: item.id, label: item.label })),
])

const statusTexts: Record<AdminStickerMarketStatus, string> = {
  new: 'stickerMarketStatusNew',
  queued: 'stickerMarketStatusQueued',
  importing: 'stickerMarketStatusImporting',
  ready: 'stickerMarketStatusReady',
  partial: 'stickerMarketStatusPartial',
  failed: 'stickerMarketStatusFailed',
}

const statusOptions = computed(() => [
  { value: '', label: adminText('stickerMarketAllStatuses') },
  ...(Object.keys(statusTexts) as AdminStickerMarketStatus[]).map((value) => ({ value, label: adminText(statusTexts[value]) })),
])

const reasonTexts: Record<string, string> = {
  timeout: 'stickerMarketReasonTimeout',
  blocked: 'stickerMarketReasonBlocked',
  tooLarge: 'stickerMarketReasonTooLarge',
  unsupported: 'stickerMarketReasonUnsupported',
  invalid: 'stickerMarketReasonInvalid',
  status: 'stickerMarketReasonStatus',
  storage: 'stickerMarketReasonStorage',
  unavailable: 'stickerMarketReasonUnavailable',
  empty: 'stickerMarketReasonEmpty',
}

// Market messageCodes are admin-only, so the page localizes them like the sticker library does.
const marketMessageTexts: Record<string, string> = {
  'admin.sticker.market.busy': 'stickerMarketBusy',
  'admin.sticker.market.disabled': 'stickerMarketDisabled',
  'admin.sticker.market.unavailable': 'stickerMarketUnavailable',
  'admin.sticker.market.notReady': 'stickerMarketNotReady',
  'admin.sticker.notFound': 'k00vh1',
}

function reasonText(reason: string) {
  const key = reasonTexts[reason]
  return key ? adminText(key) : reason
}

function statusText(value: AdminStickerMarketStatus) {
  return adminText(statusTexts[value] ?? value)
}

function statusVariant(value: AdminStickerMarketStatus): 'secondary' | 'destructive' | 'outline' {
  if (value === 'failed') return 'destructive'
  if (value === 'ready' || value === 'queued' || value === 'importing') return 'secondary'
  return 'outline'
}

// Imported packs show the stored cover members see; before that, the upstream catalog cover.
function coverSource(pack: AdminStickerMarketPack) {
  return [pack.coverUrl, pack.iconUrl].find((url) => url && !brokenCovers.value.has(url)) ?? ''
}

function markCoverBroken(url: string) {
  brokenCovers.value = new Set(brokenCovers.value).add(url)
}

function isRunning(pack: AdminStickerMarketPack) {
  return pack.status === 'queued' || pack.status === 'importing'
}

function importLabel(pack: AdminStickerMarketPack) {
  if (pack.status === 'failed' || pack.status === 'partial') return adminText('stickerMarketRetryImport')
  if (pack.status === 'ready') return adminText('stickerMarketImportAgain')
  return adminText('stickerMarketImport')
}

function marketToastError(err: unknown, fallbackKey: string) {
  const messageCode = (err as { messageCode?: string } | null)?.messageCode || ''
  const key = marketMessageTexts[messageCode]
  if (key) {
    adminToast.warning(adminText(key))
    return
  }
  adminToast.error(err, adminText(fallbackKey))
}

function formatTime(value: string | null | undefined) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function schedulePoll() {
  clearTimeout(pollTimer)
  pollTimer = undefined
  if (packs.value.some(isRunning) || (detail.value && isRunning(detail.value.pack))) {
    pollTimer = setTimeout(() => void refresh(true), POLL_INTERVAL_MS)
  }
}

async function refresh(quiet = false) {
  if (!quiet) {
    loading.value = true
    error.value = ''
  }
  try {
    market.value = await getStickerMarket({
      topic: topic.value,
      q: appliedKeyword.value,
      status: status.value,
      page: page.value,
      pageSize: pageSize.value,
    })
    if (detail.value) {
      const current = market.value.packs.find((pack) => pack.id === detail.value?.pack.id)
      const shown = detail.value.pack
      // Items arrive one by one during an import; reload the dialog whenever its counts move.
      if (current && (current.status !== shown.status || current.itemCount !== shown.itemCount || current.failedCount !== shown.failedCount)) {
        detail.value = await getStickerMarketPack(current.id)
      }
    }
  } catch (err) {
    if (!quiet) error.value = err instanceof Error ? err.message : adminText('stickerMarketLoadFailed')
  } finally {
    loading.value = false
    schedulePoll()
  }
}

function applySearch() {
  appliedKeyword.value = keyword.value.trim()
  page.value = 1
  void refresh()
}

function clearSearch() {
  keyword.value = ''
  applySearch()
}

watch([topic, status], () => {
  page.value = 1
  void refresh()
})

function updatePage(value: number) {
  page.value = value
  void refresh()
}

function updatePageSize(value: number) {
  pageSize.value = value
  page.value = 1
  void refresh()
}

async function syncCatalog() {
  syncing.value = true
  try {
    const result = await syncStickerMarket()
    adminToast.success(result.missing > 0
      ? adminText('stickerMarketSyncedWithMissing', { packs: result.packs, missing: result.missing })
      : adminText('stickerMarketSynced', { packs: result.packs }))
    await refresh()
  } catch (err) {
    marketToastError(err, 'stickerMarketLoadFailed')
  } finally {
    syncing.value = false
  }
}

function replacePack(next: AdminStickerMarketPack) {
  if (!market.value) return
  market.value.packs = market.value.packs.map((pack) => (pack.id === next.id ? next : pack))
  if (detail.value?.pack.id === next.id) detail.value = { ...detail.value, pack: next }
}

async function queueImport(pack: AdminStickerMarketPack) {
  busyPackId.value = pack.id
  try {
    replacePack(await importStickerMarketPack(pack.id))
    adminToast.success(adminText('stickerMarketImportQueued'))
    schedulePoll()
  } catch (err) {
    marketToastError(err, 'stickerMarketSaveFailed')
  } finally {
    busyPackId.value = null
  }
}

async function savePack(pack: AdminStickerMarketPack, changes: { published?: boolean, sortOrder?: number }) {
  const published = changes.published ?? pack.published
  const sortOrder = changes.sortOrder ?? pack.sortOrder
  if (published === pack.published && sortOrder === pack.sortOrder) return
  busyPackId.value = pack.id
  try {
    replacePack(await saveStickerMarketPack({ id: pack.id, published, sortOrder }))
    if (changes.published !== undefined) {
      adminToast.success(adminText(published ? 'stickerMarketPublished' : 'stickerMarketUnpublished', { name: pack.name }))
    }
  } catch (err) {
    marketToastError(err, 'stickerMarketSaveFailed')
  } finally {
    busyPackId.value = null
  }
}

function saveSortOrder(pack: AdminStickerMarketPack, event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  if (!Number.isInteger(value)) {
    (event.target as HTMLInputElement).value = String(pack.sortOrder)
    return
  }
  void savePack(pack, { sortOrder: value })
}

async function openDetail(pack: AdminStickerMarketPack) {
  detail.value = { pack, items: [] }
  detailLoading.value = true
  try {
    detail.value = await getStickerMarketPack(pack.id)
  } catch (err) {
    detail.value = null
    marketToastError(err, 'stickerMarketLoadFailed')
  } finally {
    detailLoading.value = false
    schedulePoll()
  }
}

async function toggleItem(item: AdminStickerMarketItem, isEnabled: boolean) {
  busyItemId.value = item.stickerId
  try {
    await saveStickerMarketItem(item.stickerId, isEnabled)
    item.isEnabled = isEnabled
    adminToast.success(adminText(isEnabled ? 'stickerMarketItemRestored' : 'stickerMarketItemWithdrawn'))
  } catch (err) {
    marketToastError(err, 'stickerMarketSaveFailed')
  } finally {
    busyItemId.value = null
  }
}

onMounted(() => {
  void refresh()
})

onBeforeUnmount(() => {
  clearTimeout(pollTimer)
})
</script>

<template>
  <BasicPage :title="adminText('stickerMarketTitle')" :description="adminText('stickerMarketDescription')" sticky>
    <template #actions>
      <div class="flex items-center gap-2">
        <Button variant="outline" type="button" @click="refresh()">
          <RefreshCw class="size-4" />
          {{ adminText('k004q') }}
        </Button>
        <Button type="button" :disabled="syncing || market?.enabled === false" @click="syncCatalog">
          <Loader2 v-if="syncing" class="size-4 animate-spin" />
          <RefreshCw v-else class="size-4" />
          {{ syncing ? adminText('stickerMarketSyncing') : adminText('stickerMarketSync') }}
        </Button>
      </div>
    </template>

    <p v-if="market && !market.enabled" role="status" class="mb-3 rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
      {{ adminText('stickerMarketDisabledNotice') }}
    </p>

    <div class="mb-3 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
      <Badge variant="secondary">{{ adminText('stickerMarketTotal', { total: market?.total ?? 0 }) }}</Badge>
      <span v-if="market?.lastSyncedAt">{{ adminText('stickerMarketLastSync', { time: formatTime(market.lastSyncedAt) }) }}</span>
      <span v-else-if="market">{{ adminText('stickerMarketNeverSynced') }}</span>
    </div>

    <ManagementTable
      :columns="tableColumns"
      :loading="loading"
      :error="error"
      :total="market?.total ?? 0"
      :page="page"
      :page-size="pageSize"
      @update:page="updatePage"
      @update:page-size="updatePageSize"
      @retry="refresh()"
    >
      <template #header>
        <AdminToolbar class="-mx-3 -my-2 border-b-0">
          <form class="flex min-w-0 flex-1 items-center gap-2" @submit.prevent="applySearch">
            <div class="relative min-w-0 flex-1 sm:max-w-sm">
              <Search class="pointer-events-none absolute start-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input v-model="keyword" class="ps-8" type="search" :aria-label="adminText('stickerMarketSearch')" :placeholder="adminText('stickerMarketSearch')" />
            </div>
            <Button size="sm" type="submit" variant="outline" class="h-9 px-4">{{ adminText('k00al') }}</Button>
            <Button v-if="appliedKeyword" variant="ghost" size="sm" type="button" class="h-9" @click="clearSearch">
              {{ adminText('k00at') }}
            </Button>
          </form>
          <div class="flex flex-wrap items-center gap-2">
            <AdminSelect v-model="topic" :options="topicOptions" :aria-label="adminText('stickerMarketColumnTopic')" class="w-40" />
            <AdminSelect v-model="status" :options="statusOptions" :aria-label="adminText('k007j')" class="w-36" />
          </div>
        </AdminToolbar>
      </template>

      <tr v-if="!loading && !error && packs.length === 0">
        <td :colspan="tableColumns.length" class="h-24 px-4 text-center text-muted-foreground">
          {{ market?.lastSyncedAt ? adminText('k002v') : adminText('stickerMarketNeverSynced') }}
        </td>
      </tr>
      <tr v-for="pack in packs" :key="pack.id" class="hover:bg-muted/30">
        <td class="px-4 py-2 align-middle">
          <img
            v-if="coverSource(pack)"
            :src="coverSource(pack)"
            alt=""
            referrerpolicy="no-referrer"
            class="size-10 rounded object-contain outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
            loading="lazy"
            @error="markCoverBroken(coverSource(pack))"
          />
          <div v-else-if="pack.iconText" class="flex size-10 items-center justify-center rounded bg-muted text-lg leading-none" aria-hidden="true">{{ pack.iconText }}</div>
          <div v-else class="size-10 rounded bg-muted" aria-hidden="true" />
        </td>
        <td class="max-w-64 px-4 py-2 align-middle">
          <div class="truncate font-medium" :title="pack.name">{{ pack.name }}</div>
          <div class="mt-0.5 flex flex-wrap items-center gap-1.5">
            <span class="truncate font-mono text-xs text-muted-foreground">{{ pack.upstreamId }}</span>
            <Badge v-if="pack.missing" variant="outline">{{ adminText('stickerMarketMissing') }}</Badge>
            <Badge v-if="pack.archived" variant="outline">{{ adminText('stickerMarketArchived') }}</Badge>
          </div>
        </td>
        <td class="px-4 py-2 align-middle text-muted-foreground">{{ topicLabels.get(pack.topic) || pack.topic || '—' }}</td>
        <td class="px-4 py-2 align-middle">
          <div class="tabular-nums">{{ adminText('stickerMarketItemCount', { count: pack.itemCount, total: pack.upstreamCount }) }}</div>
          <div v-if="pack.failedCount > 0" class="text-xs text-muted-foreground">{{ adminText('stickerMarketFailedCount', { count: pack.failedCount }) }}</div>
        </td>
        <td class="px-4 py-2 align-middle">
          <Badge :variant="statusVariant(pack.status)">
            <Loader2 v-if="isRunning(pack)" class="animate-spin" />
            {{ statusText(pack.status) }}
          </Badge>
          <StickerMarketImportProgress v-if="isRunning(pack)" :pack="pack" class="mt-1.5" />
          <div v-else-if="pack.lastError" class="mt-1 text-xs text-muted-foreground">{{ reasonText(pack.lastError) }}</div>
        </td>
        <td class="px-4 py-2 align-middle">
          <Switch
            :model-value="pack.published"
            :disabled="busyPackId === pack.id || (!pack.published && pack.itemCount === 0)"
            :aria-label="adminText('stickerMarketPublishLabel', { name: pack.name })"
            :title="!pack.published && pack.itemCount === 0 ? adminText('stickerMarketPublishNeedsImport') : undefined"
            @update:model-value="(checked: boolean) => savePack(pack, { published: checked })"
          />
        </td>
        <td class="px-4 py-2 align-middle">
          <Input
            :model-value="pack.sortOrder"
            type="number"
            class="h-8 w-20 tabular-nums"
            :disabled="busyPackId === pack.id"
            :aria-label="adminText('stickerMarketSortLabel', { name: pack.name })"
            :title="adminText('stickerMarketSortHint')"
            @change="saveSortOrder(pack, $event)"
          />
        </td>
        <td class="px-4 py-2 align-middle">
          <div class="flex items-center gap-2">
            <AdminActionButton
              :disabled="busyPackId === pack.id || isRunning(pack) || market?.enabled === false || pack.missing"
              @click="queueImport(pack)"
            >
              {{ importLabel(pack) }}
            </AdminActionButton>
            <AdminActionButton compact :title="adminText('stickerMarketViewStickers')" :disabled="pack.status === 'new'" @click="openDetail(pack)">
              <Eye class="size-3.5" />
            </AdminActionButton>
          </div>
        </td>
      </tr>
    </ManagementTable>

    <Dialog :open="detail !== null" @update:open="(open) => !open && (detail = null)">
      <DialogContent class="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{{ detail?.pack.name }}</DialogTitle>
          <DialogDescription>{{ adminText('stickerMarketItemsDescription') }}</DialogDescription>
        </DialogHeader>
        <StickerMarketImportProgress v-if="detail && isRunning(detail.pack)" :pack="detail.pack" />
        <div v-if="detailLoading" class="py-8 text-center text-sm text-muted-foreground">{{ adminText('k0046') }}</div>
        <p v-else-if="detail && detail.items.length === 0" class="py-8 text-center text-sm text-muted-foreground">
          {{ adminText('stickerMarketItemsEmpty') }}
        </p>
        <ul v-else-if="detail" class="grid max-h-[60vh] grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-3 overflow-y-auto p-0.5">
          <li
            v-for="item in detail.items"
            :key="item.upstreamId"
            class="flex flex-col items-center gap-2 rounded-lg border bg-background p-2"
            :class="item.stickerId && !item.isEnabled ? 'opacity-60' : ''"
          >
            <img
              v-if="item.url"
              :src="item.url"
              :alt="item.name"
              class="size-16 rounded object-contain outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
              loading="lazy"
            />
            <div v-else class="size-16 rounded bg-muted" aria-hidden="true" />
            <Switch
              v-if="item.stickerId"
              :model-value="item.isEnabled"
              :disabled="busyItemId === item.stickerId"
              :aria-label="adminText('stickerMarketItemEnabledLabel', { name: item.name })"
              @update:model-value="(checked: boolean) => toggleItem(item, checked)"
            />
            <span v-else class="text-center text-xs text-muted-foreground">{{ adminText('stickerMarketItemNotImported', { reason: reasonText(item.error) }) }}</span>
          </li>
        </ul>
      </DialogContent>
    </Dialog>
  </BasicPage>
</template>

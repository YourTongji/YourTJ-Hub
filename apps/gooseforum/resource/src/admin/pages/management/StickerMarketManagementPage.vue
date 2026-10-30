<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'

import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { Eye, Loader2, Pencil, Plus, RefreshCw, Search, Settings2, Upload } from '@lucide/vue'
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
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/admin/components/ui/dialog'
import ManagementTable from './ManagementTable.vue'
import StickerMarketImportProgress from './StickerMarketImportProgress.vue'
import StickerUploadQueue from './StickerUploadQueue.vue'
import {
  createStickerMarketPack,
  getStickerMarket,
  getStickerMarketPack,
  importStickerMarketPack,
  saveStickerMarketItem,
  saveStickerMarketPack,
  syncStickerMarket,
  uploadStickerMarketFile,
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
// Topic picker value meaning "create a category from the typed label".
const NEW_TOPIC = '__new__'

const loading = ref(false)
const syncing = ref(false)
const error = ref('')
const market = ref<AdminStickerMarketPage | null>(null)
const keyword = ref('')
const appliedKeyword = ref('')
const topic = ref('')
const status = ref('')
const source = ref('')
const page = ref(1)
const pageSize = ref(20)
const busyPackId = ref<number | null>(null)
const detail = ref<AdminStickerMarketPackDetail | null>(null)
const detailLoading = ref(false)
const busyItemId = ref<number | null>(null)
// Inline rename in the pack dialog: one item at a time, error shown under its field.
const renamingId = ref<number | null>(null)
const renameDraft = ref('')
const renameError = ref('')
let pollTimer: ReturnType<typeof setTimeout> | undefined
// Cover URLs that failed to load; the cell falls back to the next candidate.
const brokenCovers = ref(new Set<string>())
// Packs uploaded here: created with a name and category, then filled from images or ZIPs.
const createOpen = ref(false)
const creating = ref(false)
const createForm = reactive({ name: '', topic: '', topicLabel: '' })
const infoEditing = ref<AdminStickerMarketPack | null>(null)
const infoSaving = ref(false)
const infoForm = reactive({ name: '', topic: '', topicLabel: '' })
const uploadTarget = ref<AdminStickerMarketPack | null>(null)
const uploadBusy = ref(false)
const uploadPublishing = ref(false)

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

const sourceOptions = computed(() => [
  { value: '', label: adminText('stickerMarketAllSources') },
  { value: 'fluxdo', label: adminText('stickerMarketSourceFluxdo') },
  { value: 'upload', label: adminText('stickerMarketSourceUpload') },
])

const topicPickerOptions = computed(() => [
  ...(market.value?.topics ?? []).map((item) => ({ value: item.id, label: item.label })),
  { value: NEW_TOPIC, label: adminText('stickerMarketTopicNew') },
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
  'admin.sticker.nameRequired': 'stickerPackNameRequired',
  'common.request.invalidParams': 'stickerMarketTopicInvalid',
}

// Rename failures stay next to the name field instead of a toast.
const nameMessageTexts: Record<string, string> = {
  'admin.sticker.nameInvalid': 'k00vgo',
  'admin.sticker.nameExists': 'stickerMarketNameExists',
  'admin.sticker.nameReserved': 'stickerNameReserved',
}

// Imports name stickers m_ + 40 hex characters until an administrator renames them.
const GENERATED_NAME = /^m_[0-9a-f]{40}$/

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

// Imported packs show the stored cover members see; before that, a server-made
// thumbnail of the upstream cover, so the browser never contacts the upstream host.
function coverSource(pack: AdminStickerMarketPack) {
  return [pack.coverUrl, pack.iconUrl].find((url) => url && !brokenCovers.value.has(url)) ?? ''
}

function markCoverBroken(url: string) {
  brokenCovers.value = new Set(brokenCovers.value).add(url)
}

function isUploaded(pack: AdminStickerMarketPack) {
  return pack.source === 'upload'
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
      source: source.value,
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

watch(detail, (next, previous) => {
  if (next?.pack.id !== previous?.pack.id) cancelRename()
})

watch([topic, status, source], () => {
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
    await saveStickerMarketItem({ stickerId: item.stickerId, isEnabled })
    item.isEnabled = isEnabled
    adminToast.success(adminText(isEnabled ? 'stickerMarketItemRestored' : 'stickerMarketItemWithdrawn'))
  } catch (err) {
    marketToastError(err, 'stickerMarketSaveFailed')
  } finally {
    busyItemId.value = null
  }
}

function renameFieldId(item: AdminStickerMarketItem) {
  return `sticker-market-name-${item.stickerId}`
}

async function startRename(item: AdminStickerMarketItem) {
  renamingId.value = item.stickerId
  renameDraft.value = GENERATED_NAME.test(item.name) ? '' : item.name
  renameError.value = ''
  await nextTick()
  const field = document.getElementById(renameFieldId(item)) as HTMLInputElement | null
  field?.focus()
  field?.select()
}

function cancelRename() {
  renamingId.value = null
  renameError.value = ''
}

// Escape first leaves an open name field; a second press closes the dialog.
function onDialogEscape(event: KeyboardEvent) {
  if (renamingId.value === null) return
  event.preventDefault()
  cancelRename()
}

async function submitRename(item: AdminStickerMarketItem) {
  if (renamingId.value !== item.stickerId || busyItemId.value === item.stickerId) return
  const name = renameDraft.value.trim()
  if (!name || name === item.name) {
    cancelRename()
    return
  }
  busyItemId.value = item.stickerId
  renameError.value = ''
  try {
    await saveStickerMarketItem({ stickerId: item.stickerId, name, isEnabled: item.isEnabled })
    item.aliases = item.aliases.includes(name)
      ? item.aliases.filter((alias) => alias !== name)
      : [...item.aliases, item.name]
    item.name = name
    renamingId.value = null
    adminToast.success(adminText('stickerMarketRenamed', { name }))
  } catch (err) {
    const key = nameMessageTexts[(err as { messageCode?: string } | null)?.messageCode || '']
    if (key) {
      renameError.value = adminText(key)
      void nextTick(() => document.getElementById(renameFieldId(item))?.focus())
    } else {
      marketToastError(err, 'stickerMarketSaveFailed')
    }
  } finally {
    busyItemId.value = null
  }
}

function topicPayload(form: { topic: string, topicLabel: string }) {
  return form.topic === NEW_TOPIC ? { topicLabel: form.topicLabel.trim() } : { topic: form.topic }
}

// Name and category are required; returns the warning to show, or '' when valid.
function packInfoProblem(form: { name: string, topic: string, topicLabel: string }) {
  if (!form.name.trim()) return adminText('stickerPackNameRequired')
  if (form.topic === NEW_TOPIC ? !form.topicLabel.trim() : !form.topic) return adminText('stickerMarketTopicRequired')
  return ''
}

function openCreate() {
  const first = market.value?.topics?.[0]?.id ?? NEW_TOPIC
  Object.assign(createForm, { name: '', topic: first, topicLabel: '' })
  createOpen.value = true
}

async function submitCreate() {
  const problem = packInfoProblem(createForm)
  if (problem) {
    adminToast.warning(problem)
    return
  }
  creating.value = true
  try {
    const pack = await createStickerMarketPack({ name: createForm.name.trim(), ...topicPayload(createForm) })
    createOpen.value = false
    adminToast.success(adminText('stickerMarketPackCreated', { name: pack.name }))
    uploadTarget.value = pack
    await refresh()
  } catch (err) {
    marketToastError(err, 'stickerMarketSaveFailed')
  } finally {
    creating.value = false
  }
}

function openInfoEdit(pack: AdminStickerMarketPack) {
  Object.assign(infoForm, { name: pack.name, topic: pack.topic || NEW_TOPIC, topicLabel: '' })
  infoEditing.value = pack
}

async function submitInfo() {
  const pack = infoEditing.value
  if (!pack) return
  const problem = packInfoProblem(infoForm)
  if (problem) {
    adminToast.warning(problem)
    return
  }
  infoSaving.value = true
  try {
    replacePack(await saveStickerMarketPack({
      id: pack.id,
      published: pack.published,
      sortOrder: pack.sortOrder,
      name: infoForm.name.trim(),
      ...topicPayload(infoForm),
    }))
    infoEditing.value = null
    adminToast.success(adminText('k000e'))
    if (infoForm.topic === NEW_TOPIC) await refresh(true)
  } catch (err) {
    marketToastError(err, 'stickerMarketSaveFailed')
  } finally {
    infoSaving.value = false
  }
}

function openUpload(pack: AdminStickerMarketPack) {
  detail.value = null
  uploadTarget.value = pack
}

async function uploadToTarget(file: File) {
  const pack = uploadTarget.value
  if (!pack) throw new Error(adminText('k00vh1'))
  const { result, pack: next } = await uploadStickerMarketFile(pack.id, file)
  replacePack(next)
  if (uploadTarget.value?.id === next.id) uploadTarget.value = next
  return result
}

function closeUpload(open: boolean) {
  if (open || uploadBusy.value) return
  uploadTarget.value = null
  void refresh(true)
}

async function publishUploaded() {
  const pack = uploadTarget.value
  if (!pack) return
  uploadPublishing.value = true
  try {
    replacePack(await saveStickerMarketPack({ id: pack.id, published: true, sortOrder: pack.sortOrder }))
    adminToast.success(adminText('stickerMarketPublished', { name: pack.name }))
    uploadTarget.value = null
    await refresh(true)
  } catch (err) {
    marketToastError(err, 'stickerMarketSaveFailed')
  } finally {
    uploadPublishing.value = false
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
      <div class="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="icon" type="button" :aria-label="adminText('k004q')" :title="adminText('k004q')" @click="refresh()">
          <RefreshCw class="size-4" />
        </Button>
        <Button variant="outline" type="button" @click="openCreate">
          <Plus class="size-4" />
          {{ adminText('stickerMarketUploadPack') }}
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
            <AdminSelect v-model="source" :options="sourceOptions" :aria-label="adminText('stickerMarketColumnSource')" class="w-32" />
            <AdminSelect v-model="topic" :options="topicOptions" :aria-label="adminText('stickerMarketColumnTopic')" class="w-40" />
            <AdminSelect v-model="status" :options="statusOptions" :aria-label="adminText('k007j')" class="w-36" />
          </div>
        </AdminToolbar>
      </template>

      <tr v-if="!loading && !error && packs.length === 0">
        <td :colspan="tableColumns.length" class="h-24 px-4 text-center text-muted-foreground">
          {{ source === 'upload' ? adminText('stickerMarketUploadEmpty') : market?.lastSyncedAt ? adminText('k002v') : adminText('stickerMarketNeverSynced') }}
        </td>
      </tr>
      <tr v-for="pack in packs" :key="pack.id" class="hover:bg-muted/30">
        <td class="px-4 py-2 align-middle">
          <img
            v-if="coverSource(pack)"
            :src="coverSource(pack)"
            alt=""
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
            <Badge v-if="isUploaded(pack)" variant="secondary">{{ adminText('stickerMarketSourceUpload') }}</Badge>
            <span v-else class="truncate font-mono text-xs text-muted-foreground">{{ pack.upstreamId }}</span>
            <Badge v-if="pack.missing" variant="outline">{{ adminText('stickerMarketMissing') }}</Badge>
            <Badge v-if="pack.archived" variant="outline">{{ adminText('stickerMarketArchived') }}</Badge>
          </div>
        </td>
        <td class="px-4 py-2 align-middle text-muted-foreground">{{ topicLabels.get(pack.topic) || pack.topic || '—' }}</td>
        <td class="px-4 py-2 align-middle">
          <div class="tabular-nums">
            {{ isUploaded(pack) ? adminText('stickerMarketUploadedCount', { count: pack.itemCount }) : adminText('stickerMarketItemCount', { count: pack.itemCount, total: pack.upstreamCount }) }}
          </div>
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
            <template v-if="isUploaded(pack)">
              <AdminActionButton :disabled="busyPackId === pack.id" @click="openUpload(pack)">
                <Upload class="size-3.5" />
                {{ adminText('stickerMarketUploadImages') }}
              </AdminActionButton>
              <AdminActionButton compact :title="adminText('stickerMarketEditInfo')" :disabled="busyPackId === pack.id" @click="openInfoEdit(pack)">
                <Settings2 class="size-3.5" />
              </AdminActionButton>
            </template>
            <AdminActionButton
              v-else
              :disabled="busyPackId === pack.id || isRunning(pack) || market?.enabled === false || pack.missing"
              @click="queueImport(pack)"
            >
              {{ importLabel(pack) }}
            </AdminActionButton>
            <AdminActionButton compact :title="adminText('stickerMarketViewStickers')" :disabled="pack.status === 'new' && pack.itemCount === 0" @click="openDetail(pack)">
              <Eye class="size-3.5" />
            </AdminActionButton>
          </div>
        </td>
      </tr>
    </ManagementTable>

    <Dialog :open="detail !== null" @update:open="(open) => !open && (detail = null)">
      <DialogContent class="sm:max-w-2xl" @escape-key-down="onDialogEscape">
        <DialogHeader class="min-w-0 flex-row flex-wrap items-center gap-3 space-y-0 text-start">
          <img
            v-if="detail && coverSource(detail.pack)"
            :src="coverSource(detail.pack)"
            alt=""
            class="size-12 shrink-0 rounded-md object-contain outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
            @error="detail && markCoverBroken(coverSource(detail.pack))"
          />
          <div class="min-w-0 flex-1 basis-40 space-y-1.5">
            <DialogTitle class="truncate">{{ detail?.pack.name }}</DialogTitle>
            <DialogDescription class="text-pretty">{{ adminText('stickerMarketItemsDescription') }}</DialogDescription>
          </div>
          <Button v-if="detail && isUploaded(detail.pack)" variant="outline" size="sm" type="button" class="ms-auto me-6 shrink-0" @click="openUpload(detail.pack)">
            <Upload class="size-4" />
            {{ adminText('stickerMarketUploadMore') }}
          </Button>
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
            <template v-if="item.stickerId">
              <form
                v-if="renamingId === item.stickerId"
                class="flex w-full flex-col gap-1"
                @submit.prevent="submitRename(item)"
              >
                <Input
                  :id="renameFieldId(item)"
                  v-model="renameDraft"
                  maxlength="64"
                  autocomplete="off"
                  spellcheck="false"
                  class="h-7 px-2 text-base sm:text-xs"
                  :aria-label="adminText('stickerMarketNameLabel')"
                  :aria-invalid="renameError ? 'true' : undefined"
                  :aria-describedby="renameError ? `${renameFieldId(item)}-error` : undefined"
                  :placeholder="adminText('stickerMarketNameLabel')"
                  :disabled="busyItemId === item.stickerId"
                  @blur="submitRename(item)"
                />
                <p
                  v-if="renameError"
                  :id="`${renameFieldId(item)}-error`"
                  role="alert"
                  class="text-pretty text-xs leading-snug text-destructive"
                >
                  {{ renameError }}
                </p>
              </form>
              <button
                v-else
                type="button"
                class="group flex w-full min-w-0 items-center justify-center gap-1 rounded-md px-1.5 py-1 text-xs transition-colors duration-150 hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.96]"
                :title="item.name"
                :aria-label="adminText('stickerMarketRename', { name: item.name })"
                @click="startRename(item)"
              >
                <span class="truncate" :class="GENERATED_NAME.test(item.name) ? 'font-mono text-muted-foreground' : 'font-medium'">{{ item.name }}</span>
                <Pencil class="size-3 shrink-0 text-muted-foreground opacity-60 transition-opacity duration-150 group-hover:opacity-100" aria-hidden="true" />
              </button>
              <p
                v-if="item.aliases.length && renamingId !== item.stickerId"
                class="-mt-1.5 max-w-full truncate text-[11px] text-muted-foreground"
                :title="adminText('stickerAliases', { names: item.aliases.join(', ') })"
              >
                {{ adminText('stickerMarketPreviousNames') }}
              </p>
              <Switch
                :model-value="item.isEnabled"
                :disabled="busyItemId === item.stickerId"
                :aria-label="adminText('stickerMarketItemEnabledLabel', { name: item.name })"
                @update:model-value="(checked: boolean) => toggleItem(item, checked)"
              />
            </template>
            <span v-else class="text-center text-xs text-muted-foreground">{{ adminText('stickerMarketItemNotImported', { reason: reasonText(item.error) }) }}</span>
          </li>
        </ul>
      </DialogContent>
    </Dialog>

    <Dialog :open="createOpen" @update:open="(open) => !open && !creating && (createOpen = false)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ adminText('stickerMarketUploadPack') }}</DialogTitle>
          <DialogDescription class="text-pretty">{{ adminText('stickerMarketCreateDescription') }}</DialogDescription>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitCreate">
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('stickerPackNameLabel') }}
            <Input v-model="createForm.name" maxlength="128" autocomplete="off" class="text-base sm:text-sm" :placeholder="adminText('stickerPackNamePlaceholder')" />
          </label>
          <div class="grid gap-2 text-sm font-medium">
            {{ adminText('stickerMarketColumnTopic') }}
            <AdminSelect v-model="createForm.topic" :options="topicPickerOptions" :aria-label="adminText('stickerMarketColumnTopic')" class="w-full" />
            <Input
              v-if="createForm.topic === NEW_TOPIC"
              v-model="createForm.topicLabel"
              maxlength="64"
              autocomplete="off"
              class="text-base font-normal sm:text-sm"
              :aria-label="adminText('stickerMarketTopicNewLabel')"
              :placeholder="adminText('stickerMarketTopicNewLabel')"
            />
          </div>
          <ol class="grid gap-1.5 rounded-lg bg-muted/30 px-3 py-2.5 text-xs text-muted-foreground">
            <li>{{ adminText('stickerMarketCreateStep1') }}</li>
            <li>{{ adminText('stickerMarketCreateStep2') }}</li>
            <li>{{ adminText('stickerMarketCreateStep3') }}</li>
          </ol>
          <DialogFooter>
            <Button variant="outline" type="button" :disabled="creating" @click="createOpen = false">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="creating">
              <Loader2 v-if="creating" class="size-4 animate-spin" />
              {{ adminText('stickerPackCreateAndUpload') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="infoEditing !== null" @update:open="(open) => !open && !infoSaving && (infoEditing = null)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ adminText('stickerMarketEditInfo') }}</DialogTitle>
          <DialogDescription>{{ infoEditing?.name }}</DialogDescription>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitInfo">
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('stickerPackNameLabel') }}
            <Input v-model="infoForm.name" maxlength="128" autocomplete="off" class="text-base sm:text-sm" />
          </label>
          <div class="grid gap-2 text-sm font-medium">
            {{ adminText('stickerMarketColumnTopic') }}
            <AdminSelect v-model="infoForm.topic" :options="topicPickerOptions" :aria-label="adminText('stickerMarketColumnTopic')" class="w-full" />
            <Input
              v-if="infoForm.topic === NEW_TOPIC"
              v-model="infoForm.topicLabel"
              maxlength="64"
              autocomplete="off"
              class="text-base font-normal sm:text-sm"
              :aria-label="adminText('stickerMarketTopicNewLabel')"
              :placeholder="adminText('stickerMarketTopicNewLabel')"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" type="button" :disabled="infoSaving" @click="infoEditing = null">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="infoSaving">
              <Loader2 v-if="infoSaving" class="size-4 animate-spin" />
              {{ infoSaving ? adminText('k005f') : adminText('k005g') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="uploadTarget !== null" @update:open="closeUpload">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle class="truncate">{{ adminText('stickerMarketUploadTitle', { name: uploadTarget?.name || '' }) }}</DialogTitle>
          <DialogDescription class="text-pretty">{{ adminText('stickerMarketUploadDescription') }}</DialogDescription>
        </DialogHeader>
        <p v-if="uploadTarget" class="text-sm tabular-nums text-muted-foreground">
          {{ adminText('stickerMarketUploadedCount', { count: uploadTarget.itemCount }) }}
          · {{ uploadTarget.published ? adminText('stickerMarketUploadPublished') : adminText('stickerMarketUploadHidden') }}
        </p>
        <StickerUploadQueue v-if="uploadTarget" :upload="uploadToTarget" @update:busy="(busy) => (uploadBusy = busy)" />
        <DialogFooter>
          <Button variant="outline" type="button" :disabled="uploadBusy || uploadPublishing" @click="closeUpload(false)">{{ adminText('stickerUploadClose') }}</Button>
          <Button
            v-if="uploadTarget && !uploadTarget.published"
            type="button"
            :disabled="uploadBusy || uploadPublishing || uploadTarget.itemCount === 0"
            :title="uploadTarget.itemCount === 0 ? adminText('stickerMarketPublishNeedsImport') : undefined"
            @click="publishUploaded"
          >
            <Loader2 v-if="uploadPublishing" class="size-4 animate-spin" />
            {{ adminText('stickerMarketPublishNow') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </BasicPage>
</template>

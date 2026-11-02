<script setup lang="ts">import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onMounted, reactive, ref, watch } from 'vue'
import { FolderPlus, LayoutGrid, Loader2, Plus, RefreshCw, Search, Settings2, Store, Trash2, Upload, X } from '@lucide/vue'
import { RouterLink } from 'vue-router'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import AdminSelect, { type AdminSelectOption } from '@/admin/components/AdminSelect.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Button } from '@/admin/components/ui/button'
import { Checkbox } from '@/admin/components/ui/checkbox'
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
import StickerUploadQueue from './StickerUploadQueue.vue'
import {
  batchStickers,
  deleteSticker,
  deleteStickerPack,
  getStickerPacks,
  getStickers,
  importStickerPack,
  saveSticker,
  saveStickerPack,
} from '@/admin/runtime/api'
import { uploadImageFile } from '@/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AdminPayload, AdminSticker, AdminStickerBatchAction, AdminStickerPack, ManageHomeProps } from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

// 与后端 stickerservice.ValidateName 的 `[\p{L}\p{N}_\-]{1,64}` 保持一致。
const stickerNamePattern = /^[\p{L}\p{N}_-]{1,64}$/u
// 默认表情包由后端保证存在且不可删除。
const DEFAULT_PACK = 'official'
const ALL_PACKS = ''

type StatusFilter = 'all' | 'enabled' | 'disabled'

const loading = ref(false)
const error = ref('')
const stickers = ref<AdminSticker[]>([])
const packs = ref<AdminStickerPack[]>([])
const activePack = ref(ALL_PACKS)
const keyword = ref('')
const status = ref<StatusFilter>('all')
const selected = ref<Set<number>>(new Set())
const savingToggleId = ref<number | null>(null)

const editing = ref<AdminSticker | null>(null)
const saving = ref(false)
const imageFile = ref<File | null>(null)
const form = reactive({ id: 0, name: '', fileName: '', pack: DEFAULT_PACK, sortOrder: 1000, isEnabled: true })
const deletingSticker = ref<AdminSticker | null>(null)
const deleting = ref(false)

const packEditing = ref<AdminStickerPack | null>(null)
const packForm = reactive({ key: '', name: '', sortOrder: 0, coverStickerId: 0 })
const packSaving = ref(false)
const packDeleting = ref<AdminStickerPack | null>(null)
const packMoveTo = ref(DEFAULT_PACK)
const packDeleteBusy = ref(false)

const uploadOpen = ref(false)
const uploadPack = ref(DEFAULT_PACK)
const uploadBusy = ref(false)

const batchBusy = ref(false)
const batchMoveTo = ref('')
const batchDeleteOpen = ref(false)

// 站点 serverMessages 目录受 mobile Dart 生成目录门禁锁定，无法单独追加；
// sticker 域 messageCode 在页面侧本地化。
const stickerMessageTexts: Record<string, string> = {
  'admin.sticker.nameRequired': 'k00vgp',
  'admin.sticker.nameInvalid': 'k00vgo',
  'admin.sticker.nameExists': 'k00vh0',
  'admin.sticker.nameReserved': 'stickerNameReserved',
  'admin.sticker.notFound': 'k00vh1',
  'common.request.invalidParams': 'stickerPackInvalid',
}

function stickerToastError(err: unknown, fallbackKey: string) {
  const messageCode = (err as { messageCode?: string } | null)?.messageCode || ''
  const key = stickerMessageTexts[messageCode]
  if (key) {
    adminToast.warning(adminText(key))
    return
  }
  adminToast.error(err, adminText(fallbackKey))
}

const packByKey = computed(() => new Map(packs.value.map((pack) => [pack.key, pack])))
const currentPack = computed(() => packByKey.value.get(activePack.value) ?? null)
const packOptions = computed<AdminSelectOption<string>[]>(() => packs.value.map((pack) => ({ value: pack.key, label: pack.name })))
const statusOptions = computed<AdminSelectOption<StatusFilter>[]>(() => [
  { value: 'all', label: adminText('stickerFilterAll') },
  { value: 'enabled', label: adminText('stickerFilterEnabled') },
  { value: 'disabled', label: adminText('stickerFilterDisabled') },
])

const visibleStickers = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  return stickers.value.filter((sticker) => {
    if (activePack.value && sticker.pack !== activePack.value) return false
    if (status.value === 'enabled' && !sticker.isEnabled) return false
    if (status.value === 'disabled' && sticker.isEnabled) return false
    if (!query) return true
    return sticker.name.toLowerCase().includes(query) || sticker.aliases?.some((alias) => alias.toLowerCase().includes(query))
  })
})
const filtered = computed(() => keyword.value.trim() !== '' || status.value !== 'all')
const allVisibleSelected = computed(() => visibleStickers.value.length > 0 && visibleStickers.value.every((sticker) => selected.value.has(sticker.id)))
const someVisibleSelected = computed(() => visibleStickers.value.some((sticker) => selected.value.has(sticker.id)))
const coverCandidates = computed(() => stickers.value.filter((sticker) => sticker.pack === packForm.key))
const moveTargets = computed(() => packOptions.value.filter((option) => option.value !== packDeleting.value?.key))

// 切换分组或筛选后，只保留仍然可见的选择，避免批量操作命中看不见的表情。
watch([activePack, keyword, status], () => {
  const visible = new Set(visibleStickers.value.map((sticker) => sticker.id))
  selected.value = new Set([...selected.value].filter((id) => visible.has(id)))
})

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [nextStickers, nextPacks] = await Promise.all([getStickers(), getStickerPacks()])
    stickers.value = nextStickers
    packs.value = nextPacks
    if (activePack.value && !nextPacks.some((pack) => pack.key === activePack.value)) activePack.value = ALL_PACKS
    const ids = new Set(nextStickers.map((sticker) => sticker.id))
    selected.value = new Set([...selected.value].filter((id) => ids.has(id)))
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('k00vgc')
  } finally {
    loading.value = false
  }
}

function packName(key: string) {
  return packByKey.value.get(key)?.name ?? key
}

function toggleSelected(id: number, checked: boolean | 'indeterminate') {
  const next = new Set(selected.value)
  if (checked === true) next.add(id)
  else next.delete(id)
  selected.value = next
}

function toggleAllVisible(checked: boolean | 'indeterminate') {
  const next = new Set(selected.value)
  for (const sticker of visibleStickers.value) {
    if (checked === true) next.add(sticker.id)
    else next.delete(sticker.id)
  }
  selected.value = next
}

function clearFilters() {
  keyword.value = ''
  status.value = 'all'
}

function openCreate() {
  imageFile.value = null
  Object.assign(form, { id: 0, name: '', fileName: '', pack: activePack.value || DEFAULT_PACK, sortOrder: 1000, isEnabled: true })
  editing.value = { id: 0, name: '', fileName: '', url: '', pack: form.pack, sortOrder: 1000, isEnabled: true, createdBy: 0, aliases: [] }
}

function openEdit(sticker: AdminSticker) {
  imageFile.value = null
  Object.assign(form, { id: sticker.id, name: sticker.name, fileName: '', pack: sticker.pack, sortOrder: sticker.sortOrder, isEnabled: sticker.isEnabled })
  editing.value = sticker
}

function onImageChange(event: Event) {
  imageFile.value = (event.target as HTMLInputElement).files?.[0] || null
  form.fileName = ''
}

async function submitSticker() {
  const name = form.name.trim()
  if (!name) {
    adminToast.warning(adminText('k00vgp'))
    return
  }
  if (!stickerNamePattern.test(name)) {
    adminToast.warning(adminText('k00vgo'))
    return
  }
  if (!form.id && !imageFile.value && !form.fileName) {
    adminToast.warning(adminText('k00vh4'))
    return
  }
  saving.value = true
  try {
    if (imageFile.value && !form.fileName) {
      form.fileName = await uploadImageFile(imageFile.value, imageFile.value.name)
    }
    await saveSticker({
      id: form.id,
      name,
      ...(form.fileName ? { fileName: form.fileName } : {}),
      pack: form.pack,
      sortOrder: Number(form.sortOrder || 0),
      isEnabled: form.isEnabled,
    })
    editing.value = null
    await loadAll()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    stickerToastError(err, 'k00vgd')
  } finally {
    saving.value = false
  }
}

async function toggleSticker(sticker: AdminSticker, isEnabled: boolean) {
  savingToggleId.value = sticker.id
  try {
    await saveSticker({ id: sticker.id, name: sticker.name, pack: sticker.pack, sortOrder: sticker.sortOrder, isEnabled })
    sticker.isEnabled = isEnabled
    const pack = packByKey.value.get(sticker.pack)
    if (pack) pack.enabledCount += isEnabled ? 1 : -1
  } catch (err) {
    stickerToastError(err, 'k00vgd')
  } finally {
    savingToggleId.value = null
  }
}

async function confirmDelete() {
  if (!deletingSticker.value) return
  deleting.value = true
  try {
    await deleteSticker(deletingSticker.value.id)
    deletingSticker.value = null
    editing.value = null
    await loadAll()
    adminToast.success(adminText('k002u'))
  } catch (err) {
    stickerToastError(err, 'k00vge')
  } finally {
    deleting.value = false
  }
}

function openPackCreate() {
  Object.assign(packForm, { key: '', name: '', sortOrder: (packs.value.at(-1)?.sortOrder ?? 0) + 10, coverStickerId: 0 })
  packEditing.value = { key: '', name: '', sortOrder: packForm.sortOrder, coverStickerId: 0, coverUrl: '', stickerCount: 0, enabledCount: 0 }
}

function openPackEdit(pack: AdminStickerPack) {
  Object.assign(packForm, { key: pack.key, name: pack.name, sortOrder: pack.sortOrder, coverStickerId: pack.coverStickerId })
  packEditing.value = pack
}

async function submitPack() {
  const name = packForm.name.trim()
  if (!name) {
    adminToast.warning(adminText('stickerPackNameRequired'))
    return
  }
  const creating = !packForm.key
  packSaving.value = true
  try {
    const result = await saveStickerPack({
      ...(packForm.key ? { key: packForm.key } : {}),
      name,
      sortOrder: Number(packForm.sortOrder || 0),
      ...(packForm.coverStickerId ? { coverStickerId: packForm.coverStickerId } : {}),
    })
    packEditing.value = null
    await loadAll()
    activePack.value = result.key
    adminToast.success(adminText(creating ? 'stickerPackCreated' : 'k000e', { name }))
    // 新建后直接进入上传，让“建包 → 传图”是一条连续流程。
    if (creating) openUpload(result.key)
  } catch (err) {
    stickerToastError(err, 'stickerPackSaveFailed')
  } finally {
    packSaving.value = false
  }
}

function openPackDelete(pack: AdminStickerPack) {
  packMoveTo.value = DEFAULT_PACK
  packDeleting.value = pack
}

async function confirmPackDelete() {
  const pack = packDeleting.value
  if (!pack) return
  packDeleteBusy.value = true
  try {
    const result = await deleteStickerPack(pack.key, packMoveTo.value)
    packDeleting.value = null
    activePack.value = packMoveTo.value
    await loadAll()
    adminToast.success(adminText('stickerPackDeleted', { name: pack.name, count: result.moved, target: packName(packMoveTo.value) }))
  } catch (err) {
    stickerToastError(err, 'stickerPackDeleteFailed')
  } finally {
    packDeleteBusy.value = false
  }
}

function openUpload(pack = activePack.value || DEFAULT_PACK) {
  uploadPack.value = pack
  uploadOpen.value = true
}

function closeUpload(open: boolean) {
  if (!open && !uploadBusy.value) uploadOpen.value = false
}

async function runBatch(action: AdminStickerBatchAction, pack = '') {
  const ids = [...selected.value]
  if (ids.length === 0) return
  batchBusy.value = true
  try {
    const result = await batchStickers({ ids, action, ...(pack ? { pack } : {}) })
    if (action === 'delete') selected.value = new Set()
    batchDeleteOpen.value = false
    await loadAll()
    adminToast.success(adminText(`stickerBatchDone_${action}`, { count: result.changed, target: packName(pack) }))
  } catch (err) {
    stickerToastError(err, 'stickerBatchFailed')
  } finally {
    batchBusy.value = false
    batchMoveTo.value = ''
  }
}

function onBatchMove(pack: string) {
  if (pack) void runBatch('move', pack)
}

onMounted(() => {
  void loadAll()
})
</script>

<template>
  <BasicPage :title="adminText('k00vg8')" :description="adminText('k00vg9')" sticky>
    <template #actions>
      <div class="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="icon" type="button" :aria-label="adminText('k004q')" :title="adminText('k004q')" :disabled="loading" @click="loadAll">
          <RefreshCw class="size-4" :class="loading && 'animate-spin'" />
        </Button>
        <Button variant="outline" as-child>
          <RouterLink to="/admin/sticker-market">
            <Store class="size-4" />
            {{ adminText('stickerMarketOpen') }}
          </RouterLink>
        </Button>
        <Button variant="outline" type="button" @click="openCreate">
          <Plus class="size-4" />
          {{ adminText('k00vgb') }}
        </Button>
        <Button type="button" @click="openUpload()">
          <Upload class="size-4" />
          {{ adminText('stickerUploadAction') }}
        </Button>
      </div>
    </template>

    <div v-if="error" class="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-destructive/10 px-4 py-3 text-sm text-destructive" role="alert">
      <span>{{ error }}</span>
      <Button variant="outline" size="sm" type="button" @click="loadAll">{{ adminText('k004q') }}</Button>
    </div>

    <div class="grid gap-6 lg:grid-cols-[15rem_minmax(0,1fr)]">
      <aside class="min-w-0 lg:sticky lg:top-20 lg:self-start">
        <div class="mb-2 flex items-center justify-between gap-2 ps-2">
          <h2 class="text-sm font-medium text-muted-foreground">{{ adminText('stickerPacksHeading') }}</h2>
          <Button variant="ghost" size="icon-sm" type="button" :aria-label="adminText('stickerPackCreate')" :title="adminText('stickerPackCreate')" @click="openPackCreate">
            <FolderPlus class="size-4" />
          </Button>
        </div>
        <nav class="-mx-1 flex gap-1 overflow-x-auto px-1 pb-1 lg:mx-0 lg:flex-col lg:overflow-visible lg:px-0" :aria-label="adminText('stickerPacksHeading')">
          <button
            type="button"
            class="flex min-w-max items-center gap-3 rounded-lg px-2 py-1.5 text-start text-sm transition-colors duration-150 lg:min-w-0"
            :class="activePack === ALL_PACKS ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'"
            :aria-current="activePack === ALL_PACKS ? 'true' : undefined"
            @click="activePack = ALL_PACKS"
          >
            <span class="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted" aria-hidden="true">
              <LayoutGrid class="size-4" />
            </span>
            <span class="min-w-0 flex-1 truncate">{{ adminText('stickerPackAll') }}</span>
            <span class="text-xs tabular-nums text-muted-foreground">{{ stickers.length }}</span>
          </button>
          <button
            v-for="pack in packs"
            :key="pack.key"
            type="button"
            class="flex min-w-max items-center gap-3 rounded-lg px-2 py-1.5 text-start text-sm transition-colors duration-150 lg:min-w-0"
            :class="activePack === pack.key ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'"
            :aria-current="activePack === pack.key ? 'true' : undefined"
            @click="activePack = pack.key"
          >
            <img
              v-if="pack.coverUrl"
              :src="pack.coverUrl"
              alt=""
              class="size-8 shrink-0 rounded-md bg-muted/40 object-contain outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
              loading="lazy"
            />
            <span v-else class="size-8 shrink-0 rounded-md bg-muted" aria-hidden="true" />
            <span class="min-w-0 flex-1 truncate" :title="pack.name">{{ pack.name }}</span>
            <span class="text-xs tabular-nums text-muted-foreground" :title="adminText('stickerPackCountTitle', { enabled: pack.enabledCount, total: pack.stickerCount })">
              {{ pack.enabledCount === pack.stickerCount ? pack.stickerCount : `${pack.enabledCount}/${pack.stickerCount}` }}
            </span>
          </button>
        </nav>
      </aside>

      <section class="min-w-0">
        <header class="mb-4 flex flex-wrap items-center gap-x-4 gap-y-3">
          <div class="flex min-w-0 flex-1 items-center gap-3">
            <img
              v-if="currentPack?.coverUrl"
              :src="currentPack.coverUrl"
              alt=""
              class="size-11 shrink-0 rounded-lg bg-muted/40 object-contain outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
            />
            <div class="min-w-0">
              <h2 class="truncate text-base font-semibold">{{ currentPack ? currentPack.name : adminText('stickerPackAll') }}</h2>
              <p class="text-sm tabular-nums text-muted-foreground">
                {{ currentPack
                  ? adminText('stickerPackSummary', { total: currentPack.stickerCount, enabled: currentPack.enabledCount })
                  : adminText('stickerLibrarySummary', { total: stickers.length, packs: packs.length }) }}
              </p>
            </div>
          </div>
          <div v-if="currentPack" class="flex flex-wrap items-center gap-2">
            <Button variant="outline" size="sm" type="button" @click="openUpload(currentPack.key)">
              <Upload class="size-4" />
              {{ adminText('stickerUploadToPack') }}
            </Button>
            <Button variant="outline" size="sm" type="button" @click="openPackEdit(currentPack)">
              <Settings2 class="size-4" />
              {{ adminText('stickerPackEdit') }}
            </Button>
            <Button
              v-if="currentPack.key !== DEFAULT_PACK"
              variant="ghost"
              size="sm"
              type="button"
              class="text-destructive hover:bg-destructive/10 hover:text-destructive"
              @click="openPackDelete(currentPack)"
            >
              <Trash2 class="size-4" />
              {{ adminText('stickerPackDelete') }}
            </Button>
          </div>
        </header>

        <div class="mb-4 flex flex-wrap items-center gap-3">
          <label class="flex h-9 items-center gap-2 ps-1 text-sm text-muted-foreground">
            <Checkbox
              :model-value="allVisibleSelected ? true : someVisibleSelected ? 'indeterminate' : false"
              :disabled="visibleStickers.length === 0"
              @update:model-value="toggleAllVisible"
            />
            {{ adminText('stickerSelectAll') }}
          </label>
          <div class="relative min-w-48 flex-1 sm:max-w-72">
            <Search class="pointer-events-none absolute start-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <Input v-model="keyword" class="ps-8" :placeholder="adminText('stickerSearch')" :aria-label="adminText('stickerSearch')" />
          </div>
          <AdminSelect v-model="status" :options="statusOptions" :aria-label="adminText('k007j')" class="w-32" />
          <span class="ms-auto text-sm tabular-nums text-muted-foreground" aria-live="polite">
            {{ adminText('stickerVisibleCount', { count: visibleStickers.length }) }}
          </span>
        </div>

        <div v-if="loading && stickers.length === 0" class="grid grid-cols-[repeat(auto-fill,minmax(8rem,1fr))] gap-3" aria-hidden="true">
          <div v-for="index in 12" :key="index" class="aspect-[4/5] animate-pulse rounded-xl bg-muted/50" />
        </div>

        <div v-else-if="visibleStickers.length === 0 && !error" class="flex flex-col items-center gap-3 rounded-xl bg-muted/20 px-6 py-14 text-center">
          <template v-if="filtered">
            <p class="font-medium">{{ adminText('stickerEmptyFiltered') }}</p>
            <Button variant="outline" size="sm" type="button" @click="clearFilters">{{ adminText('stickerClearFilters') }}</Button>
          </template>
          <template v-else>
            <p class="font-medium">{{ adminText('stickerEmptyTitle') }}</p>
            <p class="max-w-sm text-pretty text-sm text-muted-foreground">{{ adminText('stickerEmptyBody') }}</p>
            <Button size="sm" type="button" @click="openUpload()">
              <Upload class="size-4" />
              {{ adminText('stickerUploadAction') }}
            </Button>
          </template>
        </div>

        <ul v-else class="grid grid-cols-[repeat(auto-fill,minmax(8rem,1fr))] gap-3">
          <li
            v-for="sticker in visibleStickers"
            :key="sticker.id"
            class="group relative flex flex-col rounded-xl bg-card p-2 shadow-[0_0_0_1px_var(--color-border),0_1px_2px_rgb(0_0_0/0.04)] transition-shadow duration-150"
            :class="selected.has(sticker.id) && 'shadow-[0_0_0_2px_var(--color-primary)]'"
          >
            <Checkbox
              class="absolute start-3 top-3 z-10 bg-background transition-opacity duration-150"
              :class="selected.size === 0 && !selected.has(sticker.id) ? 'opacity-0 focus-visible:opacity-100 group-hover:opacity-100' : ''"
              :model-value="selected.has(sticker.id)"
              :aria-label="adminText('stickerSelectOne', { name: sticker.name })"
              @update:model-value="(checked) => toggleSelected(sticker.id, checked)"
            />
            <button
              type="button"
              class="flex aspect-square items-center justify-center rounded-lg bg-muted/30 p-3 transition-transform duration-150 active:scale-[0.96]"
              :aria-label="adminText('stickerEditOne', { name: sticker.name })"
              @click="openEdit(sticker)"
            >
              <img
                :src="sticker.url"
                :alt="sticker.name"
                class="max-h-full max-w-full object-contain transition-opacity duration-150"
                :class="!sticker.isEnabled && 'opacity-40 grayscale'"
                loading="lazy"
              />
            </button>
            <div class="mt-2 flex min-w-0 items-start justify-between gap-2 px-1">
              <div class="min-w-0">
                <p class="truncate text-sm font-medium" :title="sticker.name">{{ sticker.name }}</p>
                <p class="truncate text-xs text-muted-foreground">
                  <template v-if="!activePack">{{ packName(sticker.pack) }}</template>
                  <template v-else-if="sticker.aliases?.length">{{ adminText('stickerAliasCount', { count: sticker.aliases.length }) }}</template>
                  <template v-else>{{ adminText('stickerSortShort', { order: sticker.sortOrder }) }}</template>
                </p>
              </div>
              <Switch
                class="mt-0.5 shrink-0"
                :model-value="sticker.isEnabled"
                :disabled="savingToggleId === sticker.id"
                :aria-label="adminText('stickerEnableOne', { name: sticker.name })"
                @update:model-value="(checked: boolean) => toggleSticker(sticker, checked)"
              />
            </div>
          </li>
        </ul>

        <a href="https://github.com/YourTongji/YourTJ-Hub/blob/dev/apps/gooseforum/app/console/stickerpresets/preset_stickers/NOTICE.md" target="_blank" rel="noopener noreferrer" class="mt-6 inline-block text-sm text-muted-foreground underline underline-offset-2 hover:text-foreground">
          {{ adminText('k00vh7') }}
        </a>
      </section>
    </div>

    <div v-if="selected.size > 0" class="pointer-events-none sticky bottom-4 z-20 mt-6 flex justify-center">
      <div
        class="pointer-events-auto flex flex-wrap items-center gap-2 rounded-2xl bg-background p-2 ps-4 shadow-[0_0_0_1px_var(--color-border),0_8px_24px_rgb(0_0_0/0.12)]"
        role="toolbar"
        :aria-label="adminText('stickerBatchToolbar')"
      >
        <span class="me-2 text-sm font-medium tabular-nums">{{ adminText('stickerSelectedCount', { count: selected.size }) }}</span>
        <Button variant="outline" size="sm" type="button" :disabled="batchBusy" @click="runBatch('enable')">{{ adminText('stickerBatchEnable') }}</Button>
        <Button variant="outline" size="sm" type="button" :disabled="batchBusy" @click="runBatch('disable')">{{ adminText('stickerBatchDisable') }}</Button>
        <AdminSelect
          v-model="batchMoveTo"
          :options="packOptions"
          :disabled="batchBusy"
          size="sm"
          class="w-36"
          :placeholder="adminText('stickerBatchMove')"
          :aria-label="adminText('stickerBatchMove')"
          @change="onBatchMove"
        />
        <Button variant="ghost" size="sm" type="button" class="text-destructive hover:bg-destructive/10 hover:text-destructive" :disabled="batchBusy" @click="batchDeleteOpen = true">
          {{ adminText('stickerBatchDelete') }}
        </Button>
        <Loader2 v-if="batchBusy" class="size-4 animate-spin text-muted-foreground" aria-hidden="true" />
        <Button variant="ghost" size="icon-sm" type="button" :aria-label="adminText('stickerSelectionClear')" :title="adminText('stickerSelectionClear')" @click="selected = new Set()">
          <X class="size-4" />
        </Button>
      </div>
    </div>

    <Dialog :open="editing !== null" @update:open="(open) => !open && !saving && (editing = null)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ form.id ? adminText('k00vgf') : adminText('k00vgb') }}</DialogTitle>
          <DialogDescription>{{ adminText('k00vgg') }}</DialogDescription>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitSticker">
          <div class="flex items-start gap-4">
            <div class="flex size-20 shrink-0 items-center justify-center rounded-xl bg-muted/40 p-2">
              <img v-if="editing?.url && !imageFile" :src="editing.url" :alt="editing.name" class="max-h-full max-w-full object-contain" />
              <Upload v-else class="size-5 text-muted-foreground" aria-hidden="true" />
            </div>
            <label class="grid min-w-0 flex-1 gap-2 text-sm font-medium">
              {{ adminText('k0088') }}
              <input type="file" accept="image/*" class="max-w-full text-sm font-normal file:me-3 file:rounded-md file:border-0 file:bg-muted file:px-3 file:py-1.5 file:text-sm" :disabled="saving" @change="onImageChange" />
              <span v-if="imageFile" class="truncate text-xs font-normal text-muted-foreground">{{ imageFile.name }}</span>
            </label>
          </div>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00af') }}
            <Input v-model="form.name" class="text-base sm:text-sm" />
            <span v-if="form.id" class="text-xs font-normal text-muted-foreground">{{ adminText('stickerRenameHint') }}</span>
            <span v-if="editing?.aliases?.length" class="text-xs font-normal text-muted-foreground">{{ adminText('stickerAliases', { names: editing.aliases.join(', ') }) }}</span>
          </label>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="grid gap-2 text-sm font-medium">
              {{ adminText('stickerPackLabel') }}
              <AdminSelect v-model="form.pack" :options="packOptions" :aria-label="adminText('stickerPackLabel')" class="w-full" :disabled="saving" />
            </div>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00bf') }}
              <Input v-model.number="form.sortOrder" type="number" class="text-base sm:text-sm" />
            </label>
          </div>
          <div class="flex items-center justify-between gap-3 rounded-lg bg-muted/30 px-3 py-2.5">
            <div class="grid gap-0.5">
              <span class="text-sm font-medium">{{ adminText('k005o') }}</span>
              <span class="text-xs text-muted-foreground">{{ adminText('k00vgs') }}</span>
            </div>
            <Switch v-model="form.isEnabled" :aria-label="adminText('k005o')" />
          </div>
          <DialogFooter class="gap-2 sm:justify-between">
            <Button
              v-if="form.id && editing"
              variant="ghost"
              type="button"
              class="text-destructive hover:bg-destructive/10 hover:text-destructive"
              :disabled="saving"
              @click="deletingSticker = editing"
            >
              <Trash2 class="size-4" />
              {{ adminText('k005i') }}
            </Button>
            <span v-else />
            <div class="flex gap-2">
              <Button variant="outline" type="button" :disabled="saving" @click="editing = null">{{ adminText('k009q') }}</Button>
              <Button type="submit" :disabled="saving">
                <Loader2 v-if="saving" class="size-4 animate-spin" />
                {{ saving ? adminText('k005f') : adminText('k005g') }}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="packEditing !== null" @update:open="(open) => !open && !packSaving && (packEditing = null)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ packForm.key ? adminText('stickerPackEdit') : adminText('stickerPackCreate') }}</DialogTitle>
          <DialogDescription>{{ adminText('stickerPackDialogDescription') }}</DialogDescription>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitPack">
          <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_7rem]">
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('stickerPackNameLabel') }}
              <Input v-model="packForm.name" maxlength="64" class="text-base sm:text-sm" :placeholder="adminText('stickerPackNamePlaceholder')" />
            </label>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00bf') }}
              <Input v-model.number="packForm.sortOrder" type="number" class="text-base sm:text-sm" />
            </label>
          </div>
          <p class="-mt-2 text-xs text-muted-foreground">{{ adminText('stickerPackSortHint') }}</p>
          <fieldset v-if="packForm.key" class="grid gap-2">
            <legend class="mb-2 text-sm font-medium">{{ adminText('stickerPackCoverLabel') }}</legend>
            <p v-if="coverCandidates.length === 0" class="text-sm text-muted-foreground">{{ adminText('stickerPackCoverEmpty') }}</p>
            <div v-else class="grid max-h-56 grid-cols-[repeat(auto-fill,minmax(3.5rem,1fr))] gap-2 overflow-y-auto p-0.5">
              <button
                type="button"
                class="flex aspect-square items-center justify-center rounded-lg bg-muted/30 p-1.5 transition-shadow duration-150"
                :class="packForm.coverStickerId === 0 ? 'shadow-[0_0_0_2px_var(--color-primary)]' : 'hover:bg-muted/60'"
                :aria-pressed="packForm.coverStickerId === 0"
                @click="packForm.coverStickerId = 0"
              >
                <span class="text-center text-[11px] leading-tight text-muted-foreground">{{ adminText('stickerPackCoverAuto') }}</span>
              </button>
              <button
                v-for="sticker in coverCandidates"
                :key="sticker.id"
                type="button"
                class="flex aspect-square items-center justify-center rounded-lg bg-muted/30 p-1.5 transition-shadow duration-150"
                :class="packForm.coverStickerId === sticker.id ? 'shadow-[0_0_0_2px_var(--color-primary)]' : 'hover:bg-muted/60'"
                :aria-pressed="packForm.coverStickerId === sticker.id"
                :aria-label="adminText('stickerPackCoverPick', { name: sticker.name })"
                :title="sticker.name"
                @click="packForm.coverStickerId = sticker.id"
              >
                <img :src="sticker.url" alt="" class="max-h-full max-w-full object-contain" loading="lazy" />
              </button>
            </div>
          </fieldset>
          <DialogFooter>
            <Button variant="outline" type="button" :disabled="packSaving" @click="packEditing = null">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="packSaving">
              <Loader2 v-if="packSaving" class="size-4 animate-spin" />
              {{ packForm.key ? adminText('k005g') : adminText('stickerPackCreateAndUpload') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="packDeleting !== null" @update:open="(open) => !open && !packDeleteBusy && (packDeleting = null)">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ adminText('stickerPackDeleteTitle', { name: packDeleting?.name || '' }) }}</DialogTitle>
          <DialogDescription>{{ adminText('stickerPackDeleteDescription', { count: packDeleting?.stickerCount ?? 0 }) }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-2 text-sm font-medium">
          {{ adminText('stickerPackMoveTo') }}
          <AdminSelect v-model="packMoveTo" :options="moveTargets" :aria-label="adminText('stickerPackMoveTo')" class="w-full" :disabled="packDeleteBusy" />
        </div>
        <DialogFooter>
          <Button variant="outline" type="button" :disabled="packDeleteBusy" @click="packDeleting = null">{{ adminText('k009q') }}</Button>
          <Button variant="destructive" type="button" :disabled="packDeleteBusy" @click="confirmPackDelete">
            <Loader2 v-if="packDeleteBusy" class="size-4 animate-spin" />
            {{ adminText('stickerPackDeleteConfirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog :open="uploadOpen" @update:open="closeUpload">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{{ adminText('stickerUploadTitle') }}</DialogTitle>
          <DialogDescription>{{ adminText('stickerUploadDescription') }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="grid gap-2 text-sm font-medium">
            {{ adminText('stickerUploadTarget') }}
            <AdminSelect v-model="uploadPack" :options="packOptions" :aria-label="adminText('stickerUploadTarget')" class="w-full" :disabled="uploadBusy" />
          </div>
          <StickerUploadQueue
            v-if="uploadOpen"
            :upload="(file) => importStickerPack(file, uploadPack)"
            @update:busy="(busy) => (uploadBusy = busy)"
            @uploaded="loadAll"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" type="button" :disabled="uploadBusy" @click="uploadOpen = false">{{ adminText('stickerUploadClose') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <AdminConfirmDialog
      :open="deletingSticker !== null"
      :title="adminText('k00vgq')"
      :description="adminText('k00vgr', { name: deletingSticker?.name || '' })"
      :loading="deleting"
      @update:open="(open) => !open && (deletingSticker = null)"
      @confirm="confirmDelete"
    />

    <AdminConfirmDialog
      :open="batchDeleteOpen"
      :title="adminText('stickerBatchDeleteTitle', { count: selected.size })"
      :description="adminText('stickerBatchDeleteDescription')"
      :loading="batchBusy"
      @update:open="(open) => !open && (batchDeleteOpen = false)"
      @confirm="runBatch('delete')"
    />
  </BasicPage>
</template>

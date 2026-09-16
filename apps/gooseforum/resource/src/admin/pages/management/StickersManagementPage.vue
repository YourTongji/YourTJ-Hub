<script setup lang="ts">import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onMounted, reactive, ref } from 'vue'
import { Edit3, Loader2, Plus, RefreshCw, Trash2, Upload } from '@lucide/vue'
import AdminActionButton from '@/admin/components/AdminActionButton.vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Button } from '@/admin/components/ui/button'
import { Badge } from '@/admin/components/ui/badge'
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
import { deleteSticker, getStickers, importStickerPack, saveSticker } from '@/admin/runtime/api'
import { uploadImageFile } from '@/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AdminPayload, AdminSticker, ManageHomeProps, StickerImportIssue, StickerImportResult } from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

// 与后端 stickerservice.ValidateName 的 `[\p{L}\p{N}_\-]{1,64}` 保持一致。
const stickerNamePattern = /^[\p{L}\p{N}_-]{1,64}$/u

const emptyStickerForm = { id: 0, name: '', fileName: '', sortOrder: 1000, isEnabled: true }

const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const importing = ref(false)
const error = ref('')
const stickers = ref<AdminSticker[]>([])
const editing = ref<AdminSticker | null>(null)
const deletingSticker = ref<AdminSticker | null>(null)
const savingToggleId = ref<number | null>(null)
const imageFile = ref<File | null>(null)
const form = reactive({ ...emptyStickerForm })
const importOpen = ref(false)
const importFile = ref<File | null>(null)
const importReport = ref<StickerImportResult | null>(null)

const tableColumns = computed(() => [
  adminText('k00vgh'),
  adminText('k00af'),
  adminText('k00bf'),
  adminText('k007j'),
  adminText('k007m'),
])

const importReasonTexts: Record<string, string> = {
  entryOpenFailed: 'k00vgu',
  tooLarge: 'k00vgv',
  invalidImage: 'k00vgw',
  unusableName: 'k00vgx',
  saveFailed: 'k00vgy',
  tooManyFiles: 'k00vgz',
  archiveTooLarge: 'k00vh6',
}

function importReasonText(issue: StickerImportIssue) {
  const key = importReasonTexts[issue.reason]
  return key ? adminText(key) : issue.reason
}

// 站点 serverMessages 目录受 mobile Dart 生成目录门禁锁定，无法单独追加；
// sticker 域 messageCode 在页面侧本地化。
const stickerMessageTexts: Record<string, string> = {
  'admin.sticker.nameRequired': 'k00vgp',
  'admin.sticker.nameInvalid': 'k00vgo',
  'admin.sticker.nameExists': 'k00vh0',
  'admin.sticker.notFound': 'k00vh1',
  'admin.sticker.importTooLarge': 'k00vh2',
  'admin.sticker.importInvalidZip': 'k00vh3',
  'upload.file.missing': 'k00vh4',
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

async function loadStickers() {
  loading.value = true
  error.value = ''
  try {
    stickers.value = await getStickers()
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('k00vgc')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  imageFile.value = null
  Object.assign(form, emptyStickerForm)
  editing.value = { id: 0, name: '', fileName: '', url: '', sortOrder: 1000, isEnabled: true, createdBy: 0 }
}

function openEdit(sticker: AdminSticker) {
  imageFile.value = null
  Object.assign(form, { fileName: '', id: sticker.id, name: sticker.name, sortOrder: sticker.sortOrder, isEnabled: sticker.isEnabled })
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
    await saveSticker({ id: form.id, name, ...(form.fileName ? { fileName: form.fileName } : {}), sortOrder: Number(form.sortOrder || 0), isEnabled: form.isEnabled })
    editing.value = null
    await loadStickers()
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
    await saveSticker({ id: sticker.id, name: sticker.name, sortOrder: sticker.sortOrder, isEnabled })
    sticker.isEnabled = isEnabled
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
    await loadStickers()
    adminToast.success(adminText('k002u'))
  } catch (err) {
    stickerToastError(err, 'k00vge')
  } finally {
    deleting.value = false
  }
}

function openImport() {
  importFile.value = null
  importReport.value = null
  importOpen.value = true
}

function onImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  importFile.value = input.files?.[0] || null
  importReport.value = null
}

async function submitImport() {
  if (!importFile.value) {
    adminToast.warning(adminText('k00vgk'))
    return
  }
  importing.value = true
  try {
    importReport.value = await importStickerPack(importFile.value)
    await loadStickers()
  } catch (err) {
    importReport.value = null
    stickerToastError(err, 'k00vh5')
  } finally {
    importing.value = false
  }
}

onMounted(() => {
  void loadStickers()
})
</script>

<template>
  <BasicPage :title="adminText('k00vg8')" :description="adminText('k00vg9')" sticky>
    <template #actions>
      <div class="flex items-center gap-2">
        <Button variant="outline" type="button" @click="loadStickers">
          <RefreshCw class="size-4" />
          {{ adminText('k004q') }}
        </Button>
        <Button variant="outline" type="button" @click="openImport">
          <Upload class="size-4" />
          {{ adminText('k00vgi') }}
        </Button>
        <Button type="button" @click="openCreate">
          <Plus class="size-4" />
          {{ adminText('k00vgb') }}
        </Button>
      </div>
    </template>

    <div class="mb-3 flex flex-wrap gap-2 text-sm text-muted-foreground">
      <Badge variant="secondary">{{ adminText('k00vga', { total: stickers.length }) }}</Badge>
      <span v-if="loading">{{ adminText('k0046') }}</span>
    </div>

    <ManagementTable
      :columns="tableColumns"
      :loading="loading"
      :error="error"
      :show-pagination="false"
      @retry="loadStickers"
    >
      <tr v-if="!loading && !error && stickers.length === 0">
        <td :colspan="tableColumns.length" class="h-24 px-4 text-center text-muted-foreground">{{ adminText('k002v') }}</td>
      </tr>
      <tr v-for="sticker in stickers" :key="sticker.id" class="hover:bg-muted/30">
        <td class="px-4 py-2 align-middle">
          <img :src="sticker.url" :alt="sticker.name" class="size-10 rounded object-contain" loading="lazy" />
        </td>
        <td class="px-4 py-2 align-middle font-medium">{{ sticker.name }}</td>
        <td class="px-4 py-2 align-middle tabular-nums">{{ sticker.sortOrder }}</td>
        <td class="px-4 py-2 align-middle">
          <Switch
            :model-value="sticker.isEnabled"
            :disabled="savingToggleId === sticker.id"
            @update:model-value="(checked: boolean) => toggleSticker(sticker, checked)"
          />
        </td>
        <td class="px-4 py-2 align-middle">
          <div class="flex items-center gap-1">
            <AdminActionButton compact :title="adminText('k005j')" @click="openEdit(sticker)">
              <Edit3 class="size-3.5" />
            </AdminActionButton>
            <AdminActionButton compact tone="danger" :title="adminText('k005i')" @click="deletingSticker = sticker">
              <Trash2 class="size-3.5" />
            </AdminActionButton>
          </div>
        </td>
      </tr>
    </ManagementTable>

    <Dialog :open="editing !== null" @update:open="(open) => !open && !saving && (editing = null)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ form.id ? adminText('k00vgf') : adminText('k00vgb') }}</DialogTitle>
          <DialogDescription>{{ adminText('k00vgg') }}</DialogDescription>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitSticker">
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00af') }}
            <Input v-model="form.name" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k0088') }}
            <input type="file" accept="image/*" :disabled="saving" @change="onImageChange" />
            <img v-if="editing?.url && !imageFile" :src="editing.url" :alt="editing.name" class="size-16 object-contain" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00bf') }}
            <Input v-model.number="form.sortOrder" type="number" />
          </label>
          <div class="grid gap-2 text-sm font-medium">
            {{ adminText('k005o') }}
            <div class="flex h-9 items-center justify-between rounded-md border bg-background px-3">
              <span class="text-sm text-muted-foreground">{{ adminText('k00vgs') }}</span>
              <Switch v-model="form.isEnabled" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" type="button" :disabled="saving" @click="editing = null">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="saving">{{ saving ? adminText('k005f') : adminText('k005g') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="importOpen" @update:open="(open) => (importOpen = open)">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{{ adminText('k00vgi') }}</DialogTitle>
          <DialogDescription>{{ adminText('k00vgt') }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <label class="inline-flex h-9 w-fit cursor-pointer items-center gap-2 rounded-md border bg-background px-3 text-sm font-medium shadow-xs hover:bg-accent">
            <Upload class="size-4" />
            {{ importFile ? importFile.name : adminText('k00vgj') }}
            <input class="hidden" type="file" accept=".zip,application/zip,application/x-zip-compressed" @change="onImportFileChange" />
          </label>
          <div v-if="importReport" class="space-y-3 rounded-lg border bg-muted/10 p-3">
            <Badge variant="secondary">{{ adminText('k00vgm', { imported: importReport.imported, skipped: importReport.skipped, failed: importReport.failed.length }) }}</Badge>
            <details v-if="importReport.failed.length > 0">
              <summary class="cursor-pointer text-sm font-medium text-destructive">
                {{ adminText('k00vgn') }} ({{ importReport.failed.length }})
              </summary>
              <ul class="mt-2 max-h-48 space-y-1 overflow-y-auto text-sm">
                <li v-for="(issue, index) in importReport.failed" :key="`${issue.name}-${index}`" class="flex items-center justify-between gap-3 rounded-md border bg-background px-2 py-1">
                  <span class="truncate font-mono text-xs">{{ issue.name }}</span>
                  <span class="shrink-0 text-xs text-muted-foreground">{{ importReasonText(issue) }}</span>
                </li>
              </ul>
            </details>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" type="button" @click="importOpen = false">{{ adminText('k009q') }}</Button>
          <Button type="button" :disabled="importing" @click="submitImport">
            <Loader2 v-if="importing" class="size-4 animate-spin" />
            <Upload v-else class="size-4" />
            {{ importing ? adminText('k00vgl') : adminText('k00uh') }}
          </Button>
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
  </BasicPage>
</template>

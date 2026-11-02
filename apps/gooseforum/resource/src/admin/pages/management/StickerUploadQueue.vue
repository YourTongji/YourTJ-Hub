<script setup lang="ts">
import { failedStickerArchive } from '@/admin/runtime/sticker-upload-retry'
import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onBeforeUnmount, ref } from 'vue'
import { AlertCircle, Check, FileArchive, ImagePlus, Loader2, X } from '@lucide/vue'
import { Button } from '@/admin/components/ui/button'
import type { StickerImportResult } from '@/admin/types'

/**
 * Upload queue shared by official and uploaded market packs. Each image or ZIP
 * archive is its own request, so every file reports its own result while the
 * rest of the queue keeps going.
 */
const props = defineProps<{
  upload: (file: File) => Promise<StickerImportResult>
}>()

const emit = defineEmits<{
  /** Fired after a run that added at least one sticker. */
  uploaded: [summary: { imported: number, skipped: number, failed: number }]
  'update:busy': [busy: boolean]
}>()

const MAX_IMAGE_BYTES = 4 << 20
const MAX_ARCHIVE_BYTES = 32 << 20
const CONCURRENCY = 2
const ARCHIVE_TYPES = ['application/zip', 'application/x-zip-compressed']

type EntryState = 'waiting' | 'uploading' | 'done' | 'failed'

interface Entry {
  id: number
  file: File
  archive: boolean
  preview: string
  state: EntryState
  result?: StickerImportResult
  error?: string
}

const reasonTexts: Record<string, string> = {
  entryOpenFailed: 'k00vgu',
  tooLarge: 'k00vgv',
  invalidImage: 'k00vgw',
  unusableName: 'k00vgx',
  saveFailed: 'k00vgy',
  tooManyFiles: 'k00vgz',
  archiveTooLarge: 'k00vh6',
}

const requestErrorTexts: Record<string, string> = {
  'admin.sticker.importTooLarge': 'k00vh2',
  'admin.sticker.importInvalidZip': 'stickerUploadUnsupported',
  'upload.file.missing': 'k00vh4',
  'admin.sticker.notFound': 'k00vh1',
}

const entries = ref<Entry[]>([])
const running = ref(false)
const dragging = ref(false)
const imageInput = ref<HTMLInputElement | null>(null)
const archiveInput = ref<HTMLInputElement | null>(null)
let nextId = 0

const waiting = computed(() => entries.value.filter((entry) => entry.state === 'waiting'))
const finished = computed(() => entries.value.filter((entry) => entry.state === 'done' || entry.state === 'failed').length)
const totals = computed(() => {
  let imported = 0
  let skipped = 0
  let failed = 0
  for (const entry of entries.value) {
    if (entry.result) {
      imported += entry.result.imported
      skipped += entry.result.skipped
      failed += entry.result.failed.length
    } else if (entry.state === 'failed') {
      failed += 1
    }
  }
  return { imported, skipped, failed }
})
const failures = computed(() => entries.value.flatMap((entry) => {
  if (entry.result) return entry.result.failed.map((issue) => ({ name: issue.name, text: reasonText(issue.reason) }))
  return entry.state === 'failed' ? [{ name: entry.file.name, text: entry.error ?? '' }] : []
}))
const progress = computed(() => (entries.value.length ? Math.round((finished.value / entries.value.length) * 100) : 0))

function reasonText(reason: string) {
  const key = reasonTexts[reason]
  return key ? adminText(key) : reason
}

function isArchive(file: File) {
  return ARCHIVE_TYPES.includes(file.type) || file.name.toLowerCase().endsWith('.zip')
}

// Checked before upload so an oversized or wrong file never costs a request.
function precheck(file: File, archive: boolean) {
  if (!archive && !file.type.startsWith('image/')) return adminText('stickerUploadUnsupported')
  if (archive && file.size > MAX_ARCHIVE_BYTES) return adminText('k00vh2')
  if (!archive && file.size > MAX_IMAGE_BYTES) return adminText('stickerUploadImageTooLarge')
  return ''
}

function addFiles(files: FileList | File[] | null | undefined) {
  for (const file of Array.from(files ?? [])) {
    const archive = isArchive(file)
    const error = precheck(file, archive)
    entries.value.push({
      id: nextId++,
      file,
      archive,
      preview: !archive && !error ? URL.createObjectURL(file) : '',
      state: error ? 'failed' : 'waiting',
      error: error || undefined,
    })
  }
}

function onPick(event: Event) {
  const input = event.target as HTMLInputElement
  addFiles(input.files)
  input.value = ''
}

function onDrop(event: DragEvent) {
  dragging.value = false
  if (!running.value) addFiles(event.dataTransfer?.files)
}

function onDragLeave(event: DragEvent) {
  if (!(event.currentTarget as HTMLElement).contains(event.relatedTarget as Node | null)) dragging.value = false
}

function remove(entry: Entry) {
  if (entry.preview) URL.revokeObjectURL(entry.preview)
  entries.value = entries.value.filter((item) => item.id !== entry.id)
}

function clear() {
  for (const entry of entries.value) {
    if (entry.preview) URL.revokeObjectURL(entry.preview)
  }
  entries.value = []
}

async function runEntry(entry: Entry) {
  entry.state = 'uploading'
  try {
    const previous = entry.result
    const file = entry.archive && previous?.failed.length
      ? await failedStickerArchive(entry.file, previous.failed.map(issue => issue.name))
      : entry.file
    const result = await props.upload(file)
    entry.result = { ...result, imported: (previous?.imported ?? 0) + result.imported, skipped: (previous?.skipped ?? 0) + result.skipped }
    if (entry.archive) entry.file = file
    entry.state = result.failed.length === 0 ? 'done' : 'failed'
    if (entry.state === 'failed') entry.error = reasonText(entry.result.failed[0]?.reason ?? '')
  } catch (err) {
    const key = requestErrorTexts[(err as { messageCode?: string } | null)?.messageCode || '']
    entry.state = 'failed'
    entry.error = key ? adminText(key) : adminText('stickerUploadFailed')
  }
}

async function start() {
  if (running.value || waiting.value.length === 0) return
  running.value = true
  emit('update:busy', true)
  const before = totals.value.imported
  const queue = [...waiting.value]
  const worker = async () => {
    for (let entry = queue.shift(); entry; entry = queue.shift()) await runEntry(entry)
  }
  await Promise.all(Array.from({ length: Math.min(CONCURRENCY, queue.length) }, worker))
  running.value = false
  emit('update:busy', false)
  if (totals.value.imported > before) emit('uploaded', { ...totals.value })
}

function retryFailed() {
  for (const entry of entries.value) {
    if (entry.state === 'failed' && !precheck(entry.file, entry.archive)) {
      entry.state = 'waiting'
      entry.error = undefined
    }
  }
  void start()
}

function entryStatus(entry: Entry) {
  switch (entry.state) {
    case 'waiting':
      return adminText('stickerUploadWaiting')
    case 'uploading':
      return adminText('stickerUploadUploading')
    case 'failed':
      return entry.error ?? ''
    default:
      return entry.result
        ? adminText('stickerUploadEntryResult', { imported: entry.result.imported, skipped: entry.result.skipped, failed: entry.result.failed.length })
        : ''
  }
}

function formatSize(bytes: number) {
  return bytes >= 1 << 20 ? `${(bytes / (1 << 20)).toFixed(1)} MB` : `${Math.max(1, Math.round(bytes / 1024))} KB`
}

onBeforeUnmount(clear)

defineExpose({ start, running })
</script>

<template>
  <div class="grid gap-3">
    <div
      class="flex flex-col items-center gap-3 rounded-xl border border-dashed px-4 py-6 text-center transition-colors duration-150"
      :class="dragging ? 'border-primary bg-primary/5' : 'border-border bg-muted/20'"
      @dragenter.prevent="dragging = !running"
      @dragover.prevent
      @dragleave="onDragLeave"
      @drop.prevent="onDrop"
    >
      <ImagePlus class="size-6 text-muted-foreground" aria-hidden="true" />
      <div class="space-y-1">
        <p class="text-sm font-medium">{{ adminText('stickerUploadDrop') }}</p>
        <p class="text-pretty text-xs text-muted-foreground">{{ adminText('stickerUploadLimits') }}</p>
      </div>
      <div class="flex flex-wrap justify-center gap-2">
        <Button variant="outline" size="sm" type="button" :disabled="running" @click="imageInput?.click()">
          <ImagePlus class="size-4" />
          {{ adminText('stickerUploadPickImages') }}
        </Button>
        <Button variant="outline" size="sm" type="button" :disabled="running" @click="archiveInput?.click()">
          <FileArchive class="size-4" />
          {{ adminText('stickerUploadPickArchive') }}
        </Button>
      </div>
      <input ref="imageInput" class="hidden" type="file" multiple accept="image/*" data-testid="sticker-upload-images" @change="onPick" />
      <input ref="archiveInput" class="hidden" type="file" multiple accept=".zip,application/zip,application/x-zip-compressed" data-testid="sticker-upload-archives" @change="onPick" />
    </div>

    <template v-if="entries.length">
      <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground">
        <span class="tabular-nums" aria-live="polite">{{ adminText('stickerUploadProgress', { done: finished, total: entries.length }) }}</span>
        <button v-if="!running" type="button" class="rounded px-1 underline-offset-2 hover:text-foreground hover:underline" @click="clear">
          {{ adminText('stickerUploadClear') }}
        </button>
      </div>
      <div class="h-1 overflow-hidden rounded-full bg-muted" aria-hidden="true">
        <div class="h-full rounded-full bg-primary transition-[width] duration-300 ease-out" :style="{ width: `${progress}%` }" />
      </div>
      <ul class="grid max-h-60 gap-1 overflow-y-auto">
        <li v-for="entry in entries" :key="entry.id" class="flex items-center gap-3 rounded-lg bg-muted/30 py-1.5 ps-1.5 pe-2">
          <img
            v-if="entry.preview"
            :src="entry.preview"
            alt=""
            class="size-9 shrink-0 rounded-md bg-background object-contain outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
          />
          <div v-else class="flex size-9 shrink-0 items-center justify-center rounded-md bg-background" aria-hidden="true">
            <FileArchive class="size-4 text-muted-foreground" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm" :title="entry.file.name">{{ entry.file.name }}</p>
            <p class="truncate text-xs tabular-nums" :class="entry.state === 'failed' ? 'text-destructive' : 'text-muted-foreground'">
              {{ formatSize(entry.file.size) }} · {{ entryStatus(entry) }}
            </p>
          </div>
          <Loader2 v-if="entry.state === 'uploading'" class="size-4 shrink-0 animate-spin text-muted-foreground" aria-hidden="true" />
          <Check v-else-if="entry.state === 'done'" class="size-4 shrink-0 text-primary" aria-hidden="true" />
          <AlertCircle v-else-if="entry.state === 'failed'" class="size-4 shrink-0 text-destructive" aria-hidden="true" />
          <button
            v-if="!running && entry.state !== 'done'"
            type="button"
            class="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 hover:bg-background hover:text-foreground active:scale-[0.96]"
            :aria-label="adminText('stickerUploadRemove', { name: entry.file.name })"
            @click="remove(entry)"
          >
            <X class="size-3.5" />
          </button>
        </li>
      </ul>
      <p v-if="finished > 0 && !running" class="text-sm tabular-nums" role="status">
        {{ adminText('stickerUploadSummary', totals) }}
      </p>
      <details v-if="failures.length && !running" class="text-sm">
        <summary class="cursor-pointer font-medium text-destructive">{{ adminText('stickerUploadFailures', { count: failures.length }) }}</summary>
        <ul class="mt-2 max-h-40 space-y-1 overflow-y-auto">
          <li v-for="(failure, index) in failures" :key="`${failure.name}-${index}`" class="flex items-center justify-between gap-3 rounded-md bg-muted/30 px-2 py-1">
            <span class="truncate font-mono text-xs">{{ failure.name }}</span>
            <span class="shrink-0 text-xs text-muted-foreground">{{ failure.text }}</span>
          </li>
        </ul>
      </details>
      <div class="flex flex-wrap justify-end gap-2">
        <Button v-if="!running && failures.length && totals.failed > 0" variant="outline" size="sm" type="button" @click="retryFailed">
          {{ adminText('stickerUploadRetry') }}
        </Button>
        <Button size="sm" type="button" :disabled="running || waiting.length === 0" @click="start">
          <Loader2 v-if="running" class="size-4 animate-spin" />
          {{ running ? adminText('stickerUploadUploading') : adminText('stickerUploadStart', { count: waiting.length }) }}
        </Button>
      </div>
    </template>
  </div>
</template>

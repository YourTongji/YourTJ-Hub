<script setup lang="ts">
/**
 * 导入 Markdown 文档（issue #1630）：选择/拖入本地 .md 文件 → 解析标题与正文 →
 * 文档里的图片逐个上传到本站（与编辑器上传同一条 processImageFile → uploadImage
 * 链路，审核规则不变）→ 预览最终正文 → 替换或插入到编辑器。
 *
 * 网络图片由浏览器直接下载（服务端不抓取外链图片）；图床不允许跨站下载时，用户可
 * 手动选择本地文件替换。导入时仍未上传的图片：网络图保留为普通链接，本地/内嵌图只保留
 * 替代文字，正文里不会出现无法显示的外链图片。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { AlertTriangle, Check, FileInput, FileText, ImageOff, Loader2, RotateCw, Upload, X } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import GfSwitch from '@/site/components/GfSwitch.vue'
import { uploadImage } from '@/runtime/api'
import { processImageFile, validateImageFile } from '@/runtime/image'
import { renderMarkdownPreview } from '@/runtime/markdown'
import { useResolvedStickers } from '@/site/composables/useResolvedStickers'
import {
  ImportImageError,
  MARKDOWN_IMPORT_ACCEPT,
  MARKDOWN_IMPORT_MAX_BYTES,
  dataUrlToFile,
  downloadRemoteImage,
  imageMarkdown,
  isMarkdownFile,
  matchLocalImage,
  parseMarkdownImport,
  rewriteImportImages,
  unresolvedImageFallback,
  type ImportImage,
  type MarkdownImportResult,
} from '@/runtime/markdown-import'

export interface MarkdownImportPayload {
  title: string
  body: string
  mode: 'replace' | 'insert'
}

const props = defineProps<{
  /** 宿主有标题字段时，文档标题可直接填入 */
  allowTitle?: boolean
  /** 宿主标题为空：默认把文档标题用作标题 */
  titleEmpty?: boolean
  /** 宿主正文为空：默认替换（空正文无需选择插入位置） */
  bodyEmpty?: boolean
  /** 预览档位，与编辑区/发布后一致 */
  profile?: 'post' | 'thought' | 'article'
}>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ apply: [payload: MarkdownImportPayload] }>()

type ItemStatus = 'pending' | 'working' | 'done' | 'needs-file' | 'failed'

interface ImportItem extends ImportImage {
  status: ItemStatus
  url: string
  error: string
  replacement: File | null
}

const { t } = useI18n()
const fileInput = ref<HTMLInputElement | null>(null)
const replaceInput = ref<HTMLInputElement | null>(null)
const dragging = ref(false)
const error = ref('')
const fileName = ref('')
const parsed = ref<MarkdownImportResult | null>(null)
const items = ref<ImportItem[]>([])
const localFiles = ref<File[]>([])
const useTitle = ref(false)
const mode = ref<'replace' | 'insert'>('replace')
let replaceTarget: ImportItem | null = null
let controller = new AbortController()

const PREVIEW_LIMIT = 12000
const CONCURRENCY = 2

const processing = computed(() => items.value.some((item) => item.status === 'pending' || item.status === 'working'))
const uploadable = computed(() => items.value.filter((item) => item.kind !== 'site'))
const doneCount = computed(() => uploadable.value.filter((item) => item.status === 'done').length)
const unresolvedCount = computed(() => uploadable.value.filter((item) => item.status === 'needs-file' || item.status === 'failed').length)
const progressPercent = computed(() => (uploadable.value.length ? Math.round((doneCount.value / uploadable.value.length) * 100) : 100))
const showTitleOption = computed(() => Boolean(props.allowTitle && parsed.value?.title))

const finalBody = computed(() => {
  const result = parsed.value
  if (!result) return ''
  const byUrl = new Map(items.value.map((item) => [item.src, item]))
  let body = rewriteImportImages(result.body, (src, alt) => {
    const item = byUrl.get(src)
    if (!item || item.kind === 'site') return undefined
    if (item.status === 'done' && item.url) return imageMarkdown(alt, item.url)
    return unresolvedImageFallback({ src, alt, kind: item.kind })
  })
  // 不用作标题时，来自首行一级标题的标题回到正文开头
  if (result.titleSource === 'heading' && result.title && !(showTitleOption.value && useTitle.value)) {
    body = `# ${result.title}\n\n${body}`
  }
  return body
})

const { resolvedUrls: stickerUrls, ensureContentStickers } = useResolvedStickers()
watch(finalBody, (body) => {
  if (body.includes('[:sticker:')) void ensureContentStickers([body])
})
const previewHtml = computed(() => renderMarkdownPreview(finalBody.value.slice(0, PREVIEW_LIMIT), { stickers: stickerUrls.value }))
const previewTruncated = computed(() => finalBody.value.length > PREVIEW_LIMIT)
const characterCount = computed(() => finalBody.value.replace(/\s+/g, '').length)
const profileClass = computed(() => (props.profile === 'article' ? 'gf-prose-article' : props.profile === 'thought' ? 'gf-prose-thought' : ''))

function reset() {
  controller.abort()
  controller = new AbortController()
  error.value = ''
  fileName.value = ''
  parsed.value = null
  items.value = []
  localFiles.value = []
  dragging.value = false
}

watch(open, (value) => {
  if (value) {
    reset()
    mode.value = props.bodyEmpty ? 'replace' : 'insert'
  } else {
    controller.abort()
  }
})

onBeforeUnmount(() => controller.abort())

function chooseFile() {
  fileInput.value?.click()
}

function formatSize(bytes: number) {
  return bytes >= 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(1)} MB` : `${Math.max(1, Math.round(bytes / 1024))} KB`
}

async function loadFiles(files: File[]) {
  error.value = ''
  const markdown = files.find(isMarkdownFile)
  if (!markdown) {
    error.value = t('editor.import.errorNoMarkdown')
    return
  }
  if (markdown.size > MARKDOWN_IMPORT_MAX_BYTES) {
    error.value = t('editor.import.errorTooLarge', { size: formatSize(MARKDOWN_IMPORT_MAX_BYTES) })
    return
  }
  controller.abort()
  controller = new AbortController()
  let text: string
  try {
    text = await markdown.text()
  } catch {
    error.value = t('editor.import.errorRead')
    return
  }
  const result = parseMarkdownImport(text, { siteOrigin: window.location.origin })
  if (!result.body.trim() && !result.title) {
    error.value = t('editor.import.errorEmpty')
    return
  }
  fileName.value = markdown.name
  localFiles.value = files.filter((file) => file !== markdown && file.type.startsWith('image/'))
  parsed.value = result
  useTitle.value = Boolean(props.titleEmpty)
  items.value = result.images.map((image) => ({
    ...image,
    status: image.kind === 'site' ? 'done' : 'pending',
    url: image.kind === 'site' ? image.src : '',
    error: '',
    replacement: null,
  }))
  void runQueue()
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (files.length) void loadFiles(files)
}

function onDrop(event: DragEvent) {
  dragging.value = false
  const files = Array.from(event.dataTransfer?.files ?? [])
  if (files.length) void loadFiles(files)
}

function errorMessage(err: unknown): string {
  if (err instanceof ImportImageError) {
    if (err.code === 'blocked') return t('editor.import.errorBlocked')
    if (err.code === 'http') return t('editor.import.errorHttp', { status: err.status })
    if (err.code === 'not-image') return t('editor.import.errorNotImage')
    return t('editor.import.errorInvalid')
  }
  return err instanceof Error && err.message ? err.message : t('api.imageUploadFailed')
}

async function sourceFile(item: ImportItem, signal: AbortSignal): Promise<File | null> {
  if (item.replacement) return item.replacement
  if (item.kind === 'remote') return downloadRemoteImage(item.src, fetch, signal)
  if (item.kind === 'data') return dataUrlToFile(item.src)
  return matchLocalImage(item.src, localFiles.value) ?? null
}

async function resolveItem(item: ImportItem) {
  const { signal } = controller
  item.status = 'working'
  item.error = ''
  try {
    const file = await sourceFile(item, signal)
    if (signal.aborted) return
    if (!file) {
      item.status = 'needs-file'
      return
    }
    const invalid = validateImageFile(file)
    if (invalid) throw new Error(invalid)
    const optimized = await processImageFile(file)
    if (signal.aborted) return
    item.url = await uploadImage(optimized.file)
    item.status = 'done'
  } catch (err) {
    if (signal.aborted) return
    item.status = 'failed'
    item.error = errorMessage(err)
  }
}

/** 两路并发：既不让长文导入等太久，也不容易触发上传频率限制 */
async function runQueue() {
  const queue = items.value.filter((item) => item.status === 'pending')
  let next = 0
  const worker = async () => {
    while (next < queue.length) await resolveItem(queue[next++])
  }
  await Promise.all(Array.from({ length: Math.min(CONCURRENCY, queue.length) }, worker))
}

function retry(item: ImportItem) {
  item.status = 'pending'
  void runQueue()
}

function retryAll() {
  for (const item of items.value) if (item.status === 'failed') item.status = 'pending'
  void runQueue()
}

function chooseReplacement(item: ImportItem) {
  replaceTarget = item
  replaceInput.value?.click()
}

function onReplacementChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  const item = replaceTarget
  replaceTarget = null
  if (!file || !item) return
  item.replacement = file
  item.status = 'pending'
  void runQueue()
}

function itemLabel(item: ImportItem) {
  if (item.alt) return item.alt
  const path = item.src.startsWith('data:') ? t('editor.import.embeddedImage') : item.src.replace(/[?#].*$/, '').split(/[\\/]/).filter(Boolean).pop()
  return path || item.src
}

function itemSource(item: ImportItem) {
  if (item.kind === 'data') return t('editor.import.embeddedImage')
  if (item.kind === 'remote') {
    try {
      return new URL(item.src.startsWith('//') ? `https:${item.src}` : item.src).host
    } catch {
      return item.src
    }
  }
  return item.src
}

function statusText(item: ImportItem) {
  switch (item.status) {
    case 'pending':
      return t('editor.import.statusPending')
    case 'working':
      return item.kind === 'remote' && !item.replacement ? t('editor.import.statusDownloading') : t('editor.import.statusUploading')
    case 'done':
      return item.kind === 'site' ? t('editor.import.statusSite') : t('editor.import.statusDone')
    case 'needs-file':
      return t('editor.import.statusNeedsFile')
    default:
      return item.error
  }
}

function apply() {
  if (!parsed.value || processing.value) return
  emit('apply', {
    title: showTitleOption.value && useTitle.value ? parsed.value.title : '',
    body: finalBody.value,
    mode: props.bodyEmpty ? 'replace' : mode.value,
  })
  open.value = false
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[100] bg-neutral/50 backdrop-blur-sm" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[100] w-[calc(100%-1.5rem)] max-w-2xl -translate-x-1/2 -translate-y-1/2 outline-none sm:w-[calc(100%-3rem)]"
      >
        <div class="gf-menu-surface flex max-h-[min(88vh,760px)] w-full flex-col overflow-hidden">
          <header class="flex items-start gap-3 px-4 pt-4 sm:px-5 sm:pt-5">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <FileInput class="h-5 w-5" />
            </div>
            <div class="min-w-0 flex-1">
              <DialogTitle class="text-base font-semibold text-base-content">{{ t('editor.import.title') }}</DialogTitle>
              <DialogDescription class="mt-1 text-sm leading-6 text-base-content/60 [text-wrap:pretty]">{{ t('editor.import.description') }}</DialogDescription>
            </div>
            <button
              type="button"
              class="relative rounded-md p-1 text-base-content/55 transition-colors hover:bg-base-300 hover:text-base-content/80 after:absolute after:-inset-2 after:content-['']"
              :aria-label="t('common.close')"
              @click="open = false"
            >
              <X class="h-4 w-4" />
            </button>
          </header>

          <div class="min-h-0 flex-1 overflow-y-auto px-4 pb-1 pt-4 sm:px-5">
            <input ref="fileInput" type="file" class="hidden" multiple :accept="`${MARKDOWN_IMPORT_ACCEPT},image/*`" @change="onFileChange" />
            <input ref="replaceInput" type="file" class="hidden" accept="image/*" @change="onReplacementChange" />

            <!-- 选择文件：整块是一个按钮，拖入同样可用 -->
            <button
              v-if="!parsed"
              type="button"
              class="flex w-full flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-6 py-10 text-center transition-colors duration-150 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              :class="dragging ? 'border-primary bg-primary/5' : 'border-base-content/20 hover:border-base-content/35 hover:bg-base-200/60'"
              @click="chooseFile"
              @dragenter.prevent="dragging = true"
              @dragover.prevent="dragging = true"
              @dragleave.prevent="dragging = false"
              @drop.prevent="onDrop"
            >
              <Upload class="h-6 w-6 text-base-content/45" />
              <span class="text-sm font-medium text-base-content">{{ t('editor.import.dropTitle') }}</span>
              <span class="max-w-sm text-xs leading-5 text-base-content/55 [text-wrap:pretty]">{{ t('editor.import.dropHint', { size: formatSize(MARKDOWN_IMPORT_MAX_BYTES) }) }}</span>
            </button>

            <template v-else>
              <!-- 文档信息 -->
              <div class="flex items-center gap-3 rounded-xl bg-base-200/70 px-3 py-2.5">
                <FileText class="h-4 w-4 shrink-0 text-base-content/55" />
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-base-content" :title="fileName">{{ fileName }}</p>
                  <p class="text-xs tabular-nums text-base-content/55">
                    {{ t('editor.import.summary', { chars: characterCount, images: uploadable.length }) }}
                  </p>
                </div>
                <button type="button" class="gf-button gf-button-sm gf-button-ghost shrink-0" @click="chooseFile">
                  {{ t('editor.import.changeFile') }}
                </button>
              </div>

              <!-- 标题 -->
              <label v-if="showTitleOption" class="mt-3 flex items-center gap-3 rounded-xl px-3 py-2 hover:bg-base-200/50">
                <span class="min-w-0 flex-1 text-sm text-base-content/80">
                  {{ t('editor.import.useTitle') }}
                  <span class="font-medium text-base-content">{{ parsed.title }}</span>
                </span>
                <GfSwitch v-model="useTitle" :label="t('editor.import.useTitle')" />
              </label>

              <!-- 图片 -->
              <section v-if="uploadable.length" class="mt-4">
                <div class="mb-2 flex items-center gap-2">
                  <h3 class="text-sm font-semibold text-base-content">{{ t('editor.import.images') }}</h3>
                  <span class="text-xs tabular-nums text-base-content/55">{{ doneCount }}/{{ uploadable.length }}</span>
                  <button
                    v-if="items.some((item) => item.status === 'failed') && !processing"
                    type="button"
                    class="gf-button gf-button-sm gf-button-ghost ml-auto"
                    @click="retryAll"
                  >
                    <RotateCw class="h-3.5 w-3.5" />
                    {{ t('editor.import.retryAll') }}
                  </button>
                </div>
                <div class="mb-2 h-1 overflow-hidden rounded-full bg-base-content/10" role="progressbar" :aria-valuenow="progressPercent" aria-valuemin="0" aria-valuemax="100">
                  <div class="h-full rounded-full bg-primary transition-[width] duration-300 ease-out" :style="{ width: `${progressPercent}%` }" />
                </div>
                <ul class="max-h-56 space-y-1 overflow-y-auto">
                  <li v-for="item in uploadable" :key="item.src" class="flex items-center gap-3 rounded-lg px-2 py-1.5">
                    <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md" :class="item.status === 'done' ? 'bg-success/10 text-success' : item.status === 'failed' || item.status === 'needs-file' ? 'bg-warning/10 text-warning' : 'bg-base-200 text-base-content/50'">
                      <Loader2 v-if="item.status === 'working'" class="h-3.5 w-3.5 animate-spin" />
                      <Check v-else-if="item.status === 'done'" class="h-3.5 w-3.5" />
                      <AlertTriangle v-else-if="item.status === 'failed'" class="h-3.5 w-3.5" />
                      <ImageOff v-else-if="item.status === 'needs-file'" class="h-3.5 w-3.5" />
                      <Upload v-else class="h-3.5 w-3.5" />
                    </span>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-sm text-base-content" :title="item.src.startsWith('data:') ? undefined : item.src">{{ itemLabel(item) }}</p>
                      <p class="truncate text-xs text-base-content/55" :class="{ 'text-warning': item.status === 'failed' || item.status === 'needs-file' }">
                        <span class="text-base-content/45">{{ itemSource(item) }}</span>
                        <span aria-hidden="true"> · </span>{{ statusText(item) }}
                      </p>
                    </div>
                    <button
                      v-if="item.status === 'failed' && item.kind === 'remote' && !item.replacement"
                      type="button"
                      class="gf-button gf-button-sm gf-button-ghost shrink-0"
                      :aria-label="t('editor.import.retry')"
                      @click="retry(item)"
                    >
                      <RotateCw class="h-3.5 w-3.5" />
                    </button>
                    <button
                      v-if="item.status === 'failed' || item.status === 'needs-file'"
                      type="button"
                      class="gf-button gf-button-sm gf-button-outline shrink-0"
                      @click="chooseReplacement(item)"
                    >
                      {{ t('editor.import.chooseImage') }}
                    </button>
                  </li>
                </ul>
                <p v-if="unresolvedCount && !processing" class="mt-2 flex items-start gap-2 text-xs leading-5 text-base-content/60">
                  <AlertTriangle class="mt-0.5 h-3.5 w-3.5 shrink-0 text-warning" />
                  <span>{{ t('editor.import.unresolvedNote', { count: unresolvedCount }) }}</span>
                </p>
              </section>

              <p v-if="parsed.htmlTagCount" class="mt-3 flex items-start gap-2 text-xs leading-5 text-base-content/60">
                <AlertTriangle class="mt-0.5 h-3.5 w-3.5 shrink-0 text-warning" />
                <span>{{ t('editor.import.htmlNote', { count: parsed.htmlTagCount }) }}</span>
              </p>

              <!-- 预览：即将写入编辑器的最终正文（未上传的图片已降级） -->
              <section class="mt-4">
                <h3 class="mb-2 text-sm font-semibold text-base-content">{{ t('editor.import.preview') }}</h3>
                <div class="max-h-72 overflow-y-auto rounded-xl border border-line px-4 py-3">
                  <div class="gf-prose gf-prose-post" :class="profileClass" v-html="previewHtml" />
                  <p v-if="previewTruncated" class="mt-3 text-xs text-base-content/50">{{ t('editor.import.previewTruncated') }}</p>
                </div>
              </section>

              <!-- 写入方式：正文为空时直接替换，不打扰 -->
              <fieldset v-if="!bodyEmpty" class="mt-4">
                <legend class="mb-2 text-sm font-semibold text-base-content">{{ t('editor.import.modeLabel') }}</legend>
                <div class="grid gap-2 sm:grid-cols-2">
                  <label
                    v-for="option in (['insert', 'replace'] as const)"
                    :key="option"
                    class="flex cursor-pointer items-start gap-2.5 rounded-xl border px-3 py-2.5 transition-colors duration-150 ease-out"
                    :class="mode === option ? 'border-primary/60 bg-primary/5' : 'border-line hover:bg-base-200/60'"
                  >
                    <input v-model="mode" type="radio" name="markdown-import-mode" :value="option" class="mt-1 accent-[var(--gf-color-primary)]" />
                    <span class="min-w-0">
                      <span class="block text-sm font-medium text-base-content">{{ option === 'insert' ? t('editor.import.modeInsert') : t('editor.import.modeReplace') }}</span>
                      <span class="block text-xs leading-5 text-base-content/55">{{ option === 'insert' ? t('editor.import.modeInsertHint') : t('editor.import.modeReplaceHint') }}</span>
                    </span>
                  </label>
                </div>
              </fieldset>
            </template>

            <p v-if="error" class="mt-3 flex items-start gap-2 rounded-xl bg-error/10 px-3 py-2.5 text-sm leading-5 text-error" role="alert">
              <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" />
              <span>{{ error }}</span>
            </p>
          </div>

          <footer class="flex flex-col-reverse gap-2 px-4 pb-4 pt-4 sm:flex-row sm:justify-end sm:px-5 sm:pb-5">
            <button type="button" class="gf-button gf-button-lg gf-button-muted active:scale-[0.96]" @click="open = false">
              {{ t('common.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-lg gf-button-primary min-w-28 active:scale-[0.96]"
              :disabled="!parsed || processing"
              @click="apply"
            >
              <Loader2 v-if="processing" class="h-4 w-4 animate-spin" />
              {{ processing ? t('editor.import.processing') : t('editor.import.confirm') }}
            </button>
          </footer>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

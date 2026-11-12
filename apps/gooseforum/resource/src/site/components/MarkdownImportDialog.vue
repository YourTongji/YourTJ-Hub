<script setup lang="ts">
/**
 * 导入 Markdown 文档（issue #1630）：选择/拖入本地 .md 文件 → 解析标题与正文 →
 * 文档里的图片逐个上传到本站（与编辑器上传同一条 processImageFile → uploadImage
 * 链路，审核规则不变）→ 预览最终正文 → 替换或插入到编辑器。
 *
 * 网络图片由浏览器直接下载（服务端不抓取外链图片）；图床不允许跨站下载时，用户可
 * 手动选择本地文件替换。本地图片按文档里的相对路径，在选择/拖入的文件夹中自动匹配；
 * 找不到的可以补选文件夹或图片后一次性重新匹配。已上传的图片可以替换、移除或还原；
 * 被替换/移除/放弃的上传不会出现在正文里，由孤儿附件清理回收（OrphanUploadTTL）。
 * 导入时仍未上传的图片：网络图保留为普通链接，本地/内嵌图只保留替代文字，正文里不会出现
 * 无法显示的外链图片。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { AlertTriangle, Check, FileInput, FileText, FolderOpen, ImageOff, Loader2, RotateCcw, RotateCw, Upload, X } from '@lucide/vue'
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
  collectDroppedFiles,
  dataUrlToFile,
  downloadRemoteImage,
  imageMarkdown,
  isImportImageFile,
  isMarkdownFile,
  localImportFiles,
  matchLocalImage,
  parseMarkdownImport,
  rewriteImportImages,
  unresolvedImageFallback,
  type ImportImage,
  type LocalImportFile,
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
  /** 自动匹配到的本地文件路径，便于核对 */
  matchedPath: string
  /** 用户选择不使用这张图：导入时按未上传处理（链接/替代文字） */
  excluded: boolean
  /** 替换前已上传的地址：替换后可以还原 */
  previousUrl: string
}

const { t } = useI18n()
const fileInput = ref<HTMLInputElement | null>(null)
const folderInput = ref<HTMLInputElement | null>(null)
const imagesInput = ref<HTMLInputElement | null>(null)
const replaceInput = ref<HTMLInputElement | null>(null)
const dragging = ref(false)
const error = ref('')
const fileName = ref('')
const parsed = ref<MarkdownImportResult | null>(null)
const items = ref<ImportItem[]>([])
const markdownPath = ref('')
const markdownChoices = ref<LocalImportFile[]>([])
const localFiles = ref<LocalImportFile[]>([])
const useTitle = ref(false)
const mode = ref<'replace' | 'insert'>('replace')
let replaceTarget: ImportItem | null = null
let controller = new AbortController()

const PREVIEW_LIMIT = 12000
const CONCURRENCY = 2

const processing = computed(() => items.value.some((item) => item.status === 'pending' || item.status === 'working'))
const uploadable = computed(() => items.value.filter((item) => item.kind !== 'site'))
const doneCount = computed(() => uploadable.value.filter((item) => item.status === 'done' && !item.excluded).length)
const excludedCount = computed(() => uploadable.value.filter((item) => item.excluded).length)
const unresolvedCount = computed(() => uploadable.value.filter((item) => !item.excluded && (item.status === 'needs-file' || item.status === 'failed')).length)
const missingLocalCount = computed(() => uploadable.value.filter((item) => !item.excluded && item.status === 'needs-file').length)
const settledCount = computed(() => doneCount.value + excludedCount.value)
const progressPercent = computed(() => (uploadable.value.length ? Math.round((settledCount.value / uploadable.value.length) * 100) : 100))
const showTitleOption = computed(() => Boolean(props.allowTitle && parsed.value?.title))

const finalBody = computed(() => {
  const result = parsed.value
  if (!result) return ''
  const byUrl = new Map(items.value.map((item) => [item.src, item]))
  let body = rewriteImportImages(result.body, (src, alt) => {
    const item = byUrl.get(src)
    if (!item || item.kind === 'site') return undefined
    if (item.status === 'done' && item.url && !item.excluded) return imageMarkdown(alt, item.url)
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
  markdownPath.value = ''
  markdownChoices.value = []
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

// 模板里的 webkitdirectory 没有类型声明，直接设属性；对话框内容打开后才渲染
watch(folderInput, (input) => {
  if (input) input.webkitdirectory = true
})
onBeforeUnmount(() => controller.abort())

function chooseFile() {
  fileInput.value?.click()
}

function chooseFolder() {
  folderInput.value?.click()
}

function chooseImages() {
  imagesInput.value?.click()
}

function formatSize(bytes: number) {
  return bytes >= 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(1)} MB` : `${Math.max(1, Math.round(bytes / 1024))} KB`
}

/** 加入本地图片（按路径去重），并让找不到文件的图片重新匹配 */
function addLocalFiles(files: readonly LocalImportFile[]) {
  const known = new Set(localFiles.value.map((entry) => entry.path))
  const added = files.filter((entry) => isImportImageFile(entry.file) && !known.has(entry.path))
  if (!added.length) return
  localFiles.value = [...localFiles.value, ...added]
  for (const item of items.value) if (item.kind === 'local' && item.status === 'needs-file' && !item.replacement) item.status = 'pending'
  void runQueue()
}

function shallowFirst(a: LocalImportFile, b: LocalImportFile) {
  return a.path.split('/').length - b.path.split('/').length || a.path.localeCompare(b.path)
}

async function loadFiles(files: LocalImportFile[]) {
  error.value = ''
  const documents = files.filter((entry) => isMarkdownFile(entry.file)).sort(shallowFirst)
  // 文件夹里同时有 .md 与 .txt 时，.txt 多半只是附带说明，优先 Markdown
  const markdownOnly = documents.filter((entry) => !/\.txt$/i.test(entry.file.name))
  const markdowns = markdownOnly.length ? markdownOnly : documents
  if (!markdowns.length) {
    // 已导入文档时，单独选择/拖入的图片或文件夹用来补齐本地图片
    if (parsed.value) addLocalFiles(files)
    else error.value = t('editor.import.errorNoMarkdown')
    return
  }
  const images = files.filter((entry) => isImportImageFile(entry.file))
  if (markdowns.length > 1) {
    // 文件夹里有多份文档：让用户选一份，图片留着做匹配
    controller.abort()
    controller = new AbortController()
    parsed.value = null
    items.value = []
    markdownChoices.value = markdowns
    localFiles.value = images
    return
  }
  await loadMarkdown(markdowns[0], images)
}

async function loadMarkdown(markdown: LocalImportFile, images: LocalImportFile[]) {
  error.value = ''
  if (markdown.file.size > MARKDOWN_IMPORT_MAX_BYTES) {
    error.value = t('editor.import.errorTooLarge', { size: formatSize(MARKDOWN_IMPORT_MAX_BYTES) })
    return
  }
  controller.abort()
  controller = new AbortController()
  let text: string
  try {
    text = await markdown.file.text()
  } catch {
    error.value = t('editor.import.errorRead')
    return
  }
  const result = parseMarkdownImport(text, { siteOrigin: window.location.origin })
  if (!result.body.trim() && !result.title) {
    error.value = t('editor.import.errorEmpty')
    return
  }
  fileName.value = markdown.file.name
  markdownPath.value = markdown.path
  markdownChoices.value = []
  localFiles.value = images
  parsed.value = result
  useTitle.value = Boolean(props.titleEmpty)
  items.value = result.images.map((image) => ({
    ...image,
    status: image.kind === 'site' ? 'done' : 'pending',
    url: image.kind === 'site' ? image.src : '',
    error: '',
    replacement: null,
    matchedPath: '',
    excluded: false,
    previousUrl: '',
  }))
  void runQueue()
}

function chooseMarkdown(markdown: LocalImportFile) {
  void loadMarkdown(markdown, localFiles.value)
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = localImportFiles(Array.from(input.files ?? []))
  input.value = ''
  if (files.length) void loadFiles(files)
}

/** 已载入文档后选择的文件夹只用来补齐图片：重新载入会丢掉替换/移除并重复上传 */
function onFolderChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = localImportFiles(Array.from(input.files ?? []))
  input.value = ''
  if (!files.length) return
  if (parsed.value) addLocalFiles(files)
  else void loadFiles(files)
}

function onImagesChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = localImportFiles(Array.from(input.files ?? []))
  input.value = ''
  if (files.length) addLocalFiles(files)
}

function onDrop(event: DragEvent) {
  dragging.value = false
  if (!event.dataTransfer) return
  void collectDroppedFiles(event.dataTransfer).then((files) => {
    if (files.length) return loadFiles(files)
  })
}

function onSupplementDrop(event: DragEvent) {
  dragging.value = false
  if (!event.dataTransfer) return
  void collectDroppedFiles(event.dataTransfer).then(addLocalFiles)
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
  const match = matchLocalImage(item.src, localFiles.value, markdownPath.value)
  item.matchedPath = match?.path ?? ''
  return match?.file ?? null
}

async function resolveItem(item: ImportItem) {
  // 重试/补选可能在队列运行中再起一个队列：已被领走的不再处理
  if (item.status !== 'pending') return
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

function excludeItem(item: ImportItem) {
  item.excluded = true
}

function restoreItem(item: ImportItem) {
  item.excluded = false
}

/** 还原到替换前已上传的图片 */
function revertReplacement(item: ImportItem) {
  if (!item.previousUrl) return
  item.url = item.previousUrl
  item.previousUrl = ''
  item.replacement = null
  item.error = ''
  item.status = 'done'
}

function onReplacementChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  const item = replaceTarget
  replaceTarget = null
  if (!file || !item) return
  if (item.status === 'done' && item.url && !item.previousUrl) item.previousUrl = item.url
  item.replacement = file
  item.excluded = false
  item.status = 'pending'
  void runQueue()
}

function itemLabel(item: ImportItem) {
  if (item.alt) return item.alt
  const path = item.src.startsWith('data:') ? t('editor.import.embeddedImage') : item.src.replace(/[?#].*$/, '').split(/[\\/]/).filter(Boolean).pop()
  return path || item.src
}

function itemSource(item: ImportItem) {
  if (item.replacement) return item.replacement.name
  if (item.kind === 'data') return t('editor.import.embeddedImage')
  if (item.matchedPath) return item.matchedPath
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
  if (item.excluded) return item.kind === 'remote' ? t('editor.import.statusExcludedLink') : t('editor.import.statusExcludedText')
  switch (item.status) {
    case 'pending':
      return t('editor.import.statusPending')
    case 'working':
      return item.kind === 'remote' && !item.replacement ? t('editor.import.statusDownloading') : t('editor.import.statusUploading')
    case 'done':
      if (item.kind === 'site') return t('editor.import.statusSite')
      return item.previousUrl ? t('editor.import.statusReplaced') : t('editor.import.statusDone')
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
            <input ref="folderInput" type="file" class="hidden" multiple @change="onFolderChange" />
            <input ref="imagesInput" type="file" class="hidden" multiple accept="image/*" @change="onImagesChange" />
            <input ref="replaceInput" type="file" class="hidden" accept="image/*" @change="onReplacementChange" />

            <!-- 选择文件：整块是一个按钮，拖入文件或文件夹同样可用 -->
            <div
              v-if="!parsed && !markdownChoices.length"
              class="rounded-xl border border-dashed transition-colors duration-150 ease-out"
              :class="dragging ? 'border-primary bg-primary/5' : 'border-base-content/20'"
              data-test="import-dropzone"
              @dragenter.prevent="dragging = true"
              @dragover.prevent="dragging = true"
              @dragleave.prevent="dragging = false"
              @drop.prevent="onDrop"
            >
              <button
                type="button"
                class="flex w-full flex-col items-center justify-center gap-2 rounded-xl px-6 pb-4 pt-10 text-center transition-colors duration-150 ease-out hover:bg-base-200/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                @click="chooseFile"
              >
                <Upload class="h-6 w-6 text-base-content/45" />
                <span class="text-sm font-medium text-base-content">{{ t('editor.import.dropTitle') }}</span>
                <span class="max-w-sm text-xs leading-5 text-base-content/55 [text-wrap:pretty]">{{ t('editor.import.dropHint', { size: formatSize(MARKDOWN_IMPORT_MAX_BYTES) }) }}</span>
              </button>
              <div class="flex justify-center pb-6">
                <button type="button" class="gf-button gf-button-sm gf-button-outline" @click="chooseFolder">
                  <FolderOpen class="h-3.5 w-3.5" />
                  {{ t('editor.import.chooseFolder') }}
                </button>
              </div>
            </div>

            <!-- 文件夹里有多份文档：选一份导入 -->
            <section v-else-if="!parsed" data-test="import-markdown-choices">
              <h3 class="mb-2 text-sm font-semibold text-base-content">{{ t('editor.import.chooseMarkdown', { count: markdownChoices.length }) }}</h3>
              <ul class="max-h-72 space-y-1 overflow-y-auto">
                <li v-for="choice in markdownChoices" :key="choice.path">
                  <button
                    type="button"
                    class="flex w-full items-center gap-3 rounded-lg px-2 py-2 text-start transition-colors duration-150 ease-out hover:bg-base-200/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                    @click="chooseMarkdown(choice)"
                  >
                    <FileText class="h-4 w-4 shrink-0 text-base-content/55" />
                    <span class="min-w-0 flex-1 truncate text-sm text-base-content" :title="choice.path">{{ choice.path }}</span>
                    <span class="shrink-0 text-xs tabular-nums text-base-content/50">{{ formatSize(choice.file.size) }}</span>
                  </button>
                </li>
              </ul>
            </section>

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
                  <span class="text-xs tabular-nums text-base-content/55">{{ settledCount }}/{{ uploadable.length }}</span>
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
                  <li v-for="item in uploadable" :key="item.src" class="flex items-center gap-3 rounded-lg px-2 py-1.5" data-test="import-image">
                    <img
                      v-if="item.status === 'done' && item.url && !item.excluded"
                      :src="item.url"
                      alt=""
                      class="h-8 w-8 shrink-0 rounded-md object-cover outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
                      loading="lazy"
                    />
                    <span v-else class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md" :class="item.excluded ? 'bg-base-200 text-base-content/40' : item.status === 'failed' || item.status === 'needs-file' ? 'bg-warning/10 text-warning' : 'bg-base-200 text-base-content/50'">
                      <ImageOff v-if="item.excluded" class="h-3.5 w-3.5" />
                      <Loader2 v-else-if="item.status === 'working'" class="h-3.5 w-3.5 animate-spin" />
                      <Check v-else-if="item.status === 'done'" class="h-3.5 w-3.5" />
                      <AlertTriangle v-else-if="item.status === 'failed'" class="h-3.5 w-3.5" />
                      <ImageOff v-else-if="item.status === 'needs-file'" class="h-3.5 w-3.5" />
                      <Upload v-else class="h-3.5 w-3.5" />
                    </span>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-sm" :class="item.excluded ? 'text-base-content/50 line-through' : 'text-base-content'" :title="item.src.startsWith('data:') ? undefined : item.src">{{ itemLabel(item) }}</p>
                      <p class="truncate text-xs text-base-content/55" :class="{ 'text-warning': !item.excluded && (item.status === 'failed' || item.status === 'needs-file') }">
                        <span class="text-base-content/45">{{ itemSource(item) }}</span>
                        <span aria-hidden="true"> · </span>{{ statusText(item) }}
                      </p>
                    </div>
                    <template v-if="item.excluded">
                      <button type="button" class="gf-button gf-button-sm gf-button-ghost shrink-0" @click="restoreItem(item)">
                        {{ t('editor.import.restoreImage') }}
                      </button>
                    </template>
                    <template v-else>
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
                        v-if="item.previousUrl && (item.status === 'done' || item.status === 'failed')"
                        type="button"
                        class="gf-button gf-button-sm gf-button-ghost shrink-0"
                        :aria-label="t('editor.import.revertReplacement')"
                        :title="t('editor.import.revertReplacement')"
                        @click="revertReplacement(item)"
                      >
                        <RotateCcw class="h-3.5 w-3.5" />
                      </button>
                      <button
                        v-if="item.status === 'failed' || item.status === 'needs-file'"
                        type="button"
                        class="gf-button gf-button-sm gf-button-outline shrink-0"
                        @click="chooseReplacement(item)"
                      >
                        {{ t('editor.import.chooseImage') }}
                      </button>
                      <template v-else-if="item.status === 'done'">
                        <button type="button" class="gf-button gf-button-sm gf-button-ghost shrink-0" @click="chooseReplacement(item)">
                          {{ t('editor.import.replaceImage') }}
                        </button>
                        <button
                          type="button"
                          class="gf-button gf-button-sm gf-button-ghost shrink-0"
                          :aria-label="t('editor.import.removeImage')"
                          :title="t('editor.import.removeImage')"
                          @click="excludeItem(item)"
                        >
                          <X class="h-3.5 w-3.5" />
                        </button>
                      </template>
                    </template>
                  </li>
                </ul>
                <!-- 本地图片没找到：补选文件夹或图片（也可拖入），按相对路径一次性重新匹配 -->
                <div
                  v-if="missingLocalCount && !processing"
                  class="mt-2 rounded-xl border border-dashed px-3 py-2.5 transition-colors duration-150 ease-out"
                  :class="dragging ? 'border-primary bg-primary/5' : 'border-warning/40 bg-warning/5'"
                  data-test="import-missing-local"
                  @dragenter.prevent="dragging = true"
                  @dragover.prevent="dragging = true"
                  @dragleave.prevent="dragging = false"
                  @drop.prevent="onSupplementDrop"
                >
                  <p class="text-sm font-medium text-base-content">{{ t('editor.import.missingLocalTitle', { count: missingLocalCount }) }}</p>
                  <p class="mt-0.5 text-xs leading-5 text-base-content/60 [text-wrap:pretty]">{{ t('editor.import.missingLocalHint') }}</p>
                  <div class="mt-2 flex flex-wrap gap-2">
                    <button type="button" class="gf-button gf-button-sm gf-button-outline" @click="chooseFolder">
                      <FolderOpen class="h-3.5 w-3.5" />
                      {{ t('editor.import.chooseFolder') }}
                    </button>
                    <button type="button" class="gf-button gf-button-sm gf-button-ghost" @click="chooseImages">
                      {{ t('editor.import.chooseImages') }}
                    </button>
                  </div>
                </div>
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

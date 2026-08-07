<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import {
  Bold,
  Check,
  ClipboardPaste,
  Code,
  Code2,
  Eye,
  Image,
  Italic,
  Link,
  List,
  ListOrdered,
  Loader2,
  MessageSquareQuote,
  Send,
  Strikethrough,
  X,
} from '@lucide/vue'
import { uploadImage } from '@/runtime/api'
import { processImageFile, validateImageFile } from '@/runtime/image'
import { markdownFromClipboard } from '@/runtime/rich-paste'
import { hasUnsupportedVisualMarkdown } from '@/runtime/rich-paste'
import { renderMarkdownPreview } from '@/runtime/markdown'
import { fencedCodeBlock, prefixMarkdownBlock, replaceMarkdownSelectionWithBlock } from '@/runtime/markdown-editing'
import VisualMarkdownEditor from '@/site/components/VisualMarkdownEditor.vue'
import { useKeyboardVisualViewportOffset } from '@/runtime/visual-viewport'
import type { PostPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  authenticated: boolean
  errorMessage: string
  mode?: 'create' | 'edit'
  open: boolean
  submitting: boolean
  successMessage: string
  target?: PostPayload
  captchaRequired?: boolean
  captchaImg?: string
  captchaLoading?: boolean
}>()

const emit = defineEmits<{
  clearTarget: []
  clearValidation: []
  imageError: [message: string]
  imageInserted: [count: number]
  submit: []
  refreshCaptcha: []
  'update:open': [value: boolean]
}>()

const captchaCode = defineModel<string>('captchaCode', { default: '' })
const content = defineModel<string>({ default: '' })
const { t } = useI18n()
// 软键盘弹出时抬高浮动面板，确保输入内容不被输入法遮挡
const { bottomOffset: keyboardOffset } = useKeyboardVisualViewportOffset()

const editorMode = ref<'visual' | 'markdown'>('visual')
const preview = ref(false)
const toolbarOpen = ref(false)
const toolbarCloseTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const linkPickerOpen = ref(false)
const linkUrl = ref('')
const visualEditor = ref<InstanceType<typeof VisualMarkdownEditor> | null>(null)
const markdownEditor = ref<HTMLTextAreaElement | null>(null)
const uploadingImage = ref(false)
const dragOver = ref(false)
const composerBusy = computed(() => props.submitting || uploadingImage.value)
const editing = computed(() => props.mode === 'edit')
const composerTitle = computed(() => editing.value ? t('topic.editOwnReply') : t('topic.joinDiscussion'))
const composerPlaceholder = computed(() => editing.value ? t('topic.editReplyPlaceholder') : t('topic.replyPlaceholder'))
const submitText = computed(() => {
  if (uploadingImage.value) return t('publish.processingImage')
  if (props.submitting) return editing.value ? t('common.saving') : t('topic.publishing')
  return editing.value ? t('common.save') : t('topic.publishReply')
})
const renderedPreview = computed(() => renderMarkdownPreview(content.value))
const showToolbar = computed(() => toolbarOpen.value || preview.value || content.value.trim().length > 0)

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    await nextTick()
    window.requestAnimationFrame(() => {
      if (editorMode.value === 'visual') visualEditor.value?.focus()
      else markdownEditor.value?.focus()
    })
  },
  { immediate: true },
)

function closeComposer() {
  if (composerBusy.value) return
  emit('update:open', false)
}

function openLinkPicker() {
  if (editorMode.value === 'markdown') {
    insert('[', '](https://)', t('publish.placeholder.link'))
    return
  }
  linkPickerOpen.value = !linkPickerOpen.value
}

async function applyLink() {
  const url = linkUrl.value.trim()
  if (!url) return
  if (editorMode.value === 'visual') {
    visualEditor.value?.setLink(url, url)
    linkPickerOpen.value = false
    linkUrl.value = ''
    await nextTick()
    visualEditor.value?.focus()
    return
  }
  insert('[', `](${url})`, t('publish.placeholder.link'))
  linkUrl.value = ''
}

function scheduleToolbarClose() {
  if (toolbarCloseTimer.value) clearTimeout(toolbarCloseTimer.value)
  toolbarCloseTimer.value = setTimeout(() => {
    if (!linkPickerOpen.value) toolbarOpen.value = false
  }, 150)
}

function keepToolbarOpen() {
  if (toolbarCloseTimer.value) clearTimeout(toolbarCloseTimer.value)
  toolbarOpen.value = true
}

async function selectEditorMode(mode: 'visual' | 'markdown') {
  if (editorMode.value === mode && !preview.value) return
  if (mode === 'visual' && hasUnsupportedVisualMarkdown(content.value)) {
    emit('imageError', t('publish.visualUnsupported'))
    return
  }
  linkPickerOpen.value = false
  editorMode.value = mode
  preview.value = false
  await nextTick()
  if (mode === 'visual') visualEditor.value?.focus()
  else markdownEditor.value?.focus()
}

async function togglePreview() {
  linkPickerOpen.value = false
  preview.value = !preview.value
  if (!preview.value) {
    await nextTick()
    if (editorMode.value === 'visual') visualEditor.value?.focus()
    else markdownEditor.value?.focus()
  }
}

type ToolbarAction = 'bold' | 'italic' | 'strike' | 'inlineCode' | 'quote' | 'code' | 'bulletList' | 'orderedList'

function applyToolbarAction(action: ToolbarAction) {
  if (editorMode.value === 'markdown') {
    if (action === 'bold') insert('**', '**', t('publish.placeholder.bold'))
    else if (action === 'italic') insert('*', '*', t('publish.placeholder.italic'))
    else if (action === 'strike') insert('~~', '~~', t('publish.placeholder.strike'))
    else if (action === 'inlineCode') insert('`', '`', 'code')
    else if (action === 'quote') insertPrefixedMarkdownBlock('> ', t('publish.placeholder.quote'))
    else if (action === 'code') insertFencedCodeBlock()
    else if (action === 'bulletList') insertPrefixedMarkdownBlock('- ', t('publish.placeholder.listItem'))
    else insertPrefixedMarkdownBlock('1. ', t('publish.placeholder.listItem'))
    return
  }

  visualEditor.value?.applyAction(action)
}

function insert(before: string, after = '', placeholder = '') {
  const el = markdownEditor.value
  if (!el) {
    content.value = content.value ? `${content.value}\n${before}${placeholder}${after}` : `${before}${placeholder}${after}`
    return
  }
  const start = el.selectionStart
  const end = el.selectionEnd
  const selected = content.value.slice(start, end) || placeholder
  content.value = `${content.value.slice(0, start)}${before}${selected}${after}${content.value.slice(end)}`
  nextTick(() => {
    el.focus()
    el.setSelectionRange(start + before.length, start + before.length + selected.length)
  })
}

function insertMarkdownBlock(text: string) {
  if (editorMode.value === 'visual') {
    visualEditor.value?.insertMarkdown(text)
    return
  }
  const el = markdownEditor.value
  if (!el) {
    content.value = content.value ? `${content.value}\n${text}` : text
    return
  }
  const start = el.selectionStart
  const end = el.selectionEnd
  const result = replaceMarkdownSelectionWithBlock(content.value, start, end, text)
  content.value = result.value
  nextTick(() => {
    el.focus()
    el.setSelectionRange(result.selectionEnd, result.selectionEnd)
  })
}

function insertPrefixedMarkdownBlock(prefix: string, placeholder: string) {
  const el = markdownEditor.value
  if (!el) {
    content.value = content.value ? `${content.value}\n${prefix}${placeholder}` : `${prefix}${placeholder}`
    return
  }
  const selected = content.value.slice(el.selectionStart, el.selectionEnd) || placeholder
  insertMarkdownBlock(prefixMarkdownBlock(selected, prefix))
}

function insertFencedCodeBlock() {
  const el = markdownEditor.value
  const selected = el ? content.value.slice(el.selectionStart, el.selectionEnd) || 'code' : 'code'
  insertMarkdownBlock(fencedCodeBlock(selected))
}

async function pastePlainText() {
  try {
    const text = await navigator.clipboard.readText()
    if (!text) return
    if (editorMode.value === 'visual') {
      visualEditor.value?.insertText(text)
    } else {
      const el = markdownEditor.value
      if (!el) {
        content.value += text
        return
      }
      const start = el.selectionStart
      const end = el.selectionEnd
      content.value = `${content.value.slice(0, start)}${text}${content.value.slice(end)}`
      nextTick(() => {
        el.focus()
        el.setSelectionRange(start + text.length, start + text.length)
      })
    }
  } catch {
    emit('imageError', t('publish.clipboardReadFailed'))
  }
}

function imageAlt(filename: string) {
  return filename.replace(/\.[^.]+$/, '').replace(/[[\]\n\r]/g, ' ').trim() || 'image'
}

function imageFilesFromList(files: FileList | File[] | null | undefined) {
  return Array.from(files || []).filter((file) => file.type.startsWith('image/'))
}

function imageFilesFromDataTransfer(dataTransfer: DataTransfer | null) {
  if (!dataTransfer) return []
  return imageFilesFromList(dataTransfer.files)
}

function hasImageDataTransfer(dataTransfer: DataTransfer | null) {
  if (!dataTransfer) return false
  if (Array.from(dataTransfer.items || []).some((item) => item.kind === 'file' && item.type.startsWith('image/'))) return true
  return imageFilesFromList(dataTransfer.files).length > 0
}

function imageFilesFromClipboard(data: DataTransfer | null) {
  if (!data) return []
  return Array.from(data.items || [])
    .filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file))
}

async function uploadImageFiles(files: File[]) {
  if (!files.length || uploadingImage.value) return

  uploadingImage.value = true
  emit('clearValidation')
  const markdownImages: string[] = []
  const failed: string[] = []

  try {
    for (const file of files) {
      const validation = validateImageFile(file)
      if (validation) {
        failed.push(`${file.name}: ${validation}`)
        continue
      }

      try {
        const optimized = await processImageFile(file)
        const url = await uploadImage(optimized.file)
        markdownImages.push(`![${imageAlt(file.name)}](${url})`)
      } catch (error) {
        failed.push(`${file.name}: ${error instanceof Error ? error.message : t('api.imageUploadFailed')}`)
      }
    }

    if (markdownImages.length) {
      insertMarkdownBlock(markdownImages.join('\n'))
      emit('imageInserted', markdownImages.length)
    }

    if (failed.length) {
      emit('imageError', failed.slice(0, 3).join(t('punctuation.semicolon')) + (failed.length > 3 ? t('publish.moreImageFailures', { count: failed.length - 3 }) : ''))
    } else if (!markdownImages.length) {
      emit('imageError', t('publish.noUploadableImages'))
    }
  } finally {
    uploadingImage.value = false
  }
}

async function handleImageInput(event: Event) {
  const input = event.target as HTMLInputElement
  const files = imageFilesFromList(input.files)
  input.value = ''
  await uploadImageFiles(files)
}

async function handlePaste(event: ClipboardEvent) {
  const files = imageFilesFromClipboard(event.clipboardData)
  if (files.length) {
    event.preventDefault()
    await uploadImageFiles(files)
    return
  }

  const markdown = markdownFromClipboard(event.clipboardData)
  if (!markdown) return
  event.preventDefault()
  insertMarkdownBlock(markdown)
}

async function handleDrop(event: DragEvent) {
  dragOver.value = false
  const files = imageFilesFromDataTransfer(event.dataTransfer)
  if (!files.length) return
  event.preventDefault()
  await uploadImageFiles(files)
}

function handleDragOver(event: DragEvent) {
  if (!hasImageDataTransfer(event.dataTransfer)) return
  event.preventDefault()
  dragOver.value = true
}

function submit() {
  if (composerBusy.value) return
  emit('submit')
}
</script>

<template>
  <Teleport v-if="open" to="body">
    <div class="pointer-events-none fixed inset-x-0 z-[90] px-3 sm:px-6" :style="{ bottom: `calc(${keyboardOffset}px + 1rem)` }">
      <div class="relative mx-auto flex w-full max-w-full justify-center">
        <Transition name="floating-reply">
          <div v-if="authenticated" class="gf-floating-surface pointer-events-auto relative flex max-h-[calc(100dvh-1rem)] w-[min(42rem,calc(100vw-1.5rem))] flex-col overflow-hidden p-3">
            <div class="mb-2 flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="text-sm font-semibold text-base-content">{{ composerTitle }}</div>
              </div>
              <button type="button" class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75 disabled:cursor-not-allowed disabled:opacity-60" :disabled="composerBusy" @click="closeComposer">
                <X class="h-4 w-4" />
              </button>
            </div>
            <div v-if="target && !editing" class="mb-2 flex min-w-0 items-center justify-between gap-3 rounded-md border border-primary/20 bg-info/10 px-3 py-2">
              <div class="min-w-0 text-sm font-medium text-base-content/75">
                {{ t('topic.replyTo', { user: `@${target.author.username}` }) }}
              </div>
              <button type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-base-100" :aria-label="t('common.cancel')" @click="emit('clearTarget')">
                <X class="h-3.5 w-3.5" />
              </button>
            </div>

            <div
              class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-line bg-base-100 transition focus-within:border-primary/50 focus-within:ring-4 focus-within:ring-primary/10"
              @focusin="keepToolbarOpen"
              @focusout="scheduleToolbarClose"
            >
              <div v-if="showToolbar" class="flex flex-wrap items-center gap-0.5 border-b border-line bg-base-100 px-1.5 py-1">
                <template v-if="!preview">
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.bold')" @mousedown.prevent @click="applyToolbarAction('bold')"><Bold class="h-4 w-4" /></button>
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.italic')" @mousedown.prevent @click="applyToolbarAction('italic')"><Italic class="h-4 w-4" /></button>
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.strike')" @mousedown.prevent @click="applyToolbarAction('strike')"><Strikethrough class="h-4 w-4" /></button>
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.inlineCode')" @mousedown.prevent @click="applyToolbarAction('inlineCode')"><Code class="h-4 w-4" /></button>
                  <div class="relative">
                    <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.link')" :aria-expanded="linkPickerOpen" @mousedown.prevent @click="openLinkPicker"><Link class="h-4 w-4" /></button>
                    <form v-if="linkPickerOpen" class="gf-menu-surface absolute bottom-full left-0 z-30 mb-1.5 flex w-72 max-w-[calc(100vw-5rem)] items-center gap-1.5 p-2 shadow-lg" @submit.prevent="applyLink">
                      <input v-model="linkUrl" type="text" inputmode="url" class="h-8 min-w-0 flex-1 rounded border border-line bg-base-100 px-2 text-sm outline-none focus:border-primary" :placeholder="t('publish.toolbar.linkUrl')" />
                      <button type="submit" class="gf-button gf-button-primary h-8 px-2.5" :disabled="!linkUrl.trim()">{{ t('publish.toolbar.applyLink') }}</button>
                    </form>
                  </div>
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.quote')" @mousedown.prevent @click="applyToolbarAction('quote')"><MessageSquareQuote class="h-4 w-4" /></button>
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.code')" @mousedown.prevent @click="applyToolbarAction('code')"><Code2 class="h-4 w-4" /></button>
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.bulletList')" @mousedown.prevent @click="applyToolbarAction('bulletList')"><List class="h-4 w-4" /></button>
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.toolbar.orderedList')" @mousedown.prevent @click="applyToolbarAction('orderedList')"><ListOrdered class="h-4 w-4" /></button>
                  <span class="mx-1 h-5 w-px bg-line" />
                  <button type="button" class="rounded p-1.5 text-base-content/55 transition hover:bg-base-200 hover:text-base-content" :title="t('publish.pastePlainText')" @mousedown.prevent @click="pastePlainText"><ClipboardPaste class="h-4 w-4" /></button>
                </template>
              </div>

              <div class="relative min-h-0 flex-1 overflow-y-auto">
                <VisualMarkdownEditor
                  v-if="!preview && editorMode === 'visual'"
                  ref="visualEditor"
                  v-model="content"
                  compact
                  :placeholder="composerPlaceholder"
                  @paste="handlePaste"
                  @drop="handleDrop"
                  @dragover="handleDragOver"
                  @dragleave="dragOver = false"
                />
                <textarea
                  v-else-if="!preview"
                  ref="markdownEditor"
                  v-model="content"
                  rows="4"
                  class="block min-h-24 w-full resize-none border-0 bg-transparent px-3 py-2.5 text-[15px] leading-relaxed outline-none placeholder:text-base-content/45"
                  :placeholder="composerPlaceholder"
                  @input="emit('clearValidation')"
                  @paste="handlePaste"
                  @drop="handleDrop"
                  @dragover="handleDragOver"
                  @dragleave="dragOver = false"
                />
                <div v-else class="gf-prose gf-prose-post min-h-24 max-w-none px-3 py-2.5">
                  <template v-if="content.trim()">
                    <div v-html="renderedPreview" />
                  </template>
                  <p v-else class="text-sm text-base-content/55">{{ t('publish.emptyPreview') }}</p>
                </div>
                <div
                  v-if="dragOver"
                  class="pointer-events-none absolute inset-3 grid place-items-center rounded-lg border-2 border-dashed border-primary/60 bg-info/10 text-sm font-semibold text-primary"
                >
                  {{ t('publish.dropToUpload') }}
                </div>
              </div>
            </div>
            <p v-if="errorMessage" class="mt-2 text-sm text-error">{{ errorMessage }}</p>
            <p v-if="successMessage" class="mt-2 text-sm text-success">{{ successMessage }}</p>
            <div v-if="captchaRequired" class="mt-2 flex flex-wrap items-center gap-2">
              <button
                type="button"
                class="relative h-9 w-24 shrink-0 overflow-hidden rounded-md border border-line"
                :disabled="captchaLoading"
                @click="emit('refreshCaptcha')"
              >
                <Loader2 v-if="captchaLoading || !captchaImg" class="mx-auto h-4 w-4 animate-spin text-base-content/55" />
                <img v-else :src="captchaImg" :alt="t('auth.captchaAlt')" class="h-full w-full object-cover" />
              </button>
              <input
                v-model="captchaCode"
                class="h-9 min-w-0 flex-1 rounded-md border border-line px-3 text-sm outline-none focus:border-primary"
                :placeholder="t('auth.captcha')"
                maxlength="8"
              />
            </div>
            <div class="mt-3 flex flex-wrap items-center gap-2">
              <label class="gf-icon-button h-9 w-9 cursor-pointer" :class="{ 'cursor-wait opacity-60': uploadingImage }" :title="t('publish.uploadImageTitle')">
                <Loader2 v-if="uploadingImage" class="h-4 w-4 animate-spin" />
                <Image v-else class="h-4 w-4" />
                <input type="file" accept="image/*" multiple class="hidden" :disabled="uploadingImage" @change="handleImageInput" />
              </label>
              <div class="inline-flex shrink-0 rounded-md border border-line p-0.5">
                <button type="button" class="rounded px-2 py-1 text-xs font-semibold whitespace-nowrap transition" :class="editorMode === 'visual' ? 'bg-neutral text-neutral-content' : 'text-base-content/55 hover:text-base-content'" @click="selectEditorMode('visual')">{{ t('publish.visualMode') }}</button>
                <button type="button" class="rounded px-2 py-1 text-xs font-semibold whitespace-nowrap transition" :class="editorMode === 'markdown' ? 'bg-neutral text-neutral-content' : 'text-base-content/55 hover:text-base-content'" @click="selectEditorMode('markdown')">{{ t('publish.markdownMode') }}</button>
              </div>
              <button type="button" class="inline-flex h-9 shrink-0 items-center gap-1 rounded-md border border-line px-2.5 text-xs font-semibold whitespace-nowrap transition" :class="preview ? 'bg-neutral text-neutral-content' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'" @click="togglePreview">
                <Eye class="h-3.5 w-3.5" />
                <span class="hidden sm:inline">{{ t('publish.preview') }}</span>
              </button>
              <div class="ml-auto flex items-center gap-2">
                <button v-if="target && !editing" type="button" class="gf-button gf-button-md gf-button-muted shrink-0" @click="emit('clearTarget')">
                  {{ t('common.cancel') }}
                </button>
                <button type="button" class="gf-button gf-button-md gf-button-primary shrink-0" :disabled="composerBusy" @click="submit">
                  <Loader2 v-if="composerBusy" class="h-4 w-4 animate-spin" />
                  <Check v-else-if="editing" class="h-4 w-4" />
                  <Send v-else class="h-4 w-4" />
                  {{ submitText }}
                </button>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </Teleport>
</template>

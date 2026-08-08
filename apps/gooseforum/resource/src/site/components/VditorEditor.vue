<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Loader2 } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import luteUrl from 'vditor/dist/js/lute/lute.min.js?url'
import antIconUrl from 'vditor/dist/js/icons/ant.js?url'
import enUrl from 'vditor/dist/js/i18n/en_US.js?url'
import jaUrl from 'vditor/dist/js/i18n/ja_JP.js?url'
import zhUrl from 'vditor/dist/js/i18n/zh_CN.js?url'
import { currentLocale } from '@/runtime/i18n'
import { loadRuntimeScript } from '@/runtime/runtime-script'
import { useSiteTheme } from '@/runtime/site-theme'
import { visualViewportScrollDelta, type VisualViewportTarget } from '@/runtime/visual-viewport'

const { t } = useI18n()
const { isDark } = useSiteTheme()

const props = defineProps<{
  modelValue: string
  placeholder: string
  uploading?: boolean
  minHeight?: number
}>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  input: []
  paste: [event: ClipboardEvent]
  drop: [event: DragEvent]
  dragover: [event: DragEvent]
  dragleave: [event: DragEvent]
  upload: [files: File[]]
  error: [error: Error]
}>()

const root = ref<HTMLElement | null>(null)
const loading = ref(true)
let editor: Vditor | null = null
let destroyed = false
let ready = false
const resolvedMinHeight = computed(() => props.minHeight ?? 320)

const TABLE_BUTTON_SELECTOR = '[data-type="table"]'
function degradeHeadingBeforeTableClick(event: MouseEvent) {
  if (!editor || !ready) return
  const target = event.target as HTMLElement | null
  if (!target?.closest?.(TABLE_BUTTON_SELECTOR)) return
  const wysiwyg = editor.vditor.wysiwyg?.element
  if (!wysiwyg) return

  // Mirror Vditor's own getEditorRange: prefer the live document selection
  // when it is inside the editor, otherwise fall back to its saved range.
  let range: Range | null = null
  const selection = window.getSelection()
  if (selection && selection.rangeCount > 0) {
    const candidate = selection.getRangeAt(0)
    if (wysiwyg.contains(candidate.startContainer)) range = candidate
  }
  range ??= editor.vditor.wysiwyg?.range ?? null
  if (!range) return

  const container = range.startContainer.nodeType === Node.ELEMENT_NODE
    ? (range.startContainer as HTMLElement)
    : range.startContainer.parentElement
  const heading = container?.closest?.('h1,h2,h3,h4,h5,h6')
  if (!heading || !wysiwyg.contains(heading)) return
  // An empty heading is replaced by the table natively; only the non-empty
  // case nests the table inside the heading.
  if (heading.textContent?.trim() === '') return

  // Vditor inserts the table at the caret; inside a heading that nests the
  // table element into the heading, and Lute then serializes the header row
  // as part of the heading text (e.g. "### | col1 | ..."), breaking the
  // published post. Degrade the heading to a plain paragraph and anchor the
  // table into a fresh empty block right after it, so the table lands as a
  // sibling block (Vditor replaces an empty block with the table HTML).
  const paragraph = document.createElement('p')
  paragraph.setAttribute('data-block', '0')
  paragraph.innerHTML = heading.innerHTML
  paragraph.querySelectorAll('[data-type="heading-marker"], wbr').forEach((node) => node.remove())
  heading.replaceWith(paragraph)

  const tableAnchor = document.createElement('p')
  tableAnchor.setAttribute('data-block', '0')
  tableAnchor.textContent = '\u200b'
  paragraph.after(tableAnchor)

  const nextRange = document.createRange()
  nextRange.setStart(tableAnchor, 0)
  nextRange.collapse(true)
  selection?.removeAllRanges()
  selection?.addRange(nextRange)
}

let viewportFrame: number | null = null
let viewportListenersBound = false

const VIEWPORT_TOP_INSET = 72
const VIEWPORT_BOTTOM_INSET = 24

const languageAssets = {
  en: { lang: 'en_US', url: enUrl },
  it: { lang: 'en_US', url: enUrl },
  ja: { lang: 'ja_JP', url: jaUrl },
  zh: { lang: 'zh_CN', url: zhUrl },
} as const

function syncEditorTheme() {
  if (!editor || !ready || destroyed) return
  editor.setTheme(isDark.value ? 'dark' : 'classic')
  // Content styling is owned by the site design tokens below, so no global
  // Vditor content-theme stylesheet is injected.
  if (editor.vditor.options.preview?.theme) {
    editor.vditor.options.preview.theme.current = isDark.value ? 'dark' : 'light'
  }
}

onMounted(async () => {
  if (!root.value) return
  const language = languageAssets[currentLocale()]

  try {
    await Promise.all([
      loadRuntimeScript(language.url, `vditorI18nScript${language.lang}`),
      loadRuntimeScript(antIconUrl, 'vditorIconScript'),
      loadRuntimeScript(luteUrl, 'vditorLuteScript'),
    ])
  } catch (error) {
    loading.value = false
    emit('error', error instanceof Error ? error : new Error(String(error)))
    return
  }
  if (destroyed || !root.value) return

  const mountTarget = root.value
  if (destroyed || !mountTarget) return

  let nextEditor: Vditor | null = null
  try {
    nextEditor = new Vditor(mountTarget, {
      _lutePath: luteUrl,
      cache: { enable: false },
      // Runtime assets are preloaded with the ids Vditor checks internally.
      cdn: '',
      counter: { enable: false },
      customWysiwygToolbar() {},
      height: 'auto',
      hint: { emojiPath: '', parse: false },
      i18n: window.VditorI18n,
      icon: 'ant',
      image: { isPreview: false },
      lang: language.lang,
      link: { isOpen: false },
      minHeight: resolvedMinHeight.value,
      mode: 'wysiwyg',
      outline: { enable: false, position: 'left' },
      placeholder: props.placeholder,
      theme: isDark.value ? 'dark' : 'classic',
      preview: {
        hljs: { enable: false, lineNumber: false },
        markdown: { codeBlockPreview: false, mathBlockPreview: false },
        mode: 'editor',
        render: { media: { enable: false } },
        theme: { current: isDark.value ? 'dark' : 'light', path: '' },
      },
      resize: { enable: false },
      toolbar: [
        'headings',
        'bold',
        'italic',
        'strike',
        'link',
        'upload',
        '|',
        'quote',
        'list',
        'ordered-list',
        'check',
        'line',
        'code',
        'inline-code',
        {
          name: 'math',
          icon: '<svg viewBox="0 0 16 16"><text x="8" y="12.5" text-anchor="middle" font-size="13" fill="currentColor">Σ</text></svg>',
          tip: t('publish.toolbar.math'),
          click() {
            if (!editor || !ready) return
            const selected = editor.getSelection()
            const math = selected ? `$${selected}$` : `$${t('publish.placeholder.math')}$`
            editor.focus()
            editor.insertMD(math)
            emit('update:modelValue', editor.getValue())
          },
        },
        'table',
        '|',
        'undo',
        'redo',
      ],
      toolbarConfig: { hide: false, pin: false },
      upload: {
        accept: 'image/*',
        multiple: true,
        handler(files) {
          if (props.uploading) return window.VditorI18n.uploading
          emit('upload', files)
          return null
        },
      },
      value: props.modelValue,
      after() {
        if (destroyed || !mountTarget.isConnected || editor !== nextEditor) {
          const staleEditor = nextEditor
          if (editor === staleEditor) editor = null
          nextEditor = null
          // Vditor still runs its icon setup after after() returns.
          queueMicrotask(() => staleEditor?.destroy())
          return
        }
        ready = true
        loading.value = false
        if (editor && props.modelValue !== editor.getValue()) editor.setValue(props.modelValue, true)
        syncEditorTheme()
        syncUploadControl()
      },
      input(value) {
        emit('update:modelValue', value)
      },
    })
    editor = nextEditor
  } catch (error) {
    loading.value = false
    emit('error', error instanceof Error ? error : new Error(String(error)))
  }
})

watch(() => props.modelValue, (value) => {
  if (!editor || !ready || value === editor.getValue()) return
  editor.setValue(value, true)
})

watch(() => props.placeholder, (placeholder) => {
  if (!editor) return
  editor.vditor.options.placeholder = placeholder
  editor.vditor.wysiwyg?.element.setAttribute('data-placeholder', placeholder)
})

watch(() => props.uploading, () => {
  queueMicrotask(syncUploadControl)
})

watch(isDark, syncEditorTheme)

onBeforeUnmount(() => {
  destroyed = true
  unbindVisualViewportListeners()
  const currentEditor = editor
  editor = null

  // Vditor does not cancel its pending Lute initialization in destroy(). Keep
  // after() intact so it can destroy the editor once initUI() has completed.
  if (!ready) return

  ready = false
  currentEditor?.destroy()
})

function focus() {
  if (ready) editor?.focus()
}

function insertMarkdown(markdown: string) {
  if (!editor || !ready) {
    emit('update:modelValue', props.modelValue ? `${props.modelValue}\n${markdown}` : markdown)
    return
  }
  editor.focus()
  editor.insertMD(markdown)
  emit('update:modelValue', editor.getValue())
}

function getValue() {
  return editor && ready ? editor.getValue() : props.modelValue
}

function syncValue() {
  const value = getValue()
  if (value !== props.modelValue) emit('update:modelValue', value)
  return value
}

function handleNativeInput() {
  emit('input')
  queueMicrotask(() => {
    syncValue()
    if (viewportListenersBound) scheduleCaretVisibilityUpdate()
  })
}

function activeCaretRect(): VisualViewportTarget | null {
  const editorRoot = root.value
  const activeElement = document.activeElement
  if (!editorRoot || !(activeElement instanceof HTMLElement) || !editorRoot.contains(activeElement)) return null

  const selection = window.getSelection()
  if (selection?.rangeCount && selection.focusNode && editorRoot.contains(selection.focusNode)) {
    const range = selection.getRangeAt(0).cloneRange()
    range.collapse(false)
    const rect = range.getBoundingClientRect()
    if (rect.top || rect.bottom || rect.height) return rect

    const focusElement = selection.focusNode instanceof Element
      ? selection.focusNode
      : selection.focusNode.parentElement
    if (focusElement) {
      const rect = focusElement.getBoundingClientRect()
      return { top: rect.top, bottom: Math.min(rect.bottom, rect.top + 32) }
    }
  }

  const rect = activeElement.getBoundingClientRect()
  return { top: rect.top, bottom: Math.min(rect.bottom, rect.top + 32) }
}

function keepCaretInVisualViewport() {
  viewportFrame = null
  const visualViewport = window.visualViewport
  const caretRect = activeCaretRect()
  if (!visualViewport || !caretRect) return

  const delta = visualViewportScrollDelta(
    caretRect,
    visualViewport,
    VIEWPORT_TOP_INSET,
    VIEWPORT_BOTTOM_INSET,
  )
  if (delta) window.scrollBy({ top: delta, behavior: 'auto' })
}

function scheduleCaretVisibilityUpdate() {
  if (viewportFrame !== null) return
  viewportFrame = window.requestAnimationFrame(keepCaretInVisualViewport)
}

function bindVisualViewportListeners() {
  const visualViewport = window.visualViewport
  if (!visualViewport || viewportListenersBound) return
  viewportListenersBound = true
  visualViewport.addEventListener('resize', scheduleCaretVisibilityUpdate)
  visualViewport.addEventListener('scroll', scheduleCaretVisibilityUpdate)
  scheduleCaretVisibilityUpdate()
}

function unbindVisualViewportListeners() {
  const visualViewport = window.visualViewport
  if (visualViewport && viewportListenersBound) {
    visualViewport.removeEventListener('resize', scheduleCaretVisibilityUpdate)
    visualViewport.removeEventListener('scroll', scheduleCaretVisibilityUpdate)
  }
  viewportListenersBound = false
  if (viewportFrame === null) return
  window.cancelAnimationFrame(viewportFrame)
  viewportFrame = null
}

function handleEditorFocusIn(event: FocusEvent) {
  const target = event.target
  if (!(target instanceof Element) || !target.closest('.vditor-wysiwyg')) return
  bindVisualViewportListeners()
}

function handleEditorFocusOut(event: FocusEvent) {
  const nextTarget = event.relatedTarget
  if (nextTarget instanceof Element && nextTarget.closest('.vditor-wysiwyg')) return
  unbindVisualViewportListeners()
}

function syncUploadControl() {
  // Vditor renders the upload entry as a div (not a button) inside
  // .vditor-toolbar__item, and its own click handler short-circuits when the
  // element carries the vditor-menu--disabled class.
  const uploadItem = root.value?.querySelector<HTMLElement>('.vditor-toolbar__item [data-type="upload"]')
  if (!uploadItem) return
  const disabled = Boolean(props.uploading)
  const input = uploadItem.querySelector<HTMLInputElement>('input[type="file"]')
  if (input) input.disabled = disabled
  uploadItem.classList.toggle('vditor-menu--disabled', disabled)
}

function forwardPaste(event: ClipboardEvent) {
  emit('paste', event)
  if (event.defaultPrevented) event.stopImmediatePropagation()
}

function forwardDrop(event: DragEvent) {
  emit('drop', event)
  if (event.defaultPrevented) event.stopImmediatePropagation()
}

function forwardDragOver(event: DragEvent) {
  emit('dragover', event)
  if (event.defaultPrevented) event.stopImmediatePropagation()
}

defineExpose({ focus, getValue, insertMarkdown, syncValue })
</script>

<template>
  <div class="vditor-editor-wrap" :class="{ 'is-uploading': uploading }" :style="{ minHeight: `${resolvedMinHeight}px` }">
    <div
      ref="root"
      class="vditor-editor"
      @click.capture="degradeHeadingBeforeTableClick"
      @focusin="handleEditorFocusIn"
      @focusout="handleEditorFocusOut"
      @input="handleNativeInput"
      @paste.capture="forwardPaste"
      @drop.capture="forwardDrop"
      @dragover.capture="forwardDragOver"
      @dragleave.capture="emit('dragleave', $event)"
    />
    <div v-if="loading" class="vditor-loading" role="status" aria-live="polite">
      <Loader2 class="h-4 w-4 animate-spin" />
      <span>{{ t('common.loadingShort') }}</span>
    </div>
  </div>
</template>

<style scoped>
.vditor-editor-wrap {
  position: relative;
  min-height: 320px;
}

.vditor-loading {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: 1px solid var(--gf-color-line);
  border-radius: 6px;
  background: var(--gf-color-base-100);
  color: var(--gf-color-base-content);
  font-size: 14px;
}

.vditor-editor :deep(.vditor) {
  background: transparent;
  border: 1px solid var(--gf-color-line);
  border-radius: 6px;
  color: var(--gf-color-base-content);
  overflow: hidden;
}

.vditor-editor :deep(.vditor-toolbar) {
  background: var(--gf-color-base-200);
  border-color: var(--gf-color-line);
  padding: 6px 8px;
}

.vditor-editor :deep(.vditor-toolbar__item button) {
  color: color-mix(in oklch, var(--gf-color-base-content) 62%, transparent);
}

.vditor-editor :deep(.vditor-toolbar__item button:hover),
.vditor-editor :deep(.vditor-toolbar__item button:focus) {
  background: var(--gf-color-base-300);
  color: var(--gf-color-base-content);
}

.vditor-editor-wrap.is-uploading :deep(.vditor-toolbar__item [data-type='upload']) {
  cursor: wait;
  opacity: 0.55;
  pointer-events: none;
}

.vditor-editor :deep(.vditor-wysiwyg) {
  background: transparent;
  color: var(--gf-color-base-content);
  font-family: inherit;
  font-size: 15px;
  line-height: 1.75;
  padding: 16px;
}

.vditor-editor :deep(.vditor-reset) {
  color: var(--gf-color-base-content);
}

.vditor-editor :deep(.vditor-reset ol) {
  list-style-type: decimal;
}

.vditor-editor :deep(.vditor-reset ol ol) {
  list-style-type: lower-alpha;
}

.vditor-editor :deep(.vditor-reset ol ol ol) {
  list-style-type: lower-roman;
}

.vditor-editor :deep(.vditor-reset a) {
  color: var(--gf-color-primary);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.vditor-editor :deep(.vditor-wysiwyg pre.vditor-reset:empty::before) {
  color: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
}

.vditor-editor :deep(.vditor-reset blockquote),
.vditor-editor :deep(.vditor-reset table),
.vditor-editor :deep(.vditor-reset td),
.vditor-editor :deep(.vditor-reset th) {
  border-color: var(--gf-color-line);
}

.vditor-editor :deep(.vditor-reset blockquote) {
  color: color-mix(in oklch, var(--gf-color-base-content) 70%, transparent);
}

.vditor-editor :deep(.vditor-reset h1),
.vditor-editor :deep(.vditor-reset h2),
.vditor-editor :deep(.vditor-reset hr) {
  border-color: var(--gf-color-line);
}

.vditor-editor :deep(.vditor-reset hr) {
  background: var(--gf-color-line);
}

.vditor-editor :deep(.vditor-reset table tr),
.vditor-editor :deep(.vditor-reset table tbody tr:nth-child(2n)) {
  background: var(--gf-color-base-100);
}

.vditor-editor :deep(.vditor-reset table tbody tr:nth-child(2n)) {
  background: var(--gf-color-base-200);
}

.vditor-editor :deep(.vditor-reset code:not(.hljs):not(.highlight-chroma)) {
  background: var(--gf-color-base-200);
  color: var(--gf-color-base-content);
}

.vditor-editor :deep(.vditor-reset pre > code) {
  background-color: var(--gf-color-base-200);
  background-image: none;
  color: var(--gf-color-base-content);
}

.vditor-editor :deep(.vditor-reset ::selection) {
  background: color-mix(in oklch, var(--gf-color-primary) 35%, transparent);
  color: var(--gf-color-base-content);
}
</style>

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
  /** 左侧大纲：发布页开启；浮动回复面板保持关闭 */
  outline?: boolean
  /** 右上角字数统计，默认开启（对齐官方 demo） */
  counter?: boolean
  /**
   * 预览/编辑内容最大宽度（px）。官方默认约 800；
   * 发布页可调高，减少宽屏 setPadding 两侧留白。
   */
  previewMaxWidth?: number
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
// 官方 IPreview.maxWidth 默认约 768~800；未传时保持库默认，不主动覆盖。
const resolvedPreviewMaxWidth = computed(() => props.previewMaxWidth)

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

// 官方 demo 风格：工具栏只显示图标；hover tooltip 保留「功能 + 快捷键」。
// vditor 原生 aria-label 形如 "粗体 <Ctrl+B>" / mac 下 "<⌘B>"，去掉尖括号更易读。
const HOTKEY_PATTERN = /\s*<([^<>]+)>\s*$/
const OUTLINE_DESKTOP_QUERY = '(min-width: 768px)'

function polishToolbarTooltips() {
  if (!root.value) return
  root.value.querySelectorAll<HTMLElement>('.vditor-toolbar__item .vditor-tooltipped').forEach((item) => {
    const original = item.getAttribute('aria-label') ?? ''
    if (!original) return
    const hotkeyMatch = original.match(HOTKEY_PATTERN)
    if (!hotkeyMatch) return
    const label = original.slice(0, hotkeyMatch.index).trim()
    const hotkey = hotkeyMatch[1]
    if (!label) return
    item.setAttribute('aria-label', `${label} ${hotkey}`)
  })
}

function outlineShouldShow(): boolean {
  return Boolean(props.outline) && window.matchMedia(OUTLINE_DESKTOP_QUERY).matches
}

function syncOutlineVisibility() {
  if (!editor || !ready || destroyed || !props.outline) return
  const outline = editor.vditor.outline
  if (!outline) return
  // 第三个参数 focus=false，避免切换大纲时抢走编辑区焦点
  outline.toggle(editor.vditor, outlineShouldShow(), false)
}

function buildToolbar(): Array<string | IMenuItem> {
  const mathItem: IMenuItem = {
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
  }
  const items: Array<string | IMenuItem> = []
  if (props.outline) items.push('outline')
  items.push(
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
    mathItem,
    'table',
    '|',
    'undo',
    'redo',
  )
  return items
}

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
      counter: { enable: props.counter !== false },
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
      // 初始按桌面宽度决定；窄屏由 syncOutlineVisibility 关闭，避免 250px 大纲挤占回复区
      outline: { enable: outlineShouldShow(), position: 'left' },
      placeholder: props.placeholder,
      theme: isDark.value ? 'dark' : 'classic',
      preview: {
        hljs: { enable: false, lineNumber: false },
        markdown: { codeBlockPreview: false, mathBlockPreview: false },
        mode: 'editor',
        // 仅在宿主显式传入时覆盖；否则走 Vditor 官方默认 maxWidth
        ...(resolvedPreviewMaxWidth.value != null
          ? { maxWidth: resolvedPreviewMaxWidth.value }
          : {}),
        render: { media: { enable: false } },
        theme: { current: isDark.value ? 'dark' : 'light', path: '' },
      },
      resize: { enable: false },
      toolbar: buildToolbar(),
      toolbarConfig: { hide: false, pin: true },
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
        polishToolbarTooltips()
        syncOutlineVisibility()
        window.addEventListener('resize', syncOutlineVisibility)
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
  window.removeEventListener('resize', syncOutlineVisibility)
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
  border-radius: var(--gf-radius-box);
  background: var(--gf-color-base-100);
  color: var(--gf-color-base-content);
  font-size: 14px;
}

/*
 * vditor 主题 token 化：把 vditor 内置变量全部映射到全站设计 token，
 * 深浅色共用一套映射（.vditor--dark 由 vditor 切换，这里用更高特异性覆盖）。
 * 注意：vditor 直接把挂载目标元素自身变成 .vditor（而非创建子节点），
 * 所以选择器必须挂在 .vditor-editor-wrap 上才能命中。
 * overflow: visible 让焦点环 box-shadow 不被裁剪，同时 pin 工具栏可正常 sticky。
 */
.vditor-editor-wrap :deep(.vditor),
.vditor-editor-wrap :deep(.vditor.vditor--dark) {
  --border-color: var(--gf-color-line);
  --second-color: color-mix(in oklch, var(--gf-color-base-content) 65%, transparent);
  --panel-background-color: var(--gf-color-base-100);
  --panel-shadow: 0 18px 40px -24px rgb(15 23 42 / calc(var(--gf-depth) * 0.45));
  --toolbar-background-color: var(--gf-color-base-200);
  --toolbar-icon-color: var(--gf-color-icon-muted);
  --toolbar-icon-hover-color: var(--gf-color-primary);
  /* 官方默认 35px 工具栏高度 */
  --toolbar-height: 35px;
  --toolbar-divider-margin-top: 8px;
  --textarea-background-color: transparent;
  --textarea-text-color: var(--gf-color-base-content);
  --count-background-color: color-mix(in oklch, var(--gf-color-base-content) 6%, transparent);
  --heading-border-color: var(--gf-color-line);
  --blockquote-color: color-mix(in oklch, var(--gf-color-base-content) 70%, transparent);

  background: transparent;
  border: 1px solid var(--gf-color-line);
  border-radius: var(--gf-radius-box);
  color: var(--gf-color-base-content);
  overflow: visible;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

/* 焦点态：与全站输入框语言一致（focus:border-primary + ring 4px primary/20） */
.vditor-editor-wrap:focus-within :deep(.vditor) {
  border-color: var(--gf-color-primary);
  box-shadow: 0 0 0 4px color-mix(in oklch, var(--gf-color-primary) 20%, transparent);
}

/* ===== 工具栏（对齐官方 demo：纯图标单行） ===== */
.vditor-editor :deep(.vditor-toolbar) {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0;
  background: var(--gf-color-base-200);
  border-bottom: 1px solid var(--gf-color-line);
  border-radius: calc(var(--gf-radius-box) - 1px) calc(var(--gf-radius-box) - 1px) 0 0;
  /* 官方默认 padding: 0 5px */
  padding: 0 5px;
  line-height: 1;
}

.vditor-editor :deep(.vditor-toolbar__item) {
  float: none;
  position: relative;
  display: inline-flex;
  align-items: center;
}

/* 纯图标按钮：接近官方 25×35 尺寸，略放大触控与 hover 反馈 */
.vditor-editor :deep(.vditor-toolbar__item .vditor-tooltipped) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  /* 官方默认：width 25px、padding 10px 5px、box-sizing border-box → 外框 25×35 */
  width: 25px;
  height: var(--toolbar-height);
  margin: 0;
  padding: 10px 5px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--gf-color-icon-muted);
  font-size: 0;
  box-sizing: border-box;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.vditor-editor :deep(.vditor-toolbar__item .vditor-tooltipped:hover),
.vditor-editor :deep(.vditor-toolbar__item .vditor-tooltipped:focus) {
  background: var(--gf-color-base-300);
  color: var(--gf-color-base-content);
  cursor: pointer;
}

.vditor-editor :deep(.vditor-toolbar__item .vditor-tooltipped:active),
.vditor-editor :deep(.vditor-toolbar__item .vditor-tooltipped.vditor-menu--current) {
  background: color-mix(in oklch, var(--gf-color-primary) 15%, transparent);
  color: var(--gf-color-primary);
}

.vditor-editor :deep(.vditor-toolbar__item svg) {
  width: 15px;
  height: 15px;
}

/* hover 悬浮提示：官方 demo 同款「功能 + 快捷键」 */
.vditor-editor :deep(.vditor-toolbar__item .vditor-tooltipped::after) {
  font-weight: 500;
  white-space: nowrap;
}

.vditor-editor :deep(.vditor-toolbar__item input[type='file']) {
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
}

.vditor-editor :deep(.vditor-toolbar__divider) {
  float: none;
  height: calc(var(--toolbar-height) - (var(--toolbar-divider-margin-top) * 2));
  /* 官方默认水平 8px */
  margin: var(--toolbar-divider-margin-top) 8px;
  border-left: 1px solid color-mix(in oklch, var(--gf-color-line) 80%, transparent);
}

.vditor-editor :deep(.vditor-toolbar--pin) {
  /* 宿主页面通过 --vditor-pin-top 声明 sticky 偏移（如发布页避开 4rem 页面头部） */
  top: var(--vditor-pin-top, 0);
  z-index: 5;
}

.vditor-editor :deep(.vditor-tooltipped::after) {
  border-radius: 6px;
}

/* 字数统计：flex 工具栏下用 margin-left:auto 推到右侧（官方 float:right 在 flex 中失效） */
.vditor-editor :deep(.vditor-counter) {
  float: none;
  margin: 0 0 0 auto;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--count-background-color);
  color: var(--gf-color-icon-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  line-height: 1.4;
  user-select: none;
}

.vditor-editor :deep(.vditor-counter--error) {
  color: var(--gf-color-error);
  background: color-mix(in oklch, var(--gf-color-error) 12%, transparent);
}

/* 左侧大纲：token 化，贴合官方 250px 布局 */
.vditor-editor :deep(.vditor-outline) {
  width: 250px;
  border-right: 1px solid var(--gf-color-line);
  background: var(--gf-color-base-100);
}

.vditor-editor :deep(.vditor-outline__title) {
  border-bottom: 1px dashed var(--gf-color-line);
  color: var(--gf-color-icon-muted);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.04em;
}

.vditor-editor :deep(.vditor-outline li > span) {
  border-radius: 4px;
  color: var(--gf-color-base-content);
  transition: color 0.15s ease, background-color 0.15s ease;
}

.vditor-editor :deep(.vditor-outline li > span:hover) {
  background: var(--gf-color-base-200);
  color: var(--gf-color-primary);
}

.vditor-editor-wrap.is-uploading :deep(.vditor-toolbar__item [data-type='upload']) {
  cursor: wait;
  opacity: 0.55;
  pointer-events: none;
}

/* ===== 编辑区（桌面端：阅读宽度居中） ===== */
.vditor-editor :deep(.vditor-wysiwyg) {
  background: transparent;
}

.vditor-editor :deep(.vditor-wysiwyg pre.vditor-reset) {
  /*
   * 官版节奏：font 16px / line-height 1.5 / 基础 padding 10px。
   * 水平 padding 由 vditor setPadding 按 preview.maxWidth(800) 计算
   *（桌面最小 35px），不要再用 max-width:760 二次收窄，否则两侧更空。
   */
  max-width: none;
  margin: 0;
  background: transparent;
  color: var(--gf-color-base-content);
  font-family: inherit;
  font-size: 16px;
  line-height: 1.5;
  padding: 10px;
  word-wrap: break-word;
  word-break: break-word;
  font-variant-ligatures: no-common-ligatures;
}

.vditor-editor :deep(.vditor-reset) {
  color: var(--gf-color-base-content);
}

/*
 * 显式锁定官版块级节奏，避免 Tailwind preflight / 全站 prose 把
 * 标题字号、段落间距冲掉或放大。数值与 vditor/dist/index.css 一致。
 */
.vditor-editor :deep(.vditor-reset p) {
  margin-top: 0;
  margin-bottom: 16px;
}

.vditor-editor :deep(.vditor-reset h1),
.vditor-editor :deep(.vditor-reset h2),
.vditor-editor :deep(.vditor-reset h3),
.vditor-editor :deep(.vditor-reset h4),
.vditor-editor :deep(.vditor-reset h5),
.vditor-editor :deep(.vditor-reset h6) {
  margin-top: 24px;
  margin-bottom: 16px;
  font-weight: 600;
  line-height: 1.25;
  color: var(--gf-color-base-content);
  border-color: var(--gf-color-line);
}

.vditor-editor :deep(.vditor-reset h1) {
  font-size: 1.75em;
}

.vditor-editor :deep(.vditor-reset h2) {
  font-size: 1.55em;
}

.vditor-editor :deep(.vditor-reset h3) {
  font-size: 1.38em;
}

.vditor-editor :deep(.vditor-reset h4) {
  font-size: 1.25em;
}

.vditor-editor :deep(.vditor-reset h5) {
  font-size: 1.13em;
}

.vditor-editor :deep(.vditor-reset h6) {
  font-size: 1em;
}

.vditor-editor :deep(.vditor-reset ul),
.vditor-editor :deep(.vditor-reset ol) {
  margin-top: 0;
  margin-bottom: 16px;
  padding-left: 2em;
}

.vditor-editor :deep(.vditor-reset li + li) {
  margin-top: 0.25em;
}

.vditor-editor :deep(.vditor-reset blockquote) {
  margin: 0 0 16px;
  padding: 0 1em;
}

.vditor-editor :deep(.vditor-reset hr) {
  height: 2px;
  margin: 24px 0;
  padding: 0;
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

/* 行内代码沿用全站 .gf-prose 语言（--tw-prose-code: error 色），所见即所得 */
.vditor-editor :deep(.vditor-reset code:not(.hljs):not(.highlight-chroma)) {
  background: var(--gf-color-base-200);
  color: var(--gf-color-error);
}

.vditor-editor :deep(.vditor-reset pre > code) {
  background-color: var(--gf-color-base-200);
  background-image: none;
  color: var(--gf-color-base-content);
}

.vditor-editor :deep(.vditor-reset blockquote),
.vditor-editor :deep(.vditor-reset table),
.vditor-editor :deep(.vditor-reset td),
.vditor-editor :deep(.vditor-reset th) {
  border-color: var(--gf-color-line);
}

.vditor-editor :deep(.vditor-reset blockquote) {
  border-left-color: var(--gf-color-line);
  color: color-mix(in oklch, var(--gf-color-base-content) 70%, transparent);
}

.vditor-editor :deep(.vditor-reset hr) {
  background: var(--gf-color-line);
  border-color: var(--gf-color-line);
}

.vditor-editor :deep(.vditor-reset table tr),
.vditor-editor :deep(.vditor-reset table tbody tr:nth-child(2n)) {
  background: var(--gf-color-base-100);
}

.vditor-editor :deep(.vditor-reset table tbody tr:nth-child(2n)) {
  background: var(--gf-color-base-200);
}

.vditor-editor :deep(.vditor-reset ::selection) {
  background: color-mix(in oklch, var(--gf-color-primary) 35%, transparent);
  color: var(--gf-color-base-content);
}

/* 平板及以下：大纲占宽过大，强制隐藏（JS 同步会关掉并清 toolbar padding） */
@media (max-width: 767px) {
  .vditor-editor :deep(.vditor-outline) {
    display: none !important;
  }
}

/* ===== 移动端（≤520px）：工具栏单行横向滚动 + 大触控目标 ===== */
@media (max-width: 520px) {
  .vditor-editor-wrap :deep(.vditor) {
    --toolbar-height: 44px;
  }

  .vditor-editor :deep(.vditor-toolbar) {
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
    padding: 0 6px;
  }

  .vditor-editor :deep(.vditor-toolbar::-webkit-scrollbar) {
    display: none;
  }

  .vditor-editor :deep(.vditor-toolbar__item) {
    flex-shrink: 0;
  }

  .vditor-editor :deep(.vditor-toolbar__item .vditor-tooltipped) {
    width: 40px;
    height: 44px;
    padding: 0;
  }

  .vditor-editor :deep(.vditor-counter) {
    flex-shrink: 0;
    margin-left: 8px;
  }

  .vditor-editor :deep(.vditor-wysiwyg pre.vditor-reset) {
    max-width: none;
    padding: 10px 12px 16px;
    /* 保持 16px 字号，避免 iOS 聚焦编辑时页面自动放大 */
    font-size: 16px;
    line-height: 1.5;
  }
}
</style>

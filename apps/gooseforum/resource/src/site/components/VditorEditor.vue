<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
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

const { t } = useI18n()
const { isDark } = useSiteTheme()

const props = defineProps<{
  modelValue: string
  placeholder: string
  uploading?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
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
      minHeight: 320,
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
  queueMicrotask(syncValue)
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
  <div class="vditor-editor-wrap" :class="{ 'is-uploading': uploading }">
    <div
      ref="root"
      class="vditor-editor"
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

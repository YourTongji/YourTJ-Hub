<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import luteUrl from 'vditor/dist/js/lute/lute.min.js?url'
import antIconUrl from 'vditor/dist/js/icons/ant.js?url'
import enUrl from 'vditor/dist/js/i18n/en_US.js?url'
import jaUrl from 'vditor/dist/js/i18n/ja_JP.js?url'
import zhUrl from 'vditor/dist/js/i18n/zh_CN.js?url'
import { currentLocale } from '@/runtime/i18n'

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
let editor: Vditor | null = null
let destroyed = false
let ready = false

const languageAssets = {
  en: { lang: 'en_US', url: enUrl },
  it: { lang: 'en_US', url: enUrl },
  ja: { lang: 'ja_JP', url: jaUrl },
  zh: { lang: 'zh_CN', url: zhUrl },
} as const

function loadRuntimeScript(url: string, id: string) {
  const existing = document.getElementById(id) as HTMLScriptElement | null
  if (existing?.dataset.loaded === 'true') return Promise.resolve()

  return new Promise<void>((resolve, reject) => {
    const script = existing || document.createElement('script')
    const cleanup = () => {
      script.removeEventListener('load', handleLoad)
      script.removeEventListener('error', handleError)
    }
    const handleLoad = () => {
      script.dataset.loaded = 'true'
      cleanup()
      resolve()
    }
    const handleError = () => {
      cleanup()
      script.remove()
      reject(new Error(`Failed to load Vditor runtime asset: ${url}`))
    }

    script.addEventListener('load', handleLoad)
    script.addEventListener('error', handleError)
    if (!existing) {
      script.id = id
      script.src = url
      script.async = true
      document.head.appendChild(script)
    }
  })
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
    emit('error', error instanceof Error ? error : new Error(String(error)))
    return
  }
  if (destroyed || !root.value) return

  editor = new Vditor(root.value, {
    _lutePath: luteUrl,
    cache: { enable: false },
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
    preview: {
      hljs: { enable: false, lineNumber: false },
      markdown: { codeBlockPreview: false, mathBlockPreview: false },
      mode: 'editor',
      render: { media: { enable: false } },
      theme: { current: 'light', path: '' },
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
      ready = true
      if (editor && props.modelValue !== editor.getValue()) editor.setValue(props.modelValue, true)
      syncUploadControl()
    },
    input(value) {
      emit('update:modelValue', value)
    },
  })
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

onBeforeUnmount(() => {
  destroyed = true
  ready = false
  editor?.destroy()
  editor = null
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
  const input = root.value?.querySelector<HTMLInputElement>('.vditor-toolbar input[type="file"]')
  if (!input) return
  const disabled = Boolean(props.uploading)
  input.disabled = disabled
  const button = input.closest<HTMLButtonElement>('button')
  if (button) button.disabled = disabled
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
  <div
    ref="root"
    class="vditor-editor"
    :class="{ 'is-uploading': uploading }"
    @input="handleNativeInput"
    @paste.capture="forwardPaste"
    @drop.capture="forwardDrop"
    @dragover.capture="forwardDragOver"
    @dragleave.capture="emit('dragleave', $event)"
  />
</template>

<style scoped>
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

.vditor-editor.is-uploading :deep(.vditor-toolbar__item button:has(input[type='file'])) {
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

.vditor-editor :deep(.vditor-wysiwyg pre.vditor-reset:empty::before) {
  color: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
}

.vditor-editor :deep(.vditor-reset blockquote),
.vditor-editor :deep(.vditor-reset table),
.vditor-editor :deep(.vditor-reset td),
.vditor-editor :deep(.vditor-reset th) {
  border-color: var(--gf-color-line);
}

.vditor-editor :deep(.vditor-reset code:not(.hljs):not(.highlight-chroma)) {
  background: var(--gf-color-base-200);
  color: var(--gf-color-base-content);
}
</style>

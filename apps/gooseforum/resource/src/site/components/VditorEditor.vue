<script lang="ts">
// Module-scoped so that fast route switches share one in-flight promise per
// runtime asset instead of reloading or clobbering each other's <script> tags.
const pendingScripts = new Map<string, Promise<void>>()
const SCRIPT_TIMEOUT_MS = 20000
</script>

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
import { useSiteTheme } from '@/runtime/site-theme'

const { t } = useI18n()
const { theme: siteTheme } = useSiteTheme()

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

const vditorTheme = computed(() => (siteTheme.value === 'gf-dark' ? 'dark' : 'classic'))

function loadRuntimeScript(url: string, id: string) {
  const existing = document.getElementById(id) as HTMLScriptElement | null
  if (existing?.dataset.loaded === 'true') return Promise.resolve()
  const pending = pendingScripts.get(id)
  if (pending) return pending

  const promise = new Promise<void>((resolve, reject) => {
    let settled = false
    let script: HTMLScriptElement | null = null
    let timeoutId = 0

    const handleLoad = () => {
      if (!script) return
      script.dataset.loaded = 'true'
      if (settled) return
      settled = true
      window.clearTimeout(timeoutId)
      script.removeEventListener('load', handleLoad)
      script.removeEventListener('error', handleError)
      pendingScripts.delete(id)
      resolve()
    }
    const handleError = () => {
      script?.remove()
      if (settled) return
      settled = true
      window.clearTimeout(timeoutId)
      pendingScripts.delete(id)
      reject(new Error(`Failed to load Vditor runtime asset: ${url}`))
    }
    const handleTimeout = () => {
      script?.remove()
      if (settled) return
      settled = true
      pendingScripts.delete(id)
      reject(new Error(`Timed out loading Vditor runtime asset: ${url}`))
    }

    // A leftover node (e.g. from a failed previous attempt) is never going to
    // fire load/error again, so rebuild it instead of waiting forever.
    existing?.remove()
    script = document.createElement('script')
    script.id = id
    script.src = url
    script.async = true
    script.addEventListener('load', handleLoad)
    script.addEventListener('error', handleError)
    timeoutId = window.setTimeout(handleTimeout, SCRIPT_TIMEOUT_MS)
    document.head.appendChild(script)
  })
  pendingScripts.set(id, promise)
  return promise
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

  try {
    editor = new Vditor(root.value, {
      _lutePath: luteUrl,
      cache: { enable: false },
      // Vditor lazily loads its remaining runtime assets (icons, i18n, lute)
      // by appending <script> tags whose ids must match the exact names above.
      // The assets are preloaded with those ids, so an empty cdn never hits
      // the network.
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
      theme: vditorTheme.value,
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
        if (destroyed) {
          editor?.destroy()
          editor = null
          return
        }
        ready = true
        loading.value = false
        if (editor && props.modelValue !== editor.getValue()) editor.setValue(props.modelValue, true)
        syncUploadControl()
      },
      input(value) {
        emit('update:modelValue', value)
      },
    })
    if (destroyed) {
      editor?.destroy()
      editor = null
    }
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

watch(vditorTheme, (theme) => {
  if (!editor) return
  editor.setTheme(theme)
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

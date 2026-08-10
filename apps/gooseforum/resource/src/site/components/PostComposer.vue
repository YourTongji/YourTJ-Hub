<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Check, Loader2, Lock, Send, X } from '@lucide/vue'
import { uploadImage } from '@/runtime/api'
import { processImageFile, validateImageFile } from '@/runtime/image'
import VditorOfficial from '@/site/components/VditorOfficial.vue'
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

const editor = ref<InstanceType<typeof VditorOfficial> | null>(null)
const editorReady = ref(false)
const editorInitFailed = ref(false)
const uploadingImage = ref(false)

// Vditor 异步就绪（after()）前在编辑区显示加载占位；初始化失败时结束 loading，避免转圈不止
watch(
  () => [editor.value?.editorReady, editor.value?.editorFailed] as const,
  ([ready, failed]) => {
    editorReady.value = !!ready
    editorInitFailed.value = !!failed
  },
  { immediate: true },
)
const composerBusy = computed(() => props.submitting || uploadingImage.value)

/** 桌面端浮动面板高度：默认更高，顶部手柄可拖拽调整 */
const MOBILE_VIEWPORT_QUERY = '(max-width: 520px)'
const DESKTOP_COMPOSER_HEIGHT = 480
const MIN_COMPOSER_HEIGHT = 240
const MAX_COMPOSER_HEIGHT = 720
const isMobileComposer = () => window.matchMedia(MOBILE_VIEWPORT_QUERY).matches
const composerHeight = ref(DESKTOP_COMPOSER_HEIGHT)
const editorArea = ref<HTMLElement | null>(null)
const draggingHeight = ref(false)
const dragStartY = ref(0)
const dragStartHeight = ref(0)

function startHeightDrag(event: PointerEvent) {
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) return
  draggingHeight.value = true
  dragStartY.value = event.clientY
  dragStartHeight.value = composerHeight.value
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function moveHeightDrag(event: PointerEvent) {
  if (!draggingHeight.value) return
  // 手柄在顶部：向上拖（clientY 减小）→ 变高；向下拖 → 变矮
  const delta = dragStartY.value - event.clientY
  composerHeight.value = Math.min(MAX_COMPOSER_HEIGHT, Math.max(MIN_COMPOSER_HEIGHT, dragStartHeight.value + delta))
  // 同步 Vditor 高度，让编辑器填满新的编辑区
  void nextTick(() => {
    if (editorArea.value && editor.value) {
      editor.value.setHeight(editorArea.value.clientHeight)
    }
  })
}

function endHeightDrag(event: PointerEvent) {
  if (!draggingHeight.value) return
  draggingHeight.value = false
  const handle = event.currentTarget as HTMLElement
  if (handle.hasPointerCapture?.(event.pointerId)) handle.releasePointerCapture(event.pointerId)
  void nextTick(() => {
    if (editorArea.value && editor.value) {
      editor.value.setHeight(editorArea.value.clientHeight)
    }
  })
}
const editing = computed(() => props.mode === 'edit')
const composerTitle = computed(() => (editing.value ? t('topic.editOwnReply') : t('topic.joinDiscussion')))
const composerPlaceholder = computed(() => (editing.value ? t('topic.editReplyPlaceholder') : t('topic.replyPlaceholder')))
const submitText = computed(() => {
  if (uploadingImage.value) return t('publish.processingImage')
  if (props.submitting) return editing.value ? t('common.saving') : t('topic.publishing')
  return editing.value ? t('common.save') : t('topic.publishReply')
})

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    // 面板关闭会卸载内层 Vditor；再次打开时清失败态，重新走 loading → ready/error
    editorInitFailed.value = false
    await nextTick()
    window.requestAnimationFrame(() => {
      if (!editorInitFailed.value) editor.value?.focus()
    })
  },
  { immediate: true },
)

function closeComposer() {
  if (composerBusy.value) return
  emit('update:open', false)
}

function handleEditorError(editorError: Error) {
  // 兜底：即使子组件尚未 expose editorFailed，也立刻结束 loading 遮罩
  editorInitFailed.value = true
  emit('imageError', editorError.message || t('common.loadFailed'))
}

function insertMarkdownBlock(text: string) {
  editor.value?.insertMarkdown(text)
}

function imageAlt(filename: string) {
  return filename.replace(/\.[^.]+$/, '').replace(/[[\]\n\r]/g, ' ').trim() || 'image'
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

function submit() {
  if (composerBusy.value) return
  emit('submit')
}
</script>

<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed inset-x-0 z-[90] px-3 sm:px-6" :style="{ bottom: `calc(${keyboardOffset}px + 1rem)` }">
      <div class="relative mx-auto flex w-full max-w-full justify-center">
        <!-- appear：覆盖首次打开时组件刚挂载、Transition 与其子元素同帧出现的场景 -->
        <Transition name="composer-rise" appear>
          <div
            v-if="open"
            class="gf-floating-surface pointer-events-auto relative flex max-h-[calc(100dvh-1rem)] w-[min(42rem,calc(100vw-1.5rem))] flex-col overflow-hidden p-3"
            :style="{
              height: isMobileComposer() ? undefined : `${composerHeight}px`,
            }"
          >
            <!-- 桌面端高度拖拽手柄（design-taste：细 grip 条、hover 高亮、ns-resize 光标、实时跟随无动画） -->
            <div
              v-if="!isMobileComposer()"
              class="composer-resize-handle"
              :class="{ 'is-active': draggingHeight }"
              role="separator"
              aria-orientation="horizontal"
              :aria-label="t('topic.resizeComposer')"
              @pointerdown="startHeightDrag"
              @pointermove="moveHeightDrag"
              @pointerup="endHeightDrag"
              @pointercancel="endHeightDrag"
            >
              <span aria-hidden="true" />
            </div>

            <div class="mb-2 flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="text-sm font-semibold text-base-content">{{ composerTitle }}</div>
              </div>
              <button type="button" class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75 disabled:cursor-not-allowed disabled:opacity-60" :disabled="composerBusy" @click="closeComposer">
                <X class="h-4 w-4" />
              </button>
            </div>
            <template v-if="authenticated">
            <div v-if="target && !editing" class="mb-2 flex min-w-0 items-center justify-between gap-3 rounded-md border border-primary/20 bg-info/10 px-3 py-2">
              <div class="min-w-0 text-sm font-medium text-base-content/75">
                {{ t('topic.replyTo', { user: `@${target.author.username}` }) }}
              </div>
              <button type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-base-100" :aria-label="t('common.cancel')" @click="emit('clearTarget')">
                <X class="h-3.5 w-3.5" />
              </button>
            </div>

            <div ref="editorArea" class="relative min-h-0 flex-1 overflow-y-auto">
              <!-- Vditor 异步就绪前显示加载占位；初始化失败则结束转圈并提示失败（可关面板重开重试） -->
              <div
                v-if="!editorReady || editorInitFailed"
                class="absolute inset-0 z-10 flex items-center justify-center gap-2 bg-base-100/50 text-sm"
                :class="editorInitFailed ? 'text-error' : 'text-base-content/55'"
                :role="editorInitFailed ? 'alert' : 'status'"
                aria-live="polite"
              >
                <Loader2 v-if="!editorInitFailed" class="h-4 w-4 animate-spin" />
                <span>{{ editorInitFailed ? t('common.loadFailed') : t('common.loadingShort') }}</span>
              </div>
              <!-- 与发布页同款官版编辑器（紧凑工具栏）：粘贴/拖拽图片走官方 upload.handler → uploadImageFiles -->
              <VditorOfficial
                ref="editor"
                v-model="content"
                :height="isMobileComposer() ? 320 : 400"
                :compact="true"
                :placeholder="composerPlaceholder"
                @input="emit('clearValidation')"
                @upload="uploadImageFiles"
                @error="handleEditorError"
              />
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
                <img v-else :src="captchaImg" :alt="t('auth.captchaAlt')" class="gf-captcha-image h-full w-full object-cover" />
              </button>
              <input
                v-model="captchaCode"
                class="h-9 min-w-0 flex-1 rounded-md border border-line px-3 text-sm outline-none focus:border-primary"
                :placeholder="t('auth.captcha')"
                maxlength="8"
              />
            </div>
            <div class="mt-3 flex flex-wrap items-center gap-2">
              <button v-if="target && !editing" type="button" class="gf-button gf-button-md gf-button-muted shrink-0" @click="emit('clearTarget')">
                {{ t('common.cancel') }}
              </button>
              <button type="button" class="gf-button gf-button-md gf-button-primary ml-auto shrink-0" :disabled="composerBusy" @click="submit">
                <Loader2 v-if="composerBusy" class="h-4 w-4 animate-spin" />
                <Check v-else-if="editing" class="h-4 w-4" />
                <Send v-else class="h-4 w-4" />
                {{ submitText }}
              </button>
            </div>
            </template>
            <div v-else class="grid min-h-40 flex-1 place-items-center px-6 py-10 text-center">
              <Lock class="h-8 w-8 text-base-content/35" />
              <p class="mt-3 text-sm font-semibold text-base-content/70">{{ t('topic.loginRequiredToComment') }}</p>
              <a href="/login" class="gf-button gf-button-md gf-button-primary mt-4">{{ t('topic.loginToComment') }}</a>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </Teleport>
</template>

<style>
/*
 * 桌面端高度拖拽手柄（design-taste：冷静工具语言，VARIANCE 4 / MOTION 2 / DENSITY 6）：
 * 细 grip 条 + 稍宽热区；hover 高亮提示可拖；拖动中加深；实时跟随、无动画。
 */
.composer-resize-handle {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 16px;
  margin: -6px -12px 6px;
  cursor: ns-resize;
  touch-action: none;
  -webkit-user-select: none;
  user-select: none;
}

.composer-resize-handle > span {
  width: 36px;
  height: 4px;
  border-radius: 999px;
  background: color-mix(in oklch, var(--gf-color-base-content) 18%, transparent);
  transition: background-color 0.15s ease;
}

.composer-resize-handle:hover > span,
.composer-resize-handle.is-active > span {
  background: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
}

.composer-resize-handle.is-active {
  background: color-mix(in oklch, var(--gf-color-primary) 6%, transparent);
}
</style>

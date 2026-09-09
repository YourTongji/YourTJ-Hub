<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Check, Loader2, LockKeyhole, LockKeyholeOpen, Send, X } from '@lucide/vue'
import { uploadImage, searchForumUsers } from '@/runtime/api'
import { processImageFile, validateImageFile } from '@/runtime/image'
import { extractMentionToken, rankMentionCandidates, type MentionUser } from '@/runtime/mention'
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
  sensitiveWords?: string[]
  /** 是否允许匿名发布（wiki 评论区，issue #524）；true 时显示匿名勾选项。 */
  allowAnonymous?: boolean
  /** @mention 本地上下文候选（issue #564）：回复目标 > 主题作者 > 参与者，按此顺序传入 */
  mentionUsers?: MentionUser[]
  /** 当前登录用户 id：mention 候选排除自己（0/缺省表示未知） */
  currentUserId?: number
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
const anonymous = defineModel<boolean>('anonymous', { default: false })
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

// ---- @mention 会话（issue #564）----
// 检测/排序为纯逻辑（@/runtime/mention），这里只做 DOM 采集、服务端查询与键盘/布局编排。
const MENTION_DEBOUNCE_MS = 300
const mentionOpen = ref(false)
const mentionQuery = ref('')
const mentionTokenLength = ref(0)
const mentionCandidates = ref<MentionUser[]>([])
const mentionActiveIndex = ref(0)
const mentionLoading = ref(false)
const mentionFailed = ref(false)
const mentionRect = ref<DOMRect | null>(null)
const mentionDocked = ref(false)
const mentionEditorElement = ref<HTMLElement | null>(null)
const mentionPanelRef = ref<HTMLElement | null>(null)
let mentionSearchSeq = 0
let mentionDebounceTimer: ReturnType<typeof setTimeout> | undefined
let mentionAbort: AbortController | null = null
/** 上次进入 debounce 的 query：query 变化时立即废弃在途旧 query 的响应 */
let lastScheduledQuery = ''

function refreshMentionSession() {
  const ctx = editor.value?.getMentionContext()
  if (!ctx || ctx.inCode) {
    closeMention()
    return
  }
  const token = extractMentionToken(ctx.prefix)
  if (!token) {
    closeMention()
    return
  }
  mentionOpen.value = true
  mentionTokenLength.value = token.length
  mentionQuery.value = token.query
  mentionRect.value = ctx.rect
  mentionEditorElement.value = ctx.element
  const local = props.mentionUsers ?? []
  const currentUserId = props.currentUserId ?? 0
  if (!token.query.trim()) {
    // 仅输入 @：本地上下文（最多 5），无上下文时展示「继续输入」提示而非全站空查询
    cancelMentionSearch()
    mentionLoading.value = false
    mentionFailed.value = false
    mentionCandidates.value = rankMentionCandidates({ local, server: [], query: '', currentUserId })
    mentionActiveIndex.value = 0
    return
  }
  // 至少 1 字符：300ms debounce 后查服务端；query 变化即废弃在途旧响应（防 debounce 窗口内旧结果覆盖）
  if (token.query !== lastScheduledQuery) {
    lastScheduledQuery = token.query
    mentionSearchSeq++
    mentionAbort?.abort()
    mentionAbort = null
  }
  clearTimeout(mentionDebounceTimer)
  mentionDebounceTimer = setTimeout(() => {
    void runMentionSearch(local, currentUserId)
  }, MENTION_DEBOUNCE_MS)
}

async function runMentionSearch(local: MentionUser[], currentUserId: number) {
  const seq = ++mentionSearchSeq
  mentionAbort?.abort()
  const controller = new AbortController()
  mentionAbort = controller
  mentionLoading.value = true
  mentionFailed.value = false
  try {
    const users = await searchForumUsers(mentionQuery.value, controller.signal)
    if (seq !== mentionSearchSeq) return
    mentionCandidates.value = rankMentionCandidates({
      local,
      server: users,
      query: mentionQuery.value,
      currentUserId,
    })
    mentionActiveIndex.value = 0
  } catch (error) {
    if (seq !== mentionSearchSeq) return
    if (error instanceof DOMException && error.name === 'AbortError') return
    mentionFailed.value = true
  } finally {
    if (seq === mentionSearchSeq) mentionLoading.value = false
  }
}

function cancelMentionSearch() {
  clearTimeout(mentionDebounceTimer)
  mentionSearchSeq++
  mentionAbort?.abort()
  mentionAbort = null
  mentionLoading.value = false
}

function closeMention() {
  cancelMentionSearch()
  lastScheduledQuery = ''
  mentionOpen.value = false
  mentionQuery.value = ''
  mentionTokenLength.value = 0
  mentionCandidates.value = []
  mentionActiveIndex.value = 0
  mentionFailed.value = false
  mentionRect.value = null
  mentionEditorElement.value = null
}

function selectMention(user: MentionUser) {
  if (!mentionOpen.value) return
  const tokenLength = mentionTokenLength.value
  closeMention()
  editor.value?.replaceMentionToken(tokenLength, `@${user.username}`)
  emit('clearValidation')
}

function moveMentionActive(delta: number) {
  const count = mentionCandidates.value.length
  if (!count) return
  mentionActiveIndex.value = (mentionActiveIndex.value + delta + count) % count
  const list = mentionPanelRef.value
  const item = list?.querySelector<HTMLElement>('[data-active="true"]')
  if (!list || !item) return
  const itemTop = item.offsetTop
  const itemBottom = itemTop + item.offsetHeight
  if (itemTop < list.scrollTop) list.scrollTop = itemTop
  else if (itemBottom > list.scrollTop + list.clientHeight) list.scrollTop = itemBottom - list.clientHeight
}

function onEditorInput() {
  emit('clearValidation')
  refreshMentionSession()
}

function onEditorAreaKeydown(event: KeyboardEvent) {
  if (event.isComposing) return
  if (!mentionOpen.value || !mentionCandidates.value.length) {
    if (mentionOpen.value && event.key === 'Escape') {
      event.preventDefault()
      closeMention()
    }
    return
  }
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      moveMentionActive(1)
      break
    case 'ArrowUp':
      event.preventDefault()
      moveMentionActive(-1)
      break
    case 'Enter':
      event.preventDefault()
      selectMention(mentionCandidates.value[mentionActiveIndex.value])
      break
    case 'Escape':
      event.preventDefault()
      closeMention()
      break
    // Tab 不做补全键：保持正常焦点导航
    case 'ArrowLeft':
    case 'ArrowRight':
    case 'Home':
    case 'End':
      // caret 移动不触发 input 事件，这里主动重检；setTimeout 保证默认动作（光标移动）已生效
      setTimeout(() => refreshMentionSession(), 0)
      break
  }
}

function onDocumentClickCapture(event: MouseEvent) {
  // 会话开启或点击发生在编辑区内时重检（点击会移动 caret，不触发 input 事件）
  const target = event.target
  if (!mentionOpen.value && !(target instanceof Node && editorArea.value?.contains(target))) return
  setTimeout(() => refreshMentionSession(), 0)
}

function onDocumentFocusOutCapture(event: FocusEvent) {
  const target = event.target
  if (!(target instanceof Node) || !editorArea.value?.contains(target)) return
  const next = event.relatedTarget as Node | null
  if (next && editorArea.value.contains(next)) return
  closeMention()
}

function onMentionOptionPointerDown(event: PointerEvent) {
  // 阻止失焦：editor 保持唯一主要 focus（issue #564 a11y）
  event.preventDefault()
}

function updateMentionLayout() {
  mentionDocked.value = window.matchMedia('(max-width: 640px)').matches
}

/** 桌面弹层：贴近 caret、宽度 352px、空间不足向上翻转、不越出编辑区水平边界 */
const mentionPanelStyle = computed(() => {
  if (mentionDocked.value || !mentionRect.value || !editorArea.value) return {}
  const surfaceRect = editorArea.value.getBoundingClientRect()
  const rect = mentionRect.value
  const PANEL_WIDTH = 352
  const GAP = 6
  const rowHeight = 52
  const panelHeight = Math.min(Math.max(mentionCandidates.value.length, 1), 8) * rowHeight + 8
  const left = Math.min(Math.max(rect.left - surfaceRect.left, 8), Math.max(surfaceRect.width - PANEL_WIDTH - 8, 8))
  const spaceBelow = surfaceRect.bottom - rect.bottom
  const spaceAbove = rect.top - surfaceRect.top
  const top = spaceBelow < panelHeight + GAP && spaceAbove > panelHeight + GAP
    ? rect.top - surfaceRect.top - panelHeight - GAP
    : rect.bottom - surfaceRect.top + GAP
  return { left: `${left}px`, top: `${top}px`, width: `${PANEL_WIDTH}px` }
})

const mentionStatusText = computed(() => {
  if (mentionLoading.value) return t('mention.loading')
  if (mentionFailed.value) return t('mention.searchFailed')
  if (!mentionCandidates.value.length) {
    return mentionQuery.value.trim() ? t('mention.noResults') : t('mention.keepTyping')
  }
  return t('mention.resultCount', { count: mentionCandidates.value.length })
})

function mentionOptionId(user: MentionUser) {
  return `gf-mention-option-${user.id}`
}

function mentionTagLabel(tag: MentionUser['tag']) {
  if (tag === 'reply-target') return t('mention.replyTarget')
  if (tag === 'topic-author') return t('mention.topicAuthor')
  if (tag === 'participant') return t('mention.participant')
  return ''
}

watch(mentionActiveIndex, (index) => {
  const optionId = mentionOpen.value && mentionCandidates.value[index]
    ? mentionOptionId(mentionCandidates.value[index])
    : ''
  mentionEditorElement.value?.setAttribute('aria-activedescendant', optionId)
  mentionEditorElement.value?.setAttribute('aria-expanded', mentionOpen.value ? 'true' : 'false')
})

watch(mentionOpen, (open) => {
  const element = mentionEditorElement.value
  if (!element) return
  if (open) {
    element.setAttribute('role', 'combobox')
    element.setAttribute('aria-autocomplete', 'list')
    element.setAttribute('aria-expanded', 'true')
    element.setAttribute('aria-controls', 'gf-mention-listbox')
    element.setAttribute('aria-activedescendant', mentionCandidates.value[mentionActiveIndex.value]
      ? mentionOptionId(mentionCandidates.value[mentionActiveIndex.value])
      : '')
  } else {
    element.removeAttribute('role')
    element.removeAttribute('aria-autocomplete')
    element.removeAttribute('aria-expanded')
    element.removeAttribute('aria-controls')
    element.removeAttribute('aria-activedescendant')
  }
})

/** 浮动面板高度：支持桌面端与移动端顶部手柄拖拽调整 */
const MOBILE_VIEWPORT_QUERY = '(max-width: 520px)'
const DESKTOP_COMPOSER_HEIGHT = 480
const MIN_COMPOSER_HEIGHT = 240
const MAX_COMPOSER_HEIGHT = 720
const isMobileComposer = () => typeof window !== 'undefined' && window.matchMedia(MOBILE_VIEWPORT_QUERY).matches
const composerHeight = ref(DESKTOP_COMPOSER_HEIGHT)
const editorArea = ref<HTMLElement | null>(null)
const draggingHeight = ref(false)
const dragStartY = ref(0)
const dragStartHeight = ref(0)

onMounted(() => {
  if (isMobileComposer()) {
    composerHeight.value = Math.min(380, Math.max(MIN_COMPOSER_HEIGHT, Math.floor(window.innerHeight * 0.6)))
  }
  updateMentionLayout()
  window.addEventListener('resize', updateMentionLayout)
  // capture 阶段监听：editorArea 可能随 authenticated 翻转晚挂载，故挂 document 上按目标过滤
  document.addEventListener('keydown', onEditorAreaKeydown, true)
  document.addEventListener('click', onDocumentClickCapture, true)
  document.addEventListener('focusout', onDocumentFocusOutCapture, true)
})

function startHeightDrag(event: PointerEvent) {
  draggingHeight.value = true
  dragStartY.value = event.clientY
  dragStartHeight.value = composerHeight.value
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function moveHeightDrag(event: PointerEvent) {
  if (!draggingHeight.value) return
  // 手柄在顶部：向上拖（clientY 减小）→ 变高；向下拖 → 变矮
  const delta = dragStartY.value - event.clientY
  const maxAllowed = isMobileComposer() ? Math.max(MIN_COMPOSER_HEIGHT, window.innerHeight - 24) : MAX_COMPOSER_HEIGHT
  composerHeight.value = Math.min(maxAllowed, Math.max(MIN_COMPOSER_HEIGHT, dragStartHeight.value + delta))
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

// 访客登录门的登录链接引用：打开面板时把键盘焦点移过去（访客态没有编辑器可聚焦）
const loginLinkRef = ref<HTMLAnchorElement | null>(null)
// 访客门进场叙事：先呈现"开锁"，短暂停留后切换为"闭锁"——登录后才能解锁评论
const lockSettled = ref(false)
let lockSettleTimer: ReturnType<typeof setTimeout> | undefined

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    // 面板关闭会卸载内层 Vditor；再次打开时清失败态，重新走 loading → ready/error
    editorInitFailed.value = false
    if (!props.authenticated) {
      lockSettled.value = false
      clearTimeout(lockSettleTimer)
      lockSettleTimer = setTimeout(() => {
        lockSettled.value = true
      }, 650)
    }
    await nextTick()
    window.requestAnimationFrame(() => {
      if (props.authenticated) {
        if (!editorInitFailed.value) editor.value?.focus()
      } else {
        loginLinkRef.value?.focus()
      }
    })
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  clearTimeout(lockSettleTimer)
  clearTimeout(mentionDebounceTimer)
  mentionAbort?.abort()
  window.removeEventListener('resize', updateMentionLayout)
  document.removeEventListener('keydown', onEditorAreaKeydown, true)
  document.removeEventListener('click', onDocumentClickCapture, true)
  document.removeEventListener('focusout', onDocumentFocusOutCapture, true)
})

// 面板关闭/重开：关闭时结束 mention 会话；打开时重算 dock/popover 布局
watch(
  () => props.open,
  (open) => {
    if (open) updateMentionLayout()
    else closeMention()
  },
  { immediate: true },
)

// 登录后回跳当前内容页；服务端只读取 ?redirect=，并用站内相对路径白名单校验。
const loginHref = computed(() => {
  const currentPath = typeof window === 'undefined' ? '' : window.location.pathname + window.location.search + window.location.hash
  return currentPath ? `/login?redirect=${encodeURIComponent(currentPath)}` : '/login'
})

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
            role="dialog"
            aria-labelledby="post-composer-title"
            class="gf-floating-surface gf-composer-surface pointer-events-auto relative flex max-h-[calc(100dvh-1rem)] w-[min(42rem,calc(100vw-1.5rem))] flex-col overflow-visible p-3"
            :style="{
              height: `${composerHeight}px`,
            }"
          >
            <!-- 高度拖拽手柄（移动端与桌面端统一提供，实时跟随无动画） -->
            <div
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
                <div id="post-composer-title" class="text-sm font-semibold text-base-content">{{ composerTitle }}</div>
              </div>
              <button type="button" class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75 disabled:cursor-not-allowed disabled:opacity-60" :disabled="composerBusy" :aria-label="t('common.close')" @click="closeComposer">
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

            <div ref="editorArea" class="relative min-h-[160px] flex-1 flex flex-col overflow-visible">
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
                :height="isMobileComposer() ? 320 : '100%'"
                :compact="true"
                :placeholder="composerPlaceholder"
                :sensitive-words="sensitiveWords"
                @input="onEditorInput"
                @upload="uploadImageFiles"
                @error="handleEditorError"
              />
              <!-- @mention 候选（issue #564）：桌面贴近 caret 浮层（空间不足向上翻转/不越界），
                   移动端（≤640px / 200% zoom 窄空间）停靠在编辑区与工具/发送区之间 -->
              <Transition name="mention">
                <div
                  v-if="mentionOpen"
                  id="gf-mention-listbox"
                  ref="mentionPanelRef"
                  role="listbox"
                  :aria-label="t('mention.listboxLabel')"
                  class="gf-mention-panel"
                  :class="{ 'is-docked': mentionDocked }"
                  :style="mentionPanelStyle"
                >
                  <div v-if="mentionLoading && !mentionCandidates.length" class="gf-mention-status">{{ t('mention.loading') }}</div>
                  <template v-else-if="mentionCandidates.length">
                    <div
                      v-for="(user, index) in mentionCandidates"
                      :id="mentionOptionId(user)"
                      :key="user.id"
                      role="option"
                      :aria-selected="index === mentionActiveIndex"
                      :aria-label="`${user.nickname || user.username} @${user.username}`"
                      class="gf-mention-option"
                      :class="{ 'is-active': index === mentionActiveIndex }"
                      :data-active="index === mentionActiveIndex || undefined"
                      @pointerdown="onMentionOptionPointerDown"
                      @click="selectMention(user)"
                    >
                      <img :src="user.avatarUrl" alt="" class="gf-mention-avatar" loading="lazy" />
                      <span class="min-w-0 flex-1">
                        <span class="gf-mention-nickname">{{ user.nickname || user.username }}</span>
                        <span class="gf-mention-username">@{{ user.username }}</span>
                      </span>
                      <span v-if="user.tag" class="gf-mention-tag">{{ mentionTagLabel(user.tag) }}</span>
                    </div>
                  </template>
                  <div v-else class="gf-mention-status" :class="{ 'is-error': mentionFailed }">
                    {{ mentionFailed ? t('mention.searchFailed') : (mentionQuery.trim() ? t('mention.noResults') : t('mention.keepTyping')) }}
                  </div>
                </div>
              </Transition>
            </div>
            <!-- mention 会话状态（loading/empty/error/result count）polite live region，面板外常驻以稳定播报 -->
            <div aria-live="polite" class="sr-only">{{ mentionOpen ? mentionStatusText : '' }}</div>

            <p v-if="errorMessage" class="mt-2 text-sm text-error">{{ errorMessage }}</p>
            <p v-if="successMessage" class="mt-2 text-sm text-success">{{ successMessage }}</p>
            <div v-if="captchaRequired" class="mt-2 flex flex-wrap items-center gap-2 shrink-0">
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
            <label v-if="allowAnonymous && !editing" class="mt-2 flex shrink-0 cursor-pointer items-center gap-2 text-[13px] text-base-content/75">
              <input v-model="anonymous" type="checkbox" class="checkbox checkbox-sm" />
              {{ t('topic.publishAnonymous') }}
            </label>
            <div class="mt-3 flex flex-wrap items-center gap-2 shrink-0">
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
            <div
              v-else
              class="flex min-h-40 flex-1 flex-col items-center justify-center gap-5 px-6 py-8 text-center"
              :class="{ 'guest-lock-settled': lockSettled }"
            >
              <!-- 开锁 → 闭锁 进场叙事：访客看到"需要登录才能解锁评论" -->
              <div class="relative h-14 w-14 shrink-0 overflow-hidden rounded-full bg-info/10 text-primary">
                <LockKeyholeOpen aria-hidden="true" class="guest-lock-icon guest-lock-open absolute inset-0 m-auto h-7 w-7" />
                <LockKeyhole aria-hidden="true" class="guest-lock-icon guest-lock-closed absolute inset-0 m-auto h-7 w-7" />
              </div>
              <div class="space-y-1.5">
                <p class="text-sm font-semibold text-base-content">{{ t('topic.loginRequiredToComment') }}</p>
                <p class="text-xs text-base-content/55">{{ t('topic.loginRequiredToCommentHint') }}</p>
              </div>
              <a
                ref="loginLinkRef"
                :href="loginHref"
                class="gf-button gf-button-md gf-button-primary"
              >{{ t('topic.loginToComment') }}</a>
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

/*
 * 访客登录门：开锁 → 闭锁 进场叙事（design-taste：trust-first 语言，VARIANCE 4 / MOTION 5）。
 * 面板打开先呈现"开锁"，650ms 后 cross-fade 到"闭锁"，隐喻"登录后才能解锁评论"。
 * 遵循 better-ui：opacity / scale / blur 驱动（0.25→1、0→1、4px→0），一次性、之后静止；
 * prefers-reduced-motion 下直接显示闭锁态。
 */
.guest-lock-icon {
  transition:
    opacity 0.45s cubic-bezier(0.2, 0, 0, 1),
    transform 0.45s cubic-bezier(0.2, 0, 0, 1),
    filter 0.45s cubic-bezier(0.2, 0, 0, 1);
}

.guest-lock-open {
  opacity: 1;
  transform: scale(1);
  filter: blur(0);
}

.guest-lock-closed {
  opacity: 0;
  transform: scale(0.25);
  filter: blur(4px);
}

.guest-lock-settled .guest-lock-open {
  opacity: 0;
  transform: scale(0.25);
  filter: blur(4px);
}

.guest-lock-settled .guest-lock-closed {
  opacity: 1;
  transform: scale(1);
  filter: blur(0);
}

@media (prefers-reduced-motion: reduce) {
  .guest-lock-icon {
    transition: none;
  }

  .guest-lock-open {
    opacity: 0;
    transform: scale(1);
    filter: none;
  }

  .guest-lock-closed {
    opacity: 1;
  }
}

/* 移动端回复框专属工具栏与编辑器美化：对称均分、优雅质感、直观触达 */
@media (max-width: 640px) {
  .gf-floating-surface .vditor {
    border: 1px solid color-mix(in oklch, var(--gf-color-base-content) 12%, transparent) !important;
    border-radius: 14px !important;
    overflow: visible !important;
    background-color: var(--color-base-100, #fff) !important;
    box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.04) !important;
    transition: border-color 0.2s ease, box-shadow 0.2s ease !important;
  }

  .gf-floating-surface .vditor:focus-within {
    border-color: color-mix(in oklch, var(--gf-color-primary) 65%, transparent) !important;
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--gf-color-primary) 12%, transparent) !important;
  }

  .gf-floating-surface .vditor-toolbar {
    display: flex !important;
    align-items: center !important;
    justify-content: space-between !important;
    padding: 6px 8px !important;
    background: color-mix(in oklch, var(--gf-color-base-200) 45%, var(--gf-color-base-100)) !important;
    border-bottom: 1px solid color-mix(in oklch, var(--gf-color-base-content) 8%, transparent) !important;
    border-radius: 14px 14px 0 0 !important;
    gap: 2px !important;
    overflow: visible !important;
    flex-wrap: nowrap !important;
  }

  .gf-floating-surface .vditor-toolbar__item {
    flex: 1 1 0 !important;
    max-width: 36px !important;
    min-width: 28px !important;
    margin: 0 !important;
    padding: 0 !important;
    display: flex !important;
    justify-content: center !important;
    align-items: center !important;
  }

  .gf-composer-surface .vditor-toolbar__item .vditor-tooltipped {
    width: 32px !important;
    height: 32px !important;
    padding: 6px !important;
    border-radius: 8px !important;
    display: inline-flex !important;
    align-items: center !important;
    justify-content: center !important;
    color: color-mix(in oklch, var(--gf-color-base-content) 75%, transparent) !important;
    transition: all 0.15s cubic-bezier(0.2, 0, 0, 1) !important;
  }

  .gf-composer-surface .vditor-toolbar__item .vditor-tooltipped:hover {
    background-color: color-mix(in oklch, var(--gf-color-base-content) 10%, transparent) !important;
    color: var(--color-base-content, #24292e) !important;
  }

  .gf-composer-surface .vditor-toolbar__item .vditor-tooltipped:active {
    transform: scale(0.9) !important;
    background-color: color-mix(in oklch, var(--gf-color-primary) 15%, transparent) !important;
  }

  .gf-composer-surface .vditor-toolbar__item .vditor-tooltipped svg {
    width: 17px !important;
    height: 17px !important;
  }

  .gf-composer-surface .vditor-panel,
  .gf-composer-surface .vditor-hint {
    z-index: 100 !important;
    max-width: min(320px, calc(100vw - 32px)) !important;
    top: calc(100% + 4px) !important;
    bottom: auto !important;
    left: 0 !important;
    right: auto !important;
    max-height: min(220px, 38vh) !important;
    overflow-y: auto !important;
  }

  .gf-composer-surface .vditor-toolbar__item:last-child > .vditor-hint,
  .gf-composer-surface .vditor-toolbar__item:last-child > .vditor-panel,
  .gf-composer-surface .vditor-toolbar__item:nth-last-child(2) > .vditor-hint,
  .gf-composer-surface .vditor-toolbar__item:nth-last-child(2) > .vditor-panel {
    right: 0 !important;
    left: auto !important;
  }

  .gf-composer-surface .vditor-emojis {
    max-height: 160px !important;
  }
}

/* 移动端与桌面端回复框：全高 Flex 自适应填满，底栏确认/取消按键永不被推挤或遮挡 */
.gf-composer-surface .vditor-official {
  height: 100% !important;
  display: flex !important;
  flex-direction: column !important;
  min-height: 0 !important;
}

.gf-composer-surface .vditor-official .vditor {
  height: 100% !important;
  flex: 1 1 0 !important;
  display: flex !important;
  flex-direction: column !important;
  min-height: 0 !important;
}

.gf-composer-surface .vditor-official .vditor-content {
  flex: 1 1 0 !important;
  height: 100% !important;
  min-height: 0 !important;
}

@media (max-width: 640px) {
  .gf-composer-surface {
    min-height: 240px;
  }
}

/* 桌面与通用回复框：工具栏下拉浮层自然向下展开，并限制最大高度防止外层容器产生滚动条 */
.gf-composer-surface .vditor-toolbar__item > .vditor-hint,
.gf-composer-surface .vditor-toolbar__item > .vditor-panel {
  top: calc(100% + 4px) !important;
  bottom: auto !important;
  max-height: min(280px, 45vh) !important;
  overflow-y: auto !important;
  scrollbar-width: thin !important;
}

.gf-composer-surface .vditor-toolbar__item:last-child > .vditor-hint,
.gf-composer-surface .vditor-toolbar__item:last-child > .vditor-panel {
  right: 0 !important;
  left: auto !important;
}

/*
 * @mention 候选面板（issue #564；design-taste：浮动工具语言，VARIANCE 4 / MOTION 2）。
 * 桌面：absolute 贴近 caret 的浮层（352px）；移动/窄空间：is-docked 停靠编辑区底部。
 * 行高 48px 起（touch target ≥44px）；自身滚动不带动页面（overscroll-behavior: contain）。
 */
.gf-mention-panel {
  position: absolute;
  z-index: 30;
  width: 352px;
  max-width: calc(100vw - 2rem);
  max-height: min(416px, 50vh);
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 4px;
  border: 1px solid color-mix(in oklch, var(--gf-color-base-content) 14%, transparent);
  border-radius: 12px;
  background: var(--color-base-100, #fff);
  box-shadow: 0 8px 24px 0 rgba(0, 0, 0, 0.12);
  scrollbar-width: thin;
}

.gf-mention-panel.is-docked {
  position: static;
  width: 100%;
  margin-top: 8px;
  max-height: 234px;
}

.gf-mention-option {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 4px 10px 4px 12px;
  border-radius: 8px;
  cursor: pointer;
  position: relative;
  -webkit-user-select: none;
  user-select: none;
}

/* active 不只靠颜色：左侧 3px 强调条 + aria-selected + 背景 */
.gf-mention-option.is-active {
  background: color-mix(in oklch, var(--gf-color-primary) 12%, transparent);
}

.gf-mention-option.is-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 999px;
  background: var(--gf-color-primary);
}

.gf-mention-avatar {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: 999px;
  object-fit: cover;
  background: color-mix(in oklch, var(--gf-color-base-content) 10%, transparent);
}

.gf-mention-nickname,
.gf-mention-username {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gf-mention-nickname {
  font-size: 14px;
  font-weight: 500;
  color: var(--gf-color-base-content);
}

.gf-mention-username {
  font-size: 12px;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
}

.gf-mention-tag {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  line-height: 1.4;
  color: color-mix(in oklch, var(--gf-color-base-content) 60%, transparent);
  background: color-mix(in oklch, var(--gf-color-base-content) 8%, transparent);
}

.gf-mention-status {
  padding: 10px 12px;
  font-size: 13px;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
}

.gf-mention-status.is-error {
  color: var(--gf-color-error, #e5484d);
}

/* 进场：轻微上浮淡入；prefers-reduced-motion 下无动画 */
.mention-enter-active {
  transition:
    opacity 0.15s cubic-bezier(0.2, 0, 0, 1),
    transform 0.15s cubic-bezier(0.2, 0, 0, 1);
}

.mention-enter-from {
  opacity: 0;
  transform: translateY(-4px);
}

@media (prefers-reduced-motion: reduce) {
  .mention-enter-active {
    transition: none;
  }

  .mention-enter-from {
    opacity: 1;
    transform: none;
  }
}
</style>

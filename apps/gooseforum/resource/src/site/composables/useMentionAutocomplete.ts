/**
 * 编辑器 @mention 会话编排（issue #564 实现，issue #590 自 PostComposer 抽取共享）。
 *
 * 职责边界：
 * - 检测/排序为纯逻辑（@/runtime/mention）；caret 上下文采集与 token 替换由宿主编辑器
 *   桥接（VditorOfficial expose 的 getMentionContext/replaceMentionToken）完成；
 * - 这里只做会话状态、服务端搜索（debounce + abort + stale 丢弃）、键盘/布局编排与
 *   编辑器 aria 状态维护；
 * - 候选面板展示由 MentionCandidates.vue 承担；PostComposer / PublishPage /
 *   QuickPublishModal 三个编辑器统一消费（issue #590）。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch, type CSSProperties } from 'vue'
import { extractMentionToken, rankMentionCandidates, type MentionUser } from '@/runtime/mention'
import { searchForumUsers } from '@/runtime/api'

/** 宿主编辑器需要提供的 mention 桥接（与 VditorOfficial expose 的 getMentionContext/replaceMentionToken 同形） */
export interface MentionEditorBridge {
  /** 当前光标处 mention 上下文；光标不在编辑器内时为 null */
  getMentionContext(): { prefix: string; rect: DOMRect | null; inCode: boolean; element: HTMLElement | null } | null
  /** 以光标为终点向前删除 queryLength 个字符并插入 replacement */
  replaceMentionToken(queryLength: number, replacement: string): boolean
}

/** 面板 listbox id 与 option id 前缀：composable 的 aria 维护与面板组件共用 */
export const MENTION_LISTBOX_ID = 'gf-mention-listbox'
export const MENTION_OPTION_ID_PREFIX = 'gf-mention-option-'

const MENTION_DEBOUNCE_MS = 300
const MENTION_DOCK_QUERY = '(max-width: 640px)'

export function useMentionAutocomplete(options: {
  /** 编辑器桥接：通常返回 VditorOfficial 组件 ref（expose 已含两个方法） */
  editor: () => MentionEditorBridge | null
  /** 本地上下文候选（回复目标 > 主题作者 > 参与者）；无上下文的发帖场景传 [] */
  localUsers: () => MentionUser[]
  /** 当前登录用户 id：候选排除自己（0 表示未知） */
  currentUserId: () => number
  /** 面板定位与键盘/点击命中过滤参照的编辑区容器 */
  surface: () => HTMLElement | null
  /** 选中候选后的宿主回调（token 替换后调用，如清发布校验态） */
  onSelect?: (user: MentionUser) => void
}) {
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
  let searchSeq = 0
  let debounceTimer: ReturnType<typeof setTimeout> | undefined
  let abort: AbortController | null = null
  /** 上次进入 debounce 的 query：query 变化时立即废弃在途旧 query 的响应 */
  let lastScheduledQuery = ''

  function refreshMentionSession() {
    const ctx = options.editor()?.getMentionContext()
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
    const local = options.localUsers()
    const currentUserId = options.currentUserId()
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
      mentionCandidates.value = rankMentionCandidates({ local, server: [], query: token.query, currentUserId })
      mentionActiveIndex.value = 0
      mentionFailed.value = false
      lastScheduledQuery = token.query
      searchSeq++
      abort?.abort()
      abort = null
    }
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      void runMentionSearch(local, currentUserId)
    }, MENTION_DEBOUNCE_MS)
  }

  async function runMentionSearch(local: MentionUser[], currentUserId: number) {
    const seq = ++searchSeq
    abort?.abort()
    const controller = new AbortController()
    abort = controller
    mentionLoading.value = true
    mentionFailed.value = false
    try {
      const users = await searchForumUsers(mentionQuery.value, controller.signal)
      if (seq !== searchSeq) return
      mentionCandidates.value = rankMentionCandidates({
        local,
        server: users,
        query: mentionQuery.value,
        currentUserId,
      })
      mentionActiveIndex.value = 0
    } catch (error) {
      if (seq !== searchSeq) return
      if (error instanceof DOMException && error.name === 'AbortError') return
      mentionFailed.value = true
    } finally {
      if (seq === searchSeq) mentionLoading.value = false
    }
  }

  function cancelMentionSearch() {
    clearTimeout(debounceTimer)
    searchSeq++
    abort?.abort()
    abort = null
    mentionLoading.value = false
  }

  function clearMentionAria(element: HTMLElement | null) {
    for (const name of ['role', 'aria-autocomplete', 'aria-expanded', 'aria-controls', 'aria-activedescendant']) {
      element?.removeAttribute(name)
    }
  }

  function closeMention() {
    clearMentionAria(mentionEditorElement.value)
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

  /** 选中候选：关闭会话并替换 token */
  function selectMention(user: MentionUser) {
    if (!mentionOpen.value) return
    const tokenLength = mentionTokenLength.value
    closeMention()
    options.editor()?.replaceMentionToken(tokenLength, `@${user.username}`)
    options.onSelect?.(user)
  }

  function moveMentionActive(delta: number) {
    const count = mentionCandidates.value.length
    if (!count) return
    mentionActiveIndex.value = (mentionActiveIndex.value + delta + count) % count
  }

  function onEditorAreaKeydown(event: KeyboardEvent) {
    const surface = options.surface()
    if (!(event.target instanceof Node) || !surface?.contains(event.target) || event.isComposing) return
    if (!mentionOpen.value || !mentionCandidates.value.length) {
      if (mentionOpen.value && event.key === 'Escape') {
        event.preventDefault()
        event.stopPropagation()
        closeMention()
      }
      return
    }
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault()
        event.stopPropagation()
        moveMentionActive(1)
        break
      case 'ArrowUp':
        event.preventDefault()
        event.stopPropagation()
        moveMentionActive(-1)
        break
      case 'Enter':
        event.preventDefault()
        event.stopPropagation()
        selectMention(mentionCandidates.value[mentionActiveIndex.value])
        break
      case 'Escape':
        event.preventDefault()
        event.stopPropagation()
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
    if (!mentionOpen.value && !(target instanceof Node && options.surface()?.contains(target))) return
    setTimeout(() => refreshMentionSession(), 0)
  }

  function onDocumentFocusOutCapture(event: FocusEvent) {
    const surface = options.surface()
    const target = event.target
    if (!(target instanceof Node) || !surface?.contains(target)) return
    const next = event.relatedTarget as Node | null
    if (next && surface.contains(next)) return
    closeMention()
  }

  /** 移动端（≤640px / 200% zoom 窄空间）候选停靠在编辑区内，桌面贴近 caret 浮层 */
  function updateMentionLayout() {
    if (typeof window === 'undefined') return
    mentionDocked.value = window.matchMedia(MENTION_DOCK_QUERY).matches
  }

  function updateMentionPosition() {
    if (mentionOpen.value) mentionRect.value = options.editor()?.getMentionContext()?.rect ?? null
  }

  /** 桌面弹层：贴近 caret、宽度 352px、空间不足向上翻转、不越出编辑区水平边界 */
  const mentionPanelStyle = computed<CSSProperties>(() => {
    const surface = options.surface()
    if (mentionDocked.value || !mentionRect.value || !surface) return {}
    const surfaceRect = surface.getBoundingClientRect()
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

  // 编辑器 aria combobox 状态随会话/候选变化维护（a11y：宿主编辑元素即 combobox 触发器）
  watch([mentionOpen, mentionActiveIndex, mentionCandidates, mentionEditorElement], () => {
    const element = mentionEditorElement.value
    if (!element) return
    if (!mentionOpen.value) {
      clearMentionAria(element)
      return
    }
    element.setAttribute('role', 'combobox')
    element.setAttribute('aria-autocomplete', 'list')
    element.setAttribute('aria-expanded', 'true')
    element.setAttribute('aria-controls', MENTION_LISTBOX_ID)
    const candidate = mentionCandidates.value[mentionActiveIndex.value]
    if (candidate) element.setAttribute('aria-activedescendant', `${MENTION_OPTION_ID_PREFIX}${candidate.id}`)
    else element.removeAttribute('aria-activedescendant')
  })

  onMounted(() => {
    updateMentionLayout()
    // capture 阶段监听：编辑区可能随宿主面板开合晚挂载，故挂 document 上按目标过滤
    window.addEventListener('resize', updateMentionLayout)
    document.addEventListener('scroll', updateMentionPosition, true)
    document.addEventListener('keydown', onEditorAreaKeydown, true)
    document.addEventListener('click', onDocumentClickCapture, true)
    document.addEventListener('focusout', onDocumentFocusOutCapture, true)
  })

  onBeforeUnmount(() => {
    clearTimeout(debounceTimer)
    abort?.abort()
    window.removeEventListener('resize', updateMentionLayout)
    document.removeEventListener('scroll', updateMentionPosition, true)
    document.removeEventListener('keydown', onEditorAreaKeydown, true)
    document.removeEventListener('click', onDocumentClickCapture, true)
    document.removeEventListener('focusout', onDocumentFocusOutCapture, true)
  })

  return {
    mentionOpen,
    mentionQuery,
    mentionCandidates,
    mentionActiveIndex,
    mentionLoading,
    mentionFailed,
    mentionDocked,
    mentionPanelStyle,
    refreshMentionSession,
    closeMention,
    selectMention,
    updateMentionLayout,
  }
}

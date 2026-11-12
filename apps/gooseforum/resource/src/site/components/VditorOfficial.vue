<script setup lang="ts">
import { textBeforeCaret, isInsideFencedCode, replaceMentionTokenInElement } from "@/runtime/mention-dom"
import { deleteStickerAtCaret, stickerBeforeInputDirection, stickerDeleteDirection } from "@/runtime/sticker-atomic"
import { decorateStickerTokens, type StickerInlineAsset } from "@/runtime/sticker-inline-editor"
import StickerSuggestions from '@/site/components/StickerSuggestions.vue'
import EditorSelectionToolbar from '@/site/components/EditorSelectionToolbar.vue'
import EditorImageToolbar from '@/site/components/EditorImageToolbar.vue'
import MarkdownImageViewer from '@/components/MarkdownImageViewer.vue'
import { useEditorFloatingTools } from '@/site/composables/useEditorFloatingTools'
import { useResolvedStickers } from '@/site/composables/useResolvedStickers'
import { useStickerSuggest } from '@/site/composables/useStickerSuggest'
import { isPersonalStickerName } from '@/site/utils/sticker-token'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import luteUrl from 'vditor/dist/js/lute/lute.min.js?url'
import antIconUrl from 'vditor/dist/js/icons/ant.js?url'
import enUrl from 'vditor/dist/js/i18n/en_US.js?url'
import deUrl from 'vditor/dist/js/i18n/de_DE.js?url'
import jaUrl from 'vditor/dist/js/i18n/ja_JP.js?url'
import zhUrl from 'vditor/dist/js/i18n/zh_CN.js?url'
import katexJsUrl from 'katex/dist/katex.min.js?url'
import katexChemUrl from 'katex/dist/contrib/mhchem.min.js?url'
import katexCssUrl from 'katex/dist/katex.min.css?url'
import contentThemeLightCssUrl from 'vditor/dist/css/content-theme/light.css?url'
import contentThemeDarkCssUrl from 'vditor/dist/css/content-theme/dark.css?url'
import hljsLightCssUrl from 'highlight.js/styles/github.css?url'
import hljsDarkCssUrl from 'highlight.js/styles/github-dark.css?url'
import hljs from 'highlight.js'
import { currentLocale } from '@/runtime/i18n'
import { loadRuntimeScript } from '@/runtime/runtime-script'
import { loadMermaid, nextMermaidDiagramId, renderDiagram } from '@/runtime/content-enhancements/mermaid'
import { useSiteTheme } from '@/runtime/site-theme'
import { renderTwemojiHtml } from '@/runtime/twemoji'
import { containsSensitiveText } from '@/site/utils/sensitive-highlight'
import { useI18n } from 'vue-i18n'
import { createLinkPreviewHintController, formatLinkPreviewHintDomains, type LinkPreviewHint } from '@/runtime/link-preview-hint'
import { editorToolbarIcon, hasEditorToolbarIcon } from '@/runtime/editor-toolbar-icons'
import { BLANK_LINE_HTML, blankLineMarker, markBlankLines, MAX_RENDERED_BLANK_LINES } from '@/runtime/blank-lines'

/**
 * 官方 Vditor Vue 示例（https://b3log.org/vditor/demo/vue.html）的 Vue 3 移植。
 * 配置与官方逐项一致：height / mode:wysiwyg / toolbarConfig.pin / cache 关，
 * 其余全走 Vditor 默认（全功能默认工具栏、preview.maxWidth 800、classic 主题）。
 * wysiwyg 模式会显示悬浮在片段左侧的块级控制指示器（</>、H1/H2、$、ToC）。
 *
 * 官方说明（https://ld246.com/article/1549638745630）中 hljs 与 KaTeX 资源
 * 由 vditor 从 `${cdn}/dist/js/highlight.js/...`、`${cdn}/dist/js/katex/...`
 * 按需加载，npm 包不捆绑。为离线可用，这里预置同 id 节点，vditor 的
 * addScript/addStyle 检测到 id 已存在即跳过 CDN：
 * - window.hljs 来自项目依赖 highlight.js；空占位 script#vditorHljsScript 与
 *   #vditorHljsThirdScript 让官方流程直接进入高亮逻辑
 * - katex.min.js / mhchem.min.js 用 loadRuntimeScript 预载（带官方 id）
 * - KaTeX 样式由 link#vditorKatexStyle 提供；高亮样式由自定义 link 按主题切换
 * - content-theme light/dark 由 syncContentTheme 本地切换（path 为空时官方不会加载；
 *   另用 CSS 把 .vditor-reset 绑到 --textarea-text-color，避免深色正文仍用 #24292e）
 *
 * 站点集成：v-model 双向、上传经 upload 事件交宿主处理、卸载销毁。
 * 不做任何样式覆盖或功能裁剪——保持官版外观与行为。
 */
// 模板根是 Vditor 挂载点 + 贴纸候选浮层（Teleport），宿主 attrs 显式落到挂载点上
defineOptions({ inheritAttrs: false })

const props = defineProps<{
  modelValue: string
  placeholder: string
  /** 编辑器固定高度（官方 demo 为 360，支持 'auto' / '100%'） */
  height?: number | string
  /** 隐藏工具栏内的上传图片按钮 */
  hideUpload?: boolean
  /** 左侧大纲：按需开启（官方默认关闭） */
  outline?: boolean
  /** 字数统计：按需开启（官方默认关闭） */
  counter?: boolean
  /** 紧凑工具栏：浮动面板等窄容器只留高频按钮，其余收进 more（桌面端） */
  compact?: boolean
  /** 普通/极简工具栏（推特/知乎风格）：仅保留核心基础排版、表情与图片上传 */
  simple?: boolean
  /** 窄容器进一步精简（课评表单等 vw-64 嵌套容器）：
   *  移动端主行 7 项（标题左一），≤320px 视口单行完整；
   *  桌面端 compact 同构但标题提升左一（SLIM_DESKTOP_TOOLBAR），不影响其他 compact 宿主 */
  slimMobile?: boolean
  /** 工具栏最左显示「向上扩展」按钮：折叠标题/分类区（发布页用） */
  headerToggle?: boolean
  /** 头部折叠态：用于按钮激活高亮 */
  headerCollapsed?: boolean
  /** 移动端 toggle 挂载容器：窄屏下悬浮开关移入宿主提供的行内容器（如「正文」label 行），
   *  避免占用工具栏宽度导致横向挤压；桌面忽略此值（保持悬浮布局） */
  toggleHost?: HTMLElement | null
  /** 后端命中敏感词时，仅用于编辑区定位，不展示敏感词本身 */
  sensitiveWords?: string[]
  /** 在 emoji 旁注入站点表情包按钮（MADR 0030）：面板由宿主渲染，点击仅 emit open-stickers */
  stickerPicker?: boolean
  /** 注入「导入 Markdown」按钮（issue #1630）：解析与图片上传由宿主弹窗负责，点击仅 emit import-markdown */
  importMarkdown?: boolean
  /**
   * 编辑区排版档位（issue #1630）：与发布后 .gf-prose-post / -thought / -article 读取同一组
   * --gf-rich-* 节奏变量（styles/prose.css），编辑时看到的字号、段距、标题与图片间距即发布结果。
   */
  profile?: 'post' | 'thought' | 'article'
}>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  input: []
  upload: [files: File[]]
  error: [error: Error]
  'toggle-header': []
  'open-stickers': []
  'import-markdown': []
}>()

const { t, te, locale } = useI18n()
const { isDark } = useSiteTheme()
const root = ref<HTMLElement | null>(null)
const editorReady = ref(false)
const linkPreviewHint = ref<LinkPreviewHint | null>(null)
/** 资源预载 / 构造失败时为 true；宿主可据此结束 loading，避免遮罩永远转圈 */
const editorFailed = ref(false)
let editor: Vditor | null = null
let destroyed = false
let ready = false
/** 监听全屏按钮 childList：官方用 innerHTML 换图标会冲掉按钮内小字 */
let fullscreenLabelObserver: MutationObserver | null = null
let sensitiveModeRefreshHandler: ((event: Event) => void) | null = null
const SENSITIVE_BLOCK_CLASS = 'gf-sensitive-block'
const EDITOR_BLOCK_SELECTOR = '[data-block="0"], p, h1, h2, h3, h4, h5, h6, li, blockquote, pre'
let linkPreviewHintElement: HTMLParagraphElement | null = null
const linkPreviewHintController = createLinkPreviewHintController({
  onChange(hint) {
    linkPreviewHint.value = hint
    syncLinkPreviewHint()
  },
})

function syncLinkPreviewHint() {
  if (!linkPreviewHintElement) return
  const hint = linkPreviewHint.value
  if (!hint) {
    linkPreviewHintElement.textContent = ''
    linkPreviewHintElement.hidden = true
    return
  }
  // 单卡沿用原文案；多卡换用带数量的文案，避免英文/德文里「a link card」在
  // 复数下不成立（项目其余文案用 {count} 插值而不走 vue-i18n 复数，保持一致）。
  linkPreviewHintElement.textContent = hint.domains.length === 1
    ? t('linkPreview.editorHint', { domain: hint.domains[0] })
    : t('linkPreview.editorHintMany', {
        count: hint.domains.length,
        domains: formatLinkPreviewHintDomains(hint.domains, t('linkPreview.editorHintSeparator')),
      })
  linkPreviewHintElement.hidden = false
}

function handleCompositionStart() {
  composing = true
  linkPreviewHintController.compositionStart()
}

function handleCompositionEnd() {
  composing = false
  linkPreviewHintController.compositionEnd(editor?.getValue() ?? props.modelValue)
  scheduleStickerDecoration()
}

/**
 * 编辑区贴纸所见即所得（sticker-inline-editor）：完整且可解析的 token 显示为与正文
 * 同尺寸的贴纸图，Markdown 不变。Vditor 每次输入按块重渲染会把块还原成纯文本，
 * MutationObserver 回调在绘制前（微任务）重新装饰，用户看不到原始 token 闪现。
 * 只装饰 wysiwyg：ir/sv 是显式的 Markdown 源码视图，保持 token 原文。
 * IME 组合期间不改 DOM（会打断输入法），组合结束后补一次。
 */
const { resolvedUrls, ensureContentStickers } = useResolvedStickers()
let composing = false

// 选区格式条与图片工具条（MADR 0131）：状态与事件见 useEditorFloatingTools
const selectionToolbar = ref<InstanceType<typeof EditorSelectionToolbar> | null>(null)
const imageToolbar = ref<InstanceType<typeof EditorImageToolbar> | null>(null)
const imageViewer = ref<InstanceType<typeof MarkdownImageViewer> | null>(null)
const floating = useEditorFloatingTools({
  root,
  editor: () => (ready && !destroyed ? editor : null),
  modeElement: () => currentModeElement(),
  composing: () => composing,
  preview: (images, index) => imageViewer.value?.open(images, index),
  insideFloating: (node) => Boolean(selectionToolbar.value?.contains(node) || imageToolbar.value?.contains(node)),
})
let stickerObserver: MutationObserver | null = null
let stickerResolveTimer: ReturnType<typeof setTimeout> | undefined
let stickerDecorationFrame = 0

function stickerAsset(name: string): StickerInlineAsset | undefined {
  const url = resolvedUrls.value.get(name)
  if (!url) return undefined
  return { url, label: isPersonalStickerName(name) || /^m_[0-9a-f]{40}$/.test(name) ? t('stickers.custom') : name }
}

function decorateStickers() {
  if (!editor || !ready || destroyed || composing || editor.vditor.currentMode !== 'wysiwyg') return
  const element = editor.vditor.wysiwyg?.element
  if (!element) return
  decorateStickerTokens(element, stickerAsset)
  // 丢弃装饰自身产生的变更记录，避免回调自我触发
  stickerObserver?.takeRecords()
}

function scheduleStickerDecoration() {
  if (stickerDecorationFrame) return
  stickerDecorationFrame = requestAnimationFrame(() => {
    stickerDecorationFrame = 0
    decorateStickers()
  })
}

function attachStickerDecoration() {
  const element = editor?.vditor.wysiwyg?.element
  if (!element || stickerObserver) return
  stickerObserver = new MutationObserver(() => decorateStickers())
  stickerObserver.observe(element, { childList: true, subtree: true, characterData: true })
  decorateStickers()
  void ensureContentStickers([props.modelValue])
}

/** 手动输入或粘贴的 token 名字未知时，静默 300ms 后批量解析；已知/60s 内查过的不重复请求。 */
function scheduleStickerResolve(value: string) {
  clearTimeout(stickerResolveTimer)
  if (!value.includes('[:sticker:')) return
  stickerResolveTimer = setTimeout(() => {
    void ensureContentStickers([value])
  }, 300)
}

const stickerSuggest = useStickerSuggest({
  bridge: () => (props.stickerPicker ? { getMentionContext, replaceStickerQuery } : null),
  label: (item) => item.displayName || (isPersonalStickerName(item.name) || /^m_[0-9a-f]{40}$/.test(item.name) ? t('stickers.custom') : item.name),
})
let stickerSuggestFrame = 0

/** 光标移动（输入、点击、方向键）统一经 selectionchange 重检候选；不在编辑区内即关闭 */
function handleSelectionChange() {
  if (!props.stickerPicker || stickerSuggestFrame) return
  stickerSuggestFrame = requestAnimationFrame(() => {
    stickerSuggestFrame = 0
    if (!composing) stickerSuggest.refresh()
  })
}

function handleEditorFocusOut(event: FocusEvent) {
  const next = event.relatedTarget as Node | null
  if (next && root.value?.contains(next)) return
  stickerSuggest.close()
}

/**
 * 贴纸 token 原子删除（issue #1332）：token 是纯文本，默认 Backspace/Delete 会
 * 逐字符拆开留下残缺原文。捕获阶段截获已命中并完成的按键，stopPropagation
 * 让 Vditor 自己的 keydown 逻辑不再按删除后的光标位置二次处理（如在块首合并上一块）。
 */
function handleStickerKeydown(event: KeyboardEvent) {
  const element = currentModeElement()
  // 按键目标必须在编辑区内：listener 挂在 Vditor 根上，toolbar 与 popover 里的输入框也会冒泡到这里
  if (!element || !(event.target instanceof Node) || !element.contains(event.target)) return
  // 候选打开时上下/回车/Tab/Esc 归候选：捕获阶段截住，Vditor 不再换行或缩进
  if (stickerSuggest.handleKeydown(event)) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  const direction = stickerDeleteDirection(event)
  if (!direction) return
  if (!deleteStickerAtCaret(element, direction)) return
  event.preventDefault()
  event.stopPropagation()
}

function handleStickerBeforeInput(event: InputEvent) {
  const element = currentModeElement()
  if (!element || !(event.target instanceof Node) || !element.contains(event.target)) return
  const direction = stickerBeforeInputDirection(event)
  if (!direction || !deleteStickerAtCaret(element, direction)) return
  event.preventDefault()
  event.stopPropagation()
}

// 暴露给浏览器测试 fixture：等待提示协调器静默（无防抖挂起、无在途请求），
// 而不是睡过防抖时常——那会把测试的覆盖窗口和一个文件里的常量隐式耦合。
function hintSettled() {
  return linkPreviewHintController.isSettled()
}

function clearSensitiveHighlights() {
  root.value?.querySelectorAll<HTMLElement>(`.${SENSITIVE_BLOCK_CLASS}`).forEach((element) => {
    element.classList.remove(SENSITIVE_BLOCK_CLASS)
  })
}

function highlightSensitiveText(words: readonly string[]) {
  clearSensitiveHighlights()
  const queries = words.map(word => word.trim()).filter(Boolean)
  if (queries.length === 0 || !editor || !ready || destroyed) return

  const mode = editor.vditor.currentMode
  const editorElement = mode === 'ir'
    ? editor.vditor.ir?.element
    : mode === 'sv'
      ? editor.vditor.sv?.element
      : editor.vditor.wysiwyg?.element
  if (!editorElement) return

  const blocks = new Set<HTMLElement>()
  const walker = document.createTreeWalker(editorElement, NodeFilter.SHOW_TEXT)
  let node: Node | null
  while ((node = walker.nextNode())) {
    const block = node.parentElement?.closest<HTMLElement>(EDITOR_BLOCK_SELECTOR)
    if (block && block !== editorElement && containsSensitiveText(block.textContent ?? '', queries)) {
      blocks.add(block)
    }
  }

  blocks.forEach((block) => block.classList.add(SENSITIVE_BLOCK_CLASS))
  blocks.values().next().value?.scrollIntoView?.({ block: 'nearest' })
}

function attachSensitiveModeRefresh() {
  const editorRoot = root.value
  if (!editorRoot || sensitiveModeRefreshHandler) return
  sensitiveModeRefreshHandler = (event) => {
    const target = event.target
    if (!(target instanceof Element) || !target.closest('button[data-mode]')) return
    requestAnimationFrame(() => {
      if (props.sensitiveWords?.length) highlightSensitiveText(props.sensitiveWords)
    })
  }
  editorRoot.addEventListener('click', sensitiveModeRefreshHandler, true)
  editorRoot.addEventListener('touchstart', sensitiveModeRefreshHandler, true)
}

function detachSensitiveModeRefresh() {
  const editorRoot = root.value
  if (!editorRoot || !sensitiveModeRefreshHandler) return
  editorRoot.removeEventListener('click', sensitiveModeRefreshHandler, true)
  editorRoot.removeEventListener('touchstart', sensitiveModeRefreshHandler, true)
  sensitiveModeRefreshHandler = null
}

/**
 * 长文工具栏（文章，桌面端，issue #1630）：按「写作意图」分组，组内按使用频率排序：
 *   段落样式 | 行内格式 | 段落结构 | 插入内容 | 撤销重做 | 全屏 + 更多
 * - outline 不在工具栏：开关在大纲面板自带按钮（见 syncOutlineToggleHost）
 * - 缩进、分割线、行内公式、上方/下方插入、编辑模式、预览与帮助是低频项，收进 more
 * - 去掉面向开发者或与站点主题重复的官方项：record、devtools、info、export、
 *   code-theme、content-theme（深浅色跟随站点主题）
 * 宿主开启 importMarkdown 时「导入」按钮插在全屏前（见 withImportItem）；
 * 窄窗口仍由折叠收纳把末尾按钮依次移入 more，功能不减。
 */
const OFFICIAL_TOOLBAR: Array<string | IMenuItem> = [
  'headings',
  '|',
  'bold',
  'italic',
  'strike',
  'inline-code',
  'link',
  '|',
  'list',
  'ordered-list',
  'check',
  'quote',
  '|',
  // 插入类工具同组：图片 / 表情（表情包紧随其后注入）/ 表格 / 代码块 / 块级公式
  uploadToolbarItem(),
  'emoji',
  'table',
  'code',
  mathBlockToolbarItem(),
  '|',
  'undo',
  'redo',
  '|',
  'fullscreen',
  {
    name: 'more',
    // 官方默认 tipPosition 为 e（右向气泡），在浮动面板（overflow hidden）里会被裁剪；
    // 统一改为 n（上向气泡），在 vditor 横向边距内显示
    tipPosition: 'n',
    // 菜单按「插入 | 段落 | 编辑模式（就地单选组） | 查看与帮助」分组，见 decorateEditorMenus
    toolbar: [
      'line',
      mathInlineToolbarItem(),
      '|',
      'insert-before',
      'insert-after',
      'outdent',
      'indent',
      '|',
      'edit-mode',
      '|',
      'preview',
      'both',
      'help',
    ],
  },
]

const languageAssets = {
  en: { lang: 'en_US', url: enUrl },
  de: { lang: 'de_DE', url: deUrl },
  ja: { lang: 'ja_JP', url: jaUrl },
  zh: { lang: 'zh_CN', url: zhUrl },
} as const

/** 上传图片按钮：本站附件仅允许图片（Lucide image 图标，tip 覆盖官方「上传图片或文件」） */
function uploadToolbarItem(): IMenuItem {
  return {
    name: 'upload',
    icon: editorToolbarIcon('upload'),
    tip: t('editor.toolbar.uploadImageTip'),
    tipPosition: 'n',
  }
}

/** 站点表情包按钮（MADR 0030，Lucide sticker 图标）：面板由宿主渲染，这里只触发 open-stickers */
function stickerToolbarItem(): IMenuItem {
  return {
    name: 'sticker',
    icon: editorToolbarIcon('sticker'),
    tip: t('editor.toolbar.stickers'),
    tipPosition: 'n',
    click() {
      emit('open-stickers')
    },
  }
}

/**
 * stickerPicker 开启时注入表情包按钮（各预设工具栏均含 emoji 项，插到 emoji 旁）。
 * 桌面放主行 emoji 后（窄窗口由折叠收纳自动收入 more）；移动端主行项数按视口宽度
 * 精调（触控 28px/项），改为收进 more 子菜单 emoji 旁，不挤占主行。
 */
function withStickerItem(list: Array<string | IMenuItem>): Array<string | IMenuItem> {
  if (!props.stickerPicker) return list
  const item = stickerToolbarItem()
  if (!window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) {
    const emojiIndex = list.findIndex((entry) => (typeof entry === 'string' ? entry : entry.name) === 'emoji')
    return emojiIndex === -1 ? [item, ...list] : [...list.slice(0, emojiIndex + 1), item, ...list.slice(emojiIndex + 1)]
  }
  const moreItem = list.find((entry): entry is IMenuItem & { name: 'more' } => typeof entry === 'object' && entry.name === 'more')
  if (!moreItem) return [item, ...list]
  const submenu = moreItem.toolbar ?? []
  const emojiSub = submenu.findIndex((entry) => (typeof entry === 'string' ? entry : entry.name) === 'emoji')
  // 子菜单没有表情项时单独成「插入」组，不混进格式组
  const nextSubmenu = emojiSub === -1 ? [item, '|', ...submenu] : [...submenu.slice(0, emojiSub + 1), item, ...submenu.slice(emojiSub + 1)]
  return list.map((entry) => (entry === moreItem ? { ...moreItem, toolbar: nextSubmenu } : entry))
}

/**
 * 回复工具栏（浮动回复面板、课评等窄容器，桌面端，issue #1630）：
 * 回复是对话，最常用的是表情/表情包、图片与引用，因此「插入」组放在最前；
 * 其后是行内格式与引用/列表/代码，标题、任务列表、表格、公式、全屏等低频项收进 more。
 */
const COMPACT_TOOLBAR: Array<string | IMenuItem> = [
  'emoji',
  uploadToolbarItem(),
  '|',
  'bold',
  'italic',
  'strike',
  'link',
  '|',
  'quote',
  'list',
  'ordered-list',
  'inline-code',
  'code',
  '|',
  'undo',
  'redo',
  {
    name: 'more',
    tipPosition: 'n',
    toolbar: [
      'headings',
      '|',
      'check',
      'table',
      'line',
      ...mathToolbarItems(),
      '|',
      'outdent',
      'indent',
      '|',
      'preview',
      'fullscreen',
      'help',
    ],
  },
]

/**
 * 移动端文章专属工具栏（Article Mobile Toolbar）：
 * 单行流线型设计（9 项核心排版高频工具 + 1 项统一收纳门），彻底杜绝双行折行臃肿，
 * 主行一键直达：标题、加粗、斜体、引用、列表、代码块、图片、链接；
 * 更多菜单优雅收纳：表格、表情、撤销/重做、删除线、有序/任务列表、行内代码、分割线、公式、全屏与预览。
 */
const ARTICLE_MOBILE_TOOLBAR: Array<string | IMenuItem> = [
  'headings',
  'bold',
  'italic',
  'quote',
  'list',
  'code',
  uploadToolbarItem(),
  'link',
  {
    name: 'more',
    tipPosition: 'n',
    toolbar: [
      'emoji',
      '|',
      'strike',
      'inline-code',
      'ordered-list',
      'check',
      '|',
      'table',
      'line',
      ...mathToolbarItems(),
      '|',
      'undo',
      'redo',
      '|',
      'preview',
      'fullscreen',
      'help',
    ],
  },
]

/**
 * 移动端回复/楼层评论专属工具栏（Compact Mobile Toolbar）：
 * 专为社区楼层回复与短评定制：聚焦最常用的高频互动与轻量排版能力，
 * 彻底消除左偏空旷（整体居中均分平衡），交互直观纯粹（去除非直观的 edit-mode / insert-after）。
 */
const COMPACT_MOBILE_TOOLBAR: Array<string | IMenuItem> = [
  'emoji',
  uploadToolbarItem(),
  'quote',
  'bold',
  'code',
  'list',
  'link',
  'undo',
  'redo',
  {
    name: 'more',
    tipPosition: 'n',
    toolbar: [
      'italic',
      'strike',
      'inline-code',
      '|',
      'ordered-list',
      'check',
      '|',
      'table',
      'line',
      '|',
      'fullscreen',
      'help',
    ],
  },
]

/**
 * 移动端短文/极简专属工具栏（Simple Mobile Toolbar - 瞬间与提问）：
 * 极简单行，高频纯粹，包含图片、表情、加粗、斜体、列表、链接、撤销与重做。
 */
const SIMPLE_MOBILE_TOOLBAR: Array<string | IMenuItem> = [
  uploadToolbarItem(),
  'emoji',
  'bold',
  'italic',
  'list',
  'link',
  'undo',
  'redo',
]

/**
 * 移动端窄容器精简工具栏（Slim Mobile Toolbar - 课评表单等 vw-64 嵌套容器）：
 * 在 compact 移动端（10 项，需 ≥330px 容器）基础上进一步收敛为主行 7 项，
 * 7×32+5=229px，≤320px 视口也单行完整不裁剪；标题放左一（与官方默认工具栏一致，
 * 课评常用分级标题），表情/图片/加粗/引用/列表直达主行，代码块等低频项收进 more，
 * 与 compact 同一「高频直达 + 低频收纳」交互模型。
 */
const SLIM_MOBILE_TOOLBAR: Array<string | IMenuItem> = [
  'headings',
  'emoji',
  uploadToolbarItem(),
  'bold',
  'quote',
  'list',
  {
    name: 'more',
    tipPosition: 'n',
    toolbar: [
      'italic',
      'strike',
      'inline-code',
      'link',
      '|',
      'ordered-list',
      'check',
      'code',
      '|',
      'table',
      'line',
      '|',
      'undo',
      'redo',
      '|',
      'fullscreen',
      'help',
    ],
  },
]

/**
 * 桌面端窄容器精简工具栏（slimMobile 桌面分支，课评编辑器专用）：
 * 与 COMPACT_TOOLBAR 同构（不影响回复浮窗等 compact 宿主），仅两处差异：
 * 标题提升到主行左一（对齐官方默认工具栏与移动端 slim），并从 more 子菜单移除避免重复。
 */
const SLIM_DESKTOP_TOOLBAR: Array<string | IMenuItem> = (() => {
  const compact = COMPACT_TOOLBAR.map((item) =>
    typeof item === 'object' && item.name === 'more'
      ? { ...item, toolbar: item.toolbar?.filter((name, index) => name !== 'headings' && !(index === 1 && name === '|')) }
      : item,
  )
  return ['headings', ...compact]
})()

/**
 * 短文工具栏（瞬间、提问、普通帖与快捷发布，桌面端，参考推特/知乎输入卡片，issue #1630）：
 * 短文以图文与表情为主，「插入」组放最前；只保留基础行内格式与列表/引用/代码，
 * 不提供大纲、公式、表格与多级菜单。
 */
const SIMPLE_TOOLBAR: Array<string | IMenuItem> = [
  uploadToolbarItem(),
  'emoji',
  '|',
  'bold',
  'italic',
  'strike',
  'link',
  '|',
  'list',
  'ordered-list',
  'quote',
  'inline-code',
  'code',
  '|',
  'undo',
  'redo',
]

/** 移动端判定：与 vditor Constants.MOBILE_WIDTH 一致 */
const MOBILE_VIEWPORT_QUERY = '(max-width: 520px)'

/** 官方建议：移动端高度 = 屏高一半，防止软键盘弹起时布局跳动 */
function resolveHeight(): number | string {
  if (typeof props.height === 'string') return props.height
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) {
    if (props.simple || props.compact) return props.height ?? 280
    return Math.floor(window.innerHeight / 2)
  }
  return props.height ?? 360
}

/** 工具栏选择：
 *  移动端优先：simple -> SIMPLE_MOBILE_TOOLBAR；slimMobile -> SLIM_MOBILE_TOOLBAR（窄容器）；
 *  compact -> COMPACT_MOBILE_TOOLBAR；默认 -> ARTICLE_MOBILE_TOOLBAR
 *  桌面端：simple -> SIMPLE_TOOLBAR；slimMobile -> SLIM_DESKTOP_TOOLBAR（compact 同构 + 标题左一）；
 *  compact -> COMPACT_TOOLBAR；默认 -> OFFICIAL_TOOLBAR
 */
function resolveToolbar(): Array<string | IMenuItem> {
  let list: Array<string | IMenuItem>
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) {
    if (props.simple) {
      list = [...SIMPLE_MOBILE_TOOLBAR]
    } else if (props.slimMobile) {
      list = [...SLIM_MOBILE_TOOLBAR]
    } else if (props.compact) {
      list = [...COMPACT_MOBILE_TOOLBAR]
    } else {
      list = [...ARTICLE_MOBILE_TOOLBAR]
    }
  } else if (props.simple) {
    list = [...SIMPLE_TOOLBAR]
  } else if (props.slimMobile) {
    list = [...SLIM_DESKTOP_TOOLBAR]
  } else {
    list = props.compact ? [...COMPACT_TOOLBAR] : [...OFFICIAL_TOOLBAR]
  }

  if (props.hideUpload) {
    list = list.filter((item) => {
      if (typeof item === 'string') return item !== 'upload'
      return (item as IMenuItem).name !== 'upload'
    })
  }
  return withLucideIcons(withImportItem(withStickerItem(list)))
}

/** Markdown 导入按钮（issue #1630）：解析与上传由宿主的导入弹窗负责，这里只发出 import-markdown。 */
function importToolbarItem(): IMenuItem {
  return {
    name: 'import-markdown',
    icon: editorToolbarIcon('import-markdown'),
    tip: t('editor.toolbar.importMarkdownTip'),
    tipPosition: 'n',
    click() {
      emit('import-markdown')
    },
  }
}

/**
 * importMarkdown 开启时注入导入按钮：桌面放在全屏前（与「导出/全屏」这类文档级操作同组），
 * 移动端主行项数按视口宽度精调，收进 more 子菜单首位。
 */
function withImportItem(list: Array<string | IMenuItem>): Array<string | IMenuItem> {
  if (!props.importMarkdown) return list
  const item = importToolbarItem()
  const nameOf = (entry: string | IMenuItem) => (typeof entry === 'string' ? entry : entry.name)
  const moreIndex = list.findIndex((entry) => nameOf(entry) === 'more')
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches && moreIndex !== -1) {
    const moreItem = list[moreIndex] as IMenuItem
    return list.map((entry, index) => (index === moreIndex ? { ...moreItem, toolbar: [item, ...(moreItem.toolbar ?? [])] } : entry))
  }
  const anchor = list.findIndex((entry) => nameOf(entry) === 'fullscreen')
  const at = anchor !== -1 ? anchor : moreIndex !== -1 ? moreIndex : list.length
  return [...list.slice(0, at), item, ...list.slice(at)]
}

/** 主行内置项统一换成 Lucide 图标（Vditor 合并工具栏时 Object.assign 保留传入的 icon）。 */
function withLucideIcons(list: Array<string | IMenuItem>): Array<string | IMenuItem> {
  return list.map((entry) => {
    const name = typeof entry === 'string' ? entry : entry.name
    if (!hasEditorToolbarIcon(name)) return entry
    if (typeof entry === 'string') return { name, icon: editorToolbarIcon(name) }
    return entry.icon ? entry : { ...entry, icon: editorToolbarIcon(name) }
  })
}

/** 行间公式 / 块级公式：自定义按钮，插入 $...$ / $$...$$ 并交给 Lute 渲染。
 *  图标用全站 lucide 图标族（Sigma / SquareSigma，见 editor-toolbar-icons）。 */
function mathToolbarItems(): IMenuItem[] {
  return [mathInlineToolbarItem(), mathBlockToolbarItem()]
}

function mathInlineToolbarItem(): IMenuItem {
  return {
    name: 'math-inline',
    icon: editorToolbarIcon('math-inline'),
    tip: t('editor.toolbar.mathInline'),
    click() {
      if (!editor || !ready) return
      const selected = editor.getSelection()
      const math = selected
        ? '$' + selected + '$'
        : '$' + t('publish.placeholder.math') + '$'
      editor.focus()
      editor.insertMD(math)
      emit('update:modelValue', editor.getValue())
    },
  }
}

function mathBlockToolbarItem(): IMenuItem {
  return {
    name: 'math-block',
    icon: editorToolbarIcon('math-block'),
    tip: t('editor.toolbar.mathBlock'),
    click() {
      if (!editor || !ready) return
      const selected = editor.getSelection()
      const math = selected
        ? '$$\n' + selected + '\n$$'
        : '$$\n' + t('publish.placeholder.math') + '\n$$'
      editor.focus()
      editor.insertMD(math)
      emit('update:modelValue', editor.getValue())
    },
  }
}

/**
 * 正文 content-theme：官方 index.css 把 .vditor-reset 写死为 #24292e，
 * 深色正文色只在 content-theme/dark.css 里覆盖。
 * 本组件 cdn/path 为空（离线），setContentTheme 不会加载样式，
 * 这里按站点深浅色自行切换 light/dark 主题表（与 hljs 同套路）。
 */
const CONTENT_THEME_LINK_ID = 'hubVditorContentTheme'
function syncContentTheme() {
  const themeName = isDark.value ? 'dark' : 'light'
  const existing = document.getElementById(CONTENT_THEME_LINK_ID) as HTMLLinkElement | null
  if (existing && existing.dataset.theme === themeName) return
  existing?.remove()
  const link = document.createElement('link')
  link.id = CONTENT_THEME_LINK_ID
  link.rel = 'stylesheet'
  link.dataset.theme = themeName
  link.href = isDark.value ? contentThemeDarkCssUrl : contentThemeLightCssUrl
  document.head.appendChild(link)
}

/** 高亮主题样式：与站点深浅色联动，替换 link 实现（不依赖 vditor 内置主题切换） */
const HIGHLIGHT_THEME_LINK_ID = 'hubVditorHljsTheme'
function syncHighlightTheme() {
  const existing = document.getElementById(HIGHLIGHT_THEME_LINK_ID) as HTMLLinkElement | null
  if (existing && existing.dataset.theme === (isDark.value ? 'dark' : 'light')) return
  existing?.remove()
  const link = document.createElement('link')
  link.id = HIGHLIGHT_THEME_LINK_ID
  link.rel = 'stylesheet'
  link.dataset.theme = isDark.value ? 'dark' : 'light'
  link.href = isDark.value ? hljsDarkCssUrl : hljsLightCssUrl
  document.head.appendChild(link)
}

/** 知乎式图标+小字：2-3 字短标签（对齐知乎工具栏文案），避免长词撑宽/换行；
 *   hover tooltip 保留官方完整「功能 + 快捷键」；
 *   文案走 i18n（editor.toolbar.*，见 locales/{zh,en,ja,de}.ts） */
const TOOLBAR_LABEL_KEYS: Record<string, string> = {
  headings: 'editor.toolbar.headings',
  bold: 'editor.toolbar.bold',
  italic: 'editor.toolbar.italic',
  strike: 'editor.toolbar.strike',
  link: 'editor.toolbar.link',
  list: 'editor.toolbar.list',
  'ordered-list': 'editor.toolbar.orderedList',
  check: 'editor.toolbar.check',
  outdent: 'editor.toolbar.outdent',
  indent: 'editor.toolbar.indent',
  quote: 'editor.toolbar.quote',
  line: 'editor.toolbar.line',
  code: 'editor.toolbar.code',
  'inline-code': 'editor.toolbar.inlineCode',
  'math-inline': 'editor.toolbar.mathInline',
  'math-block': 'editor.toolbar.mathBlock',
  'insert-before': 'editor.toolbar.insertBefore',
  'insert-after': 'editor.toolbar.insertAfter',
  emoji: 'editor.toolbar.emoji',
  sticker: 'editor.toolbar.stickers',
  'import-markdown': 'editor.toolbar.importMarkdown',
  upload: 'editor.toolbar.upload',
  table: 'editor.toolbar.table',
  undo: 'editor.toolbar.undo',
  redo: 'editor.toolbar.redo',
  fullscreen: 'editor.toolbar.fullscreen',
  'edit-mode': 'editor.toolbar.editMode',
  more: 'editor.toolbar.more',
  preview: 'editor.toolbar.preview',
  both: 'editor.toolbar.both',
  export: 'editor.toolbar.export',
  info: 'editor.toolbar.info',
  help: 'editor.toolbar.help',
}
const HOTKEY_PATTERN = /\s*<([^<>]+)>\s*$/

/** 工具栏按钮短标签：i18n 优先，缺失时回落 vditor 自带 i18n */
function toolbarLabel(type: string): string {
  const key = TOOLBAR_LABEL_KEYS[type]
  if (key && te(key)) return t(key)
  return window.VditorI18n?.[type] || ''
}

/** 「更多」菜单行与折叠项用完整名称（主行图标下的 2-3 字短标签在菜单里含义不清） */
const MENU_LABEL_KEYS: Record<string, string> = {
  headings: 'editor.menu.headings',
  bold: 'editor.menu.bold',
  italic: 'editor.menu.italic',
  strike: 'editor.menu.strike',
  link: 'editor.menu.link',
  list: 'editor.menu.list',
  'ordered-list': 'editor.menu.orderedList',
  check: 'editor.menu.check',
  outdent: 'editor.menu.outdent',
  indent: 'editor.menu.indent',
  quote: 'editor.menu.quote',
  line: 'editor.menu.line',
  code: 'editor.menu.code',
  'inline-code': 'editor.menu.inlineCode',
  'math-inline': 'editor.menu.mathInline',
  'math-block': 'editor.menu.mathBlock',
  'insert-before': 'editor.menu.insertBefore',
  'insert-after': 'editor.menu.insertAfter',
  emoji: 'editor.menu.emoji',
  sticker: 'editor.menu.stickers',
  'import-markdown': 'editor.menu.importMarkdown',
  upload: 'editor.menu.upload',
  table: 'editor.menu.table',
  undo: 'editor.menu.undo',
  redo: 'editor.menu.redo',
  fullscreen: 'editor.menu.fullscreen',
  'edit-mode': 'editor.menu.editMode',
  preview: 'editor.menu.preview',
  both: 'editor.menu.both',
  help: 'editor.menu.help',
}
const EDIT_MODE_MENU_KEYS: Record<string, string> = {
  wysiwyg: 'editor.menu.wysiwyg',
  ir: 'editor.menu.ir',
  sv: 'editor.menu.sv',
}

function menuLabel(type: string): string {
  const key = MENU_LABEL_KEYS[type]
  return key && te(key) ? t(key) : toolbarLabel(type)
}

/**
 * 在工具栏按钮内部补/保「图标下小字」（不挪到按钮外）。
 * 全屏等会 innerHTML 整按钮，需可重复调用。
 */
function ensureToolbarButtonLabel(item: HTMLElement) {
  const type = item.getAttribute('data-type') || ''
  // 同一节点会在主行与「更多」菜单（折叠收纳）之间移动，按当前位置取短/全称
  const inMenu = item.closest('.vditor-hint') !== null
  const labelText =
    (inMenu ? menuLabel(type) : toolbarLabel(type)) ||
    (item.getAttribute('aria-label') || '').replace(HOTKEY_PATTERN, '').trim()
  if (!labelText) return

  const existing = item.querySelector<HTMLElement>(':scope > .vditor-toolbar-label')
  if (existing) {
    if (existing.textContent !== labelText) existing.textContent = labelText
    return
  }

  const span = document.createElement('span')
  span.className = 'vditor-toolbar-label'
  span.textContent = labelText
  item.appendChild(span)
}

/**
 * 「更多」菜单（issue #1630）：对齐站点下拉菜单（gf-menu-surface / gf-menu-item）。
 * Vditor 的二级项是纯文本「名称 <快捷键>」，监听器直接绑在按钮节点上，因此只就地改写
 * 子节点：图标 + 名称 + 右侧快捷键，不重建按钮。编辑模式与标题这类单选组的触发按钮
 * 改作组标题、子面板就地展开（CSS），不再弹出二级浮层。标题面板按钮的监听器读
 * event.target 的 data-tag，所以不往里加子节点，「H1」等文字由 CSS 生成。
 * 可重复调用：语言热切换后再跑一次即刷新文案。
 */
const MENU_GROUP_TYPES = new Set(['edit-mode', 'headings'])

function takeHotkeyText(button: HTMLElement): string {
  let hotkey = ''
  for (const node of Array.from(button.childNodes)) {
    if (node.nodeType !== Node.TEXT_NODE) continue
    hotkey = node.nodeValue?.match(HOTKEY_PATTERN)?.[1] || hotkey
    node.remove()
  }
  return hotkey
}

function appendMenuHotkey(button: HTMLElement, hotkey: string) {
  if (!hotkey || button.querySelector(':scope > .gf-menu-kbd')) return
  const kbd = document.createElement('kbd')
  kbd.className = 'gf-menu-kbd'
  kbd.textContent = hotkey
  button.appendChild(kbd)
}

/** 组标题只是说明文字：吞掉点击，避免切换已就地展开的子面板、关闭菜单或移除当前标题 */
function swallowMenuEvent(event: Event) {
  event.preventDefault()
  event.stopImmediatePropagation()
}

function decorateMenuRow(button: HTMLElement) {
  const type = button.getAttribute('data-type') || ''
  if (!button.classList.contains('gf-menu-row')) {
    const hotkey = takeHotkeyText(button)
    if (!button.querySelector(':scope > svg') && hasEditorToolbarIcon(type)) {
      button.insertAdjacentHTML('afterbegin', editorToolbarIcon(type))
    }
    button.classList.add('gf-menu-row')
    if (MENU_GROUP_TYPES.has(type)) {
      button.classList.add('gf-menu-group-title')
      button.tabIndex = -1
      button.addEventListener('click', swallowMenuEvent, true)
      button.addEventListener('touchstart', swallowMenuEvent, { capture: true, passive: false })
    }
    ensureToolbarButtonLabel(button)
    appendMenuHotkey(button, hotkey)
    return
  }
  ensureToolbarButtonLabel(button)
}

function decorateModeOption(button: HTMLElement) {
  const mode = button.getAttribute('data-mode') || ''
  const key = EDIT_MODE_MENU_KEYS[mode]
  const text = key && te(key) ? t(key) : (button.textContent || '').replace(HOTKEY_PATTERN, '').trim()
  let label = button.querySelector<HTMLElement>(':scope > .vditor-toolbar-label')
  if (!label) {
    const hotkey = takeHotkeyText(button)
    button.classList.add('gf-menu-row', 'gf-menu-option')
    button.insertAdjacentHTML('afterbegin', editorToolbarIcon('selected'))
    label = document.createElement('span')
    label.className = 'vditor-toolbar-label'
    button.appendChild(label)
    appendMenuHotkey(button, hotkey)
  }
  if (label.textContent !== text) label.textContent = text
}

function decorateEditorMenus() {
  const toolbar = root.value?.querySelector<HTMLElement>('.vditor-toolbar')
  const menu = toolbar
    ?.querySelector<HTMLElement>(':scope > .vditor-toolbar__item > [data-type="more"]')
    ?.parentElement?.querySelector<HTMLElement>(':scope > .vditor-hint')
  if (!menu) return
  menu.classList.add('gf-editor-menu')
  menu.querySelectorAll<HTMLElement>(':scope > div > [data-type]').forEach(decorateMenuRow)
  menu.querySelectorAll<HTMLElement>('button[data-mode]').forEach(decorateModeOption)
  menu.querySelectorAll<HTMLElement>('button[data-tag]').forEach((button) => {
    button.title = (button.textContent || '').trim()
  })
}

/**
 * 官方 Fullscreen 进入/退出时执行：
 *   this.innerHTML = '<svg>…' / menuItem.icon
 * 会清掉我们塞在 button 内的小字。监听 childList，仍在按钮内补回（不外置）。
 */
function guardFullscreenToolbarLabel() {
  const button = root.value?.querySelector<HTMLElement>(
    '.vditor-toolbar__item > [data-type="fullscreen"]',
  )
  if (!button) return

  fullscreenLabelObserver?.disconnect()
  fullscreenLabelObserver = new MutationObserver(() => {
    if (destroyed || !button.isConnected) {
      fullscreenLabelObserver?.disconnect()
      fullscreenLabelObserver = null
      return
    }
    // 进入全屏时官方把图标换回 ant 的 contract：改成同族 Lucide 图标（退出时官方恢复传入的 icon）
    const officialIcon = button.querySelector('svg use')?.closest('svg')
    if (officialIcon) officialIcon.outerHTML = editorToolbarIcon('contract')
    ensureToolbarButtonLabel(button)
  })
  fullscreenLabelObserver.observe(button, { childList: true })
}

function attachToolbarLabels() {
  // 移动端保持官方 sweet-mobile 纯图标工具栏（44px 触控高度），不注入文字标签
  if (!root.value || window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) return
  root.value.querySelectorAll<HTMLElement>('.vditor-toolbar__item .vditor-tooltipped').forEach((item) => {
    ensureToolbarButtonLabel(item)
  })
  // more 子菜单：vditor 的 Custom 类会把自定义项（公式等）强制渲染成纯图标、
  // 覆盖 level-2 的文字 tip；这里给含 svg 的项补上文字标签，保持菜单项可读
  root.value.querySelectorAll<HTMLElement>('.vditor-hint button[data-type]').forEach((item) => {
    if (!item.querySelector('svg') || item.querySelector('.vditor-toolbar-label')) return
    ensureToolbarButtonLabel(item)
  })
  guardFullscreenToolbarLabel()
  attachTooltipSmartPosition()
}

/**
 * hover 气泡智能避让：气泡在按钮下方居中（left 50% + translateX(-50%)），
 * 但前几个按钮的气泡会向左超出最近裁剪容器（回复面板滚动容器等）被裁掉。
 * 这里在 hover 时用 canvas 精确测量气泡宽度，若越界则通过 CSS 变量
 * --hub-tip-shift 平移气泡，保证完整显示在容器内（发布页无裁剪容器，
 * 居中即可，左侧大纲区域由工具栏 z-index 100 覆盖）。
 */
function attachTooltipSmartPosition() {
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  const measureTipWidth = (text: string) => {
    if (!ctx) return text.length * 7 + 20
    ctx.font = '11px sans-serif'
    return Math.ceil(ctx.measureText(text).width) + 16 // 官方气泡 padding 5px 8px
  }
  root.value?.querySelectorAll<HTMLElement>('.vditor-toolbar__item .vditor-tooltipped').forEach((btn) => {
    btn.addEventListener('mouseenter', () => {
      const tipWidth = measureTipWidth(btn.getAttribute('aria-label') || '')
      const rect = btn.getBoundingClientRect()
      const center = rect.left + rect.width / 2
      let tipLeft = center - tipWidth / 2
      let tipRight = center + tipWidth / 2
      // 找最近的横向裁剪容器（overflow 非 visible）
      let container: HTMLElement | null = btn.parentElement
      while (container && container !== document.body) {
        const cs = getComputedStyle(container)
        if (cs.overflowX === 'hidden' || cs.overflowX === 'auto' || cs.overflowX === 'scroll' || cs.clipPath !== 'none') {
          break
        }
        container = container.parentElement
      }
      if (container) {
        const cr = container.getBoundingClientRect()
        let shift = 0
        if (tipLeft < cr.left) shift = cr.left - tipLeft
        else if (tipRight > cr.right) shift = cr.right - tipRight
        btn.style.setProperty('--hub-tip-shift', `${shift}px`)
      } else {
        btn.style.removeProperty('--hub-tip-shift')
      }
    })
  })
}

// ===== 语言热切换：替换 window.VditorI18n 后就地刷新文案 DOM =====
// vditor 无 setI18n API（src/index.ts 公开方法仅 updateToolbarConfig/setTheme/getValue/…），
// 全部界面文案在构造期由 window.VditorI18n 生成；切换 = 重载目标语言脚本 → 就地改
// 文本节点/属性（绝不能 innerHTML 重建带事件监听的按钮/面板）。

/** 主行按钮 tooltip（aria-label）文案：宿主自定义项走 t()（与构造期 tip 同源），
 *  其余回落 vditor 原生 i18n。上传必须用 uploadImageTip 完整 tooltip，
 *  不能回落 I.upload（vditor 默认「上传图片或文件」会覆盖宿主定制文案）。 */
const TOOLBAR_TIP_KEYS: Record<string, string> = {
  upload: 'editor.toolbar.uploadImageTip',
  'math-inline': 'editor.toolbar.mathInline',
  'math-block': 'editor.toolbar.mathBlock',
  sticker: 'editor.toolbar.stickers',
  'import-markdown': 'editor.toolbar.importMarkdownTip',
}

/** 语言热切换竞态令牌：快速连续切换时只应用最后一次 */
let i18nSeq = 0
/** 初始化期间（编辑器未 ready）切换语言时置位，after() 就绪后补刷，避免切换被静默丢弃 */
let pendingLocaleRefresh = false

/** 重载目标语言 i18n 脚本。loadRuntimeScript（runtime-script.ts:6）对
 *  dataset.loaded='true' 的节点直接 resolve 不重跑 → 切回已加载语言时
 *  window.VditorI18n 会停在上一个语言，必须先移除节点强制重跑。 */
async function loadVditorI18n(lang: string, url: string) {
  const id = `vditorI18nScript${lang}`
  document.getElementById(id)?.remove()
  await loadRuntimeScript(url, id)
}

/** 空态占位符：wysiwyg/ir/sv 的 .element 都是 pre.vditor-reset（HTMLPreElement），
 *  CSS 用 content:attr(placeholder) 显示，直接改属性即时生效。 */
function refreshPlaceholder() {
  if (!editor || !ready || destroyed) return
  const text = props.placeholder
  editor.vditor.options.placeholder = text
  editor.vditor.wysiwyg?.element.setAttribute('placeholder', text)
  editor.vditor.ir?.element.setAttribute('placeholder', text)
  editor.vditor.sv?.element.setAttribute('placeholder', text)
}

/** 主行按钮 hover tooltip：重建 aria-label，保留原热键后缀（跨语言不变）。 */
function refreshMainRowAriaLabels(I: typeof window.VditorI18n) {
  root.value
    ?.querySelectorAll<HTMLElement>('.vditor-toolbar__item > .vditor-tooltipped[data-type]')
    .forEach((btn) => {
      const type = btn.getAttribute('data-type') || ''
      const suffix = (btn.getAttribute('aria-label') || '').match(HOTKEY_PATTERN)?.[0] || ''
      const tipKey = TOOLBAR_TIP_KEYS[type]
      const tip = tipKey ? (te(tipKey) ? t(tipKey) : I[type] || '') : I[type] || ''
      btn.setAttribute('aria-label', tip + suffix)
    })
}

/** more 子菜单原生 level-2 项：改首文本节点保留事件监听（MenuItem.ts:23-24 innerHTML 生成）。
 *  必须用 `type in window.VditorI18n` 排除 content-theme/export 面板——它们的按钮
 *  data-type 是主题/格式键（dark/light/markdown/pdf/…），不在 vditor i18n 里，否则会把
 *  面板文字刷成空串。 */
function refreshHintNativeItems(I: typeof window.VditorI18n) {
  root.value
    ?.querySelectorAll<HTMLElement>('.vditor-hint button[data-type]')
    .forEach((btn) => {
      if (btn.querySelector('svg')) return // 宿主自定义图标项（公式等）走 refreshToolbarLabelText
      const type = btn.getAttribute('data-type') || ''
      if (!type || !(type in window.VditorI18n)) return
      const suffix = (btn.textContent || '').match(/\s*<[^<>]*>\s*$/)?.[0] || ''
      if (btn.firstChild?.nodeType === Node.TEXT_NODE) {
        btn.firstChild.nodeValue = (I[type] || '') + suffix
      }
    })
}

/** Headings 面板：click 事件绑在 button 上（Headings.ts:56-71），只改首文本节点。 */
const HEADING_KEYS = ['heading1', 'heading2', 'heading3', 'heading4', 'heading5', 'heading6']
function refreshHeadingsPanel(I: typeof window.VditorI18n) {
  root.value
    ?.querySelectorAll<HTMLElement>('.vditor-hint button[data-tag]')
    .forEach((btn, i) => {
      const suffix = (btn.textContent || '').match(/\s*<[^<>]*>\s*$/)?.[0] || ''
      if (btn.firstChild?.nodeType === Node.TEXT_NODE) {
        btn.firstChild.nodeValue = (I[HEADING_KEYS[i]] || '') + suffix
      }
    })
}

/** EditMode 面板：同规则改首文本节点。 */
const EDIT_MODE_KEYS: Record<string, string> = {
  wysiwyg: 'wysiwyg',
  ir: 'instantRendering',
  sv: 'splitView',
}
function refreshEditModePanel(I: typeof window.VditorI18n) {
  root.value
    ?.querySelectorAll<HTMLElement>('.vditor-hint button[data-mode]')
    .forEach((btn) => {
      const key = EDIT_MODE_KEYS[btn.getAttribute('data-mode') || '']
      if (!key) return
      const suffix = (btn.textContent || '').match(/\s*<[^<>]*>\s*$/)?.[0] || ''
      if (btn.firstChild?.nodeType === Node.TEXT_NODE) {
        btn.firstChild.nodeValue = (I[key] || '') + suffix
      }
    })
}

/** 大纲标题 + 开关按钮。标题只改首文本节点：宿主把开关按钮 append 进 title
 *  （syncOutlineToggleHost L778），textContent 会删掉按钮。开关按钮只查已存在的
 *  aria-label，绝不调用 outlineToggleButton()/headerToggleButton()——它们缺省会新建
 *  按钮，非 outline/headerToggle 模式的编辑器会残留悬浮按钮。 */
function refreshOutlineAndToggles(I: typeof window.VditorI18n) {
  const title = editor?.vditor.outline.element.querySelector<HTMLElement>('.vditor-outline__title')
  if (title?.firstChild?.nodeType === Node.TEXT_NODE) title.firstChild.nodeValue = I.outline || ''
  const scopes = [root.value, props.toggleHost]
  for (const scope of scopes) {
    scope
      ?.querySelector<HTMLElement>(`.${OUTLINE_TOGGLE_CLASS}`)
      ?.setAttribute('aria-label', I.outline || '大纲')
    scope
      ?.querySelector<HTMLElement>(`.${HEADER_TOGGLE_CLASS}`)
      ?.setAttribute('aria-label', t('publish.collapseHeader'))
  }
}

/** 图标下小字就地刷新：只重跑 ensureToolbarButtonLabel（L360 文本 diff 更新）。
 *  不能复用 attachToolbarLabels（L393-407）——它内部 attachTooltipSmartPosition 会重绑
 *  mouseenter、guardFullscreenToolbarLabel 会重建 observer，多调几次叠监听器。 */
function refreshToolbarLabelText() {
  if (!root.value) return
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) {
    // Keep the main row icon-only while refreshing every visible folded label.
    // Reusing the nodes also preserves their native Vditor listeners.
    root.value.querySelectorAll<HTMLElement>('.hub-fold-region .vditor-tooltipped')
      .forEach(ensureToolbarButtonLabel)
    return
  }
  root.value
    .querySelectorAll<HTMLElement>('.vditor-toolbar__item .vditor-tooltipped')
    .forEach(ensureToolbarButtonLabel)
  root.value
    .querySelectorAll<HTMLElement>('.vditor-hint button[data-type]')
    .forEach((item) => {
      if (item.querySelector('svg')) ensureToolbarButtonLabel(item)
    })
}

/** 语言切换主流程。 */
async function refreshVditorLocale() {
  if (!editor || !ready || destroyed) return
  // 局部 const 捕获当前实例：跨 await 时模块级 let editor 的收窄会被重置，
  // 用 const 保证 await 后续体类型安全；卸载由 destroyed + i18nSeq 双重拦截。
  const current = editor
  const newLocale = currentLocale()
  const asset = languageAssets[newLocale]
  if (!asset) return
  const seq = ++i18nSeq
  try {
    await loadVditorI18n(asset.lang, asset.url)
  } catch {
    return // 脚本加载失败：保留旧语言 DOM，下次切换/重载再试
  }
  if (destroyed || seq !== i18nSeq || !current.vditor) return // 快速切换竞态：放弃陈旧调用
  const I = window.VditorI18n
  current.vditor.options.i18n = I // 与 previewRender.ts:117 保持一致（preview 渲染期读 window.VditorI18n）
  current.vditor.options.lang = asset.lang
  refreshPlaceholder()
  refreshMainRowAriaLabels(I)
  refreshHintNativeItems(I)
  refreshHeadingsPanel(I)
  refreshEditModePanel(I)
  refreshOutlineAndToggles(I)
  refreshToolbarLabelText()
  decorateEditorMenus()
  scheduleMeasure() // 英文文案变宽可能改变折叠收纳 / 图标-only 判定
}

// ===== 工具栏图标窄屏收纳：溢出的按钮按原次序移入 more 子菜单 =====
// 方案依据：vditor 无事件委托，监听器直接绑在按钮节点上（Custom.ts/MenuItem.ts/Upload.ts 等），
// 因此收纳只能「移动整层 .vditor-toolbar__item 包裹 div」进子菜单（节点移动保留监听器与
// vditor.toolbar.elements[name] 引用），绝不能 display:none 原地隐藏或 clone 重建。
const OVERFLOW_TOLERANCE = 1 // px：折叠/恢复判定滞回，防抖动
/** 工具栏容器宽度低于该值时隐藏主行按钮的文字标签（只显示图标）：
 *  窄屏下英文/多字标签（如 "Block math"）会溢出按钮与相邻图标重叠。
 *  带滞回：低于 HIDE 才隐藏，需宽到 SHOW 才恢复（页面左边栏折叠会让工具栏宽度
 *  非单调变化，无滞回会反复跨越阈值导致标签先消失又出现）。
 *  #7：一旦触发过隐藏即锁存，之后宽度恢复也不再显示（防止侧栏消失→展开闪现）。 */
const LABEL_HIDE_THRESHOLD = 800
const LABEL_SHOW_THRESHOLD = 900
let labelsHidden = false
let labelsLatched = false // #7：一旦隐藏即永久锁存（直到模块重置），不再恢复文字标签
/** 带二级面板的项不收纳：它们的 toggleSubMenu 的 exceptElement 逻辑在折叠区内
 *  会把整个 more 面板关掉（EditMode.ts:172 / Emoji.ts:37 / Headings.ts 证据） */
const HUB_NO_FOLD_TYPES = new Set(['emoji', 'headings', 'edit-mode'])
let foldObserver: ResizeObserver | null = null
let foldRaf = 0
let toolbarPanelResizeHandler: (() => void) | null = null
let toolbarPanelClampRaf = 0
let toolbarEl: HTMLElement | null = null
let moreWrapper: HTMLElement | null = null
let hintPanel: HTMLElement | null = null
let foldRegion: HTMLElement | null = null
let mainRowSequence: HTMLElement[] = []

function isFolded(el: HTMLElement) {
  return el.parentElement === foldRegion
}

function syncDivider(d: HTMLElement) {
  const i = mainRowSequence.indexOf(d)
  const left = mainRowSequence[i - 1]
  const right = mainRowSequence[i + 1]
  const leftFolded = left ? isFolded(left) : false
  const rightFolded = right ? isFolded(right) : false
  d.style.display = leftFolded && rightFolded ? 'none' : ''
}

function syncAdjacentDividers(item: HTMLElement) {
  const i = mainRowSequence.indexOf(item)
  const left = mainRowSequence[i - 1]
  const right = mainRowSequence[i + 1]
  if (left?.classList.contains('vditor-toolbar__divider')) syncDivider(left)
  if (right?.classList.contains('vditor-toolbar__divider')) syncDivider(right)
}

function rightmostMainRowItem(): HTMLElement | null {
  for (let j = toolbarEl!.children.length - 1; j >= 0; j--) {
    const c = toolbarEl!.children[j] as HTMLElement
    if (c === moreWrapper || c.classList.contains('vditor-counter')) continue
    if (!c.classList.contains('vditor-toolbar__item')) continue
    const type = c.children[0]?.getAttribute('data-type') ?? ''
    if (HUB_NO_FOLD_TYPES.has(type)) continue
    return c
  }
  return null
}

function foldItem(item: HTMLElement) {
  const i = mainRowSequence.indexOf(item)
  const left = mainRowSequence[i - 1]
  // 原次序内跨组：左邻是分隔符 → 打组分隔标记（region 首个项由 CSS 抑制顶线）
  item.classList.toggle('hub-fold-group-start', !!left?.classList.contains('vditor-toolbar__divider'))
  foldRegion!.insertBefore(item, foldRegion!.firstChild) // 插到最前 → 折叠区保持主行左→右次序
  // Mobile main-row icons need their translated label once they become menu rows.
  const button = item.querySelector<HTMLElement>(':scope > [data-type]')
  if (button) ensureToolbarButtonLabel(button)
  syncAdjacentDividers(item)
}

function unfoldItem(item: HTMLElement) {
  const i = mainRowSequence.indexOf(item)
  let anchor: HTMLElement | null = null
  for (let j = i + 1; j < mainRowSequence.length; j++) {
    if (mainRowSequence[j].parentElement === toolbarEl) {
      anchor = mainRowSequence[j]
      break
    }
  }
  toolbarEl!.insertBefore(item, anchor ?? moreWrapper!) // 放回原次序正确位置
  item.classList.remove('hub-fold-group-start')
  // 回到主行换回短标签（菜单里用的是完整名称）
  const button = item.querySelector<HTMLElement>(':scope > [data-type]')
  if (button?.querySelector(':scope > .vditor-toolbar-label')) ensureToolbarButtonLabel(button)
  syncAdjacentDividers(item)
}

function measureToolbar() {
  foldRaf = 0
  if (destroyed || !toolbarEl?.isConnected || !foldRegion) return
  // Mobile presets can still exceed a narrow host once the counter grows.
  // Reuse node-preserving folding so every action remains available in More.
  // 恒覆盖工具栏左 padding（防 vditor setPadding 按「内容 padding + 大纲宽度」
  // 在窄屏/大纲展开时把图标挤到右端）；再按宽度滞回切换标签显隐。
  syncToolbarLeftPadding()
  if (toolbarEl.clientWidth < LABEL_HIDE_THRESHOLD) {
    labelsHidden = true
    labelsLatched = true        // #7：触发一次即锁存，之后宽度恢复不再显示
  } else if (!labelsLatched && toolbarEl.clientWidth >= LABEL_SHOW_THRESHOLD) {
    labelsHidden = false
  }
  if (root.value && root.value.classList.contains('hub-toolbar-icon-only') !== labelsHidden) {
    root.value.classList.toggle('hub-toolbar-icon-only', labelsHidden)
  }
  for (let g = 0; g < 100; g++) {
    // 折叠：右→左
    if (toolbarEl.scrollWidth <= toolbarEl.clientWidth + OVERFLOW_TOLERANCE) break
    const item = rightmostMainRowItem()
    if (!item) break
    foldItem(item)
  }
  for (let g = 0; g < 100; g++) {
    // 恢复：反序尝试放回。主行项 flex-grow 到 clientWidth，scrollWidth 无法反映富余量，
    // 因此改为「放回→查溢出→溢出则收回」的试放法；收回后停止，避免与折叠来回抖。
    const item = foldRegion.firstElementChild as HTMLElement | null
    if (!item) break
    unfoldItem(item)
    if (toolbarEl.scrollWidth > toolbarEl.clientWidth + OVERFLOW_TOLERANCE) {
      foldItem(item)
      break
    }
  }
}

function scheduleMeasure() {
  if (destroyed || foldRaf) return
  foldRaf = requestAnimationFrame(measureToolbar)
}

function initToolbarFolding() {
  if (!root.value) return
  toolbarEl = root.value.querySelector<HTMLElement>('.vditor-toolbar')
  if (!toolbarEl) return
  moreWrapper = toolbarEl.querySelector<HTMLElement>(
    ':scope > .vditor-toolbar__item > [data-type="more"]',
  )?.parentElement ?? null
  if (!moreWrapper) return // 无 more（理论不会发生）→ 不做收纳
  hintPanel = moreWrapper.querySelector<HTMLElement>(':scope > .vditor-hint')
  if (!hintPanel) return
  mainRowSequence = Array.from(toolbarEl.children).filter(
    (c): c is HTMLElement => c !== moreWrapper && !c.classList.contains('vditor-counter'),
  )
  foldRegion = document.createElement('div')
  foldRegion.className = 'hub-fold-region'
  hintPanel.insertBefore(foldRegion, hintPanel.firstChild) // 折叠区置于子菜单最顶部
  foldObserver = new ResizeObserver(scheduleMeasure)
  foldObserver.observe(toolbarEl)
  window.addEventListener('resize', scheduleMeasure)
  // vditor setPadding 在窗口缩放时会按「内容 padding + 大纲宽度」重设工具栏 paddingLeft，
  // 这里覆盖回恒定靠左值，保证缩放后工具栏左缘不变
  window.addEventListener('resize', syncToolbarLeftPadding)
  // 初始即定稿恒定靠左值（大纲初始展开后 vditor 可能已写入位移值）
  syncToolbarLeftPadding()
  measureToolbar() // 首次同步测量，避免初始闪烁
}

/**
 * 预置官方 id 的运行时资源，让 vditor 跳过 CDN：
 * - script#vditorHljsScript / #vditorHljsThirdScript：空占位，addScript 直接 resolve
 * - link#vditorKatexStyle：KaTeX 样式，addStyle 检测到存在不重复创建
 * 注意：highlightRender 会按 `${cdn}/dist/js/highlight.js/styles/github.min.css`
 * 计算并维护 link#vditorHljsStyle，属官方 404 请求，样式实际由
 * syncHighlightTheme 的 link 提供，二者互不干扰。
 */
function injectOfficialAssetPlaceholders() {
  const ensureScript = (id: string) => {
    if (document.getElementById(id)) return
    const script = document.createElement('script')
    script.id = id
    document.head.appendChild(script)
  }
  ensureScript('vditorHljsScript')
  ensureScript('vditorHljsThirdScript')

  if (!document.getElementById('vditorKatexStyle')) {
    const katexStyle = document.createElement('link')
    katexStyle.id = 'vditorKatexStyle'
    katexStyle.rel = 'stylesheet'
    katexStyle.href = katexCssUrl
    document.head.appendChild(katexStyle)
  }
}

/**
 * 覆盖 Vditor 官方 mermaidRender（预览/分屏模式的流程图渲染）。
 * 官方实现从 CDN 加载其捆绑的旧版 mermaid（11.6.0）且 securityLevel: loose，
 * 本组件 cdn 为空，离线环境下会静默失败。这里改为惰性加载本站 mermaid chunk
 * （与主题页 v-content-enhancements 共用同一引擎与 strict 安全配置），并跟随
 * 站点深浅色主题；仍遵守 Vditor 的 data-processed 契约——预览重渲染时跳过
 * 已处理的图表，渲染失败则保留源码代码块（预览为瞬时编辑态，下一次重渲染会
 * 重建代码块并再次尝试）。
 */
function overrideMermaidPreviewRender() {
  const namespace = Vditor as unknown as {
    mermaidRender: (element: HTMLElement, _cdn: string, _theme: string) => void
  }
  namespace.mermaidRender = (element) => {
    const blocks = Array.from(element.querySelectorAll<HTMLElement>('code.language-mermaid'))
    if (blocks.length === 0) return
    void (async () => {
      let mermaid: Awaited<ReturnType<typeof loadMermaid>>
      try {
        mermaid = await loadMermaid()
      } catch (error) {
        console.warn('Unable to load Mermaid for the editor preview.', error)
        return
      }
      for (const block of blocks) {
        if (block.getAttribute('data-processed') === 'true' || !block.textContent?.trim()) continue
        block.setAttribute('data-processed', 'true')
        try {
          const { svg } = await renderDiagram(mermaid, nextMermaidDiagramId('gf-vditor-mermaid'), block.textContent)
          block.innerHTML = svg
        } catch (error) {
          console.warn('Unable to render Mermaid diagram in the editor preview; preserving its source code.', error)
        }
      }
    })()
  }
}

function syncEditorTheme() {
  if (!editor || !ready || destroyed) return
  editor.setTheme(isDark.value ? 'dark' : 'classic')
  if (editor.vditor.options.preview?.theme) {
    editor.vditor.options.preview.theme.current = isDark.value ? 'dark' : 'light'
  }
  syncContentTheme()
  syncHighlightTheme()
}

/** 大纲开关：随面板移动的展开/收起按钮；
 *  移动端收起态移入宿主行内容器（toggleHost）后不在 .vditor 内，查找需同时覆盖 host */
const OUTLINE_TOGGLE_CLASS = 'hub-outline-toggle'
/** 大纲展开/收起动画：宽度 250↔0 + 透明度过渡（0.3s < 0.5s）。
 *  宽度过渡让 flex 编辑区平滑重排，正文 padding 由 JS 同步设为目标值，正文平滑不跳转。 */
const OUTLINE_WIDTH = 250
const OUTLINE_ANIM_MS = 300
let outlineVisible = true
let outlineAnimTimer = 0
let outlineEndHandler: ((event: TransitionEvent) => void) | null = null

/**
 * 大纲显隐动画：设目标宽度/透明度与正文 padding，过渡完成后收尾（display 切到终态）。
 * 宽度从 250→0（收起）会让 flex 中编辑区平滑变宽，正文 padding 同步过渡到居中值，
 * 避免 outline display:none 瞬间重排导致正文「先猛跳、再回移」。
 */
function animateOutline(show: boolean) {
  const vditor = editor?.vditor
  const outlineEl = vditor?.outline.element
  const rootEl = root.value
  if (!vditor || !outlineEl || !rootEl) {
    syncOutlineToggleHost()
    return
  }
  const reset = rootEl.querySelector<HTMLElement>('.vditor-wysiwyg pre.vditor-reset')
  if (!reset) {
    syncOutlineToggleHost()
    return
  }
  const maxWidth = vditor.options.preview?.maxWidth ?? 800
  const contentEl = rootEl.querySelector<HTMLElement>('.vditor-content')
  const fullWidth = contentEl?.clientWidth ?? reset.clientWidth
  // 与 vditor setPadding 同款公式：outline 占位（250）与否决定编辑区内容居中 padding
  const targetPad = Math.max(35, (fullWidth - (show ? OUTLINE_WIDTH : 0) - maxWidth) / 2)

  window.clearTimeout(outlineAnimTimer)
  if (outlineEndHandler) outlineEl.removeEventListener('transitionend', outlineEndHandler)

  if (show) {
    const wasHidden = outlineEl.style.display === 'none'
    outlineEl.style.display = 'block'
    if (wasHidden) {
      // 完全收起→展开：从 width:0 起播（强制 reflow 保证 transition 从 0 开始）
      outlineEl.style.width = '0px'
      outlineEl.style.opacity = '0'
      vditor.outline.render(vditor)
      void outlineEl.offsetWidth
    }
    reset.style.padding = `10px ${targetPad}px`
    outlineEl.style.width = `${OUTLINE_WIDTH}px`
    outlineEl.style.opacity = '1'
  } else {
    reset.style.padding = `10px ${targetPad}px`
    outlineEl.style.width = '0px'
    outlineEl.style.opacity = '0'
  }

  const finish = () => {
    window.clearTimeout(outlineAnimTimer)
    if (outlineEndHandler) {
      outlineEl.removeEventListener('transitionend', outlineEndHandler)
      outlineEndHandler = null
    }
    if (show) {
      outlineEl.style.width = `${OUTLINE_WIDTH}px`
      outlineEl.style.opacity = '1'
      outlineEl.style.display = 'block'
    } else {
      outlineEl.style.display = 'none'
    }
    syncOutlineToggleHost()
    // 大纲动画收尾 paddingLeft 突变后确定性重测折叠
    scheduleMeasure()
  }
  outlineEndHandler = (event) => {
    if (event.propertyName === 'width') finish()
  }
  outlineEl.addEventListener('transitionend', outlineEndHandler)
  outlineAnimTimer = window.setTimeout(finish, OUTLINE_ANIM_MS + 80)
}

function outlineToggleButton(): HTMLButtonElement | null {
  const existing =
    root.value?.querySelector<HTMLButtonElement>(`.${OUTLINE_TOGGLE_CLASS}`) ||
    props.toggleHost?.querySelector<HTMLButtonElement>(`.${OUTLINE_TOGGLE_CLASS}`)
  if (existing) return existing
  if (!root.value) return null

  const button = document.createElement('button')
  button.type = 'button'
  button.className = `${OUTLINE_TOGGLE_CLASS} vditor-tooltipped vditor-tooltipped__e`
  button.setAttribute('data-type', 'outline')
  button.setAttribute('aria-label', window.VditorI18n?.outline || '大纲')
  // 官方 Outline 按钮使用的图标是 vditor-icon-align-center（图标集无 -outline）
  button.innerHTML = '<svg><use xlink:href="#vditor-icon-align-center"></use></svg>'
  button.addEventListener('click', () => {
    if (!editor || !ready || !editor.vditor.outline) return
    const currentEditor = editor
    if (currentEditor.vditor.options.outline) {
      currentEditor.vditor.options.outline.enable = !outlineVisible
    }
    outlineVisible = !outlineVisible
    animateOutline(outlineVisible)
  })
  root.value.appendChild(button)
  return button
}

/** 工具栏内容恒定靠左、不随大纲开合移动：
 *  有悬浮「向上扩展」按钮（headerToggle）时让出 35px（按钮占 0-35px），否则用 vditor 默认 5px。
 *  显式覆盖 vditor setPadding 的「内容 padding + 大纲宽度」算法（它会让工具栏随大纲位移），
 *  保证开关大纲 / 窗口缩放后工具栏左缘恒定。仅桌面生效：移动端大纲隐藏、悬浮按钮移入行内。 */
function syncToolbarLeftPadding() {
  const toolbar = root.value?.querySelector<HTMLElement>('.vditor-toolbar')
  if (!toolbar) return
  // 桌面且有悬浮「向上扩展」按钮 → 让出 35px；移动端按钮移入行内 / 无按钮 → 5px。
  // 恒覆盖 vditor setPadding：它按「内容 padding + 大纲宽度」计算，大纲展开/窄屏下
  // 会把 padding-left 推到数百像素（如 260px），把图标挤到工具栏右端。
  toolbar.style.paddingLeft = !window.matchMedia(MOBILE_VIEWPORT_QUERY).matches && props.headerToggle ? '35px' : '5px'
}

function syncOutlineToggleHost() {
  const button = outlineToggleButton()
  const outline = editor?.vditor.outline
  if (!button || !outline) return

  // 以面板实际显示状态为准（初始 enable=true 时也需先 toggle 展开）
  const shown = outline.element.style.display === 'block'
  button.classList.toggle('vditor-menu--current', shown)

  // 移动端不提供大纲唤起：大纲在窄屏价值低，开关会遮挡编辑区/占位，
  // 直接隐藏（编辑区最大化）；桌面保持标题栏/悬浮逻辑
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) {
    button.classList.add('is-hidden')
    return
  }
  button.classList.remove('is-hidden')

  if (shown) {
    // 展开：按钮归入大纲标题栏（「大纲」文字右侧）
    const title = outline.element.querySelector('.vditor-outline__title')
    button.classList.remove(`${OUTLINE_TOGGLE_CLASS}--float`)
    button.classList.remove('hub-toggle--inline')
    if (title && button.parentElement !== title) title.appendChild(button)
  } else {
    // 收起（桌面）：按钮悬浮在左侧大纲原位置，保持可展开
    button.classList.add(`${OUTLINE_TOGGLE_CLASS}--float`)
    button.classList.remove('hub-toggle--inline')
    const editorRoot = root.value
    if (editorRoot && button.parentElement !== editorRoot) editorRoot.appendChild(button)
  }

  // 发布页「向上扩展」按钮跟随大纲显隐（仅大纲展开时浮在大纲上方，收起时隐藏避免遮挡工具栏）
  if (props.headerToggle) syncHeaderToggleHost()
  // 工具栏恒定靠左（覆盖 vditor setPadding 可能在大纲动画中写下的位移值）
  syncToolbarLeftPadding()
}

/** 「向上扩展」按钮：悬浮在大纲列正上方（工具栏左侧空白区），与工具栏工具完全分离；
 *  移动端移入宿主行内容器（toggleHost）后不在 .vditor 内，查找需同时覆盖 host */
const HEADER_TOGGLE_CLASS = 'hub-header-toggle-float'
function headerToggleButton(): HTMLButtonElement | null {
  const existing =
    root.value?.querySelector<HTMLButtonElement>(`.${HEADER_TOGGLE_CLASS}`) ||
    props.toggleHost?.querySelector<HTMLButtonElement>(`.${HEADER_TOGGLE_CLASS}`)
  if (existing) return existing
  if (!root.value) return null

  const button = document.createElement('button')
  button.type = 'button'
  button.className = `${HEADER_TOGGLE_CLASS} vditor-tooltipped vditor-tooltipped__s`
  button.setAttribute('data-type', 'header-toggle')
  button.setAttribute('aria-label', t('publish.collapseHeader'))
  // 双图标：展开态显示向上箭头（收起头部），收起态（vditor-menu--current）切换为向下箭头（展开头部）
  button.innerHTML =
    '<svg class="hub-header-toggle-icon hub-header-toggle-icon--up" viewBox="0 0 24 24" style="fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="m9 16 3-3 3 3"/></svg>' +
    '<svg class="hub-header-toggle-icon hub-header-toggle-icon--down" viewBox="0 0 24 24" style="fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="m9 12 3 3 3-3"/></svg>'
  button.addEventListener('click', () => {
    button.classList.toggle('vditor-menu--current')
    emit('toggle-header')
  })
  root.value.appendChild(button)
  return button
}

function syncHeaderToggleHost() {
  const button = headerToggleButton()
  if (!button) return

  // 移动端且宿主提供 toggleHost：按钮移入宿主行内容器（如「正文」label 行），
  // 工具栏不再需要占位 padding，完全舒展
  const mobileHost = window.matchMedia(MOBILE_VIEWPORT_QUERY).matches ? props.toggleHost : null
  if (mobileHost) {
    button.classList.add('hub-toggle--inline')
    if (button.parentElement !== mobileHost) mobileHost.appendChild(button)
    return
  }

  // 桌面：始终显示（收起大纲时按钮悬浮在左缘，不随大纲隐藏，否则用户找不到入口）
  button.classList.remove('is-hidden')
  button.classList.remove('hub-toggle--inline')

  // 工具栏恒定靠左（见 syncToolbarLeftPadding 注释）
  syncToolbarLeftPadding()
}

type LuteRenderMethod = 'Md2VditorDOM' | 'Md2VditorIRDOM' | 'SpinVditorSVDOM' | 'Md2HTML'

/**
 * 连续空行（DQ-1，docs/decisions/0128-visible-blank-lines.md）：Lute 载入 Markdown 时把一段
 * 空行合并成一个段落间隔，重新编辑再保存就丢掉了作者留的空段。载入前在这些位置插入标记
 * 段落，渲染后清空成与按 Enter 相同的空段，VditorDOM2Md 再把 k 个空段写回 k+1 行空行。
 * 编辑区（所见即所得、即时渲染、源码）全部保留；预览按发布规则渲染为有上限的占位。
 */
const SV_NEWLINE = '<span data-type="newline"><br /><span style="display: none">\n</span></span>'

function preserveBlankParagraphs(v: Vditor) {
  const lute = v.vditor.lute as unknown as Record<LuteRenderMethod, (markdown: string) => string>
  const wrap = (method: LuteRenderMethod, max: number, marked: (marker: string) => string, empty: string) => {
    const render = lute[method].bind(lute)
    lute[method] = (markdown: string) => {
      const marker = blankLineMarker(markdown)
      const source = markBlankLines(markdown, marker, max)
      if (source === markdown) return render(markdown)
      const html = render(source).replaceAll(marked(marker), empty)
      // 标记落进了其他结构（例如原始 HTML）时放弃保留，按原文渲染
      return html.includes(marker) ? render(markdown) : html
    }
  }
  wrap('Md2VditorDOM', Infinity, (marker) => `<p data-block="0">${marker}</p>`, '<p data-block="0"></p>')
  wrap('Md2VditorIRDOM', Infinity, (marker) => `<p data-block="0">${marker}</p>`, '<p data-block="0"></p>')
  // 源码模式显示原文：删掉标记行（文字连同它的换行），留下与原文相同行数的空行
  wrap('SpinVditorSVDOM', Infinity, (marker) => `<span data-type="text">${marker}</span>${SV_NEWLINE}`, '')
  wrap('Md2HTML', MAX_RENDERED_BLANK_LINES, (marker) => `<p>${marker}</p>`, BLANK_LINE_HTML)
}

let popoverObservers: MutationObserver[] = []

/**
 * 悬浮辅助栏（链接 / 表格 / 代码块等）第一行层级与边缘切割防御：
 * 1. 当 targetElement 处于编辑器第一行时，Vditor 原生计算 targetElement.offsetTop - 21 会得出负值（例如 -8px 或极小正值），
 *    导致悬浮工具栏向上插入 toolbar 底边内部，或被顶层父级容器边界切断。
 *    此处将悬浮栏自动翻转并定位于当前行下方（目标行 offsetTop + 30px），彻底避开顶部工具栏和容器顶边缘。
 * 2. 在移动视图或窄视口下，若浮层宽度可能溢出右侧屏幕或处于左侧负值，自动进行边界约束（clamp）。
 */
function initPopoverPositionCorrection(v: Vditor) {
  for (const obs of popoverObservers) obs.disconnect()
  popoverObservers = []

  const wysiwyg = v.vditor?.wysiwyg
  if (!wysiwyg) return

  const popovers = [wysiwyg.popover, wysiwyg.selectPopover].filter(Boolean) as HTMLElement[]

  for (const popover of popovers) {
    let adjusting = false
    const adjust = () => {
      if (adjusting || popover.style.display === 'none') return
      adjusting = true

      try {
        const topVal = parseFloat(popover.style.top || '0')
        const wysiwygEl = wysiwyg.element
        if (!wysiwygEl) return

        // 第一行层级与边缘切割防御：top 偏小或为负数时，翻转到当前行正下方
        if (topVal < 14) {
          const dataTop = parseFloat(popover.getAttribute('data-top') || '0')
          const targetOffsetTop = isNaN(dataTop) ? 0 : dataTop + 21
          const safeTop = Math.max(targetOffsetTop + 30, 36)
          popover.style.top = `${safeTop}px`
        }

        // 横向边缘防裁切（移动端/窄容器下防止右侧溢出屏幕或左侧负值）
        const leftVal = parseFloat(popover.style.left || '0')
        const popoverWidth = popover.offsetWidth || 280
        const containerWidth = wysiwygEl.clientWidth
        if (containerWidth > 0) {
          const maxLeft = Math.max(6, containerWidth - popoverWidth - 8)
          const safeLeft = Math.min(Math.max(6, leftVal), maxLeft)
          if (Math.abs(leftVal - safeLeft) > 1) {
            popover.style.left = `${safeLeft}px`
          }
        }
      } finally {
        adjusting = false
      }
    }

    const observer = new MutationObserver(() => {
      adjust()
    })

    observer.observe(popover, {
      attributes: true,
      attributeFilter: ['style', 'data-top'],
    })

    popoverObservers.push(observer)
  }
}

/**
 * 工具栏下拉面板横向边界自适应：
 * 防止移动端或窄屏视口下，浮层超出屏幕左/右侧边缘被浏览器截断。
 */
function initToolbarPanelPositionCorrection(v: Vditor) {
  const toolbarEl = v.vditor?.toolbar?.element
  if (!toolbarEl) return

  const clampPanelHorizontal = (panel: HTMLElement) => {
    if (panel.style.display === 'none') return
    panel.style.removeProperty('transform')

    const panelRect = panel.getBoundingClientRect()
    if (panelRect.width === 0 || panelRect.height === 0) return

    const container = toolbarEl.closest('.gf-modal-editor, .gf-composer-surface, .vditor-official') as HTMLElement | null
    const containerRect = container?.getBoundingClientRect()
    const maxRight = containerRect ? Math.min(window.innerWidth - 8, containerRect.right - 8) : window.innerWidth - 8
    const minLeft = containerRect ? Math.max(8, containerRect.left + 8) : 8

    let shiftX = 0
    if (panelRect.left < minLeft) {
      shiftX = minLeft - panelRect.left
    } else if (panelRect.right > maxRight) {
      shiftX = -(panelRect.right - maxRight)
    }

    if (shiftX !== 0) {
      panel.style.setProperty('transform', `translateX(${shiftX}px)`, 'important')
    }
  }

  /** 「更多」菜单按视口剩余空间定方向与最大高度：下方放不下且上方更宽时向上展开（底部弹层编辑器恒向上） */
  const fitMenuVertical = (menu: HTMLElement) => {
    if (menu.style.display === 'none' || !menu.parentElement) return
    const anchor = menu.parentElement.getBoundingClientRect()
    const below = window.innerHeight - anchor.bottom - 12
    const above = anchor.top - 12
    const up = Boolean(menu.closest('.gf-modal-editor')) || (menu.scrollHeight > below && above > below)
    menu.classList.toggle('gf-editor-menu--up', up)
    menu.style.setProperty('--gf-menu-max-h', `${Math.max(160, Math.floor(up ? above : below))}px`)
  }

  const runClamp = () => {
    if (toolbarPanelClampRaf) return
    toolbarPanelClampRaf = requestAnimationFrame(() => {
      toolbarPanelClampRaf = 0
      const activePanels = toolbarEl.querySelectorAll<HTMLElement>('.vditor-panel, .vditor-hint')
      for (const p of activePanels) {
        if (p.classList.contains('gf-editor-menu')) fitMenuVertical(p)
        clampPanelHorizontal(p)
      }
    })
  }

  // 捕获阶段：Vditor 的子菜单开关（toggleSubMenu）会 stopPropagation，冒泡监听收不到；
  // runClamp 在下一帧执行，届时面板已展开
  toolbarEl.addEventListener('click', () => {
    runClamp()
    setTimeout(runClamp, 60)
  }, { capture: true, passive: true })

  // 模块级保存绑定引用：卸载时据此移除，防止每个编辑器实例泄漏一个 window 监听
  toolbarPanelResizeHandler = runClamp
  window.addEventListener('resize', toolbarPanelResizeHandler, { passive: true })
}

onMounted(async () => {
  if (!root.value) return
  linkPreviewHintElement = document.createElement('p')
  linkPreviewHintElement.className = 'gf-link-preview-editor-hint'
  linkPreviewHintElement.setAttribute('aria-live', 'polite')
  linkPreviewHintElement.hidden = true
  root.value.insertAdjacentElement('afterend', linkPreviewHintElement)
  root.value.addEventListener('compositionstart', handleCompositionStart, true)
  root.value.addEventListener('compositionend', handleCompositionEnd, true)
  root.value.addEventListener('keydown', handleStickerKeydown, true)
  root.value.addEventListener('beforeinput', handleStickerBeforeInput, true)
  root.value.addEventListener('focusout', handleEditorFocusOut)
  document.addEventListener('selectionchange', handleSelectionChange)
  linkPreviewHintController.schedule(props.modelValue)
  const language = languageAssets[currentLocale()]

  // window.hljs 必须在 vditor 执行高亮前就绪
  ;(window as unknown as { hljs: typeof hljs }).hljs = hljs
  injectOfficialAssetPlaceholders()

  try {
    // KaTeX 必须先行：mhchem.min.js 依赖全局 katex，并发加载会报 __defineMacro 错误
    await Promise.all([
      loadRuntimeScript(language.url, `vditorI18nScript${language.lang}`),
      loadRuntimeScript(antIconUrl, 'vditorIconScript'),
      loadRuntimeScript(luteUrl, 'vditorLuteScript'),
      loadRuntimeScript(katexJsUrl, 'vditorKatexScript'),
    ])
    await loadRuntimeScript(katexChemUrl, 'vditorKatexChemScript')
  } catch (error) {
    editorFailed.value = true
    editorReady.value = false
    emit('error', error instanceof Error ? error : new Error(String(error)))
    return
  }
  if (destroyed || !root.value) return

  // 在编辑器创建前安装 mermaid 预览覆盖（官方 CDN 版本离线不可用）
  overrideMermaidPreviewRender()

  let nextEditor: Vditor | null = null
  try {
    nextEditor = new Vditor(root.value, {
      _lutePath: luteUrl,
      cache: { enable: false },
      // 必须提供空实现：wysiwyg 的链接浮层（genAPopover）无条件调用它，
      // 缺失会在 setPopoverPosition 之前抛 TypeError，导致浮层永远不显示
      customWysiwygToolbar() {},
      // 运行时资源走本地预载（i18n/icon/lute/katex/hljs），不依赖官方 CDN
      cdn: '',
      // 移动端 = 屏高一半（官方 sweet-mobile 建议）；桌面 = height prop
      height: resolveHeight(),
      i18n: window.VditorI18n,
      icon: 'ant',
      lang: language.lang,
      // 编辑器内图片只保留 Vditor 的编辑浮层；关闭原生双击全屏预览，避免移动端误触打开无法退出的灯箱。
      image: {
        isPreview: false,
      },
      // 所见即所得：显示悬浮在片段左侧的块级控制指示器（</>、H1/H2、$、ToC）
      mode: 'wysiwyg',
      placeholder: props.placeholder,
      // 必须始终提供 outline 对象（enable 为 false 也传完整对象）：
      // Vditor 的 merge 会保留显式 undefined，导致 initUI 读 options.outline.position 崩溃
      outline: { enable: props.outline === true, position: 'left' },
      // 字数统计按渲染后的可见文字（wysiwyg DOM 文本）而非 Markdown 源文本，
      // 与服务端正文/回复长度校验口径一致（issue #890）：标记、链接目标、图片不计入。
      counter: { enable: props.counter === true, type: 'text' },
      theme: isDark.value ? 'dark' : 'classic',
      // 移动端用官方精简工具栏，桌面用完整工具栏
      toolbar: resolveToolbar(),
      toolbarConfig: { hide: false, pin: true },
      preview: {
        // Vditor 管理此 Markdown 预览 DOM，不经过 Vue 的 v-html 指令。排到 afterRender
        // 微任务中，使代码高亮和其他 Markdown 渲染先完成，再只处理预览子树中的文本。
        parse(element) {
          queueMicrotask(() => {
            if (!element.isConnected) return
            renderTwemojiHtml(element)
            // 预览与编辑区同一套贴纸装饰：token 显示为贴纸图，而不是一串原文
            decorateStickerTokens(element, stickerAsset)
          })
        },
        // hljs/math 走官方默认（enable: true、KaTeX），资源已本地化
        theme: { current: isDark.value ? 'dark' : 'light', path: '' },
      },
      upload: {
        accept: 'image/*',
        multiple: true,
        handler(files) {
          emit('upload', files)
          // 返回 null：插入 markdown 由宿主在上传完成后通过 insertMarkdown 完成
          return null
        },
      },
      value: props.modelValue,
      after() {
        // 先绑定 editor 再判断销毁：脚本缓存命中时 after 可能早于外层赋值触发，
        // 若用 editor !== nextEditor 判断会误判为「陈旧实例」把自己销毁（输入失效）。
        editor = nextEditor
        if (destroyed || !root.value) {
          const staleEditor = nextEditor
          nextEditor = null
          queueMicrotask(() => staleEditor?.destroy())
          return
        }
        ready = true
        editorFailed.value = false
        editorReady.value = true
        syncContentTheme()
        syncHighlightTheme()
        attachToolbarLabels()
        decorateEditorMenus()
        // 大纲初始展开：enable 只是配置，需 toggle 后面板才显示（官方 Outline 按钮同逻辑）
        // 未开启大纲时不创建开关按钮，避免悬浮层覆盖编辑区（如回复浮动面板）
        if (props.outline) {
          if (editor!.vditor.options.outline) {
            editor!.vditor.options.outline.enable = true
          }
          editor!.vditor.outline.toggle(editor!.vditor, true)
          syncOutlineToggleHost()
        }
        if (props.headerToggle) syncHeaderToggleHost()
        // 工具栏收纳：须在大纲展开/header-toggle 定稿 paddingLeft 后再首次测量
        initToolbarFolding()
        // 初始化期间若切换过语言，就绪后按当前语言补刷（防切换被静默丢弃）
        if (pendingLocaleRefresh) {
          pendingLocaleRefresh = false
          void refreshVditorLocale()
        }
        // 服务端不解析 `:alias:` emoji 简码；Lute 默认解析会让编辑区与发布结果不一致，
        // 还会改写贴纸 token（`[:sticker:b:]` 被存成 `[:sticker🅱️]`）。初值在 after 之前
        // 已按默认选项渲染过一次，含简码样式文本时关掉后重渲染。
        // Lute 运行时提供 SetEmoji，Vditor 的类型声明里没有列出
        ;(nextEditor?.vditor.lute as { SetEmoji?: (enabled: boolean) => void } | undefined)?.SetEmoji?.(false)
        // 初值同样已按合并空行渲染过：含连续空行时与 getValue 不同，下面会按保留空段重渲染
        preserveBlankParagraphs(nextEditor!)
        if (props.modelValue && (props.modelValue !== nextEditor?.getValue() || /:[^\s:]+:/.test(props.modelValue))) {
          nextEditor?.setValue(props.modelValue, true)
        }
        initPopoverPositionCorrection(nextEditor!)
        initToolbarPanelPositionCorrection(nextEditor!)
        attachSensitiveModeRefresh()
        attachStickerDecoration()
        floating.attach()
      },
      input(value) {
        clearSensitiveHighlights()
        emit('update:modelValue', value)
        emit('input')
        linkPreviewHintController.schedule(value)
        // 字数变化改变工具栏末位 counter 宽度，RO 只测 border-box，需显式重测
        scheduleMeasure()
        floating.schedule()
      },
    })
    editor = nextEditor
  } catch (error) {
    editorFailed.value = true
    editorReady.value = false
    emit('error', error instanceof Error ? error : new Error(String(error)))
  }
})

watch(() => props.modelValue, (value) => {
  linkPreviewHintController.schedule(value)
  scheduleStickerResolve(value)
  if (!editor || !ready || value === editor.getValue()) return
  editor.setValue(value, true)
})

watch(
  () => [props.sensitiveWords, editorReady.value] as const,
  ([words, isReady]) => {
    if (!isReady) return
    if (words?.length) highlightSensitiveText(words)
    else clearSensitiveHighlights()
  },
  { immediate: true },
)

watch(isDark, syncEditorTheme)

// 解析结果到达（或贴纸被停用）后就地更新已渲染的块
watch(resolvedUrls, scheduleStickerDecoration)

// #8：语言切换 → 重载 vditor i18n 脚本并就地刷新全部界面文案。
// SPA 内 setLocale 只改 vue-i18n ref，绝不 location.reload()（AppShell.setLang）。
watch(locale, () => {
  syncLinkPreviewHint()
  if (!editor || !ready || destroyed) {
    // 编辑器初始化期间：标记待补刷，after() 就绪后按当前语言应用
    pendingLocaleRefresh = true
    return
  }
  void refreshVditorLocale()
})

// #8：空态占位符。父组件 placeholder 是 computed(t(...))（如 PostComposer L102），
// locale 一变 prop 即变；vditor 无热 setter，只能同步三个 mode 元素的 placeholder 属性。
watch(() => props.placeholder, () => { refreshPlaceholder() })

onBeforeUnmount(() => {
  destroyed = true
  linkPreviewHintController.dispose()
  root.value?.removeEventListener('compositionstart', handleCompositionStart, true)
  root.value?.removeEventListener('compositionend', handleCompositionEnd, true)
  root.value?.removeEventListener('keydown', handleStickerKeydown, true)
  root.value?.removeEventListener('beforeinput', handleStickerBeforeInput, true)
  root.value?.removeEventListener('focusout', handleEditorFocusOut)
  document.removeEventListener('selectionchange', handleSelectionChange)
  floating.detach()
  stickerObserver?.disconnect()
  stickerObserver = null
  clearTimeout(stickerResolveTimer)
  if (stickerDecorationFrame) cancelAnimationFrame(stickerDecorationFrame)
  if (stickerSuggestFrame) cancelAnimationFrame(stickerSuggestFrame)
  stickerDecorationFrame = stickerSuggestFrame = 0
  stickerSuggest.close()
  linkPreviewHintElement?.remove()
  linkPreviewHintElement = null
  detachSensitiveModeRefresh()
  fullscreenLabelObserver?.disconnect()
  fullscreenLabelObserver = null
  for (const obs of popoverObservers) obs.disconnect()
  popoverObservers = []
  // 工具栏收纳清理：观察器/监听/rAF 全部释放，防止陈旧实例泄漏
  foldObserver?.disconnect()
  foldObserver = null
  window.removeEventListener('resize', scheduleMeasure)
  window.removeEventListener('resize', syncToolbarLeftPadding)
  if (foldRaf) cancelAnimationFrame(foldRaf)
  foldRaf = 0
  // 工具栏面板横向避让清理：移除 window resize 监听并取消在途 rAF，
  // 该监听持有 vditor DOM 闭包，不释放会导致每次编辑器实例泄漏监听器与 DOM
  if (toolbarPanelResizeHandler) {
    window.removeEventListener('resize', toolbarPanelResizeHandler)
    toolbarPanelResizeHandler = null
  }
  if (toolbarPanelClampRaf) cancelAnimationFrame(toolbarPanelClampRaf)
  toolbarPanelClampRaf = 0
  toolbarEl = moreWrapper = hintPanel = foldRegion = null
  mainRowSequence = []
  const currentEditor = editor
  editor = null
  editorReady.value = false
  editorFailed.value = false
  // 与官方 beforeDestroy 一致：ready 后直接 destroy；未 ready 时由 after() 兜底。
  if (!ready) return
  ready = false
  currentEditor?.destroy()
})

function focus() {
  if (ready) editor?.focus()
}

/** 宿主拖拽调整高度时同步 Vditor 高度（面板级 resize 手柄用） */
function setHeight(height: number) {
  if (!editor || !ready || destroyed) return
  editor.vditor.element.style.height = `${Math.max(120, Math.floor(height))}px`
  // setPadding 只计算水平 padding，高度直接改 style 即可，内容区 flex 自适应
  if (typeof editor.vditor.options.resize?.after === 'function') {
    editor.vditor.options.resize.after(0)
  }
}

function insertMarkdown(markdown: string) {
  clearSensitiveHighlights()
  if (!editor || !ready) {
    emit('update:modelValue', props.modelValue ? `${props.modelValue}\n${markdown}` : markdown)
    return
  }
  editor.focus()
  editor.insertMD(markdown)
  emit('update:modelValue', editor.getValue())
}

/**
 * @mention 最小 caret/input hook（issue #564）：宿主不感知 Vditor 内部结构。
 * 三种编辑模式（wysiwyg/ir/sv）的编辑元素均为 contenteditable，统一走 DOM Selection API。
 */
export interface MentionCaretContext {
  /** 光标前原始文本（最多回溯 200 字符；token 以空白/终止标点结尾，足够识别） */
  prefix: string
  /** caret 视口矩形：桌面弹层定位用；选区非折叠/不可测时为 null */
  rect: DOMRect | null
  /** 光标是否位于 code/pre/a 内（wysiwyg/ir）或围栏/行内代码内（sv）——不弹候选 */
  inCode: boolean
  /** 当前编辑元素（contenteditable），宿主用于绑定 aria-activedescendant 等 */
  element: HTMLElement | null
}

const MENTION_PREFIX_LIMIT = 200

function currentModeElement(): HTMLElement | null {
  if (!editor || !ready || destroyed) return null
  const mode = editor.vditor.currentMode
  const el = mode === 'wysiwyg' ? editor.vditor.wysiwyg?.element : mode === 'ir' ? editor.vditor.ir?.element : editor.vditor.sv?.element
  return el ?? null
}

/** 从 caret 所在节点向前回溯至多 max 字符的纯文本（跨内联节点，忽略元素边界） */
/** wysiwyg/ir：光标是否位于渲染后的 code/pre/a 内（排除编辑元素根节点自身是 pre 的情况） */
function isInsideRenderedCode(node: Node, element: HTMLElement): boolean {
  const el = node instanceof Element ? node : node.parentElement
  if (!el) return false
  const hit = el.closest('code, pre, a')
  return hit !== null && hit !== element
}

/** sv：光标前的围栏（```/~~~）或行内反引号成对，奇数对则位于代码内 */
/** 当前光标处的 mention 上下文；caret 不在编辑器内或选区非折叠时返回 null */
function getMentionContext(): MentionCaretContext | null {
  const element = currentModeElement()
  if (!element) return null
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return null
  const range = selection.getRangeAt(0)
  if (!range.collapsed || !element.contains(range.startContainer)) return null
  const prefix = textBeforeCaret(range.startContainer, range.startOffset, Number.MAX_SAFE_INTEGER, element)
  const mode = editor!.vditor.currentMode
  const inCode = mode === 'sv' ? isInsideFencedCode(prefix) : isInsideRenderedCode(range.startContainer, element)
  const rect = range.getBoundingClientRect()
  return { prefix: prefix.slice(-MENTION_PREFIX_LIMIT), rect: rect.width === 0 && rect.height === 0 ? null : rect, inCode, element }
}

/**
 * 以光标为终点向前删除 queryLength 个字符（即 "@query" token）并插入 replacement。
 * 始终补尾部空格（对齐 GitHub/Slack 等主流 mention 交互）：保证插入后 token 立即被
 * 空白终止，会话不会因重新识别到 "@username" 而重开。execCommand 走原生
 * contenteditable 路径：触发 input 事件，Vditor 重渲染并同步 modelValue。
 */
function replaceMentionToken(queryLength: number, replacement: string): boolean {
  const element = currentModeElement()
  return element ? replaceMentionTokenInElement(element, queryLength, replacement) : false
}

/** 贴纸候选选中：整段替换 `[:sticker:query` 片段为完整 token，不补空格（贴纸本身就是间隔） */
function replaceStickerQuery(length: number, token: string): boolean {
  const element = currentModeElement()
  return element ? replaceMentionTokenInElement(element, length, token, false) : false
}

function getValue() {
  return editor && ready ? editor.getValue() : props.modelValue
}

function setValue(value: string) {
  if (!editor || !ready) return
  clearSensitiveHighlights()
  editor.setValue(value, true)
  if (value !== props.modelValue) emit('update:modelValue', value)
}

function syncValue() {
  const value = getValue()
  if (value !== props.modelValue) emit('update:modelValue', value)
  return value
}

defineExpose({ editorFailed, editorReady, focus, getValue, setValue, insertMarkdown, setHeight, syncValue, getMentionContext, replaceMentionToken, hintSettled })
</script>

<template>
  <!-- 用 data-locale 属性而非 :class 驱动语言样式：Vue 动态 class 重渲染会抹掉
       Vditor 命令式添加的 .vditor 类（编辑器边框/工具栏样式依赖它），属性则不影响 className -->
  <div
    v-bind="$attrs"
    ref="root"
    class="vditor-official"
    :class="{ 'vditor-official--slim': slimMobile }"
    :data-locale="locale"
    :data-gf-rich-profile="profile ?? 'post'"
  />
  <StickerSuggestions
    v-if="stickerPicker"
    :open="stickerSuggest.open.value"
    :query="stickerSuggest.query.value"
    :candidates="stickerSuggest.candidates.value"
    :active-index="stickerSuggest.activeIndex.value"
    :loading="stickerSuggest.loading.value"
    :panel-style="stickerSuggest.panelStyle.value"
    @select="stickerSuggest.select"
  />
  <EditorSelectionToolbar
    ref="selectionToolbar"
    :open="floating.selectionOpen.value"
    :host="floating.host.value"
    :anchor="floating.selectionAnchor.value"
    :boundary="floating.boundary.value"
    :available="floating.selectionAvailable.value"
    :active="floating.selectionActive.value"
    :heading="floating.selectionHeading.value"
    :prefer-below="floating.preferBelow"
    @command="floating.runCommand"
    @heading="floating.setHeading"
  />
  <EditorImageToolbar
    ref="imageToolbar"
    :host="floating.host.value"
    :image="floating.image.value"
    :boundary="floating.boundary.value"
    :width="floating.imageWidth.value"
    :resizable="floating.imageResizable.value"
    :alt="floating.imageAlt.value"
    @width="floating.setImageWidth"
    @alt="floating.setImageAlt"
    @preview="floating.previewImage()"
    @remove="floating.removeImage"
  />
  <MarkdownImageViewer ref="imageViewer" :to="floating.host.value ?? 'body'" />
</template>

<style>
/*
 * Tailwind preflight 会重置 ol/ul 的 list-style，而 vditor 官方 CSS
 * 只给 ul 补了 disc/circle/square，未补 ol 的 decimal（依赖浏览器默认）。
 * 这里按官方 ul 的写法补齐，仅作用于 vditor 编辑器内部。
 */
.vditor-official .vditor-reset ol {
  list-style-type: decimal;
}

/*
 * 设计系统映射（issue #1630）：Vditor 的外观全部由下列 CSS 变量驱动（官方 classic/dark
 * 主题写死 GitHub 灰与 #4285f4 蓝），这里统一改读站点 --gf-* token，深浅色随站点主题切换，
 * 不再依赖 .vditor--dark 的另一套色值。根节点本身就是 .vditor（Vditor 在挂载点上加类）。
 */
.vditor-official.vditor {
  --border-color: var(--gf-color-line);
  --second-color: color-mix(in oklch, var(--gf-color-base-content) 42%, transparent);
  --panel-background-color: var(--gf-color-base-100);
  --panel-shadow: 0 0 0 1px color-mix(in oklch, var(--gf-color-base-content) 8%, transparent), 0 8px 24px -8px oklch(0 0 0 / 0.18);
  /* 官方把它同时用作工具栏底色和菜单/面板的 hover 底色：这里取后者（不透明的浅底），
     工具栏本身的底色在 .vditor-toolbar 上单独设为编辑区同色 */
  --toolbar-background-color: color-mix(in oklch, var(--gf-color-base-content) 5%, var(--gf-color-base-100));
  --toolbar-icon-color: color-mix(in oklch, var(--gf-color-base-content) 62%, transparent);
  --toolbar-icon-hover-color: var(--gf-color-base-content);
  --textarea-background-color: var(--gf-color-base-100);
  --textarea-text-color: var(--gf-color-base-content);
  --resize-icon-color: var(--toolbar-icon-color);
  --resize-background-color: var(--gf-color-base-200);
  --resize-hover-icon-color: var(--gf-color-base-content);
  --resize-hover-background-color: var(--gf-color-base-300);
  --count-background-color: transparent;
  --heading-border-color: transparent;
  --blockquote-color: color-mix(in oklch, var(--gf-color-base-content) 72%, transparent);
  --ir-heading-color: var(--gf-color-primary);
  --ir-title-color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
  --ir-bi-color: var(--gf-color-base-content);
  --ir-link-color: var(--gf-color-primary);
  --ir-bracket-color: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
  --ir-paren-color: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
  --gf-editor-hover: color-mix(in oklch, var(--gf-color-base-content) 6%, transparent);
  --gf-editor-current: color-mix(in oklch, var(--gf-color-primary) 12%, transparent);
  border-color: var(--gf-color-line);
  border-radius: var(--gf-radius-box);
  background-color: var(--gf-color-base-100);
  font-family: inherit;
  transition-property: border-color, box-shadow;
  transition-duration: 0.15s;
  transition-timing-function: ease-out;
}

/* 拖入图片文件：内容区出现主色虚线框与淡底，提示松手即可上传（类名由 useEditorFloatingTools 切换） */
.vditor-official .vditor-content {
  transition-property: background-color, outline-color;
  transition-duration: 0.15s;
  transition-timing-function: ease-out;
  outline: 2px dashed transparent;
  outline-offset: -6px;
}

.vditor-official.gf-drop-active .vditor-content {
  outline-color: color-mix(in oklch, var(--gf-color-primary) 55%, transparent);
  background-color: color-mix(in oklch, var(--gf-color-primary) 4%, transparent);
}

/* 编辑中的焦点提示：整块编辑器描边转主色并加柔和外环，与站点输入框一致 */
.vditor-official.vditor:focus-within {
  border-color: color-mix(in oklch, var(--gf-color-primary) 55%, var(--gf-color-line));
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--gf-color-primary) 12%, transparent);
}

.vditor-official.vditor.vditor--fullscreen {
  border-radius: 0;
  box-shadow: none;
}

/* 工具栏底色与编辑区同面，用一条结构分隔线分开；圆角与外框同心（外框 1px 描边内侧） */
.vditor-official .vditor-toolbar {
  background-color: var(--gf-color-base-100);
  border-bottom-color: var(--gf-color-line);
  border-radius: calc(var(--gf-radius-box) - 1px) calc(var(--gf-radius-box) - 1px) 0 0;
}

.vditor-official.vditor--fullscreen .vditor-toolbar {
  border-radius: 0;
}

.vditor-official .vditor-counter {
  color: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
  font-variant-numeric: tabular-nums;
}

/*
 * 编辑区正文：读取与发布后 .gf-prose-post 相同的 --gf-rich-* 档位变量（styles/prose.css），
 * 由根节点 data-gf-rich-profile 选择 post / thought / article。官方 index.css 与
 * content-theme 的规则未分层，这里用 .vditor-official 前缀提高特异性覆盖。
 */
.vditor-official .vditor-reset {
  color: var(--gf-color-base-content);
  font-family: inherit;
  font-size: var(--gf-rich-size, 0.9375rem);
  line-height: var(--gf-rich-leading, 1.75);
}

html[data-gf-font-body] .vditor-official .vditor-reset {
  font-size: var(--gf-font-size-body, var(--gf-rich-size));
  font-family: var(--gf-font-family-body, inherit);
}

.vditor-official .vditor-reset > :first-child {
  margin-top: 0;
}

.vditor-official .vditor-reset p {
  margin-block: var(--gf-rich-flow);
}

.vditor-official .vditor-reset :is(h1, h2, h3, h4, h5, h6) {
  margin-block: 1.4em 0.5em;
  padding-bottom: 0;
  border-bottom: 0;
  color: var(--gf-color-base-content);
  font-weight: 600;
  line-height: 1.35;
}

.vditor-official .vditor-reset h1 {
  font-size: var(--gf-rich-h1);
  letter-spacing: -0.01em;
}

.vditor-official .vditor-reset h2 {
  font-size: var(--gf-rich-h2);
  letter-spacing: -0.005em;
}

.vditor-official .vditor-reset h3 {
  font-size: var(--gf-rich-h3);
}

.vditor-official .vditor-reset h4 {
  font-size: var(--gf-rich-h4);
}

.vditor-official .vditor-reset :is(h5, h6) {
  font-size: 1em;
}

.vditor-official .vditor-reset :is(ul, ol) {
  margin-block: var(--gf-rich-flow);
  padding-inline-start: 1.5em;
}

.vditor-official .vditor-reset li {
  margin-block: 0.25em;
}

.vditor-official .vditor-reset li > :is(p, ul, ol) {
  margin-block: 0.25em;
}

.vditor-official .vditor-reset blockquote {
  margin-block: var(--gf-rich-block);
  margin-inline: 0;
  padding-block: 0.125em;
  padding-inline: 1em 0;
  border-left: 0;
  border-inline-start: 3px solid color-mix(in oklch, var(--gf-color-base-content) 16%, transparent);
  color: color-mix(in oklch, var(--gf-color-base-content) 72%, transparent);
}

.vditor-official .vditor-reset blockquote p {
  margin-block: 0.4em;
}

.vditor-official .vditor-reset hr {
  height: 1px;
  margin-block: 2em;
  background-color: var(--gf-color-line);
}

.vditor-official .vditor-reset code:not(.hljs):not(.highlight-chroma) {
  padding: 0.125em 0.35em;
  border-radius: 0.25rem;
  background-color: color-mix(in oklch, var(--gf-color-base-content) 7%, transparent);
  color: color-mix(in oklch, var(--gf-color-base-content) 88%, transparent);
  font-size: 0.875em;
}

/* 代码块：与发布后的 .gf-prose-post pre 同底色、描边与圆角，去掉官方条纹背景 */
.vditor-official .vditor-reset pre > code {
  padding: 0.875em 1em;
  border: 1px solid var(--gf-color-line);
  border-radius: 0.5rem;
  background-color: var(--gf-color-base-200);
  background-image: none;
  font-size: 0.875em;
  line-height: 1.65;
}

.vditor-official .vditor-reset .vditor-wysiwyg__block,
.vditor-official .vditor-reset table,
.vditor-official .vditor-reset .language-mermaid {
  margin-block: var(--gf-rich-block);
}

.vditor-official .vditor-reset table tr {
  border-top: 0;
  background-color: transparent;
}

.vditor-official .vditor-reset table tbody tr:nth-child(2n) {
  background-color: transparent;
}

.vditor-official .vditor-reset table :is(td, th) {
  padding: 0.5em 0.75em;
  border: 0;
  border-bottom: 1px solid var(--gf-color-line);
}

.vditor-official .vditor-reset table th {
  border-bottom-width: 1.5px;
  background: color-mix(in oklch, var(--gf-color-base-content) 5%, transparent);
  font-weight: 600;
}

/*
 * 编辑区里的贴纸（sticker-inline-editor）：与正文 .gf-prose img[data-gf-sticker]
 * 同一个 2.5em 方框、同一条基线偏移，编辑时看到的行距与发布后一致。块内保留的
 * token 原文透明且推出方框外，只有背景图可见；不可编辑，整体选中/删除。
 */
.vditor-official .vditor-reset .gf-sticker-inline {
  display: inline-block;
  width: 2.5em;
  height: 2.5em;
  margin: 0 0.1em;
  overflow: hidden;
  vertical-align: -0.87em;
  color: transparent;
  white-space: nowrap;
  text-indent: 2.5em;
  background: center / contain no-repeat;
  cursor: default;
  -webkit-user-select: all;
  user-select: all;
}

.gf-link-preview-editor-hint {
  margin: 0;
  min-height: 1.75rem;
  padding: 0.35rem 0.6rem 0;
  color: color-mix(in oklch, var(--gf-color-base-content) 60%, transparent);
  font-size: 0.75rem;
  line-height: 1.25rem;
  overflow-wrap: anywhere;
}

.gf-link-preview-editor-hint[hidden] {
  display: none;
}

.vditor-official .vditor-reset ol ol {
  list-style-type: lower-alpha;
}

.vditor-official .vditor-reset ol ol ol {
  list-style-type: lower-roman;
}

/* 敏感词命中时只给现有段落着色，不包裹文本节点，避免破坏 Vditor 选区与序列化。 */
.vditor-official .gf-sensitive-block {
  background-color: color-mix(in oklch, var(--gf-color-error) 14%, var(--gf-color-base-100));
  box-shadow: inset 0 -1px 0 color-mix(in oklch, var(--gf-color-error) 64%, transparent);
}

[data-theme="gf-dark"] .vditor-official .gf-sensitive-block {
  background-color: color-mix(in oklch, var(--gf-color-error) 26%, var(--gf-color-base-100));
  box-shadow: inset 0 -1px 0 color-mix(in oklch, var(--gf-color-error) 82%, transparent);
}

/*
 * preflight 把 a 重置为 color:inherit + text-decoration:inherit，
 * vditor 官方只定义了 cursor:pointer。与发布后的正文链接一致：主色、无下划线（#1371）。
 */
.vditor-official .vditor-reset a {
  color: var(--gf-color-primary);
  font-weight: 500;
  text-decoration-line: none;
}

/*
 * 编辑区图片：与发布后同一组档位（居中、圆角、低透明度外描边、--gf-rich-media 间距与
 * 高度上限），所见即所得；emoji 保持官方 20px 特例。
 */
.vditor-official .vditor-reset img:not(.emoji):not(.twemoji) {
  display: block;
  max-width: 100%;
  max-height: var(--gf-rich-media-max-h, 360px);
  margin: var(--gf-rich-media, 1em) auto;
  border-radius: 0.5rem;
  outline: 1px solid oklch(0 0 0 / 0.1);
  outline-offset: -1px;
  object-fit: contain;
}

[data-theme='gf-dark'] .vditor-official .vditor-reset img:not(.emoji):not(.twemoji) {
  outline-color: oklch(1 0 0 / 0.1);
}

/* 相对宽度（MADR 0131）：与发布后 prose.css 同一规则，所见即所得 */
.vditor-official .vditor-reset img:is([src$='#w=25'], [src$='#w=33'], [src$='#w=50'], [src$='#w=66'], [src$='#w=75'], [src$='#w=100']):not(.emoji):not(.twemoji) {
  width: max(var(--gf-image-w), min(100%, 10rem));
  height: auto;
  max-height: none;
}

.vditor-official .vditor-reset img[src$='#w=25'] { --gf-image-w: 25%; }
.vditor-official .vditor-reset img[src$='#w=33'] { --gf-image-w: 33.333%; }
.vditor-official .vditor-reset img[src$='#w=50'] { --gf-image-w: 50%; }
.vditor-official .vditor-reset img[src$='#w=66'] { --gf-image-w: 66.667%; }
.vditor-official .vditor-reset img[src$='#w=75'] { --gf-image-w: 75%; }
.vditor-official .vditor-reset img[src$='#w=100'] { --gf-image-w: 100%; }

.vditor-official .vditor-reset p:has(> img[src*='#w='] + img[src*='#w=']) {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: center;
  gap: 0.5rem;
}

.vditor-official .vditor-reset p:has(> img[src*='#w='] + img[src*='#w=']) > img[src*='#w='] {
  width: calc(max(var(--gf-image-w), min(100%, 10rem)) - 0.5rem);
  margin: 0;
}

/* 编辑区与预览的占位提示：弱化文字色，跟随主题 */
.vditor-official .vditor-reset:empty::before {
  color: color-mix(in oklch, var(--gf-color-base-content) 38%, transparent);
}

/*
 * 大纲开关：随面板移动的展开/收起按钮。
 * 展开态：嵌在大纲标题栏右侧，跟随官方标题栏布局；
 * 收起态：悬浮在左侧大纲原位置，保证收起后仍可展开。
 */
.vditor-official .hub-outline-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 25px;
  height: 25px;
  margin-left: auto;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--toolbar-icon-color, var(--second-color));
  cursor: pointer;
}

.vditor-official .hub-outline-toggle svg {
  width: 15px;
  height: 15px;
  fill: currentColor;
}

.vditor-official .hub-outline-toggle:hover {
  color: var(--toolbar-icon-hover-color, var(--toolbar-icon-color));
  background: var(--gf-editor-hover, color-mix(in oklch, var(--gf-color-base-content) 6%, transparent));
}

.vditor-official .hub-outline-toggle.vditor-menu--current {
  color: var(--toolbar-icon-hover-color, var(--toolbar-icon-color));
}

/* 收起态：悬浮在编辑区左上角内侧，距左边框与上边框（编辑区顶部）各 6px，两距离相等。
   边框对齐编辑框 1px，小圆角，无阴影。 */
.vditor-official .hub-outline-toggle--float {
  position: absolute;
  top: calc(var(--toolbar-height, 35px) + 6px);
  left: 6px;
  z-index: 5;
  margin-left: 0;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--textarea-background-color);
  box-shadow: none;
}

.vditor-official .hub-outline-toggle--float:hover {
  background: var(--gf-editor-hover, color-mix(in oklch, var(--gf-color-base-content) 6%, transparent));
}

/*
 * 大纲展开/收起动画：宽度 250↔0 + 透明度过渡（0.3s < 0.5s）。
 * 宽度过渡驱动 flex 布局平滑重排（编辑区随大纲平滑变宽/变窄），
 * 配合 pre.vditor-reset 的 padding 过渡（JS 同步设目标值），
 * 正文始终在居中区域内平滑移动，无瞬间跳转。
 * 收起由 JS 在宽度过渡结束后再 display:none。
 */
.vditor-official .vditor-outline {
  transition: width 0.3s ease, opacity 0.3s ease;
  /* 宽度过渡期间横向裁剪；纵向保留 vditor 原有滚动能力 */
  overflow-x: hidden;
  overflow-y: auto;
}

/* 编辑区内容随大纲显隐平滑左右移动（padding 由 JS 动画设为最终目标值） */
.vditor-official .vditor-wysiwyg pre.vditor-reset {
  transition: padding 0.3s ease;
}

/* 标题栏改为 flex：让开关按钮靠右对齐 */
.vditor-official .vditor-outline__title {
  display: flex;
  align-items: center;
}

/*
 * 「向上扩展」按钮：悬浮在大纲列正上方（工具栏左侧空白区），
 * 与工具栏工具完全分离；大纲收起时隐藏避免遮挡编辑工具。
 */
.vditor-official .hub-header-toggle-float {
  position: absolute;
  top: 0;
  left: 0;
  /* 高于工具栏（40）保证悬浮在工具栏之上；低于全局弹窗（100），
     弹窗打开时按钮被遮罩盖住不悬浮在弹窗上 */
  z-index: 60;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 35px;
  height: var(--toolbar-height, 35px);
  padding: 0;
  border: 0;
  border-right: 1px solid var(--border-color);
  background: transparent;
  color: var(--toolbar-icon-color);
  cursor: pointer;
}

.vditor-official .hub-header-toggle-float svg {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 15px;
  height: 15px;
  fill: none;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
  transition: opacity 0.15s ease;
}

/* 收起态（vditor-menu--current）箭头换向：向上（收起）→ 向下（展开） */
.vditor-official .hub-header-toggle-float .hub-header-toggle-icon--down {
  opacity: 0;
}

.vditor-official .hub-header-toggle-float.vditor-menu--current .hub-header-toggle-icon--up {
  opacity: 0;
}

.vditor-official .hub-header-toggle-float.vditor-menu--current .hub-header-toggle-icon--down {
  opacity: 1;
}

.vditor-official .hub-header-toggle-float:hover {
  background: var(--gf-editor-hover, color-mix(in oklch, var(--gf-color-base-content) 6%, transparent));
  color: var(--toolbar-icon-hover-color);
}

.vditor-official .hub-header-toggle-float.vditor-menu--current {
  background: var(--gf-editor-current, color-mix(in oklch, var(--gf-color-primary) 12%, transparent));
  color: var(--gf-color-primary);
}

.vditor-official .hub-header-toggle-float.is-hidden {
  display: none;
}

/* ===== 知乎式「图标+小字」工具栏（保留官方 hover tooltip） ===== */
.vditor-official {
  /* 图标(15) + 标签(10) + 间距 + 内边距 ≈ 36px，取 40px 仅比官版 35px 高 5px */
  --toolbar-height: 40px;
}

/* 按钮改纵向布局：图标在上、小字标签在下，hover/focus 反馈与激活态沿用官方语义；
   width 100% 跟随 item 弹性分配（自适应宽度），min-width 21px 保底（2 字标签 20px） */
.vditor-official .vditor-toolbar__item .vditor-tooltipped {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  width: 100%;
  min-width: 21px;
  /* 不参与 flex 收缩：按钮多时保持等宽，禁止挤压 */
  flex-shrink: 0;
  height: var(--toolbar-height);
  margin: 0;
  padding: 4px 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--toolbar-icon-color, var(--second-color));
  box-sizing: border-box;
  transition-property: color, background-color, scale;
  transition-duration: 0.15s;
  transition-timing-function: ease-out;
}

/* 按下反馈：轻微缩放，给工具按钮触感 */
.vditor-official .vditor-toolbar__item .vditor-tooltipped:active {
  scale: 0.96;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped.vditor-menu--disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped svg {
  flex: none;
  width: 15px;
  height: 15px;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped:hover,
.vditor-official .vditor-toolbar__item .vditor-tooltipped:focus {
  background: var(--gf-editor-hover, color-mix(in oklch, var(--gf-color-base-content) 6%, transparent));
  color: var(--toolbar-icon-hover-color, var(--toolbar-icon-color));
  cursor: pointer;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped.vditor-menu--current {
  background: var(--gf-editor-current, color-mix(in oklch, var(--gf-color-primary) 12%, transparent));
  color: var(--gf-color-primary);
}

/* 工具分组的分隔线用站点 line 色 */
.vditor-official .vditor-toolbar > .vditor-toolbar__divider {
  border-left-color: var(--gf-color-line);
}

/* 小字标签：10px，单行不换行；短词不会撑宽（按钮定宽 36px 已容纳 2-3 字） */
.vditor-official .vditor-toolbar__item .vditor-toolbar-label {
  white-space: nowrap;
  font-size: 10px;
  font-weight: 500;
  line-height: 1.2;
  color: inherit;
  opacity: 0.9;
}

/* ===== 英文/德文标签更宽：10px 下溢出/截断，缩到 9px 保持单行 =====
 * 仅命中主行「图标下小字」（选择器粒度与 L1428 图标-only 隐藏规则一致），
 * 与 more 子菜单 / 折叠区 12px 文字（L1410-1417 / L1459-1466）正交。 */
.vditor-official[data-locale='en'] .vditor-toolbar > .vditor-toolbar__item > .vditor-tooltipped .vditor-toolbar-label,
.vditor-official[data-locale='de'] .vditor-toolbar > .vditor-toolbar__item > .vditor-tooltipped .vditor-toolbar-label {
  font-size: 9px;
}

/* 上传按钮内嵌 file input：随新按钮高度铺满 */
.vditor-official .vditor-toolbar__item input[type='file'] {
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
}

/* 工具栏高度跟随新按钮（官方默认 35px，这里随 --toolbar-height 提升）；
   flex 布局替代官方 float：按钮 flex:1 弹性均分富余空间（自适应：
   视口宽/大纲收起时按钮自动变宽，空间紧时自动收紧到 21px 保单行） */
.vditor-official .vditor-toolbar {
  display: flex;
  flex-wrap: nowrap;
  height: var(--toolbar-height);
}

/* 字数统计垂直居中：vditor 默认 margin(上 8px 下 0)+flex stretch 会把背景块
   拉到紧贴工具栏底边（上方留 8px、下方几乎贴边），改为 flex 垂直居中上下对称 */
.vditor-official .vditor-counter {
  align-self: center;
  margin: 0 3px 0 0;
}

/* 按钮间距：margin 实现 5px 均匀空隙；最后一个按钮不追加 margin（右边 padding 已够） */
.vditor-official .vditor-toolbar__item {
  flex: 1 1 0;
  /* 与按钮 min-width 一致：空间不足时收紧到 21px，不再换行溢出 */
  min-width: 21px;
  margin-right: 5px;
}

/* 移动端有未知名来源（实测非官方 CSS）给 item 加 0 12px 左右 padding，
   会把按钮撑到 64px 导致换行；归零，间距统一由 margin 控制 */
.vditor-official .vditor-toolbar__item {
  padding-left: 0;
  padding-right: 0;
}

.vditor-official .vditor-toolbar__item:last-child {
  margin-right: 0;
}

/* 分隔线不参与弹性分配（保持 1px 线宽） */
.vditor-official .vditor-toolbar__divider {
  flex: 0 0 auto;
}

/* 分隔线：官方水平 8px margin 收窄，左 4px、右 5px（与按钮间距一致）：
   组间留白（≈14px）至少是组内按钮间距（5px）的两倍，分组靠留白而不只靠竖线 */
.vditor-official .vditor-toolbar__divider {
  margin-left: 4px;
  margin-right: 5px;
}

/* toggle 移入宿主行内容器（移动端「正文」label 行）：
   按钮在 .vditor 之外，样式必须用全局选择器（.vditor-official 前缀不匹配），
   自带完整图标尺寸与箭头切换，非悬浮、紧凑 25px */
.hub-toggle--inline {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  position: relative;
  width: 25px;
  height: 25px;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--toolbar-icon-color, var(--second-color));
  cursor: pointer;
}

.hub-toggle--inline svg {
  width: 15px;
  height: 15px;
}

.hub-toggle--inline:hover {
  background: var(--gf-editor-hover, color-mix(in oklch, var(--gf-color-base-content) 6%, transparent));
  color: var(--toolbar-icon-hover-color, var(--toolbar-icon-color));
}

/* header-toggle 双图标：非悬浮时静态排列，按激活态切换显示 */
.hub-toggle--inline .hub-header-toggle-icon--down {
  display: none;
}

.hub-toggle--inline.vditor-menu--current .hub-header-toggle-icon--up {
  display: none;
}

.hub-toggle--inline.vditor-menu--current .hub-header-toggle-icon--down {
  display: inline;
}

/* 移动端大纲开关：不提供大纲唤起，隐藏 */
.hub-outline-toggle.is-hidden {
  display: none;
}

/* 移动端：恢复官方 sweet-mobile 触控尺寸（44px 高 / 40px 宽按钮），
   按钮不参与弹性分配（固定 40px，5 个工具一行铺满）。
   大纲开关收起态移入工具栏行内（header-toggle 右侧 35-60px），
   不再悬浮在编辑区顶部遮挡 placeholder。 */
@media (max-width: 520px) {
  .vditor-official {
    --toolbar-height: 44px;
  }

  .vditor-official .vditor-toolbar {
    overflow: visible;
    flex-wrap: nowrap !important;
    scrollbar-width: none;
    gap: 3px;
    padding: 5px 8px;
    align-items: center;
    background: color-mix(in oklch, var(--gf-color-base-200) 40%, var(--gf-color-base-100));
    border-bottom: 1px solid color-mix(in oklch, var(--gf-color-base-content) 10%, transparent);
    border-radius: 12px 12px 0 0;
  }

  .vditor-official .vditor-toolbar::-webkit-scrollbar {
    display: none;
  }

  .vditor-official .vditor-toolbar__item {
    /* 防止子元素溢出重叠：紧凑 32px，外边距归零使用 gap 统一间距 */
    flex: 0 0 auto;
    min-width: 32px;
    margin: 0;
  }

  .vditor-official .vditor-toolbar__item .vditor-tooltipped {
    width: 32px;
    min-width: 32px;
    height: 32px;
    padding: 5px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    color: color-mix(in oklch, var(--gf-color-base-content) 75%, transparent);
    transition-property: color, background-color, scale;
    transition-duration: 0.15s;
    transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
  }

  .vditor-official .vditor-toolbar__item .vditor-tooltipped:hover {
    background-color: color-mix(in oklch, var(--gf-color-base-content) 10%, transparent);
    color: var(--gf-color-base-content);
  }

  .vditor-official .vditor-toolbar__item .vditor-tooltipped:active {
    scale: 0.96;
    background-color: var(--gf-editor-current, color-mix(in oklch, var(--gf-color-primary) 12%, transparent));
  }

  .vditor-official .vditor-toolbar__item .vditor-tooltipped svg {
    width: 17px;
    height: 17px;
  }

  .vditor-official .hub-outline-toggle--float {
    top: 0;
    left: 35px;
    height: var(--toolbar-height);
  }
}

/* 移动触控端关闭悬浮文字气泡（触控点击后不会残留黑色气泡遮挡界面） */
@media (hover: none), (max-width: 640px) {
  .vditor-official .vditor-toolbar__item .vditor-tooltipped::after,
  .vditor-official .vditor-toolbar__item .vditor-tooltipped::before {
    display: none !important;
  }
}

/* more 子菜单自定义项（行间公式/大公式）：图标+文字横排，
   与官方文字菜单项（Headings、Line 等）视觉一致 */
.vditor-official .vditor-hint button svg {
  vertical-align: middle;
}

.vditor-official .vditor-hint button .vditor-toolbar-label {
  display: inline-block;
  margin-left: 6px;
  font-size: 12px;
  font-weight: 400;
  line-height: 1.4;
  vertical-align: middle;
}

/* ===== 工具栏图标窄屏收纳到 more 子菜单 ===== */
/* 折叠区容器：位于 more 子菜单最顶部；空时隐藏，不占子菜单空间 */
.vditor-official .hub-fold-region:empty {
  display: none;
}

/* ===== 窄屏只显示图标：主行按钮隐藏文字标签（保留 more 子菜单里的文字行） =====
 * 仅命中主行按钮（item 直接子级 button 内的 label）；折叠区位于 .vditor-hint 内，
 * 其 label 不在「item > button」直系，不受影响。 */
.vditor-official.hub-toolbar-icon-only .vditor-toolbar > .vditor-toolbar__item > .vditor-tooltipped .vditor-toolbar-label {
  display: none;
}

/* 折叠项：恢复为纵向堆叠的菜单行（脱离主行 flex 的 21px 均分/横排小字） */
.vditor-official .hub-fold-region .vditor-toolbar__item {
  position: relative; /* 官方 .vditor-toolbar__item 本就有，保留给 upload input / 二级面板定位 */
  flex: none;
  width: auto;
  min-width: 0;
  margin: 0;
  padding: 0;
}

/* 折叠项按钮：与「更多」菜单行（.gf-menu-row）同一行样式 */
.vditor-official .hub-fold-region .vditor-toolbar__item .vditor-tooltipped {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  width: 100%;
  min-width: 0;
  height: auto;
  min-height: 32px;
  margin: 0;
  padding: 0 10px;
  box-sizing: border-box;
  border-radius: calc(var(--gf-radius-box) - 4px);
  color: color-mix(in oklch, var(--gf-color-base-content) 80%, transparent);
  font-size: 0.875rem;
  font-weight: 500;
}

.vditor-official .hub-fold-region .vditor-toolbar__item .vditor-tooltipped svg {
  flex: none;
  width: 16px;
  height: 16px;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
}

.vditor-official .hub-fold-region .vditor-toolbar__item .vditor-toolbar-label {
  display: block;
  min-width: 0;
  margin: 0;
  overflow: hidden;
  font-size: inherit;
  font-weight: inherit;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
  opacity: 1;
}

/* 主行原次序里跨组的折叠项：上方加一条与菜单分组相同的细线；region 首项不加 */
.vditor-official .hub-fold-region .vditor-toolbar__item.hub-fold-group-start {
  margin-block-start: 4px;
  padding-block-start: 4px;
  border-block-start: 1px solid var(--gf-color-line);
  margin-inline: -4px;
  padding-inline: 4px;
}

.vditor-official .hub-fold-region .vditor-toolbar__item.hub-fold-group-start:first-child {
  margin-block-start: 0;
  padding-block-start: 0;
  border-block-start: 0;
}

/* 折叠项在子菜单内关闭 hover 气泡（行内已有文字标签，气泡冗余且会与面板重叠） */
.vditor-official .hub-fold-region .vditor-toolbar__item .vditor-tooltipped::after,
.vditor-official .hub-fold-region .vditor-toolbar__item .vditor-tooltipped::before {
  display: none;
}

/* upload 折叠进子菜单：file input 铺满整行（整行可点开文件选择器） */
.vditor-official .hub-fold-region .vditor-toolbar__item input[type='file'] {
  height: 100%;
}

/*
 * hover 提示气泡：修复三处
 * 1. 方向改下方：回复面板内工具栏位于滚动容器（overflow-y-auto）顶部，
 *    官方默认上方气泡超出容器上边界被裁剪；下方是编辑区空间充足。
 * 2. 宽度 max-content：absolute 气泡 shrink-to-fit 被包含块（按钮仅 27px）
 *    限制，导致只显示第一个字；显式按内容撑宽。
 * 3. 居中于按钮（left 50% + translateX(-50%)）：相比右缘对齐（right: 50%）
 *    向左越界更小，发布页无 overflow 容器可完整溢出显示，
 *    回复面板越界最小化。
 * pin 工具栏 z-index 40：必须低于顶栏 header（z-50，用户菜单 z-70 在其内），
 * 否则滚动吸顶时工具栏会盖住 header 与用户菜单；同时验证仍盖住编辑区
 * placeholder 图层（气泡可见）。
 */
.vditor-official .vditor-toolbar--pin {
  z-index: 40;
}

/* 气泡配色走站点 tooltip token（浅色/深色主题统一深灰底 + 近白字） */
.vditor-official .vditor-tooltipped::after {
  border-radius: 6px;
  background: var(--gf-color-tooltip-bg);
  color: var(--gf-color-tooltip-content);
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped::after {
  top: 100%;
  bottom: auto;
  left: calc(50% + var(--hub-tip-shift, 0px));
  right: auto;
  transform: translateX(-50%);
  width: max-content;
  white-space: nowrap;
  margin-top: 5px;
  margin-bottom: 0;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped::before {
  top: auto;
  bottom: -5px;
  left: 50%;
  right: auto;
  margin-left: -5px;
  border-bottom-color: var(--gf-color-tooltip-bg);
  border-top-color: transparent;
}

/* 官方 tooltip-appear 动画会覆盖 transform（translateX(-50%) 丢失导致跳位），
   用含位移的关键帧替代 */
.vditor-official .vditor-toolbar__item .vditor-tooltipped:hover::after,
.vditor-official .vditor-toolbar__item .vditor-tooltipped--hover::after,
.vditor-official .vditor-toolbar__item .vditor-tooltipped:active::after,
.vditor-official .vditor-toolbar__item .vditor-tooltipped:focus::after {
  animation-name: hub-tooltip-appear;
  animation-duration: 0.15s;
  animation-fill-mode: forwards;
  animation-timing-function: ease-in;
}

@keyframes hub-tooltip-appear {
  from {
    opacity: 0;
    transform: translate(-50%, 10px);
  }
  to {
    opacity: 1;
    transform: translate(-50%, 0);
  }
}

/* 最左侧按钮 tooltip：靠左对齐，杜绝向左溢出屏幕或容器边界 */
.vditor-official .vditor-toolbar__item:first-child .vditor-tooltipped::after {
  left: 0 !important;
  transform: none !important;
}
.vditor-official .vditor-toolbar__item:first-child .vditor-tooltipped::before {
  left: 12px !important;
}

/* 最右侧按钮 tooltip：靠右对齐，杜绝向右溢出屏幕或容器边界 */
.vditor-official .vditor-toolbar__item:last-child .vditor-tooltipped::after {
  left: auto !important;
  right: 0 !important;
  transform: none !important;
}
.vditor-official .vditor-toolbar__item:last-child .vditor-tooltipped::before {
  left: auto !important;
  right: 12px !important;
}

/* 底部工具栏弹层（如瞬间与提问 QuickPublishModal）：tooltip 向上弹出，避免被弹层底部切割 */
.gf-modal-editor .vditor-toolbar__item .vditor-tooltipped::after {
  top: auto !important;
  bottom: 100% !important;
  margin-top: 0 !important;
  margin-bottom: 5px !important;
}
.gf-modal-editor .vditor-toolbar__item .vditor-tooltipped::before {
  top: -5px !important;
  bottom: auto !important;
  border-top-color: var(--gf-color-tooltip-bg) !important;
  border-bottom-color: transparent !important;
}

/* 工具栏下拉面板与提示列表（emoji、more、headings 等）：提升层级至 100，高于吸顶工具栏与各容器，解决被裁切与穿透遮挡 */
.vditor-official .vditor-toolbar__item > .vditor-panel,
.vditor-official .vditor-toolbar__item > .vditor-hint {
  z-index: 100 !important;
  opacity: 1 !important;
  /* 与站点下拉菜单 .gf-menu-surface 同一表面：line 边框、box 圆角、同一投影；内边距 4px，
     菜单行圆角 = box 圆角 - 4px（同心圆角） */
  background-color: var(--gf-color-base-100) !important;
  border: var(--gf-border, 1px) solid var(--gf-color-line) !important;
  border-radius: var(--gf-radius-box) !important;
  box-shadow:
    0 18px 40px -24px rgb(15 23 42 / calc(var(--gf-depth, 1) * 0.45)),
    0 4px 12px -6px rgb(15 23 42 / calc(var(--gf-depth, 1) * 0.16)) !important;
  padding: 4px !important;
  box-sizing: border-box !important;
  max-height: min(280px, 48vh) !important;
  overflow-y: auto !important;
  scrollbar-width: thin !important;
  top: calc(100% + 4px) !important;
  bottom: auto !important;
  left: 0 !important;
  right: auto !important;
}

/* 底部工具栏弹层编辑器（如瞬间与提问弹层）：工具栏下拉浮层自然向上展开进入编辑正文区 */
.gf-modal-editor .vditor-official .vditor-toolbar__item > .vditor-panel,
.gf-modal-editor .vditor-official .vditor-toolbar__item > .vditor-hint {
  top: auto !important;
  bottom: calc(100% + 8px) !important;
}

.gf-modal-editor .vditor-official .vditor-toolbar__item > .vditor-panel:before,
.gf-modal-editor .vditor-official .vditor-toolbar__item > .vditor-hint:before {
  top: auto !important;
  bottom: -14px !important;
  border-top-color: var(--panel-background-color, #fff) !important;
  border-bottom-color: transparent !important;
}

/* 当工具栏项靠近右侧时（Vditor 动态计算添加的 .vditor-panel--left 或末尾项）：向左展开，防止被右侧容器边界裁切 */
.vditor-official .vditor-toolbar__item > .vditor-panel.vditor-panel--left,
.vditor-official .vditor-toolbar__item > .vditor-hint.vditor-panel--left,
.vditor-official .vditor-toolbar__item:last-child > .vditor-hint,
.vditor-official .vditor-toolbar__item:last-child > .vditor-panel,
.vditor-official .vditor-toolbar__item:nth-last-child(2) > .vditor-hint,
.vditor-official .vditor-toolbar__item:nth-last-child(2) > .vditor-panel {
  right: 0 !important;
  left: auto !important;
}

.vditor-official .vditor-toolbar__item > .vditor-panel.vditor-panel--left:before,
.vditor-official .vditor-toolbar__item > .vditor-hint.vditor-panel--left:before,
.vditor-official .vditor-toolbar__item:last-child > .vditor-panel:before,
.vditor-official .vditor-toolbar__item:nth-last-child(2) > .vditor-panel:before {
  left: auto !important;
  right: 12px !important;
}

/*
 * ===== 「更多」菜单（issue #1630，decorateEditorMenus）=====
 * 行样式对齐站点 .gf-menu-item：图标 16px + 名称 + 右侧弱化快捷键；分组用整宽细线；
 * 编辑模式 / 标题是就地展开的单选组（组标题 + 选项），不再弹二级浮层。
 */
.vditor-official .vditor-toolbar__item > .vditor-hint.gf-editor-menu {
  min-width: 232px;
  max-width: min(340px, calc(100vw - 24px));
  /* --gf-menu-max-h：打开时按视口剩余空间写入（fitMenuVertical） */
  max-height: min(560px, var(--gf-menu-max-h, 72vh)) !important;
  line-height: 20px;
}

.vditor-official .vditor-toolbar__item > .vditor-hint.gf-editor-menu.gf-editor-menu--up {
  top: auto !important;
  bottom: calc(100% + 6px) !important;
}

/* 站点菜单不带指向箭头 */
.vditor-official .vditor-toolbar__item > .vditor-hint.gf-editor-menu::before {
  display: none !important;
}

.vditor-official .gf-editor-menu .gf-menu-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 32px;
  margin: 0;
  padding: 0 10px;
  box-sizing: border-box;
  border-radius: calc(var(--gf-radius-box) - 4px);
  color: color-mix(in oklch, var(--gf-color-base-content) 80%, transparent);
  font-size: 0.875rem;
  font-weight: 500;
  line-height: 20px;
  text-align: start;
  white-space: nowrap;
  transition-property: color, background-color;
  transition-duration: 0.15s;
  transition-timing-function: ease-out;
}

.vditor-official .gf-editor-menu .gf-menu-row > svg {
  flex: none;
  width: 16px;
  height: 16px;
  margin: 0;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
  transition: color 0.15s ease-out;
}

.vditor-official .gf-editor-menu .gf-menu-row > .vditor-toolbar-label {
  display: block;
  flex: 1 1 auto;
  min-width: 0;
  margin: 0;
  overflow: hidden;
  font-size: inherit;
  font-weight: inherit;
  line-height: inherit;
  text-overflow: ellipsis;
  opacity: 1;
}

.vditor-official .gf-editor-menu .gf-menu-kbd {
  flex: none;
  margin-inline-start: auto;
  padding: 0 0 0 12px;
  border: 0;
  background: none;
  color: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
  font-family: inherit;
  font-size: 0.75rem;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  box-shadow: none;
}

.vditor-official .gf-editor-menu .gf-menu-row:not(.vditor-menu--disabled, .gf-menu-group-title):hover {
  background-color: var(--gf-editor-hover) !important;
  color: var(--gf-color-base-content) !important;
}

.vditor-official .gf-editor-menu .gf-menu-row:not(.vditor-menu--disabled, .gf-menu-group-title):hover > svg {
  color: var(--gf-color-base-content);
}

.vditor-official .gf-editor-menu .gf-menu-row:focus-visible {
  outline: 2px solid var(--gf-color-primary);
  outline-offset: -2px;
}

/* 开关类（预览等）打开时 */
.vditor-official .gf-editor-menu .gf-menu-row.vditor-menu--current:not(.gf-menu-option) {
  background-color: var(--gf-editor-current) !important;
  color: var(--gf-color-primary) !important;
}

.vditor-official .gf-editor-menu .gf-menu-row.vditor-menu--current:not(.gf-menu-option) > svg {
  color: currentColor;
}

/* 不可用（如光标不在列表里时的缩进）：保留位置，降低存在感 */
.vditor-official .gf-editor-menu .gf-menu-row.vditor-menu--disabled {
  opacity: 0.4;
  cursor: default;
}

/* 分组细线：Vditor 的分隔符原是主行竖线，在菜单里改成整宽横线；首尾与连续的隐藏 */
.vditor-official .gf-editor-menu > .vditor-toolbar__divider {
  display: block;
  float: none;
  width: auto;
  height: 1px;
  margin: 4px -4px;
  padding: 0;
  border: 0 !important;
  background-color: var(--gf-color-line);
}

.vditor-official .gf-editor-menu > .vditor-toolbar__divider:first-child,
.vditor-official .gf-editor-menu > .hub-fold-region:empty + .vditor-toolbar__divider,
.vditor-official .gf-editor-menu > .vditor-toolbar__divider + .vditor-toolbar__divider,
.vditor-official .gf-editor-menu > .vditor-toolbar__divider:last-child {
  display: none;
}

/* 窄屏收进来的主行按钮在最上方，和菜单原有项之间隔一条线 */
.vditor-official .gf-editor-menu > .hub-fold-region:not(:empty) {
  margin: 0 -4px 4px;
  padding: 0 4px 4px;
  border-block-end: 1px solid var(--gf-color-line);
}

.vditor-official .gf-editor-menu > .hub-fold-region:not(:empty) + .vditor-toolbar__divider {
  display: none;
}

/* 单选组标题（编辑模式、标题）：小号弱化文字，不可点 */
.vditor-official .gf-editor-menu .gf-menu-row.gf-menu-group-title,
.vditor-official .gf-editor-menu .gf-menu-row.gf-menu-group-title:hover {
  min-height: 28px;
  padding-block: 6px 2px;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent) !important;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: default;
  background: none !important;
}

.vditor-official .gf-editor-menu .gf-menu-group-title > svg,
.vditor-official .gf-editor-menu .gf-menu-group-title > .gf-menu-kbd {
  display: none;
}

/* 组内选项就地展开：覆盖 Vditor 二级浮层的定位与 hidePanel 写入的 display:none */
.vditor-official .gf-editor-menu .gf-menu-group-title + .vditor-hint {
  position: static !important;
  display: block !important;
  min-width: 0;
  max-width: none;
  max-height: none !important;
  margin: 0 !important;
  padding: 0 !important;
  overflow: visible !important;
  border: 0 !important;
  border-radius: 0 !important;
  background: transparent !important;
  box-shadow: none !important;
}

.vditor-official .gf-editor-menu .gf-menu-group-title + .vditor-hint::before {
  display: none !important;
}

/* 编辑模式选项：前导位只在当前模式显示对勾，名称保持对齐 */
.vditor-official .gf-editor-menu .gf-menu-option > svg {
  opacity: 0;
  color: var(--gf-color-primary);
}

.vditor-official .gf-editor-menu .gf-menu-option.vditor-menu--current {
  color: var(--gf-color-base-content);
  background: none;
}

.vditor-official .gf-editor-menu .gf-menu-option.vditor-menu--current > svg {
  opacity: 1;
  color: var(--gf-color-primary);
}

/* 标题选项：H1–H6 一排等宽小块（按钮内保持纯文本，文字由 data-tag 生成） */
.vditor-official .gf-editor-menu .gf-menu-group-title + .vditor-hint:has(> button[data-tag]) {
  display: flex !important;
  gap: 2px;
  padding: 2px 0 2px !important;
}

.vditor-official .gf-editor-menu button[data-tag] {
  display: flex;
  flex: 1 1 0;
  align-items: center;
  justify-content: center;
  min-width: 0;
  height: 32px;
  padding: 0;
  border-radius: calc(var(--gf-radius-box) - 4px);
  color: color-mix(in oklch, var(--gf-color-base-content) 80%, transparent);
  font-size: 0;
  transition-property: color, background-color;
  transition-duration: 0.15s;
}

.vditor-official .gf-editor-menu button[data-tag]::before {
  content: attr(data-tag);
  font-size: 0.8125rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  text-transform: uppercase;
}

.vditor-official .gf-editor-menu button[data-tag]:hover {
  background-color: var(--gf-editor-hover) !important;
  color: var(--gf-color-base-content) !important;
}

.vditor-official .gf-editor-menu .gf-menu-group-title.vditor-menu--disabled + .vditor-hint {
  opacity: 0.4;
  pointer-events: none;
}

/* 触控：行高 40px，隐藏键盘快捷键 */
@media (hover: none), (max-width: 640px) {
  .vditor-official .gf-editor-menu .gf-menu-row,
  .vditor-official .hub-fold-region .vditor-toolbar__item .vditor-tooltipped {
    min-height: 40px;
  }

  .vditor-official .gf-editor-menu .gf-menu-group-title {
    min-height: 30px;
  }

  .vditor-official .gf-editor-menu button[data-tag] {
    height: 40px;
  }

  .vditor-official .gf-editor-menu .gf-menu-kbd {
    display: none;
  }
}

/*
 * 正文块浮层（引用、列表、表格、代码块、链接等，Vditor 按块定位）：与编辑器浮层
 * （styles/editor-float.css）同一表面——line 描边、同心圆角（字段圆角 + 2px 内边距）、同一投影。
 * 层级低于吸顶工具栏（z-40），滚动时不会盖住工具栏和站点顶栏。
 */
.vditor-official .vditor-wysiwyg > .vditor-panel {
  z-index: 30 !important;
  padding: 2px !important;
  opacity: 1;
  /* Vditor 把浮层顶边放在块顶上方 21px：上移 4px，让浮层底边不压住块的第一行文字 */
  translate: 0 -4px;
  border: var(--gf-border, 1px) solid var(--gf-color-line) !important;
  border-radius: calc(var(--gf-radius-field) + 2px) !important;
  background-color: var(--gf-color-base-100) !important;
  color: var(--gf-color-base-content);
  box-shadow:
    0 18px 48px -24px rgb(15 23 42 / calc(var(--gf-depth, 1) * 0.5)),
    0 4px 16px -12px rgb(15 23 42 / calc(var(--gf-depth, 1) * 0.35)) !important;
  max-width: calc(100% - 16px) !important;
  box-sizing: border-box !important;
  transition-property: opacity, translate;
  transition-duration: 0.12s;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}

/* 选区格式条或图片工具条出现时让位，避免两层浮层叠在一起（类名由 useEditorFloatingTools 切换） */
.vditor-official.gf-floating-active .vditor-wysiwyg > .vditor-panel {
  opacity: 0 !important;
  translate: 0 -2px;
  pointer-events: none;
}

.vditor-official .vditor-panel .vditor-input {
  height: 1.75rem !important;
  border: var(--gf-border, 1px) solid var(--gf-color-line) !important;
  border-radius: var(--gf-radius-field) !important;
  background-color: var(--gf-color-base-100) !important;
  color: var(--gf-color-base-content) !important;
  padding: 0 0.5rem !important;
  font-size: 0.8125rem !important;
  line-height: 1.4 !important;
  margin: 0 2px !important;
  box-sizing: border-box !important;
  transition-property: border-color, box-shadow !important;
  transition-duration: 0.15s !important;
}

.vditor-official .vditor-panel .vditor-input::placeholder {
  color: color-mix(in oklch, var(--gf-color-base-content) 42%, transparent);
}

.vditor-official .vditor-panel .vditor-input:focus {
  border-color: color-mix(in oklch, var(--gf-color-primary) 60%, var(--gf-color-line)) !important;
  outline: none !important;
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--gf-color-primary) 15%, transparent) !important;
}

/* 块浮层里的图标按钮：与格式条按钮同尺寸、同圆角、同 hover 底 */
.vditor-official .vditor-panel .vditor-icon {
  border-radius: var(--gf-radius-field) !important;
  padding: 5px !important;
  width: 26px !important;
  height: 26px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  color: color-mix(in oklch, var(--gf-color-base-content) 72%, transparent) !important;
  transition-property: background-color, color, scale !important;
  transition-duration: 0.12s !important;
}

.vditor-official .vditor-panel .vditor-icon:hover {
  background-color: var(--gf-editor-hover) !important;
  color: var(--gf-color-base-content) !important;
}

.vditor-official .vditor-panel .vditor-icon:active {
  scale: 0.96;
}

/* 移动视图下的面板与表情栏防裁切：紧凑高度、自适应视口宽度 */
@media (max-width: 640px) {
  .vditor-official .vditor-toolbar {
    overflow: visible !important;
  }
  /* Link fields must wrap inside the bounded inline popover. */
  .vditor-official .vditor-wysiwyg > .vditor-panel {
    white-space: normal;
    opacity: 1;
  }
  .vditor-official .vditor-panel {
    padding: 3px 4px !important;
    max-width: min(320px, calc(100vw - 24px)) !important;
  }
  .vditor-official .vditor-emojis {
    max-height: 160px !important;
  }
  /* 16px 防止 iOS Safari 聚焦时整页放大 */
  .vditor-official .vditor-panel .vditor-input {
    width: 7.5rem !important;
    max-width: 100% !important;
    height: 2rem !important;
    font-size: 16px !important;
  }
  .vditor-official .vditor-panel .vditor-icon {
    width: 28px !important;
    height: 28px !important;
  }
}

/*
 * slim（课评）编辑器面板避让：标题/表情按钮贴工具栏左缘，vditor 对左侧按钮的
 * 面板加 --left（right:0 锚定按钮右缘），宽面板会整体伸向视口外左侧被裁/溢出
 * （楼层回复编辑器表情为左一、面板默认向右展开故无此问题）。
 * 对齐 QuickPublishModal 的纯 CSS 避让：这两类面板改为左锚定 + 视口宽度封顶，
 * 表情面板左移一格（32px）与标题面板对齐编辑器左缘展开；箭头随锚定方向复位。
 * 仅作用于 slimMobile 宿主（.vditor-official--slim）+ 移动端视口，其他编辑器不受影响。
 * 注：标题面板类名是 .vditor-hint.vditor-panel--arrow（无 .vditor-panel），需单独匹配。
 */
@media (max-width: 520px) {
  .vditor-official--slim .vditor-toolbar__item:has(> [data-type='headings']) > .vditor-hint,
  .vditor-official--slim .vditor-toolbar__item:has(> [data-type='emoji']) > .vditor-panel {
    left: 0 !important;
    right: auto !important;
    max-width: min(320px, calc(100vw - 48px)) !important;
  }

  /* 表情是第二格（标题 32px）：左移一格与标题面板对齐编辑器左缘 */
  .vditor-official--slim .vditor-toolbar__item:has(> [data-type='emoji']) > .vditor-panel {
    left: -32px !important;
  }

  .vditor-official--slim
    .vditor-toolbar__item:has(> [data-type='headings'])
    > .vditor-hint.vditor-panel--arrow:before,
  .vditor-official--slim
    .vditor-toolbar__item:has(> [data-type='emoji'])
    > .vditor-panel.vditor-panel--arrow:before {
    left: 12px !important;
    right: auto !important;
  }
}
</style>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import luteUrl from 'vditor/dist/js/lute/lute.min.js?url'
import antIconUrl from 'vditor/dist/js/icons/ant.js?url'
import enUrl from 'vditor/dist/js/i18n/en_US.js?url'
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
import { useSiteTheme } from '@/runtime/site-theme'
import { useI18n } from 'vue-i18n'

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
const props = defineProps<{
  modelValue: string
  placeholder: string
  /** 编辑器固定高度（官方 demo 为 360） */
  height?: number
  /** 左侧大纲：按需开启（官方默认关闭） */
  outline?: boolean
  /** 字数统计：按需开启（官方默认关闭） */
  counter?: boolean
  /** 紧凑工具栏：浮动面板等窄容器只留高频按钮，其余收进 more（桌面端） */
  compact?: boolean
  /** 工具栏最左显示「向上扩展」按钮：折叠标题/分类区（发布页用） */
  headerToggle?: boolean
  /** 头部折叠态：用于按钮激活高亮 */
  headerCollapsed?: boolean
  /** 移动端 toggle 挂载容器：窄屏下悬浮开关移入宿主提供的行内容器（如「正文」label 行），
   *  避免占用工具栏宽度导致横向挤压；桌面忽略此值（保持悬浮布局） */
  toggleHost?: HTMLElement | null
}>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  input: []
  upload: [files: File[]]
  error: [error: Error]
  'toggle-header': []
}>()

const { t, te } = useI18n()
const { isDark } = useSiteTheme()
const root = ref<HTMLElement | null>(null)
const editorReady = ref(false)
let editor: Vditor | null = null
let destroyed = false
let ready = false
/** 监听全屏按钮 childList：官方用 innerHTML 换图标会冲掉按钮内小字 */
let fullscreenLabelObserver: MutationObserver | null = null

/**
 * 官方默认工具栏（vditor src/ts/util/Options.ts），调整：
 * 1. outline 从工具栏移除：开关改为大纲面板自带的按钮
 *    （展开时在标题栏右侧、收起时悬浮左缘，见 syncOutlineToggleHost）
 * 2. 去掉 record（录音）按钮
 * 3. 新增行间公式 / 块级公式自定义按钮（mathToolbarItems）
 * 4. 仅「插入上方/插入下方」低频项收进 more 子菜单：
 *    主行 24 个高频按钮在发布页容器宽度内刚好单行（KISS：功能不减，入口统一）
 * 其余与官版逐项一致。
 */
const OFFICIAL_TOOLBAR: Array<string | IMenuItem> = [
  'headings',
  'bold',
  'italic',
  'strike',
  'link',
  '|',
  'list',
  'ordered-list',
  'check',
  'outdent',
  'indent',
  '|',
  'quote',
  'line',
  'code',
  'inline-code',
  ...mathToolbarItems(),
  '|',
  // 插入类工具同组：表情 / 图片 / 表格
  'emoji',
  uploadToolbarItem(),
  'table',
  '|',
  'undo',
  'redo',
  '|',
  'fullscreen',
  'edit-mode',
  {
    name: 'more',
    // 官方默认 tipPosition 为 e（右向气泡），在浮动面板（overflow hidden）里会被裁剪；
    // 统一改为 n（上向气泡），在 vditor 横向边距内显示
    tipPosition: 'n',
    toolbar: ['both', 'insert-before', 'insert-after', '|', 'code-theme', 'content-theme', 'export', 'preview', 'devtools', 'info', 'help'],
  },
]

const languageAssets = {
  en: { lang: 'en_US', url: enUrl },
  it: { lang: 'en_US', url: enUrl },
  ja: { lang: 'ja_JP', url: jaUrl },
  zh: { lang: 'zh_CN', url: zhUrl },
} as const

/** 上传图片按钮：本站附件仅允许图片（Lucide image 图标，tip 覆盖官方「上传图片或文件」） */
function uploadToolbarItem(): IMenuItem {
  return {
    name: 'upload',
    icon: '<svg viewBox="0 0 24 24" style="fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',
    tip: t('editor.toolbar.uploadImageTip'),
    tipPosition: 'n',
  }
}

/**
 * 紧凑工具栏（浮动回复面板等窄容器）：主行只保留高频按钮，
 * 标题/缩进/公式/表格/全屏等全部收进 more 子菜单，保证尽量单行。
 */
const COMPACT_TOOLBAR: Array<string | IMenuItem> = [
  'bold',
  'italic',
  'strike',
  'link',
  '|',
  'list',
  'ordered-list',
  'check',
  '|',
  'quote',
  'code',
  'inline-code',
  '|',
  'emoji',
  uploadToolbarItem(),
  '|',
  'undo',
  'redo',
  '|',
  'edit-mode',
  {
    name: 'more',
    // 上向气泡：避免右向气泡超出浮动面板（overflow hidden）被裁剪
    tipPosition: 'n',
    toolbar: [
      'headings',
      'outdent',
      'indent',
      'line',
      ...mathToolbarItems(),
      'table',
      'insert-before',
      'insert-after',
      '|',
      'fullscreen',
      'both',
      'preview',
      'export',
      'devtools',
      'info',
      'help',
    ],
  },
]

/**
 * 官方移动端最佳配置（https://b3log.org/vditor/demo/sweet-mobile.html）：
 * 精简工具栏：emoji / link / upload / edit-mode + more（insert-after/fullscreen/preview/info/help）。
 */
const MOBILE_TOOLBAR: Array<string | IMenuItem> = [
  'emoji',
  uploadToolbarItem(),
  'link',
  'edit-mode',
  {
    name: 'more',
    // 上向气泡：避免右向气泡超出浮动面板（overflow hidden）被裁剪
    tipPosition: 'n',
    toolbar: ['insert-after', 'fullscreen', 'preview', 'info', 'help'],
  },
]

/** 移动端判定：与 vditor Constants.MOBILE_WIDTH 一致 */
const MOBILE_VIEWPORT_QUERY = '(max-width: 520px)'

/** 官方建议：移动端高度 = 屏高一半，防止软键盘弹起时布局跳动 */
function resolveHeight(): number {
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) {
    return Math.floor(window.innerHeight / 2)
  }
  return props.height ?? 360
}

/** 工具栏选择：移动端官方精简 > 桌面 compact 紧凑 > 桌面完整 */
function resolveToolbar(): Array<string | IMenuItem> {
  if (window.matchMedia(MOBILE_VIEWPORT_QUERY).matches) return MOBILE_TOOLBAR
  return props.compact ? COMPACT_TOOLBAR : OFFICIAL_TOOLBAR
}

/** 行间公式 / 块级公式：自定义按钮，插入 $...$ / $$...$$ 并交给 Lute 渲染。
 *  图标用全站 lucide 图标族（Sigma / SquareSigma，与 @lucide/vue 及上传按钮统一），
 *  内联 style 覆盖官方 toolbar svg 的 fill/stroke-width 规则。 */
function mathToolbarItems(): IMenuItem[] {
  const mathInline: IMenuItem = {
    name: 'math-inline',
    icon: '<svg viewBox="0 0 24 24" style="fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round"><path d="M18 7V5a1 1 0 0 0-1-1H6.5a.5.5 0 0 0-.4.8l4.5 6a2 2 0 0 1 0 2.4l-4.5 6a.5.5 0 0 0 .4.8H17a1 1 0 0 0 1-1v-2"/></svg>',
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
  const mathBlock: IMenuItem = {
    name: 'math-block',
    icon: '<svg viewBox="0 0 24 24" style="fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M16 8.9V7H8l4 5-4 5h8v-1.9"/></svg>',
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
  return [mathInline, mathBlock]
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
 *   文案走 i18n（editor.toolbar.*，见 locales/{zh,en,ja,it}.ts） */
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

/**
 * 在工具栏按钮内部补/保「图标下小字」（不挪到按钮外）。
 * 全屏等会 innerHTML 整按钮，需可重复调用。
 */
function ensureToolbarButtonLabel(item: HTMLElement) {
  const type = item.getAttribute('data-type') || ''
  const labelText =
    toolbarLabel(type) ||
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

  // 大纲收起时 vditor 会把工具栏 padding-left 恢复为 5px，
  // 悬浮按钮（header-toggle 0-35px）会盖住第一个工具，这里补偿 35px；
  // 大纲展开时不动（保留 vditor 设置的大纲占位宽度）
  const outline = editor?.vditor.outline
  const outlineShown = props.outline === true && outline?.element.style.display === 'block'
  const toolbar = root.value?.querySelector<HTMLElement>('.vditor-toolbar')
  if (toolbar && !outlineShown) {
    toolbar.style.paddingLeft = '35px'
  }
}

onMounted(async () => {
  if (!root.value) return
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
    emit('error', error instanceof Error ? error : new Error(String(error)))
    return
  }
  if (destroyed || !root.value) return

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
      // 所见即所得：显示悬浮在片段左侧的块级控制指示器（</>、H1/H2、$、ToC）
      mode: 'wysiwyg',
      placeholder: props.placeholder,
      // 必须始终提供 outline 对象（enable 为 false 也传完整对象）：
      // Vditor 的 merge 会保留显式 undefined，导致 initUI 读 options.outline.position 崩溃
      outline: { enable: props.outline === true, position: 'left' },
      counter: { enable: props.counter === true },
      theme: isDark.value ? 'dark' : 'classic',
      // 移动端用官方精简工具栏，桌面用完整工具栏
      toolbar: resolveToolbar(),
      toolbarConfig: { hide: false, pin: true },
      preview: {
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
        editorReady.value = true
        syncContentTheme()
        syncHighlightTheme()
        attachToolbarLabels()
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
        if (props.modelValue && props.modelValue !== nextEditor?.getValue()) {
          nextEditor?.setValue(props.modelValue, true)
        }
      },
      input(value) {
        emit('update:modelValue', value)
      },
    })
    editor = nextEditor
  } catch (error) {
    emit('error', error instanceof Error ? error : new Error(String(error)))
  }
})

watch(() => props.modelValue, (value) => {
  if (!editor || !ready || value === editor.getValue()) return
  editor.setValue(value, true)
})

watch(isDark, syncEditorTheme)

onBeforeUnmount(() => {
  destroyed = true
  fullscreenLabelObserver?.disconnect()
  fullscreenLabelObserver = null
  const currentEditor = editor
  editor = null
  editorReady.value = false
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

defineExpose({ editorReady, focus, getValue, insertMarkdown, setHeight, syncValue })
</script>

<template>
  <div ref="root" class="vditor-official" />
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

.vditor-official .vditor-reset ol ol {
  list-style-type: lower-alpha;
}

.vditor-official .vditor-reset ol ol ol {
  list-style-type: lower-roman;
}

/*
 * 正文文字色：官方 index.css 写死 .vditor-reset { color: #24292e }，
 * 不读 --textarea-text-color；.vditor--dark 只换 chrome token。
 * content-theme 异步加载前用 chrome token 立即跟主题，避免深色下仍是深色字。
 * 选择器加 .vditor-official 提高特异性，压过官方 content-theme 的硬编码色。
 */
.vditor-official .vditor-reset {
  color: var(--textarea-text-color);
}

/*
 * preflight 把 a 重置为 color:inherit + text-decoration:inherit，
 * vditor 官方只定义了 cursor:pointer（颜色/下划线依赖浏览器默认）。
 * 用官方链接色 #4285f4（与 link-ref/工具栏 hover 色一致）补齐。
 */
.vditor-official .vditor-reset a {
  color: #4285f4;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.vditor-official .vditor-reset a:hover {
  color: #3b76d6;
}

/*
 * 编辑区图片等比缩放：原图超宽/超高时收拢到适中尺寸（宽 80% / 高 ≤480px），
 * 小图不受影响；emoji 保持官方 20px 特例。
 */
.vditor-official .vditor-reset img:not(.emoji) {
  max-width: 80%;
  max-height: 480px;
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
  background: var(--toolbar-background-color, var(--panel-background-color));
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
  background: var(--toolbar-background-color);
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
  background: var(--toolbar-background-color);
  color: var(--toolbar-icon-hover-color);
}

.vditor-official .hub-header-toggle-float.vditor-menu--current {
  background: color-mix(in oklch, var(--gf-color-primary) 15%, transparent);
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
  border-radius: 4px;
  background: transparent;
  color: var(--toolbar-icon-color, var(--second-color));
  box-sizing: border-box;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped svg {
  flex: none;
  width: 15px;
  height: 15px;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped:hover,
.vditor-official .vditor-toolbar__item .vditor-tooltipped:focus {
  background: var(--toolbar-background-color, var(--panel-background-color));
  color: var(--toolbar-icon-hover-color, var(--toolbar-icon-color));
  cursor: pointer;
}

.vditor-official .vditor-toolbar__item .vditor-tooltipped.vditor-menu--current {
  background: color-mix(in oklch, var(--gf-color-primary) 15%, transparent);
  color: var(--gf-color-primary);
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

/* 分隔线：官方水平 8px margin 移除，间距统一由 toolbar gap 提供 */
.vditor-official .vditor-toolbar__divider {
  margin-left: 0;
  margin-right: 0;
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
  background: var(--toolbar-background-color, var(--panel-background-color));
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

  .vditor-official .vditor-toolbar__item {
    /* 自适应：按钮弹性均分富余空间，宽屏舒展、窄屏自动收紧（min-width 40px 触控保底） */
    flex: 1 1 0;
    min-width: 0;
  }

  .vditor-official .vditor-toolbar__item .vditor-tooltipped {
    width: 100%;
    min-width: 40px;
    padding: 8px 4px;
  }

  .vditor-official .hub-outline-toggle--float {
    top: 0;
    left: 35px;
    height: var(--toolbar-height);
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
  border-bottom-color: #3b3e43;
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
</style>

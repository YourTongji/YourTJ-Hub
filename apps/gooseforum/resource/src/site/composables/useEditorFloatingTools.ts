/**
 * 编辑器浮层工具（选区格式条 + 图片工具条，MADR 0131）的状态与事件。
 * Vditor 实例由宿主（VditorOfficial）持有，这里只通过访问器读取；格式命令交给 Vditor 自己的
 * 工具栏按钮执行，图片改动直接改编辑区 DOM 后记撤销栈并触发 input，与官方图片浮层同一路径。
 */
import { ref, shallowRef, watchEffect, type Ref } from 'vue'
import type Vditor from 'vditor'
import { floatingHost, intersectRect, rectVisibleIn, viewportRect, type FloatingRect } from '@/runtime/floating-position'
import {
  editableImage,
  editorImages,
  editorImageSource,
  insideRenderedBlock,
  removeEditorImage,
  setEditorImageAlt,
  setEditorImageSource,
  type EditorMode,
} from '@/runtime/editor-image-dom'
import { canSetImageWidth, imageWidthOf, withImageWidth, type ImageWidth } from '@/runtime/image-width'
import type { HeadingTag, SelectionCommand } from '@/site/components/EditorSelectionToolbar.vue'
import type { MarkdownPreviewImage } from '@/components/MarkdownImageViewer.vue'

interface FloatingToolsOptions {
  root: Ref<HTMLElement | null>
  editor: () => Vditor | null
  modeElement: () => HTMLElement | null
  composing: () => boolean
  /** 打开大图预览（编辑区全部图片，可左右切换） */
  preview: (images: MarkdownPreviewImage[], index: number) => void
  /** 浮层是否包含该节点（点击浮层不算点到外面） */
  insideFloating: (node: Node | null) => boolean
}

const COMMANDS: SelectionCommand[] = ['bold', 'italic', 'strike', 'inline-code', 'link', 'quote']
const ACTIVE_SELECTORS: Record<SelectionCommand, string> = {
  bold: 'strong, b, [data-type="strong"]',
  italic: 'em, i, [data-type="em"]',
  strike: 's, del, strike, [data-type="s"]',
  'inline-code': 'code, [data-type="code"]',
  link: 'a, [data-type="a"]',
  quote: 'blockquote',
}

/** Vditor 工具栏按钮的事件名（与 vditor util/compatibility.getEventName 一致） */
function vditorEventName() {
  return navigator.userAgent.includes('iPhone') ? 'touchstart' : 'click'
}

function toRect(rect: DOMRect): FloatingRect {
  return { top: rect.top, left: rect.left, right: rect.right, bottom: rect.bottom }
}

export function useEditorFloatingTools(options: FloatingToolsOptions) {
  const host = ref<HTMLElement | null>(null)
  const boundary = ref<FloatingRect | null>(null)
  const selectionOpen = ref(false)
  const selectionAnchor = ref<FloatingRect | null>(null)
  const selectionAvailable = ref<SelectionCommand[]>([])
  const selectionActive = ref<SelectionCommand[]>([])
  const selectionHeading = ref<number | null>(0)
  const image = shallowRef<HTMLImageElement | null>(null)
  const imageWidth = ref<ImageWidth | null>(null)
  const imageResizable = ref(true)
  const imageAlt = ref('')
  const preferBelow = typeof matchMedia === 'function' && matchMedia('(pointer: coarse)').matches

  let frame = 0
  let pointerSelecting = false

  // 格式条或图片工具条出现时，Vditor 跟随段落的块浮层（引用、列表、链接…）让位，避免两层叠在一起。
  // 只用 classList：宿主根节点的 class 由 Vditor 维护，不能交给 Vue 的 class 绑定覆盖。
  watchEffect(() => {
    options.root.value?.classList.toggle('gf-floating-active', selectionOpen.value || image.value !== null)
  })

  function mode(): EditorMode | null {
    return (options.editor()?.vditor.currentMode as EditorMode | undefined) ?? null
  }

  function toolbarElement(type: string): HTMLElement | null {
    return options.editor()?.vditor.toolbar?.elements?.[type] ?? null
  }

  /** 浮层可出现的区域：纵向为编辑内容区，横向可越出编辑区直到视口（窄屏上编辑区比工具条还窄）；
   *  再与挂载点相交，并去掉吸顶工具栏遮住的部分 */
  function computeBoundary(): FloatingRect | null {
    const element = options.modeElement()
    if (!element) return null
    const content = (element.closest('.vditor-content') ?? element) as HTMLElement
    const viewport = viewportRect()
    const contentRect = toRect(content.getBoundingClientRect())
    let rect = intersectRect(viewport, { ...contentRect, left: viewport.left, right: viewport.right })
    // 回复面板、弹层等会裁切内容的滚动容器：滚出容器的部分同样不可见
    for (let el = content.parentElement; el && el !== document.body && el !== document.documentElement; el = el.parentElement) {
      const style = getComputedStyle(el)
      if (style.overflowY !== 'visible') {
        const box = el.getBoundingClientRect()
        rect = { ...rect, top: Math.max(rect.top, box.top + el.clientTop), bottom: Math.min(rect.bottom, box.top + el.clientTop + el.clientHeight) }
      }
    }
    if (host.value && host.value !== document.body) rect = intersectRect(rect, toRect(host.value.getBoundingClientRect()))
    else rect = belowPageHeader(rect)
    const toolbar = options.editor()?.vditor.toolbar?.element
    if (toolbar && toolbar.offsetParent !== null) {
      const bar = toolbar.getBoundingClientRect()
      if (bar.bottom > rect.top && bar.top <= rect.top + 1) rect = { ...rect, top: bar.bottom }
    }
    return rect.bottom > rect.top && rect.right > rect.left ? rect : null
  }

  /** 页面吸顶顶栏盖住的部分不算可见（全屏编辑器盖在顶栏之上，不扣除） */
  function belowPageHeader(rect: FloatingRect): FloatingRect {
    if (options.root.value?.classList.contains('vditor--fullscreen')) return rect
    for (const header of document.querySelectorAll<HTMLElement>('body header')) {
      const position = getComputedStyle(header).position
      if (position !== 'fixed' && position !== 'sticky') continue
      const bar = header.getBoundingClientRect()
      if (bar.height && bar.top <= rect.top + 1 && bar.bottom > rect.top) rect = { ...rect, top: bar.bottom }
    }
    return rect
  }

  function hideSelection() {
    selectionOpen.value = false
  }

  function headingLevel(node: Node, element: HTMLElement): number | null {
    if (!toolbarElement('headings')) return null
    const el = node instanceof Element ? node : node.parentElement
    if (el?.closest('td, th')) return null
    const heading = el?.closest('h1, h2, h3, h4, h5, h6')
    return heading && element.contains(heading) ? Number(heading.tagName.slice(1)) : 0
  }

  function updateSelection() {
    const element = options.modeElement()
    const currentMode = mode()
    if (!element || !currentMode || currentMode === 'sv' || pointerSelecting || options.composing() || image.value) return hideSelection()
    const selection = window.getSelection()
    if (!selection || selection.rangeCount === 0 || selection.isCollapsed) return hideSelection()
    const range = selection.getRangeAt(0)
    if (!element.contains(range.commonAncestorContainer) || !range.toString().trim()) return hideSelection()
    if (insideRenderedBlock(range.startContainer, element) || insideRenderedBlock(range.endContainer, element)) return hideSelection()
    const rects = range.getClientRects()
    const box = range.getBoundingClientRect()
    const anchor = box.width || box.height ? toRect(box) : rects.length ? toRect(rects[0]) : null
    if (!anchor || !boundary.value || !rectVisibleIn(anchor, boundary.value)) return hideSelection()
    selectionAnchor.value = anchor
    selectionAvailable.value = COMMANDS.filter((type) => toolbarElement(type))
    const start = range.startContainer instanceof Element ? range.startContainer : range.startContainer.parentElement
    selectionActive.value = COMMANDS.filter((type) => {
      const hit = start?.closest(ACTIVE_SELECTORS[type])
      return Boolean(hit && hit !== element && element.contains(hit))
    })
    selectionHeading.value = headingLevel(range.startContainer, element)
    selectionOpen.value = true
  }

  function update() {
    frame = 0
    if (image.value && !image.value.isConnected) clearImage()
    // 键盘或程序选中了别处的文字：图片选中让位给格式条
    const selection = window.getSelection()
    if (image.value && selection && !selection.isCollapsed && selection.rangeCount && !selection.getRangeAt(0).intersectsNode(image.value)) clearImage()
    boundary.value = computeBoundary()
    updateSelection()
  }

  function schedule() {
    if (frame) return
    frame = requestAnimationFrame(update)
  }

  /** 改完编辑区 DOM：记撤销栈并按 Markdown 触发 input（与 Vditor 官方图片浮层一致） */
  function commit() {
    const instance = options.editor()
    if (!instance) return
    instance.vditor.undo?.addToUndoStack(instance.vditor)
    instance.vditor.options.input?.(instance.getValue())
  }

  function runCommand(command: SelectionCommand) {
    const button = toolbarElement(command)?.firstElementChild
    if (!(button instanceof HTMLElement)) return
    button.dispatchEvent(new Event(vditorEventName(), { bubbles: true, cancelable: true }))
    // Vditor 改写选区所在块后再定位一次（链接会打开 Vditor 自己的地址浮层，格式条让位）
    if (command === 'link') hideSelection()
    setTimeout(schedule, 0)
  }

  function setHeading(tag: HeadingTag | null) {
    const item = toolbarElement('headings')
    const action = item?.firstElementChild
    if (!item || !(action instanceof HTMLElement)) return
    const event = () => new Event(vditorEventName(), { bubbles: true, cancelable: true })
    if (tag) {
      item.querySelector<HTMLElement>(`[data-tag="${tag}"]`)?.dispatchEvent(event())
    } else if (selectionHeading.value) {
      // 「正文」= 取消标题：Vditor 以按钮高亮判断当前是否标题，高亮可能尚未刷新
      action.classList.add('vditor-menu--current')
      action.dispatchEvent(event())
    }
    setTimeout(schedule, 0)
  }

  function selectImage(img: HTMLImageElement) {
    const currentMode = mode()
    if (!currentMode) return
    const src = editorImageSource(img, currentMode)
    // 先前的文字选区收拢到图片后面：否则它会让图片选中立刻让位给格式条，键入也会覆盖那段文字
    const selection = window.getSelection()
    if (selection && !selection.isCollapsed) {
      const range = document.createRange()
      range.setStartAfter(img)
      range.collapse(true)
      selection.removeAllRanges()
      selection.addRange(range)
    }
    image.value = img
    imageWidth.value = imageWidthOf(src)
    imageResizable.value = canSetImageWidth(src)
    imageAlt.value = img.getAttribute('alt') ?? ''
    hideSelection()
    boundary.value = computeBoundary()
  }

  function clearImage() {
    image.value = null
  }

  function setImageWidth(width: ImageWidth | null) {
    const img = image.value
    const currentMode = mode()
    if (!img || !currentMode) return
    const next = withImageWidth(editorImageSource(img, currentMode), width)
    setEditorImageSource(img, next, currentMode)
    imageWidth.value = imageWidthOf(next)
    commit()
  }

  function setImageAlt(alt: string) {
    const img = image.value
    const currentMode = mode()
    if (!img || !currentMode) return
    setEditorImageAlt(img, alt, currentMode)
    imageAlt.value = alt
    commit()
  }

  function removeImage() {
    const img = image.value
    const currentMode = mode()
    if (!img || !currentMode) return
    clearImage()
    removeEditorImage(img, currentMode)
    commit()
  }

  function previewImage(target = image.value) {
    const element = options.modeElement()
    if (!element || !target) return
    const images = editorImages(element)
    options.preview(
      images.map((img) => ({ src: img.currentSrc || img.getAttribute('src') || '', alt: img.getAttribute('alt') ?? '' })),
      Math.max(0, images.indexOf(target)),
    )
  }

  // 捕获阶段先于 Vditor 的点击处理：单击图片只选中（不弹出 Vditor 的地址输入浮层），双击预览
  function handleClick(event: MouseEvent) {
    const element = options.modeElement()
    if (!element || mode() === 'sv') return
    const img = editableImage(event.target, element)
    if (!img) return
    event.preventDefault()
    event.stopPropagation()
    const popover = options.editor()?.vditor.wysiwyg?.popover
    if (popover) popover.style.display = 'none'
    selectImage(img)
  }

  function handleDoubleClick(event: MouseEvent) {
    const element = options.modeElement()
    if (!element || mode() === 'sv') return
    const img = editableImage(event.target, element)
    if (!img) return
    event.preventDefault()
    event.stopPropagation()
    selectImage(img)
    previewImage(img)
  }

  function handlePointerDown(event: PointerEvent) {
    const target = event.target as Node | null
    if (options.insideFloating(target)) return
    if (image.value && target !== image.value) clearImage()
    const element = options.modeElement()
    if (element && target && element.contains(target)) {
      pointerSelecting = true
      hideSelection()
    }
  }

  function handlePointerUp() {
    if (!pointerSelecting) return
    pointerSelecting = false
    // 等浏览器定稿选区（双击选词、拖选结束）再出现
    setTimeout(schedule, 0)
  }

  function handleKeydown(event: KeyboardEvent) {
    if (!image.value || event.isComposing) return
    if (event.key === 'Backspace' || event.key === 'Delete') {
      event.preventDefault()
      event.stopPropagation()
      removeImage()
    } else if (event.key === 'Escape') {
      event.preventDefault()
      event.stopPropagation()
      clearImage()
    } else if (!['Shift', 'Control', 'Alt', 'Meta', 'Tab'].includes(event.key)) {
      clearImage()
    }
  }

  // 拖入文件：编辑内容区出现虚线框提示松手即可上传（上传本身仍由 Vditor 处理）
  let dragDepth = 0

  function draggingFiles(event: DragEvent) {
    return Boolean(event.dataTransfer?.types.includes('Files')) && mode() !== null
  }

  function setDropActive(active: boolean) {
    options.root.value?.classList.toggle('gf-drop-active', active)
  }

  function handleDragEnter(event: DragEvent) {
    if (!draggingFiles(event)) return
    dragDepth += 1
    setDropActive(true)
  }

  function handleDragLeave(event: DragEvent) {
    if (!draggingFiles(event)) return
    dragDepth = Math.max(0, dragDepth - 1)
    if (!dragDepth) setDropActive(false)
  }

  function handleDrop() {
    dragDepth = 0
    setDropActive(false)
  }

  function attach() {
    const root = options.root.value
    if (!root) return
    host.value = floatingHost(root)
    root.addEventListener('dragenter', handleDragEnter, true)
    root.addEventListener('dragleave', handleDragLeave, true)
    root.addEventListener('drop', handleDrop, true)
    root.addEventListener('click', handleClick, true)
    root.addEventListener('dblclick', handleDoubleClick, true)
    root.addEventListener('keydown', handleKeydown, true)
    document.addEventListener('pointerdown', handlePointerDown, true)
    document.addEventListener('pointerup', handlePointerUp, true)
    document.addEventListener('selectionchange', schedule)
    document.addEventListener('scroll', schedule, { capture: true, passive: true })
    window.addEventListener('resize', schedule, { passive: true })
  }

  function detach() {
    const root = options.root.value
    root?.removeEventListener('dragenter', handleDragEnter, true)
    root?.removeEventListener('dragleave', handleDragLeave, true)
    root?.removeEventListener('drop', handleDrop, true)
    handleDrop()
    root?.removeEventListener('click', handleClick, true)
    root?.removeEventListener('dblclick', handleDoubleClick, true)
    root?.removeEventListener('keydown', handleKeydown, true)
    document.removeEventListener('pointerdown', handlePointerDown, true)
    document.removeEventListener('pointerup', handlePointerUp, true)
    document.removeEventListener('selectionchange', schedule)
    document.removeEventListener('scroll', schedule, { capture: true })
    window.removeEventListener('resize', schedule)
    if (frame) cancelAnimationFrame(frame)
    frame = 0
    clearImage()
    hideSelection()
  }

  return {
    host,
    boundary,
    selectionOpen,
    selectionAnchor,
    selectionAvailable,
    selectionActive,
    selectionHeading,
    preferBelow,
    image,
    imageWidth,
    imageResizable,
    imageAlt,
    schedule,
    attach,
    detach,
    runCommand,
    setHeading,
    setImageWidth,
    setImageAlt,
    removeImage,
    previewImage,
    clearImage,
  }
}

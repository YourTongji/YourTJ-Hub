import type { ObjectDirective } from 'vue'
import { h, render } from 'vue'
import { Check, Copy } from '@lucide/vue'
import { i18n } from './i18n'

/**
 * 为 gf-prose 容器内的每个 <pre> 代码块挂载右上角"复制"按钮。
 *
 * 与 code-highlight / math-render 指令协作：本指令只负责复制，不改变代码内容。
 * 每个代码块包一层 .gf-code-block 相对定位容器，按钮 absolute 定位在其右上角，
 * 这样 pre 横向滚动时按钮不随内容移动，也不会遮挡代码。
 *
 * 绑定幂等：用 data-gf-code-copy 标记，重复挂载 / updated 时跳过已处理节点。
 */

interface CopyState {
  /** 最近一次 copy 尝试的时间戳，用于节流与还原 Check 图标 */
  copiedAt: number
}

const BOUND_MARKER = 'data-gf-code-copy'
const CHECK_RESET_MS = 1500
const MIN_INTERVAL_MS = 400

/** 当前 locale 下"复制代码"文案 */
function copyLabel(): string {
  return i18n.global.t('common.copyCode')
}

/** 当前 locale 下"已复制"文案 */
function copiedLabel(): string {
  return i18n.global.t('common.codeCopied')
}

/** 当前 locale 下"复制失败"文案 */
function failedLabel(): string {
  return i18n.global.t('common.copyFailed')
}

/** 从代码块中提取纯文本源码（含换行），与 highlighter 无关 */
function codeText(pre: HTMLElement): string {
  const codeElement = pre.querySelector('code')
  return (codeElement?.textContent ?? pre.textContent).replace(/\n$/, '')
}

async function writeToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    // 剪贴板 API 可能因权限被拒，回退到 execCommand
    try {
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      const ok = document.execCommand('copy')
      textarea.remove()
      return ok
    } catch {
      return false
    }
  }
}

function createCopyButton(): HTMLButtonElement {
  const button = document.createElement('button')
  button.type = 'button'
  button.className = 'gf-code-copy'
  button.setAttribute('aria-label', copyLabel())
  button.title = copyLabel()

  // 两个图标槽：默认展示 Copy，复制成功后 cross-fade 到 Check（由 CSS 控制显隐）
  const defaultIcon = document.createElement('span')
  defaultIcon.className = 'gf-code-copy__icon-default'
  render(h(Copy, { 'aria-hidden': 'true' }), defaultIcon)
  button.appendChild(defaultIcon)

  const checkIcon = document.createElement('span')
  checkIcon.className = 'gf-code-copy__icon-check'
  render(h(Check, { 'aria-hidden': 'true' }), checkIcon)
  button.appendChild(checkIcon)

  const label = document.createElement('span')
  label.className = 'gf-code-copy__label'
  label.textContent = copyLabel()
  button.appendChild(label)

  const state: CopyState = { copiedAt: 0 }

  button.addEventListener('click', async (event) => {
    event.preventDefault()
    event.stopPropagation()
    const now = Date.now()
    if (now - state.copiedAt < MIN_INTERVAL_MS) return
    state.copiedAt = now

    const succeeded = await writeToClipboard(codeText(button.closest('.gf-code-block')?.querySelector('pre') as HTMLElement))
    button.classList.toggle('gf-code-copy--copied', succeeded)
    button.classList.toggle('gf-code-copy--failed', !succeeded)
    button.title = succeeded ? copiedLabel() : failedLabel()
    button.setAttribute('aria-label', button.title)

    window.setTimeout(() => {
      button.classList.remove('gf-code-copy--copied', 'gf-code-copy--failed')
      button.title = copyLabel()
      button.setAttribute('aria-label', copyLabel())
    }, CHECK_RESET_MS)
  })

  return button
}

/** 把一个 <pre> 包装进 .gf-code-block 容器并挂上复制按钮 */
function bindCopyButton(pre: HTMLElement) {
  const wrapper = document.createElement('div')
  wrapper.className = 'gf-code-block'
  pre.replaceWith(wrapper)
  wrapper.appendChild(pre)
  wrapper.appendChild(createCopyButton())
  pre.setAttribute(BOUND_MARKER, 'true')
}

function enhanceCopyButtons(root: ParentNode) {
  const codeBlocks = Array.from(root.querySelectorAll<HTMLElement>('pre'))
  for (const pre of codeBlocks) {
    if (pre.hasAttribute(BOUND_MARKER)) continue
    bindCopyButton(pre)
  }
}

export const codeCopyDirective: ObjectDirective<HTMLElement> = {
  mounted: enhanceCopyButtons,
  updated: enhanceCopyButtons,
}

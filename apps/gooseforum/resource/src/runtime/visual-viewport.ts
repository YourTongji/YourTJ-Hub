import { onBeforeUnmount, ref } from 'vue'

/**
 * 追踪软键盘（visual viewport 收缩）相对布局视口的底部偏移。
 *
 * 安卓/iOS 软键盘弹出时不会改变布局视口（layout viewport）高度，
 * 但会压缩 visual viewport：`visualViewport.height` 变小、
 * `visualViewport.offsetTop` 增大。此时 `position: fixed; bottom: N`
 * 仍相对布局视口底部定位，浮动面板会被键盘遮挡。
 * 通过本 composable 拿到偏移量，把 fixed 元素的 bottom 动态抬高即可
 * 让面板跟随键盘上浮。
 *
 * 不支持 visualViewport 的旧浏览器返回 0，保持原始定位（静默回退）。
 */
export function useKeyboardVisualViewportOffset() {
  const bottomOffset = ref(0)

  if (typeof window === 'undefined') return { bottomOffset }
  const visualViewport = window.visualViewport
  if (!visualViewport) return { bottomOffset }

  const update = () => {
    const layoutHeight = window.innerHeight
    const visibleBottom = visualViewport.height + visualViewport.offsetTop
    bottomOffset.value = Math.max(0, layoutHeight - visibleBottom)
  }

  update()
  visualViewport.addEventListener('resize', update)
  visualViewport.addEventListener('scroll', update)
  onBeforeUnmount(() => {
    visualViewport.removeEventListener('resize', update)
    visualViewport.removeEventListener('scroll', update)
  })

  return { bottomOffset }
}

export interface HeadingRect {
  id: string
  /** getBoundingClientRect().top，标题顶边相对视口顶部的距离（px）。 */
  top: number
}

/**
 * 滚动侦测（scroll-spy）决策纯函数：给定文档序的标题顶边位置，
 * 返回最后一个顶边位于阅读线（offset）之上或齐平的标题 id。
 * 所有标题都在阅读线之下时回退到第一个标题（页面顶部场景）；
 * 空表返回空串。
 */
export function resolveActiveHeading(headings: HeadingRect[], offset: number): string {
  let current = ''
  for (const heading of headings) {
    if (heading.top <= offset) current = heading.id
    else break // 文档序标题 top 单调不减，第一个低于阅读线的即可终止扫描
  }
  return current || headings[0]?.id || ''
}

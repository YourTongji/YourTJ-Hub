// 通用 line-clamp 失效断言（issue #1692 / PR #1693）：
// 静态门禁（scripts/check-line-clamp.mjs）只覆盖源码可见的类串形态；本断言在真实
// 浏览器里以 computed style 为最终裁决——级联（important/变体/源序）、跨组件传导、
// 运行时拼接类名全部已被浏览器算完，按构造覆盖静态门禁的全部边界。
// 判定规则：元素携带 line-clamp-N（N≥1）类时，computed display 必须是截断形态
// （-webkit-box，或新版 Chromium 对 line-clamp 生效元素序列化的 flow-root——
// CSS Overflow 4 block-ellipsis 路径），否则截断已失效。不能只比较
// -webkit-line-clamp 计算值：display 被覆盖时该属性仍保留类值（实测 Chromium 151），
// 不是生效信号。line-clamp-none 是显式取消截断，不参与判定；带变体前缀的类按当前
// 渲染状态自然裁决（断点/主题/状态由调用方页面决定）。
//
// 用法（任意 *.browser.mjs 套件，页面加载并注入样式后）：
//   import { collectLineClampViolations, assertNoLineClampViolations } from './helpers/line-clamp-assert.mjs'
//   await assertNoLineClampViolations(page)
// 覆盖范围 = 该套件渲染到的页面与状态；未渲染的分支不在本断言范围内（与静态门禁互补）。

// 在页面内收集违例。返回 [{ tag, classes, display, lineClamp, expected, text }]。
export async function collectLineClampViolations(page) {
  return page.evaluate(() => {
    const clampClass = /^(?:.*:)?line-clamp-(\d+)$/
    const violations = []
    for (const element of document.querySelectorAll('[class]')) {
      const classes = typeof element.className === 'string' ? element.className : ''
      let expected = null
      for (const token of classes.split(/\s+/)) {
        const m = clampClass.exec(token)
        if (m) expected = m[1] // 多个 clamp 类时取最后一个（与源序语义一致的粗粒度）
      }
      if (expected === null) continue
      const style = getComputedStyle(element)
      // display:none（hidden 变体等）下元素不渲染，截断无从谈起；只裁决可见元素。
      if (style.display === 'none') continue
      // 截断生效的 display 形态：旧版 -webkit-box；新版 Chromium 序列化 flow-root。
      const clampDisplay = style.display === '-webkit-box' || style.display === 'flow-root'
      if (!clampDisplay) {
        violations.push({
          tag: element.tagName.toLowerCase(),
          classes: classes.slice(0, 200),
          display: style.display,
          lineClamp: style.webkitLineClamp,
          expected,
          text: (element.textContent || '').trim().slice(0, 80),
        })
      }
    }
    return violations
  })
}

// 断言页面无截断失效违例；失败信息含元素类串与 computed 值，可直接定位。
export async function assertNoLineClampViolations(page, { assert } = {}) {
  const violations = await collectLineClampViolations(page)
  const message =
    `line-clamp truncation silently disabled on ${violations.length} element(s):\n` +
    violations
      .map(
        (v) =>
          `  <${v.tag}> expected -webkit-box/line-clamp:${v.expected}, got display:${v.display}/line-clamp:${v.lineClamp}\n` +
          `    class="${v.classes}" text="${v.text}"`,
      )
      .join('\n')
  if (assert) assert.equal(violations.length, 0, message)
  else if (violations.length > 0) throw new Error(message)
  return violations
}

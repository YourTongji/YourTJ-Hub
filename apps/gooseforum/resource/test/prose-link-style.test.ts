import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

// 正文链接的呈现只在 CSS 里：服务端 markdown2html 负责把绝对 http(s) 链接
// 标成 target="_blank"，样式侧据此区分外链与站内链接。
const css = readFileSync(
  new URL('../src/styles/prose.css', import.meta.url),
  'utf8',
)

function rule(pattern: RegExp): { selector: string; body: string } {
  const match = css.match(pattern)
  expect(match, `missing rule ${pattern}`).not.toBeNull()
  return { selector: match![1], body: match![2] }
}

describe('post body link styling', () => {
  it('keeps links on the theme color without an underline', () => {
    const base = rule(/\.gf-prose-post (a) \{([^}]*)\}/)
    const apply = base.body.match(/@apply ([^;]*);/)
    expect(apply, 'base rule must style links through @apply').not.toBeNull()
    // 逐个 token 断言：`hover:text-primary` 不能满足 `text-primary`。
    const utilities = apply![1].split(/\s+/)
    expect(utilities).toContain('text-primary')
    expect(utilities).toContain('no-underline')
  })

  it('prefixes the external-link icon only on server-marked links', () => {
    const icon = rule(/\.gf-prose-post (a\[target='_blank'\][^{]*) \{([^}]*)\}/)
    expect(icon.selector).toContain("a[target='_blank']")
    expect(icon.selector).toContain(':not(:has(> img))')
    for (const declaration of [
      "content: ''",
      'display: inline-block',
      'width: 0.85em',
      'height: 0.85em',
      'background-color: currentColor',
      '-webkit-mask-image',
    ]) {
      expect(icon.body).toContain(declaration)
    }
    // 标准属性带边界匹配：`-webkit-mask-image` / `-webkit-mask-size` 是不同属性，
    // 不能作为子串满足标准属性的断言。
    expect(icon.body).toMatch(
      /(^|[;\s])mask-image:\s*url\("data:image\/svg\+xml,/,
    )
    expect(icon.body).toMatch(/(^|[;\s])mask-size:\s*contain/)
  })

  it('gives a visible press feedback', () => {
    const active = rule(/\.gf-prose-post (a:active) \{([^}]*)\}/)
    expect(active.body).toContain('opacity: 0.65')
  })
})

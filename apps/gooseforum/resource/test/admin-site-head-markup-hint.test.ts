import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'
import en from '../src/locales/en'
import ja from '../src/locales/ja'

// 回归（issue #1244）：站点设置「外部资源链接 / Meta 标签」只接受 head 内元素，
// 管理台要写明这条边界；admin-raw 语言包没有 i18n 门禁，四份文案需在此对齐。
const root = resolve(__dirname, '..')
const pageSrc = readFileSync(resolve(root, 'src/admin/pages/AdminSettingsPage.vue'), 'utf8')

function adminRawFile(locale: string): string {
  const name = locale === 'zh' ? 'admin-raw.generated.ts' : `admin-raw.${locale}.generated.ts`
  return resolve(root, 'src/locales', name)
}

describe('站点设置 head 注入提示', () => {
  test('externalLinks 字段旁渲染提示文案', () => {
    const start = pageSrc.indexOf("adminText('k008e')")
    expect(start).toBeGreaterThanOrEqual(0)
    expect(pageSrc.slice(start, start + 400)).toContain("adminText('siteExternalLinksHint')")
  })

  test('四种语言都提供该文案', () => {
    for (const locale of ['zh', 'en', 'ja', 'de']) {
      const src = readFileSync(adminRawFile(locale), 'utf8')
      expect(src, `${locale} 缺少 siteExternalLinksHint`).toContain('"siteExternalLinksHint":')
    }
  })

  test('日语提示不回退到英文，并保留 head 元数据和 CSP 边界', () => {
    const hint = ja.adminRaw.siteExternalLinksHint
    expect(hint).not.toBe(en.adminRaw.siteExternalLinksHint)
    expect(hint).toMatch(/\p{Script=Hiragana}/u)
    for (const token of [
      '<head>', 'meta', 'link', 'script', 'style', 'noscript',
      'charset', 'viewport', 'title', 'description', 'robots', 'color-scheme', 'theme-color',
      'og:', 'twitter:', 'article:', 'rel', 'canonical', 'prev', 'next', 'manifest', 'icon',
      'apple-touch-icon', 'CSP',
    ]) {
      expect(hint).toContain(token)
    }
  })
})

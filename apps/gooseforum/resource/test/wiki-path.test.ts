import { describe, expect, test } from 'vitest'
import { isValidWikiPath } from '../src/admin/utils/wiki'

describe('isValidWikiPath（与后端 wikiservice.ValidatePath 对齐）', () => {
  test('接受 namespace/slug 与嵌套路径', () => {
    expect(isValidWikiPath('guide/getting-started')).toBe(true)
    expect(isValidWikiPath('deployment/waline')).toBe(true)
    expect(isValidWikiPath('guide/sub/page-name')).toBe(true)
    expect(isValidWikiPath('guide/v2-docs-2026')).toBe(true)
  })

  test('大写输入按小写归一后通过', () => {
    expect(isValidWikiPath('Guide/Getting-Started')).toBe(true)
    expect(isValidWikiPath('guide/UPPER')).toBe(true)
  })

  test('首尾空白被忽略，前导斜杠被拒绝', () => {
    expect(isValidWikiPath('  guide/getting-started  ')).toBe(true)
    expect(isValidWikiPath('/guide/getting-started')).toBe(false)
  })

  test('拒绝中文、空格、下划线、点号、连续连字符', () => {
    expect(isValidWikiPath('guide/快速开始')).toBe(false)
    expect(isValidWikiPath('guide/使用指南')).toBe(false)
    expect(isValidWikiPath('guide/a b')).toBe(false)
    expect(isValidWikiPath('guide/my_page')).toBe(false)
    expect(isValidWikiPath('guide/a.b')).toBe(false)
    expect(isValidWikiPath('guide/my--page')).toBe(false)
  })

  test('拒绝空 slug 段与 ..', () => {
    expect(isValidWikiPath('guide/')).toBe(false)
    expect(isValidWikiPath('guide/..')).toBe(false)
    expect(isValidWikiPath('guide//page')).toBe(false)
  })

  test('拒绝缺少 slug 段的裸 namespace 与空值', () => {
    expect(isValidWikiPath('guide')).toBe(false)
    expect(isValidWikiPath('')).toBe(false)
    expect(isValidWikiPath('   ')).toBe(false)
  })

  test('拒绝超长段与超长路径', () => {
    expect(isValidWikiPath(`guide/${'a'.repeat(65)}`)).toBe(false)
    expect(isValidWikiPath(`guide/${'a'.repeat(64)}`)).toBe(true)
    expect(isValidWikiPath(`${'a'.repeat(255)}/b`)).toBe(false)
  })
})

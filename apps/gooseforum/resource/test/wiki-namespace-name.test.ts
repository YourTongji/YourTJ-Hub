import { describe, expect, test } from 'vitest'
import { isValidNamespaceName } from '../src/admin/utils/wiki'

describe('isValidNamespaceName（与后端 wikiservice.ValidateNamespace 对齐）', () => {
  test('接受小写字母、数字、连字符组合', () => {
    expect(isValidNamespaceName('guide')).toBe(true)
    expect(isValidNamespaceName('deploy')).toBe(true)
    expect(isValidNamespaceName('getting-started')).toBe(true)
    expect(isValidNamespaceName('v2-docs-2026')).toBe(true)
  })

  test('大写输入按小写归一后通过', () => {
    expect(isValidNamespaceName('Guide')).toBe(true)
    expect(isValidNamespaceName('My-Namespace')).toBe(true)
    expect(isValidNamespaceName('MYNAMESPACE')).toBe(true)
  })

  test('首尾空白被忽略', () => {
    expect(isValidNamespaceName('  guide  ')).toBe(true)
  })

  test('拒绝中文、下划线、点号、空格、斜杠、连续连字符', () => {
    expect(isValidNamespaceName('使用指南')).toBe(false)
    expect(isValidNamespaceName('My_Namespace')).toBe(false)
    expect(isValidNamespaceName('my.name')).toBe(false)
    expect(isValidNamespaceName('my namespace')).toBe(false)
    expect(isValidNamespaceName('my/namespace')).toBe(false)
    expect(isValidNamespaceName('my--ns')).toBe(false)
  })

  test('拒绝空值与超长名称', () => {
    expect(isValidNamespaceName('')).toBe(false)
    expect(isValidNamespaceName('   ')).toBe(false)
    expect(isValidNamespaceName('a'.repeat(65))).toBe(false)
    expect(isValidNamespaceName('a'.repeat(64))).toBe(true)
  })
})

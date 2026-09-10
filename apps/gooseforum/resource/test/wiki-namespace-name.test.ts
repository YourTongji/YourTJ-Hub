import { describe, expect, test } from 'vitest'
import { isValidNamespaceName } from '../src/admin/utils/wiki'

describe('isValidNamespaceName（与后端 wikiservice.ValidateNamespace 对齐）', () => {
  test('接受小写字母、数字、连字符组合', () => {
    expect(isValidNamespaceName('guide')).toBe(true)
    expect(isValidNamespaceName('deploy')).toBe(true)
    expect(isValidNamespaceName('getting-started')).toBe(true)
    expect(isValidNamespaceName('v2-docs-2026')).toBe(true)
    expect(isValidNamespaceName('1guide')).toBe(true)
    expect(isValidNamespaceName('2026')).toBe(true)
  })

  test('保留大小写（不再小写归一）', () => {
    expect(isValidNamespaceName('Guide')).toBe(true)
    expect(isValidNamespaceName('My-Namespace')).toBe(true)
    expect(isValidNamespaceName('MYNAMESPACE')).toBe(true)
  })

  test('接受中文等 Unicode 命名空间（GitHub 顶层目录名）', () => {
    expect(isValidNamespaceName('同济新手教程')).toBe(true)
    expect(isValidNamespaceName('使用指南')).toBe(true)
    expect(isValidNamespaceName('校园生活')).toBe(true)
    expect(isValidNamespaceName('日本語ドキュメント')).toBe(true)
  })

  test('首尾空白被忽略（trim 后再校验）', () => {
    expect(isValidNamespaceName('  guide  ')).toBe(true)
    expect(isValidNamespaceName('  同济新手教程  ')).toBe(true)
  })

  test('拒绝点开头、空格、斜杠、保留字符（下划线/中间点为合法目录名，允许）', () => {
    expect(isValidNamespaceName('.hidden')).toBe(false)
    expect(isValidNamespaceName('..')).toBe(false)
    expect(isValidNamespaceName('my namespace')).toBe(false)
    expect(isValidNamespaceName('中文 空格')).toBe(false)
    expect(isValidNamespaceName('my/namespace')).toBe(false)
    expect(isValidNamespaceName('my:ns')).toBe(false)
    expect(isValidNamespaceName('my*ns')).toBe(false)
    expect(isValidNamespaceName('my<ns')).toBe(false)
    expect(isValidNamespaceName('my"ns')).toBe(false)
    expect(isValidNamespaceName('my|ns')).toBe(false)
    expect(isValidNamespaceName('my?ns')).toBe(false)
    expect(isValidNamespaceName('my\\ns')).toBe(false)
    // 文件系统目录名合法字符：下划线、中间点、连续连字符不再拒绝
    expect(isValidNamespaceName('My_Namespace')).toBe(true)
    expect(isValidNamespaceName('my.name')).toBe(true)
  })

  test('拒绝空值与超长名称（按字符计数）', () => {
    expect(isValidNamespaceName('')).toBe(false)
    expect(isValidNamespaceName('   ')).toBe(false)
    expect(isValidNamespaceName('a'.repeat(65))).toBe(false)
    expect(isValidNamespaceName('a'.repeat(64))).toBe(true)
    expect(isValidNamespaceName('济'.repeat(64))).toBe(true)
    expect(isValidNamespaceName('济'.repeat(65))).toBe(false)
  })
})

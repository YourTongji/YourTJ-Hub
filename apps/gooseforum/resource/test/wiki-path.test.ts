import { describe, expect, test } from 'vitest'
import { isValidWikiPath } from '../src/admin/utils/wiki'

describe('isValidWikiPath（与后端 wikiservice.ValidatePath 对齐）', () => {
  test('接受 namespace/slug 与嵌套路径', () => {
    expect(isValidWikiPath('guide/getting-started')).toBe(true)
    expect(isValidWikiPath('deployment/waline')).toBe(true)
    expect(isValidWikiPath('guide/sub/page-name')).toBe(true)
    expect(isValidWikiPath('guide/v2-docs-2026')).toBe(true)
  })

  test('保留大小写（不再小写归一）', () => {
    expect(isValidWikiPath('Guide/Getting-Started')).toBe(true)
    expect(isValidWikiPath('guide/UPPER')).toBe(true)
  })

  test('接受中文路径段（GitHub 仓库相对路径）', () => {
    expect(isValidWikiPath('同济新手教程/学校/简介')).toBe(true)
    expect(isValidWikiPath('同济新手教程/学业/课程与选课')).toBe(true)
    expect(isValidWikiPath('guide/快速开始')).toBe(true)
    expect(isValidWikiPath('guide/使用指南')).toBe(true)
  })

  test('首尾空白被忽略，前导斜杠被拒绝', () => {
    expect(isValidWikiPath('  guide/getting-started  ')).toBe(true)
    expect(isValidWikiPath('  同济新手教程/学校  ')).toBe(true)
    expect(isValidWikiPath('/guide/getting-started')).toBe(false)
  })

  test('拒绝空格、点开头、保留字符（下划线/中间点/连续连字符为合法目录名，允许）', () => {
    expect(isValidWikiPath('guide/a b')).toBe(false)
    expect(isValidWikiPath('guide/.hidden')).toBe(false)
    expect(isValidWikiPath('guide/a:b')).toBe(false)
    expect(isValidWikiPath('guide/a*b')).toBe(false)
    expect(isValidWikiPath('guide/a?b')).toBe(false)
    expect(isValidWikiPath('guide/a<b')).toBe(false)
    expect(isValidWikiPath('guide/a>b')).toBe(false)
    expect(isValidWikiPath('guide/a|b')).toBe(false)
    expect(isValidWikiPath('guide/a"b')).toBe(false)
    expect(isValidWikiPath('guide/a\\b')).toBe(false)
    // 文件系统目录名合法字符：下划线、中间点、连续连字符不再拒绝
    expect(isValidWikiPath('guide/my_page')).toBe(true)
    expect(isValidWikiPath('guide/my--page')).toBe(true)
    expect(isValidWikiPath('guide/a.b')).toBe(true)
  })

  test('拒绝空 slug 段、点开头段与 ..', () => {
    expect(isValidWikiPath('guide/')).toBe(false)
    expect(isValidWikiPath('guide/..')).toBe(false)
    expect(isValidWikiPath('guide/.hidden')).toBe(false)
    expect(isValidWikiPath('guide//page')).toBe(false)
  })

  test('拒绝缺少 slug 段的裸 namespace 与空值', () => {
    expect(isValidWikiPath('guide')).toBe(false)
    expect(isValidWikiPath('')).toBe(false)
    expect(isValidWikiPath('   ')).toBe(false)
  })

  test('拒绝超长段与超长路径（按字符计数）', () => {
    expect(isValidWikiPath(`guide/${'a'.repeat(65)}`)).toBe(false)
    expect(isValidWikiPath(`guide/${'a'.repeat(64)}`)).toBe(true)
    expect(isValidWikiPath(`guide/${'济'.repeat(65)}`)).toBe(false)
    expect(isValidWikiPath(`${'a'.repeat(255)}/b`)).toBe(false)
  })
})

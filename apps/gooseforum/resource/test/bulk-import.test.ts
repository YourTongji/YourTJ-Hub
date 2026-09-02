import { describe, expect, test } from 'vitest'
import {
  BULK_IMPORT_LIMIT,
  BULK_IMPORT_PREVIEW_LIMIT,
  parseImportText,
} from '../src/admin/bulkImport'

describe('parseImportText 批量粘贴解析', () => {
  test('多分隔符混合：换行、逗号、分号、空格、Tab、全角符号', () => {
    const result = parseImportText('Alice\nBob, Charlie；Dave;Eve\tFrank，Grace', [])
    expect(result).toEqual({
      added: ['Alice', 'Bob', 'Charlie', 'Dave', 'Eve', 'Frank', 'Grace'],
      skipped: 0,
      truncated: false,
    })
  })

  test('去空：连续分隔符、纯空白行不产生条目', () => {
    const result = parseImportText('\n  , ; ，；\t\nAlice,,Bob;;\n\nCharlie   \n', [])
    expect(result.added).toEqual(['Alice', 'Bob', 'Charlie'])
    expect(result.skipped).toBe(0)
    expect(result.truncated).toBe(false)
  })

  test('去重：大小写不敏感，保留首个大小写形态', () => {
    const result = parseImportText('Admin\nadmin\nADMIN\nAlice\nalice', [])
    expect(result.added).toEqual(['Admin', 'Alice'])
    expect(result.skipped).toBe(3)
  })

  test('与既有数组合并去重：existing 视为已去重集合', () => {
    const existing = ['root', 'Admin']
    const result = parseImportText('Alice\nROOT\nalice\nBob', existing)
    expect(result.added).toEqual(['Alice', 'Bob'])
    expect(result.skipped).toBe(2)
  })

  test('existing 不因解析被改写', () => {
    const existing = ['root', 'Admin']
    parseImportText('Alice\nRoot', existing)
    expect(existing).toEqual(['root', 'Admin'])
  })

  test('全重复：无新增，全部跳过', () => {
    const existing = ['alice', 'bob']
    const result = parseImportText('Alice\nBOB\nalice', existing)
    expect(result.added).toEqual([])
    expect(result.skipped).toBe(3)
  })

  test('全新增：全部计入', () => {
    const result = parseImportText('aaa\nbbb\nccc', [])
    expect(result.added).toEqual(['aaa', 'bbb', 'ccc'])
    expect(result.skipped).toBe(0)
  })

  test('超大输入：超过上限截断并标记', () => {
    const text = Array.from({ length: BULK_IMPORT_LIMIT + 3 }, (_, i) => `word${i}`).join('\n')
    const result = parseImportText(text, [])
    expect(result.added).toHaveLength(BULK_IMPORT_LIMIT)
    expect(result.truncated).toBe(true)
    expect(result.skipped).toBe(0)
  })

  test('截断边界：恰好上限不标记，多一条才截断', () => {
    const atLimit = parseImportText(
      Array.from({ length: BULK_IMPORT_LIMIT }, (_, i) => `item${i}`).join('\n'),
      [],
    )
    expect(atLimit.truncated).toBe(false)
    const overLimit = parseImportText(
      Array.from({ length: BULK_IMPORT_LIMIT + 1 }, (_, i) => `item${i}`).join('\n'),
      [],
    )
    expect(overLimit.truncated).toBe(true)
    expect(overLimit.added).toHaveLength(BULK_IMPORT_LIMIT)
  })

  test('空串与纯空白：无新增', () => {
    for (const text of ['', '   ', '\n\t\n', ',,，；;']) {
      const result = parseImportText(text, [])
      expect(result.added).toEqual([])
      expect(result.skipped).toBe(0)
      expect(result.truncated).toBe(false)
    }
  })

  test('首尾空白被 trim，条目保留首个大小写形态（与 chips 显示一致）', () => {
    const result = parseImportText('  Alice  \n  bob ', [])
    expect(result.added).toEqual(['Alice', 'bob'])
    expect(result.added[0]).toBe('Alice')
  })

  test('重复出现在截断丢弃段不影响结果', () => {
    const text = Array.from({ length: BULK_IMPORT_LIMIT + 5 }, (_, i) => `dup${i}`).join('\n')
    const result = parseImportText(text, [])
    expect(result.skipped).toBe(0)
    expect(result.truncated).toBe(true)
    expect(result.added[0]).toBe('dup0')
    expect(result.added).toHaveLength(BULK_IMPORT_LIMIT)
  })

  test('预览上限常量：只影响 UI 渲染，不影响解析结果', () => {
    expect(BULK_IMPORT_PREVIEW_LIMIT).toBeGreaterThan(0)
    const result = parseImportText(
      Array.from({ length: BULK_IMPORT_PREVIEW_LIMIT + 1 }, (_, i) => `w${i}`).join('\n'),
      [],
    )
    expect(result.added).toHaveLength(BULK_IMPORT_PREVIEW_LIMIT + 1)
  })
})

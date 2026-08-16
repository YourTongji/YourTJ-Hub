import { describe, expect, test } from 'vitest'
import { flattenAdminTree } from '../src/admin/utils/wiki-tree'
import type { WikiPageNode } from '../src/admin/types'

describe('flattenAdminTree', () => {
  test('flattens nested wiki tree with depth indices (issue #289)', () => {
    const tree: WikiPageNode[] = [
      { pageId: 1, path: 'guide/getting-started', sourcePath: 'guide/getting-started', title: '快速开始', sortOrder: 1 },
      {
        pageId: 0,
        path: 'guide/draft',
        sourcePath: '',
        title: 'draft',
        sortOrder: 3,
        children: [
          { pageId: 2, path: 'guide/draft/ideas', sourcePath: 'guide/draft/ideas', title: '草稿想法', sortOrder: 3 },
        ],
      },
    ]
    const flat = flattenAdminTree(tree)
    expect(flat.map((n) => `${n.depth}:${n.path}`)).toEqual([
      '0:guide/getting-started',
      '0:guide/draft',
      '1:guide/draft/ideas',
    ])
  })

  test('preserves directory nodes with pageId 0 and empty sourcePath', () => {
    const flat = flattenAdminTree([
      {
        pageId: 0,
        path: 'docs/a',
        sourcePath: '',
        title: 'a',
        sortOrder: 0,
        children: [{ pageId: 3, path: 'docs/a/b', sourcePath: 'docs/a/b', title: 'B', sortOrder: 0 }],
      },
    ])
    expect(flat[0]).toMatchObject({ pageId: 0, sourcePath: '', depth: 0 })
    expect(flat[1]).toMatchObject({ pageId: 3, sourcePath: 'docs/a/b', depth: 1 })
  })

  test('handles undefined/empty input', () => {
    expect(flattenAdminTree(undefined)).toEqual([])
    expect(flattenAdminTree([])).toEqual([])
  })
})

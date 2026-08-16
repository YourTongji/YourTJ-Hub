import type { WikiPageNode } from '@/admin/types'

// AdminFlatNode 管理端页面树扁平化节点（层级以 depth 缩进呈现，issue #289）。
export interface AdminFlatNode {
  pageId: number
  path: string
  sourcePath: string
  title: string
  sortOrder: number
  depth: number
}

// flattenAdminTree 把后端返回的嵌套 wiki 树扁平化（后端已按同级 order 排序，
// 目录节点 pageId=0 保留在输出中以便管理端以缩进/图标区分）。
export function flattenAdminTree(pages: WikiPageNode[] | undefined, depth = 0): AdminFlatNode[] {
  const out: AdminFlatNode[] = []
  for (const p of pages ?? []) {
    out.push({
      pageId: p.pageId,
      path: p.path,
      sourcePath: p.sourcePath,
      title: p.title,
      sortOrder: p.sortOrder,
      depth,
    })
    out.push(...flattenAdminTree(p.children, depth + 1))
  }
  return out
}

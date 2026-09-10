import { describe, expect, test } from 'vitest'
import type { PostPayload } from '@gooseforum/client'
import { buildReplyForest, flattenReplyForest, postTreeIndentLevel } from '../src/runtime/reply-forest'

function post(id: number, postNo: number, replyToPostId?: number): PostPayload {
  return { id, postNo, replyToPostId } as unknown as PostPayload
}

function ids(nodes: ReturnType<typeof buildReplyForest>): number[] {
  return nodes.map((node) => node.post.id)
}

describe('buildReplyForest', () => {
  test('正常链：回复 #1 为根，链内回复嵌套到父节点', () => {
    // #1 提问；#2/#3 回答（回复 #1）；#4 回复 #3；#5 回复 #2
    const posts = [
      post(1, 1),
      post(2, 2, 1),
      post(3, 3, 1),
      post(4, 4, 3),
      post(5, 5, 2),
    ]
    const roots = buildReplyForest(posts, { firstPostId: 1 })

    expect(ids(roots)).toEqual([1, 2, 3])
    expect(ids(roots[1]!.children)).toEqual([5])
    expect(ids(roots[2]!.children)).toEqual([4])
    expect(roots[2]!.children[0]!.children).toHaveLength(0)
  })

  test('三层链真实嵌套：孙回复挂在父回复下而非链根（depth 2）', () => {
    // #2 回答；#3 回复 #2；#4 回复 #3 → 4 挂在 3 下，3 挂在 2 下
    const posts = [post(1, 1), post(2, 2, 1), post(3, 3, 2), post(4, 4, 3)]
    const roots = buildReplyForest(posts, { firstPostId: 1 })

    expect(ids(roots)).toEqual([1, 2])
    const answer = roots[1]!
    expect(ids(answer.children)).toEqual([3])
    expect(ids(answer.children[0]!.children)).toEqual([4])

    const rows = flattenReplyForest(roots)
    expect(rows.map((row) => row.post.id)).toEqual([1, 2, 3, 4])
    expect(rows.map((row) => row.depth)).toEqual([0, 0, 1, 2])
  })

  test('无目标与回复首楼均落根节点', () => {
    const posts = [post(1, 1), post(2, 2), post(3, 3, 1)]
    const roots = buildReplyForest(posts, { firstPostId: 1 })
    expect(ids(roots)).toEqual([1, 2, 3])
    roots.forEach((node) => expect(node.children).toHaveLength(0))
  })

  test('目标未加载 → 根节点兜底（深链场景）', () => {
    // 只加载了 #7（回复一个不存在的 #4）
    const posts = [post(7, 7, 4)]
    const roots = buildReplyForest(posts, { firstPostId: undefined })
    expect(ids(roots)).toEqual([7])
  })

  test('成环 → 全部落根节点，不死循环', () => {
    const posts = [post(10, 2, 11), post(11, 3, 10)]
    const roots = buildReplyForest(posts, { firstPostId: undefined })
    expect(ids(roots)).toEqual([10, 11])
    roots.forEach((node) => expect(node.children).toHaveLength(0))
  })

  test('超过 64 跳的深链 → 根节点兜底，浅层仍正常嵌套', () => {
    // c71 回复首楼（根）；c70 回复 c71 …… c2 回复 c3（从 c2 上溯 69 跳，超限）
    const posts: PostPayload[] = [post(1, 1), post(71, 71, 1)]
    for (let no = 70; no >= 2; no -= 1) posts.push(post(no, no, no + 1))
    const roots = buildReplyForest(posts, { firstPostId: 1 })

    // c2（id 2）上溯 69 跳超限 → 根
    expect(ids(roots)).toContain(2)
    // c70（id 70）上溯 1 跳到 c71，正常嵌套
    const c71 = roots.find((node) => node.post.id === 71)
    expect(c71).toBeDefined()
    expect(ids(c71!.children)).toContain(70)
  })

  test('forceFlatIds 命中 → 根节点（append 边界规则）', () => {
    const posts = [post(1, 1), post(5, 5, 1), post(6, 6, 5)]
    const roots = buildReplyForest(posts, { firstPostId: 1, forceFlatIds: new Set([6]) })
    expect(ids(roots)).toEqual([1, 5, 6])
  })

  test('根节点与 children 均按 postNo 升序', () => {
    const posts = [
      post(30, 6, 10),
      post(10, 2, 1),
      post(20, 4, 1),
      post(1, 1),
      post(25, 5, 10),
      post(15, 3, 1),
    ]
    const roots = buildReplyForest(posts, { firstPostId: 1 })
    expect(ids(roots)).toEqual([1, 10, 15, 20])
    expect(ids(roots[1]!.children)).toEqual([25, 30])
  })
})

describe('flattenReplyForest', () => {
  test('DFS 展平：深度递增、hasChildren 正确', () => {
    const posts = [post(1, 1), post(2, 2, 1), post(3, 3, 1), post(4, 4, 2)]
    const roots = buildReplyForest(posts, { firstPostId: 1 })
    const rows = flattenReplyForest(roots)

    expect(rows.map((row) => row.post.id)).toEqual([1, 2, 4, 3])
    expect(rows.map((row) => row.depth)).toEqual([0, 0, 1, 0])
    expect(rows.map((row) => row.hasChildren)).toEqual([false, true, false, false])
  })

  test('collapsedIds 折叠节点时隐藏其全部后代', () => {
    const posts = [post(1, 1), post(2, 2, 1), post(4, 4, 2), post(5, 5, 4)]
    const roots = buildReplyForest(posts, { firstPostId: 1 })

    const expanded = flattenReplyForest(roots)
    expect(expanded.map((row) => row.post.id)).toEqual([1, 2, 4, 5])

    const collapsed = flattenReplyForest(roots, new Set([2]))
    expect(collapsed.map((row) => row.post.id)).toEqual([1, 2])
    expect(collapsed[1]!.hasChildren).toBe(true)
  })
})

describe('postTreeIndentLevel', () => {
  test('深度上限 4：≥4 不再增加缩进', () => {
    expect(postTreeIndentLevel(0)).toBe(0)
    expect(postTreeIndentLevel(1)).toBe(1)
    expect(postTreeIndentLevel(3)).toBe(3)
    expect(postTreeIndentLevel(4)).toBe(4)
    expect(postTreeIndentLevel(9)).toBe(4)
  })
})

import type { PostPayload } from '@gooseforum/client'

/** 回复森林节点：post 加上它在树中的层级与子节点。 */
export interface ForestNode {
  post: PostPayload
  depth: number
  children: ForestNode[]
}

/** DFS 展平后的渲染行。 */
export interface ForestRow {
  post: PostPayload
  depth: number
  hasChildren: boolean
}

/** 链上溯跳数上限：超过视为脏数据（成环/超长链），兜底为根节点平铺。 */
const MAX_CHAIN_HOPS = 64

/** 缩进深度上限：≥ 此深度不再增加缩进（内容不丢，视觉封顶）。 */
export const TREE_INDENT_DEPTH_LIMIT = 4

/**
 * 把楼层列表构建为回复森林（真实父子嵌套：孙回复挂在父回复下，非链根拍平）。
 *
 * 分层规则（全部兜底为根节点，保证内容零丢失）：
 * - `replyToPostId` 为 0/null、等于 firstPostId（回复首楼=回答/话题级回复）→ 根
 * - 回复目标不在已加载窗口（深链/跨窗）→ 根
 * - 命中 `forceFlatIds`（append 窗口边界规则）→ 根
 * - 祖先链成环或超过 MAX_CHAIN_HOPS 跳 → 根
 *
 * 根节点与 children 均按 postNo 升序。
 */
export function buildReplyForest(
  posts: PostPayload[],
  options: { firstPostId?: number; forceFlatIds?: ReadonlySet<number> } = {},
): ForestNode[] {
  const byId = new Map<number, PostPayload>()
  for (const post of posts) byId.set(post.id, post)

  const nodes = new Map<number, ForestNode>()
  for (const post of posts) nodes.set(post.id, { post, depth: 0, children: [] })

  // 沿父链上溯验证祖先链可解析（终止于无目标/首楼/未加载，而非成环或超限）。
  // 成环/超 64 跳的节点自身兜底为根，避免与环内节点互相挂靠后从森林中不可达。
  function hasResolvableAncestry(post: PostPayload): boolean {
    const seen = new Set<number>([post.id])
    let current: PostPayload = post
    let hops = 0
    for (;;) {
      const pid = current.replyToPostId
      if (!pid || pid === options.firstPostId || !byId.has(pid)) return true
      if (seen.has(pid) || hops++ >= MAX_CHAIN_HOPS) return false
      seen.add(pid)
      current = byId.get(pid)!
    }
  }

  // 返回 post 的直接挂靠父节点；null = 自身为根节点。
  // 兜底为根：无目标、回复首楼、forceFlat、目标未加载、祖先链成环/超限。
  function attachParent(post: PostPayload): PostPayload | null {
    if (options.forceFlatIds?.has(post.id)) return null
    const pid = post.replyToPostId
    if (!pid || pid === options.firstPostId) return null
    const parent = byId.get(pid)
    if (!parent) return null
    if (!hasResolvableAncestry(post)) return null
    return parent
  }

  const roots: ForestNode[] = []
  for (const post of posts) {
    const parent = attachParent(post)
    if (parent == null) {
      roots.push(nodes.get(post.id)!)
    } else {
      nodes.get(parent.id)!.children.push(nodes.get(post.id)!)
    }
  }

  const byPostNo = (a: ForestNode, b: ForestNode) => (a.post.postNo || 0) - (b.post.postNo || 0)
  roots.sort(byPostNo)
  for (const node of nodes.values()) node.children.sort(byPostNo)

  return roots
}

/**
 * DFS 展平森林为渲染行（免递归组件）。depth 在展平时逐层重算，
 * 折叠节点隐藏其全部后代。
 */
export function flattenReplyForest(
  roots: ForestNode[],
  collapsedIds?: ReadonlySet<number>,
): ForestRow[] {
  const rows: ForestRow[] = []
  const stack: ForestNode[] = [...roots].reverse()

  while (stack.length) {
    const node = stack.pop()!
    rows.push({
      post: node.post,
      depth: node.depth,
      hasChildren: node.children.length > 0,
    })

    const collapsed = collapsedIds?.has(node.post.id) ?? false
    if (!collapsed) {
      for (let i = node.children.length - 1; i >= 0; i -= 1) {
        const child = node.children[i]!
        stack.push({ ...child, depth: node.depth + 1 })
      }
    }
  }
  return rows
}

/** 渲染缩进层级：深度封顶 TREE_INDENT_DEPTH_LIMIT。 */
export function postTreeIndentLevel(depth: number): number {
  return Math.min(depth, TREE_INDENT_DEPTH_LIMIT)
}

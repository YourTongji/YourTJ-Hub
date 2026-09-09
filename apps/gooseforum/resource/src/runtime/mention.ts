/**
 * 帖子编辑器 @mention 会话纯逻辑（issue #564）。
 * 与 DOM/Vditor 解耦：输入侧喂入光标前文本，输出识别到的 token 与排序后的候选，
 * 便于在 happy-dom 下做确定性单元测试。DOM 采集（光标上下文）与插入由
 * VditorOfficial.vue 的 getMentionContext/replaceMentionToken 负责。
 */

/** @mention 候选用户：本地上下文（回复目标/主题作者/参与者）与服务端搜索结果共用形状 */
export interface MentionUser {
  id: number
  username: string
  nickname?: string
  avatarUrl: string
  /** 本地上下文弱标签：正在回复 / 主题作者 / 参与者（仅本地候选带） */
  tag?: 'reply-target' | 'topic-author' | 'participant'
}

/** 识别出的 mention token：光标前文本中 [start, start + length) 即 "@query" */
export interface MentionToken {
  start: number
  length: number
  query: string
}

/**
 * @ 触发边界：@ 前一个字符不能是单词字符/数字/下划线。
 * "a@b" / "foo@bar"（邮箱、标识符）不触发；"谢谢@张三"（CJK 前）触发。
 */
const MENTION_TRIGGER_BOUNDARY = /[^A-Za-z0-9_]/

/** token 内终止字符：空白与常见（中英文）终止标点 */
const TOKEN_TERMINATOR = /[\s，。！？；：、（）【】「」『』“”‘’…,.!?;:()<>"'/\\|{}\[\]-]/

/**
 * 从光标前文本中提取最近的 @mention token。
 *
 * 规则（issue #564）：@ 位于开头或由空白/常见标点分隔；空格、换行、终止标点或
 * caret 离开 token 后关闭；邮箱/标识符内嵌的 @ 不触发。
 */
export function extractMentionToken(prefix: string): MentionToken | null {
  if (!prefix) return null
  let query = ''
  for (let i = prefix.length - 1; i >= 0; i--) {
    const ch = prefix[i]
    if (ch === '@') {
      const prev = i > 0 ? prefix[i - 1] : undefined
      if (prev === undefined || MENTION_TRIGGER_BOUNDARY.test(prev)) {
        return { start: i, length: query.length + 1, query }
      }
      // 邮箱/标识符内的 @（如 "a@b"、"@foo@bar" 的第二个 @）：不视为 mention
      return null
    }
    if (TOKEN_TERMINATOR.test(ch)) return null
    query = ch + query
  }
  return null
}

/** 用户匹配强度分值：username exact > username prefix > nickname exact > nickname prefix > 包含 */
function userMatchScore(user: MentionUser, query: string): number {
  const q = query.toLowerCase()
  const username = user.username.toLowerCase()
  const nickname = (user.nickname ?? user.username).toLowerCase()
  if (username === q) return 100
  if (username.startsWith(q)) return 80
  if (nickname === q) return 60
  if (nickname.startsWith(q)) return 40
  if (username.includes(q)) return 20
  if (nickname.includes(q)) return 10
  return 0
}

const LOCAL_TAG_ORDER: Record<NonNullable<MentionUser['tag']>, number> = {
  'reply-target': 0,
  'topic-author': 1,
  participant: 2,
}

export interface MentionRankInput {
  /** 本地上下文（回复目标 > 主题作者 > 参与者），按 tag 顺序传入亦可，内部会再排序 */
  local: MentionUser[]
  /** 服务端搜索结果（scope=users） */
  server: MentionUser[]
  query: string
  /** 当前登录用户 id：候选排除自己（0 表示无） */
  currentUserId?: number
  /** 展示上限：默认 8（空 query 时本地上下文最多 5） */
  limit?: number
}

/**
 * 合并去重并排序候选：
 * - 空 query：仅本地上下文，最多 5 个，不查服务端；
 * - 有 query：本地上下文（须匹配 query）优先，再按匹配强度排服务端结果；
 * - 按 userId 去重；当前用户默认不进入候选。
 */
export function rankMentionCandidates(input: MentionRankInput): MentionUser[] {
  const { local, server, query, currentUserId = 0, limit = 8 } = input
  const seen = new Set<number>()
  const out: MentionUser[] = []
  const push = (user: MentionUser) => {
    if (user.id === currentUserId || seen.has(user.id)) return
    seen.add(user.id)
    out.push(user)
  }

  const localSorted = [...local].sort((a, b) => {
    const ta = a.tag ? LOCAL_TAG_ORDER[a.tag] : 3
    const tb = b.tag ? LOCAL_TAG_ORDER[b.tag] : 3
    return ta - tb
  })

  const q = query.trim()
  if (!q) {
    for (const user of localSorted) push(user)
    return out.slice(0, 5)
  }

  for (const user of localSorted) {
    if (userMatchScore(user, q) > 0) push(user)
  }
  const serverSorted = [...server]
    .filter(user => userMatchScore(user, q) > 0)
    .sort((a, b) => userMatchScore(b, q) - userMatchScore(a, q))
  for (const user of serverSorted) push(user)
  return out.slice(0, limit)
}
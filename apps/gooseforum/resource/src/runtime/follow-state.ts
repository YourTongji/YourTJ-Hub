// 会话内「我是否关注了某用户」的单一事实源（issue #593）。
// UserCard 的挂载级缓存与 UserPage 的本地 ref 各自为政，一处关注/取关后
// 另一处继续展示旧态；这里集中登记最近一次已知状态，并在变更时广播
// goose:follow-changed（沿用 goose:page / goose:unread 的 window 事件约定），
// 各展示面订阅纠正。注意：这里只是会话内缓存，SSR 载荷与 /api/user-card
// 仍是权威来源，登记值随时可被回源结果覆盖。

export interface FollowChangeDetail {
  userId: number
  isFollowing: boolean
}

export const FOLLOW_CHANGE_EVENT = 'goose:follow-changed'

const known = new Map<number, boolean>()
// 每用户变更代次：广播一次递增一次。回源请求落地时对比代次，
// 可识别「请求发出后、响应落地前发生了同 tab 关注变更」，避免旧快照回退新状态。
const changeSeq = new Map<number, number>()
let lastSeq = 0

export function getKnownFollowState(userId: number): boolean | undefined {
  return known.get(userId)
}

// 读取用户当前的变更代次（请求发出前快照用）。
export function getFollowChangeSeq(userId: number): number {
  return changeSeq.get(userId) ?? 0
}

export function recordFollowState(userId: number, isFollowing: boolean) {
  known.set(userId, isFollowing)
}

// 关注/取关成功后调用：登记新状态并广播，供 UserCard/UserPage 等即时同步。
export function broadcastFollowChange(userId: number, isFollowing: boolean) {
  recordFollowState(userId, isFollowing)
  changeSeq.set(userId, ++lastSeq)
  window.dispatchEvent(
    new CustomEvent<FollowChangeDetail>(FOLLOW_CHANGE_EVENT, {
      detail: { userId, isFollowing },
    }),
  )
}

export function onFollowChange(handler: (detail: FollowChangeDetail) => void): () => void {
  const listener = (event: Event) => handler((event as CustomEvent<FollowChangeDetail>).detail)
  window.addEventListener(FOLLOW_CHANGE_EVENT, listener)
  return () => window.removeEventListener(FOLLOW_CHANGE_EVENT, listener)
}

// 仅供测试：清空会话内登记状态。
export function resetFollowState() {
  known.clear()
  changeSeq.clear()
  lastSeq = 0
}

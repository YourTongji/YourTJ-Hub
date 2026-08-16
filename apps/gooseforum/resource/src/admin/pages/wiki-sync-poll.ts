// wiki 手动同步后的前端轮询决策（issue #290）。
// 后端 POST /api/admin/wiki/sync 立即返回 accepted，run 行由后台 goroutine
// 创建（可能晚于响应），且同步可能持续数分钟 → 管理端轮询 status/runs
// 直到出现比触发时更新的 run 行进入终态，或达到有界轮询上限。

import type { WikiSyncStatus } from '@/admin/runtime/api'

/** 轮询间隔。 */
export const WIKI_SYNC_POLL_INTERVAL_MS = 3000

/** 最大轮询次数（3s × 100 ≈ 5 分钟有界等待；超时后停止并提示）。 */
export const WIKI_SYNC_POLL_MAX_ATTEMPTS = 100

/**
 * isSyncTerminal 判断轮询是否到达终态：status.lastRun 是 id 严格大于
 * 触发前 lastRun（beforeRunId）的新 run 行且状态为 success/failed。
 * 覆盖 accepted-before-row：POST 返回后 run 行尚未创建时（lastRun 仍为旧
 * 行或为空）必须继续轮询，不能把旧 run 的终态当成本次同步完成。
 */
export function isSyncTerminal(status: WikiSyncStatus | null, beforeRunId: number): boolean {
  const last = status?.lastRun
  if (!last) return false
  if (last.id <= beforeRunId) return false
  return last.status === 'success' || last.status === 'failed'
}

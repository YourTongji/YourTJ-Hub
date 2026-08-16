import { describe, expect, test } from 'vitest'
import { isSyncTerminal } from '../src/admin/pages/wiki-sync-poll'
import type { WikiSyncStatus } from '../src/admin/runtime/api'

// issue #290 回归：手动同步 accepted 后，前端必须轮询到「比触发时更新的
// run 行进入终态」才算完成。
function status(lastRun: WikiSyncStatus['lastRun']): WikiSyncStatus {
  return {
    enabled: true,
    repo: 'https://github.com/YourTongji/YourTJ-Wiki.git',
    branch: 'main',
    headSha: '',
    lastRun,
    pages: { total: 0, namespaces: 0 },
  }
}

function run(id: number, state: 'running' | 'success' | 'failed') {
  return {
    id,
    headSha: '',
    trigger: 'manual',
    status: state,
    pagesAdded: 0,
    pagesUpdated: 0,
    pagesDeleted: 0,
    startedAt: '2026-08-16T00:00:00Z',
  }
}

describe('isSyncTerminal (wiki sync polling, issue #290)', () => {
  test('accepted-before-row：POST 后 run 行尚未创建时（无 lastRun）继续轮询', () => {
    expect(isSyncTerminal(status(undefined), 0)).toBe(false)
  })

  test('lastRun 仍是触发前的旧 run（id 不大于 beforeRunId）→ 未完成', () => {
    const before = run(7, 'success')
    expect(isSyncTerminal(status(before), 7)).toBe(false)
  })

  test('新 run 行仍 running → 未完成，继续轮询', () => {
    expect(isSyncTerminal(status(run(8, 'running')), 7)).toBe(false)
  })

  test('新 run 行 success → 终态', () => {
    expect(isSyncTerminal(status(run(8, 'success')), 7)).toBe(true)
  })

  test('新 run 行 failed → 终态（错误细节由状态/运行记录展示）', () => {
    expect(isSyncTerminal(status(run(8, 'failed')), 7)).toBe(true)
  })

  test('无任何历史记录时首个终态 run → 终态', () => {
    expect(isSyncTerminal(status(run(1, 'success')), 0)).toBe(true)
  })
})

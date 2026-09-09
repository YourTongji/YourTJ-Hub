// Schedule cloud synchronization: reconcile before writes, serialize uploads, and
// require a choice for divergent snapshots or account changes. Server revisions
// guard every PUT; dirty local state survives page exit and transient failures.

import { shallowRef, type ShallowRef } from 'vue'
import { setSolidifyHook, useScheduleStore } from './useScheduleStore'

import type { components } from '@gooseforum/client/openapi'

export type PkSyncPayload = components['schemas']['PkPlansPutRequest']
export type PkSyncRemoteSnapshot = components['schemas']['PkPlansSnapshotData']

// ---- 传输层 ----

/** 同步错误类别：401 未登录 / 服务端拒绝（400 结构校验、403 冻结）/ 网络错误。 */
export type PkSyncErrorKind = 'unauthenticated' | 'rejected' | 'network'

export class PkSyncError extends Error {
  readonly status: number
  readonly kind: PkSyncErrorKind

  constructor(message: string, status: number, kind: PkSyncErrorKind) {
    super(message)
    this.name = 'PkSyncError'
    this.status = status
    this.kind = kind
  }
}

export interface PkSyncTransport {
  /** GET 云端快照；云端空返回 null。401/网络错误/结构失败抛 PkSyncError。 */
  fetchCloudSnapshot(): Promise<PkSyncRemoteSnapshot | null>
  /** PUT 整包快照；返回服务端新 updatedAt。 */
  putCloudSnapshot(payload: PkSyncPayload): Promise<{ updatedAt: string }>
}

/** PK 域信封（{code, msg, data}；成功 code===0，负载在 data）。 */
interface PkSyncEnvelope {
  code?: number
  msg?: string
  data?: unknown
}

/**
 * 默认传输层：同源 fetch（cookie 自动携带），信封解析对齐 runtime/pk-api.ts，
 * 但错误以 PkSyncError（kind + status）抛出供同步状态机区分处理。
 */
function createFetchTransport(): PkSyncTransport {
  async function request(path: string, init?: RequestInit): Promise<Response> {
    try {
      return await fetch(path, init)
    } catch {
      throw new PkSyncError('网络错误', 0, 'network')
    }
  }

  /** 解析 PK 信封：401 视为未登录；code!==0 视为服务端拒绝；非 2xx 兜底。 */
  async function readEnvelope(response: Response): Promise<unknown> {
    if (response.status === 401) {
      // 401 由论坛中间件产生（forum 信封 auth.required），视为未登录。
      throw new PkSyncError('未登录', 401, 'unauthenticated')
    }
    const envelope = (await response.json().catch(() => undefined)) as PkSyncEnvelope | undefined
    const code = envelope?.code
    if (code !== undefined && code !== 0) {
      throw new PkSyncError(envelope?.msg || '请求失败', response.status, 'rejected')
    }
    if (!response.ok) {
      throw new PkSyncError(`HTTP ${response.status}`, response.status, 'rejected')
    }
    return envelope?.data
  }

  return {
    async fetchCloudSnapshot() {
      const data = await readEnvelope(await request('/api/pk/plans'))
      if (data === null || data === undefined) return null
      if (typeof data !== 'object') {
        throw new PkSyncError('云端快照格式异常', 0, 'rejected')
      }
      return data as PkSyncRemoteSnapshot
    },
    async putCloudSnapshot(payload) {
      const response = await request('/api/pk/plans', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      const data = (await readEnvelope(response)) as { updatedAt?: unknown } | undefined
      return { updatedAt: typeof data?.updatedAt === 'string' ? data.updatedAt : '' }
    },
  }
}

// ---- 同步控制器（纯逻辑；测试注入 fake transport）----

/** 本地变更 → 上传的防抖窗口（ms）。 */
export const PK_SYNC_DEBOUNCE_MS = 3000

export interface ScheduleSyncController {
  /** 冲突弹窗状态：非空 = 云端快照待二选一（只读消费，写走下方方法）。 */
  readonly conflict: ShallowRef<PkSyncRemoteSnapshot | null>
  /** 进页同步：GET + 四分支决策（自动上传 / 整包采用 / 弹窗 / 不动）。失败静默。 */
  syncOnPageEnter(): Promise<void>
  /** 本地方案变更入口（store.solidify 尾部钩子）：标脏 + 防抖 PUT。 */
  onLocalChange(): void
  /** best-effort 冲刷未落盘的防抖 PUT（visibilitychange hidden / 离页）。 */
  flushPendingUpload(): void
  /** 弹窗选择「使用云端」：整包采用云端快照（不回灌上传）。 */
  useCloud(): void
  /** 弹窗选择「保留本机」：立即 PUT 本机快照覆盖云端。 */
  keepLocal(): void
  /** 关闭弹窗（暂缓决策）：本轮不再询问，下次进页若仍分歧再提示。 */
  dismissConflict(): void
  /** 启用同步（已登录进页时调用）。 */
  start(userId?: number): void
  /** 停止同步（登出/离页）：取消防抖与弹窗，本地数据不动。 */
  stop(): void
  /** 是否有未上传的本地变更（诊断/测试）。 */
  isDirty(): boolean
}

export function createScheduleSyncController(deps: { transport: PkSyncTransport }): ScheduleSyncController {
  const store = useScheduleStore()
  const conflict = shallowRef<PkSyncRemoteSnapshot | null>(null)
  let enabled = false
  let dirty = false
  let reconciled = false
  let authStopped = false
  let entering = false
  let recheckAfterEntry = false
  let putting: Promise<void> | null = null
  let generation = 0
  let seq = 0
  let owner = 0
  let baseUpdatedAt = ''
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  function clearDebounce(): void {
    if (debounceTimer !== null) clearTimeout(debounceTimer)
    debounceTimer = null
  }

  function isLocalEmpty(): boolean {
    const plans = store.state.plans
    return plans.length === 1 && plans[0].stagedCourses.length === 0 &&
      plans[0].selectedCourses.length === 0 && plans[0].customEvents.length === 0
  }

  function scheduleUpload(): void {
    clearDebounce()
    if (!enabled || !reconciled || authStopped) return
    debounceTimer = setTimeout(() => {
      debounceTimer = null
      void pushSnapshot()
    }, PK_SYNC_DEBOUNCE_MS)
  }

  async function pushSnapshot(): Promise<void> {
    if (!enabled || !reconciled || putting || authStopped || !dirty) return
    const run = generation
    const revision = seq
    let retryConflict = false
    putting = (async () => {
      try {
        const payload = { ...store.snapshotForSync(), baseUpdatedAt } as PkSyncPayload
        const { updatedAt } = await deps.transport.putCloudSnapshot(payload)
        if (!enabled || generation !== run) return
        if (!updatedAt) throw new Error('Missing sync revision')
        baseUpdatedAt = updatedAt
        if (revision === seq) dirty = !store.markSynced(updatedAt)
        else scheduleUpload()
      } catch (err) {
        if (!enabled || generation !== run) return
        if (err instanceof PkSyncError) {
          if (err.status === 409) {
            reconciled = false
            retryConflict = true
          } else if (err.kind === 'unauthenticated') authStopped = true
          else if (err.status === 400 || err.status === 403) {
            console.warn('[pk-sync] 上传被拒绝：', err.message)
          }
        }
      }
    })()
    await putting
    putting = null
    if (retryConflict) {
      if (entering) recheckAfterEntry = true
      else await syncOnPageEnter()
    }
  }

  function onLocalChange(): void {
    if (!enabled) return
    dirty = true
    store.markSyncDirty()
    seq++
    scheduleUpload()
  }

  function flushPendingUpload(): void {
    clearDebounce()
    void pushSnapshot()
  }

  function emptyCloud(): PkSyncRemoteSnapshot {
    return {
      plans: [{ id: 'empty', name: store.state.plans[0].name, createdAt: Date.now(), stagedCourses: [], selectedCourses: [], customEvents: [] }],
      activePlanId: 'empty', majorSelected: {}, weekView: { week: null, useCurrent: false }, updatedAt: '',
    }
  }

  async function syncOnPageEnter(): Promise<void> {
    if (!enabled || entering) return
    const run = generation
    entering = true
    reconciled = false
    clearDebounce()
    authStopped = false
    try {
      // A reconciliation never races a previous upload or adopts its obsolete base.
      if (putting) await putting
      if (!enabled || generation !== run) return
      const snapshot = await deps.transport.fetchCloudSnapshot()
      if (!enabled || generation !== run) return
      baseUpdatedAt = snapshot?.updatedAt ?? ''
      const previousOwner = store.getSyncOwner()
      if (previousOwner && previousOwner !== owner) {
        conflict.value = snapshot ?? emptyCloud()
        return
      }
      if (!store.setSyncOwner(owner)) return
      if (snapshot === null) {
        reconciled = true
        if (!isLocalEmpty() || dirty) {
          dirty = true
          store.markSyncDirty()
          await pushSnapshot()
        }
        return
      }
      if (!dirty && !store.isSyncDirty() && !store.getSyncedAt() && isLocalEmpty()) {
        store.applyRemoteSnapshot(snapshot)
        dirty = !store.markSynced(snapshot.updatedAt)
        reconciled = true
        return
      }
      if (!dirty && !store.isSyncDirty() && store.getSyncedAt() === snapshot.updatedAt) {
        reconciled = true
        return
      }
      conflict.value = snapshot
    } catch (err) {
      if (generation === run && err instanceof PkSyncError && err.status === 401) authStopped = true
    } finally {
      if (generation === run) {
        entering = false
        if (recheckAfterEntry) {
          recheckAfterEntry = false
          void syncOnPageEnter()
        }
      }
    }
  }

  function useCloud(): void {
    const snapshot = conflict.value
    if (!enabled || !snapshot || putting || !store.setSyncOwner(owner)) return
    conflict.value = null
    clearDebounce()
    store.applyRemoteSnapshot(snapshot)
    dirty = !store.markSynced(snapshot.updatedAt)
    reconciled = true
  }

  function keepLocal(): void {
    if (!enabled || !conflict.value || !store.setSyncOwner(owner)) return
    conflict.value = null
    clearDebounce()
    dirty = true
    store.markSyncDirty()
    reconciled = true
    void pushSnapshot()
  }

  function dismissConflict(): void {
    conflict.value = null
    // Dismissal postpones a decision; subsequent edits must not overwrite the cloud.
    reconciled = false
  }

  function start(userId = 0): void {
    generation++
    owner = userId
    enabled = true
    entering = false
    reconciled = false
    dirty = store.isSyncDirty()
    authStopped = false
  }

  function stop(): void {
    enabled = false
    generation++
    entering = false
    reconciled = false
    clearDebounce()
    conflict.value = null
  }

  return { conflict, syncOnPageEnter, onLocalChange, flushPendingUpload, useCloud,
    keepLocal, dismissConflict, start, stop, isDirty: () => dirty }
}

// ---- 单例接线（排课页生命周期驱动；未登录不 start 即零网络）----

export const scheduleSync = createScheduleSyncController({ transport: createFetchTransport() })

let wired = false

/** 首次启动时接线：注入 store.solidify 尾部钩子 + visibilitychange 冲刷（幂等）。 */
function wireOnce(): void {
  if (wired) return
  wired = true
  setSolidifyHook(() => scheduleSync.onLocalChange())
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') scheduleSync.flushPendingUpload()
    })
  }
}

/** 启动云同步（已登录进入排课页时调用；之后由 solidify 钩子驱动增量上传）。 */
export function startScheduleSync(userId: number): void {
  wireOnce()
  scheduleSync.start(userId)
}

/** 停止云同步（离开排课页/登出后调用）：取消防抖与弹窗，本地数据原样保留。 */
export function stopScheduleSync(): void {
  scheduleSync.stop()
}

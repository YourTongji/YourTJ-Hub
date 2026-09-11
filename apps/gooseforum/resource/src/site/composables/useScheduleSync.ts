// Schedule cloud synchronization: upload local edits against their server revision,
// auto-resume after network failures, and preserve divergent local plans as
// "[本地自动恢复]方案x" instead of blocking on a conflict dialog. Server revisions
// guard every PUT; dirty local state survives page exit and transient failures.

import { shallowRef, type ShallowRef } from 'vue'
import { clonePlansAsAutoRestore, MAX_PLANS, setSolidifyHook, useScheduleStore } from './useScheduleStore'

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
/** 定时自动保存/网络恢复重试心跳（ms）。 */
export const PK_SYNC_AUTOSAVE_MS = 10000

/** 进页对账结果（#571）：供「同步最新」按钮与进页提示反映方案层同步结果。
 *  merged=分歧已按云端为主合并；adopted=已整包采用云端；uploaded=本地方案已上传；
 *  idle=两端一致无需动作；blocked=合并超出容量受限；failed=对账未完成（网络/未登录/写入失败）。 */
export type PkSyncReconcileResult = 'merged' | 'adopted' | 'uploaded' | 'idle' | 'blocked' | 'failed'
export interface ScheduleSyncController {
  /** 自动恢复提示：非空 = 云端分歧时本地方案已保留为恢复方案（页面 flash 后清空）。 */
  readonly notice: ShallowRef<string | null>
  /** 进页同步：GET 对账，干净本地采用新云端，分歧本地并入自动恢复合并（云端为主，
   *  #571/#573），空本地绝不覆盖非空云端。网络失败静默，由心跳重试。返回对账结果。 */
  syncOnPageEnter(): Promise<PkSyncReconcileResult>
  /** 本地方案变更入口（store.solidify 尾部钩子）：标脏 + 防抖 PUT。 */
  onLocalChange(): void
  /** best-effort 冲刷未落盘的防抖 PUT（visibilitychange hidden / 离页）。 */
  flushPendingUpload(): void
  /** 手动保存（「保存课表」按钮）：立即 PUT 本地方案，返回是否上传成功。 */
  saveNow(adoptPreviousOwner?: boolean): Promise<boolean>
  needsOwnerConfirmation(): boolean
  readonly mergeBlocked: ShallowRef<boolean>
  /** 清空自动恢复提示（页面 flash 后调用）。 */
  clearNotice(): void
  /** 启用同步（已登录进页时调用）：启动防抖 + 定时心跳（自动保存/网络恢复）。 */
  start(userId?: number): void
  /** 停止同步（登出/离页）：取消防抖、心跳与提示，本地数据不动。 */
  stop(): void
  /** 是否有未上传的本地变更（诊断/测试）。 */
  isDirty(): boolean
}

export function createScheduleSyncController(deps: { transport: PkSyncTransport }): ScheduleSyncController {
  const store = useScheduleStore()
  const notice = shallowRef<string | null>(null)
  const mergeBlocked = shallowRef(false)
  let enabled = false
  let dirty = false
  let reconciled = false
  let authStopped = false
  let entering = false
  let rejected = false
  let recovering = false
  let putting: Promise<boolean> | null = null
  let heartbeatTimer: ReturnType<typeof setInterval> | null = null
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

  /** 本地与云端方案内容是否完全不同（JSON 比较；空/缺失任一侧不算分歧）。 */
  function plansDiverge(snapshot: PkSyncRemoteSnapshot): boolean {
    const cloudPlans = Array.isArray(snapshot.plans) ? snapshot.plans : []
    if (cloudPlans.length === 0) return false
    if (isLocalEmpty()) return false
    return JSON.stringify(store.state.plans) !== JSON.stringify(cloudPlans)
  }

  /** 本地方案数组与云端方案数组是否不同（不看 dirty/同步时钟；空本地保护判断用）。
   *  云端无方案（[]）时不视为分歧：空对空无需保护，正常上传即可。 */
  function localPlansDifferFromCloud(snapshot: PkSyncRemoteSnapshot): boolean {
    const cloudPlans = Array.isArray(snapshot.plans) ? snapshot.plans : []
    return cloudPlans.length > 0 && JSON.stringify(store.state.plans) !== JSON.stringify(cloudPlans)
  }

  /** 云端分歧合并（#571/#573）：云端为主，本地方案克隆为「[本地自动恢复]方案x」追加上传。
   *  先上传后落盘：PUT 成功才把合并结果写入本地；失败路径本地保持合并前状态，
   *  重试/重载从同一份本地源重新克隆，恢复方案不翻倍（幂等，#571 review）。
   *  409 时以最新云端在函数内有界重试合并（本地源不变）；上传期间用户又编辑时不落盘
   *  合并快照，并强制下轮走完整重对账重新合并（避免心跳盲传本地盖掉云端合并结果）。
   *  返回 merged（上传成功）/ blocked（超出容量）/ failed（上传失败，由重对账兜底重试）。 */
  async function mergeAndUpload(snapshot: PkSyncRemoteSnapshot): Promise<'merged' | 'blocked' | 'failed'> {
    if (putting) await putting
    const run = generation
    let current = snapshot
    // 409 重试上限：每次 409 都意味着云端被其他端推进；超出按失败交由重对账兜底。
    for (let attempt = 0; attempt < 3; attempt += 1) {
      const cloudPlans = Array.isArray(current.plans) ? current.plans : []
      const recoveryPlans = clonePlansAsAutoRestore(store.state.plans, cloudPlans)
      const mergedPlans = [...cloudPlans, ...recoveryPlans]
      const merged = { ...current, plans: mergedPlans }
      // Never attempt an upload the server cannot accept.
      if (mergedPlans.length > MAX_PLANS || new TextEncoder().encode(JSON.stringify(merged)).length > 1024 * 1024) {
        mergeBlocked.value = true
        rejected = true
        reconciled = false
        return 'blocked'
      }
      const payload = {
        plans: mergedPlans,
        activePlanId: current.activePlanId,
        majorSelected: current.majorSelected,
        weekView: current.weekView,
        baseUpdatedAt: current.updatedAt,
      } as PkSyncPayload
      baseUpdatedAt = current.updatedAt
      reconciled = true
      const revision = seq
      let conflict = false
      let uploadedAt = ''
      putting = (async (): Promise<boolean> => {
        try {
          const { updatedAt } = await deps.transport.putCloudSnapshot(payload)
          if (!enabled || generation !== run) return false
          if (!updatedAt) throw new Error('Missing sync revision')
          baseUpdatedAt = updatedAt
          uploadedAt = updatedAt
          return true
        } catch (err) {
          if (!enabled || generation !== run) return false
          if (err instanceof PkSyncError) {
            if (err.status === 409) conflict = true
            else if (err.kind === 'unauthenticated') authStopped = true
            else if (err.status === 400 || err.status === 403) {
              console.warn('[pk-sync] 合并上传被拒绝：', err.message)
              rejected = true
            }
            // 网络错误：本地保持合并前状态，心跳/重对账从同一本地源重试。
          }
          // 失败不落盘：重试/重载天然幂等（#571 review）。
          return false
        }
      })()
      const uploaded = await putting
      putting = null
      if (uploaded) {
        if (revision === seq) {
          // 上传成功才落盘合并结果（applyingRemote 守卫下原子替换，不触发钩子/回灌）。
          store.applyRemoteSnapshot({
            plans: mergedPlans,
            activePlanId: current.activePlanId,
            majorSelected: current.majorSelected,
            weekView: current.weekView,
          })
          notice.value = recoveryPlans.map((plan) => plan.name).join('、')
          dirty = !store.markSynced(uploadedAt)
        } else {
          // 上传期间用户又编辑：不落盘合并快照（避免覆盖新编辑），置 reconciled=false
          // 让下轮走完整重对账重新合并，而不是盲传本地盖掉云端刚合并的结果。
          dirty = true
          store.markSyncDirty()
          reconciled = false
        }
        return 'merged'
      }
      if (!conflict) {
        // 失败（网络/其他非 409）：置回未对账，让心跳重跑完整对账，从同一份
        // 未落盘的本地源重新合并（幂等）。不可标 dirty 走 pushSnapshot——那会
        // 盲传本地快照覆盖云端（#571 review）。
        reconciled = false
        return 'failed'
      }
      // 云端版本已前进：拉取最新云端，从同一份未落盘的本地源重试合并（不翻倍）。
      try {
        const newer = await deps.transport.fetchCloudSnapshot()
        if (!enabled || generation !== run) return 'failed'
        if (newer === null) {
          // 云端已被清空：交由重对账按「云端空 + 本地非空」路径重新上传本地。
          reconciled = false
          return 'failed'
        }
        current = newer
      } catch {
        reconciled = false
        return 'failed'
      }
    }
    reconciled = false
    return 'failed'
  }

  async function pushSnapshot(): Promise<boolean> {
    if (!enabled || !reconciled || putting || authStopped || !dirty) return false
    const run = generation
    const revision = seq
    let conflict = false
    putting = (async () => {
      try {
        const payload = { ...store.snapshotForSync(), baseUpdatedAt } as PkSyncPayload
        const { updatedAt } = await deps.transport.putCloudSnapshot(payload)
        if (!enabled || generation !== run) return false
        if (!updatedAt) throw new Error('Missing sync revision')
        baseUpdatedAt = updatedAt
        if (revision === seq) dirty = !store.markSynced(updatedAt)
        else scheduleUpload()
        return true
      } catch (err) {
        if (!enabled || generation !== run) return false
        if (err instanceof PkSyncError) {
          if (err.status === 409) conflict = true
          else if (err.kind === 'unauthenticated') authStopped = true
          else if (err.status === 400 || err.status === 403) {
            console.warn('[pk-sync] 上传被拒绝：', err.message)
            rejected = true
          }
          // 网络错误（kind==='network'）：dirty 保持，心跳自动重试。
        }
        return false
      }
    })()
    const uploaded = await putting
    putting = null
    if (conflict) {
      // 云端版本已前进（其他端写入）：重对账，分歧则合并保留本地。
      await reconcileDivergence()
    }
    return uploaded
  }

  /** 409 / 网络恢复后的重对账：拉取最新云端；本地有未上传变更且内容分歧时合并保留。 */
  async function reconcileDivergence(): Promise<void> {
    if (!enabled || recovering) return
    const run = generation
    recovering = true
    try {
      const snapshot = await deps.transport.fetchCloudSnapshot()
      if (!enabled || generation !== run) return
      baseUpdatedAt = snapshot?.updatedAt ?? ''
      if (snapshot === null) {
        reconciled = true
        dirty = true
        store.markSyncDirty()
        await pushSnapshot()
        return
      }
      if (!dirty && !store.isSyncDirty() && store.getSyncedAt() === snapshot.updatedAt) {
        reconciled = true
        return
      }
      if (dirty && store.getSyncedAt() !== snapshot.updatedAt && plansDiverge(snapshot)) {
        await mergeAndUpload(snapshot)
        return
      }
      if (isLocalEmpty() && localPlansDifferFromCloud(snapshot)) {
        // 空本地（含对账完成前标脏的瞬态窗口）且方案内容与云端不同：采用云端，
        // 绝不把空方案回灌覆盖非空云端（#571）。内容一致时仍走默认上传以保留本地意图。
        store.applyRemoteSnapshot(snapshot)
        dirty = !store.markSynced(snapshot.updatedAt)
        reconciled = true
        return
      }
      // 默认总是上传本地方案（以最近修改为准）。
      reconciled = true
      dirty = true
      store.markSyncDirty()
      await pushSnapshot()
    } catch (err) {
      if (generation === run && err instanceof PkSyncError && err.status === 401) authStopped = true
      // 网络错误：dirty 保持，心跳自动重试。
    } finally {
      if (generation === run) recovering = false
    }
  }

  function onLocalChange(): void {
    if (!enabled) return
    rejected = false
    mergeBlocked.value = false
    dirty = true
    store.markSyncDirty()
    seq++
    scheduleUpload()
  }

  function flushPendingUpload(): void {
    clearDebounce()
    if (!reconciled && !authStopped && !rejected) void syncOnPageEnter()
    else void pushSnapshot()
  }

  function needsOwnerConfirmation(): boolean {
    const previousOwner = store.getSyncOwner()
    return previousOwner !== 0 && previousOwner !== owner
  }

  async function saveNow(adoptPreviousOwner = false): Promise<boolean> {
    if (!enabled || authStopped) return false
    if (needsOwnerConfirmation()) {
      if (!adoptPreviousOwner || !store.setSyncOwner(owner)) return false
      dirty = true
      store.markSyncDirty()
    }
    clearDebounce()
    if (!reconciled) {
      await syncOnPageEnter()
      return reconciled && !dirty
    }
    rejected = false
    mergeBlocked.value = false
    dirty = true
    store.markSyncDirty()
    await pushSnapshot()
    return !dirty
  }

  async function syncOnPageEnter(): Promise<PkSyncReconcileResult> {
    if (!enabled || entering) return 'idle'
    const run = generation
    entering = true
    reconciled = false
    rejected = false
    clearDebounce()
    authStopped = false
    try {
      // A reconciliation never races a previous upload or adopts its obsolete base.
      if (putting) await putting
      if (!enabled || generation !== run) return 'failed'
      const snapshot = await deps.transport.fetchCloudSnapshot()
      if (!enabled || generation !== run) return 'failed'
      baseUpdatedAt = snapshot?.updatedAt ?? ''
      const previousOwner = store.getSyncOwner()
      if (previousOwner && previousOwner !== owner) {
        // 账号切换：不自动上传上一账号的本地方案；云端非空则采用云端，云端空则保留本地。
        if (snapshot) {
          store.applyRemoteSnapshot(snapshot)
          dirty = !store.markSynced(snapshot.updatedAt)
        } else {
          // Retain ownership across reloads and edits until the user explicitly
          // chooses to save these local plans to the currently signed-in account.
          return 'idle'
        }
        store.setSyncOwner(owner)
        reconciled = true
        return 'adopted'
      }
      if (!store.setSyncOwner(owner)) return 'failed'
      if (snapshot === null) {
        reconciled = true
        if (!isLocalEmpty() || dirty) {
          dirty = true
          store.markSyncDirty()
          return (await pushSnapshot()) ? 'uploaded' : 'failed'
        }
        return 'idle'
      }
      if (!dirty && !store.isSyncDirty() && store.getSyncedAt() === snapshot.updatedAt) {
        reconciled = true
        return 'idle'
      }
      // A known, clean local revision is a cache. Adopt newer cloud data instead
      // of publishing stale content over another device's edits.
      if (!dirty && !store.isSyncDirty() && store.getSyncedAt()) {
        store.applyRemoteSnapshot(snapshot)
        dirty = !store.markSynced(snapshot.updatedAt)
        reconciled = true
        return 'adopted'
      }
      if (isLocalEmpty() && localPlansDifferFromCloud(snapshot)) {
        // 空本地（含对账完成前标脏的瞬态窗口）且方案内容与云端不同：整包采用云端，
        // 绝不把空方案回灌覆盖非空云端（#571）。方案一致时（仅周次/专业等元数据差异）
        // 上传无害，保持原上传路径以保留本地意图。
        store.applyRemoteSnapshot(snapshot)
        dirty = !store.markSynced(snapshot.updatedAt)
        reconciled = true
        return 'adopted'
      }
      // Divergent local state never overwrites the cloud: dirty edits (offline
      // window) and never-synced locals (#571, e.g. plans made before login) are
      // both preserved as recovery clones while cloud data stays authoritative.
      if (
        (dirty || !store.getSyncedAt()) &&
        store.getSyncedAt() !== snapshot.updatedAt &&
        plansDiverge(snapshot)
      ) {
        return mergeAndUpload(snapshot)
      }
      if (!store.getSyncedAt()) {
        // 从未云同步且内容与云端一致：采用云端建立同步时钟（零 PUT）。
        store.applyRemoteSnapshot(snapshot)
        dirty = !store.markSynced(snapshot.updatedAt)
        reconciled = true
        return 'adopted'
      }
      reconciled = true
      dirty = true
      store.markSyncDirty()
      return (await pushSnapshot()) ? 'uploaded' : 'failed'
    } catch (err) {
      if (generation === run && err instanceof PkSyncError && err.status === 401) authStopped = true
      return 'failed'
    } finally {
      if (generation === run) entering = false
    }
  }

  function clearNotice(): void {
    notice.value = null
  }

  function start(userId = 0): void {
    generation++
    owner = userId
    enabled = true
    entering = false
    reconciled = false
    rejected = false
    dirty = store.isSyncDirty()
    authStopped = false
    if (heartbeatTimer === null) {
      // 定时自动保存 + 网络恢复自动补传：心跳推进未落盘上传。
      heartbeatTimer = setInterval(() => {
        if (!enabled || putting || authStopped || rejected) return
        if (!reconciled) void syncOnPageEnter()
        else if (dirty) void pushSnapshot()
      }, PK_SYNC_AUTOSAVE_MS)
    }
  }

  function stop(): void {
    enabled = false
    generation++
    entering = false
    reconciled = false
    clearDebounce()
    if (heartbeatTimer !== null) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
    notice.value = null
  }

  return { notice, mergeBlocked, needsOwnerConfirmation, syncOnPageEnter, onLocalChange, flushPendingUpload, saveNow, clearNotice,
    start, stop, isDirty: () => dirty }
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
    // 网络恢复立即补传（心跳兜底）。
    window.addEventListener('online', () => scheduleSync.flushPendingUpload())
  }
}

/** 启动云同步（已登录进入排课页时调用；之后由 solidify 钩子驱动增量上传）。 */
export function startScheduleSync(userId: number): void {
  wireOnce()
  scheduleSync.start(userId)
}

/** 停止云同步（离开排课页/登出后调用）：取消防抖、心跳与提示，本地数据原样保留。 */
export function stopScheduleSync(): void {
  scheduleSync.stop()
}

// Per-plan CAS with a persisted common ancestor. Content alone becomes dirty;
// week/major/active-plan preferences stay on this device. No clean-state heartbeat.
import { shallowRef } from 'vue'
import { setSolidifyHook, useScheduleStore, MAX_PLANS } from './useScheduleStore'
import { mergeSchedulePlan, schedulePlanKey, type PlanMergeConflict } from './schedule-plan-merge'
import type { PkPlan } from '@/site/types/pk'

export interface PkPlanItem {
  plan: PkPlan
  revision: number
  updatedAt: string
}
export class PkSyncError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly remote: PkPlanItem | null = null,
  ) {
    super(message)
  }
}
export interface PkSyncTransport {
  list(): Promise<PkPlanItem[]>
  put(plan: PkPlan, baseRevision: number): Promise<PkPlanItem>
  remove(planId: string, baseRevision: number): Promise<void>
}
function createFetchTransport(): PkSyncTransport {
  async function request(method = 'GET', body?: unknown) {
    const response = await fetch('/api/pk/plan-items', {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    const envelope = await response.json()
    if (!response.ok || envelope.code !== 0)
      throw new PkSyncError(envelope.msg || 'Sync failed', response.status, envelope.data ?? null)
    return envelope.data
  }
  return {
    list: () => request(),
    put: (plan, baseRevision) => request('PUT', { plan, baseRevision }),
    remove: (planId, baseRevision) => request('DELETE', { planId, baseRevision }),
  }
}
export interface PlanSyncConflict {
  id: string
  base: PkPlan | null
  local: PkPlan | null
  remote: PkPlanItem | null
  fields: PlanMergeConflict[]
}
interface SyncCache {
  placeholder: { id: string; key: string } | null
  bases: Record<string, PkPlanItem>
  plans?: PkPlan[]
  drafts?: Record<string, PkPlan>
}
export const PK_SYNC_DEBOUNCE_MS = 3000
export const PK_SYNC_FOCUS_MS = 30000
export type PkSyncReconcileResult =
  | 'merged'
  | 'adopted'
  | 'uploaded'
  | 'idle'
  | 'blocked'
  | 'failed'
const copy = <T>(value: T): T => JSON.parse(JSON.stringify(value))
const cacheKey = (owner: number) => `pk.planSync.v3.${owner}`

export function createScheduleSyncController(deps: { transport: PkSyncTransport }) {
  const store = useScheduleStore()
  const conflicts = shallowRef<PlanSyncConflict[]>([])
  const drafts = shallowRef<Record<string, PkPlan>>({})
  const mergeBlocked = shallowRef(false)
  const mergeBlockedReason = shallowRef<'capacity' | 'rejected' | null>(null)
  const notice = shallowRef<string | null>(null)
  let bases: Record<string, PkPlanItem> = Object.create(null)
  let owner = 0,
    generation = 0,
    lastRead = 0,
    retryDelay = PK_SYNC_DEBOUNCE_MS
  let enabled = false,
    reconciled = false,
    authStopped = false,
    persistenceFailed = false
  const adoptionRequired = shallowRef(false)
  let timer: ReturnType<typeof setTimeout> | null = null
  let operation: Promise<PkSyncReconcileResult> | null = null
  let placeholder: { id: string; key: string } | null = null
  const setBlocked = (reason: 'capacity' | 'rejected') => {
    mergeBlocked.value = true
    mergeBlockedReason.value = reason
  }
  const clearBlocked = () => {
    mergeBlocked.value = false
    mergeBlockedReason.value = null
  }
  // A placeholder stops masking its plan once that plan has a server base:
  // clearing it afterwards is an update, never the DELETE of a real plan.
  const contentPlans = () =>
    store.state.plans.filter(
      (p) =>
        !(placeholder?.id === p.id && placeholder.key === schedulePlanKey(p)) || !!bases[p.id],
    )
  const local = (id: string) => contentPlans().find((p) => p.id === id) ?? null
  const dirtyIDs = () =>
    [...new Set([...Object.keys(bases), ...contentPlans().map((p) => p.id)])].filter(
      (id) => schedulePlanKey(local(id)) !== schedulePlanKey(bases[id]?.plan ?? null),
    )
  const isDirty = () => persistenceFailed || dirtyIDs().length > 0
  const current = (run: number) => enabled && run === generation && store.getSyncOwner() === owner
  function clearTimer() {
    if (timer !== null) clearTimeout(timer)
    timer = null
  }
  function readCache(id: number): SyncCache | null {
    try {
      const raw = localStorage.getItem(cacheKey(id))
      if (!raw) return null
      const data = JSON.parse(raw)
      // Plans may be absent when persistence degraded under storage pressure;
      // start() rebuilds them from the acknowledged bases.
      return data && data.bases ? (data as SyncCache) : null
    } catch {
      return null
    }
  }
  function persist(): boolean {
    if (!enabled || adoptionRequired.value || store.getSyncOwner() !== owner) return false
    const shapes: Array<SyncCache> = [
      {
        placeholder,
        bases,
        plans: store.snapshotForSync().plans,
        drafts: drafts.value,
      },
      { placeholder, bases, plans: [], drafts: drafts.value },
      { placeholder, bases, plans: [], drafts: {} },
    ]
    for (const shape of shapes) {
      try {
        localStorage.setItem(cacheKey(owner), JSON.stringify(shape))
        persistenceFailed = false
        return true
      } catch {
        // Storage pressure: retry with the next, smaller shape.
      }
    }
    persistenceFailed = true
    return false
  }
  function apply(id: string, plan: PkPlan | null) {
    const plans = copy(contentPlans())
    const index = plans.findIndex((p) => p.id === id)
    if (index >= 0) {
      if (plan) plans[index] = copy(plan)
      else plans.splice(index, 1)
    } else if (plan) plans.push(copy(plan))
    else return
    store.applyPlanItems(plans)
    if (!plans.length)
      placeholder = { id: store.state.plans[0].id, key: schedulePlanKey(store.state.plans[0]) }
  }
  function reconcileItem(id: string, remote: PkPlanItem | null) {
    const base = bases[id]?.plan ?? null,
      own = local(id)
    const result = mergeSchedulePlan(base, own, remote?.plan ?? null)
    conflicts.value = conflicts.value.filter((c) => c.id !== id)
    if (result.conflicts.length) {
      conflicts.value = [
        ...conflicts.value,
        { id, base: copy(base), local: copy(own), remote: copy(remote), fields: result.conflicts },
      ]
      return
    }
    apply(id, result.plan)
    if (remote) bases[id] = copy(remote)
    else if (!result.plan) delete bases[id]
  }
  function archive(id: string, plan: PkPlan): boolean {
    drafts.value = { ...drafts.value, [id]: copy(plan) }
    // Preserve a durable recovery copy before removing the visible local plan.
    if (!persist()) return false
    apply(id, null)
    delete bases[id]
    conflicts.value = conflicts.value.filter((c) => c.id !== id)
    notice.value = plan.name
    return true
  }
  function trimOverflow(remoteIDs: Set<string>) {
    const overflow = store.state.plans.length - MAX_PLANS
    if (overflow <= 0) return
    const candidates = store.state.plans.filter((p) => !remoteIDs.has(p.id)).slice(-overflow)
    for (const plan of candidates) {
      if (!archive(plan.id, plan)) break
    }
    setBlocked('capacity')
  }
  function schedule(delay = PK_SYNC_DEBOUNCE_MS) {
    clearTimer()
    if (!enabled || authStopped || adoptionRequired.value || mergeBlocked.value || !isDirty())
      return
    if (dirtyIDs().every((id) => conflicts.value.some((c) => c.id === id))) return
    timer = setTimeout(() => {
      timer = null
      void run(!reconciled)
    }, delay)
  }
  function fail(error: unknown) {
    if (error instanceof PkSyncError && error.status === 401) authStopped = true
    else if (error instanceof PkSyncError && [400, 403].includes(error.status))
      setBlocked('rejected')
    else {
      schedule(retryDelay)
      retryDelay = Math.min(retryDelay * 2, 60000)
    }
  }
  async function upload(runID: number) {
    for (const id of dirtyIDs()) {
      if (!current(runID)) return
      if (conflicts.value.some((c) => c.id === id)) continue
      const plan = copy(local(id)),
        base = bases[id]?.revision ?? 0
      try {
        if (plan) {
          const item = await deps.transport.put(plan, base)
          if (!current(runID)) return
          if (!Number.isSafeInteger(item.revision) || item.revision < 1)
            throw new Error('Missing plan revision')
          bases[id] = copy(item)
        } else {
          await deps.transport.remove(id, base)
          if (!current(runID)) return
          delete bases[id]
        }
        persist()
        retryDelay = PK_SYNC_DEBOUNCE_MS
      } catch (error) {
        if (!current(runID)) return
        if (error instanceof PkSyncError && error.status === 410) reconcileItem(id, null)
        else if (error instanceof PkSyncError && error.status === 409 && error.remote)
          reconcileItem(id, error.remote)
        else if (error instanceof PkSyncError && error.status === 409) {
          setBlocked('capacity')
          return
        } else {
          fail(error)
          return
        }
        persist()
      }
    }
    schedule()
  }
  async function perform(read: boolean, runID: number): Promise<PkSyncReconcileResult> {
    try {
      if (read) {
        const items = await deps.transport.list()
        if (!current(runID)) return 'failed'
        // Server order is plan_id (random); present plans in creation order.
        items.sort(
          (a, b) =>
            a.plan.createdAt - b.plan.createdAt ||
            (a.plan.id < b.plan.id ? -1 : a.plan.id > b.plan.id ? 1 : 0),
        )
        const remote = new Map(items.map((item) => [item.plan.id, item]))
        // A fresh empty default is a UI placeholder, not an intentional new plan.
        if (
          !Object.keys(bases).length &&
          store.getSyncedAt() &&
          !store.isSyncDirty() &&
          items.length
        )
          store.applyPlanItems(items.map((item) => item.plan))
        for (const id of new Set([
          ...Object.keys(bases),
          ...contentPlans().map((p) => p.id),
          ...remote.keys(),
        ]))
          reconcileItem(id, remote.get(id) ?? null)
        trimOverflow(new Set(remote.keys()))
        reconciled = true
        lastRead = Date.now()
        persist()
      }
      await upload(runID)
      if (!current(runID)) return 'failed'
      return conflicts.value.length || mergeBlocked.value
        ? 'blocked'
        : isDirty()
          ? 'failed'
          : 'merged'
    } catch (error) {
      if (current(runID)) fail(error)
      return 'failed'
    }
  }
  function run(read: boolean): Promise<PkSyncReconcileResult> {
    if (!enabled || authStopped || adoptionRequired.value) return Promise.resolve('idle')
    if (operation) return operation
    clearTimer()
    const runID = generation
    const pending = perform(read, runID)
    operation = pending
    void pending.finally(() => {
      if (operation === pending) operation = null
    })
    return pending
  }
  function start(userId: number) {
    stop()
    owner = userId
    enabled = userId > 0
    if (!enabled) return
    const previous = store.getSyncOwner(),
      cache = readCache(owner)
    bases = Object.assign(Object.create(null), cache?.bases ?? {})
    placeholder = cache?.placeholder ?? null
    drafts.value = cache?.drafts ?? {}
    // Never transfer another account's plans. Each owner has a separate cache.
    if (previous && previous !== owner) {
      if (!store.setSyncOwner(owner)) {
        adoptionRequired.value = true
        return
      }
      store.applyPlanItems(
        cache?.plans?.length ? cache.plans : Object.values(bases).map((b) => b.plan),
      )
      if (!cache)
        placeholder = { id: store.state.plans[0].id, key: schedulePlanKey(store.state.plans[0]) }
    } else if (
      !previous &&
      !(
        store.state.plans.length === 1 &&
        store.state.plans.every(
          (p) => !p.stagedCourses.length && !p.selectedCourses.length && !p.customEvents.length,
        )
      )
    ) {
      adoptionRequired.value = true
      return
    } else if (!store.setSyncOwner(owner)) {
      adoptionRequired.value = true
      return
    }
    if (
      !cache &&
      !store.isSyncDirty() &&
      store.state.plans.length === 1 &&
      store.state.plans.every((p) => !p.stagedCourses.length && !p.customEvents.length)
    )
      placeholder = { id: store.state.plans[0].id, key: schedulePlanKey(store.state.plans[0]) }
    persist()
  }
  function stop() {
    clearTimer()
    enabled = false
    generation++
    reconciled = false
    operation = null
    conflicts.value = []
    drafts.value = {}
    notice.value = null
    adoptionRequired.value = false
    authStopped = false
    persistenceFailed = false
    clearBlocked()
    placeholder = null
    retryDelay = PK_SYNC_DEBOUNCE_MS
    lastRead = 0
  }
  function onLocalChange() {
    if (!enabled) return
    store.markSyncDirty()
    persist()
    schedule()
  }
  async function saveNow(adoptPreviousOwner = false) {
    if (!enabled || authStopped) return false
    if (adoptionRequired.value) {
      if (!adoptPreviousOwner || !store.setSyncOwner(owner)) return false
      adoptionRequired.value = false
      bases = Object.create(null)
    }
    mergeBlocked.value = false
    mergeBlockedReason.value = null
    persist()
    await run(!reconciled)
    return !isDirty() && !conflicts.value.length
  }
  async function resolveConflict(id: string, choices: Record<string, 'local' | 'remote'>) {
    const conflict = conflicts.value.find((c) => c.id === id)
    if (!conflict) return
    // Recompute against edits made while the panel was open.
    const result = mergeSchedulePlan(
      conflict.base,
      local(id),
      conflict.remote?.plan ?? null,
      choices,
    )
    if (result.conflicts.length) return
    if (!conflict.remote && result.plan) {
      if (!archive(id, result.plan)) return
    } else {
      apply(id, result.plan)
      if (conflict.remote) bases[id] = copy(conflict.remote)
      else delete bases[id]
    }
    conflicts.value = conflicts.value.filter((c) => c.id !== id)
    persist()
    await run(false)
  }
  function restoreDraft(id: string) {
    const draft = drafts.value[id]
    if (!enabled || !draft) return false
    if (contentPlans().length >= MAX_PLANS) {
      mergeBlocked.value = true
      return false
    }
    const plan = { ...copy(draft), id: `plan_${crypto.randomUUID()}` }
    apply(plan.id, plan)
    const next = { ...drafts.value }
    delete next[id]
    drafts.value = next
    clearBlocked()
    onLocalChange()
    return true
  }
  return {
    conflicts,
    drafts,
    notice,
    mergeBlocked,
    mergeBlockedReason,
    start,
    stop,
    onLocalChange,
    isDirty,
    saveNow,
    resolveConflict,
    restoreDraft,
    needsOwnerConfirmation: () => adoptionRequired.value,
    clearNotice: () => {
      notice.value = null
    },
    syncOnPageEnter: () => {
      clearBlocked()
      return run(true)
    },
    onFocus: () =>
      Date.now() - lastRead >= PK_SYNC_FOCUS_MS ? run(true) : Promise.resolve('idle' as const),
    flushPendingUpload: () => {
      void run(!reconciled)
    },
  }
}
export type ScheduleSyncController = ReturnType<typeof createScheduleSyncController>
export const scheduleSync = createScheduleSyncController({ transport: createFetchTransport() })
let wired = false
export function startScheduleSync(userId: number) {
  if (!wired) {
    wired = true
    setSolidifyHook(() => scheduleSync.onLocalChange())
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'hidden') scheduleSync.flushPendingUpload()
        else void scheduleSync.onFocus()
      })
      window.addEventListener('online', () => scheduleSync.flushPendingUpload())
      window.addEventListener('focus', () => {
        void scheduleSync.onFocus()
      })
    }
  }
  scheduleSync.start(userId)
}
export function stopScheduleSync() {
  scheduleSync.stop()
}

import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import {
  createScheduleSyncController,
  PK_SYNC_DEBOUNCE_MS,
  PkSyncError,
  type PkPlanItem,
  type ScheduleSyncController,
} from '../src/site/composables/useScheduleSync'
import { useScheduleStore, setSolidifyHook } from '../src/site/composables/useScheduleStore'
import type { PkPlan } from '../src/site/types/pk'
const plan = (id = 'p', name = 'Plan'): PkPlan => ({
  id,
  name,
  createdAt: 1,
  stagedCourses: [],
  selectedCourses: [],
  customEvents: [],
})
const item = (p = plan(), revision = 1): PkPlanItem => ({
  plan: p,
  revision,
  updatedAt: '2026-09-21T00:00:00Z',
})
let controller: ScheduleSyncController
const store = useScheduleStore()
let memory: Map<string, string>
function setup(items: PkPlanItem[] = [item()]) {
  const list = vi.fn(async () => structuredClone(items))
  const put = vi.fn(async (p: PkPlan, rev: number) => item(structuredClone(p), rev + 1))
  const remove = vi.fn(async () => {})
  controller = createScheduleSyncController({ transport: { list, put, remove } })
  setSolidifyHook(() => controller.onLocalChange())
  controller.start(7)
  return { list, put, remove }
}
beforeEach(() => {
  vi.useFakeTimers()
  memory = new Map()
  const storage = {
    getItem: (k: string) => memory.get(k) ?? null,
    setItem: (k: string, v: string) => {
      memory.set(k, v)
    },
    removeItem: (k: string) => memory.delete(k),
  }
  vi.stubGlobal('localStorage', storage)
  vi.stubGlobal('window', { localStorage: storage })
  setSolidifyHook(null)
  store.applyRemoteSnapshot({
    plans: [plan('placeholder')],
    activePlanId: 'placeholder',
    majorSelected: {},
    weekView: { week: null, useCurrent: false },
  })
})
afterEach(() => {
  controller?.stop()
  setSolidifyHook(null)
  vi.useRealTimers()
  vi.unstubAllGlobals()
})
describe('per-plan sync state machine', () => {
  test('adopts remote without uploading the fresh empty placeholder', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    expect(store.state.plans.map((p) => p.id)).toEqual(['p'])
    expect(t.put).not.toHaveBeenCalled()
    expect(controller.isDirty()).toBe(false)
  })
  test('only uploads the changed plan and persists its base', async () => {
    const t = setup([item(), item(plan('q'))])
    await controller.syncOnPageEnter()
    store.renamePlan('p', 'Changed')
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(t.put).toHaveBeenCalledTimes(1)
    expect(t.put.mock.calls[0][0].id).toBe('p')
    expect(t.put.mock.calls[0][1]).toBe(1)
    expect(JSON.parse(memory.get('pk.planSync.v3.7')!).bases.p.revision).toBe(2)
  })
  test('week, active plan and major preferences neither write nor poll', async () => {
    const t = setup([item(), item(plan('q'))])
    await controller.syncOnPageEnter()
    store.setWeekView({ week: 4, useCurrent: false })
    store.switchPlan('q')
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: 'm' })
    store.solidify()
    await vi.advanceTimersByTimeAsync(120000)
    expect(t.put).not.toHaveBeenCalled()
    expect(t.list).toHaveBeenCalledTimes(1)
  })
  test('clean remote deletion removes the plan without reviving its old ID', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    t.list.mockResolvedValue([])
    await controller.syncOnPageEnter()
    expect(store.state.plans.some((p) => p.id === 'p')).toBe(false)
    expect(t.put).not.toHaveBeenCalled()
    expect(controller.isDirty()).toBe(false)
  })
  test('dirty remote deletion becomes one conflict and an independent recovery draft', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    store.renamePlan('p', 'Local')
    t.list.mockResolvedValue([])
    await controller.syncOnPageEnter()
    await controller.syncOnPageEnter()
    expect(controller.conflicts.value).toHaveLength(1)
    expect(t.put).not.toHaveBeenCalled()
    await controller.resolveConflict('p', { '[]': 'local' })
    expect(Object.values(controller.drafts.value).map((p) => p.name)).toEqual(['Local'])
    expect(store.state.plans.some((p) => p.id === 'p')).toBe(false)
    expect(controller.restoreDraft('p')).toBe(true)
    await vi.advanceTimersByTimeAsync(3000)
    expect(t.put.mock.calls[0][0].id).not.toBe('p')
    expect(t.put.mock.calls[0][1]).toBe(0)
  })
  test('conflict resolution changes only the conflicting field', async () => {
    const b = plan()
    b.customEvents = [{ id: 'e', label: 'Original', day: 1, sections: [1], weeks: [1] }]
    const t = setup([item(b)])
    await controller.syncOnPageEnter()
    store.state.plans[0].customEvents[0].label = 'Local'
    store.solidify()
    const remote = structuredClone(b)
    remote.name = 'Remote name'
    remote.customEvents[0].label = 'Remote'
    t.list.mockResolvedValue([item(remote, 2)])
    await controller.syncOnPageEnter()
    expect(controller.conflicts.value[0].fields.map((c) => c.path)).toEqual([
      ['events', 'e', 'label'],
    ])
    await controller.resolveConflict('p', { '["events","e","label"]': 'local' })
    expect(t.put.mock.calls[0][0]).toMatchObject({
      name: 'Remote name',
      customEvents: [{ label: 'Local' }],
    })
    expect(t.put.mock.calls[0][1]).toBe(2)
  })
  test('409 carries the remote revision and merges disjoint edits without another GET', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    store.renamePlan('p', 'Local')
    const remote = plan()
    remote.customEvents = [{ id: 'e', label: 'Remote event', day: 1, sections: [1], weeks: [1] }]
    t.put.mockRejectedValueOnce(new PkSyncError('conflict', 409, item(remote, 2)))
    await vi.advanceTimersByTimeAsync(6000)
    expect(t.list).toHaveBeenCalledTimes(1)
    expect(t.put).toHaveBeenCalledTimes(2)
    expect(t.put.mock.calls[1][0]).toMatchObject({
      name: 'Local',
      customEvents: [{ label: 'Remote event' }],
    })
    expect(t.put.mock.calls[1][1]).toBe(2)
  })
  test('new edits during upload remain dirty and use the acknowledged revision', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    let finish!: (value: PkPlanItem) => void
    t.put.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    store.renamePlan('p', 'First')
    await vi.advanceTimersByTimeAsync(3000)
    store.renamePlan('p', 'Second')
    finish(item(plan('p', 'First'), 2))
    await vi.advanceTimersByTimeAsync(3000)
    expect(t.put.mock.calls[1][0].name).toBe('Second')
    expect(t.put.mock.calls[1][1]).toBe(2)
  })
  test('network failure retries only dirty content with backoff', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    t.put.mockRejectedValueOnce(new Error('offline'))
    store.renamePlan('p', 'Local')
    await vi.advanceTimersByTimeAsync(6000)
    expect(t.put).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(60000)
    expect(t.put).toHaveBeenCalledTimes(2)
  })
  test('failed initial read prevents upload', async () => {
    const t = setup()
    t.list.mockRejectedValue(new Error('offline'))
    store.renamePlan('placeholder', 'Local')
    await controller.syncOnPageEnter()
    expect(t.put).not.toHaveBeenCalled()
    t.list.mockResolvedValue([])
    await vi.advanceTimersByTimeAsync(3000)
    expect(t.put).toHaveBeenCalledTimes(1)
  })
  test('quota response preserves pending local changes and stops automatic retry', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    t.put.mockRejectedValue(new PkSyncError('quota', 409))
    store.renamePlan('p', 'Local')
    await vi.advanceTimersByTimeAsync(120000)
    expect(t.put).toHaveBeenCalledTimes(1)
    expect(controller.isDirty()).toBe(true)
    expect(store.state.plans[0].name).toBe('Local')
  })
  test('stale callback cannot mutate a different account', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    let finish!: (v: PkPlanItem[]) => void
    t.list.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    const pending = controller.syncOnPageEnter()
    controller.start(8)
    finish([item(plan('private', 'Previous account'))])
    await pending
    expect(store.state.plans.some((p) => p.id === 'private')).toBe(false)
    expect(t.put).not.toHaveBeenCalled()
  })
  test('switching accounts preserves separate offline caches', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    store.renamePlan('p', 'Offline')
    controller.start(8)
    t.list.mockResolvedValue([item(plan('q', 'Other account'))])
    await controller.syncOnPageEnter()
    expect(store.state.plans.map((p) => p.id)).toEqual(['q'])
    controller.start(7)
    expect(store.state.plans[0].name).toBe('Offline')
  })
  test('guest content needs explicit adoption before upload', async () => {
    store.state.plans[0].customEvents = [
      { id: 'e', label: 'Guest', day: 1, sections: [1], weeks: [1] },
    ]
    store.solidify()
    const t = setup([])
    await controller.syncOnPageEnter()
    expect(controller.needsOwnerConfirmation()).toBe(true)
    expect(t.put).not.toHaveBeenCalled()
    expect(await controller.saveNow(true)).toBe(true)
    expect(t.put).toHaveBeenCalledTimes(1)
  })
  test('authentication failures stop background retries', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    t.put.mockRejectedValue(new PkSyncError('auth', 401))
    store.renamePlan('p', 'Local')
    await vi.advanceTimersByTimeAsync(120000)
    expect(t.put).toHaveBeenCalledTimes(1)
  })
  test('deleting a local plan uses its own revision', async () => {
    const t = setup([item(), item(plan('q'))])
    await controller.syncOnPageEnter()
    store.deletePlan('p')
    await vi.advanceTimersByTimeAsync(3000)
    expect(t.remove).toHaveBeenCalledWith('p', 1)
    expect(t.put).not.toHaveBeenCalled()
  })
  test('focus refresh is throttled and no request runs while stopped', async () => {
    const t = setup()
    await controller.syncOnPageEnter()
    await controller.onFocus()
    expect(t.list).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(30000)
    await controller.onFocus()
    expect(t.list).toHaveBeenCalledTimes(2)
    controller.stop()
    await controller.syncOnPageEnter()
    expect(t.list).toHaveBeenCalledTimes(2)
  })
})
test('a recovery draft must persist before its last visible local copy is removed', async () => {
  const t = setup()
  await controller.syncOnPageEnter()
  store.renamePlan('p', 'Unsaved')
  t.list.mockResolvedValue([])
  await controller.syncOnPageEnter()
  vi.spyOn(localStorage, 'setItem').mockImplementation((key, value) => {
    if (key.startsWith('pk.planSync')) throw new Error('full')
    memory.set(key, value)
  })
  await controller.resolveConflict('p', { '[]': 'local' })
  expect(store.state.plans.find((p) => p.id === 'p')?.name).toBe('Unsaved')
  expect(controller.conflicts.value).toHaveLength(1)
})

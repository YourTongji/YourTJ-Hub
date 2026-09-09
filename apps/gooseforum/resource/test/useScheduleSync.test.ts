import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import {
  createScheduleSyncController,
  PK_SYNC_DEBOUNCE_MS,
  PkSyncError,
  type PkSyncPayload,
  type PkSyncRemoteSnapshot,
  type PkSyncTransport,
  type ScheduleSyncController,
} from '../src/site/composables/useScheduleSync'
import { setSolidifyHook, useScheduleStore } from '../src/site/composables/useScheduleStore'

// 云同步状态机（useScheduleSync）单测：传输层注入 fake，vi.useFakeTimers 驱动
// 防抖/停摆语义。覆盖：进页 GET 四分支、防抖合并上传、applyingRemote 防回灌、
// 400/401 停止、登出停止、visibility 冲刷、未登录零网络。

function makeStorage(initial: Record<string, string> = {}) {
  const storage = new Map<string, string>(Object.entries(initial))
  return {
    localStorage: {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => storage.set(key, value)),
      removeItem: vi.fn((key: string) => storage.delete(key)),
    },
  }
}

const UPDATED_AT = '2025-09-01T00:00:00.000000000Z'
const UPDATED_AT_2 = '2025-09-02T00:00:00.000000000Z'

/** 云端快照构造（wire 形状，镜像契约 fixtures/pk-plans-get-success.json）。 */
function makeSnapshot(overrides: Partial<PkSyncRemoteSnapshot> = {}): PkSyncRemoteSnapshot {
  return {
    plans: [
      {
        id: 'plan_cloud',
        name: '方案 1',
        createdAt: 1725000000000,
        stagedCourses: [],
        selectedCourses: [],
        customEvents: [],
      },
    ],
    activePlanId: 'plan_cloud',
    majorSelected: { calendarId: 121, grade: 2025, major: '00301' },
    weekView: { week: null, useCurrent: false },
    updatedAt: UPDATED_AT,
    ...overrides,
  }
}

function makeTransport() {
  const fetchCloudSnapshot = vi.fn<() => Promise<PkSyncRemoteSnapshot | null>>()
  const putCloudSnapshot = vi.fn<(payload: PkSyncPayload) => Promise<{ updatedAt: string }>>()
  const transport: PkSyncTransport = { fetchCloudSnapshot, putCloudSnapshot }
  return { transport, fetchCloudSnapshot, putCloudSnapshot }
}

type Store = ReturnType<typeof useScheduleStore>

/** 重置模块级单例 store 到单方案空态（对齐 useScheduleStore.test.ts 惯例）。 */
function resetStore(store: Store): void {
  store.clearStagedAndSelectedCourses()
  store.setMajorInfo({ calendarId: undefined, grade: undefined, major: undefined })
  while (store.state.plans.length > 1) {
    store.deletePlan(store.state.plans[0].id)
  }
  store.deletePlan(store.state.plans[0].id)
  store.setWeekView({ week: null, useCurrent: false })
}

/** 让本机非空（一个备选课程即脱离 localEmpty）。pushStagedCourse 不触发 solidify。 */
function seedLocalContent(store: Store): void {
  store.pushStagedCourse({
    courseCode: '122004',
    courseName: '高数',
    courseNameReserved: '高数',
    credit: 4,
    courseType: '必',
    courseNature: [],
    teacher: [],
    status: 0,
    courseDetail: [],
  })
}

function setup(storage: Record<string, string> = {}) {
  vi.stubGlobal('window', makeStorage(storage))
  // 先摘钩子再重置（deletePlan/switchPlan 内部会走 solidify）。
  setSolidifyHook(null)
  const store = useScheduleStore()
  resetStore(store)
  const { transport, fetchCloudSnapshot, putCloudSnapshot } = makeTransport()
  const controller = createScheduleSyncController({ transport })
  // 接线模拟 startScheduleSync 的 wireOnce（钩子指向本用例 controller）。
  setSolidifyHook(() => controller.onLocalChange())
  return { store, controller, fetchCloudSnapshot, putCloudSnapshot }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  setSolidifyHook(null)
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('useScheduleSync（排课方案云同步状态机）', () => {
  test('进页分支①：云端空 + 本机空 → 不动（零 PUT）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)

    await controller.syncOnPageEnter()

    expect(fetchCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.getSyncedAt()).toBe('')
  })

  test('进页分支②：云端空 + 本机非空 → 首登自动上传本机快照', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(null)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })

    await controller.syncOnPageEnter()

    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    const payload = putCloudSnapshot.mock.calls[0][0]
    expect(payload.plans).toHaveLength(1)
    expect(payload.plans[0].stagedCourses[0]?.courseCode).toBe('122004')
    expect(payload.activePlanId).toBe(store.state.activePlanId)
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
    expect(controller.isDirty()).toBe(false)
  })

  test('进页分支③：云端快照 + 本机空 → 整包采用（云端）且不回灌上传', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())

    await controller.syncOnPageEnter()

    expect(store.state.plans[0]?.id).toBe('plan_cloud')
    expect(store.state.activePlanId).toBe('plan_cloud')
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
    expect(controller.isDirty()).toBe(false)
    // applyingRemote 守卫：整包采用后推进虚拟时钟，不得触发任何 PUT。
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS * 2)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
  })

  test('进页分支④a：本机非空 + syncedAt 一致且非 dirty → 不动', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({
      'pk.syncedAt': JSON.stringify(UPDATED_AT),
    })
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())

    await controller.syncOnPageEnter()

    expect(controller.conflict.value).toBeNull()
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(1)
  })

  test('进页分支④b：本机非空 + syncedAt 不一致 → 弹窗二选一；「使用云端」整包采用', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())

    await controller.syncOnPageEnter()
    expect(controller.conflict.value?.updatedAt).toBe(UPDATED_AT)

    controller.useCloud()

    expect(controller.conflict.value).toBeNull()
    expect(store.state.plans[0]?.id).toBe('plan_cloud')
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
    expect(controller.isDirty()).toBe(false)
    // 「使用云端」不得回灌上传。
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
  })

  test('弹窗选「保留本地」→ 立即 PUT 本机快照覆盖云端', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    await controller.syncOnPageEnter()
    controller.keepLocal()
    await vi.advanceTimersByTimeAsync(0)

    expect(controller.conflict.value).toBeNull()
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(putCloudSnapshot.mock.calls[0][0].plans[0]?.stagedCourses[0]?.courseCode).toBe('122004')
    expect(store.getSyncedAt()).toBe(UPDATED_AT_2)
    expect(controller.isDirty()).toBe(false)
  })

  test('本会话 dirty（有未上传变更）时即使 syncedAt 一致也弹窗；关闭后下次进页再提示', async () => {
    const { store, controller, fetchCloudSnapshot } = setup({
      'pk.syncedAt': JSON.stringify(UPDATED_AT),
    })
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())

    controller.onLocalChange()
    await controller.syncOnPageEnter()
    expect(controller.conflict.value).not.toBeNull()

    // 关闭弹窗 = 暂缓，本轮不再问。
    controller.dismissConflict()
    expect(controller.conflict.value).toBeNull()

    // 下次进页仍分歧 → 再提示。
    await controller.syncOnPageEnter()
    expect(controller.conflict.value).not.toBeNull()
  })

  test('本地变更经 solidify 钩子防抖 3s 合并：多次变更只 PUT 一次', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })

    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS - 1)
    store.setWeekView({ week: 2, useCurrent: false })
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS - 1)
    expect(putCloudSnapshot).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    // 快照出口只含契约四字段。
    const payload = putCloudSnapshot.mock.calls[0][0]
    expect(Object.keys(payload).sort()).toEqual(
      ['activePlanId', 'baseUpdatedAt', 'majorSelected', 'plans', 'weekView'].sort(),
    )
    expect(controller.isDirty()).toBe(false)
  })

  test('applyRemoteSnapshot 直接调用同样防回灌（守卫不依赖进页路径）', async () => {
    const { store, controller, putCloudSnapshot } = setup()
    controller.start()

    store.applyRemoteSnapshot(makeSnapshot())

    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS * 2)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.state.plans[0]?.id).toBe('plan_cloud')
  })

  test('PUT 400：静默停止本轮（无自动重试，dirty 保持），下次变更开新一轮', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    putCloudSnapshot.mockRejectedValue(new PkSyncError('方案数超出上限', 400, 'rejected'))
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(controller.isDirty()).toBe(true)

    // 无自动重试。
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS * 5)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)

    // 新的本地方案变更开新一轮（允许再次尝试）。
    store.setWeekView({ week: 2, useCurrent: false })
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    warnSpy.mockRestore()
  })

  test('PUT 401：停止触发直至下次进页（重新登录后恢复）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    putCloudSnapshot.mockRejectedValue(new PkSyncError('未登录', 401, 'unauthenticated'))

    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)

    // 401 停摆：本地再变更不再上传。
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS * 2)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)

    // 下次进页解除停摆：云端空 + 本机非空 → 自动上传成功。
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
  })

  test('登出停止：stop 后取消防抖不再上传，本地数据原样保留', async () => {
    const { store, controller, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)

    store.solidify()
    controller.stop()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS * 2)

    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(1)
  })

  test('visibility hidden → best-effort 冲刷未到期的防抖 PUT', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })

    store.solidify()
    // 防抖窗口未到，模拟切后台冲刷。
    controller.flushPendingUpload()
    await vi.advanceTimersByTimeAsync(0)

    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
  })

  test('未登录（未 start）：全程零网络请求，钩子触发也不标脏', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    seedLocalContent(store)

    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS * 2)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(controller.isDirty()).toBe(false)

    await controller.syncOnPageEnter()
    expect(fetchCloudSnapshot).not.toHaveBeenCalled()
  })

  test('进页 GET 失败（网络错误/401 探测）静默不作为', async () => {
    const { controller, fetchCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockRejectedValue(new PkSyncError('未登录', 401, 'unauthenticated'))

    await expect(controller.syncOnPageEnter()).resolves.toBeUndefined()
    expect(controller.conflict.value).toBeNull()
  })
  test('failed entry GET never permits a blind upload', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockRejectedValue(new PkSyncError('offline', 0, 'network'))
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
  })

  test('entry GET and unresolved conflict suspend all uploads', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    let resolve!: (value: PkSyncRemoteSnapshot) => void
    fetchCloudSnapshot.mockReturnValue(new Promise(r => { resolve = r }))
    const entering = controller.syncOnPageEnter()
    seedLocalContent(store)
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    resolve(makeSnapshot())
    await entering
    controller.flushPendingUpload()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
  })

  test('a stopped controller ignores late entry results', async () => {
    const { store, controller, fetchCloudSnapshot } = setup()
    controller.start()
    let resolve!: (value: PkSyncRemoteSnapshot) => void
    fetchCloudSnapshot.mockReturnValue(new Promise(r => { resolve = r }))
    const entering = controller.syncOnPageEnter()
    controller.stop()
    resolve(makeSnapshot())
    await entering
    expect(store.state.activePlanId).not.toBe('plan_cloud')
    expect(controller.conflict.value).toBeNull()
  })

  test('keep-local remains pending after a transient error', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    await controller.syncOnPageEnter()
    putCloudSnapshot.mockRejectedValueOnce(new PkSyncError('offline', 0, 'network'))
    controller.keepLocal()
    await vi.advanceTimersByTimeAsync(0)
    expect(controller.isDirty()).toBe(true)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })
    controller.flushPendingUpload()
    await vi.advanceTimersByTimeAsync(0)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
  })

  test('edits made during PUT remain pending and upload the newer snapshot', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    let resolve!: (value: { updatedAt: string }) => void
    putCloudSnapshot.mockReturnValueOnce(new Promise(r => { resolve = r }))
    seedLocalContent(store)
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    store.createPlan()
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    resolve({ updatedAt: UPDATED_AT })
    await vi.advanceTimersByTimeAsync(0)
    expect(controller.isDirty()).toBe(true)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(putCloudSnapshot.mock.calls[1][0].plans).toHaveLength(2)
  })

  test('account changes require a choice even when the next cloud is empty', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({ 'pk.syncOwner': '1' })
    seedLocalContent(store)
    controller.start(2)
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(controller.conflict.value).not.toBeNull()
    controller.useCloud()
    expect(store.getSyncOwner()).toBe(2)
    expect(store.state.plans[0].stagedCourses).toHaveLength(0)
  })

  test('catalog persistence without snapshot changes does not upload', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    await controller.syncOnPageEnter()
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
  })

  test('failed plan persistence cannot advance the cloud clock', async () => {
    const { store, controller, fetchCloudSnapshot } = setup()
    controller.start()
    const setItem = window.localStorage.setItem as ReturnType<typeof vi.fn>
    const save = setItem.getMockImplementation()!
    setItem.mockImplementation((key, value) => {
      if (key === 'pk.plans') throw new Error('quota')
      return save(key, value)
    })
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    await controller.syncOnPageEnter()
    expect(store.getSyncedAt()).toBe('')
    expect(controller.isDirty()).toBe(true)
  })

  test('409 re-fetches cloud and suspends the stale upload', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValueOnce(makeSnapshot())
    await controller.syncOnPageEnter()
    store.setWeekView({ week: 2, useCurrent: false })
    putCloudSnapshot.mockRejectedValue(new PkSyncError('conflict', 409, 'rejected'))
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({ updatedAt: UPDATED_AT_2 }))
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot.mock.calls[0][0].baseUpdatedAt).toBe(UPDATED_AT)
    expect(controller.conflict.value?.updatedAt).toBe(UPDATED_AT_2)
    controller.flushPendingUpload()
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
  })

  test('an initial create conflict fetches the winning cloud snapshot', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValueOnce(null).mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockRejectedValue(new PkSyncError('conflict', 409, 'rejected'))
    await controller.syncOnPageEnter()
    await vi.advanceTimersByTimeAsync(0)
    expect(controller.conflict.value).not.toBeNull()
  })

})

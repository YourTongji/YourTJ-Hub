import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import {
  createScheduleSyncController,
  PK_SYNC_AUTOSAVE_MS,
  PK_SYNC_DEBOUNCE_MS,
  PkSyncError,
  type PkSyncPayload,
  type PkSyncRemoteSnapshot,
  type PkSyncTransport,
  type ScheduleSyncController,
} from '../src/site/composables/useScheduleSync'
import { setSolidifyHook, useScheduleStore } from '../src/site/composables/useScheduleStore'
import { i18n } from '../src/runtime/i18n'

/** 默认方案名（跟随当前 locale，构造云端方案名用）。 */
function defaultPlanName(n: number): string {
  return i18n.global.t('schedule.planDefaultName', { n })
}

/** 自动恢复方案名（跟随当前 locale）。 */
function autoRestoreName(n: number): string {
  return i18n.global.t('schedule.planAutoRestoreName', { n })
}

// 云同步状态机（useScheduleSync）单测：传输层注入 fake，vi.useFakeTimers 驱动
// 防抖/心跳语义。覆盖（#573/#571）：进页对账（云端为主：干净本地采用云端、从未
// 云同步的分歧本地并入自动恢复合并保留为「[本地自动恢复]方案x」、空本地绝不覆盖
// 非空云端）、对账结果返回值、网络错误心跳自动补传、手动保存 saveNow、账号切换
// 不自动上传上一账号方案、applyingRemote 防回灌、400/401 停止、登出停止、
// visibility 冲刷、未登录零网络。

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
        name: defaultPlanName(1),
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

describe('useScheduleSync（排课方案云同步状态机 #573）', () => {
  test('进页：云端空 + 本机空 → 不动（零 PUT）', async () => {
    const { controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)

    await controller.syncOnPageEnter()

    expect(fetchCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(controller.isDirty()).toBe(false)
  })

  test('进页：云端空 + 本机非空 → 自动上传本机快照', async () => {
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

  test('进页：云端有快照 + 本机空 → 整包采用（云端）且不回灌上传', async () => {
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

  test('进页：本机非空 + syncedAt 一致且非 dirty → 不动', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({
      'pk.syncedAt': JSON.stringify(UPDATED_AT),
    })
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())

    await controller.syncOnPageEnter()

    expect(controller.notice.value).toBeNull()
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(1)
  })

  test('进页：从未云同步的非空本地方案与云端分歧 → 并入自动恢复合并，云端为主（#571）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    const result = await controller.syncOnPageEnter()

    expect(result).toBe('merged')
    expect(controller.notice.value).not.toBeNull()
    // 云端为主：云端方案在前，本地方案克隆为恢复方案追加保留（不盲传覆盖云端）。
    expect(store.state.plans).toHaveLength(2)
    expect(store.state.plans[0]?.id).toBe('plan_cloud')
    expect(store.state.plans[1]?.name).toBe(autoRestoreName(2))
    expect(store.state.plans[1]?.stagedCourses[0]?.courseCode).toBe('122004')
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(putCloudSnapshot.mock.calls[0][0].plans).toHaveLength(2)
    expect(store.getSyncedAt()).toBe(UPDATED_AT_2)
    expect(controller.isDirty()).toBe(false)
  })

  test('进页：本机 dirty（有未上传变更）时即使 syncedAt 一致也总是上传；同一云端版本直接上传', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({
      'pk.syncedAt': JSON.stringify(UPDATED_AT),
    })
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    controller.onLocalChange()
    await controller.syncOnPageEnter()

    // 云端版本没有前进，本地修改直接上传，不产生重复恢复方案。
    expect(controller.notice.value).toBeNull()
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(store.state.plans).toHaveLength(1)
    expect(store.getSyncedAt()).toBe(UPDATED_AT_2)
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

    // 心跳不重试 400（rejected 标志）。
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS * 2)
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

    // 401 停摆：本地再变更不再上传（心跳也不重试）。
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS * 2)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)

    // 下次进页解除停摆：云端空 + 本机非空 → 自动上传成功。
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
  })

  test('登出停止：stop 后取消防抖与心跳，不再上传，本地数据原样保留', async () => {
    const { store, controller, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)

    store.solidify()
    controller.stop()
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS * 2)

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
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS * 2)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(controller.isDirty()).toBe(false)

    await controller.syncOnPageEnter()
    expect(fetchCloudSnapshot).not.toHaveBeenCalled()
  })

  test('进页 GET 失败（网络错误/401 探测）静默不作为，心跳重试对账', async () => {
    const { controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockRejectedValue(new PkSyncError('未登录', 401, 'unauthenticated'))

    await expect(controller.syncOnPageEnter()).resolves.toBe('failed')
    expect(controller.notice.value).toBeNull()
    // 401 停摆：心跳不再重试对账。
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS * 2)
    expect(fetchCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
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

  test('网络错误 PUT：dirty 保持，心跳自动补传成功', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    putCloudSnapshot.mockRejectedValueOnce(new PkSyncError('offline', 0, 'network'))

    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(controller.isDirty()).toBe(true)

    // 心跳窗口到 → 自动重试补传成功。
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
    expect(controller.isDirty()).toBe(false)
  })

  test('saveNow 手动保存：立即 PUT 本地方案', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })

    store.solidify()
    const ok = await controller.saveNow()
    await vi.advanceTimersByTimeAsync(0)

    expect(ok).toBe(true)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
  })

  test('saveNow 网络失败：返回 false，dirty 保持待心跳补传', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()
    seedLocalContent(store)
    putCloudSnapshot.mockRejectedValue(new PkSyncError('offline', 0, 'network'))

    store.solidify()
    const ok = await controller.saveNow()
    await vi.advanceTimersByTimeAsync(0)

    expect(ok).toBe(false)
    expect(controller.isDirty()).toBe(true)

    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS)
    expect(controller.isDirty()).toBe(false)
  })

  test('云端分歧自动恢复（#573）：网络恢复补传时本地与云端全部不同 → 保留本地为恢复方案并上传合并', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    // 本机有未上传变更（网络异常期间编辑）→ 进页发现与云端分歧。
    store.solidify()
    await controller.syncOnPageEnter()

    expect(controller.notice.value).not.toBeNull()
    expect(controller.notice.value).toBe(store.state.plans[1]?.name ?? null)
    // 云端为主 + 本地方案克隆为恢复方案追加保留。
    expect(store.state.plans).toHaveLength(2)
    expect(store.state.plans[0]?.id).toBe('plan_cloud')
    expect(store.state.plans[1]?.name).toBe(autoRestoreName(2))
    expect(store.state.plans[1]?.stagedCourses[0]?.courseCode).toBe('122004')
    // 合并快照上传（云端 + 恢复方案）。
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(putCloudSnapshot.mock.calls[0][0].plans).toHaveLength(2)
    expect(controller.isDirty()).toBe(false)
  })

  test('云端分歧但本地与云端内容一致 → 不产生恢复方案，直接推进同步时钟', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    store.applyRemoteSnapshot(makeSnapshot())
    store.markSynced(UPDATED_AT)
    await controller.syncOnPageEnter()

    expect(controller.notice.value).toBeNull()
    expect(store.state.plans).toHaveLength(1)
  })

  test('409 云端版本前进：重对账后默认上传本地（无恢复方案，无弹窗）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    fetchCloudSnapshot.mockResolvedValueOnce(makeSnapshot())
    await controller.syncOnPageEnter()
    // 本地周次变更（未上传），云端被其他端推进。
    store.setWeekView({ week: 2, useCurrent: false })
    putCloudSnapshot.mockRejectedValueOnce(new PkSyncError('conflict', 409, 'rejected'))
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({ updatedAt: UPDATED_AT_2 }))
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    await vi.advanceTimersByTimeAsync(0)

    // 409 后重对账 → 总是上传本地（内容一致不产生恢复方案）。
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(putCloudSnapshot.mock.calls[0][0].baseUpdatedAt).toBe(UPDATED_AT)
    expect(putCloudSnapshot.mock.calls[1][0].baseUpdatedAt).toBe(UPDATED_AT_2)
    expect(controller.notice.value).toBeNull()
    expect(store.getSyncedAt()).toBe(UPDATED_AT_2)
  })

  test('一个已停止的 controller 忽略迟到的进页结果', async () => {
    const { store, controller, fetchCloudSnapshot } = setup()
    controller.start()
    let resolve!: (value: PkSyncRemoteSnapshot) => void
    fetchCloudSnapshot.mockReturnValue(new Promise(r => { resolve = r }))
    const entering = controller.syncOnPageEnter()
    controller.stop()
    resolve(makeSnapshot())
    await entering
    expect(store.state.activePlanId).not.toBe('plan_cloud')
    expect(controller.notice.value).toBeNull()
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

  test('账号切换：云端非空 → 采用云端，不自动上传上一账号本地方案', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({ 'pk.syncOwner': '1' })
    seedLocalContent(store)
    controller.start(2)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    await controller.syncOnPageEnter()

    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(controller.notice.value).toBeNull()
    expect(store.state.plans[0]?.id).toBe('plan_cloud')
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(0)
    expect(store.getSyncOwner()).toBe(2)
  })

  test('账号切换：云端空 → 保留本地方案但不上传（不污染新账号云端）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({ 'pk.syncOwner': '1' })
    seedLocalContent(store)
    controller.start(2)
    fetchCloudSnapshot.mockResolvedValue(null)
    await controller.syncOnPageEnter()

    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(1)
    expect(store.getSyncOwner()).toBe(1)
    expect(controller.isDirty()).toBe(false)
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

  test('恢复方案命名接续云端方案序号（[本地自动恢复]方案 {n}）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({
      plans: [
        { id: 'cloud_1', name: defaultPlanName(1), createdAt: 1, stagedCourses: [], selectedCourses: [], customEvents: [] },
        { id: 'cloud_2', name: defaultPlanName(2), createdAt: 2, stagedCourses: [], selectedCourses: [], customEvents: [] },
      ],
      activePlanId: 'cloud_1',
    }))
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    store.solidify()
    await controller.syncOnPageEnter()

    expect(store.state.plans).toHaveLength(3)
    expect(store.state.plans[2]?.name).toBe(autoRestoreName(3))
    expect(controller.notice.value).toContain(autoRestoreName(3))
  })

  test('进页 GET 挂起期间不上传；对账完成后按策略上传', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    let resolve!: (value: PkSyncRemoteSnapshot) => void
    fetchCloudSnapshot.mockReturnValue(new Promise(r => { resolve = r }))
    const entering = controller.syncOnPageEnter()
    seedLocalContent(store)
    store.solidify()
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    // 修复后 notice 仅在合并上传成功后给出：补 PUT 成功桩（本用例聚焦 GET 挂起语义）。
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })
    resolve(makeSnapshot())
    await entering
    // 对账完成：本机 dirty + 内容分歧 → 恢复方案 + 合并上传。
    expect(controller.notice.value).not.toBeNull()
    expect(store.state.plans).toHaveLength(2)
  })

  test('进页：从未云同步的本地方案与云端一致 → 采用云端建立时钟，不产生恢复方案（#571）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    // 云端快照取自本机当前状态（深拷贝）：内容一致但从未 markSynced（无 syncedAt）。
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({ plans: JSON.parse(JSON.stringify(store.state.plans)) }))

    const result = await controller.syncOnPageEnter()

    expect(result).toBe('adopted')
    expect(store.state.plans).toHaveLength(1)
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
    expect(controller.notice.value).toBeNull()
    expect(putCloudSnapshot).not.toHaveBeenCalled()
  })

  test('进页：空脏本地 + 云端非空 → 采用云端，绝不上传空方案覆盖云端（#571）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    controller.onLocalChange() // 对账完成前的瞬态窗口：本地为空但已标脏
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())

    const result = await controller.syncOnPageEnter()

    expect(result).toBe('adopted')
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.state.plans[0]?.id).toBe('plan_cloud')
    expect(store.getSyncedAt()).toBe(UPDATED_AT)
    expect(controller.isDirty()).toBe(false)
    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS * 2)
    expect(putCloudSnapshot).not.toHaveBeenCalled()
  })

  test('进页对账返回结果：云端空+本机非空 → uploaded；时钟一致 → idle；GET 失败 → failed', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(null)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
    await expect(controller.syncOnPageEnter()).resolves.toBe('uploaded')

    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({ updatedAt: UPDATED_AT }))
    await expect(controller.syncOnPageEnter()).resolves.toBe('idle')

    fetchCloudSnapshot.mockRejectedValue(new PkSyncError('offline', 0, 'network'))
    await expect(controller.syncOnPageEnter()).resolves.toBe('failed')
  })

  test('409 后重对账：空脏本地不覆盖非空云端，采用云端（#571）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({
      'pk.syncedAt': JSON.stringify(UPDATED_AT),
    })
    controller.start()
    seedLocalContent(store)
    store.solidify() // 建立含课程的落盘基线（否则清空后 payload 无变化、钩子不触发）
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })
    await controller.syncOnPageEnter()

    // 清空内容（空本地，dirty）后 PUT 撞 409，云端同时被其他端推进。
    // clearStagedAndSelectedCourses 不走 solidify，显式触发：此时 payload 与基线
    // 不同 → 钩子生效，进入防抖上传。
    store.clearStagedAndSelectedCourses()
    store.solidify()
    putCloudSnapshot.mockRejectedValueOnce(new PkSyncError('conflict', 409, 'rejected'))
    const CLOUD_3 = '2025-09-03T00:00:00.000000000Z'
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({
      plans: [{
        id: 'plan_other',
        name: defaultPlanName(1),
        createdAt: 3,
        stagedCourses: [{ courseCode: '199999', courseName: '他端新课', courseNameReserved: '他端新课', credit: 2, courseType: '必', courseNature: [], teacher: [], status: 0, courseDetail: [] }],
        selectedCourses: [],
        customEvents: [],
      }],
      activePlanId: 'plan_other',
      updatedAt: CLOUD_3,
    }))
    putCloudSnapshot.mockResolvedValue({ updatedAt: CLOUD_3 })

    await vi.advanceTimersByTimeAsync(PK_SYNC_DEBOUNCE_MS)
    await vi.advanceTimersByTimeAsync(0)
    // PUT #1 = 首次对账上传课程内容；PUT #2 = 清空后的防抖上传（撞 409）。
    // 关键保护：409 重对账发现空脏本地与云端分歧 → 采用云端，不再第三次回传空方案。
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(putCloudSnapshot.mock.calls[1][0].plans[0]?.stagedCourses).toHaveLength(0)
    expect(store.state.plans[0]?.id).toBe('plan_other')
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(1)
    expect(store.getSyncedAt()).toBe(CLOUD_3)
  })

  test('合并 PUT 网络失败：本地保持合并前状态，重对账重新合并不翻倍（#571 review）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    store.solidify()
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
    putCloudSnapshot.mockRejectedValueOnce(new PkSyncError('offline', 0, 'network'))

    // 上传失败必须返回 failed（而非 blocked），且合并结果不落盘到本地。
    await expect(controller.syncOnPageEnter()).resolves.toBe('failed')
    expect(store.state.plans).toHaveLength(1)
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(1)
    expect(store.getSyncedAt()).toBe('')

    // 心跳重对账（失败未落盘，与「失败后刷新页面」是同一条重放路径）
    // → 从同一份本地源重新合并，恢复方案恰好一套。
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS)

    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(store.state.plans).toHaveLength(2)
    expect(store.state.plans[0]?.id).toBe('plan_cloud')
    expect(store.state.plans[1]?.name).toBe(autoRestoreName(2))
    expect(store.getSyncedAt()).toBe(UPDATED_AT_2)
  })

  test('合并 PUT 409：立即以新云端版本重对账重合并，恢复方案不翻倍（#571 review）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    store.solidify()
    fetchCloudSnapshot.mockResolvedValueOnce(makeSnapshot())
    putCloudSnapshot.mockRejectedValueOnce(new PkSyncError('conflict', 409, 'rejected'))
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({ updatedAt: UPDATED_AT_2 }))
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    await expect(controller.syncOnPageEnter()).resolves.toBe('merged')

    // 第一次 PUT 以 UPDATED_AT 为 base 撞 409，立即重对账后以 UPDATED_AT_2 重合并成功。
    expect(putCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(putCloudSnapshot.mock.calls[0][0].baseUpdatedAt).toBe(UPDATED_AT)
    expect(putCloudSnapshot.mock.calls[1][0].baseUpdatedAt).toBe(UPDATED_AT_2)
    expect(store.state.plans).toHaveLength(2)
    expect(store.state.plans[1]?.name).toBe(autoRestoreName(2))
    expect(store.getSyncedAt()).toBe(UPDATED_AT_2)
  })

  test('恢复方案序号接续云端已有恢复方案名（#571 review）', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({
      plans: [
        { id: 'cloud_1', name: defaultPlanName(1), createdAt: 1, stagedCourses: [], selectedCourses: [], customEvents: [] },
        { id: 'cloud_2', name: autoRestoreName(2), createdAt: 2, stagedCourses: [], selectedCourses: [], customEvents: [] },
      ],
      activePlanId: 'cloud_1',
    }))
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })

    store.solidify()
    await controller.syncOnPageEnter()

    expect(store.state.plans).toHaveLength(3)
    expect(store.state.plans[2]?.name).toBe(autoRestoreName(3))
  })
})

describe('sync review regressions', () => {
  test('clean local snapshot adopts a newer cloud revision without PUT', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({
      'pk.syncedAt': JSON.stringify(UPDATED_AT),
    })
    seedLocalContent(store)
    controller.start()
    fetchCloudSnapshot.mockResolvedValue(makeSnapshot({ updatedAt: UPDATED_AT_2 }))
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })
    await controller.syncOnPageEnter()
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.state.activePlanId).toBe('plan_cloud')
    expect(store.getSyncedAt()).toBe(UPDATED_AT_2)
  })

  test('dirty local state retries entry GET after a network failure', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
    controller.start()
    seedLocalContent(store)
    store.solidify()
    fetchCloudSnapshot.mockRejectedValueOnce(new PkSyncError('offline', 0, 'network'))
    fetchCloudSnapshot.mockResolvedValue(null)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
    await controller.syncOnPageEnter()
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS)
    expect(fetchCloudSnapshot).toHaveBeenCalledTimes(2)
    expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
    expect(controller.isDirty()).toBe(false)
  })

  test('switching to an empty account never reassigns or auto-uploads previous owner data', async () => {
    const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({ 'pk.syncOwner': '1' })
    seedLocalContent(store)
    controller.start(2)
    fetchCloudSnapshot.mockResolvedValue(null)
    putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
    await controller.syncOnPageEnter()
    store.setWeekView({ week: 2, useCurrent: false })
    await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS * 2)
    controller.stop()
    controller.start(2)
    await controller.syncOnPageEnter()
    expect(putCloudSnapshot).not.toHaveBeenCalled()
    expect(store.getSyncOwner()).toBe(1)
    expect(store.state.plans[0]?.stagedCourses).toHaveLength(1)
  })
})


test('offline edits against an unchanged cloud revision do not duplicate plans', async () => {
  const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({
    'pk.syncedAt': JSON.stringify(UPDATED_AT),
  })
  seedLocalContent(store)
  controller.start()
  controller.onLocalChange()
  fetchCloudSnapshot.mockResolvedValue(makeSnapshot())
  putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT_2 })
  await controller.syncOnPageEnter()
  expect(store.state.plans).toHaveLength(1)
  expect(putCloudSnapshot.mock.calls[0][0].plans).toHaveLength(1)
  expect(controller.notice.value).toBeNull()
})

test('account transfer requires explicit consent before saving', async () => {
  const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup({ 'pk.syncOwner': '1' })
  seedLocalContent(store)
  controller.start(2)
  fetchCloudSnapshot.mockResolvedValue(null)
  putCloudSnapshot.mockResolvedValue({ updatedAt: UPDATED_AT })
  await controller.syncOnPageEnter()
  expect(await controller.saveNow()).toBe(false)
  expect(putCloudSnapshot).not.toHaveBeenCalled()
  expect(await controller.saveNow(true)).toBe(true)
  expect(putCloudSnapshot).toHaveBeenCalledTimes(1)
  expect(store.getSyncOwner()).toBe(2)
})

test('oversized conflict recovery keeps both local and cloud sources intact', async () => {
  const { store, controller, fetchCloudSnapshot, putCloudSnapshot } = setup()
  seedLocalContent(store)
  controller.start()
  store.solidify()
  const local = store.snapshotForSync()
  const cloud = makeSnapshot({ plans: Array.from({ length: 10 }, (_, i) => ({
    id: `cloud_${i}`, name: `Cloud ${i}`, createdAt: i,
    selectedCourses: [], stagedCourses: [], customEvents: [],
  })) })
  fetchCloudSnapshot.mockResolvedValue(cloud)
  const result = await controller.syncOnPageEnter()
  await vi.advanceTimersByTimeAsync(PK_SYNC_AUTOSAVE_MS * 2)
  expect(store.snapshotForSync()).toEqual(local)
  expect(controller.mergeBlocked.value).toBe(true)
  expect(putCloudSnapshot).not.toHaveBeenCalled()
  expect(controller.isDirty()).toBe(true)
  expect(result).toBe('blocked')
})

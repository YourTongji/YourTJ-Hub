import { describe, expect, test, vi } from 'vitest'
import {
  createSectionTimesRefresher,
  sectionTimesEqual,
  type SectionTimesRefresherDeps,
} from '../src/site/utils/sectionTimesRefresh'
import type { SectionTime } from '../src/site/utils/sectionTimes'

const T1: SectionTime[] = [
  { section: 1, start: '08:00', end: '08:45' },
  { section: 2, start: '08:50', end: '09:35' },
]
const T2: SectionTime[] = [
  { section: 1, start: '08:05', end: '08:50' },
  { section: 2, start: '08:55', end: '09:40' },
]

function makeDeps(overrides: { fetchPayload?: () => Promise<unknown>; apply?: (times: SectionTime[]) => void } = {}): {
  deps: SectionTimesRefresherDeps
  applySpy: ReturnType<typeof vi.fn>
  fetchSpy: ReturnType<typeof vi.fn>
} {
  const fetchSpy = vi.fn(async () => ({ props: { sectionTimes: T1 } }))
  const applySpy = vi.fn()
  const deps: SectionTimesRefresherDeps = {
    fetchPayload: overrides.fetchPayload ?? fetchSpy,
    apply: overrides.apply ?? applySpy,
  }
  return { deps, applySpy, fetchSpy }
}

describe('sectionTimesEqual', () => {
  test('空数组与 undefined 等价', () => {
    expect(sectionTimesEqual(undefined, [])).toBe(true)
    expect(sectionTimesEqual([], undefined)).toBe(true)
    expect(sectionTimesEqual(undefined, undefined)).toBe(true)
  })

  test('逐节比较顺序敏感', () => {
    expect(sectionTimesEqual(T1, [...T1])).toBe(true)
    expect(sectionTimesEqual(T1, [...T1].reverse())).toBe(false)
    expect(sectionTimesEqual(T1, T2)).toBe(false)
    expect(sectionTimesEqual(T1, [T1[0]])).toBe(false)
  })
})

describe('createSectionTimesRefresher', () => {
  test('作息表变化时回调 apply，未变化时不回调', async () => {
    const { deps, applySpy, fetchSpy } = makeDeps()
    const refresher = createSectionTimesRefresher(deps, { minIntervalMs: 0, now: () => 1000, initial: T1 })

    // 首次：payload 与 initial 相同 → 不回调。
    const first = await refresher.refresh()
    expect(first).toBe(false)
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    expect(applySpy).not.toHaveBeenCalled()

    // 第二次：作息变化 → 回调。
    fetchSpy.mockResolvedValueOnce({ props: { sectionTimes: T2 } })
    const second = await refresher.refresh()
    expect(second).toBe(true)
    expect(applySpy).toHaveBeenCalledTimes(1)
    expect(applySpy).toHaveBeenCalledWith(T2)
  })

  test('初始无 initial 时首次拉取即应用', async () => {
    const { deps, applySpy } = makeDeps()
    const refresher = createSectionTimesRefresher(deps, { minIntervalMs: 0, now: () => 2000 })
    await refresher.refresh()
    expect(applySpy).toHaveBeenCalledTimes(1)
    expect(applySpy).toHaveBeenCalledWith(T1)
  })

  test('minIntervalMs 节流：间隔内重复触发不拉取', async () => {
    const { deps, fetchSpy } = makeDeps()
    let clock = 10_000
    const refresher = createSectionTimesRefresher(deps, { minIntervalMs: 5000, now: () => clock })
    await refresher.refresh()
    expect(fetchSpy).toHaveBeenCalledTimes(1)

    // 3000ms 后再次触发 → 节流跳过。
    clock = 13_000
    const throttled = await refresher.refresh()
    expect(throttled).toBe(false)
    expect(fetchSpy).toHaveBeenCalledTimes(1)

    // 6000ms 后允许再拉（数据未变化 → 不回调，但拉取已发生）。
    clock = 16_000
    const later = await refresher.refresh()
    expect(later).toBe(false)
    expect(fetchSpy).toHaveBeenCalledTimes(2)
  })

  test('拉取失败静默：不回调、不抛出、后续可重试', async () => {
    const failFetch = vi.fn(async () => {
      throw new Error('network down')
    })
    const { deps, applySpy } = makeDeps({ fetchPayload: failFetch })
    const refresher = createSectionTimesRefresher(deps, { minIntervalMs: 0, now: () => 3000 })
    await expect(refresher.refresh()).resolves.toBe(false)
    expect(applySpy).not.toHaveBeenCalled()
    expect(failFetch).toHaveBeenCalledTimes(1)

    // 网络恢复后（时间前进）重试成功。
    failFetch.mockResolvedValueOnce({ props: { sectionTimes: T2 } })
    const retried = await refresher.refresh()
    expect(retried).toBe(true)
    expect(applySpy).toHaveBeenCalledTimes(1)
  })

  test('dispose 后不再拉取', async () => {
    const { deps, fetchSpy } = makeDeps()
    const refresher = createSectionTimesRefresher(deps, { minIntervalMs: 0, now: () => 4000 })
    refresher.dispose()
    const result = await refresher.refresh()
    expect(result).toBe(false)
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  test('payload 无 sectionTimes 时静默跳过（旧 payload 兼容）', async () => {
    const { deps, applySpy } = makeDeps({ fetchPayload: vi.fn(async () => ({ props: {} })) })
    const refresher = createSectionTimesRefresher(deps, { minIntervalMs: 0, now: () => 5000 })
    const result = await refresher.refresh()
    expect(result).toBe(false)
    expect(applySpy).not.toHaveBeenCalled()
  })

  test('extract 从 payload 提取作息表', () => {
    const { deps } = makeDeps()
    const refresher = createSectionTimesRefresher(deps)
    expect(refresher.extract({ props: { sectionTimes: T1 } })).toEqual(T1)
    expect(refresher.extract({ props: {} })).toBeUndefined()
    expect(refresher.extract(null)).toBeUndefined()
  })
})

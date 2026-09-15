import { afterEach, describe, expect, test, vi } from 'vitest'
import { maintenanceActive, maintenancePoller, type SearchMaintenanceJob } from '../src/admin/runtime/search-maintenance'

afterEach(() => vi.useRealTimers())
describe('search maintenance polling', () => {
  test('only pending/running/retrying tasks are active', () => {
    for (const status of [0, 1, 2, 3, 4] as const) {
      expect(maintenanceActive({ status } as SearchMaintenanceJob)).toBe([0, 1, 4].includes(status))
    }
  })
  test('does not overlap, poll hidden pages, or continue after completion', async () => {
    vi.useFakeTimers()
    let release!: (active: boolean) => void
    const load = vi.fn(() => new Promise<boolean>(resolve => { release = resolve }))
    let visible = true
    const poller = maintenancePoller(load, () => visible)
    const first = poller.refresh()
    await poller.refresh()
    await vi.advanceTimersByTimeAsync(10000)
    expect(load).toHaveBeenCalledTimes(1)
    release(true); await first
    visible = false
    await vi.advanceTimersByTimeAsync(10000)
    expect(load).toHaveBeenCalledTimes(1)
    visible = true
    const second = poller.refresh(); release(false); await second
    await vi.advanceTimersByTimeAsync(20000)
    expect(load).toHaveBeenCalledTimes(2)
    poller.stop()
  })
  test('unmount during a request never schedules another poll', async () => {
    vi.useFakeTimers()
    let release!: (active: boolean) => void
    const load = vi.fn(() => new Promise<boolean>(resolve => { release = resolve }))
    const poller = maintenancePoller(load, () => true)
    const request = poller.refresh(); poller.stop(); release(true); await request
    await vi.advanceTimersByTimeAsync(15000)
    await poller.refresh()
    expect(load).toHaveBeenCalledTimes(1)
  })
})

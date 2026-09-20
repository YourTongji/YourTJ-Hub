// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import type { StatusSnapshot } from '../src/types'
import StatusPage from '../src/StatusPage.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
import { getStatus } from '../src/runtime/status-api'

vi.mock('../src/runtime/status-api', () => ({ getStatus: vi.fn() }))
const fixture = JSON.parse(readFileSync('test/fixtures/status-connected.json', 'utf8')).result as StatusSnapshot
let page: VueWrapper | undefined
let hidden = false
const connected = () => { const data = structuredClone(fixture); data.uptime = { state: 'unconfigured', data: null }; return data }

beforeEach(async () => {
  await setLocale('zh')
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-09-14T12:00:00Z'))
  hidden = false
  vi.spyOn(document, 'hidden', 'get').mockImplementation(() => hidden)
  vi.mocked(getStatus).mockReset().mockResolvedValue(connected())
})
afterEach(() => { page?.unmount(); page = undefined; vi.restoreAllMocks(); vi.useRealTimers() })
async function open() { page = mount(StatusPage, { global: { plugins: [i18n] } }); await flushPromises(); return page }

it('renders live metrics and distinguishes a real zero from missing data', async () => {
  const data = connected(); data.traffic.data!.activeVisitors = 0
  vi.mocked(getStatus).mockResolvedValue(data)
  const wrapper = await open()
  expect(wrapper.get('#status-signal').text()).toBe('服务器探针正常')
  expect(wrapper.get('.status-signal-icon').classes()).toContain('is-ok')
  expect(wrapper.get('.status-signal-icon svg').classes()).toContain('lucide-circle-check')
  expect(wrapper.get('.status-footer').text()).toBe('© 2026 YourTJ Community | 公开统计数据')
  expect(wrapper.findAll('.status-source-chip')).toHaveLength(3)
  expect(wrapper.get('.status-source-chip').text()).toContain('Komari')
  expect(wrapper.get('.metric-active strong').text()).toBe('0')
  expect(wrapper.text()).toContain('8,060')
  expect(wrapper.findAll('.chart-bucket').length).toBeGreaterThan(20)
})
it('keeps missing or stale sources out of the live state', async () => {
  const data = connected(); data.server.state = 'stale'; data.traffic = { state: 'unconfigured', data: null }
  vi.mocked(getStatus).mockResolvedValue(data)
  const wrapper = await open()
  expect(wrapper.get('#status-signal').text()).toBe('暂无法确认状态')
  expect(wrapper.get('.metric-active strong').text()).toBe('—')
  expect(wrapper.text()).toContain('配置数据源后将在此显示实时数据')
  expect(wrapper.text()).toContain('数据已过期')
})
it('does not keep a healthy headline after a failed refresh', async () => {
  const wrapper = await open()
  vi.mocked(getStatus).mockRejectedValueOnce(new Error('offline'))
  await wrapper.get('.status-refresh').trigger('click'); await flushPromises()
  expect(wrapper.get('#status-signal').text()).toBe('暂无法确认状态')
  expect(wrapper.text()).toContain('8,060')
  expect(wrapper.get('[role="alert"]').text()).toContain('刷新失败')
})
it('marks an old probe sample as missing signal without claiming host downtime', async () => {
  const data = connected(); data.server.data!.current!.observedAt = '2026-09-14T11:55:00Z'
  vi.mocked(getStatus).mockResolvedValue(data)
  const wrapper = await open()
  expect(wrapper.get('#status-signal').text()).toBe('服务器探针暂无新数据')
})
it('rejects out-of-order responses after a period change', async () => {
  let resolveOld!: (value: StatusSnapshot) => void
  vi.mocked(getStatus).mockReturnValueOnce(new Promise(resolve => { resolveOld = resolve }))
  const wrapper = await open()
  const weekly = connected(); weekly.range = '7d'; weekly.traffic.data!.visitors = 7777
  vi.mocked(getStatus).mockResolvedValueOnce(weekly)
  await wrapper.findAll('.status-range button')[1]!.trigger('click'); await flushPromises()
  resolveOld(connected()); await flushPromises()
  expect(wrapper.text()).toContain('7,777')
  expect(wrapper.findAll('.status-range button')[1]!.attributes('aria-pressed')).toBe('true')
  expect(vi.mocked(getStatus).mock.calls[0]![1].aborted).toBe(true)
})
it('stops polling when hidden and unmounted, and refreshes on return', async () => {
  const wrapper = await open()
  hidden = true; document.dispatchEvent(new Event('visibilitychange'))
  await vi.advanceTimersByTimeAsync(60_000)
  expect(getStatus).toHaveBeenCalledTimes(1)
  hidden = false; document.dispatchEvent(new Event('visibilitychange')); await flushPromises()
  expect(getStatus).toHaveBeenCalledTimes(2)
  wrapper.unmount(); page = undefined
  await vi.advanceTimersByTimeAsync(60_000)
  expect(getStatus).toHaveBeenCalledTimes(2)
  expect(vi.mocked(getStatus).mock.calls[1]![1].aborted).toBe(true)
})
it('shows chart tooltips to keyboard users', async () => {
  const wrapper = await open()
  await wrapper.findAll('.chart-bucket').at(-1)!.trigger('focus')
  expect(wrapper.get('.chart-tooltip').text()).toContain('页面浏览')
  await wrapper.findAll('.chart-bucket').at(-1)!.trigger('blur')
  expect(wrapper.find('.chart-tooltip').exists()).toBe(false)
})

it('shows resource chart tooltips to keyboard users', async () => {
  const wrapper = await open()
  await wrapper.findAll('.resource-point-hit').at(-1)!.trigger('focus')
  expect(wrapper.get('.resource-tooltip').text()).toContain('CPU')
  await wrapper.findAll('.resource-point-hit').at(-1)!.trigger('blur')
  expect(wrapper.find('.resource-tooltip').exists()).toBe(false)
})

it('switches resource history independently and hides the previous scope while loading', async () => {
  const wrapper = await open()
  let resolveDay!: (value: StatusSnapshot) => void
  vi.mocked(getStatus).mockReturnValueOnce(new Promise(resolve => { resolveDay = resolve }))
  await wrapper.get('[aria-label="资源趋势时间范围"] button:nth-child(3)').trigger('click')
  expect(getStatus).toHaveBeenLastCalledWith('24h', expect.any(AbortSignal), '24h')
  expect(wrapper.find('.resource-history').exists()).toBe(false)
  expect(wrapper.text()).toContain('8,060')
  const daily = connected(); daily.serverRange = '24h'
  resolveDay(daily); await flushPromises()
  expect(wrapper.get('.resource-history svg').attributes('aria-label')).toContain('1 天')
  const weeklyTraffic = connected(); weeklyTraffic.range = '7d'; weeklyTraffic.serverRange = '24h'
  vi.mocked(getStatus).mockResolvedValueOnce(weeklyTraffic)
  await wrapper.get('.status-traffic .status-range button:nth-child(2)').trigger('click'); await flushPromises()
  expect(getStatus).toHaveBeenLastCalledWith('7d', expect.any(AbortSignal), '24h')
  expect(wrapper.get('[aria-label="资源趋势时间范围"] button:nth-child(3)').attributes('aria-pressed')).toBe('true')
})

it('ignores an older resource-scope response after a newer selection', async () => {
  const wrapper = await open()
  let resolveDay!: (value: StatusSnapshot) => void
  vi.mocked(getStatus).mockReturnValueOnce(new Promise(resolve => { resolveDay = resolve }))
  await wrapper.get('[aria-label="资源趋势时间范围"] button:nth-child(3)').trigger('click')
  await wrapper.get('[aria-label="资源趋势时间范围"] button:nth-child(1)').trigger('click'); await flushPromises()
  const daily = connected(); daily.serverRange = '24h'; daily.server.data!.current!.cpu = 99
  resolveDay(daily); await flushPromises()
  expect(wrapper.get('.resource-history svg').attributes('aria-label')).toContain('1 小时')
  expect(wrapper.get('.status-resource strong').text()).not.toBe('99.0%')
  expect(vi.mocked(getStatus).mock.calls[1]![1].aborted).toBe(true)
})

it('uses independent availability checks for the headline and does not fall back to a healthy probe on failure', async () => {
  const data = structuredClone(fixture)
  vi.mocked(getStatus).mockResolvedValue(data)
  const wrapper = await open()
  expect(wrapper.get('#status-signal').text()).toBe('所有公开服务正常')
  expect(wrapper.get('.status-signal-icon svg').classes()).toContain('lucide-circle-check')
  data.uptime.data!.monitors[0]!.current!.status = 'down'
  vi.mocked(getStatus).mockResolvedValue(structuredClone(data))
  await wrapper.get('.status-refresh').trigger('click'); await flushPromises()
  expect(wrapper.get('#status-signal').text()).toBe('服务异常')
  expect(wrapper.get('.status-signal-icon').classes()).toContain('is-error')
  expect(wrapper.get('.status-signal-icon svg').classes()).toContain('lucide-circle-x')
  const unavailable = connected(); unavailable.uptime = { state: 'unavailable', data: null }
  vi.mocked(getStatus).mockResolvedValue(unavailable)
  await wrapper.get('.status-refresh').trigger('click'); await flushPromises()
  expect(wrapper.get('#status-signal').text()).toBe('暂无法确认状态')
  expect(wrapper.get('.status-signal-icon').classes()).toContain('is-muted')
  expect(wrapper.get('.status-signal-icon svg').classes()).toContain('lucide-circle-help')
})

it.each(['sample', 'fetch'] as const)('rejects a fallback probe with a future %s timestamp', async field => {
  const data = connected()
  if (field === 'sample') data.server.data!.current!.observedAt = '2026-09-14T12:10:00Z'
  else data.server.fetchedAt = '2026-09-14T12:10:00Z'
  vi.mocked(getStatus).mockResolvedValue(data)
  const wrapper = await open()
  expect(wrapper.get('#status-signal').text()).not.toBe('服务器探针正常')
})

it.each([59_999, 60_000, 60_001])('bounds tolerated future probe skew at %i ms', async skew => {
  const data = connected()
  data.server.fetchedAt = data.server.data!.current!.observedAt = new Date(Date.now() + skew).toISOString()
  vi.mocked(getStatus).mockResolvedValue(data)
  const wrapper = await open()
  expect(wrapper.get('#status-signal').text() === '服务器探针正常').toBe(skew <= 60_000)
})

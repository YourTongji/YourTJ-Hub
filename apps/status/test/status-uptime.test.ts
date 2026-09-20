// @vitest-environment happy-dom
import { beforeEach, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import type { StatusSnapshot } from '../src/types'
import StatusUptime from '../src/components/StatusUptime.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
const fixture = JSON.parse(readFileSync('test/fixtures/status-connected.json', 'utf8')).result as StatusSnapshot
beforeEach(async () => { await setLocale('zh') })
const now = Date.parse('2026-09-14T12:00:01Z')

it('shows uptime, latency, real check history and an independent status link', async () => {
  const wrapper = mount(StatusUptime, { props: { source: structuredClone(fixture.uptime), now, failed: false, loading: false }, global: { plugins: [i18n] } })
  try {
    expect(wrapper.get('.uptime-percentage').text()).toBe('99.95%')
    expect(wrapper.get('.uptime-ping').text()).toBe('628 ms')
    expect(wrapper.get('.monitor-state').text()).toBe('可访问')
    expect(wrapper.findAll('.heartbeat-strip button')).toHaveLength(3)
    expect(wrapper.get('.uptime-external').attributes('href')).toBe('https://uptime.mortis.de5.net/status/a')
    expect(wrapper.get('.uptime-source').text()).toContain('数据来自 Uptime Kuma已连接')
    expect(wrapper.get('.uptime-source .source-badge').classes()).toContain('connected')
    expect(wrapper.get('.uptime-source time').text()).toContain('更新于')
    expect(wrapper.text()).toContain('可能不足 24 小时')
    await wrapper.get('.heartbeat-strip button').trigger('focus')
    expect(wrapper.get('[role="status"]').text()).toContain('访问异常')
    await wrapper.setProps({ now: now + 360_000 })
    expect(wrapper.get('.monitor-state').text()).toBe('状态未知')
    expect(wrapper.text()).toContain('数据已过期')
  } finally { wrapper.unmount() }
})

it('keeps the check detail visible across gaps between bars', async () => {
  const wrapper = mount(StatusUptime, { props: { source: structuredClone(fixture.uptime), now, failed: false, loading: false }, global: { plugins: [i18n] } })
  try {
    const bars = wrapper.findAll('.heartbeat-strip button')
    await bars[0]!.trigger('mouseenter')
    const detail = wrapper.get('[role="status"]').element
    await bars[0]!.trigger('mouseleave')
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).not.toBe('true')
    await bars[1]!.trigger('mouseenter')
    expect(wrapper.get('[role="status"]').element).toBe(detail)
    expect(wrapper.get('[role="status"]').text()).toBe(bars[1]!.attributes('aria-label'))
  } finally { wrapper.unmount() }
})

it('cancels dismissal on re-entry and hides after leaving the history', async () => {
  vi.useFakeTimers()
  const wrapper = mount(StatusUptime, { props: { source: structuredClone(fixture.uptime), now, failed: false, loading: false }, global: { plugins: [i18n] } })
  try {
    const history = wrapper.get('.heartbeat-history')
    const bars = wrapper.findAll('.heartbeat-strip button')
    await bars[0]!.trigger('mouseenter')
    await history.trigger('mouseleave')
    await vi.advanceTimersByTimeAsync(60)
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).not.toBe('true')
    await bars[1]!.trigger('mouseenter')
    await vi.advanceTimersByTimeAsync(200)
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).not.toBe('true')
    await history.trigger('mouseleave')
    await vi.advanceTimersByTimeAsync(200)
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).toBe('true')
    await bars[0]!.trigger('focus')
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).not.toBe('true')
    await history.trigger('focusout', { relatedTarget: bars[1]!.element })
    await bars[1]!.trigger('focus')
    await vi.advanceTimersByTimeAsync(200)
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).not.toBe('true')
    await history.trigger('focusout', { relatedTarget: null })
    await vi.advanceTimersByTimeAsync(200)
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).toBe('true')
    await bars[0]!.trigger('mouseenter')
    await history.trigger('mouseleave')
    wrapper.unmount()
    expect(vi.getTimerCount()).toBe(0)
  } finally { wrapper.unmount(); vi.useRealTimers() }
})

it('retains keyboard details when the pointer leaves and allows Escape to dismiss', async () => {
  vi.useFakeTimers()
  const wrapper = mount(StatusUptime, { attachTo: document.body, props: { source: structuredClone(fixture.uptime), now, failed: false, loading: false }, global: { plugins: [i18n] } })
  try {
    const bar = wrapper.get<HTMLButtonElement>('.heartbeat-strip button')
    bar.element.focus()
    await wrapper.vm.$nextTick()
    await wrapper.get('.heartbeat-history').trigger('mouseleave')
    await vi.advanceTimersByTimeAsync(200)
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).toBe('false')
    await bar.trigger('keydown', { key: 'Escape' })
    expect(wrapper.get('[role="status"]').attributes('aria-hidden')).toBe('true')
  } finally { wrapper.unmount(); vi.useRealTimers() }
})

it.each(['down', 'pending', 'maintenance', 'unknown'] as const)('preserves the %s state without showing healthy', async status => {
  const source = structuredClone(fixture.uptime)
  source.data!.monitors[0]!.current!.status = status
  const wrapper = mount(StatusUptime, { props: { source, now, failed: false, loading: false }, global: { plugins: [i18n] } })
  try { expect(wrapper.get('.monitor-state').classes()).toContain(`check-${status}`) } finally { wrapper.unmount() }
})

it('does not trust a future source fetch even when its latest check is recent', () => {
  const source = structuredClone(fixture.uptime)
  source.fetchedAt = new Date(now + 60_001).toISOString()
  const wrapper = mount(StatusUptime, { props: { source, now, failed: false, loading: false }, global: { plugins: [i18n] } })
  try {
    expect(wrapper.get('.monitor-state').classes()).toContain('check-unknown')
    expect(wrapper.text()).toContain('数据已过期')
  } finally { wrapper.unmount() }
})

it('distinguishes missing metrics from zero and stale checks from a freshly fetched response', async () => {
  const source = structuredClone(fixture.uptime)
  const monitor = source.data!.monitors[0]!
  monitor.uptime24h = 0; monitor.current!.ping = 0; monitor.current!.time = '2026-09-14T11:50:00Z'
  const wrapper = mount(StatusUptime, { props: { source, now, failed: false, loading: false }, global: { plugins: [i18n] } })
  try {
    expect(wrapper.get('.uptime-percentage').text()).toBe('0.00%')
    expect(wrapper.get('.uptime-ping').text()).toBe('0 ms')
    expect(wrapper.get('.monitor-state').text()).toBe('状态未知')
    const missing = structuredClone(source); missing.data!.monitors[0]!.uptime24h = null; missing.data!.monitors[0]!.current = null
    await wrapper.setProps({ source: missing })
    expect(wrapper.get('.uptime-percentage').text()).toBe('—')
    expect(wrapper.get('.uptime-ping').text()).toBe('—')
  } finally { wrapper.unmount() }
})

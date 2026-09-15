// @vitest-environment happy-dom
import { expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusResourceChart from '../src/components/StatusResourceChart.vue'
import { i18n } from '../src/runtime/i18n'

it('positions samples across the full daily span and connects downsampled intervals', async () => {
  const end = Date.parse('2026-09-14T12:00:00Z')
  const point = (minutesAgo: number) => ({ time: new Date(end - minutesAgo * 60_000).toISOString(), cpu: 20, memoryPercent: 50 })
  const wrapper = mount(StatusResourceChart, {
    props: { range: '24h', asOf: new Date(end).toISOString(), points: [point(1440), point(1428), point(720), point(0)] },
    global: { plugins: [i18n] },
  })
  try {
    expect(wrapper.findAll('circle').map(p => Number(p.attributes('cx')))).toEqual([0, 6, 360, 720])
    expect(wrapper.get('.cpu-line').attributes('d')).toBe('M0.00,130.00 L6.00,130.00 M360.00,130.00 M720.00,130.00')
    expect(wrapper.get('.resource-ticks').text()).toMatch(/13/)
    await wrapper.setProps({ range: '7d', points: [point(10080), point(5040), point(0)] })
    expect(wrapper.findAll('circle').map(p => Number(p.attributes('cx')))).toEqual([0, 360, 720])
  } finally { wrapper.unmount() }
})

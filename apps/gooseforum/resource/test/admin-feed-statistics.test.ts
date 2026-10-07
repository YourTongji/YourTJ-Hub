// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { expect, test, vi } from 'vitest'
import { createI18n } from 'vue-i18n'
import FeedStatistics from '../src/admin/pages/stats/FeedStatistics.vue'

vi.mock('../src/admin/runtime/api', () => ({
  getFeedSummary: vi.fn().mockResolvedValue({
    enabled: true,
    rankingReady: true,
    metricsEnabled: true,
    rolloutPercent: 20,
    rawRetentionDays: 30,
    paramsHash: 'hash',
    rows: [],
    periods: [],
    truncated: false,
    health: { accepted: 20, dropped: 0, queueLength: 0, backgroundFailures: 0 },
  }),
}))

test('collection health uses dashboard components instead of raw JSON', async () => {
  const page = mount(FeedStatistics, {
    global: {
      plugins: [
        createI18n({
          legacy: false,
          locale: 'en',
          missingWarn: false,
          fallbackWarn: false,
          messages: {},
        }),
      ],
    },
  })
  await flushPromises()
  expect(page.find('pre').exists()).toBe(false)
  expect(page.find('[data-testid="capture-status"]').exists()).toBe(true)
  page.unmount()
})

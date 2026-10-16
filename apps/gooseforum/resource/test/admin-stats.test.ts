// @vitest-environment happy-dom
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { createI18n } from 'vue-i18n'
import AdminSelect from '../src/admin/components/AdminSelect.vue'
import ManagementTable from '../src/admin/pages/management/ManagementTable.vue'
import StatsPage from '../src/admin/pages/StatsPage.vue'
import CategoryStatsSection from '../src/admin/pages/stats/CategoryStatsSection.vue'
import DateRangePicker from '../src/admin/pages/stats/DateRangePicker.vue'
import StatsFilterBar from '../src/admin/pages/stats/StatsFilterBar.vue'
import UserDetailPanel from '../src/admin/pages/stats/UserDetailPanel.vue'
import UserStatsSection from '../src/admin/pages/stats/UserStatsSection.vue'
import {
  getAdminStatsCategorySummary,
  getAdminStatsCategoryTrend,
  getAdminStatsUserDetail,
  getAdminStatsUserRanking,
  getGithubReleases,
  getServerVersion,
  getSiteStatistics,
  getTrafficOverview,
} from '../src/admin/runtime/api'
import type {
  AdminStatsCategorySummaryResult,
  AdminStatsCategoryTrendResult,
  AdminStatsFilter,
  AdminStatsUserDetailResult,
  AdminStatsUserRankingResult,
} from '../src/admin/runtime/api'
import type { AdminPayload, DailyTraffic, ManageHomeProps } from '../src/admin/types'

vi.mock('../src/admin/runtime/api', () => ({
  getAdminStatsCategorySummary: vi.fn(),
  getAdminStatsCategoryTrend: vi.fn(),
  getAdminStatsUserDetail: vi.fn(),
  getAdminStatsUserRanking: vi.fn(),
  getGithubReleases: vi.fn(),
  getServerVersion: vi.fn(),
  getSiteStatistics: vi.fn(),
  getTrafficOverview: vi.fn(),
}))

// 断言用的显式文案：locale 文件里不重复维护测试拷贝，只取需要识别的片段。
function messages() {
  return {
    en: {
      statsAdmin: {
        filterTitle: 'FILTER_BAR',
        range7: 'PRESET_7',
        range30: 'PRESET_30',
        rangeCustom: 'PRESET_CUSTOM',
        timezoneHint: 'TZ {zone}',
        rangeLimitHint: 'LIMIT {days}',
        categoryTitle: 'CATEGORY_SECTION',
        totalsRow: 'SITE_TOTAL',
        trendTitle: 'TREND_SECTION',
        trendEmpty: 'TREND_EMPTY',
        trendFor: 'TREND_OF {name}',
        estimatedBadge: 'ESTIMATED_BADGE',
        estimatedDays: 'ESTIMATED_DAYS',
        estimatedNone: 'NO_ESTIMATED_DAYS',
        userTitle: 'USER_SECTION',
        userEmpty: 'USER_EMPTY',
        searchRequired: 'SEARCH_REQUIRED',
        detailTitle: 'DETAIL_SECTION',
        detailEstimated: 'SOURCE_ESTIMATED',
        detailExact: 'SOURCE_EXACT',
        detailEmpty: 'DETAIL_EMPTY',
        categoryAll: 'ALL_CATEGORIES',
        invalidFormat: 'BAD_FORMAT',
        invalidOrder: 'START_AFTER_END',
        invalidSpan: 'SPAN_TOO_LONG',
        invalidKept: 'NOT_APPLIED',
      },
    },
  }
}

function globalOptions() {
  return {
    plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: messages() })],
    // happy-dom 下 unovis 的 tooltip 会在卸载时抛错（MutationObserver.disconnect），
    // 趋势图在上层断言中不需要真实绘制，直接桩掉图表容器。
    stubs: { ChartContainer: true },
  }
}

// 契约（AdminStatsCoverage.timezone）实际返回形如 `CST (UTC+08:00)`，样例与契约保持一致。
const coverage = { timezone: 'CST (UTC+08:00)', maxRangeDays: 366, estimated: false }
const baseFilter: AdminStatsFilter = { startDate: '2026-10-01', endDate: '2026-10-07', categoryId: 0 }

function categorySummary(estimated = true): AdminStatsCategorySummaryResult {
  return {
    categories: [
      { categoryId: 3, name: 'Campus', topicCount: 5, replyCount: 9, contributorCount: 4, agentContributorCount: 1 },
      { categoryId: 7, name: 'Courses', topicCount: 2, replyCount: 0, contributorCount: 2, agentContributorCount: 0 },
    ],
    totals: { topicCount: 6, replyCount: 9, contributorCount: 5, agentContributorCount: 1 },
    coverage: { ...coverage, estimated },
  }
}

function categoryTrend(categoryId: number, estimatedDays: number): AdminStatsCategoryTrendResult {
  return {
    categoryId,
    days: [
      { date: '2026-10-01', topicCount: 1, replyCount: 2, estimated: false },
      { date: '2026-10-02', topicCount: 0, replyCount: 0, estimated: estimatedDays > 0 },
    ],
    coverage,
  }
}

function userRanking(): AdminStatsUserRankingResult {
  return {
    // 按回复数排序：alice 9 条回复排在前，尽管 bob 的主题数更多（前端不得重排）。
    items: [
      { userId: 11, username: 'alice', nickname: 'Alice', actorType: 0, topicCount: 1, replyCount: 9, likeCount: 3 },
      { userId: 12, username: 'bob', nickname: 'Bob', actorType: 0, topicCount: 5, replyCount: 1, likeCount: 8 },
    ],
    page: 1,
    pageSize: 20,
    total: 45,
    coverage,
  }
}

// 页面级用例用：板块清单按名字派生，便于断言“哪一次响应胜出”。
function pageSummary(names: string[], estimated = false): AdminStatsCategorySummaryResult {
  return {
    categories: names.map((name, index) => ({
      categoryId: index + 1,
      name,
      topicCount: index + 1,
      replyCount: 0,
      contributorCount: 1,
      agentContributorCount: 0,
    })),
    totals: { topicCount: names.length, replyCount: 0, contributorCount: 1, agentContributorCount: 0 },
    coverage: { ...coverage, estimated },
  }
}

function traffic(regCount: number): DailyTraffic {
  return { date: '2026-10-01', regCount, topicCount: 1, replyCount: 2, courseReviewCount: 0 }
}

function userDetail(): AdminStatsUserDetailResult {
  return {
    user: { userId: 11, username: 'alice', nickname: 'Alice', actorType: 0 },
    summary: { topicCount: 1, replyCount: 2, likeCount: 3 },
    days: [
      { date: '2026-10-01', topicCount: 1, replyCount: 0, estimated: false },
      { date: '2026-10-02', topicCount: 0, replyCount: 0, estimated: false },
      { date: '2026-10-03', topicCount: 0, replyCount: 2, estimated: true },
    ],
    coverage: { ...coverage, estimated: true },
  }
}

let wrapper: VueWrapper | undefined
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.resetAllMocks()
})

describe('stats filter bar', () => {
  function mountBar(invalidMessage?: string) {
    return mount(StatsFilterBar, {
      props: {
        presets: [
          { key: '7', label: 'PRESET_7', startDate: '2026-10-07', endDate: '2026-10-13' },
          { key: '30', label: 'PRESET_30', startDate: '2026-09-14', endDate: '2026-10-13' },
        ],
        activePreset: '7',
        startDate: '2026-10-07',
        endDate: '2026-10-13',
        categoryId: 0,
        categoryOptions: [{ value: 0, label: 'All categories' }, { value: 3, label: 'Campus' }],
        timezone: coverage.timezone,
        maxRangeDays: 366,
        invalidMessage,
      },
      global: globalOptions(),
    })
  }

  test('preset buttons emit the range preset and expose the timezone/limit hints', async () => {
    wrapper = mountBar()
    expect(wrapper.text()).toContain('FILTER_BAR')
    expect(wrapper.text()).toContain('CST (UTC+08:00)')
    expect(wrapper.text()).toContain('366')

    const preset30 = wrapper.findAll('button').find(button => button.text() === 'PRESET_30')
    await preset30!.trigger('click')
    expect(wrapper.emitted('selectPreset')).toEqual([['30']])
  })

  test('shows the explicit invalid-range message instead of silently filtering', () => {
    wrapper = mountBar({ invalidMessage: 'START_AFTER_END' })
    const box = wrapper.get('[data-testid="stats-filter-invalid"]')
    expect(box.text()).toContain('START_AFTER_END')
    expect(box.text()).toContain('NOT_APPLIED')
  })
})

describe('category statistics', () => {
  function mountSection(filter: AdminStatsFilter, summary?: AdminStatsCategorySummaryResult) {
    return mount(CategoryStatsSection, {
      props: { filter, summary, summaryLoading: false, summaryError: '' },
      global: globalOptions(),
    })
  }

  test('renders every category, the de-duplicated site total and the estimated badge', () => {
    wrapper = mountSection(baseFilter, categorySummary())
    const text = wrapper.text()
    expect(text).toContain('CATEGORY_SECTION')
    expect(text).toContain('Campus')
    expect(text).toContain('Courses')
    expect(text).toContain('ESTIMATED_BADGE')

    const totals = wrapper.get('[data-testid="stats-category-totals"]')
    expect(totals.text()).toContain('SITE_TOTAL')
    expect(totals.text()).toContain('9')
  })

  test('selecting a category requests its trend with the current filter and marks estimated days', async () => {
    vi.mocked(getAdminStatsCategoryTrend).mockResolvedValue(categoryTrend(3, 1))
    wrapper = mountSection(baseFilter, categorySummary(false))
    await flushPromises()
    expect(getAdminStatsCategoryTrend).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('TREND_EMPTY')
    expect(wrapper.text()).not.toContain('ESTIMATED_BADGE')

    const courses = wrapper.findAll('tbody tr').find(row => row.text().includes('Courses'))
    await courses!.trigger('click')
    expect(wrapper.emitted('select-category')).toEqual([[7]])

    await wrapper.setProps({ filter: { ...baseFilter, categoryId: 3 } })
    await flushPromises()
    expect(getAdminStatsCategoryTrend).toHaveBeenCalledWith({
      startDate: '2026-10-01',
      endDate: '2026-10-07',
      categoryId: 3,
    })
    const estimated = wrapper.get('[data-testid="stats-trend-estimated"]')
    expect(estimated.text()).toContain('ESTIMATED_DAYS')
    expect(estimated.text()).not.toContain('NO_ESTIMATED_DAYS')
  })

  test('a slow response for a previous category cannot overwrite the newer filter', async () => {
    let resolveSlow!: (value: AdminStatsCategoryTrendResult) => void
    vi.mocked(getAdminStatsCategoryTrend)
      .mockReturnValueOnce(new Promise(resolve => { resolveSlow = resolve }))
      .mockResolvedValueOnce(categoryTrend(7, 1))

    wrapper = mountSection({ ...baseFilter, categoryId: 3 }, categorySummary(false))
    await flushPromises()
    await wrapper.setProps({ filter: { ...baseFilter, categoryId: 7 } })
    await flushPromises()
    expect(wrapper.get('[data-testid="stats-trend-estimated"]').text()).toContain('ESTIMATED_DAYS')

    resolveSlow(categoryTrend(3, 0))
    await flushPromises()
    expect(wrapper.text()).not.toContain('NO_ESTIMATED_DAYS')
  })
})

describe('user statistics', () => {
  function mountUserSection(filter: AdminStatsFilter = baseFilter) {
    return mount(UserStatsSection, { props: { filter }, global: globalOptions() })
  }

  test('keeps the backend ordering instead of re-sorting rows on the client', async () => {
    vi.mocked(getAdminStatsUserRanking).mockResolvedValue(userRanking())
    wrapper = mountUserSection()
    await flushPromises()

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('Alice')
    expect(rows[1].text()).toContain('Bob')
    expect(getAdminStatsUserRanking).toHaveBeenCalledWith({
      startDate: '2026-10-01',
      endDate: '2026-10-07',
      categoryId: 0,
      sortBy: 'topics',
      page: 1,
      pageSize: 20,
      actorType: 0,
    })
  })

  test('sorting, paging and account type re-query with the same filter', async () => {
    vi.mocked(getAdminStatsUserRanking).mockResolvedValue(userRanking())
    wrapper = mountUserSection()
    await flushPromises()

    const selects = wrapper.findAllComponents(AdminSelect)
    await selects[1].vm.$emit('update:modelValue', 'likes')
    await flushPromises()
    expect(getAdminStatsUserRanking).toHaveBeenLastCalledWith(expect.objectContaining({ sortBy: 'likes', page: 1 }))

    await selects[0].vm.$emit('update:modelValue', 1)
    await flushPromises()
    expect(getAdminStatsUserRanking).toHaveBeenLastCalledWith(expect.objectContaining({ actorType: 1, page: 1 }))

    wrapper.findComponent(ManagementTable).vm.$emit('update:page', 2)
    await flushPromises()
    expect(getAdminStatsUserRanking).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, actorType: 1 }))
  })

  test('a username or numeric id search opens the detail panel, an empty search does not', async () => {
    vi.mocked(getAdminStatsUserRanking).mockResolvedValue(userRanking())
    vi.mocked(getAdminStatsUserDetail).mockResolvedValue(userDetail())
    wrapper = mountUserSection()
    await flushPromises()

    const input = wrapper.get('form input')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.get('[data-testid="stats-search-error"]').text()).toContain('SEARCH_REQUIRED')
    expect(getAdminStatsUserDetail).not.toHaveBeenCalled()

    await input.setValue('alice')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(getAdminStatsUserDetail).toHaveBeenLastCalledWith({
      startDate: '2026-10-01',
      endDate: '2026-10-07',
      categoryId: 0,
      username: 'alice',
    })

    await input.setValue('42')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(getAdminStatsUserDetail).toHaveBeenLastCalledWith({
      startDate: '2026-10-01',
      endDate: '2026-10-07',
      categoryId: 0,
      userId: 42,
    })
    expect(wrapper.text()).toContain('DETAIL_SECTION')
  })

  test('a numeric search missing as an id retries it as a username', async () => {
    // username 允许 6-32 位字母数字下划线（component.ValidateUsername），因此「123456」既可能是
    // userId 也可能是用户名：ID 查询未命中时按用户名重试，两次都不中才展示错误。
    const notFound = Object.assign(new Error('用户不存在'), { messageCode: 'user.notFound' })
    vi.mocked(getAdminStatsUserRanking).mockResolvedValue(userRanking())
    vi.mocked(getAdminStatsUserDetail)
      .mockRejectedValueOnce(notFound)
      .mockResolvedValueOnce(userDetail())
    wrapper = mountUserSection()
    await flushPromises()

    const input = wrapper.get('form input')
    await input.setValue('123456')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(getAdminStatsUserDetail).toHaveBeenNthCalledWith(1, expect.objectContaining({ userId: 123456 }))
    expect(getAdminStatsUserDetail).toHaveBeenNthCalledWith(2, expect.objectContaining({ username: '123456' }))
    expect(wrapper.findComponent(UserDetailPanel).text()).not.toContain('用户不存在')

    // 用户名回退也只尝试一次：再未命中时停在错误态，不再循环重试。
    vi.mocked(getAdminStatsUserDetail)
      .mockReset()
      .mockRejectedValue(notFound)
    await input.setValue('654321')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(getAdminStatsUserDetail).toHaveBeenCalledTimes(2)
    expect(wrapper.findComponent(UserDetailPanel).text()).toContain('用户不存在')
  })

  test('a slow detail response for a previous user cannot overwrite the newer one', async () => {
    vi.mocked(getAdminStatsUserRanking).mockResolvedValue(userRanking())
    let resolveSlow!: (value: AdminStatsUserDetailResult) => void
    vi.mocked(getAdminStatsUserDetail)
      .mockReturnValueOnce(new Promise(resolve => { resolveSlow = resolve }))
      .mockResolvedValueOnce({ ...userDetail(), summary: { topicCount: 0, replyCount: 0, likeCount: 0 }, days: [] })

    wrapper = mountUserSection()
    await flushPromises()

    const input = wrapper.get('form input')
    await input.setValue('alice')
    await wrapper.get('form').trigger('submit')
    await input.setValue('bob')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    const bobPanel = wrapper.findComponent(UserDetailPanel)
    expect(bobPanel.text()).toContain('DETAIL_EMPTY')

    resolveSlow(userDetail())
    await flushPromises()
    expect(wrapper.findComponent(UserDetailPanel).text()).toContain('DETAIL_EMPTY')
  })
})

describe('user detail panel', () => {
  test('shows per-day values with explicit zeros and marks estimated days', async () => {
    vi.mocked(getAdminStatsUserDetail).mockResolvedValue(userDetail())
    wrapper = mount(UserDetailPanel, {
      props: { filter: baseFilter, target: { userId: 11 } },
      global: globalOptions(),
    })
    await flushPromises()

    expect(getAdminStatsUserDetail).toHaveBeenCalledWith({
      startDate: '2026-10-01',
      endDate: '2026-10-07',
      categoryId: 0,
      userId: 11,
    })
    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(3)
    expect(rows[1].text()).toContain('0')
    expect(rows[1].attributes('data-estimated')).toBe('false')
    expect(rows[2].attributes('data-estimated')).toBe('true')
    expect(rows[2].text()).toContain('SOURCE_ESTIMATED')
    expect(rows[1].text()).toContain('SOURCE_EXACT')
  })

  test('reports a failed query instead of showing imaginary zeros', async () => {
    vi.mocked(getAdminStatsUserDetail).mockRejectedValue(new Error('admin.stats.fetchFailed'))
    wrapper = mount(UserDetailPanel, {
      props: { filter: baseFilter, target: { username: 'ghost' } },
      global: globalOptions(),
    })
    await flushPromises()
    expect(wrapper.text()).toContain('admin.stats.fetchFailed')
    expect(wrapper.findAll('tbody tr')).toHaveLength(0)
  })
})

describe('stats page filter dispatch', () => {
  // 页面级用例：区间校验与 activeFilter 单点分发（AC-1/AC-8 的直接实现点）。
  const TODAY = new Date('2026-10-13T12:00:00')
  const payload = {
    component: 'manage-home',
    props: {},
    meta: { title: 'admin' },
    layout: {},
    url: '/admin',
    version: 'test',
  } as unknown as AdminPayload<ManageHomeProps>
  const trafficStub = {
    props: ['data', 'loading'],
    template: '<div data-testid="traffic-stub">{{ JSON.stringify(data) }}</div>',
  }

  beforeEach(() => {
    // 只伪 Date：区间预设依赖“今天”，其余计时器保持真实以便 flushPromises 正常推进。
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(TODAY)
    vi.mocked(getSiteStatistics).mockResolvedValue({
      userCount: 1,
      userMonthCount: 0,
      topicMaxId: 2,
      topicMonthCount: 0,
      postMaxId: 3,
      linksCount: 0,
    })
    vi.mocked(getServerVersion).mockResolvedValue({ version: 'test', commit: 'abc', buildDate: '2026-10-13', mode: 'development' })
    vi.mocked(getGithubReleases).mockResolvedValue([])
    vi.mocked(getTrafficOverview).mockResolvedValue([traffic(1)])
    vi.mocked(getAdminStatsCategorySummary).mockResolvedValue(pageSummary(['Campus']))
    vi.mocked(getAdminStatsUserRanking).mockResolvedValue(userRanking())
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function mountPage() {
    return mount(StatsPage, {
      props: { payload },
      global: {
        ...globalOptions(),
        stubs: { TrafficOverview: trafficStub, ChartContainer: true },
      },
    })
  }

  async function clickButton(text: string) {
    const button = wrapper!.findAll('button').find(item => item.text() === text)
    expect(button, `button ${text}`).toBeTruthy()
    await button!.trigger('click')
  }

  function categorySelect() {
    const bar = wrapper!.findComponent(StatsFilterBar)
    return bar.findAllComponents(AdminSelect)[0]
  }

  test('an invalid range is reported and dispatches no request', async () => {
    wrapper = mountPage()
    await flushPromises()
    expect(getAdminStatsCategorySummary).toHaveBeenCalledTimes(1)
    expect(getTrafficOverview).toHaveBeenCalledTimes(1)
    expect(wrapper!.find('[data-testid="stats-filter-invalid"]').exists()).toBe(false)

    await clickButton('PRESET_CUSTOM')
    const picker = () => wrapper!.findComponent(DateRangePicker)
    expect(picker().exists()).toBe(true)

    // 起 > 止
    await picker().vm.$emit('update:startDate', '2026-10-20')
    await flushPromises()
    expect(wrapper!.get('[data-testid="stats-filter-invalid"]').text()).toContain('START_AFTER_END')

    // 跨度 > 366 天
    await picker().vm.$emit('update:endDate', '2027-11-30')
    await flushPromises()
    expect(wrapper!.get('[data-testid="stats-filter-invalid"]').text()).toContain('SPAN_TOO_LONG')

    // 格式非法
    await picker().vm.$emit('update:startDate', '2026-1-5')
    await flushPromises()
    expect(wrapper!.get('[data-testid="stats-filter-invalid"]').text()).toContain('BAD_FORMAT')

    // 三次非法输入都不得派发请求，且保留上一次有效结果
    expect(getAdminStatsCategorySummary).toHaveBeenCalledTimes(1)
    expect(getTrafficOverview).toHaveBeenCalledTimes(1)
    expect(wrapper!.text()).toContain('Campus')

    // 恢复合法区间后重新派发，提示消失
    await picker().vm.$emit('update:startDate', '2026-10-01')
    await picker().vm.$emit('update:endDate', '2026-10-05')
    await flushPromises()
    expect(wrapper!.find('[data-testid="stats-filter-invalid"]').exists()).toBe(false)
    expect(getAdminStatsCategorySummary).toHaveBeenCalledTimes(2)
    expect(getTrafficOverview).toHaveBeenCalledTimes(2)
    expect(getTrafficOverview).toHaveBeenLastCalledWith('2026-10-01', '2026-10-05')
  })

  test('range presets re-query both time-series endpoints, a category change does not touch traffic', async () => {
    vi.mocked(getAdminStatsCategoryTrend).mockResolvedValue(categoryTrend(3, 0))
    wrapper = mountPage()
    await flushPromises()
    expect(getTrafficOverview).toHaveBeenLastCalledWith('2026-10-07', '2026-10-13')

    await clickButton('PRESET_30')
    await flushPromises()
    expect(getTrafficOverview).toHaveBeenCalledTimes(2)
    expect(getTrafficOverview).toHaveBeenLastCalledWith('2026-09-14', '2026-10-13')
    expect(getAdminStatsCategorySummary).toHaveBeenCalledTimes(2)
    expect(getAdminStatsCategorySummary).toHaveBeenLastCalledWith({ startDate: '2026-09-14', endDate: '2026-10-13' })
    expect(getAdminStatsUserRanking).toHaveBeenLastCalledWith(
      expect.objectContaining({ startDate: '2026-09-14', endDate: '2026-10-13', categoryId: 0 }),
    )
    expect(wrapper.text()).toContain('2026-09-14 ~ 2026-10-13')

    // 重复选中同一预设不产生多余请求（含下方区块的排行/趋势取数）
    await clickButton('PRESET_30')
    await flushPromises()
    expect(getAdminStatsCategorySummary).toHaveBeenCalledTimes(2)
    expect(getTrafficOverview).toHaveBeenCalledTimes(2)
    expect(getAdminStatsUserRanking).toHaveBeenCalledTimes(2)

    // 板块筛选只作用于板块/用户维度
    await categorySelect().vm.$emit('update:modelValue', 3)
    await flushPromises()
    expect(getTrafficOverview).toHaveBeenCalledTimes(2)
    expect(getAdminStatsCategorySummary).toHaveBeenCalledTimes(2)
    expect(getAdminStatsUserRanking).toHaveBeenLastCalledWith(
      expect.objectContaining({ startDate: '2026-09-14', endDate: '2026-10-13', categoryId: 3 }),
    )
    expect(getAdminStatsCategoryTrend).toHaveBeenLastCalledWith({ startDate: '2026-09-14', endDate: '2026-10-13', categoryId: 3 })
  })

  test('clearing the category selection restores the whole-site view', async () => {
    vi.mocked(getAdminStatsCategoryTrend).mockResolvedValue(categoryTrend(3, 1))
    wrapper = mountPage()
    await flushPromises()
    expect(wrapper.text()).toContain('TREND_EMPTY')

    await categorySelect().vm.$emit('update:modelValue', 3)
    await flushPromises()
    expect(getAdminStatsCategoryTrend).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('ESTIMATED_DAYS')

    await categorySelect().vm.$emit('update:modelValue', 0)
    await flushPromises()
    expect(getAdminStatsCategoryTrend).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('TREND_EMPTY')
    expect(getAdminStatsUserRanking).toHaveBeenLastCalledWith(expect.objectContaining({ categoryId: 0 }))
  })

  test('a slow summary response cannot overwrite the newer filter result', async () => {
    let resolveSlow!: (value: AdminStatsCategorySummaryResult) => void
    vi.mocked(getAdminStatsCategorySummary)
      .mockReturnValueOnce(new Promise(resolve => { resolveSlow = resolve }))
      .mockResolvedValueOnce(pageSummary(['Fresh']))

    wrapper = mountPage()
    await flushPromises()
    await clickButton('PRESET_30')
    await flushPromises()
    expect(wrapper.text()).toContain('Fresh')

    resolveSlow(pageSummary(['Stale']))
    await flushPromises()
    expect(wrapper.text()).toContain('Fresh')
    expect(wrapper.text()).not.toContain('Stale')
  })

  test('a slow traffic response cannot overwrite the newer range result', async () => {
    let resolveSlow!: (value: DailyTraffic[]) => void
    vi.mocked(getTrafficOverview)
      .mockReturnValueOnce(new Promise(resolve => { resolveSlow = resolve }))
      .mockResolvedValueOnce([traffic(99)])

    wrapper = mountPage()
    await flushPromises()
    await clickButton('PRESET_30')
    await flushPromises()
    const trafficText = () => wrapper!.get('[data-testid="traffic-stub"]').text()
    expect(trafficText()).toContain('"regCount":99')

    resolveSlow([traffic(1)])
    await flushPromises()
    expect(trafficText()).toContain('"regCount":99')
    expect(trafficText()).not.toContain('"regCount":1,')
  })
})

// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import CourseManagementPage from '../src/site/pages/CourseManagementPage.vue'
import {
  approveCourseRelation,
  fetchAdminCourses,
  fetchCourseRelations,
  ignoreCourseRelation,
  resetCourseRelation,
  type AdminCourseItem,
  type CourseRelationItem,
} from '../src/runtime/api'
import type { CourseManagementPageProps, LayoutPayload } from '@gooseforum/client'

// 课程管理页「课程沿革」审核面板（PR #361 review）：mock 全部管理 API，
// 直接挂载页面切到 relations tab，验证 pending 候选渲染与批准/忽略交互。
vi.mock('../src/runtime/api', () => ({
  approveCourseRelation: vi.fn(),
  createAdminCourse: vi.fn(),
  createCourseRelation: vi.fn(),
  deleteAdminCourse: vi.fn(),
  deleteAdminReview: vi.fn(),
  fetchAdminCourses: vi.fn(),
  fetchAdminReviews: vi.fn(),
  fetchCourseRelations: vi.fn(),
  getCourseDetail: vi.fn(),
  ignoreCourseRelation: vi.fn(),
  mergeCourseRelation: vi.fn(),
  moderationCourseReviewStatus: vi.fn(),
  rebuildCourseStats: vi.fn(),
  resetCourseRelation: vi.fn(),
  undoMergeCourseRelation: vi.fn(),
  updateAdminCourse: vi.fn(),
  updateAdminReview: vi.fn(),
}))

function course(id: number, name: string): AdminCourseItem {
  return {
    id,
    primaryCode: `CS${id}`,
    name,
    department: '计算机科学与技术学院',
    creditX10: 30,
    status: 0,
    aliases: [],
    instructors: [],
    reviewCount: 0,
    createdAt: '2026-01-01 00:00:00',
  }
}

function relation(overrides: Partial<CourseRelationItem> & { id: number }): CourseRelationItem {
  return {
    id: overrides.id,
    fromCourseId: 101,
    toCourseId: 102,
    relationType: 'SPLIT_FROM',
    source: 'rule',
    confidence: 0.85,
    evidenceJson: JSON.stringify({ reason: '拆分候选' }),
    manual: false,
    status: 'pending',
    createdAt: '2026-01-01 00:00:00',
    updatedAt: '2026-01-01 00:00:00',
    fromCourse: { id: 101, primaryCode: 'CS101', name: '数据结构', department: '计算机科学与技术学院', creditX10: 30, status: 0 },
    toCourse: { id: 102, primaryCode: 'CS102', name: '算法设计', department: '计算机科学与技术学院', creditX10: 30, status: 0 },
    ...overrides,
  }
}

const course101 = course(101, '数据结构')
const course102 = course(102, '算法设计')

// 模拟服务端列表：approve/ignore 成功后从列表移除对应候选，验证刷新后的渲染。
let relationList: CourseRelationItem[] = []

function mountPage() {
  return mount(CourseManagementPage, {
    props: {
      layout: {} as LayoutPayload,
      props: {} as CourseManagementPageProps,
    },
    global: { plugins: [i18n] },
  })
}

async function clickButton(wrapper: VueWrapper, text: string) {
  const button = wrapper.findAll('button').find((b) => b.text().trim() === text)
  expect(button, `按钮「${text}」应存在`).toBeTruthy()
  await button!.trigger('click')
}

async function openRelationsTab(wrapper: VueWrapper) {
  await clickButton(wrapper, '课程沿革')
  await flushPromises()
}

describe('课程管理页「课程沿革」审核面板', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh'
    // 每个测试独立统计调用次数（mock 实现随后在下方重建，clear 不清实现）。
    vi.clearAllMocks()
    relationList = []
    vi.mocked(fetchAdminCourses).mockResolvedValue({
      list: [course101, course102],
      page: 1,
      size: 20,
      total: 2,
      hasNext: false,
    })
    vi.mocked(fetchCourseRelations).mockImplementation(async () => ({
      list: [...relationList],
      page: 1,
      size: 20,
      total: relationList.length,
      hasNext: false,
    }))
    vi.mocked(approveCourseRelation).mockImplementation(async (relationId: number) => {
      relationList = []
      return relation({ id: relationId })
    })
    vi.mocked(ignoreCourseRelation).mockImplementation(async (relationId: number) => {
      relationList = []
      return relation({ id: relationId })
    })
    vi.mocked(resetCourseRelation).mockImplementation(async (relationId: number) => {
      relationList = relationList.filter((r) => r.id !== relationId)
      return relation({ id: relationId })
    })
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  test('切到沿革 tab 按 pending 拉取并渲染候选行（类型/来源/置信度/状态/按钮）', async () => {
    relationList = [
      relation({ id: 1, relationType: 'EQUIVALENT' }),
      relation({ id: 2, relationType: 'SPLIT_FROM' }),
    ]
    const wrapper = mountPage()
    await openRelationsTab(wrapper)

    expect(fetchCourseRelations).toHaveBeenCalledWith('pending', '', 1, 20)
    const text = wrapper.text()
    // 旧卡/新卡名称（来自列表项附带的 from/to 课程摘要）。
    expect(text).toContain('数据结构')
    expect(text).toContain('算法设计')
    // 类型、来源、置信度、状态、证据入口。
    expect(text).toContain('等价')
    expect(text).toContain('拆分')
    expect(text).toContain('规则')
    expect(text).toContain('85%')
    expect(text).toContain('待审核')
    expect(wrapper.findAll('summary').filter((s) => s.text() === '查看证据')).toHaveLength(2)
    // 可合并类型(EQUIVALENT)给「确认等价合并」；不可合并类型(SPLIT_FROM)给「批准」；pending 都给「忽略」。
    const buttons = wrapper.findAll('button').map((b) => b.text().trim())
    expect(buttons.filter((t) => t === '确认等价合并')).toHaveLength(1)
    expect(buttons.filter((t) => t === '批准')).toHaveLength(1)
    expect(buttons.filter((t) => t === '忽略')).toHaveLength(2)

    wrapper.unmount()
  })

  test('点击批准调用 approve API，刷新后 pending 行消失并显示空态', async () => {
    relationList = [relation({ id: 2, relationType: 'SPLIT_FROM' })]
    const wrapper = mountPage()
    await openRelationsTab(wrapper)
    expect(wrapper.text()).toContain('85%')

    await clickButton(wrapper, '批准')
    await flushPromises()

    expect(approveCourseRelation).toHaveBeenCalledWith(2)
    expect(fetchCourseRelations).toHaveBeenLastCalledWith('pending', '', 1, 20)
    expect(wrapper.text()).not.toContain('85%')
    expect(wrapper.text()).toContain('暂无沿革候选')

    wrapper.unmount()
  })

  test('点击忽略调用 ignore API，刷新后 pending 行消失并显示空态', async () => {
    relationList = [relation({ id: 3, relationType: 'RELATED' })]
    const wrapper = mountPage()
    await openRelationsTab(wrapper)

    await clickButton(wrapper, '忽略')
    await flushPromises()
    expect(ignoreCourseRelation).toHaveBeenCalledWith(3)
    expect(wrapper.text()).not.toContain('85%')
    expect(wrapper.text()).toContain('暂无沿革候选')

    wrapper.unmount()
  })

  test('批准失败：保留候选行并把错误渲染到页头', async () => {
    relationList = [relation({ id: 2, relationType: 'SPLIT_FROM' })]
    vi.mocked(approveCourseRelation).mockRejectedValue(new Error('模拟审批失败'))
    const wrapper = mountPage()
    await openRelationsTab(wrapper)

    await clickButton(wrapper, '批准')
    await flushPromises()

    expect(wrapper.text()).toContain('模拟审批失败')
    expect(wrapper.text()).toContain('85%')
    // 失败不触发列表重拉。
    expect(fetchCourseRelations).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  test('审批进行中行内按钮禁用，完成后按最新列表刷新', async () => {
    relationList = [relation({ id: 2, relationType: 'SPLIT_FROM' })]
    let resolveApprove!: (value: CourseRelationItem) => void
    vi.mocked(approveCourseRelation).mockImplementation(
      () => new Promise<CourseRelationItem>((resolve) => (resolveApprove = resolve)),
    )
    const wrapper = mountPage()
    await openRelationsTab(wrapper)

    await clickButton(wrapper, '批准')
    await flushPromises()

    const approveButton = () => wrapper.findAll('button').find((b) => b.text().trim() === '批准')!
    const ignoreButton = () => wrapper.findAll('button').find((b) => b.text().trim() === '忽略')!
    expect(approveButton().attributes('disabled')).toBeDefined()
    expect(ignoreButton().attributes('disabled')).toBeDefined()

    relationList = []
    resolveApprove(relation({ id: 2, relationType: 'SPLIT_FROM' }))
    await flushPromises()

    expect(fetchCourseRelations).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('暂无沿革候选')

    wrapper.unmount()
  })

  test('沿革状态 tab 切换按对应状态重新拉取', async () => {
    relationList = [relation({ id: 1, relationType: 'EQUIVALENT' })]
    const wrapper = mountPage()
    await openRelationsTab(wrapper)
    expect(fetchCourseRelations).toHaveBeenLastCalledWith('pending', '', 1, 20)

    await clickButton(wrapper, '已批准')
    await flushPromises()

    expect(fetchCourseRelations).toHaveBeenLastCalledWith('approved', '', 1, 20)

    wrapper.unmount()
  })

  test('切换类型筛选按对应类型重新拉取', async () => {
    relationList = [relation({ id: 2, relationType: 'SPLIT_FROM' })]
    const wrapper = mountPage()
    await openRelationsTab(wrapper)
    expect(fetchCourseRelations).toHaveBeenLastCalledWith('pending', '', 1, 20)

    const select = wrapper.find('select')
    await select.setValue('SPLIT_FROM')
    await flushPromises()

    expect(fetchCourseRelations).toHaveBeenLastCalledWith('pending', 'SPLIT_FROM', 1, 20)

    wrapper.unmount()
  })

  test('撤回已批准候选：调用 reset API 刷新列表', async () => {
    relationList = [relation({ id: 7, status: 'approved' })]
    const wrapper = mountPage()
    await openRelationsTab(wrapper)

    await clickButton(wrapper, '撤回')
    await flushPromises()

    expect(resetCourseRelation).toHaveBeenCalledWith(7)
    expect(wrapper.text()).toContain('暂无沿革候选')

    wrapper.unmount()
  })
})

// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

const getPkCourseReviewBrief = vi.fn()
vi.mock('../src/runtime/pk-api', () => ({
  getPkCourseReviewBrief: (input: { courseCode: string; teacherName: string }) =>
    getPkCourseReviewBrief(input),
}))

const listCourseReviews = vi.fn()
const getCourseSummary = vi.fn()
vi.mock('../src/runtime/api', () => ({
  listCourseReviews: (courseId: number, offeringId = 0, cursor = '', pageSize = 20) =>
    listCourseReviews(courseId, offeringId, cursor, pageSize),
  getCourseSummary: (courseId: number, refresh = false, checkOnly = false) =>
    getCourseSummary(courseId, refresh, checkOnly),
}))

import ScheduleDetailList from '../src/site/components/schedule/ScheduleDetailList.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkCourseDetail, PkStagedCourse } from '../src/site/types/pk'

function makeDetail(code: string): PkCourseDetail {
  return {
    code,
    campus: '四平路校区',
    teachers: [{ teacherCode: 'T1', teacherName: '张伟' }],
    teachingLanguage: '中文',
    arrangementInfo: [
      {
        arrangementText: '星期一1-2节[1-16周] 四平路校区 A101',
        occupyDay: 1,
        occupyTime: [1, 2],
        occupyWeek: [1, 16],
        occupyRoom: 'A101',
        teacherAndCode: '张伟(T1)',
      },
    ],
  }
}

function makeStaged(courseCode: string, details: PkCourseDetail[]): PkStagedCourse {
  return {
    courseCode,
    courseName: courseCode,
    courseNameReserved: `课程${courseCode}`,
    credit: 3,
    courseType: '必',
    teacher: [],
    status: 0,
    courseDetail: details,
  }
}

function mountList() {
  return mount(ScheduleDetailList, { global: { plugins: [i18n] } })
}

describe('ScheduleDetailList 课评摘要与跳转', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    listCourseReviews.mockResolvedValue({ total: 0, list: [] })
    getCourseSummary.mockResolvedValue({ status: 'insufficient_data' })
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
    store.setClickedCourseInfo({ courseCode: '110001', courseName: '高等数学A(上)' })
    store.pushStagedCourse(makeStaged('110001', [makeDetail('110001.01')]))
  })

  test('展示课评评分与条数（有 courseId 时直达详情页）', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 7,
    })

    const wrapper = mountList()
    await flushPromises()

    expect(getPkCourseReviewBrief).toHaveBeenCalledWith({
      courseCode: '110001',
      teacherName: '',
      calendarId: 121,
    })
    // 摘要区显示平均分与条数（happy-dom 下 i18n 为 en）。
    expect(wrapper.text()).toContain('4.2')
    expect(wrapper.text()).toContain('7 reviews')
    // 课评按钮直达详情页。
    const link = wrapper.find('a[href="/courses/42"]')
    expect(link.exists()).toBe(true)
  })

  test('未匹配课评目录时回退课程搜索页', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 0,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: null,
      reviewCount: 0,
    })

    const wrapper = mountList()
    await flushPromises()

    const link = wrapper.find('a[href="/courses?keyword=110001"]')
    expect(link.exists()).toBe(true)
  })

  test('课评请求失败时显示加载失败且保留搜索回退入口', async () => {
    getPkCourseReviewBrief.mockRejectedValue(new Error('network'))

    const wrapper = mountList()
    await flushPromises()

    expect(wrapper.find('a[href="/courses?keyword=110001"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Failed to load schedule data')
  })
  test('教学班行内显示 offering 级课评摘要并跳转聚焦', async () => {
    // P13 classes：教学班 code ↔ offering.class_code（去掉点号归一化）匹配。
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 7,
      classes: [
        {
          classCode: '11000101',
          offeringId: 101,
          teachers: ['张伟'],
          ratingAvg: 4.5,
          reviewCount: 2,
        },
        {
          classCode: '11000102',
          offeringId: 102,
          teachers: ['李娜'],
          ratingAvg: null,
          reviewCount: 0,
        },
      ],
    })

    // 单条课程携带两个教学班（currentCourse 按 courseCode 匹配第一条记录）。
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setClickedCourseInfo({ courseCode: '110001', courseName: '高等数学A(上)' })
    store.pushStagedCourse(makeStaged('110001', [makeDetail('110001.01'), makeDetail('110001.02')]))
    const wrapper = mountList()
    await flushPromises()

    // 教学班行内课评链接：评分 + 条数，直达 /courses/42?offeringId=101 聚焦该班评价。
    const link = wrapper.find('a[href="/courses/42?offeringId=101"]')
    expect(link.exists()).toBe(true)
    expect(link.text()).toContain('4.5')
    expect(link.text()).toContain('2 reviews')
    // 无评分的教学班也有入口（offeringId 102），显示 0 条。
    const link2 = wrapper.find('a[href="/courses/42?offeringId=102"]')
    expect(link2.exists()).toBe(true)
    expect(link2.text()).toContain('0 reviews')
  })

  test('右侧浮动课评面板展示课程级评分卡与班级课评列表', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 7,
      ratingDistribution: [0, 0, 1, 2, 4],
      classes: [
        {
          classCode: '11000101',
          offeringId: 101,
          teachers: ['张伟'],
          ratingAvg: 4.5,
          reviewCount: 2,
        },
      ],
    })

    const wrapper = mountList()
    await flushPromises()

    // 右栏课评面板（role=complementary，与课评详情页 aside 语义一致）。
    const aside = wrapper.find('aside[role="complementary"]')
    expect(aside.exists()).toBe(true)
    // 课程级评分仪表卡（RatingSummaryCard）渲染均分与条数。
    expect(aside.text()).toContain('4.2')
    expect(aside.text()).toContain('7 reviews')
    // 班级课评列表：班号 + 聚焦链接 + 完整课评入口。
    expect(aside.text()).toContain('11000101')
    expect(aside.find('a[href="/courses/42?offeringId=101"]').exists()).toBe(true)
    expect(aside.find('a[href="/courses/42"]').exists()).toBe(true)
  })

  test('教学班无 offering 匹配时不显示行内课评链接', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 7,
      classes: [],
    })

    const wrapper = mountList()
    await flushPromises()

    // 无 classes 时教学班行内不渲染课评链接（旧数据包班号为空等场景）。
    expect(wrapper.find('a[href^="/courses/42?offeringId="]').exists()).toBe(false)
  })

  test('展示精选学生评价列表并调用 listCourseReviews', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.5,
      reviewCount: 1,
    })
    listCourseReviews.mockResolvedValue({
      total: 1,
      list: [
        {
          id: 99,
          author: { kind: 'member', label: '同济学子' },
          rating: 5,
          content: '老师讲得非常通俗易懂，强烈推荐！',
          createdAt: '2026-03-01',
        },
      ],
    })

    const wrapper = mountList()
    await flushPromises()

    expect(listCourseReviews).toHaveBeenCalledWith(42, 0, '', 3)
    const aside = wrapper.find('aside[role="complementary"]')
    expect(aside.text()).toContain('同济学子')
    expect(aside.text()).toContain('老师讲得非常通俗易懂，强烈推荐！')
  })

  test('支持切换并同步历史全部教学班课评（calendarId: 0）', async () => {
    getPkCourseReviewBrief.mockImplementation((input: { calendarId: number }) => {
      if (input.calendarId === 0) {
        return Promise.resolve({
          courseId: 42,
          courseCode: '110001',
          courseName: '高等数学A(上)',
          teacherName: '',
          ratingAvg: 4.3,
          reviewCount: 12,
          classes: [
            {
              classCode: '11000199',
              offeringId: 999,
              teachers: ['王历史'],
              ratingAvg: 4.8,
              reviewCount: 5,
            },
          ],
        })
      }
      return Promise.resolve({
        courseId: 42,
        courseCode: '110001',
        courseName: '高等数学A(上)',
        teacherName: '',
        ratingAvg: 4.0,
        reviewCount: 2,
        classes: [],
      })
    })

    const wrapper = mountList()
    await flushPromises()

    // 初始展示本学期教学班（当前 classes 为空）
    expect(wrapper.text()).toContain('No specific reviews for this semester\'s classes yet')

    // 点击切换至「历史全部教学班」
    const historyBtn = wrapper.findAll('button').find((b) => b.text().includes('All Historical Classes'))
    expect(historyBtn).toBeDefined()
    await historyBtn!.trigger('click')
    await flushPromises()

    // 触发 calendarId: 0 查询并渲染历史班级
    expect(getPkCourseReviewBrief).toHaveBeenCalledWith(
      expect.objectContaining({ calendarId: 0, courseCode: '110001' }),
    )
    const aside = wrapper.find('aside[role="complementary"]')
    expect(aside.text()).toContain('11000199')
    expect(aside.text()).toContain('王历史')
    expect(aside.text()).toContain('4.8')
  })

  test('点击具体教学班卡片时切换评价预览至该班，并支持重置回全课', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 7,
      classes: [
        {
          classCode: '11000101',
          offeringId: 101,
          teachers: ['张伟'],
          ratingAvg: 4.5,
          reviewCount: 2,
        },
        {
          classCode: '11000102',
          offeringId: 102,
          teachers: ['李娜'],
          ratingAvg: 3.8,
          reviewCount: 1,
        },
      ],
    })

    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setClickedCourseInfo({ courseCode: '110001', courseName: '高等数学A(上)' })
    store.pushStagedCourse(makeStaged('110001', [makeDetail('110001.01'), makeDetail('110001.02')]))

    const wrapper = mountList()
    await flushPromises()

    // 初始全课评论加载
    expect(listCourseReviews).toHaveBeenCalledWith(42, 0, '', 3)

    // 找到第一个教学班卡片（110001.01）并点击卡片
    const cards = wrapper.findAll('div.group.relative.rounded-xl.border')
    expect(cards.length).toBe(2)
    await cards[0].trigger('click')
    await flushPromises()

    // 验证教学班卡片高亮与预览状态指示
    expect(cards[0].classes()).toContain('border-primary')
    expect(cards[0].text()).toContain('Previewing')

    // 验证触发了对应 offeringId: 101 的教学班评论查询
    expect(listCourseReviews).toHaveBeenCalledWith(42, 101, '', 3)

    // 验证右侧面板渲染专属预览横幅与班级、教师信息
    const aside = wrapper.find('aside[role="complementary"]')
    expect(aside.text()).toContain('110001.01')
    expect(aside.text()).toContain('张伟')
    expect(aside.text()).toContain('View all reviews')

    // 点击「查看全课评价」重置按钮
    const resetBtn = aside.findAll('button').find((b) => b.text().includes('View all reviews'))
    expect(resetBtn).toBeDefined()
    await resetBtn!.trigger('click')
    await flushPromises()

    // 验证已恢复全课评论查询
    expect(listCourseReviews).toHaveBeenLastCalledWith(42, 0, '', 3)
  })

  test('加入课表按钮具备充裕边距样式且点击不触发卡片评价选中', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 7,
      classes: [
        {
          classCode: '11000101',
          offeringId: 101,
          teachers: ['张伟'],
          ratingAvg: 4.5,
          reviewCount: 2,
        },
      ],
    })

    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setClickedCourseInfo({ courseCode: '110001', courseName: '高等数学A(上)' })
    store.pushStagedCourse(makeStaged('110001', [makeDetail('110001.01')]))

    const wrapper = mountList()
    await flushPromises()

    const addBtn = wrapper.find('button.gf-button.gf-button-xs')
    expect(addBtn.exists()).toBe(true)
    // 验证具备充裕边距与防挤压类
    expect(addBtn.classes()).toContain('px-3')
    expect(addBtn.classes()).toContain('shrink-0')
    expect(addBtn.classes()).toContain('whitespace-nowrap')
    expect(addBtn.text()).toContain('Add to schedule')

    // 点击加入课表按钮
    await addBtn.trigger('click')
    await flushPromises()

    // 选课生效入表（status: 1 暂存入表）
    const stagedCourse = store.state.commonLists.stagedCourses.find((c) => c.courseCode === '110001')
    expect(stagedCourse?.courseDetail[0].status).toBe(1)

    // 教学班选择后，按钮变更为「已加入」状态（gf-button-primary，包含 Check 图标与 Added 文案）
    expect(addBtn.classes()).toContain('gf-button-primary')
    expect(addBtn.text()).toContain('Added')

    // 再次点击已加入按钮：退选当前班，按钮回归「加入课表」状态
    await addBtn.trigger('click')
    await flushPromises()
    expect(stagedCourse?.courseDetail[0].status).toBe(0)
    expect(addBtn.classes()).toContain('gf-button-secondary')
    expect(addBtn.text()).toContain('Add to schedule')
  })

  test('超长教学班课评列表默认折叠展示前 4 项，并支持展开与收起', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 20,
      classes: [
        { classCode: '11000101', offeringId: 101, teachers: ['老师1'], ratingAvg: 4.5, reviewCount: 5 },
        { classCode: '11000102', offeringId: 102, teachers: ['老师2'], ratingAvg: 4.0, reviewCount: 3 },
        { classCode: '11000103', offeringId: 103, teachers: ['老师3'], ratingAvg: 4.2, reviewCount: 4 },
        { classCode: '11000104', offeringId: 104, teachers: ['老师4'], ratingAvg: 4.8, reviewCount: 6 },
        { classCode: '11000105', offeringId: 105, teachers: ['老师5'], ratingAvg: 3.9, reviewCount: 1 },
        { classCode: '11000106', offeringId: 106, teachers: ['老师6'], ratingAvg: 4.1, reviewCount: 1 },
      ],
    })

    const wrapper = mountList()
    await flushPromises()

    const aside = wrapper.find('aside[role="complementary"]')
    // 默认折叠：仅展示前 4 个班级
    expect(aside.text()).toContain('11000101')
    expect(aside.text()).toContain('11000104')
    expect(aside.text()).not.toContain('11000105')
    expect(aside.text()).not.toContain('11000106')

    // 存在展开全部按钮（Show all 6 classes）
    const toggleBtn = aside.findAll('button').find((b) => b.text().includes('Show all 6 classes'))
    expect(toggleBtn).toBeDefined()
    expect(toggleBtn!.attributes('aria-expanded')).toBe('false')

    // 点击展开全部教学班
    await toggleBtn!.trigger('click')
    await flushPromises()

    expect(aside.text()).toContain('11000105')
    expect(aside.text()).toContain('11000106')
    expect(toggleBtn!.text()).toContain('Collapse class list')
    expect(toggleBtn!.attributes('aria-expanded')).toBe('true')

    // 再次点击收起
    await toggleBtn!.trigger('click')
    await flushPromises()

    expect(aside.text()).not.toContain('11000105')
    expect(toggleBtn!.text()).toContain('Show all 6 classes')
    expect(toggleBtn!.attributes('aria-expanded')).toBe('false')
  })

  test('折叠态下若聚焦后排教学班，自动保留激活班级可见', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.2,
      reviewCount: 20,
      classes: [
        { classCode: '11000101', offeringId: 101, teachers: ['老师1'], ratingAvg: 4.5, reviewCount: 5 },
        { classCode: '11000102', offeringId: 102, teachers: ['老师2'], ratingAvg: 4.0, reviewCount: 3 },
        { classCode: '11000103', offeringId: 103, teachers: ['老师3'], ratingAvg: 4.2, reviewCount: 4 },
        { classCode: '11000104', offeringId: 104, teachers: ['老师4'], ratingAvg: 4.8, reviewCount: 6 },
        { classCode: '11000105', offeringId: 105, teachers: ['老师5'], ratingAvg: 3.9, reviewCount: 1 },
      ],
    })

    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setClickedCourseInfo({ courseCode: '110001', courseName: '高等数学A(上)' })
    store.pushStagedCourse(
      makeStaged('110001', [
        makeDetail('110001.01'),
        makeDetail('110001.02'),
        makeDetail('110001.03'),
        makeDetail('110001.04'),
        makeDetail('110001.05'),
      ]),
    )

    const wrapper = mountList()
    await flushPromises()

    // 点击左侧第 5 个班级（110001.05）
    const cards = wrapper.findAll('div.group.relative.rounded-xl.border')
    expect(cards.length).toBe(5)
    await cards[4].trigger('click')
    await flushPromises()

    const aside = wrapper.find('aside[role="complementary"]')
    // 虽然未整体展开，但第 5 项因被激活而在可见列表中呈现
    expect(aside.text()).toContain('11000105')
  })

  test('精选学生评价完整展示全文且不包含 line-clamp-3 截断类', async () => {
    getPkCourseReviewBrief.mockResolvedValue({
      courseId: 42,
      courseCode: '110001',
      courseName: '高等数学A(上)',
      teacherName: '',
      ratingAvg: 4.5,
      reviewCount: 2,
    })
    listCourseReviews.mockResolvedValue({
      total: 2,
      list: [
        {
          id: 101,
          author: { kind: 'member', label: '同济学子' },
          rating: 5,
          contentHtml: '<p>第一段很长评价内容...</p><p>第二段依然非常重要，期末重点考第5章！</p>',
          content: '第一段很长评价内容...\n第二段依然非常重要，期末重点考第5章！',
          createdAt: '2026-03-01',
        },
        {
          id: 102,
          author: { kind: 'member', label: '匿名同学' },
          rating: 4,
          content: '纯文本长评价：给分非常公正，平时作业按时交即可拿到高分。',
          createdAt: '2026-03-02',
        },
      ],
    })

    const wrapper = mountList()
    await flushPromises()

    const aside = wrapper.find('aside[role="complementary"]')
    expect(aside.text()).toContain('期末重点考第5章！')
    expect(aside.text()).toContain('纯文本长评价：给分非常公正')

    // 确认已彻底移除 line-clamp-3，不截断全文
    const clampedEls = aside.findAll('.line-clamp-3')
    expect(clampedEls.length).toBe(0)
  })
})

// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

const getPkCourseReviewBrief = vi.fn()
vi.mock('../src/runtime/pk-api', () => ({
  getPkCourseReviewBrief: (input: { courseCode: string; teacherName: string }) =>
    getPkCourseReviewBrief(input),
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
})

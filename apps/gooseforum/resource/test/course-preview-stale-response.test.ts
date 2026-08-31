// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi, type MockInstance } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import CoursePreviewPane from '../src/site/components/CoursePreviewPane.vue'
import {
  getCourseRelated,
  listCourseReviews,
  type CourseRelatedResult,
  type ReviewPage,
  type ReviewPayload,
} from '../src/runtime/api'
import type { CourseSummaryPayload } from '@gooseforum/client'

// 预览面板切换课程时并发加载相关课程/评价（PR #338 review 竞态）：
// mock API 并手动控制每个课程请求的 resolve 时机，复现
// 「快速点选 A→B 后 A 的晚到响应覆盖 B 的数据」。
vi.mock('../src/runtime/api', () => ({
  bookmarkCourse: vi.fn(async () => true),
  getCourseRelated: vi.fn(),
  listCourseReviews: vi.fn(),
  // AISummaryCard 默认折叠不请求；提供兜底实现避免真实 fetch。
  getCourseSummary: vi.fn(async () => ({ status: 'disabled' })),
}))

type Deferred<T> = { promise: Promise<T>; resolve: (value: T) => void }

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((r) => {
    resolve = r
  })
  return { promise, resolve }
}

function courseFixture(id: number, name: string): CourseSummaryPayload {
  return {
    id,
    primaryCode: `CS${id}`,
    name,
    department: '计算机科学与技术学院',
    creditX10: 30,
    teacherName: `老师${id}`,
  }
}

function relatedItem(id: number, name: string) {
  return {
    id,
    primaryCode: `CS${id}0`,
    name,
    department: '计算机科学与技术学院',
    ratingAvg: 4.5,
    ratingCount: 3,
    reviewCount: 3,
  }
}

function reviewFixture(id: number, content: string): ReviewPayload {
  return {
    id,
    offeringId: 1,
    rating: 5,
    content,
    contentHtml: '',
    author: { kind: 'member', label: `用户${id}` },
    viewer: { canEdit: false, canDelete: false, isHelpful: false, isDisliked: false },
    helpfulCount: 0,
    dislikeCount: 0,
    createdAt: '2026-01-01 00:00:00',
    updatedAt: '2026-01-01 00:00:00',
  }
}

const relatedPending = new Map<number, Deferred<CourseRelatedResult>>()
const reviewsPending = new Map<number, Deferred<ReviewPage>>()

function mountPane(course: CourseSummaryPayload) {
  return mount(CoursePreviewPane, {
    props: { course, isAuthenticated: false, bookmarkedCourseIds: [] },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
}

let matchMediaSpy: MockInstance

describe('CoursePreviewPane 切换课程的过期响应守卫', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh'
    relatedPending.clear()
    reviewsPending.clear()
    // 桌面端（非 mobile 抽屉）：避免面板模态行为干扰断言。
    matchMediaSpy = vi.spyOn(window, 'matchMedia').mockReturnValue({
      matches: false,
      media: '(max-width: 1023.98px)',
      addEventListener: () => {},
      removeEventListener: () => {},
    } as unknown as MediaQueryList)
    vi.mocked(getCourseRelated).mockImplementation((courseId: number) => {
      const pending = deferred<CourseRelatedResult>()
      relatedPending.set(courseId, pending)
      return pending.promise
    })
    vi.mocked(listCourseReviews).mockImplementation((courseId: number) => {
      const pending = deferred<ReviewPage>()
      reviewsPending.set(courseId, pending)
      return pending.promise
    })
  })

  afterEach(() => {
    matchMediaSpy.mockRestore()
    document.body.innerHTML = ''
    document.body.style.overflow = ''
  })

  test('旧课程的晚到响应不得覆盖新课程的相关课程与评价', async () => {
    const wrapper = mountPane(courseFixture(1, '课程A'))
    await flushPromises()
    expect(relatedPending.get(1)).toBeDefined()
    expect(reviewsPending.get(1)).toBeDefined()

    // 快速切换到课程 B：A 的两个请求仍在途。
    await wrapper.setProps({ course: courseFixture(2, '课程B') })
    await flushPromises()
    expect(relatedPending.get(2)).toBeDefined()
    expect(reviewsPending.get(2)).toBeDefined()

    // B 的响应先返回：渲染 B 的相关课程。
    relatedPending.get(2)!.resolve({
      teacherOtherCourses: [relatedItem(21, 'B老师其他课')],
      sameCourseOtherTeachers: [],
    })
    reviewsPending.get(2)!.resolve({ list: [], total: 0 })
    await flushPromises()
    expect(wrapper.text()).toContain('B老师其他课')

    // A 的响应晚到：必须按代际丢弃，不得覆盖 B 的数据。
    relatedPending.get(1)!.resolve({
      teacherOtherCourses: [relatedItem(11, 'A老师其他课')],
      sameCourseOtherTeachers: [],
    })
    reviewsPending.get(1)!.resolve({ list: [reviewFixture(11, 'A的评价内容')], total: 1 })
    await flushPromises()
    expect(wrapper.text()).toContain('B老师其他课')
    expect(wrapper.text()).not.toContain('A老师其他课')
    expect(wrapper.text()).not.toContain('A的评价内容')

    wrapper.unmount()
  })
})

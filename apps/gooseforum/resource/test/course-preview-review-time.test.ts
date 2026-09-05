// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi, type MockInstance } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import CoursePreviewPane from '../src/site/components/CoursePreviewPane.vue'
import { formatDateTime } from '../src/runtime/format'
import {
  getCourseRelated,
  listCourseReviews,
  type ReviewPage,
  type ReviewPayload,
} from '../src/runtime/api'
import type { CourseSummaryPayload } from '@gooseforum/client'

// Issue #401：预览面板评价时间戳曾以裸 ISO 时间渲染（如 `2026-06-07T16:41:13Z`），
// 与详情页 reviewDateLabel() 的 formatDateTime 展示不一致。
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

function courseFixture(id: number): CourseSummaryPayload {
  return {
    id,
    primaryCode: `CS${id}`,
    name: `课程${id}`,
    department: '计算机科学与技术学院',
    creditX10: 30,
    teacherName: `老师${id}`,
  }
}

const reviewsPending = new Map<number, Deferred<ReviewPage>>()

let matchMediaSpy: MockInstance

describe('CoursePreviewPane 用户评价时间戳格式化', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh'
    reviewsPending.clear()
    // 桌面端（非 mobile 抽屉）：避免面板模态行为干扰断言。
    matchMediaSpy = vi.spyOn(window, 'matchMedia').mockReturnValue({
      matches: false,
      media: '(max-width: 1023.98px)',
      addEventListener: () => {},
      removeEventListener: () => {},
    } as unknown as MediaQueryList)
    vi.mocked(getCourseRelated).mockResolvedValue({
      teacherOtherCourses: [],
      sameCourseOtherTeachers: [],
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

  test('评价时间戳以 formatDateTime 格式化展示，而非裸 ISO 时间', async () => {
    const wrapper = mount(CoursePreviewPane, {
      props: { course: courseFixture(1), isAuthenticated: false, bookmarkedCourseIds: [] },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    const createdAt = '2026-06-07T16:41:13Z'
    const review: ReviewPayload = {
      id: 1,
      offeringId: 1,
      rating: 5,
      content: '评价内容',
      contentHtml: '',
      author: { kind: 'member', label: '用户1' },
      viewer: { canEdit: false, canDelete: false, isHelpful: false, isDisliked: false },
      helpfulCount: 0,
      dislikeCount: 0,
      createdAt,
      updatedAt: createdAt,
    }
    reviewsPending.get(1)!.resolve({ list: [review], total: 1 })
    await flushPromises()

    // 与详情页 reviewDateLabel() 一致：展示 formatDateTime 输出。
    expect(wrapper.text()).toContain(formatDateTime(createdAt))
    // 不再出现裸 ISO 时间戳。
    expect(wrapper.text()).not.toContain(createdAt)

    wrapper.unmount()
  })
})

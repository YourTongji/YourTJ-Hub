// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import relatedFixture from '../../../../packages/api-contract/fixtures/course-related-success.json'
import { i18n } from '../src/runtime/i18n'
import { getCourseRelated, type RelatedCourseItem } from '../src/runtime/api'
import CoursePreviewPane from '../src/site/components/CoursePreviewPane.vue'

vi.mock('../src/runtime/api', () => ({
  bookmarkCourse: vi.fn(async () => true),
  getCourseRelated: vi.fn(),
  listCourseReviews: vi.fn(async () => ({ list: [], total: 0 })),
  getCourseSummary: vi.fn(async () => ({ status: 'disabled' })),
}))

let wrapper: VueWrapper | undefined

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  sessionStorage.clear()
})

afterEach(() => {
  wrapper?.unmount()
  vi.clearAllMocks()
})

async function renderOtherTeachers(items: RelatedCourseItem[]) {
  vi.mocked(getCourseRelated).mockResolvedValue({
    teacherOtherCourses: relatedFixture.result.teacherOtherCourses,
    sameCourseOtherTeachers: items,
  })
  wrapper = mount(CoursePreviewPane, {
    props: {
      course: { id: 42, primaryCode: '100001', name: '高等数学(A)上', department: '数学科学学院', creditX10: 50 },
      isAuthenticated: false,
      bookmarkedCourseIds: [],
    },
    global: { plugins: [i18n], directives: { 'code-highlight': {}, 'math-render': {} } },
  })
  await flushPromises()
  return wrapper.findAll('section').find(section => section.find('h3').text() === '该课程的其他老师')!
}

describe('CoursePreviewPane 同课程其他教师', () => {
  test('renders each identity teacher from the contract response without an instructors array', async () => {
    const items = ['李四', '王五', '赵六'].map((teacherName, index) => ({
      ...relatedFixture.result.sameCourseOtherTeachers[0],
      id: 47 + index,
      teacherName,
    }))
    const section = await renderOtherTeachers(items)
    const links = section.findAll('a')
    expect(links).toHaveLength(items.length)
    items.forEach((item, index) => {
      expect(links[index].get('h4').text()).toBe(item.teacherName)
      expect(links[index].attributes('href')).toBe(`/courses/${item.id}`)
    })
    expect(section.text()).not.toContain('无教师')
  })

  test('uses the course identity teacher even when an offering teacher list is present', async () => {
    const section = await renderOtherTeachers([{
      ...relatedFixture.result.sameCourseOtherTeachers[0],
      instructors: ['张三', '李四'],
    }])
    expect(section.get('h4').text()).toBe('李四')
  })

  test.each([undefined, ''])('shows the localized fallback when teacherName is %s', async teacherName => {
    const section = await renderOtherTeachers([{
      ...relatedFixture.result.sameCourseOtherTeachers[0],
      teacherName,
    }])
    expect(section.get('h4').text()).toBe(i18n.global.t('coursesPage.noTeacher'))
  })
})

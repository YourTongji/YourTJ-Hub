import { describe, expect, test } from 'vitest'
import { comparePlans } from '../src/site/utils/pkPlanCompare'
import type { PkPlan, PkStagedCourse } from '../src/site/types/pk'

function makePlan(overrides: Partial<PkPlan> = {}): PkPlan {
  return {
    id: 'plan_1',
    name: '方案 1',
    createdAt: 1,
    stagedCourses: [],
    selectedCourses: [],
    customEvents: [],
    ...overrides,
  }
}

function staged(courseCode: string, name: string, classCodes: string[]): PkStagedCourse {
  return {
    courseCode,
    courseName: name,
    courseNameReserved: name,
    credit: 3,
    courseType: '必',
    teacher: [],
    status: classCodes.length > 0 ? 1 : 0,
    courseDetail: classCodes.map((code) => ({
      code,
      campus: '',
      teachers: [],
      teachingLanguage: '',
      status: 1,
      arrangementInfo: [
        { arrangementText: '', occupyDay: 1, occupyTime: [1], occupyWeek: [1], occupyRoom: '', teacherAndCode: '' },
      ],
    })),
  }
}

describe('comparePlans 方案对比', () => {
  test('相同方案无差异', () => {
    const a = makePlan({
      stagedCourses: [staged('122004', '高数', ['122004.01'])],
      customEvents: [{ id: 'e1', label: '有事', day: 5, sections: [1], weeks: [1] }],
    })
    const b = makePlan({
      stagedCourses: [staged('122004', '高数', ['122004.01'])],
      customEvents: [{ id: 'e2', label: '有事', day: 5, sections: [1], weeks: [1] }],
    })
    const result = comparePlans([a, b])
    expect(result.diffCount).toBe(0)
    expect(result.courses[0].different).toBe(false)
    expect(result.placeholders[0].different).toBe(false)
  })

  test('同课程不同班级 → 差异行', () => {
    const a = makePlan({ stagedCourses: [staged('122004', '高数', ['122004.01'])] })
    const b = makePlan({ stagedCourses: [staged('122004', '高数', ['122004.02'])] })
    const result = comparePlans([a, b])
    expect(result.diffCount).toBe(1)
    expect(result.courses).toHaveLength(1)
    expect(result.courses[0].different).toBe(true)
    expect(result.courses[0].classesByPlan).toEqual([['122004.01'], ['122004.02']])
  })

  test('课程只在一套方案中 → 差异行', () => {
    const a = makePlan({ stagedCourses: [staged('122004', '高数', ['122004.01'])] })
    const b = makePlan()
    const result = comparePlans([a, b])
    expect(result.diffCount).toBe(1)
    expect(result.courses[0].classesByPlan).toEqual([['122004.01'], []])
    expect(result.courses[0].different).toBe(true)
  })

  test('未排入课表的课程不参与对比', () => {
    const a = makePlan({ stagedCourses: [staged('G200', '通识', [])] })
    const b = makePlan()
    const result = comparePlans([a, b])
    expect(result.courses).toHaveLength(0)
    expect(result.diffCount).toBe(0)
  })

  test('占位不同 → 各为差异行', () => {
    const a = makePlan({ customEvents: [{ id: 'e1', label: '有事', day: 5, sections: [1], weeks: [1] }] })
    const b = makePlan({ customEvents: [{ id: 'e2', label: '有事', day: 5, sections: [2], weeks: [1] }] })
    const result = comparePlans([a, b])
    expect(result.placeholders).toHaveLength(2)
    expect(result.placeholders.filter((row) => row.different)).toHaveLength(2)
    expect(result.diffCount).toBe(2)
  })

  test('空方案对比无差异', () => {
    const result = comparePlans([makePlan(), makePlan()])
    expect(result.courses).toHaveLength(0)
    expect(result.placeholders).toHaveLength(0)
    expect(result.diffCount).toBe(0)
  })
})
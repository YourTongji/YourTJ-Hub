import { describe, expect, test } from 'vitest'
import { mergeCourses } from '../src/site/utils/course-merge'
import type { CourseSummaryPayload } from '@gooseforum/client'

function course(id: number): CourseSummaryPayload {
  return { id, primaryCode: `C${id}`, name: `Course ${id}`, department: 'D', creditX10: 0 }
}

describe('mergeCourses（无限滚动课程去重合并）', () => {
  test('空列表返回原样', () => {
    expect(mergeCourses([], [])).toEqual([])
  })

  test('追加无重复项', () => {
    const merged = mergeCourses([course(1), course(2)], [course(3)])
    expect(merged.map((c) => c.id)).toEqual([1, 2, 3])
  })

  test('跨页边界重复项去重（保留先出现者）', () => {
    const merged = mergeCourses([course(1), course(2)], [course(2), course(3)])
    expect(merged.map((c) => c.id)).toEqual([1, 2, 3])
  })

  test('全重复不追加', () => {
    const merged = mergeCourses([course(1)], [course(1)])
    expect(merged.map((c) => c.id)).toEqual([1])
  })
})

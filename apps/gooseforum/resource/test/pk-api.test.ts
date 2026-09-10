import { afterEach, describe, expect, test, vi } from 'vitest'

// Node 测试环境无 document：mock i18n 为透传键，验证 PK API 适配层
// 按 OpenAPI 契约的 wire 形状解析（键存在性由 pnpm check:i18n 门禁保证）。
vi.mock('../src/runtime/i18n', () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}))

import {
  getPkCampuses,
  getPkCoursesByMajor,
  getPkCoursesByNature,
  getPkFaculties,
  getPkGrades,
  getPkLatestUpdate,
  getPkMajors,
  searchPkCourses,
} from '../src/runtime/pk-api'

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('pk-api wire shape adaptation', () => {
  test('getPkGrades unwraps {gradeList}', async () => {
    // 契约 PkGradesResult：data = { gradeList: [...] }（fixtures/pk-grades-success.json）。
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: '查询成功', data: { gradeList: [2025, 2024] } })),
    )
    await expect(getPkGrades(121)).resolves.toEqual([2025, 2024])
  })

  test('getPkMajors passes through array', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          jsonResponse({ code: 0, msg: '查询成功', data: [{ code: '00301', name: '2025(00301 数学类)' }] }),
        ),
    )
    await expect(getPkMajors(2025, 121)).resolves.toEqual([{ code: '00301', name: '2025(00301 数学类)' }])
  })

  test('getPkCampuses maps campusId/campusName to code/name', async () => {
    // 契约 PkCampusItem：data = [{ campusId, campusName }]。
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: '查询成功', data: [{ campusId: '1', campusName: '四平路校区' }] })),
    )
    await expect(getPkCampuses()).resolves.toEqual([{ code: '1', name: '四平路校区' }])
  })

  test('getPkFaculties maps facultyId/facultyName to code/name', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: '查询成功', data: [{ facultyId: '000033', facultyName: '继续教育学院' }] })),
    )
    await expect(getPkFaculties()).resolves.toEqual([{ code: '000033', name: '继续教育学院' }])
  })

  test('getPkCoursesByMajor maps courses to courseDetail', async () => {
    // 契约 PkCourseByMajorItem：data 数组元素的教学班字段是 courses，前端 PkCourse 用 courseDetail。
    const item = {
      courseCode: 'TJCS101',
      courseName: '计算机程序设计',
      faculty: '计算机科学与技术系',
      facultyI18n: '计算机科学与技术系',
      credit: 3,
      grade: 2025,
      courseNature: ['专业必修'],
      courses: [
        {
          code: 'TJCS10101',
          teachers: [{ teacherCode: 'T1', teacherName: '张三' }],
          campus: '四平路校区',
          teachingLanguage: '中文',
          arrangementInfo: [],
          isExclusive: true,
        },
      ],
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: '查询成功', data: [item] })))
    const result = await getPkCoursesByMajor(2025, '00301', 122)
    expect(result).toHaveLength(1)
    expect(result[0].courseCode).toBe('TJCS101')
    expect(result[0].courseDetail).toEqual(item.courses)
    expect(result[0].grade).toBe(2025)
  })

  test('getPkCoursesByNature flattens groups with courseNature label', async () => {
    // 契约 PkCourseByNatureItem：data 为分组数组，courses 元素带 courseLabelName。
    const group = {
      courseLabelId: 2,
      courseLabelIds: [2],
      courseLabelName: '通识选修课',
      crossDiscipline: false,
      courses: [
        {
          campus: ['嘉定校区'],
          courseCode: 'TJCS201',
          courseName: '数据结构与算法',
          faculty: '计算机科学与技术系',
          facultyI18n: '计算机科学与技术系',
          credit: 4,
          courseLabelName: '通识选修课',
          crossDiscipline: false,
        },
      ],
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: '查询成功', data: [group] })))
    const result = await getPkCoursesByNature(122, [2])
    expect(result).toHaveLength(1)
    expect(result[0].courseCode).toBe('TJCS201')
    expect(result[0].courseNature).toEqual(['通识选修课'])
  })

  test('searchPkCourses unwraps {courses, sizeLimit}', async () => {
    // 契约 PkSearchResult：data = { courses: [...], sizeLimit }。
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse({
          code: 0,
          msg: '查询成功',
          data: {
            courses: [{ courseCode: 'TJCS101', courseName: '计算机程序设计' }],
            sizeLimit: 100,
          },
        }),
      ),
    )
    const result = await searchPkCourses({ calendarId: 122, courseName: '计算机' })
    expect(result).toHaveLength(1)
    expect(result[0].courseCode).toBe('TJCS101')
  })

  test('getPkLatestUpdate unwraps bare string', async () => {
    // 契约 PkLatestUpdateResponse：data 为字符串或 null。
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: '查询成功', data: '2026-08-17' })))
    await expect(getPkLatestUpdate()).resolves.toEqual({ latestSyncAt: '2026-08-17' })

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: '查询成功', data: null })))
    await expect(getPkLatestUpdate()).resolves.toEqual({ latestSyncAt: null })
  })
})

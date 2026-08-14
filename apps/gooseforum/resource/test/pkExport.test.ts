import { describe, expect, test, vi } from 'vitest'

// Node 测试环境无 document：mock i18n 为固定映射，验证导出 i18n 化逻辑
// （键存在性由 pnpm check:i18n 门禁保证）。
vi.mock('../src/runtime/i18n', () => ({
  i18n: {
    global: {
      t: (key: string, named?: Record<string, unknown>) => {
        const map: Record<string, string> = {
          'schedule.exportWeekRange': 'Wk {range}',
          'schedule.exportWeekJoin': ', ',
          'schedule.exportColCourseName': 'Course Name',
          'schedule.exportColWeekday': 'Day',
          'schedule.exportColStart': 'Start Period',
          'schedule.exportColEnd': 'End Period',
          'schedule.exportColTeacher': 'Teacher',
          'schedule.exportColRoom': 'Room',
          'schedule.exportColWeeks': 'Weeks',
          'schedule.exportColCode': 'Course Code',
          'schedule.exportColTeacherName': 'Teacher Name',
          'schedule.exportSheetName': 'Schedule',
        }
        const tmpl = map[key] ?? key
        return named ? tmpl.replace(/\{(\w+)\}/g, (_, k) => String(named[k] ?? '')) : tmpl
      },
    },
  },
}))

import {
  codesToCsvRows,
  codesToXlsRows,
  extractTeacherNames,
  formatWeeks,
  jsonToCsv,
  xlsRowsToXml,
} from '../src/site/utils/pkExport'
import type { PkStagedCourse } from '../src/site/types/pk'

const stagedCourse: PkStagedCourse = {
  courseCode: '122004',
  courseName: '高等数学(122004)',
  courseNameReserved: '高等数学',
  credit: 4,
  courseType: '必',
  teacher: [{ teacherName: '张三', teacherCode: '001' }],
  status: 1,
  courseDetail: [
    {
      code: '122004.01',
      campus: '四平',
      teachers: [{ teacherName: '张三', teacherCode: '001' }],
      teachingLanguage: '中文',
      arrangementInfo: [
        {
          arrangementText: '[1-8周] 周一第3-4节',
          occupyDay: 1,
          occupyTime: [3, 4],
          occupyWeek: [1, 2, 3, 4, 5, 6, 7, 8],
          occupyRoom: 'A101',
          teacherAndCode: '张三(001)',
        },
      ],
    },
  ],
}

describe('formatWeeks', () => {
  test('连续区间', () => {
    // 导出周数文案随界面语言（测试环境默认 en）：Wk 前缀 + ', ' 分隔
    expect(formatWeeks([1, 2, 3, 4])).toBe('Wk 1-4')
  })

  test('多段区间', () => {
    expect(formatWeeks([1, 2, 5, 6])).toBe('Wk 1-2, Wk 5-6')
  })

  test('单周', () => {
    expect(formatWeeks([3])).toBe('Wk 3')
  })

  test('空', () => {
    expect(formatWeeks([])).toBe('')
  })
})

describe('extractTeacherNames', () => {
  test('提取姓名去掉工号', () => {
    expect(extractTeacherNames('张三(001),李四(002)')).toBe('张三,李四')
  })

  test('空返回空串', () => {
    expect(extractTeacherNames('')).toBe('')
  })
})

describe('jsonToCsv', () => {
  test('普通行', () => {
    const csv = jsonToCsv([
      { courseName: '高等数学', occupyDay: 1, start: 3, end: 4, teacherName: '张三', occupyRoom: 'A101', occucpyWeek: 'Wk 1-8' },
    ])
    expect(csv).toContain('高等数学,1,3,4,张三,A101,Wk 1-8')
  })

  test('字段含逗号/引号时转义', () => {
    const csv = jsonToCsv([
      { courseName: '计算机,"原理"', occupyDay: 1, start: 3, end: 4, teacherName: '张三', occupyRoom: 'A101', occucpyWeek: 'Wk 1-8' },
    ])
    expect(csv).toContain('"计算机,""原理"""')
  })
})

describe('codesToCsvRows', () => {
  test('从已选班级构造 CSV 行（表头 + 每段一行）', () => {
    const rows = codesToCsvRows(['122004.01'], [stagedCourse])
    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({ courseName: 'Course Name', occupyDay: 'Day' })
    expect(rows[1]).toMatchObject({
      courseName: '高等数学',
      occupyDay: 1,
      start: 3,
      end: 4,
      teacherName: '张三',
      occupyRoom: 'A101',
      occucpyWeek: 'Wk 1-8',
    })
  })

  test('找不到课程时跳过', () => {
    const rows = codesToCsvRows(['999999.01'], [stagedCourse])
    expect(rows).toHaveLength(1) // 仅表头
  })
})

describe('codesToXlsRows', () => {
  test('从已选班级构造 XLS 行', () => {
    const rows = codesToXlsRows(['122004.01'], [stagedCourse])
    expect(rows[0]).toEqual({ code: '122004.01', courseName: '高等数学', teacherName: '张三' })
  })
})

describe('xlsRowsToXml', () => {
  test('生成 SpreadsheetML 2003 结构并转义', () => {
    const xml = xlsRowsToXml([{ code: 'A<1>', courseName: 'B&B', teacherName: 'C' }], '课表')
    expect(xml).toContain('<?xml version="1.0" encoding="UTF-8"?>')
    expect(xml).toContain('<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"')
    expect(xml).toContain('<Worksheet ss:Name="课表">')
    expect(xml).toContain('Course Code')
    expect(xml).toContain('<Data ss:Type="String">A&lt;1&gt;</Data>')
    expect(xml).toContain('<Data ss:Type="String">B&amp;B</Data>')
    expect(xml).toContain('</Workbook>')
  })
})

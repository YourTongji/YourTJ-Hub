// 排课器课表导出：CSV（手写字符串）与 XLS（SpreadsheetML 2003 XML）。
//
// 不引入第三方导出库：CSV 手写拼接（带引号转义 + UTF-8 BOM），
// XLS 生成 Excel 兼容的 SpreadsheetML 2003 格式（.xls 扩展名，Excel/WPS 可直接打开）。
// 导出内容遍历 selectedCourses + stagedCourses 的班级时间槽，与课表数据同源，保证一致。

import { getCourseBaseCode } from '@/site/utils/pkConflict'
import type { PkCsvCourse, PkStagedCourse, PkXlsCourse } from '@/site/types/pk'

const CSV_HEADER: PkCsvCourse = {
  courseName: '课程名称',
  occupyDay: '星期',
  start: '开始节数',
  end: '结束节数',
  teacherName: '老师',
  occupyRoom: '地点',
  occucpyWeek: '周数',
}

const XLS_HEADER = ['课程代码', '课程名称', '教师姓名']

/**
 * 将已选班级课号数组映射为 CSV 行（一门课的一个时间段一行）。
 * codes 为班级课号（含班号）；staged 为备选/已选课程池。
 */
export function codesToCsvRows(codes: readonly string[], staged: readonly PkStagedCourse[]): PkCsvCourse[] {
  const rows: PkCsvCourse[] = [CSV_HEADER]
  for (const code of codes) {
    const base = getCourseBaseCode(code)
    const course = staged.find((c) => c.courseCode === base)
    if (!course) continue
    const targetClass = course.courseDetail.find((d) => d.code === code)
    if (!targetClass) continue
    for (const arr of targetClass.arrangementInfo) {
      rows.push({
        courseName: course.courseNameReserved,
        occupyDay: arr.occupyDay,
        start: arr.occupyTime[0],
        end: arr.occupyTime.length === 1 ? arr.occupyTime[0] : arr.occupyTime[arr.occupyTime.length - 1],
        teacherName: extractTeacherNames(arr.teacherAndCode),
        occupyRoom: arr.occupyRoom,
        occucpyWeek: formatWeeks(arr.occupyWeek),
      })
    }
  }
  return rows
}

/** 将已选班级课号数组映射为 XLS 行。 */
export function codesToXlsRows(codes: readonly string[], staged: readonly PkStagedCourse[]): PkXlsCourse[] {
  const rows: PkXlsCourse[] = []
  for (const code of codes) {
    const base = getCourseBaseCode(code)
    const course = staged.find((c) => c.courseCode === base)
    if (!course) continue
    const targetClass = course.courseDetail.find((d) => d.code === code)
    if (!targetClass) continue
    rows.push({
      code,
      courseName: course.courseNameReserved,
      teacherName: course.teacher.map((t) => t.teacherName).filter(Boolean).join(','),
    })
  }
  return rows
}

/** 将展开周数组格式化为可读区间文本，如 [1,2,3,4,5,6,7,8] → "1-8周"。 */
export function formatWeeks(weeks: readonly number[]): string {
  const sorted = [...weeks].sort((a, b) => a - b)
  if (sorted.length === 0) return ''
  const ranges: string[] = []
  let start = sorted[0]
  let prev = sorted[0]
  for (let i = 1; i <= sorted.length; i++) {
    const cur = sorted[i]
    if (cur === prev + 1) {
      prev = cur
      continue
    }
    ranges.push(start === prev ? `${start}周` : `${start}-${prev}周`)
    start = cur
    prev = cur
  }
  return ranges.join('、')
}

/** 从 "教师名(工号),教师名2(工号2)" 提取纯教师名列表。 */
export function extractTeacherNames(teacherAndCode: string): string {
  return String(teacherAndCode ?? '')
    .split(',')
    .map((part) => part.split('(')[0].trim())
    .filter(Boolean)
    .join(',')
}

/** 手写 CSV 字符串（带字段引号转义）。 */
export function jsonToCsv(rows: PkCsvCourse[]): string {
  return rows
    .map((row) =>
      [
        row.courseName,
        row.occupyDay,
        row.start,
        row.end,
        row.teacherName,
        row.occupyRoom,
        row.occucpyWeek,
      ]
        .map((value) => csvEscape(value))
        .join(','),
    )
    .join('\r\n')
}

/** 生成 SpreadsheetML 2003（.xls）XML 字符串。 */
export function xlsRowsToXml(rows: PkXlsCourse[], sheetName = '辅助表'): string {
  const escape = xmlEscape
  const lines: string[] = [
    `<?xml version="1.0" encoding="UTF-8"?>`,
    `<?mso-application progid="Excel.Sheet"?>`,
    `<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"`,
    ` xmlns:o="urn:schemas-microsoft-com:office:office"`,
    ` xmlns:x="urn:schemas-microsoft-com:office:excel"`,
    ` xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"`,
    ` xmlns:html="http://www.w3.org/TR/REC-html40">`,
    `<Worksheet ss:Name="${escape(sheetName)}">`,
    `<Table>`,
  ]

  lines.push(`<Row>${XLS_HEADER.map((h) => `<Cell><Data ss:Type="String">${escape(h)}</Data></Cell>`).join('')}</Row>`)
  for (const row of rows) {
    lines.push(
      `<Row><Cell><Data ss:Type="String">${escape(row.code)}</Data></Cell>` +
        `<Cell><Data ss:Type="String">${escape(row.courseName)}</Data></Cell>` +
        `<Cell><Data ss:Type="String">${escape(row.teacherName)}</Data></Cell></Row>`,
    )
  }

  lines.push(`</Table>`, `</Worksheet>`, `</Workbook>`)
  return lines.join('\n')
}

/** 触发浏览器下载。 */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = window.URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  window.URL.revokeObjectURL(url)
}

/** 下载 CSV（带 UTF-8 BOM，Excel 正确识别中文）。 */
export function downloadCsv(filename: string, csv: string): void {
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' })
  downloadBlob(blob, filename)
}

/** 下载 XLS（SpreadsheetML XML）。 */
export function downloadXls(filename: string, xml: string): void {
  const blob = new Blob([xml], { type: 'application/vnd.ms-excel;charset=utf-8;' })
  downloadBlob(blob, filename)
}

function csvEscape(value: string | number): string {
  let s = String(value)
  // 防 Excel 公式注入：以 = + - @ 开头的单元格加前导单引号，令其按文本显示。
  if (/^[=+\-@]/.test(s)) s = `'${s}`
  if (/[",\r\n]/.test(s)) return `"${s.replace(/"/g, '""')}"`
  return s
}

function xmlEscape(value: string): string {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

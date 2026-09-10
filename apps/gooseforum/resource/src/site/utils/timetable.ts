// 课表时间段与节次映射工具函数

/**
 * 将课表行/节次（1..11 或 1..12）映射为教务系统及后端 P10 `/api/pk/courses-by-time` 对应的大节 Section（1..6）。
 * - calendarId >= 120 为 2025-2026 学年第 1 学期及以后的 11 节新课制；
 * - 其余或缺省为 12 节旧课制。
 *
 * 映射规则：
 * - 1-2 节  → Section 1
 * - 3-4 节  → Section 2
 * - 5-6 节  → Section 3
 * - 7-8 节  → Section 4
 * - 9-10 节 → Section 5
 * - 11 节（新制）或 11-12 节（旧制） → Section 6
 */
export function getRowSection(row: number, calendarId = 0): number {
  if (calendarId >= 120) {
    switch (row) {
      case 1:
      case 2:
        return 1
      case 3:
      case 4:
        return 2
      case 5:
      case 6:
        return 3
      case 7:
      case 8:
        return 4
      case 9:
      case 10:
        return 5
      case 11:
        return 6
      default:
        return -1
    }
  } else {
    switch (row) {
      case 1:
      case 2:
        return 1
      case 3:
      case 4:
        return 2
      case 5:
      case 6:
        return 3
      case 7:
      case 8:
        return 4
      case 9:
      case 10:
        return 5
      case 11:
      case 12:
        return 6
      default:
        return -1
    }
  }
}

/**
 * 大节 Section（1..6）对应的标准节次范围文字描述。
 */
export function getSectionRangeText(section: number, calendarId = 0): string {
  switch (section) {
    case 1:
      return '1-2'
    case 2:
      return '3-4'
    case 3:
      return '5-6'
    case 4:
      return '7-8'
    case 5:
      return '9-10'
    case 6:
      return calendarId >= 120 ? '11' : '11-12'
    default:
      return ''
  }
}

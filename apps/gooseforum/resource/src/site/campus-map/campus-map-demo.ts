import type { CampusEvent } from '@gooseforum/client'

// Local preview fixtures only. CampusMapMinePanel loads this module only in Vite DEV mode.
export const campusMapDemoEvents: CampusEvent[] = [
  { name: '高等数学（示例）', teacher: '示例教师', room: '北115', campus: '四平路校区', credits: '4', day: 1, start: 1, end: 2, weeks: [6] },
  { name: '大学物理（示例）', teacher: '示例教师', room: '南115', campus: '四平路校区', credits: '3', day: 1, start: 3, end: 4, weeks: [6] },
  { name: '数据结构（示例）', teacher: '示例教师', room: '济事南楼', campus: '嘉定校区', credits: '3', day: 1, start: 5, end: 6, weeks: [6] },
  { name: '软件工程（示例）', teacher: '示例教师', room: '济事北楼', campus: '嘉定校区', credits: '2', day: 1, start: 7, end: 8, weeks: [6] },
]

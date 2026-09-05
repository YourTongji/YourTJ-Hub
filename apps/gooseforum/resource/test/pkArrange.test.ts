import { describe, expect, test } from 'vitest'
import {
  buildWeekMask,
  consolidateSameClassArrangements,
  detectWeekParity,
  maxRowsForCalendar,
  parseArrangeInfoText,
  weekMasksOverlap,
  weeksOverlap,
} from '../src/site/utils/pkArrange'
import { getRowSection } from '../src/site/utils/timetable'

describe('buildWeekMask', () => {
  test('构建位掩码', () => {
    expect(buildWeekMask([1, 2, 3])).toBe(0b111)
  })

  test('忽略非法周次（越界/非整数）', () => {
    expect(buildWeekMask([0, 17, 2])).toBe(1 << 1)
    expect(buildWeekMask([])).toBe(0)
  })
})
describe('weeksOverlap / weekMasksOverlap', () => {
  test('数组交集', () => {
    expect(weeksOverlap([1, 2], [2, 3])).toBe(true)
    expect(weeksOverlap([1], [2])).toBe(false)
    expect(weeksOverlap([], [1])).toBe(false)
  })

  test('掩码交集', () => {
    expect(weekMasksOverlap(buildWeekMask([1, 2]), buildWeekMask([2, 3]))).toBe(true)
    expect(weekMasksOverlap(buildWeekMask([1]), buildWeekMask([2]))).toBe(false)
  })
})

describe('parseArrangeInfoText', () => {
  test('解析标准文本', () => {
    const result = parseArrangeInfoText('1-8周 周一 3-4节 同济楼A201')
    expect(result).toHaveLength(1)
    expect(result[0]).toMatchObject({
      weekStart: 1,
      weekEnd: 8,
      day: 1,
      sectionStart: 3,
      sectionEnd: 4,
      location: '同济楼A201',
      weekParity: null,
      specificWeeks: [],
    })
  })

  test('解析多段（分号分隔）', () => {
    const result = parseArrangeInfoText('1-8周 周一 3-4节 同济楼A201；9-16周 周三 5-6节 线上')
    expect(result).toHaveLength(2)
    expect(result[1]).toMatchObject({ day: 3, sectionStart: 5, sectionEnd: 6, location: '线上' })
  })

  test('解析单双周', () => {
    const result = parseArrangeInfoText('1-8周(单周) 周二 1-2节 教室A')
    expect(result[0].weekParity).toBe('odd')
    expect(result[0].day).toBe(2)
  })

  test('解析枚举周次', () => {
    const result = parseArrangeInfoText('1、3、5周 周四 7-8节 教室B')
    expect(result[0].specificWeeks).toEqual([1, 3, 5])
  })

  test('无星期信息段忽略', () => {
    const result = parseArrangeInfoText('1-8周 3-4节 无星期段')
    expect(result).toHaveLength(0)
  })

  test('空文本返回空数组', () => {
    expect(parseArrangeInfoText('')).toEqual([])
    expect(parseArrangeInfoText('   ')).toEqual([])
  })
})

describe('maxRowsForCalendar / getRowSection', () => {
  test('新 11 节制（calendarId >= 120）', () => {
    expect(maxRowsForCalendar(120)).toBe(11)
    expect(maxRowsForCalendar(121)).toBe(11)
  })

  test('旧 12 节制', () => {
    expect(maxRowsForCalendar(119)).toBe(12)
    expect(maxRowsForCalendar(undefined)).toBe(12)
  })

  test('节段映射（旧 12 节制，权威实现为 utils/timetable）', () => {
    expect(getRowSection(1, 119)).toBe(1)
    expect(getRowSection(3, 119)).toBe(2)
    expect(getRowSection(9, 119)).toBe(5)
    expect(getRowSection(10, 119)).toBe(5)
    expect(getRowSection(11, 119)).toBe(6)
    expect(getRowSection(12, 119)).toBe(6)
  })

  test('节段映射（新 11 节制）', () => {
    expect(getRowSection(9, 121)).toBe(5)
    expect(getRowSection(10, 121)).toBe(5)
    expect(getRowSection(11, 121)).toBe(6)
  })
})

describe('detectWeekParity', () => {
  test('纯单周（奇数）返回 odd', () => {
    expect(detectWeekParity([1, 3, 5, 7, 9, 11, 13, 15])).toBe('odd')
    expect(detectWeekParity([3])).toBe('odd')
  })

  test('纯双周（偶数）返回 even', () => {
    expect(detectWeekParity([2, 4, 6, 8, 10, 12, 14, 16])).toBe('even')
    expect(detectWeekParity([2])).toBe('even')
  })

  test('全周或混合周次返回 null', () => {
    expect(detectWeekParity([1, 2, 3, 4, 5, 6, 7, 8])).toBeNull()
    expect(detectWeekParity([1, 2])).toBeNull()
    expect(detectWeekParity([])).toBeNull()
    expect(detectWeekParity(undefined)).toBeNull()
  })
})

describe('consolidateSameClassArrangements', () => {
  test('合并同一教学班在同节次的多段周次安排', () => {
    // 模拟「现代分析测试技术 12117901」在周一 5-6 节包含的 7 段排课
    const segments = [
      { code: '12117901', courseName: '现代分析测试技术', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [1, 2, 16], occupyRoom: '南219', teacherAndCode: '黄湘通(T01)' },
      { code: '12117901', courseName: '现代分析测试技术', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [3, 4], occupyRoom: '南219', teacherAndCode: '乔培军(T02)' },
      { code: '12117901', courseName: '现代分析测试技术', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [5, 6], occupyRoom: '南219', teacherAndCode: '江小英(T03)' },
      { code: '12117901', courseName: '现代分析测试技术', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [7], occupyRoom: '南219', teacherAndCode: '张灵敏(T04)' },
      { code: '12117901', courseName: '现代分析测试技术', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [8, 9, 10, 15], occupyRoom: '南219', teacherAndCode: '李艳丽(T05)' },
      { code: '12117901', courseName: '现代分析测试技术', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [12, 13], occupyRoom: '南219', teacherAndCode: '谌微微(T06)' },
      { code: '12117901', courseName: '现代分析测试技术', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [11], occupyRoom: '南219', teacherAndCode: '徐娟(T07)' },
    ]

    const consolidated = consolidateSameClassArrangements(segments)
    expect(consolidated).toHaveLength(1)
    const item = consolidated[0]
    expect(item.code).toBe('12117901')
    expect(item.courseName).toBe('现代分析测试技术')
    expect(item.occupyRoom).toBe('南219')
    // 周次合并为 1-13,15-16
    expect(item.occupyWeek).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 16])
    expect(item.arrangementText).toContain('[1-13,15-16周]')
    expect(item.arrangementText).toContain('周1 第5-6节 南219')
    // 教师聚合去重
    expect(item.teacherAndCode).toContain('黄湘通')
    expect(item.teacherAndCode).toContain('乔培军')
  })

  test('不同教学班（不同 code）不予合并，保持独立展示', () => {
    const courses = [
      { code: '10001', courseName: '高等数学', occupyDay: 1, occupyTime: [1, 2], occupyWeek: [1, 3, 5], occupyRoom: 'A101', teacherAndCode: '张三' },
      { code: '10002', courseName: '线性代数', occupyDay: 1, occupyTime: [1, 2], occupyWeek: [2, 4, 6], occupyRoom: 'B202', teacherAndCode: '李四' },
    ]

    const consolidated = consolidateSameClassArrangements(courses)
    expect(consolidated).toHaveLength(2)
    expect(consolidated[0].code).toBe('10001')
    expect(consolidated[1].code).toBe('10002')
  })

  test('不同教室合并用 / 拼接', () => {
    const courses = [
      { code: '10001', courseName: '物理实验', occupyDay: 2, occupyTime: [3, 4], occupyWeek: [1, 2], occupyRoom: '物化楼101' },
      { code: '10001', courseName: '物理实验', occupyDay: 2, occupyTime: [3, 4], occupyWeek: [3, 4], occupyRoom: '物化楼202' },
    ]

    const consolidated = consolidateSameClassArrangements(courses)
    expect(consolidated).toHaveLength(1)
    expect(consolidated[0].occupyRoom).toBe('物化楼101 / 物化楼202')
  })
})

import { describe, expect, test } from 'vitest'
import {
  buildWeekMask,
  getRowSection,
  maxRowsForCalendar,
  parseArrangeInfoText,
  weekMasksOverlap,
  weeksOverlap,
} from '../src/site/utils/pkArrange'

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

  test('节段映射', () => {
    expect(getRowSection(1, 119)).toBe(1)
    expect(getRowSection(3, 119)).toBe(2)
    expect(getRowSection(9, 119)).toBe(5)
    expect(getRowSection(11, 119)).toBe(6)
  })
})

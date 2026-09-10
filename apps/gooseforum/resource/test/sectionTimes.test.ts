import { describe, expect, test } from 'vitest'
import {
  DEFAULT_SECTION_TIMES_11,
  DEFAULT_SECTION_TIMES_12,
  dayPartBoundaries,
  dayPartOfStart,
  parseHHMM,
  sectionTimesFor,
  type SectionTime,
} from '../src/site/utils/sectionTimes'

describe('sectionTimes 作息表', () => {
  test('默认 12 节表锚点：3=10:00、5=13:30、7=15:30、10=18:30', () => {
    expect(DEFAULT_SECTION_TIMES_12).toHaveLength(12)
    const bySection = new Map(DEFAULT_SECTION_TIMES_12.map((item) => [item.section, item.start]))
    expect(bySection.get(3)).toBe('10:00')
    expect(bySection.get(5)).toBe('13:30')
    expect(bySection.get(7)).toBe('15:30')
    expect(bySection.get(10)).toBe('18:30')
  })

  test('默认表节次连续且时间递增', () => {
    const times = DEFAULT_SECTION_TIMES_12
    for (let i = 0; i < times.length; i++) {
      expect(times[i].section).toBe(i + 1)
      if (i > 0) {
        expect(parseHHMM(times[i].start)!).toBeGreaterThan(parseHHMM(times[i - 1].end)!)
      }
      expect(parseHHMM(times[i].end)!).toBeGreaterThan(parseHHMM(times[i].start)!)
    }
  })

  test('11 节新制默认表：白天 1-8 节沿用，晚间 9/10/11 = 18:30/19:20/20:10', () => {
    expect(DEFAULT_SECTION_TIMES_11).toHaveLength(11)
    const bySection = new Map(DEFAULT_SECTION_TIMES_11.map((item) => [item.section, item]))
    // 2025-2026 学年起作息由 12 节调整为 11 节：取消 17:10-17:55 节次，晚间自第 9 节 18:30 起。
    expect(bySection.get(9)).toEqual({ section: 9, start: '18:30', end: '19:15' })
    expect(bySection.get(10)).toEqual({ section: 10, start: '19:20', end: '20:05' })
    expect(bySection.get(11)).toEqual({ section: 11, start: '20:10', end: '20:55' })
    for (let section = 1; section <= 8; section++) {
      expect(bySection.get(section)).toEqual(DEFAULT_SECTION_TIMES_12[section - 1])
    }
  })

  test('sectionTimesFor：11 节制用 11 节表，12/0 节制用历史 12 节表', () => {
    expect(sectionTimesFor(11)).toEqual(DEFAULT_SECTION_TIMES_11)
    expect(sectionTimesFor(12)).toEqual(DEFAULT_SECTION_TIMES_12)
    expect(sectionTimesFor(0)).toEqual(DEFAULT_SECTION_TIMES_12)
  })

  test('11 节制视图合并后台覆盖（晚间覆盖第 9 节），12 节制历史视图忽略覆盖', () => {
    const overrides: SectionTime[] = [
      { section: 1, start: '08:30', end: '09:15' },
      { section: 5, start: '13:00', end: '13:45' },
      { section: 9, start: '19:00', end: '19:45' },
      { section: 12, start: '21:00', end: '21:45' }, // 遗留第 12 行：11 节制视图忽略
    ]
    const merged = sectionTimesFor(11, overrides)
    expect(merged[0]).toEqual(overrides[0])
    expect(merged[4]).toEqual(overrides[1])
    expect(merged[8]).toEqual(overrides[2])
    expect(merged[1]).toEqual(DEFAULT_SECTION_TIMES_11[1])
    expect(merged).toHaveLength(11)
    // 历史学期（12 节制）始终用内置历史作息，不受管理端配置影响。
    expect(sectionTimesFor(12, overrides)).toEqual(DEFAULT_SECTION_TIMES_12)
  })

  test('空覆盖回退默认', () => {
    expect(sectionTimesFor(11, [])).toEqual(DEFAULT_SECTION_TIMES_11)
    expect(sectionTimesFor(12, null)).toEqual(DEFAULT_SECTION_TIMES_12)
    expect(sectionTimesFor(12, undefined)).toEqual(DEFAULT_SECTION_TIMES_12)
  })

  test('时段分组推导', () => {
    expect(dayPartOfStart('08:00')).toBe('morning')
    expect(dayPartOfStart('11:59')).toBe('morning')
    expect(dayPartOfStart('13:30')).toBe('afternoon')
    expect(dayPartOfStart('17:59')).toBe('afternoon')
    expect(dayPartOfStart('18:30')).toBe('evening')
    expect(dayPartOfStart('bad')).toBe('morning')
  })

  test('分组切分点：12 节制 1=上午、5=下午、10=晚上；11 节制晚上从第 9 节起', () => {
    const boundaries12 = dayPartBoundaries(DEFAULT_SECTION_TIMES_12)
    expect(boundaries12.morning).toBe(1)
    expect(boundaries12.afternoon).toBe(5)
    expect(boundaries12.evening).toBe(10)
    const boundaries11 = dayPartBoundaries(DEFAULT_SECTION_TIMES_11)
    expect(boundaries11.morning).toBe(1)
    expect(boundaries11.afternoon).toBe(5)
    expect(boundaries11.evening).toBe(9)
  })

  test('parseHHMM 容错', () => {
    expect(parseHHMM('08:00')).toBe(480)
    expect(parseHHMM('23:59')).toBe(1439)
    expect(parseHHMM('24:00')).toBeNull()
    expect(parseHHMM('12:60')).toBeNull()
    expect(parseHHMM('')).toBeNull()
    expect(parseHHMM('abc')).toBeNull()
  })
})

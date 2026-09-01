import { describe, expect, test } from 'vitest'
import {
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

  test('11 节制 = 12 节制前 11 节；12 节默认表', () => {
    expect(sectionTimesFor(11)).toHaveLength(11)
    expect(sectionTimesFor(12)).toHaveLength(12)
    expect(sectionTimesFor(0)).toHaveLength(12)
    expect(sectionTimesFor(11)[10]).toEqual(DEFAULT_SECTION_TIMES_12[10])
  })

  test('后台覆盖按 section 合并，缺失项回退默认', () => {
    const overrides: SectionTime[] = [
      { section: 1, start: '08:30', end: '09:15' },
      { section: 5, start: '13:00', end: '13:45' },
    ]
    const merged = sectionTimesFor(12, overrides)
    expect(merged[0]).toEqual(overrides[0])
    expect(merged[4]).toEqual(overrides[1])
    expect(merged[1]).toEqual(DEFAULT_SECTION_TIMES_12[1])
    expect(merged).toHaveLength(12)
  })

  test('空覆盖回退默认', () => {
    expect(sectionTimesFor(12, [])).toEqual(DEFAULT_SECTION_TIMES_12)
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

  test('分组切分点：默认表 1=上午、5=下午、10=晚上', () => {
    const boundaries = dayPartBoundaries(DEFAULT_SECTION_TIMES_12)
    expect(boundaries.morning).toBe(1)
    expect(boundaries.afternoon).toBe(5)
    expect(boundaries.evening).toBe(10)
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

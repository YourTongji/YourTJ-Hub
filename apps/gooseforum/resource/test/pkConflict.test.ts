import { describe, expect, test } from 'vitest'
import {
  canAddCourse,
  createEmptyOccupied,
  deleteOccupied,
  findConflicts,
  getCourseBaseCode,
  insertOccupied,
  isClassOfCourse,
  isSameCourse,
  type PkConflictItem,
} from '../src/site/utils/pkConflict'
import type { PkArrangement, PkCourseDetail } from '../src/site/types/pk'

function arr(day: number, time: number[], weeks: number[]): PkArrangement {
  return {
    arrangementText: '',
    occupyDay: day,
    occupyTime: time,
    occupyWeek: weeks,
    occupyRoom: '',
    teacherAndCode: '',
  }
}

function detail(code: string, arrangementInfo: PkArrangement[]): PkCourseDetail {
  return { code, campus: '', teachers: [], teachingLanguage: '', arrangementInfo }
}

/** 生成连续周数组 [start..end]。 */
function makeWeeks(start: number, end: number): number[] {
  return Array.from({ length: end - start + 1 }, (_, i) => start + i)
}

describe('getCourseBaseCode / isSameCourse / isClassOfCourse', () => {
  test('带点号班级课号', () => {
    expect(getCourseBaseCode('122004.01')).toBe('122004')
  })

  test('无点号（后两位为班号）', () => {
    expect(getCourseBaseCode('12200401')).toBe('122004')
  })

  test('短课号原样', () => {
    expect(getCourseBaseCode('A1')).toBe('A1')
  })

  test('同课判断', () => {
    expect(isSameCourse('122004.01', '122004.02')).toBe(true)
    expect(isSameCourse('12200401', '12200402')).toBe(true)
    expect(isSameCourse('122004.01', '122005.01')).toBe(false)
  })

  test('班级归属判断', () => {
    expect(isClassOfCourse('122004.01', '122004')).toBe(true)
    expect(isClassOfCourse('12200401', '122004')).toBe(true)
    expect(isClassOfCourse('122005.01', '122004')).toBe(false)
  })
})

describe('insertOccupied / deleteOccupied', () => {
  test('插入后可在对应格子找到', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3, 4], [1, 2])], '122004.01', '高等数学')
    expect(occupied).toHaveLength(12)
    expect(occupied[0]).toHaveLength(7)
    expect(occupied[2][0]).toHaveLength(1)
    expect(occupied[3][0]).toHaveLength(1)
    expect(occupied[2][0][0]).toMatchObject({ code: '122004.01', courseName: '高等数学', occupyWeek: [1, 2] })
  })

  test('不可变：删除返回新表，原表不变', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], [1, 2])], '122004.01', '高等数学')
    const next = deleteOccupied(occupied, '122004.01')
    expect(next[2][0]).toHaveLength(0)
    expect(occupied[2][0]).toHaveLength(1)
  })

  test('越界时间忽略', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [13], [1])], 'x', '课程')
    // 无越界插入：全部格子为空，且不产生第 13 行。
    expect(occupied).toHaveLength(12)
    expect(occupied.flat().flat()).toHaveLength(0)
  })
})

describe('canAddCourse', () => {
  test('空表可加', () => {
    const occupied = createEmptyOccupied()
    expect(canAddCourse([arr(1, [3], [1, 8])], occupied, '122004.01').canAdd).toBe(true)
  })

  test('同时间同周次冲突', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], makeWeeks(1, 8))], '122004.01', '高数')
    const result = canAddCourse([arr(1, [3], makeWeeks(5, 6))], occupied, '122005.01')
    expect(result.canAdd).toBe(false)
    expect(result.collideCourse).toContain('122004.01')
  })

  test('周次无交集不冲突', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], makeWeeks(1, 4))], '122004.01', '高数')
    expect(canAddCourse([arr(1, [3], makeWeeks(9, 12))], occupied, '122005.01').canAdd).toBe(true)
  })

  test('同课号换班为隐式替换', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], makeWeeks(1, 8))], '122004.01', '高数')
    // 同基础课号、不同班、不同时间 → 可加入（旧班被替换）
    expect(canAddCourse([arr(2, [3], makeWeeks(1, 8))], occupied, '122004.02').canAdd).toBe(true)
  })
})

describe('findConflicts', () => {
  test('列出全部冲突课程（按基础课号去重）', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], [1, 8])], '122004.01', '高数')
    occupied = insertOccupied(occupied, [arr(1, [4], [1, 8])], '122005.01', '英语')
    const candidate = detail('122006.01', [arr(1, [3, 4], [1, 8])])
    const conflicts: PkConflictItem[] = findConflicts(candidate, occupied)
    expect(conflicts).toHaveLength(2)
    expect(conflicts.map((c) => c.code).sort()).toEqual(['122004.01', '122005.01'])
  })

  test('无冲突返回空', () => {
    const occupied = createEmptyOccupied()
    const candidate = detail('122006.01', [arr(1, [3], [1, 8])])
    expect(findConflicts(candidate, occupied)).toEqual([])
  })
})

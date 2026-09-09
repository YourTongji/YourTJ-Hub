import { describe, expect, test } from 'vitest'
import {
  canAddCourse,
  createEmptyOccupied,
  deleteOccupied,
  findClassConflicts,
  findConflicts,
  getCourseBaseCode,
  insertOccupied,
  isArrangementConflicted,
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

describe('findClassConflicts (候选班级冲突预检)', () => {
  test('同时间同周次检出外部课程冲突', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], makeWeeks(1, 8))], '122004.01', '高数')
    const candidate = detail('122005.01', [arr(1, [3], makeWeeks(3, 4))])
    const conflicts = findClassConflicts(candidate, occupied)
    expect(conflicts).toHaveLength(1)
    expect(conflicts[0].code).toBe('122004.01')
    expect(conflicts[0].courseName).toBe('高数')
  })

  test('同门课程换班隐式替换不视为冲突', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], makeWeeks(1, 8))], '122004.01', '高数')
    // 候选为同一门课程的另一个班级 122004.02，即使时段重叠也不应当作外部冲突
    const candidate = detail('122004.02', [arr(1, [3], makeWeeks(1, 8))])
    const conflicts = findClassConflicts(candidate, occupied)
    expect(conflicts).toHaveLength(0)
  })

  test('自定义事件占位符 custom: 正常参与冲突计算且不与同自定义项冲突', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(2, [1, 2], makeWeeks(1, 10))], 'custom:team_meeting', '组会')
    const candidate = detail('122008.01', [arr(2, [1], makeWeeks(2, 4))])
    const conflicts = findClassConflicts(candidate, occupied)
    expect(conflicts).toHaveLength(1)
    expect(conflicts[0].code).toBe('custom:team_meeting')
    expect(conflicts[0].courseName).toBe('组会')
  })
})

describe('isArrangementConflicted (单时段冲突判断)', () => {
  test('重叠时段返回 true，无交集时段返回 false', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3, 4], makeWeeks(1, 8))], '122004.01', '高数')

    const slot1 = arr(1, [3], makeWeeks(2, 4))
    const slot2 = arr(1, [5], makeWeeks(1, 8))
    const slot3 = arr(2, [3], makeWeeks(1, 8))

    expect(isArrangementConflicted(slot1, '122005.01', occupied)).toBe(true)
    expect(isArrangementConflicted(slot2, '122005.01', occupied)).toBe(false)
    expect(isArrangementConflicted(slot3, '122005.01', occupied)).toBe(false)
  })

  test('与自身课程的时段重叠不判定为冲突', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [3], makeWeeks(1, 8))], '122004.01', '高数')
    const selfSlot = arr(1, [3], makeWeeks(1, 8))
    expect(isArrangementConflicted(selfSlot, '122004.02', occupied)).toBe(false)
  })
})

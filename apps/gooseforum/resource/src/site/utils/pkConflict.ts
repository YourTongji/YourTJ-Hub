// 排课器冲突检测：课号归一化、12×7 占用表操作、冲突判定（可强制替换）。
//
// 算法对齐上游 courseManipulate.ts，占用格为 12×7 三维数组（occupyCell[][][]）；
// 冲突判据 = 同一天 + 同一节次 + 周次交集非空。
// 本模块函数均为纯函数：occupied 一律返回新数组，不就地修改入参。

import type { PkArrangement, PkCourseDetail, PkOccupyCell } from '@/site/types/pk'
import { weeksOverlap } from './pkArrange'

/** 占用格行数（12 节）。 */
export const OCCUPY_ROWS = 12
/** 占用格列数（7 天）。 */
export const OCCUPY_COLS = 7

/** 空 12×7 占用表。 */
export function createEmptyOccupied(): PkOccupyCell[][][] {
  return Array.from({ length: OCCUPY_ROWS }, () =>
    Array.from({ length: OCCUPY_COLS }, () => [] as PkOccupyCell[]),
  )
}

/**
 * 从班级课号取基础课号，兼容 "12200401" 和 "122004.01" 两种格式。
 * 上游约定：后两位是班号；带点号时取点前部分。
 */
export function getCourseBaseCode(code: string): string {
  const value = String(code ?? '').trim()
  const dot = value.lastIndexOf('.')
  if (dot > 0) return value.substring(0, dot)
  return value.length > 2 ? value.slice(0, -2) : value
}

/** 判断班级课号是否属于某门课。 */
export function isClassOfCourse(classCode: string, courseCode: string): boolean {
  return getCourseBaseCode(classCode) === String(courseCode ?? '').trim()
}

/** 判断两个班级课号是否属于同一门课。 */
export function isSameCourse(code1: string, code2: string): boolean {
  return getCourseBaseCode(code1) === getCourseBaseCode(code2)
}

/** 从占用表删除一门课（返回新表）。 */
export function deleteOccupied(occupied: PkOccupyCell[][][], code: string): PkOccupyCell[][][] {
  return occupied.map((row) =>
    row.map((cell) => cell.filter((item) => !isSameCourse(item.code, code))),
  )
}

/** 向占用表插入一门课（返回新表）。 */
export function insertOccupied(
  occupied: PkOccupyCell[][][],
  arrangementInfo: PkArrangement[],
  code: string,
  courseName: string,
): PkOccupyCell[][][] {
  const next = occupied.map((row) => row.map((cell) => [...cell]))
  for (const arr of arrangementInfo) {
    for (const time of arr.occupyTime) {
      if (time < 1 || time > OCCUPY_ROWS || arr.occupyDay < 1 || arr.occupyDay > OCCUPY_COLS) continue
      next[time - 1][arr.occupyDay - 1] = [
        ...next[time - 1][arr.occupyDay - 1],
        { code, courseName, occupyWeek: arr.occupyWeek },
      ]
    }
  }
  return next
}

/**
 * 判断一门课能否加入课表。
 * 若占用表中已存在同一基础课号的课程，先移除旧课再判定（同课换班 = 隐式替换，不报冲突）。
 */
export function canAddCourse(
  arrangementInfo: PkArrangement[],
  occupied: PkOccupyCell[][][],
  code: string,
): { canAdd: boolean; collideCourse?: string } {
  const existingCode = occupied
    .flat()
    .flat()
    .find((item) => isSameCourse(item.code, code))?.code

  if (existingCode) {
    return canAddCourse(arrangementInfo, deleteOccupied(occupied, existingCode), code)
  }

  for (const arr of arrangementInfo) {
    for (const time of arr.occupyTime) {
      const cell = occupied[time - 1]?.[arr.occupyDay - 1]
      if (!cell) continue
      const collideItem = cell.find((item) => weeksOverlap(arr.occupyWeek, item.occupyWeek))
      if (collideItem) {
        return { canAdd: false, collideCourse: `${collideItem.code} ${collideItem.courseName}` }
      }
    }
  }
  return { canAdd: true }
}

/** 与候选课程冲突的已占课程（供「强制替换/放弃」弹窗展示）。 */
export interface PkConflictItem {
  /** 班级课号 */
  code: string
  courseName: string
}

/**
 * 找出候选课程与占用表的所有冲突课程（按基础课号去重）。
 * 与 canAddCourse 不同：这里列举全部冲突项而非只报第一个。
 */
export function findConflicts(
  candidate: PkCourseDetail,
  occupied: PkOccupyCell[][][],
): PkConflictItem[] {
  const conflicts = new Map<string, PkConflictItem>()
  for (const arr of candidate.arrangementInfo) {
    for (const time of arr.occupyTime) {
      const cell = occupied[time - 1]?.[arr.occupyDay - 1]
      if (!cell) continue
      for (const item of cell) {
        if (weeksOverlap(arr.occupyWeek, item.occupyWeek)) {
          const base = getCourseBaseCode(item.code)
          conflicts.set(base, { code: item.code, courseName: item.courseName })
        }
      }
    }
  }
  return [...conflicts.values()]
}

/** 自定义占位事件在占用表中的伪课号前缀（避免与真实课号碰撞）。 */
export const CUSTOM_EVENT_CODE_PREFIX = 'custom:'

/** 冲突派生用的基础标识：custom 伪课号原样保留（getCourseBaseCode 会误裁尾部字符）。 */
function conflictBaseOf(code: string): string {
  return code.startsWith(CUSTOM_EVENT_CODE_PREFIX) ? code : getCourseBaseCode(code)
}

/**
 * 从占用表派生当前课表的全部冲突（容忍式冲突模型）：
 * 同一格子（天+节次）内周次有交集的两个不同基础课号互为冲突。
 * 返回 Map<基础课号, 冲突项[]>（含 custom: 占位事件；供课表 ⚠、列表红标、
 * 统计计数共用同一判据，与 canAddCourse/findConflicts 一致）。
 */
export function deriveConflicts(occupied: PkOccupyCell[][][]): Map<string, PkConflictItem[]> {
  const result = new Map<string, PkConflictItem[]>()
  const pushConflict = (base: string, item: PkConflictItem) => {
    const list = result.get(base)
    if (list) {
      if (!list.some((existing) => existing.code === item.code)) list.push(item)
    } else {
      result.set(base, [item])
    }
  }
  for (const row of occupied) {
    for (const cell of row) {
      if (cell.length < 2) continue
      for (let i = 0; i < cell.length; i++) {
        for (let j = i + 1; j < cell.length; j++) {
          const a = cell[i]
          const b = cell[j]
          if (!weeksOverlap(a.occupyWeek, b.occupyWeek)) continue
          if (conflictBaseOf(a.code) === conflictBaseOf(b.code)) continue // 同课换班/同事件不标冲突
          pushConflict(conflictBaseOf(a.code), { code: b.code, courseName: b.courseName })
          pushConflict(conflictBaseOf(b.code), { code: a.code, courseName: a.courseName })
        }
      }
    }
  }
  return result
}

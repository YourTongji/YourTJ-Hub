import { isClassOfCourse } from '@/site/utils/pkConflict'
import type { PkPlan, PkStagedCourse, PkCustomEvent } from '@/site/types/pk'

export interface PlanMergeConflict {
  path: string[]
  local: unknown
  remote: unknown
}
export interface PlanMergeResult {
  plan: PkPlan | null
  conflicts: PlanMergeConflict[]
}

// Treat optional null wire fields like omitted local fields; object insertion order
// and set order are not edits. Courses form one selection unit, events merge by field.
export function planValueKey(value: unknown): string {
  const normalize = (v: unknown): unknown => {
    if (Array.isArray(v)) return v.map(normalize)
    if (v && typeof v === 'object')
      return Object.fromEntries(
        Object.entries(v)
          .filter(([, x]) => x != null)
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([k, x]) => [k, normalize(x)]),
      )
    return v ?? null
  }
  return JSON.stringify(normalize(value))
}
const equal = (a: unknown, b: unknown) => planValueKey(a) === planValueKey(b)
const record = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === 'object' && !Array.isArray(value)
function project(plan: PkPlan): Record<string, unknown> {
  const courses: Record<string, unknown> = Object.create(null)
  for (const course of plan.stagedCourses) {
    courses[course.courseCode] = {
      course: { ...course, courseNature: course.courseNature ?? [] },
      selected: plan.selectedCourses
        .filter((code) => isClassOfCourse(code, course.courseCode))
        .sort(),
    }
  }
  return {
    id: plan.id,
    name: plan.name,
    createdAt: plan.createdAt,
    courses,
    otherSelected: plan.selectedCourses
      .filter((code) => !plan.stagedCourses.some((c) => isClassOfCourse(code, c.courseCode)))
      .sort(),
    events: Object.fromEntries(plan.customEvents.map((event) => [event.id, event])),
  }
}
function restore(value: Record<string, unknown>): PkPlan {
  const courses = Object.values(
    value.courses as Record<string, { course: PkStagedCourse; selected: string[] }>,
  )
  return {
    id: value.id as string,
    name: value.name as string,
    createdAt: value.createdAt as number,
    stagedCourses: courses.map((c) => c.course),
    selectedCourses: [...courses.flatMap((c) => c.selected), ...(value.otherSelected as string[])],
    customEvents: Object.values(value.events as Record<string, PkCustomEvent>),
  }
}

/** Missing base is an ID collision, never permission to invent a common ancestor. */
export function mergeSchedulePlan(
  base: PkPlan | null,
  local: PkPlan | null,
  remote: PkPlan | null,
  choices: Record<string, 'local' | 'remote'> = {},
): PlanMergeResult {
  const conflicts: PlanMergeConflict[] = []
  const merge = (b: unknown, l: unknown, r: unknown, path: string[]): unknown => {
    if (equal(l, r)) return l
    if (equal(l, b)) return r
    if (equal(r, b)) return l
    if (record(b) && record(l) && record(r) && !(path[0] === 'courses' && path.length === 2)) {
      const result: Record<string, unknown> = Object.create(null)
      for (const key of new Set([...Object.keys(b), ...Object.keys(l), ...Object.keys(r)])) {
        const value = merge(b[key], l[key], r[key], [...path, key])
        if (value != null) result[key] = value
      }
      return result
    }
    const choice = choices[JSON.stringify(path)]
    if (choice) return choice === 'local' ? l : r
    conflicts.push({ path, local: l ?? null, remote: r ?? null })
    return l
  }
  // The collection nodes may be empty; they still share a known ancestor.
  const result = merge(
    base && project(base),
    local && project(local),
    remote && project(remote),
    [],
  )
  return { plan: record(result) ? restore(result) : null, conflicts }
}
export function schedulePlanKey(plan: PkPlan | null): string {
  return planValueKey(plan && project(plan))
}

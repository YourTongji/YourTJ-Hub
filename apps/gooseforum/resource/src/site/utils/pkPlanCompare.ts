// 排课方案对比（纯函数，无 Vue 依赖）：把多套方案压成「课程 × 方案」「占位 × 方案」
// 两张对照表并标注差异行，供 SchedulePlanCompareDialog 渲染。
//
// 口径：课程行只纳入至少在一套方案中已排入课表（status=待选/已选）的基础课号；
// 单元格是该方案排入的班级课号（升序），空数组表示该方案未排此课。占位行按
// (label, day, sections, weeks) 签名取并集，布尔表示各方案是否含该占位。
// 一行只要各方案签名不完全一致即视为差异。

import { COURSE_STATUS } from '@/site/composables/useScheduleStore'
import type { PkCustomEvent, PkPlan } from '@/site/types/pk'

export interface PlanCourseRow {
  courseCode: string
  courseName: string
  /** 各方案已排入课表的班级课号（升序）；空数组 = 该方案未排此课。 */
  classesByPlan: string[][]
  different: boolean
}

export interface PlanPlaceholderRow {
  signature: string
  label: string
  day: number
  sections: number[]
  weeks: number[]
  presentByPlan: boolean[]
  different: boolean
}

export interface PlanComparison {
  courses: PlanCourseRow[]
  placeholders: PlanPlaceholderRow[]
  /** 差异行数（课程行 + 占位行）。 */
  diffCount: number
}

function isScheduled(status: number | undefined): boolean {
  return status === COURSE_STATUS.STAGED || status === COURSE_STATUS.SELECTED
}

function scheduledClasses(plan: PkPlan, baseCode: string): { name: string; classes: string[] } | null {
  const course = plan.stagedCourses.find((item) => item.courseCode === baseCode)
  if (!course) return null
  const classes = course.courseDetail
    .filter((detail) => isScheduled(detail.status))
    .map((detail) => detail.code)
    .sort()
  return { name: course.courseNameReserved || course.courseName, classes }
}

function placeholderSignature(event: PkCustomEvent): string {
  return [
    event.label,
    event.day,
    [...event.sections].sort((a, b) => a - b).join(','),
    [...event.weeks].sort((a, b) => a - b).join(','),
  ].join('|')
}

export function comparePlans(plans: PkPlan[]): PlanComparison {
  const courseCodes = new Set<string>()
  for (const plan of plans) {
    for (const course of plan.stagedCourses) {
      if (course.courseDetail.some((detail) => isScheduled(detail.status))) {
        courseCodes.add(course.courseCode)
      }
    }
  }

  const courses: PlanCourseRow[] = [...courseCodes].sort().map((courseCode) => {
    const perPlan = plans.map((plan) => scheduledClasses(plan, courseCode))
    const classesByPlan = perPlan.map((entry) => entry?.classes ?? [])
    return {
      courseCode,
      courseName: perPlan.find((entry) => entry)?.name ?? courseCode,
      classesByPlan,
      different: new Set(classesByPlan.map((classes) => classes.join(','))).size > 1,
    }
  })

  const placeholders: PlanPlaceholderRow[] = []
  const seen = new Set<string>()
  for (const plan of plans) {
    for (const event of plan.customEvents) {
      const signature = placeholderSignature(event)
      if (seen.has(signature)) continue
      seen.add(signature)
      const presentByPlan = plans.map((item) =>
        item.customEvents.some((candidate) => placeholderSignature(candidate) === signature),
      )
      placeholders.push({
        signature,
        label: event.label,
        day: event.day,
        sections: [...event.sections],
        weeks: [...event.weeks],
        presentByPlan,
        different: new Set(presentByPlan).size > 1,
      })
    }
  }

  return {
    courses,
    placeholders,
    diffCount:
      courses.filter((row) => row.different).length +
      placeholders.filter((row) => row.different).length,
  }
}

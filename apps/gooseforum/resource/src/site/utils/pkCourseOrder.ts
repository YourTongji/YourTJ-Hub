type CourseCodeItem = { courseCode: string }

/** 稳定地将满足计划条件的项目置顶，保留两组内的原始顺序。 */
export function sortPlannedFirst<T>(items: readonly T[], isPlanned: (item: T) => boolean): T[] {
  const planned: T[] = []
  const other: T[] = []

  for (const item of items) {
    const bucket = isPlanned(item) ? planned : other
    bucket.push(item)
  }

  return [...planned, ...other]
}

/** 将专业计划内课程置顶，同时保留两组内的接口/备选池顺序。 */
export function sortPlannedCoursesFirst<T extends CourseCodeItem>(
  courses: readonly T[],
  compulsoryCourses: readonly CourseCodeItem[],
): T[] {
  const plannedCodes = new Set(compulsoryCourses.map((course) => course.courseCode))
  return sortPlannedFirst(courses, (course) => plannedCodes.has(course.courseCode))
}

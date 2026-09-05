type CourseCodeItem = { courseCode: string }

/** 将专业计划内课程置顶，同时保留两组内的接口/备选池顺序。 */
export function sortPlannedCoursesFirst<T extends CourseCodeItem>(
  courses: readonly T[],
  compulsoryCourses: readonly CourseCodeItem[],
): T[] {
  const plannedCodes = new Set(compulsoryCourses.map((course) => course.courseCode))
  const planned: T[] = []
  const other: T[] = []

  for (const course of courses) {
    const bucket = plannedCodes.has(course.courseCode) ? planned : other
    bucket.push(course)
  }

  return [...planned, ...other]
}

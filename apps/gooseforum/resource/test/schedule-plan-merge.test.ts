import { describe, expect, it } from 'vitest'
import { mergeSchedulePlan, schedulePlanKey } from '../src/site/composables/schedule-plan-merge'
import type { PkPlan, PkStagedCourse } from '../src/site/types/pk'
const plan = (): PkPlan => ({
  id: 'p',
  name: 'Plan',
  createdAt: 1,
  stagedCourses: [],
  selectedCourses: [],
  customEvents: [],
})
const course = (id: string, cls: string): PkStagedCourse => ({
  courseCode: id,
  courseName: id,
  courseNameReserved: id,
  credit: 1,
  courseType: '',
  courseNature: [],
  teacher: [],
  status: 2,
  courseDetail: [
    {
      code: `${id}.${cls}`,
      campus: '',
      arrangementInfo: [],
      teachers: [],
      teachingLanguage: '',
      status: 2,
    },
  ],
})
describe('per-plan three-way merge', () => {
  it('combines edits to different courses without cloning plans', () => {
    const b = plan(),
      l = plan(),
      r = plan()
    l.stagedCourses = [course('a', '1')]
    l.selectedCourses = ['a.1']
    r.stagedCourses = [course('b', '1')]
    r.selectedCourses = ['b.1']
    const result = mergeSchedulePlan(b, l, r)
    expect(result.conflicts).toEqual([])
    expect(result.plan?.selectedCourses).toEqual(['a.1', 'b.1'])
  })
  it('merges different event fields but isolates a conflicting field', () => {
    const b = plan()
    b.customEvents = [{ id: 'e', label: 'Original', day: 1, sections: [1], weeks: [1] }]
    const l = structuredClone(b),
      r = structuredClone(b)
    l.customEvents[0].label = 'Local'
    r.customEvents[0].day = 2
    expect(mergeSchedulePlan(b, l, r).plan?.customEvents[0]).toMatchObject({
      label: 'Local',
      day: 2,
    })
    r.customEvents[0].label = 'Remote'
    const result = mergeSchedulePlan(b, l, r)
    expect(result.conflicts.map((c) => c.path)).toEqual([['events', 'e', 'label']])
    const resolved = mergeSchedulePlan(b, l, r, { '["events","e","label"]': 'remote' })
    expect(resolved.conflicts).toEqual([])
    expect(resolved.plan?.customEvents[0]).toMatchObject({ label: 'Remote', day: 2 })
  })
  it('does not combine two alternative classes of the same course', () => {
    const b = plan()
    b.stagedCourses = [course('a', '1')]
    b.selectedCourses = ['a.1']
    const l = structuredClone(b),
      r = structuredClone(b)
    l.stagedCourses = [course('a', '2')]
    l.selectedCourses = ['a.2']
    r.stagedCourses = [course('a', '3')]
    r.selectedCourses = ['a.3']
    expect(mergeSchedulePlan(b, l, r).conflicts.map((c) => c.path)).toEqual([['courses', 'a']])
  })
  it('retains delete/edit conflicts and unknown-base collisions for explicit choice', () => {
    const b = plan(),
      l = { ...plan(), name: 'Edited' }
    expect(mergeSchedulePlan(b, l, null).conflicts.map((c) => c.path)).toEqual([[]])
    expect(mergeSchedulePlan(null, l, b).conflicts).toHaveLength(1)
    expect(mergeSchedulePlan(b, b, null)).toEqual({ plan: null, conflicts: [] })
  })
  it('ignores null optional wire keys and selection ordering', () => {
    const b = plan()
    b.stagedCourses = [course('a', '1')]
    const r = structuredClone(b)
    Object.assign(r.stagedCourses[0].courseDetail[0], { isExclusive: null, teachingClassId: null })
    expect(schedulePlanKey(b)).toEqual(schedulePlanKey(r))
  })
})

import { readFileSync } from 'node:fs'
const cases = JSON.parse(
  readFileSync(
    new URL('../../../../packages/api-contract/fixtures/pk-plan-merge-cases.json', import.meta.url),
    'utf8',
  ),
) as {
  name: string
  base: PkPlan | null
  local: PkPlan | null
  remote: PkPlan | null
  expected: PkPlan | null
  conflicts: string[][]
  choices: Record<string, 'local' | 'remote'>
}[]
for (const fixture of cases)
  it(`shared merge fixture: ${fixture.name}`, () => {
    const result = mergeSchedulePlan(fixture.base, fixture.local, fixture.remote, fixture.choices)
    expect(schedulePlanKey(result.plan)).toBe(schedulePlanKey(fixture.expected))
    expect(result.conflicts.map((c) => c.path)).toEqual(fixture.conflicts)
  })

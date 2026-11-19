import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';

// Exercise the same dependency-free model used by the ArkTS card extension.
const source = await readFile(new URL('../entry/src/main/ets/widget/ScheduleData.ets', import.meta.url), 'utf8');
const code = stripTypeScriptTypes(source, { mode: 'strip', disableExperimentalWarning: true });
const { parseSchedule, schoolDate, nextClass } = await import(`data:text/javascript;base64,${Buffer.from(code).toString('base64')}`);
const course = {
  stableId: 'course-1', name: '课程', teacher: '', room: '', campus: '',
  startAt: '2026-11-06T08:00:00+08:00', endAt: '2026-11-06T09:35:00+08:00',
  colorSlot: 1, startSection: 1, endSection: 2,
};
function projection() {
  return {
    schemaVersion: 2, timezone: 'Asia/Shanghai',
    identity: { siteKey: 'site', accountScope: 'account', bindingRevision: 'revision' },
    generatedAt: '2026-11-06T07:00:00+08:00',
    days: [{ date: '2026-11-06', week: 10, kind: 'normal', courses: [{ ...course }] }],
  };
}

test('school date uses Shanghai independently of the host timezone', () => {
  assert.equal(schoolDate(Date.parse('2026-11-05T16:00:00Z')), '2026-11-06');
});

test('course start is inclusive and end is exclusive', () => {
  const value = parseSchedule(JSON.stringify(projection()));
  assert.equal(nextClass(value, Date.parse(course.startAt) - 1).status, 'upcoming');
  assert.equal(nextClass(value, Date.parse(course.startAt)).status, 'inClass');
  assert.equal(nextClass(value, Date.parse(course.endAt)).status, 'needsRefresh');
  assert.equal(nextClass(value, Date.parse('2026-11-14T08:00:00+08:00')).status, 'stale');
});

test('invalid identity, dates, courses and unknown schemas fail closed', () => {
  for (const raw of ['null', '[]', '{}', '{', '"text"']) assert.equal(parseSchedule(raw), null);
  for (const mutate of [
    value => { value.identity.accountScope = ''; },
    value => { value.timezone = 'UTC'; },
    value => { value.schemaVersion = 3; },
    value => { value.days[0].date = '2026-02-30'; },
    value => { value.days[0].courses[0].endAt = course.startAt; },
    value => { value.days[0].courses[0].colorSlot = 9; },
    value => { value.days.push(value.days[0]); },
  ]) {
    const value = projection();
    mutate(value);
    assert.equal(parseSchedule(JSON.stringify(value)), null);
  }
});

test('next class spans days without claiming incomplete data is empty', () => {
  const value = projection();
  value.days[0].courses = [];
  value.days.push({ date: '2026-11-07', week: 10, kind: 'normal', courses: [{
    ...course, startAt: '2026-11-07T08:00:00+08:00', endAt: '2026-11-07T09:35:00+08:00',
  }] });
  const state = nextClass(parseSchedule(JSON.stringify(value)), Date.parse(course.endAt));
  assert.equal(state.status, 'upcoming');
  assert.equal(state.day.date, '2026-11-07');
  assert.equal(nextClass(null, Date.now()).status, 'needsData');
});

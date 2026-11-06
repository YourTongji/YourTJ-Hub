export interface ScheduleCourse {
  stableId: string;
  name: string;
  teacher: string;
  room: string;
  campus: string;
  startAt: string;
  endAt: string;
  colorSlot: number;
  startSection: number;
  endSection: number;
}

export interface ScheduleDay {
  date: string;
  week: number | null;
  kind: string;
  adjustmentLabel?: string;
  courses: ScheduleCourse[];
}

export interface ScheduleIdentity {
  siteKey: string;
  accountScope: string;
  bindingRevision: string;
}

export interface ScheduleProjection {
  schemaVersion: number;
  timezone: string;
  identity: ScheduleIdentity;
  generatedAt: string;
  days: ScheduleDay[];
}

export class ScheduleState {
  status: string;
  course: ScheduleCourse | null;
  day: ScheduleDay | null;

  constructor(
    status: string,
    course: ScheduleCourse | null = null,
    day: ScheduleDay | null = null
  ) {
    this.status = status;
    this.course = course;
    this.day = day;
  }
}

const STALE_AFTER = 7 * 24 * 60 * 60 * 1000;

export function schoolDate(now: number): string {
  return new Date(now + 8 * 60 * 60 * 1000).toISOString().slice(0, 10);
}

function hasText(value: string): boolean {
  return typeof value === 'string' && value.trim().length > 0 &&
    value !== 'null' && value !== 'undefined';
}

function validDate(value: string): boolean {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const parsed = Date.parse(`${value}T00:00:00+08:00`);
  return Number.isFinite(parsed) && schoolDate(parsed) === value;
}

export function parseSchedule(raw: string | null): ScheduleProjection | null {
  if (raw === null || raw.length > 1024 * 1024) return null;
  try {
    const value = JSON.parse(raw) as ScheduleProjection;
    if (value === null || value.schemaVersion !== 2 || value.timezone !== 'Asia/Shanghai' ||
      !value.identity || !hasText(value.identity.siteKey) ||
      !hasText(value.identity.accountScope) || !hasText(value.identity.bindingRevision) ||
      !hasText(value.generatedAt) || !Number.isFinite(Date.parse(value.generatedAt)) ||
      !Array.isArray(value.days) || value.days.length === 0 || value.days.length > 8) return null;
    const dates = new Set<string>();
    for (const day of value.days) {
      if (!day || !validDate(day.date) || dates.has(day.date) || !Array.isArray(day.courses) ||
        !['none', 'normal', 'holiday', 'moved', 'makeup', 'unknown'].includes(day.kind)) return null;
      dates.add(day.date);
      for (const course of day.courses) {
        if (!course || !hasText(course.stableId) || !hasText(course.name) ||
          typeof course.startAt !== 'string' || typeof course.endAt !== 'string' ||
          !Number.isFinite(Date.parse(course.startAt)) ||
          !Number.isFinite(Date.parse(course.endAt)) ||
          Date.parse(course.endAt) <= Date.parse(course.startAt) ||
          !Number.isInteger(course.colorSlot) || course.colorSlot < 1 || course.colorSlot > 8 ||
          !Number.isInteger(course.startSection) || course.startSection < 1 ||
          !Number.isInteger(course.endSection) || course.endSection < course.startSection) return null;
      }
      day.courses.sort((a: ScheduleCourse, b: ScheduleCourse) => Date.parse(a.startAt) - Date.parse(b.startAt));
    }
    value.days.sort((a: ScheduleDay, b: ScheduleDay) => a.date.localeCompare(b.date));
    return value;
  } catch (_) {
    return null;
  }
}

export function nextClass(value: ScheduleProjection | null, now: number): ScheduleState {
  if (value === null) return new ScheduleState('needsData');
  if (now - Date.parse(value.generatedAt) > STALE_AFTER) return new ScheduleState('stale');
  const date = schoolDate(now);
  const today = value.days.find((day: ScheduleDay) => day.date === date);
  if (!today || today.kind === 'unknown') return new ScheduleState('needsRefresh');
  const current = today.courses.find((course: ScheduleCourse) =>
    Date.parse(course.startAt) <= now && now < Date.parse(course.endAt));
  if (current) return new ScheduleState('inClass', current, today);
  let next: ScheduleCourse | null = null;
  let nextDay: ScheduleDay | null = null;
  for (const day of value.days) {
    if (day.date < date) continue;
    for (const course of day.courses) {
      if (Date.parse(course.startAt) > now &&
        (next === null || Date.parse(course.startAt) < Date.parse(next.startAt))) {
        next = course;
        nextDay = day;
      }
    }
  }
  if (next) return new ScheduleState('upcoming', next, nextDay);
  if (value.days.length < 8 || value.days.some((day: ScheduleDay) =>
    day.date >= date && day.kind === 'unknown')) return new ScheduleState('needsRefresh');
  return new ScheduleState('noneUpcoming');
}

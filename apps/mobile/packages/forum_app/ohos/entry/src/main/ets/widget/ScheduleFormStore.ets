import { common } from '@kit.AbilityKit';
import { preferences } from '@kit.ArkData';
import { formBindingData, formProvider } from '@kit.FormKit';
import { ScheduleCourse, nextClass, parseSchedule, schoolDate } from './ScheduleData';

export const WIDGET_STORE = 'yourtj_schedule_widget';

export class ScheduleCardData {
  kind: string = 'nextClass';
  showTomorrow: boolean = false;
  status: string = 'needsData';
  courseName: string = '';
  courseTime: string = '';
  courseLocation: string = '';
  courseTeacher: string = '';
  date: string = '';
  tomorrowDate: string = '';
  week: number = 0;
  generatedAt: string = '';
  todayCourses: ScheduleCourse[] = [];
  tomorrowCourses: ScheduleCourse[] = [];
  todayKnown: boolean = false;
  tomorrowKnown: boolean = false;
  transparency: number = 9;
}

export function widgetPreferences(context: common.Context): preferences.Preferences {
  if (!preferences.isStorageTypeSupported(preferences.StorageType.GSKV)) {
    throw new Error('Cross-process widget storage is unavailable');
  }
  // GSKV writes are synchronous and visible to the separate form process.
  return preferences.getPreferencesSync(context, {
    name: WIDGET_STORE,
    storageType: preferences.StorageType.GSKV
  });
}

export function cardData(context: common.Context, kind: string, id: string = '', now: number = Date.now()): ScheduleCardData {
  const store = widgetPreferences(context);
  const projection = parseSchedule(store.getSync('schedule_widget_projection', '') as string);
  const data = new ScheduleCardData();
  data.kind = kind;
  data.showTomorrow = kind === 'timeline' && id !== '' && (store.getSync(`form_tomorrow_${id}`, false) as boolean);
  data.date = schoolDate(now);
  data.tomorrowDate = schoolDate(now + 24 * 60 * 60 * 1000);
  data.transparency = Math.min(15, Math.max(0,
    store.getSync('schedule_widget_transparency_percent', 9) as number));
  const state = nextClass(projection, now);
  data.status = projection === null ? store.getSync('schedule_widget_empty_state', 'needsData') as string : state.status;
  // Expired or cleared data must never remain visible on the desktop.
  if (projection === null || state.status === 'stale') return data;
  data.generatedAt = projection.generatedAt;
  const today = projection.days.find(day => day.date === data.date);
  const tomorrow = projection.days.find(day => day.date === data.tomorrowDate);
  data.todayKnown = today !== undefined && today.kind !== 'unknown';
  data.tomorrowKnown = tomorrow !== undefined && tomorrow.kind !== 'unknown';
  data.week = today?.week ?? 0;
  data.todayCourses = today?.courses ?? [];
  data.tomorrowCourses = tomorrow?.courses ?? [];
  if (state.course) {
    data.courseName = state.course.name;
    data.courseTime = `${state.course.startAt.slice(11, 16)}–${state.course.endAt.slice(11, 16)}`;
    data.courseLocation = [state.course.campus, state.course.room].filter(value => !!value).join(' · ');
    data.courseTeacher = state.course.teacher;
  }
  return data;
}

export async function refreshCards(context: common.Context): Promise<void> {
  const store = widgetPreferences(context);
  const ids = store.getSync('form_ids', []) as string[];
  await Promise.all(ids.map(async (id: string) => {
    const kind = store.getSync(`form_kind_${id}`, 'nextClass') as string;
    await formProvider.updateForm(id, formBindingData.createFormBindingData(cardData(context, kind, id)));
    await formProvider.setFormNextRefreshTime(id, 5);
  }));
}

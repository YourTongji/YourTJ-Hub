import { describe, it, expect } from '@ohos/hypium';
import { common } from '@kit.AbilityKit';
import { ScheduleProjection } from '../../../main/ets/widget/ScheduleData';
import { cardData, widgetPreferences } from '../../../main/ets/widget/ScheduleFormStore';

// Run only on the isolated test device; restore every shared value after the check.
export default function scheduleFormStoreTest(context: common.UIAbilityContext): void {
  describe('ScheduleFormStore', () => {
    it('distinguishesUnknownDaysAndClearsPrivateCourses', 0, () => {
      const store = widgetPreferences(context);
      const projectionKey = 'schedule_widget_projection';
      const emptyKey = 'schedule_widget_empty_state';
      const toggleKey = 'form_tomorrow_test';
      const oldProjection = store.getSync(projectionKey, null);
      const oldEmpty = store.getSync(emptyKey, null);
      const oldToggle = store.getSync(toggleKey, null);
      const now = Date.parse('2026-11-07T08:00:00+08:00');
      try {
        const projection: ScheduleProjection = {
          schemaVersion: 2, timezone: 'Asia/Shanghai',
          identity: { siteKey: 'test', accountScope: 'test', bindingRevision: 'test' },
          generatedAt: '2026-11-07T08:00:00+08:00',
          days: [{ date: '2026-11-08', week: 10, kind: 'normal', courses: [{
            stableId: 'test-course', name: 'Sample course', teacher: '', room: '', campus: '',
            startAt: '2026-11-08T09:00:00+08:00', endAt: '2026-11-08T10:00:00+08:00',
            colorSlot: 1, startSection: 1, endSection: 2
          }] }]
        };
        store.putSync(projectionKey, JSON.stringify(projection));
        store.putSync(toggleKey, true);
        const data = cardData(context, 'timeline', 'test', now);
        expect(data.todayKnown).assertFalse();
        expect(data.tomorrowKnown).assertTrue();
        expect(data.tomorrowCourses.length).assertEqual(1);
        expect(data.showTomorrow).assertTrue();
        const expired = cardData(context, 'timeline', 'test', now + 8 * 86400000);
        expect(expired.status).assertEqual('stale');
        expect(expired.tomorrowCourses.length).assertEqual(0);
        store.deleteSync(projectionKey);
        store.putSync(emptyKey, 'signedOut');
        const cleared = cardData(context, 'timeline', 'test', now);
        expect(cleared.status).assertEqual('signedOut');
        expect(cleared.generatedAt).assertEqual('');
        expect(cleared.tomorrowCourses.length).assertEqual(0);
      } finally {
        if (oldProjection === null) store.deleteSync(projectionKey); else store.putSync(projectionKey, oldProjection);
        if (oldEmpty === null) store.deleteSync(emptyKey); else store.putSync(emptyKey, oldEmpty);
        if (oldToggle === null) store.deleteSync(toggleKey); else store.putSync(toggleKey, oldToggle);
      }
    });
  });
}

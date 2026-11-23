import { describe, it, expect } from '@ohos/hypium';
import { parseSchedule, schoolDate, nextClass } from '../../../main/ets/widget/ScheduleData';

export default function scheduleDataTest(): void {
  describe('ScheduleData', () => {
    it('usesSchoolTimezone', 0, () => {
      expect(schoolDate(Date.parse('2026-11-05T16:00:00Z'))).assertEqual('2026-11-06');
    });
    it('rejectsMissingIdentityAndMalformedData', 0, () => {
      expect(parseSchedule('null')).assertNull();
      expect(parseSchedule('{}')).assertNull();
      expect(parseSchedule('{')).assertNull();
      expect(parseSchedule('{"schemaVersion":2,"timezone":"Asia/Shanghai","days":[]}')).assertNull();
    });
    it('neverShowsMissingProjectionAsNoClasses', 0, () => {
      expect(nextClass(null, Date.now()).status).assertEqual('needsData');
    });
  });
}

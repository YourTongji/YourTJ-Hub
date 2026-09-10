/// 课表时间段与节次映射工具（移植自 web `resource/src/site/utils/timetable.ts`）。
library;

import 'pk_arrange.dart';

/// 将课表行/节次（1..11 或 1..12）映射为教务系统及后端 P10
/// `/api/pk/courses-by-time` 对应的大节 Section（1..6）。
/// - calendarId >= [kNewSectionSystemMinCalendarId] 为 11 节新课制；
/// - 其余或缺省为 12 节旧课制。
/// 映射：1-2→1、3-4→2、5-6→3、7-8→4、9-10→5、11(新)或 11-12(旧)→6；越界 -1。
int getRowSection(int row, [int calendarId = 0]) {
  if (row >= 1 && row <= 10) {
    return ((row - 1) ~/ 2) + 1;
  }
  if (calendarId >= kNewSectionSystemMinCalendarId) {
    return row == 11 ? 6 : -1;
  }
  return (row == 11 || row == 12) ? 6 : -1;
}

/// 大节 Section（1..6）对应的标准节次范围文字描述。
String getSectionRangeText(int section, [int calendarId = 0]) {
  switch (section) {
    case 1:
      return '1-2';
    case 2:
      return '3-4';
    case 3:
      return '5-6';
    case 4:
      return '7-8';
    case 5:
      return '9-10';
    case 6:
      return calendarId >= kNewSectionSystemMinCalendarId ? '11' : '11-12';
    default:
      return '';
  }
}

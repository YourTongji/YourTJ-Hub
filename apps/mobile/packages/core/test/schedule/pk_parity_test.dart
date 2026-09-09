// 排课器纯算法与 Web 行为对照测试。
//
// 断言值不是手推的：由 node 直接执行 web 端真实 TS 实现
// （resource/src/site/utils/*.ts @ e523ab9a）生成（research/tsref/）。
// 任何断言失败都意味着 Dart 移植偏离了 Web 行为。
library;

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('pkArrange · buildWeekMask/overlap', () {
    test('位掩码与非法周次忽略', () {
      expect(buildWeekMask([1, 2, 3, 8, 16]), 32903);
      expect(buildWeekMask([0, 17, 5]), 16); // 0/17 越界忽略
      expect(weeksOverlap([1, 2, 3], [3, 4]), isTrue);
      expect(weeksOverlap([1, 2], [3, 4]), isFalse);
      expect(weekMasksOverlap(0x3, 0x6), isTrue);
    });
  });

  group('pkArrange · parseArrangeInfoText', () {
    test('多段解析（含地点提取）', () {
      final result = parseArrangeInfoText(
        '1-8周 周一 3-4节 同济楼A201；9-16周 周三 5-6节 线上',
      );
      expect(result, hasLength(2));
      expect(result[0].weekStart, 1);
      expect(result[0].weekEnd, 8);
      expect(result[0].weekParity, isNull);
      expect(result[0].specificWeeks, isEmpty);
      expect(result[0].day, 1);
      expect(result[0].sectionStart, 3);
      expect(result[0].sectionEnd, 4);
      expect(result[0].location, '同济楼A201');
      expect(result[1].day, 3);
      expect(result[1].location, '线上');
    });

    test('单周奇偶与显式枚举周', () {
      final odd = parseArrangeInfoText('1-15周(单周) 星期二 第2节');
      expect(odd, hasLength(1));
      expect(odd[0].weekParity, PkWeekParity.odd);
      expect(odd[0].specificWeeks, [1, 3, 5, 7, 9, 11, 13, 15]);
      expect(odd[0].day, 2);
      expect(odd[0].sectionStart, 2);
      expect(odd[0].location, '');

      final enumd = parseArrangeInfoText('[1,3,5周] 周五 1-2节');
      expect(enumd[0].specificWeeks, [1, 3, 5]);
      expect(enumd[0].weekStart, 1);
      expect(enumd[0].weekEnd, 5);
    });

    test('「周天」识别；无星期段的段被丢弃', () {
      expect(parseArrangeInfoText('1-8周 周天 3-4节').single.day, 7);
      expect(parseArrangeInfoText('1-8周 3-4节 教1-101'), isEmpty);
    });

    test('7 段同位安排逐段解析（consolidate 的输入形态）', () {
      final result = parseArrangeInfoText(
        '[1-2 16周] 周一 5-6节 A；[3-4周] 周一 5-6节 B；[5-6周] 周一 5-6节 C',
      );
      expect(result, hasLength(3));
      // TS 正则对「[1-2 16周]」内空格分隔的双区间不识别，只捕获到 16 —— 一比一保留该容错差异
      expect(result[0].weekStart, 16);
      expect(result[0].weekEnd, 16);
      expect(result[0].specificWeeks, isEmpty);
      expect(result[1].weekStart, 3);
      expect(result[2].location, 'C');
    });
  });

  group('pkArrange · 行数与周次工具', () {
    test('calendarId>=120 = 11 节制', () {
      expect(maxRowsForCalendar(120), 11);
      expect(maxRowsForCalendar(131), 11);
      expect(maxRowsForCalendar(119), 12);
      expect(maxRowsForCalendar(null), 12);
    });

    test('clusterBySections 部分重叠同格', () {
      final clusters = clusterBySections<int>(
        [3, 5, 2, 9],
        (t) => switch (t) {
          3 => [3, 4],
          5 => [5, 6],
          2 => [2, 3],
          _ => [9, 10, 11],
        },
      );
      expect(clusters.map((c) => (c.start, c.end)).toList(), [
        (2, 4), // 2-3 与 3-4 部分重叠归同格
        (5, 6),
        (9, 11),
      ]);
    });

    test('currentWeekForDate 周一始、结束日含当天、超 16 周', () {
      DateTime d(int m, int day) => DateTime(2026, m, day);
      expect(currentWeekForDate('2026-08-31', d(9, 6)), 1); // 首周周日
      expect(currentWeekForDate('2026-08-31', d(9, 9)), 2);
      expect(currentWeekForDate('2026-08-31', d(8, 30)), isNull); // 开学前
      expect(
        currentWeekForDate('2026-08-31', d(9, 20), '2026-09-20'),
        3,
      ); // 结束日当天仍属学期
      expect(currentWeekForDate('2026-08-31', d(9, 21), '2026-09-20'), isNull);
      expect(currentWeekForDate('2026-09-02', d(9, 8)), 1); // 周中开课延伸到首个周日
    });

    test('formatWeeksText 与奇偶判定', () {
      expect(formatWeeksText([1, 2, 3, 5, 8, 9, 10]), '1-3,5,8-10');
      expect(formatWeeksText([]), '');
      expect(detectWeekParity([1, 3, 5]), PkWeekParity.odd);
      expect(detectWeekParity([2, 4, 6]), PkWeekParity.even);
      expect(detectWeekParity([1, 2, 3]), isNull);
    });
  });

  group('pkArrange · consolidateSameClassArrangements（7-slice 病理用例）', () {
    test('同班多段合并：周次并集 + 教师去重 + 文本重生成', () {
      PkConsolidatableCourse seg(
        List<int> weeks,
        String teacher,
        String room,
      ) => PkConsolidatableCourse(
        code: '12117901',
        courseName: '现代分析测试技术',
        occupyDay: 1,
        occupyTime: [5, 6],
        occupyWeek: weeks,
        teacherAndCode: teacher,
        arrangementText: 'x',
        occupyRoom: room,
        showText: 'x',
      );

      final merged = consolidateSameClassArrangements([
        seg([1, 2, 16], '张三(T001)', '化学馆A101'),
        seg([3, 4], '李四(T002),张三(T001)', '化学馆A102'),
        seg([5, 6], '李四(T002)', '化学馆A102'),
      ]);
      expect(merged, hasLength(1));
      expect(merged[0].occupyWeek, [1, 2, 3, 4, 5, 6, 16]);
      expect(merged[0].occupyTime, [5, 6]);
      expect(merged[0].teacherAndCode, '张三、李四');
      expect(merged[0].occupyRoom, '化学馆A101 / 化学馆A102');
      expect(merged[0].arrangementText, '[1-6,16周] 周1 第5-6节 化学馆A101 / 化学馆A102');
      expect(
        merged[0].showText,
        '张三、李四 现代分析测试技术(12117901) [1-6,16周] 周1 第5-6节 化学馆A101 / 化学馆A102',
      );
    });
  });

  group('pkConflict · 课号归一化', () {
    test('点号与末两位两种班号格式', () {
      expect(getCourseBaseCode('122004.01'), '122004');
      expect(getCourseBaseCode('12200401'), '122004');
      expect(getCourseBaseCode('12'), '12'); // 过短不裁
      expect(isSameCourse('122004.01', '12200402'), isTrue);
      expect(isSameCourse('122004.01', '122005.01'), isFalse);
    });
  });

  group('pkConflict · 占用表与冲突判定', () {
    PkArrangement arr(int day, List<int> time, List<int> weeks) =>
        PkArrangement(
          arrangementText: 'x',
          occupyDay: day,
          occupyTime: time,
          occupyWeek: weeks,
          occupyRoom: 'R',
          teacherAndCode: 'T',
        );

    PkCourseDetail det(String code, List<PkArrangement> arrangements) =>
        PkCourseDetail(
          arrangementInfo: arrangements,
          campus: '四平路',
          code: code,
          status: 2,
          teachers: const [],
          teachingLanguage: '中文',
        );

    test('insert/delete 纯函数语义', () {
      final d1 = det('122004.01', [
        arr(1, [3, 4], [1, 2, 3, 4]),
      ]);
      final d2 = det('123005.01', [
        arr(2, [5, 6], [1, 2, 3, 4]),
      ]);
      var occ = insertOccupied(
        createEmptyOccupied(),
        d1.arrangementInfo,
        d1.code,
        '高等数学',
      );
      occ = insertOccupied(occ, d2.arrangementInfo, d2.code, '大学英语');
      expect(occ[2][0].single.code, '122004.01'); // 时间3→行2，周一→列0
      expect(occ[4][1], hasLength(1));
      final afterDelete = deleteOccupied(occ, '123005.01');
      expect(afterDelete[4][1], isEmpty); // 删除只影响同基础课号
      expect(afterDelete[2][0], hasLength(1)); // 其他课不受影响
      expect(occ[4][1], hasLength(1)); // 原表未被就地修改
    });

    test('canAddCourse：同课换班隐式替换；同位单双周相撞', () {
      final occupied = insertOccupied(
        createEmptyOccupied(),
        [
          arr(1, [3, 4], [1, 2, 3, 4]),
        ],
        '122004.01',
        '高等数学',
      );

      // 同基础课号换班：先移除旧班再判定 → 可加入
      final swap = det('122004.02', [
        arr(5, [1, 2], [1, 2]),
      ]);
      expect(
        canAddCourse(swap.arrangementInfo, occupied, swap.code).canAdd,
        isTrue,
      );

      // 单双周（[2,4]）与全周（[1-4]）同天同节 → 交集非空 → 冲突
      final parityClash = det('124006.01', [
        arr(1, [3, 4], [2, 4]),
      ]);
      final clash = canAddCourse(
        parityClash.arrangementInfo,
        occupied,
        parityClash.code,
      );
      expect(clash.canAdd, isFalse);
      expect(clash.collideCourse, '122004.01 高等数学');

      // 不同天不冲突
      final otherDay = det('125007.01', [
        arr(4, [3, 4], [1, 2]),
      ]);
      expect(
        canAddCourse(otherDay.arrangementInfo, occupied, otherDay.code).canAdd,
        isTrue,
      );
    });

    test('findConflicts 列举全部冲突（按基础课号去重）', () {
      var occ = insertOccupied(
        createEmptyOccupied(),
        [
          arr(1, [3, 4], [1, 2, 3, 4]),
        ],
        '122004.01',
        '高等数学',
      );
      occ = insertOccupied(
        occ,
        [
          arr(2, [5, 6], [1, 2, 3, 4]),
        ],
        '123005.01',
        '大学英语',
      );
      final candidate = det('125007.01', [
        arr(1, [3, 4], [2, 4]),
        arr(2, [5, 6], [1]),
      ]);
      final conflicts = findConflicts(candidate, occ);
      expect(
        conflicts.map((c) => c.code),
        containsAll(['122004.01', '123005.01']),
      );
    });

    test('findClassConflicts 候选班级冲突预检（同门换班不互斥，保留外部冲突）', () {
      var occ = insertOccupied(
        createEmptyOccupied(),
        [
          arr(1, [3, 4], [1, 2, 3, 4]),
        ],
        '122004.01',
        '高等数学',
      );
      occ = insertOccupied(
        occ,
        [
          arr(2, [5, 6], [1, 2, 3, 4]),
        ],
        '123005.01',
        '大学英语',
      );

      // 同门课程换班（122004.02），时间与 122004.01 重叠，但不应判定为与自己冲突
      final candidateSameCourse = det('122004.02', [
        arr(1, [3, 4], [1, 2]),
      ]);
      final sameConflicts = findClassConflicts(candidateSameCourse, occ);
      expect(sameConflicts, isEmpty);

      // 与外部课程冲突（123005.01）
      final candidateDiffCourse = det('125007.01', [
        arr(2, [5, 6], [2]),
      ]);
      final diffConflicts = findClassConflicts(candidateDiffCourse, occ);
      expect(diffConflicts, hasLength(1));
      expect(diffConflicts.single.code, '123005.01');
      expect(diffConflicts.single.courseName, '大学英语');

      // 自定义占位符 custom: 正常参与冲突
      var occWithCustom = insertOccupied(
        createEmptyOccupied(),
        [
          arr(3, [1, 2], [1, 2]),
        ],
        'custom:meeting',
        '组会',
      );
      final candidateCustomConflict = det('129009.01', [
        arr(3, [1], [1]),
      ]);
      final customConflicts = findClassConflicts(
        candidateCustomConflict,
        occWithCustom,
      );
      expect(customConflicts, hasLength(1));
      expect(customConflicts.single.code, 'custom:meeting');
    });

    test('isArrangementConflicted 单时段颗粒度预检', () {
      final occ = insertOccupied(
        createEmptyOccupied(),
        [
          arr(1, [3, 4], [1, 2, 3, 4]),
        ],
        '122004.01',
        '高等数学',
      );

      // 与 122004 冲突的时段
      final clashSlot = arr(1, [3], [2]);
      expect(isArrangementConflicted(clashSlot, '125007.01', occ), isTrue);

      // 无冲突时段
      final freeSlot = arr(1, [5], [2]);
      expect(isArrangementConflicted(freeSlot, '125007.01', occ), isFalse);

      // 同课换班的时段重叠不判定为冲突
      expect(isArrangementConflicted(clashSlot, '122004.02', occ), isFalse);
    });

    test('deriveConflicts：双向 + custom 事件 + 同课不互斥', () {
      var occ = insertOccupied(
        createEmptyOccupied(),
        [
          arr(1, [3, 4], [1, 2, 3, 4]),
        ],
        '122004.01',
        '高等数学',
      );
      occ = insertOccupied(
        occ,
        [
          arr(1, [3, 4], [2, 4]),
        ],
        '125007.01',
        '体育',
      );

      final derived = deriveConflicts(occ);
      expect(derived['122004']!.single.code, '125007.01');
      expect(derived['125007']!.single.code, '122004.01');

      // custom 事件与真实课程、与其他 custom 事件均按判据冲突
      var occ2 = insertOccupied(
        createEmptyOccupied(),
        [
          arr(1, [1, 2], [1, 2]),
        ],
        'custom:evt_1',
        '有事A',
      );
      occ2 = insertOccupied(
        occ2,
        [
          arr(1, [1, 2], [2, 3]),
        ],
        'custom:evt_2',
        '有事B',
      );
      occ2 = insertOccupied(
        occ2,
        [
          arr(1, [1, 2], [3]),
        ],
        '111001.01',
        '课程X',
      );
      final derived2 = deriveConflicts(
        occ2,
      ).map((k, v) => MapEntry(k, v.map((e) => e.code).toList()));
      expect(derived2['custom:evt_1'], ['custom:evt_2']); // 周次交集 {2}
      expect(derived2['custom:evt_2'], ['custom:evt_1', '111001.01']);
      expect(derived2['111001'], ['custom:evt_2']); // base=111001
      expect(conflictBaseOf('custom:evt_1'), 'custom:evt_1'); // custom 前缀不裁尾
    });
  });

  group('pkTimetable · 节次映射', () {
    test('新11节制与旧12节制', () {
      expect(getRowSection(11, 120), 6);
      expect(getRowSection(12, 120), -1); // 新制无第12节
      expect(getRowSection(11, 100), 6);
      expect(getRowSection(12, 100), 6);
      expect(getRowSection(5), 3);
      expect(getSectionRangeText(6, 120), '11');
      expect(getSectionRangeText(6, 100), '11-12');
      expect(getSectionRangeText(3), '5-6');
    });
  });

  group('pkGrid · 行高分配（mobile/poster 双度量）', () {
    test('多课格撑高行高，单门课不拉伸', () {
      PkCourseOnTable course(String code, String name, List<int> weeks) =>
          PkCourseOnTable(
            showText: name,
            courseName: name,
            code: code,
            occupyTime: const [3, 4],
            occupyDay: 1,
            occupyWeek: weeks,
          );

      List<List<List<PkCourseOnTable>>> emptyGrid() => List.generate(
        12,
        (_) => List.generate(7, (_) => <PkCourseOnTable>[]),
      );

      final cellCourses = emptyGrid();
      cellCourses[1][0] = [
        course('101', '单双周A', const [1, 3, 5]),
        course('102', '单双周B', const [2, 4]),
      ];

      final grid = GridLayout(
        cellCourses: cellCourses,
        cellSpans: List.generate(12, (_) => List<int>.filled(7, 1)),
        occupiedGrid: List.generate(12, (_) => List<bool>.filled(7, false)),
      );

      final mobile = computeRowHeights(grid, kInteractiveRowMetricsMobile);
      expect(mobile[1], 124); // 2*58+4+4=124，其余行 52
      expect(mobile[0], 52);
      expect(mobile.where((h) => h == 52).length, 11);

      final poster = computeRowHeights(grid, kPosterRowMetrics);
      expect(poster[1], 192); // 2*90+4+8=192
      expect(poster[0], 76);

      expect(
        cellInnerHeightFor(2, mobile, kInteractiveRowMetricsMobile, 3),
        100,
      );
      expect(cardMinHeightFor(1, mobile, 1, kInteractiveRowMetricsMobile), 48);
      expect(cardMinHeightFor(1, mobile, 2, kInteractiveRowMetricsMobile), 48);
    });

    test('教师名清洗与紧凑展示', () {
      expect(
        teacherNameOf(
          PkCourseOnTable(
            showText: '',
            courseName: '',
            code: '',
            occupyTime: const [],
            occupyDay: 1,
            teacherAndCode: '张三(T001)',
          ),
        ),
        '张三',
      );
      expect(compactTeacherName('张三、李四', 2), '张三、李四');
      expect(compactTeacherName('张三、李四、王五', 2), '张三 等');
    });

    test('formatDisplayWeeks 单双周精简', () {
      String parityLabel(PkWeekParity p) => p == PkWeekParity.odd ? '单周' : '双周';
      String weeksTemplate(String range) => '共$range周';
      expect(
        formatDisplayWeeks(
          [1, 3, 5, 7, 9, 11, 13, 15],
          parityLabel,
          weeksTemplate,
        ),
        '1-15周(单周)',
      );
      expect(
        formatDisplayWeeks([1, 2, 3], parityLabel, weeksTemplate),
        '共1-3周',
      );
    });

    test('dayPartKeyForRow 默认作息切分点', () {
      expect(dayPartKeyForRow(1, kDefaultSectionTimes12), 'morning');
      expect(dayPartKeyForRow(5, kDefaultSectionTimes12), 'afternoon');
      expect(dayPartKeyForRow(10, kDefaultSectionTimes12), 'evening');
      expect(dayPartKeyForRow(2, kDefaultSectionTimes12), isNull);
    });
  });

  group('pkSectionTimes · 默认表与覆盖合并（PR #496 十一节语义）', () {
    test('现行 11 节制锚点：晚间自第 9 节 18:30 起', () {
      String startOf(List<SectionTime> table, int section) =>
          table.firstWhere((s) => s.section == section).start;
      // 历史 12 节制表：第 9 节 17:10、晚间 10/11/12 节（calendarId<120）。
      expect(startOf(kDefaultSectionTimes12, 9), '17:10');
      expect(startOf(kDefaultSectionTimes12, 10), '18:30');
      expect(kDefaultSectionTimes12, hasLength(12));
      // 现行 11 节制：晚间重新编号为 9/10/11 节（2025-2026 学年起）。
      expect(startOf(kDefaultSectionTimes11, 3), '10:00');
      expect(startOf(kDefaultSectionTimes11, 5), '13:30');
      expect(startOf(kDefaultSectionTimes11, 7), '15:30');
      expect(startOf(kDefaultSectionTimes11, 9), '18:30');
      expect(kDefaultSectionTimes11, hasLength(11));
    });

    test('override：11 节制按 section 覆盖（未知忽略）；12 节制恒历史表', () {
      // 12 节制为历史学期：恒返回内置历史表、忽略覆盖（web 同语义）。
      final historical = sectionTimesFor(12, const [
        SectionTime(section: 3, start: '09:55', end: '10:40'),
      ]);
      expect(historical, hasLength(12));
      expect(historical[2].start, '10:00'); // 覆盖被忽略，保持历史默认
      // 11 节制：覆盖按 section 对齐补齐缺口，未知 section 忽略。
      final merged = sectionTimesFor(11, const [
        SectionTime(section: 3, start: '09:55', end: '10:40'),
        SectionTime(section: 20, start: '99:99', end: '99:99'),
      ]);
      expect(merged[2].start, '09:55');
      expect(merged[1].start, '08:50'); // 未覆盖保持默认
      expect(merged, hasLength(11));
      // 覆盖为空/null 时回退默认表。
      expect(sectionTimesFor(11, null), hasLength(11));
      expect(sectionTimesFor(11, const []), hasLength(11));
    });

    test('parseHHMM 与 dayPart（11 节制晚间切点为第 9 节）', () {
      expect(parseHHMM('08:05'), 485);
      expect(parseHHMM('9:05'), 545);
      expect(parseHHMM('24:00'), isNull);
      expect(parseHHMM('abc'), isNull);
      expect(dayPartOfStart('08:00'), 'morning');
      expect(dayPartOfStart('13:30'), 'afternoon');
      expect(dayPartOfStart('18:30'), 'evening');
      expect(dayPartBoundaries(kDefaultSectionTimes12), {
        'morning': 1,
        'afternoon': 5,
        'evening': 10,
      });
      // 现行 11 节制：晚间自第 9 节起（node 参照 run3 输出）。
      expect(dayPartBoundaries(kDefaultSectionTimes11), {
        'morning': 1,
        'afternoon': 5,
        'evening': 9,
      });
    });
  });

  group('pkCourseOrder · planned-first 稳定分区', () {
    test('计划内置顶且各自保持原序', () {
      final result = sortPlannedCoursesFirst(
        ['b', 'a', 'c', 'd'],
        ['c', 'a'],
        (s) => s,
      );
      expect(result, ['a', 'c', 'b', 'd']);
    });
  });

  group('pk_models · JSON 往返（localStorage schema 兼容）', () {
    test('PkPlan toJson/fromJson roundtrip 保留全部字段', () {
      final plan = PkPlan(
        id: 'plan_lz0abc_1',
        name: '方案一',
        createdAt: 1757000000000,
        stagedCourses: [
          PkStagedCourse(
            courseCode: '122004',
            courseName: '高等数学',
            courseNameReserved: '高等数学A',
            credit: 5.0,
            courseType: '必',
            teacher: [PkTeacher(teacherName: '张三', teacherCode: 'T001')],
            status: 2,
            courseDetail: [
              PkCourseDetail(
                arrangementInfo: [
                  PkArrangement(
                    arrangementText: '[1-8周] 周一 3-4节 教1-101',
                    occupyDay: 1,
                    occupyTime: const [3, 4],
                    occupyWeek: List.generate(8, (i) => i + 1),
                    occupyRoom: '教1-101',
                    teacherAndCode: '张三(T001)',
                  ),
                ],
                campus: '四平路',
                code: '122004.01',
                teachingClassId: 9001,
                isExclusive: true,
                status: 2,
                teachers: const [],
                teachingLanguage: '中文',
              ),
            ],
          ),
        ],
        selectedCourses: const ['122004.01'],
        customEvents: [
          PkCustomEvent(
            id: 'evt_1',
            label: '有事',
            day: 3,
            sections: const [1, 2],
            weeks: const [1, 2],
          ),
        ],
      );

      final restored = PkPlan.fromJson(plan.toJson());
      expect(restored.id, plan.id);
      expect(
        restored.stagedCourses.single.courseDetail.single.teachingClassId,
        9001,
      );
      expect(
        restored.stagedCourses.single.courseDetail.single.isExclusive,
        isTrue,
      );
      expect(restored.stagedCourses.single.credit, 5.0);
      expect(restored.customEvents.single.sections, [1, 2]);
      expect(restored.selectedCourses, ['122004.01']);
    });
  });
}

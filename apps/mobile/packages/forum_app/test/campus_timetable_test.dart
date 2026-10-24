import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter/gestures.dart' show kLongPressTimeout;
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart' show RenderParagraph;
import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/campus/campus_calendar_export.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/campus/timetable/campus_timetable.dart';
import 'package:forum_app/src/pages/campus/timetable/timetable_grid.dart';
import 'package:forum_app/src/pages/campus/timetable/timetable_layout.dart';
import 'package:forum_app/src/pages/campus/timetable/timetable_memo_store.dart';
import 'package:forum_app/src/storage/user_work_database.dart';
import 'package:ui_kit/ui_kit.dart';

import 'campus_native_test.dart' show campusTestApp;
import 'fixtures/campus_fixtures.dart';
import 'test_font_helpers.dart';

CampusEvent _event(String name, int day, int start, int end, List<int> weeks) =>
    CampusEvent(
      name: name,
      teacher: '教师',
      room: 'A101',
      campus: '四平路',
      day: day,
      start: start,
      end: end,
      weeks: weeks,
      credits: '2',
    );

Future<void> _openTimetable(WidgetTester tester) async {
  tester.widget<GfTabBar>(find.byType(GfTabBar)).onSelected('timetable');
  await tester.pumpAndSettle();
}

Finder _cardOf(bool Function(TimetableItem) test) =>
    find.byWidgetPredicate((w) => w is TimetableSessionCard && test(w.item));

class _ReadFailingUserWorkDatabase extends UserWorkDatabase {
  _ReadFailingUserWorkDatabase(this.backing) : super(NativeDatabase.memory());

  final UserWorkDatabase backing;

  @override
  int get generation => backing.generation;

  @override
  Future<Map<String, String>> readDomain(String scope, String domain) =>
      Future.error(StateError('read failed'));

  @override
  Future<void> writeBatch(
    String scope,
    String domain,
    Map<String, String?> entries, {
    bool Function()? isCurrent,
    int? generation,
  }) => backing.writeBatch(
    scope,
    domain,
    entries,
    isCurrent: isCurrent,
    generation: generation,
  );
}

void main() {
  group('week layout', () {
    test('empty weekends fold away unless pinned or used', () {
      final events = [
        _event('数学', 1, 1, 2, [1]),
      ];
      const prefs = TimetablePrefs();
      expect(
        buildTimetableWeek(
          events: events,
          week: 1,
          blocks: const [],
          prefs: prefs,
        ).days,
        [1, 2, 3, 4, 5],
      );
      expect(
        buildTimetableWeek(
          events: events,
          week: 1,
          blocks: const [],
          prefs: const TimetablePrefs(alwaysShowWeekend: true),
        ).days,
        [1, 2, 3, 4, 5, 6, 7],
      );
      final saturday = TimetableBlock(
        id: 'b',
        title: '社团',
        days: const [6],
        start: 3,
        end: 4,
      );
      expect(
        buildTimetableWeek(
          events: events,
          week: 1,
          blocks: [saturday],
          prefs: prefs,
        ).days,
        hasLength(7),
      );
    });

    test('other-week sessions only ghost into free cells', () {
      final events = [
        _event('本周课', 1, 1, 2, [1]),
        _event('撞格课', 1, 2, 3, [2]),
        _event('空格课', 1, 5, 6, [2]),
      ];
      final layout = buildTimetableWeek(
        events: events,
        week: 1,
        blocks: const [],
        prefs: const TimetablePrefs(showOtherWeeks: true),
      );
      final items = [for (final c in layout.clusters[0]) ...c.items];
      expect(items.map((i) => i.title), ['本周课', '空格课']);
      expect(items.last.kind, TimetableItemKind.otherWeek);
      expect(layout.isFree(1, 5), isTrue, reason: 'ghosts stay drawable');
      expect(layout.isFree(1, 1), isFalse);
      expect(layout.courseCount, 1);
    });

    test('blocks repeat by week rule', () {
      final block = TimetableBlock(
        id: 'b',
        title: '自习',
        days: const [2, 4],
        start: 9,
        end: 10,
        weeks: const [2],
      );
      List<String> titles(int week) => [
        for (final day in buildTimetableWeek(
          events: const [],
          week: week,
          blocks: [block],
          prefs: const TimetablePrefs(),
        ).clusters)
          for (final c in day) ...c.items.map((i) => i.title),
      ];
      expect(titles(1), isEmpty);
      expect(titles(2), ['自习', '自习']);
    });

    test('ignores blocks beyond the visible section count', () {
      final layout = buildTimetableWeek(
        events: const [],
        week: 1,
        blocks: [
          const TimetableBlock(
            id: 'b',
            title: '旧学期第十二节',
            days: [1],
            start: 12,
            end: 12,
          ),
        ],
        prefs: const TimetablePrefs(),
      );
      expect(layout.clusters.expand((day) => day), isEmpty);
    });

    test('now position follows sections and snaps across breaks', () {
      final times = sectionTimesFor(11, null);
      expect(timetableNowPosition(times, 11, 7 * 60), isNull);
      expect(timetableNowPosition(times, 11, 8 * 60), 0);
      expect(
        timetableNowPosition(times, 11, 8 * 60 + 30),
        closeTo(30 / 45, 1e-9),
      );
      // 09:35–10:00 is the long break before section 3.
      expect(timetableNowPosition(times, 11, 9 * 60 + 45), 2);
      expect(timetableNowPosition(times, 11, 22 * 60), isNull);
    });

    test('week dates anchor on the current teaching week', () {
      // 2026-10-21 is a Wednesday in campus time.
      final now = DateTime.utc(2026, 10, 21, 4);
      final dates = timetableWeekDates(9, 8, now);
      expect(dates.first, DateTime.utc(2026, 10, 26));
      expect(dates.last, DateTime.utc(2026, 11, 1));
    });
  });

  test('memos persist per account scope', () async {
    final database = UserWorkDatabase.instance;
    final store = TimetableMemoStore(database: database, scope: 'site:1');
    await Future<void>.delayed(Duration.zero);
    final key = timetableCourseKey('数学', '教师');
    await store.setNote(key, null, '带计算器');
    await store.setNote(key, 3, '小测');
    await store.saveBlock(
      const TimetableBlock(
        id: 'b1',
        title: '社团',
        days: [3],
        start: 9,
        end: 10,
        color: 4,
      ),
    );
    await store.setPrefs(const TimetablePrefs(showOtherWeeks: true));
    store.dispose();

    final reloaded = TimetableMemoStore(database: database, scope: 'site:1');
    await Future<void>.delayed(const Duration(milliseconds: 10));
    final state = reloaded.state;
    expect(state.loaded, isTrue);
    expect(state.notesIn(key, 3), ['小测', '带计算器']);
    expect(state.notesIn(key, 4), ['带计算器']);
    expect(state.blocks.single.color, 4);
    expect(state.prefs.showOtherWeeks, isTrue);
    reloaded.dispose();

    final other = TimetableMemoStore(database: database, scope: 'site:2');
    await Future<void>.delayed(const Duration(milliseconds: 10));
    expect(other.state.notes, isEmpty);
    other.dispose();
  });

  test(
    'failed memo load prevents edits from replacing stored values',
    () async {
      final backing = UserWorkDatabase(NativeDatabase.memory());
      const scope = 'memo-read-failure';
      await backing.writeBatch(scope, kTimetableMemoDomain, {
        'blocks': 'existing blocks',
        'notes': 'existing notes',
      });
      final failing = _ReadFailingUserWorkDatabase(backing);
      final store = TimetableMemoStore(database: failing, scope: scope);
      await Future<void>.delayed(Duration.zero);
      expect(store.state.error, isNotNull);
      await store.saveBlock(
        const TimetableBlock(
          id: 'new',
          title: 'new',
          days: [1],
          start: 1,
          end: 1,
        ),
      );
      await store.setNote('course', null, 'new');
      await store.deleteBlock('old');
      await store.setPrefs(const TimetablePrefs(showOtherWeeks: true));
      expect(await backing.readDomain(scope, kTimetableMemoDomain), {
        'blocks': 'existing blocks',
        'notes': 'existing notes',
      });
      expect(store.state.error, isNotNull);
      store.dispose();
      await failing.close();
      await backing.close();
    },
  );

  testWidgets('phone timetable fits one screen without scrolling', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _openTimetable(tester);
    final grid = find.byType(TimetableGrid);
    expect(grid, findsOneWidget);
    final rect = tester.getRect(grid);
    expect(rect.right, lessThanOrEqualTo(390));
    expect(rect.bottom, lessThanOrEqualTo(844));
    final list = tester
        .widget<ListView>(
          find.byKey(const PageStorageKey('campus-list-timetable')),
        )
        .controller!;
    expect(list.position.maxScrollExtent, lessThanOrEqualTo(1));
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('tapping a weekday focuses and widens that day', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _openTimetable(tester);
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    final tuesday = _cardOf((i) => i.day == 2 && i.start == 5);
    final before = tester.getSize(tuesday).width;
    await tester.tap(
      find.descendant(
        of: find.byType(TimetableGrid),
        matching: find.text(l.scheduleDayTue),
      ),
    );
    await tester.pumpAndSettle();
    final focused = tester.widget<TimetableSessionCard>(tuesday);
    expect(focused.expanded, isTrue);
    expect(tester.getSize(tuesday).width, greaterThan(before * 1.5));
    // Focused cells reveal the teacher.
    expect(
      find.descendant(of: tuesday, matching: find.text('示例教师')),
      findsOneWidget,
    );
    await tester.tap(
      find.descendant(
        of: find.byType(TimetableGrid),
        matching: find.text(l.scheduleDayTue),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.getSize(tuesday).width, closeTo(before, 1));
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('long-pressing a free cell draws and saves a block', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _openTimetable(tester);
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    final monday = tester.getRect(_cardOf((i) => i.day == 1 && i.start == 1));
    final row = (monday.height + 3) / 2;
    // Section 4 on Monday is free in the fixture.
    final gesture = await tester.startGesture(
      Offset(monday.center.dx, monday.bottom + row * 1.5),
    );
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 50));
    await gesture.moveBy(Offset(0, row));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    expect(find.text(l.timetableAddBlock), findsOneWidget);
    await tester.enterText(find.byType(TextField).first, '社团例会');
    await tester.pump();
    await tester.ensureVisible(find.text(l.timetableSaveBlock));
    await tester.pumpAndSettle();
    await tester.tap(find.text(l.timetableSaveBlock));
    await tester.pumpAndSettle();
    final block = _cardOf((i) => i.kind == TimetableItemKind.block);
    expect(block, findsOneWidget);
    final item = tester.widget<TimetableSessionCard>(block).item;
    expect((item.day, item.start, item.end), (1, 4, 5));
    expect(
      find.descendant(of: block, matching: find.text('社团例会')),
      findsOneWidget,
    );
    // The painted range does not outlive the editor.
    expect(_selection(l), findsNothing);

    // A saved block opens a read view first; editing is a separate step.
    await tester.tap(block);
    await tester.pumpAndSettle();
    expect(find.text(l.timetableBlock), findsWidgets);
    expect(find.text(l.timetableSaveBlock), findsNothing);
    await tester.tap(find.text(l.timetableEditBlock));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableSaveBlock), findsOneWidget);
    // Day, section and week controls stay folded behind the summary line.
    expect(find.text(l.timetableBlockDays), findsNothing);
    // One summary line: day and clock range lead, the week rule trails.
    final when = find.textContaining(
      RegExp(r'\d{2}:\d{2}–\d{2}:\d{2}  ·  '),
      findRichText: true,
    );
    expect(when, findsOneWidget);
    expect(
      tester.widget<RichText>(when).text.toPlainText(),
      startsWith('${l.scheduleDayMon} '),
    );
    await tester.tap(when);
    await tester.pumpAndSettle();
    expect(find.text(l.timetableBlockDays), findsOneWidget);
    // Steppers name the section and the time it lands on.
    expect(find.text(l.schedulePeriodRange('4')), findsOneWidget);
    expect(find.text(l.schedulePeriodRange('5')), findsOneWidget);
    // Nothing changed yet: Save waits, Cancel simply leaves.
    GfButton button(String label) =>
        tester.widget<GfButton>(find.widgetWithText(GfButton, label));
    expect(button(l.timetableSaveBlock).onPressed, isNull);
    await tester.tap(find.text(l.commonCancel));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableSaveBlock), findsNothing);
    expect(block, findsOneWidget);

    // Delete asks twice in place before removing the block.
    await tester.tap(block);
    await tester.pumpAndSettle();
    await tester.tap(find.text(l.timetableEditBlock));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip(l.timetableDeleteBlock));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableConfirmDelete), findsOneWidget);
    expect(block, findsOneWidget);
    await tester.tap(find.text(l.timetableConfirmDelete));
    await tester.pumpAndSettle();
    expect(block, findsNothing);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('a pending cell selection clears on dismiss or outside tap', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _openTimetable(tester);
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    final monday = tester.getRect(_cardOf((i) => i.day == 1 && i.start == 1));
    final row = (monday.height + 3) / 2;
    final free = Offset(monday.center.dx, monday.bottom + row * 1.5);
    await tester.tapAt(free);
    await tester.pump();
    expect(_selection(l), findsOneWidget);
    // Outside the grid: the header's status line just above it.
    final grid = tester.getRect(find.byType(TimetableGrid));
    await tester.tapAt(Offset(grid.center.dx, grid.top - 18));
    await tester.pumpAndSettle();
    expect(_selection(l), findsNothing);

    await tester.tapAt(free);
    await tester.pump();
    await tester.tap(_selection(l));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableSaveBlock), findsOneWidget);
    Navigator.of(tester.element(find.text(l.timetableSaveBlock))).pop();
    await tester.pumpAndSettle();
    expect(_selection(l), findsNothing);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('course notes save from the details sheet', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _openTimetable(tester);
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    await tester.tap(_cardOf((i) => i.day == 2 && i.start == 5));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableCourseNote), findsOneWidget);
    await tester.enterText(find.byType(TextField).first, '带计算器');
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.text(l.timetableNotesSaved), findsOneWidget);
    final container = ProviderScope.containerOf(
      tester.element(find.byType(CampusPage)),
    );
    expect(
      container
          .read(timetableMemoProvider)
          .noteFor(timetableCourseKey('程序设计（演示）', '示例教师'), null),
      '带计算器',
    );
    Navigator.of(tester.element(find.text(l.timetableCourseNote))).pop();
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<TimetableSessionCard>(
            _cardOf((i) => i.day == 2 && i.start == 5),
          )
          .notes,
      ['带计算器'],
    );
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('options sheet keeps display switches and opens each tool', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _openTimetable(tester);
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    // The export tools no longer take timetable space.
    expect(find.text(l.campusExportCalendar), findsNothing);
    await tester.tap(find.byTooltip(l.timetableOptions));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableAlwaysShowWeekend), findsOneWidget);
    await tester.tap(find.text(l.timetableAlwaysShowWeekend));
    await tester.pumpAndSettle();
    // Tools sit one level down, so the sheet itself stays short.
    expect(find.text(l.timetableWidgetManual), findsNothing);
    expect(find.text(l.timetableMemosNone), findsOneWidget);
    await tester.tap(find.text(l.timetableWidgets));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableWidgetManual), findsOneWidget);
    Navigator.of(tester.element(find.text(l.timetableWidgetManual))).pop();
    await tester.pumpAndSettle();
    await tester.tap(find.text(l.campusExportCalendar));
    await tester.pumpAndSettle();
    expect(find.byType(CampusCalendarExportButton), findsOneWidget);
    Navigator.of(tester.element(find.byType(CampusCalendarExportButton))).pop();
    await tester.pumpAndSettle();
    // Adding from the memo sheet closes both sheets before the editor opens.
    await tester.tap(find.text(l.timetableMemos));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableNoMemos), findsOneWidget);
    await tester.tap(find.text(l.timetableAddBlock));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableMemosIntro), findsNothing);
    expect(find.text(l.timetableAlwaysShowWeekend), findsNothing);
    expect(find.text(l.timetableSaveBlock), findsOneWidget);
    Navigator.of(tester.element(find.text(l.timetableSaveBlock))).pop();
    await tester.pumpAndSettle();
    expect(
      tester.widget<TimetableGrid>(find.byType(TimetableGrid)).layout.days,
      hasLength(7),
    );
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('header add button opens the block editor directly', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _openTimetable(tester);
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    // Both header actions share the tinted capsule; options uses the
    // duotone settings glyph rather than the ambiguous slider.
    expect(
      find.byWidgetPredicate(
        (w) => w is GfSymbol && w.name == 'settings-duotone',
      ),
      findsOneWidget,
    );
    final svg = await rootBundle.loadString(
      'packages/ui_kit/assets/icons/settings-duotone.svg',
    );
    expect(svg, contains('opacity="0.28"'));
    await tester.tap(find.byTooltip(l.timetableAddBlock));
    await tester.pumpAndSettle();
    expect(find.text(l.timetableSaveBlock), findsOneWidget);
    Navigator.of(tester.element(find.text(l.timetableSaveBlock))).pop();
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  for (final (locale, scale) in [
    (const Locale('zh'), 1.0),
    (const Locale('en'), 1.0),
    (const Locale('de'), 1.6),
  ]) {
    testWidgets('block editor time reads in one line ($locale x$scale)', (
      tester,
    ) async {
      await loadTestFonts(tester);
      tester.view.physicalSize = const Size(360, 720);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        campusTestApp(FakeCampusRepository(), scale: scale, locale: locale),
      );
      await tester.pumpAndSettle();
      await _openTimetable(tester);
      final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
      await tester.tap(find.byTooltip(l.timetableAddBlock));
      await tester.pumpAndSettle();
      final when = find.textContaining(
        RegExp(r'\d{2}:\d{2}–\d{2}:\d{2}  ·  '),
        findRichText: true,
      );
      if (scale == 1.0) {
        // Day, clock range and week rule all fit without an ellipsis.
        expect(
          tester.renderObject<RenderParagraph>(when).didExceedMaxLines,
          isFalse,
        );
      }
      await tester.tap(when);
      await tester.pumpAndSettle();
      // The seven days share one row.
      final days = [
        for (final label in timetableWeekdayLabels(l))
          tester.getCenter(find.text(label).last).dy,
      ];
      expect(days.toSet().length, 1);
      expect(tester.takeException(), isNull);
      Navigator.of(tester.element(find.text(l.timetableSaveBlock))).pop();
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }

  for (final (size, scale) in [
    (const Size(320, 568), 2.0),
    (const Size(844, 390), 1.0),
  ]) {
    testWidgets('timetable stays intact at $size text x$scale', (tester) async {
      tester.view.physicalSize = size;
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        campusTestApp(
          FakeCampusRepository(),
          scale: scale,
          locale: const Locale('de'),
        ),
      );
      await tester.pumpAndSettle();
      await _openTimetable(tester);
      expect(find.byType(TimetableGrid), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }

  testWidgets('tablet shows the focused day beside the week', (tester) async {
    tester.view.physicalSize = const Size(1180, 820);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      campusTestApp(FakeCampusRepository(), child: const CampusPage()),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('campus-module-timetable')));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    expect(find.byType(TimetableGrid), findsOneWidget);
    await tester.tap(
      find.descendant(
        of: find.byType(TimetableGrid),
        matching: find.text(l.scheduleDayTue),
      ),
    );
    await tester.pumpAndSettle();
    // The agenda lists Tuesday's full session details.
    expect(find.textContaining('教学楼 B202'), findsWidgets);
    expect(
      tester.widget<TimetableGrid>(find.byType(TimetableGrid)).expandFocus,
      isFalse,
    );
    expect(
      tester.getRect(find.byType(TimetableGrid)).bottom,
      lessThanOrEqualTo(820),
    );
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  test('header height scales with text', () {
    expect(campusTimetableHeaderHeight(1), 78);
    expect(campusTimetableHeaderHeight(2), greaterThan(140));
  });
}

Finder _selection(AppLocalizations l) => find.byWidgetPredicate(
  (w) => w is Semantics && w.properties.label == l.timetableAddBlockHere,
);

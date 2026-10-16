import 'dart:async';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:forum_app/src/pages/campus/campus_calendar_export.dart';
import 'package:forum_app/src/pages/campus/campus_calendar_rules.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'campus_native_test.dart' show campusTestApp;
import 'fixtures/campus_fixtures.dart';

class _Paths extends PathProviderPlatform {
  _Paths(this.path);
  final String path;
  @override
  Future<String?> getTemporaryPath() async => path;
}

void main() {
  test(
    'expired export authorization enables the existing reauthorization flow',
    () async {
      final repo = FakeCampusRepository()
        ..exportError = const ApiException(
          fallbackMessage: '',
          messageCode: 'campus.authorizationRequired',
        );
      final controller = CampusController(repo);
      await controller.refresh();
      await expectLater(
        controller.exportCalendar(),
        throwsA(isA<ApiException>()),
      );
      expect(controller.state.needsAuthorization, isTrue);
      expect(controller.state.status?.binding, isNotNull);
      controller.dispose();
    },
  );
  test('refresh and disposal fence late calendar exports', () async {
    final repo = FakeCampusRepository()..pendingExport = Completer();
    final controller = CampusController(repo);
    await controller.refresh();
    final first = controller.exportCalendar();
    await controller.refresh();
    repo.pendingExport!.complete(
      const CampusCalendarExport(
        filename: 'private.ics',
        content: 'old account',
        eventCount: 1,
      ),
    );
    expect(await first, isNull);
    repo.pendingExport = Completer();
    final second = controller.exportCalendar();
    controller.dispose();
    repo.pendingExport!.complete(
      const CampusCalendarExport(
        filename: 'private.ics',
        content: 'disposed account',
        eventCount: 1,
      ),
    );
    expect(await second, isNull);
    expect(repo.cancellations.every((c) => c.isCancelled), isTrue);
  });

  testWidgets(
    'shares a calendar file with an iPad anchor and removes staging files',
    (tester) async {
      final directory = Directory.systemTemp.createTempSync(
        'campus-export-test-',
      );
      final original = PathProviderPlatform.instance;
      PathProviderPlatform.instance = _Paths(directory.path);
      const channel = MethodChannel('dev.fluttercommunity.plus/share');
      addTearDown(() {
        PathProviderPlatform.instance = original;
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(channel, null);
        directory.deleteSync(recursive: true);
      });
      final shared = Completer<void>();
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            expect(call.method, 'share');
            final args = call.arguments as Map;
            expect(args['mimeTypes'], ['text/calendar']);
            expect(args['originWidth'], greaterThan(0));
            expect(args['originHeight'], greaterThan(0));
            final file = File((args['paths'] as List).single as String);
            expect(file.path, endsWith('.ics'));
            expect(
              await file.readAsString(),
              startsWith('BEGIN:VCALENDAR\r\n'),
            );
            shared.complete();
            return 'calendar-app';
          });
      final repo = FakeCampusRepository();
      await tester.pumpWidget(
        campusTestApp(
          repo,
          child: const Scaffold(
            body: CampusCalendarExportButton(enabled: true),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        tester.widget<SwitchListTile>(find.byType(SwitchListTile)).value,
        isTrue,
      );
      await tester.tap(find.text('开启调休规则'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('导出课程日历'));
      // Advance widget microtasks and the real filesystem loop independently.
      for (var i = 0; i < 100; i++) {
        await tester.pump();
        await tester.runAsync(
          () => Future<void>.delayed(const Duration(milliseconds: 10)),
        );
        if (shared.isCompleted && directory.listSync().isEmpty) break;
      }
      expect(shared.isCompleted, isTrue);
      expect(repo.lastApplyAdjustments, isFalse);
      expect(directory.listSync(), isEmpty);
      await tester.pumpAndSettle();
      expect(repo.requested.where((s) => s == 'calendar-export').length, 1);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'published adjustments refresh for the widget and when expanded',
    (tester) async {
      final repo = FakeCampusRepository();
      await tester.pumpWidget(
        campusTestApp(
          repo,
          child: const Scaffold(
            body: CampusCalendarExportButton(enabled: true),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        repo.requested.where((key) => key == 'calendar-rules'),
        hasLength(1),
      );
      await tester.tap(find.text('查看已发布的调休规则'));
      await tester.pumpAndSettle();
      expect(
        repo.requested.where((key) => key == 'calendar-rules'),
        hasLength(2),
      );
      expect(find.text('国庆补课'), findsOneWidget);
      expect(find.textContaining('2026年10月6日'), findsOneWidget);
      expect(find.textContaining('2026年9月20日'), findsOneWidget);
      await tester.tap(find.text('查看已发布的调休规则'));
      await tester.pumpAndSettle();
      expect(find.text('国庆补课'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('shows incomplete calendar errors and supports retry', (
    tester,
  ) async {
    final repo = FakeCampusRepository()
      ..exportError = const ApiException(
        fallbackMessage: '',
        messageCode: 'campus.calendarIncomplete',
      );
    await tester.pumpWidget(
      campusTestApp(
        repo,
        child: const Scaffold(body: CampusCalendarExportButton(enabled: true)),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('导出课程日历'));
    await tester.pumpAndSettle();
    expect(find.textContaining('暂时无法准确导出'), findsOneWidget);
    await tester.tap(find.text('导出课程日历'));
    await tester.pumpAndSettle();
    expect(repo.requested.where((s) => s == 'calendar-export').length, 2);
  });

  testWidgets(
    'leaving the private view cancels pending export without sharing',
    (tester) async {
      final repo = FakeCampusRepository()..pendingExport = Completer();
      await tester.pumpWidget(
        campusTestApp(
          repo,
          child: const Scaffold(
            body: CampusCalendarExportButton(enabled: true),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('导出课程日历'));
      await tester.pump();
      await tester.pumpWidget(const SizedBox());
      repo.pendingExport!.complete(
        const CampusCalendarExport(
          filename: 'private.ics',
          content: 'late private data',
          eventCount: 1,
        ),
      );
      await tester.pumpAndSettle();
      expect(repo.cancellations.every((c) => c.isCancelled), isTrue);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'rules ExpansionTile remount does not collide with list scroll offset',
    (tester) async {
      // 复刻 campus 页结构：PageStorageKey 列表内的 ExpansionTile 若无独立
      // key，会与列表滚动偏移（double）共用同一 PageStorage 标识符；滚动后
      // 重挂载（tab 切换/刷新）时 initState 读展开状态 cast 崩溃。
      final repo = FakeCampusRepository();
      final bucket = PageStorageBucket();
      Widget buildApp({required bool showRules}) => campusTestApp(
        repo,
        child: PageStorage(
          bucket: bucket,
          child: Scaffold(
            body: ListView(
              key: const PageStorageKey<String>('campus-list-timetable'),
              children: <Widget>[
                if (showRules) const CampusCalendarRulesView(),
                const SizedBox(height: 3000),
              ],
            ),
          ),
        ),
      );

      await tester.pumpWidget(buildApp(showRules: true));
      await tester.pumpAndSettle();
      expect(find.text('查看已发布的调休规则'), findsOneWidget);

      // 滚动 → didEndScroll 将 pixels（double）写入共享标识符。
      await tester.drag(find.byType(ListView), const Offset(0, -300));
      await tester.pumpAndSettle();

      // 重挂载 rules tile：修复前 initState 读到 double，cast bool? 崩溃。
      await tester.pumpWidget(buildApp(showRules: false));
      await tester.pumpAndSettle();
      await tester.pumpWidget(buildApp(showRules: true));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      // 滚动偏移恢复后 tile 在视口上方，先滚回顶部再断言。
      await tester.drag(find.byType(ListView), const Offset(0, 300));
      await tester.pumpAndSettle();
      expect(find.text('查看已发布的调休规则'), findsOneWidget);
    },
  );
}

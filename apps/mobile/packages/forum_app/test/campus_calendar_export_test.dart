import 'dart:async';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:forum_app/src/pages/campus/campus_calendar_export.dart';
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
      expect(directory.listSync(), isEmpty);
      await tester.pumpAndSettle();
      expect(repo.requested.where((s) => s == 'calendar-export').length, 1);
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
}

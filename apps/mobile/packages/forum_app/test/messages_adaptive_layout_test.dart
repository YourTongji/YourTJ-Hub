import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';

import 'chat_avatar_profile_test.dart' show pumpConversation;
import 'chat_visible_read_test.dart' show pumpChat;
import 'pages_behavior_test.dart' show CountingPageRepository, makeChatMessage;

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  testWidgets(
    'wide messages keep the directory and retain chat across resizing',
    (tester) async {
      final app = await pumpChat(
        tester,
        targetUserId: null,
        messages: List.generate(40, (index) => makeChatMessage(index + 1)),
      );
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();

      expect(app.navigator.currentState!.canPop(), isFalse);
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        tester
            .widgetList<GfConversationRow>(find.byType(GfConversationRow))
            .singleWhere((row) => row.name == 'bob')
            .active,
        isTrue,
      );
      final field = find.byType(TextField).last;
      await tester.enterText(field, 'unfinished iPad message');
      final input = tester.widget<TextField>(field).controller!;
      final scroll = tester
          .widget<ListView>(find.byType(ListView).last)
          .controller!;
      await tester.drag(find.byType(ListView).last, const Offset(0, 250));
      await tester.pumpAndSettle();
      final calls = app.repo.afterCalls;
      final reads = app.repo.batches.length;

      await tester.binding.setSurfaceSize(const Size(500, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      expect(input.text, 'unfinished iPad message');
      expect(
        tester.widget<ListView>(find.byType(ListView).last).controller,
        same(scroll),
      );

      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      expect(app.repo.afterCalls, calls);
      expect(app.repo.batches.length, reads);
      expect(scroll.offset, greaterThan(0));
      await tester.binding.setSurfaceSize(const Size(500, 800));
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(find.textContaining('unfinished iPad message'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'compact conversation still pushes and wide rotation shows directory',
    (tester) async {
      final app = await pumpChat(tester, targetUserId: null, messages: []);
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();
      expect(app.navigator.currentState!.canPop(), isTrue);
      expect(find.byType(GfConversationRow), findsNothing);
      final input = tester
          .widget<TextField>(find.byType(TextField).last)
          .controller!;
      await tester.enterText(find.byType(TextField).last, 'compact draft');
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      expect(input.text, 'compact draft');
    },
  );

  testWidgets(
    'switching peers and target routes keeps drafts with their owner',
    (tester) async {
      final router = await pumpConversation(tester, messages: []);
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, 'Bob only');
      await tester.tap(find.text('carol'));
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        isEmpty,
      );
      await tester.enterText(find.byType(TextField).last, 'Carol only');
      router.go('/messages?userId=2&username=bob');
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        'Bob only',
      );
      router.go('/messages?userId=3&username=carol');
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        'Carol only',
      );
    },
  );

  testWidgets('covered and inactive split views stop chat polling', (
    tester,
  ) async {
    final app = await pumpChat(tester, targetUserId: 2, messages: []);
    await tester.binding.setSurfaceSize(const Size(1100, 800));
    await tester.pumpAndSettle();
    app.navigator.currentState!.push(
      MaterialPageRoute<void>(
        builder: (_) => const Scaffold(body: Text('Covered')),
      ),
    );
    await tester.pumpAndSettle();
    final coveredCalls = app.repo.afterCalls;
    await tester.pump(const Duration(seconds: 31));
    expect(app.repo.afterCalls, coveredCalls);
    app.navigator.currentState!.pop();
    await tester.pumpAndSettle();
    app.active.value = false;
    await tester.pumpAndSettle();
    final inactiveCalls = app.repo.afterCalls;
    await tester.pump(const Duration(seconds: 31));
    expect(app.repo.afterCalls, inactiveCalls);
  });

  testWidgets(
    'wide new conversations stay in the detail pane and retain drafts',
    (tester) async {
      final app = await pumpChat(tester, targetUserId: null, messages: []);
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('New message'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Dave'));
      await tester.pumpAndSettle();
      expect(app.navigator.currentState!.canPop(), isFalse);
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        app.repo.afterCalls,
        0,
        reason: 'no server conversation exists before the first send',
      );
      await tester.enterText(find.byType(TextField).last, 'Draft for Dave');
      await tester.pumpAndSettle();
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        isEmpty,
      );
      final draftRow = find.widgetWithText(GfConversationRow, 'dave');
      await tester.tap(draftRow);
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        'Draft for Dave',
      );
    },
  );

  testWidgets(
    'hidden compact directory stops polling and large text stays single pane',
    (tester) async {
      final app = await pumpChat(tester, targetUserId: 2, messages: []);
      final pages =
          app.container.read(pageRepositoryProvider) as CountingPageRepository;
      final compactCalls = pages.fetchCalls;
      await tester.pump(const Duration(seconds: 16));
      await tester.pumpAndSettle();
      expect(pages.fetchCalls, compactCalls);
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      final input = tester
          .widget<TextField>(find.byType(TextField).last)
          .controller;
      tester.platformDispatcher.textScaleFactorTestValue = 2;
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      final hiddenCalls = pages.fetchCalls;
      await tester.pump(const Duration(seconds: 16));
      await tester.pumpAndSettle();
      expect(pages.fetchCalls, hiddenCalls);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'visible reads use window coordinates inside an offset reading column',
    (tester) async {
      final app = await pumpChat(
        tester,
        targetUserId: null,
        messages: [makeChatMessage(1)],
      );
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(1600, 800);
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(tester.view.resetPhysicalSize);
      await tester.binding.setSurfaceSize(const Size(1600, 800));
      app.navigator.currentState!.push(
        MaterialPageRoute<void>(
          builder: (context) => Align(
            alignment: Alignment.centerRight,
            child: SizedBox(
              width: 500,
              child: MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(size: const Size(500, 800)),
                child: const MessagesPage(targetUserId: 2),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpAndSettle();
      expect(app.repo.batches, [
        [1],
      ]);
    },
  );
}

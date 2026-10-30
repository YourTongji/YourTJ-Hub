import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/theme_mode.dart';

import '../test/pages_behavior_test.dart' show MemTokenStorage;

/// Full app, real HTTP and native keyboard. Use an isolated seed-demo server;
/// messages are sent only between its demo accounts. Supply a live local forum
/// session JWT through YOURTJ_TEST_TOKEN; login/captcha are outside this journey.
/// No HTTP repository mocks.
/// With YOURTJ_TEST_NATIVE_WINDOW=true, change Duo closed -> open -> closed
/// when the test prints each transition request. Screenshots stay outside Git.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  const enabled = bool.fromEnvironment('YOURTJ_TEST_LOCAL_CHAT');
  const nativeWindow = bool.fromEnvironment('YOURTJ_TEST_NATIVE_WINDOW');
  const token = String.fromEnvironment('YOURTJ_TEST_TOKEN');

  testWidgets('chat survives keyboard and Duo changes then sends over HTTP', (
    tester,
  ) async {
    expect(
      ['localhost', '127.0.0.1', '::1'],
      contains(Uri.parse(AppConfig.apiBaseUrl).host),
      reason: 'Only an isolated local server may receive these test messages',
    );
    expect(token, isNotEmpty);
    final storage = MemTokenStorage(token);
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(storage),
        // Notification permission is a separate native acceptance journey.
        pushBootstrapProvider.overrideWith((_) {}),
      ],
    );

    Future<void> until(
      bool Function() ready,
      String description, {
      Duration timeout = const Duration(seconds: 45),
    }) async {
      final deadline = DateTime.now().add(timeout);
      while (!ready() && DateTime.now().isBefore(deadline)) {
        await tester.pump(const Duration(milliseconds: 100));
      }
      expect(ready(), isTrue, reason: description);
      expect(tester.takeException(), isNull);
    }

    Future<void> capture(String name) async {
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot(name);
      }
    }

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const GfApp(locale: Locale('en')),
      ),
    );
    appRouter.go('/messages?userId=2&username=demo2');
    final field = find.descendant(
      of: find.byType(GfChatInput),
      matching: find.byType(TextField),
    );
    await until(() => field.evaluate().length == 1, 'real chat opens');

    for (final brightness in Brightness.values) {
      container
          .read(themeModeProvider.notifier)
          .setMode(
            brightness == Brightness.dark ? ThemeMode.dark : ThemeMode.light,
          );
      await tester.pumpAndSettle();
      final text =
          'Duo acceptance ${brightness.name} ${DateTime.now().millisecondsSinceEpoch}';
      await tester.tap(field);
      await tester.enterText(field, text);
      final controller = tester.widget<TextField>(field).controller!;
      var previous = MediaQueryData.fromView(tester.view);
      for (var pose = 0; pose < (nativeWindow ? 3 : 1); pose++) {
        if (pose > 0) {
          debugPrint(
            'CHAT E2E: change Duo to ${pose == 1 ? "OPEN" : "CLOSED"} (${brightness.name})',
          );
          await until(
            () => MediaQueryData.fromView(tester.view).size != previous.size,
            'native Duo pose changes',
            timeout: const Duration(minutes: 3),
          );
        }
        // Re-tapping an already focused iOS field opens its native selection
        // menu. Keep the existing input focus across the window transition.
        if (tester.view.viewInsets.bottom == 0) await tester.tap(field);
        await until(
          () => tester.view.viewInsets.bottom > 0,
          'native keyboard is visible',
        );
        await tester.pumpAndSettle();
        final window = MediaQueryData.fromView(tester.view);
        previous = window;
        expect(tester.widget<TextField>(field).controller, same(controller));
        expect(controller.text, text);
        final send = find.byKey(const Key('chat-send'));
        expect(send.hitTestable(), findsOneWidget);
        expect(field.hitTestable(), findsOneWidget);
        expect(
          tester.getRect(send).bottom,
          lessThanOrEqualTo(window.size.height - window.viewInsets.bottom),
        );
        expect(tester.takeException(), isNull);
        await capture('1436-chat-${brightness.name}-pose-$pose');
      }
      final send = find.byKey(const Key('chat-send'));
      await until(
        () => tester.widget<IconButton>(send).onPressed != null,
        'server conversation state permits send',
      );
      await tester.tap(send);
      await until(
        () => controller.text.isEmpty,
        'server acknowledgement clears sent draft',
      );
      expect(find.text(text), findsOneWidget);
      final response = await container
          .read(pageRepositoryProvider)
          .fetch('/messages');
      expect(
        response.toJson().toString(),
        contains(text),
        reason: 'fresh real HTTP read contains committed message',
      );
      await capture('1436-chat-${brightness.name}-sent');
    }
    expect(find.byType(MessagesPage), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
    container.dispose();
  }, skip: !enabled);
}

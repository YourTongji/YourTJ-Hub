import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:integration_test/integration_test.dart';

/// Opt-in registration against an isolated loopback backend; never a shared site.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  const enabled = bool.fromEnvironment('YOURTJ_TEST_LOCAL_REGISTRATION');
  testWidgets('real registration accepts and persists the issued session', (
    tester,
  ) async {
    expect(
      ['localhost', '127.0.0.1', '10.0.2.2', '::1'],
      contains(Uri.parse(AppConfig.apiBaseUrl).host),
      reason: 'Registration may only mutate an isolated local backend',
    );
    final container = ProviderContainer();
    await container.read(tokenStorageProvider).clear();
    var captureSurfaceReady = false;
    Future<void> capture(String name) async {
      if (!const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) return;
      if (Platform.isAndroid && !captureSurfaceReady) {
        await binding.convertFlutterSurfaceToImage();
        captureSurfaceReady = true;
      }
      await tester.pump(const Duration(milliseconds: 300));
      await binding.takeScreenshot(name);
    }

    try {
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const GfApp(locale: Locale('en')),
        ),
      );
      appRouter.go('/login');
      Future<void> until(bool Function() ready, String reason) async {
        final deadline = DateTime.now().add(const Duration(seconds: 30));
        while (!ready() && DateTime.now().isBefore(deadline)) {
          await tester.pump(const Duration(milliseconds: 200));
        }
        if (!ready()) {
          debugPrint(
            tester
                .widgetList<Text>(find.byType(Text))
                .map((w) => w.data ?? '')
                .join(' | '),
          );
          await capture('real-registration-failed');
        }
        expect(ready(), isTrue, reason: reason);
        expect(tester.takeException(), isNull);
      }

      await until(
        () =>
            find.byType(LoginPage).evaluate().isNotEmpty &&
            find.byType(TextField).evaluate().isNotEmpty,
        'Real login options load',
      );
      final l = AppLocalizations.of(tester.element(find.byType(LoginPage)));
      await tester.ensureVisible(find.text(l.loginModeRegister).first);
      await tester.pumpAndSettle();
      expect(
        find.text(l.loginModeRegister).first.hitTestable(),
        findsOneWidget,
      );
      await tester.tap(find.text(l.loginModeRegister).first);
      Finder field(String label) => find.byWidgetPredicate(
        (w) => w is TextField && w.decoration?.labelText == label,
      );
      await until(
        () => field(l.authEmail).evaluate().isNotEmpty,
        'The backend permits password registration',
      );
      final username = 'qanative${DateTime.now().millisecondsSinceEpoch}';
      for (final entry in {
        l.authUsername: username,
        l.authEmail: '$username@example.invalid',
        l.authPassword: 'QaPass123456',
        l.authConfirmPassword: 'QaPass123456',
      }.entries) {
        await tester.ensureVisible(field(entry.key));
        await tester.pumpAndSettle();
        await tester.enterText(field(entry.key), entry.value);
        await tester.pump(const Duration(milliseconds: 300));
        expect(
          tester.widget<TextField>(field(entry.key)).controller!.text,
          entry.value,
        );
      }
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pump(const Duration(milliseconds: 600));
      final consent = find.byType(CheckboxListTile);
      if (consent.evaluate().isNotEmpty &&
          !tester.widget<CheckboxListTile>(consent).value!) {
        await tester.ensureVisible(consent);
        await tester.tap(
          find.descendant(of: consent, matching: find.byType(Checkbox)),
        );
      }
      final submit = find.byWidgetPredicate(
        (w) => w is GfButton && w.size == GfButtonSize.extraLarge,
      );
      await tester.ensureVisible(submit);
      await tester.pumpAndSettle();
      expect(submit.hitTestable(), findsOneWidget);
      await capture('real-registration-form');
      await tester.tap(submit);
      await until(
        () =>
            find.byType(LoginPage).evaluate().isEmpty &&
            find.byType(HomePage).evaluate().isNotEmpty,
        'Accepted registration enters authenticated home',
      );
      expect(await container.read(tokenStorageProvider).read(), isNotEmpty);
      final settings = parsePageProps<SettingsPageProps>(
        await container.read(pageRepositoryProvider).fetch('/settings'),
      )!;
      expect(settings.user.username, username);
      await tester.pump(const Duration(seconds: 1));
      await capture('real-registration-home');
      debugPrint(
        'Real registration: issued session stored, authenticated settings confirms the new owner.',
      );
    } finally {
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(milliseconds: 300));
      container.dispose();
    }
  }, skip: !enabled);
}

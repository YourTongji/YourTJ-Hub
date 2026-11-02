import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/email_verification_recovery.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets(
        'recovery preserves its page and serializes navigation: $locale $brightness',
        (tester) async {
          tester.view.physicalSize = const Size(320, 844);
          tester.view.devicePixelRatio = 1;
          tester.platformDispatcher.textScaleFactorTestValue = 2;
          addTearDown(tester.view.reset);
          addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
          final saved = Completer<bool>();
          var saves = 0;
          final controller = TextEditingController(
            text: 'retained unsent text',
          );
          addTearDown(controller.dispose);
          final router = GoRouter(
            routes: [
              GoRoute(
                path: '/',
                builder: (_, _) => Scaffold(
                  body: SingleChildScrollView(
                    child: Column(
                      children: [
                        TextField(controller: controller),
                        EmailVerificationRecovery(
                          compact: true,
                          error: const ApiException(
                            fallbackMessage: '',
                            messageCode: 'permission.emailRequired',
                          ),
                          preserveDraft: () {
                            saves++;
                            return saved.future;
                          },
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              GoRoute(
                path: '/settings/account',
                builder: (_, _) =>
                    const Scaffold(body: Text('account security')),
              ),
            ],
          );
          addTearDown(router.dispose);
          await tester.pumpWidget(
            MaterialApp.router(
              routerConfig: router,
              theme: gfThemeData(brightness),
              locale: Locale(locale),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
            ),
          );
          await tester.pumpAndSettle();
          final action = find.byType(GfButton);
          final rect = tester.getRect(action);
          expect(rect.width, lessThanOrEqualTo(320));
          expect(rect.height, greaterThanOrEqualTo(44));
          await tester.tap(action);
          await tester.pump();
          await tester.tap(action);
          expect(saves, 1);
          expect(router.state.uri.path, '/');
          saved.complete(true);
          await tester.pumpAndSettle();
          expect(router.state.uri.path, '/settings/account');
          router.pop();
          await tester.pumpAndSettle();
          expect(controller.text, 'retained unsent text');
          expect(saves, 1);
          expect(tester.takeException(), isNull);
          await tester.pumpWidget(const SizedBox());
        },
      );
    }
  }
  for (final code in [
    null,
    'auth.required',
    'permission.userFrozen',
    'unknown',
  ]) {
    testWidgets('email recovery does not misclassify $code', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: EmailVerificationRecovery(
            error: code == null
                ? null
                : ApiException(fallbackMessage: '', messageCode: code),
          ),
        ),
      );
      expect(find.byType(GfButton), findsNothing);
    });
  }
}

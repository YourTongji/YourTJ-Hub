import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import '../test/login_input_ime_flags_test.dart' show pumpLoginInputs;

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  const baseline = bool.fromEnvironment('YOURTJ_GUIDANCE_BASELINE');
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets('registration guidance $locale ${brightness.name}', (
        tester,
      ) async {
        await pumpLoginInputs(
          tester,
          locale: locale,
          brightness: brightness,
          width: 320,
          scale: 2,
          native: true,
        );
        final l10n = AppLocalizations.of(
          tester.element(find.byType(LoginPage)),
        );
        await tester.ensureVisible(find.text(l10n.loginModeRegister).first);
        await tester.tap(find.text(l10n.loginModeRegister).first);
        await tester.pumpAndSettle();
        if (!baseline) {
          final password = find.byWidgetPredicate(
            (widget) =>
                widget is TextField &&
                widget.decoration?.labelText == l10n.authPassword,
          );
          expect(
            tester.widget<TextField>(password).decoration?.helperText,
            contains('6–64'),
          );
          await tester.ensureVisible(
            find.text(l10n.authRegistrationPasswordRules),
          );
          await tester.pumpAndSettle();
        }
        await tester.pump(const Duration(milliseconds: 400));
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot(
            'register-guidance-${baseline ? "before" : "after"}-$locale-${brightness.name}',
          );
        }
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        await tester.pumpAndSettle();
      });
    }
  }
}

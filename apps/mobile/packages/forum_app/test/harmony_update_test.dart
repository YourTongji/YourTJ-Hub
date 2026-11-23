import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/info/site_info_page.dart';
import 'package:forum_app/src/updates/update_host.dart';

void main() {
  final harmony = TargetPlatform.values.where((value) => value.name == 'ohos');
  for (final succeeds in [true, false]) {
    testWidgets(
      'HarmonyOS explicit update opens AppTest; launch succeeds=$succeeds',
      (tester) async {
        const launcher = MethodChannel('plugins.flutter.io/url_launcher');
        final opened = <String>[];
        tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          launcher,
          (call) async {
            if (call.method == 'launch') {
              opened.add((call.arguments as Map)['url'] as String);
            }
            return succeeds;
          },
        );
        addTearDown(
          () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
            launcher,
            null,
          ),
        );
        final navigator = GlobalKey<NavigatorState>();
        final host = appUpdateHostKey;
        await tester.pumpWidget(
          MaterialApp(
            navigatorKey: navigator,
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (_, child) => MobileUpdateHost(
              key: host,
              navigatorKey: navigator,
              child: child!,
            ),
            home: const SiteInfoIndexPage(),
          ),
        );
        await tester.pumpAndSettle();
        expect(supportsMobileUpdates, isTrue);
        expect(supportsMobileReleaseNotes, isTrue);
        expect(opened, isEmpty);
        host.currentState!.didChangeAppLifecycleState(
          AppLifecycleState.resumed,
        );
        await tester.pumpAndSettle();
        expect(opened, isEmpty);
        final l10n = AppLocalizations.of(navigator.currentContext!);
        expect(find.text(l10n.updateCheck), findsOneWidget);
        expect(find.text(l10n.releaseNotesHistory), findsOneWidget);
        await tester.tap(find.text(l10n.updateCheck));
        await tester.pumpAndSettle();
        expect(opened, [harmonyAppTestGroupUrl]);
        expect(find.text(l10n.updateLatest), findsNothing);
        expect(
          find.text(l10n.updateFailed),
          succeeds ? findsNothing : findsOneWidget,
        );
        // Failed launches release the guard so the user's next check can retry.
        await host.currentState!.check(force: true);
        expect(opened.length, 2);
      },
      // Stock Flutter has no OHOS enum, but even skipped tests need a variant.
      variant: TargetPlatformVariant(
        harmony.isEmpty ? {TargetPlatform.android} : harmony.toSet(),
      ),
      skip: harmony.isEmpty,
    );
  }
}

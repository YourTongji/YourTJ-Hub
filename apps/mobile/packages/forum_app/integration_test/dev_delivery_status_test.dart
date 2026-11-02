import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/updates/preview_status.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  testWidgets('dev identity and status remain informational on native widths', (
    tester,
  ) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    expect(AppConfig.buildId, '202611030001');
    final source = AppConfig.sourceSha;
    var converted = false;
    for (final brightness in Brightness.values) {
      for (final width in [320.0, 900.0]) {
        for (final state in ['current', 'building', 'ready', 'offline']) {
          await tester.pumpWidget(const SizedBox());
          await tester.pumpWidget(
            MaterialApp(
              locale: const Locale('zh'),
              theme: gfThemeData(brightness),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: TextScaler.linear(width == 320 ? 2 : 1)),
                child: Center(
                  child: SizedBox(width: width, child: child),
                ),
              ),
              home: PreviewStatusHost(
                fetchStatus: () async {
                  if (state == 'offline') {
                    throw const SocketException('isolated offline fixture');
                  }
                  return {
                    'schema': 2,
                    'updatedAt': 1,
                    'source': state == 'current' ? source : 'b' * 40,
                    'platforms': {
                      'ios': {'source': state == 'ready' ? 'b' * 40 : source},
                    },
                  };
                },
                child: Scaffold(
                  appBar: GfAppBar(title: const GfText('YourTJ')),
                  body: const Center(child: Text('开发页面继续可用')),
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          expect(find.text('开发页面继续可用'), findsOneWidget);
          expect(find.text(AppConfig.previewLabel), findsOneWidget);
          expect(tester.takeException(), isNull);
          if (Platform.isAndroid && !converted) {
            await binding.convertFlutterSurfaceToImage();
            converted = true;
          }
          final l10n = await AppLocalizations.delegate.load(const Locale('zh'));
          if (state != 'current') {
            expect(find.text(l10n.previewStatusLink), findsOneWidget);
            expect(
              find.text(
                state == 'building'
                    ? l10n.previewBuildPending
                    : state == 'ready'
                    ? l10n.previewStale
                    : l10n.previewStatusUnknown,
              ),
              findsOneWidget,
            );
          } else {
            expect(find.text(l10n.previewStatusLink), findsNothing);
          }
          await tester.pump(const Duration(milliseconds: 250));
          await tester.runAsync(
            () => Future<void>.delayed(const Duration(milliseconds: 200)),
          );
          await binding.takeScreenshot(
            'dev-${brightness.name}-${width == 320 ? '320' : 'native'}-$state',
          );
        }
      }
    }
    await tester.pumpWidget(const SizedBox());
    debugDefaultTargetPlatformOverride = null;
  });
}

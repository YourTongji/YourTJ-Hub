import 'dart:io';
import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/updates/android_release.dart';
import 'package:forum_app/src/updates/release_notes_page.dart';
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
    final previousHttp = HttpOverrides.current;
    HttpOverrides.global = _OfflineHistory();
    try {
      SharedPreferences.setMockInitialValues({
        'mobile.releaseNotes.${Uri.encodeComponent(mobileReleaseNotesUrl)}.json':
            jsonEncode({
              'schemaVersion': 1,
              'releases': [
                for (final build in [33, 43])
                  {
                    'version': build == 43 ? '1.0.18' : '1.0.17',
                    'buildNumber': build,
                    if (build == 43) 'displayBuild': '202611030003',
                    'channels': ['ios-app-store'],
                    'highlights': [
                      {
                        'id': 'entry-$build',
                        'title': '版本编号',
                        'summary': '发布历史保留每个版本的编号。',
                        'platforms': ['ios-app-store'],
                        'kind': 'improvement',
                      },
                    ],
                    'breaking': [],
                    'requiredActions': [],
                    'testflightNotes': [],
                  },
              ],
            }),
      });
      for (final brightness in Brightness.values) {
        for (final width in [320.0, 900.0]) {
          await tester.pumpWidget(const SizedBox());
          await tester.pumpWidget(
            MaterialApp(
              debugShowCheckedModeBanner: false,
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
              home: const ReleaseNotesPage(),
            ),
          );
          await tester.pumpAndSettle();
          expect(
            find.textContaining('202611030003', findRichText: true),
            findsOneWidget,
          );
          expect(
            find.textContaining('(33)', findRichText: true),
            findsOneWidget,
          );
          expect(tester.takeException(), isNull);
          await tester.pump(const Duration(milliseconds: 250));
          await tester.runAsync(
            () => Future<void>.delayed(const Duration(milliseconds: 200)),
          );
          await binding.takeScreenshot(
            'history-${brightness.name}-${width == 320 ? '320' : 'native'}',
          );
        }
      }
      await tester.pumpWidget(const SizedBox());
    } finally {
      HttpOverrides.global = previousHttp;
    }
    await tester.pumpWidget(const SizedBox());
    debugDefaultTargetPlatformOverride = null;
  });
}

// Exercise cached history without contacting public services or replacing
// the date-ID fixture with an unrelated remote catalog.
class _OfflineHistory extends HttpOverrides {
  @override
  HttpClient createHttpClient(SecurityContext? context) {
    throw const SocketException('Isolated release-history fixture');
  }
}

import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/updates/android_release.dart';
import 'package:forum_app/src/updates/release_notes_page.dart';
import 'package:forum_app/src/updates/update_host.dart';
import 'package:forum_app/src/updates/update_prompt_sheet.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher_platform_interface/link.dart';
import 'package:url_launcher_platform_interface/url_launcher_platform_interface.dart';

// The real native page reads its local catalog cache. Block refresh traffic so
// these checks need neither production access nor a published test release.
class _Offline extends HttpOverrides {
  @override
  HttpClient createHttpClient(SecurityContext? context) =>
      throw const SocketException('Offline release-history test');
}

Map<String, Object> _release(int build, String channel, String title) => {
  'version': '1.0.$build',
  'buildNumber': build,
  'channels': [channel],
  'highlights': [
    if (channel != 'ios-testflight')
      {
        'id': 'note-$build',
        'title': title,
        'summary': '$title details',
        'kind': 'feature',
        'platforms': [channel],
      },
  ],
  'breaking': <Object>[],
  'requiredActions': <Object>[],
  'testflightNotes': [
    if (channel == 'ios-testflight') {'id': 'beta-$build', 'text': title},
  ],
};

class _Launcher extends UrlLauncherPlatform {
  @override
  LinkDelegate? get linkDelegate => null;

  final urls = <String>[];
  bool succeeds = true;

  @override
  Future<bool> launchUrl(String url, LaunchOptions options) async {
    expect(options.mode, PreferredLaunchMode.externalApplication);
    urls.add(url);
    return succeeds;
  }
}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  Future<void> showHistory(
    WidgetTester tester,
    List<Map<String, Object>> releases,
  ) async {
    final previous = HttpOverrides.current;
    HttpOverrides.global = _Offline();
    addTearDown(() => HttpOverrides.global = previous);
    final data = jsonEncode({'schemaVersion': 1, 'releases': releases});
    SharedPreferences.setMockInitialValues({
      for (final url in [mobileReleaseNotesUrl, githubReleaseNotesUrl])
        'mobile.releaseNotes.${Uri.encodeComponent(url)}.json': data,
    });
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const ReleaseNotesPage(),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  }

  testWidgets(
    'history shows public platform notes without a channel picker',
    (tester) async {
      await showHistory(tester, [
        _release(10, 'ios-app-store', 'Public iOS change'),
        _release(11, 'android', 'Public Android change'),
        _release(12, 'ios-testflight', 'Beta testing instructions'),
      ]);
      expect(find.byType(SegmentedButton<String>), findsNothing);
      expect(find.text('TestFlight'), findsNothing);
      expect(find.text('App Store'), findsNothing);
      expect(find.text('Beta testing instructions'), findsNothing);
      expect(
        find.text(
          Platform.isIOS ? 'Public iOS change' : 'Public Android change',
        ),
        findsOneWidget,
      );
      expect(
        find.text(
          Platform.isIOS ? 'Public Android change' : 'Public iOS change',
        ),
        findsNothing,
      );
    },
    skip: !Platform.isIOS && !Platform.isAndroid,
  );

  testWidgets(
    'beta-only catalog leaves public history empty',
    (tester) async {
      await showHistory(tester, [
        _release(12, 'ios-testflight', 'Beta testing instructions'),
      ]);
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.releaseNotesEmpty), findsOneWidget);
      expect(find.text('Beta testing instructions'), findsNothing);
      expect(find.byType(SegmentedButton<String>), findsNothing);
    },
    skip: !Platform.isIOS && !Platform.isAndroid,
  );

  testWidgets(
    'TestFlight delegates manual updates without a custom notes prompt',
    (tester) async {
      final previousHttp = HttpOverrides.current;
      final previousLauncher = UrlLauncherPlatform.instance;
      final launcher = _Launcher();
      UrlLauncherPlatform.instance = launcher;
      HttpOverrides.global = _Offline();
      final messenger =
          TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
      messenger.setMockMethodCallHandler(updateChannel, (call) async {
        expect(call.method, 'getInfo');
        return {
          'channel': 'ios-testflight',
          'version': '1.0.11',
          'buildNumber': 11,
        };
      });
      addTearDown(() {
        HttpOverrides.global = previousHttp;
        UrlLauncherPlatform.instance = previousLauncher;
        messenger.setMockMethodCallHandler(updateChannel, null);
      });
      final data = jsonEncode({
        'schemaVersion': 1,
        'historyCoverage': {
          'source': 'github-release-receipts',
          'publishedAt': '2026-10-03T00:00:00Z',
          'byChannel': {
            'ios-testflight': {
              'completeFromBuild': 11,
              'throughBuild': 12,
              'coveredBuilds': [12],
            },
          },
        },
        'releases': [
          _release(12, 'ios-testflight', 'Beta testing instructions'),
        ],
      });
      SharedPreferences.setMockInitialValues({
        for (final url in [mobileReleaseNotesUrl, githubReleaseNotesUrl])
          'mobile.releaseNotes.${Uri.encodeComponent(url)}.json': data,
      });
      final navigator = GlobalKey<NavigatorState>();
      final host = GlobalKey<MobileUpdateHostState>();
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: navigator,
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MobileUpdateHost(
            key: host,
            navigatorKey: navigator,
            child: child!,
          ),
          home: const Scaffold(body: Text('App content')),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(UpdatePromptSheet), findsNothing);
      expect(launcher.urls, isEmpty);
      await host.currentState!.check(force: true);
      await tester.pumpAndSettle();
      expect(launcher.urls, [testFlightAppUrl]);
      expect(find.byType(UpdatePromptSheet), findsNothing);
      expect(find.text('Beta testing instructions'), findsNothing);
      launcher.succeeds = false;
      await host.currentState!.check(force: true);
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.updateFailed), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
    skip: !Platform.isIOS,
  );
}

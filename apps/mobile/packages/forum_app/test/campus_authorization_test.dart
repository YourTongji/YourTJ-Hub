import 'package:core/core.dart';
import 'package:dio/dio.dart' as dio;
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/admin/admin_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;
import 'fixtures/admin_webview_fakes.dart';

void main() {
  testWidgets(
    'native campus handoff keeps Bearer first-party and returns after callback',
    (tester) async {
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final previous = WebViewPlatform.instance;
      final platform = TestWebPlatform();
      WebViewPlatform.instance = platform;
      addTearDown(() {
        if (previous != null) WebViewPlatform.instance = previous;
      });
      final storage = MemoryTokenStorage();
      await storage.write('native-test-session');
      final uri = Uri.parse(
        'https://api.tongji.edu.cn/keycloak/realms/OpenPlatform/protocol/openid-connect/auth?response_type=code&state=test',
      );
      final navigator = GlobalKey<NavigatorState>();
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          apiClientProvider.overrideWithValue(
            GfApiClient(
              dio: dio.Dio(),
              tokenStorage: storage,
              baseUrl: 'https://forum.example',
            ),
          ),
        ],
      );
      addTearDown(container.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            navigatorKey: navigator,
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(body: Text('native campus')),
          ),
        ),
      );
      final completed = navigator.currentState!.push<bool>(
        MaterialPageRoute(
          builder: (_) => AdminPage(
            target: MobileWebTarget.campus,
            campusAuthorizationUrl: uri,
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        platform.controller.lastRequest!.uri.path,
        '/api/auth/mobile-web-session',
      );
      expect(
        platform.controller.lastRequest!.headers['Authorization'],
        'Bearer native-test-session',
      );
      expect(
        await platform.delegate.request!(
          const NavigationRequest(
            url: 'https://forum.example/campus',
            isMainFrame: true,
          ),
        ),
        NavigationDecision.prevent,
      );
      await tester.pump();
      expect(platform.controller.lastRequest!.uri, uri);
      expect(platform.controller.lastRequest!.headers, isEmpty);
      expect(
        await platform.delegate.request!(
          const NavigationRequest(
            url: 'https://api.tongji.edu.cn.evil.test/',
            isMainFrame: true,
          ),
        ),
        NavigationDecision.prevent,
      );
      expect(
        await platform.delegate.request!(
          const NavigationRequest(
            url:
                'https://forum.example/api/campus/tongji/callback?code=test&state=test',
            isMainFrame: true,
          ),
        ),
        NavigationDecision.navigate,
      );
      expect(
        await platform.delegate.request!(
          const NavigationRequest(
            url: 'https://forum.example/campus?authorization=ready',
            isMainFrame: true,
          ),
        ),
        NavigationDecision.prevent,
      );
      await tester.pumpAndSettle();
      expect(await completed, isTrue);
      expect(find.text('native campus'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      debugDefaultTargetPlatformOverride = null;
    },
  );
}

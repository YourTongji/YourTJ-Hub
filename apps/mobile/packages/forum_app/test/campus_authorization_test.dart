import 'package:core/core.dart';
import 'package:dio/dio.dart' as dio;
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:webview_flutter/webview_flutter.dart' show WebViewWidget;
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/admin/admin_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;
import 'fixtures/admin_webview_fakes.dart';

void main() {
  testWidgets(
    'native campus tolerates iOS handoff cancellation but preserves real failures',
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
      Future<void> handOffToSchool() async {
        expect(
          await platform.delegate.request!(
            const NavigationRequest(
              url: 'https://forum.example/campus',
              isMainFrame: true,
            ),
          ),
          NavigationDecision.prevent,
        );
      }

      Future<void> retry() async {
        expect(find.text('Failed to load'), findsOneWidget);
        expect(find.byType(WebViewWidget), findsNothing);
        await tester.tap(find.text('Retry'));
        await tester.pumpAndSettle();
        expect(find.byType(WebViewWidget), findsOneWidget);
        expect(
          platform.controller.lastRequest!.uri.path,
          '/api/auth/mobile-web-session',
        );
      }

      // A cancelled load before the app has handed off is still a failure.
      platform.delegate.resourceError!(
        const WebResourceError(
          errorCode: -999,
          description: 'Cancelled before handoff',
          isForMainFrame: true,
        ),
      );
      await tester.pump();
      await retry();
      // Reject invalid sessions; a navigation exception must not bypass HTTP 401.
      platform.delegate.httpError!(
        HttpResponseError(
          request: WebResourceRequest(
            uri: platform.controller.lastRequest!.uri,
          ),
          response: const WebResourceResponse(uri: null, statusCode: 401),
        ),
      );
      await tester.pump();
      await retry();
      await handOffToSchool();
      // WebKit may identify the original request, the redirect, or omit the URL.
      for (final code in [-999, 102]) {
        for (final url in <String?>[
          'https://forum.example/api/auth/mobile-web-session?target=campus',
          'https://forum.example/campus',
          null,
        ]) {
          platform.delegate.resourceError!(
            WebResourceError(
              errorCode: code,
              description: 'Navigation cancelled during handoff',
              isForMainFrame: true,
              url: url,
            ),
          );
          await tester.pump();
          expect(
            find.byType(WebViewWidget),
            findsOneWidget,
            reason: 'Expected handoff cancellation $code at $url',
          );
          expect(find.text('Retry'), findsNothing);
        }
      }
      // Once the school page starts navigating, the handoff window is closed:
      // a URL-less cancellation is then a real school failure with retry.
      expect(
        await platform.delegate.request!(
          NavigationRequest(url: uri.toString(), isMainFrame: true),
        ),
        NavigationDecision.navigate,
      );
      platform.delegate.resourceError!(
        const WebResourceError(
          errorCode: -999,
          description: 'School page cancelled without URL',
          isForMainFrame: true,
        ),
      );
      await tester.pump();
      await retry();
      await handOffToSchool();
      // Only the replaced handoff is exempt: school cancellations, offline and
      // certificate failures still offer retry, without claiming state expiry.
      for (final code in [-999, 102, -1009, -1202]) {
        platform.delegate.resourceError!(
          WebResourceError(
            errorCode: code,
            description: 'School page failed',
            isForMainFrame: true,
            url: uri.toString(),
          ),
        );
        await tester.pump();
        await retry();
        await handOffToSchool();
      }
      platform.delegate.resourceError!(
        const WebResourceError(
          errorCode: -1009,
          description: 'An image failed',
          isForMainFrame: false,
        ),
      );
      await tester.pump();
      expect(find.byType(WebViewWidget), findsOneWidget);
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

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
    'console HTTP status without a request still offers retry',
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
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const AdminPage(target: MobileWebTarget.admin),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        platform.controller.lastRequest!.uri.path,
        '/api/auth/mobile-web-session',
      );
      // WebKit omits the request: the tracked console main frame attributes
      // the status to the page itself (#970).
      expect(
        await platform.delegate.request!(
          const NavigationRequest(
            url: 'https://forum.example/admin',
            isMainFrame: true,
          ),
        ),
        NavigationDecision.navigate,
      );
      platform.delegate.httpError!(
        const HttpResponseError(
          response: WebResourceResponse(uri: null, statusCode: 500),
        ),
      );
      await tester.pump();
      expect(
        find.text(
          'The admin console is unavailable. Check your permissions, '
          'session and connection, then retry.',
        ),
        findsOneWidget,
      );
      expect(find.byType(WebViewWidget), findsNothing);
      await tester.tap(find.text('Retry'));
      await tester.pumpAndSettle();
      expect(find.byType(WebViewWidget), findsOneWidget);
      // Outside the console surfaces the status must not blank the page.
      expect(
        await platform.delegate.request!(
          const NavigationRequest(
            url: 'https://forum.example/topics',
            isMainFrame: true,
          ),
        ),
        NavigationDecision.navigate,
      );
      platform.delegate.httpError!(
        const HttpResponseError(
          response: WebResourceResponse(uri: null, statusCode: 500),
        ),
      );
      await tester.pump();
      expect(find.byType(WebViewWidget), findsOneWidget);
      expect(find.text('Retry'), findsNothing);
      debugDefaultTargetPlatformOverride = null;
    },
  );
}

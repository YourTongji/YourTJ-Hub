import 'package:core/core.dart';
import 'package:dio/dio.dart' as dio;
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/admin/admin_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:webview_flutter/webview_flutter.dart';

import 'fixtures/admin_webview_fakes.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  final harmony = TargetPlatform.values
      .where((value) => value.name == 'ohos')
      .firstOrNull;
  testWidgets(
    'HarmonyOS opens the school authorization view and keeps the bearer on the first-party handoff',
    (tester) async {
      debugDefaultTargetPlatformOverride = harmony;
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final previous = WebViewPlatform.instance;
      final platform = TestWebPlatform();
      WebViewPlatform.instance = platform;
      addTearDown(() {
        if (previous != null) WebViewPlatform.instance = previous;
      });
      final storage = MemoryTokenStorage();
      await storage.write('synthetic-test-session');
      final school = Uri.parse(
        'https://api.tongji.edu.cn/keycloak/realms/OpenPlatform/protocol/openid-connect/auth?response_type=code&state=synthetic',
      );
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
            home: AdminPage(
              target: MobileWebTarget.campus,
              campusAuthorizationUrl: school,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(WebViewWidget), findsOneWidget);
      expect(
        platform.controller.lastRequest!.headers['Authorization'],
        'Bearer synthetic-test-session',
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
      expect(platform.controller.lastRequest!.uri, school);
      expect(platform.controller.lastRequest!.headers, isEmpty);
      debugDefaultTargetPlatformOverride = null;
    },
    skip: harmony == null,
  );
}

import 'package:core/core.dart';
import 'package:dio/dio.dart' as dio;
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:webview_flutter/webview_flutter.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/admin/admin_page.dart';
import 'package:forum_app/src/pages/campus/campus_connection.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/admin_webview_fakes.dart';
import 'fixtures/campus_fixtures.dart';

class _Repository extends FakeCampusRepository {
  final schoolUri = Uri.parse(
    'https://api.tongji.edu.cn/keycloak/realms/OpenPlatform/protocol/openid-connect/auth?response_type=code&state=synthetic',
  );
  int starts = 0;
  @override
  Future<Uri> start(String mode, {dio.CancelToken? cancelToken}) async {
    starts++;
    return schoolUri;
  }
}

Future<void> _pumpRoute(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 600));
  await tester.pump(const Duration(milliseconds: 600));
}

void main() {
  testWidgets(
    'campus authentication covers the shell and retains its branch on iOS and Android',
    (tester) async {
      for (final platform in [TargetPlatform.iOS, TargetPlatform.android]) {
        addTearDown(() async {
          await tester.pumpWidget(const SizedBox());
          await _pumpRoute(tester);
        });
        debugDefaultTargetPlatformOverride = platform;
        addTearDown(() => debugDefaultTargetPlatformOverride = null);
        final previous = WebViewPlatform.instance;
        final web = TestWebPlatform();
        WebViewPlatform.instance = web;
        addTearDown(() {
          if (previous != null) WebViewPlatform.instance = previous;
        });
        final repository = _Repository();
        final controller = CampusController(repository);
        await controller.refresh();
        final token = CampusMemoryTokenStorage()
          ..value = 'synthetic-test-session';
        final container = ProviderContainer(
          overrides: [
            campusRepositoryProvider.overrideWithValue(repository),
            campusControllerProvider.overrideWith((ref) => controller),
            tokenStorageProvider.overrideWithValue(token),
            apiClientProvider.overrideWithValue(
              GfApiClient(
                dio: dio.Dio(),
                tokenStorage: token,
                baseUrl: 'https://forum.example',
              ),
            ),
          ],
        );
        final root = GlobalKey<NavigatorState>();
        final branch = GlobalKey<NavigatorState>();
        final pageScroll = ScrollController();
        addTearDown(pageScroll.dispose);
        final router = GoRouter(
          navigatorKey: root,
          initialLocation: '/campus',
          routes: [
            ShellRoute(
              navigatorKey: branch,
              builder: (_, state, child) => Scaffold(
                body: child,
                bottomNavigationBar: const SizedBox(
                  height: 72,
                  child: Center(child: Text('campus dock')),
                ),
              ),
              routes: [
                GoRoute(
                  path: '/campus',
                  builder: (_, _) => ListView(
                    controller: pageScroll,
                    children: const [
                      CampusConnection(compact: true),
                      SizedBox(height: 1500),
                    ],
                  ),
                ),
              ],
            ),
          ],
        );
        var disposed = false;
        void dispose() {
          if (disposed) return;
          disposed = true;
          router.dispose();
          container.dispose();
        }

        addTearDown(dispose);
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              theme: gfThemeData(Brightness.light),
              locale: const Locale('en'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
            ),
          ),
        );
        await _pumpRoute(tester);
        expect(find.text('campus dock').hitTestable(), findsOneWidget);
        final l = AppLocalizations.of(
          tester.element(find.byType(CampusConnection)),
        );
        await tester.tap(find.text(l.campusReauthorize));
        await _pumpRoute(tester);
        expect(find.byType(WebViewWidget), findsOneWidget);
        expect(find.text('campus dock').hitTestable(), findsNothing);
        expect(root.currentState!.canPop(), isTrue);
        expect(branch.currentState!.canPop(), isFalse);
        final route =
            ModalRoute.of(tester.element(find.byType(AdminPage)))!
                as MaterialPageRoute;
        expect(route.fullscreenDialog, isTrue);
        expect(repository.starts, 1);
        expect(
          web.controller.lastRequest!.headers['Authorization'],
          'Bearer synthetic-test-session',
        );
        // A failed school page still occupies the whole root route; retry retains it.
        web.delegate.resourceError!(
          const WebResourceError(
            errorCode: -1009,
            description: 'offline fixture',
            isForMainFrame: true,
          ),
        );
        await _pumpRoute(tester);
        expect(find.text('campus dock').hitTestable(), findsNothing);
        await tester.tap(find.text(l.commonRetry));
        await _pumpRoute(tester);
        expect(find.byType(WebViewWidget), findsOneWidget);
        await tester.tap(find.byTooltip(l.commonClose));
        await _pumpRoute(tester);
        expect(find.byType(AdminPage), findsNothing);
        expect(find.text('campus dock').hitTestable(), findsOneWidget);
        expect(find.byType(CampusConnection), findsOneWidget);
        expect(router.state.uri.path, '/campus');
        expect(repository.starts, 1);
        for (final exit in ['back', 'complete']) {
          await tester.tap(find.text(l.campusReauthorize));
          await _pumpRoute(tester);
          expect(find.text('campus dock').hitTestable(), findsNothing);
          if (exit == 'back') {
            await tester.tap(find.byTooltip(l.commonBack));
          } else {
            await web.delegate.request!(
              const NavigationRequest(
                url: 'https://forum.example/campus',
                isMainFrame: true,
              ),
            );
            await web.delegate.request!(
              const NavigationRequest(
                url: 'https://forum.example/campus?authorization=ready',
                isMainFrame: true,
              ),
            );
          }
          await _pumpRoute(tester);
          expect(find.byType(AdminPage), findsNothing);
          expect(find.text('campus dock').hitTestable(), findsOneWidget);
          expect(router.state.uri.path, '/campus');
        }
        expect(repository.starts, 3);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        dispose();
        await _pumpRoute(tester);
        debugDefaultTargetPlatformOverride = null;
      }
    },
  );
}

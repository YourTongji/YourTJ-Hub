import 'dart:async';
import 'dart:io';
import 'package:core/core.dart';
import 'package:dio/dio.dart' as dio;
import 'package:dio/io.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/admin/admin_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/app_locale.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Controller extends PlatformWebViewController {
  _Controller(super.params) : super.implementation();
  Completer<bool>? historyResult;
  int backCalls = 0;
  int loads = 0;
  LoadRequestParams? lastRequest;
  @override
  Future<bool> canGoBack() => historyResult?.future ?? Future.value(false);
  @override
  Future<void> goBack() async {
    backCalls++;
  }

  @override
  Future<void> clearCache() async {}
  @override
  Future<void> clearLocalStorage() async {}
  @override
  Future<void> loadHtmlString(String html, {String? baseUrl}) async {}
  @override
  Future<void> loadRequest(LoadRequestParams params) async {
    loads++;
    lastRequest = params;
  }

  @override
  Future<void> setJavaScriptMode(JavaScriptMode mode) async {}
  @override
  Future<void> setPlatformNavigationDelegate(
    PlatformNavigationDelegate delegate,
  ) async {}
  @override
  Future<void> setOnJavaScriptAlertDialog(
    Future<void> Function(JavaScriptAlertDialogRequest) callback,
  ) async {}
  @override
  Future<void> setOnJavaScriptConfirmDialog(
    Future<bool> Function(JavaScriptConfirmDialogRequest) callback,
  ) async {}
}

class _Delegate extends PlatformNavigationDelegate {
  _Delegate(super.params) : super.implementation();
  NavigationRequestCallback? request;
  @override
  Future<void> setOnNavigationRequest(
    NavigationRequestCallback callback,
  ) async {
    request = callback;
  }

  @override
  Future<void> setOnProgress(ProgressCallback callback) async {}
  @override
  Future<void> setOnWebResourceError(WebResourceErrorCallback callback) async {}
  @override
  Future<void> setOnHttpError(HttpResponseErrorCallback callback) async {}
}

class _Cookies extends PlatformWebViewCookieManager {
  _Cookies(super.params, this.writes) : super.implementation();
  final List<WebViewCookie> writes;
  @override
  Future<bool> clearCookies() async => true;
  @override
  Future<void> setCookie(WebViewCookie cookie) async {
    writes.add(cookie);
  }
}

class _Widget extends PlatformWebViewWidget {
  _Widget(super.params) : super.implementation();
  @override
  Widget build(BuildContext context) => const SizedBox();
}

class _Platform extends WebViewPlatform {
  final cookieWrites = <WebViewCookie>[];
  late _Delegate delegate;
  late _Controller controller;
  @override
  PlatformWebViewController createPlatformWebViewController(
    PlatformWebViewControllerCreationParams params,
  ) => controller = _Controller(params);
  @override
  PlatformNavigationDelegate createPlatformNavigationDelegate(
    PlatformNavigationDelegateCreationParams params,
  ) => delegate = _Delegate(params);
  @override
  PlatformWebViewCookieManager createPlatformCookieManager(
    PlatformWebViewCookieManagerCreationParams params,
  ) => _Cookies(params, cookieWrites);
  @override
  PlatformWebViewWidget createPlatformWebViewWidget(
    PlatformWebViewWidgetCreationParams params,
  ) => _Widget(params);
}

class _DownloadClient extends DioForNative {
  var started = Completer<void>();
  dio.CancelToken? cancellation;
  var result = Completer<dio.Response>();
  void reset() {
    started = Completer<void>();
    result = Completer<dio.Response>();
    cancellation = null;
  }

  @override
  Future<dio.Response> download(
    String urlPath,
    dynamic savePath, {
    dio.ProgressCallback? onReceiveProgress,
    Map<String, dynamic>? queryParameters,
    dio.CancelToken? cancelToken,
    bool deleteOnError = true,
    dio.FileAccessMode fileAccessMode = dio.FileAccessMode.write,
    String lengthHeader = dio.Headers.contentLengthHeader,
    Object? data,
    dio.Options? options,
  }) {
    cancellation = cancelToken;
    started.complete();
    return result.future;
  }
}

class _Paths extends PathProviderPlatform {
  _Paths(this.path);
  final String path;
  @override
  Future<String?> getTemporaryPath() async => path;
}

void main() {
  testWidgets(
    'session invalidation cancels export and ignores pending browser history',
    (tester) async {
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final previous = WebViewPlatform.instance;
      final platform = _Platform();
      WebViewPlatform.instance = platform;
      addTearDown(() {
        if (previous != null) WebViewPlatform.instance = previous;
      });
      final directory = Directory.systemTemp.createTempSync(
        'yourtj-admin-test-',
      );
      addTearDown(() => directory.deleteSync(recursive: true));
      final previousPaths = PathProviderPlatform.instance;
      PathProviderPlatform.instance = _Paths(directory.path);
      addTearDown(() => PathProviderPlatform.instance = previousPaths);
      final storage = MemoryTokenStorage();
      await storage.write('test-session');
      final client = _DownloadClient();
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          apiClientProvider.overrideWithValue(
            GfApiClient(
              dio: client,
              tokenStorage: storage,
              baseUrl: 'https://local.example',
            ),
          ),
        ],
      );
      addTearDown(container.dispose);
      SharedPreferences.setMockInitialValues({});
      container.read(appLocaleProvider.notifier).setLocale(const Locale("de"));
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const AdminPage(),
          ),
        ),
      );
      await tester.pump(const Duration(milliseconds: 200));
      expect(platform.delegate.request, isNotNull);
      expect(platform.cookieWrites.last.name, "lang");
      expect(platform.cookieWrites.last.value, "de");
      expect(platform.cookieWrites.last.domain, "local.example");
      expect(platform.controller.lastRequest?.headers["Accept-Language"], "de");
      platform.controller.historyResult = Completer<bool>();
      await tester.tap(find.byTooltip('Back'));
      await tester.runAsync(() async {
        await platform.delegate.request!(
          const NavigationRequest(
            url: 'https://local.example/api/admin/data/export/download/1',
            isMainFrame: true,
          ),
        );
        for (var i = 0; i < 150 && !client.started.isCompleted; i++) {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        }
        expect(
          client.started.isCompleted,
          isTrue,
          reason: 'Export reaches the download transport',
        );
        container.read(offlineCacheEpochProvider.notifier).invalidate();
        final cancelled = client.cancellation!.isCancelled;
        platform.controller.historyResult!.complete(true);
        client.result.completeError(StateError('Test download finished'));
        await Future<void>.delayed(const Duration(milliseconds: 100));
        expect(
          cancelled,
          isTrue,
          reason:
              'A stale account download must be cancelled at the session boundary',
        );
      });
      await tester.pump();
      expect(tester.takeException(), isNull);
      expect(platform.controller.backCalls, 0);
      await storage.write('new-account-session');
      client.reset();
      final previousController = platform.controller;
      await tester.runAsync(() async {
        await tester.tap(find.text('Retry'));
        await Future<void>.delayed(const Duration(milliseconds: 100));
      });
      await tester.pump(const Duration(milliseconds: 200));
      expect(
        platform.controller,
        isNot(same(previousController)),
        reason: 'Retry creates a fresh browser',
      );
      expect(
        platform.controller.loads,
        1,
        reason: 'The new session finishes the handoff',
      );
      var resumed = false;
      await tester.runAsync(() async {
        await platform.delegate.request!(
          const NavigationRequest(
            url: 'https://local.example/api/admin/data/export/download/2',
            isMainFrame: true,
          ),
        );
        for (var i = 0; i < 150 && !client.started.isCompleted; i++) {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        }
        resumed = client.started.isCompleted;
        if (resumed) {
          expect(client.cancellation!.isCancelled, isFalse);
          client.result.completeError(StateError('Finish retried export'));
          await Future<void>.delayed(const Duration(milliseconds: 100));
        }
      });
      expect(
        resumed,
        isTrue,
        reason: 'Retry must create an export scope for the new session',
      );
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      debugDefaultTargetPlatformOverride = null;
    },
  );
}

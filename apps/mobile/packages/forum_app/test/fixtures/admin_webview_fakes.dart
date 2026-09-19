import 'dart:async';
import 'package:flutter/material.dart';
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';

class TestWebController extends PlatformWebViewController {
  TestWebController(super.params) : super.implementation();
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

class TestWebDelegate extends PlatformNavigationDelegate {
  TestWebDelegate(super.params) : super.implementation();
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

class TestWebCookies extends PlatformWebViewCookieManager {
  TestWebCookies(super.params, this.writes) : super.implementation();
  final List<WebViewCookie> writes;
  @override
  Future<bool> clearCookies() async => true;
  @override
  Future<void> setCookie(WebViewCookie cookie) async {
    writes.add(cookie);
  }
}

class TestWebWidget extends PlatformWebViewWidget {
  TestWebWidget(super.params) : super.implementation();
  @override
  Widget build(BuildContext context) => const SizedBox();
}

class TestWebPlatform extends WebViewPlatform {
  final cookieWrites = <WebViewCookie>[];
  late TestWebDelegate delegate;
  late TestWebController controller;
  @override
  PlatformWebViewController createPlatformWebViewController(
    PlatformWebViewControllerCreationParams params,
  ) => controller = TestWebController(params);
  @override
  PlatformNavigationDelegate createPlatformNavigationDelegate(
    PlatformNavigationDelegateCreationParams params,
  ) => delegate = TestWebDelegate(params);
  @override
  PlatformWebViewCookieManager createPlatformCookieManager(
    PlatformWebViewCookieManagerCreationParams params,
  ) => TestWebCookies(params, cookieWrites);
  @override
  PlatformWebViewWidget createPlatformWebViewWidget(
    PlatformWebViewWidgetCreationParams params,
  ) => TestWebWidget(params);
}

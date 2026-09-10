import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:core/core.dart';
import 'package:file_selector/file_selector.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:webview_flutter/webview_flutter.dart';
import 'package:webview_flutter_android/webview_flutter_android.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../app_locale.dart';
import 'admin_navigation.dart';

/// The complete first-party console shares Web's authorization and forms. The
/// native session is sent once in a header; no credentials enter URL or script.
class AdminPage extends ConsumerStatefulWidget {
  const AdminPage({super.key, this.target = MobileWebTarget.admin});
  final MobileWebTarget target;
  @override
  ConsumerState<AdminPage> createState() => _AdminPageState();
}

class _AdminPageState extends ConsumerState<AdminPage> {
  // Serialize teardown across route instances so a previous view cannot clear
  // the next one's cookie after it has established a new session.
  static Future<void> _cleanup = Future.value();
  WebViewController? _controller;
  late final AdminNavigation _navigation;
  var _cancel = CancelToken();
  int _progress = 0;
  bool _failed = false;
  bool _downloading = false;
  bool _allowPop = false;
  bool _starting = false;

  @override
  void initState() {
    super.initState();
    _navigation = AdminNavigation(
      Uri.parse(ref.read(apiClientProvider).baseUrl),
    );
    unawaited(_start());
  }

  Future<void> _start() async {
    if (_starting) return;
    _starting = true;
    try {
      if (kIsWeb ||
          ![
            TargetPlatform.android,
            TargetPlatform.iOS,
            TargetPlatform.macOS,
          ].contains(defaultTargetPlatform) ||
          !_navigation.isSecureOrigin) {
        throw StateError('Unsupported embedded browser origin or platform');
      }
      // A failed previous cleanup is retried before another credential is used.
      await _cleanup.catchError((Object _) {});
      await WebViewCookieManager().clearCookies();
      if (!mounted) return;
      final epoch = ref.read(offlineCacheEpochProvider);
      final token = await ref.read(tokenStorageProvider).read();
      if (!mounted) return;
      if (token == null || token.isEmpty) {
        context.pushReplacement('/login');
        return;
      }
      if (epoch != ref.read(offlineCacheEpochProvider)) return;
      if (_cancel.isCancelled) _cancel = CancelToken();
      final controller = _controller ?? WebViewController();
      _controller = controller;
      await controller.clearCache();
      await controller.clearLocalStorage();
      await controller.setJavaScriptMode(JavaScriptMode.unrestricted);
      await controller.setOnJavaScriptConfirmDialog(
        (request) => _confirm(request.message),
      );
      await controller.setOnJavaScriptAlertDialog((request) async {
        await _confirm(request.message, alert: true);
      });
      await controller.setNavigationDelegate(
        NavigationDelegate(
          onProgress: (value) {
            if (mounted) setState(() => _progress = value);
          },
          onNavigationRequest: _navigate,
          onWebResourceError: (error) {
            if (error.isForMainFrame == true) _fail();
          },
          onHttpError: (error) {
            // Subresource failures must not blank a working console.
            final uri = error.request?.uri;
            if (uri?.path == '/api/auth/mobile-web-session' ||
                (uri != null &&
                    (uri.path.startsWith('/admin') ||
                        uri.path.startsWith('/moderation')) &&
                    error.response?.statusCode != null &&
                    error.response!.statusCode >= 400)) {
              _fail();
            }
          },
        ),
      );
      if (controller.platform is AndroidWebViewController) {
        await (controller.platform as AndroidWebViewController)
            .setOnShowFileSelector((params) async {
              if (!mounted) return [];
              if (params.mode == FileSelectorMode.openMultiple) {
                return (await openFiles())
                    .map((f) => Uri.file(f.path).toString())
                    .toList();
              }
              final file = await openFile();
              return file == null ? [] : [Uri.file(file.path).toString()];
            });
      }
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() {
        _failed = false;
        _progress = 0;
      });
      final language = resolveAppLocale(
        ref.read(appLocaleProvider),
      ).languageCode;
      await WebViewCookieManager().setCookie(
        WebViewCookie(
          name: 'lang',
          value: language,
          domain: _navigation.origin.host,
          path: '/',
        ),
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      await controller.loadRequest(
        _navigation.origin
            .resolve('/api/auth/mobile-web-session')
            .replace(queryParameters: {'target': widget.target.name}),
        headers: {
          'Authorization': 'Bearer $token',
          'Accept-Language': language,
        },
      );
    } catch (_) {
      _fail();
    } finally {
      _starting = false;
    }
  }

  Future<NavigationDecision> _navigate(NavigationRequest request) async {
    final uri = Uri.tryParse(request.url);
    if (uri == null || !mounted) return NavigationDecision.prevent;
    if (_navigation.isExport(uri)) {
      if (request.isMainFrame) unawaited(_download(uri));
      return NavigationDecision.prevent;
    }
    if (_navigation.isSameOrigin(uri)) {
      if (uri.path == '/login') {
        _fail();
        return NavigationDecision.prevent;
      }
      return NavigationDecision.navigate;
    }
    if (request.isMainFrame &&
        ['https', 'http', 'mailto'].contains(uri.scheme)) {
      // External pages never inherit cookies or the initial Bearer header.
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    }
    return NavigationDecision.prevent;
  }

  Future<void> _download(Uri uri) async {
    if (_downloading || _cancel.isCancelled || !_navigation.isExport(uri)) {
      return;
    }
    final epoch = ref.read(offlineCacheEpochProvider);
    final cancellation = _cancel;
    setState(() => _downloading = true);
    Directory? directory;
    try {
      final client = ref.read(apiClientProvider);
      directory = await (await getTemporaryDirectory()).createTemp(
        'yourtj-export-',
      );
      final file = File('${directory.path}/download.tmp');
      final response = await client.dio.download(
        uri.toString(),
        file.path,
        cancelToken: cancellation,
        options: Options(
          followRedirects: false,
          validateStatus: (s) => s == 200,
        ),
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      // Fail closed on a login/error envelope returned with HTTP 200.
      if (response.headers
              .value('content-disposition')
              ?.contains('attachment') !=
          true) {
        throw StateError('Export did not return an attachment');
      }
      final disposition = response.headers.value('content-disposition') ?? '';
      final filename = RegExp(
        r'filename="([a-zA-Z0-9_.-]+)"',
      ).firstMatch(disposition)?.group(1);
      if (filename == null || filename.startsWith('.')) {
        throw StateError('Invalid export filename');
      }
      final exported = await file.rename('${directory.path}/$filename');
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      final box = context.findRenderObject() as RenderBox?;
      await SharePlus.instance.share(
        ShareParams(
          files: [XFile(exported.path)],
          sharePositionOrigin: box == null
              ? null
              : box.localToGlobal(Offset.zero) & box.size,
        ),
      );
    } catch (_) {
      if (mounted && !cancellation.isCancelled) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(AppLocalizations.of(context).adminDownloadFailed),
          ),
        );
      }
    } finally {
      if (directory != null && await directory.exists()) {
        await directory.delete(recursive: true);
      }
      if (mounted) setState(() => _downloading = false);
    }
  }

  String _title(AppLocalizations l10n) => switch (widget.target) {
    MobileWebTarget.admin => l10n.profileAdmin,
    MobileWebTarget.moderation => l10n.profileModeration,
    MobileWebTarget.courseManagement => l10n.coursesManagement,
    MobileWebTarget.courseReviews => l10n.coursesReviewModeration,
  };

  Future<bool> _confirm(String message, {bool alert = false}) async {
    if (!mounted) return false;
    final l10n = AppLocalizations.of(context);
    return await showDialog<bool>(
          context: context,
          builder: (context) => AlertDialog(
            title: Text(_title(l10n)),
            content: SingleChildScrollView(child: Text(message)),
            actions: [
              if (!alert)
                TextButton(
                  onPressed: () => Navigator.pop(context, false),
                  child: Text(l10n.commonCancel),
                ),
              FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: Text(l10n.commonConfirm),
              ),
            ],
          ),
        ) ??
        false;
  }

  void _fail() {
    if (mounted) setState(() => _failed = true);
  }

  Future<void> _back() async {
    final controller = _controller;
    if (controller != null && !_failed) {
      try {
        final canGoBack = await controller.canGoBack();
        // Session teardown can replace the controller while history is queried.
        if (!mounted || _controller != controller || _failed) return;
        if (canGoBack) {
          await controller.goBack();
          return;
        }
      } catch (_) {
        if (!mounted || _controller != controller) return;
      }
    }
    if (mounted) {
      setState(() => _allowPop = true);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.pop();
      });
    }
  }

  void _clearBrowser() {
    final controller = _controller;
    _controller = null;
    if (controller == null) return;
    final preceding = _cleanup;
    _cleanup = () async {
      await preceding.catchError((Object _) {});
      Object? failure;
      for (final clear in <Future<void> Function()>[
        () => controller.loadHtmlString(''),
        controller.clearLocalStorage,
        controller.clearCache,
        () async {
          await WebViewCookieManager().clearCookies();
        },
      ]) {
        try {
          await clear();
        } catch (error) {
          failure = error;
        }
      }
      if (failure != null) {
        throw StateError('Could not clear embedded browser state');
      }
    }();
    unawaited(_cleanup.catchError((Object _) {}));
  }

  @override
  void dispose() {
    _cancel.cancel();
    _clearBrowser();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (_, _) {
      _cancel.cancel('Session changed');
      _clearBrowser();
      _fail();
    });
    return PopScope(
      canPop: _allowPop,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) unawaited(_back());
      },
      child: Scaffold(
        appBar: GfAppBar(
          title: Text(_title(l10n)),
          leading: IconButton(
            icon: const Icon(Icons.arrow_back),
            tooltip: l10n.commonBack,
            onPressed: _back,
          ),
          actions: [
            IconButton(
              icon: const Icon(Icons.close),
              tooltip: l10n.commonClose,
              onPressed: () {
                setState(() => _allowPop = true);
                WidgetsBinding.instance.addPostFrameCallback((_) {
                  if (mounted) context.pop();
                });
              },
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Column(
            children: [
              if (_progress < 100 || _downloading)
                LinearProgressIndicator(
                  value: _downloading ? null : _progress / 100,
                ),
              Expanded(
                child: _failed
                    ? Center(
                        child: Padding(
                          padding: const EdgeInsets.all(24),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const Icon(
                                Icons.admin_panel_settings_outlined,
                                size: 48,
                              ),
                              const SizedBox(height: 16),
                              Text(
                                l10n.adminUnavailable,
                                textAlign: TextAlign.center,
                              ),
                              const SizedBox(height: 16),
                              FilledButton(
                                onPressed: _start,
                                child: Text(l10n.commonRetry),
                              ),
                            ],
                          ),
                        ),
                      )
                    : _controller == null
                    ? const Center(child: CircularProgressIndicator())
                    : WebViewWidget(controller: _controller!),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

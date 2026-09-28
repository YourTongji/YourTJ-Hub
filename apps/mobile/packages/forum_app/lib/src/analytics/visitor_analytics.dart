import 'dart:async';
import 'dart:collection';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

/// Only official, native release builds can contribute to production analytics.
bool visitorAnalyticsAvailable({
  required String apiBaseUrl,
  required bool release,
  required bool web,
  required TargetPlatform platform,
}) =>
    release &&
    !web &&
    (platform == TargetPlatform.android || platform == TargetPlatform.iOS) &&
    (apiBaseUrl == 'https://f.yourtj.de' ||
        apiBaseUrl == 'https://f.yourtj.de/');

/// No original URL, identifiers, search terms or fragments cross this boundary.
String? publicScreen(Uri uri) {
  final path = uri.path;
  if (path == '/') return '/app/home';
  if (path == '/search') return '/app/search';
  if (RegExp(r'^/p/[0-9]+$').hasMatch(path)) return '/app/topic';
  if (RegExp(r'^/c/[^/]+/[0-9]+$').hasMatch(path)) return '/app/category';
  if (path == '/courses') return '/app/courses';
  if (RegExp(r'^/courses/[^/]+$').hasMatch(path)) return '/app/course';
  if (path == '/wiki' || path.startsWith('/wiki/')) return '/app/wiki';
  return null;
}

/// Best-effort page views for Umami's joint device/OS/browser report. This Dio
/// is separate from authenticated forum transports. No persistent identifiers,
/// disk queue, retries, credentials, referrers or free-form event data are used.
class VisitorAnalytics {
  VisitorAnalytics({
    required this.os,
    Dio? transport,
    DateTime Function()? clock,
  }) : _dio = transport ?? Dio(),
       _clock = clock ?? DateTime.now {
    _dio.options = BaseOptions(
      connectTimeout: const Duration(seconds: 3),
      sendTimeout: const Duration(seconds: 3),
      receiveTimeout: const Duration(seconds: 3),
      followRedirects: false,
      maxRedirects: 0,
    );
  }

  final String os;
  final Dio _dio;
  final DateTime Function() _clock;
  final _pending = Queue<({String screen, String device})>();
  bool _enabled = false, _active = false, _sending = false, _disposed = false;
  int _generation = 0;
  String? _lastPath, _cache;
  DateTime? _lastVisit;
  CancelToken? _cancel;

  void setEnabled(bool value) {
    if (_enabled == value || _disposed) return;
    _enabled = value;
    if (!value) {
      _stop();
      _cache = null;
      _lastPath = null;
      _lastVisit = null;
    }
  }

  void setActive(bool value) {
    if (_disposed) return;
    _active = value;
    if (!value) _stop();
  }

  void visit(Uri uri, {required bool tablet}) {
    if (!_enabled || !_active || _disposed) return;
    final screen = publicScreen(uri);
    if (screen == null) {
      _lastPath = null;
      return;
    }
    final now = _clock();
    if (_lastPath == uri.path &&
        _lastVisit != null &&
        now.difference(_lastVisit!) < const Duration(minutes: 30)) {
      return;
    }
    _lastPath = uri.path; // Memory-only navigation deduplication; never sent.
    _lastVisit = now;
    if (_pending.length == 3) _pending.removeFirst();
    _pending.add((screen: screen, device: tablet ? 'tablet' : 'mobile'));
    unawaited(_drain());
  }

  Future<void> _drain() async {
    if (_sending) return;
    _sending = true;
    try {
      while (_enabled && _active && !_disposed && _pending.isNotEmpty) {
        final event = _pending.removeFirst();
        final generation = _generation;
        final cancel = _cancel = CancelToken();
        try {
          final response = await _dio.post<Object?>(
            'https://umi.yourtj.de/api/send',
            cancelToken: cancel,
            options: Options(
              headers: {
                // Include device family for visitor separation and avoid the bot
                // heuristic for a bare Product/1.0 (Word) native user agent.
                'User-Agent': 'YourTJApp/1.0 ($os; ${event.device})',
                if (_cache != null) 'x-umami-cache': _cache,
              },
            ),
            data: {
              'type': 'event',
              'payload': {
                'website': '36750dcd-8c48-46ab-9dfb-2f09cdcef501',
                'hostname': 'f.yourtj.de',
                'url': event.screen,
                'browser': 'yourtj-app',
                'os': os,
                'device': event.device,
                'tag': 'yourtj-app',
              },
            },
          );
          final data = response.data;
          final cache = data is Map ? data['cache'] : null;
          if (generation == _generation &&
              cache is String &&
              cache.length <= 16384 &&
              RegExp(r'^[!-~]+$').hasMatch(cache)) {
            _cache = cache;
          }
        } catch (_) {
          // Analytics must not affect navigation or generate sensitive logs.
        }
      }
    } finally {
      _sending = false;
      _cancel = null;
    }
  }

  void _stop() {
    _generation++;
    _pending.clear();
    _cancel?.cancel();
  }

  void dispose() {
    _disposed = true;
    _stop();
    _cache = null;
    _lastPath = null;
    _dio.close(force: true);
  }
}

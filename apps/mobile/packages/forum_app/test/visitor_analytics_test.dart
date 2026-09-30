import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/analytics/analytics_host.dart';
import 'package:forum_app/src/analytics/visitor_analytics.dart';

class _Adapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  Completer<void>? hold;
  bool fail = false;
  bool cancelled = false;
  Future<void> waitForRequests(int count) async {
    final deadline = DateTime.now().add(const Duration(seconds: 5));
    while (requests.length < count && DateTime.now().isBefore(deadline)) {
      await Future<void>.delayed(Duration.zero);
    }
    expect(requests, hasLength(count));
  }

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? stream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    cancelFuture?.then((_) => cancelled = true);
    if (hold != null) await hold!.future;
    if (fail) throw StateError('offline');
    return ResponseBody.fromString(
      '{"cache":"memory-only-token"}',
      200,
      headers: {
        'content-type': ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

Future<void> flush() => Future<void>.delayed(const Duration(milliseconds: 10));

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => SharedPreferences.setMockInitialValues({}));

  test('only official Android/iOS release builds can collect', () {
    bool allowed({
      String base = 'https://f.yourtj.de',
      bool release = true,
      bool web = false,
      TargetPlatform platform = TargetPlatform.android,
    }) => visitorAnalyticsAvailable(
      apiBaseUrl: base,
      release: release,
      web: web,
      platform: platform,
    );
    expect(allowed(), isTrue);
    expect(allowed(platform: TargetPlatform.iOS), isTrue);
    for (final base in [
      'http://f.yourtj.de',
      'https://dev.yourtj.de',
      '',
      'https://f.yourtj.de.attacker.test',
      'https://f.yourtj.de/private',
    ]) {
      expect(allowed(base: base), isFalse);
    }
    expect(allowed(release: false), isFalse);
    expect(allowed(web: true), isFalse);
    expect(allowed(platform: TargetPlatform.macOS), isFalse);
  });

  test('maps public route categories and excludes private screens', () {
    expect(
      publicScreen(Uri.parse('/p/987?secret=private#message')),
      '/app/topic',
    );
    expect(
      publicScreen(Uri.parse('/wiki/private-title?q=secret')),
      '/app/wiki',
    );
    expect(publicScreen(Uri.parse('/c/private-slug/33')), '/app/category');
    for (final path in [
      '/chat?userId=7',
      '/campus',
      '/schedule',
      '/login',
      '/settings',
      '/settings/account',
      '/admin',
      '/moderation',
      '/notifications',
      '/messages',
      '/profile',
      '/u/42',
      '/publish',
      '/drafts',
      '/my-content',
    ]) {
      expect(publicScreen(Uri.parse(path)), isNull, reason: path);
    }
  });

  test(
    'sends only allowlisted pageview fields; no credentials; deduplicates rebuilds',
    () async {
      final adapter = _Adapter();
      final analytics = VisitorAnalytics(
        os: 'Android OS',
        transport: Dio()..httpClientAdapter = adapter,
      );
      addTearDown(analytics.dispose);
      analytics.visit(Uri.parse('/p/123?q=private#secret'), tablet: false);
      await flush();
      expect(adapter.requests, isEmpty);
      analytics.setActive(true);
      analytics.visit(Uri.parse('/p/123?q=private#secret'), tablet: false);
      await flush();
      final request = adapter.requests.single;
      expect(request.uri.toString(), 'https://umi.yourtj.de/api/send');
      expect(request.data, {
        'type': 'event',
        'payload': {
          'website': '36750dcd-8c48-46ab-9dfb-2f09cdcef501',
          'hostname': 'f.yourtj.de',
          'url': '/app/topic',
          'browser': 'yourtj-app',
          'os': 'Android OS',
          'device': 'mobile',
          'tag': 'yourtj-app',
        },
      });
      expect(jsonEncode(request.data), isNot(contains('private')));
      expect(
        request.headers.keys.map((key) => key.toLowerCase()),
        isNot(contains('authorization')),
      );
      expect(
        request.headers.keys.map((key) => key.toLowerCase()),
        isNot(contains('cookie')),
      );
      expect(request.followRedirects, isFalse);
      expect(
        request.headers['User-Agent'],
        'YourTJApp/1.0 (Android OS; mobile)',
      );
      analytics.visit(Uri.parse('/p/123?q=changed'), tablet: false);
      await flush();
      expect(adapter.requests, hasLength(1));
      analytics.visit(Uri.parse('/p/456'), tablet: true);
      await flush();
      expect(adapter.requests.last.data['payload']['device'], 'tablet');
      expect(
        adapter.requests.last.headers['x-umami-cache'],
        'memory-only-token',
      );
    },
  );

  test(
    'pauses in background, resumes without duplicates, refreshes after 30 minutes',
    () async {
      final adapter = _Adapter();
      var time = DateTime.utc(2026);
      final analytics = VisitorAnalytics(
        os: 'iOS',
        transport: Dio()..httpClientAdapter = adapter,
        clock: () => time,
      );
      addTearDown(analytics.dispose);
      analytics.setActive(true);
      analytics.visit(Uri.parse('/'), tablet: false);
      await flush();
      analytics.setActive(false);
      analytics.visit(Uri.parse('/search?q=secret'), tablet: false);
      analytics.setActive(true);
      analytics.visit(Uri.parse('/'), tablet: false);
      await flush();
      expect(adapter.requests, hasLength(1));
      time = time.add(const Duration(minutes: 31));
      analytics.visit(Uri.parse('/'), tablet: false);
      await flush();
      expect(adapter.requests, hasLength(2));
      analytics.visit(Uri.parse('/settings'), tablet: false);
      analytics.visit(Uri.parse('/'), tablet: false);
      await flush();
      expect(adapter.requests, hasLength(3));
    },
  );

  test(
    'bounded memory queue, backgrounding cancels in-flight and discards unsent views',
    () async {
      final adapter = _Adapter()..hold = Completer<void>();
      final analytics = VisitorAnalytics(
        os: 'iOS',
        transport: Dio()..httpClientAdapter = adapter,
      );
      addTearDown(analytics.dispose);
      analytics.setActive(true);
      for (var i = 0; i < 12; i++) {
        analytics.visit(Uri.parse('/p/$i'), tablet: false);
      }
      await flush();
      expect(adapter.requests, hasLength(1));
      adapter.hold!.complete();
      // Wait for the observable drain, not a 10 ms wall-clock assumption on CI.
      await adapter.waitForRequests(4);
      adapter.hold = Completer<void>();
      analytics.visit(Uri.parse('/search'), tablet: false);
      await flush();
      analytics.visit(Uri.parse('/'), tablet: false);
      analytics.setActive(false);
      await flush();
      expect(adapter.cancelled, isTrue);
      adapter.hold!.complete();
      await flush();
      expect(adapter.requests, hasLength(5));
      analytics.setActive(true);
      analytics.visit(Uri.parse('/'), tablet: false);
      await flush();
      // Canceled views are not replayed when the foreground host resumes.
      analytics.visit(Uri.parse('/courses'), tablet: false);
      await flush();
      expect(adapter.requests, hasLength(6));
    },
  );

  test('network failures are swallowed without retrying', () async {
    final adapter = _Adapter()..fail = true;
    final analytics = VisitorAnalytics(
      os: 'iOS',
      transport: Dio()..httpClientAdapter = adapter,
    );
    addTearDown(analytics.dispose);
    analytics.setActive(true);
    analytics.visit(Uri.parse('/'), tablet: false);
    await flush();
    expect(adapter.requests, hasLength(1));
    adapter.fail = false;
    analytics.visit(Uri.parse('/courses'), tablet: false);
    await flush();
    expect(adapter.requests, hasLength(2));
  });

  for (final previousChoice in <bool?>[null, false, true]) {
    testWidgets(
      'automatically counts public destinations with legacy preference $previousChoice',
      (tester) async {
        SharedPreferences.setMockInitialValues({
          'visitor_analytics_opt_in': ?previousChoice,
        });
        tester.binding.handleAppLifecycleStateChanged(
          AppLifecycleState.resumed,
        );
        Future<void> settleAnalytics() async {
          await tester.pumpAndSettle();
          // Dio schedules transport/transform work on subsequent event-loop turns.
          for (var i = 0; i < 10; i++) {
            await tester.pump(const Duration(milliseconds: 20));
          }
        }

        final adapter = _Adapter();
        final analytics = VisitorAnalytics(
          os: 'iOS',
          transport: Dio()..httpClientAdapter = adapter,
        );
        final router = GoRouter(
          routes: [
            for (final path in ['/', '/search', '/settings'])
              GoRoute(path: path, builder: (_, _) => Text(path)),
          ],
        );
        final container = ProviderContainer(
          overrides: [visitorAnalyticsProvider.overrideWithValue(analytics)],
        );
        addTearDown(() {
          container.dispose();
          analytics.dispose();
          router.dispose();
        });
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              builder: (_, child) =>
                  AnalyticsHost(router: router, child: child!),
            ),
          ),
        );
        await settleAnalytics();
        expect(adapter.requests, hasLength(1));
        router.go('/settings');
        await settleAnalytics();
        expect(adapter.requests, hasLength(1));
        router.go('/search?q=secret');
        await settleAnalytics();
        expect(adapter.requests.last.data['payload']['url'], '/app/search');
        expect(adapter.requests, hasLength(2));
        tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
        router.go('/');
        await settleAnalytics();
        expect(adapter.requests, hasLength(2));
        tester.binding.handleAppLifecycleStateChanged(
          AppLifecycleState.resumed,
        );
        await settleAnalytics();
        expect(adapter.requests, hasLength(3));
        await tester.pumpWidget(const SizedBox.shrink());
        analytics.visit(Uri.parse('/search'), tablet: false);
        await settleAnalytics();
        expect(adapter.requests, hasLength(3));
      },
    );
  }
}

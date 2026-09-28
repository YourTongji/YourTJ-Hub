import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/analytics/analytics_consent.dart';
import 'package:forum_app/src/analytics/analytics_host.dart';
import 'package:forum_app/src/analytics/analytics_setting.dart';
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
    'defaults off; later revocation wins over pending preference restoration',
    () async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      expect(container.read(analyticsConsentProvider), isFalse);
      await flush();
      expect(container.read(analyticsConsentProvider), isFalse);
      expect(
        await container
            .read(analyticsConsentProvider.notifier)
            .setEnabled(true),
        isTrue,
      );
      expect(container.read(analyticsConsentProvider), isTrue);
      final restored = ProviderContainer();
      addTearDown(restored.dispose);
      restored.read(analyticsConsentProvider);
      await restored.read(analyticsConsentProvider.notifier).setEnabled(false);
      await flush();
      expect(restored.read(analyticsConsentProvider), isFalse);
      expect(
        (await SharedPreferences.getInstance()).getBool(
          AnalyticsConsent.preferenceKey,
        ),
        isFalse,
      );
    },
  );

  test(
    'sends only allowlisted pageview fields; no credentials; deduplicates rebuilds',
    () async {
      final adapter = _Adapter();
      final analytics = VisitorAnalytics(
        os: 'Android OS',
        transport: Dio()..httpClientAdapter = adapter,
      );
      addTearDown(analytics.dispose);
      analytics.setActive(true);
      analytics.visit(Uri.parse('/p/123?q=private#secret'), tablet: false);
      await flush();
      expect(adapter.requests, isEmpty);
      analytics.setEnabled(true);
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
      analytics.setEnabled(false);
      analytics.setEnabled(true);
      analytics.visit(Uri.parse('/'), tablet: false);
      await flush();
      expect(adapter.requests.last.headers['x-umami-cache'], isNull);
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
      analytics.setEnabled(true);
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
    'bounded memory queue, revocation cancels in-flight and discards unsent views',
    () async {
      final adapter = _Adapter()..hold = Completer<void>();
      final analytics = VisitorAnalytics(
        os: 'iOS',
        transport: Dio()..httpClientAdapter = adapter,
      );
      addTearDown(analytics.dispose);
      analytics.setEnabled(true);
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
      analytics.setEnabled(false);
      await flush();
      expect(adapter.cancelled, isTrue);
      adapter.hold!.complete();
      await flush();
      expect(adapter.requests, hasLength(5));
      analytics.setEnabled(true);
      analytics.visit(Uri.parse('/'), tablet: false);
      await flush();
      expect(adapter.requests.last.headers['x-umami-cache'], isNull);
    },
  );

  test('network failures are swallowed without retrying', () async {
    final adapter = _Adapter()..fail = true;
    final analytics = VisitorAnalytics(
      os: 'iOS',
      transport: Dio()..httpClientAdapter = adapter,
    );
    addTearDown(analytics.dispose);
    analytics.setEnabled(true);
    analytics.setActive(true);
    analytics.visit(Uri.parse('/'), tablet: false);
    await flush();
    expect(adapter.requests, hasLength(1));
    adapter.fail = false;
    analytics.visit(Uri.parse('/courses'), tablet: false);
    await flush();
    expect(adapter.requests, hasLength(2));
  });

  testWidgets(
    'router integration counts visible public destinations after consent',
    (tester) async {
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
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
            builder: (_, child) => AnalyticsHost(router: router, child: child!),
          ),
        ),
      );
      await settleAnalytics();
      expect(adapter.requests, isEmpty);
      await tester.runAsync(
        () =>
            container.read(analyticsConsentProvider.notifier).setEnabled(true),
      );
      await settleAnalytics();
      expect(container.read(analyticsConsentProvider), isTrue);
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
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await settleAnalytics();
      expect(adapter.requests, hasLength(3));
      await tester.runAsync(
        () =>
            container.read(analyticsConsentProvider.notifier).setEnabled(false),
      );
      router.go('/search');
      await settleAnalytics();
      expect(adapter.requests, hasLength(3));
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets('device preference is readable and usable without an account', (
    tester,
  ) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: Locale('en'),
          home: Scaffold(
            body: SingleChildScrollView(child: AnalyticsSetting()),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsNothing);
    expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);
    expect(tester.takeException(), isNull);
  });
}

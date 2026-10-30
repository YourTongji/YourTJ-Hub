import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_behavior_test.dart'
    show
        CountingPageRepository,
        EmptySessionsUserRepository,
        LogoutAuthRepository,
        MemTokenStorage;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/account_return.dart';
import 'package:go_router/go_router.dart';

class _ReturningAuth extends LogoutAuthRepository {
  _ReturningAuth(super.client);
  bool verified = false;
  int reads = 0;
  @override
  Future<ActivationStatus> getActivationStatus({
    CancelToken? cancelToken,
  }) async {
    reads++;
    return ActivationStatus(
      currentEmail: 'owner@example.test',
      pendingEmail: '',
      targetEmail: verified ? '' : 'owner@example.test',
      targetKind: verified ? 'none' : 'current',
      state: verified ? 'verified' : 'ready',
      remainingToday: 3,
      dailyLimit: 3,
      retryAfterSeconds: 0,
    );
  }
}

void main() {
  testWidgets(
    'native repeated verification hint refreshes the open owner settings without stacking',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      tester.view.physicalSize = const Size(600, 1800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      const channel = MethodChannel('yourtj/account_return');
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        channel,
        (_) async => null,
      );
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          channel,
          null,
        ),
      );
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage('session'),
        baseUrl: 'http://fake.local',
      );
      final auth = _ReturningAuth(client);
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(MemTokenStorage('session')),
          authRepositoryProvider.overrideWithValue(auth),
          pageRepositoryProvider.overrideWithValue(
            CountingPageRepository(client),
          ),
          userRepositoryProvider.overrideWithValue(
            EmptySessionsUserRepository(client),
          ),
        ],
      );
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (_, _) =>
                const Scaffold(body: Text('unfinished operation')),
          ),
          GoRoute(
            path: '/settings/account',
            builder: (_, _) => const SettingsPage(initialSection: 'account'),
          ),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: AccountReturnHost(
            router: router,
            child: MaterialApp.router(
              routerConfig: router,
              theme: gfThemeData(Brightness.light),
              locale: const Locale('zh'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      Future<void> deliver() async {
        tester.binding.channelBuffers.push(
          'yourtj/account_return',
          const StandardMethodCodec().encodeSuccessEnvelope(
            'yourtj://account/email-verified',
          ),
          (_) {},
        );
        await tester.pumpAndSettle();
      }

      await deliver();
      expect(auth.reads, 1);
      final settingsState = tester.state(find.byType(SettingsPage));
      auth.verified = true;
      await deliver();
      expect(
        auth.reads,
        2,
        reason:
            'a fresh native hint must re-read the current session even while settings is open',
      );
      expect(tester.state(find.byType(SettingsPage)), same(settingsState));
      expect(
        find.text(AppLocalizationsZh().activationStatusVerified),
        findsOneWidget,
      );
      router.pop();
      await tester.pumpAndSettle();
      expect(find.text('unfinished operation'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
      router.dispose();
      container.dispose();
    },
  );

  testWidgets(
    'email return preserves an unfinished operation and ignores OIDC and forged payloads',
    (tester) async {
      final links = StreamController<Uri>();
      var settingsVisits = 0;
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (_, _) =>
                const Scaffold(body: TextField(key: Key('draft'))),
          ),
          GoRoute(
            path: '/settings/account',
            builder: (_, state) {
              settingsVisits++;
              return Scaffold(body: Text('Current account settings'));
            },
          ),
        ],
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            accountReturnLinksProvider.overrideWith((ref) => links.stream),
          ],
          child: AccountReturnHost(
            router: router,
            child: MaterialApp.router(routerConfig: router),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byKey(const Key('draft')),
        'unsent confirmed draft',
      );
      for (final raw in [
        'yourtj://callback?code=oauth-code',
        'yourtj://account/email-verified?token=forged',
        'https://evil.test/account/email-verified',
        'yourtj://account/other',
      ]) {
        links.add(Uri.parse(raw));
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('draft')), findsOneWidget);
      }
      links.add(Uri.parse('yourtj://account/email-verified'));
      await tester.pumpAndSettle();
      expect(find.text('Current account settings'), findsOneWidget);
      // Repeated delivery must not stack settings or replace the unfinished operation.
      links.add(Uri.parse('yourtj://account/email-verified'));
      await tester.pumpAndSettle();
      router.pop();
      await tester.pumpAndSettle();
      expect(find.text('unsent confirmed draft'), findsOneWidget);
      expect(settingsVisits, 1);
      await tester.pumpWidget(const SizedBox());
      router.dispose();
      await tester.runAsync(links.close);
    },
  );
  testWidgets('a buffered cold-start return waits for the navigation host', (
    tester,
  ) async {
    final container = ProviderContainer(
      overrides: [
        accountReturnLinksProvider.overrideWith(
          (ref) => Stream.value(Uri.parse('yourtj://account/email-verified')),
        ),
      ],
    );
    await container.read(accountReturnLinksProvider.future);
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => const Scaffold(body: Text('home')),
        ),
        GoRoute(
          path: '/settings/account',
          builder: (_, _) =>
              const Scaffold(body: Text('Current account settings')),
        ),
      ],
    );
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: AccountReturnHost(
          router: router,
          child: MaterialApp.router(routerConfig: router),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Current account settings'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    router.dispose();
    container.dispose();
  });
}

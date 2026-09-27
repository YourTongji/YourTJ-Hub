import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/auth_navigation.dart';
import 'package:go_router/go_router.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  for (final sessionCurrent in [true, false]) {
    testWidgets(
      'restoration waits for root redirect and checks session $sessionCurrent',
      (tester) async {
        final rootReady = Completer<String?>();
        var current = true;
        final router = GoRouter(
          initialLocation: '/login',
          redirect: (_, state) =>
              state.uri.path == '/' ? rootReady.future : null,
          routes: [
            GoRoute(
              path: '/',
              builder: (_, _) => const Scaffold(body: Text('Home')),
            ),
            GoRoute(
              path: '/login',
              builder: (_, _) => const Scaffold(body: Text('Login')),
            ),
            GoRoute(
              path: '/drafts',
              builder: (_, _) => const Scaffold(body: Text('Drafts')),
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(MaterialApp.router(routerConfig: router));
        await tester.pumpAndSettle();
        restoreAuthContext(router, '/drafts', isCurrent: () => current);
        await tester.pump();
        current = sessionCurrent;
        rootReady.complete(null);
        await tester.pumpAndSettle();
        expect(router.state.uri.path, sessionCurrent ? '/drafts' : '/');
        expect(router.canPop(), sessionCurrent);
      },
    );
  }

  test('return locations preserve only known native route shapes', () {
    for (final value in [
      '/',
      '/p/123?postNo=8',
      '/u/12',
      '/u/12/following',
      '/u/12/followers',
      '/settings/widgets',
      '/courses/7?reviewId=9',
      '/publish?type=question',
      '/publish?local=draft-123',
      '/wiki/guide/start#heading',
      '/settings/security',
      '/settings/profile?edit=1',
      '/chat?userId=2&username=Alice',
    ]) {
      expect(safeAuthReturnTo(value), value, reason: value);
    }
  });
  test(
    'external, recursive, encoded authority and malformed destinations are rejected',
    () {
      for (final value in [
        '',
        'https://example.com',
        '//example.com',
        '/\\example.com',
        '/login',
        '/login?returnTo=%2Fp%2F123',
        '/unknown',
        '/settings/profile?edit=0',
        '/settings/profile?edit=true',
        '/settings/profile?edit=1&edit=1',
        '/settings/account?edit=1',
        '/p/not-a-number',
        '/u/12/unknown',
        '/u/not-a-number/followers',
        '/p/-1',
        '/p/1?postNo=2&postNo=3',
        '/p/1?redirect=https://example.com',
        '/chat',
        '/chat?userId=0',
        '/wiki/%2F%2Fevil',
        '/wiki/%5cevil',
        '/wiki/%00',
        '/wiki/%FF',
        '/wiki/start#%FF',
        '/search?q=%0A',
        '/p/1#fragment',
      ]) {
        expect(safeAuthReturnTo(value), isNull, reason: value);
      }
    },
  );

  testWidgets(
    'nested route login captures its top location and guests do not build protected pages',
    (tester) async {
      final storage = MemoryTokenStorage();
      var publishBuilds = 0;
      late final GoRouter router;
      router = GoRouter(
        initialLocation: '/',
        redirect: (context, state) => authNavigationRedirect(
          requested: state.uri,
          previousLocation: router.routerDelegate.currentConfiguration.isEmpty
              ? null
              : router.state.uri.toString(),
          tokenStorage: storage,
        ),
        routes: [
          ShellRoute(
            builder: (_, _, child) => Scaffold(body: child),
            routes: [
              GoRoute(path: '/', builder: (_, _) => const Text('Home')),
              GoRoute(path: '/p/:id', builder: (_, _) => const Text('Topic')),
            ],
          ),
          GoRoute(
            path: '/login',
            builder: (_, _) => const Scaffold(body: Text('Login')),
          ),
          GoRoute(
            path: '/publish',
            builder: (_, _) {
              publishBuilds++;
              return const Scaffold(body: Text('Composer'));
            },
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(child: MaterialApp.router(routerConfig: router)),
      );
      await tester.pumpAndSettle();
      router.push('/p/123?postNo=8');
      await tester.pumpAndSettle();
      router.push('/login');
      await tester.pumpAndSettle();
      expect(router.state.uri.queryParameters['returnTo'], '/p/123?postNo=8');
      router.pop();
      await tester.pumpAndSettle();
      expect(router.state.uri.toString(), '/p/123?postNo=8');
      expect(publishBuilds, 0);
      router.go('/publish?type=article');
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/login');
      expect(
        router.state.uri.queryParameters['returnTo'],
        '/publish?type=article',
      );
      expect(publishBuilds, 0);
      await storage.write('new-session');
      router.go(
        safeAuthReturnTo(router.state.uri.queryParameters['returnTo'])!,
      );
      await tester.pumpAndSettle();
      expect(router.state.uri.toString(), '/publish?type=article');
      expect(publishBuilds, 1);
    },
  );

  test('malformed or duplicate outer login targets fall back safely', () async {
    for (final location in [
      '/login?returnTo=%FF',
      '/login?returnTo=%2Fp%2F1&returnTo=%2Fp%2F2',
    ]) {
      expect(
        await authNavigationRedirect(
          requested: Uri.parse(location),
          tokenStorage: MemoryTokenStorage(),
        ),
        authLoginLocation(),
      );
    }
  });

  test(
    'profile editor return target survives the unauthenticated redirect',
    () async {
      const destination = '/settings/profile?edit=1';
      final redirect = await authNavigationRedirect(
        requested: Uri.parse(destination),
        tokenStorage: MemoryTokenStorage(),
      );

      expect(Uri.parse(redirect!).queryParameters['returnTo'], destination);
    },
  );
}

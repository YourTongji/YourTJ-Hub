import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/account_return.dart';
import 'package:go_router/go_router.dart';

void main() {
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

import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/startup_experience.dart';
import 'package:forum_app/src/storage/media_host.dart';
import 'package:forum_app/src/storage/storage_providers.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'widget_test.dart'
    show MemoryTokenStorage, NoopOfflineCache, RouterPageRepository;

class _Pages extends RouterPageRepository {
  _Pages(super.client);
  int reads = 0;
  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) {
    reads++;
    return super.fetch(path, cancelToken: cancelToken);
  }
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  Future<({ProviderContainer container, _Pages pages})> pump(
    WidgetTester tester,
    Future<void> Function() bootstrap,
  ) async {
    final tokens = MemoryTokenStorage();
    final pages = _Pages(
      GfApiClient(
        dio: Dio(),
        tokenStorage: tokens,
        baseUrl: 'http://fake.local',
      ),
    );
    final container = ProviderContainer(
      overrides: [
        storageBootstrapProvider.overrideWith((ref) => bootstrap()),
        tokenStorageProvider.overrideWithValue(tokens),
        currentUserProvider.overrideWith((ref) async => null),
        offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
        offlineChatCacheProvider.overrideWithValue(NoopOfflineCache()),
        pageRepositoryProvider.overrideWithValue(pages),
      ],
    );
    addTearDown(container.dispose);
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.reset);
    appRouter.go('/');
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const GfApp(locale: Locale('en')),
      ),
    );
    await tester.pump();
    return (container: container, pages: pages);
  }

  Future<void> dispose(WidgetTester tester) async {
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 1));
  }

  testWidgets('bootstrap waits before mounting business hosts', (tester) async {
    final recovery = Completer<void>();
    final app = await pump(tester, () => recovery.future);
    expect(find.byType(GfBottomNavigation), findsNothing);
    expect(find.byType(StartupExperience), findsNothing);
    expect(find.byType(MediaHost), findsNothing);
    expect(app.pages.reads, 0);
    recovery.complete();
    await tester.pumpAndSettle();
    expect(find.byType(GfBottomNavigation), findsOneWidget);
    expect(app.pages.reads, greaterThan(0));
    await dispose(tester);
  });

  testWidgets('failed recovery stays closed until retry finishes', (
    tester,
  ) async {
    var attempts = 0;
    final recovered = Completer<void>();
    final app = await pump(tester, () {
      attempts++;
      return attempts == 1
          ? Future<void>.error(StateError('recovery failed'))
          : recovered.future;
    });
    await tester.pumpAndSettle();
    expect(find.byType(GfBottomNavigation), findsNothing);
    expect(app.pages.reads, 0);
    await tester.tap(find.byKey(const Key('storage-gate-retry')));
    await tester.pump();
    expect(attempts, 2);
    expect(find.byType(GfBottomNavigation), findsNothing);
    expect(app.pages.reads, 0);
    recovered.complete();
    await tester.pumpAndSettle();
    expect(find.byType(GfBottomNavigation), findsOneWidget);
    await dispose(tester);
  });
  testWidgets(
    'active reset unmounts routes and a failed reset remains gated through retry',
    (tester) async {
      var attempts = 0;
      final retry = Completer<void>();
      final app = await pump(tester, () {
        attempts++;
        return attempts == 1 ? Future<void>.value() : retry.future;
      });
      await tester.pumpAndSettle();
      expect(find.byType(GfBottomNavigation), findsOneWidget);
      final reads = app.pages.reads;
      app.container.read(storageResetStateProvider.notifier).state =
          const AsyncValue<void>.loading();
      await tester.pump();
      expect(find.byType(GfBottomNavigation), findsNothing);
      expect(find.byType(StartupExperience), findsNothing);
      expect(find.byType(MediaHost), findsNothing);
      app.container
          .read(storageResetStateProvider.notifier)
          .state = AsyncValue<void>.error(
        StateError('private internal path'),
        StackTrace.current,
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('storage-gate-retry')), findsOneWidget);
      expect(find.textContaining('private internal path'), findsNothing);
      expect(app.pages.reads, reads);
      await tester.tap(find.byKey(const Key('storage-gate-retry')));
      await tester.pump();
      expect(attempts, 2);
      expect(find.byType(GfBottomNavigation), findsNothing);
      expect(find.byKey(const Key('storage-gate-retry')), findsNothing);
      app.container.read(storageResetStateProvider.notifier).state = null;
      await tester.pump();
      // The previous successful bootstrap value cannot bypass a retry in flight.
      expect(find.byType(GfBottomNavigation), findsNothing);
      retry.complete();
      await tester.pumpAndSettle();
      expect(find.byType(GfBottomNavigation), findsOneWidget);
      await dispose(tester);
    },
  );

  testWidgets(
    'a bootstrap refresh carrying previous success still unmounts routes',
    (tester) async {
      var attempts = 0;
      final retry = Completer<void>();
      final app = await pump(
        tester,
        () => ++attempts == 1 ? Future<void>.value() : retry.future,
      );
      await tester.pumpAndSettle();
      expect(find.byType(GfBottomNavigation), findsOneWidget);
      app.container.invalidate(storageBootstrapProvider);
      // invalidate schedules a microtask; read starts that refresh before the
      // next frame so this checks a loading value carrying previous success.
      final refreshing = app.container.read(storageBootstrapProvider);
      expect(refreshing.isLoading, isTrue);
      expect(refreshing.hasValue, isTrue);
      // Flush the invalidation microtask before rendering one frame. A pump
      // without a duration only schedules that frame when the tree is idle.
      await tester.pump(Duration.zero);
      expect(find.byType(GfBottomNavigation), findsNothing);
      expect(find.byType(StartupExperience), findsNothing);
      expect(find.byType(MediaHost), findsNothing);
      retry.complete();
      await tester.pumpAndSettle();
      expect(find.byType(GfBottomNavigation), findsOneWidget);
      await dispose(tester);
    },
  );
}

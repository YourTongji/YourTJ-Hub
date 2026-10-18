import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  testWidgets(
    'bounded editor keeps its draft and full-screen routes escape the column',
    (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(1440, 900);
      addTearDown(tester.view.reset);
      final input = TextEditingController();
      final scroll = ScrollController();
      addTearDown(input.dispose);
      addTearDown(scroll.dispose);
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.dark),
          home: ReadingPageWindow(
            child: Builder(
              builder: (context) => Scaffold(
                body: SingleChildScrollView(
                  controller: scroll,
                  child: Column(
                    children: [
                      TextField(controller: input),
                      TextButton(
                        onPressed: () => Navigator.of(context).push(
                          MaterialPageRoute<void>(
                            builder: (_) =>
                                const Scaffold(key: Key('full-screen-media')),
                          ),
                        ),
                        child: const Text('Open media'),
                      ),
                      const SizedBox(height: 2000),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), 'Unsubmitted iPad draft');
      await tester.tap(find.text('Open media'));
      await tester.pumpAndSettle();
      expect(
        tester.getSize(find.byKey(const Key('full-screen-media'))).width,
        1440,
      );
      Navigator.of(
        tester.element(find.byKey(const Key('full-screen-media'))),
      ).pop();
      await tester.pumpAndSettle();
      // Finish editing before reading elsewhere; a focused native field
      // intentionally scrolls itself into view when the keyboard resizes.
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      expect(scroll.position.maxScrollExtent, greaterThan(180));
      scroll.jumpTo(180);
      await tester.pumpAndSettle();
      final inputState = tester.state(find.byType(TextField));
      for (final width in [768.0, 390.0, 1440.0]) {
        tester.view.physicalSize = Size(width, 900);
        tester.view.viewInsets = const FakeViewPadding(bottom: 300);
        await tester.pumpAndSettle();
        expect(tester.state(find.byType(TextField)), same(inputState));
        expect(input.text, 'Unsubmitted iPad draft');
        expect(scroll.offset, 180);
        expect(tester.takeException(), isNull);
      }
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'pushed settings bounds its complete page and keeps scroll on resize',
    (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(1440, 360);
      tester.view.padding = const FakeViewPadding(
        top: 24,
        bottom: 20,
        left: 44,
        right: 32,
      );
      tester.view.viewPadding = const FakeViewPadding(
        top: 24,
        bottom: 20,
        left: 44,
        right: 32,
      );
      addTearDown(tester.view.reset);
      final route = appRouter.configuration.routes
          .whereType<GoRoute>()
          .singleWhere((route) => route.path == '/settings');
      final router = GoRouter(initialLocation: '/settings', routes: [route]);
      addTearDown(router.dispose);
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
        ],
      );
      addTearDown(container.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            theme: gfThemeData(Brightness.light),
            locale: const Locale('de'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(textScaler: const TextScaler.linear(2)),
              child: child!,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final page = find.byType(SettingsPage);
      expect(tester.getSize(page).width, 1120);
      expect(tester.getSize(find.byType(GfAppBar)).width, 1120);
      final state = tester.state(page);
      final context = tester.element(page);
      expect(MediaQuery.sizeOf(context).width, 1120);
      expect(MediaQuery.paddingOf(context).left, 0);
      expect(MediaQuery.paddingOf(context).right, 0);
      expect(MediaQuery.paddingOf(context).top, 24);
      expect(MediaQuery.paddingOf(context).bottom, 20);
      final scroll = tester
          .state<ScrollableState>(find.byType(Scrollable).first)
          .position;
      expect(scroll.maxScrollExtent, greaterThan(180));
      scroll.jumpTo(180);
      await tester.pumpAndSettle();
      for (final width in [1024.0, 768.0, 600.0, 390.0, 1440.0]) {
        tester.view.physicalSize = Size(width, 360);
        await tester.pumpAndSettle();
        expect(tester.state(page), same(state));
        expect(router.state.uri.path, '/settings');
        expect(scroll.maxScrollExtent, greaterThan(180));
        expect(scroll.pixels, 180);
        expect(tester.getSize(page).width, lessThanOrEqualTo(1120));
        expect(tester.takeException(), isNull);
      }
    },
  );
}

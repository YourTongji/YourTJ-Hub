import 'dart:ui' show Tristate, SemanticsAction;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:forum_app/src/navigation/reading_chrome.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/widgets/root_surface.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show MemTokenStorage, NoopCache;

class _ReadingPage extends StatefulWidget {
  const _ReadingPage(this.name);
  final String name;
  @override
  State<_ReadingPage> createState() => _ReadingPageState();
}

class _ReadingPageState extends State<_ReadingPage> {
  final scroll = ScrollController();
  final input = TextEditingController();
  @override
  void dispose() {
    scroll.dispose();
    input.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => RootSurface(
    title: widget.name,
    body: (top, bottom) => ListView(
      key: ValueKey('body-${widget.name}'),
      controller: scroll,
      padding: EdgeInsets.only(top: top, bottom: bottom),
      children: [
        TextField(controller: input),
        for (var i = 0; i < 50; i++)
          SizedBox(height: 60, child: Text('row $i')),
      ],
    ),
  );
}

void main() {
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    testWidgets(
      'adaptive shell keeps reading state and safe insets ($locale)',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = const Size(390, 844);
        tester.view.padding = const FakeViewPadding(top: 44, bottom: 34);
        tester.view.viewPadding = const FakeViewPadding(top: 44, bottom: 34);
        addTearDown(tester.view.reset);
        final container = ProviderContainer(
          overrides: [
            tokenStorageProvider.overrideWithValue(MemTokenStorage()),
            currentUserProvider.overrideWith((_) async => null),
            offlineTopicCacheProvider.overrideWithValue(NoopCache()),
            offlineChatCacheProvider.overrideWithValue(NoopCache()),
            accountLayoutProvider.overrideWith(
              (_) async => LayoutPayload.fromJson(minimalLayoutJson()),
            ),
          ],
        );
        addTearDown(container.dispose);
        final router = GoRouter(
          routes: [
            StatefulShellRoute.indexedStack(
              builder: (_, _, shell) => GfShell(navigationShell: shell),
              branches: [
                for (final path in [
                  '/',
                  '/campus',
                  '/notifications',
                  '/messages',
                ])
                  StatefulShellBranch(
                    routes: [
                      GoRoute(
                        path: path,
                        builder: (_, _) => _ReadingPage(path),
                      ),
                    ],
                  ),
              ],
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              theme: gfThemeData(Brightness.light),
              locale: Locale(locale),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(context).copyWith(
                  textScaler: const TextScaler.linear(2),
                  disableAnimations: true,
                ),
                child: child!,
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final state = tester.state<_ReadingPageState>(
          find.byType(_ReadingPage),
        );
        state.input.text = 'Retained work';
        state.scroll.jumpTo(300);
        await tester.pumpAndSettle();
        expect(find.byType(GfBottomNavigation), findsOneWidget);

        for (final width in [600.0, 768.0, 1024.0, 1440.0]) {
          tester.view.physicalSize = Size(width, 420);
          await tester.pumpAndSettle();
          expect(find.byType(ReadingNavigationRail), findsOneWidget);
          expect(find.byType(GfBottomNavigation), findsNothing);
          expect(
            tester.getSize(find.byKey(const ValueKey('body-/'))).width,
            lessThanOrEqualTo(720),
          );
          expect(tester.state(find.byType(_ReadingPage)), same(state));
          expect(state.input.text, 'Retained work');
          expect(state.scroll.offset, 300);
          expect(tester.takeException(), isNull);
        }
        tester.view.physicalSize = const Size(1440, 200);
        await tester.pumpAndSettle();
        container.read(readingChromeProvider).show();
        await tester.drag(
          find.byType(ReadingNavigationRail),
          const Offset(0, -100),
        );
        await tester.pumpAndSettle();
        expect(
          container.read(readingChromeProvider).hidden,
          isFalse,
          reason:
              'Scrolling persistent destinations must not hide reading controls',
        );
        expect(state.scroll.offset, 300);
        tester.view.physicalSize = const Size(1440, 420);
        tester.view.padding = const FakeViewPadding(
          top: 44,
          bottom: 34,
          right: 44,
        );
        tester.view.viewPadding = const FakeViewPadding(
          top: 44,
          bottom: 34,
          right: 44,
        );
        await tester.pumpAndSettle();
        final l10n = AppLocalizations.of(tester.element(find.byType(GfShell)));
        final semantics = tester.ensureSemantics();
        await tester.pumpAndSettle();
        final campusNode = tester.getSemantics(
          find.byKey(const ValueKey('rail-destination-1')),
        );
        expect(campusNode.label, contains(l10n.navCampus));
        expect(campusNode.flagsCollection.isButton, isTrue);
        expect(
          campusNode.getSemanticsData().hasAction(SemanticsAction.tap),
          isTrue,
        );
        final homeNode = tester.getSemantics(
          find.byKey(const ValueKey('rail-destination-0')),
        );
        expect(homeNode.flagsCollection.isSelected, Tristate.isTrue);
        semantics.dispose();
        final readingBounds = tester.getRect(
          find.byKey(const ValueKey('body-/')),
        );
        await tester.tap(find.byTooltip(l10n.navPublish));
        await tester.pumpAndSettle();
        expect(
          tester.getRect(find.byKey(const ValueKey('compose-menu'))).right,
          closeTo(readingBounds.right - 16, 1),
        );
        await tester.tap(find.byTooltip(l10n.commonClose));
        await tester.pumpAndSettle();
        tester.view.viewInsets = const FakeViewPadding(bottom: 160);
        await tester.pumpAndSettle();
        expect(find.byType(ReadingNavigationRail), findsOneWidget);
        expect(tester.state(find.byType(_ReadingPage)), same(state));
        expect(state.input.text, 'Retained work');
        expect(tester.takeException(), isNull);
        tester.view.viewInsets = const FakeViewPadding();
        await tester.pumpAndSettle();
        await tester.tap(find.byTooltip(l10n.navCampus));
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/campus');
        expect(
          tester.getSize(find.byKey(const ValueKey('body-/campus'))).width,
          greaterThan(720),
        );
        await tester.tap(find.byTooltip(l10n.navHome));
        await tester.pumpAndSettle();
        expect(state.scroll.offset, 300);
        tester.view.physicalSize = const Size(320, 640);
        await tester.pumpAndSettle();
        expect(find.byType(ReadingNavigationRail), findsNothing);
        expect(find.byType(GfBottomNavigation), findsOneWidget);
        expect(tester.state(find.byType(_ReadingPage)), same(state));
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
      },
    );
  }
}

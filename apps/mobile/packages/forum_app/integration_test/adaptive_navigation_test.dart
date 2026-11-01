import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:forum_app/src/navigation/window_reserved_regions.dart';
import 'package:forum_app/src/navigation/reading_chrome.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/widgets/root_surface.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:ui_kit/ui_kit.dart';

import '../test/fixtures/page_fixtures.dart';
import '../test/pages_behavior_test.dart' show MemTokenStorage, NoopCache;

/// Exercises the production shell with synthetic content and window insets.
/// These sizes are stress cases, not specifications of a particular device.
/// Run on iOS for glass acceptance. With YOURTJ_TEST_NATIVE_WINDOW=true,
/// Device Hub drives real window changes instead of the synthetic stress cases.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  const nativeWindow = bool.fromEnvironment('YOURTJ_TEST_NATIVE_WINDOW');
  testWidgets('native navigation preserves work across asymmetric windows', (
    tester,
  ) async {
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
    final router = GoRouter(
      routes: [
        StatefulShellRoute.indexedStack(
          builder: (_, _, shell) => GfShell(navigationShell: shell),
          branches: [
            for (final path in ['/', '/campus', '/notifications', '/messages'])
              StatefulShellBranch(
                routes: [
                  GoRoute(
                    path: path,
                    builder: (_, _) => const _DemoReadingPage(),
                  ),
                ],
              ),
          ],
        ),
      ],
    );
    final scenario = ValueNotifier(0);
    const sizes = [
      Size(400, 700),
      Size(840, 700),
      Size(640, 700),
      Size(840, 700),
    ];
    const insets = [
      EdgeInsets.fromLTRB(0, 24, 56, 20),
      EdgeInsets.fromLTRB(24, 24, 76, 20),
      EdgeInsets.fromLTRB(96, 24, 0, 20),
      EdgeInsets.fromLTRB(76, 24, 24, 20),
    ];
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: ValueListenableBuilder<int>(
          valueListenable: scenario,
          builder: (_, index, _) => MaterialApp.router(
            theme: gfThemeData(
              index >= 4 ? Brightness.dark : Brightness.light,
            ).copyWith(platform: TargetPlatform.iOS),
            routerConfig: router,
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => nativeWindow
                ? child!
                : Align(
                    alignment: Alignment.topLeft,
                    child: SizedBox.fromSize(
                      size: sizes[index % 4],
                      child: MediaQuery(
                        data: MediaQuery.of(context).copyWith(
                          size: sizes[index % 4],
                          padding: insets[index % 4],
                          viewPadding: insets[index % 4],
                          viewInsets: EdgeInsets.zero,
                        ),
                        child: child!,
                      ),
                    ),
                  ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final state = tester.state<_DemoReadingPageState>(
      find.byType(_DemoReadingPage),
    );
    state.input.text = '未发布的演示草稿';
    state.scroll.jumpTo(160);
    await tester.pumpAndSettle();
    container.read(readingChromeProvider).show();
    await tester.pumpAndSettle();
    var previousWindow = MediaQueryData.fromView(tester.view);
    for (var i = 0; i < 8; i++) {
      if (nativeWindow && i % 4 != 0) {
        // The native window must change externally: open, rotate, then close
        // the Duo in Device Hub. Repeat after the dark theme starts.
        debugPrint('Duo acceptance: waiting for window transition $i');
        final deadline = DateTime.now().add(const Duration(minutes: 2));
        while (true) {
          await tester.pump(const Duration(milliseconds: 100));
          final current = MediaQueryData.fromView(tester.view);
          if (current.size != previousWindow.size ||
              current.padding != previousWindow.padding) {
            break;
          }
          expect(
            DateTime.now().isBefore(deadline),
            isTrue,
            reason: 'Change the Duo pose/window in Device Hub',
          );
        }
      }
      scenario.value = i;
      await tester.pumpAndSettle();
      expect(tester.state(find.byType(_DemoReadingPage)), same(state));
      expect(state.input.text, '未发布的演示草稿');
      expect(state.scroll.offset, 160);
      var window = MediaQueryData.fromView(tester.view);
      WindowReservedRegions? regions;
      if (nativeWindow) {
        final deadline = DateTime.now().add(const Duration(seconds: 15));
        do {
          await tester.pump(const Duration(milliseconds: 100));
          window = MediaQueryData.fromView(tester.view);
          regions = container.read(windowReservedRegionsProvider).value;
        } while ((regions == null ||
                regions.size != window.size ||
                regions.safeArea != window.viewPadding) &&
            DateTime.now().isBefore(deadline));
        expect(
          regions,
          isNotNull,
          reason: 'iOS 27.1 must publish reserved regions',
        );
      }
      final edgeFrame = regions?.navigationFrame(window, window.size);
      if (nativeWindow) {
        debugPrint(
          'Duo acceptance $i: size=${window.size} '
          'padding=${window.padding} edge=$edgeFrame '
          'nativeSize=${regions?.size} safe=${regions?.safeArea} '
          'occlusions=${regions?.occlusions}',
        );
      }
      if (nativeWindow && i == 0) {
        expect(
          edgeFrame,
          isNotNull,
          reason: 'Start Duo closed with its clear side pocket',
        );
      }
      previousWindow = window;
      final wide = nativeWindow
          ? edgeFrame != null ||
                window.size.width - window.padding.horizontal >= 600
          : i % 2 == 1;
      expect(
        find.byType(ReadingNavigationRail),
        wide ? findsOneWidget : findsNothing,
      );
      expect(
        find.byType(GfBottomNavigation),
        wide ? findsNothing : findsOneWidget,
      );
      if (edgeFrame != null) {
        final rail = tester.getRect(find.byType(ReadingNavigationRail));
        final content = tester.getRect(find.byType(RootSurface));
        expect(rail.overlaps(content), isTrue);
        // Reading can flow behind the translucent rail, but the final item
        // must be scrollable completely above it.
        // Lazy row measurements can refine maxScrollExtent after the jump.
        for (var attempt = 0; attempt < 4; attempt++) {
          state.scroll.jumpTo(state.scroll.position.maxScrollExtent);
          await tester.pumpAndSettle();
          if ((state.scroll.position.maxScrollExtent - state.scroll.offset)
                  .abs() <
              .5) {
            break;
          }
        }
        expect(
          state.scroll.offset,
          closeTo(state.scroll.position.maxScrollExtent, .5),
        );
        final glass = tester.getRect(
          find.descendant(
            of: find.byType(ReadingNavigationRail),
            matching: find.byType(GfNavigationSurface),
          ),
        );
        expect(
          tester.getRect(find.byKey(const ValueKey('demo-row-29'))).bottom,
          lessThan(glass.top),
        );
        state.scroll.jumpTo(160);
        await tester.pumpAndSettle();
        for (final occlusion in regions!.occlusions) {
          expect(rail.overlaps(occlusion), isFalse);
        }
      }
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot(
          '${nativeWindow ? 'duo-native' : 'adaptive-navigation'}-$i',
        );
      }
      // Exercise the production shell's menu on both a bottom dock and the
      // native side pocket. The guest gets only public campus tools.
      final campus = find.byKey(
        ValueKey(wide ? 'rail-destination-1' : 'gf-bottom-navigation-target-1'),
      );
      await tester.longPress(campus);
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('gf-dock-menu-row-0')), findsOneWidget);
      expect(find.byKey(const ValueKey('gf-dock-menu-row-3')), findsNothing);
      final menu = tester.getRect(
        find.byKey(const ValueKey('gf-dock-menu-panel')),
      );
      expect(menu.left, greaterThanOrEqualTo(window.padding.left + 16));
      expect(
        menu.right,
        lessThanOrEqualTo(window.size.width - window.padding.right - 16),
      );
      if (wide) {
        final source = tester.getRect(campus);
        expect(menu.overlaps(source), isFalse);
      }
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot(
          '${nativeWindow ? 'duo-native' : 'adaptive-navigation'}-$i-menu',
        );
      }
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('gf-dock-menu-panel')), findsNothing);
      if (wide) {
        await tester.tap(find.byTooltip('校园').first);
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/campus');
        await tester.tap(find.byTooltip('首页').first);
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/');
      }
      if (nativeWindow && edgeFrame != null) {
        state.scroll.jumpTo(0);
        await tester.pumpAndSettle();
        final contentWidth = tester.getRect(find.byType(RootSurface)).width;
        await tester.tap(find.byType(TextField));
        final opening = DateTime.now().add(const Duration(seconds: 15));
        while (tester.view.viewInsets.bottom == 0 &&
            DateTime.now().isBefore(opening)) {
          await tester.pump(const Duration(milliseconds: 100));
        }
        expect(tester.view.viewInsets.bottom, greaterThan(0));
        await tester.pumpAndSettle();
        expect(find.byType(ReadingNavigationRail), findsNothing);
        expect(find.byType(GfBottomNavigation), findsNothing);
        expect(tester.getRect(find.byType(RootSurface)).width, contentWidth);
        expect(state.input.text, '未发布的演示草稿');
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot('duo-keyboard-$i');
        }
        FocusManager.instance.primaryFocus?.unfocus();
        final closing = DateTime.now().add(const Duration(seconds: 15));
        while (tester.view.viewInsets.bottom != 0 &&
            DateTime.now().isBefore(closing)) {
          await tester.pump(const Duration(milliseconds: 100));
        }
        expect(tester.view.viewInsets.bottom, 0);
        await tester.pumpAndSettle();
        expect(find.byType(ReadingNavigationRail), findsOneWidget);
        expect(tester.getRect(find.byType(RootSurface)).width, contentWidth);
        expect(tester.state(find.byType(_DemoReadingPage)), same(state));
        state.scroll.jumpTo(160);
        await tester.pumpAndSettle();
        debugPrint(
          'Duo keyboard $i: hidden and restored without a width change',
        );
      }
      expect(tester.takeException(), isNull);
    }
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
    router.dispose();
    container.dispose();
    scenario.dispose();
  });
}

class _DemoReadingPage extends StatefulWidget {
  const _DemoReadingPage();
  @override
  State<_DemoReadingPage> createState() => _DemoReadingPageState();
}

class _DemoReadingPageState extends State<_DemoReadingPage> {
  final input = TextEditingController();
  final scroll = ScrollController();
  @override
  void dispose() {
    input.dispose();
    scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => RootSurface(
    title: '窗口适配 · 演示内容',
    body: (top, bottom) => ListView(
      controller: scroll,
      padding: EdgeInsets.fromLTRB(16, top, 16, bottom),
      children: [
        TextField(controller: input),
        for (var i = 0; i < 30; i++)
          Padding(
            key: ValueKey('demo-row-$i'),
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                GfText(
                  '校园随记 ${i + 1}（演示）',
                  style: GfTheme.typographyOf(context).title3,
                ),
                const SizedBox(height: 8),
                const GfText('展开窗口继续阅读，收起后保留当前位置。导航与内容分别避让系统区域。'),
                const Divider(),
              ],
            ),
          ),
      ],
    ),
  );
}

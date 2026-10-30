import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:forum_app/src/navigation/window_reserved_regions.dart';
import 'package:forum_app/src/updates/preview_status.dart';
import 'package:forum_app/src/widgets/root_surface.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

void main() {
  const right = WindowReservedRegions(
    size: Size(466, 678),
    safeArea: EdgeInsets.only(right: 84, bottom: 34),
    occlusions: [Rect.fromLTWH(382, 0, 84, 190)],
  );
  const media = MediaQueryData(
    size: Size(466, 678),
    padding: EdgeInsets.only(right: 84, bottom: 34),
    viewPadding: EdgeInsets.only(right: 84, bottom: 34),
  );

  test(
    'side pocket follows independent insets and excludes every occlusion',
    () {
      final frame = right.navigationFrame(media, media.size)!;
      expect(frame, const Rect.fromLTRB(382, 190, 466, 644));
      const left = WindowReservedRegions(
        size: Size(466, 678),
        safeArea: EdgeInsets.only(left: 84, bottom: 34),
        occlusions: [
          Rect.fromLTWH(0, 0, 84, 190),
          Rect.fromLTWH(0, 500, 84, 144),
        ],
      );
      final leftMedia = media.copyWith(
        padding: left.safeArea,
        viewPadding: left.safeArea,
      );
      expect(
        left.navigationFrame(leftMedia, leftMedia.size),
        const Rect.fromLTRB(0, 190, 84, 500),
      );
    },
  );

  test('stale, short or keyboard-covered pockets use safe navigation', () {
    expect(
      right.navigationFrame(
        media.copyWith(size: const Size(678, 466)),
        const Size(678, 466),
      ),
      isNull,
    );
    expect(
      right.navigationFrame(
        media.copyWith(viewPadding: const EdgeInsets.only(left: 84)),
        media.size,
      ),
      isNull,
    );
    expect(
      right.navigationFrame(
        media.copyWith(viewInsets: const EdgeInsets.only(bottom: 300)),
        media.size,
      ),
      isNull,
    );
    const covered = WindowReservedRegions(
      size: Size(466, 678),
      safeArea: EdgeInsets.only(right: 84, bottom: 34),
      occlusions: [Rect.fromLTWH(382, 0, 84, 500)],
    );
    expect(covered.navigationFrame(media, media.size), isNull);
  });

  test('invalid native messages never grant outside-safe-area placement', () {
    expect(WindowReservedRegions.fromMessage(null), isNull);
    expect(
      WindowReservedRegions.fromMessage({
        'width': double.nan,
        'height': 678,
        'safeArea': [0, 0, 84, 34],
        'occlusions': [],
      }),
      isNull,
    );
    expect(
      WindowReservedRegions.fromMessage({
        'width': 466,
        'height': 678,
        'safeArea': [0, 0, 84, 34],
        'occlusions': [
          [0, 0, -1, 190],
        ],
      }),
      isNull,
    );
    final parsed = WindowReservedRegions.fromMessage({
      'width': 466,
      'height': 678,
      'safeArea': [0, 0, 84, 34],
      'occlusions': [
        [382, 0, 84, 190],
      ],
    })!;
    expect(
      parsed.navigationFrame(media, media.size),
      right.navigationFrame(media, media.size),
    );
  });

  testWidgets('content extends beneath navigation in the clear side strip', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(466, 678);
    tester.view.padding = const FakeViewPadding(right: 84, bottom: 34);
    tester.view.viewPadding = const FakeViewPadding(right: 84, bottom: 34);
    addTearDown(tester.view.reset);
    const reserved = WindowReservedRegions(
      size: Size(466, 678),
      safeArea: EdgeInsets.only(right: 84, bottom: 34),
      occlusions: [Rect.fromLTWH(382, 0, 84, 190)],
    );
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(
          Brightness.light,
        ).copyWith(platform: TargetPlatform.iOS),
        home: ReadingWindow(
          reservedRegions: reserved,
          navigationRailWidth: 88,
          rail: const SizedBox.expand(key: Key('rail')),
          bottomNavigation: const SizedBox(height: 56, key: Key('bottom')),
          child: const SizedBox.expand(key: Key('content')),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('bottom')), findsNothing);
    expect(find.byKey(const Key('rail')), findsOneWidget);
    final rail = tester.getRect(find.byKey(const Key('rail')));
    final content = tester.getRect(find.byKey(const Key('content')));
    expect(content, const Rect.fromLTWH(0, 0, 466, 678));
    expect(rail.left, greaterThanOrEqualTo(382));
    expect(rail.right, lessThanOrEqualTo(466));
    expect(rail.top, greaterThanOrEqualTo(190));
    expect(rail.bottom, lessThanOrEqualTo(644));
    expect(rail.overlaps(content), isTrue);
    expect(tester.takeException(), isNull);
  });

  for (final brightness in Brightness.values) {
    testWidgets('side icons retain accessible actions ($brightness)', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = media.size;
      addTearDown(tester.view.reset);
      final semantics = tester.ensureSemantics();
      const items = [
        GfBottomNavigationItem(
          label: '首页',
          symbol: 'home',
          selectedSymbol: 'home-filled',
        ),
        GfBottomNavigationItem(
          label: '校园',
          symbol: 'school',
          selectedSymbol: 'school-filled',
        ),
        GfBottomNavigationItem(
          label: '通知',
          symbol: 'bell',
          selectedSymbol: 'bell-filled',
          badge: true,
          badgeSemanticLabel: '未读',
        ),
        GfBottomNavigationItem(
          label: '私信',
          symbol: 'chat',
          selectedSymbol: 'chat-filled',
        ),
      ];
      var selected = 0;
      var highContrast = false;
      late StateSetter update;
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(brightness).copyWith(platform: TargetPlatform.iOS),
          home: StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return MediaQuery(
                data: media.copyWith(
                  textScaler: const TextScaler.linear(2),
                  highContrast: highContrast,
                ),
                child: ReadingWindow(
                  reservedRegions: right,
                  rail: ReadingNavigationRail(
                    items: items,
                    currentIndex: selected,
                    onSelected: (index) => setState(() => selected = index),
                  ),
                  bottomNavigation: const SizedBox.shrink(),
                  child: const SizedBox.expand(),
                ),
              );
            },
          ),
        ),
      );
      await tester.pumpAndSettle();
      final rail = find.byType(ReadingNavigationRail);
      final glass = find.descendant(
        of: rail,
        matching: find.byType(GfNavigationSurface),
      );
      final glassRect = tester.getRect(glass);
      expect(glassRect.width, 56);
      expect(glassRect.bottom, 636);
      expect(glassRect.left, greaterThanOrEqualTo(382));
      expect(
        find.descendant(of: rail, matching: find.byType(BackdropFilter)),
        findsOneWidget,
      );
      for (var i = 0; i < items.length; i++) {
        expect(find.text(items[i].label), findsNothing);
        final node = tester.getSemantics(
          find.byKey(ValueKey('rail-destination-$i')),
        );
        expect(node.label, contains(items[i].label));
        if (i == 2) expect(node.label, contains('未读'));
        final button = find.descendant(
          of: find.byTooltip(items[i].label),
          matching: find.byType(TextButton),
        );
        expect(tester.getSize(button).width, greaterThanOrEqualTo(48));
        expect(tester.getSize(button).height, greaterThanOrEqualTo(48));
        await tester.tap(button);
        await tester.pumpAndSettle();
        expect(selected, i);
      }
      update(() => highContrast = true);
      await tester.pumpAndSettle();
      expect(
        find.descendant(of: rail, matching: find.byType(BackdropFilter)),
        findsNothing,
      );
      semantics.dispose();
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('preview notice leaves navigation in the system side pocket', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = media.size;
    tester.view.padding = const FakeViewPadding(right: 84, bottom: 34);
    tester.view.viewPadding = const FakeViewPadding(right: 84, bottom: 34);
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: PreviewStatusHost(
          fetchStatus: () async => null,
          child: ReadingWindow(
            reservedRegions: right,
            rail: const SizedBox.expand(key: Key('rail')),
            bottomNavigation: const SizedBox(height: 56, key: Key('bottom')),
            child: const SizedBox.expand(key: Key('content')),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining('Cannot verify'), findsOneWidget);
    expect(find.byKey(const Key('bottom')), findsNothing);
    final rail = tester.getRect(find.byKey(const Key('rail')));
    final content = tester.getRect(find.byKey(const Key('content')));
    expect(content.top, greaterThan(0));
    expect(rail.top, greaterThanOrEqualTo(190));
    expect(rail.bottom, 644);
    expect(content.right, 466);
    expect(rail.overlaps(content), isTrue);
    await tester.pumpWidget(const SizedBox.shrink());
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'edge content fills the view while header and compose stay safe',
    (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = media.size;
      tester.view.padding = const FakeViewPadding(right: 84, bottom: 34);
      tester.view.viewPadding = const FakeViewPadding(right: 84, bottom: 34);
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            accountLayoutProvider.overrideWith(
              (_) async => LayoutPayload.fromJson(minimalLayoutJson()),
            ),
          ],
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ReadingWindow(
              reservedRegions: right,
              rail: const SizedBox.expand(key: Key('rail')),
              bottomNavigation: const SizedBox.shrink(),
              child: RootSurface(
                actions: [
                  IconButton(
                    key: const Key('search'),
                    onPressed: () {},
                    icon: const Icon(Icons.search),
                  ),
                ],
                body: (top, bottom) => const SizedBox.expand(key: Key('body')),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.getRect(find.byKey(const Key('body'))).width, 466);
      expect(
        tester.getRect(find.byKey(const Key('search'))).right,
        lessThanOrEqualTo(382),
      );
      expect(
        tester.getRect(find.byKey(const ValueKey('root-compose-action'))).right,
        366,
      );
      expect(tester.takeException(), isNull);
    },
  );
}

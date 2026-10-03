import 'dart:ui' as ui;
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:core/core.dart';
import 'package:forum_app/src/navigation/reading_chrome.dart';
import 'package:forum_app/src/widgets/root_surface.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/pages/schedule/scheduler_web_tip.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';

void main() {
  testWidgets('an empty feed still supports pull to refresh', (tester) async {
    var refreshed = 0;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: RefreshIndicator(
            onRefresh: () async {
              refreshed++;
            },
            child: GfTopicList(
              loading: false,
              topics: const [],
              hasMore: false,
              onLoadMore: () {},
            ),
          ),
        ),
      ),
    );
    await tester.drag(find.text('暂无话题'), const Offset(0, 400));
    await tester.pumpAndSettle();
    expect(refreshed, 1);
  });
  testWidgets('reading controls move without changing the content viewport', (
    tester,
  ) async {
    final container = ProviderContainer(
      overrides: [
        accountLayoutProvider.overrideWith(
          (_) async => LayoutPayload.fromJson(minimalLayoutJson()),
        ),
      ],
    );
    addTearDown(container.dispose);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetDevicePixelRatio);
    tester.view.padding = FakeViewPadding(top: 62);
    addTearDown(tester.view.resetPadding);
    final boundary = GlobalKey();
    final scroll = ScrollController();
    addTearDown(scroll.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AccountDrawerLayer(
            key: accountDrawerLayerKey,
            onChanged: (open) {
              if (open) container.read(readingChromeProvider).show();
            },
            child: RepaintBoundary(
              key: boundary,
              child: RootSurface(
                title: 'YourTJ',
                showLogo: true,
                onAction: () {},
                body: (top, bottom) => ListView(
                  controller: scroll,
                  padding: EdgeInsets.only(top: top, bottom: bottom),
                  children: List.generate(
                    30,
                    (i) => SizedBox(height: 60, child: Text('row $i')),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    scroll.jumpTo(120);
    await tester.pump();
    final position = tester.getTopLeft(find.text('row 4'));
    final viewport = scroll.position.viewportDimension;
    // Mid-gesture the header tracks the finger instead of waiting.
    final logo = tester.getTopLeft(find.byType(GfLogo)).dy;
    container.read(readingChromeProvider).update(32, 120);
    await tester.pump();
    expect(tester.getTopLeft(find.byType(GfLogo)).dy, closeTo(logo - 28.5, 1));
    expect(tester.getTopLeft(find.text('row 4')), position);
    container.read(readingChromeProvider)
      ..update(40, 120)
      ..settle(120);
    await tester.pumpAndSettle();
    expect(tester.getTopLeft(find.text('row 4')), position);
    expect(scroll.offset, 120);
    expect(scroll.position.viewportDimension, viewport);
    await tester.runAsync(() async {
      final image =
          await (boundary.currentContext!.findRenderObject()
                  as RenderRepaintBoundary)
              .toImage();
      final pixels = (await image.toByteData(
        format: ui.ImageByteFormat.rawRgba,
      ))!;
      var ink = 0;
      for (var y = 0; y < 62; y++) {
        for (var x = 0; x < image.width; x++) {
          final index = (y * image.width + x) * 4;
          if (pixels.getUint8(index) < 200 &&
              pixels.getUint8(index + 1) < 200 &&
              pixels.getUint8(index + 2) < 200) {
            ink++;
          }
        }
      }
      image.dispose();
      expect(
        ink,
        0,
        reason: 'hidden toolbar must not paint in the system safe area',
      );
    });

    container.read(readingChromeProvider).show();
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('我的'));
    await tester.pumpAndSettle();
    expect(find.byType(Drawer), findsOneWidget);
  });
  testWidgets('scheduler tip exposes the full public web destination', (
    tester,
  ) async {
    var opened = false;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(body: SchedulerWebTip(onOpen: () => opened = true)),
      ),
    );
    expect(fullSchedulerUri.toString(), 'https://f.yourtj.de/schedule');
    expect(fullSchedulerUri.userInfo, isEmpty);
    await tester.tap(find.byType(TextButton));
    expect(opened, isTrue);
  });
  test('chrome follows the finger and settles in the last direction', () {
    final chrome = ReadingChrome();
    addTearDown(chrome.dispose);
    chrome.update(16, 200);
    expect(chrome.reveal.value.value, .75);
    expect(chrome.reveal.value.animate, isFalse);
    expect(chrome.hidden, isFalse);
    chrome.update(24, 224);
    expect(chrome.reveal.value.value, .375);
    expect(chrome.hidden, isTrue);
    chrome.settle(224);
    expect(chrome.reveal.value.value, 0);
    expect(chrome.reveal.value.animate, isTrue);
    // Any upward travel pulls chrome back immediately.
    chrome.update(-16, 208);
    expect(chrome.reveal.value.value, .25);
    chrome.settle(208);
    expect(chrome.reveal.value.value, 1);
    expect(chrome.hidden, isFalse);
  });
  test('the top keeps chrome attached; bounce and locks are ignored', () {
    final chrome = ReadingChrome();
    addTearDown(chrome.dispose);
    chrome.update(32, 32);
    expect(chrome.reveal.value.value, .5);
    chrome.settle(32);
    expect(chrome.reveal.value.value, 1);
    chrome.update(200, 400);
    expect(chrome.hidden, isTrue);
    chrome.update(-30, 1030, maxScrollExtent: 1000);
    expect(chrome.reveal.value.value, 0);
    chrome.update(-80, -20);
    expect(chrome.hidden, isFalse);
    chrome.update(100, 200, locked: true);
    expect(chrome.reveal.value.value, 1);
    chrome.update(64, 264);
    expect(chrome.hidden, isTrue);
    chrome.show();
    expect(chrome.hidden, isFalse);
  });
}

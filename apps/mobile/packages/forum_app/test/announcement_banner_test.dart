import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/announcement_banner.dart';

void main() {
  Future<void> pump(
    WidgetTester tester,
    AnnouncementPayload announcement, {
    Locale locale = const Locale('zh'),
  }) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: locale,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Column(
              children: [AnnouncementBanner(announcement: announcement)],
            ),
          ),
        ),
      ),
    );
    await tester.pump();
  }

  testWidgets('announcement body is indented alongside a leading bell', (
    tester,
  ) async {
    await pump(
      tester,
      const AnnouncementPayload(
        enabled: true,
        html: '<p>Announcement body</p>',
      ),
    );
    final bell = find.byWidgetPredicate(
      (widget) => widget is GfSymbol && widget.name == 'bell',
    );
    expect(bell, findsOneWidget);
    final text = find.text('Announcement body', findRichText: true).last;
    expect(
      tester.getTopLeft(text).dx,
      greaterThanOrEqualTo(tester.getTopRight(bell).dx + 10),
    );
  });

  testWidgets('HTML-only announcements display their body', (tester) async {
    await pump(
      tester,
      const AnnouncementPayload(
        enabled: true,
        html: '',
        items: [
          AnnouncementItemPayload(
            id: '1',
            title: '',
            html: '<p>欢迎来到 <strong>YourTJ</strong></p>',
          ),
        ],
      ),
    );
    expect(find.textContaining('欢迎来到', findRichText: true), findsWidgets);
  });
  testWidgets('announcement chrome follows the selected locale', (
    tester,
  ) async {
    await pump(
      tester,
      const AnnouncementPayload(
        enabled: true,
        html: '<p>System maintenance</p>',
      ),
      locale: const Locale('en'),
    );

    expect(find.text('Announcement'), findsOneWidget);
    expect(find.text('Collapse'), findsOneWidget);
  });
  testWidgets('legacy HTML announcements remain readable', (tester) async {
    await pump(
      tester,
      const AnnouncementPayload(enabled: true, html: '<p>选课提醒</p>'),
    );
    expect(find.text('选课提醒', findRichText: true), findsWidgets);
  });
  testWidgets('empty items do not leave a colored empty strip', (tester) async {
    await pump(
      tester,
      const AnnouncementPayload(
        enabled: true,
        html: '',
        items: [AnnouncementItemPayload(id: 'empty', title: '', html: '')],
      ),
    );
    expect(tester.getSize(find.byType(AnnouncementBanner)).height, 0);
  });
  testWidgets(
    'rotation uses refreshed items and stops for a single replacement',
    (tester) async {
      await pump(
        tester,
        const AnnouncementPayload(
          enabled: true,
          html: '',
          items: [
            AnnouncementItemPayload(
              id: '1',
              title: 'First',
              html: '<p>Body one</p>',
            ),
            AnnouncementItemPayload(
              id: '2',
              title: 'Second',
              html: '<p>Body two</p>',
            ),
          ],
        ),
      );
      await tester.pump(const Duration(seconds: 5));
      expect(find.text('Second'), findsOneWidget);
      await pump(
        tester,
        const AnnouncementPayload(
          enabled: true,
          html: '',
          items: [
            AnnouncementItemPayload(
              id: '3',
              title: 'Replacement',
              html: '<p>Updated body</p>',
            ),
          ],
        ),
      );
      await tester.pump(const Duration(seconds: 10));
      expect(find.text('Replacement'), findsOneWidget);
      expect(find.text('Second'), findsNothing);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'large text grows the banner and disabled announcements stay hidden',
    (tester) async {
      const payload = AnnouncementPayload(
        enabled: true,
        html:
            '<p>A complete announcement with a link: <a href="/about">About YourTJ</a></p>',
      );
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: MediaQuery(
              data: const MediaQueryData(textScaler: TextScaler.linear(2)),
              child: const Scaffold(
                body: SingleChildScrollView(
                  child: SizedBox(
                    width: 320,
                    child: AnnouncementBanner(announcement: payload),
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      expect(
        tester.getSize(find.byType(AnnouncementBanner)).height,
        greaterThan(38),
      );
      expect(
        find.textContaining('About YourTJ', findRichText: true),
        findsWidgets,
      );
      expect(tester.takeException(), isNull);
      await pump(tester, payload.copyWith(enabled: false));
      expect(tester.getSize(find.byType(AnnouncementBanner)).height, 0);
    },
  );

  testWidgets(
    'announcement can be collapsed to single-line ticker and re-expanded',
    (tester) async {
      await pump(
        tester,
        const AnnouncementPayload(
          enabled: true,
          html: '',
          items: [
            AnnouncementItemPayload(
              id: '1',
              title: '维护通知',
              html: '<p>今晚 22:00 进行系统维护</p>',
            ),
          ],
        ),
      );

      // Initial state: expanded
      expect(find.text('收起'), findsOneWidget);
      expect(find.text('今晚 22:00 进行系统维护', findRichText: true), findsWidgets);
      final expandedHeight = tester
          .getSize(find.byType(AnnouncementBanner))
          .height;

      // Tap collapse button
      await tester.tap(find.byKey(const Key('announcement-collapse-btn')));
      await tester.pumpAndSettle();

      // Collapsed state: compact snippet visible, full body hidden
      expect(find.text('收起'), findsNothing);
      expect(
        find.byKey(const Key('announcement-collapsed-tap')),
        findsOneWidget,
      );
      final collapsedHeight = tester
          .getSize(find.byType(AnnouncementBanner))
          .height;
      expect(collapsedHeight, lessThan(expandedHeight));

      // Tap collapsed ticker to re-expand
      await tester.tap(find.byKey(const Key('announcement-collapsed-tap')));
      await tester.pumpAndSettle();

      // Re-expanded
      expect(find.text('收起'), findsOneWidget);
      expect(find.text('今晚 22:00 进行系统维护', findRichText: true), findsWidgets);
    },
  );

  testWidgets(
    'announcement keeps collapsed state after scrolling out and back',
    (tester) async {
      const payload = AnnouncementPayload(
        enabled: true,
        html: '',
        items: [
          AnnouncementItemPayload(
            id: '1',
            title: '维护通知',
            html: '<p>今晚 22:00 进行系统维护</p>',
          ),
        ],
      );
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: ListView(
                children: const [
                  AnnouncementBanner(announcement: payload),
                  SizedBox(height: 720),
                ],
              ),
            ),
          ),
        ),
      );
      await tester.pump();

      await tester.tap(find.byKey(const Key('announcement-collapse-btn')));
      await tester.pumpAndSettle();
      expect(
        find.byKey(const Key('announcement-collapsed-tap')),
        findsOneWidget,
      );

      await tester.drag(find.byType(ListView), const Offset(0, -500));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(ListView), const Offset(0, 500));
      await tester.pumpAndSettle();

      expect(
        find.byKey(const Key('announcement-collapsed-tap')),
        findsOneWidget,
      );
      expect(find.text('收起'), findsNothing);
    },
  );

  testWidgets('collapsed state is shared across latest hot and popular tabs', (
    tester,
  ) async {
    const payload = AnnouncementPayload(
      enabled: true,
      html: '',
      items: [
        AnnouncementItemPayload(
          id: '1',
          title: '维护通知',
          html: '<p>今晚 22:00 进行系统维护</p>',
        ),
      ],
    );
    int selectedTab = 0;
    bool collapsed = false;

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: StatefulBuilder(
              builder: (context, setState) => ListView(
                children: [
                  for (int index = 0; index < 3; index++)
                    TextButton(
                      key: Key('announcement-tab-$index'),
                      onPressed: () => setState(() => selectedTab = index),
                      child: Text('$index'),
                    ),
                  AnnouncementBanner(
                    key: ValueKey<int>(selectedTab),
                    announcement: payload,
                    collapsed: collapsed,
                    onCollapsedChanged: (value) =>
                        setState(() => collapsed = value),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('announcement-collapse-btn')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('announcement-collapsed-tap')), findsOneWidget);

    for (final int tab in [1, 2, 0]) {
      await tester.tap(find.byKey(Key('announcement-tab-$tab')));
      await tester.pumpAndSettle();
      expect(
        find.byKey(const Key('announcement-collapsed-tap')),
        findsOneWidget,
        reason: 'tab $tab should use the shared collapsed state',
      );
      expect(find.text('收起'), findsNothing);
    }
  });

  testWidgets('fluid pill dots and prev/next chevrons switch announcements', (
    tester,
  ) async {
    await pump(
      tester,
      const AnnouncementPayload(
        enabled: true,
        html: '',
        items: [
          AnnouncementItemPayload(
            id: '1',
            title: 'First Notice',
            html: '<p>Content 1</p>',
          ),
          AnnouncementItemPayload(
            id: '2',
            title: 'Second Notice',
            html: '<p>Content 2</p>',
          ),
        ],
      ),
    );

    expect(find.text('First Notice'), findsOneWidget);
    expect(find.text('Second Notice'), findsNothing);

    // Tap second dot
    await tester.tap(find.byKey(const Key('announcement-dot-1')));
    await tester.pumpAndSettle();

    expect(find.text('Second Notice'), findsOneWidget);
    expect(find.text('First Notice'), findsNothing);

    // Tap prev button
    await tester.tap(find.byKey(const Key('announcement-prev-btn')));
    await tester.pumpAndSettle();

    expect(find.text('First Notice'), findsOneWidget);

    // Tap next button
    await tester.tap(find.byKey(const Key('announcement-next-btn')));
    await tester.pumpAndSettle();

    expect(find.text('Second Notice'), findsOneWidget);
  });
}

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/src/widgets/announcement_banner.dart';

void main() {
  Future<void> pump(
    WidgetTester tester,
    AnnouncementPayload announcement,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
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
}

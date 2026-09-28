import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('profile badges and statistics stay compact and left aligned', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      gfApp(
        SingleChildScrollView(
          child: GfUserCard(
            avatarUrl: '',
            name: 'Alice',
            username: 'alice',
            details: const SizedBox(
              key: ValueKey('profile-test-meta-row'),
              height: 44,
            ),
            coloredBadges: const [
              GfUserBadge(label: 'Badge one', color: Colors.blue),
              GfUserBadge(label: 'Badge two', color: Colors.green),
              GfUserBadge(label: 'Badge three', color: Colors.red),
            ],
            stats: const [
              ('Following', '1,234,567'),
              ('Followers', '2,345,678'),
              ('Topics', '3,456,789'),
              ('Replies', '4,567,890'),
              ('Likes', '5,678,901'),
            ],
            statActions: {0: () {}},
          ),
        ),
      ),
    );

    final badgeRow = tester.widget<Wrap>(
      find.descendant(
        of: find.byKey(const ValueKey('profile-badges-row')),
        matching: find.byType(Wrap),
      ),
    );
    expect(badgeRow.alignment, WrapAlignment.start);
    expect(badgeRow.spacing, 2);
    final metadataBounds = tester.getRect(
      find.byKey(const ValueKey('profile-test-meta-row')),
    );
    final badgeBounds = tester.getRect(
      find.byKey(const ValueKey('profile-badges-row')),
    );
    final statsBounds = tester.getRect(
      find.byKey(const ValueKey('profile-stats-row')),
    );
    expect(badgeBounds.top - metadataBounds.bottom, closeTo(3, .01));
    expect(statsBounds.top - badgeBounds.bottom, closeTo(3, .01));
    final medallions = tester
        .widgetList<GfBadgeMedallion>(find.byType(GfBadgeMedallion))
        .toList();
    expect(medallions.map((badge) => badge.size), everyElement(34));
    final badgeRects = List.generate(
      medallions.length,
      (index) => tester.getRect(find.byType(GfBadgeMedallion).at(index)),
    );
    expect(badgeRects.first.left, lessThan(badgeRects.last.left));

    final statsRow = find.byKey(const ValueKey('profile-stats-row'));
    expect(
      tester.widget<SingleChildScrollView>(statsRow).scrollDirection,
      Axis.horizontal,
    );
    for (final value in [
      '1,234,567',
      '2,345,678',
      '3,456,789',
      '4,567,890',
      '5,678,901',
    ]) {
      final text = tester.widget<Text>(find.text(value));
      expect(text.maxLines, 1);
      expect(text.softWrap, isFalse);
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('preview badge rows stay single-line with a scroll cue', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(240, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      gfApp(
        SingleChildScrollView(
          child: GfUserCard(
            avatarUrl: '',
            name: 'A very long profile display name',
            username: 'a_very_long_account_name',
            compact: true,
            nameBadges: const [
              GfBadge(label: 'Admin', variant: GfBadgeVariant.warning),
              GfBadge(label: 'Online', variant: GfBadgeVariant.success),
            ],
            coloredBadges: const [
              GfUserBadge(label: 'One', color: Colors.blue),
              GfUserBadge(label: 'Two', color: Colors.green),
              GfUserBadge(label: 'Three', color: Colors.red),
              GfUserBadge(label: 'Four', color: Colors.orange),
              GfUserBadge(label: 'Five', color: Colors.purple),
            ],
          ),
        ),
      ),
    );

    final nameRow = find.byKey(const ValueKey('profile-name-row'));
    expect(tester.widget<Row>(nameRow), isA<Row>());
    final name = tester.widget<Text>(
      find.descendant(of: nameRow, matching: find.byType(Text)).first,
    );
    expect(name.maxLines, 1);
    expect(name.overflow, TextOverflow.ellipsis);

    final badgeScroll = find.byKey(const ValueKey('profile-badges-scroll'));
    expect(
      tester.widget<SingleChildScrollView>(badgeScroll).scrollDirection,
      Axis.horizontal,
    );
    expect(
      find.descendant(
        of: find.byKey(const ValueKey('profile-badges-row')),
        matching: find.byType(Wrap),
      ),
      findsNothing,
    );
    expect(tester.takeException(), isNull);
  });

  for (final brightness in Brightness.values) {
    for (final scale in [1.0, 2.0]) {
      testWidgets(
        'profile identity is compact without overlapping avatar or actions ($brightness, $scale)',
        (tester) async {
          tester.view.physicalSize = const Size(320, 1000);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.reset);
          await tester.pumpWidget(
            gfApp(
              MediaQuery(
                data: MediaQueryData(textScaler: TextScaler.linear(scale)),
                child: SingleChildScrollView(
                  child: GfUserCard(
                    avatarUrl: '',
                    name: 'Alice',
                    username: 'alice',
                    bio: 'Campus life',
                    signature: 'Stay curious',
                    actions: OutlinedButton(
                      onPressed: () {},
                      style: OutlinedButton.styleFrom(
                        minimumSize: const Size(96, 44),
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 8,
                        ),
                        textStyle: const TextStyle(
                          fontSize: 14,
                          height: 1.4,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      child: const Text('Edit profile'),
                    ),
                  ),
                ),
              ),
              brightness: brightness,
            ),
          );
          final cover = tester.getRect(
            find.byKey(const Key('profile-cover-image')),
          );
          final name = tester.getRect(find.text('Alice'));
          final username = tester.getRect(find.text('@alice'));
          final bio = tester.getRect(find.text('Campus life'));
          final avatar = tester.getRect(find.byType(GfAvatar));
          final action = tester.getRect(find.byType(OutlinedButton));
          expect(cover.height, GfUserCard.coverHeightFor(320));
          // GfAvatar is inside a 4px outer ring on every side.
          expect(name.top, greaterThanOrEqualTo(avatar.bottom + 4 + 8));
          // The test font can wrap "Edit profile" even at 1x. Natural action
          // height must win over the 56px minimum instead of clipping the label.
          expect(
            name.top - cover.bottom,
            closeTo((action.height + 8).clamp(56, double.infinity), .01),
          );
          expect(username.top - name.bottom, closeTo(2, .01));
          expect(bio.top - username.bottom, closeTo(8, .01));
          expect(action.height, greaterThanOrEqualTo(44));
          expect(name.top, greaterThanOrEqualTo(action.bottom + 4));
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
}

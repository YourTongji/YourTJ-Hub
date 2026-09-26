import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

class _Storage implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

Map<String, dynamic> _badge(int index) => {
  'code': 'badge-$index',
  'type': 'achievement',
  'grantMode': 'auto',
  'name': index == 0 ? 'First post' : 'Badge $index',
  'description': 'Shared a first post with the community.',
  'iconType': 'asset',
  'iconKey': '',
  'iconUrl': '',
  'color': 'blue',
  'level': 'bronze',
  'isEnabled': true,
  'isWearable': true,
  'sortOrder': index,
  'source': 'auto',
  'reason': 'Thanks for joining the discussion.',
  'grantedAt': '2026-09-20T08:00:00Z',
};

class _Badges extends PageRepository {
  _Badges()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Storage(),
          baseUrl: 'http://fake.local',
        ),
      );

  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async {
    final json = userProfilePayloadJson();
    final props = json['props'] as Map<String, dynamic>;
    final user = props['user'] as Map<String, dynamic>;
    user['isAdmin'] = true;
    user['badges'] = List.generate(5, _badge);
    user['displayBadges'] = List.generate(5, _badge);
    props['badges'] = List.generate(5, _badge);
    return parsePayload(json);
  }
}

Future<void> _pump(
  WidgetTester tester, {
  String stream = 'timeline',
  double scale = 1,
  Brightness brightness = Brightness.light,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        currentUserProvider.overrideWith((_) async => null),
        pageRepositoryProvider.overrideWithValue(_Badges()),
      ],
      child: MaterialApp(
        theme: gfThemeData(brightness),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('en'),
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
        home: ProfilePage(userId: 1, initialStream: stream),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  for (final brightness in Brightness.values) {
    testWidgets(
      'role chip and earned badge buttons are distinct ($brightness)',
      (tester) async {
        tester.view.physicalSize = const Size(390, 900);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final semantics = tester.ensureSemantics();
        try {
          await _pump(tester, brightness: brightness);
          final card = tester.widget<GfUserCard>(find.byType(GfUserCard));
          expect(card.coloredBadges, hasLength(5));
          expect(find.text('Admin'), findsOneWidget);
          expect(find.text('First post'), findsNothing);
          final target = find.byTooltip(
            'First post\nShared a first post with the community.',
          );
          await tester.ensureVisible(target);
          await tester.pumpAndSettle();
          expect(tester.getSize(target).shortestSide, greaterThanOrEqualTo(44));
          expect(
            tester
                .getSemantics(find.bySemanticsLabel('First post'))
                .flagsCollection
                .isButton,
            isTrue,
          );
          await tester.tap(target);
          await tester.pumpAndSettle();
          expect(find.text('First post'), findsOneWidget);
          expect(
            find.text('Shared a first post with the community.'),
            findsOneWidget,
          );
          expect(
            find.text('Thanks for joining the discussion.'),
            findsOneWidget,
          );
          expect(find.textContaining('Earned on'), findsOneWidget);
          await tester.tap(find.byTooltip('Close'));
          await tester.pumpAndSettle();
          expect(find.text('First post'), findsNothing);
          expect(tester.takeException(), isNull);
        } finally {
          semantics.dispose();
        }
      },
    );
  }

  for (final scale in [1.0, 2.0]) {
    testWidgets('badge gallery adapts columns at text scale $scale', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await _pump(tester, stream: 'badges', scale: scale);
      await tester.scrollUntilVisible(
        find.byWidgetPredicate(
          (widget) =>
              widget is GfAchievementCard && widget.title == 'First post',
        ),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.pumpAndSettle();
      final card = find.byType(GfAchievementCard).first;
      final width = tester.getSize(card).width;
      expect(width, closeTo(scale == 1 ? 173 : 358, 1));
      await tester.tap(card);
      await tester.pumpAndSettle();
      expect(find.text('Thanks for joining the discussion.'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}

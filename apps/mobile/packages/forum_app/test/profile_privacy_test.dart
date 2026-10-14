// issue #1124: profile privacy in the app. The owner, other members and
// guests each get their own copy; hidden sections never render rows; a
// setting only shows as saved after the server confirms it.
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/pages/settings/profile_privacy_page.dart';
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

GfApiClient _client() => GfApiClient(
  dio: Dio(),
  tokenStorage: _Storage(),
  baseUrl: 'http://fake.local',
);

const _activityRow = '发布了话题 移动端测试话题';

class _Profile extends PageRepository {
  _Profile({required this.own, this.visibility}) : super(_client());
  final bool own;
  Map<String, dynamic>? visibility;
  int fetches = 0;

  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async {
    fetches++;
    final json = own ? userProfilePayloadJson() : peerProfilePayloadJson();
    final props = json['props'] as Map<String, dynamic>;
    if (visibility != null) props['visibility'] = visibility;
    return parsePayload(json);
  }
}

class _Follows extends TopicRepository {
  _Follows() : super(_client());
  final calls = <bool>[];

  @override
  Future<bool> followUser({
    required int userId,
    required bool isFollowing,
  }) async {
    calls.add(isFollowing);
    return true;
  }
}

class _Privacy extends UserRepository {
  _Privacy() : super(_client());
  Map<String, String> audiences = {
    for (final key in profilePrivacyAudiences.keys) key: 'everyone',
  };
  final updates = <(String, String)>[];
  bool failLoad = false;
  bool failSave = false;

  @override
  Future<UserPrivacySettingsPayload> getPrivacySettings() async {
    if (failLoad) {
      failLoad = false;
      throw StateError('offline');
    }
    return UserPrivacySettingsPayload(Map.of(audiences));
  }

  @override
  Future<UserPrivacySettingsPayload> updatePrivacySetting(
    String section,
    String audience,
  ) async {
    updates.add((section, audience));
    if (failSave) throw StateError('offline');
    audiences[section] = audience;
    return UserPrivacySettingsPayload(Map.of(audiences));
  }
}

Map<String, dynamic> _visibility(Map<String, Object> activity) => {
  'profileTopics': {'visible': true},
  'profileActivity': activity,
  'profileLikes': {'visible': true},
  'followingList': {'visible': true},
  'followersList': {'visible': true},
};

Widget _app(List<Override> overrides, Widget home) => ProviderScope(
  overrides: overrides,
  child: MaterialApp(
    theme: gfThemeData(Brightness.light),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    locale: const Locale('en'),
    home: home,
  ),
);

Future<void> _profile(
  WidgetTester tester, {
  required bool own,
  bool signedIn = false,
  Map<String, dynamic>? visibility,
}) async {
  tester.view.physicalSize = const Size(390, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    _app([
      currentUserProvider.overrideWith(
        (_) async => signedIn ? const CurrentUser(id: 9, username: 'c') : null,
      ),
      pageRepositoryProvider.overrideWithValue(
        _Profile(own: own, visibility: visibility),
      ),
    ], ProfilePage(userId: own ? 1 : 2)),
  );
  await tester.pumpAndSettle();
}

Future<_Privacy> _settings(WidgetTester tester, {_Privacy? repo}) async {
  final privacy = repo ?? _Privacy();
  await tester.pumpWidget(
    _app([
      currentUserProvider.overrideWith(
        (_) async => const CurrentUser(id: 1, username: 'alice'),
      ),
      userRepositoryProvider.overrideWithValue(privacy),
    ], const ProfilePrivacyPage()),
  );
  await tester.pumpAndSettle();
  return privacy;
}

String _rowValue(WidgetTester tester, String key) {
  final row = find.byKey(ValueKey('profile-privacy-row-$key'));
  return tester
      .widgetList<Text>(find.descendant(of: row, matching: find.byType(Text)))
      .last
      .data!;
}

Future<void> _pick(WidgetTester tester, String key, String audience) async {
  await tester.tap(find.byKey(ValueKey('profile-privacy-row-$key')));
  await tester.pumpAndSettle();
  await tester.tap(find.byKey(ValueKey('profile-privacy-option-$audience')));
  await tester.pumpAndSettle();
}

void main() {
  group('profile page', () {
    testWidgets('guests learn why and can sign in when that helps', (
      tester,
    ) async {
      await _profile(
        tester,
        own: false,
        visibility: _visibility({'visible': false, 'reason': 'signin'}),
      );
      expect(find.text('This section isn’t public'), findsOneWidget);
      expect(
        find.text('Bob shows this only to signed-in members.'),
        findsOneWidget,
      );
      expect(find.byKey(const ValueKey('profile-section-sign-in')), findsOne);
      expect(find.text(_activityRow), findsNothing);
      // The tab is marked before it is opened; other tabs stay plain.
      expect(find.byKey(const ValueKey('profile-tab-lock-timeline')), findsOne);
      expect(
        find.byKey(const ValueKey('profile-tab-lock-topics')),
        findsNothing,
      );
      expect(
        find.byKey(const ValueKey('profile-tab-lock-bookmarks')),
        findsNothing,
      );
      expect(tester.takeException(), isNull);
    });

    testWidgets('guests get no sign-in for mutual or private sections', (
      tester,
    ) async {
      for (final reason in ['mutual', 'private']) {
        // A fresh page per reason; the mounted one keeps its loaded payload.
        await tester.pumpWidget(const SizedBox());
        await _profile(
          tester,
          own: false,
          visibility: _visibility({'visible': false, 'reason': reason}),
        );
        expect(find.byKey(const ValueKey('profile-section-hidden')), findsOne);
        expect(
          find.byKey(const ValueKey('profile-section-sign-in')),
          findsNothing,
          reason: reason,
        );
      }
      expect(
        find.text('Bob chose not to show this on their profile.'),
        findsOne,
      );
    });

    testWidgets('signed-in members are told to follow, not to sign in', (
      tester,
    ) async {
      await _profile(
        tester,
        own: false,
        signedIn: true,
        visibility: _visibility({'visible': false, 'reason': 'followers'}),
      );
      expect(
        find.text('Bob shows this only to followers. Follow to see it.'),
        findsOneWidget,
      );
      expect(
        find.byKey(const ValueKey('profile-section-sign-in')),
        findsNothing,
      );
      expect(find.text(_activityRow), findsNothing);
    });

    testWidgets('following reveals a followers-only section in place', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final profile = _Profile(
        own: false,
        visibility: _visibility({'visible': false, 'reason': 'followers'}),
      );
      final follows = _Follows();
      await tester.pumpWidget(
        _app([
          currentUserProvider.overrideWith(
            (_) async => const CurrentUser(id: 9, username: 'c'),
          ),
          pageRepositoryProvider.overrideWithValue(profile),
          topicRepositoryProvider.overrideWithValue(follows),
        ], const ProfilePage(userId: 2)),
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('profile-section-hidden')), findsOne);

      // The server now counts the viewer as a follower.
      profile.visibility = _visibility({'visible': true});
      final reads = profile.fetches;
      await tester.tap(find.text('Follow'));
      await tester.pumpAndSettle();
      expect(follows.calls, [false]);
      expect(profile.fetches, greaterThan(reads));
      expect(
        find.byKey(const ValueKey('profile-section-hidden')),
        findsNothing,
      );
      expect(find.text(_activityRow), findsOneWidget);
      expect(
        find.byKey(const ValueKey('profile-tab-lock-timeline')),
        findsNothing,
      );
    });

    testWidgets('the owner keeps the rows; the tab lock says who can see', (
      tester,
    ) async {
      final semantics = tester.ensureSemantics();
      await _profile(
        tester,
        own: true,
        signedIn: true,
        visibility: _visibility({'visible': true, 'audience': 'self'}),
      );
      expect(find.text(_activityRow), findsOneWidget);
      expect(
        find.byKey(const ValueKey('profile-section-hidden')),
        findsNothing,
      );
      // No banner above the list: the audience lives on the tab.
      expect(find.text('Only you can see this.'), findsNothing);
      final lock = find.byKey(const ValueKey('profile-tab-lock-timeline'));
      expect(lock, findsOne);
      expect(
        find.bySemanticsLabel(RegExp(r'\nOnly you can see this\.$')),
        findsNWidgets(2), // activity and the always-private bookmarks
      );

      await tester.longPress(lock);
      await tester.pumpAndSettle();
      expect(find.textContaining('Only you can see this.'), findsOne);
      await tester.pump(const Duration(seconds: 3));
      await tester.pumpAndSettle();
      semantics.dispose();
    });

    testWidgets('a public section carries no lock for its owner', (
      tester,
    ) async {
      await _profile(
        tester,
        own: true,
        signedIn: true,
        visibility: _visibility({'visible': true, 'audience': 'everyone'}),
      );
      expect(
        find.byKey(const ValueKey('profile-tab-lock-timeline')),
        findsNothing,
      );
    });

    testWidgets('profiles from older servers stay public without notes', (
      tester,
    ) async {
      await _profile(tester, own: true, signedIn: true);
      expect(find.text(_activityRow), findsOneWidget);
      expect(
        find.byKey(const ValueKey('profile-tab-lock-timeline')),
        findsNothing,
      );
      // Bookmarks were always owner-only, so they stay marked either way.
      expect(
        find.byKey(const ValueKey('profile-tab-lock-bookmarks')),
        findsOne,
      );
    });
  });

  group('settings page', () {
    testWidgets('bookmarks show as fixed to the owner and cannot be picked', (
      tester,
    ) async {
      final repo = await _settings(tester);
      final row = find.byKey(const ValueKey('profile-privacy-row-bookmarks'));
      expect(_rowValue(tester, 'bookmarks'), 'Only you');
      expect(
        find.descendant(
          of: row,
          matching: find.text(
            'Posts you bookmarked. Always visible only to you',
          ),
        ),
        findsOne,
      );
      await tester.tap(row);
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('profile-privacy-option-self')),
        findsNothing,
      );
      expect(repo.updates, isEmpty);
    });

    testWidgets('narrowing saves at once and offers only allowed audiences', (
      tester,
    ) async {
      final repo = await _settings(tester);
      expect(_rowValue(tester, 'profileLikes'), 'Everyone');
      await tester.tap(
        find.byKey(const ValueKey('profile-privacy-row-profileLikes')),
      );
      await tester.pumpAndSettle();
      expect(find.text('Who can see Liked posts'), findsOneWidget);
      expect(
        find.byKey(const ValueKey('profile-privacy-option-members')),
        findsNothing,
      );
      expect(
        find.byKey(const ValueKey('profile-privacy-option-mutual')),
        findsNothing,
      );
      await tester.tap(
        find.byKey(const ValueKey('profile-privacy-option-self')),
      );
      await tester.pumpAndSettle();
      expect(repo.updates, [('profileLikes', 'self')]);
      expect(_rowValue(tester, 'profileLikes'), 'Only you');
      expect(find.text('Visibility saved to your account'), findsOneWidget);
      await tester.pump(const Duration(seconds: 5));
      await tester.pumpAndSettle();
    });

    testWidgets('a failed save keeps the confirmed value', (tester) async {
      final repo = _Privacy()..failSave = true;
      await _settings(tester, repo: repo);
      await _pick(tester, 'profileTopics', 'followers');
      expect(repo.updates, [('profileTopics', 'followers')]);
      expect(_rowValue(tester, 'profileTopics'), 'Everyone');
      expect(find.textContaining('Unable to save.'), findsOneWidget);
      await tester.pump(const Duration(seconds: 5));
      await tester.pumpAndSettle();
    });

    testWidgets('widening asks first; cancel keeps the setting', (
      tester,
    ) async {
      final repo = _Privacy()..audiences['followingList'] = 'self';
      await _settings(tester, repo: repo);
      await _pick(tester, 'followingList', 'everyone');
      final dialog = find.byKey(const ValueKey('profile-privacy-widen'));
      expect(dialog, findsOneWidget);
      expect(
        find.textContaining('“Following list” will be visible to “Everyone”'),
        findsOneWidget,
      );
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(repo.updates, isEmpty);
      expect(_rowValue(tester, 'followingList'), 'Only you');

      await _pick(tester, 'followingList', 'everyone');
      await tester.tap(find.text('Show to more people'));
      await tester.pumpAndSettle();
      expect(repo.updates, [('followingList', 'everyone')]);
      expect(_rowValue(tester, 'followingList'), 'Everyone');
      await tester.pump(const Duration(seconds: 5));
      await tester.pumpAndSettle();
    });

    testWidgets('a load failure offers retry', (tester) async {
      await _settings(tester, repo: _Privacy()..failLoad = true);
      expect(find.text('Unable to load profile visibility.'), findsOneWidget);
      await tester.tap(find.text('Retry'));
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('profile-privacy-row-profileTopics')),
        findsOneWidget,
      );
    });

    testWidgets('large text keeps every row usable', (tester) async {
      tester.view.physicalSize = const Size(320, 640);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            currentUserProvider.overrideWith(
              (_) async => const CurrentUser(id: 1, username: 'alice'),
            ),
            userRepositoryProvider.overrideWithValue(_Privacy()),
          ],
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('de'),
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(textScaler: const TextScaler.linear(2)),
              child: child!,
            ),
            home: const ProfilePrivacyPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      final row = find.byKey(
        const ValueKey('profile-privacy-row-followersList'),
      );
      await tester.scrollUntilVisible(row, 200);
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
    });
  });
}

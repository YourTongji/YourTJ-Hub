import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/pages/profile/anonymous_profile_page.dart';
import 'package:forum_app/src/pages/profile/profile_header_sliver.dart';
import 'package:forum_app/src/pages/profile/profile_edit_button.dart';
import 'package:forum_app/src/pages/profile/profile_tabs.dart';
import 'package:forum_app/src/pages/settings/anonymous_identity_page.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

final uid = 'a' * 32;
final persona = AnonymousPersona(
  publicUid: uid,
  name: '躲进云里的猫',
  avatarUrl: '',
  profileUrl: '/a/$uid',
);
Map<String, dynamic> props(String name) => {
  'persona': {
    'kind': 'persona',
    'publicUid': uid,
    'name': name,
    'avatarUrl': '',
    'profileUrl': '/a/$uid',
  },
  'topics': [],
  'replies': [
    {'id': 1, 'url': '/p/1/2', 'excerpt': '这是匿名回复'},
  ],
  'topicCount': 0,
  'replyCount': 1,
  'page': 1,
  'hasNext': false,
};
AnonymousIdentityState state(AnonymousPersona? p) => AnonymousIdentityState(
  persona: p,
  day: '2026-10-07',
  remaining: 10,
  resetsAt: DateTime(2026, 10, 8),
  availableAt: DateTime(2027, 10, 7),
  nameSelectedAt: DateTime(2026, 10, 7),
  lexiconVersion: 'phrase6-v1',
  disabled: false,
  governanceDisabled: false,
  batches: [],
);
void main() {
  for (final brightness in Brightness.values) {
    for (final width in [320.0, 768.0]) {
      testWidgets(
        'shared persona profile and management fit $brightness $width at 2x text',
        (tester) async {
          tester.view.physicalSize = Size(width, 1100);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.resetPhysicalSize);
          addTearDown(tester.view.resetDevicePixelRatio);
          final dio = Dio()
            ..interceptors.add(
              InterceptorsWrapper(
                onRequest: (o, h) => h.resolve(
                  Response(
                    requestOptions: o,
                    statusCode: 200,
                    data: {'props': props(persona.name)},
                  ),
                ),
              ),
            );
          await tester.pumpWidget(
            ProviderScope(
              overrides: [
                apiClientProvider.overrideWithValue(
                  GfApiClient(
                    dio: dio,
                    tokenStorage: MemoryTokenStorage(),
                    baseUrl: 'http://fake.local',
                  ),
                ),
                currentUserProvider.overrideWith(
                  (ref) async => const CurrentUser(id: 1, username: 'owner'),
                ),
                anonymousIdentityProvider.overrideWith(
                  (ref) async => state(persona),
                ),
              ],
              child: MaterialApp(
                locale: const Locale('zh'),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                theme: gfThemeData(brightness),
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(
                    context,
                  ).copyWith(textScaler: const TextScaler.linear(2)),
                  child: child!,
                ),
                home: AnonymousProfilePage(uid: uid),
              ),
            ),
          );
          await tester.pumpAndSettle();
          expect(find.byType(ProfileHeaderSliver), findsOneWidget);
          expect(find.byType(ProfileEditButton), findsOneWidget);
          expect(find.byType(ProfileTabs), findsOneWidget);
          expect(find.text('@'), findsNothing);
          final button = tester.getRect(find.byType(ProfileEditButton));
          final avatar = tester.getRect(find.byType(GfAvatar).first);
          expect(button.left, greaterThanOrEqualTo(avatar.right));
          expect(button.right, lessThanOrEqualTo(width));
          expect(tester.takeException(), isNull);
          await tester.tap(find.byTooltip('回复'));
          await tester.pumpAndSettle();
          await tester.drag(
            find.byType(CustomScrollView),
            const Offset(0, -400),
          );
          await tester.pumpAndSettle();
          expect(find.text('这是匿名回复'), findsOneWidget);
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
  for (final signedIn in [false, true]) {
    testWidgets('visitor has no management control (signedIn=$signedIn)', (
      tester,
    ) async {
      var privateReads = 0;
      final dio = Dio()
        ..interceptors.add(
          InterceptorsWrapper(
            onRequest: (o, h) => h.resolve(
              Response(
                requestOptions: o,
                statusCode: 200,
                data: {'props': props(persona.name)},
              ),
            ),
          ),
        );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            apiClientProvider.overrideWithValue(
              GfApiClient(
                dio: dio,
                tokenStorage: MemoryTokenStorage(),
                baseUrl: 'http://fake.local',
              ),
            ),
            currentUserProvider.overrideWith(
              (ref) async => signedIn
                  ? const CurrentUser(id: 2, username: 'visitor')
                  : null,
            ),
            anonymousIdentityProvider.overrideWith((ref) async {
              privateReads++;
              return state(
                AnonymousPersona(
                  publicUid: 'b' * 32,
                  name: '抱着松果的熊',
                  avatarUrl: '',
                  profileUrl: '/a/${'b' * 32}',
                ),
              );
            }),
          ],
          child: MaterialApp(
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            theme: gfThemeData(Brightness.light),
            home: AnonymousProfilePage(uid: uid),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(ProfileEditButton), findsNothing);
      expect(privateReads, signedIn ? 1 : 0);
      expect(find.text('抱着松果的熊'), findsNothing);
    });
  }
  testWidgets('session change rejects an older public profile response', (
    tester,
  ) async {
    final delayed = Completer<void>();
    var reads = 0;
    final dio = Dio()
      ..interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) async {
            final first = reads++ == 0;
            if (first) await delayed.future;
            h.resolve(
              Response(
                requestOptions: o,
                statusCode: 200,
                data: {'props': props(first ? '过期响应' : '当前响应')},
              ),
            );
          },
        ),
      );
    late ProviderContainer container;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          apiClientProvider.overrideWithValue(
            GfApiClient(
              dio: dio,
              tokenStorage: MemoryTokenStorage(),
              baseUrl: 'http://fake.local',
            ),
          ),
          currentUserProvider.overrideWith((ref) async => null),
        ],
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          theme: gfThemeData(Brightness.light),
          home: Builder(
            builder: (context) {
              container = ProviderScope.containerOf(context);
              return AnonymousProfilePage(uid: uid);
            },
          ),
        ),
      ),
    );
    await tester.pump();
    await tester.pump();
    container.read(offlineCacheEpochProvider.notifier).state++;
    await tester.pump();
    await tester.pumpAndSettle();
    delayed.complete();
    await tester.pumpAndSettle();
    expect(find.text('过期响应'), findsNothing);
    expect(find.text('当前响应'), findsWidgets);
  });
}

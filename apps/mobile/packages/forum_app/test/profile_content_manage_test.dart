import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:go_router/go_router.dart';
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

class _Profiles extends PageRepository {
  _Profiles({required this.own}) : super(_client());
  final bool own;
  final paths = <String>[];

  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async {
    paths.add(path);
    return parsePayload(
      own ? userProfilePayloadJson() : peerProfilePayloadJson(),
    );
  }
}

class _Content extends ContentRepository {
  _Content() : super(_client());
  final listTypes = <String>[];

  @override
  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) async {
    listTypes.add(contentType);
    return const UserContentPage(items: [], hasMore: false, nextCursorId: 0);
  }
}

/// A profile tab by the first line of its tooltip (locked tabs add a reason).
Finder _tab(String label) => find.byWidgetPredicate(
  (w) => w is Tooltip && w.message?.split('\n').first == label,
);

Future<void> _select(WidgetTester tester, String label) async {
  tester
      .widget<InkWell>(
        find.descendant(of: _tab(label), matching: find.byType(InkWell)).first,
      )
      .onTap!();
  await tester.pumpAndSettle();
}

Future<(GoRouter, _Profiles, _Content)> _pump(
  WidgetTester tester, {
  required bool own,
}) async {
  final profiles = _Profiles(own: own);
  final content = _Content();
  final router = GoRouter(
    initialLocation: own ? '/u/1' : '/u/2',
    routes: appRouter.configuration.routes,
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        currentUserProvider.overrideWith((ref) async => null),
        pageRepositoryProvider.overrideWithValue(profiles),
        contentRepositoryProvider.overrideWithValue(content),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('zh'),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return (router, profiles, content);
}

Finder get _manage => find.widgetWithText(GfButton, '管理');

void main() {
  testWidgets('own topics stream opens content management on topics', (
    tester,
  ) async {
    final (router, profiles, content) = await _pump(tester, own: true);
    // The mixed activity stream has no single management filter.
    expect(_manage, findsNothing);

    await _select(tester, '内容');
    expect(_manage, findsOneWidget);

    await tester.tap(_manage);
    await tester.pumpAndSettle();
    expect(router.state.uri.toString(), '/my-content?type=topic');
    expect(content.listTypes, ['topic']);
    expect(
      tester
          .widget<GfSegmented<String>>(find.byType(GfSegmented<String>))
          .selected,
      'topic',
    );

    // Returning reloads the stream so deletions there show up here.
    final reads = profiles.paths.where((p) => p == '/u/1/activity/topics');
    expect(reads, hasLength(1));
    router.pop();
    await tester.pumpAndSettle();
    expect(_manage, findsOneWidget);
    expect(reads, hasLength(2));
  });

  testWidgets('other profiles offer no content management', (tester) async {
    await _pump(tester, own: false);
    await _select(tester, '内容');
    expect(_manage, findsNothing);
  });
}

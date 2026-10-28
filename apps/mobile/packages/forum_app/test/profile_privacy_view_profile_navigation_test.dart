// issue #1400: "View my profile" inside profile privacy pushes the profile on
// top of settings, so back returns to the privacy page instead of exiting.
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
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Pages extends PageRepository {
  _Pages(super.client);

  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async =>
      parsePayload(userProfilePayloadJson());
}

class _Privacy extends UserRepository {
  _Privacy(super.client);

  @override
  Future<UserPrivacySettingsPayload> getPrivacySettings() async =>
      UserPrivacySettingsPayload({
        for (final key in profilePrivacyAudiences.keys) key: 'everyone',
      });
}

void main() {
  testWidgets('view my profile pushes over settings and pops back', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    final storage = MemoryTokenStorage();
    await storage.write('session');
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final router = GoRouter(
      initialLocation: '/settings/profile-privacy',
      routes: appRouter.configuration.routes,
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          currentUserProvider.overrideWith(
            (_) async => const CurrentUser(id: 1, username: 'alice'),
          ),
          pageRepositoryProvider.overrideWithValue(_Pages(client)),
          userRepositoryProvider.overrideWithValue(_Privacy(client)),
        ],
        child: MaterialApp.router(
          routerConfig: router,
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(const ValueKey('profile-privacy-row-bookmarks')),
      findsOneWidget,
    );

    final button = find.byKey(const ValueKey('profile-privacy-view-profile'));
    await tester.ensureVisible(button);
    await tester.pumpAndSettle();
    await tester.tap(button);
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/profile');
    expect(router.canPop(), isTrue);
    expect(find.byTooltip('Back'), findsWidgets);

    expect(await tester.binding.handlePopRoute(), isTrue);
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/settings/profile-privacy');
    expect(
      find.byKey(const ValueKey('profile-privacy-view-profile')),
      findsOne,
    );
    expect(tester.takeException(), isNull);
  });
}

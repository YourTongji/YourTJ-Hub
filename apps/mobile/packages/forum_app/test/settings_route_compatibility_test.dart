import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/navigation/auth_navigation.dart';
import 'package:forum_app/src/pages/settings/badge_display_dialog.dart';
import 'package:forum_app/src/pages/settings/profile_edit_dialog.dart';
import 'package:forum_app/src/pages/settings/storage_settings_panel.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/storage/storage_providers.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_behavior_test.dart' show settingsPayloadJson;
import 'pages_smoke_test.dart' show MemoryTokenStorage;
import 'storage_settings_test.dart' show FakeDeviceStorage;

class _Settings extends PageRepository {
  _Settings(super.client);
  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async =>
      PagePayload.fromJson(settingsPayloadJson());
}

class _Sessions extends UserRepository {
  _Sessions(super.client);
  @override
  Future<List<UserSessionPayload>> listSessions() async => [];
}

void main() {
  for (final path in [
    '/settings/profile',
    '/settings/profile?edit=1',
    '/settings/account',
    '/settings/security',
    '/settings/binding',
    '/settings/privacy',
    '/settings/data',
    '/settings/privacy-controls',
    '/settings/notifications',
    '/settings/badges',
  ]) {
    testWidgets('production settings route preserves $path', (tester) async {
      final storage = MemoryTokenStorage();
      await storage.write('session');
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: storage,
        baseUrl: 'http://fake.local',
      );
      final router = GoRouter(
        initialLocation: path,
        routes: appRouter.configuration.routes,
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            tokenStorageProvider.overrideWithValue(storage),
            pageRepositoryProvider.overrideWithValue(_Settings(client)),
            userRepositoryProvider.overrideWithValue(_Sessions(client)),
            deviceStorageProvider.overrideWithValue(FakeDeviceStorage()),
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
      if (path.startsWith('/settings/profile') && !path.contains('privacy')) {
        expect(find.byType(ProfileEditPage), findsOneWidget);
      } else if (path == '/settings/privacy' || path == '/settings/data') {
        expect(find.byType(StorageSettingsPanel), findsOneWidget);
      } else if (path == '/settings/badges') {
        expect(find.byType(BadgeDisplayDialog), findsOneWidget);
      } else if (path == '/settings/privacy-controls') {
        expect(find.text('Profile visibility'), findsWidgets);
      } else if (path == '/settings/notifications') {
        expect(find.byType(GfSwitchRow), findsWidgets);
      } else {
        expect(find.text('Account & security'), findsWidgets);
      }
      expect(safeAuthReturnTo(path), path);
      if (path == '/settings/profile' || path == '/settings/badges') {
        await tester.tap(
          find.byTooltip(path == '/settings/profile' ? 'Cancel' : 'Back'),
        );
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/settings');
      }
      expect(tester.takeException(), isNull);
    });
  }
}

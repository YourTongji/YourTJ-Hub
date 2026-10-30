import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';

import 'pages_behavior_test.dart'
    show
        CountingPageRepository,
        MemTokenStorage,
        NoopCache,
        PollingChatRepository;

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  for (final noticeHeight in [0.0, 104.0]) {
    testWidgets(
      'chat input clears keyboard in full shell, notice $noticeHeight',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = const Size(466, 678);
        tester.view.padding = const FakeViewPadding(right: 84, bottom: 34);
        tester.view.viewPadding = const FakeViewPadding(right: 84, bottom: 34);
        addTearDown(tester.view.reset);
        final storage = MemTokenStorage();
        final client = GfApiClient(
          dio: Dio(),
          tokenStorage: storage,
          baseUrl: 'http://fake.local',
        );
        final container = ProviderContainer(
          overrides: [
            tokenStorageProvider.overrideWithValue(storage),
            currentUserProvider.overrideWith(
              (_) async => const CurrentUser(id: 1, username: 'alice'),
            ),
            pageRepositoryProvider.overrideWithValue(
              CountingPageRepository(client),
            ),
            chatRepositoryProvider.overrideWithValue(
              PollingChatRepository(client, messages: []),
            ),
            offlineTopicCacheProvider.overrideWithValue(NoopCache()),
            offlineChatCacheProvider.overrideWithValue(NoopCache()),
          ],
        );
        addTearDown(container.dispose);
        final router = GoRouter(
          initialLocation: '/messages',
          routes: [
            StatefulShellRoute.indexedStack(
              builder: (_, _, shell) => GfShell(navigationShell: shell),
              branches: [
                for (final path in [
                  '/',
                  '/campus',
                  '/notifications',
                  '/messages',
                ])
                  StatefulShellBranch(
                    routes: [
                      GoRoute(
                        path: path,
                        builder: (_, _) => path == '/messages'
                            ? const MessagesPage(
                                targetUserId: 2,
                                targetUsername: 'bob',
                              )
                            : const SizedBox.shrink(),
                      ),
                    ],
                  ),
              ],
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              theme: gfThemeData(Brightness.light),
              locale: const Locale('zh'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (_, child) => Column(
                children: [
                  SizedBox(height: noticeHeight),
                  Expanded(child: child!),
                ],
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final field = find.descendant(
          of: find.byType(GfChatInput),
          matching: find.byType(TextField),
        );
        await tester.enterText(field, 'unsent work survives window changes');
        final input = tester.widget<TextField>(field).controller!;

        for (final size in [
          const Size(466, 678),
          const Size(951, 669),
          const Size(466, 678),
        ]) {
          tester.view.physicalSize = size;
          tester.view.padding = const FakeViewPadding(right: 84);
          tester.view.viewInsets = const FakeViewPadding(bottom: 300);
          await tester.pumpAndSettle();
          expect(tester.takeException(), isNull);
          final send = find.byKey(const Key('chat-send'));
          final sendRect = tester.getRect(send);
          expect(sendRect.bottom, lessThanOrEqualTo(size.height - 300));
          expect(send.hitTestable(), findsOneWidget);
          expect(field.hitTestable(), findsOneWidget);
          expect(tester.widget<TextField>(field).controller, same(input));
          expect(input.text, 'unsent work survives window changes');
        }
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
      },
    );
  }
}

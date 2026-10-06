import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/anonymous_moderation_dialog.dart';
import 'package:forum_app/src/providers.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  for (final canReveal in [false, true]) {
    testWidgets('anonymous governance separates reveal permission $canReveal', (
      tester,
    ) async {
      final pending = Completer<Map<String, dynamic>>();
      var requests = 0;
      final dio = Dio();
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) async {
            requests++;
            expect(options.path, endsWith('/reveal'));
            expect(options.data['reason'], '处理举报');
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: await pending.future,
              ),
            );
          },
        ),
      );
      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(
            GfApiClient(
              dio: dio,
              tokenStorage: MemoryTokenStorage(),
              baseUrl: 'http://fake.local',
            ),
          ),
        ],
      );
      addTearDown(container.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (context) => Scaffold(
                body: TextButton(
                  onPressed: () => showAnonymousModeration(
                    context,
                    postId: 1,
                    publicUid: 'a' * 32,
                    canReveal: canReveal,
                  ),
                  child: const Text('管理'),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('管理'));
      await tester.pumpAndSettle();
      expect(find.text('限制该账号发言'), findsOneWidget);
      expect(find.text('审计揭示身份'), canReveal ? findsOneWidget : findsNothing);
      if (!canReveal) return;
      final reveal = find.widgetWithText(TextButton, '审计揭示身份');
      expect(tester.widget<TextButton>(reveal).onPressed, isNull);
      await tester.enterText(find.byType(TextField), '处理举报');
      await tester.pump();
      await tester.tap(reveal);
      await tester.pumpAndSettle();
      expect(requests, 1);
      // Logout/account switch closes the private dialog even before the response arrives.
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      expect(find.byType(AlertDialog), findsNothing);
      pending.complete({
        'code': 0,
        'result': {
          'publicUid': 'a' * 32,
          'userId': 123,
          'username': 'private-owner',
        },
      });
      await tester.pumpAndSettle();
      expect(find.textContaining('private-owner'), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }
}

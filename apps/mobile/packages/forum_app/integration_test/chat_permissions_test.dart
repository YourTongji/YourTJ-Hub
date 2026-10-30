import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/settings/profile_privacy_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:integration_test/integration_test.dart';
import 'package:ui_kit/ui_kit.dart';

class _Storage implements TokenStorage {
  @override
  Future<String?> read() async => 'synthetic-session';
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

// Native UI + actual HTTP parsing, with synthetic owner-only loopback data.
// No remote account is accessed and no message is sent.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('message permissions save and recover in both themes', (
    tester,
  ) async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    addTearDown(() => server.close(force: true));
    var policy = 'everyone';
    var fail = false;
    final writes = <String>[];
    server.listen((request) async {
      expectSync(request.uri.path, '/api/user/privacy-settings');
      expectSync(
        request.headers.value(HttpHeaders.authorizationHeader),
        'Bearer synthetic-session',
      );
      request.response.headers.contentType = ContentType.json;
      if (request.method == 'PATCH') {
        final body =
            jsonDecode(await utf8.decoder.bind(request).join())
                as Map<String, dynamic>;
        writes.add(body['chatMessagePolicy'] as String);
        if (fail) {
          request.response.statusCode = 503;
          request.response.write(
            jsonEncode({
              'code': 1,
              'messageCode': 'common.operation.failed',
              'result': null,
            }),
          );
          await request.response.close();
          return;
        }
        policy = body['chatMessagePolicy'] as String;
      }
      request.response.write(
        jsonEncode({
          'code': 0,
          'result': {
            for (final key in profilePrivacyAudiences.keys) key: 'everyone',
            'chatMessagePolicy': policy,
          },
        }),
      );
      await request.response.close();
    });
    for (final brightness in Brightness.values) {
      await tester.pumpWidget(const SizedBox.shrink());
      policy = 'everyone';
      fail = false;
      final repository = UserRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Storage(),
          baseUrl: 'http://127.0.0.1:${server.port}',
        ),
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            currentUserProvider.overrideWith(
              (_) async => const CurrentUser(id: 1, username: 'demo'),
            ),
            userRepositoryProvider.overrideWithValue(repository),
          ],
          child: MaterialApp(
            theme: gfThemeData(brightness),
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePrivacyPage(chatOnly: true),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final row = find.byKey(const ValueKey('chat-message-policy'));
      expect(tester.widget<GfSettingRow>(row).value, '所有人');
      await binding.takeScreenshot('chat-permissions-${brightness.name}');
      await tester.tap(row);
      await tester.pumpAndSettle();
      await binding.takeScreenshot(
        'chat-permissions-picker-${brightness.name}',
      );
      await tester.tap(
        find.byKey(const ValueKey('profile-privacy-option-nobody')),
      );
      await tester.pumpAndSettle();
      expect(writes.last, 'nobody');
      expect(tester.widget<GfSettingRow>(row).value, '禁止新私信');
      await tester.pump(const Duration(seconds: 4));
      await tester.pumpAndSettle();
      fail = true;
      await tester.tap(row);
      await tester.pumpAndSettle();
      await tester.tap(
        find.byKey(const ValueKey('profile-privacy-option-everyone')),
      );
      await tester.pumpAndSettle();
      expect(tester.widget<GfSettingRow>(row).value, '禁止新私信');
      expect(find.byKey(const ValueKey('profile-privacy-widen')), findsNothing);
      await binding.takeScreenshot(
        'chat-permissions-failed-${brightness.name}',
      );
      expect(tester.takeException(), isNull);
      await tester.pump(const Duration(seconds: 4));
      await tester.pumpAndSettle();
    }
  });
}

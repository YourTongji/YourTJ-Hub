import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show NoopCache;
import 'pages_smoke_test.dart' show MemoryTokenStorage;

/// Serves the login page props; auth requests never leave the harness.
class _LoginOptions implements HttpClientAdapter {
  @override
  void close({bool force = false}) {}

  @override
  Future<ResponseBody> fetch(
    RequestOptions request,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    return ResponseBody.fromString(
      jsonEncode({
        'component': 'auth.login',
        'props': {
          'initialMode': 'login',
          'redirectUrl': '/',
          'githubUrl': '/api/auth/github',
          'googleReady': false,
          'tongjiReady': false,
          'tongjiUrl': '/api/auth/tongji',
          'allowedDomains': <String>[],
          'termsOfServiceEnabled': false,
          'privacyPolicyEnabled': false,
        },
        'layout': minimalLayoutJson(),
        'url': '/login',
        'version': '1',
        'meta': {'title': 'Login'},
      }),
      200,
      headers: {Headers.contentTypeHeader: ['application/json']},
    );
  }
}

/// Regression for issue #1111.
///
/// The Flutter Android engine maps `enableSuggestions: false` to an inputType
/// with `TYPE_TEXT_VARIATION_VISIBLE_PASSWORD`. ColorOS (OPPO/OnePlus/realme)
/// treats any field with that variation as a credential field: it force-switches
/// to the secure keyboard, hides clipboard entries from the suggestion strip and
/// blocks pasting. Only real password fields may keep `enableSuggestions: false`;
/// every other auth field must stay a plain text field.
void main() {
  Future<void> pumpLogin(WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(390, 1100));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final storage = MemoryTokenStorage();
    final staged = MemoryTokenStorage();
    final options = _LoginOptions();
    final container = ProviderContainer(
      overrides: [
        dioProvider.overrideWithValue(Dio()..httpClientAdapter = options),
        authDioProvider.overrideWithValue(Dio()..httpClientAdapter = options),
        tokenStorageProvider.overrideWithValue(storage),
        offlineTopicCacheProvider.overrideWithValue(NoopCache()),
        offlineChatCacheProvider.overrideWithValue(NoopCache()),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: LoginPage(authController: null, authTokenStorage: staged),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  Finder fieldByLabel(String label) => find.byWidgetPredicate(
    (widget) => widget is TextField && widget.decoration?.labelText == label,
  );

  bool enableSuggestionsOf(WidgetTester tester, Finder field) =>
      tester.widget<TextField>(field).enableSuggestions;

  testWidgets(
    'only password fields keep enableSuggestions off; username stays plain text',
    (tester) async {
      await pumpLogin(tester);
      final l10n = AppLocalizations.of(tester.element(find.byType(LoginPage)));

      final username = fieldByLabel(l10n.authUsernameOrEmail);
      final password = fieldByLabel(l10n.authPassword);
      expect(username, findsOneWidget);
      expect(password, findsOneWidget);
      expect(enableSuggestionsOf(tester, username), isTrue);
      expect(enableSuggestionsOf(tester, password), isFalse);

      await tester.tap(find.text(l10n.loginModeRegister).first);
      await tester.pumpAndSettle();

      final email = fieldByLabel(l10n.authEmail);
      final confirm = fieldByLabel(l10n.authConfirmPassword);
      expect(email, findsOneWidget);
      expect(confirm, findsOneWidget);
      expect(enableSuggestionsOf(tester, email), isTrue);
      expect(enableSuggestionsOf(tester, confirm), isFalse);
    },
  );
}

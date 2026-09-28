import 'dart:convert';
import 'dart:ui' show SemanticsAction, Tristate;

import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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
    expect(request.path, '/login');
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
      headers: {
        Headers.contentTypeHeader: ['application/json'],
      },
    );
  }
}

class _Auth extends AuthController {
  _Auth(GfApiClient client, TokenStorage storage)
    : super(
        authRepository: AuthRepository(client),
        apiClient: client,
        tokenStorage: storage,
      );

  int captchaLoads = 0;

  @override
  Future<void> loadCaptcha({
    bool preservePhaseOnError = false,
    bool silentOnError = false,
  }) async {
    captchaLoads++;
  }
}

void main() {
  const passwordToggleKey = Key('login-password-visibility');
  const confirmToggleKey = Key('register-confirm-password-visibility');

  Future<void> pumpLogin(WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(390, 1100));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final storage = MemoryTokenStorage();
    final staged = MemoryTokenStorage();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final auth = _Auth(client, staged);
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
          home: LoginPage(authController: auth, authTokenStorage: staged),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  Finder fieldByLabel(String label) => find.byWidgetPredicate(
    (widget) => widget is TextField && widget.decoration?.labelText == label,
  );

  Finder toggleSymbol(Key key) =>
      find.descendant(of: find.byKey(key), matching: find.byType(GfSymbol));

  AppLocalizations l10nOf(WidgetTester tester) =>
      AppLocalizations.of(tester.element(find.byType(LoginPage)));

  testWidgets(
    'login password starts obscured behind a labelled reveal toggle',
    (tester) async {
      final semanticsHandle = tester.ensureSemantics();
      await pumpLogin(tester);
      final l10n = l10nOf(tester);

      final password = fieldByLabel(l10n.authPassword);
      expect(password, findsOneWidget);
      expect(tester.widget<TextField>(password).obscureText, isTrue);

      final toggle = find.byKey(passwordToggleKey);
      expect(toggle, findsOneWidget);
      expect(
        tester.widget<GfSymbol>(toggleSymbol(passwordToggleKey)).name,
        'eye',
      );
      final semantics = tester
          .getSemantics(find.byTooltip(l10n.authShowPassword))
          .getSemanticsData();
      expect(semantics.label, l10n.authShowPassword);
      expect(semantics.flagsCollection.isButton, isTrue);
      expect(semantics.flagsCollection.isEnabled, Tristate.isTrue);
      expect(semantics.hasAction(SemanticsAction.tap), isTrue);
      expect(semantics.flagsCollection.isToggled, Tristate.isFalse);
      semanticsHandle.dispose();
    },
  );

  testWidgets('reveal toggle shows the password and keeps text, focus, caret', (
    tester,
  ) async {
    final semanticsHandle = tester.ensureSemantics();
    await pumpLogin(tester);
    final l10n = l10nOf(tester);

    final password = fieldByLabel(l10n.authPassword);
    await tester.enterText(password, 'correct horse battery staple');
    await tester.pump();

    final field = tester.widget<TextField>(password);
    final controller = field.controller!;
    final focusNode = field.focusNode!;
    final selection = controller.selection;
    expect(focusNode.hasFocus, isTrue);

    await tester.tap(find.byKey(passwordToggleKey));
    await tester.pumpAndSettle();

    expect(tester.widget<TextField>(password).obscureText, isFalse);
    expect(controller.text, 'correct horse battery staple');
    expect(controller.selection, selection);
    expect(focusNode.hasFocus, isTrue);
    expect(
      tester.widget<GfSymbol>(toggleSymbol(passwordToggleKey)).name,
      'eye-off',
    );
    final semantics = tester
        .getSemantics(find.byTooltip(l10n.authHidePassword))
        .getSemanticsData();
    expect(semantics.label, l10n.authHidePassword);
    expect(semantics.flagsCollection.isToggled, Tristate.isTrue);
    semanticsHandle.dispose();
  });

  testWidgets('reveal toggle stays reachable by keyboard traversal', (
    tester,
  ) async {
    final semanticsHandle = tester.ensureSemantics();
    await pumpLogin(tester);
    final l10n = l10nOf(tester);

    final password = fieldByLabel(l10n.authPassword);
    await tester.enterText(password, 'secret');
    await tester.pump();

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pumpAndSettle();

    final semantics = tester
        .getSemantics(find.byTooltip(l10n.authShowPassword))
        .getSemanticsData();
    expect(semantics.flagsCollection.isFocused, Tristate.isTrue);
    semanticsHandle.dispose();
  });

  testWidgets('toggling back obscures the password again', (tester) async {
    await pumpLogin(tester);
    final l10n = l10nOf(tester);

    final password = fieldByLabel(l10n.authPassword);
    await tester.enterText(password, 'hunter2');
    await tester.tap(find.byKey(passwordToggleKey));
    await tester.pumpAndSettle();
    expect(tester.widget<TextField>(password).obscureText, isFalse);

    await tester.tap(find.byKey(passwordToggleKey));
    await tester.pumpAndSettle();

    expect(tester.widget<TextField>(password).obscureText, isTrue);
    expect(tester.widget<TextField>(password).controller!.text, 'hunter2');
    expect(find.byTooltip(l10n.authShowPassword), findsOneWidget);
  });

  testWidgets('reveal toggle keeps a 44px minimum tap target', (tester) async {
    await pumpLogin(tester);

    final size = tester.getSize(find.byKey(passwordToggleKey));
    expect(size.width, greaterThanOrEqualTo(44));
    expect(size.height, greaterThanOrEqualTo(44));
  });

  testWidgets('register confirm password can be revealed the same way', (
    tester,
  ) async {
    await pumpLogin(tester);
    final l10n = l10nOf(tester);

    await tester.tap(find.text(l10n.loginModeRegister).first);
    await tester.pumpAndSettle();

    final confirm = fieldByLabel(l10n.authConfirmPassword);
    expect(confirm, findsOneWidget);
    expect(tester.widget<TextField>(confirm).obscureText, isTrue);

    await tester.enterText(confirm, 'second secret');
    await tester.tap(find.byKey(confirmToggleKey));
    await tester.pumpAndSettle();

    expect(tester.widget<TextField>(confirm).obscureText, isFalse);
    expect(tester.widget<TextField>(confirm).controller!.text, 'second secret');
  });
}

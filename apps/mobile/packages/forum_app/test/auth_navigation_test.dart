import 'dart:async';

import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show InstantLoginController, NoopCache;
import 'pages_smoke_test.dart' show MemoryTokenStorage;

Future<GoRouter> pumpLogin(
  WidgetTester tester, {
  String? returnTo,
  AuthController Function(GfApiClient, TokenStorage)? makeAuth,
}) async {
  final storage = MemoryTokenStorage();
  final staged = MemoryTokenStorage();
  final client = GfApiClient(
    dio: Dio(),
    tokenStorage: staged,
    baseUrl: 'http://fake.local',
  );
  final auth =
      makeAuth?.call(client, staged) ??
      InstantLoginController(apiClient: client, tokenStorage: staged);
  final dio = Dio()
    ..interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: {
                'component': 'auth.login',
                'props': {
                  'initialMode': 'login',
                  'redirectUrl': '/',
                  'githubUrl': '/oauth/github',
                  'googleReady': false,
                  'tongjiReady': true,
                  'tongjiUrl': '/api/auth/tongji',
                  'allowedDomains': [],
                  'termsOfServiceEnabled': false,
                  'privacyPolicyEnabled': false,
                },
                'layout': minimalLayoutJson(),
                'url': '/login',
                'version': '1',
                'meta': {'title': 'Login'},
              },
            ),
          );
        },
      ),
    );
  final container = ProviderContainer(
    overrides: [
      tokenStorageProvider.overrideWithValue(storage),
      authDioProvider.overrideWithValue(dio),
      offlineTopicCacheProvider.overrideWithValue(NoopCache()),
      offlineChatCacheProvider.overrideWithValue(NoopCache()),
    ],
  );
  addTearDown(container.dispose);
  final router = GoRouter(
    initialLocation: Uri(
      path: '/login',
      queryParameters: returnTo == null ? null : {'returnTo': returnTo},
    ).toString(),
    routes: [
      GoRoute(
        path: '/',
        builder: (_, _) => const Scaffold(body: Text('Home destination')),
      ),
      GoRoute(
        path: '/p/:id',
        builder: (_, state) =>
            Scaffold(body: Text('Topic destination ${state.uri}')),
      ),
      for (final target in [
        '/drafts',
        '/my-content',
        '/recycle-bin',
        '/my-course-reviews',
      ])
        GoRoute(
          path: target,
          builder: (_, _) => Scaffold(
            appBar: GfAppBar(title: Text(target)),
            body: Text('Destination $target'),
          ),
        ),
      GoRoute(
        path: '/login',
        builder: (_, _) =>
            LoginPage(authController: auth, authTokenStorage: staged),
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
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

void main() {
  for (final target in ['https://example.com', '/login?returnTo=%2Fp%2F1']) {
    testWidgets('login falls back to Home for $target', (tester) async {
      final router = await pumpLogin(tester, returnTo: target);
      final button = find.byWidgetPredicate(
        (w) => w is GfButton && w.size == GfButtonSize.extraLarge,
      );
      await tester.ensureVisible(button);
      await tester.tap(button);
      await tester.pumpAndSettle();
      expect(router.state.uri.toString(), '/');
    });
  }
  testWidgets('login restores the original topic and reply position', (
    tester,
  ) async {
    final router = await pumpLogin(tester, returnTo: '/p/123?postNo=8');
    final button = find.byWidgetPredicate(
      (w) => w is GfButton && w.size == GfButtonSize.extraLarge,
    );
    await tester.ensureVisible(button);
    await tester.tap(button);
    await tester.pumpAndSettle();
    expect(router.state.uri.toString(), '/p/123?postNo=8');
  });
  testWidgets(
    'login fields expose native credential autofill and disable corrections',
    (tester) async {
      await pumpLogin(tester);
      final fields = tester
          .widgetList<TextField>(find.byType(TextField))
          .toList();
      expect(fields.first.autofillHints, contains(AutofillHints.username));
      expect(fields.first.autocorrect, isFalse);
      expect(fields.first.textInputAction, TextInputAction.next);
      expect(fields[1].autofillHints, contains(AutofillHints.password));
      expect(fields[1].enableSuggestions, isFalse);
    },
  );

  for (final target in [
    '/drafts',
    '/my-content',
    '/recycle-bin',
    '/my-course-reviews',
  ]) {
    testWidgets(
      'restored $target can return to Home without revisiting login',
      (tester) async {
        final router = await pumpLogin(tester, returnTo: target);
        final button = find.byWidgetPredicate(
          (w) => w is GfButton && w.size == GfButtonSize.extraLarge,
        );
        await tester.ensureVisible(button);
        await tester.tap(button);
        await tester.pumpAndSettle();
        expect(router.state.uri.path, target);
        expect(router.canPop(), isTrue);
        router.pop();
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/');
        expect(find.byType(LoginPage), findsNothing);
      },
    );
  }

  for (final register in [true, false]) {
    testWidgets(
      'repeated IME submit starts only one ${register ? 'registration' : 'reset'} request',
      (tester) async {
        late _PendingAuth auth;
        await pumpLogin(
          tester,
          makeAuth: (client, storage) => auth = _PendingAuth(client, storage),
        );
        final mode = find.text(register ? 'Sign up' : 'Forgot password?').first;
        await tester.ensureVisible(mode);
        await tester.tap(mode);
        await tester.pumpAndSettle();
        final field = tester.widgetList<TextField>(find.byType(TextField)).last;
        field.onSubmitted!('');
        field.onSubmitted!('');
        expect(auth.calls, 1);
        auth.pending.complete();
        await tester.pumpAndSettle();
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(seconds: 4));
      },
    );
  }
}

class _PendingAuth extends AuthController {
  _PendingAuth(GfApiClient client, TokenStorage storage)
    : super(
        authRepository: AuthRepository(client),
        apiClient: client,
        tokenStorage: storage,
      );
  int calls = 0;
  bool _pending = false;
  final pending = Completer<void>();
  @override
  bool get busy => _pending;
  Future<void> _request() async {
    calls++;
    _pending = true;
    notifyListeners();
    await pending.future;
    _pending = false;
    notifyListeners();
  }

  @override
  Future<void> register({
    required String username,
    required String email,
    required String password,
    String? captchaId,
    String? captchaCode,
  }) => _request();
  @override
  Future<void> forgotPassword({
    required String email,
    String? captchaId,
    String? captchaCode,
  }) => _request();
}

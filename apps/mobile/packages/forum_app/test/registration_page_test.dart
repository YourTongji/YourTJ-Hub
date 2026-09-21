import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';
import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show NoopCache;
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Options implements HttpClientAdapter {
  _Options({
    required this.domains,
    required this.policies,
    this.fail = false,
    this.tongji = false,
  });
  final headers = <String?>[];
  @override
  void close({bool force = false}) {}
  final List<String> domains;
  final bool policies;
  final bool tongji;
  bool fail;
  @override
  Future<ResponseBody> fetch(
    RequestOptions request,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    expect(request.path, '/login');
    if (fail) {
      throw DioException(
        requestOptions: request,
        type: DioExceptionType.connectionError,
      );
    }
    final authorization = request.headers['Authorization'] as String?;
    headers.add(authorization);
    final payload = authorization != null
        ? homePayloadJson()
        : {
            'component': 'auth.login',
            'props': {
              'initialMode': 'login',
              'redirectUrl': '/',
              'githubUrl': '',
              'googleReady': false,
              'tongjiReady': tongji,
              'tongjiUrl': '/api/auth/tongji',
              'allowedDomains': domains,
              'termsOfServiceEnabled': policies,
              'privacyPolicyEnabled': policies,
            },
            'layout': minimalLayoutJson(),
            'url': '/login',
            'version': '1',
            'meta': {'title': 'Login'},
          };
    return ResponseBody.fromString(
      jsonEncode(payload),
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
  final emails = <String>[];
  int captchaLoads = 0;
  bool? lastPreservePhaseOnError;
  bool? lastSilentOnError;
  Completer<void>? captchaGate;
  Object? captchaError;
  CaptchaPayload? captchaPayload;

  @override
  CaptchaPayload? get captcha => captchaPayload;

  @override
  Future<void> loadCaptcha({
    bool preservePhaseOnError = false,
    bool silentOnError = false,
  }) async {
    captchaLoads++;
    lastPreservePhaseOnError = preservePhaseOnError;
    lastSilentOnError = silentOnError;
    final Completer<void>? gate = captchaGate;
    if (gate != null) await gate.future;
    if (captchaError != null) throw captchaError!;
    captchaPayload = CaptchaPayload(
      captchaId: 'test-captcha',
      captchaImg:
          'data:image/png;base64,${base64Encode(img.encodePng(img.Image(width: 2, height: 2)))}',
    );
    notifyListeners();
  }

  @override
  Future<void> register({
    required String username,
    required String email,
    required String password,
    String? captchaId,
    String? captchaCode,
  }) async {
    emails.add(email);
  }
}

void main() {
  Future<({_Auth auth, _Options options, ProviderContainer container})> pump(
    WidgetTester tester, {
    List<String> domains = const [],
    bool policies = false,
    bool fail = false,
    bool oldSession = false,
    bool register = true,
    bool tongji = false,
  }) async {
    await tester.binding.setSurfaceSize(const Size(390, 1100));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final storage = MemoryTokenStorage();
    if (oldSession) await storage.write('old-token');
    final staged = MemoryTokenStorage();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final auth = _Auth(client, staged);
    final options = _Options(
      domains: domains,
      policies: policies,
      fail: fail,
      tongji: tongji,
    );
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
    if (register) await tester.tap(find.text('Sign up').first);
    await tester.pumpAndSettle();
    return (auth: auth, options: options, container: container);
  }

  Finder input(String label) => find.byWidgetPredicate(
    (widget) => widget is TextField && widget.decoration?.labelText == label,
  );
  Finder submit() => find.byWidgetPredicate(
    (widget) => widget is GfButton && widget.size == GfButtonSize.extraLarge,
  );
  Future<void> fill(
    WidgetTester tester,
    String emailLabel,
    String email,
  ) async {
    await tester.enterText(input('Username'), 'mobile');
    await tester.enterText(input(emailLabel), email);
    await tester.enterText(input('Password'), 'test-password');
    await tester.enterText(input('Confirm password'), 'test-password');
  }

  for (final register in [false, true]) {
    testWidgets(
      'Tongji sign-in appears in register=$register when configured',
      (tester) async {
        await pump(tester, register: register, tongji: true);
        final button = find.widgetWithText(
          OutlinedButton,
          'Continue with Tongji SSO',
        );
        expect(button, findsOneWidget);
        expect(tester.widget<OutlinedButton>(button).onPressed, isNotNull);
        expect(find.textContaining('student-ID@tongji.edu.cn'), findsOneWidget);
        expect(tester.takeException(), isNull);
      },
    );
  }
  testWidgets('Tongji sign-in stays hidden without campus configuration', (
    tester,
  ) async {
    await pump(tester, register: false);
    expect(find.text('Continue with Tongji SSO'), findsNothing);
  });

  testWidgets('login captcha starts folded before password interaction', (
    tester,
  ) async {
    await pump(tester, register: false);

    expect(find.byKey(const Key('login-captcha')), findsNothing);
  });

  testWidgets('password blur reveals captcha and refocus keeps it visible', (
    tester,
  ) async {
    final h = await pump(tester, register: false);
    final password = input('Password');
    final username = input('Username or email');

    await tester.tap(password);
    await tester.pump();
    expect(find.byKey(const Key('login-captcha')), findsNothing);
    expect(h.auth.captchaLoads, 1);

    await tester.tap(username);
    await tester.pump();
    // Reveal is committed from a post-frame callback so the focus handoff
    // itself is not rebuilt mid-transfer; the next frame contains the image.
    await tester.pump();
    expect(find.byKey(const Key('login-captcha')), findsOneWidget);
    expect(input('Captcha'), findsOneWidget);
    expect(h.auth.captchaLoads, 1);
    expect(h.auth.lastPreservePhaseOnError, isTrue);

    final captcha = input('Captcha');
    await tester.tap(captcha);
    await tester.pump();
    expect(tester.widget<TextField>(captcha).focusNode?.hasFocus, isTrue);

    await tester.tap(password);
    await tester.pump();
    await tester.tap(username);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('login-captcha')), findsOneWidget);
    expect(h.auth.captchaLoads, 1);
  });

  testWidgets('focus bounce during captcha request stays latched and deduped', (
    tester,
  ) async {
    final h = await pump(tester, register: false);
    h.auth.captchaGate = Completer<void>();

    await tester.tap(input('Password'));
    await tester.pump();
    expect(h.auth.captchaLoads, 1);
    expect(find.byKey(const Key('login-captcha')), findsNothing);

    await tester.tap(input('Username or email'));
    await tester.pump();
    await tester.tap(input('Password'));
    await tester.pump();
    await tester.tap(input('Username or email'));
    await tester.pump();

    expect(find.byKey(const Key('login-captcha')), findsOneWidget);
    expect(h.auth.captchaLoads, 1);

    h.auth.captchaGate!.complete();
    await tester.pumpAndSettle();
    expect(input('Captcha'), findsOneWidget);
  });

  testWidgets(
    'blank-space pointer intent reveals captcha without focus battle',
    (tester) async {
      final h = await pump(tester, register: false);

      await tester.tap(input('Password'));
      await tester.pump();
      expect(find.byKey(const Key('login-captcha')), findsNothing);

      await tester.tapAt(const Offset(385, 1080));
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('login-captcha')), findsOneWidget);
      expect(h.auth.captchaLoads, 1);
    },
  );

  testWidgets(
    'captcha prefetch failure stays silent and retries when visible',
    (tester) async {
      final h = await pump(tester, register: false);
      h.auth.captchaError = StateError('offline');

      await tester.tap(input('Password'));
      await tester.pump();
      expect(h.auth.captchaLoads, 1);
      expect(find.text('Failed to load captcha'), findsNothing);
      expect(find.byKey(const Key('login-captcha')), findsNothing);

      h.auth.captchaError = null;
      await tester.tap(input('Username or email'));
      await tester.pumpAndSettle();

      expect(h.auth.captchaLoads, 2);
      expect(find.byKey(const Key('login-captcha')), findsOneWidget);
      expect(input('Captcha'), findsOneWidget);
    },
  );

  testWidgets('captcha dark filter changes with theme without refetching', (
    tester,
  ) async {
    final h = await pump(tester, register: false);
    final username = input('Username or email');
    await tester.tap(input('Password'));
    await tester.tap(username);
    await tester.pumpAndSettle();

    expect(find.byType(ColorFiltered), findsNothing);
    expect(h.auth.captchaLoads, 1);
    final String? captchaId = h.auth.captcha?.captchaId;

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: h.container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.dark),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: LoginPage(
            authController: h.auth,
            authTokenStorage: MemoryTokenStorage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final filter = tester.widget<ColorFiltered>(find.byType(ColorFiltered));
    expect(filter.colorFilter.toString(), contains('255.0'));
    expect(h.auth.captchaLoads, 1);
    expect(h.auth.captcha?.captchaId, captchaId);
  });

  testWidgets(
    'sign-in loads social availability before entering registration',
    (tester) async {
      final h = await pump(tester, register: false, oldSession: true);
      expect(h.options.headers, [null]);
      final github = tester.widget<OutlinedButton>(
        find.widgetWithText(OutlinedButton, 'Continue with GitHub'),
      );
      final google = tester.widget<OutlinedButton>(
        find.widgetWithText(OutlinedButton, 'Continue with Google'),
      );
      expect(github.onPressed, isNotNull);
      expect(google.onPressed, isNull);
    },
  );

  testWidgets('registration options never carry the previous account session', (
    tester,
  ) async {
    final h = await pump(tester, oldSession: true, domains: ['tongji.edu.cn']);
    expect(find.byKey(const Key('register-email-domain')), findsOneWidget);
    expect(h.options.headers, [null]);
  });
  testWidgets(
    'registration joins selected domain and requires published policies',
    (tester) async {
      final h = await pump(
        tester,
        domains: ['tongji.edu.cn', 's.tongji.edu.cn'],
        policies: true,
      );
      await fill(tester, 'Email username', 'student');
      expect(tester.widget<GfButton>(submit()).onPressed, isNull);
      expect(find.text('Privacy policy'), findsOneWidget);
      await tester.tap(find.byKey(const Key('register-email-domain')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('@s.tongji.edu.cn').last);
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.byType(CheckboxListTile));
      await tester.tap(find.byType(CheckboxListTile));
      await tester.pumpAndSettle();
      await tester.ensureVisible(submit());
      await tester.tap(submit());
      await tester.pumpAndSettle();
      expect(h.auth.emails, ['student@s.tongji.edu.cn']);
      await tester.pump(const Duration(seconds: 4));
    },
  );
  testWidgets(
    'unrestricted registration keeps the full email and rejects mismatched passwords',
    (tester) async {
      final h = await pump(tester);
      await fill(tester, 'Email', 'person@example.test');
      expect(find.byType(CheckboxListTile), findsNothing);
      expect(find.byKey(const Key('register-email-domain')), findsNothing);
      await tester.enterText(input('Confirm password'), 'different');
      await tester.ensureVisible(submit());
      await tester.tap(submit());
      await tester.pumpAndSettle();
      expect(h.auth.emails, isEmpty);
      expect(find.text('The passwords do not match'), findsOneWidget);
      await tester.enterText(input('Confirm password'), 'test-password');
      await tester.ensureVisible(submit());
      await tester.tap(submit());
      await tester.pumpAndSettle();
      expect(h.auth.emails, ['person@example.test']);
      await tester.pump(const Duration(seconds: 4));
    },
  );
  testWidgets('failed registration options can retry without losing the form', (
    tester,
  ) async {
    final h = await pump(tester, fail: true);
    await tester.enterText(input('Username'), 'retained');
    expect(tester.widget<GfButton>(submit()).onPressed, isNull);
    h.options.fail = false;
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(
      tester.widget<TextField>(input('Username')).controller!.text,
      'retained',
    );
    expect(tester.widget<GfButton>(submit()).onPressed, isNotNull);
  });
}

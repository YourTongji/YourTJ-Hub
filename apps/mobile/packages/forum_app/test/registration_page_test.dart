import 'dart:async';
import 'dart:convert';
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
              'githubUrl': '/api/auth/github',
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
  final loginRequests =
      <({String username, String password, String? captchaCode})>[];

  @override
  LoginPhase get phase =>
      loginRequests.isEmpty ? super.phase : LoginPhase.failed;

  @override
  String get error =>
      loginRequests.isEmpty ? super.error : 'Invalid credentials';

  @override
  Future<void> login({
    required String username,
    required String password,
    String? captchaId,
    String? captchaCode,
  }) async {
    loginRequests.add((
      username: username,
      password: password,
      captchaCode: captchaCode,
    ));
    notifyListeners();
  }

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
    Locale locale = const Locale('en'),
    double width = 390,
    double textScale = 1,
  }) async {
    await tester.binding.setSurfaceSize(Size(width, 1100));
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
          locale: locale,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: TextScaler.linear(textScale)),
            child: child!,
          ),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: LoginPage(authController: auth, authTokenStorage: staged),
        ),
      ),
    );
    await tester.pumpAndSettle();
    if (register) {
      final l10n = AppLocalizations.of(tester.element(find.byType(LoginPage)));
      await tester.tap(find.text(l10n.loginModeRegister).first);
    }
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

  for (final language in ['zh', 'en', 'ja', 'de']) {
    for (final width in [320.0, 390.0]) {
      testWidgets('$language auth modes at $width and 200% text', (
        tester,
      ) async {
        await pump(
          tester,
          register: false,
          tongji: true,
          policies: true,
          locale: Locale(language),
          width: width,
          textScale: 2,
        );
        final l10n = AppLocalizations.of(
          tester.element(find.byType(LoginPage)),
        );
        expect(tester.takeException(), isNull);
        await tester.ensureVisible(input(l10n.authPassword));
        await tester.tap(input(l10n.authPassword));
        await tester.testTextInput.receiveAction(TextInputAction.next);
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('login-captcha')), findsOneWidget);
        expect(
          tester.getSize(input(l10n.authCaptcha)).width,
          greaterThanOrEqualTo(140),
        );
        expect(tester.takeException(), isNull);
        await tester.ensureVisible(find.text(l10n.loginModeRegister).first);
        await tester.tap(find.text(l10n.loginModeRegister).first);
        await tester.pumpAndSettle();
        expect(
          tester
              .widget<TextField>(input(l10n.authConfirmPassword))
              .autofillHints,
          contains(AutofillHints.newPassword),
        );
        expect(tester.takeException(), isNull);
        await tester.ensureVisible(find.text(l10n.loginModeLogin).first);
        await tester.tap(find.text(l10n.loginModeLogin).first);
        await tester.pumpAndSettle();
        await tester.ensureVisible(find.text(l10n.authForgotPassword));
        await tester.tap(find.text(l10n.authForgotPassword));
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
      });
    }
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

  for (final delayed in [false, true]) {
    testWidgets(
      'explicit username focus wins over ${delayed ? 'delayed' : 'prefetched'} captcha',
      (tester) async {
        final h = await pump(tester, register: false);
        if (delayed) h.auth.captchaGate = Completer<void>();
        await tester.enterText(input('Username or email'), 'mobile');
        await tester.enterText(input('Password'), 'secret');
        await tester.tap(input('Username or email'));
        await tester.pumpAndSettle();
        if (delayed) {
          h.auth.captchaGate!.complete();
          await tester.pumpAndSettle();
        }

        expect(input('Captcha'), findsOneWidget);
        expect(
          tester
              .widget<TextField>(input('Username or email'))
              .focusNode!
              .hasFocus,
          isTrue,
        );
        expect(
          tester.widget<TextField>(input('Captcha')).focusNode!.hasFocus,
          isFalse,
        );
        expect(
          tester.widget<TextField>(input('Username or email')).controller!.text,
          'mobile',
        );
        expect(
          tester.widget<TextField>(input('Password')).controller!.text,
          'secret',
        );
      },
      variant: TargetPlatformVariant({
        TargetPlatform.android,
        TargetPlatform.iOS,
      }),
    );
  }

  testWidgets(
    'blank outside tap reveals captcha without opening another input',
    (tester) async {
      await pump(tester, register: false);
      await tester.enterText(input('Password'), 'secret');
      await tester.tapAt(const Offset(385, 1080));
      await tester.pumpAndSettle();

      expect(input('Captcha'), findsOneWidget);
      expect(
        tester.widget<TextField>(input('Password')).focusNode!.hasFocus,
        isFalse,
      );
      expect(
        tester.widget<TextField>(input('Captcha')).focusNode!.hasFocus,
        isFalse,
      );
      expect(tester.testTextInput.isVisible, isFalse);
      expect(
        tester.widget<TextField>(input('Password')).controller!.text,
        'secret',
      );
    },
    variant: TargetPlatformVariant({
      TargetPlatform.android,
      TargetPlatform.iOS,
    }),
  );

  testWidgets(
    'keyboard next enters password through a secure IME transition',
    (tester) async {
      addTearDown(tester.view.resetViewInsets);
      await pump(tester, register: false);
      await tester.enterText(input('Username or email'), 'mobile');
      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      await tester.pump();

      await tester.testTextInput.receiveAction(TextInputAction.next);
      await tester.pump();
      tester.view.viewInsets = const FakeViewPadding();
      await tester.pump();
      await tester.pump();

      expect(
        tester.widget<TextField>(input('Password')).focusNode!.hasFocus,
        isTrue,
      );
      expect(find.byKey(const Key('login-captcha')), findsNothing);
      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      await tester.pump();
      await tester.enterText(input('Password'), 'secret');
      await tester.testTextInput.receiveAction(TextInputAction.next);
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(input('Captcha')).focusNode!.hasFocus,
        isTrue,
      );
      expect(
        tester.widget<TextField>(input('Password')).controller!.text,
        'secret',
      );
    },
    variant: TargetPlatformVariant.only(TargetPlatform.android),
  );

  testWidgets(
    'dismissing the secure keyboard does not reopen it on captcha',
    (tester) async {
      addTearDown(tester.view.resetViewInsets);
      await pump(tester, register: false);
      await tester.enterText(input('Password'), 'secret');
      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      await tester.pump();
      tester.testTextInput.hide();
      tester.view.viewInsets = const FakeViewPadding();
      await tester.pumpAndSettle();

      expect(input('Captcha'), findsOneWidget);
      expect(
        tester.widget<TextField>(input('Password')).focusNode!.hasFocus,
        isFalse,
      );
      expect(
        tester.widget<TextField>(input('Captcha')).focusNode!.hasFocus,
        isFalse,
      );
      expect(tester.testTextInput.isVisible, isFalse);
    },
    variant: TargetPlatformVariant.only(TargetPlatform.android),
  );

  for (final label in ['Username or email', 'Captcha']) {
    testWidgets(
      'outside tap dismisses the $label keyboard and preserves its input',
      (tester) async {
        await pump(tester, register: false);
        if (label == 'Captcha') {
          await tester.enterText(input('Password'), 'secret');
          await tester.testTextInput.receiveAction(TextInputAction.next);
          await tester.pumpAndSettle();
        }
        await tester.enterText(input(label), 'retained');
        await tester.tap(
          find.byWidgetPredicate(
            (widget) => widget is Image && widget.semanticLabel == 'YourTJ',
          ),
        );
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(input(label)).focusNode!.hasFocus,
          isFalse,
        );
        expect(tester.testTextInput.isVisible, isFalse);
        expect(
          tester.widget<TextField>(input(label)).controller!.text,
          'retained',
        );
      },
      variant: TargetPlatformVariant({
        TargetPlatform.android,
        TargetPlatform.iOS,
      }),
    );
  }

  testWidgets('registration next advances exactly one editable field', (
    tester,
  ) async {
    await pump(tester);
    await tester.enterText(input('Username'), 'mobile');
    for (final label in ['Email', 'Password', 'Confirm password']) {
      await tester.testTextInput.receiveAction(TextInputAction.next);
      await tester.pumpAndSettle();
      final editable = tester.widget<EditableText>(
        find.descendant(of: input(label), matching: find.byType(EditableText)),
      );
      expect(
        editable.focusNode.hasFocus,
        isTrue,
        reason: 'Next should focus $label',
      );
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'failed login preserves every field and allows a deliberate retry',
    (tester) async {
      final h = await pump(tester, register: false);
      await tester.enterText(input('Username or email'), 'mobile');
      await tester.enterText(input('Password'), 'secret');
      await tester.testTextInput.receiveAction(TextInputAction.next);
      await tester.pumpAndSettle();
      await tester.enterText(input('Captcha'), '1234');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();

      expect(find.text('Invalid credentials'), findsOneWidget);
      expect(h.auth.loginRequests, [
        (username: 'mobile', password: 'secret', captchaCode: '1234'),
      ]);
      expect(
        tester.widget<TextField>(input('Username or email')).controller!.text,
        'mobile',
      );
      expect(
        tester.widget<TextField>(input('Password')).controller!.text,
        'secret',
      );
      expect(
        tester.widget<TextField>(input('Captcha')).controller!.text,
        '1234',
      );
      expect(tester.testTextInput.isVisible, isFalse);

      await tester.tap(input('Username or email'));
      await tester.pumpAndSettle();
      expect(
        tester
            .widget<TextField>(input('Username or email'))
            .focusNode!
            .hasFocus,
        isTrue,
      );
      await tester.enterText(input('Username or email'), 'mobile-fixed');
      await tester.testTextInput.receiveAction(TextInputAction.next);
      await tester.pump();
      await tester.testTextInput.receiveAction(TextInputAction.next);
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(input('Captcha')).focusNode!.hasFocus,
        isTrue,
      );
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();
      expect(h.auth.loginRequests.last.username, 'mobile-fixed');
      expect(h.auth.loginRequests.length, 2);
    },
  );

  testWidgets(
    'switching mode cancels a delayed captcha handoff and keeps credentials',
    (tester) async {
      final h = await pump(tester, register: false);
      h.auth.captchaGate = Completer<void>();
      await tester.enterText(input('Username or email'), 'mobile');
      await tester.enterText(input('Password'), 'secret');
      await tester.testTextInput.receiveAction(TextInputAction.next);
      await tester.pump();
      await tester.tap(find.text('Sign up').first);
      await tester.pumpAndSettle();
      h.auth.captchaGate!.complete();
      await tester.pumpAndSettle();

      expect(input('Captcha'), findsNothing);
      expect(tester.testTextInput.isVisible, isFalse);
      expect(
        tester.widget<TextField>(input('Username')).controller!.text,
        'mobile',
      );
      expect(
        tester.widget<TextField>(input('Password')).controller!.text,
        'secret',
      );
      await tester.tap(find.text('Sign in').first);
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('login-captcha')), findsNothing);
      expect(tester.testTextInput.isVisible, isFalse);
      expect(
        tester.widget<TextField>(input('Username or email')).controller!.text,
        'mobile',
      );
    },
  );

  testWidgets('leaving login cancels a delayed captcha handoff', (
    tester,
  ) async {
    final h = await pump(tester, register: false);
    h.auth.captchaGate = Completer<void>();
    await tester.enterText(input('Password'), 'secret');
    await tester.testTextInput.receiveAction(TextInputAction.next);
    await tester.pump();
    await tester.pumpWidget(const SizedBox.shrink());
    h.auth.captchaGate!.complete();
    await tester.pumpAndSettle();

    expect(find.byType(LoginPage), findsNothing);
    expect(tester.testTextInput.isVisible, isFalse);
    expect(tester.takeException(), isNull);
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
      expect(github.onPressed, isNotNull);
      expect(
        find.widgetWithText(OutlinedButton, 'Continue with Google'),
        findsNothing,
      );
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

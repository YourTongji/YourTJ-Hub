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

/// Serves the `/login` page props so [LoginPage] can load registration options.
class _PageProps implements HttpClientAdapter {
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

/// Login-page controller with an injectable captcha challenge.
///
/// Starts in [LoginPhase.needsCaptcha] with an already loaded challenge so the
/// captcha row is visible without driving the password handoff, and every
/// successful load returns a distinguishable image so tests can prove the
/// displayed challenge really changed.
class _CaptchaAuth extends AuthController {
  _CaptchaAuth(GfApiClient client, TokenStorage storage)
    : super(
        authRepository: AuthRepository(client),
        apiClient: client,
        tokenStorage: storage,
      ) {
    // Generation 0 is the already-visible challenge; every load returns a
    // higher generation so an unchanged image proves a broken refresh.
    _payload = _payloadFor(0);
  }

  int captchaLoads = 0;
  int loginCalls = 0;
  Completer<void>? gate;
  Object? loadError;
  String fakeError = '';
  CaptchaPayload? _payload;
  LoginPhase _phase = LoginPhase.needsCaptcha;

  static CaptchaPayload _payloadFor(int generation) => CaptchaPayload(
    captchaId: 'captcha-$generation',
    captchaImg:
        'data:image/png;base64,${base64Encode(img.encodePng(img.Image(width: generation + 1, height: 2)))}',
  );

  @override
  LoginPhase get phase => _phase;

  @override
  String get error => fakeError;

  @override
  CaptchaPayload? get captcha => _payload;

  @override
  Future<void> loadCaptcha({
    bool preservePhaseOnError = false,
    bool silentOnError = false,
  }) async {
    captchaLoads++;
    final Completer<void>? pending = gate;
    if (pending != null) await pending.future;
    if (loadError != null) {
      // 与 AuthController 一致:不保留 phase 时刷新失败会把登录阶段打成
      // failed,验证码行(重试入口)随之消失。
      if (!preservePhaseOnError) _phase = LoginPhase.failed;
      if (!silentOnError) fakeError = 'Failed to load captcha';
      notifyListeners();
      return;
    }
    _payload = _payloadFor(captchaLoads);
    fakeError = '';
    notifyListeners();
  }

  @override
  Future<void> login({
    required String username,
    required String password,
    String? captchaId,
    String? captchaCode,
  }) async {
    loginCalls++;
    notifyListeners();
  }
}

void main() {
  Finder input(String label) => find.byWidgetPredicate(
    (widget) => widget is TextField && widget.decoration?.labelText == label,
  );
  Finder refreshTarget() => find.byKey(const Key('login-captcha-refresh'));
  Finder submitButton() => find.byWidgetPredicate(
    (widget) => widget is GfButton && widget.size == GfButtonSize.extraLarge,
  );
  String shownImage(WidgetTester tester) =>
      tester.widget<GfCaptchaImage>(find.byType(GfCaptchaImage)).imageData;

  Future<_Harness> pumpLogin(WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(390, 1100));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final storage = MemoryTokenStorage();
    final staged = MemoryTokenStorage();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: staged,
      baseUrl: 'http://fake.local',
    );
    final auth = _CaptchaAuth(client, staged);
    final adapter = _PageProps();
    final container = ProviderContainer(
      overrides: [
        dioProvider.overrideWithValue(Dio()..httpClientAdapter = adapter),
        authDioProvider.overrideWithValue(Dio()..httpClientAdapter = adapter),
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
    return _Harness(auth: auth);
  }

  testWidgets(
    'tapping the captcha requests a new challenge and clears the typed code',
    (tester) async {
      final h = await pumpLogin(tester);
      final String before = shownImage(tester);
      expect(h.auth.captchaLoads, 0);

      await tester.enterText(input('Captcha'), '1234');
      await tester.tap(refreshTarget());
      await tester.pumpAndSettle();

      expect(h.auth.captchaLoads, 1);
      expect(h.auth.captcha?.captchaId, 'captcha-1');
      expect(shownImage(tester), isNot(before));
      expect(
        tester.widget<TextField>(input('Captcha')).controller!.text,
        isEmpty,
      );
    },
  );

  testWidgets('a second tap during a held refresh changes nothing', (
    tester,
  ) async {
    final h = await pumpLogin(tester);
    h.auth.gate = Completer<void>();

    await tester.tap(refreshTarget());
    await tester.pump();
    // 在途请求显示进度覆盖层,旧图仍在。
    expect(find.byType(GfProgressIndicator), findsOneWidget);
    expect(find.byType(GfCaptchaImage), findsOneWidget);
    await tester.enterText(input('Captcha'), '5678');
    await tester.pump();

    final Widget imageDuring = tester.widget(find.byType(GfCaptchaImage));
    final TextEditingController controller = tester
        .widget<TextField>(input('Captcha'))
        .controller!;
    // 刷新在途时验证码图本身不再接受点击(单飞的唯一守卫)。
    expect(tester.widget<InkWell>(refreshTarget()).onTap, isNull);

    await tester.tap(refreshTarget());
    await tester.pump();

    expect(h.auth.captchaLoads, 1);
    // 第二次点击不得重新进入刷新状态(否则这里会因 setState 重建而换实例)。
    expect(
      identical(tester.widget(find.byType(GfCaptchaImage)), imageDuring),
      isTrue,
      reason: '第二次点击不得重新进入刷新状态',
    );
    expect(controller.text, '5678');

    h.auth.gate!.complete();
    await tester.pumpAndSettle();

    expect(h.auth.captchaLoads, 1);
    expect(h.auth.captcha?.captchaId, 'captcha-1');
    expect(
      shownImage(tester),
      isNot((imageDuring as GfCaptchaImage).imageData),
    );
    expect(find.byType(GfProgressIndicator), findsNothing);
    // 新挑战到达后才作废旧输入。
    expect(controller.text, isEmpty);
  });

  testWidgets(
    'a failed refresh keeps the old image, the valid code and the retry target',
    (tester) async {
      final h = await pumpLogin(tester);
      final String before = shownImage(tester);
      await tester.enterText(input('Captcha'), '1234');
      h.auth.loadError = StateError('offline');

      await tester.tap(refreshTarget());
      await tester.pumpAndSettle();

      expect(h.auth.captchaLoads, 1);
      expect(find.text('Failed to load captcha'), findsOneWidget);
      // preservePhaseOnError 为 true 时验证码行(重试入口)保留。
      expect(find.byType(GfCaptchaImage), findsOneWidget);
      expect(shownImage(tester), before);
      // 旧图在服务端仍然有效,旧输入保留,可直接提交。
      expect(
        tester.widget<TextField>(input('Captcha')).controller!.text,
        '1234',
      );

      h.auth.loadError = null;
      await tester.tap(refreshTarget());
      await tester.pumpAndSettle();

      expect(h.auth.captchaLoads, 2);
      expect(find.text('Failed to load captcha'), findsNothing);
      expect(shownImage(tester), isNot(before));
      expect(
        tester.widget<TextField>(input('Captcha')).controller!.text,
        isEmpty,
      );
    },
  );

  testWidgets('submit is inert while a refresh is in flight', (tester) async {
    final h = await pumpLogin(tester);
    final l10n = AppLocalizations.of(tester.element(find.byType(LoginPage)));
    await tester.enterText(input('Captcha'), '1234');
    await tester.tap(input('Captcha'));
    await tester.pumpAndSettle();
    h.auth.gate = Completer<void>();

    await tester.tap(refreshTarget());
    await tester.pump();
    expect(tester.widget<GfButton>(submitButton()).onPressed, isNull);

    // 按钮与键盘提交路径都不能在刷新在途时发起登录请求。
    await tester.tap(submitButton());
    await tester.pump();
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();

    expect(h.auth.loginCalls, 0);
    expect(find.byType(GfStatusMessage), findsNothing);

    h.auth.gate!.complete();
    await tester.pumpAndSettle();
    expect(tester.widget<GfButton>(submitButton()).onPressed, isNotNull);
    expect(
      tester.widget<TextField>(input('Captcha')).controller!.text,
      isEmpty,
    );

    // 服务端已要求验证码:空验证码只做本地校验,不发请求、不消耗限流额度。
    await tester.tap(submitButton());
    await tester.pumpAndSettle();
    expect(h.auth.loginCalls, 0);
    expect(find.text(l10n.authCaptchaRequired), findsOneWidget);
  });

  testWidgets('refresh keeps the captcha field focused', (tester) async {
    final h = await pumpLogin(tester);
    await tester.tap(input('Captcha'));
    await tester.pumpAndSettle();
    final FocusNode node = tester
        .widget<TextField>(input('Captcha'))
        .focusNode!;
    expect(node.hasFocus, isTrue);
    expect(tester.testTextInput.isVisible, isTrue);

    await tester.tap(refreshTarget());
    await tester.pumpAndSettle();

    expect(h.auth.captchaLoads, 1);
    expect(node.hasFocus, isTrue);
    expect(tester.testTextInput.isVisible, isTrue);
  });

  testWidgets('leaving the page during a refresh does not leak state', (
    tester,
  ) async {
    final h = await pumpLogin(tester);
    h.auth.gate = Completer<void>();

    await tester.tap(refreshTarget());
    await tester.pump();
    expect(find.byType(GfProgressIndicator), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    h.auth.gate!.complete();
    await tester.pumpAndSettle();

    expect(find.byType(LoginPage), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('refresh target is a labeled button with a 44px target', (
    tester,
  ) async {
    final SemanticsHandle handle = tester.ensureSemantics();
    await pumpLogin(tester);
    final l10n = AppLocalizations.of(tester.element(find.byType(LoginPage)));

    expect(find.bySemanticsLabel(l10n.authRefreshCaptcha), findsOneWidget);
    final Size size = tester.getSize(refreshTarget());
    expect(size.width, greaterThanOrEqualTo(44));
    expect(size.height, greaterThanOrEqualTo(44));
    handle.dispose();
  });
}

class _Harness {
  _Harness({required this.auth});

  final _CaptchaAuth auth;
}

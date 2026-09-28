import 'dart:convert';

import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/apple/apple_sign_in_button.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/auth/sign_in_methods_sheet.dart';
import 'package:forum_app/src/providers.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'test_font_helpers.dart';
import 'pages_behavior_test.dart' show NoopCache;
import 'pages_smoke_test.dart' show MemoryTokenStorage;

const Key moreMethodsKey = Key('login-more-methods');
const Key captchaKey = Key('login-captcha');

final Finder _brand = find.byWidgetPredicate(
  (widget) => widget is Image && widget.semanticLabel == 'YourTJ',
);

/// Serves the `/login` page payload with a configurable provider set.
class _LoginOptions implements HttpClientAdapter {
  _LoginOptions({
    required this.tongji,
    required this.google,
    required this.github,
    required this.apple,
    required this.policies,
  });

  final bool tongji;
  final bool google;
  final bool github;
  final bool apple;
  final bool policies;

  @override
  void close({bool force = false}) {}

  @override
  Future<ResponseBody> fetch(
    RequestOptions request,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (request.path != '/login') {
      throw DioException(
        requestOptions: request,
        type: DioExceptionType.connectionError,
      );
    }
    return ResponseBody.fromString(
      jsonEncode({
        'component': 'auth.login',
        'props': {
          'initialMode': 'login',
          'redirectUrl': '/',
          'githubUrl': github ? '/api/auth/github' : '',
          'googleReady': google,
          'appleReady': apple,
          'tongjiReady': tongji,
          'tongjiUrl': tongji ? '/api/auth/tongji' : '',
          'allowedDomains': <String>[],
          'termsOfServiceEnabled': policies,
          'privacyPolicyEnabled': policies,
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

/// Serves a captcha immediately so the revealed row matches the loaded state.
class _CaptchaAuth extends AuthController {
  _CaptchaAuth(GfApiClient client, TokenStorage storage)
    : super(
        authRepository: AuthRepository(client),
        apiClient: client,
        tokenStorage: storage,
      );

  CaptchaPayload? _payload;

  @override
  CaptchaPayload? get captcha => _payload;

  @override
  Future<void> loadCaptcha({
    bool preservePhaseOnError = false,
    bool silentOnError = false,
  }) async {
    _payload = CaptchaPayload(
      captchaId: 'test-captcha',
      captchaImg:
          'data:image/png;base64,'
          '${base64Encode(img.encodePng(img.Image(width: 60, height: 24)))}',
    );
    notifyListeners();
  }
}

class _Harness {
  _Harness(this.tester, this.container, this.l10n);

  final WidgetTester tester;
  final ProviderContainer container;
  final AppLocalizations l10n;

  Finder get control => find.byKey(moreMethodsKey);

  Finder get captcha => find.byKey(captchaKey);

  Rect get controlRect => tester.getRect(control);

  Rect get viewportRect =>
      tester.getRect(find.byType(SingleChildScrollView).first);

  ScrollPosition get scroll => tester
      .state<ScrollableState>(
        find
            .descendant(
              of: find.byType(SingleChildScrollView).first,
              matching: find.byType(Scrollable),
            )
            .first,
      )
      .position;

  double get scrollExtent => scroll.maxScrollExtent;

  double get contentHeight => scroll.maxScrollExtent + scroll.viewportDimension;

  Finder input(String label) => find.byWidgetPredicate(
    (widget) => widget is TextField && widget.decoration?.labelText == label,
  );

  /// Types a password and taps outside its region, which is what reveals the
  /// captcha for every user who fills the password field.
  Future<void> revealCaptcha() async {
    await tester.enterText(input(l10n.authPassword), 'secret');
    await tester.pumpAndSettle();
    await tester.tapAt(const Offset(2, 300));
    await tester.pumpAndSettle();
  }

  Future<void> openControl() async {
    await tester.ensureVisible(control);
    await tester.pumpAndSettle();
    await tester.tap(control);
    await tester.pumpAndSettle();
  }
}

Future<_Harness> _pumpLogin(
  WidgetTester tester, {
  Size size = const Size(390, 844),
  double topInset = 0,
  double bottomInset = 0,
  bool deviceFonts = false,
  bool tongji = true,
  bool google = true,
  bool github = true,
  bool apple = false,
  bool policies = true,
  Locale locale = const Locale('en'),
  double textScale = 1,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  tester.view.padding = FakeViewPadding(top: topInset, bottom: bottomInset);
  tester.view.viewPadding = tester.view.padding;
  addTearDown(tester.view.reset);
  if (deviceFonts) await loadTestFonts(tester);
  final storage = MemoryTokenStorage();
  final staged = MemoryTokenStorage();
  final client = GfApiClient(
    dio: Dio(),
    tokenStorage: storage,
    baseUrl: 'http://fake.local',
  );
  final adapter = _LoginOptions(
    tongji: tongji,
    google: google,
    github: github,
    apple: apple,
    policies: policies,
  );
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
        locale: locale,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(textScale)),
          child: child!,
        ),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: LoginPage(
          // A fresh page per pump: the state owns the loaded login options.
          key: UniqueKey(),
          authController: _CaptchaAuth(client, staged),
          authTokenStorage: staged,
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return _Harness(
    tester,
    container,
    AppLocalizations.of(tester.element(find.byType(LoginPage))),
  );
}

void main() {
  // Real device metrics: bundled Roboto, safe-area insets and the captcha row
  // every password user sees. The control must be reachable without scrolling.
  for (final (String label, Size size, double top, double bottom) in [
    ('a 390x844 phone with iPhone insets', Size(390, 844), 47.0, 34.0),
    ('an iPhone SE 375x667', Size(375, 667), 20.0, 0.0),
    ('a 360x640 phone', Size(360, 640), 24.0, 0.0),
  ]) {
    for (final language in ['zh', 'en', 'ja', 'de']) {
      testWidgets('$label keeps the $language sign-in control reachable', (
        tester,
      ) async {
        final h = await _pumpLogin(
          tester,
          size: size,
          topInset: top,
          bottomInset: bottom,
          deviceFonts: true,
          locale: Locale(language),
        );
        expect(tester.takeException(), isNull);
        await h.revealCaptcha();
        expect(h.captcha, findsOneWidget);

        expect(h.control, findsOneWidget);
        expect(h.controlRect.height, greaterThanOrEqualTo(44));
        expect(h.controlRect.bottom, lessThanOrEqualTo(h.viewportRect.bottom));
        expect(h.control.hitTestable(), findsOneWidget);
        // Only German on the smallest surface keeps a few pixels of card
        // padding below the fold; everything the user acts on is above it.
        final double residual = size == const Size(360, 640) && language == 'de'
            ? 16
            : 0;
        expect(h.scrollExtent, lessThanOrEqualTo(residual));
        expect(tester.takeException(), isNull);
      });
    }
  }

  testWidgets('the synthetic test font only leaves German short of the fold', (
    tester,
  ) async {
    // FlutterTest glyphs are one em wide, roughly twice the bundled Roboto.
    // German wraps to a three-line title and a four-line subtitle there, 78 px
    // more than English; closing that gap would need four-pixel field gaps.
    for (final language in ['zh', 'en', 'ja', 'de']) {
      final h = await _pumpLogin(
        tester,
        topInset: 47,
        bottomInset: 34,
        locale: Locale(language),
      );
      await h.revealCaptcha();
      expect(tester.takeException(), isNull);
      expect(h.control, findsOneWidget);
      if (language == 'de') {
        expect(h.scrollExtent, lessThanOrEqualTo(56));
      } else {
        expect(h.scrollExtent, 0);
        expect(h.control.hitTestable(), findsOneWidget);
      }
    }
  });

  testWidgets('a short form area drops the decorative header', (tester) async {
    final short = await _pumpLogin(
      tester,
      size: const Size(375, 667),
      topInset: 20,
      deviceFonts: true,
    );
    expect(_brand, findsNothing);
    expect(find.text(short.l10n.authLoginSubtitle), findsNothing);
    expect(find.text(short.l10n.authLoginTitle), findsWidgets);

    final tall = await _pumpLogin(
      tester,
      topInset: 47,
      bottomInset: 34,
      deviceFonts: true,
    );
    expect(_brand, findsOneWidget);
    expect(find.text(tall.l10n.authLoginSubtitle), findsOneWidget);

    // The keyboard shrinks the same area, so the header yields to the form.
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    await tester.pumpAndSettle();
    expect(_brand, findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('login and registration share a centered brand', (tester) async {
    final h = await _pumpLogin(
      tester,
      topInset: 47,
      bottomInset: 34,
      deviceFonts: true,
    );
    final Image loginBrand = tester.widget<Image>(_brand);
    final Rect loginBrandRect = tester.getRect(_brand);
    expect(loginBrand.width, 192);
    expect(loginBrand.height, 44);
    expect(
      loginBrandRect.center.dx,
      closeTo(h.viewportRect.center.dx - 2, 0.5),
    );

    await tester.tap(find.text(h.l10n.loginModeRegister).first);
    await tester.pumpAndSettle();

    final Image registrationBrand = tester.widget<Image>(_brand);
    final Rect brandRect = tester.getRect(_brand);
    expect(registrationBrand.width, 192);
    expect(registrationBrand.height, 44);
    expect(brandRect.center.dx, closeTo(loginBrandRect.center.dx, 0.5));
    expect(
      brandRect.center.dy,
      lessThan(tester.view.physicalSize.height * 0.3),
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('the control opens one sheet with every available provider', (
    tester,
  ) async {
    final h = await _pumpLogin(tester, apple: true);
    expect(
      tester.widget<OutlinedButton>(h.control).focusNode?.hasFocus,
      isFalse,
    );

    await tester.tap(h.control);
    await tester.pumpAndSettle();

    expect(find.byType(SignInMethodsSheet), findsOneWidget);
    expect(find.text(h.l10n.authSignInMethods), findsWidgets);
    expect(find.text(h.l10n.loginTongji), findsOneWidget);
    expect(find.text(h.l10n.loginGoogle), findsOneWidget);
    expect(find.text(h.l10n.loginGithub), findsOneWidget);
    expect(find.text(h.l10n.loginTongjiHint), findsOneWidget);
    expect(find.text(h.l10n.loginTongjiPolicies), findsOneWidget);
    // Apple stays gated to native iOS sign-in.
    expect(
      find.byType(AppleSignInButton),
      defaultTargetPlatform == TargetPlatform.iOS
          ? findsOneWidget
          : findsNothing,
    );
    for (final label in [
      h.l10n.loginTongji,
      h.l10n.loginGoogle,
      h.l10n.loginGithub,
    ]) {
      final target = find.ancestor(
        of: find.text(label),
        matching: find.byType(OutlinedButton),
      );
      expect(target, findsOneWidget);
      final size = tester.getSize(target);
      expect(size.height, greaterThanOrEqualTo(44));
      expect(size.width, greaterThanOrEqualTo(44));
    }
    // Focus leaves the page control and enters the sheet.
    expect(
      tester.widget<OutlinedButton>(h.control).focusNode?.hasFocus,
      isFalse,
    );
    final sheetContext = tester.element(find.byType(SignInMethodsSheet));
    expect(FocusScope.of(sheetContext).hasFocus, isTrue);
  });

  testWidgets('dismissing the sheet restores the form and the control focus', (
    tester,
  ) async {
    final h = await _pumpLogin(tester);
    await h.openControl();
    expect(find.byType(SignInMethodsSheet), findsOneWidget);

    await tester.tapAt(const Offset(195, 40));
    await tester.pumpAndSettle();

    expect(find.byType(SignInMethodsSheet), findsNothing);
    expect(find.text(h.l10n.loginGoogle), findsNothing);
    expect(find.text(h.l10n.authPassword), findsOneWidget);
    expect(h.control, findsOneWidget);
    expect(
      tester.widget<OutlinedButton>(h.control).focusNode?.hasFocus,
      isTrue,
    );
  });

  testWidgets('Escape dismisses the sheet', (tester) async {
    final h = await _pumpLogin(tester);
    await h.openControl();
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.byType(SignInMethodsSheet), findsNothing);
  });

  testWidgets('an unconfigured provider never appears', (tester) async {
    final none = await _pumpLogin(
      tester,
      tongji: false,
      google: false,
      github: false,
    );
    expect(none.control, findsNothing);

    final googleOnly = await _pumpLogin(
      tester,
      tongji: false,
      google: true,
      github: false,
    );
    expect(googleOnly.control, findsOneWidget);
    await googleOnly.openControl();
    expect(find.text(googleOnly.l10n.loginGoogle), findsOneWidget);
    expect(find.text(googleOnly.l10n.loginTongji), findsNothing);
    expect(find.text(googleOnly.l10n.loginGithub), findsNothing);
    expect(find.text(googleOnly.l10n.loginTongjiHint), findsNothing);
  });

  testWidgets('a small phone with large text still reaches every provider', (
    tester,
  ) async {
    final h = await _pumpLogin(
      tester,
      size: const Size(320, 568),
      textScale: 2,
    );
    await h.openControl();
    expect(find.byType(SignInMethodsSheet), findsOneWidget);
    expect(find.text(h.l10n.loginTongji), findsOneWidget);
    expect(find.text(h.l10n.loginGoogle), findsOneWidget);
    expect(find.text(h.l10n.loginGithub), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('the folded control and its sheet are announced', (tester) async {
    final SemanticsHandle handle = tester.ensureSemantics();
    final h = await _pumpLogin(tester);

    final SemanticsNode control = tester.semantics.find(h.control);
    expect(control.flagsCollection.isButton, isTrue);
    expect(control.label, contains(h.l10n.authMoreSignInMethods));

    await h.openControl();
    expect(
      tester.semantics.find(find.text(h.l10n.authSignInMethods)).label,
      contains(h.l10n.authSignInMethods),
    );
    expect(find.bySemanticsLabel(h.l10n.loginTongji), findsOneWidget);
    expect(find.bySemanticsLabel(h.l10n.loginGoogle), findsOneWidget);
    handle.dispose();
  });

  testWidgets('choosing a provider closes the sheet with that method', (
    tester,
  ) async {
    SignInMethod? selected;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) => Scaffold(
            body: Center(
              child: TextButton(
                onPressed: () async {
                  selected = await showGfBottomSheet<SignInMethod>(
                    context,
                    builder: (_) => const SignInMethodsSheet(
                      methods: [
                        SignInMethod.tongji,
                        SignInMethod.google,
                        SignInMethod.github,
                      ],
                      termsOfServiceEnabled: false,
                      privacyPolicyEnabled: false,
                    ),
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Continue with Google'));
    await tester.pumpAndSettle();
    expect(selected, SignInMethod.google);
    expect(find.byType(SignInMethodsSheet), findsNothing);
  });

  group('availableSignInMethods', () {
    LoginPageProps options({
      bool tongji = false,
      bool google = false,
      bool github = false,
      bool apple = false,
    }) => LoginPageProps(
      initialMode: 'login',
      redirectUrl: '/',
      githubUrl: github ? '/api/auth/github' : '',
      googleReady: google,
      appleReady: apple,
      tongjiReady: tongji,
    );

    test('login lists every published provider in display order', () {
      expect(
        availableSignInMethods(
          options(tongji: true, google: true, github: true, apple: true),
          loginMode: true,
          nativeAppleAvailable: true,
        ),
        [
          SignInMethod.apple,
          SignInMethod.tongji,
          SignInMethod.google,
          SignInMethod.github,
        ],
      );
    });

    test('Apple needs a configured site and native iOS sign-in', () {
      expect(
        availableSignInMethods(
          options(apple: true),
          loginMode: true,
          nativeAppleAvailable: false,
        ),
        isEmpty,
      );
      expect(
        availableSignInMethods(
          options(apple: true),
          loginMode: true,
          nativeAppleAvailable: true,
        ),
        [SignInMethod.apple],
      );
    });

    test('registration keeps only Tongji', () {
      expect(
        availableSignInMethods(
          options(tongji: true, google: true, github: true, apple: true),
          loginMode: false,
          nativeAppleAvailable: true,
        ),
        [SignInMethod.tongji],
      );
    });
  });
}

import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/pages/campus/campus_connection.dart';
import 'package:forum_app/src/pages/campus/campus_helpers.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/server_messages.dart';
import 'package:forum_app/src/widgets/activation_email_action.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/campus_fixtures.dart';

const _gate = ApiException(
  fallbackMessage: '',
  messageCode: 'permission.emailRequired',
  statusCode: 403,
);
const _sent = GfResponse<Object?>(
  code: 0,
  messageCode: 'auth.activation.resendSuccess',
);
GfApiClient _client() =>
    GfApiClient(dio: Dio(), tokenStorage: CampusMemoryTokenStorage());

class _Auth extends AuthRepository {
  _Auth() : super(_client());
  int calls = 0;
  ApiException? error;
  CancelToken? cancel;
  Completer<GfResponse<Object?>>? pending;
  @override
  Future<GfResponse<Object?>> resendActivationEmail({
    CancelToken? cancelToken,
  }) async {
    calls++;
    cancel = cancelToken;
    if (error != null) throw error!;
    return pending?.future ?? _sent;
  }
}

class _Campus extends FakeCampusRepository {
  bool pendingActivation = true;
  @override
  Future<Uri> start(String mode, {CancelToken? cancelToken}) async =>
      throw _gate;
  @override
  Future<void> confirm({CancelToken? cancelToken}) async {
    if (pendingActivation) throw _gate;
    await super.confirm(cancelToken: cancelToken);
  }
}

class _Controller extends CampusController {
  _Controller(super.repository, CampusStatus status) {
    tab = 'connection';
    state = CampusViewState(status: status, loading: false);
  }
}

class _Users extends UserRepository {
  _Users() : super(_client());
  @override
  Future<List<UserSessionPayload>> listSessions() async => [];
}

Widget _app(
  ProviderContainer container,
  Widget child, {
  String locale = 'zh',
}) => UncontrolledProviderScope(
  container: container,
  child: MaterialApp(
    theme: gfThemeData(Brightness.light),
    locale: Locale(locale),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: SingleChildScrollView(child: child)),
  ),
);
void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  test('campus email gate tells the user to activate their account', () {
    expect(campusError(AppLocalizationsZh(), _gate), '请先到邮箱激活账号');
  });
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    testWidgets('resend feedback in $locale at narrow width and large text', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 844);
      tester.view.devicePixelRatio = 1;
      tester.platformDispatcher.textScaleFactorTestValue = 2;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      final auth = _Auth();
      final container = ProviderContainer(
        overrides: [authRepositoryProvider.overrideWithValue(auth)],
      );
      addTearDown(container.dispose);
      await tester.pumpWidget(
        _app(container, const ActivationEmailAction(), locale: locale),
      );
      await tester.pumpAndSettle();
      final l = AppLocalizations.of(
        tester.element(find.byType(ActivationEmailAction)),
      );
      for (final error in <ApiException?>[
        null,
        const ApiException(
          fallbackMessage: '',
          messageCode: 'auth.activation.resendCooldown',
          params: {'retryAfterSeconds': 58},
        ),
        const ApiException(
          fallbackMessage: '',
          messageCode: 'auth.activation.resendDaily',
          params: {'limit': 3},
        ),
        const ApiException(
          fallbackMessage: '',
          messageCode: 'auth.activation.resendFailed',
        ),
        const ApiException(
          fallbackMessage: '',
          messageCode: 'auth.activation.alreadyVerified',
        ),
        const ApiException(
          fallbackMessage: '',
          messageCode: 'auth.activation.disabled',
        ),
      ]) {
        auth.error = error;
        await tester.tap(find.text(l.authResendActivationEmail));
        await tester.pumpAndSettle();
        expect(
          find.text(
            resolveErrorMessage(
              l,
              error ??
                  const ApiException(
                    fallbackMessage: '',
                    messageCode: 'auth.activation.resendSuccess',
                  ),
            ),
          ),
          findsOneWidget,
        );
        expect(tester.takeException(), isNull);
      }
      expect(auth.calls, 6);
    });
  }
  testWidgets(
    'resend serializes, cancels on leaving and drops old-session feedback',
    (tester) async {
      final auth = _Auth()..pending = Completer<GfResponse<Object?>>();
      final old = auth.pending!;
      final container = ProviderContainer(
        overrides: [authRepositoryProvider.overrideWithValue(auth)],
      );
      addTearDown(container.dispose);
      await tester.pumpWidget(_app(container, const ActivationEmailAction()));
      await tester.pumpAndSettle();
      await tester.tap(find.text('重发验证邮件'));
      await tester.pump();
      await tester.tap(find.text('重发验证邮件'));
      expect(auth.calls, 1);
      final cancel = auth.cancel!;
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      expect(cancel.isCancelled, isTrue);
      auth.pending = null;
      old.complete(_sent);
      await tester.pumpAndSettle();
      expect(find.text('验证邮件已重新发送，请查收邮箱'), findsNothing);
      await tester.tap(find.text('重发验证邮件'));
      await tester.pumpAndSettle();
      expect(auth.calls, 2);
      expect(find.text('验证邮件已重新发送，请查收邮箱'), findsOneWidget);
      auth.pending = Completer<GfResponse<Object?>>();
      await tester.tap(find.text('重发验证邮件'));
      await tester.pump();
      final leaving = auth.cancel!;
      await tester.pumpWidget(const SizedBox());
      expect(leaving.isCancelled, isTrue);
      auth.pending!.complete(_sent);
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
    },
  );
  for (final candidate in [false, true]) {
    testWidgets(
      'campus ${candidate ? 'confirm' : 'start'} gate offers resend',
      (tester) async {
        final campus = _Campus();
        final auth = _Auth();
        final status = CampusStatus(
          enabled: true,
          binding: candidate ? testBinding : null,
          candidate: candidate
              ? const CampusCandidate(
                  maskedId: 'DE••••MO',
                  mode: 'replace',
                  expiresAt: '2026-10-10T00:00:00Z',
                )
              : null,
        );
        final container = ProviderContainer(
          overrides: [
            authRepositoryProvider.overrideWithValue(auth),
            campusRepositoryProvider.overrideWithValue(campus),
            campusControllerProvider.overrideWith(
              (ref) => _Controller(campus, status),
            ),
          ],
        );
        await tester.pumpWidget(_app(container, const CampusConnection()));
        await tester.pumpAndSettle();
        final l = AppLocalizations.of(
          tester.element(find.byType(CampusConnection)),
        );
        await tester.tap(
          find.text(candidate ? l.campusConfirmBinding : l.campusBind),
        );
        await tester.pumpAndSettle();
        expect(find.text(l.authActivationRequired), findsOneWidget);
        await tester.ensureVisible(find.text(l.authResendActivationEmail));
        await tester.tap(find.text(l.authResendActivationEmail));
        await tester.pumpAndSettle();
        expect(auth.calls, 1);
        if (candidate) {
          campus.pendingActivation = false;
          await tester.ensureVisible(find.text(l.campusConfirmBinding));
          await tester.tap(find.text(l.campusConfirmBinding));
          await tester.pumpAndSettle();
          expect(find.text(l.authActivationRequired), findsNothing);
          expect(find.text(l.campusUnbind), findsOneWidget);
        }
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        container.dispose();
        await tester.pump();
      },
    );
  }
  testWidgets('account security exposes resend to signed-in users', (
    tester,
  ) async {
    final tokens = CampusMemoryTokenStorage()..value = 'test-session';
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(tokens),
        authRepositoryProvider.overrideWithValue(_Auth()),
        userRepositoryProvider.overrideWithValue(_Users()),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const SettingsPage(initialSection: 'security'),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(ActivationEmailAction), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}

import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/campus/campus_connection.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/server_messages.dart';
import 'package:forum_app/src/widgets/activation_email_action.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import '../test/fixtures/campus_fixtures.dart';

class _ConnectionController extends CampusController {
  _ConnectionController(super.repository, CampusStatus status) {
    state = CampusViewState(status: status, loading: false);
  }

  @override
  Future<void> refresh({bool reuseCache = false}) async {
    state = CampusViewState(status: await repository.status(), loading: false);
  }
}

/// Real widgets and HTTP transport, with synthetic responses confined to loopback.
/// No email is sent, school authorization opened or remote account accessed.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('native email activation feedback and campus recovery', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final requests = <String>[];
    var resendCode = 'auth.activation.resendSuccess';
    var activated = false;
    var candidate = false;
    var bound = true;
    server.listen((request) async {
      requests.add('${request.method} ${request.uri.path}');
      expectSync(
        request.headers.value(HttpHeaders.authorizationHeader),
        'Bearer synthetic-session',
      );
      request.response.headers.contentType = ContentType.json;
      Object? result;
      String? error;
      switch (request.uri.path) {
        case '/api/resend-activation-email':
          if (resendCode != 'auth.activation.resendSuccess') error = resendCode;
        case '/api/user/sessions':
          result = [];
        case '/api/campus/tongji/start':
          error = 'permission.emailRequired';
        case '/api/campus/tongji/confirm':
          if (!activated) {
            error = 'permission.emailRequired';
          } else {
            candidate = false;
          }
        case '/api/campus/tongji/unbind':
          bound = false;
        case '/api/campus/status':
          result = {
            'enabled': true,
            'binding': bound
                ? {
                    'maskedId': 'DE••••MO',
                    'revision': 'demo-revision',
                    'boundAt': '2026-10-10T00:00:00Z',
                    'needsAuthorization': false,
                  }
                : null,
            'candidate': candidate
                ? {
                    'maskedId': 'DE••••MO',
                    'mode': 'replace',
                    'expiresAt': '2026-10-10T00:00:00Z',
                  }
                : null,
          };
        default:
          throw StateError('Unexpected request: ${request.uri}');
      }
      if (error == 'permission.emailRequired') {
        request.response.statusCode = 403;
      }
      request.response.write(
        jsonEncode({
          'code': error == null ? 0 : 1,
          'messageCode':
              error ??
              (request.uri.path == '/api/resend-activation-email'
                  ? resendCode
                  : null),
          'params': {
            if (error == 'auth.activation.resendCooldown')
              'retryAfterSeconds': 58,
            if (error == 'auth.activation.resendDaily') 'limit': 3,
          },
          'result': result,
        }),
      );
      await request.response.close();
    });
    final tokens = CampusMemoryTokenStorage()..value = 'synthetic-session';
    final dio = Dio();
    final client = GfApiClient(
      dio: dio,
      tokenStorage: tokens,
      baseUrl: 'http://127.0.0.1:${server.port}',
    );
    ProviderContainer? container;
    Future<void> show(
      Widget page,
      String locale,
      Brightness brightness, {
      double scale = 1,
      double? width,
      CampusStatus? campusStatus,
    }) async {
      await tester.pumpWidget(const SizedBox());
      container?.dispose();
      container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(client),
          tokenStorageProvider.overrideWithValue(tokens),
          if (campusStatus != null)
            campusControllerProvider.overrideWith(
              (ref) =>
                  _ConnectionController(CampusRepository(client), campusStatus),
            ),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container!,
          child: MaterialApp(
            theme: gfThemeData(brightness),
            locale: Locale(locale),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(textScaler: TextScaler.linear(scale)),
              child: Center(
                child: SizedBox(width: width, child: child),
              ),
            ),
            home: page,
          ),
        ),
      );
      await tester.pumpAndSettle();
    }

    Future<void> capture(String name) async {
      expect(tester.takeException(), isNull);
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot(name);
      }
    }

    try {
      for (final brightness in Brightness.values) {
        await show(
          const SettingsPage(initialSection: 'security'),
          'zh',
          brightness,
        );
        final l = AppLocalizations.of(
          tester.element(find.byType(SettingsPage)),
        );
        expect(find.byType(ActivationEmailAction), findsOneWidget);
        await capture('email-security-${brightness.name}');
        for (final code in [
          'auth.activation.resendSuccess',
          'auth.activation.resendCooldown',
          'auth.activation.resendDaily',
          'auth.activation.resendFailed',
        ]) {
          resendCode = code;
          await tester.ensureVisible(find.text(l.authResendActivationEmail));
          await tester.tap(find.text(l.authResendActivationEmail));
          await tester.pumpAndSettle();
          final message = resolveErrorMessage(
            l,
            ApiException(
              fallbackMessage: '',
              messageCode: code,
              params: const {'retryAfterSeconds': 58, 'limit': 3},
            ),
          );
          expect(find.text(message), findsOneWidget);
          await capture('email-${code.split('.').last}-${brightness.name}');
        }
        resendCode = 'auth.activation.resendSuccess';
        await show(
          Scaffold(
            appBar: GfAppBar(title: Text(l.campusConnection)),
            body: const SingleChildScrollView(
              padding: EdgeInsets.all(16),
              child: CampusConnection(),
            ),
          ),
          'zh',
          brightness,
          campusStatus: const CampusStatus(
            enabled: true,
            binding: testBinding,
            candidate: null,
          ),
        );
        expect(find.byType(ActivationEmailAction), findsNothing);
        await tester.tap(find.text(l.campusReauthorize));
        await tester.pumpAndSettle();
        expect(find.text(l.authActivationRequired), findsOneWidget);
        expect(find.text(l.campusUnbind), findsOneWidget);
        await capture('campus-email-gate-${brightness.name}');
        await tester.ensureVisible(find.text(l.campusUnbind));
        await tester.tap(find.text(l.campusUnbind));
        await tester.pumpAndSettle();
        await tester.tap(find.widgetWithText(TextButton, l.campusUnbind));
        await tester.pumpAndSettle();
        expect(bound, isFalse);
        expect(find.text(l.campusBind), findsOneWidget);
        bound = true;
      }
      for (final locale in ['zh', 'en', 'ja', 'de']) {
        for (final brightness in Brightness.values) {
          resendCode = 'auth.activation.resendCooldown';
          await show(
            const SettingsPage(initialSection: 'security'),
            locale,
            brightness,
            scale: 2,
            width: 320,
          );
          final l = AppLocalizations.of(
            tester.element(find.byType(SettingsPage)),
          );
          await tester.tap(find.text(l.authResendActivationEmail));
          await tester.pumpAndSettle();
          expect(
            find.text(
              resolveErrorMessage(
                l,
                const ApiException(
                  fallbackMessage: '',
                  messageCode: 'auth.activation.resendCooldown',
                  params: {'retryAfterSeconds': 58},
                ),
              ),
            ),
            findsOneWidget,
          );
          await capture('email-narrow-$locale-${brightness.name}');
        }
      }
      candidate = true;
      activated = false;
      await show(
        const Scaffold(
          body: SingleChildScrollView(
            padding: EdgeInsets.all(16),
            child: CampusConnection(),
          ),
        ),
        'zh',
        Brightness.dark,
        campusStatus: const CampusStatus(
          enabled: true,
          binding: testBinding,
          candidate: CampusCandidate(
            maskedId: 'DE••••MO',
            mode: 'replace',
            expiresAt: '2026-10-10T00:00:00Z',
          ),
        ),
      );
      final l = AppLocalizations.of(
        tester.element(find.byType(CampusConnection)),
      );
      await tester.tap(find.text(l.campusConfirmBinding));
      await tester.pumpAndSettle();
      expect(find.text(l.authActivationRequired), findsOneWidget);
      await capture('campus-confirm-email-gate');
      activated = true;
      await tester.tap(find.text(l.campusConfirmBinding));
      await tester.pumpAndSettle();
      expect(find.text(l.authActivationRequired), findsNothing);
      expect(find.byType(ActivationEmailAction), findsNothing);
      expect(find.text(l.campusUnbind), findsOneWidget);
      await capture('campus-confirm-recovered');
      expect(
        requests.where(
          (request) => request == 'POST /api/resend-activation-email',
        ),
        hasLength(16),
      );
      expect(
        requests.where(
          (request) => request == 'POST /api/campus/tongji/unbind',
        ),
        hasLength(2),
      );
      expect(
        requests.where(
          (request) => request == 'POST /api/campus/tongji/confirm',
        ),
        hasLength(2),
      );
      debugPrint(
        'Email activation acceptance: ${requests.length} authenticated loopback requests, both themes, four locales, narrow 200% text, gate/recovery/unbind.',
      );
    } finally {
      await tester.pumpWidget(const SizedBox());
      container?.dispose();
      dio.close(force: true);
      await server.close(force: true);
    }
  });
}

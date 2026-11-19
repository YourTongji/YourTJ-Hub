import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show NoopCache;
import 'pages_smoke_test.dart' show MemoryTokenStorage;
import 'test_font_helpers.dart';

class _LoginAdapter implements HttpClientAdapter {
  bool failStatus = true;
  bool enabled = true;
  int statusReads = 0;
  Future<void>? statusGate;

  @override
  void close({bool force = false}) {}

  @override
  Future<ResponseBody> fetch(
    RequestOptions request,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    Object response;
    if (request.path == '/api/auth/passkey/status') {
      statusReads++;
      await statusGate;
      if (failStatus) throw StateError('status unavailable');
      response = {
        'code': 0,
        'result': {'enabled': enabled},
      };
    } else if (request.path == '/login') {
      response = {
        'component': 'auth.login',
        'props': {
          'initialMode': 'login',
          'redirectUrl': '/',
          'githubUrl': '/api/auth/github',
          'githubReady': true,
          'googleReady': false,
          'allowedDomains': <String>[],
          'termsOfServiceEnabled': false,
          'privacyPolicyEnabled': false,
        },
        'layout': minimalLayoutJson(),
        'url': '/login',
        'version': '1',
        'meta': {'title': 'Login'},
      };
    } else {
      throw StateError('unexpected request: ${request.path}');
    }
    return ResponseBody.fromString(
      jsonEncode(response),
      200,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
      },
    );
  }
}

Future<void> _mount(
  WidgetTester tester,
  _LoginAdapter adapter, {
  Future<bool> Function()? probe,
  String locale = 'en',
  Brightness brightness = Brightness.light,
}) async {
  final storage = MemoryTokenStorage();
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
  final theme = gfThemeData(brightness);
  await tester.pumpWidget(
    RepaintBoundary(
      key: const Key('passkey-proof'),
      child: UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: theme.copyWith(
            textTheme: theme.textTheme.apply(fontFamily: 'Roboto'),
          ),
          locale: Locale(locale),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: LoginPage(
            authTokenStorage: storage,
            passkeyAvailability: probe,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  for (final locale in ['en', 'zh', 'ja', 'de']) {
    testWidgets(
      'status failure is localized and keeps other sign-in methods: $locale',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = Size(
          locale == 'en' || locale == 'zh' ? 390 : 1280,
          1100,
        );
        addTearDown(tester.view.reset);
        await loadTestFonts(tester);
        final adapter = _LoginAdapter();
        await _mount(
          tester,
          adapter,
          locale: locale,
          brightness: locale == 'zh' || locale == 'de'
              ? Brightness.dark
              : Brightness.light,
        );
        final l10n = await AppLocalizations.delegate.load(Locale(locale));
        expect(find.text(l10n.authPasskeyAvailabilityFailed), findsOneWidget);
        expect(
          find.byKey(const Key('login-passkey-status-retry')),
          findsOneWidget,
        );
        expect(find.byKey(const Key('login-passkey')), findsNothing);
        expect(find.byKey(const Key('login-more-methods')), findsOneWidget);
        expect(
          tester
              .widgetList<GfButton>(find.byType(GfButton))
              .firstWhere((b) => b.variant == GfButtonVariant.primary)
              .onPressed,
          isNotNull,
        );
        expect(adapter.statusReads, 1);
        final output = Platform.environment['YOURTJ_TEST_SCREENSHOT_DIR'];
        if (output != null) {
          await tester.runAsync(() async {
            final image = await tester
                .renderObject<RenderRepaintBoundary>(
                  find.byKey(const Key('passkey-proof')),
                )
                .toImage();
            final bytes = await image.toByteData(
              format: ui.ImageByteFormat.png,
            );
            await Directory(output).create(recursive: true);
            await File(
              '$output/passkey-status-$locale.png',
            ).writeAsBytes(bytes!.buffer.asUint8List());
            image.dispose();
          });
        }
      },
    );
  }

  testWidgets('retry recovers availability and cannot start duplicate probes', (
    tester,
  ) async {
    final adapter = _LoginAdapter();
    var probes = 0;
    await _mount(
      tester,
      adapter,
      probe: () async {
        probes++;
        return true;
      },
    );
    final gate = Completer<void>();
    adapter.failStatus = false;
    adapter.statusGate = gate.future;
    final retry = find.byKey(const Key('login-passkey-status-retry'));
    await tester.ensureVisible(retry);
    await tester.tap(retry);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    expect(adapter.statusReads, 2);
    expect(tester.widget<TextButton>(retry).onPressed, isNull);
    gate.complete();
    await tester.pumpAndSettle();
    expect(probes, 1);
    expect(find.byKey(const Key('login-passkey')), findsOneWidget);
    expect(retry, findsNothing);
  });

  testWidgets('device probe errors are retryable rather than unsupported', (
    tester,
  ) async {
    final adapter = _LoginAdapter()..failStatus = false;
    var probes = 0;
    await _mount(
      tester,
      adapter,
      probe: () async {
        if (++probes == 1) throw StateError('platform probe failed');
        return true;
      },
    );
    final retry = find.byKey(const Key('login-passkey-status-retry'));
    expect(retry, findsOneWidget);
    await tester.ensureVisible(retry);
    await tester.tap(retry);
    await tester.pumpAndSettle();
    expect(probes, 2);
    expect(find.byKey(const Key('login-passkey')), findsOneWidget);
    expect(retry, findsNothing);
  });

  for (final enabled in [false, true]) {
    testWidgets(
      'disabled or unsupported is not reported as an error (enabled=$enabled)',
      (tester) async {
        final adapter = _LoginAdapter()
          ..failStatus = false
          ..enabled = enabled;
        var probes = 0;
        await _mount(
          tester,
          adapter,
          probe: () async {
            probes++;
            return false;
          },
        );
        expect(probes, enabled ? 1 : 0);
        expect(
          find.byKey(const Key('login-passkey-status-retry')),
          findsNothing,
        );
        expect(find.byKey(const Key('login-passkey')), findsNothing);
      },
    );
  }

  testWidgets('late availability result does not update a disposed page', (
    tester,
  ) async {
    final gate = Completer<void>();
    final adapter = _LoginAdapter()
      ..failStatus = false
      ..statusGate = gate.future;
    await _mount(tester, adapter, probe: () async => true);
    await tester.pumpWidget(const SizedBox());
    gate.complete();
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'failed probe keeps retry and other methods reachable on a small phone',
    (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(375, 667);
      addTearDown(tester.view.reset);
      await loadTestFonts(tester);
      await _mount(tester, _LoginAdapter());
      for (final key in ['login-passkey-status-retry', 'login-more-methods']) {
        final control = find.byKey(Key(key));
        await tester.ensureVisible(control);
        await tester.pumpAndSettle();
        expect(control.hitTestable(), findsOneWidget);
        expect(tester.getRect(control).bottom, lessThanOrEqualTo(667));
      }
      expect(tester.takeException(), isNull);
    },
  );
}

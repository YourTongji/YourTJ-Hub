import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart'
    show debugDefaultTargetPlatformOverride;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/apple/apple_sign_in_button.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import '../fixtures/page_fixtures.dart';
import '../pages_behavior_test.dart' show NoopCache;
import '../pages_smoke_test.dart' show MemoryTokenStorage;

/// The login page payload with every provider available.
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
          'googleReady': true,
          'appleReady': true,
          'tongjiReady': true,
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

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const platformViews = MethodChannel('flutter/platform_views');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  setUp(() {
    messenger.setMockMethodCallHandler(platformViews, (call) async => null);
  });
  tearDown(() => messenger.setMockMethodCallHandler(platformViews, null));

  testWidgets(
    'the login page hands the native button the provider row geometry',
    (tester) async {
      // The framework verifies debug overrides before tear-downs run, so the
      // platform override is always cleared inside the test body.
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      try {
        await _expectLoginRowGeometry(tester, messenger, platformViews);
      } finally {
        debugDefaultTargetPlatformOverride = null;
      }
    },
  );
}

Future<void> _expectLoginRowGeometry(
  WidgetTester tester,
  TestDefaultBinaryMessenger messenger,
  MethodChannel platformViews,
) async {
  await tester.binding.setSurfaceSize(const Size(390, 1100));
  addTearDown(() => tester.binding.setSurfaceSize(null));

  final creations = <Map<Object?, Object?>>[];
  messenger.setMockMethodCallHandler(platformViews, (call) async {
    if (call.method == 'create') {
      final args = call.arguments as Map<Object?, Object?>;
      creations.add(
        const StandardMessageCodec().decodeMessage(
              ByteData.sublistView(args['params']! as Uint8List),
            )!
            as Map<Object?, Object?>,
      );
    }
    return null;
  });

  final adapter = _LoginOptions();
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

  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        theme: gfThemeData(Brightness.dark),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: LoginPage(authTokenStorage: storage),
      ),
    ),
  );
  await tester.pumpAndSettle();

  // Providers are created only after opening the more-methods sheet.
  expect(creations, isEmpty);
  await tester.tap(find.byKey(const Key('login-more-methods')));
  await tester.pumpAndSettle();

  expect(creations, hasLength(1));
  expect(creations.single['dark'], isTrue);
  expect(creations.single['height'], AppleSignInButton.height);
  expect(creations.single['radius'], AppleSignInButton.radius);
  // The native button occupies the same row as the other providers, whose
  // stadium silhouette is what the radius has to match.
  final sibling = tester.getSize(
    find.descendant(
      of: find.byType(BottomSheet),
      matching: find.byType(OutlinedButton),
    ).first,
  );
  expect(sibling.height, creations.single['height']);
  expect(creations.single['radius'], sibling.height / 2);
}

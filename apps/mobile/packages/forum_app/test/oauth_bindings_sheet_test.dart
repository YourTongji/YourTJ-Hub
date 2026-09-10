import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/oauth_bindings_sheet.dart';
import 'package:forum_app/src/providers.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Bindings extends UserRepository {
  _Bindings(super.client);
  int reads = 0;
  String? disconnected;
  Map<String, OAuthBindingPayload> values = {
    'github': const OAuthBindingPayload(bound: false),
    'google': const OAuthBindingPayload(bound: false),
  };
  @override
  Future<Map<String, OAuthBindingPayload>> getOAuthBindings() async {
    reads++;
    return Map.of(values);
  }

  @override
  Future<bool> unbindOAuth(String provider) async {
    disconnected = provider;
    values[provider] = const OAuthBindingPayload(bound: false);
    return true;
  }
}

void main() {
  Future<void> pump(
    WidgetTester tester,
    _Bindings repo,
    GfApiClient client,
    Future<bool> Function(Uri) open,
  ) async {
    final container = ProviderContainer(
      overrides: [
        apiClientProvider.overrideWithValue(client),
        userRepositoryProvider.overrideWithValue(repo),
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
          home: Scaffold(
            body: OAuthBindingsSheet(
              username: 'alice',
              googleReady: false,
              openBrowser: open,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  GfApiClient client() => GfApiClient(
    dio: Dio(),
    tokenStorage: MemoryTokenStorage(),
    baseUrl: 'https://forum.example',
  );

  testWidgets(
    'browser entry names the native account, carries no credential and refreshes on return',
    (tester) async {
      final api = client();
      final repo = _Bindings(api);
      Uri? opened;
      await pump(tester, repo, api, (uri) async {
        opened = uri;
        return true;
      });
      expect(find.textContaining('@alice'), findsOneWidget);
      expect(find.text('Not enabled on this site'), findsOneWidget);
      await tester.tap(find.text('Manage connections in browser'));
      await tester.pumpAndSettle();
      expect(opened.toString(), 'https://forum.example/settings?tab=binding');
      repo.values['github'] = const OAuthBindingPayload(bound: true);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();
      expect(repo.reads, 2);
      expect(find.text('Bound'), findsOneWidget);
    },
  );

  testWidgets(
    'a configured-off provider can still be disconnected if already bound',
    (tester) async {
      final api = client();
      final repo = _Bindings(api)
        ..values['google'] = const OAuthBindingPayload(bound: true);
      await pump(tester, repo, api, (_) async => true);
      await tester.tap(find.text('Unbind'));
      await tester.pumpAndSettle();
      expect(repo.disconnected, 'google');
      expect(repo.reads, 2);
      expect(find.text('Not enabled on this site'), findsOneWidget);
      expect(find.text('Bound'), findsNothing);
    },
  );
}

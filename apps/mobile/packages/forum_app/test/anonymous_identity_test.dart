import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/anonymous_identity_page.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  const long = '中华人民共和国道路交通安全法实施条例';
  for (final width in [320.0, 768.0]) {
    testWidgets(
      'name selection confirms explicitly and fits at $width with large text',
      (tester) async {
        tester.view.physicalSize = Size(width, 1000);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        var confirms = 0;
        final dio = Dio();
        dio.interceptors.add(
          InterceptorsWrapper(
            onRequest: (options, handler) {
              if (options.path.endsWith('/confirm')) confirms++;
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: 200,
                  data: {
                    'code': 0,
                    'result': {
                      'kind': 'persona',
                      'publicUid': 'a' * 32,
                      'name': long,
                      'avatarUrl': '',
                      'profileUrl': '/a/${'a' * 32}',
                    },
                  },
                ),
              );
            },
          ),
        );
        final client = GfApiClient(
          dio: dio,
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'http://fake.local',
        );
        final state = AnonymousIdentityState.fromJson({
          'persona': null,
          'day': '2026-10-06',
          'remaining': 9,
          'resetsAt': '2026-10-06T16:00:00Z',
          'nameSelectedAt': null,
          'nameChangeAvailableAt': null,
          'disabled': false,
          'governanceDisabled': false,
          'lexiconVersion': 'THUOCL-a30ce79',
          'batches': [
            {
              'id': 'b' * 32,
              'day': '2026-10-06',
              'createdAt': '2026-10-06T00:00:00Z',
              'expiresAt': '2026-10-06T16:00:00Z',
              'words': [
                long,
                'C++',
                '人',
                '数学',
                '大学',
                '上海',
                '同学',
                '春天',
                '星辰',
                '通济',
              ],
            },
          ],
        });
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              apiClientProvider.overrideWithValue(client),
              anonymousIdentityProvider.overrideWith((ref) async => state),
            ],
            child: MaterialApp(
              locale: const Locale('zh'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              theme: gfThemeData(Brightness.light),
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: const TextScaler.linear(2)),
                child: child!,
              ),
              home: const AnonymousIdentityPage(),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final candidate = find.text(long);
        await tester.ensureVisible(candidate);
        await tester.tap(candidate);
        await tester.pumpAndSettle();
        expect(confirms, 0);
        expect(find.byType(AlertDialog), findsOneWidget);
        expect(tester.takeException(), isNull);
        final confirm = find.widgetWithText(FilledButton, '确认花名（锁定一年）');
        await tester.ensureVisible(confirm);
        await tester.tap(confirm);
        await tester.pumpAndSettle();
        expect(confirms, 1);
        expect(tester.takeException(), isNull);
      },
    );
  }
  test(
    'local writing recovery retains a persona and rejects an unknown identity',
    () {
      const draft = LocalDraft(
        key: 'reply-1',
        title: '',
        content: '匿名正文',
        contentType: 0,
        topicId: 1,
        categories: [],
        images: [],
        updatedAt: 1,
        kind: DraftKind.reply,
        identity: 'persona',
      );
      expect(LocalDraft.fromJson(draft.toJson()).identity, 'persona');
      expect(
        LocalDraft.fromJson({...draft.toJson()}..remove('identity')).identity,
        'member',
      );
      expect(
        () => LocalDraft.fromJson({...draft.toJson(), 'identity': 'forged'}),
        throwsFormatException,
      );
    },
  );
}

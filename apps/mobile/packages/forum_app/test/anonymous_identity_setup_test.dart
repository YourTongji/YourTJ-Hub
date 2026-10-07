import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/identity_picker.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class Fixture {
  final state = <String, dynamic>{
    'persona': null,
    'day': '2026-10-07',
    'remaining': 10,
    'resetsAt': '2026-10-07T16:00:00Z',
    'nameSelectedAt': null,
    'nameChangeAvailableAt': null,
    'disabled': false,
    'governanceDisabled': false,
    'lexiconVersion': 'test',
    'batches': <Map<String, dynamic>>[],
  };
  final keys = <String>[];
  final stored = <String, Map<String, dynamic>>{};
  int confirms = 0;
  bool failDraw = false, failConfirm = false;
  Completer<void>? confirmWait;
  late final client = GfApiClient(
    dio: dio,
    tokenStorage: MemoryTokenStorage(),
    baseUrl: 'http://fake.local',
  );
  late final dio = Dio()
    ..interceptors.add(
      InterceptorsWrapper(
        onRequest: (o, h) async {
          Object? result = state;
          if (o.path.endsWith('/batches')) {
            final body = o.data as Map;
            final key = body['requestKey'] as String;
            keys.add(key);
            if (!stored.containsKey(key)) {
              final batch = <String, dynamic>{
                'id': 'batch${stored.length}',
                'day': state['day'],
                'createdAt': '2026-10-07T00:00:00Z',
                'expiresAt': state['resetsAt'],
                'words': [
                  '星辰',
                  '中华人民共和国道路交通安全法实施条例',
                  'C++',
                  '银杏',
                  '春天',
                  '数学',
                  '微风',
                  '海棠',
                  '山川',
                  '北斗',
                ],
              };
              stored[key] = batch;
              (state['batches'] as List).add(batch);
              state['remaining'] = (state['remaining'] as int) - 1;
            }
            if (failDraw) {
              failDraw = false;
              h.reject(
                DioException(
                  requestOptions: o,
                  type: DioExceptionType.connectionError,
                ),
              );
              return;
            }
            result = stored[key];
          } else if (o.path.endsWith('/confirm')) {
            confirms++;
            await confirmWait?.future;
            if (failConfirm) {
              h.reject(
                DioException(
                  requestOptions: o,
                  type: DioExceptionType.connectionError,
                ),
              );
              return;
            }
            result = {
              'kind': 'persona',
              'publicUid': 'a' * 32,
              'name': '星辰',
              'avatarUrl': '',
              'profileUrl': '/a/${'a' * 32}',
            };
            state['persona'] = result;
            state['nameChangeAvailableAt'] = '2099-10-07T00:00:00Z';
          }
          h.resolve(
            Response(
              requestOptions: o,
              statusCode: 200,
              data: {'code': 0, 'result': result},
            ),
          );
        },
      ),
    );
}

Future<ProviderContainer> host(
  WidgetTester tester,
  Fixture fixture,
  TextEditingController draft,
  ValueChanged<String> onChanged, {
  double width = 390,
  double scale = 1,
  Locale locale = const Locale('zh'),
}) async {
  tester.view.physicalSize = Size(width, 844);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        apiClientProvider.overrideWithValue(fixture.client),
        composerMemberProvider.overrideWith((ref) async => null),
      ],
      child: MaterialApp(
        locale: locale,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        theme: gfThemeData(Brightness.light),
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
        home: Scaffold(
          body: Column(
            children: [
              TextField(controller: draft),
              IdentityPicker(value: 'member', onChanged: onChanged),
            ],
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return ProviderScope.containerOf(tester.element(find.byType(IdentityPicker)));
}

Future<void> open(WidgetTester tester) async {
  await tester.tap(find.byType(PopupMenuButton<String>));
  await tester.pumpAndSettle();
  await tester.tap(find.text('设置匿名身份'));
  await tester.pumpAndSettle();
}

Future<void> tap(WidgetTester tester, String text) async {
  if (find.text(text).evaluate().isEmpty) {
    await tester.drag(find.byType(ListView).last, const Offset(0, 2000));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text(text),
      160,
      scrollable: find.descendant(
        of: find.byType(ListView).last,
        matching: find.byType(Scrollable),
      ),
      maxScrolls: 30,
    );
  }
  await Scrollable.ensureVisible(
    tester.element(find.text(text).first),
    alignment: .35,
  );
  await tester.pumpAndSettle();
  await tester.tap(find.text(text).first);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('profile navigation does not refocus the composer', (
    tester,
  ) async {
    final f = Fixture();
    f.state['persona'] = {
      'kind': 'persona',
      'publicUid': 'a' * 32,
      'name': '星辰',
      'avatarUrl': '',
      'profileUrl': '/a/${'a' * 32}',
    };
    f.state['nameChangeAvailableAt'] = '2099-10-07T00:00:00Z';
    final draft = TextEditingController(text: 'Keep this reply');
    addTearDown(draft.dispose);
    final navigating = Completer<String?>();
    final router = GoRouter(
      redirect: (_, state) =>
          state.uri.path.startsWith('/a/') ? navigating.future : null,
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => Scaffold(
            body: Column(
              children: [
                TextField(controller: draft),
                IdentityPicker(value: 'persona', onChanged: (_) {}),
              ],
            ),
          ),
        ),
        GoRoute(
          path: '/a/:uid',
          builder: (_, _) => const Scaffold(body: Text('Anonymous profile')),
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          apiClientProvider.overrideWithValue(f.client),
          composerMemberProvider.overrideWith((ref) async => null),
        ],
        child: MaterialApp.router(
          routerConfig: router,
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          theme: gfThemeData(Brightness.light),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(TextField));
    await tester.pumpAndSettle();
    final draftFocus = FocusManager.instance.primaryFocus!;
    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('管理匿名身份'));
    await tester.pumpAndSettle();
    await tester.tap(
      find.descendant(of: find.byType(BottomSheet), matching: find.text('星辰')),
    );
    await tester.pumpAndSettle();
    expect(draftFocus.hasFocus, isFalse);
    navigating.complete(null);
    await tester.pumpAndSettle();
    expect(find.text('Anonymous profile'), findsOneWidget);
    expect(draftFocus.hasFocus, isFalse);
    expect(draft.text, 'Keep this reply');
    expect(tester.takeException(), isNull);
  });
  testWidgets(
    'composer sets up in a sheet and preserves the draft and identity until confirmation',
    (tester) async {
      final f = Fixture();
      final draft = TextEditingController(text: '保留这段尚未发布的回复');
      addTearDown(draft.dispose);
      var identity = 'member';
      await host(tester, f, draft, (value) => identity = value);
      await tester.tap(find.byType(TextField));
      await tester.pumpAndSettle();
      final draftFocus = FocusManager.instance.primaryFocus!;
      await open(tester);
      expect(draftFocus.hasFocus, isFalse);
      expect(tester.takeException(), isNull);
      expect(find.byType(BottomSheet), findsOneWidget);
      await tap(tester, '选择花名');
      await tap(tester, '星辰');
      expect(f.confirms, 0);
      expect(identity, 'member');
      expect(draft.text, '保留这段尚未发布的回复');
      final footer = find.byKey(
        const ValueKey('anonymous-confirmation-footer'),
      );
      expect(
        find.descendant(of: footer, matching: find.textContaining('一年内不可更改')),
        findsOneWidget,
      );
      await tap(tester, '确认使用此花名');
      expect(f.confirms, 1);
      expect(identity, 'persona');
      expect(find.byType(BottomSheet), findsNothing);
      expect(draft.text, '保留这段尚未发布的回复');
      expect(draftFocus.hasFocus, isTrue);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets(
    'ambiguous draw reuses request key; failed confirmation and cancellation preserve identity',
    (tester) async {
      final f = Fixture()..failDraw = true;
      final draft = TextEditingController(text: '草稿');
      addTearDown(draft.dispose);
      var identity = 'member';
      await host(tester, f, draft, (value) => identity = value);
      await open(tester);
      await tap(tester, '选择花名');
      await tap(tester, '选择花名');
      expect(f.keys[0], f.keys[1]);
      expect(f.state['remaining'], 9);
      await tap(tester, '星辰');
      f.failConfirm = true;
      await tap(tester, '确认使用此花名');
      expect(identity, 'member');
      expect(find.byType(BottomSheet), findsOneWidget);
      await tester.tap(find.byTooltip('关闭'));
      await tester.pumpAndSettle();
      expect(find.byType(BottomSheet), findsNothing);
      expect(identity, 'member');
      expect(draft.text, '草稿');
    },
  );
  testWidgets('session change rejects a delayed successful confirmation', (
    tester,
  ) async {
    final f = Fixture();
    final draft = TextEditingController(text: '草稿');
    addTearDown(draft.dispose);
    var changed = 0;
    final container = await host(tester, f, draft, (_) => changed++);
    await open(tester);
    await tap(tester, '选择花名');
    await tap(tester, '星辰');
    f.confirmWait = Completer<void>();
    await tester.tap(find.text('确认使用此花名'));
    await tester.pump();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    f.confirmWait!.complete();
    await tester.pumpAndSettle();
    expect(changed, 0);
    expect(find.byType(BottomSheet), findsNothing);
    expect(tester.takeException(), isNull);
  });
  testWidgets('previous batches remain selectable without another draw', (
    tester,
  ) async {
    final f = Fixture();
    final draft = TextEditingController();
    addTearDown(draft.dispose);
    await host(tester, f, draft, (_) {});
    await open(tester);
    await tap(tester, '选择花名');
    await tap(tester, '换一批');
    expect(f.keys.length, 2);
    expect(find.text('星辰'), findsOneWidget);
    await tap(tester, '第 2 批');
    await tester.tap(
      find.widgetWithText(CheckedPopupMenuItem<String>, '第 1 批'),
    );
    await tester.pumpAndSettle();
    expect(find.text('第 1 批'), findsOneWidget);
    expect(f.keys.length, 2);
    expect(find.text('星辰'), findsOneWidget);
  });
  for (final locale in [
    const Locale('zh'),
    const Locale('en'),
    const Locale('ja'),
    const Locale('de'),
  ]) {
    testWidgets(
      '320px setup and pinned confirmation fit at 2x text in ${locale.languageCode}',
      (tester) async {
        final f = Fixture();
        final draft = TextEditingController();
        addTearDown(draft.dispose);
        await host(
          tester,
          f,
          draft,
          (_) {},
          width: 320,
          scale: 2,
          locale: locale,
        );
        final l = await AppLocalizations.delegate.load(locale);
        await tester.tap(find.byType(PopupMenuButton<String>));
        await tester.pumpAndSettle();
        await tap(tester, l.anonymousSetup);
        await tap(tester, l.anonymousChooseName);
        await tap(tester, '中华人民共和国道路交通安全法实施条例');
        final r = tester.getRect(
          find.byKey(const ValueKey('anonymous-confirmation-footer')),
        );
        expect(r.left, greaterThanOrEqualTo(0));
        expect(r.right, lessThanOrEqualTo(320));
        expect(r.bottom, lessThanOrEqualTo(844));
        expect(tester.takeException(), isNull);
      },
    );
  }
  testWidgets('setup remains scrollable above the keyboard at 2x text', (
    tester,
  ) async {
    final f = Fixture();
    final draft = TextEditingController(text: 'Keep this reply');
    addTearDown(draft.dispose);
    var identity = 'member';
    await host(
      tester,
      f,
      draft,
      (value) => identity = value,
      width: 320,
      scale: 2,
      locale: const Locale('en'),
    );
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    addTearDown(tester.view.resetViewInsets);
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(tester.element(find.byType(IdentityPicker)));
    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tap(tester, l.anonymousSetup);
    await tap(tester, l.anonymousChooseName);
    await tap(tester, '星辰');
    final footer = tester.getRect(
      find.byKey(const ValueKey('anonymous-confirmation-footer')),
    );
    expect(footer.bottom, lessThanOrEqualTo(544));
    expect(tester.getSize(find.byType(ListView)).height, greaterThan(0));
    await tap(tester, l.anonymousConfirmName);
    expect(identity, 'persona');
    expect(draft.text, 'Keep this reply');
    expect(tester.takeException(), isNull);
  });
}

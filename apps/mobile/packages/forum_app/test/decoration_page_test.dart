import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/decoration_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/user_decoration.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class Decorations extends UserRepository {
  Decorations()
    : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage()));
  final calls = <Map<String, String>>[];
  DecorationState state = DecorationState(
    catalog: const [
      DecorationStyle(
        code: 'frame',
        name: 'Frame',
        kind: 'avatarFrame',
        assetScale: 1.4,
        lightColors: ['#123456', '#654321'],
        darkColors: ['#abcdef', '#fedcba'],
      ),
      DecorationStyle(
        code: 'name',
        name: 'Name',
        kind: 'nicknameStyle',
        lightColors: ['#123456', '#654321'],
        darkColors: ['#abcdef', '#fedcba'],
      ),
    ],
    ownedCodes: ['frame'],
    avatarFrame: '',
    nicknameStyle: '',
  );
  @override
  Future<DecorationState> getDecorations() async => state;
  @override
  Future<DecorationState> equipDecorations({
    required String avatarFrame,
    required String nicknameStyle,
  }) async {
    calls.add({'avatarFrame': avatarFrame, 'nicknameStyle': nicknameStyle});
    state = DecorationState(
      catalog: state.catalog,
      ownedCodes: state.ownedCodes,
      avatarFrame: avatarFrame,
      nicknameStyle: nicknameStyle,
    );
    return state;
  }
}

void main() {
  testWidgets(
    'single-stop styles display solid avatar frame and nickname in both themes',
    (tester) async {
      for (final brightness in [Brightness.light, Brightness.dark]) {
        await tester.pumpWidget(
          MaterialApp(
            key: ValueKey(brightness),
            theme: gfThemeData(brightness),
            home: Scaffold(
              body: Builder(
                builder: (context) {
                  final frame = DecorationStyle.fromJson({
                    'code': 'frame',
                    'name': 'Frame',
                    'kind': 'avatarFrame',
                    'lightColors': ['#123456'],
                    'darkColors': ['#abcdef'],
                  });
                  final name = DecorationStyle.fromJson({
                    'code': 'name',
                    'name': 'Name',
                    'kind': 'nicknameStyle',
                    'lightColors': ['#123456'],
                    'darkColors': ['#abcdef'],
                  });
                  final colors = decorationColors(context, name);
                  expect(colors.length, 2);
                  expect(colors.first, colors.last);
                  return Column(
                    children: [
                      GfAvatar(
                        src: '',
                        frameColors: decorationColors(context, frame),
                      ),
                      GfDecoratedName('Alice', colors: colors),
                    ],
                  );
                },
              ),
            ),
          ),
        );
        expect(find.byType(ShaderMask), findsOneWidget);
        final avatar = tester.widget<GfAvatar>(find.byType(GfAvatar));
        expect(avatar.frameColors.length, 2);
        expect(avatar.frameColors.first, avatar.frameColors.last);
        expect(tester.takeException(), isNull);
      }
    },
  );
  testWidgets(
    'unowned styles preview only and equip removal preserves the other slot',
    (tester) async {
      final repo = Decorations();
      final container = ProviderContainer(
        overrides: [userRepositoryProvider.overrideWithValue(repo)],
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
            home: const DecorationPage(name: 'Alice'),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Not owned'), findsOneWidget);
      final nameRow = find.ancestor(
        of: find.text('Name'),
        matching: find.byType(ListTile),
      );
      final equip = find.descendant(
        of: nameRow,
        matching: find.widgetWithText(TextButton, 'Equip'),
      );
      expect(tester.widget<TextButton>(equip).onPressed, isNull);
      await tester.tap(
        find.descendant(
          of: nameRow,
          matching: find.widgetWithText(TextButton, 'Preview'),
        ),
      );
      await tester.pump();
      expect(find.byType(ShaderMask), findsOneWidget);
      expect(repo.calls, isEmpty);
      final frameRow = find.ancestor(
        of: find.text('Frame'),
        matching: find.byType(ListTile),
      );
      await tester.tap(
        find.descendant(
          of: frameRow,
          matching: find.widgetWithText(TextButton, 'Equip'),
        ),
      );
      await tester.pumpAndSettle();
      expect(repo.calls.single, {'avatarFrame': 'frame', 'nicknameStyle': ''});
      expect(
        tester.widget<GfAvatar>(find.byType(GfAvatar).first).frameScale,
        1.4,
      );
      await tester.tap(find.widgetWithText(TextButton, 'Equip').first);
      await tester.pumpAndSettle();
      expect(repo.calls.last, {'avatarFrame': '', 'nicknameStyle': ''});
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      expect(find.byType(DecorationPage), findsOneWidget);
      expect(find.text('Frame'), findsNothing);
    },
  );
  testWidgets('server decoration labels use shared Twemoji rendering', (
    tester,
  ) async {
    final repo = Decorations();
    repo.state = DecorationState(
      catalog: const [
        DecorationStyle(
          code: 'name',
          name: 'Colorful 🐦‍🔥',
          kind: 'nicknameStyle',
          lightColors: ['#123456', '#654321'],
          darkColors: ['#abcdef', '#fedcba'],
        ),
      ],
      ownedCodes: const [],
      avatarFrame: '',
      nicknameStyle: '',
    );
    final container = ProviderContainer(
      overrides: [userRepositoryProvider.overrideWithValue(repo)],
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
          home: const DecorationPage(name: 'Alice'),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.bySemanticsLabel('🐦‍🔥'), findsOneWidget);
    expect(find.byType(GfText), findsWidgets);
  });

  testWidgets(
    'nickname color selection handles theme and insufficient contrast',
    (tester) async {
      final colors = <List<Color>>[];
      for (final brightness in [Brightness.light, Brightness.dark]) {
        await tester.pumpWidget(
          MaterialApp(
            key: ValueKey(brightness),
            theme: gfThemeData(brightness),
            home: Builder(
              builder: (context) {
                colors.add(
                  decorationColors(
                    context,
                    const DecorationStyle(
                      code: 'name',
                      name: 'Name',
                      kind: 'nicknameStyle',
                      lightColors: ['#123456', '#654321'],
                      darkColors: ['#abcdef', '#fedcba'],
                    ),
                  ),
                );
                expect(
                  decorationColors(
                    context,
                    const DecorationStyle(
                      code: 'bad',
                      name: 'Bad',
                      kind: 'nicknameStyle',
                      lightColors: ['#ffffff', '#ffffff'],
                      darkColors: ['#000000', '#000000'],
                    ),
                  ),
                  isEmpty,
                );
                return const SizedBox();
              },
            ),
          ),
        );
      }
      expect(colors.first.first, const Color(0xff123456));
      expect(colors.last.first, const Color(0xffabcdef));
    },
  );
}

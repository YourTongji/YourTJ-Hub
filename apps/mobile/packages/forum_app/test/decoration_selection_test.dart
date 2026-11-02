import 'dart:async';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/decoration_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import 'decoration_page_test.dart' show Decorations;

class ControlledDecorations extends Decorations {
  bool failNext = false;
  Completer<DecorationState>? pending;

  @override
  Future<DecorationState> equipDecorations({
    required String avatarFrame,
    required String nicknameStyle,
  }) async {
    if (failNext || pending != null) {
      calls.add({'avatarFrame': avatarFrame, 'nicknameStyle': nicknameStyle});
      if (failNext) {
        failNext = false;
        throw StateError('Save failed');
      }
      return pending!.future;
    }
    return super.equipDecorations(
      avatarFrame: avatarFrame,
      nicknameStyle: nicknameStyle,
    );
  }
}

Future<ProviderContainer> openDecorations(
  WidgetTester tester,
  Decorations repo, {
  Locale locale = const Locale('en'),
  Brightness brightness = Brightness.light,
  double scale = 1,
}) async {
  final container = ProviderContainer(
    overrides: [userRepositoryProvider.overrideWithValue(repo)],
  );
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        theme: gfThemeData(brightness),
        locale: locale,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
        home: const DecorationPage(name: 'Alice'),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return container;
}

Finder choice(String kind, String code) =>
    find.byKey(ValueKey('decoration-option-$kind-$code'));
Finder get apply => find.byKey(const Key('decoration-apply'));

void main() {
  testWidgets('style labels and frame thumbnails display their real effects', (
    tester,
  ) async {
    await openDecorations(tester, Decorations());
    final frame = tester.widget<GfAvatar>(
      find.descendant(
        of: choice('avatarFrame', 'frame'),
        matching: find.byType(GfAvatar),
      ),
    );
    expect(frame.frameColors, isNotEmpty);
    await tester.tap(find.text('Colorful names'));
    await tester.pumpAndSettle();
    final label = tester.widget<GfDecoratedName>(
      find.descendant(
        of: choice('nicknameStyle', 'name'),
        matching: find.byType(GfDecoratedName),
      ),
    );
    expect(label.colors, isNotEmpty);
    expect(find.byType(ListTile), findsNothing);
  });

  testWidgets(
    'locked selection previews without saving and keeps the other draft',
    (tester) async {
      final repo = Decorations();
      await openDecorations(tester, repo);
      await tester.tap(find.text('Colorful names'));
      await tester.pumpAndSettle();
      await tester.tap(choice('nicknameStyle', 'name'));
      await tester.pumpAndSettle();
      expect(repo.calls, isEmpty);
      expect(tester.widget<GfButton>(apply).onPressed, isNull);
      final hero = find.byKey(const Key('decoration-preview-name'));
      expect(tester.widget<GfDecoratedName>(hero).colors, isNotEmpty);
      await tester.tap(find.text('Avatar frames'));
      await tester.pumpAndSettle();
      await tester.tap(choice('avatarFrame', 'frame'));
      await tester.pumpAndSettle();
      expect(repo.calls, isEmpty);
      await tester.tap(apply);
      await tester.pumpAndSettle();
      expect(repo.calls.single, {'avatarFrame': 'frame', 'nicknameStyle': ''});
      expect(tester.widget<GfDecoratedName>(hero).colors, isNotEmpty);
      await tester.tap(choice('avatarFrame', ''));
      await tester.pumpAndSettle();
      await tester.tap(apply);
      await tester.pumpAndSettle();
      expect(repo.calls.last, {'avatarFrame': '', 'nicknameStyle': ''});
    },
  );

  testWidgets('failed equipment keeps the selection available for retry', (
    tester,
  ) async {
    final repo = ControlledDecorations()..failNext = true;
    repo.state = DecorationState(
      catalog: [
        ...repo.state.catalog,
        for (var i = 0; i < 20; i++)
          DecorationStyle(
            code: 'extra-$i',
            name: 'Extra frame $i',
            kind: 'avatarFrame',
            lightColors: const ['#123456'],
            darkColors: const ['#abcdef'],
          ),
      ],
      ownedCodes: repo.state.ownedCodes,
      avatarFrame: '',
      nicknameStyle: '',
    );
    await openDecorations(tester, repo);
    await tester.tap(choice('avatarFrame', 'frame'));
    await tester.pumpAndSettle();
    await tester.drag(find.byType(ListView), const Offset(0, -4000));
    await tester.pumpAndSettle();
    await tester.tap(apply);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('decoration-save-error')), findsOneWidget);
    expect(
      find.byKey(const Key('decoration-save-error')).hitTestable(),
      findsOneWidget,
    );
    expect(
      tester.getRect(find.byKey(const Key('decoration-save-error'))).bottom,
      lessThanOrEqualTo(tester.getRect(apply).top),
    );
    expect(repo.state.avatarFrame, isEmpty);
    expect(tester.widget<GfButton>(apply).onPressed, isNotNull);
    await tester.tap(apply);
    await tester.pumpAndSettle();
    expect(repo.calls, hasLength(2));
    expect(repo.state.avatarFrame, 'frame');
    expect(tester.widget<GfButton>(apply).onPressed, isNull);
  });

  testWidgets('late equipment acknowledgement cannot cross an account switch', (
    tester,
  ) async {
    final repo = ControlledDecorations()
      ..pending = Completer<DecorationState>();
    final container = await openDecorations(tester, repo);
    await tester.tap(choice('avatarFrame', 'frame'));
    await tester.pumpAndSettle();
    await tester.tap(apply);
    await tester.pump();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pump();
    repo.pending!.complete(repo.state);
    await tester.pumpAndSettle();
    expect(choice('avatarFrame', 'frame'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'selected nickname labels validate contrast on their actual fill',
    (tester) async {
      final repo = Decorations()
        ..state = const DecorationState(
          catalog: [
            DecorationStyle(
              code: 'near',
              name: 'Near contrast floor',
              kind: 'nicknameStyle',
              lightColors: ['#949494'],
              darkColors: ['#cccccc'],
            ),
          ],
          ownedCodes: ['near'],
          avatarFrame: '',
          nicknameStyle: '',
        );
      await openDecorations(tester, repo);
      await tester.tap(find.text('Colorful names'));
      await tester.pumpAndSettle();
      final label = find.descendant(
        of: choice('nicknameStyle', 'near'),
        matching: find.byType(GfDecoratedName),
      );
      expect(tester.widget<GfDecoratedName>(label).colors, isNotEmpty);
      await tester.tap(choice('nicknameStyle', 'near'));
      await tester.pumpAndSettle();
      expect(tester.widget<GfDecoratedName>(label).colors, isEmpty);
    },
  );

  for (final language in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets('selection fits 320px 200% $language ${brightness.name}', (
        tester,
      ) async {
        tester.view.physicalSize = const Size(320, 900);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        final repo = Decorations();
        await openDecorations(
          tester,
          repo,
          locale: Locale(language),
          brightness: brightness,
          scale: 2,
        );
        await tester.ensureVisible(choice('avatarFrame', 'frame'));
        await tester.pumpAndSettle();
        await tester.tap(choice('avatarFrame', 'frame'));
        await tester.pumpAndSettle();
        expect(tester.widget<GfButton>(apply).onPressed, isNotNull);
        expect(tester.takeException(), isNull);
        final control = tester.widget<GfSegmented<String>>(
          find.byType(GfSegmented<String>),
        );
        control.onSelected('nicknameStyle');
        await tester.pumpAndSettle();
        await tester.ensureVisible(choice('nicknameStyle', 'name'));
        await tester.pumpAndSettle();
        await tester.tap(choice('nicknameStyle', 'name'));
        await tester.pumpAndSettle();
        expect(tester.widget<GfButton>(apply).onPressed, isNull);
        expect(tester.takeException(), isNull);
      });
    }
  }
}

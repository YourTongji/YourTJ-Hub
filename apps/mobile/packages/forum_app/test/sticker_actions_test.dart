import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/src/widgets/stickers/sticker_card.dart';
import 'package:forum_app/src/widgets/stickers/sticker_image.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_page.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:forum_app/src/widgets/stickers/sticker_picker.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_behavior_test.dart' show MemTokenStorage;

const _item = StickerItemPayload(
  name: 'dance',
  displayName: 'Dance',
  url: 'https://example.test/dance.gif',
);

class _Repository extends StickerRepository {
  _Repository()
    : super(GfApiClient(dio: Dio(), tokenStorage: MemTokenStorage()));
  List<StickerItemPayload> members = [_item];
  final removed = <String>[];
  final saved = <String>[];
  int mineCalls = 0;
  bool failRemove = false;
  bool failSave = false;
  @override
  Future<List<StickerItemPayload>> list() async => [_item];
  @override
  Future<List<StickerItemPayload>> mine() async {
    mineCalls++;
    return [...members];
  }
  @override
  Future<StickerItemPayload> save({
    String? stickerName,
    String? fileName,
    String? displayName,
  }) async {
    saved.add(stickerName!);
    if (failSave) throw StateError('offline');
    final item = StickerItemPayload(
      name: stickerName,
      url: 'https://example.test/$stickerName.gif',
      isOfficial: false,
    );
    members.add(item);
    return item;
  }

  @override
  Future<void> remove(String name) async {
    removed.add(name);
    if (failRemove) throw StateError('offline');
    members.removeWhere((item) => item.name == name);
  }
}

Future<StickerCollection> _pump(
  WidgetTester tester,
  _Repository repository, {
  Widget child = const StickerLibraryPage(),
  Brightness brightness = Brightness.light,
}) async {
  final state = StickerCollection(repository, StickerLibrary(repository));
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        stickerCollectionProvider.overrideWith((_) => state),
        // StickerImage taps open the sticker card, which checks the session
        // before touching the account-owned library.
        tokenStorageProvider.overrideWith((_) => MemTokenStorage('session')),
      ],
      child: MaterialApp(theme: gfThemeData(brightness), home: child),
    ),
  );
  await tester.pumpAndSettle();
  return state;
}

void main() {
  testWidgets('embedded sticker copies expose a single outer button label', (
    tester,
  ) async {
    final handle = tester.ensureSemantics();
    final repository = _Repository();
    final state = StickerCollection(repository, StickerLibrary(repository));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerCollectionProvider.overrideWith((_) => state)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: SizedBox(height: 400, child: StickerPicker(onInsert: (_) {})),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Official'));
    await tester.pumpAndSettle();
    expect(
      tester.getSemantics(find.text('Dance')),
      matchesSemantics(
        isButton: true,
        isFocusable: true,
        hasTapAction: true,
        hasFocusAction: true,
        hasLongPressAction: true,
        hint: 'Tap to insert. Hold to preview.',
        children: [],
      ),
    );
    handle.dispose();
  });

  testWidgets('unavailable stickers remain removable from their row menu', (
    tester,
  ) async {
    final repository = _Repository()
      ..members = [
        const StickerItemPayload(
          name: 'dance',
          displayName: 'Dance',
          url: '',
          isEnabled: false,
        ),
      ];
    final state = await _pump(tester, repository);
    await tester.tap(find.byTooltip('Sticker actions: Dance'));
    await tester.pumpAndSettle();
    expect(find.text('View larger'), findsNothing);
    await tester.tap(find.text('Remove'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Remove'));
    await tester.pumpAndSettle();
    expect(state.mine, isEmpty);
  });

  testWidgets('batch removal also requires confirmation', (tester) async {
    final repository = _Repository();
    final state = await _pump(tester, repository);
    await tester.tap(find.text('Select'));
    await tester.pump();
    await tester.tap(find.byType(Checkbox));
    await tester.pump();
    await tester.tap(find.text('Remove (1)'));
    await tester.pumpAndSettle();
    expect(repository.removed, isEmpty);
    await tester.tap(find.text('Remove'));
    await tester.pumpAndSettle();
    expect(state.mine, isEmpty);
  });

  testWidgets('row actions fit a 320 pixel window with large text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final repository = _Repository();
    await _pump(
      tester,
      repository,
      child: const MediaQuery(
        data: MediaQueryData(
          size: Size(320, 640),
          textScaler: TextScaler.linear(2),
        ),
        child: StickerLibraryPage(),
      ),
    );
    final action = find.byTooltip('Sticker actions: Dance');
    await tester.scrollUntilVisible(
      action,
      180,
      scrollable: find
          .descendant(
            of: find.byType(ReorderableListView),
            matching: find.byType(Scrollable),
          )
          .first,
    );
    expect(tester.getSize(action).width, greaterThanOrEqualTo(44));
    expect(tester.getSize(action).height, greaterThanOrEqualTo(44));
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(find.text('Remove'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  for (final brightness in Brightness.values) {
    testWidgets('row exposes confirmed removal without selecting $brightness', (
      tester,
    ) async {
      final repository = _Repository();
      final state = await _pump(tester, repository, brightness: brightness);
      await tester.tap(find.byTooltip('Sticker actions: Dance'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Remove'));
      await tester.pumpAndSettle();
      expect(repository.removed, isEmpty);
      expect(find.text('Sent stickers will remain available.'), findsOneWidget);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(state.mine, [_item]);
      await tester.tap(find.byTooltip('Sticker actions: Dance'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Remove'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Remove'));
      await tester.pumpAndSettle();
      expect(repository.removed, ['dance']);
      expect(state.mine, isEmpty);
    });
  }

  testWidgets('failed removal keeps the row and can be retried', (
    tester,
  ) async {
    final repository = _Repository()..failRemove = true;
    final state = await _pump(tester, repository);
    Future<void> remove() async {
      await tester.tap(find.byTooltip('Sticker actions: Dance'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Remove'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Remove'));
      await tester.pumpAndSettle();
    }

    await remove();
    expect(state.mine, [_item]);
    expect(
      find.text('Could not complete this action. Try again.'),
      findsOneWidget,
    );
    repository.failRemove = false;
    await remove();
    expect(state.mine, isEmpty);
  });

  testWidgets('picker previews without inserting or changing recent use', (
    tester,
  ) async {
    final inserted = <String>[];
    final state = await _pump(
      tester,
      _Repository(),
      child: Scaffold(
        body: SizedBox(
          height: 400,
          child: StickerPicker(onInsert: inserted.add),
        ),
      ),
    );
    for (final tab in ['Official', 'Mine']) {
      await tester.tap(find.widgetWithText(TextButton, tab));
      await tester.pumpAndSettle();
      await tester.longPress(find.text('Dance'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('View larger'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));
      expect(tester.widget<GfImageViewer>(find.byType(GfImageViewer)).images, [
        _item.url,
      ]);
      expect(inserted, isEmpty);
      expect(state.recent, isEmpty);
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
    }
    await tester.tap(find.text('Dance'));
    await tester.pumpAndSettle();
    expect(inserted, [_item.token]);
    expect(state.recent, [_item]);
    await tester.tap(find.widgetWithText(TextButton, 'Recent'));
    await tester.pumpAndSettle();
    await tester.longPress(find.text('Dance'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('View larger'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(tester.widget<GfImageViewer>(find.byType(GfImageViewer)).images, [
      _item.url,
    ]);
    expect(inserted, [_item.token]);
    expect(state.recent, [_item]);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
  });

  testWidgets('library row tap opens a single-sticker preview', (tester) async {
    await _pump(tester, _Repository());
    await tester.tap(find.text('Dance'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(tester.widget<GfImageViewer>(find.byType(GfImageViewer)).images, [
      _item.url,
    ]);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
  });

  testWidgets('inline preview consumes tap while message keeps long press', (
    tester,
  ) async {
    var taps = 0;
    var holds = 0;
    await _pump(
      tester,
      _Repository(),
      child: Scaffold(
        body: GestureDetector(
          onTap: () => taps++,
          onLongPress: () => holds++,
          child: const StickerImage(
            name: 'dance',
            url: 'https://example.test/dance.gif',
            deferLongPress: true,
          ),
        ),
      ),
    );
    await tester.longPress(find.byType(StickerImage));
    await tester.pumpAndSettle();
    expect(holds, 1);
    expect(find.byType(GfImageViewer), findsNothing);
    await tester.tap(find.byType(StickerImage));
    await tester.pumpAndSettle();
    expect(find.byType(StickerCard), findsOneWidget);
    expect(find.byType(GfImageViewer), findsNothing);
    expect(taps, 0);
  });

  group('sticker card', () {
    Future<void> open(WidgetTester tester, String name) async {
      await tester.tap(find.byType(StickerImage));
      await tester.pumpAndSettle();
      expect(find.byType(StickerCard), findsOneWidget);
      // Opening never inserts or collects anything by itself.
      expect(
        tester.getSize(find.byType(GfNetworkImage).last),
        const Size.square(StickerCard.stageSize),
      );
    }

    Widget sticker(String name) => Scaffold(
      body: Center(
        child: StickerImage(name: name, url: 'https://example.test/$name.gif'),
      ),
    );

    testWidgets('collects a shared sticker and confirms in place', (
      tester,
    ) async {
      final repository = _Repository()..members = [];
      final state = await _pump(tester, repository, child: sticker('dance'));
      await state.library.load();
      await open(tester, 'dance');
      expect(find.text('Dance'), findsOneWidget);
      expect(repository.saved, isEmpty);

      await tester.tap(find.text('Save to my stickers'));
      await tester.pumpAndSettle();
      expect(repository.saved, ['dance']);
      expect(find.text('In your stickers'), findsOneWidget);
      expect(find.text('Save to my stickers'), findsNothing);

      await tester.tapAt(const Offset(4, 4));
      await tester.pumpAndSettle();
      expect(find.byType(StickerCard), findsNothing);
    });

    testWidgets('shows a sticker already in your library as saved', (
      tester,
    ) async {
      await _pump(tester, _Repository(), child: sticker('dance'));
      await open(tester, 'dance');
      expect(find.text('In your stickers'), findsOneWidget);
    });

    testWidgets('keeps the card open with a retryable error', (tester) async {
      final repository = _Repository()
        ..members = []
        ..failSave = true;
      await _pump(tester, repository, child: sticker('u_${'a' * 48}'));
      await open(tester, 'u_${'a' * 48}');
      expect(find.text('Custom sticker'), findsOneWidget);
      await tester.tap(find.text('Save to my stickers'));
      await tester.pumpAndSettle();
      expect(
        find.text('Could not complete this action. Try again.'),
        findsOneWidget,
      );
      repository.failSave = false;
      await tester.tap(find.text('Save to my stickers'));
      await tester.pumpAndSettle();
      expect(find.text('In your stickers'), findsOneWidget);
      expect(
        find.text('Could not complete this action. Try again.'),
        findsNothing,
      );
    });

    testWidgets('read-only excerpts open the card without collection', (
      tester,
    ) async {
      await _pump(
        tester,
        _Repository(),
        child: const Scaffold(
          body: Center(
            child: StickerImage(
              name: 'dance',
              url: 'https://example.test/dance.gif',
              collectible: false,
            ),
          ),
        ),
      );
      await open(tester, 'dance');
      expect(find.byType(GfButton), findsNothing);
    });

    testWidgets('guests get a sign-in prompt instead of library requests', (
      tester,
    ) async {
      final repository = _Repository();
      final router = GoRouter(
        initialLocation: '/',
        routes: [
          GoRoute(path: '/', builder: (context, state) => sticker('dance')),
          GoRoute(
            path: '/login',
            builder: (context, state) =>
                const Scaffold(body: Text('login page')),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            stickerCollectionProvider.overrideWith(
              (_) => StickerCollection(repository, StickerLibrary(repository)),
            ),
            tokenStorageProvider.overrideWith((_) => MemTokenStorage()),
          ],
          child: MaterialApp.router(
            theme: gfThemeData(Brightness.light),
            routerConfig: router,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await open(tester, 'dance');
      // 客访不触碰账号态表情库:不发 my-stickers 请求(401 会触发全局拆会话)。
      expect(repository.mineCalls, 0);
      expect(find.text('Sign in to save'), findsOneWidget);
      expect(find.text('Save to my stickers'), findsNothing);

      await tester.tap(find.text('Sign in to save'));
      await tester.pumpAndSettle();
      expect(find.text('login page'), findsOneWidget);
      expect(find.byType(StickerCard), findsNothing);
    });
  });

  testWidgets('account change invalidates a pending removal confirmation', (
    tester,
  ) async {
    final repository = _Repository();
    final container = ProviderContainer(
      overrides: [
        stickerCollectionProvider.overrideWith((ref) {
          ref.watch(offlineCacheEpochProvider);
          return StickerCollection(repository, StickerLibrary(repository));
        }),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: const StickerLibraryPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Sticker actions: Dance'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Remove'));
    await tester.pumpAndSettle();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    expect(find.text('Sent stickers will remain available.'), findsNothing);
    expect(repository.removed, isEmpty);
  });
}

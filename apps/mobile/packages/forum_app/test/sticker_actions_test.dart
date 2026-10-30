import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/src/widgets/stickers/sticker_card.dart';
import 'package:forum_app/src/widgets/stickers/sticker_image.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_page.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:forum_app/src/widgets/stickers/sticker_market_page.dart';
import 'package:forum_app/src/widgets/stickers/sticker_picker.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_behavior_test.dart' show MemTokenStorage;

const _item = StickerItemPayload(
  name: 'dance',
  displayName: 'Dance',
  url: 'https://example.test/dance.gif',
);

const _pack = StickerMarketPackPayload(
  id: 9201,
  name: 'Pack One',
  topic: 'anime',
  stickerCount: 1,
);
const _marketItem = StickerItemPayload(
  name: 'm_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  displayName: 'Pack One 1',
  url: 'https://example.test/m.png',
  pack: 'market',
  isOfficial: false,
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
  List<StickerMarketPackPayload> packs = [];
  final savedPacks = <int>[];
  Completer<List<StickerMarketPackPayload>>? pendingPacks;
  @override
  Future<List<StickerMarketPackPayload>> myPacks() async {
    final pending = pendingPacks;
    pendingPacks = null;
    return pending != null ? await pending.future : [...packs];
  }

  @override
  Future<StickerMarketPackDetailPayload> marketPack(int id) async =>
      const StickerMarketPackDetailPayload(
        pack: _pack,
        stickers: [_marketItem],
      );
  @override
  Future<StickerMarketPagePayload> market({
    String topic = '',
    int page = 1,
    int pageSize = 24,
  }) async => const StickerMarketPagePayload(
    topics: [StickerMarketTopicPayload(id: 'anime', label: 'Anime')],
    packs: [_pack],
    total: 1,
  );
  @override
  Future<StickerMarketPackPayload> savePack(int packId) async {
    savedPacks.add(packId);
    packs.add(_pack);
    return _pack;
  }

  @override
  Future<void> removePack(int packId) async =>
      packs.removeWhere((pack) => pack.id == packId);
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

/// Picker cells show only the image; their accessible label names them.
Finder _pickerSticker(String label) => find.byWidgetPredicate(
  (widget) =>
      widget is Semantics &&
      widget.properties.label == label &&
      widget.child is InkWell,
);

/// Text outside any Material falls back to the debug error style, drawn with
/// a double underline.
bool _hasMissingMaterialText(WidgetTester tester) {
  for (final text in tester.widgetList<RichText>(find.byType(RichText))) {
    var found = false;
    text.text.visitChildren((span) {
      found = span.style?.decorationStyle == TextDecorationStyle.double;
      return !found;
    });
    if (found) return true;
  }
  return false;
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
      tester.getSemantics(_pickerSticker('Dance')),
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

  testWidgets('holding a picker sticker previews it without inserting', (
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
    StickerImage bubbleImage() => tester.widget<StickerImage>(
      find.byWidgetPredicate((w) => w is StickerImage && w.size == 120),
    );
    for (final tab in ['Official', 'Mine']) {
      await tester.tap(find.widgetWithText(TextButton, tab));
      await tester.pumpAndSettle();
      final gesture = await tester.startGesture(
        tester.getCenter(_pickerSticker('Dance')),
      );
      await tester.pump(kLongPressTimeout + const Duration(milliseconds: 50));
      await tester.pump();
      expect(bubbleImage().url, _item.url);
      expect(_hasMissingMaterialText(tester), isFalse);
      await gesture.up();
      await tester.pumpAndSettle();
      expect(
        find.byWidgetPredicate((w) => w is StickerImage && w.size == 120),
        findsNothing,
      );
      expect(inserted, isEmpty);
      expect(state.recent, isEmpty);
    }
    // Pointer and assistive users keep the preview and save menu.
    await tester.tap(_pickerSticker('Dance'), buttons: kSecondaryButton);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View larger'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(tester.widget<GfImageViewer>(find.byType(GfImageViewer)).images, [
      _item.url,
    ]);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    await tester.tap(_pickerSticker('Dance'));
    await tester.pumpAndSettle();
    expect(inserted, [_item.token]);
    expect(state.recent, [_item]);
  });

  testWidgets('search opens from an icon and covers every tab and pack', (
    tester,
  ) async {
    const wave = StickerItemPayload(
      name: 'wave',
      displayName: 'Wave',
      url: 'https://example.test/wave.gif',
    );
    final inserted = <String>[];
    await _pump(
      tester,
      _Repository()
        ..members = [_item, wave]
        ..packs = [_pack],
      child: Scaffold(
        body: SizedBox(
          height: StickerPicker.comfortableHeight,
          child: StickerPicker(onInsert: inserted.add),
        ),
      ),
    );
    expect(find.byType(GfSearchField), findsNothing);
    expect(find.byTooltip('Manage my stickers'), findsOneWidget);

    await tester.tap(find.byTooltip('Search all stickers'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(TextButton, 'Official'), findsNothing);
    await tester.enterText(find.byType(TextField), 'pack one');
    await tester.pumpAndSettle();
    // A subscribed pack is searchable without opening its tab first.
    await tester.tap(_pickerSticker('Pack One 1'));
    expect(inserted, [_marketItem.token]);

    await tester.enterText(find.byType(TextField), 'wav');
    await tester.pumpAndSettle();
    expect(_pickerSticker('Wave'), findsOneWidget);
    expect(_pickerSticker('Dance'), findsNothing);

    await tester.tap(find.widgetWithText(TextButton, 'Cancel'));
    await tester.pumpAndSettle();
    expect(find.byType(GfSearchField), findsNothing);
    expect(find.widgetWithText(TextButton, 'Official'), findsOneWidget);
  });

  testWidgets('sliding while holding switches the previewed sticker', (
    tester,
  ) async {
    const wave = StickerItemPayload(
      name: 'wave',
      displayName: 'Wave',
      url: 'https://example.test/wave.gif',
    );
    final inserted = <String>[];
    await _pump(
      tester,
      _Repository()..members = [_item, wave],
      child: Scaffold(
        body: Align(
          alignment: Alignment.bottomCenter,
          child: SizedBox(
            height: 300,
            child: StickerPicker(onInsert: inserted.add),
          ),
        ),
      ),
    );
    await tester.tap(find.widgetWithText(TextButton, 'Mine'));
    await tester.pumpAndSettle();
    String previewed() => tester
        .widget<StickerImage>(
          find.byWidgetPredicate((w) => w is StickerImage && w.size == 120),
        )
        .url;
    final gesture = await tester.startGesture(
      tester.getCenter(_pickerSticker('Dance')),
    );
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 50));
    await tester.pump();
    expect(previewed(), _item.url);
    await gesture.moveTo(tester.getCenter(_pickerSticker('Wave')));
    await tester.pump();
    expect(previewed(), wave.url);
    await gesture.cancel();
    await tester.pumpAndSettle();
    expect(
      find.byWidgetPredicate((w) => w is StickerImage && w.size == 120),
      findsNothing,
    );
    expect(inserted, isEmpty);
  });

  testWidgets('releasing over the bubble save target collects the sticker', (
    tester,
  ) async {
    final repository = _Repository()..members = [];
    final state = await _pump(
      tester,
      repository,
      child: Scaffold(
        body: Align(
          alignment: Alignment.bottomCenter,
          child: SizedBox(height: 300, child: StickerPicker(onInsert: (_) {})),
        ),
      ),
    );
    await tester.tap(find.widgetWithText(TextButton, 'Official'));
    await tester.pumpAndSettle();
    final gesture = await tester.startGesture(
      tester.getCenter(_pickerSticker('Dance')),
    );
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 50));
    await tester.pump();
    await gesture.moveTo(tester.getCenter(find.text('Save')));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    expect(repository.saved, ['dance']);
    expect(state.mine.map((item) => item.name), ['dance']);
  });

  testWidgets('leaving the save target for the preview does not collect', (
    tester,
  ) async {
    final repository = _Repository()..members = [];
    final state = await _pump(
      tester,
      repository,
      child: Scaffold(
        body: Align(
          alignment: Alignment.bottomCenter,
          child: SizedBox(height: 300, child: StickerPicker(onInsert: (_) {})),
        ),
      ),
    );
    await tester.tap(find.widgetWithText(TextButton, 'Official'));
    await tester.pumpAndSettle();
    final gesture = await tester.startGesture(
      tester.getCenter(_pickerSticker('Dance')),
    );
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 50));
    await tester.pump();
    await gesture.moveTo(tester.getCenter(find.text('Save')));
    await tester.pump();
    await gesture.moveTo(
      tester.getCenter(
        find.byWidgetPredicate((w) => w is StickerImage && w.size == 120),
      ),
    );
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    expect(repository.saved, isEmpty);
    expect(state.mine, isEmpty);
  });

  for (final adding in [true, false]) {
    test(
      'late pack snapshot cannot undo ${adding ? 'add' : 'remove'}',
      () async {
        final repository = _Repository()..packs = adding ? [] : [_pack];
        final pending = Completer<List<StickerMarketPackPayload>>();
        repository.pendingPacks = pending;
        final state = StickerCollection(repository, StickerLibrary(repository));
        addTearDown(state.dispose);
        state.packs = [...repository.packs];
        final loading = state.loadPacks(refresh: true);
        if (adding) {
          await state.addPack(_pack);
        } else {
          await state.removePack(_pack);
        }
        pending.complete(adding ? [] : [_pack]);
        await loading;
        expect(state.hasPack(_pack.id), adding);
      },
    );
  }

  testWidgets('subscribed packs appear in the picker and insert tokens', (
    tester,
  ) async {
    final inserted = <String>[];
    final repository = _Repository()..packs = [_pack];
    await _pump(
      tester,
      repository,
      child: Scaffold(
        body: SizedBox(
          height: 400,
          child: StickerPicker(onInsert: inserted.add),
        ),
      ),
    );
    await tester.tap(find.widgetWithText(TextButton, 'Packs'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(ChoiceChip, 'Pack One'), findsOneWidget);
    await tester.tap(_pickerSticker('Pack One 1'));
    await tester.pumpAndSettle();
    expect(inserted, [_marketItem.token]);
  });

  testWidgets('the market adds a pack only after the server confirms', (
    tester,
  ) async {
    final repository = _Repository();
    final state = await _pump(
      tester,
      repository,
      child: Builder(
        builder: (context) => Scaffold(
          body: Center(
            child: TextButton(
              onPressed: () => openStickerMarket(context),
              child: const Text('open'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();
    expect(find.text('Pack One'), findsOneWidget);
    expect(find.text('1 sticker'), findsOneWidget);
    await tester.tap(find.widgetWithText(GfButton, 'Add'));
    await tester.pumpAndSettle();
    expect(repository.savedPacks, [9201]);
    expect(state.packs.map((pack) => pack.id), [9201]);
    expect(find.widgetWithText(GfButton, 'Remove'), findsOneWidget);

    await tester.tap(find.text('Pack One'));
    await tester.pumpAndSettle();
    expect(find.text('Pack One 1'), findsOneWidget);
    await tester.tap(find.widgetWithText(GfButton, 'Remove sticker pack'));
    await tester.pumpAndSettle();
    expect(state.packs, isEmpty);
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
      // 访客不触碰账号态表情库:不发 my-stickers 请求(401 会触发全局拆会话)。
      expect(repository.mineCalls, 0);
      expect(find.text('Sign in to save'), findsOneWidget);
      expect(find.text('Save to my stickers'), findsNothing);

      await tester.tap(find.text('Sign in to save'));
      await tester.pumpAndSettle();
      expect(find.text('login page'), findsOneWidget);
      expect(find.byType(StickerCard), findsNothing);
    });

    testWidgets('router navigation closes the card with its page', (
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

      // The card's sign-in callback must never run on a defunct context. That
      // is guaranteed because go_router rebuilds its pages on navigation and
      // drops root-navigator dialogs with the page that pushed them — a 401
      // teardown navigating to /login closes the card instead of orphaning it.
      router.go('/login');
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

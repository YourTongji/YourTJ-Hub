import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_page.dart';
import 'package:forum_app/src/widgets/stickers/sticker_picker.dart';
import 'package:forum_app/src/widgets/stickers/resolved_sticker_content.dart';
import 'package:forum_app/src/providers.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

const official = StickerItemPayload(
  name: 'smile',
  url: '/smile.png',
  displayName: 'Smile',
  pack: 'Faces',
);
const personal = StickerItemPayload(
  name: 'user_42',
  url: '/private.png',
  displayName: 'Mine',
  isOfficial: false,
);

class _Repository extends StickerRepository {
  _Repository() : super(GfApiClient(dio: Dio(), tokenStorage: _Tokens()));
  List<StickerItemPayload> members = [official];
  List<StickerItemPayload> officialItems = [official];
  bool failOrder = false;
  int resolveFailures = 0;
  Completer<StickerItemPayload>? pendingSave;
  Completer<List<StickerItemPayload>>? pendingMine;
  List<String>? resolvedNames;
  @override
  Future<List<StickerItemPayload>> list() async => [...officialItems];
  @override
  Future<List<StickerItemPayload>> mine() async {
    final pending = pendingMine;
    pendingMine = null;
    return pending == null ? [...members] : pending.future;
  }

  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async {
    resolvedNames = names;
    if (resolveFailures-- > 0) throw StateError('offline');
    return [personal];
  }

  @override
  Future<StickerItemPayload> save({
    String? stickerName,
    String? fileName,
    String? displayName,
  }) async {
    final item = pendingSave == null ? personal : await pendingSave!.future;
    members = [...members.where((entry) => entry.name != item.name), item];
    return item;
  }

  @override
  Future<void> remove(String name) async {
    members.removeWhere((e) => e.name == name);
  }

  @override
  Future<void> reorder(List<String> names) async {
    if (failOrder) throw StateError('stale order');
    members = names
        .map((name) => members.firstWhere((e) => e.name == name))
        .toList();
  }
}

void main() {
  test(
    'busy library rejects every skipped mutation instead of succeeding',
    () async {
      final repository = _Repository()..pendingSave = Completer();
      final collection = StickerCollection(
        repository,
        StickerLibrary(repository),
      );
      addTearDown(collection.dispose);
      await collection.loadMine();
      final pending = collection.save(stickerName: personal.name);
      await expectLater(
        collection.save(stickerName: official.name),
        throwsStateError,
      );
      await expectLater(collection.remove({official.name}), throwsStateError);
      await expectLater(collection.reorder(0, 0), throwsStateError);
      expect(collection.busy, isTrue);
      expect(collection.mine, [official]);
      repository.pendingSave!.complete(personal);
      await pending;
      expect(collection.mine, [official, personal]);
      expect(collection.busy, isFalse);
    },
  );

  testWidgets('picker never confirms a collection skipped by another write', (
    tester,
  ) async {
    final repository = _Repository()..members = [];
    final state = StickerCollection(repository, StickerLibrary(repository));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerCollectionProvider.overrideWith((_) => state)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: SizedBox(height: 300, child: StickerPicker(onInsert: (_) {})),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Official'));
    await tester.pumpAndSettle();
    await tester.longPress(find.text('Smile'));
    await tester.pumpAndSettle();
    expect(find.text('Save to my stickers'), findsOneWidget);
    repository.pendingSave = Completer();
    final pending = state.save(stickerName: personal.name);
    // The action sheet was opened before another surface began its write.
    await tester.tap(find.text('Save to my stickers'));
    await tester.pumpAndSettle();
    expect(find.text('Saved to your stickers'), findsNothing);
    expect(
      find.text('Could not complete this action. Try again.'),
      findsOneWidget,
    );
    repository.pendingSave!.complete(personal);
    await pending;
    expect(state.mine, [personal]);
  });

  testWidgets('sticker management clears private page state across accounts', (
    tester,
  ) async {
    final container = ProviderContainer(
      overrides: [
        stickerCollectionProvider.overrideWith((ref) {
          ref.watch(offlineCacheEpochProvider);
          final repository = _Repository();
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
    await tester.tap(find.widgetWithText(TextButton, 'Select'));
    await tester.pump();
    await tester.tap(find.byType(Checkbox));
    await tester.pump();
    expect(find.text('Remove (1)'), findsOneWidget);
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    expect(find.text('Remove (1)'), findsNothing);
    expect(find.byType(Checkbox), findsNothing);
    expect(find.widgetWithText(TextButton, 'Select'), findsOneWidget);
    expect(find.text('Smile'), findsOneWidget);
  });

  test(
    'selection replacement preserves the rest of a draft and never sends',
    () {
      final controller = TextEditingController(text: 'hello selected tail');
      addTearDown(controller.dispose);
      insertStickerText(
        controller,
        personal.token,
        selection: const TextSelection(baseOffset: 6, extentOffset: 14),
      );
      expect(controller.text, 'hello ${personal.token} tail');
      expect(controller.selection.baseOffset, 6 + personal.token.length);
      insertStickerText(controller, official.token);
      expect(controller.text, 'hello ${personal.token}${official.token} tail');
    },
  );
  test(
    'shared personal assets resolve without exposing them as official',
    () async {
      final repository = _Repository();
      final library = StickerLibrary(repository);
      await library.resolveContent('[:sticker:smile:] ${personal.token}');
      expect(repository.resolvedNames, ['user_42']);
      expect(library.urlByName['user_42'], '/private.png');
      expect(await library.load(), [official]);
    },
  );
  test('remove membership preserves historical asset resolution', () async {
    final repository = _Repository()..members = [personal];
    final library = StickerLibrary(repository);
    final collection = StickerCollection(repository, library);
    addTearDown(collection.dispose);
    await collection.loadMine();
    await collection.remove({'user_42'});
    expect(collection.mine, isEmpty);
    expect(library.urlByName['user_42'], '/private.png');
  });
  test('failed stale reorder refreshes concurrent membership', () async {
    final repository = _Repository()..members = [official, personal];
    final collection = StickerCollection(
      repository,
      StickerLibrary(repository),
    );
    addTearDown(collection.dispose);
    await collection.loadMine();
    repository.members = [official];
    repository.failOrder = true;
    await expectLater(collection.reorder(0, 1), throwsStateError);
    expect(collection.mine, [official]);
    expect(collection.busy, isFalse);
  });
  test(
    'old-session save completion never populates a disposed library',
    () async {
      final repository = _Repository()..pendingSave = Completer();
      final collection = StickerCollection(
        repository,
        StickerLibrary(repository),
      );
      final future = collection.save(stickerName: personal.name);
      collection.dispose();
      repository.pendingSave!.complete(personal);
      await future;
      expect(collection.mine, isEmpty);
      expect(collection.library.urlByName, isEmpty);
    },
  );
  test(
    'recent is deduplicated and isolated between account/site instances',
    () {
      final repo = _Repository();
      final first = StickerCollection(repo, StickerLibrary(repo));
      final second = StickerCollection(repo, StickerLibrary(repo));
      addTearDown(first.dispose);
      addTearDown(second.dispose);
      first.used(personal);
      first.used(official);
      first.used(personal);
      expect(first.recent, [personal, official]);
      expect(second.recent, isEmpty);
    },
  );

  test(
    'a stale library fetch cannot overwrite a just-saved membership',
    () async {
      final repo = _Repository()..pendingMine = Completer();
      final pendingMine = repo.pendingMine!;
      final state = StickerCollection(repo, StickerLibrary(repo));
      addTearDown(state.dispose);
      final fetch = state.loadMine();
      await state.save(stickerName: personal.name);
      pendingMine.complete([]);
      await fetch;
      expect(state.mine, [official, personal]);
      expect(state.mineLoaded, isTrue);
    },
  );

  test(
    'disabled official membership survives while resolution and recent are removed',
    () async {
      final repo = _Repository();
      final library = StickerLibrary(repo);
      final state = StickerCollection(repo, library);
      addTearDown(state.dispose);
      await state.loadOfficial();
      state.used(official);
      expect(library.urlByName, contains('smile'));
      repo.officialItems = [];
      repo.members = [
        const StickerItemPayload(
          name: 'smile',
          url: '/smile.png',
          isEnabled: false,
        ),
      ];
      await state.loadOfficial(refresh: true);
      await state.loadMine();
      expect(state.mine.single.isEnabled, isFalse);
      expect(state.recent, isEmpty);
      expect(library.urlByName, isNot(contains('smile')));
    },
  );

  test('old servers default stickers to enabled official items', () {
    final item = StickerItemPayload.fromJson({
      'name': 'smile',
      'url': '/smile.png',
    });
    expect(item.isEnabled, isTrue);
    expect(item.isOfficial, isTrue);
  });

  testWidgets('picker remains usable on narrow enlarged-text windows', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 500);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final repository = _Repository();
    final state = StickerCollection(repository, StickerLibrary(repository));
    await state.loadOfficial();
    await state.loadMine();
    final inserted = <String>[];
    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerCollectionProvider.overrideWith((_) => state)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: MediaQuery(
              data: const MediaQueryData(
                size: Size(320, 500),
                textScaler: TextScaler.linear(2),
              ),
              child: SizedBox(
                height: 220,
                child: StickerPicker(onInsert: inserted.add),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.text('Smile').last);
    await tester.pump();
    await tester.tap(find.text('Smile').last);
    expect(inserted, [official.token]);
  });

  testWidgets('failed content resolution offers an explicit retry', (
    tester,
  ) async {
    final repository = _Repository()..resolveFailures = 1;
    final library = StickerLibrary(repository);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerLibraryProvider.overrideWithValue(library)],
        child: MaterialApp(
          home: Scaffold(
            body: ResolvedStickerContent(
              content: personal.token,
              builder: (urls) => Text(urls[personal.name] ?? 'unresolved'),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('unresolved'), findsOneWidget);
    await tester.tap(find.widgetWithText(TextButton, 'Retry'));
    await tester.pumpAndSettle();
    expect(find.text('/private.png'), findsOneWidget);
  });

  testWidgets('mounted content follows resolved-library changes', (
    tester,
  ) async {
    final library = StickerLibrary(_Repository());
    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerLibraryProvider.overrideWithValue(library)],
        child: MaterialApp(
          home: ResolvedStickerContent(
            content: personal.token,
            builder: (urls) => Text(urls[personal.name] ?? 'unresolved'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('/private.png'), findsOneWidget);
    library.remember([
      StickerItemPayload(
        name: personal.name,
        url: personal.url,
        isEnabled: false,
      ),
    ]);
    await tester.pump();
    expect(find.text('unresolved'), findsOneWidget);
  });
}

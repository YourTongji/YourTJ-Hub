import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/stickers/resolved_sticker_content.dart';

import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Stickers extends StickerRepository {
  _Stickers()
    : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage()));
  final requests = <List<String>>[];
  bool fail = false;
  @override
  Future<List<StickerItemPayload>> list() async => [];
  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async {
    requests.add(names);
    if (fail) throw const NetworkException(fallbackMessage: 'offline');
    return [];
  }
}

void main() {
  for (final fail in [false, true]) {
    testWidgets(
      'ordinary typing does not re-resolve missing stickers fail=$fail',
      (tester) async {
        final repository = _Stickers()..fail = fail;
        final library = StickerLibrary(repository);
        final container = ProviderContainer(
          overrides: [stickerLibraryProvider.overrideWithValue(library)],
        );
        addTearDown(container.dispose);
        addTearDown(library.dispose);
        Future<void> render(String text) async {
          await tester.pumpWidget(
            UncontrolledProviderScope(
              container: container,
              child: MaterialApp(
                home: Scaffold(
                  body: ResolvedStickerContent(
                    content: text,
                    builder: (_) => Text(text),
                  ),
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
        }

        await render('[:sticker:missing:]');
        await render('[:sticker:missing:] h');
        await render('[:sticker:missing:] hello');
        expect(repository.requests, [
          ['missing'],
        ]);
        if (fail) {
          await tester.tap(find.text('Retry'));
          await tester.pumpAndSettle();
          expect(repository.requests.length, 2);
        }
        await render('[:sticker:another:] hello');
        expect(repository.requests.last, ['another']);
        expect(repository.requests.length, fail ? 3 : 2);
      },
    );
  }
}

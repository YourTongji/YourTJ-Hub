import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/images/image_upload.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/publish/publish_page.dart';
import 'package:forum_app/src/pages/publish/publish_type.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:image_picker/image_picker.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'fixtures/sticker_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Picker extends ImagePicker {
  int multiCalls = 0;
  int? limit;
  final files = [
    XFile.fromData(
      Uint8List.fromList([1]),
      name: 'first.jpg',
      path: 'first.jpg',
    ),
    XFile.fromData(
      Uint8List.fromList([2]),
      name: 'second.jpg',
      path: 'second.jpg',
    ),
  ];
  Completer<List<XFile>>? pending;

  @override
  Future<List<XFile>> pickMultiImage({
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    int? limit,
    bool requestFullMetadata = true,
  }) async {
    multiCalls++;
    this.limit = limit;
    return pending == null ? files : pending!.future;
  }

  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) async => files.first;
}

class _Files extends FileRepository {
  _Files(super.client);
  final names = <String>[];
  final results = <Completer<String>>[];

  @override
  Future<String> uploadImage({
    required List<int> bytes,
    required String filename,
  }) {
    names.add(filename);
    final result = Completer<String>();
    results.add(result);
    return result.future;
  }
}

/// 捕获真实 [FileRepository] 上送的 multipart 文件名，让 core 的改名收敛点
/// （issue #969）在页面真实链路里被断言。
class _CaptureUploadsAdapter implements HttpClientAdapter {
  final filenames = <String>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final form = options.data as FormData;
    filenames.add(form.files.single.value.filename!);
    return ResponseBody.fromString(
      jsonEncode({'code': 0, 'result': 'https://example.com/photo.jpg'}),
      200,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

class _Pages extends PageRepository {
  _Pages(super.client);
  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async =>
      PagePayload.fromJson({
        'component': PageComponent.publish,
        'props': {
          'topicId': 0,
          'isEditing': false,
          'categories': [
            {'id': 1, 'name': '校园', 'color': '#2563eb'},
          ],
          'topic': {
            'title': '',
            'content': '',
            'categoryIds': [],
            'topicStatus': 0,
          },
        },
        'meta': {'title': '发布'},
        'layout': minimalLayoutJson(),
        'url': '/publish',
        'version': '1',
      });
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  late ProviderContainer container;
  late _Files files;
  late _Picker picker;
  const scope = 'http%3A%2F%2Ffake.local:1';

  Future<void> pumpPage(
    WidgetTester tester, {
    int type = 1,
    String? draftKey,
    Locale locale = const Locale('zh'),
    double textScale = 1,
    FileRepository? uploads,
  }) async {
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'http://fake.local',
    );
    files = _Files(client);
    picker = _Picker();
    final stickers = ComposerStickerRepository(client);
    final library = StickerLibrary(stickers);
    addTearDown(library.dispose);
    container = ProviderContainer(
      overrides: [
        apiClientProvider.overrideWithValue(client),
        currentUserProvider.overrideWith(
          (ref) async => const CurrentUser(id: 1, username: 'alice'),
        ),
        pageRepositoryProvider.overrideWithValue(_Pages(client)),
        fileRepositoryProvider.overrideWithValue(uploads ?? files),
        imagePickerProvider.overrideWithValue(picker),
        stickerLibraryProvider.overrideWithValue(library),
        stickerCollectionProvider.overrideWith(
          (ref) => StickerCollection(stickers, library),
        ),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: locale,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: TextScaler.linear(textScale)),
            child: child!,
          ),
          home: PublishPage(initialContentType: type, localDraftKey: draftKey),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  Future<void> selectImages(WidgetTester tester) async {
    final l10n = AppLocalizations.of(tester.element(find.byType(PublishPage)));
    final gallery = find.byKey(const Key('publish-gallery-add'));
    if (gallery.evaluate().isNotEmpty) {
      await tester.ensureVisible(gallery);
      await tester.pumpAndSettle();
      await tester.tap(gallery);
    } else {
      await tester.tap(find.byTooltip(l10n.publishToolImage));
    }
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    await tester.tap(find.text(l10n.publishPhotoLibrary));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pump();
  }

  testWidgets(
    'sticker caret follows an image upload while the picker stays open',
    (tester) async {
      await pumpPage(tester, type: 3);
      final editor = tester
          .widget<QuillEditor>(find.byType(QuillEditor))
          .controller;
      editor.replaceText(
        0,
        0,
        'before\nafter',
        const TextSelection.collapsed(offset: 7),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('表情库'));
      await tester.pumpAndSettle();
      await selectImages(tester);
      files.results.first.complete('https://example.com/first.jpg');
      await tester.pump();
      await tester.pump();
      files.results.last.complete('https://example.com/second.jpg');
      await tester.pumpAndSettle();
      final caret = editor.selection.baseOffset;
      expect(caret, greaterThan(7));
      final before = editor.document.toPlainText();
      await tester.ensureVisible(find.text('Smile').last);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Smile').last);
      await tester.pumpAndSettle();
      expect(
        editor.document.toPlainText(),
        before.replaceRange(caret, caret, '[:sticker:smile:]'),
      );
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(seconds: 1));
    },
  );

  testWidgets('gallery selection starts a bounded multi-image queue', (
    tester,
  ) async {
    await pumpPage(tester);
    await selectImages(tester);
    expect(picker.multiCalls, 1);
    expect(picker.limit, 9);
    expect(files.names, ['first.jpg']);
    expect(find.text('second.jpg'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('re-encoded png bytes upload under the sniffed jpg name', (
    tester,
  ) async {
    final adapter = _CaptureUploadsAdapter();
    final uploads = FileRepository(
      GfApiClient(
        dio: Dio()..httpClientAdapter = adapter,
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'http://fake.local',
      ),
    );
    await pumpPage(tester, uploads: uploads);
    picker.files
      ..clear()
      ..add(
        XFile.fromData(
          Uint8List.fromList([0xFF, 0xD8, 0xFF, 0xE0]),
          name: 'photo.png',
          path: 'photo.png',
        ),
      );
    await selectImages(tester);
    expect(adapter.filenames, ['photo.jpg']);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets(
    'retry keeps photo order and gallery edits survive draft recovery',
    (tester) async {
      await pumpPage(tester);
      await selectImages(tester);
      files.results.first.completeError(
        const NetworkException(fallbackMessage: 'offline'),
      );
      await tester.pumpAndSettle();
      expect(files.names, ['first.jpg']);
      expect(
        tester
            .widget<PublishTypeSwitcher>(find.byType(PublishTypeSwitcher))
            .onChanged,
        isNull,
      );
      expect(
        tester
            .widget<PopScope>(
              find.byWidgetPredicate((widget) => widget is PopScope),
            )
            .canPop,
        isFalse,
      );
      final l10n = AppLocalizations.of(
        tester.element(find.byType(PublishPage)),
      );
      await tester.tap(find.byTooltip(l10n.commonBack));
      await tester.pump();
      expect(find.text(l10n.publishMediaPendingWarning), findsWidgets);
      expect(find.byType(PublishPage), findsOneWidget);
      await tester.tap(
        find.descendant(
          of: find.byKey(const Key('publish-upload-queue')),
          matching: find.byTooltip('重试'),
        ),
      );
      await tester.pump();
      expect(picker.multiCalls, 1);
      files.results.last.complete('https://example.com/first.jpg');
      await tester.pump();
      await tester.pump();
      expect(files.names, ['first.jpg', 'first.jpg', 'second.jpg']);
      final partial = await container.read(writingStoreProvider).drafts(scope);
      expect(partial.single.images, ['https://example.com/first.jpg']);
      files.results.last.complete('https://example.com/second.jpg');
      await tester.pumpAndSettle();
      final gallery = tester.widget<ReorderableListView>(
        find.byType(ReorderableListView),
      );
      gallery.onReorderItem!(0, 1);
      await tester.pump(const Duration(milliseconds: 800));
      await tester.pumpAndSettle();
      final saved =
          (await container.read(writingStoreProvider).drafts(scope)).single;
      expect(saved.images, [
        'https://example.com/second.jpg',
        'https://example.com/first.jpg',
      ]);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
      await pumpPage(tester, draftKey: saved.key);
      expect(find.byType(ReorderableListView), findsOneWidget);
      final remove = find.byTooltip('移除图片').first;
      await tester.ensureVisible(remove);
      await tester.tap(remove);
      await tester.pump(const Duration(milliseconds: 800));
      await tester.pumpAndSettle();
      expect(
        (await container.read(writingStoreProvider).drafts(scope))
            .single
            .images,
        ['https://example.com/first.jpg'],
      );
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );

  testWidgets('removing an in-flight photo ignores its late upload result', (
    tester,
  ) async {
    await pumpPage(tester);
    await selectImages(tester);
    await tester.tap(find.byTooltip('移除图片').first);
    files.results.first.complete('https://example.com/removed.jpg');
    await tester.pump();
    await tester.pump();
    expect(files.names, ['first.jpg', 'second.jpg']);
    files.results.last.complete('https://example.com/kept.jpg');
    await tester.pumpAndSettle();
    expect(
      (await container.read(writingStoreProvider).drafts(scope)).single.images,
      ['https://example.com/kept.jpg'],
    );
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('session changes fence uploads and queued photo selections', (
    tester,
  ) async {
    await pumpPage(tester);
    await selectImages(tester);
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    files.results.first.complete('https://example.com/old-account.jpg');
    await tester.pumpAndSettle();
    expect(files.names, ['first.jpg']);
    expect(await container.read(writingStoreProvider).drafts(scope), isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
    await pumpPage(tester);
    picker.pending = Completer<List<XFile>>();
    await selectImages(tester);
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    picker.pending!.complete(picker.files);
    await tester.pumpAndSettle();
    expect(files.names, isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets(
    'pickers ignoring the nine-image limit cannot silently drop photos',
    (tester) async {
      await pumpPage(tester);
      picker.files.addAll(
        List.generate(
          8,
          (i) => XFile.fromData(Uint8List(1), name: '$i.jpg', path: '$i.jpg'),
        ),
      );
      await selectImages(tester);
      expect(files.names, isEmpty);
      expect(find.byKey(const Key('publish-upload-queue')), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );

  testWidgets('article photos can be added while earlier ones upload', (
    tester,
  ) async {
    await pumpPage(tester, type: 3);
    final editor = tester
        .widget<QuillEditor>(find.byType(QuillEditor))
        .controller;
    editor.document.insert(0, 'before\nafter\n');
    editor.updateSelection(
      const TextSelection.collapsed(offset: 7),
      ChangeSource.local,
    );
    await tester.pumpAndSettle();
    await selectImages(tester);
    expect(find.byKey(const Key('publish-upload-queue')), findsOneWidget);
    // The first batch is still uploading; a second batch at the end of the
    // body must not wait for it or reuse its insertion point.
    editor.updateSelection(
      TextSelection.collapsed(offset: editor.document.length - 1),
      ChangeSource.local,
    );
    await selectImages(tester);
    expect(picker.multiCalls, 2);
    for (var i = 0; i < 4; i++) {
      files.results[i].complete('https://example.com/$i.jpg');
      await tester.pump();
      await tester.pump();
    }
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('publish-upload-queue')), findsNothing);
    final saved =
        (await container.read(writingStoreProvider).drafts(scope)).single;
    final content = saved.content;
    int at(String needle) => content.indexOf(needle);
    expect(at('before'), lessThan(at('0.jpg')));
    expect(at('0.jpg'), lessThan(at('1.jpg')));
    expect(at('1.jpg'), lessThan(at('after')));
    expect(at('after'), lessThan(at('2.jpg')));
    expect(at('2.jpg'), lessThan(at('3.jpg')));
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets(
    'article queue retains its insertion point through ongoing edits',
    (tester) async {
      await pumpPage(tester, type: 3);
      final editor = tester
          .widget<QuillEditor>(find.byType(QuillEditor))
          .controller;
      editor.document.insert(0, 'before\nafter\n');
      editor.updateSelection(
        const TextSelection.collapsed(offset: 7),
        ChangeSource.local,
      );
      await tester.pumpAndSettle();
      await selectImages(tester);
      editor.document.insert(0, 'intro\n');
      await tester.pump();
      files.results.first.complete('https://example.com/first.jpg');
      await tester.pump();
      await tester.pump();
      files.results.last.complete('https://example.com/second.jpg');
      await tester.pumpAndSettle();
      final saved =
          (await container.read(writingStoreProvider).drafts(scope)).single;
      expect(
        saved.content.indexOf('intro'),
        lessThan(saved.content.indexOf('before')),
      );
      expect(
        saved.content.indexOf('before'),
        lessThan(saved.content.indexOf('first.jpg')),
      );
      expect(
        saved.content.indexOf('first.jpg'),
        lessThan(saved.content.indexOf('second.jpg')),
      );
      expect(
        saved.content.indexOf('second.jpg'),
        lessThan(saved.content.indexOf('after')),
      );
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );

  testWidgets(
    'queue failure controls fit a narrow large-text keyboard layout',
    (tester) async {
      tester.view.physicalSize = const Size(320, 700);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await pumpPage(tester, locale: const Locale('de'), textScale: 1.6);
      await selectImages(tester);
      files.results.first.completeError(
        const NetworkException(fallbackMessage: 'offline'),
      );
      await tester.pumpAndSettle();
      tester.view.viewInsets = const FakeViewPadding(bottom: 260);
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      final l10n = AppLocalizations.of(
        tester.element(find.byType(PublishPage)),
      );
      final remove = find.byTooltip(l10n.publishRemoveImage).last;
      await tester.ensureVisible(remove);
      await tester.tap(remove);
      await tester.pumpAndSettle();
      expect(find.text('second.jpg'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );
}

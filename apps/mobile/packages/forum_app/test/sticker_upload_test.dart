import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/images/image_upload.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_page.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:image_picker/image_picker.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Photos extends ImagePicker {
  int calls = 0;
  XFile? file = XFile.fromData(
    Uint8List.fromList([71, 73, 70]),
    name: 'dance.gif',
    path: 'dance.gif',
  );
  Completer<XFile?>? pending;

  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) async {
    calls++;
    expect(source, ImageSource.gallery);
    expect(maxWidth, isNull);
    expect(maxHeight, isNull);
    expect(imageQuality, isNull);
    expect(requestFullMetadata, isFalse);
    return pending == null ? file : pending!.future;
  }
}

class _Files extends FileRepository {
  _Files(super.client);
  final uploads = <({String name, List<int> bytes})>[];
  bool fail = false;

  @override
  Future<String> uploadImage({
    required List<int> bytes,
    required String filename,
  }) async {
    uploads.add((name: filename, bytes: bytes));
    if (fail) throw StateError('offline');
    return '/uploaded.gif';
  }
}

class _Stickers extends StickerRepository {
  _Stickers(super.client);
  final saves = <({String? file, String? label})>[];
  bool fail = false;
  @override
  Future<List<StickerItemPayload>> mine() async => [];
  @override
  Future<StickerItemPayload> save({
    String? stickerName,
    String? fileName,
    String? displayName,
  }) async {
    saves.add((file: fileName, label: displayName));
    if (fail) throw StateError('offline');
    return StickerItemPayload(
      name: 'user_1',
      url: fileName!,
      displayName: displayName!,
      isOfficial: false,
    );
  }
}

void main() {
  late _Photos photos;
  late _Files files;
  late _Stickers stickers;
  late ProviderContainer container;

  Future<void> pumpPage(WidgetTester tester) async {
    final client = GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage());
    photos = _Photos();
    files = _Files(client);
    stickers = _Stickers(client);
    container = ProviderContainer(
      overrides: [
        imagePickerProvider.overrideWithValue(photos),
        fileRepositoryProvider.overrideWithValue(files),
        stickerCollectionProvider.overrideWith((ref) {
          final library = StickerLibrary(stickers);
          ref.onDispose(library.dispose);
          return StickerCollection(stickers, library);
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
  }

  Future<void> choosePhotos(WidgetTester tester) async {
    await tester.tap(find.text('Add sticker'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    await tester.tap(find.text('Choose from photos'));
    // A pending native picker keeps the progress indicator animating.
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
  }

  testWidgets('add sticker offers photos and files before opening a picker', (
    tester,
  ) async {
    await pumpPage(tester);
    await tester.tap(find.text('Add sticker'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('Choose from photos'), findsOneWidget);
    expect(find.text('Choose from files'), findsOneWidget);
    expect(photos.calls, 0);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(files.uploads, isEmpty);
    expect(stickers.saves, isEmpty);
  });

  testWidgets(
    'photo picker uploads GIF bytes through the existing file service',
    (tester) async {
      await pumpPage(tester);
      await choosePhotos(tester);
      await tester.pumpAndSettle();
      expect(photos.calls, 1);
      expect(files.uploads.single.name, 'dance.gif');
      expect(files.uploads.single.bytes, [71, 73, 70]);
      expect(stickers.saves.single, (file: '/uploaded.gif', label: 'dance'));
      expect(find.text('dance'), findsOneWidget);
    },
  );

  testWidgets(
    'cancelled photo selection does not upload and allows another try',
    (tester) async {
      await pumpPage(tester);
      photos.file = null;
      await choosePhotos(tester);
      await tester.pumpAndSettle();
      expect(files.uploads, isEmpty);
      expect(stickers.saves, isEmpty);
      expect(
        find.text('Could not complete this action. Try again.'),
        findsNothing,
      );
      await tester.tap(find.text('Add sticker'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
      expect(find.text('Choose from photos'), findsOneWidget);
    },
  );

  testWidgets('oversized photos are rejected before upload', (tester) async {
    await pumpPage(tester);
    photos.file = XFile.fromData(
      Uint8List(4 * 1024 * 1024 + 1),
      name: 'large.png',
      path: 'large.png',
    );
    await choosePhotos(tester);
    await tester.pumpAndSettle();
    expect(files.uploads, isEmpty);
    expect(stickers.saves, isEmpty);
    expect(find.text('Choose an image or GIF up to 4 MB'), findsOneWidget);
    expect(find.text('Retry'), findsNothing);
  });

  for (final failUpload in [true, false]) {
    testWidgets(
      'retry retains photo and uploaded URL, upload failure=$failUpload',
      (tester) async {
        await pumpPage(tester);
        files.fail = failUpload;
        stickers.fail = !failUpload;
        await choosePhotos(tester);
        await tester.pumpAndSettle();
        expect(find.text('Retry'), findsOneWidget);
        files.fail = false;
        stickers.fail = false;
        await tester.tap(find.text('Retry'));
        await tester.pumpAndSettle();
        expect(photos.calls, 1);
        expect(files.uploads.length, failUpload ? 2 : 1);
        expect(stickers.saves.last, (file: '/uploaded.gif', label: 'dance'));
        expect(find.text('Retry'), findsNothing);
      },
    );
  }

  testWidgets('account change discards a pending photo selection', (
    tester,
  ) async {
    await pumpPage(tester);
    final pending = photos.pending = Completer<XFile?>();
    await choosePhotos(tester);
    expect(photos.calls, 1);
    container.invalidate(stickerCollectionProvider);
    await tester.pumpAndSettle();
    pending.complete(photos.file);
    await tester.pumpAndSettle();
    expect(files.uploads, isEmpty);
    expect(stickers.saves, isEmpty);
  });

  testWidgets('file selection remains available and cancellation is harmless', (
    tester,
  ) async {
    await pumpPage(tester);
    const channel = MethodChannel('plugins.flutter.io/file_selector');
    int fileCalls = 0;
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
      call,
    ) async {
      expect(call.method, 'openFile');
      expect(call.arguments['multiple'], isFalse);
      fileCalls++;
      return null;
    });
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        channel,
        null,
      ),
    );
    await tester.tap(find.text('Add sticker'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    await tester.tap(find.text('Choose from files'));
    await tester.pumpAndSettle();
    expect(fileCalls, 1);
    expect(photos.calls, 0);
    expect(files.uploads, isEmpty);
    expect(stickers.saves, isEmpty);
  });
}

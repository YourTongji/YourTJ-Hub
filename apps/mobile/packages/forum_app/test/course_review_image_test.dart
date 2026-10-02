import 'dart:convert';
import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/images/image_upload.dart';
import 'package:forum_app/src/pages/courses/review_form_sheet.dart';
import 'package:forum_app/src/providers.dart';
import 'package:image_picker/image_picker.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';

import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Photos extends ImagePicker {
  XFile? file = XFile.fromData(_png, name: 'photo.png', path: 'photo.png');
  int calls = 0;

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
    return file;
  }
}

class _Files extends FileRepository {
  _Files() : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage()));

  int calls = 0;
  String? filename;
  Object? error;

  @override
  Future<String> uploadImage({
    required List<int> bytes,
    required String filename,
  }) async {
    calls++;
    this.filename = filename;
    expect(bytes, orderedEquals(_png));
    if (error != null) throw error!;
    return '/file/img/photo.png';
  }
}

final Uint8List _png = Uint8List.fromList(
  img.encodePng(img.Image(width: 4, height: 4, numChannels: 4)),
);

void main() {
  late _Photos photos;
  late _Files files;

  setUp(() {
    photos = _Photos();
    files = _Files();
  });

  Future<BuildContext> pumpSheet(WidgetTester tester) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.reset);
    late BuildContext page;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          imagePickerProvider.overrideWithValue(photos),
          fileRepositoryProvider.overrideWithValue(files),
        ],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          home: Scaffold(
            body: Builder(
              builder: (context) {
                page = context;
                return const SizedBox();
              },
            ),
          ),
        ),
      ),
    );
    showGfBottomSheet<void>(
      page,
      height: 600,
      keyboardAware: true,
      builder: (_) => CourseReviewFormSheet(
        pageContext: page,
        offerings: const <CourseOfferingPayload>[],
        repository: CourseRepository(
          GfApiClient(
            dio: Dio(),
            tokenStorage: MemoryTokenStorage(),
            baseUrl: 'http://fake.local',
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return page;
  }

  String documentJson(WidgetTester tester) {
    final QuillEditor editor = tester.widget<QuillEditor>(
      find.byType(QuillEditor),
    );
    return jsonEncode(editor.controller.document.toDelta().toJson());
  }

  testWidgets('review editor uploads and embeds an image', (tester) async {
    final BuildContext page = await pumpSheet(tester);
    final AppLocalizations l10n = AppLocalizations.of(page);

    expect(find.byTooltip(l10n.publishToolImage), findsOneWidget);
    await tester.tap(find.byTooltip(l10n.publishToolImage));
    await tester.pumpAndSettle();

    expect(photos.calls, 1);
    expect(files.calls, 1);
    expect(files.filename, 'photo.png');
    expect(documentJson(tester), contains('"image":"/file/img/photo.png"'));
    expect(find.byType(GfNetworkImage), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('failed image upload keeps the editor usable', (tester) async {
    files.error = const ApiException(fallbackMessage: 'upload failed');
    final BuildContext page = await pumpSheet(tester);
    final AppLocalizations l10n = AppLocalizations.of(page);

    await tester.tap(find.byTooltip(l10n.publishToolImage));
    await tester.pumpAndSettle();

    expect(files.calls, 1);
    expect(documentJson(tester), isNot(contains('"image"')));
    expect(find.text(l10n.courseReviewUnknownError), findsOneWidget);
    expect(find.byType(QuillEditor), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
}

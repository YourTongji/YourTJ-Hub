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
  Completer<void>? gate;

  @override
  Future<String> uploadImage({
    required List<int> bytes,
    required String filename,
  }) async {
    calls++;
    this.filename = filename;
    expect(bytes, orderedEquals(_png));
    await gate?.future;
    if (error != null) throw error!;
    return '/file/img/photo.png';
  }
}

class _Reviews extends CourseRepository {
  _Reviews()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );

  final List<String> submitted = <String>[];

  ReviewPayload _echo(int offeringId, int rating, String content) =>
      ReviewPayload(
        id: 7,
        offeringId: offeringId,
        rating: rating,
        content: content,
        contentHtml: '<p>$content</p>',
        author: const ReviewAuthorPayload(kind: 'member', label: 'me'),
        viewer: const ReviewViewerPayload(
          canEdit: true,
          canDelete: true,
          isHelpful: false,
        ),
        helpfulCount: 0,
        createdAt: '2026-09-01T08:00:00+08:00',
        updatedAt: '2026-09-01T08:00:00+08:00',
      );

  @override
  Future<ReviewPayload> createReview(CreateCourseReviewInput input) async {
    submitted.add(input.content);
    return _echo(input.offeringId, input.rating, input.content);
  }

  @override
  Future<ReviewPayload> updateReview(
    int reviewId,
    UpdateCourseReviewInput input,
  ) async {
    submitted.add(input.content ?? '');
    return _echo(901, input.rating ?? 0, input.content ?? '');
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

  Future<BuildContext> pumpSheet(
    WidgetTester tester, {
    CourseRepository? repository,
    ReviewPayload? editing,
  }) async {
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
        initialOfferingId: 901,
        editing: editing,
        repository:
            repository ??
            CourseRepository(
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

  // 上传未完成时发布会提交不含图片的旧正文并关闭 sheet，图片随之丢失；
  // 新建与编辑两条路径都要等上传插入正文后才能提交。
  for (final bool editing in <bool>[false, true]) {
    testWidgets(
      '${editing ? 'editing' : 'new'} review cannot submit while an image uploads',
      (tester) async {
        final _Reviews reviews = _Reviews();
        files.gate = Completer<void>();
        final BuildContext page = await pumpSheet(
          tester,
          repository: reviews,
          editing: editing
              ? const ReviewPayload(
                  id: 7,
                  offeringId: 901,
                  rating: 4,
                  content: '原评价',
                  contentHtml: '<p>原评价</p>',
                  author: ReviewAuthorPayload(kind: 'member', label: 'me'),
                  viewer: ReviewViewerPayload(
                    canEdit: true,
                    canDelete: true,
                    isHelpful: false,
                  ),
                  helpfulCount: 0,
                  createdAt: '2026-09-01T08:00:00+08:00',
                  updatedAt: '2026-09-01T08:00:00+08:00',
                )
              : null,
        );
        final AppLocalizations l10n = AppLocalizations.of(page);
        final Finder submit = find.text(
          editing ? l10n.commonSave : l10n.reviewSubmit,
        );
        if (!editing) {
          final QuillController controller = tester
              .widget<QuillEditor>(find.byType(QuillEditor))
              .controller;
          controller.replaceText(
            0,
            controller.document.length - 1,
            '好课',
            const TextSelection.collapsed(offset: 2),
          );
          await tester.tap(find.byKey(const ValueKey('review-rating-5')));
          await tester.pump();
        }

        await tester.tap(find.byTooltip(l10n.publishToolImage));
        await tester.pump();
        expect(files.calls, 1);

        await tester.tap(submit, warnIfMissed: false);
        await tester.pump();
        expect(reviews.submitted, isEmpty);
        expect(find.byType(CourseReviewFormSheet), findsOneWidget);

        files.gate!.complete();
        await tester.pumpAndSettle();
        expect(documentJson(tester), contains('"image":"/file/img/photo.png"'));

        await tester.tap(submit);
        await tester.pumpAndSettle();
        expect(reviews.submitted, hasLength(1));
        expect(reviews.submitted.single, contains('/file/img/photo.png'));
        expect(tester.takeException(), isNull);

        await tester.pumpWidget(const SizedBox());
        await tester.pumpAndSettle();
      },
    );
  }
}

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/courses/review_form_sheet.dart';
import 'package:forum_app/src/pages/topic/post_edit_sheet.dart';
import 'package:forum_app/src/widgets/language_picker.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  for (final kind in ['review', 'reply', 'language']) {
    testWidgets('$kind sheet fits a small phone with large text and keyboard', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(320, 568);
      tester.view.padding = const FakeViewPadding(top: 24, bottom: 20);
      tester.view.viewPadding = const FakeViewPadding(top: 24, bottom: 20);
      addTearDown(tester.view.reset);
      late BuildContext page;
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('de'),
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(textScaler: const TextScaler.linear(2)),
              child: child!,
            ),
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
      if (kind == 'language') {
        showAppLanguagePicker(page);
      } else {
        final props = topicDetailPayloadJson()['props'] as Map<String, dynamic>;
        final post = PostPayload.fromJson(
          (props['postStream']['posts'] as List).first as Map<String, dynamic>,
        );
        showGfBottomSheet<void>(
          page,
          height: 600,
          keyboardAware: true,
          builder: (_) => kind == 'review'
              ? CourseReviewFormSheet(
                  pageContext: page,
                  offerings: const [],
                  repository: CourseRepository(
                    GfApiClient(
                      dio: Dio(),
                      tokenStorage: MemoryTokenStorage(),
                      baseUrl: 'http://fake.local',
                    ),
                  ),
                )
              : PostEditSheet(post: post),
        );
      }
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      if (kind != 'language') {
        if (kind == 'review') {
          await tester.scrollUntilVisible(
            find.byType(TextField),
            150,
            scrollable: find
                .descendant(
                  of: find.byType(ListView),
                  matching: find.byType(Scrollable),
                )
                .first,
          );
        } else {
          await tester.ensureVisible(find.byType(TextField));
        }
        await tester.enterText(
          find.byType(TextField),
          'Draft survives keyboard',
        );
        tester.view.viewInsets = const FakeViewPadding(bottom: 300);
        tester.view.padding = const FakeViewPadding(top: 24);
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        expect(find.text('Draft survives keyboard'), findsOneWidget);
      }
      final l10n = AppLocalizations.of(page);
      final action = find.text(
        kind == 'language'
            ? '日本語'
            : kind == 'review'
            ? l10n.commonCancel
            : l10n.commonSave,
      );
      await tester.ensureVisible(action);
      await tester.pumpAndSettle();
      expect(action.hitTestable(), findsOneWidget);
      final rect = tester.getRect(action);
      expect(rect.bottom, lessThanOrEqualTo(kind == 'language' ? 548 : 268));
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }
}

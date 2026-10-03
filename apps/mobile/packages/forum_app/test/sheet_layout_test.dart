import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/courses/review_form_sheet.dart';
import 'package:forum_app/src/pages/topic/post_edit_sheet.dart';
import 'package:forum_app/src/widgets/language_picker.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

/// 课评写评/编辑 sheet 的定位：面板本身（Material）。
Finder _panel() => find
    .ancestor(
      of: find.byType(CourseReviewFormSheet),
      matching: find.byType(Material),
    )
    .first;

Future<BuildContext> _pumpReviewSheet(
  WidgetTester tester, {
  Size size = const Size(390, 844),
  double textScale = 1,
  Locale locale = const Locale('zh'),
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.reset);
  late BuildContext page;
  await tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        theme: gfThemeData(Brightness.light),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: locale,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(textScale)),
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
  showGfBottomSheet<void>(
    page,
    height: 600,
    keyboardAware: true,
    builder: (_) => CourseReviewFormSheet(
      pageContext: page,
      offerings: const [],
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
          // 编辑器优先结构：正文直接挂在面板的 Expanded 里，
          // 不再需要先滚动内层 ListView 才能碰到编辑器。
          final editor = find.byType(QuillEditor);
          expect(editor, findsOneWidget);
          final quill = tester.widget<QuillEditor>(editor).controller;
          quill.replaceText(
            0,
            quill.document.length - 1,
            'Draft survives keyboard',
            const TextSelection.collapsed(offset: 23),
          );
        } else {
          await tester.ensureVisible(find.byType(TextField));
          await tester.enterText(
            find.byType(TextField),
            'Draft survives keyboard',
          );
        }
        tester.view.viewInsets = const FakeViewPadding(bottom: 300);
        tester.view.padding = const FakeViewPadding(top: 24);
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        if (kind == 'review') {
          // 面板整体仍在视口内（键盘不会把面板顶部顶出屏幕）。
          expect(tester.getRect(_panel()).top, greaterThanOrEqualTo(0));
          expect(
            tester
                .widget<QuillEditor>(find.byType(QuillEditor))
                .controller
                .document
                .toPlainText(),
            contains('Draft survives keyboard'),
          );
        } else {
          expect(find.text('Draft survives keyboard'), findsOneWidget);
        }
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

  testWidgets('review sheet editor keeps its toolbar and actions pinned', (
    tester,
  ) async {
    await _pumpReviewSheet(tester);
    final l10n = AppLocalizations.of(
      tester.element(find.byType(CourseReviewFormSheet)),
    );

    final toolbar = find.byKey(const Key('rich-markdown-toolbar'));
    final cancel = find.text(l10n.commonCancel);
    final editor = find.byType(QuillEditor);
    expect(toolbar, findsOneWidget);
    final beforeToolbar = tester.getRect(toolbar);
    final beforeCancel = tester.getRect(cancel);
    final beforeEditor = tester.getSize(editor);

    // 500+ 字符的长正文：正文在自身内部滚动，工具栏与底部动作不漂移。
    final quill = tester.widget<QuillEditor>(editor).controller;
    final long = List<String>.generate(
      60,
      (index) => '第 $index 行评价内容',
    ).join('\n');
    expect(long.length, greaterThan(500));
    quill.replaceText(
      0,
      quill.document.length - 1,
      long,
      TextSelection.collapsed(offset: long.length),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(tester.getRect(toolbar), beforeToolbar);
    expect(tester.getRect(cancel), beforeCancel);
    expect(tester.getSize(editor), beforeEditor);
    // 正文高度不变说明它自己滚动了，而不是把面板撑高。
    expect(tester.getSize(editor).height, beforeEditor.height);
    expect(tester.widget<QuillEditor>(editor).config.scrollable, isTrue);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('review sheet shares one row between offering and rating', (
    tester,
  ) async {
    await _pumpReviewSheet(tester);

    final Finder chip = find.byKey(const Key('course-review-offering'));
    final Finder star5 = find.byKey(const ValueKey<String>('review-rating-5'));
    expect(chip, findsOneWidget);
    expect(star5, findsOneWidget);

    final Rect chipRect = tester.getRect(chip);
    final Rect starRect = tester.getRect(star5);
    // 同一行：垂直中线一致（行内居中），chip 在左、星级在右，没有孤行控件。
    expect(
      (chipRect.center.dy - starRect.center.dy).abs(),
      lessThanOrEqualTo(2),
    );
    expect(chipRect.left, lessThan(starRect.left));
    expect(chipRect.right, lessThanOrEqualTo(starRect.left + 1));
    // 身份行在下一行，完整文案仍在。
    expect(tester.getRect(find.text('匿名发布')).top, greaterThan(chipRect.bottom));
    expect(find.text('对公众隐藏身份'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('review sheet stacks offering above stars when narrow', (
    tester,
  ) async {
    // 320dp：放不下「240dp 星级 + 可读 chip」时显式降级两行——chip 在上、
    // 星级在下一行贴右，不再靠 Wrap 的偶然折行。
    await _pumpReviewSheet(tester, size: const Size(320, 844));

    final Finder chip = find.byKey(const Key('course-review-offering'));
    final Finder star5 = find.byKey(const ValueKey<String>('review-rating-5'));
    expect(chip, findsOneWidget);
    expect(star5, findsOneWidget);
    final Rect chipRect = tester.getRect(chip);
    final Rect starRect = tester.getRect(star5);
    expect(starRect.center.dy, greaterThan(chipRect.center.dy + 8));
    // chip 贴左、星级行贴右（星级右缘越过 chip 右缘）。
    expect(chipRect.left, lessThan(starRect.left));
    expect(starRect.right, greaterThan(chipRect.right));
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('review sheet keeps the rating reachable at 200% text', (
    tester,
  ) async {
    await _pumpReviewSheet(
      tester,
      size: const Size(320, 568),
      textScale: 2,
      locale: const Locale('de'),
    );
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(tester.getRect(_panel()).top, greaterThanOrEqualTo(0));
    // 必填的星级与班次仍在面板内（键盘只剩 244px 时也不能被裁掉）。
    expect(
      find.byKey(const ValueKey<String>('review-rating-5')),
      findsOneWidget,
    );
    expect(find.byKey(const Key('course-review-offering')), findsOneWidget);
    // 极端字号下面板只剩 216px，元信息退回单行横向滚动：此时必须优先保证
    // 必填的五颗星全部可见（240px ≤ 288px），班次 chip 允许需要横滑。
    final panelRect = tester.getRect(_panel());
    for (final key in <Key>[
      const ValueKey<String>('review-rating-1'),
      const ValueKey<String>('review-rating-5'),
    ]) {
      final rect = tester.getRect(find.byKey(key));
      expect(rect.left, greaterThanOrEqualTo(panelRect.left));
      expect(rect.right, lessThanOrEqualTo(panelRect.right));
    }
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('review sheet meta keeps stars and anonymous switch on screen', (
    tester,
  ) async {
    // 日常机型（390×844、100% 字号）：五颗星、班次 chip、匿名开关都必须
    // 直接可见，不需要用户猜到去横向滑动（回归：单行可滚元信息曾把星级
    // 4–5 与匿名开关推到屏幕外）。
    await _pumpReviewSheet(tester);

    final panelRect = tester.getRect(_panel());
    for (int star = 1; star <= 5; star++) {
      final rect = tester.getRect(
        find.byKey(ValueKey<String>('review-rating-$star')),
      );
      expect(rect.left, greaterThanOrEqualTo(panelRect.left));
      expect(rect.right, lessThanOrEqualTo(panelRect.right));
    }
    expect(find.byType(Switch).hitTestable(), findsOneWidget);
    expect(find.byKey(const Key('course-review-offering')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey<String>('review-rating-5')));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
}

import 'package:dio/dio.dart';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/editor/course_review_templates.dart';
import 'package:forum_app/src/widgets/editor/rich_markdown_editor.dart';
import 'package:forum_app/src/widgets/markdown_view.dart';
import 'package:ui_kit/ui_kit.dart';

class _MemoryTokenStorage implements TokenStorage {
  @override
  Future<String?> read() async => null;

  @override
  Future<void> write(String token) async {}

  @override
  Future<void> clear() async {}
}

class _EmptyStickerRepository extends StickerRepository {
  _EmptyStickerRepository()
    : super(
        GfApiClient(
          dio: Dio(BaseOptions(baseUrl: 'http://test')),
          tokenStorage: _MemoryTokenStorage(),
        ),
      );

  @override
  Future<List<StickerItemPayload>> list() async => const [];

  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async =>
      const [];
}

void main() {
  test('course review template Markdown mirrors Web source', () {
    expect(
      courseReviewTemplates.map((template) => template.id).toList(),
      <String>[
        'comprehensive',
        'quick',
        'teacher-focused',
        'exam-focused',
        'workload',
        'blank',
      ],
    );
    expect(
      courseReviewTemplates.map((template) => template.content).toList(),
      <String>[
        '## 课程内容\n\n## 教学方式\n\n## 作业与考核\n\n## 收获与建议\n',
        '**总体评价：**\n\n**优点：**\n-\n\n**缺点：**\n-\n\n**建议：**\n',
        '## 教学态度\n\n## 授课风格\n\n## 师生互动\n\n## 总体印象\n',
        '## 考试形式\n\n## 考试难度\n\n## 备考建议\n\n## 给分情况\n',
        '## 课时安排\n\n## 作业量\n\n## 项目/实验\n\n## 时间投入\n',
        '',
      ],
    );
  });

  test('Markdown converter retains review formatting', () {
    const markdown =
        '## 课程\n\n**清晰**\n\n- 例子\n\n| 项目 | 结果 |\n| --- | --- |\n| 作业 | 适量 |\n';
    final converter = MarkdownConverter();
    final converted = converter.documentToMarkdown(
      converter.mdToDocument(markdown),
    );
    expect(converted, contains('## 课程'));
    expect(converted, contains('**清晰**'));
    expect(converted, contains('- 例子'));
    expect(converted, contains('| 项目 | 结果 |'));
    expect(converted, contains('| 作业 | 适量 |'));
  });

  testWidgets('shared editor toolbar stays within a 320px viewport', (
    tester,
  ) async {
    final controller = QuillController.basic();
    final focusNode = FocusNode();
    addTearDown(controller.dispose);
    addTearDown(focusNode.dispose);

    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: Align(
            alignment: Alignment.topLeft,
            child: SizedBox(
              width: 320,
              child: RichMarkdownEditor(
                controller: controller,
                focusNode: focusNode,
                placeholder: 'Review',
                editorBuilder: (_) => const SizedBox(height: 200),
              ),
            ),
          ),
        ),
      ),
    );

    expect(find.byKey(const Key('rich-markdown-toolbar')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('editor block styles come from the shared rich-content profile', (
    tester,
  ) async {
    final converter = MarkdownConverter();
    final controller = QuillController(
      document: converter.mdToDocument('## 课程内容\n\n正文\n'),
      selection: const TextSelection.collapsed(offset: 0),
    );
    final focusNode = FocusNode();
    final library = StickerLibrary(_EmptyStickerRepository());
    addTearDown(controller.dispose);
    addTearDown(focusNode.dispose);
    addTearDown(library.dispose);

    late GfRichContentTypography profile;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerLibraryProvider.overrideWithValue(library)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Builder(
              builder: (context) {
                profile = GfRichContentTypography.of(context, compact: true);
                return RichMarkdownEditor(
                  controller: controller,
                  focusNode: focusNode,
                  placeholder: 'Review',
                  profile: profile,
                );
              },
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // 课评模板以 `##` 开头，编辑器档位必须跟阅读态一致，而不是 Quill 的
    // 默认 34 / 30 / 24。
    final styles = tester
        .widget<QuillEditor>(find.byType(QuillEditor))
        .config
        .customStyles!;
    expect(styles.h1!.style.fontSize, closeTo(22.475, 0.001));
    expect(styles.h2!.style.fontSize, closeTo(20.15, 0.001));
    expect(styles.h3!.style.fontSize, closeTo(18.29, 0.001));
    expect(styles.h4!.style.fontSize, closeTo(16.74, 0.001));
    expect(styles.paragraph!.style.fontSize, closeTo(15.5, 0.001));
    expect(styles.h1!.style.fontSize, isNot(34));
    expect(styles.h2!.style.fontSize, isNot(30));
    expect(styles.h3!.style.fontSize, isNot(24));
    expect(styles.paragraph!.verticalSpacing, const VerticalSpacing(4, 4));
    // 行内代码 / 引用 / 列表同样由 profile 提供。
    expect(styles.inlineCode!.style.fontSize, closeTo(14.5, 0.001));
    expect(styles.quote!.style.color, profile.quote.color);
    expect(styles.lists!.style.fontSize, closeTo(15.5, 0.001));

    // 真正渲染出来的标题字号（不是仅 config 值）。
    final heading = tester.widget<RichText>(
      find.textContaining('课程内容', findRichText: true).first,
    );
    expect(heading.text.style!.fontSize, closeTo(20.15, 0.001));

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('fill mode scrolls the body inside a bounded height', (
    tester,
  ) async {
    final converter = MarkdownConverter();
    final controller = QuillController(
      document: converter.mdToDocument(
        List<String>.generate(40, (index) => '第 $index 行评价').join('\n'),
      ),
      selection: const TextSelection.collapsed(offset: 0),
    );
    final focusNode = FocusNode();
    final library = StickerLibrary(_EmptyStickerRepository());
    addTearDown(controller.dispose);
    addTearDown(focusNode.dispose);
    addTearDown(library.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerLibraryProvider.overrideWithValue(library)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Align(
              alignment: Alignment.topLeft,
              child: SizedBox(
                width: 320,
                height: 220,
                child: RichMarkdownEditor(
                  controller: controller,
                  focusNode: focusNode,
                  placeholder: 'Review',
                  fill: true,
                ),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    // 工具栏常驻在固定高度内（不随正文行数移动）。
    final toolbar = find.byKey(const Key('rich-markdown-toolbar'));
    expect(toolbar, findsOneWidget);
    expect(tester.getRect(toolbar).bottom, lessThanOrEqualTo(220));
    // 正文自己滚动（scrollable + 内层 Scrollable），不会撑破卡片。
    final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
    expect(editor.config.scrollable, isTrue);
    expect(editor.config.minHeight, 0);
    expect(editor.config.scrollBottomInset, 12);
    expect(
      find.descendant(
        of: find.byType(QuillEditor),
        matching: find.byWidgetPredicate(
          (widget) =>
              widget is Scrollable &&
              widget.axisDirection == AxisDirection.down,
        ),
      ),
      findsOneWidget,
    );
    expect(
      tester.getSize(find.byType(QuillEditor)).height,
      lessThanOrEqualTo(172),
    );

    // 追加长文后工具栏位置不变（正文内部滚动，而不是整块上滚）。
    final before = tester.getRect(toolbar);
    controller.replaceText(
      0,
      controller.document.length - 1,
      List<String>.generate(120, (index) => '追加第 $index 行').join('\n'),
      const TextSelection.collapsed(offset: 0),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(tester.getRect(toolbar), before);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('existing Markdown table uses the shared Markdown renderer', (
    tester,
  ) async {
    final converter = MarkdownConverter();
    final controller = QuillController(
      document: converter.mdToDocument(
        '| 项目 | 结果 |\n| --- | --- |\n| 作业 | 适量 |\n',
      ),
      selection: const TextSelection.collapsed(offset: 0),
    );
    final focusNode = FocusNode();
    final library = StickerLibrary(_EmptyStickerRepository());
    addTearDown(controller.dispose);
    addTearDown(focusNode.dispose);
    addTearDown(library.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerLibraryProvider.overrideWithValue(library)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: RichMarkdownEditor(
              controller: controller,
              focusNode: focusNode,
              placeholder: 'Review',
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(GfMarkdownView), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });
}

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

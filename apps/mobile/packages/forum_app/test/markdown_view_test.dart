import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/src/widgets/markdown_view.dart';

void main() {
  testWidgets('父级重建复用 markdown 解析树，内容变化时才重建', (tester) async {
    late StateSetter rebuild;
    String markdown = '**第一版**';

    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: StatefulBuilder(
            builder: (BuildContext context, StateSetter setState) {
              rebuild = setState;
              return GfMarkdownView(data: markdown);
            },
          ),
        ),
      ),
    );

    final MarkdownWidget first = tester.widget<MarkdownWidget>(
      find.byType(MarkdownWidget),
    );

    rebuild(() {});
    await tester.pump();

    final MarkdownWidget afterParentRebuild = tester.widget<MarkdownWidget>(
      find.byType(MarkdownWidget),
    );
    expect(identical(first, afterParentRebuild), isTrue);

    rebuild(() => markdown = '**第二版**');
    await tester.pump();

    final MarkdownWidget afterContentChange = tester.widget<MarkdownWidget>(
      find.byType(MarkdownWidget),
    );
    expect(identical(first, afterContentChange), isFalse);
    expect(find.text('第二版'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });
}

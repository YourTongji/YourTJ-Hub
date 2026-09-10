import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/src/widgets/markdown_view.dart';

void main() {
  testWidgets('short replies have compact block spacing and readable text', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: const Scaffold(
          body: Column(
            children: [
              GfMarkdownView(data: 'First paragraph\n\nSecond paragraph'),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      tester.getSize(find.byType(GfMarkdownView)).height,
      lessThanOrEqualTo(64),
    );
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('embedded markdown does not repeat device safe-area spacing', (
    tester,
  ) async {
    Future<double> measure(EdgeInsets insets) async {
      await tester.pumpWidget(
        MaterialApp(
          home: MediaQuery(
            data: MediaQueryData(padding: insets),
            child: const Scaffold(
              body: Column(children: [GfMarkdownView(data: 'A short reply')]),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      return tester.getSize(find.byType(GfMarkdownView)).height;
    }

    final baseline = await measure(EdgeInsets.zero);
    final withInsets = await measure(
      const EdgeInsets.only(top: 62, bottom: 34),
    );
    expect(withInsets, baseline);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

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

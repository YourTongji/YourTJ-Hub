import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

String? _emailValidator(String? value) =>
    value != null && value.contains('@') ? null : '邮箱格式不正确';

void main() {
  group('GfListSection', () {
    testWidgets('header 与 footer 分居卡片上下', (tester) async {
      await tester.pumpWidget(
        gfApp(
          const GfListSection(
            header: '账号',
            footer: '退出后需要重新登录',
            children: [GfFieldRow(label: '昵称')],
          ),
        ),
      );
      final row = tester.getRect(find.byType(GfFieldRow));
      expect(tester.getTopLeft(find.text('账号')).dy, lessThan(row.top));
      expect(
        tester.getTopLeft(find.text('退出后需要重新登录')).dy,
        greaterThan(row.bottom),
      );
      // header 以语义标记声明为组标题。
      final semantics = tester.ensureSemantics();
      expect(
        tester.getSemantics(find.text('账号')).getSemanticsData().flagsCollection.isHeader,
        isTrue,
      );
      semantics.dispose();
      expect(tester.takeException(), isNull);
    });

    testWidgets('hairline 分隔按 dividerIndent 缩进并与 icon 行标题对齐', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          const GfListSection(
            dividerIndent: GfListSection.iconRowIndent,
            children: [
              GfFieldRow(
                label: '昵称',
                leading: Icon(Icons.person, size: GfIconSize.lg),
              ),
              GfFieldRow(label: '邮箱'),
            ],
          ),
        ),
      );
      final dividers = find.byWidgetPredicate(
        (w) =>
            w is Padding &&
            w.padding ==
                const EdgeInsetsDirectional.only(
                  start: GfListSection.iconRowIndent,
                ),
      );
      expect(dividers, findsOneWidget);
      final card = tester.getRect(find.byType(GfFieldRow).first);
      final hairline = find.descendant(
        of: dividers,
        matching: find.byType(Container),
      );
      expect(hairline, findsOneWidget);
      final hairlineRect = tester.getRect(hairline);
      // 分隔线从 icon 行标题处开始，右端顶到卡片边缘，粗细为发线宽度。
      expect(hairlineRect.left, card.left + GfListSection.iconRowIndent);
      expect(hairlineRect.right, card.right);
      expect(tester.getSize(hairline).height, gfHairline(tester.element(hairline)));
      // iconRowIndent 正好等于 icon 行标题的缩进，分隔线与标题左对齐。
      expect(tester.getRect(find.text('昵称')).left, hairlineRect.left);
      expect(tester.takeException(), isNull);
    });

    testWidgets('children 为空时卡片不留分隔', (tester) async {
      const colors = GfColors.light;
      await tester.pumpWidget(
        gfApp(
          const GfListSection(
            header: '空组',
            footer: '暂无设置项',
            children: [],
          ),
        ),
      );
      expect(find.text('空组'), findsOneWidget);
      expect(find.text('暂无设置项'), findsOneWidget);
      expect(
        find.byWidgetPredicate(
          (w) => w is Container && w.color == colors.separator,
        ),
        findsNothing,
      );
      // 卡片本身仍渲染（圆角 grouped 卡片，区别于 Scaffold 的 Material）。
      expect(
        find.byWidgetPredicate(
          (w) => w is Material && w.shape is RoundedRectangleBorder,
        ),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    });
  });

  group('GfFieldRow', () {
    testWidgets('计数器超限变红', (tester) async {
      await tester.pumpWidget(
        gfApp(
          const GfListSection(
            children: [GfFieldRow(label: '简介', maxLength: 5)],
          ),
        ),
      );
      final colors = GfTheme.colorsOf(tester.element(find.text('简介')));
      await tester.enterText(find.byType(EditableText), '1234');
      await tester.pump();
      expect(find.text('4/5'), findsOneWidget);
      expect(
        tester.widget<Text>(find.text('4/5')).style?.color,
        colors.textPlaceholder,
      );
      await tester.enterText(find.byType(EditableText), '123456');
      await tester.pump();
      expect(find.text('6/5'), findsOneWidget);
      expect(
        tester.widget<Text>(find.text('6/5')).style?.color,
        colors.error,
      );
      expect(tester.takeException(), isNull);
    });

    testWidgets('validator 错误显示在错误色中，聚焦时 label 变主色', (
      tester,
    ) async {
      final formKey = GlobalKey<FormState>();
      await tester.pumpWidget(
        gfApp(
          Form(
            key: formKey,
            child: const GfListSection(
              children: [
                GfFieldRow(label: '邮箱', validator: _emailValidator),
              ],
            ),
          ),
        ),
      );
      final colors = GfTheme.colorsOf(tester.element(find.text('邮箱')));
      expect(
        tester.widget<Text>(find.text('邮箱')).style?.color,
        colors.textSecondary,
      );
      // 点击行内任意位置即聚焦，label 变主色。
      await tester.tap(find.text('邮箱'));
      await tester.pump();
      expect(
        tester.widget<Text>(find.text('邮箱')).style?.color,
        colors.primary,
      );
      await tester.enterText(find.byType(EditableText), 'not-an-email');
      await tester.pump();
      expect(formKey.currentState!.validate(), isFalse);
      await tester.pump();
      expect(find.text('邮箱格式不正确'), findsOneWidget);
      expect(
        tester.widget<Text>(find.text('邮箱格式不正确')).style?.color,
        colors.error,
      );
      expect(tester.takeException(), isNull);
    });
  });
}

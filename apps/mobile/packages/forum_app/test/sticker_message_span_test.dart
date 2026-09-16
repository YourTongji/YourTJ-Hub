import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/src/widgets/sticker_message_span.dart';

void main() {
  const Map<String, String> urlByName = <String, String>{
    'smile': '/file/img/stickers/smile.png',
  };

  test('无 token 返回 null,走纯文本路径', () {
    expect(buildStickerMessageSpan('普通消息', urlByName), isNull);
  });

  test('全部未知 token 返回 null(回退纯文本,原文展示)', () {
    expect(buildStickerMessageSpan('[:sticker:ghost:]', urlByName), isNull);
  });

  test('已知 token 生成含 WidgetSpan 的富文本', () {
    final InlineSpan? span = buildStickerMessageSpan(
      '前 [:sticker:smile:] 后',
      urlByName,
    );
    expect(span, isNotNull);
    final TextSpan textSpan = span! as TextSpan;
    expect(textSpan.children, hasLength(3));
    expect((textSpan.children![0] as TextSpan).text, '前 ');
    expect(textSpan.children![1], isA<WidgetSpan>());
    expect((textSpan.children![2] as TextSpan).text, ' 后');
  });

  testWidgets('贴纸段渲染为语义标注的 56 方块占位', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: Text.rich(
            buildStickerMessageSpan('[:sticker:smile:]', urlByName)!,
          ),
        ),
      ),
    );
    await tester.pump();
    // 语义层携带表情名(无障碍兜底,语义同 web alt 属性)。
    expect(
      find.byWidgetPredicate(
        (Widget widget) =>
            widget is Semantics && widget.properties.label == 'smile',
      ),
      findsOneWidget,
    );
    // 加载完成前渲染 56x56 占位方块,行高不跳动。
    expect(tester.getSize(find.byType(SizedBox).first), const Size(56, 56));
  });
}

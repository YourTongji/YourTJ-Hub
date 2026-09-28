import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  for (final brightness in Brightness.values) {
    for (final mine in [false, true]) {
      testWidgets(
        'unframed message preserves content bounds, alignment, time and selection ($brightness, mine: $mine)',
        (tester) async {
          final bubbleKey = GlobalKey();
          const contentKey = Key('message-content');
          const paneKey = Key('conversation-pane');
          await tester.pumpWidget(
            gfApp(
              SizedBox(
                key: paneKey,
                width: 300,
                height: 180,
                child: GfMessageBubble(
                  text: '[:sticker:hello:]',
                  mine: mine,
                  showBubble: false,
                  selectable: true,
                  bubbleKey: bubbleKey,
                  time: '12:34',
                  content: const SizedBox(
                    key: contentKey,
                    width: 80,
                    height: 72,
                    child: Text('Fallback'),
                  ),
                ),
              ),
              brightness: brightness,
            ),
          );

          final bubble = tester.widget<Container>(find.byKey(bubbleKey));
          expect(bubble.decoration, isNull);
          expect(bubble.color, isNull);
          expect(bubble.padding ?? EdgeInsets.zero, EdgeInsets.zero);
          final bodyBounds = tester.getRect(find.byKey(bubbleKey));
          expect(bodyBounds, tester.getRect(find.byKey(contentKey)));
          expect(bodyBounds.size, const Size(80, 72));
          final paneBounds = tester.getRect(find.byKey(paneKey));
          final timestamp = tester.getRect(find.text('12:34'));
          if (mine) {
            expect(bodyBounds.right, paneBounds.right);
            expect(timestamp.right, bodyBounds.right);
          } else {
            expect(bodyBounds.left, paneBounds.left);
            expect(timestamp.left, bodyBounds.left);
          }
          expect(timestamp.top, bodyBounds.bottom + 4);
          expect(find.byType(SelectionArea), findsOneWidget);
          expect(
            DefaultTextStyle.of(
              tester.element(find.byKey(contentKey)),
            ).style.color,
            GfColors.forBrightness(brightness).baseContent,
          );
          expect(tester.takeException(), isNull);
        },
      );

      testWidgets(
        'default message keeps its surface ($brightness, mine: $mine)',
        (tester) async {
          final bubbleKey = GlobalKey();
          await tester.pumpWidget(
            gfApp(
              GfMessageBubble(
                text: 'Ordinary message',
                mine: mine,
                bubbleKey: bubbleKey,
              ),
              brightness: brightness,
            ),
          );
          final colors = GfColors.forBrightness(brightness);
          final bubble = tester.widget<Container>(find.byKey(bubbleKey));
          final decoration = bubble.decoration! as BoxDecoration;
          expect(
            decoration.color,
            mine ? const Color(0xFF2563EB) : colors.base300,
          );
          expect(decoration.borderRadius, BorderRadius.circular(20));
          expect(
            bubble.padding,
            const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          );
          expect(
            DefaultTextStyle.of(
              tester.element(find.text('Ordinary message')),
            ).style.color,
            mine ? Colors.white : colors.baseContent,
          );
        },
      );
    }
  }

  for (final selectable in [false, true]) {
    testWidgets('long-press reports once (selectable: $selectable)', (
      tester,
    ) async {
      var presses = 0;
      await tester.pumpWidget(
        gfApp(
          GfMessageBubble(
            text: '可长按的消息',
            mine: false,
            selectable: selectable,
            onLongPress: () => presses++,
          ),
        ),
      );
      final handle = tester.ensureSemantics();
      await tester.longPress(find.text('可长按的消息'));
      await tester.pumpAndSettle();
      expect(presses, 1);
      expect(
        tester
            .getSemantics(find.text('可长按的消息'))
            .getSemanticsData()
            .hasAction(SemanticsAction.longPress),
        isTrue,
        reason: 'screen readers announce the action menu affordance',
      );
      expect(
        find.byType(SelectionArea),
        selectable ? findsOneWidget : findsNothing,
      );
      await tester.tap(find.text('可长按的消息'));
      await tester.pumpAndSettle();
      expect(presses, 1, reason: 'a plain tap must not open the menu');
      expect(tester.takeException(), isNull);
      handle.dispose();
    });
  }

  testWidgets('a selectable bubble still drags text selection', (tester) async {
    await tester.pumpWidget(
      gfApp(
        GfMessageBubble(
          text: '可选择的消息内容',
          mine: false,
          selectable: true,
          onLongPress: () {},
        ),
      ),
    );
    final gesture = await tester.startGesture(
      tester.getCenter(find.text('可选择的消息内容')),
      kind: PointerDeviceKind.mouse,
    );
    await gesture.moveBy(const Offset(60, 0));
    await gesture.up();
    await tester.pumpAndSettle();
    expect(find.byType(SelectionArea), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  for (final mine in [false, true]) {
    testWidgets('unframed message keeps maximum width (mine: $mine)', (
      tester,
    ) async {
      final bubbleKey = GlobalKey();
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 300,
            child: GfMessageBubble(
              text: '',
              mine: mine,
              showBubble: false,
              maxWidthFactor: .8,
              bubbleKey: bubbleKey,
              content: const SizedBox(width: 1000, height: 72),
            ),
          ),
        ),
      );
      expect(tester.getSize(find.byKey(bubbleKey)), const Size(240, 72));
      expect(tester.takeException(), isNull);
    });
  }
}

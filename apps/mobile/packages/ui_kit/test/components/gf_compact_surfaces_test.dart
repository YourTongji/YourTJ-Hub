import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('segments fit intrinsic rows and preserve large text targets', (
    tester,
  ) async {
    var selected = '';
    await tester.pumpWidget(
      gfApp(
        MediaQuery(
          data: const MediaQueryData(textScaler: TextScaler.linear(2)),
          child: Row(
            children: [
              GfSegmented<String>(
                expanded: false,
                segments: const [('Public', 'public'), ('Private', 'private')],
                selected: 'public',
                onSelected: (value) => selected = value,
              ),
            ],
          ),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    await tester.tap(find.text('Private'));
    expect(selected, 'private');
    for (final hit
        in find
            .descendant(
              of: find.byType(GfSegmented<String>),
              matching: find.byType(InkWell),
            )
            .evaluate()) {
      expect(
        (hit.renderObject! as RenderBox).size.height,
        greaterThanOrEqualTo(48),
      );
    }
  });

  testWidgets('menu labels wrap at large type without clipping', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        Center(
          child: SizedBox(
            width: 220,
            child: MediaQuery(
              data: const MediaQueryData(textScaler: TextScaler.linear(2)),
              child: GfMenuSurface(
                child: GfMenuItem(
                  label: 'Manage saved collections',
                  symbol: 'bookmark',
                  onTap: () {},
                ),
              ),
            ),
          ),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    final textRect = tester.getRect(find.text('Manage saved collections'));
    final itemRect = tester.getRect(find.byType(GfMenuItem));
    expect(itemRect.contains(textRect.topLeft), isTrue);
    expect(itemRect.contains(textRect.bottomRight), isTrue);
    expect(itemRect.height, greaterThanOrEqualTo(48));
  });

  testWidgets('message bubble respects conversation pane width', (
    tester,
  ) async {
    final bubbleKey = GlobalKey();
    await tester.pumpWidget(
      gfApp(
        Center(
          child: SizedBox(
            width: 240,
            child: GfMessageBubble(
              text:
                  'A message with enough text to fill a narrow conversation pane.',
              mine: false,
              bubbleKey: bubbleKey,
            ),
          ),
        ),
      ),
    );
    expect(
      tester.getSize(find.byKey(bubbleKey)).width,
      lessThanOrEqualTo(240 * .88),
    );
    final decoration =
        tester.widget<Container>(find.byKey(bubbleKey)).decoration!
            as BoxDecoration;
    expect(decoration.borderRadius, BorderRadius.circular(20));
    expect(decoration.boxShadow, isNull);
  });
}

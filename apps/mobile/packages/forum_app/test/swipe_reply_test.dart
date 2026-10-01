import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/messages/swipe_reply.dart';
import 'package:ui_kit/ui_kit.dart';

const _message = ValueKey('message');

Future<void> _pump(
  WidgetTester tester,
  VoidCallback? onReply, {
  bool reducedMotion = false,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: gfThemeData(Brightness.light),
      home: MediaQuery(
        data: MediaQueryData(disableAnimations: reducedMotion),
        child: Scaffold(
          body: Center(
            child: SwipeReply(
              label: 'Reply',
              onReply: onReply,
              child: const SizedBox(
                key: _message,
                width: 320,
                height: 64,
                child: ColoredBox(color: Colors.blue),
              ),
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  for (final distance in [8.8, 26.4, 61.6, 63.9, 64.0, 64.1, 88.0]) {
    for (final cancel in [false, true]) {
      testWidgets('$distance px ${cancel ? 'cancel' : 'release'}', (
        tester,
      ) async {
        var replies = 0;
        await _pump(tester, () => replies++);
        final origin = tester.getTopLeft(find.byKey(_message));
        final gesture = await tester.startGesture(
          tester.getCenter(find.byKey(_message)),
        );
        // Accept horizontal intent first, then place the drag at the boundary.
        await gesture.moveBy(const Offset(-24, 0));
        await tester.pump();
        await gesture.moveBy(Offset(24 - distance, 0));
        await tester.pump();
        if (cancel) {
          await gesture.cancel();
        } else {
          await gesture.up();
        }
        await tester.pumpAndSettle();
        expect(replies, !cancel && distance >= 64 ? 1 : 0);
        expect(tester.getTopLeft(find.byKey(_message)), origin);
      });
    }
  }

  testWidgets('a second pointer aborts the whole reply gesture', (
    tester,
  ) async {
    var replies = 0;
    await _pump(tester, () => replies++);
    final origin = tester.getTopLeft(find.byKey(_message));
    final first = await tester.startGesture(
      tester.getCenter(find.byKey(_message)),
      pointer: 1,
    );
    await first.moveBy(const Offset(-80, 0));
    await tester.pump();
    final second = await tester.startGesture(
      tester.getCenter(find.byKey(_message)),
      pointer: 2,
    );
    await first.up();
    await second.moveBy(const Offset(-80, 0));
    await second.up();
    await tester.pumpAndSettle();
    expect(replies, 0);
    expect(tester.getTopLeft(find.byKey(_message)), origin);
    await tester.drag(find.byKey(_message), const Offset(-80, 0));
    await tester.pumpAndSettle();
    expect(replies, 1, reason: 'a fresh single-pointer gesture still works');
  });

  for (final progress in [.1, .3, .7, 1.0]) {
    testWidgets('return can be taken over at $progress', (tester) async {
      var replies = 0;
      await _pump(tester, () => replies++);
      final origin = tester.getTopLeft(find.byKey(_message)).dx;
      double offset() => origin - tester.getTopLeft(find.byKey(_message)).dx;
      final first = await tester.startGesture(
        tester.getCenter(find.byKey(_message)),
      );
      await first.moveBy(const Offset(-48, 0));
      await tester.pump();
      await first.up();
      await tester.pump();
      await tester.pump(Duration(microseconds: (180000 * progress).round()));
      final before = offset();
      final next = await tester.startGesture(
        tester.getCenter(find.byKey(_message)),
      );
      await next.moveBy(const Offset(-24, 0));
      await tester.pump();
      expect(offset(), closeTo(before + 24, .001));
      await tester.pump(const Duration(milliseconds: 60));
      expect(offset(), closeTo(before + 24, .001));
      await next.moveBy(const Offset(100, 0));
      await next.up();
      await tester.pumpAndSettle();
      expect(offset(), 0);
      expect(replies, 0);
    });
  }

  testWidgets('enabling reduced motion settles an active return', (
    tester,
  ) async {
    void onReply() {}
    await _pump(tester, onReply);
    final origin = tester.getTopLeft(find.byKey(_message));
    await tester.drag(find.byKey(_message), const Offset(-48, 0));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 30));
    expect(tester.getTopLeft(find.byKey(_message)), isNot(origin));
    await _pump(tester, onReply, reducedMotion: true);
    expect(tester.getTopLeft(find.byKey(_message)), origin);
  });
}

import 'package:flutter/widgets.dart';

/// Keeps the message at the bottom of the reading area above the composer when
/// the keyboard, sticker picker or draft preview changes the viewport height.
class ChatViewportScrollPhysics extends ScrollPhysics {
  const ChatViewportScrollPhysics({super.parent});

  @override
  ChatViewportScrollPhysics applyTo(ScrollPhysics? ancestor) =>
      ChatViewportScrollPhysics(parent: buildParent(ancestor));

  @override
  double adjustPositionForNewDimensions({
    required ScrollMetrics oldPosition,
    required ScrollMetrics newPosition,
    required bool isScrolling,
    required double velocity,
  }) {
    final resize =
        oldPosition.viewportDimension - newPosition.viewportDimension;
    if (resize != 0 && !oldPosition.outOfRange) {
      // Correct during layout, before painting or sampling visible reads. Using
      // viewport delta (not content extent) also keeps history in place when new
      // messages arrive. Repeated layouts converge on the same absolute target.
      return (oldPosition.pixels + resize).clamp(
        newPosition.minScrollExtent,
        newPosition.maxScrollExtent,
      );
    }
    return super.adjustPositionForNewDimensions(
      oldPosition: oldPosition,
      newPosition: newPosition,
      isScrolling: isScrolling,
      velocity: velocity,
    );
  }
}

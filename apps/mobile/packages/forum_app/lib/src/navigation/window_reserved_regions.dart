import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final windowReservedRegionsProvider = StreamProvider<WindowReservedRegions?>((
  ref,
) async* {
  if (kIsWeb || defaultTargetPlatform != TargetPlatform.iOS) {
    yield null;
    return;
  }
  try {
    await for (final message in const EventChannel(
      'yourtj/window_reserved_regions',
    ).receiveBroadcastStream()) {
      yield WindowReservedRegions.fromMessage(message);
    }
  } on MissingPluginException {
    // Older SDKs/OS versions retain the ordinary safe-area navigation.
    yield null;
  } on PlatformException {
    yield null;
  }
});

/// Geometry supplied by the current iOS Flutter view, in logical pixels.
/// Only a matching snapshot can authorize controls outside its safe area.
@immutable
class WindowReservedRegions {
  // Four 48dp destinations, inner padding and an 8dp outer margin.
  static const navigationClearance = 224.0;
  const WindowReservedRegions({
    required this.size,
    required this.safeArea,
    required this.occlusions,
  });

  final Size size;
  final EdgeInsets safeArea;
  final List<Rect> occlusions;

  static WindowReservedRegions? fromMessage(Object? message) {
    if (message is! Map) return null;
    final width = message['width'];
    final height = message['height'];
    final safe = _numbers(message['safeArea']);
    final regions = message['occlusions'];
    if (width is! num ||
        height is! num ||
        !width.isFinite ||
        !height.isFinite ||
        width <= 0 ||
        height <= 0 ||
        safe == null ||
        safe.any((value) => value < 0) ||
        regions is! List) {
      return null;
    }
    final frames = <Rect>[];
    for (final value in regions) {
      final rect = _numbers(value);
      if (rect == null || rect[2] < 0 || rect[3] < 0) return null;
      frames.add(Rect.fromLTWH(rect[0], rect[1], rect[2], rect[3]));
    }
    return WindowReservedRegions(
      size: Size(width.toDouble(), height.toDouble()),
      safeArea: EdgeInsets.fromLTRB(safe[0], safe[1], safe[2], safe[3]),
      occlusions: List.unmodifiable(frames),
    );
  }

  static List<double>? _numbers(Object? value) {
    if (value is! List ||
        value.length != 4 ||
        value.any((item) => item is! num || !item.isFinite)) {
      return null;
    }
    return value.map((item) => (item as num).toDouble()).toList();
  }

  /// Uses a clear vertical interval in an existing side inset; content never
  /// gives up more width. Geometry must agree with Flutter after each resize.
  /// [topInset] is the space occupied above a bottom-aligned root viewport,
  /// such as the preview notice. Returned coordinates are local to that viewport.
  Rect? navigationFrame(
    MediaQueryData media,
    Size layoutSize, {
    double topInset = 0,
  }) {
    bool near(double a, double b) => (a - b).abs() < .5;
    if (!topInset.isFinite ||
        topInset < 0 ||
        layoutSize.height <= 0 ||
        !near(size.width, layoutSize.width) ||
        !near(size.height, layoutSize.height + topInset) ||
        !near(size.width, media.size.width) ||
        !near(size.height, media.size.height) ||
        !near(safeArea.left, media.viewPadding.left) ||
        !near(safeArea.top, media.viewPadding.top) ||
        !near(safeArea.right, media.viewPadding.right) ||
        !near(safeArea.bottom, media.viewPadding.bottom)) {
      return null;
    }

    const minimumHeight = navigationClearance;
    const minimumWidth = 72.0;
    final bottom =
        size.height -
        (media.viewInsets.bottom > safeArea.bottom
            ? media.viewInsets.bottom
            : safeArea.bottom);
    final top = safeArea.top > topInset ? safeArea.top : topInset;
    // Prefer the right pocket; rotation/Split View can put it on the left.
    for (final strip in [
      Rect.fromLTRB(size.width - safeArea.right, top, size.width, bottom),
      Rect.fromLTRB(0, top, safeArea.left, bottom),
    ]) {
      if (strip.width < minimumWidth || strip.height < minimumHeight) continue;
      final blockers =
          occlusions
              .where((frame) => frame.overlaps(strip))
              .map((frame) => frame.intersect(strip))
              .toList()
            ..sort((a, b) => a.top.compareTo(b.top));
      var start = strip.top;
      final gaps = <Rect>[];
      for (final blocker in blockers) {
        if (blocker.top - start >= minimumHeight) {
          gaps.add(Rect.fromLTRB(strip.left, start, strip.right, blocker.top));
        }
        if (blocker.bottom > start) start = blocker.bottom;
      }
      if (strip.bottom - start >= minimumHeight) {
        gaps.add(Rect.fromLTRB(strip.left, start, strip.right, strip.bottom));
      }
      if (gaps.isNotEmpty) return gaps.last.shift(Offset(0, -topInset));
    }
    return null;
  }
}

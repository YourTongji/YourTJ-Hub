import 'dart:math' as math;

import 'package:flutter/material.dart';

/// Deterministic "beam" identity avatar (36-unit smiling face) for surfaces
/// that have no real portrait: anonymous and legacy course reviews, and the
/// fallback when a member avatar fails to load.
///
/// The geometry is an algorithm-identical port of the Web implementation in
/// `apps/gooseforum/resource/src/site/utils/course-review-share.ts`
/// (`buildBeamAvatarDataUri`), which in turn mirrors `boring-avatars@2.0.4`
/// `variant="beam"`. The seed formula used by the review surfaces is
/// `'<author label>-<review id>'`, so one review keeps the same face on the
/// forum Web app and on this app.
///
/// [beamColors] is deliberately a constant shared with the Web palette and
/// **not** a theme token: it is an identity contract, so deriving it from
/// design tokens would silently give the same user two different avatars.
class GfBeamAvatar extends StatelessWidget {
  const GfBeamAvatar({super.key, required this.seed, this.size = 40});

  /// Stable identity seed; an empty seed renders the shared anonymous face.
  final String seed;

  /// Edge length in logical pixels.
  final double size;

  @override
  Widget build(BuildContext context) => SizedBox.square(
    dimension: size,
    child: CustomPaint(painter: _BeamAvatarPainter(seed: seed)),
  );
}

/// Shared identity palette (same order and values as the Web
/// `AVATAR_COLORS`).
const List<Color> beamColors = <Color>[
  Color(0xFF0F172A),
  Color(0xFF38BDF8),
  Color(0xFFF8FAFC),
  Color(0xFFF59E0B),
  Color(0xFF22C55E),
];

/// Anonymous fallback seed: the Web and the legacy client both render this
/// face for an empty seed instead of an empty box.
const String beamAnonymousSeed = '匿名用户';

class _BeamAvatarPainter extends CustomPainter {
  const _BeamAvatarPainter({required this.seed});

  final String seed;

  @override
  void paint(Canvas canvas, Size size) {
    final data = GfBeamAvatarData(seed: seed);
    final scale = size.shortestSide / GfBeamAvatarData.canvasSize;
    final dx = (size.width - size.shortestSide) / 2;
    final dy = (size.height - size.shortestSide) / 2;

    canvas.save();
    canvas.translate(dx, dy);
    canvas.scale(scale);
    canvas.clipPath(
      Path()..addOval(
        const Rect.fromLTWH(
          0,
          0,
          GfBeamAvatarData.canvasSize,
          GfBeamAvatarData.canvasSize,
        ),
      ),
    );

    canvas.drawRect(
      const Rect.fromLTWH(
        0,
        0,
        GfBeamAvatarData.canvasSize,
        GfBeamAvatarData.canvasSize,
      ),
      Paint()..color = data.backgroundColor,
    );

    canvas.save();
    canvas.translate(data.wrapperTranslateX, data.wrapperTranslateY);
    canvas.translate(18, 18);
    canvas.rotate(data.wrapperRotate * math.pi / 180);
    canvas.translate(-18, -18);
    canvas.scale(data.wrapperScale);
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(
          0,
          0,
          GfBeamAvatarData.canvasSize,
          GfBeamAvatarData.canvasSize,
        ),
        Radius.circular(data.isCircle ? GfBeamAvatarData.canvasSize : 6),
      ),
      Paint()..color = data.wrapperColor,
    );
    canvas.restore();

    canvas.save();
    canvas.translate(data.faceTranslateX, data.faceTranslateY);
    canvas.translate(18, 18);
    canvas.rotate(data.faceRotate * math.pi / 180);
    canvas.translate(-18, -18);

    final facePaint = Paint()..color = data.faceColor;
    final mouthY = 19.0 + data.mouthSpread;
    if (data.isMouthOpen) {
      final path = Path()
        ..moveTo(15, mouthY)
        ..cubicTo(17, mouthY + 1, 19, mouthY + 1, 21, mouthY);
      canvas.drawPath(
        path,
        Paint()
          ..color = data.faceColor
          ..style = PaintingStyle.stroke
          ..strokeCap = StrokeCap.round
          ..strokeWidth = 1,
      );
    } else {
      // The Web template is `a1,0.75 0 0,0 10,0`: SVG F.6.6 grows those radii
      // to rx=5 / ry=3.75 so the arc can span the 10-unit chord, and F.6.5 puts
      // the centre at (18, mouthY) with the arc below it. Drawing that same
      // bottom half keeps the closed mouth 3.75 units deep instead of 0.75, so
      // one seed looks identical on Web and App.
      canvas.drawArc(
        Rect.fromLTWH(13, mouthY - 3.75, 10, 7.5),
        0,
        math.pi,
        false,
        facePaint,
      );
    }
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromLTWH(14 - data.eyeSpread, 14, 1.5, 2),
        const Radius.circular(1),
      ),
      facePaint,
    );
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromLTWH(20 + data.eyeSpread, 14, 1.5, 2),
        const Radius.circular(1),
      ),
      facePaint,
    );
    canvas.restore();
    canvas.restore();
  }

  @override
  bool shouldRepaint(covariant _BeamAvatarPainter oldDelegate) =>
      oldDelegate.seed != seed;
}

/// The beam parameters a [GfBeamAvatar] derives from its seed.
///
/// Exposed (instead of private like the painter) so the parity unit test can
/// pin the integer math against the Web/TypeScript port without comparing
/// pixels.
@visibleForTesting
class GfBeamAvatarData {
  GfBeamAvatarData({required String seed})
    : hash = _hashCode(seed.trim().isEmpty ? beamAnonymousSeed : seed.trim()) {
    final int length = beamColors.length;
    wrapperColor = _colorAt(hash, length);
    faceColor = _readableFaceColor(wrapperColor);
    backgroundColor = _colorAt(hash + 13, length);

    final int tx = _getUnit(hash, 10, 1);
    wrapperTranslateX = tx < 5 ? tx + canvasSize / 9 : tx.toDouble();
    final int ty = _getUnit(hash, 10, 2);
    wrapperTranslateY = ty < 5 ? ty + canvasSize / 9 : ty.toDouble();

    wrapperRotate = _getUnit(hash, 360).toDouble();
    wrapperScale = 1 + _getUnit(hash, canvasSize ~/ 12) / 10;
    isMouthOpen = _bool(hash, 2);
    isCircle = _bool(hash, 1);
    eyeSpread = _getUnit(hash, 5).toDouble();
    mouthSpread = _getUnit(hash, 3).toDouble();
    faceRotate = _getUnit(hash, 10, 3).toDouble();
    faceTranslateX = wrapperTranslateX > canvasSize / 6
        ? wrapperTranslateX / 2
        : _getUnit(hash, 8, 1).toDouble();
    faceTranslateY = wrapperTranslateY > canvasSize / 6
        ? wrapperTranslateY / 2
        : _getUnit(hash, 7, 2).toDouble();
  }

  static const double canvasSize = 36.0;

  /// Java-style 31-hash truncated to signed 32 bits, then made positive —
  /// identical to the Web `hashCode()` / `Math.abs()` pair.
  final int hash;

  late final Color wrapperColor;
  late final Color faceColor;
  late final Color backgroundColor;
  late final double wrapperTranslateX;
  late final double wrapperTranslateY;
  late final double wrapperRotate;
  late final double wrapperScale;
  late final bool isMouthOpen;
  late final bool isCircle;
  late final double eyeSpread;
  late final double mouthSpread;
  late final double faceRotate;
  late final double faceTranslateX;
  late final double faceTranslateY;

  static int _hashCode(String name) {
    int hash = 0;
    for (final int unit in name.codeUnits) {
      hash = ((hash << 5) - hash + unit).toSigned(32);
    }
    return hash.abs();
  }

  static int _digit(int number, int n) =>
      (number / math.pow(10, n)).floor() % 10;

  static bool _bool(int number, int n) => _digit(number, n).isEven;

  static int _getUnit(int number, int range, [int? index]) {
    final int value = number % range;
    return index != null && _digit(number, index).isEven ? -value : value;
  }

  static Color _colorAt(int number, int length) => beamColors[number % length];

  static Color _readableFaceColor(Color color) {
    final int red = (color.r * 255).round();
    final int green = (color.g * 255).round();
    final int blue = (color.b * 255).round();
    final double luminance = (red * 299 + green * 587 + blue * 114) / 1000;
    return luminance >= 128 ? Colors.black : Colors.white;
  }
}

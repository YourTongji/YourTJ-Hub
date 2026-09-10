import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui' show Offset, Rect, Size;

import 'package:image/image.dart' as img;

/// Normalize EXIF before previewing so the exported crop uses the same axes.
class ProfileCropSource {
  const ProfileCropSource(this.bytes, this.width, this.height);
  final Uint8List bytes;
  final int width;
  final int height;
}

ProfileCropSource prepareProfileCrop(Uint8List bytes) {
  img.Image? decoded;
  try {
    decoded = img.decodeImage(bytes);
  } catch (_) {
    throw const FormatException('Invalid image');
  }
  if (decoded == null) throw const FormatException('Invalid image');
  final source = img.bakeOrientation(decoded);
  return ProfileCropSource(img.encodePng(source), source.width, source.height);
}

/// A centered image covers the crop viewport at zoom 1. Pan is clamped so no
/// uncovered pixels can enter either the preview or the saved image.
class ProfileCropGeometry {
  ProfileCropGeometry({
    required this.source,
    required this.viewport,
    required this.zoom,
    required Offset offset,
  }) {
    final maxX = math.max(0.0, (source.width * scale - viewport.width) / 2);
    final maxY = math.max(0.0, (source.height * scale - viewport.height) / 2);
    this.offset = Offset(
      offset.dx.clamp(-maxX, maxX),
      offset.dy.clamp(-maxY, maxY),
    );
  }
  final Size source;
  final Size viewport;
  final double zoom;
  late final Offset offset;
  double get scale =>
      math.max(viewport.width / source.width, viewport.height / source.height) *
      zoom;
  Rect get destination => Rect.fromCenter(
    center: viewport.center(Offset.zero) + offset,
    width: source.width * scale,
    height: source.height * scale,
  );
  Rect get selection => Rect.fromLTWH(
    (source.width - viewport.width / scale) / 2 - offset.dx / scale,
    (source.height - viewport.height / scale) / 2 - offset.dy / scale,
    viewport.width / scale,
    viewport.height / scale,
  );
}

class ProfileCropExport {
  const ProfileCropExport(this.source, this.selection, {required this.cover});
  final ProfileCropSource source;
  final Rect selection;
  final bool cover;
}

Uint8List exportProfileCrop(ProfileCropExport request) {
  final source = img.decodePng(request.source.bytes)!;
  final r = request.selection;
  final x = r.left.round().clamp(0, source.width - 1);
  final y = r.top.round().clamp(0, source.height - 1);
  final crop = img.copyCrop(
    source,
    x: x,
    y: y,
    width: r.width.round().clamp(1, source.width - x),
    height: r.height.round().clamp(1, source.height - y),
  );
  return img.encodeWebP(
    img.copyResize(
      crop,
      width: request.cover ? 1600 : 300,
      height: request.cover ? 320 : 300,
      interpolation: img.Interpolation.average,
    ),
  );
}

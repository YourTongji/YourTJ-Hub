// Run from forum_app: dart run tool/generate_launcher_artwork.dart
// Requires librsvg's rsvg-convert. Web's vector mark is the artwork source.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:image/image.dart' as img;

void main() {
  final source = File('../../../gooseforum/resource/src/site/assets/logo.svg');
  final rendered = Process.runSync('rsvg-convert', [
    '--width',
    '2048',
    '--height',
    '2048',
    source.path,
  ], stdoutEncoding: null);
  if (rendered.exitCode != 0) {
    throw StateError('SVG export failed: ${rendered.stderr}');
  }
  final vector = img.decodePng(
    Uint8List.fromList(rendered.stdout as List<int>),
  )!;
  var left = vector.width, top = vector.height, right = 0, bottom = 0;
  for (final pixel in vector) {
    if (pixel.a == 0) continue;
    if (pixel.x < left) left = pixel.x;
    if (pixel.y < top) top = pixel.y;
    if (pixel.x > right) right = pixel.x;
    if (pixel.y > bottom) bottom = pixel.y;
  }
  final mark = img.copyCrop(
    vector,
    x: left,
    y: top,
    width: right - left + 1,
    height: bottom - top + 1,
  );
  // Fit the complete mark without clipping or baked-in rounded corners.
  final fitted = img.copyResize(
    mark,
    width: mark.width >= mark.height ? 730 : null,
    height: mark.height > mark.width ? 730 : null,
    interpolation: img.Interpolation.average,
  );
  final foreground = img.Image(width: 1024, height: 1024, numChannels: 4);
  img.compositeImage(
    foreground,
    fitted,
    dstX: (1024 - fitted.width) ~/ 2,
    dstY: (1024 - fitted.height) ~/ 2,
  );
  final icon = img.Image(width: 1024, height: 1024, numChannels: 3);
  img.fill(icon, color: img.ColorRgb8(255, 255, 255));
  img.compositeImage(icon, foreground);
  final monochrome = foreground.clone();
  for (final pixel in monochrome) {
    pixel
      ..r = 255
      ..g = 255
      ..b = 255;
  }
  final output = Directory('assets/launcher')..createSync(recursive: true);
  for (final entry in {
    'icon': icon,
    'foreground': foreground,
    'monochrome': monochrome,
  }.entries) {
    File(
      '${output.path}/${entry.key}.png',
    ).writeAsBytesSync(img.encodePng(entry.value));
  }
  File('${output.path}/source.json').writeAsStringSync(
    '${const JsonEncoder.withIndent('  ').convert({'source': source.path, 'sha256': sha256.convert(source.readAsBytesSync()).toString()})}\n',
  );
  // 0.14.4 mistakes new Xcode ASSETCATALOG boolean keys for icon names.
  // Runner already selects AppIcon; asset generation must not mutate build settings.
  final project = File('ios/Runner.xcodeproj/project.pbxproj');
  final settings = project.readAsBytesSync();
  try {
    final result = Process.runSync(Platform.resolvedExecutable, [
      'pub',
      'global',
      'run',
      'flutter_launcher_icons',
    ]);
    stdout.write(result.stdout);
    if (result.exitCode != 0) {
      throw StateError('Launcher generation failed: ${result.stderr}');
    }
  } finally {
    project.writeAsBytesSync(settings);
  }
  stdout.writeln('Generated YourTJ launcher icons from the Web vector mark.');
}

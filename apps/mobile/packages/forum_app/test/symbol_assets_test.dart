import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('every literal GfSymbol name has a UI kit asset', () {
    final assets = Directory('../ui_kit/assets/icons')
        .listSync()
        .map((entry) => entry.uri.pathSegments.last)
        .where((name) => name.endsWith('.svg'))
        .map((name) => name.substring(0, name.length - 4))
        .toSet();
    // `thumbs-down` is drawn as a rotated `thumbs-up` by the course review row.
    const drawnFromOtherGlyphs = {'thumbs-down'};
    final literal = RegExp(r"(?:GfSymbol\(\s*|symbol:\s*)'([a-z0-9-]+)'");
    final missing = <String>[];
    for (final file in Directory('lib').listSync(recursive: true)) {
      if (file is! File || !file.path.endsWith('.dart')) continue;
      for (final match in literal.allMatches(file.readAsStringSync())) {
        final name = match[1]!;
        if (!assets.contains(name) && !drawnFromOtherGlyphs.contains(name)) {
          missing.add('${file.path}: $name');
        }
      }
    }
    // A missing SVG renders nothing, so a list row silently loses its icon.
    expect(missing, isEmpty);
  });
}

import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:markdown/markdown.dart' as md;

void main() {
  final Map<String, dynamic> fixture =
      jsonDecode(
            File(
              '../../../../packages/api-contract/fixtures/markdown-image-width.json',
            ).readAsStringSync(),
          )
          as Map<String, dynamic>;

  test('presets and minimum match the shared fixture', () {
    expect(imageWidthPresets, fixture['presets']);
    expect(imageMinRenderedWidth, fixture['minRenderedWidth']);
  });

  for (final dynamic raw in fixture['parse'] as List<dynamic>) {
    final Map<String, dynamic> item = Map<String, dynamic>.from(raw as Map);
    test('parse ${item['src']}', () {
      expect(imageWidthOf(item['src'] as String), item['width']);
    });
  }

  for (final dynamic raw in fixture['set'] as List<dynamic>) {
    final Map<String, dynamic> item = Map<String, dynamic>.from(raw as Map);
    test('set ${item['src']} → ${item['width']}', () {
      expect(
        withImageWidth(item['src'] as String, item['width'] as int?),
        item['result'],
      );
    });
  }

  for (final dynamic raw in fixture['snap'] as List<dynamic>) {
    final Map<String, dynamic> item = Map<String, dynamic>.from(raw as Map);
    test('snap ${item['ratio']}', () {
      expect(snapImageWidth((item['ratio'] as num).toDouble()), item['width']);
    });
  }

  group('sizedImageRow', () {
    md.Element paragraph(String source) =>
        md.Document().parse(source).single as md.Element;

    test('image-only paragraph with a sized image becomes a row', () {
      final row = sizedImageRow(paragraph('![a](/a.png#w=50) ![b](/b.png)'));
      expect(row?.map((a) => a['src']), <String>['/a.png#w=50', '/b.png']);
    });

    test('unsized images and mixed text keep inline layout', () {
      expect(sizedImageRow(paragraph('![a](/a.png) ![b](/b.png)')), isNull);
      expect(sizedImageRow(paragraph('see ![a](/a.png#w=50)')), isNull);
      expect(sizedImageRow(paragraph('[![a](/a.png#w=50)](/x)')), isNull);
    });
  });
}

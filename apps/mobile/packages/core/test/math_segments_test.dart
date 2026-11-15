import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final Map<String, dynamic> fixture =
      jsonDecode(
            File(
              '../../../../packages/api-contract/fixtures/markdown-math-segments.json',
            ).readAsStringSync(),
          )
          as Map<String, dynamic>;

  for (final dynamic raw in fixture['cases'] as List<dynamic>) {
    final Map<String, dynamic> item = Map<String, dynamic>.from(raw as Map);
    final String source = item['source'] as String;
    test('shared fixture: ${jsonEncode(source)}', () {
      final List<Map<String, Object>> expected = <Map<String, Object>>[
        for (final dynamic segment in item['segments'] as List<dynamic>)
          <String, Object>{
            'text': (segment as Map)['text'] as String,
            'display': segment['display'] as bool,
          },
      ];
      expect(
        extractMathSegments(source)
            .map((s) => <String, Object>{'text': s.text, 'display': s.display})
            .toList(),
        expected,
      );
    });
  }

  group('ProtectedMath', () {
    test('replaces math with tokens Markdown leaves alone and splits back', () {
      final math = ProtectedMath.of(r'a $x_1*y_2$ b $$\frac{a}{b}$$');
      expect(math.source, isNot(contains(r'$')));
      expect(math.source, matches(RegExp(r'^a GFMATH0Z b GFMATH1Z$')));
      final pieces = math.split(math.source)!;
      expect(pieces.map((p) => p.math?.text ?? p.text), <String>[
        'a ',
        'x_1*y_2',
        ' b ',
        r'\frac{a}{b}',
      ]);
      expect(pieces[3].math!.display, isTrue);
      expect(pieces[1].text, r'$x_1*y_2$');
      expect(
        math.restore('`${math.source}`'),
        r'`a $x_1*y_2$ b $$\frac{a}{b}$$`',
      );
    });

    test('tokens never collide with the source', () {
      final math = ProtectedMath.of(r'GFMATH0Z $x$');
      expect(math.source, r'GFMATH0Z GFMATHX0Z');
      expect(math.split('GFMATH0Z')?.single.math, isNull);
      expect(math.restore(math.source), r'GFMATH0Z $x$');
    });

    test('text without math keeps the original path', () {
      final math = ProtectedMath.of('cost \$5 and \$10');
      expect(math.isEmpty, isTrue);
      expect(math.source, 'cost \$5 and \$10');
      expect(math.split('anything'), isNull);
    });
  });
}

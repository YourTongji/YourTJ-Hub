import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter_quill/quill_delta.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final Map<String, dynamic> fixture =
      jsonDecode(
            File(
              '../../../../packages/api-contract/fixtures/markdown-blank-lines.json',
            ).readAsStringSync(),
          )
          as Map<String, dynamic>;

  test('fixture cap matches the renderer cap', () {
    expect(fixture['maxRendered'], maxRenderedBlankLines);
  });

  for (final dynamic rawCase in fixture['cases'] as List<dynamic>) {
    final Map<String, dynamic> item = Map<String, dynamic>.from(
      rawCase as Map<dynamic, dynamic>,
    );
    test('shared fixture: ${item['name']}', () {
      final String markdown = item['markdown'] as String;
      expect(
        markBlankLines(
          markdown,
          fixture['marker'] as String,
          max: item['unlimited'] == true ? null : fixture['maxRendered'] as int,
        ),
        (item['marked'] as String?) ?? markdown,
      );
    });
  }

  test('marker never occurs in the source', () {
    expect(blankLineMarker('甲'), 'GFBLANKLINE');
    expect(blankLineMarker('GFBLANKLINE GFBLANKLINE0'), 'GFBLANKLINE1');
  });

  group('MarkdownConverter keeps empty paragraphs', () {
    final MarkdownConverter converter = MarkdownConverter();

    test('Web blank-line runs load as empty lines and save back', () {
      for (final int empty in <int>[1, 2, 5]) {
        final String source = '甲\n${'\n' * (empty + 1)}乙';
        final Delta delta = converter.mdToDelta(source);
        expect(delta.toJson(), <Map<String, dynamic>>[
          <String, dynamic>{'insert': '甲\n${'\n' * empty}乙\n'},
        ]);
        expect(converter.deltaToMarkdown(delta).trim(), source);
      }
    });

    test('a paragraph break stays a single blank line', () {
      final Delta delta = Delta()..insert('甲\n乙\n');
      expect(converter.deltaToMarkdown(delta).trim(), '甲\n\n乙');
      expect(converter.mdToDelta('甲\n\n乙').toJson(), delta.toJson());
    });

    test('empty lines around a heading keep their count', () {
      final Delta delta = Delta()
        ..insert('标题')
        ..insert('\n', <String, dynamic>{'header': 1})
        ..insert('\n正文\n');
      final String markdown = converter.deltaToMarkdown(delta).trim();
      expect(markdown, '# 标题\n\n\n正文');
      expect(converter.mdToDelta(markdown).toJson(), delta.toJson());
    });

    test('blank lines inside fenced code stay literal', () {
      const String source = '```\na\n\n\nb\n```';
      final String saved = converter
          .deltaToMarkdown(converter.mdToDelta(source))
          .trim();
      expect(saved, contains('a\n\n\nb'));
      expect(saved, isNot(contains('GFBLANKLINE')));
    });

    test('text that looks like the marker survives', () {
      const String source = 'GFBLANKLINE\n\n\n乙';
      final Delta delta = converter.mdToDelta(source);
      expect(delta.toJson(), <Map<String, dynamic>>[
        <String, dynamic>{'insert': 'GFBLANKLINE\n\n乙\n'},
      ]);
      expect(converter.deltaToMarkdown(delta).trim(), source);
    });
  });
}

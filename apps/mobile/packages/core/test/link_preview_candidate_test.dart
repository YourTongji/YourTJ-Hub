import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final File fixtureFile = File(
    '../../../../packages/api-contract/fixtures/link-preview-markdown-candidates.json',
  );
  final Map<String, dynamic> fixture =
      jsonDecode(fixtureFile.readAsStringSync()) as Map<String, dynamic>;
  final List<dynamic> cases = fixture['cases'] as List<dynamic>;

  for (final dynamic rawCase in cases) {
    final Map<String, dynamic> fixtureCase = Map<String, dynamic>.from(
      rawCase as Map<dynamic, dynamic>,
    );
    test('shared candidate fixture: ${fixtureCase['name']}', () {
      expect(
        scanLinkPreviewCandidates(fixtureCase['markdown'] as String),
        (fixtureCase['urls'] as List<dynamic>).cast<String>(),
      );
    });
  }

  test('split blocks preserve non-preview markdown', () {
    const String source = 'Before\n\nhttps://example.com/a\n\nAfter';
    final List<LinkPreviewMarkdownBlock> blocks = splitLinkPreviewMarkdown(
      source,
    );
    expect(blocks.map((block) => block.url), <String?>[
      null,
      'https://example.com/a',
      null,
    ]);
    expect(
      blocks
          .where((block) => !block.isPreview)
          .map((block) => block.markdown.trim()),
      <String>['Before', 'After'],
    );
  });
}

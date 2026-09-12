import 'package:core/core.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_quill/quill_delta.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/publish/embed_image_move.dart';

Iterable<Node> _childrenOf(Node node) {
  if (node is Block) return node.children;
  if (node is Line) return node.children;
  return const Iterable.empty();
}

Embed? _findEmbed(Document document) {
  for (final Node top in document.root.children) {
    for (final Node mid in _childrenOf(top)) {
      if (mid is Embed) return mid;
      for (final Node leaf in _childrenOf(mid)) {
        if (leaf is Embed) return leaf;
      }
    }
  }
  return null;
}

Document _documentOf(String plain) {
  // Merge consecutive text characters into single ops and serialize embeds
  // through toJson, mirroring the production Markdown hydration path —
  // quill's internal _delta must match the tree's merged serialization or
  // every later compose trips an assertion.
  final Delta delta = Delta();
  final StringBuffer text = StringBuffer();
  for (final String char in plain.split('')) {
    if (char == '\uFFFC') {
      if (text.isNotEmpty) {
        delta.insert(text.toString());
        text.clear();
      }
      delta.insert(BlockEmbed.image('u1').toJson());
    } else {
      text.write(char);
    }
  }
  if (text.isNotEmpty) delta.insert(text.toString());
  return Document.fromDelta(delta);
}

/// Flat-text view of a document where embeds render as \uFFFC — the same
/// notation the test sources use, so expectations read like the input.
String _flat(Document document) {
  final StringBuffer buffer = StringBuffer();
  for (final Operation op in document.toDelta().toList()) {
    buffer.write(op.data is String ? op.data as String : '\uFFFC');
  }
  return buffer.toString();
}

void main() {
  group('locateEmbedOffset', () {
    test('returns the embed document offset', () {
      final Document doc = _documentOf('A\n\uFFFC\nB\n');
      final Embed? embed = _findEmbed(doc);
      expect(embed, isNotNull);
      expect(locateEmbedOffset(doc, embed!), 2);
    });

    test('returns null for a node from another document (stale)', () {
      final Document doc = _documentOf('A\n');
      final Document other = _documentOf('\uFFFC\n');
      final Embed? stale = _findEmbed(other);
      expect(locateEmbedOffset(doc, stale!), isNull);
    });
  });

  group('clampDropToLineEnd', () {
    test('snaps a mid-line drop to the end of its line', () {
      final Document doc = _documentOf('AB\nCD\n');
      expect(clampDropToLineEnd(1, doc), 2);
      expect(clampDropToLineEnd(4, doc), 5);
    });

    test('keeps offsets at line starts and document bounds', () {
      final Document doc = _documentOf('AB\nCD\n');
      // Offsets inside the "AB" line all snap to that line's newline (2).
      expect(clampDropToLineEnd(0, doc), 2);
      expect(clampDropToLineEnd(1, doc), 2);
      expect(clampDropToLineEnd(2, doc), 2);
      // Offsets inside the "CD" line snap to 5.
      expect(clampDropToLineEnd(3, doc), 5);
      expect(clampDropToLineEnd(5, doc), 5);
      expect(clampDropToLineEnd(6, doc), 6);
      expect(clampDropToLineEnd(99, doc), 6);
      expect(clampDropToLineEnd(-1, doc), 0);
    });
  });

  group('moveComposerImageEmbed', () {
    // Drop semantics: the image appends to the paragraph the user dropped
    // on — a solo-line image prefers its own line directly below it. The
    // document's final line has no slot below, so a drop there glues the
    // image to the line end instead (expanded embeds render identically).
    test('moves a solo-line image down onto a later paragraph', () {
      final Document doc = _documentOf('A\n\uFFFC\nB\nC\nD\n');
      final Embed embed = _findEmbed(doc)!;
      // Drop inside the "C" line — image lands on its own line below C.
      final result = moveComposerImageEmbed(doc, embed, 7);
      expect(result, isNotNull);
      doc.compose(result!.delta, ChangeSource.local);
      expect(_flat(doc), 'A\nB\nC\n\uFFFC\nD\n');
      expect(result.insertOffset, 6);
      expect(result.insertLength, 2);
    });

    test('moves a solo-line image up under an earlier paragraph', () {
      final Document doc = _documentOf('A\nB\n\uFFFC\n');
      final Embed embed = _findEmbed(doc)!;
      // Drop inside the "A" line — image lands directly below A.
      final result = moveComposerImageEmbed(doc, embed, 1);
      expect(result, isNotNull);
      doc.compose(result!.delta, ChangeSource.local);
      expect(_flat(doc), 'A\n\uFFFC\nB\n');
      expect(result.insertOffset, 2);
      expect(result.insertLength, 2);
    });

    test('returns null when the drop targets the current slot', () {
      final Document doc = _documentOf('A\n\uFFFC\nB\n');
      final Embed embed = _findEmbed(doc)!;
      // Dropping on the paragraph directly above the image, or on the
      // image's own line, keeps the image exactly where it is.
      expect(moveComposerImageEmbed(doc, embed, 0), isNull);
      expect(moveComposerImageEmbed(doc, embed, 1), isNull);
      expect(moveComposerImageEmbed(doc, embed, 2), isNull);
      expect(moveComposerImageEmbed(doc, embed, 3), isNull);
    });

    test('moves an inline image to the end of the dropped-on line', () {
      final Document doc = _documentOf('pre\uFFFCst\nnext\n');
      final Embed embed = _findEmbed(doc)!;
      final result = moveComposerImageEmbed(doc, embed, 9);
      expect(result, isNotNull);
      doc.compose(result!.delta, ChangeSource.local);
      expect(_flat(doc), 'prest\nnext\uFFFC\n');
      expect(result.insertLength, 1);
    });

    test('returns null for stale nodes and out-of-range targets', () {
      final Document doc = _documentOf('A\n');
      final Document other = _documentOf('\uFFFC\n');
      final Embed stale = _findEmbed(other)!;
      expect(moveComposerImageEmbed(doc, stale, 0), isNull);
      final Document doc2 = _documentOf('A\n\uFFFC\n');
      final Embed embed = _findEmbed(doc2)!;
      expect(moveComposerImageEmbed(doc2, embed, -1), isNull);
      expect(moveComposerImageEmbed(doc2, embed, 99), isNull);
    });

    test('moving through a QuillController stays a single undo step', () {
      final Document doc = _documentOf('A\nB\n\uFFFC\nC\n');
      final QuillController controller = QuillController(
        document: doc,
        selection: const TextSelection.collapsed(offset: 0),
      );
      final String before = _flat(controller.document);
      final Embed embed = _findEmbed(controller.document)!;
      final result = moveComposerImageEmbed(controller.document, embed, 1);
      controller.compose(
        result!.delta,
        controller.selection,
        ChangeSource.local,
      );
      expect(_flat(controller.document), 'A\n\uFFFC\nB\nC\n');

      controller.undo();
      expect(_flat(controller.document), before);
    });

    test('moves a trailing solo-line image up (undo limitation upstream)', () {
      // flutter_quill cannot re-insert content at index == document length
      // when replaying history, so undoing a move whose SOURCE was the
      // trailing line trips a QuillContainer assertion. The same crash is
      // reachable without this feature: natively deleting a trailing image
      // and pressing undo hits it too. The forward move itself is safe and
      // asserted here; fixing the upstream history limitation is out of
      // scope for this change.
      final Document doc = _documentOf('A\nB\n\uFFFC\n');
      final Embed embed = _findEmbed(doc)!;
      final result = moveComposerImageEmbed(doc, embed, 1);
      expect(result, isNotNull);
      doc.compose(result!.delta, ChangeSource.local);
      expect(_flat(doc), 'A\n\uFFFC\nB\n');
    });

    test('moved document round-trips through the markdown converter', () {
      final Document doc = _documentOf('A\n\uFFFC\nB\nC\nD\n');
      final Embed embed = _findEmbed(doc)!;
      final result = moveComposerImageEmbed(doc, embed, 7);
      doc.compose(result!.delta, ChangeSource.local);
      final String markdown = MarkdownConverter().documentToMarkdown(doc);
      expect(markdown, contains('A'));
      expect(markdown, contains('D'));
      // The image must now sit between C and D in the exported markdown.
      expect(markdown.indexOf('C'), lessThan(markdown.indexOf('u1')));
      expect(markdown.indexOf('u1'), lessThan(markdown.indexOf('D')));
    });
  });
}

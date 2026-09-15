import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_quill/quill_delta.dart';

/// Payload carried while a composer image embed is being dragged.
///
/// [node] is the live quill leaf at drag start. Handlers re-locate it by
/// identity against the current document before mutating anything, so a drag
/// whose document was replaced mid-flight (edit-mode reload) is a safe no-op.
class ComposerImageDragPayload {
  const ComposerImageDragPayload({required this.node, required this.imageUrl});

  final Embed node;
  final String imageUrl;
}

/// Result of a successful embed move expressed as a single Delta.
class ComposerImageMoveResult {
  const ComposerImageMoveResult({
    required this.delta,
    required this.insertOffset,
    required this.insertLength,
  });

  /// The single delta that relocates the embed.
  final Delta delta;

  /// Document offset where the embed landed after the move.
  final int insertOffset;

  /// Number of characters inserted (embed plus, for solo-line moves, the
  /// trailing line break).
  final int insertLength;
}

/// Returns the document offset of [node], or null when it does not belong to
/// [document] (stale node after a document replacement).
int? locateEmbedOffset(Document document, Node node) {
  Iterable<Node> childrenOf(Node parent) {
    if (parent is Block) return parent.children;
    if (parent is Line) return parent.children;
    return const Iterable.empty();
  }

  for (final Node top in document.root.children) {
    for (final Node mid in childrenOf(top)) {
      if (identical(mid, node)) return mid.documentOffset;
      for (final Node leaf in childrenOf(mid)) {
        if (identical(leaf, node)) return leaf.documentOffset;
      }
    }
  }
  return null;
}

/// Flattens the document into a plain string where every non-text operation
/// (embed) is represented by the object replacement character.
String documentFlatText(Document document) {
  final StringBuffer flat = StringBuffer();
  for (final Operation op in document.toDelta().toList()) {
    flat.write(op.data is String ? op.data as String : '\uFFFC');
  }
  return flat.toString();
}

/// Returns the attributes of the operation covering flat-text [offset].
Map<String, dynamic>? attributesAtOffset(List<Operation> ops, int offset) {
  int cursor = 0;
  for (final Operation op in ops) {
    final int length = op.length ?? 0;
    if (offset < cursor + length) return op.attributes;
    cursor += length;
  }
  return null;
}

/// Returns the index of the newline that terminates the visual line
/// containing [offset] — the anchor for "the image lands at the end of the
/// paragraph the user dropped on".
int clampDropToLineEnd(int offset, Document document) {
  final int length = document.length;
  if (offset < 0) return 0;
  if (offset >= length) return length;
  final String text = documentFlatText(document);
  for (int i = offset; i < length; i++) {
    if (text[i] == '\n') return i;
  }
  return length;
}

/// Computes the single Delta that relocates the embed leaf [node] to the end
/// of the line containing [targetOffset].
///
/// Drop semantics: the image appends to the paragraph the user dropped on.
/// A solo-line image (image alone on its own line) prefers the slot directly
/// below that paragraph (its own new line); when the dropped-on paragraph is
/// the document's final line there is no slot below it, so the image glues
/// to the line's end instead — expanded embeds render identically. Inline
/// images always append to the end of the dropped-on line's text.
/// The source side collapses cleanly without touching block formatting: a
/// solo image prefers its OWN trailing newline (the line-above's — possibly
/// styled — newline survives untouched); only when that newline is the
/// document's mandatory final one does it fall back to the preceding
/// newline, whose block attributes are re-applied to the surviving newline.
/// Inline images leave surrounding text intact.
///
/// Returns null when the move is impossible or a no-op: stale node, drop on
/// the image's own line, or a drop whose slot is exactly where the image
/// already sits. Composing the returned delta in one
/// [QuillController.compose] call keeps undo/redo a single step.
ComposerImageMoveResult? moveComposerImageEmbed(
  Document document,
  Embed node,
  int targetOffset,
) {
  final int? located = locateEmbedOffset(document, node);
  if (located == null) return null;
  final int embedOffset = located;
  final int documentLength = document.length;
  if (targetOffset < 0 || targetOffset > documentLength) return null;

  final List<Operation> ops = document.toDelta().toList();
  final String text = documentFlatText(document);
  String charAt(int index) =>
      index >= 0 && index < text.length ? text[index] : '';

  // The op covering embedOffset must be the embed itself.
  int cursor = 0;
  Operation? embedOp;
  for (final Operation op in ops) {
    final int length = op.length ?? 0;
    if (embedOffset < cursor + length) {
      embedOp = op;
      break;
    }
    cursor += length;
  }
  if (embedOp == null || embedOp.data is String) return null;
  if (charAt(embedOffset) != '\uFFFC') return null;

  final String charBefore = charAt(embedOffset - 1);
  final String charAfter = charAt(embedOffset + 1);
  final bool soloLine =
      (embedOffset == 0 || charBefore == '\n') &&
      (charAfter == '\n' || charAfter.isEmpty);
  final int delLength = soloLine ? 2 : 1;
  Map<String, dynamic>? consumedLineAttrs;
  late final int delStart;
  if (soloLine && embedOffset + 1 < text.length - 1 && charAfter == '\n') {
    // Preferred: take the image's own trailing newline. The line-above's
    // newline — which carries its block style (heading/list/quote) —
    // survives untouched.
    delStart = embedOffset;
  } else if (soloLine && embedOffset > 0) {
    // Fallback (trailing image): the image's trailing newline IS the
    // document's final newline. Quill's history replay re-inserts the
    // deleted range at its start offset and cannot insert at index ==
    // document length, so consuming the final newline would crash undo.
    // Take the preceding newline instead and carry its block attributes
    // over to the newline that survives the collapse.
    delStart = embedOffset - 1;
    consumedLineAttrs = attributesAtOffset(ops, delStart);
  } else {
    delStart = embedOffset;
  }

  final int dropNewline = targetOffset >= documentLength
      ? documentLength - 1
      : clampDropToLineEnd(targetOffset, document);

  bool soloInsert = soloLine;
  int insertOffset = soloInsert ? dropNewline + 1 : dropNewline;
  if (insertOffset > documentLength - 1) {
    // No slot exists below the final line; glue to the line end instead.
    insertOffset = documentLength - 1;
    soloInsert = false;
  }
  final int insertLength = soloInsert ? 2 : 1;

  // No-ops: the image already occupies the target slot — dropping on the
  // paragraph directly above it, on its own line, or (when it is already
  // the last line) anywhere at or below itself.
  if (insertOffset == embedOffset) return null;
  if (soloLine &&
      (insertOffset == embedOffset + delLength ||
          insertOffset == embedOffset + 1)) {
    return null;
  }

  final Delta delta = Delta();
  if (insertOffset > embedOffset) {
    // Moving down: delete first, then express the insert in post-deletion
    // coordinates at the same original offset.
    delta.retain(delStart);
    delta.delete(delLength);
    delta.retain(insertOffset - delStart - delLength);
    delta.insert(embedOp.data, embedOp.attributes);
    if (soloInsert) delta.insert('\n');
    return ComposerImageMoveResult(
      delta: delta,
      insertOffset: insertOffset - delLength,
      insertLength: insertLength,
    );
  }

  // Moving up: insert first, then delete at the source position. Delta
  // retains/deletes always use ORIGINAL document coordinates — an insert
  // does not advance the stream cursor.
  delta.retain(insertOffset);
  delta.insert(embedOp.data, embedOp.attributes);
  if (soloInsert) delta.insert('\n');
  delta.retain(delStart - insertOffset);
  delta.delete(delLength);
  if (consumedLineAttrs != null && consumedLineAttrs.isNotEmpty) {
    delta.retain(1, consumedLineAttrs);
  }
  return ComposerImageMoveResult(
    delta: delta,
    insertOffset: insertOffset,
    insertLength: insertLength,
  );
}

import 'dart:async';

import 'package:flutter/services.dart' show TextSelection;
import 'package:flutter_quill/flutter_quill.dart' as quill;
import 'package:flutter_quill/quill_delta.dart' as quill_delta;

import 'sticker_token.dart';

/// Rich (Quill) editors hold each sticker as one inline embed, so it shows as
/// the image it publishes as and is deleted as a unit (MADR 0106 supplement).
/// Markdown keeps the `[:sticker:name:]` token: [MarkdownConverter] turns
/// tokens into embeds on load and writes embeds back as tokens on save.
const String stickerEmbedType = 'sticker';

class StickerEmbed extends quill.Embeddable {
  const StickerEmbed(String name) : super(stickerEmbedType, name);
}

/// The Markdown source of a sticker embed.
String stickerEmbedToken(quill.Embed embed) =>
    '[:sticker:${embed.value.data}:]';

final RegExp _mathPattern = RegExp(
  r'\$\$[\s\S]+?\$\$|\$(?!\s)[^\n]+?(?<!\s)\$'
  r'|\\\([\s\S]+?\\\)|\\\[[\s\S]+?\\\]',
);

/// Complete tokens in the document's prose that should become embeds, in
/// document offsets. Code blocks, inline code, links and math stay text, like
/// the server renderer; so do tokens escaped by an odd run of backslashes and
/// tokens split across differently styled runs.
List<StickerTokenRange> documentStickerTokens(quill.Document document) {
  final plain = document.toPlainText();
  if (!containsStickerToken(plain)) return const [];
  final math = _mathPattern.allMatches(plain).toList();
  final ranges = <StickerTokenRange>[];

  void visitLine(quill.Line line) {
    if (line.style.containsKey(quill.Attribute.codeBlock.key)) return;
    for (final leaf in line.children) {
      if (leaf is! quill.QuillText) continue;
      final style = leaf.style;
      if (style.containsKey(quill.Attribute.inlineCode.key) ||
          style.containsKey(quill.Attribute.link.key)) {
        continue;
      }
      final offset = leaf.documentOffset;
      for (final match in stickerTokenPattern.allMatches(leaf.value)) {
        var slashes = 0;
        for (
          var i = offset + match.start - 1;
          i >= 0 && plain[i] == r'\';
          i--
        ) {
          slashes++;
        }
        final start = offset + match.start;
        final end = offset + match.end;
        if (slashes.isOdd ||
            math.any((span) => start < span.end && span.start < end)) {
          continue;
        }
        ranges.add(StickerTokenRange(start, end, match.group(1)!));
      }
    }
  }

  // Quill blocks (quotes, lists, code) hold lines; they never nest.
  for (final child in document.root.children) {
    if (child is quill.Line) {
      visitLine(child);
    } else if (child is quill.Block) {
      child.children.whereType<quill.Line>().forEach(visitLine);
    }
  }
  return ranges;
}

/// [delta] with every eligible token replaced by an embed carrying the
/// token's inline attributes (bold, italic…), so emphasis is not split.
quill_delta.Delta embedStickerTokens(quill_delta.Delta delta) {
  final hasToken = delta.toList().any(
    (op) => op.data is String && containsStickerToken(op.data! as String),
  );
  if (!hasToken) return delta;
  final document = quill.Document.fromDelta(delta);
  final ranges = documentStickerTokens(document);
  if (ranges.isEmpty) return delta;
  final change = quill_delta.Delta();
  var cursor = 0;
  for (final range in ranges) {
    final style = document.collectStyle(range.start, range.end - range.start);
    final attributes = <String, dynamic>{
      for (final attribute in style.attributes.values)
        if (attribute.scope == quill.AttributeScope.inline)
          attribute.key: attribute.value,
    };
    change
      ..retain(range.start - cursor)
      ..delete(range.end - range.start)
      ..insert(<String, String>{
        stickerEmbedType: range.name,
      }, attributes.isEmpty ? null : attributes);
    cursor = range.end;
  }
  return delta.compose(change);
}

/// Converts tokens typed, pasted or inserted into a live editor. The edit
/// joins the keystroke's undo entry, so undo returns to the unclosed fragment
/// instead of re-creating the token. Returns whether anything changed.
bool embedStickerTokensIn(quill.QuillController controller) {
  final ranges = documentStickerTokens(controller.document);
  if (ranges.isEmpty) return false;
  var selection = controller.selection;
  for (final range in ranges.reversed) {
    final length = range.end - range.start;
    int shift(int offset) {
      if (offset >= range.end) return offset - length + 1;
      if (offset > range.start) return range.start + 1;
      return offset;
    }

    final style = controller.document.collectStyle(
      range.start,
      range.end - range.start,
    );
    controller.replaceText(
      range.start,
      length,
      StickerEmbed(range.name),
      null,
      ignoreFocus: true,
    );
    for (final attribute in style.attributes.values) {
      if (attribute.scope == quill.AttributeScope.inline) {
        controller.formatText(range.start, 1, attribute);
      }
    }
    if (selection.isValid) {
      selection = TextSelection(
        baseOffset: shift(selection.baseOffset),
        extentOffset: shift(selection.extentOffset),
      );
    }
  }
  if (selection.isValid) {
    controller.updateSelection(selection, quill.ChangeSource.local);
  }
  return true;
}

/// Keeps a live editor's typed and inserted tokens as embeds. A token is
/// complete only once its closing `]` arrives, so other edits skip the scan.
class StickerTokenWatcher {
  StickerTokenWatcher(this.controller) {
    _subscription = controller.document.changes.listen(_changed);
  }

  final quill.QuillController controller;
  late final StreamSubscription<quill.DocChange> _subscription;
  bool _scheduled = false;
  bool _cancelled = false;

  void _changed(quill.DocChange change) {
    if (_scheduled || _cancelled) return;
    final closesToken = change.change.toList().any(
      (op) =>
          op.isInsert &&
          op.data is String &&
          (op.data! as String).contains(']'),
    );
    if (!closesToken) return;
    _scheduled = true;
    // Never edit the document from inside its own change notification.
    scheduleMicrotask(() {
      _scheduled = false;
      if (!_cancelled) embedStickerTokensIn(controller);
    });
  }

  Future<void> cancel() {
    _cancelled = true;
    return _subscription.cancel();
  }
}

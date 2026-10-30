import 'dart:async';
import 'dart:math' as math;

import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import 'sticker_image.dart';

/// Plain-text composers (replies, post edits, the simple publisher, chat)
/// show every resolved `[:sticker:name:]` token as the image it publishes as,
/// at the published inline size, while the text value stays the Markdown
/// token (MADR 0106 supplement).
///
/// - A token draws as one inline image followed by its remaining characters
///   at zero width, so every offset in [value] still maps onto the layout.
/// - The caret never rests inside an image token: moving into one lands on
///   its far side; selections widen to whole tokens.
/// - One delete removes a whole token or a half-typed `[:sticker:` fragment
///   (issue #1332), matching the web editors.
/// - Unknown and disabled tokens, and Markdown tokens inside code or links,
///   stay ordinary text, exactly as they render once published.
/// - Lines holding an image grow to fit it instead of overlapping their
///   neighbours; fields apply [GfInlineContentController.strut].
class StickerTextEditingController extends TextEditingController
    implements GfInlineContentController {
  StickerTextEditingController({super.text, this.markdown = true});

  /// Posts follow Markdown rules; chat messages are plain text.
  final bool markdown;

  /// Characters after the image placeholder are laid out but invisible.
  static const double _hiddenFontSize = 0.01;

  StickerLibrary? _library;
  Set<String> _requestedNames = const {};
  String? _rangesText;
  List<StickerTokenRange> _ranges = const [];
  String _shownSignature = '';

  /// Session resolver supplying sticker URLs; hosts assign it from their
  /// provider on every build, which is cheap when it is unchanged.
  StickerLibrary? get library => _library;
  set library(StickerLibrary? next) {
    if (identical(next, _library)) return;
    _library?.removeListener(_libraryChanged);
    _library = next;
    _requestedNames = const {};
    next?.addListener(_libraryChanged);
    _resolveNames();
  }

  List<StickerTokenRange> get _tokenRanges {
    if (_rangesText != text) {
      _rangesText = text;
      _ranges = stickerTokenRanges(text, markdown: markdown);
    }
    return _ranges;
  }

  /// Token ranges currently drawn as images.
  @visibleForTesting
  List<StickerTokenRange> get imageRanges {
    final urls = _library?.urlByName;
    if (urls == null || urls.isEmpty) return const [];
    return [
      for (final range in _tokenRanges)
        if (urls[range.name]?.isNotEmpty ?? false) range,
    ];
  }

  String _signature(List<StickerTokenRange> ranges) =>
      ranges.map((range) => '${range.start}:${range.name}').join(',');

  void _libraryChanged() {
    // Only a change to what this field draws repaints it; listeners of the
    // text (draft autosave, dirty flags) otherwise never hear from the library.
    if (_signature(imageRanges) != _shownSignature) notifyListeners();
  }

  void _resolveNames() {
    final library = _library;
    if (library == null) return;
    final names = {for (final range in _tokenRanges) range.name};
    if (setEquals(names, _requestedNames)) return;
    _requestedNames = names;
    final known = library.urlByName;
    if (names.every(known.containsKey)) return;
    unawaited(library.resolveContent(text).catchError((Object _) {}));
  }

  @override
  set value(TextEditingValue newValue) {
    super.value = _atomic(value, newValue);
    _resolveNames();
  }

  TextEditingValue _atomic(TextEditingValue old, TextEditingValue next) {
    if (next.text.length < old.text.length) {
      final removed = _removedRange(old, next);
      if (removed == null) return next;
      final expanded = expandStickerDeletion(
        old.text,
        removed.start,
        removed.end,
      );
      if (expanded == removed) return next;
      // The IME composed against the shorter text; drop its region, the field
      // sends the corrected value back to the input method.
      return TextEditingValue(
        text: old.text.replaceRange(expanded.start, expanded.end, ''),
        selection: TextSelection.collapsed(offset: expanded.start),
      );
    }
    if (next.text == old.text && next.selection != old.selection) {
      return next.copyWith(selection: _snap(old.selection, next.selection));
    }
    return next;
  }

  /// The contiguous range [old] lost to become [next], or null when the edit
  /// replaced text. A collapsed caret disambiguates repeated characters.
  static TextRange? _removedRange(TextEditingValue old, TextEditingValue next) {
    final removedLength = old.text.length - next.text.length;
    final caret = old.selection;
    if (caret.isValid) {
      final starts = caret.isCollapsed
          ? [caret.start - removedLength, caret.start]
          : [if (caret.end - caret.start == removedLength) caret.start];
      for (final start in starts) {
        if (start < 0 || start + removedLength > old.text.length) continue;
        final range = TextRange(start: start, end: start + removedLength);
        if (old.text.replaceRange(range.start, range.end, '') == next.text) {
          return range;
        }
      }
    }
    var prefix = 0;
    while (prefix < next.text.length &&
        old.text.codeUnitAt(prefix) == next.text.codeUnitAt(prefix)) {
      prefix++;
    }
    var suffix = 0;
    while (suffix < next.text.length - prefix &&
        old.text.codeUnitAt(old.text.length - 1 - suffix) ==
            next.text.codeUnitAt(next.text.length - 1 - suffix)) {
      suffix++;
    }
    if (prefix + suffix != next.text.length) return null;
    return TextRange(start: prefix, end: old.text.length - suffix);
  }

  TextSelection _snap(TextSelection old, TextSelection next) {
    if (!next.isValid) return next;
    final images = imageRanges;
    if (images.isEmpty) return next;
    StickerTokenRange? inside(int offset) {
      for (final range in images) {
        if (offset > range.start && offset < range.end) return range;
      }
      return null;
    }

    if (next.isCollapsed) {
      final range = inside(next.baseOffset);
      if (range == null) return next;
      // Every inner offset draws at the image's trailing edge; only a caret
      // stepping back from beyond the token lands before it.
      final back =
          old.isValid && old.isCollapsed && old.baseOffset >= range.end;
      return TextSelection.collapsed(offset: back ? range.start : range.end);
    }
    int widen(int offset, {required bool towardStart}) {
      final range = inside(offset);
      if (range == null) return offset;
      return towardStart ? range.start : range.end;
    }

    final forward = next.baseOffset <= next.extentOffset;
    return TextSelection(
      baseOffset: widen(next.baseOffset, towardStart: forward),
      extentOffset: widen(next.extentOffset, towardStart: !forward),
    );
  }

  @override
  TextSpan buildTextSpan({
    required BuildContext context,
    TextStyle? style,
    required bool withComposing,
  }) {
    final images = imageRanges;
    _shownSignature = _signature(images);
    if (images.isEmpty) {
      return super.buildTextSpan(
        context: context,
        style: style,
        withComposing: withComposing,
      );
    }
    final urls = _library!.urlByName;
    final composing = withComposing && value.isComposingRangeValid
        ? value.composing
        : TextRange.empty;
    final composingStyle =
        style?.merge(const TextStyle(decoration: TextDecoration.underline)) ??
        const TextStyle(decoration: TextDecoration.underline);
    final hidden = TextStyle(
      fontSize: _hiddenFontSize,
      color: Colors.transparent,
      letterSpacing: 0,
      wordSpacing: 0,
      decoration: TextDecoration.none,
    );
    final size = (style?.fontSize ?? 16) * StickerImage.inlineScale;
    final children = <InlineSpan>[];

    void addText(int start, int end) {
      if (start >= end) return;
      if (composing.isValid &&
          !composing.isCollapsed &&
          composing.start < end &&
          start < composing.end) {
        final from = math.max(start, composing.start);
        final to = math.min(end, composing.end);
        if (from > start) {
          children.add(TextSpan(text: text.substring(start, from)));
        }
        children.add(
          TextSpan(text: text.substring(from, to), style: composingStyle),
        );
        if (end > to) children.add(TextSpan(text: text.substring(to, end)));
        return;
      }
      children.add(TextSpan(text: text.substring(start, end)));
    }

    var cursor = 0;
    for (final range in images) {
      addText(cursor, range.start);
      children
        ..add(
          WidgetSpan(
            alignment: PlaceholderAlignment.middle,
            // Taps belong to the field (caret placement), not the sticker card.
            child: IgnorePointer(
              child: StickerImage(
                key: ValueKey('composer-sticker-${range.start}'),
                name: range.name,
                url: urls[range.name]!,
                size: size,
                collectible: false,
                excludeSemantics: true,
              ),
            ),
          ),
        )
        ..add(
          TextSpan(
            text: text.substring(range.start + 1, range.end),
            style: hidden,
          ),
        );
      cursor = range.end;
    }
    addText(cursor, text.length);
    return TextSpan(style: style, children: children);
  }

  @override
  double? get inlineExtentScale =>
      imageRanges.isEmpty ? null : StickerImage.inlineScale;

  @override
  void dispose() {
    _library?.removeListener(_libraryChanged);
    _library = null;
    super.dispose();
  }
}

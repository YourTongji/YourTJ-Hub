import 'package:flutter/cupertino.dart' as cupertino;
import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'twemoji_asset_map.g.dart';

/// A text segment that should use the bundled Twemoji artwork.
@immutable
class TwemojiSegment {
  const TwemojiSegment.text(this.value) : asset = null;
  const TwemojiSegment.emoji(this.value, this.asset);

  final String value;
  final String? asset;

  bool get isEmoji => asset != null;

  @override
  bool operator ==(Object other) =>
      other is TwemojiSegment && other.value == value && other.asset == asset;

  @override
  int get hashCode => Object.hash(value, asset);
}

const String twemojiVersion = twemojiAssetVersion;

// Derived once from the complete manifest so the common no-emoji path can
// skip the sequence parser without maintaining a second Unicode range table.
final Set<int> _twemojiFirstCodepoints = twemojiAssetBySequence.keys
    .map((sequence) => sequence.runes.first)
    .toSet();

bool _mightContainTwemoji(String text) {
  var offset = 0;
  while (offset < text.length) {
    final first = text.codeUnitAt(offset);
    var codepoint = first;
    var codeUnitLength = 1;
    if (first >= 0xd800 && first <= 0xdbff && offset + 1 < text.length) {
      final second = text.codeUnitAt(offset + 1);
      if (second >= 0xdc00 && second <= 0xdfff) {
        codepoint = 0x10000 + ((first - 0xd800) << 10) + (second - 0xdc00);
        codeUnitLength = 2;
      }
    }

    if (_twemojiFirstCodepoints.contains(codepoint)) {
      final isKeycapStarter =
          codepoint == 0x23 ||
          codepoint == 0x2a ||
          (codepoint >= 0x30 && codepoint <= 0x39);
      if (!isKeycapStarter) return true;

      var suffixOffset = offset + codeUnitLength;
      if (suffixOffset < text.length &&
          text.codeUnitAt(suffixOffset) == 0xfe0f) {
        suffixOffset++;
      }
      if (suffixOffset < text.length &&
          text.codeUnitAt(suffixOffset) == 0x20e3) {
        return true;
      }
    }
    offset += codeUnitLength;
  }
  return false;
}

class _TwemojiSelectionRegistry {
  _TwemojiSelectionRegistry(String selectionSource) {
    _sourceCodeUnits.addAll(selectionSource.codeUnits);
  }

  final Map<String, int> _placeholderByEmoji = <String, int>{};
  final Map<int, String> emojiByPlaceholder = <int, String>{};
  final Set<int> _sourceCodeUnits = <int>{};
  var _nextPlaceholder = 0xe000;
  var _isSafeToCopy = true;

  void reserveSource(String source) {
    for (final codeUnit in source.codeUnits) {
      // A new source character can only collide when a selection owner failed
      // to include dynamically displayed text in selectionSource. Do not copy
      // an ambiguous marker as if it were source text.
      if (emojiByPlaceholder.containsKey(codeUnit)) _isSafeToCopy = false;
      _sourceCodeUnits.add(codeUnit);
    }
  }

  int? placeholderFor(String emoji) {
    if (!_isSafeToCopy) return null;
    final existing = _placeholderByEmoji[emoji];
    if (existing != null) return existing;

    while (_nextPlaceholder <= 0xf8ff &&
        (_sourceCodeUnits.contains(_nextPlaceholder) ||
            emojiByPlaceholder.containsKey(_nextPlaceholder))) {
      _nextPlaceholder++;
    }
    // If the finite PUA pool is exhausted by valid input, keep this Emoji as
    // Unicode text rather than borrow Flutter's native U+FFFC placeholder.
    if (_nextPlaceholder > 0xf8ff) return null;

    final placeholder = _nextPlaceholder++;
    _placeholderByEmoji[emoji] = placeholder;
    emojiByPlaceholder[placeholder] = emoji;
    return placeholder;
  }

  String? restoreSelectedText(String text) {
    if (!_isSafeToCopy) return null;
    return String.fromCharCodes(
      text.runes.expand((codepoint) {
        final emoji = emojiByPlaceholder[codepoint];
        return emoji == null ? <int>[codepoint] : emoji.runes;
      }),
    );
  }
}

InlineSpan buildTwemojiTextSpan(
  BuildContext context,
  InlineSpan sourceSpan, {
  required TextStyle inheritedStyle,
  required TextScaler textScaler,
}) {
  final registry = _TwemojiSelectionScope.maybeOf(context);
  // Register plain and Emoji text before allocating any sibling placeholder.
  registry?.reserveSource(sourceSpan.toPlainText());
  if (sourceSpan is TextSpan &&
      !_mightContainTwemoji(
        sourceSpan.toPlainText(includePlaceholders: false),
      )) {
    return sourceSpan;
  }
  return _TwemojiSpanBuilder(
    textScaler,
    registry,
  ).build(sourceSpan, inheritedStyle);
}

class _TwemojiSelectionScope extends InheritedWidget {
  const _TwemojiSelectionScope({required this.registry, required super.child});

  final _TwemojiSelectionRegistry registry;

  static _TwemojiSelectionRegistry? maybeOf(BuildContext context) => context
      .dependOnInheritedWidgetOfExactType<_TwemojiSelectionScope>()
      ?.registry;

  @override
  bool updateShouldNotify(_TwemojiSelectionScope oldWidget) =>
      !identical(registry, oldWidget.registry);
}

/// A platform-adaptive selection area that restores inline image placeholders
/// to their original Unicode Emoji sequences when selected text is copied.
class GfSelectionArea extends StatefulWidget {
  const GfSelectionArea({
    super.key,
    required this.child,
    required this.selectionSource,
    this.focusNode,
    this.selectionControls,
    this.magnifierConfiguration,
    this.contextMenuBuilder,
  });

  final Widget child;

  /// Complete original text represented by [child], including decoded HTML
  /// entities and asynchronous rich-text labels. Reserve it before any Emoji
  /// placeholder is allocated, and update it when selectable content changes.
  final String selectionSource;
  final FocusNode? focusNode;
  final TextSelectionControls? selectionControls;
  final TextMagnifierConfiguration? magnifierConfiguration;
  final GfSelectionAreaContextMenuBuilder? contextMenuBuilder;

  @override
  State<GfSelectionArea> createState() => _GfSelectionAreaState();
}

typedef GfSelectionAreaContextMenuBuilder =
    Widget Function(
      BuildContext context,
      SelectableRegionState selection,
      List<ContextMenuButtonItem> buttonItems,
    );

class _GfSelectionAreaState extends State<GfSelectionArea> {
  late _TwemojiSelectionRegistry _registry = _TwemojiSelectionRegistry(
    widget.selectionSource,
  );
  String? _selectedText;
  var _selectionGeneration = 0;

  @override
  void didUpdateWidget(covariant GfSelectionArea oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.selectionSource != widget.selectionSource) {
      _registry = _TwemojiSelectionRegistry(widget.selectionSource);
      _selectedText = null;
      _selectionGeneration++;
    }
  }

  List<ContextMenuButtonItem> _buttonItems(SelectableRegionState selection) =>
      selection.contextMenuButtonItems
          .map((item) {
            if (item.type != ContextMenuButtonType.copy) return item;
            return ContextMenuButtonItem(
              type: ContextMenuButtonType.copy,
              onPressed: () {
                _copySelectedText();
                _finishCopy(selection);
              },
            );
          })
          .toList(growable: false);

  void _copySelectedText() {
    final selectedText = _selectedText;
    if (selectedText == null) return;
    final restoredText = _registry.restoreSelectedText(selectedText);
    if (restoredText == null) return;
    Clipboard.setData(ClipboardData(text: restoredText));
  }

  void _finishCopy(SelectableRegionState selection) {
    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
      case TargetPlatform.fuchsia:
        selection.clearSelection();
      case TargetPlatform.iOS:
        selection.hideToolbar(false);
      case TargetPlatform.linux:
      case TargetPlatform.macOS:
      case TargetPlatform.windows:
        selection.hideToolbar();
    }
  }

  @override
  Widget build(BuildContext context) => Actions(
    // SelectableRegion's CopySelectionTextIntent is overridable. Place this
    // action above SelectionArea so its keyboard shortcut finds it while the
    // selection region dispatches copy.
    actions: <Type, Action<Intent>>{
      CopySelectionTextIntent: CallbackAction<CopySelectionTextIntent>(
        onInvoke: (_) {
          _copySelectedText();
          return null;
        },
      ),
    },
    child: SelectionArea(
      key: ValueKey<int>(_selectionGeneration),
      focusNode: widget.focusNode,
      selectionControls: widget.selectionControls,
      magnifierConfiguration: widget.magnifierConfiguration,
      onSelectionChanged: (content) => _selectedText = content?.plainText,
      contextMenuBuilder: (context, selection) {
        final buttonItems = _buttonItems(selection);
        return widget.contextMenuBuilder?.call(
              context,
              selection,
              buttonItems,
            ) ??
            AdaptiveTextSelectionToolbar.buttonItems(
              anchors: selection.contextMenuAnchors,
              buttonItems: buttonItems,
            );
      },
      child: _TwemojiSelectionScope(registry: _registry, child: widget.child),
    ),
  );
}

/// Matches the longest complete sequence in the pinned Twemoji 17.0.3 map.
/// The returned strings always concatenate to [text] unchanged.
List<TwemojiSegment> parseTwemojiText(String text) {
  final codepoints = text.runes.toList(growable: false);
  final result = <TwemojiSegment>[];
  final plain = StringBuffer();
  var index = 0;

  void flushPlain() {
    if (plain.isEmpty) return;
    result.add(TwemojiSegment.text(plain.toString()));
    plain.clear();
  }

  while (index < codepoints.length) {
    var candidate = '';
    var hasJoiner = false;
    var bestEnd = -1;
    String? bestAsset;
    final limit = (index + twemojiMaximumCodepoints)
        .clamp(0, codepoints.length)
        .toInt();

    for (var end = index; end < limit; end++) {
      candidate += String.fromCharCode(codepoints[end]);
      hasJoiner = hasJoiner || codepoints[end] == 0x200d;
      final key = hasJoiner ? candidate : candidate.replaceAll('\uFE0F', '');
      final asset = twemojiAssetBySequence[key];
      if (asset != null) {
        bestEnd = end + 1;
        bestAsset = asset;
      }
    }

    // If this is an unsupported ZWJ or modifier sequence, leave the complete
    // cluster as text instead of painting only its first recognized emoji.
    if (bestAsset != null && bestEnd > index) {
      final joinedEnd = _consumeEmojiExtensions(codepoints, bestEnd);
      if (joinedEnd < codepoints.length && codepoints[joinedEnd] == 0x200d) {
        final unsupportedEnd = _consumeJoinedSequence(codepoints, index);
        for (var cursor = index; cursor < unsupportedEnd; cursor++) {
          plain.writeCharCode(codepoints[cursor]);
        }
        index = unsupportedEnd;
        continue;
      }
      if (joinedEnd > bestEnd) {
        final extended = String.fromCharCodes(
          codepoints.sublist(index, joinedEnd),
        );
        final hasJoinerInExtended = extended.contains('\u200D');
        final extendedKey = hasJoinerInExtended
            ? extended
            : extended.replaceAll('\uFE0F', '');
        final extendedAsset = twemojiAssetBySequence[extendedKey];
        if (extendedAsset == null) {
          for (var cursor = index; cursor < joinedEnd; cursor++) {
            plain.writeCharCode(codepoints[cursor]);
          }
          index = joinedEnd;
          continue;
        }
        bestEnd = joinedEnd;
        bestAsset = extendedAsset;
      }

      flushPlain();
      result.add(
        TwemojiSegment.emoji(
          String.fromCharCodes(codepoints.sublist(index, bestEnd)),
          bestAsset,
        ),
      );
      index = bestEnd;
      continue;
    }

    final joinedEnd = _consumeJoinedSequence(codepoints, index);
    if (joinedEnd > index + 1) {
      for (var cursor = index; cursor < joinedEnd; cursor++) {
        plain.writeCharCode(codepoints[cursor]);
      }
      index = joinedEnd;
      continue;
    }

    plain.writeCharCode(codepoints[index]);
    index++;
  }

  flushPlain();
  return result;
}

int _consumeEmojiExtensions(List<int> codepoints, int start) {
  var end = start;
  while (end < codepoints.length && _isEmojiExtension(codepoints[end])) {
    end++;
  }
  return end;
}

int _consumeJoinedSequence(List<int> codepoints, int start) {
  var end = _consumeEmojiExtensions(codepoints, start + 1);
  var hasJoiner = false;
  while (end < codepoints.length && codepoints[end] == 0x200d) {
    hasJoiner = true;
    end++;
    if (end < codepoints.length) end++;
    end = _consumeEmojiExtensions(codepoints, end);
  }
  return hasJoiner ? end : start + 1;
}

bool _isEmojiExtension(int codepoint) =>
    codepoint == 0xfe0e ||
    codepoint == 0xfe0f ||
    codepoint == 0x20e3 ||
    (codepoint >= 0x1f3fb && codepoint <= 0x1f3ff) ||
    (codepoint >= 0xe0020 && codepoint <= 0xe007f);

/// Inline text renderer shared by all mobile app surfaces.
///
/// Input fields deliberately remain Flutter's native text widgets so IME and
/// cursor behavior are untouched. Display text keeps its original Unicode in
/// the span tree; image spans expose the same sequence as semantic text.
class GfText extends Text {
  const GfText(
    super.data, {
    super.key,
    super.style,
    super.strutStyle,
    super.textAlign,
    super.textDirection,
    super.locale,
    super.softWrap,
    super.overflow,
    super.textScaler,
    super.maxLines,
    super.semanticsLabel,
    super.textWidthBasis,
    super.textHeightBehavior,
    super.selectionColor,
  });

  const GfText.rich(
    super.textSpan, {
    super.key,
    super.style,
    super.strutStyle,
    super.textAlign,
    super.textDirection,
    super.locale,
    super.softWrap,
    super.overflow,
    super.textScaler,
    super.maxLines,
    super.semanticsLabel,
    super.textWidthBasis,
    super.textHeightBehavior,
    super.selectionColor,
  }) : super.rich();

  @override
  Widget build(BuildContext context) {
    final defaultTextStyle = DefaultTextStyle.of(context);
    var effectiveStyle = style;
    if (effectiveStyle == null || effectiveStyle.inherit) {
      effectiveStyle = defaultTextStyle.style.merge(effectiveStyle);
    }
    if (MediaQuery.boldTextOf(context)) {
      effectiveStyle = effectiveStyle.merge(
        const TextStyle(fontWeight: FontWeight.bold),
      );
    }
    final scaler = textScaler ?? MediaQuery.textScalerOf(context);
    final sourceSpan = TextSpan(
      text: data,
      style: effectiveStyle,
      locale: locale,
      children: textSpan == null ? null : <InlineSpan>[textSpan!],
    );
    final rendered =
        buildTwemojiTextSpan(
              context,
              sourceSpan,
              inheritedStyle: effectiveStyle,
              textScaler: scaler,
            )
            as TextSpan;
    final lineHeightScaleFactor =
        MediaQuery.maybeLineHeightScaleFactorOverrideOf(context);
    final letterSpacing = MediaQuery.maybeLetterSpacingOverrideOf(context);
    final wordSpacing = MediaQuery.maybeWordSpacingOverrideOf(context);
    final overrideStyle = TextStyle(
      height: lineHeightScaleFactor,
      letterSpacing: letterSpacing,
      wordSpacing: wordSpacing,
    );
    final effectiveSpan = _applyTextStyleOverrides(rendered, overrideStyle);
    final effectiveStrutStyle = strutStyle?.merge(
      StrutStyle(height: lineHeightScaleFactor),
    );
    final selectionStyle = DefaultSelectionStyle.of(context);
    Widget result = RichText(
      text: effectiveSpan,
      textAlign: textAlign ?? defaultTextStyle.textAlign ?? TextAlign.start,
      textDirection: textDirection,
      locale: locale,
      softWrap: softWrap ?? defaultTextStyle.softWrap,
      overflow:
          overflow ?? effectiveStyle.overflow ?? defaultTextStyle.overflow,
      textScaler: scaler,
      maxLines: maxLines ?? defaultTextStyle.maxLines,
      strutStyle: effectiveStrutStyle,
      textWidthBasis: textWidthBasis ?? defaultTextStyle.textWidthBasis,
      textHeightBehavior:
          textHeightBehavior ??
          defaultTextStyle.textHeightBehavior ??
          DefaultTextHeightBehavior.maybeOf(context),
      selectionRegistrar: SelectionContainer.maybeOf(context),
      selectionColor:
          selectionColor ??
          selectionStyle.selectionColor ??
          DefaultSelectionStyle.defaultColor,
    );
    if (SelectionContainer.maybeOf(context) != null) {
      result = MouseRegion(
        cursor: selectionStyle.mouseCursor ?? SystemMouseCursors.text,
        child: result,
      );
    }
    if (semanticsLabel != null || semanticsIdentifier != null) {
      result = Semantics(
        textDirection: textDirection,
        label: semanticsLabel,
        identifier: semanticsIdentifier,
        child: ExcludeSemantics(
          excluding: semanticsLabel != null,
          child: result,
        ),
      );
    }
    return result;
  }
}

TextSpan _applyTextStyleOverrides(TextSpan span, TextStyle overrideStyle) {
  final children = span.children?.map((child) {
    if (child is TextSpan && child.runtimeType == TextSpan) {
      return _applyTextStyleOverrides(child, overrideStyle);
    }
    return child;
  }).toList();
  return TextSpan(
    text: span.text,
    children: children,
    style: span.style?.merge(overrideStyle) ?? overrideStyle,
    recognizer: span.recognizer,
    mouseCursor: span.mouseCursor,
    onEnter: span.onEnter,
    onExit: span.onExit,
    semanticsLabel: span.semanticsLabel,
    semanticsIdentifier: span.semanticsIdentifier,
    locale: span.locale,
    spellOut: span.spellOut,
  );
}

/// Selectable text maps a selected image placeholder back to its full Unicode
/// sequence when the platform Copy action is used.
class GfSelectableText extends StatelessWidget {
  const GfSelectableText(
    this.data, {
    super.key,
    this.focusNode,
    this.style,
    this.strutStyle,
    this.textAlign,
    this.textDirection,
    this.textScaler,
    this.showCursor = false,
    this.autofocus = false,
    this.minLines,
    this.maxLines,
    this.cursorWidth = 2,
    this.cursorHeight,
    this.cursorRadius,
    this.cursorColor,
    this.selectionColor,
    this.dragStartBehavior = DragStartBehavior.start,
    this.enableInteractiveSelection = true,
    this.selectionControls,
    this.onTap,
    this.scrollPhysics,
    this.scrollBehavior,
    this.semanticsLabel,
    this.textHeightBehavior,
    this.textWidthBasis,
    this.magnifierConfiguration,
    this.contextMenuBuilder,
  }) : textSpan = null;

  const GfSelectableText.rich(
    this.textSpan, {
    super.key,
    this.focusNode,
    this.style,
    this.strutStyle,
    this.textAlign,
    this.textDirection,
    this.textScaler,
    this.showCursor = false,
    this.autofocus = false,
    this.minLines,
    this.maxLines,
    this.cursorWidth = 2,
    this.cursorHeight,
    this.cursorRadius,
    this.cursorColor,
    this.selectionColor,
    this.dragStartBehavior = DragStartBehavior.start,
    this.enableInteractiveSelection = true,
    this.selectionControls,
    this.onTap,
    this.scrollPhysics,
    this.scrollBehavior,
    this.semanticsLabel,
    this.textHeightBehavior,
    this.textWidthBasis,
    this.magnifierConfiguration,
    this.contextMenuBuilder,
  }) : data = null;

  final String? data;
  final InlineSpan? textSpan;
  final FocusNode? focusNode;
  final TextStyle? style;
  final StrutStyle? strutStyle;
  final TextAlign? textAlign;
  final TextDirection? textDirection;
  final TextScaler? textScaler;
  final bool showCursor;
  final bool autofocus;
  final int? minLines;
  final int? maxLines;
  final double cursorWidth;
  final double? cursorHeight;
  final Radius? cursorRadius;
  final Color? cursorColor;
  final Color? selectionColor;
  final DragStartBehavior dragStartBehavior;
  final bool enableInteractiveSelection;
  final TextSelectionControls? selectionControls;
  final GestureTapCallback? onTap;
  final ScrollPhysics? scrollPhysics;
  final ScrollBehavior? scrollBehavior;
  final String? semanticsLabel;
  final TextHeightBehavior? textHeightBehavior;
  final TextWidthBasis? textWidthBasis;
  final TextMagnifierConfiguration? magnifierConfiguration;
  final EditableTextContextMenuBuilder? contextMenuBuilder;

  @override
  Widget build(BuildContext context) {
    final inheritedStyle = DefaultTextStyle.of(context).style;
    final baseStyle = inheritedStyle.merge(style);
    final scaler = textScaler ?? MediaQuery.textScalerOf(context);
    final InlineSpan inputSpan = textSpan ?? TextSpan(text: data ?? '');
    // SelectableText.rich is built from a TextSpan root. Gf's public rich API
    // accepts any InlineSpan, so wrap a root WidgetSpan instead of asserting
    // that the caller supplied a TextSpan. This keeps its one-unit source
    // offset and leaves nested WidgetSpan rendering unchanged.
    final TextSpan sourceSpan = inputSpan is TextSpan
        ? inputSpan
        : TextSpan(children: <InlineSpan>[inputSpan]);
    final containsEmoji = _mightContainTwemoji(
      sourceSpan.toPlainText(includePlaceholders: false),
    );

    if (containsEmoji && _TwemojiSelectionScope.maybeOf(context) != null) {
      Widget richText = GfText.rich(
        sourceSpan,
        style: style,
        strutStyle: strutStyle,
        textAlign: textAlign,
        textDirection: textDirection,
        textScaler: textScaler,
        maxLines: maxLines,
        semanticsLabel: semanticsLabel,
        textWidthBasis: textWidthBasis,
        textHeightBehavior: textHeightBehavior,
        selectionColor: selectionColor,
      );
      if (!enableInteractiveSelection) {
        richText = SelectionContainer.disabled(child: richText);
      }
      if (onTap != null) {
        richText = GestureDetector(
          behavior: HitTestBehavior.translucent,
          onTap: onTap,
          child: richText,
        );
      }
      return Focus(focusNode: focusNode, autofocus: autofocus, child: richText);
    }

    final builder = _TwemojiSpanBuilder(scaler);
    final rendered = builder.build(sourceSpan, baseStyle) as TextSpan;
    Widget buildContextMenu(BuildContext context, EditableTextState state) {
      List<ContextMenuButtonItem> withTwemojiCopy(
        List<ContextMenuButtonItem> items,
      ) => items
          .map((item) {
            if (item.type != ContextMenuButtonType.copy) return item;
            return item.copyWith(
              onPressed: () {
                _copySelectableText(state, sourceSpan, builder);
                state.hideToolbar();
              },
            );
          })
          .toList(growable: false);

      final customBuilder = contextMenuBuilder;
      if (customBuilder == null) {
        return AdaptiveTextSelectionToolbar.buttonItems(
          anchors: state.contextMenuAnchors,
          buttonItems: withTwemojiCopy(state.contextMenuButtonItems),
        );
      }

      final customMenu = customBuilder(context, state);
      if (customMenu is AdaptiveTextSelectionToolbar &&
          customMenu.buttonItems != null) {
        return AdaptiveTextSelectionToolbar.buttonItems(
          anchors: customMenu.anchors,
          buttonItems: withTwemojiCopy(customMenu.buttonItems!),
        );
      }
      if (customMenu is cupertino.CupertinoAdaptiveTextSelectionToolbar &&
          customMenu.buttonItems != null) {
        return cupertino.CupertinoAdaptiveTextSelectionToolbar.buttonItems(
          anchors: customMenu.anchors,
          buttonItems: withTwemojiCopy(customMenu.buttonItems!),
        );
      }
      return customMenu;
    }

    return Actions(
      actions: <Type, Action<Intent>>{
        CopySelectionTextIntent: _TwemojiCopySelectionAction(
          sourceSpan: sourceSpan,
          builder: builder,
        ),
      },
      child: SelectableText.rich(
        rendered,
        focusNode: focusNode,
        style: style,
        strutStyle: strutStyle,
        textAlign: textAlign,
        textDirection: textDirection,
        textScaler: textScaler,
        showCursor: showCursor,
        autofocus: autofocus,
        minLines: minLines,
        maxLines: maxLines,
        cursorWidth: cursorWidth,
        cursorHeight: cursorHeight,
        cursorRadius: cursorRadius,
        cursorColor: cursorColor,
        selectionColor: selectionColor,
        dragStartBehavior: dragStartBehavior,
        enableInteractiveSelection: enableInteractiveSelection,
        selectionControls: selectionControls,
        onTap: onTap,
        scrollPhysics: scrollPhysics,
        scrollBehavior: scrollBehavior,
        semanticsLabel: semanticsLabel,
        textHeightBehavior: textHeightBehavior,
        textWidthBasis: textWidthBasis,
        magnifierConfiguration: magnifierConfiguration,
        contextMenuBuilder: buildContextMenu,
      ),
    );
  }
}

class _TwemojiSpanBuilder {
  _TwemojiSpanBuilder(this.textScaler, [this.selectionRegistry]);

  final TextScaler textScaler;
  final _TwemojiSelectionRegistry? selectionRegistry;
  final List<int> _displayToSource = <int>[0];
  var _sourceOffset = 0;

  InlineSpan build(InlineSpan span, TextStyle inheritedStyle) {
    if (span is TextSpan &&
        !_mightContainTwemoji(span.toPlainText(includePlaceholders: false))) {
      _recordIdentityOffsets(span);
      return span;
    }
    return _transform(span, inheritedStyle);
  }

  void _recordIdentityOffsets(InlineSpan span) {
    if (span is WidgetSpan) {
      _sourceOffset++;
      _displayToSource.add(_sourceOffset);
      return;
    }
    if (span is TextSpan) {
      _recordTextOffsets(span.text ?? '');
      for (final child in span.children ?? const <InlineSpan>[]) {
        _recordIdentityOffsets(child);
      }
      return;
    }
    _recordTextOffsets(span.toPlainText());
  }

  void _recordTextOffsets(String text) {
    for (var i = 0; i < text.length; i++) {
      _sourceOffset++;
      _displayToSource.add(_sourceOffset);
    }
  }

  TextSelection sourceRangeForDisplayRange(int start, int end) {
    final safeStart = start.clamp(0, _displayToSource.length - 1).toInt();
    final safeEnd = end.clamp(0, _displayToSource.length - 1).toInt();
    return TextSelection(
      baseOffset: _displayToSource[safeStart],
      extentOffset: _displayToSource[safeEnd],
    );
  }

  InlineSpan _transform(InlineSpan span, TextStyle inheritedStyle) {
    if (span is WidgetSpan) {
      _sourceOffset++;
      _displayToSource.add(_sourceOffset);
      return span;
    }
    if (span is! TextSpan) return span;

    final style = inheritedStyle.merge(span.style);
    final transformedChildren = <InlineSpan>[];
    if (span.text case final text?) {
      final segments = _mightContainTwemoji(text)
          ? parseTwemojiText(text)
          : text.isEmpty
          ? const <TwemojiSegment>[]
          : <TwemojiSegment>[TwemojiSegment.text(text)];
      for (final segment in segments) {
        if (!segment.isEmoji) {
          final value = segment.value;
          transformedChildren.add(
            TextSpan(
              text: value,
              recognizer: span.recognizer,
              mouseCursor: span.mouseCursor,
              onEnter: span.onEnter,
              onExit: span.onExit,
            ),
          );
          _sourceOffset += value.length;
          for (var i = 0; i < value.length; i++) {
            _displayToSource.add(_sourceOffset - value.length + i + 1);
          }
          continue;
        }

        final value = segment.value;
        final sourceStart = _sourceOffset;
        _sourceOffset += value.length;
        _displayToSource.add(_sourceOffset);
        final fontSize = style.fontSize ?? 14;
        final dimension = textScaler
            .scale(fontSize)
            .clamp(1.0, 512.0)
            .toDouble();
        final tapRecognizer = span.recognizer;
        final VoidCallback? onTap = tapRecognizer is TapGestureRecognizer
            ? tapRecognizer.onTap
            : null;
        Widget emoji = Semantics(
          label: value,
          image: true,
          container: true,
          excludeSemantics: true,
          onTap: onTap,
          child: Image.asset(
            'assets/twemoji-public/twemoji/72x72/${segment.asset}',
            package: 'ui_kit',
            width: dimension,
            height: dimension,
            cacheWidth: 72,
            cacheHeight: 72,
            filterQuality: FilterQuality.medium,
            errorBuilder: (context, error, stack) =>
                Text(value, style: TextStyle(fontSize: dimension, height: 1)),
          ),
        );
        if (onTap != null) {
          emoji = GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: onTap,
            child: emoji,
          );
        }
        final placeholder = selectionRegistry?.placeholderFor(value);
        if (selectionRegistry != null && placeholder == null) {
          transformedChildren.add(
            TextSpan(
              text: value,
              style: style,
              recognizer: span.recognizer,
              mouseCursor: span.mouseCursor,
              onEnter: span.onEnter,
              onExit: span.onExit,
            ),
          );
          continue;
        }
        transformedChildren.add(
          _TwemojiWidgetSpan(
            placeholder: placeholder ?? PlaceholderSpan.placeholderCodeUnit,
            child: emoji,
          ),
        );
        assert(_sourceOffset > sourceStart);
      }
    }

    for (final child in span.children ?? const <InlineSpan>[]) {
      transformedChildren.add(_transform(child, style));
    }

    return TextSpan(
      style: span.style,
      children: transformedChildren,
      recognizer: span.recognizer,
      mouseCursor: span.mouseCursor,
      onEnter: span.onEnter,
      onExit: span.onExit,
      semanticsLabel: span.semanticsLabel,
      locale: span.locale,
      spellOut: span.spellOut,
    );
  }
}

class _TwemojiWidgetSpan extends WidgetSpan {
  const _TwemojiWidgetSpan({required this.placeholder, required super.child})
    : super(alignment: PlaceholderAlignment.middle);

  final int placeholder;

  @override
  void computeToPlainText(
    StringBuffer buffer, {
    bool includeSemanticsLabels = true,
    bool includePlaceholders = true,
  }) {
    if (includePlaceholders) buffer.writeCharCode(placeholder);
  }
}

bool _copySelectableText(
  EditableTextState state,
  InlineSpan sourceSpan,
  _TwemojiSpanBuilder builder,
) {
  final selection = state.textEditingValue.selection;
  if (!selection.isValid || selection.isCollapsed) return false;
  final sourceRange = builder.sourceRangeForDisplayRange(
    selection.start,
    selection.end,
  );
  final sourceText = sourceSpan.toPlainText(includePlaceholders: true);
  Clipboard.setData(
    ClipboardData(
      text: sourceText.substring(
        sourceRange.start.clamp(0, sourceText.length).toInt(),
        sourceRange.end.clamp(0, sourceText.length).toInt(),
      ),
    ),
  );
  return true;
}

/// Overrides EditableText's overridable copy action so desktop keyboard copy
/// uses source offsets just like the long-press toolbar. Native WidgetSpan
/// placeholders remain one-to-one in the offset map and are never decoded as
/// Emoji.
class _TwemojiCopySelectionAction
    extends ContextAction<CopySelectionTextIntent> {
  _TwemojiCopySelectionAction({
    required this.sourceSpan,
    required this.builder,
  });

  final InlineSpan sourceSpan;
  final _TwemojiSpanBuilder builder;

  @override
  void invoke(CopySelectionTextIntent intent, [BuildContext? context]) {
    final state = context?.findAncestorStateOfType<EditableTextState>();
    if (state == null || intent.collapseSelection) return;
    if (_copySelectableText(state, sourceSpan, builder)) {
      state.clipboardStatus.update();
    }
  }
}

/// Returns selected source text for tests and selection adapters.
@visibleForTesting
String twemojiCopyTextForDisplayRange(
  InlineSpan sourceSpan,
  int displayStart,
  int displayEnd,
) {
  final builder = _TwemojiSpanBuilder(TextScaler.noScaling);
  builder.build(sourceSpan, const TextStyle());
  final range = builder.sourceRangeForDisplayRange(displayStart, displayEnd);
  final source = sourceSpan.toPlainText(includePlaceholders: true);
  return source.substring(
    range.start.clamp(0, source.length).toInt(),
    range.end.clamp(0, source.length).toInt(),
  );
}

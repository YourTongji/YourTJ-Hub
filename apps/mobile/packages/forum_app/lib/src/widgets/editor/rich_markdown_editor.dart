import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../markdown_view.dart';

/// Shared Quill editing surface. The caller owns the controller and its
/// Markdown conversion/draft lifecycle; this widget only presents the editor.
class RichMarkdownEditor extends StatelessWidget {
  const RichMarkdownEditor({
    super.key,
    required this.controller,
    required this.focusNode,
    required this.placeholder,
    this.editorBuilder,
    this.onHeading,
    this.onHeadingLongPress,
    this.onInsertLink,
    this.showToolbar = true,
    this.enabled = true,
  });

  final QuillController controller;
  final FocusNode focusNode;
  final String placeholder;
  final Widget Function(BuildContext context)? editorBuilder;
  final VoidCallback? onHeading;
  final VoidCallback? onHeadingLongPress;
  final VoidCallback? onInsertLink;
  final bool showToolbar;
  final bool enabled;

  @override
  Widget build(BuildContext context) => Column(
    mainAxisSize: MainAxisSize.min,
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: <Widget>[
      if (showToolbar)
        RichMarkdownToolbar(
          controller: controller,
          onHeading: onHeading,
          onHeadingLongPress: onHeadingLongPress,
          onInsertLink: onInsertLink,
          enabled: enabled,
        ),
      ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 220),
        child:
            editorBuilder?.call(context) ??
            _QuillBody(
              controller: controller,
              focusNode: focusNode,
              placeholder: placeholder,
              enabled: enabled,
            ),
      ),
    ],
  );
}

/// The same horizontally scrollable toolbar used by the composer and reviews.
class RichMarkdownToolbar extends StatelessWidget {
  const RichMarkdownToolbar({
    super.key,
    required this.controller,
    this.onHeading,
    this.onHeadingLongPress,
    this.onInsertLink,
    this.enabled = true,
  });

  final QuillController controller;
  final VoidCallback? onHeading;
  final VoidCallback? onHeadingLongPress;
  final VoidCallback? onInsertLink;
  final bool enabled;

  void _toggle(Attribute attribute) {
    final current = controller.getSelectionStyle().attributes[attribute.key];
    controller.formatSelection(
      current?.value == attribute.value
          ? Attribute.clone(attribute, null)
          : attribute,
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    return ListenableBuilder(
      listenable: controller,
      builder: (context, _) {
        final attributes = controller.getSelectionStyle().attributes;
        bool selected(Attribute attribute) =>
            attributes[attribute.key]?.value == attribute.value;
        return ColoredBox(
          key: const Key('rich-markdown-toolbar'),
          color: colors.base200.withValues(alpha: 0.55),
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
            child: Row(
              children: <Widget>[
                _ToolButton(
                  symbol: 'undo-2',
                  tooltip: l10n.publishUndo,
                  onPressed: enabled && controller.hasUndo
                      ? controller.undo
                      : null,
                ),
                _ToolButton(
                  symbol: 'redo-2',
                  tooltip: l10n.publishRedo,
                  onPressed: enabled && controller.hasRedo
                      ? controller.redo
                      : null,
                ),
                _ToolButton(
                  symbol: 'heading',
                  tooltip: l10n.publishHeading,
                  selected: attributes[Attribute.header.key]?.value != null,
                  onPressed: enabled
                      ? (onHeading ?? () => _toggle(Attribute.h2))
                      : null,
                  onLongPress: enabled ? onHeadingLongPress : null,
                ),
                _ToolButton(
                  symbol: 'link',
                  tooltip: l10n.publishToolLink,
                  onPressed: enabled ? onInsertLink : null,
                ),
                _ToolButton(
                  symbol: 'bold',
                  tooltip: l10n.publishToolBold,
                  selected: selected(Attribute.bold),
                  onPressed: enabled ? () => _toggle(Attribute.bold) : null,
                ),
                _ToolButton(
                  symbol: 'italic',
                  tooltip: l10n.publishToolItalic,
                  selected: selected(Attribute.italic),
                  onPressed: enabled ? () => _toggle(Attribute.italic) : null,
                ),
                _ToolButton(
                  symbol: 'strikethrough',
                  tooltip: l10n.publishToolStrike,
                  selected: selected(Attribute.strikeThrough),
                  onPressed: enabled
                      ? () => _toggle(Attribute.strikeThrough)
                      : null,
                ),
                _ToolButton(
                  symbol: 'quote',
                  tooltip: l10n.publishToolQuote,
                  selected: selected(Attribute.blockQuote),
                  onPressed: enabled
                      ? () => _toggle(Attribute.blockQuote)
                      : null,
                ),
                _ToolButton(
                  symbol: 'code',
                  tooltip: l10n.publishToolCode,
                  selected: selected(Attribute.inlineCode),
                  onPressed: enabled
                      ? () => _toggle(Attribute.inlineCode)
                      : null,
                ),
                _ToolButton(
                  symbol: 'unordered-list',
                  tooltip: l10n.publishToolBulletList,
                  selected: selected(Attribute.ul),
                  onPressed: enabled ? () => _toggle(Attribute.ul) : null,
                ),
                _ToolButton(
                  symbol: 'list-ordered',
                  tooltip: l10n.publishToolOrderedList,
                  selected: selected(Attribute.ol),
                  onPressed: enabled ? () => _toggle(Attribute.ol) : null,
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _QuillBody extends StatelessWidget {
  const _QuillBody({
    required this.controller,
    required this.focusNode,
    required this.placeholder,
    required this.enabled,
  });

  final QuillController controller;
  final FocusNode focusNode;
  final String placeholder;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final defaults = DefaultStyles.getInstance(context);
    return AbsorbPointer(
      absorbing: !enabled,
      child: QuillEditor.basic(
        controller: controller,
        focusNode: focusNode,
        config: QuillEditorConfig(
          scrollable: false,
          padding: const EdgeInsets.symmetric(vertical: 8),
          placeholder: placeholder,
          customStyles: DefaultStyles(
            paragraph: defaults.paragraph!.copyWith(
              style: readingBodyStyle(context),
              verticalSpacing: const VerticalSpacing(4, 4),
            ),
            placeHolder: defaults.placeHolder!.copyWith(
              style: readingBodyStyle(
                context,
              ).copyWith(color: GfTheme.colorsOf(context).iconMuted),
            ),
          ),
          embedBuilders: const <EmbedBuilder>[
            _MarkdownImageBuilder(),
            _MarkdownTableBuilder(),
          ],
        ),
      ),
    );
  }
}

class _MarkdownImageBuilder extends EmbedBuilder {
  const _MarkdownImageBuilder();

  @override
  String get key => BlockEmbed.imageType;

  @override
  Widget build(BuildContext context, EmbedContext embedContext) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 4),
    child: GfNetworkImage(
      resolveApiAssetUrl(embedContext.node.value.data.toString()),
      fit: BoxFit.contain,
      cacheWidth:
          (MediaQuery.sizeOf(context).width *
                  MediaQuery.devicePixelRatioOf(context))
              .round(),
      errorBuilder: (_, _, _) => const GfSymbol('image-off'),
    ),
  );
}

class _MarkdownTableBuilder extends EmbedBuilder {
  const _MarkdownTableBuilder();

  // This embed type is owned by MarkdownConverter's markdown_quill table
  // support. GfMarkdownView parses and renders the preserved Markdown source.
  @override
  String get key => 'x-embed-table';

  @override
  Widget build(BuildContext context, EmbedContext embedContext) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 8),
    child: IgnorePointer(
      child: GfMarkdownView(
        data: '${embedContext.node.value.data}\n',
        compact: true,
      ),
    ),
  );
}

class _ToolButton extends StatelessWidget {
  const _ToolButton({
    required this.symbol,
    required this.tooltip,
    required this.onPressed,
    this.onLongPress,
    this.selected,
  });

  final String symbol;
  final String tooltip;
  final VoidCallback? onPressed;
  final VoidCallback? onLongPress;
  final bool? selected;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return MergeSemantics(
      child: Semantics(
        toggled: selected,
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: selected == true
                ? colors.primary.withValues(alpha: 0.12)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).field),
          ),
          child: GfIconButton(
            symbol: symbol,
            tooltip: tooltip,
            size: 44,
            iconSize: 23,
            color: onPressed == null
                ? colors.iconMuted.withValues(alpha: 0.4)
                : selected == true
                ? colors.primary
                : colors.iconMuted,
            onPressed: onPressed,
            onLongPress: onLongPress,
          ),
        ),
      ),
    );
  }
}

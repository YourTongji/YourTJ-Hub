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
    this.onInsertImage,
    this.insertingImage = false,
    this.showToolbar = true,
    this.enabled = true,
    this.fill = false,
    this.profile,
  });

  final QuillController controller;
  final FocusNode focusNode;
  final String placeholder;
  final Widget Function(BuildContext context)? editorBuilder;
  final VoidCallback? onHeading;
  final VoidCallback? onHeadingLongPress;
  final VoidCallback? onInsertLink;

  /// Optional image action. Only shown when the caller owns an upload pipeline;
  /// [insertingImage] keeps the button inert while that upload is in flight.
  final VoidCallback? onInsertImage;
  final bool insertingImage;
  final bool showToolbar;
  final bool enabled;

  /// Editor-first layout for surfaces that own a bounded height (the course
  /// review sheet): the toolbar stays pinned and the body scrolls inside the
  /// space that is left. Only meaningful when the caller gives this widget a
  /// bound, for example inside an [Expanded].
  final bool fill;

  /// Block typography for the body. Defaults to the reading profile
  /// ([GfRichContentTypography.of]) so the editor matches the renderer its
  /// Markdown converts to; course reviews pass the compact profile.
  final GfRichContentTypography? profile;

  @override
  Widget build(BuildContext context) {
    final GfRichContentTypography profile =
        this.profile ?? GfRichContentTypography.of(context);
    final Widget? customBody = editorBuilder?.call(context);
    return Column(
      mainAxisSize: fill ? MainAxisSize.max : MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        if (showToolbar)
          RichMarkdownToolbar(
            controller: controller,
            onHeading: onHeading,
            onHeadingLongPress: onHeadingLongPress,
            onInsertLink: onInsertLink,
            onInsertImage: onInsertImage,
            insertingImage: insertingImage,
            enabled: enabled,
          ),
        if (fill)
          Expanded(
            child:
                customBody ??
                _QuillBody(
                  controller: controller,
                  focusNode: focusNode,
                  placeholder: placeholder,
                  enabled: enabled,
                  profile: profile,
                  fill: true,
                ),
          )
        else
          ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 220),
            child:
                customBody ??
                _QuillBody(
                  controller: controller,
                  focusNode: focusNode,
                  placeholder: placeholder,
                  enabled: enabled,
                  profile: profile,
                  fill: false,
                ),
          ),
      ],
    );
  }
}

/// The same horizontally scrollable toolbar used by the composer and reviews.
class RichMarkdownToolbar extends StatelessWidget {
  const RichMarkdownToolbar({
    super.key,
    required this.controller,
    this.onHeading,
    this.onHeadingLongPress,
    this.onInsertLink,
    this.onInsertImage,
    this.insertingImage = false,
    this.enabled = true,
    this.tinted = true,
  });

  final QuillController controller;
  final VoidCallback? onHeading;
  final VoidCallback? onHeadingLongPress;
  final VoidCallback? onInsertLink;
  final VoidCallback? onInsertImage;
  final bool insertingImage;
  final bool enabled;

  /// Paints the toolbar's own band; off when a host row already frames it.
  final bool tinted;

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
          color: tinted
              ? colors.base200.withValues(alpha: 0.55)
              : Colors.transparent,
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
                if (onInsertImage != null)
                  _ToolButton(
                    symbol: 'image',
                    tooltip: l10n.publishToolImage,
                    onPressed: enabled && !insertingImage
                        ? onInsertImage
                        : null,
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

class _QuillBody extends StatefulWidget {
  const _QuillBody({
    required this.controller,
    required this.focusNode,
    required this.placeholder,
    required this.enabled,
    required this.profile,
    required this.fill,
  });

  final QuillController controller;
  final FocusNode focusNode;
  final String placeholder;
  final bool enabled;
  final GfRichContentTypography profile;
  final bool fill;

  @override
  State<_QuillBody> createState() => _QuillBodyState();
}

class _QuillBodyState extends State<_QuillBody> {
  /// Owned only in [RichMarkdownEditor.fill] mode. The widget must not let
  /// `QuillEditor.basic` create one per build: a fresh controller has no
  /// clients, so every keystroke would reset the body's scroll offset.
  ScrollController? _scrollController;

  @override
  void initState() {
    super.initState();
    if (widget.fill) _scrollController = ScrollController();
  }

  @override
  void didUpdateWidget(_QuillBody oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.fill && _scrollController == null) {
      _scrollController = ScrollController();
    }
  }

  @override
  void dispose() {
    _scrollController?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AbsorbPointer(
      absorbing: !widget.enabled,
      child: QuillEditor.basic(
        controller: widget.controller,
        focusNode: widget.focusNode,
        scrollController: _scrollController,
        config: QuillEditorConfig(
          // In fill mode the editor owns an internal scroll view so the sheet's
          // pinned toolbar and action row never move while typing.
          scrollable: widget.fill,
          minHeight: widget.fill ? 0 : null,
          scrollBottomInset: widget.fill ? 12 : 0,
          padding: const EdgeInsets.symmetric(vertical: 8),
          placeholder: widget.placeholder,
          customStyles: _editorDefaultStyles(context, widget.profile),
          embedBuilders: const <EmbedBuilder>[
            _MarkdownImageBuilder(),
            _MarkdownTableBuilder(),
          ],
        ),
      ),
    );
  }
}

/// Maps the shared rich-content profile onto Quill's block styles so the
/// editor and the reader render the same Markdown identically (h1..h6, quote,
/// code, inline code, lists, links). Sizes come from
/// [GfRichContentTypography] only — flutter_quill's own defaults (h1 34 / h2 30
/// / h3 24) must never leak into a template.
DefaultStyles _editorDefaultStyles(
  BuildContext context,
  GfRichContentTypography profile,
) {
  final DefaultStyles defaults = DefaultStyles.getInstance(context);
  final GfColors colors = GfTheme.colorsOf(context);

  DefaultTextBlockStyle block(DefaultTextBlockStyle base, TextStyle style) =>
      base.copyWith(style: style);

  return DefaultStyles(
    paragraph: defaults.paragraph!.copyWith(
      style: profile.body,
      verticalSpacing: const VerticalSpacing(4, 4),
    ),
    placeHolder: defaults.placeHolder!.copyWith(
      style: profile.body.copyWith(color: colors.iconMuted),
    ),
    h1: block(defaults.h1!, profile.h1),
    h2: block(defaults.h2!, profile.h2),
    h3: block(defaults.h3!, profile.h3),
    h4: block(defaults.h4!, profile.h4),
    h5: block(defaults.h5!, profile.h5),
    h6: block(defaults.h6!, profile.h6),
    quote: block(defaults.quote!, profile.quote),
    code: block(
      defaults.code!,
      // The profile's code style is deliberately colorless; this editor has no
      // syntax highlighting, so it supplies the reading text color itself.
      profile.code.copyWith(color: colors.baseContent),
    ),
    inlineCode: InlineCodeStyle(
      style: profile.inlineCode,
      backgroundColor: defaults.inlineCode!.backgroundColor,
      radius: defaults.inlineCode!.radius,
    ),
    lists: defaults.lists!.copyWith(style: profile.body),
    // Same link treatment as the reader (GfMarkdownView's LinkConfig).
    link: TextStyle(
      color: colors.primary,
      decoration: TextDecoration.underline,
    ),
  );
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
        child: AnimatedContainer(
          duration: GfMotion.duration(context, GfMotion.selection),
          decoration: BoxDecoration(
            color: selected == true
                ? colors.primary.withValues(alpha: 0.12)
                : colors.primary.withValues(alpha: 0),
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

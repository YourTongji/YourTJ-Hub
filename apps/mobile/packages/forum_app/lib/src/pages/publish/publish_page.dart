import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../images/image_upload.dart';
import '../../server_messages.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/status_views.dart';

/// Global topic composer aligned with the Web publish workspace.
///
/// The page keeps Markdown as its storage contract while using Quill for rich
/// editing. Narrow layouts switch between editing and a live prose preview;
/// wider layouts show both at once. The full `/publish` page payload owns
/// categories and edit prefill so draft entry points remain consistent.
class PublishPage extends ConsumerStatefulWidget {
  const PublishPage({
    super.key,
    this.topicId,
    this.editTitle,
    this.editContent,
    this.editCategoryIds,
    @visibleForTesting this.markdownConverter,
  });

  final int? topicId;

  /// Compatibility fallbacks for callers that already have edit data. The
  /// server page payload remains authoritative when it contains values.
  final String? editTitle;
  final String? editContent;
  final List<int>? editCategoryIds;

  @visibleForTesting
  final MarkdownConverter? markdownConverter;

  @override
  ConsumerState<PublishPage> createState() => _PublishPageState();
}

enum _ComposeMode { edit, preview }

class _PublishPageState extends ConsumerState<PublishPage> {
  static const int _maxCategories = 3;
  static const double _wideWorkspaceBreakpoint = 760;
  static const Duration _previewDebounceDuration = Duration(milliseconds: 200);

  final TextEditingController _title = TextEditingController();
  final List<int> _categoryIds = <int>[];
  final List<PublishCategoryPayload> _categories = <PublishCategoryPayload>[];
  late final MarkdownConverter _converter;

  late QuillController _quill;
  late StreamSubscription<DocChange> _documentChanges;
  late int _currentTopicId;

  _ComposeMode _mode = _ComposeMode.edit;
  bool _loading = true;
  bool _submitting = false;
  bool _uploading = false;
  String _loadError = '';
  String _error = '';
  String _message = '';
  String _previewMarkdown = '';
  Timer? _previewDebounce;

  @override
  void initState() {
    super.initState();
    _converter = widget.markdownConverter ?? MarkdownConverter();
    _currentTopicId = widget.topicId ?? 0;
    _title.text = widget.editTitle ?? '';
    _categoryIds.addAll(widget.editCategoryIds ?? const <int>[]);
    _quill = _createController(widget.editContent ?? '');
    _previewMarkdown = _markdownFromEditor();
    _loadEditorData();
  }

  QuillController _createController(String markdown) {
    final QuillController controller = QuillController(
      document: _converter.mdToDocument(markdown),
      selection: const TextSelection.collapsed(offset: 0),
    );
    _documentChanges = controller.document.changes.listen(
      (DocChange _) => _handleEditorChanged(),
    );
    return controller;
  }

  void _replaceEditorDocument(String markdown) {
    _previewDebounce?.cancel();
    _previewDebounce = null;
    unawaited(_documentChanges.cancel());
    _quill.dispose();
    _quill = _createController(markdown);
    _previewMarkdown = _markdownFromEditor();
  }

  String _markdownFromEditor() {
    return _converter.documentToMarkdown(_quill.document).trim();
  }

  void _handleEditorChanged() {
    if (!mounted) return;
    final bool previewVisible =
        _mode == _ComposeMode.preview ||
        MediaQuery.sizeOf(context).width >= _wideWorkspaceBreakpoint;
    _previewDebounce?.cancel();
    if (!previewVisible) {
      _previewDebounce = null;
      return;
    }
    _previewDebounce = Timer(_previewDebounceDuration, _refreshPreview);
  }

  void _refreshPreview() {
    _previewDebounce = null;
    if (!mounted) return;
    final String markdown = _markdownFromEditor();
    if (markdown == _previewMarkdown) return;
    setState(() => _previewMarkdown = markdown);
  }

  void _selectMode(_ComposeMode mode) {
    if (mode == _mode) return;
    _previewDebounce?.cancel();
    _previewDebounce = null;
    final String preview = mode == _ComposeMode.preview
        ? _markdownFromEditor()
        : _previewMarkdown;
    setState(() {
      _mode = mode;
      _previewMarkdown = preview;
    });
  }

  String get _payloadPath =>
      _currentTopicId > 0 ? '/publish?id=$_currentTopicId' : '/publish';

  Future<void> _loadEditorData() async {
    if (mounted) {
      setState(() {
        _loading = true;
        _loadError = '';
      });
    }

    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch(_payloadPath);
      final PublishPageProps? props = parsePageProps<PublishPageProps>(payload);
      if (props == null) {
        throw const FormatException('publish props');
      }
      if (!mounted) return;

      final String payloadTitle = props.topic.title.trim().isNotEmpty
          ? props.topic.title
          : (widget.editTitle ?? '');
      final String payloadContent = props.topic.content.trim().isNotEmpty
          ? props.topic.content
          : (widget.editContent ?? '');
      final List<int> payloadCategoryIds = props.topic.categoryIds.isNotEmpty
          ? props.topic.categoryIds
          : (widget.editCategoryIds ?? const <int>[]);

      _replaceEditorDocument(payloadContent);
      setState(() {
        _currentTopicId = props.topicId > 0 ? props.topicId : _currentTopicId;
        _categories
          ..clear()
          ..addAll(props.categories);
        _categoryIds
          ..clear()
          ..addAll(payloadCategoryIds.take(_maxCategories));
        _title.text = payloadTitle;
        _loading = false;
      });
    } catch (error) {
      if (!mounted) return;
      final AppLocalizations l10n = AppLocalizations.of(context);
      setState(() {
        _loading = false;
        _loadError = resolveErrorMessage(l10n, error);
      });
    }
  }

  @override
  void dispose() {
    _previewDebounce?.cancel();
    _title.dispose();
    unawaited(_documentChanges.cancel());
    _quill.dispose();
    super.dispose();
  }

  void _goBack() {
    if (context.canPop()) {
      context.pop();
    } else {
      context.go('/');
    }
  }

  void _toggleCategory(PublishCategoryPayload category, bool selected) {
    setState(() {
      _error = '';
      _message = '';
      if (!selected) {
        _categoryIds.remove(category.id);
        return;
      }
      if (_categoryIds.length < _maxCategories &&
          !_categoryIds.contains(category.id)) {
        _categoryIds.add(category.id);
      }
    });
  }

  Future<void> _pickAndInsertImage() async {
    if (_uploading) return;
    setState(() => _uploading = true);
    try {
      final String? url = await pickAndUploadImage(ref: ref);
      if (url == null || !mounted) return;
      final int selectionOffset = _quill.selection.baseOffset;
      final int insertAt = selectionOffset < 0 ? 0 : selectionOffset;
      _quill.document.insert(insertAt, BlockEmbed.image(url));
      _quill.updateSelection(
        TextSelection.collapsed(offset: insertAt + 1),
        ChangeSource.local,
      );
    } catch (error) {
      if (mounted) {
        final AppLocalizations l10n = AppLocalizations.of(context);
        showGfToast(context, l10n.publishImageFailed('$error'), error: true);
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _submit({required int topicStatus}) async {
    if (_submitting || _loading) return;

    final String title = _title.text.trim();
    final String content = _markdownFromEditor();
    final AppLocalizations l10n = AppLocalizations.of(context);

    String validationError = '';
    if (title.isEmpty) {
      validationError = l10n.publishTitleRequired;
    } else if (_categoryIds.isEmpty) {
      validationError = l10n.publishCategoryRequired;
    } else if (content.isEmpty) {
      validationError = l10n.publishContentRequired;
    }
    if (validationError.isNotEmpty) {
      setState(() {
        _error = validationError;
        _message = '';
      });
      return;
    }

    setState(() {
      _submitting = true;
      _error = '';
      _message = '';
    });
    try {
      final int id = await ref
          .read(topicRepositoryProvider)
          .writeTopic(
            topicId: _currentTopicId,
            title: title,
            content: content,
            categoryIds: List<int>.of(_categoryIds),
            topicStatus: topicStatus,
          );
      if (!mounted) return;

      final int resolvedId = id > 0 ? id : _currentTopicId;
      if (resolvedId > 0) _currentTopicId = resolvedId;
      if (topicStatus == 1 && resolvedId > 0) {
        context.pushReplacement('/p/$resolvedId');
        return;
      }
      setState(() {
        _message = topicStatus == 1
            ? l10n.publishSuccess
            : l10n.publishSavedDraft;
      });
    } on ApiException catch (error) {
      if (mounted) setState(() => _error = error.messageKey);
    } catch (error) {
      if (mounted) setState(() => _error = l10n.publishFailed('$error'));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: GfAppBar(
        leading: GfIconButton(
          icon: Icons.arrow_back_rounded,
          tooltip: l10n.commonBack,
          size: 44,
          onPressed: _goBack,
        ),
        title: Text(
          _currentTopicId == 0 ? l10n.publishTitle : l10n.publishEditTitle,
        ),
        actions: <Widget>[
          GfButton(
            key: const Key('publish-appbar-submit'),
            label: l10n.publishPublish,
            variant: GfButtonVariant.primary,
            size: GfButtonSize.small,
            loading: _submitting,
            onPressed: _loading || _uploading
                ? null
                : () => _submit(topicStatus: 1),
          ),
        ],
      ),
      body: _buildBody(l10n),
    );
  }

  Widget _buildBody(AppLocalizations l10n) {
    if (_loading) return const _PublishWorkspaceSkeleton();
    if (_loadError.isNotEmpty) {
      return GfErrorRetry(message: _loadError, onRetry: _loadEditorData);
    }

    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final bool wide = constraints.maxWidth >= _wideWorkspaceBreakpoint;
        final EdgeInsets pagePadding = EdgeInsets.symmetric(
          horizontal: wide ? 24 : 16,
          vertical: 16,
        );

        return SingleChildScrollView(
          padding: pagePadding.copyWith(
            bottom: pagePadding.bottom + MediaQuery.paddingOf(context).bottom,
          ),
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 1120),
              child: GfPanel(
                emphasized: wide,
                padding: EdgeInsets.all(wide ? 20 : 0),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[
                    _buildTopicFields(l10n),
                    const SizedBox(height: 20),
                    _buildBodyHeader(l10n, wide: wide),
                    const SizedBox(height: 8),
                    if (wide)
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          Expanded(child: _buildEditor(l10n)),
                          const SizedBox(width: 16),
                          Expanded(child: _buildPreview(l10n)),
                        ],
                      )
                    else if (_mode == _ComposeMode.edit)
                      _buildEditor(l10n)
                    else
                      _buildPreview(l10n),
                    const SizedBox(height: 16),
                    if (_error.isNotEmpty)
                      GfStatusMessage(message: _error)
                    else if (_message.isNotEmpty)
                      GfStatusMessage(
                        message: _message,
                        variant: GfStatusMessageVariant.success,
                      ),
                    if (_error.isNotEmpty || _message.isNotEmpty)
                      const SizedBox(height: 12),
                    _buildFooter(l10n),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildTopicFields(AppLocalizations l10n) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        GfInput(
          controller: _title,
          maxLength: 100,
          labelText: l10n.publishTitleField,
          hintText: l10n.publishTitleHint,
          textInputAction: TextInputAction.next,
          style: type.body.copyWith(fontSize: 16, fontWeight: FontWeight.w600),
          onChanged: (_) {
            if (_error.isNotEmpty || _message.isNotEmpty) {
              setState(() {
                _error = '';
                _message = '';
              });
            }
          },
        ),
        const SizedBox(height: 16),
        Row(
          children: <Widget>[
            Expanded(
              child: Text(
                l10n.categoryTitle,
                style: type.small.copyWith(
                  color: colors.baseContent.withValues(alpha: 0.75),
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            Text(
              '${_categoryIds.length}/$_maxCategories',
              style: type.caption.copyWith(
                color: _categoryIds.isEmpty ? colors.iconMuted : colors.primary,
                fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: <Widget>[
            for (final PublishCategoryPayload category in _categories)
              GfSelectTag(
                label: category.name,
                selected: _categoryIds.contains(category.id),
                onChanged:
                    !_categoryIds.contains(category.id) &&
                        _categoryIds.length >= _maxCategories
                    ? null
                    : (bool selected) => _toggleCategory(category, selected),
              ),
          ],
        ),
      ],
    );
  }

  Widget _buildBodyHeader(AppLocalizations l10n, {required bool wide}) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return Row(
      children: <Widget>[
        Expanded(
          child: Text(
            l10n.publishBodyField,
            style: type.small.copyWith(
              color: colors.baseContent.withValues(alpha: 0.75),
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        if (!wide)
          SizedBox(
            width: 176,
            child: GfSegmented<_ComposeMode>(
              segments: <(String, _ComposeMode)>[
                (l10n.composeEdit, _ComposeMode.edit),
                (l10n.composePreview, _ComposeMode.preview),
              ],
              selected: _mode,
              onSelected: _selectMode,
            ),
          ),
      ],
    );
  }

  Widget _buildEditor(AppLocalizations l10n) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    return Container(
      key: const Key('publish-editor'),
      decoration: BoxDecoration(
        color: colors.base100,
        border: Border.all(color: colors.line, width: borders.width),
        borderRadius: BorderRadius.circular(radii.box),
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          _buildToolbar(l10n),
          const GfDivider(),
          ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 340),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: QuillEditor.basic(
                controller: _quill,
                config: QuillEditorConfig(
                  placeholder: l10n.publishBodyPlaceholder,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildToolbar(AppLocalizations l10n) {
    final GfColors colors = GfTheme.colorsOf(context);

    return ColoredBox(
      color: colors.base200.withValues(alpha: 0.55),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
        child: Row(
          children: <Widget>[
            _toolButton(
              icon: Icons.format_bold_rounded,
              tooltip: l10n.publishToolBold,
              onPressed: () => _quill.formatSelection(Attribute.bold),
            ),
            _toolButton(
              icon: Icons.format_italic_rounded,
              tooltip: l10n.publishToolItalic,
              onPressed: () => _quill.formatSelection(Attribute.italic),
            ),
            _toolButton(
              icon: Icons.format_strikethrough_rounded,
              tooltip: l10n.publishToolStrike,
              onPressed: () => _quill.formatSelection(Attribute.strikeThrough),
            ),
            _toolButton(
              icon: Icons.format_quote_rounded,
              tooltip: l10n.publishToolQuote,
              onPressed: () => _quill.formatSelection(Attribute.blockQuote),
            ),
            _toolButton(
              icon: Icons.code_rounded,
              tooltip: l10n.publishToolCode,
              onPressed: () => _quill.formatSelection(Attribute.inlineCode),
            ),
            _toolButton(
              icon: Icons.format_list_bulleted_rounded,
              tooltip: l10n.publishToolBulletList,
              onPressed: () => _quill.formatSelection(Attribute.ul),
            ),
            _toolButton(
              icon: Icons.format_list_numbered_rounded,
              tooltip: l10n.publishToolOrderedList,
              onPressed: () => _quill.formatSelection(Attribute.ol),
            ),
            _toolButton(
              icon: _uploading
                  ? Icons.hourglass_top_rounded
                  : Icons.image_outlined,
              tooltip: l10n.publishToolImage,
              onPressed: _uploading ? null : _pickAndInsertImage,
            ),
          ],
        ),
      ),
    );
  }

  Widget _toolButton({
    required IconData icon,
    required String tooltip,
    required VoidCallback? onPressed,
  }) {
    return GfIconButton(
      icon: icon,
      tooltip: tooltip,
      size: 44,
      iconSize: 20,
      onPressed: onPressed,
    );
  }

  Widget _buildPreview(AppLocalizations l10n) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return Container(
      key: const Key('publish-preview'),
      constraints: const BoxConstraints(minHeight: 390),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.base100,
        border: Border.all(color: colors.line, width: borders.width),
        borderRadius: BorderRadius.circular(radii.box),
      ),
      child: _previewMarkdown.isEmpty
          ? Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  Icon(
                    Icons.article_outlined,
                    size: 32,
                    color: colors.iconMuted,
                  ),
                  const SizedBox(height: 10),
                  Text(
                    l10n.publishPreviewEmpty,
                    textAlign: TextAlign.center,
                    style: type.small.copyWith(color: colors.iconMuted),
                  ),
                ],
              ),
            )
          : GfMarkdownView(data: _previewMarkdown, selectable: true),
    );
  }

  Widget _buildFooter(AppLocalizations l10n) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: <Widget>[
        GfButton(
          key: const Key('publish-save-draft'),
          label: l10n.publishSaveDraft,
          variant: GfButtonVariant.secondary,
          size: GfButtonSize.large,
          loading: _submitting,
          onPressed: _uploading ? null : () => _submit(topicStatus: 0),
        ),
        const SizedBox(width: 8),
        GfButton(
          key: const Key('publish-footer-submit'),
          label: l10n.publishPublish,
          variant: GfButtonVariant.primary,
          size: GfButtonSize.large,
          loading: _submitting,
          icon: const Icon(Icons.send_rounded, size: 18),
          onPressed: _uploading ? null : () => _submit(topicStatus: 1),
        ),
      ],
    );
  }
}

class _PublishWorkspaceSkeleton extends StatelessWidget {
  const _PublishWorkspaceSkeleton();

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: ListView(
        physics: const NeverScrollableScrollPhysics(),
        padding: const EdgeInsets.all(16),
        children: const <Widget>[
          GfSkeleton(height: 52, radius: 8),
          SizedBox(height: 16),
          GfSkeleton(width: 80, height: 14, radius: 5),
          SizedBox(height: 8),
          Row(
            children: <Widget>[
              GfSkeleton(width: 72, height: 36, radius: 8),
              SizedBox(width: 8),
              GfSkeleton(width: 72, height: 36, radius: 8),
              SizedBox(width: 8),
              GfSkeleton(width: 72, height: 36, radius: 8),
            ],
          ),
          SizedBox(height: 20),
          GfSkeleton(width: 56, height: 14, radius: 5),
          SizedBox(height: 8),
          GfSkeleton(height: 44, radius: 8),
          SizedBox(height: 1),
          GfSkeleton(height: 340, radius: 8),
        ],
      ),
    );
  }
}

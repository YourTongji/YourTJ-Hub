import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../asset_url.dart';
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
    this.initialContentType = 3,
    this.editTitle,
    this.editContent,
    this.editCategoryIds,
    @visibleForTesting this.markdownConverter,
  });

  final int? topicId;
  final int initialContentType;

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

  final TextEditingController _simple = TextEditingController();
  final List<String> _images = [];
  int _contentType = 3;
  bool _formatting = false;
  bool _dirty = false;
  bool _allowPop = false;
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
  CaptchaPayload? _captcha;
  final _captchaCode = TextEditingController();
  bool _captchaLoading = false;
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
    _contentType = widget.initialContentType;
    _simple.text = widget.editContent ?? '';
    _title.text = widget.editTitle ?? '';
    _categoryIds.addAll(widget.editCategoryIds ?? const <int>[]);
    _quill = _createController('');
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
    // Decode before replacing the live controller. A failed conversion must
    // leave the previous document and its subscription safe to retry/dispose.
    final previousController = _quill;
    final previousChanges = _documentChanges;
    final nextController = _createController(markdown);
    unawaited(previousChanges.cancel());
    previousController.dispose();
    _quill = nextController;
    _previewMarkdown = _markdownFromEditor();
  }

  String _markdownFromEditor() {
    return _contentType == 3
        ? _converter.documentToMarkdown(_quill.document).trim()
        : _simple.text.trim();
  }

  void _handleEditorChanged() {
    if (!mounted) return;
    _dirty = true;
    _allowPop = false;
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
    FocusManager.instance.primaryFocus?.unfocus();
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

      // The publish payload lacks gallery metadata; read the topic's existing
      // projection before editing so a simple-text edit cannot remove photos.
      List<String> existingImages = [];
      if (props.isEditing &&
          props.topic.contentType != 3 &&
          props.topic.contentType != 0) {
        final detail = await ref
            .read(pageRepositoryProvider)
            .topicDetail(props.topicId);
        final existing = parsePageProps<TopicDetailProps>(detail);
        if (existing == null || existing.topic.id != props.topicId) {
          throw const FormatException(
            'Existing topic gallery could not be read',
          );
        }
        existingImages = existing.topic.images ?? const [];
        if (!mounted) return;
      }
      _contentType = props.isEditing
          ? (props.topic.contentType == 0 ? 3 : props.topic.contentType)
          : widget.initialContentType;
      _simple.text = payloadContent;
      _images
        ..clear()
        ..addAll(existingImages);
      _replaceEditorDocument(_contentType == 3 ? payloadContent : '');
      _dirty = false;
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
    _captchaCode.dispose();
    _title.dispose();
    _simple.dispose();
    unawaited(_documentChanges.cancel());
    _quill.dispose();
    super.dispose();
  }

  Future<void> _goBack() async {
    if (_mode == _ComposeMode.preview) {
      _selectMode(_ComposeMode.edit);
      return;
    }
    if (_dirty && !_allowPop) {
      final l10n = AppLocalizations.of(context);
      final discard = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(l10n.publishLeaveTitle),
          content: Text(l10n.publishLeaveBody),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(l10n.publishContinue),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(l10n.publishDiscard),
            ),
          ],
        ),
      );
      if (discard != true || !mounted) return;
    }
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _leave();
    });
  }

  void _leave() {
    if (context.canPop()) {
      context.pop();
    } else {
      context.go('/');
    }
  }

  void _changeContentType(int? value) {
    if (value == null || value == _contentType) return;
    var markdown = _markdownFromEditor();
    final photos = <String>[];
    if (_contentType == 3 && value != 3) {
      for (final operation in _quill.document.toDelta().toJson()) {
        final insert = operation['insert'];
        if (insert is Map && insert['image'] is String) {
          photos.add(insert['image'] as String);
        }
      }
      if (photos.length > 9) {
        showGfToast(
          context,
          AppLocalizations.of(context).publishGalleryTooMany,
          error: true,
        );
        return;
      }
      markdown = _quill.document.toPlainText().replaceAll('\uFFFC', '').trim();
    } else if (value == 3 && _images.isNotEmpty) {
      markdown =
          '$markdown\n\n${_images.map((url) => '![image]($url)').join('\n\n')}';
    }
    try {
      if (value == 3) _replaceEditorDocument(markdown);
    } catch (error) {
      showGfToast(
        context,
        resolveErrorMessage(AppLocalizations.of(context), error),
        error: true,
      );
      return;
    }
    setState(() {
      if (_contentType == 3 || value == 3) {
        _images
          ..clear()
          ..addAll(photos);
      }
      _contentType = value;
      _simple.text = markdown;
      _previewMarkdown = _markdownFromEditor();
      _dirty = true;
      _allowPop = false;
    });
  }

  void _toggleFormat(Attribute attribute) {
    final current = _quill.getSelectionStyle().attributes[attribute.key];
    _quill.formatSelection(
      current?.value == attribute.value
          ? Attribute.clone(attribute, null)
          : attribute,
    );
  }

  void _toggleCategory(PublishCategoryPayload category, bool selected) {
    setState(() {
      _dirty = true;
      _allowPop = false;
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
    if (_uploading || _submitting) return;
    final l10n = AppLocalizations.of(context);
    final source = await showModalBottomSheet<ImageSource>(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.photo_library_outlined),
              title: Text(l10n.publishPhotoLibrary),
              onTap: () => Navigator.pop(context, ImageSource.gallery),
            ),
            ListTile(
              leading: const Icon(Icons.camera_alt_outlined),
              title: Text(l10n.publishCamera),
              onTap: () => Navigator.pop(context, ImageSource.camera),
            ),
          ],
        ),
      ),
    );
    if (source == null || !mounted) return;
    setState(() => _uploading = true);
    try {
      final String? url = await pickAndUploadImage(ref: ref, source: source);
      if (url == null || !mounted) return;
      _dirty = true;
      _allowPop = false;
      if (_contentType != 3) {
        setState(() => _images.add(url));
        return;
      }
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

  Future<void> _loadCaptcha() async {
    if (_captchaLoading) return;
    setState(() => _captchaLoading = true);
    try {
      final captcha = await ref.read(authRepositoryProvider).getCaptcha();
      if (mounted) {
        setState(() {
          _captcha = captcha;
          _captchaCode.clear();
        });
      }
    } catch (error) {
      if (mounted) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _captchaLoading = false);
    }
  }

  Future<void> _submit({required int topicStatus}) async {
    if (_submitting || _loading || _loadError.isNotEmpty) return;

    final String title = _title.text.trim().isNotEmpty
        ? _title.text.trim()
        : _contentType == 2
        ? (_simple.text.trim().isEmpty
              ? AppLocalizations.of(context).publishImageOnlyTitle
              : _simple.text
                    .trim()
                    .split('\n')
                    .first
                    .characters
                    .take(60)
                    .toString())
        : '';
    final String content = _markdownFromEditor().isEmpty && _images.isNotEmpty
        ? AppLocalizations.of(context).publishImageOnlyTitle
        : _markdownFromEditor();
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
        if (_categoryIds.isEmpty) _mode = _ComposeMode.preview;
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
            captchaId: _captcha?.captchaId,
            captchaCode: _captchaCode.text.trim(),
            topicId: _currentTopicId,
            title: title,
            content: content,
            categoryIds: List<int>.of(_categoryIds),
            topicStatus: topicStatus,
            contentType: _contentType,
            images: _contentType == 3 ? null : List.of(_images),
          );
      if (!mounted) return;

      _dirty = false;
      _allowPop = true;
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
      if (mounted) setState(() => _error = resolveErrorMessage(l10n, error));
      if (error.messageCode == 'common.captchaRequired' ||
          error.messageCode == 'auth.captcha.invalid') {
        await _loadCaptcha();
      }
    } catch (error) {
      if (mounted) setState(() => _error = l10n.publishFailed('$error'));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);

    return PopScope(
      canPop: _allowPop || (!_dirty && _mode == _ComposeMode.edit),
      onPopInvokedWithResult: (didPop, result) {
        if (!didPop) _goBack();
      },
      child: Scaffold(
        backgroundColor: GfTheme.colorsOf(context).base100,
        appBar: GfAppBar(
          leading: GfIconButton(
            icon: Icons.arrow_back_rounded,
            tooltip: l10n.commonBack,
            size: 44,
            onPressed: _goBack,
          ),
          title: DropdownButtonHideUnderline(
            child: DropdownButton<int>(
              value: _contentType,
              items: [
                DropdownMenuItem(value: 2, child: Text(l10n.publishMoment)),
                DropdownMenuItem(value: 1, child: Text(l10n.publishQuestion)),
                DropdownMenuItem(value: 3, child: Text(l10n.publishArticle)),
              ],
              onChanged:
                  _currentTopicId > 0 || _submitting || _uploading || _loading
                  ? null
                  : _changeContentType,
            ),
          ),
          actions: <Widget>[
            if (MediaQuery.viewInsetsOf(context).bottom > 0)
              GfIconButton(
                icon: Icons.keyboard_hide_rounded,
                tooltip: l10n.commonHideKeyboard,
                size: 44,
                onPressed: () => FocusManager.instance.primaryFocus?.unfocus(),
              ),
            GfButton(
              key: const Key('publish-appbar-submit'),
              label: _mode == _ComposeMode.edit
                  ? l10n.publishNext
                  : l10n.publishPublish,
              variant: GfButtonVariant.primary,
              size: GfButtonSize.small,
              loading: _submitting,
              onPressed: _loading || _uploading || _loadError.isNotEmpty
                  ? null
                  : () => _mode == _ComposeMode.edit
                        ? _selectMode(_ComposeMode.preview)
                        : _submit(topicStatus: 1),
            ),
          ],
        ),
        body: AbsorbPointer(absorbing: _submitting, child: _buildBody(l10n)),
        bottomNavigationBar:
            _mode == _ComposeMode.edit && !_loading && _loadError.isEmpty
            ? Padding(
                padding: EdgeInsets.only(
                  bottom: MediaQuery.viewInsetsOf(context).bottom,
                ),
                child: SafeArea(
                  top: false,
                  child: AbsorbPointer(
                    absorbing: _submitting,
                    child: _buildWritingToolbar(l10n),
                  ),
                ),
              )
            : null,
      ),
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
          keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
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
                    if (_mode == _ComposeMode.edit && _contentType != 3) ...[
                      _buildGallery(l10n, editing: true),
                      const SizedBox(height: 16),
                    ],
                    if (_mode == _ComposeMode.edit || wide) ...[
                      _buildTopicFields(l10n),
                      const SizedBox(height: 4),
                    ],
                    if (wide) ...[
                      _buildBodyHeader(l10n),
                      const SizedBox(height: 8),
                    ],
                    if (wide)
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          Expanded(child: _buildEditor(l10n)),
                          const SizedBox(width: 16),
                          Expanded(child: _buildPreview(l10n, framed: true)),
                        ],
                      )
                    else if (_mode == _ComposeMode.edit)
                      _buildEditor(l10n)
                    else
                      _buildPreview(l10n),
                    if (_mode == _ComposeMode.preview) ...[
                      const SizedBox(height: 24),
                      _buildTopicFields(l10n, classification: true),
                    ],
                    const SizedBox(height: 16),
                    if (_captcha != null) ...[
                      Row(
                        children: [
                          InkWell(
                            onTap: _captchaLoading ? null : _loadCaptcha,
                            child: Image.memory(
                              base64Decode(
                                _captcha!.captchaImg.split(',').last,
                              ),
                              width: 128,
                              height: 48,
                              gaplessPlayback: true,
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: GfInput(
                              controller: _captchaCode,
                              labelText: l10n.authCaptcha,
                            ),
                          ),
                          IconButton(
                            onPressed: _captchaLoading ? null : _loadCaptcha,
                            tooltip: l10n.authGetCode,
                            icon: const Icon(Icons.refresh),
                          ),
                        ],
                      ),
                      const SizedBox(height: 16),
                    ],
                    if (_error.isNotEmpty)
                      GfStatusMessage(message: _error)
                    else if (_message.isNotEmpty)
                      GfStatusMessage(
                        message: _message,
                        variant: GfStatusMessageVariant.success,
                      ),
                    if (_error.isNotEmpty || _message.isNotEmpty)
                      const SizedBox(height: 12),
                    if (_mode == _ComposeMode.preview) _buildFooter(l10n),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildTopicFields(
    AppLocalizations l10n, {
    bool classification = false,
  }) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        if (!classification)
          TextField(
            controller: _title,
            maxLength: 100,
            decoration: InputDecoration(
              hintText: l10n.publishTitleHint,
              hintStyle: TextStyle(
                color: colors.iconMuted.withValues(alpha: 0.75),
                fontWeight: FontWeight.w500,
              ),
              border: InputBorder.none,
              enabledBorder: InputBorder.none,
              focusedBorder: InputBorder.none,
              filled: false,
              contentPadding: const EdgeInsets.symmetric(vertical: 10),
              counterText: '',
            ),
            textInputAction: TextInputAction.next,
            style: type.heading.copyWith(
              fontSize: 22,
              fontWeight: FontWeight.w600,
            ),
            onChanged: (_) {
              _dirty = true;
              _allowPop = false;
              if (_error.isNotEmpty || _message.isNotEmpty) {
                setState(() {
                  _error = '';
                  _message = '';
                });
              }
            },
          ),
        if (classification) ...[
          Row(
            children: <Widget>[
              Expanded(
                child: Text(
                  l10n.publishClassification,
                  style: type.small.copyWith(
                    color: colors.baseContent.withValues(alpha: 0.75),
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              Text(
                '${_categoryIds.length}/$_maxCategories',
                style: type.caption.copyWith(
                  color: _categoryIds.isEmpty
                      ? colors.iconMuted
                      : colors.primary,
                  fontFeatures: const <FontFeature>[
                    FontFeature.tabularFigures(),
                  ],
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
      ],
    );
  }

  Widget _buildBodyHeader(AppLocalizations l10n) => Text(
    l10n.publishBodyField,
    style: GfTheme.typographyOf(
      context,
    ).small.copyWith(color: GfTheme.colorsOf(context).iconMuted),
  );

  Widget _buildEditor(AppLocalizations l10n) {
    if (_contentType != 3) {
      return TextField(
        key: const Key('publish-editor'),
        controller: _simple,
        keyboardType: TextInputType.multiline,
        textInputAction: TextInputAction.newline,
        style: GfTheme.typographyOf(
          context,
        ).body.copyWith(fontSize: 17, height: 1.45),
        decoration: InputDecoration(
          hintText: l10n.publishBodyPlaceholder,
          border: InputBorder.none,
          enabledBorder: InputBorder.none,
          focusedBorder: InputBorder.none,
          filled: false,
          contentPadding: const EdgeInsets.symmetric(vertical: 8),
        ),
        minLines: 6,
        maxLines: 80,
        onChanged: (_) {
          _handleEditorChanged();
        },
      );
    }
    final type = GfTheme.typographyOf(context);
    final defaults = DefaultStyles.getInstance(context);
    return ConstrainedBox(
      key: const Key('publish-editor'),
      constraints: const BoxConstraints(minHeight: 220),
      child: QuillEditor.basic(
        controller: _quill,
        config: QuillEditorConfig(
          scrollable: false,
          padding: const EdgeInsets.symmetric(vertical: 8),
          placeholder: l10n.publishBodyPlaceholder,
          customStyles: DefaultStyles(
            paragraph: defaults.paragraph!.copyWith(
              style: type.body,
              verticalSpacing: const VerticalSpacing(4, 4),
            ),
            placeHolder: defaults.placeHolder!.copyWith(
              style: type.body.copyWith(
                color: GfTheme.colorsOf(context).iconMuted,
              ),
            ),
          ),
          embedBuilders: [_ComposerImageBuilder()],
        ),
      ),
    );
  }

  Widget _buildWritingToolbar(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    return DecoratedBox(
      key: const Key('publish-writing-tools'),
      decoration: BoxDecoration(
        color: colors.base100,
        border: Border(top: BorderSide(color: colors.line)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (_contentType == 3 && _formatting) _buildToolbar(l10n),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            child: Row(
              children: [
                if (_contentType == 3)
                  Flexible(
                    child: TextButton.icon(
                      onPressed: () =>
                          setState(() => _formatting = !_formatting),
                      icon: Icon(
                        _formatting
                            ? Icons.keyboard_arrow_down_rounded
                            : Icons.text_fields_rounded,
                        size: 20,
                      ),
                      label: Text(
                        l10n.publishFormatting,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      style: TextButton.styleFrom(
                        foregroundColor: _formatting
                            ? colors.primary
                            : colors.iconMuted,
                        backgroundColor: _formatting
                            ? colors.primary.withValues(alpha: 0.08)
                            : colors.base200,
                        minimumSize: const Size(44, 44),
                      ),
                    ),
                  ),
                _toolButton(
                  icon: _uploading
                      ? Icons.hourglass_top_rounded
                      : Icons.image_outlined,
                  tooltip: l10n.publishToolImage,
                  onPressed:
                      _uploading || (_contentType != 3 && _images.length >= 9)
                      ? null
                      : _pickAndInsertImage,
                ),
                Expanded(
                  child: Align(
                    alignment: Alignment.centerRight,
                    child: GfButton(
                      key: const Key('publish-save-draft'),
                      label: l10n.publishSaveDraft,
                      variant: GfButtonVariant.ghost,
                      loading: _submitting,
                      onPressed: _uploading
                          ? null
                          : () => _submit(topicStatus: 0),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _insertLink() async {
    final l10n = AppLocalizations.of(context);
    String value = '';
    final url = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.publishToolLink),
        content: TextField(
          autofocus: true,
          keyboardType: TextInputType.url,
          decoration: const InputDecoration(hintText: 'https://'),
          onChanged: (text) => value = text,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, value.trim()),
            child: Text(l10n.commonConfirm),
          ),
        ],
      ),
    );
    if (url == null || !mounted) return;
    final uri = Uri.tryParse(url);
    if (uri == null ||
        !['https', 'http', 'mailto'].contains(uri.scheme) ||
        (uri.scheme != 'mailto' && uri.host.isEmpty)) {
      showGfToast(context, l10n.publishLinkInvalid, error: true);
      return;
    }
    final selection = _quill.selection;
    if (selection.isCollapsed || !selection.isValid) {
      final start = selection.isValid ? selection.start : 0;
      _quill.replaceText(
        start,
        0,
        url,
        TextSelection(baseOffset: start, extentOffset: start + url.length),
      );
    }
    _quill.formatSelection(LinkAttribute(url));
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
              icon: Icons.undo,
              tooltip: l10n.publishUndo,
              onPressed: _quill.undo,
            ),
            _toolButton(
              icon: Icons.redo,
              tooltip: l10n.publishRedo,
              onPressed: _quill.redo,
            ),
            _toolButton(
              icon: Icons.title,
              tooltip: l10n.publishHeading,
              onPressed: () => _toggleFormat(Attribute.h2),
            ),
            _toolButton(
              icon: Icons.link,
              tooltip: l10n.publishToolLink,
              onPressed: _insertLink,
            ),

            _toolButton(
              icon: Icons.format_bold_rounded,
              tooltip: l10n.publishToolBold,
              onPressed: () => _toggleFormat(Attribute.bold),
            ),
            _toolButton(
              icon: Icons.format_italic_rounded,
              tooltip: l10n.publishToolItalic,
              onPressed: () => _toggleFormat(Attribute.italic),
            ),
            _toolButton(
              icon: Icons.format_strikethrough_rounded,
              tooltip: l10n.publishToolStrike,
              onPressed: () => _toggleFormat(Attribute.strikeThrough),
            ),
            _toolButton(
              icon: Icons.format_quote_rounded,
              tooltip: l10n.publishToolQuote,
              onPressed: () => _toggleFormat(Attribute.blockQuote),
            ),
            _toolButton(
              icon: Icons.code_rounded,
              tooltip: l10n.publishToolCode,
              onPressed: () => _toggleFormat(Attribute.inlineCode),
            ),
            _toolButton(
              icon: Icons.format_list_bulleted_rounded,
              tooltip: l10n.publishToolBulletList,
              onPressed: () => _toggleFormat(Attribute.ul),
            ),
            _toolButton(
              icon: Icons.format_list_numbered_rounded,
              tooltip: l10n.publishToolOrderedList,
              onPressed: () => _toggleFormat(Attribute.ol),
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

  Widget _buildPreview(AppLocalizations l10n, {bool framed = false}) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return Container(
      key: const Key('publish-preview'),
      constraints: BoxConstraints(minHeight: framed ? 390 : 120),
      padding: framed ? const EdgeInsets.all(16) : EdgeInsets.zero,
      decoration: BoxDecoration(
        color: colors.base100,
        border: framed
            ? Border.all(color: colors.line, width: borders.width)
            : null,
        borderRadius: BorderRadius.circular(radii.box),
      ),
      child: _previewMarkdown.isEmpty && _images.isEmpty
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
          : Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (_images.isNotEmpty)
                  GfMediaCarousel(
                    images: _images.map(resolveApiAssetUrl).toList(),
                  ),
                if (_title.text.trim().isNotEmpty) ...[
                  const SizedBox(height: 16),
                  Text(
                    _title.text.trim(),
                    style: type.heading.copyWith(
                      fontSize: 22,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
                if (_contentType == 3)
                  GfMarkdownView(data: _previewMarkdown, selectable: true)
                else
                  SelectableText(
                    _simple.text,
                    style: type.body.copyWith(fontSize: 17, height: 1.45),
                  ),
              ],
            ),
    );
  }

  Widget _buildGallery(AppLocalizations l10n, {required bool editing}) {
    if (editing && _images.isEmpty) {
      final colors = GfTheme.colorsOf(context);
      final type = GfTheme.typographyOf(context);
      return Material(
        color: colors.base200,
        borderRadius: BorderRadius.circular(16),
        child: InkWell(
          onTap: _uploading ? null : _pickAndInsertImage,
          borderRadius: BorderRadius.circular(16),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                _uploading
                    ? const SizedBox.square(
                        dimension: 24,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Icon(
                        Icons.add_photo_alternate_outlined,
                        size: 28,
                        color: colors.primary,
                      ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(l10n.publishGallery, style: type.bodyStrong),
                      const SizedBox(height: 4),
                      Text(
                        l10n.publishGalleryHint,
                        style: type.caption.copyWith(color: colors.iconMuted),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (editing) ...[
          Text(l10n.publishGallery, style: GfTheme.typographyOf(context).body),
          Text(
            l10n.publishGalleryHint,
            style: GfTheme.typographyOf(context).caption,
          ),
          const SizedBox(height: 10),
        ],
        if (_images.isNotEmpty)
          SizedBox(
            height: editing ? 120 : 260,
            child: editing
                ? ReorderableListView.builder(
                    scrollDirection: Axis.horizontal,
                    itemCount: _images.length,
                    onReorderItem: (oldIndex, newIndex) => setState(() {
                      _images.insert(newIndex, _images.removeAt(oldIndex));
                      _dirty = true;
                      _allowPop = false;
                    }),
                    itemBuilder: (context, index) => SizedBox(
                      key: ValueKey('$index:${_images[index]}'),
                      width: 128,
                      child: Stack(
                        children: [
                          Positioned.fill(
                            child: Padding(
                              padding: const EdgeInsets.all(4),
                              child: Image.network(
                                resolveApiAssetUrl(_images[index]),
                                fit: BoxFit.contain,
                                errorBuilder: (_, _, _) =>
                                    const Icon(Icons.broken_image_outlined),
                              ),
                            ),
                          ),
                          Positioned(
                            right: 0,
                            top: 0,
                            child: IconButton(
                              tooltip: l10n.publishRemoveImage,
                              onPressed: () => setState(() {
                                _images.removeAt(index);
                                _dirty = true;
                                _allowPop = false;
                              }),
                              icon: const Icon(Icons.cancel),
                            ),
                          ),
                        ],
                      ),
                    ),
                  )
                : PageView(
                    children: [
                      for (final url in _images)
                        Image.network(
                          resolveApiAssetUrl(url),
                          fit: BoxFit.contain,
                          errorBuilder: (_, _, _) =>
                              const Icon(Icons.broken_image_outlined),
                        ),
                    ],
                  ),
          ),
        if (editing)
          GfButton(
            label: l10n.publishToolImage,
            icon: const Icon(Icons.add_photo_alternate_outlined),
            variant: GfButtonVariant.outline,
            loading: _uploading,
            onPressed: _images.length >= 9 || _uploading
                ? null
                : _pickAndInsertImage,
          ),
      ],
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
          label: _mode == _ComposeMode.edit
              ? l10n.publishNext
              : l10n.publishPublish,
          variant: GfButtonVariant.primary,
          size: GfButtonSize.large,
          loading: _submitting,
          icon: const Icon(Icons.send_rounded, size: 18),
          onPressed: _uploading
              ? null
              : () => _mode == _ComposeMode.edit
                    ? _selectMode(_ComposeMode.preview)
                    : _submit(topicStatus: 1),
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

class _ComposerImageBuilder extends EmbedBuilder {
  @override
  String get key => BlockEmbed.imageType;
  @override
  Widget build(BuildContext context, EmbedContext embedContext) =>
      Image.network(
        resolveApiAssetUrl(embedContext.node.value.data.toString()),
        fit: BoxFit.contain,
        errorBuilder: (_, _, _) => const Icon(Icons.broken_image_outlined),
      );
}

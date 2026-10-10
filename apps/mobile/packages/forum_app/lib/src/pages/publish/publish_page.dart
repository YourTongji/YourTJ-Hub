import '../../widgets/stickers/sticker_draft_preview.dart';
import '../../widgets/identity_picker.dart';
import '../../widgets/stickers/sticker_picker.dart';
import '../../widgets/stickers/sticker_strings.dart';
import 'dart:async';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../local/writing_store.dart';
import '../../asset_url.dart';
import '../../format.dart';
import '../../images/image_upload.dart';
import '../../images/composer_upload_queue.dart';
import '../../server_messages.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/editor/rich_markdown_editor.dart';
import '../../widgets/status_views.dart';
import '../../widgets/moderation_blocked_dialog.dart';
import 'embed_image_move.dart';
import 'publish_type.dart';

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
    this.localDraftKey,
    this.initialContentType = 3,
    this.editTitle,
    this.editContent,
    this.editCategoryIds,
    @visibleForTesting this.markdownConverter,
  });

  final int? topicId;
  final String? localDraftKey;
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

class _PublishPageState extends ConsumerState<PublishPage>
    with WidgetsBindingObserver {
  /// Articles take up to three categories; moments and questions take one,
  /// matching the web quick composer.
  static const int _maxArticleCategories = 3;
  static const double _wideWorkspaceBreakpoint = 760;
  static const Duration _previewDebounceDuration = Duration(milliseconds: 200);
  static const double _dragAutoscrollEdge = 56;

  int get _categoryLimit => _contentType == 3 ? _maxArticleCategories : 1;
  static const double _dragAutoscrollStep = 18;

  final TextEditingController _simple = TextEditingController();
  final FocusNode _simpleFocusNode = FocusNode(debugLabel: 'simple-body');
  final GlobalKey _bodyEditorContainerKey = GlobalKey();
  bool _stickerOpen = false;
  TextSelection? _stickerSelection;
  final List<String> _images = [];
  int _contentType = 3;
  bool _dirty = false;
  late final WritingStore _localStore;
  late final OfflineCacheEpoch _session;
  late final int _epoch;
  late String _draftKey;
  DraftKind? _draftKind;
  String? _owner;
  Timer? _autosave;
  int _revision = 0;
  int _savedRevision = 0;
  bool _localRestored = false;
  String _localStatus = '';
  bool _localSaveFailed = false;
  bool _finished = false;
  bool _saveStatusScheduled = false;
  bool _allowPop = false;
  final TextEditingController _title = TextEditingController();
  final FocusNode _titleFocusNode = FocusNode(debugLabel: 'topic-title');
  final List<int> _categoryIds = <int>[];
  final MenuController _categoryMenu = MenuController();
  bool _categoryMenuOpen = false;
  final List<PublishCategoryPayload> _categories = <PublishCategoryPayload>[];
  late final MarkdownConverter _converter;

  late QuillController _quill;
  final GlobalKey<EditorState> _editorKey = GlobalKey<EditorState>();
  // QuillEditor.basic otherwise creates new input nodes on each page rebuild.
  final FocusNode _editorFocusNode = FocusNode(debugLabel: 'article-body');
  final ScrollController _editorScrollController = ScrollController();
  final ScrollController _pageScrollController = ScrollController();
  final GlobalKey _pageScrollViewKey = GlobalKey();
  late StreamSubscription<DocChange> _documentChanges;
  late int _currentTopicId;
  bool _agentRepliesDisabled = false;

  _ComposeMode _mode = _ComposeMode.edit;
  double? _editScrollOffset;
  bool _restoreEditorFocus = false;
  int _modeRevision = 0;
  String _identity = "member";
  bool _loading = true;
  bool _submitting = false;
  late final ComposerUploadQueue _uploads;
  bool _pickingImage = false;
  bool get _uploading => _pickingImage || _uploads.hasPending;
  bool get _activelyUploading =>
      _pickingImage ||
      _uploads.items.any(
        (item) => item.status == ComposerUploadStatus.uploading,
      );
  // Body offsets for article photos: one captured while the picker is open,
  // then one per queued upload. Each follows later edits until it lands.
  int? _pickInsertAt;
  final Map<int, int> _mediaInsertAts = {};
  CaptchaPayload? _captcha;
  final _captchaCode = TextEditingController();
  bool _captchaLoading = false;
  bool _showPublishCaptchaExplanation = false;
  String _loadError = '';
  String _error = '';
  String _message = '';
  String _previewMarkdown = '';
  Timer? _previewDebounce;
  Timer? _dragAutoscrollTimer;
  Offset? _dragPointer;
  // Where a dragged body image would land, painted over the editor without
  // rebuilding it on every pointer move.
  final ValueNotifier<double?> _dropLineY = ValueNotifier(null);
  final GlobalKey _editorDropKey = GlobalKey();
  int? _dropOffset;

  @override
  void initState() {
    super.initState();
    _localStore = ref.read(writingStoreProvider);
    _session = ref.read(offlineCacheEpochProvider.notifier);
    _epoch = ref.read(offlineCacheEpochProvider);
    _draftKey =
        widget.localDraftKey ??
        (widget.topicId == null
            ? newTopicDraftKey()
            : topicDraftKey(widget.topicId!, published: true));
    WidgetsBinding.instance.addObserver(this);
    _title.addListener(_textChanged);
    _simple.addListener(_textChanged);
    _converter = widget.markdownConverter ?? MarkdownConverter();
    _currentTopicId = widget.topicId ?? 0;
    _contentType = widget.initialContentType;
    _simple.text = widget.editContent ?? '';
    _title.text = widget.editTitle ?? '';
    _categoryIds.addAll(widget.editCategoryIds ?? const <int>[]);
    _quill = _createController('');
    _simpleFocusNode.addListener(_bodyFocusChanged);
    _editorFocusNode.addListener(_bodyFocusChanged);
    _titleFocusNode.addListener(_bodyFocusChanged);
    final files = ref.read(fileRepositoryProvider);
    _uploads = ComposerUploadQueue(
      isCurrent: () => mounted && _sessionCurrent && !_finished,
      upload: (file) async {
        final bytes = await file.readAsBytes();
        if (!mounted || !_sessionCurrent || _finished) {
          throw StateError('Image upload session changed');
        }
        return files.uploadImage(bytes: bytes, filename: file.name);
      },
      onUploaded: _insertUploadedImage,
    )..addListener(_uploadsChanged);
    _previewMarkdown = _markdownFromEditor();
    _initializeDraft();
  }

  bool get _sessionCurrent => _session.isCurrent(_epoch);

  void _uploadsChanged() {
    if (!mounted || !_sessionCurrent) return;
    setState(() {
      final ids = {for (final item in _uploads.items) item.id};
      _mediaInsertAts.removeWhere((id, _) => !ids.contains(id));
    });
  }

  Future<void> _initializeDraft() async {
    try {
      final owner = await ref.read(writingScopeProvider.future);
      if (!mounted || !_sessionCurrent) return;
      if (!owner.endsWith(':0')) _owner = owner;
    } catch (_) {
      /* Network editor remains usable if identity storage fails. */
    }
    if (mounted && _sessionCurrent) await _loadEditorData();
  }

  String _lastTitleText = '';
  String _lastSimpleText = '';

  // Controller notifications include selection and IME composition changes.
  // Only changed text is new user work; merely focusing must not save a draft.
  void _textChanged() {
    final changed =
        _lastTitleText != _title.text || _lastSimpleText != _simple.text;
    _lastTitleText = _title.text;
    _lastSimpleText = _simple.text;
    if (changed) _markDirty();
  }

  void _markDirty() {
    if (_loading || _finished || !_sessionCurrent) return;
    _dirty = true;
    _allowPop = false;
    _revision++;
    _localSaveFailed = false;
    if (!_saveStatusScheduled) {
      _saveStatusScheduled = true;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _saveStatusScheduled = false;
        // A guest has no persistent owner; do not show a pending save forever.
        if (mounted &&
            _sessionCurrent &&
            !_finished &&
            !_localSaveFailed &&
            _owner != null &&
            _revision != _savedRevision) {
          setState(() {
            _localStatus = AppLocalizations.of(context).draftLocalSaving;
            _localSaveFailed = false;
          });
        }
      });
    }
    _autosave?.cancel();
    _autosave = Timer(const Duration(milliseconds: 700), _saveLocal);
  }

  Future<bool> _saveLocal({bool notify = true}) async {
    _autosave?.cancel();
    if (_finished ||
        !_sessionCurrent ||
        !_dirty ||
        _revision == _savedRevision) {
      return true;
    }
    final owner = _owner;
    if (owner == null) return false;
    final revision = _revision;
    final l10n = notify ? AppLocalizations.of(context) : null;
    if (notify && mounted) {
      setState(() {
        _localStatus = l10n!.draftLocalSaving;
        _localSaveFailed = false;
      });
    }
    try {
      final draft = LocalDraft(
        agentRepliesDisabled: _agentRepliesDisabled,
        identity: _identity,
        key: _draftKey,
        kind: _draftKind ?? DraftKind.newTopic,
        title: _title.text,
        content: _contentType == 3
            ? _converter.documentToMarkdown(_quill.document)
            : _simple.text,
        contentType: _contentType,
        topicId: _currentTopicId,
        categories: List.of(_categoryIds),
        images: List.of(_images),
        updatedAt: DateTime.now().millisecondsSinceEpoch,
      );
      await _localStore.save(owner, draft, isCurrent: () => _sessionCurrent);
      if (!mounted || !_sessionCurrent || _finished) return false;
      _savedRevision = revision;
      if (notify && revision == _revision) {
        setState(
          () => _localStatus = _uploads.hasPending
              ? l10n!.publishMediaSavedPartial
              : l10n!.draftLocalSaved,
        );
      }
      return true;
    } catch (_) {
      if (notify && mounted && _sessionCurrent) {
        setState(() {
          _localSaveFailed = true;
          _localStatus = l10n!.draftLocalSaveFailed;
        });
      }
      return false;
    }
  }

  Future<void> _restoreLocal() async {
    final owner = _owner;
    if (owner == null || _dirty) return;
    final drafts = await _localStore.drafts(owner);
    if (!mounted || !_sessionCurrent || _dirty) return;
    final unknownTopicKind =
        widget.localDraftKey == null &&
        _currentTopicId > 0 &&
        _draftKind == null;
    // Offline metadata cannot tell a cloud draft from a published topic. Prefer
    // the newest matching recovery copy across both modern identities and v1.
    final recoveryKeys = {
      topicDraftKey(_currentTopicId, published: true),
      topicDraftKey(_currentTopicId, published: false),
      'topic-$_currentTopicId',
    };
    var matching = drafts.where(
      (draft) =>
          draft.kind != DraftKind.reply &&
          (unknownTopicKind
              ? recoveryKeys.contains(draft.key)
              : draft.key == _draftKey),
    );
    if (matching.isEmpty &&
        widget.localDraftKey == null &&
        _currentTopicId > 0) {
      matching = drafts.where((draft) => draft.key == 'topic-$_currentTopicId');
    }
    if (matching.isEmpty) return;
    final draft = matching.first;
    _agentRepliesDisabled = draft.agentRepliesDisabled;
    _draftKey = draft.key;
    _draftKind ??= draft.kind;
    _contentType = draft.contentType;
    _currentTopicId = draft.topicId;
    _identity = draft.identity;
    _title.text = draft.title;
    _simple.text = draft.content;
    _images
      ..clear()
      ..addAll(draft.images);
    _categoryIds
      ..clear()
      ..addAll(draft.categories);
    _replaceEditorDocument(_contentType == 3 ? draft.content : '');
    _localRestored = true;
    _dirty = true;
    _localStatus = AppLocalizations.of(context).draftLocalRestored;
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _uploads.setActive(state == AppLifecycleState.resumed);
    if (state != AppLifecycleState.resumed) unawaited(_saveLocal());
  }

  QuillController _createController(String markdown) {
    final QuillController controller = QuillController(
      document: _converter.mdToDocument(markdown),
      selection: const TextSelection.collapsed(offset: 0),
    );
    controller.addListener(() {
      // Image uploads and editor mutations can move the caret with the sticker
      // panel open; the cached selection must follow the live document.
      if (_stickerOpen && controller.selection.isValid) {
        _stickerSelection = controller.selection;
      }
    });
    _documentChanges = controller.document.changes.listen((DocChange change) {
      if (_pickInsertAt != null) {
        _pickInsertAt = change.change.transformPosition(_pickInsertAt!);
      }
      _mediaInsertAts.updateAll(
        (_, offset) => change.change.transformPosition(offset),
      );
      _handleEditorChanged();
    });
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
    _markDirty();
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
    _stickerOpen = false;
    _quill.skipRequestKeyboard = false;
    final revision = ++_modeRevision;
    if (mode == _ComposeMode.preview) {
      _editScrollOffset = _pageScrollController.hasClients
          ? _pageScrollController.offset
          : null;
      _restoreEditorFocus = _contentType == 3 && _editorFocusNode.hasFocus;
    }
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
    if (mode == _ComposeMode.edit) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted ||
            !_sessionCurrent ||
            revision != _modeRevision ||
            _mode != _ComposeMode.edit) {
          return;
        }
        final offset = _editScrollOffset;
        if (offset != null && _pageScrollController.hasClients) {
          _pageScrollController.jumpTo(
            offset.clamp(0.0, _pageScrollController.position.maxScrollExtent),
          );
        }
        if (_restoreEditorFocus && _contentType == 3) {
          _editorFocusNode.requestFocus();
        }
      });
    }
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
      if (!mounted || !_sessionCurrent) return;

      final String payloadTitle = props.topic.title.trim().isNotEmpty
          ? props.topic.title
          : (widget.editTitle ?? '');
      final String payloadContent = props.topic.content.trim().isNotEmpty
          ? props.topic.content
          : (widget.editContent ?? '');
      final List<int> payloadCategoryIds = props.topic.categoryIds.isNotEmpty
          ? props.topic.categoryIds
          : (widget.editCategoryIds ?? const <int>[]);

      final existingImages = props.topic.images;
      if (_owner == null &&
          payload.layout.viewer.isAuthenticated &&
          payload.layout.viewer.id > 0) {
        _owner = writingScope(
          Uri.parse(ref.read(apiClientProvider).baseUrl).origin,
          payload.layout.viewer.id,
        );
      }
      if (props.isEditing) {
        _draftKind = props.topic.topicStatus == 0
            ? DraftKind.serverDraft
            : DraftKind.topicEdit;
        if (widget.localDraftKey == null && !_localRestored) {
          _draftKey = topicDraftKey(
            props.topicId,
            published: props.topic.topicStatus != 0,
          );
        }
      }
      final keepEditing = _dirty;
      if (!keepEditing) {
        _identity = props.topic.identity;
        _contentType = props.isEditing
            ? (props.topic.contentType == 0 ? 3 : props.topic.contentType)
            : widget.initialContentType;
        _simple.text = payloadContent;
        _images
          ..clear()
          ..addAll(existingImages);
        _replaceEditorDocument(_contentType == 3 ? payloadContent : '');
        _dirty = false;
      }
      setState(() {
        _currentTopicId = props.topicId > 0 ? props.topicId : _currentTopicId;
        _categories
          ..clear()
          ..addAll(props.categories);
        if (!keepEditing) {
          _categoryIds
            ..clear()
            ..addAll(payloadCategoryIds.take(_maxArticleCategories));
          _title.text = payloadTitle;
        }
      });
      try {
        await _restoreLocal();
      } catch (_) {
        /* Show the server editor if local storage cannot be read. */
      }
      if (mounted && _sessionCurrent) setState(() => _loading = false);
    } catch (error) {
      if (!mounted || !_sessionCurrent) return;
      try {
        await _restoreLocal();
      } catch (_) {
        /* Keep the load error retryable. */
      }
      if (!mounted || !_sessionCurrent) return;
      final AppLocalizations l10n = AppLocalizations.of(context);
      setState(() {
        _loading = false;
        _loadError = resolveErrorMessage(l10n, error);
      });
    }
  }

  @override
  void dispose() {
    // Capture text before disposing controllers; the write retains its owner.
    if (_dirty && !_finished && _sessionCurrent) {
      unawaited(_saveLocal(notify: false));
    }
    _autosave?.cancel();
    _uploads.dispose();
    WidgetsBinding.instance.removeObserver(this);
    _title.removeListener(_textChanged);
    _simple.removeListener(_textChanged);
    _previewDebounce?.cancel();
    _dragAutoscrollTimer?.cancel();
    _dropLineY.dispose();
    _pageScrollController.dispose();
    _editorScrollController.dispose();
    _simpleFocusNode.dispose();
    _editorFocusNode.dispose();
    _captchaCode.dispose();
    _title.dispose();
    _titleFocusNode.dispose();
    _simple.dispose();
    unawaited(_documentChanges.cancel());
    _quill.dispose();
    super.dispose();
  }

  Future<void> _goBack() async {
    if (_submitting) return;
    if (_stickerOpen) {
      FocusManager.instance.primaryFocus?.unfocus();
      _quill.skipRequestKeyboard = false;
      setState(() => _stickerOpen = false);
      return;
    }
    if (_uploading) {
      showGfToast(
        context,
        AppLocalizations.of(context).publishMediaPendingWarning,
      );
      return;
    }
    if (_mode == _ComposeMode.preview) {
      _selectMode(_ComposeMode.edit);
      return;
    }
    if (_dirty && !_allowPop) {
      final l10n = AppLocalizations.of(context);
      // One column of full-width choices, strongest first, within thumb
      // reach; dismissing the sheet keeps editing.
      final choice = await showGfBottomSheet<String>(
        context,
        builder: (context) {
          final type = GfTheme.typographyOf(context);
          return SafeArea(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(20, 4, 20, 16),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(l10n.publishLeaveTitle, style: type.title3),
                  const SizedBox(height: 6),
                  Text(
                    l10n.publishLeaveBody,
                    style: type.body.copyWith(
                      color: GfTheme.colorsOf(context).iconMuted,
                    ),
                  ),
                  const SizedBox(height: 20),
                  GfButton(
                    label: l10n.draftKeepAndLeave,
                    size: GfButtonSize.extraLarge,
                    expanded: true,
                    onPressed: () => Navigator.pop(context, 'save'),
                  ),
                  const SizedBox(height: 8),
                  GfButton(
                    label: l10n.publishDiscard,
                    variant: GfButtonVariant.secondary,
                    size: GfButtonSize.extraLarge,
                    expanded: true,
                    onPressed: () => Navigator.pop(context, 'discard'),
                  ),
                  const SizedBox(height: 4),
                  GfButton(
                    label: l10n.publishContinue,
                    variant: GfButtonVariant.ghost,
                    size: GfButtonSize.extraLarge,
                    expanded: true,
                    onPressed: () => Navigator.pop(context, 'continue'),
                  ),
                ],
              ),
            ),
          );
        },
      );
      if (!mounted || choice == null || choice == 'continue') return;
      if (choice == 'save') {
        if (_owner == null) {
          setState(() {
            _localSaveFailed = true;
            _localStatus = l10n.draftLocalSaveFailed;
          });
          return;
        }
        if (!await _saveLocal() || !mounted || !_sessionCurrent) return;
      } else {
        _autosave?.cancel();
        try {
          if (_owner != null) {
            await _localStore.delete(
              _owner!,
              _draftKey,
              isCurrent: () => _sessionCurrent,
            );
          }
        } catch (_) {
          if (mounted) {
            setState(() {
              _localSaveFailed = true;
              _localStatus = l10n.draftLocalSaveFailed;
            });
          }
          return;
        }
        if (!mounted || !_sessionCurrent) return;
      }
      _finished = true;
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
    if (_uploading || value == null || value == _contentType) return;
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
      _stickerOpen = false;
      _stickerSelection = null;
      _quill.skipRequestKeyboard = false;
      _contentType = value;
      // Short types hold a single category; keep the first pick.
      if (_categoryIds.length > _categoryLimit) {
        _categoryIds.removeRange(_categoryLimit, _categoryIds.length);
      }
      _simple.text = markdown;
      _previewMarkdown = _markdownFromEditor();
      _markDirty();
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

  /// 长按标题按钮弹出级别菜单;选中当前级别再次确认可取消标题。
  Future<void> _showHeadingLevelMenu(AppLocalizations l10n) async {
    final int? currentHeader =
        _quill.getSelectionStyle().attributes[Attribute.header.key]?.value
            as int?;
    final Attribute? selected = await showGfBottomSheet<Attribute>(
      context,
      keyboardAware: true,
      builder: (sheetContext) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          padding: EdgeInsets.zero,
          children: <Widget>[
            for (final (attribute, label, level) in <(Attribute, String, int)>[
              (Attribute.h1, l10n.publishHeadingLevel1, 1),
              (Attribute.h2, l10n.publishHeadingLevel2, 2),
              (Attribute.h3, l10n.publishHeadingLevel3, 3),
            ])
              ListTile(
                title: Text(label),
                trailing: currentHeader == level
                    ? const GfSymbol('check', size: 22)
                    : null,
                onTap: () => Navigator.pop(sheetContext, attribute),
              ),
          ],
        ),
      ),
    );
    if (selected == null || !mounted) return;
    _toggleFormat(selected);
  }

  void _toggleCategory(PublishCategoryPayload category, bool selected) {
    setState(() {
      _markDirty();
      _allowPop = false;
      _error = '';
      _message = '';
      if (!selected) {
        _categoryIds.remove(category.id);
        return;
      }
      if (_categoryLimit == 1) {
        _categoryIds
          ..clear()
          ..add(category.id);
        return;
      }
      if (_categoryIds.length < _categoryLimit &&
          !_categoryIds.contains(category.id)) {
        _categoryIds.add(category.id);
      }
    });
  }

  /// Moves an article image to another position from the preview step; one
  /// composed Delta, so a single undo restores the previous order.
  void _reorderArticleImages(int from, int to) {
    final delta = reorderDocumentImages(_quill.document, from, to);
    if (delta == null) return;
    _quill.compose(delta, _quill.selection, ChangeSource.local);
    _previewDebounce?.cancel();
    _refreshPreview();
    HapticFeedback.selectionClick();
    unawaited(_saveLocal());
  }

  Future<void> _pickAndInsertImage() async {
    // Pending uploads keep running; only a second picker is refused.
    if (_pickingImage || _submitting || !_sessionCurrent || _finished) return;
    final l10n = AppLocalizations.of(context);
    final remaining = _contentType == 3
        ? 9
        : 9 - _images.length - _uploads.items.length;
    if (remaining <= 0) {
      showGfToast(context, l10n.publishGalleryTooMany, error: true);
      return;
    }
    _pickInsertAt = _contentType == 3
        ? _quill.selection.baseOffset.clamp(0, _quill.document.length - 1)
        : null;
    setState(() => _pickingImage = true);
    try {
      final source = await showGfBottomSheet<ImageSource>(
        context,
        keyboardAware: true,
        builder: (context) => SafeArea(
          child: ListView(
            shrinkWrap: true,
            padding: EdgeInsets.zero,
            children: [
              ListTile(
                leading: const GfSymbol('image', size: 23),
                title: Text(l10n.publishPhotoLibrary),
                onTap: () => Navigator.pop(context, ImageSource.gallery),
              ),
              ListTile(
                leading: const GfSymbol('camera', size: 23),
                title: Text(l10n.publishCamera),
                onTap: () => Navigator.pop(context, ImageSource.camera),
              ),
            ],
          ),
        ),
      );
      if (source == null || !mounted || !_sessionCurrent || _finished) return;
      final picker = ref.read(imagePickerProvider);
      final List<XFile> selected;
      if (source == ImageSource.gallery && remaining > 1) {
        selected = await picker.pickMultiImage(
          maxWidth: 2048,
          imageQuality: 85,
          limit: remaining,
          requestFullMetadata: false,
        );
      } else {
        final photo = await picker.pickImage(
          source: source,
          maxWidth: 2048,
          imageQuality: 85,
          requestFullMetadata: false,
        );
        selected = [?photo];
      }
      if (!mounted || !_sessionCurrent || _finished) return;
      // Some native pickers ignore limit. Reject the batch without silently
      // dropping selected photos or exceeding the gallery contract.
      if (selected.length > remaining) {
        showGfToast(context, l10n.publishGalleryTooMany, error: true);
        return;
      }
      final ids = _uploads.add(selected);
      if (_pickInsertAt case final at?) {
        for (final id in ids) {
          _mediaInsertAts[id] = at;
        }
      }
    } catch (error) {
      if (mounted && _sessionCurrent) {
        final AppLocalizations l10n = AppLocalizations.of(context);
        showGfToast(
          context,
          l10n.publishImageFailed(
            error is ApiException
                ? resolveErrorMessage(l10n, error)
                : l10n.commonLoadFailed,
          ),
          error: true,
        );
      }
    } finally {
      _pickInsertAt = null;
      if (mounted) setState(() => _pickingImage = false);
    }
  }

  void _bodyFocusChanged() {
    if (_stickerOpen &&
        (_simpleFocusNode.hasFocus ||
            _editorFocusNode.hasFocus ||
            _titleFocusNode.hasFocus)) {
      _quill.skipRequestKeyboard = false;
      setState(() => _stickerOpen = false);
    }
  }

  void _keepEditorVisible() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final editorContext = _bodyEditorContainerKey.currentContext;
      if (mounted && _stickerOpen && editorContext != null) {
        Scrollable.ensureVisible(editorContext, alignment: 0);
      }
    });
  }

  void _pickSticker() {
    if (!_stickerOpen) {
      _stickerSelection = _contentType == 3
          ? _quill.selection
          : _simple.selection;
    }
    FocusManager.instance.primaryFocus?.unfocus();
    if (_stickerOpen) {
      _quill.skipRequestKeyboard = false;
      setState(() => _stickerOpen = false);
      (_contentType == 3 ? _editorFocusNode : _simpleFocusNode).requestFocus();
    } else {
      _quill.skipRequestKeyboard = true;
      setState(() => _stickerOpen = true);
      _keepEditorVisible();
    }
  }

  void _insertSticker(String token) {
    if (!_sessionCurrent || _finished || _submitting) return;
    if (_contentType == 3) {
      final max = _quill.document.length - 1;
      final range = _stickerSelection ?? _quill.selection;
      final start = (range.isValid ? range.start : max).clamp(0, max);
      final end = (range.isValid ? range.end : max).clamp(start, max);
      _quill.replaceText(
        start,
        end - start,
        token,
        TextSelection.collapsed(offset: start + token.length),
        // skipRequestKeyboard is consumed by Quill's first request. Every
        // insertion must preserve accessory/search focus, including on Android.
        ignoreFocus: true,
      );
      _stickerSelection = _quill.selection;
    } else {
      insertStickerText(_simple, token, selection: _stickerSelection);
      _stickerSelection = _simple.selection;
    }
    _keepEditorVisible();
  }

  void _insertUploadedImage(String url, int id) {
    if (!mounted || !_sessionCurrent || _finished) return;
    if (_contentType != 3) {
      setState(() => _images.add(url));
      _markDirty();
    } else {
      final insertAt =
          (_mediaInsertAts.remove(id) ?? _quill.document.length - 1).clamp(
            0,
            _quill.document.length - 1,
          );
      final selection = _quill.selection;
      final change = _quill.document.insert(insertAt, BlockEmbed.image(url));
      _quill.updateSelection(
        TextSelection(
          baseOffset: change.transformPosition(
            selection.baseOffset.clamp(0, _quill.document.length - 1),
          ),
          extentOffset: change.transformPosition(
            selection.extentOffset.clamp(0, _quill.document.length - 1),
          ),
        ),
        ChangeSource.local,
      );
      _markDirty();
    }
    _allowPop = false;
    unawaited(_saveLocal());
  }

  void _startDragAutoscroll() {
    HapticFeedback.selectionClick();
    _dragPointer = null;
    _dragAutoscrollTimer?.cancel();
    _dragAutoscrollTimer = Timer.periodic(
      const Duration(milliseconds: 50),
      (_) => _tickDragAutoscroll(),
    );
  }

  void _stopDragAutoscroll() {
    _dragPointer = null;
    _dragAutoscrollTimer?.cancel();
    _dragAutoscrollTimer = null;
    _dropOffset = null;
    _dropLineY.value = null;
  }

  /// Tracks the paragraph under a dragged image and marks the slot below it,
  /// with a light tick each time the slot changes.
  void _updateDropLine(Offset pointer) {
    _dragPointer = pointer;
    final EditorState? editorState = _editorKey.currentState;
    final RenderObject? layer = _editorDropKey.currentContext
        ?.findRenderObject();
    if (editorState == null || layer is! RenderBox || !layer.hasSize) return;
    final RenderEditor editor = editorState.renderEditor;
    final int offset = clampDropToLineEnd(
      editor.getPositionForOffset(pointer).offset,
      _quill.document,
    ).clamp(0, _quill.document.length - 1);
    if (_dropOffset != null && _dropOffset != offset) {
      HapticFeedback.selectionClick();
    }
    _dropOffset = offset;
    final Rect caret = editor.getLocalRectForCaret(
      TextPosition(offset: offset),
    );
    _dropLineY.value = layer
        .globalToLocal(editor.localToGlobal(caret.bottomLeft))
        .dy;
  }

  void _tickDragAutoscroll() {
    final Offset? pointer = _dragPointer;
    if (pointer == null) return;
    final ScrollController scroll = _pageScrollController;
    if (!scroll.hasClients) return;
    // Drag offsets are global; edge probes must be viewport-local.
    final RenderObject? renderObject = _pageScrollViewKey.currentContext
        ?.findRenderObject();
    if (renderObject is! RenderBox ||
        !renderObject.attached ||
        !renderObject.hasSize) {
      return;
    }
    final double localY = renderObject.globalToLocal(pointer).dy;
    final double maxOffset = scroll.position.maxScrollExtent;
    final double viewportBottom =
        scroll.position.viewportDimension - _dragAutoscrollEdge;
    // Speed grows with how deep the finger sits in the edge band, so a
    // light nudge creeps and a hard push travels.
    double speed(double depth) =>
        _dragAutoscrollStep * (depth / _dragAutoscrollEdge).clamp(.2, 1.0);
    if (localY < _dragAutoscrollEdge && scroll.offset > 0) {
      scroll.jumpTo(
        (scroll.offset - speed(_dragAutoscrollEdge - localY)).clamp(
          0.0,
          maxOffset,
        ),
      );
      _updateDropLine(pointer);
    } else if (localY > viewportBottom && scroll.offset < maxOffset) {
      scroll.jumpTo(
        (scroll.offset + speed(localY - viewportBottom)).clamp(0.0, maxOffset),
      );
      _updateDropLine(pointer);
    }
  }

  /// Moves the dragged composer image to the paragraph under the drop point.
  ///
  /// The embed is relocated with a single Delta so undo/redo stays one step;
  /// the document-changes listener flips _dirty/_allowPop and refreshes the
  /// debounced preview, exactly like any other edit.
  void _handleComposerImageDrop(
    ComposerImageDragPayload payload,
    Offset globalPosition,
  ) {
    final EditorState? editorState = _editorKey.currentState;
    if (editorState == null) return;
    final TextPosition position = editorState.renderEditor.getPositionForOffset(
      globalPosition,
    );
    final int targetOffset = clampDropToLineEnd(
      position.offset,
      _quill.document,
    );
    final ComposerImageMoveResult? move = moveComposerImageEmbed(
      _quill.document,
      payload.node,
      targetOffset,
    );
    if (move == null) return;
    _quill.compose(move.delta, _quill.selection, ChangeSource.local);
    _quill.updateSelection(
      TextSelection.collapsed(offset: move.insertOffset + move.insertLength),
      ChangeSource.local,
    );
    HapticFeedback.lightImpact();
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
    if (_uploading) {
      showGfToast(
        context,
        AppLocalizations.of(context).publishMediaPendingWarning,
      );
      return;
    }
    if (_submitting || _loading || !_sessionCurrent) return;
    if (_loadError.isNotEmpty) {
      if (topicStatus == 0 && _localRestored) {
        _markDirty();
        await _saveLocal();
      }
      return;
    }

    // 瞬间允许无标题：留空即空标题提交，不从正文首行/图片占位提取（issue #895）。
    final String title = _title.text.trim();
    final String content = _markdownFromEditor().isEmpty && _images.isNotEmpty
        ? AppLocalizations.of(context).publishImageOnlyTitle
        : _markdownFromEditor();
    final AppLocalizations l10n = AppLocalizations.of(context);

    String validationError = '';
    if (title.isEmpty && _contentType != 2) {
      validationError = l10n.publishTitleRequired;
    } else if (_categoryIds.isEmpty) {
      validationError = l10n.publishCategoryRequired;
    } else if (content.isEmpty) {
      validationError = l10n.publishContentRequired;
    }
    if (topicStatus == 0 && validationError.isNotEmpty) {
      _markDirty();
      await _saveLocal();
      return;
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
      final WriteTopicResult written = await ref
          .read(topicRepositoryProvider)
          .writeTopicResult(
            captchaId: _captcha?.captchaId,
            captchaCode: _captchaCode.text.trim(),
            identity: _currentTopicId > 0 ? null : _identity,
            topicId: _currentTopicId,
            title: title,
            content: content,
            categoryIds: List<int>.of(_categoryIds),
            topicStatus: topicStatus,
            agentRepliesDisabled: _agentRepliesDisabled,
            contentType: _contentType,
            images: _contentType == 3 ? null : List.of(_images),
          );
      final int id = written.id;
      if (!mounted || !_sessionCurrent) return;
      _autosave?.cancel();
      // The server write succeeded; deletion must follow any in-flight autosave.
      _finished = true;
      try {
        if (_owner != null) {
          await _localStore.delete(
            _owner!,
            _draftKey,
            isCurrent: () => _sessionCurrent,
          );
        }
      } catch (_) {
        /* Keep recovery data if deletion fails; server ack remains valid. */
      }
      if (!mounted || !_sessionCurrent) return;
      _dirty = false;
      _allowPop = true;
      final int resolvedId = id > 0 ? id : _currentTopicId;
      if (resolvedId > 0) _currentTopicId = resolvedId;
      // 待审(issue #975):明确提示“已提交审核,通过后公开”,与 Web 同语义。
      if (written.pendingReview) {
        showGfToast(
          context,
          pendingReviewMessage(l10n, checking: written.checking),
        );
      }
      if (topicStatus == 1 && resolvedId > 0) {
        context.pushReplacement('/p/$resolvedId');
        return;
      }
      _finished = false;
      _draftKey = topicDraftKey(_currentTopicId, published: topicStatus == 1);
      _draftKind = topicStatus == 1
          ? DraftKind.topicEdit
          : DraftKind.serverDraft;
      _localStatus = '';
      setState(() {
        _message = topicStatus == 1
            ? l10n.publishSuccess
            : l10n.publishSavedDraft;
        _showPublishCaptchaExplanation = false;
      });
    } on ApiException catch (error) {
      if (!mounted || !_sessionCurrent) return;
      if (mounted && _sessionCurrent) {
        setState(() => _error = resolveErrorMessage(l10n, error));
      }
      // AI 图文审查拦截(issue #975):弹出友好提示,草稿与图片保持不变。
      if (await showModerationBlockedDialog(context, error)) return;
      if (error.messageCode == 'common.captchaRequired' ||
          error.messageCode == 'auth.captcha.invalid') {
        if (error.messageCode == 'common.captchaRequired') {
          setState(
            () => _showPublishCaptchaExplanation =
                error.params?['action'] == 'topic.write',
          );
        }
        await _loadCaptcha();
      }
    } catch (error) {
      if (mounted && _sessionCurrent) {
        setState(() => _error = l10n.publishFailed('$error'));
      }
    } finally {
      if (mounted && _sessionCurrent) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (ref.watch(offlineCacheEpochProvider) != _epoch) {
      return const SizedBox.shrink();
    }
    final AppLocalizations l10n = AppLocalizations.of(context);

    return PopScope(
      canPop:
          !_stickerOpen &&
          !_submitting &&
          !_uploading &&
          (_allowPop || (!_dirty && _mode == _ComposeMode.edit)),
      onPopInvokedWithResult: (didPop, result) {
        if (!didPop) _goBack();
      },
      child: Scaffold(
        backgroundColor: GfTheme.colorsOf(context).base100,
        appBar: GfAppBar(
          bottom: _loading || _loadError.isNotEmpty && !_localRestored
              ? null
              : _StepProgress(
                  step: _mode == _ComposeMode.edit ? 1 : 2,
                  label: l10n.publishStepOf(
                    _mode == _ComposeMode.edit ? 1 : 2,
                    2,
                  ),
                ),
          leading: GfIconButton(
            symbol: 'chevron-left',
            tooltip: l10n.commonBack,
            size: 44,
            onPressed: _goBack,
          ),
          // The type switcher lives above the fields; the bar names the step
          // only when that switcher is out of view.
          title: _showTypeSwitcher(context)
              ? const SizedBox.shrink()
              : Text(
                  _mode == _ComposeMode.preview
                      ? l10n.composePreview
                      : PublishType.fromValue(_contentType).label(l10n),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: GfTheme.typographyOf(context).bodyStrong,
                ),
          actions: <Widget>[
            if (MediaQuery.viewInsetsOf(context).bottom > 0)
              GfIconButton(
                symbol: 'keyboard-hide',
                tooltip: l10n.commonHideKeyboard,
                size: 44,
                onPressed: () => FocusManager.instance.primaryFocus?.unfocus(),
              ),
            if (!_loading && _loadError.isEmpty)
              GfIconButton(
                key: const Key('publish-save-draft'),
                // One icon action in both steps keeps the bar inside long
                // locale labels (de/ja) and frees the writing row for tools.
                symbol: 'save',
                tooltip: l10n.publishSaveDraft,
                size: 44,
                onPressed: _uploading || _submitting
                    ? null
                    : () => _submit(topicStatus: 0),
              ),
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: _buildSubmitAction(l10n),
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
                  child: ConstrainedBox(
                    constraints: BoxConstraints(
                      maxHeight: _stickerOpen
                          ? ((MediaQuery.sizeOf(context).height -
                                        MediaQuery.viewInsetsOf(
                                          context,
                                        ).bottom -
                                        MediaQuery.viewPaddingOf(
                                          context,
                                        ).vertical -
                                        kToolbarHeight -
                                        _StepProgress.height) *
                                    .65)
                                .clamp(0.0, double.infinity)
                          : double.infinity,
                    ),
                    child: AbsorbPointer(
                      absorbing: _submitting,
                      child: _buildWritingToolbar(l10n),
                    ),
                  ),
                ),
              )
            : null,
      ),
    );
  }

  bool _showTypeSwitcher(BuildContext context) =>
      _mode == _ComposeMode.edit &&
      _currentTopicId == 0 &&
      !_loading &&
      !_stickerOpen &&
      MediaQuery.viewInsetsOf(context).bottom == 0;

  Widget _buildSubmitAction(AppLocalizations l10n) {
    final editing = _mode == _ComposeMode.edit;
    final label = editing ? l10n.publishNext : l10n.publishPublish;
    final text = TextPainter(
      text: TextSpan(
        text: label,
        style: (Theme.of(context).textTheme.labelLarge ?? const TextStyle())
            .copyWith(fontSize: 14, fontWeight: FontWeight.w600),
      ),
      textDirection: Directionality.of(context),
      textScaler: MediaQuery.textScalerOf(context),
      maxLines: 1,
    )..layout();
    // Reserve back, a short type label, spacing, and the optional icon actions.
    final available =
        MediaQuery.sizeOf(context).width -
        56 -
        40 -
        16 -
        (editing ? 0 : 44) -
        (MediaQuery.viewInsetsOf(context).bottom > 0 ? 44 : 0);
    final compact = text.width + 24 > available || text.height > 32;
    text.dispose();
    final enabled =
        !_loading &&
        !_uploading &&
        !_submitting &&
        ((editing && _localRestored) || _loadError.isEmpty);
    void submit() =>
        editing ? _selectMode(_ComposeMode.preview) : _submit(topicStatus: 1);
    if (compact) {
      return IconButton.filled(
        key: const Key('publish-appbar-submit'),
        tooltip: label,
        onPressed: enabled ? submit : null,
        icon: _submitting
            ? const SizedBox.square(
                dimension: 20,
                child: GfProgressIndicator(strokeWidth: 2),
              )
            : GfSymbol(editing ? 'arrow-right' : 'arrow-up', size: 20),
      );
    }
    return GfButton(
      key: const Key('publish-appbar-submit'),
      label: label,
      variant: GfButtonVariant.primary,
      size: GfButtonSize.small,
      loading: _submitting,
      onPressed: enabled ? submit : null,
    );
  }

  Widget _buildBody(AppLocalizations l10n) {
    if (_loading) return const _PublishWorkspaceSkeleton();
    if (_loadError.isNotEmpty && !_localRestored) {
      return GfErrorRetry(message: _loadError, onRetry: _loadEditorData);
    }

    // Scaffold consumes viewInsets for its resized body; read them from the
    // page context before entering that body's LayoutBuilder.
    final keyboardOpen = MediaQuery.viewInsetsOf(context).bottom > 0;
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final bool wide = constraints.maxWidth >= _wideWorkspaceBreakpoint;
        final typing =
            _mode == _ComposeMode.edit && (keyboardOpen || _stickerOpen);
        final articleImages = _mode == _ComposeMode.preview && _contentType == 3
            ? documentImageUrls(_quill.document)
            : const <String>[];
        final EdgeInsets pagePadding = EdgeInsets.symmetric(
          horizontal: wide ? 24 : 20,
          vertical: typing ? 8 : 20,
        );

        return SingleChildScrollView(
          key: _pageScrollViewKey,
          controller: _pageScrollController,
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
                    if (_localStatus.isNotEmpty) ...[
                      Row(
                        children: [
                          _buildSaveStatusMark(l10n),
                          const SizedBox(width: 6),
                          Expanded(
                            child: Text(
                              _localStatus,
                              style: GfTheme.typographyOf(context).caption
                                  .copyWith(
                                    color: _localSaveFailed
                                        ? GfTheme.colorsOf(context).error
                                        : GfTheme.colorsOf(context).iconMuted,
                                  ),
                            ),
                          ),
                          if (_localSaveFailed)
                            TextButton(
                              onPressed: _saveLocal,
                              child: Text(l10n.commonRetry),
                            ),
                        ],
                      ),
                      const SizedBox(height: 8),
                    ],
                    if (_loadError.isNotEmpty)
                      TextButton.icon(
                        onPressed: _loadEditorData,
                        icon: const GfSymbol('refresh-cw', size: 22),
                        label: Text(l10n.commonRetry),
                      ),
                    if (_showTypeSwitcher(this.context)) ...[
                      PublishTypeSwitcher(
                        value: _contentType,
                        onChanged: _submitting || _uploading
                            ? null
                            : _changeContentType,
                      ),
                      const SizedBox(height: 20),
                    ],
                    // Moments lead with their photos; questions attach them
                    // after the body (below).
                    if (_mode == _ComposeMode.edit &&
                        _contentType == 2 &&
                        (!typing ||
                            _images.isNotEmpty ||
                            _uploads.hasPending)) ...[
                      _buildGallery(l10n, editing: true),
                      const SizedBox(height: 12),
                    ],
                    if (_mode == _ComposeMode.edit || wide) ...[
                      KeyedSubtree(
                        key: const ValueKey('publish-title-fields'),
                        child: _buildTopicFields(l10n),
                      ),
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
                      _StepEntrance(child: _buildPreview(l10n)),
                    if (_mode == _ComposeMode.edit &&
                        _contentType == 1 &&
                        (!typing ||
                            _images.isNotEmpty ||
                            _uploads.hasPending)) ...[
                      const SizedBox(height: 12),
                      _buildGallery(l10n, editing: true),
                    ],
                    if (articleImages.length > 1) ...[
                      const SizedBox(height: 12),
                      _StepEntrance(
                        delay: const Duration(milliseconds: 30),
                        child: _buildArticleImageOrder(l10n, articleImages),
                      ),
                    ],
                    if (_mode == _ComposeMode.preview) ...[
                      const SizedBox(height: 16),
                      _StepEntrance(
                        delay: const Duration(milliseconds: 60),
                        child: _buildPublishSettings(l10n),
                      ),
                      const SizedBox(height: 16),
                    ],
                    if (_captcha != null) ...[
                      if (_showPublishCaptchaExplanation)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 12),
                          child: Text(
                            l10n.publishCaptchaExplanation,
                            style: Theme.of(context).textTheme.bodySmall,
                          ),
                        ),
                      Row(
                        children: [
                          InkWell(
                            onTap: _captchaLoading ? null : _loadCaptcha,
                            child: GfCaptchaImage(
                              imageData: _captcha!.captchaImg,
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
                            icon: const GfSymbol('refresh-cw', size: 22),
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
        ...[
          TextField(
            key: const Key('publish-title'),
            controller: _title,
            focusNode: _titleFocusNode,
            maxLength: _titleMaxLength,
            minLines: 1,
            maxLines: 3,
            decoration: InputDecoration(
              // Moments may stay untitled; the hint says so instead of
              // hiding the field behind an extra tap.
              hintText: _contentType == 2
                  ? l10n.publishMomentTitleHint
                  : l10n.publishTitleField,
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
              fontSize: 24,
              fontWeight: FontWeight.w700,
            ),
            onChanged: (_) {
              _markDirty();
              _allowPop = false;
              if (_error.isNotEmpty || _message.isNotEmpty) {
                setState(() {
                  _error = '';
                  _message = '';
                });
              }
            },
          ),
          _buildTitleRule(),
        ],
      ],
    );
  }

  static const int _titleMaxLength = 100;

  /// Small mark before the local save status: a clock while writing, a
  /// check once stored, history after a restore and an alert on failure.
  Widget _buildSaveStatusMark(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    final (String symbol, Color color) = _localSaveFailed
        ? ('circle-alert', colors.error)
        : _localStatus == l10n.draftLocalSaving
        ? ('clock', colors.iconMuted)
        : _localStatus == l10n.draftLocalRestored
        ? ('history', colors.iconMuted)
        : ('check', colors.iconMuted);
    return GfSymbol(symbol, size: 14, color: color);
  }

  /// Hairline between title and body that takes the accent while the title
  /// is focused, with the length counter shown only while typing in it.
  Widget _buildTitleRule() {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final duration = GfMotion.duration(context, GfMotion.selection);
    return ListenableBuilder(
      listenable: Listenable.merge([_titleFocusNode, _title]),
      builder: (context, _) {
        final focused = _titleFocusNode.hasFocus;
        // The counter takes room only while the title is being written, so
        // short screens keep every line for the body.
        return AnimatedSize(
          duration: duration,
          alignment: AlignmentDirectional.topStart,
          child: Row(
            children: [
              Expanded(
                child: AnimatedContainer(
                  duration: duration,
                  height: 1,
                  color: focused
                      ? colors.primary.withValues(alpha: .55)
                      : colors.line,
                ),
              ),
              if (focused)
                Padding(
                  padding: const EdgeInsetsDirectional.only(start: 10),
                  child: Text(
                    '${_title.text.characters.length}/$_titleMaxLength',
                    style: type.caption.copyWith(
                      color: colors.iconMuted,
                      fontFeatures: const [FontFeature.tabularFigures()],
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }

  /// Preview step settings in one quiet group: where the post goes, who
  /// publishes it, and whether bots may reply. Each row leads with its own
  /// symbol so the group scans as a list of decisions.
  Widget _buildPublishSettings(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    return Material(
      key: const Key('publish-settings'),
      color: colors.base200.withValues(alpha: .6),
      borderRadius: BorderRadius.circular(20),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _buildCategoryMenu(l10n),
          const GfDivider(inset: 16),
          _buildIdentityRow(l10n),
          if (_currentTopicId == 0) ...[
            const GfDivider(inset: 16),
            IgnorePointer(
              ignoring: _submitting,
              child: GfSwitchRow(
                key: const Key('publish-agent-replies'),
                symbol: 'message-circle',
                iconColor: colors.iconMuted,
                title: l10n.agentRepliesAllow,
                description: l10n.agentRepliesHelp,
                value: !_agentRepliesDisabled,
                onChanged: (value) {
                  setState(() => _agentRepliesDisabled = !value);
                  _markDirty();
                },
              ),
            ),
          ],
        ],
      ),
    );
  }

  /// Category choice as a dropdown the width of its row. Moments and
  /// questions take one category and the menu closes on pick; articles take
  /// up to three with the menu left open. Each category shows its colour as a
  /// soft swatch so the list scans by colour before name.
  Widget _buildCategoryMenu(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final duration = GfMotion.duration(context, GfMotion.selection);
    final limit = _categoryLimit;
    final multi = limit > 1;
    final picked = [
      for (final id in _categoryIds)
        ?_categories.where((c) => c.id == id).firstOrNull,
    ];
    final full = _categoryIds.length >= limit;
    // Dark surfaces read elevation from lightness, not shadow.
    final menuColor = Theme.of(context).brightness == Brightness.dark
        ? Color.alphaBlend(
            colors.baseContent.withValues(alpha: .06),
            colors.base100,
          )
        : colors.base100;
    Widget swatch(String hex) {
      final color = colorFromHex(hex);
      return Container(
        width: 28,
        height: 28,
        decoration: BoxDecoration(
          color: color.withValues(alpha: .14),
          borderRadius: BorderRadius.circular(9),
        ),
        alignment: Alignment.center,
        child: Container(
          width: 10,
          height: 10,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
      );
    }

    return LayoutBuilder(
      builder: (context, constraints) {
        // The menu hangs 8dp inside the settings card on both sides.
        final menuWidth = constraints.maxWidth - 16;
        return MenuAnchor(
          controller: _categoryMenu,
          onOpen: () => setState(() => _categoryMenuOpen = true),
          onClose: () => setState(() => _categoryMenuOpen = false),
          alignmentOffset: const Offset(8, 4),
          style: MenuStyle(
            backgroundColor: WidgetStatePropertyAll(menuColor),
            surfaceTintColor: const WidgetStatePropertyAll(Colors.transparent),
            elevation: const WidgetStatePropertyAll(10),
            shadowColor: WidgetStatePropertyAll(
              Colors.black.withValues(alpha: .14),
            ),
            shape: WidgetStatePropertyAll(
              RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(18),
                side: BorderSide(color: colors.line.withValues(alpha: .6)),
              ),
            ),
            padding: const WidgetStatePropertyAll(EdgeInsets.all(6)),
            minimumSize: WidgetStatePropertyAll(Size(menuWidth, 0)),
            maximumSize: WidgetStatePropertyAll(Size(menuWidth, 380)),
          ),
          menuChildren: [
            if (multi)
              Padding(
                padding: const EdgeInsets.fromLTRB(12, 8, 12, 6),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        l10n.publishCategoryLimit(limit),
                        style: type.caption.copyWith(color: colors.iconMuted),
                      ),
                    ),
                    Text(
                      '${_categoryIds.length}/$limit',
                      style: type.caption.copyWith(
                        color: _categoryIds.isEmpty
                            ? colors.iconMuted
                            : colors.primary,
                        fontFeatures: const [FontFeature.tabularFigures()],
                      ),
                    ),
                  ],
                ),
              ),
            for (final category in _categories)
              Builder(
                builder: (context) {
                  final selected = _categoryIds.contains(category.id);
                  return MenuItemButton(
                    key: ValueKey('publish-category-${category.id}'),
                    closeOnActivate: !multi,
                    onPressed: multi && !selected && full
                        ? null
                        : () => _toggleCategory(category, !multi || !selected),
                    leadingIcon: swatch(category.color),
                    trailingIcon: AnimatedScale(
                      scale: selected ? 1 : .25,
                      duration: duration,
                      curve: GfMotion.enterCurve,
                      child: AnimatedOpacity(
                        opacity: selected ? 1 : 0,
                        duration: duration,
                        child: GfSymbol(
                          'check',
                          size: 18,
                          color: colors.primary,
                        ),
                      ),
                    ),
                    style: MenuItemButton.styleFrom(
                      minimumSize: Size(menuWidth - 12, 52),
                      padding: const EdgeInsets.symmetric(horizontal: 10),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      backgroundColor: selected
                          ? colors.primary.withValues(alpha: .08)
                          : Colors.transparent,
                      foregroundColor: colors.baseContent,
                      disabledForegroundColor: colors.baseContent.withValues(
                        alpha: .38,
                      ),
                      textStyle: type.body.copyWith(
                        fontWeight: selected
                            ? FontWeight.w600
                            : FontWeight.w400,
                      ),
                    ),
                    child: Semantics(
                      selected: selected,
                      child: Text(
                        category.name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  );
                },
              ),
          ],
          builder: (context, controller, _) => Semantics(
            expanded: _categoryMenuOpen,
            child: GfSettingRow(
              key: const Key('publish-category-menu'),
              symbol: 'folder',
              iconColor: colors.iconMuted,
              title: l10n.publishCategoryLabel,
              subtitleWidget: picked.isEmpty
                  ? Text(
                      multi
                          ? l10n.publishClassification
                          : l10n.publishCategoryPickOne,
                    )
                  : Padding(
                      padding: const EdgeInsets.only(top: 4),
                      child: Wrap(
                        spacing: 6,
                        runSpacing: 6,
                        children: [
                          for (final category in picked)
                            Container(
                              padding: const EdgeInsetsDirectional.fromSTEB(
                                8,
                                3,
                                10,
                                3,
                              ),
                              decoration: BoxDecoration(
                                color: colors.base100,
                                borderRadius: BorderRadius.circular(999),
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Container(
                                    width: 7,
                                    height: 7,
                                    decoration: BoxDecoration(
                                      color: colorFromHex(category.color),
                                      shape: BoxShape.circle,
                                    ),
                                  ),
                                  const SizedBox(width: 6),
                                  Text(
                                    category.name,
                                    style: type.small.copyWith(
                                      color: colors.baseContent.withValues(
                                        alpha: .85,
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                        ],
                      ),
                    ),
              trailing: AnimatedRotation(
                turns: _categoryMenuOpen ? .5 : 0,
                duration: duration,
                child: GfSymbol(
                  'chevron-down',
                  size: 18,
                  color: colors.iconMuted,
                ),
              ),
              onTap: _submitting
                  ? null
                  : () => controller.isOpen
                        ? controller.close()
                        : controller.open(),
            ),
          ),
        );
      },
    );
  }

  /// Article images in reading order, reorderable from the preview step. A
  /// long-press drag changes which image fills each slot of the body; the
  /// numbers match the order readers will meet them.
  Widget _buildArticleImageOrder(AppLocalizations l10n, List<String> urls) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    const double tile = 64;
    final cacheWidth = (tile * 2 * MediaQuery.devicePixelRatioOf(context))
        .round();
    return Material(
      key: const Key('publish-image-order'),
      color: colors.base200.withValues(alpha: .6),
      borderRadius: BorderRadius.circular(20),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SizedBox(
                  width: 24,
                  child: GfSymbol('images', size: 22, color: colors.iconMuted),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.publishImageOrderTitle,
                        style: const TextStyle(fontSize: 16, height: 1.35),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        l10n.publishImageOrderHint,
                        style: type.caption.copyWith(color: colors.iconMuted),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          SizedBox(
            height: tile,
            child: ReorderableListView.builder(
              key: const Key('publish-image-order-strip'),
              scrollDirection: Axis.horizontal,
              // The strip starts on the text edge and runs to the card's
              // edge, so a longer row peeks past it.
              padding: const EdgeInsetsDirectional.only(
                start: 54,
                end: 8,
              ).resolve(Directionality.of(context)),
              itemCount: urls.length,
              proxyDecorator: (child, _, _) => Material(
                color: Colors.transparent,
                elevation: 6,
                borderRadius: BorderRadius.circular(10),
                child: child,
              ),
              onReorderItem: _reorderArticleImages,
              itemBuilder: (context, index) => Padding(
                key: ValueKey('publish-image-order-$index:${urls[index]}'),
                padding: const EdgeInsetsDirectional.only(end: 8),
                child: Semantics(
                  label: l10n.publishImageOrderPosition(index + 1),
                  image: true,
                  child: SizedBox.square(
                    dimension: tile,
                    child: Stack(
                      fit: StackFit.expand,
                      children: [
                        ClipRRect(
                          borderRadius: BorderRadius.circular(10),
                          child: ColoredBox(
                            color: colors.base200,
                            child: GfNetworkImage(
                              resolveApiAssetUrl(urls[index]),
                              fit: BoxFit.cover,
                              cacheWidth: cacheWidth,
                              errorBuilder: (_, _, _) => Center(
                                child: GfSymbol(
                                  'image-off',
                                  size: 20,
                                  color: colors.iconMuted,
                                ),
                              ),
                            ),
                          ),
                        ),
                        PositionedDirectional(
                          start: 4,
                          bottom: 4,
                          child: ExcludeSemantics(
                            child: Container(
                              constraints: const BoxConstraints(minWidth: 18),
                              height: 18,
                              padding: const EdgeInsets.symmetric(
                                horizontal: 5,
                              ),
                              alignment: Alignment.center,
                              decoration: BoxDecoration(
                                color: Colors.black.withValues(alpha: .55),
                                borderRadius: BorderRadius.circular(9),
                              ),
                              child: Text(
                                '${index + 1}',
                                style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 11,
                                  height: 1,
                                  fontWeight: FontWeight.w600,
                                  fontFeatures: [FontFeature.tabularFigures()],
                                ),
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(height: 14),
        ],
      ),
    );
  }

  /// "Post as" row; the picker's own pill inset is pulled back so its avatar
  /// shares the text edge of the other rows.
  Widget _buildIdentityRow(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    final rtl = Directionality.of(context) == TextDirection.rtl;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 24,
            child: GfSymbol('user-round', size: 22, color: colors.iconMuted),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.anonymousPublishAs,
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.35,
                    color: colors.baseContent,
                  ),
                ),
                Transform.translate(
                  offset: Offset(rtl ? 6 : -6, 0),
                  child: IdentityPicker(
                    value: _identity,
                    disabled: _submitting || _currentTopicId > 0,
                    onChanged: (v) {
                      setState(() {
                        _identity = v;
                        _markDirty();
                      });
                    },
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBodyHeader(AppLocalizations l10n) => Text(
    l10n.publishBodyField,
    style: GfTheme.typographyOf(
      context,
    ).small.copyWith(color: GfTheme.colorsOf(context).iconMuted),
  );

  Widget _buildEditor(AppLocalizations l10n) =>
      KeyedSubtree(key: _bodyEditorContainerKey, child: _buildBodyEditor(l10n));

  Widget _buildBodyEditor(AppLocalizations l10n) {
    if (_contentType != 3) {
      return TextField(
        key: const Key('publish-editor'),
        controller: _simple,
        focusNode: _simpleFocusNode,
        keyboardType: TextInputType.multiline,
        textInputAction: TextInputAction.newline,
        style: readingBodyStyle(context),
        decoration: InputDecoration(
          hintText: PublishType.fromValue(_contentType).hint(l10n),
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
    final defaults = DefaultStyles.getInstance(context);
    return RichMarkdownEditor(
      key: const Key('publish-editor'),
      controller: _quill,
      focusNode: _editorFocusNode,
      placeholder: PublishType.fromValue(_contentType).hint(l10n),
      onHeading: () => _toggleFormat(Attribute.h2),
      onHeadingLongPress: () => _showHeadingLevelMenu(l10n),
      onInsertLink: _insertLink,
      showToolbar: false,
      editorBuilder: (context) => DragTarget<ComposerImageDragPayload>(
        onMove: (DragTargetDetails<ComposerImageDragPayload> details) =>
            _updateDropLine(details.offset + _imageDragAnchor),
        onLeave: (Object? _) {
          _dragPointer = null;
          _dropOffset = null;
          _dropLineY.value = null;
        },
        onAcceptWithDetails:
            (DragTargetDetails<ComposerImageDragPayload> details) {
              _stopDragAutoscroll();
              _handleComposerImageDrop(
                details.data,
                details.offset + _imageDragAnchor,
              );
            },
        builder:
            (
              BuildContext context,
              List<ComposerImageDragPayload?> candidateData,
              List<dynamic> rejectedData,
            ) => Stack(
              key: _editorDropKey,
              clipBehavior: Clip.none,
              children: [
                QuillEditor.basic(
                  controller: _quill,
                  focusNode: _editorFocusNode,
                  scrollController: _editorScrollController,
                  config: QuillEditorConfig(
                    scrollable: false,
                    editorKey: _editorKey,
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    placeholder: PublishType.fromValue(_contentType).hint(l10n),
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
                    embedBuilders: [
                      _ComposerImageBuilder(
                        onDragStarted: _startDragAutoscroll,
                        onDragEnded: _stopDragAutoscroll,
                      ),
                    ],
                  ),
                ),
                ValueListenableBuilder<double?>(
                  valueListenable: _dropLineY,
                  builder: (context, y, _) => y == null
                      ? const SizedBox.shrink()
                      : PositionedDirectional(
                          key: const Key('publish-image-drop-line'),
                          start: 0,
                          end: 0,
                          top: y - 1.5,
                          height: 3,
                          child: IgnorePointer(
                            child: DecoratedBox(
                              decoration: BoxDecoration(
                                color: GfTheme.colorsOf(context).primary,
                                borderRadius: BorderRadius.circular(2),
                              ),
                            ),
                          ),
                        ),
                ),
              ],
            ),
      ),
    );
  }

  Widget _buildWritingToolbar(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    return LayoutBuilder(
      builder: (context, constraints) => DecoratedBox(
        key: const Key('publish-writing-tools'),
        decoration: BoxDecoration(
          color: colors.base100,
          border: Border(top: BorderSide(color: colors.line)),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Flexible(
              child: SingleChildScrollView(
                child: ListenableBuilder(
                  listenable: _contentType == 3 ? _quill : _simple,
                  builder: (context, _) {
                    final text = _contentType == 3
                        ? _quill.document.toPlainText()
                        : _simple.text;
                    if (!containsStickerToken(text)) {
                      return const SizedBox.shrink();
                    }
                    return StickerDraftPreview(
                      content: _markdownFromEditor(),
                      markdown: true,
                    );
                  },
                ),
              ),
            ),
            if (_contentType == 3 && _uploads.hasPending)
              _buildUploadTray(l10n),
            // One row above the keyboard: media and stickers stay pinned at
            // the start, article formatting scrolls after them.
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
              child: Row(
                children: [
                  // Publish owns media insertion; the shared toolbar handles
                  // Markdown formatting only.
                  if (_contentType == 3 ||
                      MediaQuery.viewInsetsOf(context).bottom > 0)
                    _toolButton(
                      symbol: _activelyUploading ? 'clock' : 'gallery',
                      tooltip: l10n.publishToolImage,
                      onPressed: _pickingImage ? null : _pickAndInsertImage,
                    ),
                  _toolButton(
                    symbol: _stickerOpen ? 'keyboard' : 'emoji-circle',
                    tooltip: _stickerOpen
                        ? StickerStrings(context).keyboard
                        : StickerStrings(context).title,
                    selected: _stickerOpen,
                    onPressed: _pickingImage ? null : _pickSticker,
                  ),
                  if (_contentType == 3) ...[
                    Container(
                      width: 1,
                      height: 24,
                      margin: const EdgeInsets.symmetric(horizontal: 4),
                      color: colors.line,
                    ),
                    Expanded(child: _buildToolbar(l10n)),
                  ],
                ],
              ),
            ),
            if (_stickerOpen)
              Flexible(
                flex: 3,
                child: ConstrainedBox(
                  constraints: BoxConstraints(
                    maxHeight: constraints.maxHeight * .6,
                  ),
                  child: GfComposerPanel(
                    child: StickerPicker(onInsert: _insertSticker),
                  ),
                ),
              ),
            // The move hint only matters once the body holds an image.
            if (!_stickerOpen &&
                _contentType == 3 &&
                MediaQuery.viewInsetsOf(context).bottom == 0)
              ListenableBuilder(
                listenable: _quill,
                builder: (context, _) =>
                    _quill.document.toDelta().toList().any(
                      (op) => op.data is Map,
                    )
                    ? Padding(
                        padding: const EdgeInsets.fromLTRB(16, 0, 16, 10),
                        child: Text(
                          l10n.publishBodyDragHint,
                          style: GfTheme.typographyOf(context).caption.copyWith(
                            color: GfTheme.colorsOf(context).iconMuted,
                          ),
                        ),
                      )
                    : const SizedBox.shrink(),
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _insertLink() async {
    final l10n = AppLocalizations.of(context);
    String value = '';
    final url = await showDialog<String>(
      context: context,
      animationStyle: GfMotion.dialogStyle(context),
      builder: (context) => AlertDialog(
        title: Text(l10n.publishToolLink),
        content: GfInput(
          autofocus: true,
          keyboardType: TextInputType.url,
          hintText: 'https://',
          autocorrect: false,
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

  Widget _buildToolbar(AppLocalizations l10n) => RichMarkdownToolbar(
    controller: _quill,
    tinted: false,
    onHeading: () => _toggleFormat(Attribute.h2),
    onHeadingLongPress: () => _showHeadingLevelMenu(l10n),
    onInsertLink: _insertLink,
  );

  Widget _toolButton({
    required String symbol,
    required String tooltip,
    required VoidCallback? onPressed,
    VoidCallback? onLongPress,
    bool? selected,
  }) {
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

  Widget _buildPreview(AppLocalizations l10n, {bool framed = false}) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    final publishType = PublishType.fromValue(_contentType);
    // Preview reads as the finished post: a card headed by its type, so the
    // second step looks unlike the open writing canvas of the first.
    return Container(
      key: const Key('publish-preview'),
      constraints: BoxConstraints(minHeight: framed ? 390 : 160),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.base100,
        border: Border.all(color: colors.line, width: borders.width),
        borderRadius: BorderRadius.circular(framed ? radii.box : 20),
      ),
      child: _previewMarkdown.isEmpty && _images.isEmpty
          ? Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  GfSymbol('file-text', size: 32, color: colors.iconMuted),
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
                Row(
                  children: [
                    GfSymbol(
                      publishType.symbol,
                      size: 16,
                      color: publishType.color(context),
                    ),
                    const SizedBox(width: 6),
                    Flexible(
                      child: Text(
                        publishType.label(l10n),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: type.small.copyWith(
                          color: publishType.color(context),
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                if (_images.isNotEmpty)
                  GfMediaCarousel(
                    images: _images.map(resolveApiAssetUrl).toList(),
                  ),
                if (_title.text.trim().isNotEmpty) ...[
                  const SizedBox(height: 16),
                  Text(
                    _title.text.trim(),
                    key: const Key('publish-preview-title'),
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
                    style: readingBodyStyle(context),
                  ),
              ],
            ),
    );
  }

  static const double _galleryTile = 84;

  /// Composer photos as a strip of square thumbnails; long-press reorders,
  /// the trailing tile adds more until the nine-photo limit.
  Widget _buildGallery(AppLocalizations l10n, {required bool editing}) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final cacheWidth =
        (_galleryTile * 2 * MediaQuery.devicePixelRatioOf(context)).round();
    final Widget? add = _images.length + _uploads.items.length >= 9
        ? null
        : _buildGalleryAddTile(l10n);
    return Column(
      key: const Key('publish-gallery'),
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (_uploads.hasPending) ...[
          _buildUploadQueue(l10n),
          const SizedBox(height: 12),
        ],
        if (_images.isEmpty && add != null)
          _buildGalleryAddTile(l10n, empty: true)
        else if (_images.isNotEmpty)
          SizedBox(
            height: _galleryTile,
            child: ReorderableListView.builder(
              scrollDirection: Axis.horizontal,
              itemCount: _images.length,
              footer: add,
              proxyDecorator: (child, _, _) => Material(
                color: Colors.transparent,
                elevation: 6,
                borderRadius: BorderRadius.circular(12),
                child: child,
              ),
              onReorderItem: (oldIndex, newIndex) => setState(() {
                _images.insert(newIndex, _images.removeAt(oldIndex));
                _markDirty();
                _allowPop = false;
              }),
              itemBuilder: (context, index) => Padding(
                key: ValueKey('$index:${_images[index]}'),
                padding: const EdgeInsetsDirectional.only(end: 8),
                child: SizedBox.square(
                  dimension: _galleryTile,
                  child: Stack(
                    fit: StackFit.expand,
                    children: [
                      ClipRRect(
                        borderRadius: BorderRadius.circular(12),
                        child: ColoredBox(
                          color: colors.base200,
                          child: GfNetworkImage(
                            resolveApiAssetUrl(_images[index]),
                            fit: BoxFit.cover,
                            cacheWidth: cacheWidth,
                            errorBuilder: (_, _, _) => Center(
                              child: GfSymbol(
                                'image-off',
                                color: colors.iconMuted,
                              ),
                            ),
                          ),
                        ),
                      ),
                      PositionedDirectional(
                        top: 0,
                        end: 0,
                        child: Tooltip(
                          message: l10n.publishRemoveImage,
                          excludeFromSemantics: true,
                          child: Semantics(
                            button: true,
                            label: l10n.publishRemoveImage,
                            child: InkResponse(
                              radius: 20,
                              onTap: () => setState(() {
                                _images.removeAt(index);
                                _markDirty();
                                _allowPop = false;
                              }),
                              child: SizedBox.square(
                                dimension: 40,
                                child: Center(
                                  child: Container(
                                    width: 22,
                                    height: 22,
                                    decoration: BoxDecoration(
                                      color: Colors.black.withValues(
                                        alpha: .55,
                                      ),
                                      shape: BoxShape.circle,
                                    ),
                                    child: const Center(
                                      child: GfSymbol(
                                        'x',
                                        size: 13,
                                        color: Colors.white,
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        if (_images.length > 1) ...[
          const SizedBox(height: 6),
          Text(
            l10n.publishGalleryHint,
            style: type.caption.copyWith(color: colors.iconMuted),
          ),
        ],
      ],
    );
  }

  /// Photo entry in the type's colour. Empty, it is a full-width card that
  /// explains the strip; with photos, it becomes the strip's trailing tile.
  Widget _buildGalleryAddTile(AppLocalizations l10n, {bool empty = false}) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final busy = _activelyUploading || _pickingImage;
    final count = _images.length + _uploads.items.length;
    final accent = _pickingImage
        ? colors.iconMuted
        : PublishType.fromValue(_contentType).color(context);
    final title = _contentType == 2
        ? l10n.publishGallery
        : l10n.publishToolImage;
    final Widget icon = busy
        ? SizedBox.square(
            dimension: empty ? 24 : 20,
            child: const GfProgressIndicator(strokeWidth: 2),
          )
        : GfSymbol('gallery-duotone', size: empty ? 28 : 24, color: accent);
    final countLabel = Text(
      '$count/9',
      style: type.caption.copyWith(
        color: colors.iconMuted,
        fontFeatures: const [FontFeature.tabularFigures()],
      ),
    );
    return Semantics(
      key: const Key('publish-gallery-add'),
      button: true,
      enabled: !_pickingImage,
      label: empty ? title : '${l10n.publishToolImage} $count/9',
      excludeSemantics: true,
      onTap: _pickingImage ? null : _pickAndInsertImage,
      child: Material(
        color: colors.base200,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(empty ? 16 : 12),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: _pickingImage ? null : _pickAndInsertImage,
          child: empty
              ? Padding(
                  padding: const EdgeInsets.all(16),
                  child: Row(
                    children: [
                      icon,
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(title, style: type.bodyStrong),
                            const SizedBox(height: 4),
                            Text(
                              l10n.publishGalleryHint,
                              style: type.caption.copyWith(
                                color: colors.iconMuted,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                )
              : SizedBox.square(
                  dimension: _galleryTile,
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [icon, const SizedBox(height: 6), countLabel],
                  ),
                ),
        ),
      ),
    );
  }

  String _uploadStatus(AppLocalizations l10n, ComposerUpload item) =>
      item.status == ComposerUploadStatus.uploading
      ? l10n.publishMediaUploading
      : item.error == null
      ? l10n.publishMediaWaiting
      : item.error is ApiException
      ? resolveErrorMessage(l10n, item.error!)
      : l10n.commonLoadFailed;

  /// Article uploads ride just above the writing row so progress stays in
  /// sight while typing; each photo lands where the caret was when picked.
  Widget _buildUploadTray(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final failed = _uploads.items
        .where((item) => item.status == ComposerUploadStatus.failed)
        .firstOrNull;
    return Padding(
      key: const Key('publish-upload-queue'),
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            failed == null
                ? l10n.publishMediaPendingWarning
                : _uploadStatus(l10n, failed),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: type.caption.copyWith(
              color: failed == null ? colors.iconMuted : colors.error,
            ),
          ),
          const SizedBox(height: 8),
          SizedBox(
            height: 56,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: _uploads.items.length,
              separatorBuilder: (_, _) => const SizedBox(width: 8),
              itemBuilder: (context, index) {
                final item = _uploads.items[index];
                final bool isFailed =
                    item.status == ComposerUploadStatus.failed;
                return Semantics(
                  key: ValueKey('upload-${item.id}'),
                  container: true,
                  label: '${item.file.name}, ${_uploadStatus(l10n, item)}',
                  child: SizedBox.square(
                    dimension: 56,
                    child: Stack(
                      fit: StackFit.expand,
                      children: [
                        ClipRRect(
                          borderRadius: BorderRadius.circular(10),
                          child: Image.file(
                            File(item.file.path),
                            fit: BoxFit.cover,
                            cacheWidth: 112,
                            excludeFromSemantics: true,
                            color: item.status == ComposerUploadStatus.queued
                                ? colors.base100.withValues(alpha: .5)
                                : isFailed
                                ? Colors.black.withValues(alpha: .45)
                                : null,
                            colorBlendMode: BlendMode.srcATop,
                            errorBuilder: (_, _, _) => ColoredBox(
                              color: colors.base200,
                              child: const GfSymbol('image', size: 20),
                            ),
                          ),
                        ),
                        if (item.status == ComposerUploadStatus.uploading)
                          const Center(
                            child: SizedBox.square(
                              dimension: 18,
                              child: GfProgressIndicator(strokeWidth: 2),
                            ),
                          ),
                        if (isFailed)
                          Center(
                            child: IconButton(
                              tooltip: l10n.commonRetry,
                              onPressed: () => _uploads.retry(item.id),
                              color: Colors.white,
                              icon: const GfSymbol(
                                'refresh-cw',
                                size: 20,
                                color: Colors.white,
                              ),
                            ),
                          ),
                        PositionedDirectional(
                          top: -10,
                          end: -10,
                          child: Tooltip(
                            message: l10n.publishRemoveImage,
                            excludeFromSemantics: true,
                            child: Semantics(
                              button: true,
                              label: l10n.publishRemoveImage,
                              child: InkResponse(
                                onTap: () => _uploads.remove(item.id),
                                radius: 20,
                                child: SizedBox.square(
                                  dimension: 36,
                                  child: Center(
                                    child: DecoratedBox(
                                      decoration: BoxDecoration(
                                        color: Colors.black.withValues(
                                          alpha: .55,
                                        ),
                                        shape: BoxShape.circle,
                                      ),
                                      child: const Padding(
                                        padding: EdgeInsets.all(3),
                                        child: GfSymbol(
                                          'x',
                                          size: 12,
                                          color: Colors.white,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildUploadQueue(AppLocalizations l10n) {
    final colors = GfTheme.colorsOf(context);
    return Container(
      key: const Key('publish-upload-queue'),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            l10n.publishMediaQueueTitle,
            style: GfTheme.typographyOf(context).bodyStrong,
          ),
          const SizedBox(height: 4),
          Text(l10n.publishMediaPendingWarning),
          Text(
            l10n.publishMediaTemporary,
            style: GfTheme.typographyOf(context).caption,
          ),
          for (final item in _uploads.items)
            Padding(
              key: ValueKey('upload-${item.id}'),
              padding: const EdgeInsets.only(top: 8),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  SizedBox.square(
                    dimension: 48,
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(8),
                      child: Stack(
                        fit: StackFit.expand,
                        children: [
                          Image.file(
                            File(item.file.path),
                            fit: BoxFit.cover,
                            cacheWidth: 96,
                            excludeFromSemantics: true,
                            errorBuilder: (_, _, _) =>
                                const GfSymbol('image', size: 22),
                          ),
                          if (item.status == ComposerUploadStatus.uploading)
                            const Center(
                              child: SizedBox.square(
                                dimension: 20,
                                child: GfProgressIndicator(strokeWidth: 2),
                              ),
                            ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          item.file.name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        Text(
                          item.status == ComposerUploadStatus.uploading
                              ? l10n.publishMediaUploading
                              : item.error == null
                              ? l10n.publishMediaWaiting
                              : item.error is ApiException
                              ? resolveErrorMessage(l10n, item.error!)
                              : l10n.commonLoadFailed,
                          style: GfTheme.typographyOf(context).caption,
                        ),
                      ],
                    ),
                  ),
                  if (item.status == ComposerUploadStatus.failed)
                    IconButton(
                      tooltip: l10n.commonRetry,
                      onPressed: () => _uploads.retry(item.id),
                      icon: const GfSymbol('refresh-cw', size: 22),
                    ),
                  IconButton(
                    tooltip: l10n.publishRemoveImage,
                    onPressed: () => _uploads.remove(item.id),
                    icon: const GfSymbol('x', size: 22),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

/// Two-step position under the app bar: half the hairline while writing,
/// the full line on preview.
class _StepProgress extends StatelessWidget implements PreferredSizeWidget {
  const _StepProgress({required this.step, required this.label});

  final int step;
  final String label;

  static const double height = 2;

  @override
  Size get preferredSize => const Size.fromHeight(height);

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Semantics(
      label: label,
      child: SizedBox(
        height: height,
        child: TweenAnimationBuilder<double>(
          tween: Tween(end: step / 2),
          duration: GfMotion.duration(context, GfMotion.layout),
          curve: GfMotion.layoutCurve,
          builder: (context, value, _) => Align(
            alignment: AlignmentDirectional.centerStart,
            child: FractionallySizedBox(
              widthFactor: value,
              child: ColoredBox(color: colors.primary),
            ),
          ),
        ),
      ),
    );
  }
}

/// One short rise-and-fade when a step's content first appears.
class _StepEntrance extends StatelessWidget {
  const _StepEntrance({required this.child, this.delay = Duration.zero});

  final Widget child;
  final Duration delay;

  @override
  Widget build(BuildContext context) {
    final duration = GfMotion.duration(context, GfMotion.layout + delay);
    if (duration == Duration.zero) return child;
    final start = delay.inMicroseconds / duration.inMicroseconds;
    return TweenAnimationBuilder<double>(
      tween: Tween(begin: 0, end: 1),
      duration: duration,
      curve: Interval(start, 1, curve: GfMotion.enterCurve),
      child: child,
      builder: (context, t, child) => Opacity(
        opacity: t,
        child: Transform.translate(
          offset: Offset(0, GfMotion.rise * (1 - t)),
          child: child,
        ),
      ),
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

const Size _imageDragFeedback = Size(132, 96);
const Offset _imageDragAnchor = Offset(66, 112);

class _ComposerImageBuilder extends EmbedBuilder {
  const _ComposerImageBuilder({this.onDragStarted, this.onDragEnded});

  final VoidCallback? onDragStarted;
  final VoidCallback? onDragEnded;

  @override
  String get key => BlockEmbed.imageType;

  @override
  Widget build(BuildContext context, EmbedContext embedContext) {
    final Widget image = GfNetworkImage(
      resolveApiAssetUrl(embedContext.node.value.data.toString()),
      fit: BoxFit.contain,
      cacheWidth:
          (MediaQuery.sizeOf(context).width *
                  MediaQuery.devicePixelRatioOf(context))
              .round(),
      errorBuilder: (_, _, _) => const GfSymbol('image-off'),
    );
    final Widget thumbnail = GfNetworkImage(
      resolveApiAssetUrl(embedContext.node.value.data.toString()),
      fit: BoxFit.cover,
      cacheWidth: (_imageDragFeedback.width * 2).round(),
      errorBuilder: (_, _, _) => const GfSymbol('image-off'),
    );
    return LongPressDraggable<ComposerImageDragPayload>(
      data: ComposerImageDragPayload(
        node: embedContext.node,
        imageUrl: embedContext.node.value.data.toString(),
      ),
      // The lifted copy rides above the fingertip so the drop line under the
      // finger stays visible.
      dragAnchorStrategy: (_, _, _) => _imageDragAnchor,
      onDragStarted: onDragStarted,
      onDragCompleted: onDragEnded,
      onDraggableCanceled: (Velocity velocity, Offset offset) =>
          onDragEnded?.call(),
      childWhenDragging: Opacity(opacity: 0.35, child: image),
      feedback: Material(
        elevation: 8,
        borderRadius: BorderRadius.circular(12),
        clipBehavior: Clip.antiAlias,
        child: SizedBox.fromSize(size: _imageDragFeedback, child: thumbnail),
      ),
      child: image,
    );
  }
}

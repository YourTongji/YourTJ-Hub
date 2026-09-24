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
import '../../images/image_upload.dart';
import '../../images/composer_upload_queue.dart';
import '../../server_messages.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/status_views.dart';
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
  static const int _maxCategories = 3;
  static const double _wideWorkspaceBreakpoint = 760;
  static const Duration _previewDebounceDuration = Duration(milliseconds: 200);
  static const double _dragAutoscrollEdge = 56;
  static const double _dragAutoscrollStep = 12;

  final TextEditingController _simple = TextEditingController();
  final List<String> _images = [];
  int _contentType = 3;
  bool _formatting = false;
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
  final List<int> _categoryIds = <int>[];
  final List<PublishCategoryPayload> _categories = <PublishCategoryPayload>[];
  late final MarkdownConverter _converter;

  late QuillController _quill;
  final GlobalKey<EditorState> _editorKey = GlobalKey<EditorState>();
  final ScrollController _pageScrollController = ScrollController();
  final GlobalKey _pageScrollViewKey = GlobalKey();
  late StreamSubscription<DocChange> _documentChanges;
  late int _currentTopicId;

  _ComposeMode _mode = _ComposeMode.edit;
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
  int? _mediaInsertAt;
  CaptchaPayload? _captcha;
  final _captchaCode = TextEditingController();
  bool _captchaLoading = false;
  String _loadError = '';
  String _error = '';
  String _message = '';
  String _previewMarkdown = '';
  Timer? _previewDebounce;
  Timer? _dragAutoscrollTimer;
  Offset? _dragPointer;

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
    _title.addListener(_markDirty);
    _simple.addListener(_markDirty);
    _converter = widget.markdownConverter ?? MarkdownConverter();
    _currentTopicId = widget.topicId ?? 0;
    _contentType = widget.initialContentType;
    _simple.text = widget.editContent ?? '';
    _title.text = widget.editTitle ?? '';
    _categoryIds.addAll(widget.editCategoryIds ?? const <int>[]);
    _quill = _createController('');
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
      if (!_uploads.hasPending) _mediaInsertAt = null;
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
    _draftKey = draft.key;
    _draftKind ??= draft.kind;
    _contentType = draft.contentType;
    _currentTopicId = draft.topicId;
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
    _documentChanges = controller.document.changes.listen((DocChange change) {
      if (_mediaInsertAt != null) {
        _mediaInsertAt = change.change.transformPosition(_mediaInsertAt!);
      }
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
        if (!mounted || !_sessionCurrent) return;
      }
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
            ..addAll(payloadCategoryIds.take(_maxCategories));
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
    _title.removeListener(_markDirty);
    _simple.removeListener(_markDirty);
    _previewDebounce?.cancel();
    _dragAutoscrollTimer?.cancel();
    _pageScrollController.dispose();
    _captchaCode.dispose();
    _title.dispose();
    _simple.dispose();
    unawaited(_documentChanges.cancel());
    _quill.dispose();
    super.dispose();
  }

  Future<void> _goBack() async {
    if (_submitting) return;
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
      final choice = await showDialog<String>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(l10n.publishLeaveTitle),
          content: Text(l10n.publishLeaveBody),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, 'continue'),
              child: Text(l10n.publishContinue),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context, 'discard'),
              child: Text(l10n.publishDiscard),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, 'save'),
              child: Text(l10n.draftKeepAndLeave),
            ),
          ],
        ),
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
      _contentType = value;
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
                    ? const Icon(Icons.check)
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
      if (_categoryIds.length < _maxCategories &&
          !_categoryIds.contains(category.id)) {
        _categoryIds.add(category.id);
      }
    });
  }

  Future<void> _pickAndInsertImage() async {
    if (_uploading || _submitting || !_sessionCurrent || _finished) return;
    final l10n = AppLocalizations.of(context);
    final remaining = _contentType == 3 ? 9 : 9 - _images.length;
    if (remaining <= 0) {
      showGfToast(context, l10n.publishGalleryTooMany, error: true);
      return;
    }
    _mediaInsertAt = _contentType == 3
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
      _uploads.add(selected);
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
      if (mounted) setState(() => _pickingImage = false);
    }
  }

  void _insertUploadedImage(String url) {
    if (!mounted || !_sessionCurrent || _finished) return;
    if (_contentType != 3) {
      setState(() => _images.add(url));
      _markDirty();
    } else {
      final insertAt = (_mediaInsertAt ?? _quill.document.length - 1).clamp(
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
    if (localY < _dragAutoscrollEdge && scroll.offset > 0) {
      scroll.jumpTo(
        (scroll.offset - _dragAutoscrollStep).clamp(0.0, maxOffset),
      );
    } else if (localY > viewportBottom && scroll.offset < maxOffset) {
      scroll.jumpTo(
        (scroll.offset + _dragAutoscrollStep).clamp(0.0, maxOffset),
      );
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
      });
    } on ApiException catch (error) {
      if (!mounted || !_sessionCurrent) return;
      if (mounted && _sessionCurrent) {
        setState(() => _error = resolveErrorMessage(l10n, error));
      }
      if (error.messageCode == 'common.captchaRequired' ||
          error.messageCode == 'auth.captcha.invalid') {
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
          !_submitting &&
          !_uploading &&
          (_allowPop || (!_dirty && _mode == _ComposeMode.edit)),
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
              // The preview step puts the draft and publish buttons in the
              // AppBar, which squeezes the title slot on narrow screens. Long
              // locale labels (de "Entwurf speichern"/"Veröffentlichen", ja
              // "下書きを保存"/"投稿") overflowed it; expanding and ellipsizing
              // keeps the title shrinkable instead of overflowing the row.
              isExpanded: true,
              borderRadius: BorderRadius.circular(20),
              dropdownColor: GfTheme.colorsOf(context).base100,
              icon: const GfSymbol('chevron-down', size: 16),
              selectedItemBuilder: (context) => [
                for (final type in PublishType.values)
                  LayoutBuilder(
                    builder: (context, constraints) => Row(
                      children: [
                        if (constraints.maxWidth >= 100) ...[
                          PublishTypeIcon(type, size: 28),
                          const SizedBox(width: 8),
                        ],
                        Expanded(
                          child: Text(
                            type.label(l10n),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: GfTheme.typographyOf(
                              context,
                            ).bodyStrong.copyWith(fontSize: 16),
                          ),
                        ),
                      ],
                    ),
                  ),
              ],
              items: [
                for (final type in PublishType.values)
                  DropdownMenuItem(
                    value: type.value,
                    child: Row(
                      children: [
                        PublishTypeIcon(type, size: 28),
                        const SizedBox(width: 8),
                        Flexible(
                          child: Text(
                            type.label(l10n),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: GfTheme.typographyOf(
                              context,
                            ).bodyStrong.copyWith(fontSize: 16),
                          ),
                        ),
                      ],
                    ),
                  ),
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
            if (_mode == _ComposeMode.preview)
              GfIconButton(
                key: const Key('publish-save-draft'),
                // Icon action keeps the preview AppBar inside the bar even for
                // long locale labels (de/ja): two labelled buttons overflowed
                // the 390 px actions row.
                icon: _submitting
                    ? Icons.hourglass_top_rounded
                    : Icons.save_outlined,
                tooltip: l10n.publishSaveDraft,
                size: 44,
                onPressed: _uploading || _submitting
                    ? null
                    : () => _submit(topicStatus: 0),
              ),
            _buildSubmitAction(l10n),
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
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : Icon(
                editing ? Icons.arrow_forward_rounded : Icons.send_rounded,
                size: 20,
              ),
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

    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final bool wide = constraints.maxWidth >= _wideWorkspaceBreakpoint;
        final EdgeInsets pagePadding = EdgeInsets.symmetric(
          horizontal: wide ? 24 : 20,
          vertical: 20,
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
                        icon: const Icon(Icons.cloud_off_outlined),
                        label: Text(l10n.commonRetry),
                      ),
                    if (_uploads.hasPending) ...[
                      _buildUploadQueue(l10n),
                      const SizedBox(height: 16),
                    ],
                    _buildComposeGuide(l10n),
                    const SizedBox(height: 20),
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
                      Container(
                        padding: const EdgeInsets.all(16),
                        decoration: BoxDecoration(
                          color: GfTheme.colorsOf(
                            context,
                          ).base200.withValues(alpha: .65),
                          borderRadius: BorderRadius.circular(20),
                          border: Border.all(
                            color: GfTheme.colorsOf(context).line,
                          ),
                        ),
                        child: _buildTopicFields(l10n, classification: true),
                      ),
                    ],
                    const SizedBox(height: 16),
                    if (_captcha != null) ...[
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
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildComposeGuide(AppLocalizations l10n) {
    final type = PublishType.fromValue(_contentType);
    final colors = GfTheme.colorsOf(context);
    final typography = GfTheme.typographyOf(context);
    final preview = _mode == _ComposeMode.preview;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            for (final step in [0, 1]) ...[
              if (step == 1) const SizedBox(width: 8),
              Expanded(
                child: Container(
                  height: 3,
                  decoration: BoxDecoration(
                    color: step == 0 || preview
                        ? type.color(context)
                        : colors.line,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
            ],
          ],
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 12,
          runSpacing: 4,
          children: [
            Text(
              '1 · ${l10n.composeEdit}',
              style: typography.caption.copyWith(
                color: preview ? colors.iconMuted : type.color(context),
                fontWeight: FontWeight.w600,
              ),
            ),
            Text(
              '2 · ${l10n.composePreview}',
              style: typography.caption.copyWith(
                color: preview ? type.color(context) : colors.iconMuted,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ),
        if (!preview) ...[
          const SizedBox(height: 12),
          Text(
            type.hint(l10n),
            style: typography.small.copyWith(color: colors.iconMuted),
          ),
        ],
      ],
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
            minLines: 1,
            maxLines: 3,
            decoration: InputDecoration(
              hintText: l10n.publishTitleField,
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
        style: readingBodyStyle(context),
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
    final defaults = DefaultStyles.getInstance(context);
    return ConstrainedBox(
      key: const Key('publish-editor'),
      constraints: const BoxConstraints(minHeight: 220),
      child: DragTarget<ComposerImageDragPayload>(
        onMove: (DragTargetDetails<ComposerImageDragPayload> details) {
          _dragPointer = details.offset;
        },
        onLeave: (Object? _) {
          _dragPointer = null;
        },
        onAcceptWithDetails:
            (DragTargetDetails<ComposerImageDragPayload> details) {
              _stopDragAutoscroll();
              _handleComposerImageDrop(details.data, details.offset);
            },
        builder:
            (
              BuildContext context,
              List<ComposerImageDragPayload?> candidateData,
              List<dynamic> rejectedData,
            ) => QuillEditor.basic(
              controller: _quill,
              config: QuillEditorConfig(
                scrollable: false,
                editorKey: _editorKey,
                padding: const EdgeInsets.symmetric(vertical: 8),
                placeholder: l10n.publishBodyPlaceholder,
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
                if (_contentType == 3)
                  _toolButton(
                    icon: _activelyUploading
                        ? Icons.hourglass_top_rounded
                        : Icons.image_outlined,
                    tooltip: l10n.publishToolImage,
                    onPressed: _uploading ? null : _pickAndInsertImage,
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
          if (_contentType == 3)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 10),
              child: Text(
                l10n.publishBodyDragHint,
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
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
              onLongPress: () => _showHeadingLevelMenu(l10n),
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
    VoidCallback? onLongPress,
  }) {
    return GfIconButton(
      icon: icon,
      tooltip: tooltip,
      size: 44,
      iconSize: 20,
      onPressed: onPressed,
      onLongPress: onLongPress,
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
                    style: readingBodyStyle(context),
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
                _activelyUploading
                    ? const SizedBox.square(
                        dimension: 24,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : GfSymbol(
                        'image',
                        size: 28,
                        color: PublishType.fromValue(
                          _contentType,
                        ).color(context),
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
                      _markDirty();
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
                                _markDirty();
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
            loading: _pickingImage,
            onPressed: _images.length >= 9 || _uploading
                ? null
                : _pickAndInsertImage,
          ),
      ],
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
                                const Icon(Icons.photo_outlined),
                          ),
                          if (item.status == ComposerUploadStatus.uploading)
                            const Center(
                              child: SizedBox.square(
                                dimension: 20,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
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
                      icon: const Icon(Icons.refresh),
                    ),
                  IconButton(
                    tooltip: l10n.publishRemoveImage,
                    onPressed: () => _uploads.remove(item.id),
                    icon: const Icon(Icons.close),
                  ),
                ],
              ),
            ),
        ],
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

class _ComposerImageBuilder extends EmbedBuilder {
  const _ComposerImageBuilder({this.onDragStarted, this.onDragEnded});

  final VoidCallback? onDragStarted;
  final VoidCallback? onDragEnded;

  @override
  String get key => BlockEmbed.imageType;

  @override
  Widget build(BuildContext context, EmbedContext embedContext) {
    final Widget image = Image.network(
      resolveApiAssetUrl(embedContext.node.value.data.toString()),
      fit: BoxFit.contain,
      errorBuilder: (_, _, _) => const Icon(Icons.broken_image_outlined),
    );
    return LongPressDraggable<ComposerImageDragPayload>(
      data: ComposerImageDragPayload(
        node: embedContext.node,
        imageUrl: embedContext.node.value.data.toString(),
      ),
      dragAnchorStrategy: pointerDragAnchorStrategy,
      onDragStarted: onDragStarted,
      onDragCompleted: onDragEnded,
      onDraggableCanceled: (Velocity velocity, Offset offset) =>
          onDragEnded?.call(),
      childWhenDragging: Opacity(opacity: 0.35, child: image),
      feedback: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 180),
        child: Opacity(opacity: 0.9, child: image),
      ),
      child: image,
    );
  }
}

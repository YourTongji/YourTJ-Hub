import 'dart:async';
import '../../local/writing_store.dart';
import '../../widgets/email_verification_recovery.dart';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../images/image_upload.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/moderation_blocked_dialog.dart';
import '../../widgets/stickers/sticker_suggestion_bar.dart';
import '../../widgets/stickers/sticker_text_controller.dart';
import '../../widgets/stickers/sticker_picker.dart';
import '../../widgets/stickers/sticker_strings.dart';
import 'mention_panel.dart';
import 'mention_search.dart';
import 'mention_session.dart';

/// Opens the full-screen reply editor and resolves `true` once saved.
///
/// It reuses the topic reply composer (image, stickers, mentions, hide and a
/// docked action row) with an input that fills the page, like editing a topic
/// on the publish page.
Future<bool?> showPostEditor(
  BuildContext context, {
  required int postId,
  required String content,
}) => Navigator.of(context, rootNavigator: true).push<bool>(
  MaterialPageRoute<bool>(
    fullscreenDialog: true,
    builder: (_) => PostEditor(postId: postId, content: content),
  ),
);

class PostEditor extends ConsumerStatefulWidget {
  const PostEditor({super.key, required this.postId, required this.content});
  final int postId;
  final String content;
  @override
  ConsumerState<PostEditor> createState() => _PostEditorState();
}

class _PostEditorState extends ConsumerState<PostEditor>
    with WidgetsBindingObserver {
  late final StickerTextEditingController _text = StickerTextEditingController(
    text: widget.content,
  )..selection = TextSelection.collapsed(offset: widget.content.length);
  late final FocusNode _focus = FocusNode(onKeyEvent: _mentionKeyEvent);
  final FocusNode _panelFocus = FocusNode(debugLabel: 'post-edit-panel');
  late final MentionSessionController _mentions = MentionSessionController(
    searchUsers: ref.read(mentionUserSearchProvider),
  );
  GfInputAccessory _accessory = GfInputAccessory.keyboard;
  TextSelection? _stickerSelection;
  String? _imageUrl;
  bool _busy = false;
  bool _uploading = false;
  bool _allowPop = false;
  String? _error;
  String? _localStatus;
  late final WritingStore _localStore;
  late final OfflineCacheEpoch _writingSession;
  late final int _writingEpoch;
  late final Future<String?> _owner;
  Timer? _autosave;
  int _revision = 0, _savedRevision = -1;
  late String _lastText = widget.content;
  bool _restoring = false, _finished = false;
  bool _restoredCopy = false;
  bool get _sessionCurrent => _writingSession.isCurrent(_writingEpoch);
  String get _draftKey => postEditDraftKey(widget.postId);
  ApiException? _writeFailure;

  bool get _stickerOpen => _accessory == GfInputAccessory.panel;

  @override
  void initState() {
    super.initState();
    _localStore = ref.read(writingStoreProvider);
    _writingSession = ref.read(offlineCacheEpochProvider.notifier);
    _writingEpoch = ref.read(offlineCacheEpochProvider);
    _owner = ref
        .read(writingScopeProvider.future)
        .then<String?>(
          (scope) => scope.endsWith(':0') ? null : scope,
          onError: (Object _) => null,
        );
    WidgetsBinding.instance.addObserver(this);
    unawaited(_restoreLocal());
    _text.addListener(_valueChanged);
    _focus.addListener(_focusChanged);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _focus.requestFocus();
    });
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _autosave?.cancel();
    unawaited(_saveLocal(notify: false));
    _mentions.dispose();
    _focus.dispose();
    _panelFocus.dispose();
    _text.dispose();
    super.dispose();
  }

  void _valueChanged() {
    if (!_restoring && _text.text != _lastText) {
      _lastText = _text.text;
      _revision++;
      _restoredCopy = false;
      _autosave?.cancel();
      _autosave = Timer(const Duration(milliseconds: 700), _saveLocal);
    }
    final TextEditingValue value = _text.value;
    // Typing while the panel is open moves the next sticker's position too.
    if (_stickerOpen && value.selection.isValid) {
      _stickerSelection = value.selection;
    }
    final int base = value.selection.baseOffset;
    if (!_focus.hasFocus || base < 0 || base != value.selection.extentOffset) {
      _mentions.handleValue(null);
      return;
    }
    _mentions.handleValue(value.text.substring(0, base));
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) unawaited(_saveLocal());
  }

  Future<void> _restoreLocal() async {
    try {
      final owner = await _owner;
      if (owner == null || !_sessionCurrent) return;
      final drafts = await _localStore.drafts(owner);
      if (!mounted || !_sessionCurrent || _revision != 0) return;
      final draft = drafts
          .where(
            (d) =>
                d.kind == DraftKind.postEdit &&
                d.postId == widget.postId &&
                d.key == _draftKey,
          )
          .firstOrNull;
      if (draft == null) return;
      _restoring = true;
      _text.text = draft.content;
      _lastText = draft.content;
      _restoring = false;
      _revision = _savedRevision = 1;
      _restoredCopy = true;
      setState(
        () => _localStatus = AppLocalizations.of(context).draftLocalRestored,
      );
    } catch (_) {
      // Recovery storage must not prevent opening the server's editable text.
    }
  }

  Future<bool> _saveLocal({bool notify = true}) async {
    _autosave?.cancel();
    if (_finished || _revision == 0) return true;
    if (!_sessionCurrent) return false;
    if (_revision == _savedRevision) return true;
    final revision = _revision;
    // Capture before identity or disk awaits. A later account cannot adopt this.
    final draft = LocalDraft(
      key: _draftKey,
      kind: DraftKind.postEdit,
      postId: widget.postId,
      postOriginalContent: widget.content,
      title: '',
      content: _text.text,
      contentType: 2,
      topicId: 0,
      categories: const [],
      images: const [],
      updatedAt: DateTime.now().millisecondsSinceEpoch,
    );
    try {
      final owner = await _owner;
      if (owner == null) {
        if (_normalized(draft.content) != _normalized(widget.content)) {
          throw StateError('Draft requires an account');
        }
        return true;
      }
      if (_normalized(draft.content) == _normalized(widget.content)) {
        await _localStore.delete(
          owner,
          _draftKey,
          isCurrent: () => _sessionCurrent && !_finished,
        );
      } else {
        await _localStore.save(
          owner,
          draft,
          isCurrent: () => _sessionCurrent && !_finished,
        );
      }
      if (!_sessionCurrent || _finished) return false;
      _savedRevision = revision;
      if (mounted && notify && revision == _revision) {
        setState(
          () => _localStatus = AppLocalizations.of(context).draftLocalSaved,
        );
      }
      return true;
    } catch (_) {
      if (mounted && notify && _sessionCurrent && !_finished) {
        setState(
          () => _error = AppLocalizations.of(context).draftLocalSaveFailed,
        );
      }
      return false;
    }
  }

  Future<bool> _deleteRecovery() async {
    _autosave?.cancel();
    try {
      final owner = await _owner;
      if (!_sessionCurrent) return false;
      if (owner != null) {
        await _localStore.delete(
          owner,
          _draftKey,
          isCurrent: () => _sessionCurrent,
        );
      }
      return true;
    } catch (_) {
      if (mounted && _sessionCurrent) {
        setState(
          () => _error = AppLocalizations.of(context).draftLocalSaveFailed,
        );
      }
      return false;
    }
  }

  void _focusChanged() {
    if (!_focus.hasFocus) _mentions.close();
    final GfInputAccessory next = _focus.hasFocus
        ? GfInputAccessory.keyboard
        : _accessory == GfInputAccessory.keyboard
        ? GfInputAccessory.none
        : _accessory;
    if (next != _accessory) setState(() => _accessory = next);
  }

  KeyEventResult _mentionKeyEvent(FocusNode node, KeyEvent event) =>
      handleMentionKeyEvent(
        session: _mentions,
        controller: _text,
        event: event,
        onSelect: _selectMention,
      );

  void _selectMention(MentionToken token, MentionUser user) {
    _text.value = applyMentionReplacement(
      _text.value,
      MentionReplacement(
        start: token.start,
        length: token.length,
        replacement: '@${user.username} ',
      ),
    );
  }

  /// Declares the target accessory before moving focus, so the slot holds the
  /// outgoing surface until the incoming one covers it.
  void _toggleStickers() {
    if (_stickerOpen) {
      setState(() => _accessory = GfInputAccessory.keyboard);
      _focus.requestFocus();
    } else {
      _stickerSelection = _text.selection;
      setState(() => _accessory = GfInputAccessory.panel);
      _panelFocus.requestFocus();
    }
  }

  void _hideAccessory() {
    setState(() => _accessory = GfInputAccessory.none);
    FocusManager.instance.primaryFocus?.unfocus();
  }

  void _insertSticker(String token) {
    if (_busy) return;
    insertStickerText(_text, token, selection: _stickerSelection);
    _stickerSelection = _text.selection;
  }

  /// Saved content keeps leading indentation; only trailing whitespace is
  /// dropped. The dirty check compares the same normalized value.
  static String _normalized(String text) => text.trimRight();

  Future<void> _close() async {
    if (_busy || _uploading) return;
    final l10n = AppLocalizations.of(context);
    if (_normalized(_text.text) != _normalized(widget.content)) {
      final discard = await showGfAlertDialog<bool>(
        context,
        builder: (context) => GfAlertDialog(
          title: GfText(l10n.publishLeaveTitle),
          content: GfText(l10n.publishLeaveBody),
          actions: [
            GfDialogAction(
              onPressed: () => Navigator.pop(context, false),
              child: GfText(l10n.commonCancel),
            ),
            GfDialogAction(
              destructive: true,
              onPressed: () => Navigator.pop(context, true),
              child: GfText(l10n.publishDiscard),
            ),
          ],
        ),
      );
      if (!mounted || discard != true) return;
    }
    // Refuse a destructive discard if its tombstone cannot be persisted.
    if (_revision > 0 && !await _deleteRecovery()) return;
    if (!mounted || !_sessionCurrent) return;
    _finished = true;
    _pop(false);
  }

  void _pop(bool saved) {
    final route = ModalRoute.of(context);
    final navigator = Navigator.of(context);
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      // A 401 may replace the stack in the same frame. Never pop its new login
      // page while this old editor remains mounted during route teardown.
      if (mounted && route?.isActive == true && route?.isCurrent == true) {
        navigator.pop(saved);
      }
    });
  }

  Future<void> _save() async {
    if (_busy || _uploading || _text.text.trim().isEmpty) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      _busy = true;
      _error = null;
      _writeFailure = null;
    });
    try {
      if (!await _saveLocal() || !mounted || !_sessionCurrent) return;
      final UpdatePostResult updated = await ref
          .read(postRepositoryProvider)
          .updatePost(postId: widget.postId, content: _normalized(_text.text));
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      _finished = true;
      // A server acknowledgement remains successful even if local cleanup fails.
      await _deleteRecovery();
      if (!mounted || !_sessionCurrent) return;
      if (updated.pendingReview) {
        final AppLocalizations l10n = AppLocalizations.of(context);
        showGfToast(
          context,
          pendingReviewMessage(l10n, checking: updated.checking),
        );
      }
      _pop(true);
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() {
          _error = resolveErrorMessage(AppLocalizations.of(context), error);
          _writeFailure = error is ApiException ? error : null;
        });
        // AI 图文审查拦截(issue #975):弹出友好提示,编辑内容保持不变。
        await showModerationBlockedDialog(context, error);
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Inserts the uploaded image on its own line at the caret, like a reply;
  /// the newest image shows a removable thumbnail.
  Future<void> _image() async {
    if (_busy || _uploading) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _uploading = true);
    try {
      final url = await pickAndUploadImage(ref: ref);
      if (!mounted ||
          epoch != ref.read(offlineCacheEpochProvider) ||
          url == null) {
        return;
      }
      final String text = _text.text;
      final int offset = _text.selection.baseOffset.clamp(0, text.length);
      final String before = text.substring(0, offset);
      final String after = text.substring(offset);
      final String prefix = before.isEmpty || before.endsWith('\n') ? '' : '\n';
      final String suffix = after.isEmpty || after.startsWith('\n') ? '' : '\n';
      final String insertion = '$prefix![image]($url)$suffix';
      _text.value = TextEditingValue(
        text: '$before$insertion$after',
        selection: TextSelection.collapsed(offset: offset + insertion.length),
      );
      setState(() => _imageUrl = url);
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = resolveErrorMessage(AppLocalizations.of(context), error);
          _writeFailure = error is ApiException ? error : null;
        });
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  void _removeImage() {
    final String? url = _imageUrl;
    if (url == null) return;
    final String token = '![image]($url)';
    final int start = _text.text.indexOf(token);
    if (start >= 0) {
      _text.value = TextEditingValue(
        text: _text.text.replaceRange(start, start + token.length, ''),
        selection: TextSelection.collapsed(offset: start),
      );
    }
    setState(() => _imageUrl = null);
  }

  @override
  Widget build(BuildContext context) {
    _text.library = ref.watch(stickerLibraryProvider);
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final stickers = StickerStrings(context);
    ref.listen(offlineCacheEpochProvider, (_, _) => _pop(false));
    return PopScope(
      canPop: _allowPop,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) return;
        // Back first folds an open sticker panel, like the topic composer.
        if (_stickerOpen) {
          _hideAccessory();
          return;
        }
        _close();
      },
      child: Scaffold(
        // The accessory slot is the only consumer of the keyboard inset.
        resizeToAvoidBottomInset: false,
        appBar: GfAppBar(
          leading: GfIconButton(
            symbol: 'x',
            tooltip: l10n.commonClose,
            size: 44,
            onPressed: _busy || _uploading ? null : _close,
          ),
          title: GfText(l10n.topicEditReply),
        ),
        body: LayoutBuilder(
          builder: (context, box) => Column(
            children: [
              Expanded(
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 560),
                    child: Column(
                      children: [
                        ListenableBuilder(
                          listenable: _mentions,
                          builder: (context, _) => MentionCandidatesPanel(
                            session: _mentions,
                            messages: MentionPanelMessages(
                              listboxLabel: l10n.mentionListboxLabel,
                              loading: l10n.mentionLoading,
                              noResults: l10n.mentionNoResults,
                              searchFailed: l10n.mentionSearchFailed,
                              keepTyping: l10n.mentionKeepTyping,
                              tagReplyTarget: l10n.mentionTagReplyTarget,
                              tagTopicAuthor: l10n.mentionTagTopicAuthor,
                              tagParticipant: l10n.mentionTagParticipant,
                              agentLabel: l10n.mentionAgentLabel,
                            ),
                            onSelect: _selectMention,
                          ),
                        ),
                        Expanded(
                          child: ValueListenableBuilder<TextEditingValue>(
                            valueListenable: _text,
                            builder: (context, value, _) => GfPostComposer(
                              key: const Key('post-edit-composer'),
                              controller: _text,
                              focusNode: _focus,
                              expandEditor: true,
                              hintText: l10n.topicReplyHint,
                              publishLabel: l10n.commonSave,
                              onPublish: _save,
                              publishing: _busy,
                              uploading: _uploading,
                              canPublish: value.text.trim().isNotEmpty,
                              preview: StickerSuggestionBar.text(
                                controller: _text,
                              ),
                              onPickImage: _image,
                              imageTooltip: l10n.publishToolImage,
                              imageUrl: _imageUrl == null
                                  ? null
                                  : resolveApiAssetUrl(_imageUrl!),
                              onRemoveImage: _removeImage,
                              removeImageTooltip: l10n.publishRemoveImage,
                              onPickSticker: _toggleStickers,
                              stickerOpen: _stickerOpen,
                              stickerTooltip: _stickerOpen
                                  ? stickers.keyboard
                                  : stickers.title,
                              hideKeyboardLabel: l10n.commonHideKeyboard,
                              hidePanelLabel: stickers.hidePanel,
                              onHideAccessory: _hideAccessory,
                              toolbar: _error != null || _restoredCopy
                                  ? Column(
                                      mainAxisSize: MainAxisSize.min,
                                      crossAxisAlignment:
                                          CrossAxisAlignment.stretch,
                                      children: [
                                        Semantics(
                                          liveRegion: true,
                                          child: Tooltip(
                                            message: (_error ?? _localStatus)!,
                                            child: GfText(
                                              (_error ?? _localStatus)!,
                                              maxLines: 3,
                                              overflow: TextOverflow.ellipsis,
                                              style:
                                                  GfTheme.typographyOf(
                                                    context,
                                                  ).meta.copyWith(
                                                    color: _error != null
                                                        ? colors.error
                                                        : colors.baseContent
                                                              .withValues(
                                                                alpha: .6,
                                                              ),
                                                  ),
                                            ),
                                          ),
                                        ),
                                        if (_error != null &&
                                            _writeFailure?.messageCode ==
                                                'permission.emailRequired')
                                          EmailVerificationRecovery(
                                            compact: true,
                                            error: _writeFailure,
                                            preserveDraft: _saveLocal,
                                          ),
                                      ],
                                    )
                                  : null,
                              status:
                                  _error != null ||
                                      _restoredCopy ||
                                      _localStatus == null
                                  ? null
                                  : Tooltip(
                                      message: (_error ?? _localStatus)!,
                                      child: GfText(
                                        (_error ?? _localStatus)!,
                                        maxLines: 1,
                                        overflow: TextOverflow.ellipsis,
                                        style: GfTheme.typographyOf(context)
                                            .meta
                                            .copyWith(
                                              color: _error != null
                                                  ? colors.error
                                                  : colors.baseContent
                                                        .withValues(alpha: .6),
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
              GfInputAccessorySlot(
                accessory: _accessory,
                panelFocusNode: _panelFocus,
                availableHeight: box.maxHeight,
                minPanelHeight: StickerPicker.comfortableHeight,
                panelBuilder: (_) => StickerPicker(onInsert: _insertSticker),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

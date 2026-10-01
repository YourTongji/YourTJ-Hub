import '../../offline/drift_cache.dart';
import '../../widgets/cache_snapshot_hint.dart';
import '../../messages/chat_message_bubble.dart';
import '../../messages/chat_image.dart';
import '../../images/image_upload.dart';
import '../../messages/chat_message_row.dart';
import '../../messages/chat_forwarding.dart';
import '../../messages/forward_messages_page.dart';
import '../../messages/forwarded_message.dart';
import '../../report_content.dart';
import '../../user_blocks.dart';
import '../../widgets/stickers/sticker_draft_preview.dart';
import '../../widgets/stickers/sticker_picker.dart';
import '../../widgets/stickers/sticker_strings.dart';
import '../../widgets/stickers/sticker_library_page.dart';
import '../../widgets/stickers/sticker_library_state.dart';
import '../../private_notes.dart';
import '../../widgets/root_surface.dart';
import '../../messages/chat_outbox.dart';
import '../../messages/chat_drafts.dart';
import '../../messages/chat_reply.dart';
import '../../messages/chat_timeline.dart';
import '../../messages/chat_viewport_scroll_physics.dart';
import '../../messages/visible_chat_reads.dart';
import '../../navigation/route_visibility.dart';
import '../../realtime/realtime_updates.dart';

import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:scroll_to_index/scroll_to_index.dart';
import 'package:image_picker/image_picker.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:dio/dio.dart';

import 'package:core/core.dart';

import '../../asset_url.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../format.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';

/// 私信(IM)页(web messages.index 的移动端形态):
/// 会话列表 + 消息游标分页 + 前台事件对账（失联时轮询）+ 可见已读 + 离线缓存。
class MessagesPage extends ConsumerStatefulWidget {
  const MessagesPage({
    super.key,
    this.targetUserId,
    this.targetUsername = '',
    this.targetAvatarUrl = '',
  });

  final int? targetUserId;
  final String targetUsername;
  final String targetAvatarUrl;

  @override
  ConsumerState<MessagesPage> createState() => _MessagesPageState();
}

class _MessagesPageState extends ConsumerState<MessagesPage>
    with WidgetsBindingObserver {
  AsyncValue<List<ChatItemPayload>> _conversations = const AsyncValue.loading();
  List<UserConnectionPayload> _suggestedUsers = const [];
  String _viewerAvatar = '';
  String _viewerUsername = '';
  final TextEditingController _conversationSearch = TextEditingController();
  Timer? _pollTimer;
  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();
  late final GfTabScrollRegistry _tabScrollRegistry;
  late final int _ownerEpoch;
  bool _pollingConfigured = false;
  int _seenRealtimeRevision = 0;
  bool _foreground = true;
  int _loadGeneration = 0;
  bool _cacheCleared = false;
  bool _fromCache = false;
  bool _cacheRefreshing = false;
  DateTime? _snapshotTime;
  bool _loadingRequest = false;
  bool _loadDirty = false;
  CancelToken? _loadCancel;
  ChatItemPayload? _targetConversation;
  bool _serverConversationsResolved = false;

  @override
  void initState() {
    super.initState();
    _ownerEpoch = ref.read(offlineCacheEpochProvider);
    WidgetsBinding.instance.addObserver(this);
    _tabScrollRegistry = ref.read(tabScrollRegistryProvider);
    _foreground =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    _seenRealtimeRevision = ref
        .read(realtimeInvalidationsProvider)
        .chatRevision;
    if (widget.targetUserId == null) {
      _tabScrollRegistry.register(
        GfShellDestination.messages,
        _scrollToTopController,
      );
    }
    _load();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final active = TickerMode.valuesOf(context).enabled;
    _syncPolling(active && _foreground && !ref.read(realtimeHealthyProvider));
    final revision = ref.read(realtimeInvalidationsProvider).chatRevision;
    if (active && revision != _seenRealtimeRevision) {
      _seenRealtimeRevision = revision;
      unawaited(_load(silent: true));
    }
  }

  void _syncPolling(bool shouldPoll) {
    shouldPoll =
        shouldPoll && _ownerEpoch == ref.read(offlineCacheEpochProvider);
    final bool wasConfigured = _pollingConfigured;
    _pollingConfigured = true;
    if (shouldPoll == (_pollTimer != null)) return;

    _pollTimer?.cancel();
    _pollTimer = null;
    if (!shouldPoll) {
      _loadCancel?.cancel('messages page hidden');
      return;
    }
    if (wasConfigured) _load(silent: true);
    _pollTimer = Timer.periodic(
      const Duration(seconds: 15),
      (_) => _load(silent: true),
    );
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _syncPolling(
      _foreground &&
          TickerMode.valuesOf(context).enabled &&
          !ref.read(realtimeHealthyProvider),
    );
  }

  @override
  void didUpdateWidget(covariant MessagesPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.targetUserId == null && widget.targetUserId != null) {
      _tabScrollRegistry.unregister(
        GfShellDestination.messages,
        _scrollToTopController,
      );
    } else if (oldWidget.targetUserId != null && widget.targetUserId == null) {
      _tabScrollRegistry.register(
        GfShellDestination.messages,
        _scrollToTopController,
      );
    }
    if (oldWidget.targetUserId == widget.targetUserId &&
        oldWidget.targetUsername == widget.targetUsername &&
        oldWidget.targetAvatarUrl == widget.targetAvatarUrl) {
      return;
    }
    final List<ChatItemPayload>? items = _conversations.valueOrNull;
    setState(() {
      _targetConversation = items == null
          ? null
          : _targetConversationFor(items);
    });
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _loadCancel?.cancel('messages page disposed');
    WidgetsBinding.instance.removeObserver(this);
    _conversationSearch.dispose();
    _tabScrollRegistry.unregister(
      GfShellDestination.messages,
      _scrollToTopController,
    );
    super.dispose();
  }

  Future<void> _load({bool silent = false}) async {
    if (_ownerEpoch != ref.read(offlineCacheEpochProvider) ||
        (silent && _cacheCleared)) {
      return;
    }
    if (_loadingRequest) {
      _loadDirty = true;
      return;
    }
    _loadingRequest = true;
    _cacheCleared = false;
    setState(() => _cacheRefreshing = true);
    final epoch = ref.read(offlineCacheEpochProvider);
    final generation = ++_loadGeneration;
    final cache = captureChatCache(ref.read(offlineChatCacheProvider));
    if (!cacheRequestCurrent(cache, CacheCategory.chat)) {
      setState(() {
        _loadingRequest = false;
        _cacheRefreshing = false;
        _cacheCleared = true;
        _conversations = const AsyncValue.data([]);
      });
      return;
    }
    final storage = ref.read(tokenStorageProvider);
    final cancel = _loadCancel = CancelToken();
    bool current() =>
        mounted &&
        !cancel.isCancelled &&
        generation == _loadGeneration &&
        epoch == ref.read(offlineCacheEpochProvider) &&
        cacheRequestCurrent(cache, CacheCategory.chat);
    var networkShown = false;
    final localRead = () async {
      if (silent || _serverConversationsResolved) return;
      try {
        if (cache is! DriftOfflineCache && !await hasSessionToken(storage)) {
          return;
        }
        final cached = await cache.getConversations();
        if (!current() ||
            networkShown ||
            _loadDirty ||
            !(cache is DriftOfflineCache
                ? cache.snapshotFound
                : cached.isNotEmpty)) {
          return;
        }
        setState(() {
          _fromCache = true;
          _snapshotTime = cache is DriftOfflineCache
              ? cache.snapshotTime
              : null;
          _conversations = AsyncValue.data(cached);
          _targetConversation = _targetConversationFor(cached);
        });
      } catch (_) {}
    }();
    try {
      final payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/messages', cancelToken: cancel);
      if (!current() || _loadDirty) return;
      final parsed = parsePageProps<MessagesPageProps>(payload);
      if (parsed == null) throw const FormatException('messages props');
      networkShown = true;
      final items = parsed.conversations;
      setState(() {
        _fromCache = false;
        _conversations = AsyncValue.data(items);
        _serverConversationsResolved = true;
        _suggestedUsers = parsed.suggestedUsers;
        _viewerAvatar = resolveApiAssetUrl(payload.layout.viewer.avatarUrl);
        _viewerUsername = payload.layout.viewer.username;
        _targetConversation = _targetConversationFor(items);
      });
      try {
        if (current()) await cache.putConversations(items);
      } catch (_) {}
    } catch (error, stack) {
      if (!current() || _loadDirty) return;
      if (revokesSnapshot(error)) {
        // Revoke the visible snapshot immediately, even when disk cleanup fails.
        if (mounted) {
          setState(() {
            _fromCache = false;
            _snapshotTime = null;
            _conversations = AsyncValue.error(error, stack);
          });
        }
        if (cache is DriftOfflineCache) {
          try {
            await cache.removeConversations();
          } catch (_) {}
        }
        return;
      }
      await localRead;
      if (!current()) return;
      if (!_conversations.hasValue) {
        setState(() => _conversations = AsyncValue.error(error, stack));
      }
    } finally {
      if (generation == _loadGeneration) {
        _loadingRequest = false;
        if (mounted) setState(() => _cacheRefreshing = false);
      }
      if (identical(_loadCancel, cancel)) _loadCancel = null;
      if (_loadDirty &&
          mounted &&
          !_cacheCleared &&
          generation == _loadGeneration &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        _loadDirty = false;
        unawaited(_load(silent: true));
      }
    }
  }

  ChatItemPayload? _targetConversationFor(List<ChatItemPayload> items) {
    final int? targetUserId = widget.targetUserId;
    if (targetUserId == null || targetUserId <= 0) return null;

    for (final ChatItemPayload item in items) {
      if (item.peerId == targetUserId) return item;
    }

    final String targetUsername = widget.targetUsername.trim();
    if (targetUsername.isEmpty) return null;
    return ChatItemPayload(
      id: 0,
      peerId: targetUserId,
      peerUsername: targetUsername,
      peerAvatar: resolveApiAssetUrl(widget.targetAvatarUrl),
      lastMsg: '',
      lastMsgTime: '',
      unreadCount: 0,
      convId: 0,
      peerUrl: '/u/$targetUserId',
    );
  }

  Future<void> _openConversation(ChatItemPayload conv) async {
    // Restored peers may already have a conversation on another device.
    if (conv.convId == 0 && !_serverConversationsResolved) return;
    await Navigator.of(context, rootNavigator: true).push(
      MaterialPageRoute<void>(
        builder: (_) => _ConversationPage(
          conv: conv,
          viewerAvatar: _viewerAvatar,
          viewerUsername: _viewerUsername,
        ),
      ),
    );
    // 返回后刷新会话列表未读数。
    if (mounted) _load(silent: true);
  }

  /// 发起新会话:弹可联系用户列表(web startChat 语义),选中后发消息进入会话。
  Future<void> _startNewChat() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final int visibleUsers = _suggestedUsers.length.clamp(0, 5);
    final double preferredHeight = 164 + visibleUsers * 64;
    final double sheetHeight = math.min(
      MediaQuery.sizeOf(context).height * 0.62,
      math.max(292, preferredHeight),
    );
    final UserConnectionPayload? selected =
        await showGfBottomSheet<UserConnectionPayload>(
          context,
          height: sheetHeight,
          keyboardAware: true,
          builder: (BuildContext ctx) => _NewChatSheet(
            users: _suggestedUsers,
            title: l10n.messagesNew,
            searchHint: l10n.messagesSearchUsers,
            emptyMessage: l10n.messagesNoContactableUsers,
          ),
        );
    if (selected == null || !mounted) return;
    // 已有会话则直接打开;新会话先在本地进入聊天页,第一条真实消息才创建
    // 服务端会话,与 Web startChat 语义一致。
    ChatItemPayload? existing;
    for (final c in _conversations.valueOrNull ?? const <ChatItemPayload>[]) {
      if (c.peerId == selected.id) {
        existing = c;
        break;
      }
    }
    if (existing != null) {
      await _openConversation(existing);
      return;
    }
    await _openConversation(
      ChatItemPayload(
        id: 0,
        peerId: selected.id,
        peerUsername: selected.username,
        peerNickname: selected.nickname.isEmpty ? null : selected.nickname,
        peerAvatar: resolveApiAssetUrl(selected.avatarUrl),
        lastMsg: '',
        lastMsgTime: '',
        unreadCount: 0,
        convId: 0,
        peerUrl: selected.url,
      ),
    );
  }

  Widget _conversationBody(ChatDrafts drafts, double top, double bottom) {
    final l10n = AppLocalizations.of(context);
    final waitingForServer =
        _conversations.isLoading && drafts.items.isNotEmpty;
    final hasStatus =
        drafts.error != null ||
        _conversations.hasError ||
        waitingForServer ||
        _fromCache ||
        _cacheCleared;
    final items = [...?_conversations.valueOrNull];
    final peers = items.map((item) => item.peerId).toSet();
    final local = drafts.items.toList()
      ..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
    items.insertAll(
      0,
      local
          .where((draft) => peers.add(draft.peerId))
          .map((draft) => draft.conversation),
    );
    Widget body;
    if ((_conversations.isLoading || drafts.loading) && items.isEmpty) {
      body = const GfLoading();
    } else if (_conversations.hasError && items.isEmpty) {
      body = GfErrorRetry(
        message: resolveErrorMessage(l10n, _conversations.error!),
        onRetry: _load,
      );
    } else {
      body = GfScrollToTop(
        semanticLabel: l10n.commonBackToTop,
        showButton: false,
        controller: _scrollToTopController,
        builder: (_, controller) => _ConversationList(
          controller: controller,
          padding: EdgeInsets.only(top: hasStatus ? 0 : top, bottom: bottom),
          items: items,
          drafts: {for (final draft in local) draft.peerId: draft},
          canOpenNewConversation: _serverConversationsResolved,
          query: _conversationSearch.text,
          emptyMessage: l10n.messagesEmpty,
          emptyDescription: l10n.messagesEmptyDescription,
          actionLabel: l10n.messagesNew,
          onStart: _startNewChat,
          onOpen: _openConversation,
        ),
      );
    }
    return Column(
      children: [
        if (hasStatus) SizedBox(height: top),
        if (_fromCache || _cacheCleared)
          CacheSnapshotHint(
            savedAt: _snapshotTime,
            refreshing: _cacheRefreshing,
            cleared: _cacheCleared,
            onRetry: _load,
          ),
        if (waitingForServer) const LinearProgressIndicator(),
        _ChatDraftStatus(drafts: drafts),
        if (_conversations.hasError && items.isNotEmpty)
          Row(
            children: [
              Expanded(
                child: Text(resolveErrorMessage(l10n, _conversations.error!)),
              ),
              TextButton(onPressed: _load, child: Text(l10n.commonRetry)),
            ],
          ),
        Expanded(child: body),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final drafts = ref.watch(chatDraftsProvider);
    ref.listen<int>(
      cacheClearEpochProvider.select(
        (epochs) => epochs[CacheCategory.chat] ?? 0,
      ),
      (_, _) {
        _loadGeneration++;
        _loadCancel?.cancel('chat cache cleared');
        setState(() {
          _loadingRequest = false;
          _loadDirty = false;
          _cacheCleared = true;
          _cacheRefreshing = false;
          _fromCache = false;
          _serverConversationsResolved = false;
          _conversations = const AsyncValue.data([]);
          _suggestedUsers = const [];
        });
      },
    );
    ref.listen(offlineCacheEpochProvider, (_, _) {
      _pollTimer?.cancel();
      _pollTimer = null;
      _loadCancel?.cancel('messages session changed');
      _conversationSearch.clear();
      setState(() {
        _conversations = const AsyncValue.loading();
        _serverConversationsResolved = false;
        _targetConversation = null;
        _suggestedUsers = [];
        _viewerAvatar = '';
        _viewerUsername = '';
      });
    });
    if (_ownerEpoch != ref.read(offlineCacheEpochProvider)) {
      return const SizedBox.shrink();
    }
    ref.listen(realtimeHealthyProvider, (_, healthy) {
      _syncPolling(
        _foreground && TickerMode.valuesOf(context).enabled && !healthy,
      );
    });
    ref.listen(realtimeInvalidationsProvider, (previous, next) {
      if (previous?.chatRevision == next.chatRevision ||
          !_foreground ||
          !TickerMode.valuesOf(context).enabled) {
        return;
      }
      _seenRealtimeRevision = next.chatRevision;
      unawaited(_load(silent: true));
    });
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ChatItemPayload? targetConversation = _targetConversation;
    if (targetConversation != null) {
      return _ConversationPage(
        key: ValueKey<int>(targetConversation.peerId),
        conv: targetConversation,
        viewerAvatar: _viewerAvatar,
        viewerUsername: _viewerUsername,
      );
    }
    return RootSurface(
      title: l10n.messagesTitle,
      actionLabel: l10n.messagesNew,
      actionSymbol: 'message-favorite',
      onAction: _startNewChat,
      toolbarHeight: 64,
      toolbar: Padding(
        padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
        child: GfSearchField(
          controller: _conversationSearch,
          hintText: l10n.messagesSearchConversations,
          clearLabel: l10n.courseCopyClearSearch,
          onChanged: (_) => setState(() {}),
        ),
      ),
      body: (top, bottom) => _conversationBody(drafts, top, bottom),
    );
  }
}

/// 单会话聊天页。
class _ConversationPage extends ConsumerStatefulWidget {
  const _ConversationPage({
    super.key,
    required this.conv,
    required this.viewerAvatar,
    required this.viewerUsername,
  });

  final ChatItemPayload conv;
  final String viewerAvatar;
  final String viewerUsername;

  @override
  ConsumerState<_ConversationPage> createState() => _ConversationPageState();
}

class _ConversationPageState extends ConsumerState<_ConversationPage>
    with WidgetsBindingObserver {
  late final ChatDrafts _drafts;
  bool _restoringDraft = false;
  final List<ChatMessagePayload> _messages = [];
  final TextEditingController _input = TextEditingController();
  final FocusNode _composerFocus = FocusNode();
  bool _selecting = false;
  bool _attachingImage = false;
  final Set<int> _selectedMessages = {};
  final AutoScrollController _scrollController = AutoScrollController();
  bool _loading = true;
  bool _loadingOlder = false;
  bool _historyReady = false;
  bool _hasMoreBefore = false;
  bool _hasMoreAfter = false;
  Timer? _pollTimer;
  late int _convId;
  int _latestId = 0;
  int _nextBeforeId = 0;
  int _nextAfterId = 0;
  bool _loadingNewer = false;
  bool _pollingConfigured = false;
  int _seenRealtimeRevision = 0;
  bool _cacheCleared = false;
  bool _fromCache = false;
  bool _cacheRefreshing = false;
  DateTime? _snapshotTime;

  /// An access denial that revoked the cached thread; surfaces as an error
  /// instead of an empty conversation while a retry is available.
  Object? _loadError;
  int _loadGeneration = 0;
  bool _loadingRequest = false;
  bool _loadDirty = false;
  CancelToken? _loadCancel;
  final _viewportKey = GlobalKey();
  final Map<int, GlobalKey> _bubbleKeys = {};
  late final VisibleChatReads _visibleReads;
  late final int _sessionEpoch;
  bool _foreground = true;
  bool _unseenNewMessages = false;
  bool _readSyncFailed = false;
  bool _readSyncUnsupported = false;
  int _scrollAdjustmentGeneration = 0;
  bool _adjustingScroll = false;
  double? _messageViewportHeight;
  ChatReplyTarget? _replyTarget;
  int _replySelection = 0;
  bool _replyJumpInProgress = false;
  int? _replyReturnToMessageId;
  List<ChatMessagePayload>? _replyReturnMessages;
  double _replyReturnOffset = 0;
  bool _replyReturnHasMoreBefore = false;
  int _replyReturnNextBeforeId = 0;
  bool _replyReturnHasMoreAfter = false;
  int _replyReturnNextAfterId = 0;
  int? _highlightedMessageId;
  Timer? _replyHighlightTimer;

  bool get _sessionCurrent =>
      mounted && _sessionEpoch == ref.read(offlineCacheEpochProvider);
  bool get _canObserve =>
      _sessionCurrent &&
      _historyReady &&
      !_cacheCleared &&
      _foreground &&
      !shellDrawerOpen.value &&
      !_adjustingScroll &&
      TickerMode.valuesOf(context).enabled &&
      routeIsUncovered(context);

  Set<int> _visibleMessageIds() {
    final viewport = _viewportKey.currentContext?.findRenderObject();
    if (viewport is! RenderBox || !viewport.hasSize) return {};
    final bounds = (viewport.localToGlobal(Offset.zero) & viewport.size)
        .intersect(Offset.zero & MediaQuery.sizeOf(context));
    return {
      for (final message in _messages)
        if (!message.isSelf &&
            message.id > 0 &&
            message.isRead == 0 &&
            _bubbleVisible(message.id, bounds))
          message.id,
    };
  }

  bool _bubbleVisible(int id, Rect viewport) {
    final bubble = _bubbleKeys[id]?.currentContext?.findRenderObject();
    return bubble is RenderBox &&
        bubble.attached &&
        bubble.hasSize &&
        bubbleIsVisible(
          bubble.localToGlobal(Offset.zero) & bubble.size,
          viewport,
        );
  }

  void _visibilityChanged() {
    if (!_sessionCurrent) return;
    _visibleReads.suspend();
    _visibleReads.changed(restartDwell: true);
    if (!_foreground ||
        shellDrawerOpen.value ||
        !TickerMode.valuesOf(context).enabled ||
        !routeIsUncovered(context)) {
      return;
    }
    final revision = ref.read(realtimeInvalidationsProvider).chatRevision;
    if (revision != _seenRealtimeRevision) {
      _seenRealtimeRevision = revision;
      unawaited(_load(silent: true));
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) unawaited(_drafts.flush());
    _foreground = state == AppLifecycleState.resumed;
    _visibleReads.suspend();
    _syncPolling(
      _foreground &&
          TickerMode.valuesOf(context).enabled &&
          !ref.read(realtimeHealthyProvider),
    );
    if (_foreground) _visibleReads.changed(restartDwell: true);
  }

  @override
  void didChangeMetrics() => _visibleReads.changed(restartDwell: true);

  @override
  void initState() {
    super.initState();
    _drafts = ref.read(chatDraftsProvider);
    _input.addListener(_draftChanged);
    _drafts.addListener(_restoreDraft);
    _restoreDraft();
    _convId = widget.conv.convId > 0
        ? widget.conv.convId
        : ref.read(chatOutboxProvider(widget.conv.peerId)).conversationId;
    _sessionEpoch = ref.read(offlineCacheEpochProvider);
    _seenRealtimeRevision = ref
        .read(realtimeInvalidationsProvider)
        .chatRevision;
    _foreground =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    _visibleReads = VisibleChatReads(
      isEnabled: () => _canObserve,
      visibleIds: _visibleMessageIds,
      acknowledge: (ids) async {
        final result = await ref
            .read(chatRepositoryProvider)
            .markVisible(convId: _convId, messageIds: ids);
        if (result.convId != _convId) {
          throw StateError('Read receipt conversation mismatch');
        }
        return result.acknowledgedMessageIds.toSet();
      },
      onAcknowledged: (ids) {
        if (!_sessionCurrent) return;
        setState(() {
          _readSyncFailed = false;
          for (var i = 0; i < _messages.length; i++) {
            if (ids.contains(_messages[i].id)) {
              _messages[i] = _messages[i].copyWith(isRead: 1);
            }
          }
        });
      },
      canRetry: (error) => !_unsupportedReadApi(error),
      onFailure: (error) {
        if (_sessionCurrent) {
          setState(() {
            _readSyncFailed = true;
            _readSyncUnsupported = _unsupportedReadApi(error);
          });
        }
      },
    );
    WidgetsBinding.instance.addObserver(this);
    routeVisibilityChanges.addListener(_visibilityChanged);
    shellDrawerOpen.addListener(_visibilityChanged);
    _load();
    _scrollController.addListener(_onScroll);
    // 表情包库未就绪时拉一次(会话级缓存),完成后刷新气泡分段渲染。
    _ensureStickers();
  }

  void _draftChanged() {
    if (!_restoringDraft) _drafts.update(widget.conv, _input.value);
  }

  void _restoreDraft() {
    if (!mounted || !_drafts.current) return;
    final value =
        _drafts.forPeer(widget.conv.peerId)?.value ?? TextEditingValue.empty;
    if (_input.text == value.text && _input.selection == value.selection) {
      return;
    }
    _restoringDraft = true;
    _input.value = value;
    _restoringDraft = false;
  }

  bool _unsupportedReadApi(Object error) =>
      error is ApiException &&
      (error.statusCode == 404 || error.statusCode == 405);

  void _ensureStickers() {
    ref
        .read(stickerLibraryProvider)
        .load()
        .then((_) {
          if (mounted) setState(() {});
        })
        .catchError((Object _) {
          // 拉取失败保持纯文本渲染;库不缓存失败,下次进入会话重试。
        });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final active =
        _foreground && _sessionCurrent && TickerMode.valuesOf(context).enabled;
    _syncPolling(active && !ref.read(realtimeHealthyProvider));
    final revision = ref.read(realtimeInvalidationsProvider).chatRevision;
    if (active && revision != _seenRealtimeRevision) {
      _seenRealtimeRevision = revision;
      unawaited(_load(silent: true));
    }
    _visibleReads.changed(restartDwell: true);
  }

  void _syncPolling(bool shouldPoll) {
    final bool wasConfigured = _pollingConfigured;
    _pollingConfigured = true;
    if (shouldPoll == (_pollTimer != null)) return;

    _pollTimer?.cancel();
    _pollTimer = null;
    if (!shouldPoll) {
      _loadCancel?.cancel('conversation hidden');
      return;
    }
    if (wasConfigured) _load(silent: true);
    _pollTimer = Timer.periodic(
      const Duration(seconds: 15),
      (_) => _load(silent: true),
    );
  }

  @override
  void didUpdateWidget(covariant _ConversationPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    final int nextConvId = widget.conv.convId;
    if (_convId > 0 || nextConvId <= 0) return;

    _convId = nextConvId;
    _loading = true;
    _historyReady = false;
    unawaited(_load(silent: true));
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _input.removeListener(_draftChanged);
    _drafts.removeListener(_restoreDraft);
    unawaited(_drafts.flush());
    _pollTimer?.cancel();
    _replyHighlightTimer?.cancel();
    _loadCancel?.cancel('conversation disposed');
    routeVisibilityChanges.removeListener(_visibilityChanged);
    shellDrawerOpen.removeListener(_visibilityChanged);
    _visibleReads.dispose();
    _input.dispose();
    _composerFocus.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    _visibleReads.changed(restartDwell: true);
    _dismissReplyReturnAtBottom();
    if (_unseenNewMessages &&
        _scrollController.hasClients &&
        _scrollController.position.extentAfter < 24) {
      setState(() => _unseenNewMessages = false);
    }
    if (_scrollController.hasClients &&
        _scrollController.position.pixels < 80) {
      _loadOlder();
    }
    if (_scrollController.hasClients &&
        _scrollController.position.extentAfter < 80) {
      _loadNewer();
    }
  }

  void _dismissReplyReturnAtBottom() {
    if (!_sessionCurrent ||
        _replyReturnToMessageId == null ||
        _replyJumpInProgress ||
        _adjustingScroll ||
        _loadingNewer ||
        _hasMoreAfter ||
        !_scrollController.hasClients ||
        _scrollController.position.extentAfter >= 24) {
      return;
    }
    // A paginated history window's edge is not the end of the conversation.
    // Reaching the latest messages completes the detour without restoring the
    // saved snapshot or moving the user's viewport again.
    setState(() {
      _replyReturnToMessageId = null;
      _replyReturnMessages = null;
    });
    unawaited(_load(silent: true));
  }

  bool _onUserScroll(ScrollStartNotification notification) {
    if (notification.dragDetails != null) {
      _scrollAdjustmentGeneration++;
      _adjustingScroll = false;
      _visibleReads.changed(restartDwell: true);
    }
    return false;
  }

  Future<void> _load({bool silent = false}) async {
    if (!_sessionCurrent ||
        _replyJumpInProgress ||
        _replyReturnToMessageId != null ||
        (silent && _cacheCleared)) {
      return;
    }
    _cacheCleared = false;
    _loadError = null;
    if (_loadingRequest) {
      _loadDirty = true;
      return;
    }
    if (_convId <= 0) {
      _convId = ref.read(chatOutboxProvider(widget.conv.peerId)).conversationId;
    }
    if (_convId <= 0) {
      if (mounted) {
        setState(() {
          _loading = false;
          // A new conversation has no older messages to reconcile against.
          _historyReady = true;
        });
      }
      return;
    }
    _loadingRequest = true;
    setState(() {
      _cacheRefreshing = true;
      if (!silent && !_historyReady) _loading = true;
    });
    final epoch = ref.read(offlineCacheEpochProvider);
    final generation = ++_loadGeneration;
    final cache = captureChatCache(ref.read(offlineChatCacheProvider));
    if (!cacheRequestCurrent(cache, CacheCategory.chat)) {
      setState(() {
        _loadingRequest = false;
        _cacheRefreshing = false;
        _cacheCleared = true;
        _loading = false;
        _historyReady = false;
      });
      return;
    }
    final storage = ref.read(tokenStorageProvider);
    final convId = _convId;
    final cancel = _loadCancel = CancelToken();
    bool current() =>
        _sessionCurrent &&
        !cancel.isCancelled &&
        generation == _loadGeneration &&
        epoch == ref.read(offlineCacheEpochProvider) &&
        cacheRequestCurrent(cache, CacheCategory.chat);
    final initial = !_historyReady;
    var networkShown = false;
    final localRead = () async {
      if (!initial || silent) return;
      try {
        if (cache is! DriftOfflineCache && !await hasSessionToken(storage)) {
          return;
        }
        final cached = await cache.getMessages(convId);
        if (!current() ||
            networkShown ||
            !(cache is DriftOfflineCache
                ? cache.snapshotFound
                : cached.isNotEmpty)) {
          return;
        }
        setState(() {
          _fromCache = true;
          _snapshotTime = cache is DriftOfflineCache
              ? cache.snapshotTime
              : null;
          _messages
            ..clear()
            ..addAll(cached);
          _loading = false;
          // Snapshot display grants neither send reconciliation nor read receipts.
          _historyReady = false;
        });
        _scrollToBottom();
      } catch (_) {}
    }();
    try {
      final response = await ref
          .read(chatRepositoryProvider)
          .getMessages(
            convId: convId,
            afterId: initial ? 0 : _latestId,
            cancelToken: cancel,
          );
      if (!current()) return;
      networkShown = true;
      final pinnedToBottom =
          !_scrollController.hasClients ||
          _scrollController.position.extentAfter < 80;
      final seenIds = initial
          ? <int>{}
          : _messages.map((message) => message.id).toSet();
      final newMessages = response.list
          .where((message) => seenIds.add(message.id))
          .toList();
      setState(() {
        // The first server window replaces a snapshot, including deleted rows.
        if (initial) _messages.clear();
        _fromCache = false;
        _messages.addAll(newMessages);
        _messages.sort((a, b) => a.id.compareTo(b.id));
        if (!initial &&
            (!pinnedToBottom || !_canObserve) &&
            newMessages.any((m) => !m.isSelf && m.isRead == 0)) {
          _unseenNewMessages = true;
        }
        if (initial || response.latestId > _latestId) {
          _latestId = response.latestId;
        }
        _historyReady = true;
        if (initial) {
          _hasMoreBefore = response.hasMoreBefore;
          _nextBeforeId = response.nextBeforeId;
          // An afterId response describes its newer window, not the oldest
          // loaded cursor. Preserve newer pagination for quote jumps and
          // bottom-edge loads.
          _hasMoreAfter = response.hasMoreAfter;
          _nextAfterId = response.latestId;
        }
        _loading = false;
      });
      ref.read(chatOutboxProvider(widget.conv.peerId)).reconcile(_messages);
      if (initial || pinnedToBottom) _scrollToBottom();
      if (newMessages.isNotEmpty && current()) {
        try {
          await cache.putMessages(convId, newMessages);
        } catch (_) {}
      }
    } catch (error) {
      if (!current() || _loadDirty) return;
      if (revokesSnapshot(error)) {
        // Revoke the visible snapshot immediately, even when disk cleanup fails.
        if (mounted) {
          setState(() {
            _fromCache = false;
            _snapshotTime = null;
            _loadError = error;
            _messages.clear();
            _loading = false;
            _historyReady = false;
          });
        }
        if (cache is DriftOfflineCache) {
          try {
            await cache.removeMessages(convId);
          } catch (_) {}
        }
        return;
      }
      await localRead;
      if (current()) setState(() => _loading = false);
    } finally {
      if (generation == _loadGeneration) {
        _loadingRequest = false;
        if (mounted) setState(() => _cacheRefreshing = false);
      }
      if (identical(_loadCancel, cancel)) _loadCancel = null;
      if (_loadDirty &&
          mounted &&
          !_cacheCleared &&
          generation == _loadGeneration &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        _loadDirty = false;
        unawaited(_load(silent: true));
      }
    }
  }

  Future<void> _loadOlder() async {
    if (!_sessionCurrent ||
        _replyJumpInProgress ||
        !_historyReady ||
        _cacheCleared ||
        _loadingOlder ||
        !_hasMoreBefore ||
        _nextBeforeId <= 0) {
      return;
    }
    final clearEpoch =
        ref.read(cacheClearEpochProvider)[CacheCategory.chat] ?? 0;
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _loadingOlder = true);
    try {
      final ChatMessagesResponse resp = await ref
          .read(chatRepositoryProvider)
          .getMessages(convId: _convId, beforeId: _nextBeforeId);
      if (!mounted ||
          clearEpoch !=
              (ref.read(cacheClearEpochProvider)[CacheCategory.chat] ?? 0) ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final previousExtent = _scrollController.hasClients
          ? _scrollController.position.maxScrollExtent
          : 0.0;
      final previousOffset = _scrollController.hasClients
          ? _scrollController.offset
          : 0.0;
      final viewport = _viewportKey.currentContext?.findRenderObject();
      (GlobalKey, double)? anchor;
      if (viewport is RenderBox && viewport.hasSize) {
        final bounds = viewport.localToGlobal(Offset.zero) & viewport.size;
        for (final message in _messages) {
          final key = _bubbleKeys[message.id];
          final box = key?.currentContext?.findRenderObject();
          if (box is RenderBox &&
              box.attached &&
              box.hasSize &&
              (box.localToGlobal(Offset.zero) & box.size).overlaps(bounds)) {
            anchor = (key!, box.localToGlobal(Offset.zero).dy);
            break;
          }
        }
      }
      setState(() {
        final Set<int> existing = _messages.map((m) => m.id).toSet();
        _messages.addAll(resp.list.where((m) => !existing.contains(m.id)));
        _messages.sort((a, b) => a.id.compareTo(b.id));
        _hasMoreBefore = resp.hasMoreBefore;
        _nextBeforeId = resp.nextBeforeId;
      });
      _restoreHistoryPosition(previousExtent, previousOffset, anchor);
    } catch (_) {
      // 历史消息加载失败保持当前列表，允许下一次滚动重试。
    } finally {
      if (mounted &&
          clearEpoch ==
              (ref.read(cacheClearEpochProvider)[CacheCategory.chat] ?? 0)) {
        setState(() => _loadingOlder = false);
      }
    }
  }

  Future<void> _loadNewer() async {
    if (!_sessionCurrent ||
        _replyJumpInProgress ||
        _loadingNewer ||
        !_hasMoreAfter ||
        _nextAfterId <= 0) {
      return;
    }
    final afterId = _nextAfterId;
    final epoch = ref.read(offlineCacheEpochProvider);
    _loadingNewer = true;
    try {
      final response = await ref
          .read(chatRepositoryProvider)
          .getMessages(convId: _convId, afterId: afterId);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (response.latestId <= afterId) {
        setState(() => _hasMoreAfter = false);
        return;
      }
      setState(() {
        final existing = _messages.map((message) => message.id).toSet();
        _messages.addAll(
          response.list.where((message) => existing.add(message.id)),
        );
        _messages.sort((a, b) => a.id.compareTo(b.id));
        _hasMoreAfter = response.hasMoreAfter;
        _nextAfterId = response.latestId;
        if (response.latestId > _latestId) _latestId = response.latestId;
      });
    } catch (_) {
      // Retry on a later bottom-edge scroll; keep the currently loaded window.
    } finally {
      _loadingNewer = false;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _dismissReplyReturnAtBottom();
      });
    }
  }

  void _restoreHistoryPosition(
    double previousExtent,
    double previousOffset,
    (GlobalKey, double)? anchor,
  ) {
    _settleScroll((position) {
      final box = anchor?.$1.currentContext?.findRenderObject();
      return box is RenderBox && box.attached && box.hasSize
          ? position.pixels + box.localToGlobal(Offset.zero).dy - anchor!.$2
          : previousOffset + position.maxScrollExtent - previousExtent;
    });
  }

  void _scrollToBottom() =>
      _settleScroll((position) => position.maxScrollExtent);

  Future<void> _jumpToQuotedMessage(int messageId, int sourceMessageId) async {
    if (!_sessionCurrent ||
        _selecting ||
        _loadingOlder ||
        _loadingNewer ||
        _loadingRequest ||
        _replyJumpInProgress ||
        messageId <= 0 ||
        sourceMessageId <= 0) {
      return;
    }
    final epoch = _sessionEpoch;
    setState(() => _replyJumpInProgress = true);
    try {
      final targetIsLoaded = _messages.any(
        (message) => message.id == messageId,
      );
      ChatMessagesResponse? around;
      if (!targetIsLoaded) {
        around = await ref
            .read(chatRepositoryProvider)
            .getMessages(convId: _convId, aroundId: messageId, limit: 30);
        if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
        if (!around.list.any((message) => message.id == messageId)) {
          showGfToast(
            context,
            AppLocalizations.of(context).commonLoadFailed,
            error: true,
          );
          return;
        }
      }

      setState(() {
        _replyReturnMessages = List.of(_messages);
        _replyReturnOffset = _scrollController.hasClients
            ? _scrollController.offset
            : 0;
        _replyReturnHasMoreBefore = _hasMoreBefore;
        _replyReturnNextBeforeId = _nextBeforeId;
        _replyReturnHasMoreAfter = _hasMoreAfter;
        _replyReturnNextAfterId = _nextAfterId;
        _replyReturnToMessageId = sourceMessageId;
        if (around != null) {
          _messages
            ..clear()
            ..addAll(around.list);
          final visibleIds = around.list.map((message) => message.id).toSet();
          _bubbleKeys.removeWhere((id, _) => !visibleIds.contains(id));
          _hasMoreBefore = around.hasMoreBefore;
          _nextBeforeId = around.nextBeforeId;
          _hasMoreAfter = around.hasMoreAfter;
          _nextAfterId = around.latestId;
        }
      });
      await _revealQuotedMessage(messageId);
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _replyJumpInProgress = false);
      }
    }
  }

  Future<void> _revealQuotedMessage(int messageId) async {
    final generation = ++_scrollAdjustmentGeneration;
    _adjustingScroll = true;
    _visibleReads.suspend();
    try {
      await WidgetsBinding.instance.endOfFrame;
      if (!_sessionCurrent || !_scrollController.hasClients) return;
      final index = _messages.indexWhere((message) => message.id == messageId);
      if (index < 0) return;
      // An around-ID response still contains lazy, variable-height rows. Seek
      // by index first so the target exists before measuring its exact position.
      if (!mounted) return;
      final reducedMotion = GfMotion.reducedOf(context);
      BuildContext? targetContext;
      if (reducedMotion) {
        targetContext = await _jumpToQuotedMessageWithoutAnimation(
          index,
          messageId,
          generation,
        );
        if (!_sessionCurrent || generation != _scrollAdjustmentGeneration) {
          return;
        }
      } else {
        await _scrollController.scrollToIndex(
          index,
          preferPosition: AutoScrollPosition.begin,
          duration: const Duration(milliseconds: 260),
        );
        if (!_sessionCurrent || generation != _scrollAdjustmentGeneration) {
          return;
        }
        targetContext =
            _bubbleKeys[messageId]?.currentContext ??
            _scrollController.tagMap[index]?.context;
        if (targetContext != null && targetContext.mounted) {
          if (!mounted) return;
          await Scrollable.ensureVisible(
            targetContext,
            alignment: 0.12,
            duration: GfMotion.duration(context, GfMotion.layout),
            curve: Curves.easeInOut,
          );
        }
      }
      if (!mounted ||
          !_sessionCurrent ||
          targetContext == null ||
          !targetContext.mounted) {
        return;
      }
      _replyHighlightTimer?.cancel();
      setState(() => _highlightedMessageId = messageId);
      _replyHighlightTimer = Timer(const Duration(milliseconds: 1400), () {
        if (mounted && _highlightedMessageId == messageId) {
          setState(() => _highlightedMessageId = null);
        }
      });
    } finally {
      if (generation == _scrollAdjustmentGeneration) {
        _adjustingScroll = false;
        if (_sessionCurrent) _visibleReads.changed(restartDwell: true);
      }
    }
  }

  /// The scroll_to_index package asserts on a zero duration. Reduced motion
  /// therefore advances the lazy list in discrete viewport-sized jumps until
  /// the target tag mounts, then aligns it without an animation.
  Future<BuildContext?> _jumpToQuotedMessageWithoutAnimation(
    int index,
    int messageId,
    int generation,
  ) async {
    while (mounted &&
        _sessionCurrent &&
        generation == _scrollAdjustmentGeneration) {
      final targetContext =
          _bubbleKeys[messageId]?.currentContext ??
          _scrollController.tagMap[index]?.context;
      if (targetContext != null && targetContext.mounted) {
        await Scrollable.ensureVisible(
          targetContext,
          alignment: 0.12,
          duration: Duration.zero,
        );
        return targetContext;
      }
      if (!_scrollController.hasClients) return null;
      final position = _scrollController.position;
      final tags = _scrollController.tagMap.keys.toList()..sort();
      if (tags.isEmpty) {
        if (position.maxScrollExtent <= position.minScrollExtent) return null;
        final offset = position.pixels;
        await WidgetsBinding.instance.endOfFrame;
        if (!_scrollController.hasClients) return null;
        // No tags means no mounted row reports progress by itself, so detect
        // growth through the scroll position: if nothing moved after a frame,
        // lazy pagination is exhausted and the target cannot be reached.
        if (_scrollController.tagMap.isEmpty &&
            (_scrollController.position.pixels - offset).abs() < 0.5) {
          return null;
        }
        continue;
      }
      final nearest = (index - tags.first).abs() <= (index - tags.last).abs()
          ? tags.first
          : tags.last;
      final direction = index >= nearest ? 1.0 : -1.0;
      final step = position.viewportDimension * 0.7;
      final next = (position.pixels + direction * step)
          .clamp(position.minScrollExtent, position.maxScrollExtent)
          .toDouble();
      if ((next - position.pixels).abs() < 0.5) return null;
      _scrollController.jumpTo(next);
      await WidgetsBinding.instance.endOfFrame;
    }
    return null;
  }

  Future<void> _returnToReplySource() async {
    final sourceId = _replyReturnToMessageId;
    final savedMessages = _replyReturnMessages;
    if (!_sessionCurrent ||
        _replyJumpInProgress ||
        sourceId == null ||
        savedMessages == null) {
      return;
    }
    final epoch = _sessionEpoch;
    _adjustingScroll = true;
    _visibleReads.suspend();
    setState(() {
      _replyJumpInProgress = true;
      _messages
        ..clear()
        ..addAll(savedMessages);
      final savedIds = savedMessages.map((message) => message.id).toSet();
      _bubbleKeys.removeWhere((id, _) => !savedIds.contains(id));
      _hasMoreBefore = _replyReturnHasMoreBefore;
      _nextBeforeId = _replyReturnNextBeforeId;
      _hasMoreAfter = _replyReturnHasMoreAfter;
      _nextAfterId = _replyReturnNextAfterId;
      _replyReturnToMessageId = null;
      _replyReturnMessages = null;
    });
    try {
      await WidgetsBinding.instance.endOfFrame;
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (_scrollController.hasClients) {
        final target = _replyReturnOffset
            .clamp(0.0, _scrollController.position.maxScrollExtent)
            .toDouble();
        if (GfMotion.reducedOf(context)) {
          _scrollController.jumpTo(target);
        } else {
          await _scrollController.animateTo(
            target,
            duration: GfMotion.overlay,
            curve: GfMotion.layoutCurve,
          );
        }
      }
      if (!_sessionCurrent) return;
      _replyHighlightTimer?.cancel();
      setState(() => _highlightedMessageId = sourceId);
      _replyHighlightTimer = Timer(const Duration(milliseconds: 1400), () {
        if (mounted && _highlightedMessageId == sourceId) {
          setState(() => _highlightedMessageId = null);
        }
      });
    } finally {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        _adjustingScroll = false;
        setState(() => _replyJumpInProgress = false);
        _visibleReads.changed(restartDwell: true);
        unawaited(_load(silent: true));
      }
    }
  }

  void _settleScroll(double Function(ScrollPosition) targetFor) {
    final generation = ++_scrollAdjustmentGeneration;
    _adjustingScroll = true;
    _visibleReads.suspend();
    // Lazy variable-height rows can revise extents after the first jump. Resolve
    // the real anchor over bounded frames; intermediate positions are not reads.
    void settle(int remaining) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (generation != _scrollAdjustmentGeneration) return;
        if (!_sessionCurrent ||
            !_foreground ||
            !_scrollController.hasClients ||
            !TickerMode.valuesOf(context).enabled ||
            !routeIsUncovered(context)) {
          _adjustingScroll = false;
          return;
        }
        final position = _scrollController.position;
        final target = targetFor(position).clamp(0.0, position.maxScrollExtent);
        if ((target - position.pixels).abs() <= 1 || remaining == 0) {
          _adjustingScroll = false;
          _visibleReads.changed(restartDwell: true);
          return;
        }
        _scrollController.jumpTo(target);
        settle(remaining - 1);
        WidgetsBinding.instance.ensureVisualUpdate();
      });
    }

    settle(8);
  }

  Future<void> _showAttachments() async {
    if (!_sessionCurrent || _attachingImage) return;
    final l10n = AppLocalizations.of(context);
    final action = await showGfBottomSheet<String>(
      context,
      builder: (context) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(
            leading: const GfSymbol('image'),
            title: Text(l10n.publishToolImage),
            enabled:
                _historyReady &&
                !ref.read(chatOutboxProvider(widget.conv.peerId)).sending,
            onTap: () => Navigator.pop(context, 'image'),
          ),
          ListTile(
            leading: const GfSymbol('smile'),
            title: Text(StickerStrings(context).add),
            onTap: () => Navigator.pop(context, 'stickers'),
          ),
        ],
      ),
    );
    if (!mounted || !_sessionCurrent) return;
    if (action == 'image') {
      await _attachImage();
    } else if (action == 'stickers') {
      await Navigator.of(context).push(
        MaterialPageRoute<void>(builder: (_) => const StickerLibraryPage()),
      );
    }
  }

  Future<void> _attachImage() async {
    if (!_sessionCurrent ||
        !_historyReady ||
        _attachingImage ||
        ref.read(chatOutboxProvider(widget.conv.peerId)).sending) {
      return;
    }
    final picker = ref.read(imagePickerProvider);
    final files = ref.read(fileRepositoryProvider);
    final outbox = ref.read(chatOutboxProvider(widget.conv.peerId));
    final imageKey = newChatMessageId();
    String? uploadedUrl;
    PendingMessage? pending;
    setState(() => _attachingImage = true);
    try {
      final file = await picker.pickImage(
        source: ImageSource.gallery,
        maxWidth: 2048,
        imageQuality: 85,
        requestFullMetadata: false,
      );
      if (!_sessionCurrent || file == null) return;
      final bytes = await file.readAsBytes();
      if (!mounted || !_sessionCurrent) return;
      final url = await showDialog<String>(
        context: context,
        barrierDismissible: false,
        builder: (_) => ChatImagePreview(
          bytes: bytes,
          ownerEpoch: _sessionEpoch,
          upload: () async {
            await outbox.prepareImageUpload();
            uploadedUrl ??= await files.uploadImage(
              bytes: bytes,
              filename: file.name,
            );
            pending = await outbox.enqueueImage(
              uploadedUrl!,
              _latestId,
              clientMessageId: imageKey,
            );
            return uploadedUrl!;
          },
        ),
      );
      if (!_sessionCurrent || !_historyReady || url == null) return;
      _scrollToBottom();
      await _sendPending(pending!, fromImageUpload: true);
    } catch (error) {
      if (mounted && _sessionCurrent) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _attachingImage = false);
    }
  }

  Future<void> _send(String value) async {
    if (!_historyReady || _attachingImage) return;
    final text = value.trim();
    if (text.isEmpty) return;
    final outbox = ref.read(chatOutboxProvider(widget.conv.peerId));
    if (!_drafts.current || outbox.sending) {
      return;
    }
    _draftChanged();
    final reply = _replyTarget;
    final content = reply == null ? text : reply.compose(text);
    final revision = _drafts.forPeer(widget.conv.peerId)?.revision;
    final submitted = _input.value;
    final failed = outbox.items
        .where(
          (item) =>
              item.state == DeliveryState.failed &&
              item.draftRevision == revision &&
              item.content == content &&
              item.replyToMessageId == reply?.messageId,
        )
        .firstOrNull;
    final message =
        failed ??
        outbox.enqueue(
          content,
          _latestId,
          draftRevision: revision,
          // Keep the whole pre-send composer state, not just the string, so a
          // failure can restore sticker tokens, newlines and the caret.
          draftValue: submitted,
          replyToMessageId: reply?.messageId,
        );
    // 引用随本次发送进入 outbox;失败时由 _sendPending 挂回,重试沿用同一条内容。
    if (reply != null) setState(() => _replyTarget = null);
    _scrollToBottom();
    await _sendPending(message, reply: reply);
  }

  /// 引用头里的发送者标签:对方取会话对手的用户名,自己取页面布局的 viewer
  /// 用户名(JWT 无 username 声明,currentUser 的用户名恒为空)。
  String _replySender(ChatMessagePayload message) {
    final String username = message.isSelf
        ? widget.viewerUsername
        : widget.conv.peerUsername;
    final String trimmed = username.trim();
    if (trimmed.isNotEmpty) return '@$trimmed';
    return message.isSelf ? AppLocalizations.of(context).messageReplySelf : '';
  }

  /// 消息内已解析的表情名(长按菜单的收藏入口)。
  List<String> _resolvedStickerNames(String content) {
    final Map<String, String> resolved = ref
        .read(stickerLibraryProvider)
        .urlByName;
    return parseStickerSegments(content, resolved)
        .whereType<StickerImageSegment>()
        .map((segment) => segment.name)
        .toList(growable: false);
  }

  /// 长按消息呼出操作菜单:回复/复制/收藏表情;举报仅对方消息,沿用
  /// chat_message 链路。
  Future<void> _showMessageActions(ChatMessagePayload message) async {
    final l10n = AppLocalizations.of(context);
    final isForwardedHistory =
        message.msgType == 4 || message.forwarded != null;
    final stickerNames = isForwardedHistory
        ? const <String>[]
        : _resolvedStickerNames(message.content);
    final collection = stickerNames.isEmpty
        ? null
        : ref.read(stickerCollectionProvider);
    final canCollect = collection?.active ?? false;
    final box = _bubbleKeys[message.id]?.currentContext?.findRenderObject();
    final overlay = Navigator.of(
      context,
      rootNavigator: true,
    ).overlay?.context.findRenderObject();
    if (box is! RenderBox || overlay is! RenderBox || !_sessionCurrent) return;
    final origin = box.localToGlobal(Offset.zero, ancestor: overlay);
    final action = await showMenu<String>(
      context: context,
      useRootNavigator: true,
      position: RelativeRect.fromRect(
        origin & box.size,
        Offset.zero & overlay.size,
      ),
      items: [
        if (_historyReady)
          PopupMenuItem(value: 'reply', child: Text(l10n.messageReply)),
        if (!isForwardedHistory)
          PopupMenuItem(value: 'copy', child: Text(l10n.messagesCopyAll)),
        if (_historyReady)
          PopupMenuItem(value: 'forward', child: Text(l10n.messageForward)),
        if (_historyReady)
          PopupMenuItem(value: 'select', child: Text(l10n.messageSelect)),
        if (_historyReady && !isForwardedHistory && canCollect)
          PopupMenuItem(
            value: 'collect',
            child: Text(StickerStrings(context).collect),
          ),
        if (_historyReady && !message.isSelf)
          PopupMenuItem(value: 'report', child: Text(l10n.messageReport)),
      ],
    );
    if (!mounted || !_sessionCurrent || action == null) return;
    if (action != 'copy' && (!_historyReady || _cacheCleared)) return;
    switch (action) {
      case 'reply':
        _replyTo(message);
      case 'forward':
        await _forward([message.id]);
      case 'select':
        _composerFocus.unfocus();
        setState(() {
          _selecting = true;
          _selectedMessages.add(message.id);
        });
      case 'copy':
        await Clipboard.setData(ClipboardData(text: message.content));
        if (mounted) showGfToast(context, l10n.messageCopied);
      case 'collect':
        final strings = StickerStrings(context);
        try {
          for (final name in stickerNames) {
            await ref.read(stickerCollectionProvider).save(stickerName: name);
          }
          if (mounted) {
            showGfToast(context, strings.saved);
          }
        } catch (error) {
          if (mounted) {
            showGfToast(context, strings.failure(error), error: true);
          }
        }
      case 'report':
        await showContentReport(
          context,
          targetType: 'chat_message',
          targetId: message.id,
        );
    }
  }

  void _replyTo(ChatMessagePayload message) {
    if (!_sessionCurrent || _selecting) return;
    final isForwardedHistory =
        message.msgType == 4 || message.forwarded != null;
    setState(() {
      _replySelection++;
      _replyTarget = ChatReplyTarget(
        messageId: message.id,
        sender: _replySender(message),
        content: isForwardedHistory
            ? '[${AppLocalizations.of(context).messageForwardHistory}]'
            : message.msgType == 2
            ? chatImageReplyMarker
            : message.content,
      );
    });
    _composerFocus.requestFocus();
  }

  void _toggleMessage(int id) {
    if (_selectedMessages.contains(id)) {
      setState(() => _selectedMessages.remove(id));
    } else if (_selectedMessages.length < 50) {
      setState(() => _selectedMessages.add(id));
    } else {
      showGfToast(
        context,
        AppLocalizations.of(context).messageForwardLimit(50),
      );
    }
  }

  void _endSelection() => setState(() {
    _selecting = false;
    _selectedMessages.clear();
  });

  Future<void> _forward(List<int> ids) async {
    if (!_historyReady || _cacheCleared) return;
    if (!_sessionCurrent || _convId <= 0) return;
    _composerFocus.unfocus();
    await Navigator.of(context, rootNavigator: true).push(
      MaterialPageRoute<void>(
        builder: (_) => ForwardMessagesPage(
          convId: _convId,
          messageIds: ids,
          ownerEpoch: _sessionEpoch,
        ),
      ),
    );
    if (!_sessionCurrent) return;
    if (ref.read(chatForwardingProvider(_convId)).batch?.complete ?? false) {
      _endSelection();
    }
    unawaited(_load(silent: true));
  }

  Future<void> _sendPending(
    PendingMessage message, {
    ChatReplyTarget? reply,
    bool fromImageUpload = false,
  }) async {
    if (!_historyReady || (_attachingImage && !fromImageUpload)) return;
    final outbox = ref.read(chatOutboxProvider(widget.conv.peerId));
    if (outbox.sending) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    final peerId = widget.conv.peerId;
    final drafts = _drafts;
    final replySelection = _replySelection;
    final selectedReply = _replyTarget;
    var restoredDraft = false;
    var acknowledgedDraft = false;
    final convId = await outbox.send(message);
    if (convId != null) {
      acknowledgedDraft =
          message.draftRevision != null &&
          drafts.forPeer(peerId)?.revision == message.draftRevision;
      drafts.acknowledge(peerId, message.draftRevision, convId);
    } else if (message.state == DeliveryState.failed &&
        mounted &&
        epoch == ref.read(offlineCacheEpochProvider)) {
      // Only a real failure rehydrates: a null return for an attempt another
      // callback already claimed (same-frame double tap) must not restore.
      // The pending bubble stays for retry with the same clientMessageId, and
      // the submitted draft wins unless the user composed newer text.
      restoredDraft = drafts.restoreFailed(
        widget.conv,
        message.draftRevision,
        message.draftValue,
      );
    }
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    if (convId == null) {
      // Restore the quote only with its own draft, and never undo a later
      // selection/cancellation even when the composer text is unchanged.
      if (restoredDraft &&
          reply != null &&
          _replySelection == replySelection &&
          _replyTarget == null) {
        setState(() => _replyTarget = reply);
      }
      return;
    }
    // Retrying the failed bubble bypasses _send's preview reset. Release only
    // that acknowledged draft's quote, preserving any newer reply selection.
    if (acknowledgedDraft &&
        selectedReply != null &&
        _replySelection == replySelection &&
        selectedReply.compose(message.draftValue?.text.trim() ?? '') ==
            message.content) {
      setState(() => _replyTarget = null);
    }
    if (_convId <= 0 && convId > 0) _convId = convId;
    await _load(silent: true);
  }

  @override
  Widget build(BuildContext context) {
    ref.listen<int>(
      cacheClearEpochProvider.select(
        (epochs) => epochs[CacheCategory.chat] ?? 0,
      ),
      (_, _) {
        _loadGeneration++;
        _loadCancel?.cancel('chat cache cleared');
        _visibleReads.suspend();
        setState(() {
          _messages.clear();
          _bubbleKeys.clear();
          _cacheCleared = true;
          _fromCache = false;
          _cacheRefreshing = false;
          _loadingRequest = false;
          _loadDirty = false;
          _loading = false;
          _loadingOlder = false;
          _historyReady = false;
          _hasMoreBefore = false;
          _latestId = 0;
          _nextBeforeId = 0;
          _unseenNewMessages = false;
          _selecting = false;
          _selectedMessages.clear();
        });
      },
    );
    ref.listen(realtimeHealthyProvider, (_, healthy) {
      _syncPolling(
        _foreground &&
            _sessionCurrent &&
            TickerMode.valuesOf(context).enabled &&
            !healthy,
      );
    });
    ref.listen(realtimeInvalidationsProvider, (previous, next) {
      if (previous?.chatRevision == next.chatRevision || !_sessionCurrent) {
        return;
      }
      final convId = next.chatConvId;
      if (convId != 0 && convId != _convId) {
        return;
      }
      if (!_foreground ||
          !TickerMode.valuesOf(context).enabled ||
          !routeIsUncovered(context)) {
        return;
      }
      _seenRealtimeRevision = next.chatRevision;
      unawaited(_load(silent: true));
    });
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final String peerName = privateDisplayName(
      context,
      widget.conv.peerId,
      widget.conv.peerUsername,
      widget.conv.peerNickname,
    );
    final outbox = ref.watch(chatOutboxProvider(widget.conv.peerId));
    final forwarding = ref.watch(chatForwardingProvider(_convId));
    ref.watch(chatDraftsProvider);
    ref.listen(offlineCacheEpochProvider, (_, epoch) {
      if (epoch == _sessionEpoch) return;
      _restoringDraft = true;
      _input.clear();
      _restoringDraft = false;
      _visibleReads.suspend();
      _pollTimer?.cancel();
      _pollTimer = null;
      setState(() {
        _messages.clear();
        _bubbleKeys.clear();
        _loading = false;
        _historyReady = false;
        _unseenNewMessages = false;
        _replyTarget = null;
        _replyJumpInProgress = false;
        _replyReturnToMessageId = null;
        _replyReturnMessages = null;
        _highlightedMessageId = null;
        _selecting = false;
        _selectedMessages.clear();
      });
      _replyHighlightTimer?.cancel();
    });
    if (!_drafts.current) return const SizedBox.shrink();
    _visibleReads.changed();
    final List<ChatTimelineItem> timeline = buildChatTimeline(_messages);

    return PopScope(
      canPop: !_selecting,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && _selecting) _endSelection();
      },
      child: Scaffold(
        appBar: GfAppBar(
          leading: _selecting
              ? IconButton(
                  onPressed: _endSelection,
                  tooltip: l10n.commonCancel,
                  icon: const Icon(Icons.close),
                )
              : null,
          actions: _selecting
              ? []
              : [UserBlockButton(userId: widget.conv.peerId, moreMenu: true)],
          title: _selecting
              ? Text(l10n.messagesSelected(_selectedMessages.length))
              : Row(
                  children: <Widget>[
                    _PeerAvatarButton(
                      key: const Key('chat-peer-avatar-appbar'),
                      peerId: widget.conv.peerId,
                      label: l10n.messagesViewProfile(peerName),
                      src: resolveApiAssetUrl(widget.conv.peerAvatar),
                      size: 36,
                      ring: true,
                      alignment: Alignment.centerLeft,
                    ),
                    // 44 命中区右侧的留白即是间距,补 2 保持标题与旧版 10 的视觉间距。
                    const SizedBox(width: 2),
                    Expanded(
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          Text(
                            peerName,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          Text(
                            l10n.messagesConversation,
                            style: TextStyle(
                              color: colors.baseContent.withValues(alpha: 0.5),
                              fontSize: 11,
                              fontWeight: FontWeight.w400,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
        ),
        body: Column(
          children: <Widget>[
            if (_fromCache || _cacheCleared)
              CacheSnapshotHint(
                savedAt: _snapshotTime,
                refreshing: _cacheRefreshing,
                cleared: _cacheCleared,
                onRetry: _load,
              ),
            Expanded(
              child: NotificationListener<Notification>(
                onNotification: (notification) {
                  if (notification is ScrollStartNotification) {
                    return _onUserScroll(notification);
                  }
                  if (notification is ScrollEndNotification &&
                      notification.depth == 0) {
                    _dismissReplyReturnAtBottom();
                  }
                  if (notification is ScrollMetricsNotification &&
                      notification.depth == 0 &&
                      notification.metrics.viewportDimension !=
                          _messageViewportHeight) {
                    _messageViewportHeight =
                        notification.metrics.viewportDimension;
                    _visibleReads.changed(restartDwell: true);
                  }
                  return false;
                },
                child: Stack(
                  children: [
                    Positioned.fill(
                      child: ColoredBox(
                        key: _viewportKey,
                        color: colors.base100,
                        child: _loading
                            ? const GfLoading()
                            : _messages.isEmpty && outbox.items.isEmpty
                            ? (_loadError != null
                                  ? GfErrorRetry(
                                      message: resolveErrorMessage(
                                        l10n,
                                        _loadError!,
                                      ),
                                      onRetry: _load,
                                    )
                                  : _ChatEmptyState(
                                      title: l10n.messagesStartChat,
                                      description: l10n.messagesFirstMessageTo(
                                        privateDisplayName(
                                          context,
                                          widget.conv.peerId,
                                          widget.conv.peerUsername,
                                          widget.conv.peerNickname,
                                        ),
                                      ),
                                    ))
                            : ListView.builder(
                                controller: _scrollController,
                                physics: const ChatViewportScrollPhysics(),
                                padding: const EdgeInsets.fromLTRB(
                                  12,
                                  12,
                                  12,
                                  18,
                                ),
                                itemCount:
                                    timeline.length +
                                    outbox.items.length +
                                    (_loadingOlder ? 1 : 0),
                                itemBuilder: (BuildContext context, int index) {
                                  if (_loadingOlder && index == 0) {
                                    return const Padding(
                                      padding: EdgeInsets.only(bottom: 10),
                                      child: GfLoadingIndicator(small: true),
                                    );
                                  }
                                  final int messageIndex =
                                      index - (_loadingOlder ? 1 : 0);
                                  if (messageIndex >= timeline.length) {
                                    final pending = outbox
                                        .items[messageIndex - timeline.length];
                                    final reason = pending.error is ApiException
                                        ? resolveErrorMessage(
                                            l10n,
                                            pending.error!,
                                          )
                                        : null;
                                    final failureLabel =
                                        reason == null ||
                                            reason == l10n.commonLoadFailed
                                        ? l10n.messagesFailed
                                        : '${l10n.messagesFailed} · $reason';
                                    return Padding(
                                      key: ValueKey('pending-${pending.id}'),
                                      padding: const EdgeInsets.symmetric(
                                        vertical: 8,
                                      ),
                                      child: Column(
                                        crossAxisAlignment:
                                            CrossAxisAlignment.end,
                                        children: [
                                          ChatMessageBubble(
                                            text: pending.content,
                                            msgType: pending.msgType,
                                            mine: true,
                                          ),
                                          if (pending.state ==
                                              DeliveryState.failed) ...[
                                            Padding(
                                              padding: const EdgeInsets.only(
                                                top: 4,
                                              ),
                                              child: Text(
                                                failureLabel,
                                                style: TextStyle(
                                                  color: colors.error,
                                                  fontSize: 12,
                                                ),
                                                textAlign: TextAlign.end,
                                              ),
                                            ),
                                            TextButton.icon(
                                              onPressed: _historyReady
                                                  ? () => _sendPending(pending)
                                                  : null,
                                              icon: const GfSymbol(
                                                'circle-alert',
                                                size: 18,
                                              ),
                                              label: Text(l10n.messagesRetry),
                                            ),
                                          ] else
                                            Padding(
                                              padding: const EdgeInsets.only(
                                                top: 4,
                                              ),
                                              child: Text(
                                                pending.state ==
                                                        DeliveryState.sending
                                                    ? l10n.messagesSending
                                                    : l10n.messagesSent,
                                                style: TextStyle(
                                                  fontSize: 12,
                                                  color: colors.iconMuted,
                                                ),
                                              ),
                                            ),
                                        ],
                                      ),
                                    );
                                  }
                                  final ChatTimelineItem item =
                                      timeline[messageIndex];
                                  final ChatMessagePayload message =
                                      item.message;
                                  final DateTime? day = item.day;
                                  return AutoScrollTag(
                                    key: ValueKey(message.id),
                                    controller: _scrollController,
                                    index: messageIndex,
                                    child: AnimatedContainer(
                                      duration: GfMotion.duration(
                                        context,
                                        GfMotion.content,
                                      ),
                                      decoration: BoxDecoration(
                                        color:
                                            _highlightedMessageId == message.id
                                            ? colors.primary.withValues(
                                                alpha: 0.08,
                                              )
                                            : Colors.transparent,
                                        borderRadius: BorderRadius.circular(10),
                                      ),
                                      child: Column(
                                        children: <Widget>[
                                          if (item.showDaySeparator &&
                                              day != null)
                                            _DatePill(
                                              key: ValueKey<String>(
                                                'chat-date-separator-${message.id}',
                                              ),
                                              label: formatChatDayLabel(
                                                day,
                                                l10n: l10n,
                                              ),
                                            ),
                                          Semantics(
                                            container: _selecting,
                                            excludeSemantics: _selecting,
                                            enabled: _selecting ? true : null,
                                            checked: _selecting
                                                ? _selectedMessages.contains(
                                                    message.id,
                                                  )
                                                : null,
                                            label: _selecting
                                                ? '${_replySender(message)}: ${chatReplyExcerpt(message.content)}'
                                                : null,
                                            onTap: _selecting
                                                ? () =>
                                                      _toggleMessage(message.id)
                                                : null,
                                            child: Row(
                                              children: [
                                                _MessageSelectionControl(
                                                  visible: _selecting,
                                                  selected: _selectedMessages
                                                      .contains(message.id),
                                                  onChanged: () =>
                                                      _toggleMessage(
                                                        message.id,
                                                      ),
                                                ),
                                                Expanded(
                                                  child: GestureDetector(
                                                    behavior:
                                                        HitTestBehavior.opaque,
                                                    onTap: _selecting
                                                        ? () => _toggleMessage(
                                                            message.id,
                                                          )
                                                        : null,
                                                    child: IgnorePointer(
                                                      ignoring: _selecting,
                                                      child: _MessageRow(
                                                        bubbleKey: _bubbleKeys
                                                            .putIfAbsent(
                                                              message.id,
                                                              GlobalKey.new,
                                                            ),
                                                        message: message,
                                                        peerId:
                                                            widget.conv.peerId,
                                                        peerProfileLabel: l10n
                                                            .messagesViewProfile(
                                                              peerName,
                                                            ),
                                                        peerAvatar: widget
                                                            .conv
                                                            .peerAvatar,
                                                        viewerAvatar:
                                                            widget.viewerAvatar,
                                                        showTime:
                                                            item.showTimestamp,
                                                        onSwipeReply: _selecting
                                                            ? null
                                                            : () => _replyTo(
                                                                message,
                                                              ),
                                                        replyToMessageId: message
                                                            .replyToMessageId,
                                                        onQuoteTap:
                                                            _selecting ||
                                                                message.replyToMessageId ==
                                                                    null
                                                            ? null
                                                            : () => _jumpToQuotedMessage(
                                                                message
                                                                    .replyToMessageId!,
                                                                message.id,
                                                              ),
                                                        onLongPress: () =>
                                                            unawaited(
                                                              _showMessageActions(
                                                                message,
                                                              ),
                                                            ),
                                                      ),
                                                    ),
                                                  ),
                                                ),
                                              ],
                                            ),
                                          ),
                                        ],
                                      ),
                                    ),
                                  );
                                },
                              ),
                      ),
                    ),
                    if (_replyReturnToMessageId != null)
                      Positioned(
                        top: 16,
                        left: 24,
                        right: 0,
                        child: Align(
                          alignment: Alignment.centerRight,
                          child: ElevatedButton.icon(
                            key: const Key('chat-reply-return'),
                            onPressed: _replyJumpInProgress
                                ? null
                                : _returnToReplySource,
                            style: ElevatedButton.styleFrom(
                              backgroundColor: colors.base100,
                              foregroundColor: colors.primary,
                              surfaceTintColor: Colors.transparent,
                              shadowColor: colors.baseContent.withValues(
                                alpha: 0.08,
                              ),
                              elevation: 1,
                              minimumSize: const Size(0, 44),
                              padding: const EdgeInsets.symmetric(
                                horizontal: 16,
                                vertical: 10,
                              ),
                              textStyle: const TextStyle(
                                fontSize: 13,
                                fontWeight: FontWeight.w600,
                              ),
                              side: BorderSide(color: colors.line),
                              shape: const RoundedRectangleBorder(
                                borderRadius: BorderRadius.horizontal(
                                  left: Radius.circular(24),
                                ),
                              ),
                            ),
                            icon: const GfSymbol('arrow-down', size: 18),
                            label: Text(l10n.messagesReturnToReplySource),
                          ),
                        ),
                      ),
                    if (_unseenNewMessages)
                      Positioned(
                        right: 12,
                        bottom: 12,
                        child: FloatingActionButton.small(
                          key: const Key('chat-new-messages'),
                          heroTag: null,
                          tooltip: l10n.messagesNewMessages,
                          onPressed: _scrollToBottom,
                          child: const GfSymbol('arrow-down'),
                        ),
                      ),
                  ],
                ),
              ),
            ),
            if (_selecting)
              SafeArea(
                top: false,
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: SizedBox(
                    width: double.infinity,
                    child: FilledButton.icon(
                      onPressed: _selectedMessages.isEmpty
                          ? null
                          : () => _forward(_selectedMessages.toList()),
                      icon: const Icon(Icons.forward),
                      label: Text(l10n.messageForward),
                    ),
                  ),
                ),
              )
            else
              SafeArea(
                top: false,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (_readSyncFailed)
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 12),
                        child: Row(
                          children: [
                            Expanded(
                              child: Text(
                                _readSyncUnsupported
                                    ? l10n.messagesReadUnavailable
                                    : l10n.messagesReadSyncFailed,
                              ),
                            ),
                            if (!_readSyncUnsupported)
                              TextButton(
                                onPressed: _canObserve
                                    ? _visibleReads.retry
                                    : null,
                                child: Text(l10n.commonRetry),
                              ),
                          ],
                        ),
                      ),
                    if (!_historyReady &&
                        !_loading &&
                        !_fromCache &&
                        !_cacheCleared)
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 12),
                        child: Row(
                          children: [
                            Expanded(child: Text(l10n.commonLoadFailed)),
                            TextButton.icon(
                              onPressed: _load,
                              icon: const GfSymbol('refresh-cw', size: 18),
                              label: Text(l10n.commonRetry),
                            ),
                          ],
                        ),
                      ),
                    if (forwarding.unfinished)
                      ListTile(
                        dense: true,
                        title: Text(l10n.messageForwardPending),
                        trailing: TextButton(
                          onPressed: () =>
                              _forward(forwarding.batch!.messageIds),
                          child: Text(l10n.messageForwardResume),
                        ),
                      ),
                    _ChatDraftStatus(
                      drafts: _drafts,
                      peerId: widget.conv.peerId,
                    ),
                    if (outbox.imageRecoveryError != null)
                      ListTile(
                        title: Text(l10n.commonLoadFailed),
                        trailing: TextButton(
                          onPressed: outbox.restoreImages,
                          child: Text(l10n.commonRetry),
                        ),
                      ),
                    if (_replyTarget != null)
                      _ReplyPreview(
                        target: _replyTarget!,
                        cancelLabel: l10n.messageReplyCancel,
                        onCancel: () => setState(() {
                          _replySelection++;
                          _replyTarget = null;
                        }),
                      ),
                    GfChatInput(
                      controller: _input,
                      focusNode: _composerFocus,
                      previewBuilder: (text) =>
                          StickerDraftPreview(content: text),
                      accessoryBuilder: (insert) =>
                          StickerPicker(onInsert: insert),
                      onAttach: _showAttachments,
                      attachLabel: l10n.messagesAttachments,
                      clearOnSend: false,
                      enabled: _drafts.current,
                      hintText: l10n.messagesInputHint,
                      sendLabel: l10n.commonSend,
                      emojiLabel: l10n.messagesEmoji,
                      keyboardLabel: l10n.messagesKeyboard,
                      canSend:
                          _historyReady && !_attachingImage && !outbox.sending,
                      onSend: _send,
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _ConversationList extends StatelessWidget {
  const _ConversationList({
    required this.controller,
    this.padding = EdgeInsets.zero,
    required this.items,
    required this.drafts,
    required this.canOpenNewConversation,
    required this.query,
    required this.emptyMessage,
    required this.emptyDescription,
    required this.actionLabel,
    required this.onStart,
    required this.onOpen,
  });

  final ScrollController controller;
  final EdgeInsets padding;
  final List<ChatItemPayload> items;
  final Map<int, ChatDraft> drafts;
  final bool canOpenNewConversation;
  final String query;
  final String emptyMessage;
  final String emptyDescription;
  final String actionLabel;
  final VoidCallback onStart;
  final ValueChanged<ChatItemPayload> onOpen;

  @override
  Widget build(BuildContext context) {
    final String normalized = query.trim().toLowerCase();
    final List<ChatItemPayload> filtered = items.where((ChatItemPayload item) {
      return normalized.isEmpty ||
          item.peerUsername.toLowerCase().contains(normalized) ||
          item.lastMsg.toLowerCase().contains(normalized) ||
          (drafts[item.peerId]?.value.text.toLowerCase().contains(normalized) ??
              false);
    }).toList();
    if (filtered.isEmpty) {
      return _ConversationEmptyState(
        title: emptyMessage,
        description: emptyDescription,
        actionLabel: actionLabel,
        onStart: onStart,
      );
    }
    final AppLocalizations l10n = AppLocalizations.of(context);
    return ListView.separated(
      padding: padding,
      controller: controller,
      itemCount: filtered.length,
      separatorBuilder: (_, _) => const GfDivider(),
      itemBuilder: (BuildContext context, int index) {
        final ChatItemPayload conversation = filtered[index];
        final draft = drafts[conversation.peerId];
        final messagePreview = isChatImagePreviewUrl(conversation.lastMsg)
            ? '[${l10n.messagesImage}]'
            : stickerPreviewLabel(
                conversation.lastMsg,
              ).replaceAll(RegExp(r'\s+'), ' ').trim();
        return GfConversationRow(
          avatarUrl: resolveApiAssetUrl(conversation.peerAvatar),
          name: privateDisplayName(
            context,
            conversation.peerId,
            conversation.peerUsername,
            conversation.peerNickname,
          ),
          lastMessage: draft != null
              ? '${l10n.messagesDraftLabel} · ${stickerPreviewLabel(draft.value.text)}'
              : conversation.lastMsg.isEmpty
              ? l10n.messagesNoMessagesYet
              : messagePreview.startsWith('[Chat history]')
              ? messagePreview.replaceAll(
                  '[Chat history]',
                  '[${l10n.messageForwardHistory}]',
                )
              : messagePreview,
          time: formatChatTime(conversation.lastMsgTime, l10n: l10n),
          unreadCount: conversation.unreadCount,
          unreadLabel: l10n.notificationsUnread,
          onTap: conversation.convId == 0 && !canOpenNewConversation
              ? null
              : () => onOpen(conversation),
        );
      },
    );
  }
}

class _ChatDraftStatus extends StatelessWidget {
  const _ChatDraftStatus({required this.drafts, this.peerId});
  final ChatDrafts drafts;
  final int? peerId;
  @override
  Widget build(BuildContext context) {
    if (!drafts.current) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    if (drafts.error != null) {
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: Row(
          children: [
            Expanded(child: Text(l10n.messagesDraftStorageFailed)),
            TextButton(onPressed: drafts.flush, child: Text(l10n.commonRetry)),
          ],
        ),
      );
    }
    if (peerId == null || !(drafts.forPeer(peerId!)?.hasText ?? false)) {
      return const SizedBox.shrink();
    }
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 0),
      child: Text(
        drafts.isDirty(peerId!) ? l10n.draftLocalSaving : l10n.draftLocalSaved,
      ),
    );
  }
}

/// 输入框上方的引用预览:发送者 + 有界摘要,可单独取消。
class _ReplyPreview extends StatelessWidget {
  const _ReplyPreview({
    required this.target,
    required this.cancelLabel,
    required this.onCancel,
  });

  final ChatReplyTarget target;
  final String cancelLabel;
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Padding(
      key: const Key('chat-reply-preview'),
      padding: const EdgeInsets.fromLTRB(12, 4, 4, 0),
      child: Container(
        padding: const EdgeInsets.fromLTRB(10, 6, 0, 6),
        decoration: BoxDecoration(
          color: colors.base200,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          children: <Widget>[
            Container(
              width: 3,
              height: 34,
              decoration: BoxDecoration(
                color: colors.primary,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  if (target.sender.isNotEmpty)
                    Text(
                      target.sender,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.primary,
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  Text(
                    localizedChatReplyExcerpt(
                      target.excerpt,
                      imageLabel: AppLocalizations.of(context).messagesImage,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: colors.baseContent.withValues(alpha: 0.7),
                      fontSize: 13,
                    ),
                  ),
                ],
              ),
            ),
            GfIconButton(
              symbol: 'x',
              tooltip: cancelLabel,
              size: 44,
              iconSize: 18,
              onPressed: onCancel,
            ),
          ],
        ),
      ),
    );
  }
}

class _NewChatSheet extends StatefulWidget {
  const _NewChatSheet({
    required this.users,
    required this.title,
    required this.searchHint,
    required this.emptyMessage,
  });

  final List<UserConnectionPayload> users;
  final String title;
  final String searchHint;
  final String emptyMessage;

  @override
  State<_NewChatSheet> createState() => _NewChatSheetState();
}

class _NewChatSheetState extends State<_NewChatSheet> {
  final TextEditingController _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final String query = _search.text.trim().toLowerCase();
    final List<UserConnectionPayload> users = widget.users.where((user) {
      return query.isEmpty ||
          user.username.toLowerCase().contains(query) ||
          user.nickname.toLowerCase().contains(query);
    }).toList();
    return SafeArea(
      top: false,
      child: Column(
        children: <Widget>[
          ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 48),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Row(
                children: <Widget>[
                  Expanded(
                    child: Text(
                      widget.title,
                      style: TextStyle(
                        color: colors.baseContent,
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  GfIconButton(
                    symbol: 'x',
                    tooltip: AppLocalizations.of(context).commonClose,
                    size: 44,
                    iconSize: 18,
                    onPressed: () => Navigator.pop(context),
                  ),
                ],
              ),
            ),
          ),
          Container(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
            decoration: BoxDecoration(
              border: Border(bottom: BorderSide(color: colors.line)),
            ),
            child: GfSearchField(
              controller: _search,
              hintText: widget.searchHint,
              clearLabel: AppLocalizations.of(context).courseCopyClearSearch,
              autofocus: true,
              onChanged: (_) => setState(() {}),
            ),
          ),
          Expanded(
            child: users.isEmpty
                ? _NewChatEmptyState(message: widget.emptyMessage)
                : ListView.builder(
                    padding: const EdgeInsets.all(8),
                    itemCount: users.length,
                    itemBuilder: (BuildContext context, int index) {
                      final UserConnectionPayload user = users[index];
                      return _NewChatUserRow(
                        user: user,
                        onTap: () => Navigator.pop(context, user),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

class _ConversationEmptyState extends StatelessWidget {
  const _ConversationEmptyState({
    required this.title,
    required this.description,
    required this.actionLabel,
    required this.onStart,
  });

  final String title;
  final String description;
  final String actionLabel;
  final VoidCallback onStart;

  @override
  Widget build(BuildContext context) {
    return GfEmpty(
      symbol: 'message-circle',
      message: title,
      description: description,
      action: GfButton(
        label: actionLabel,
        icon: const GfSymbol('message-circle', size: 20),
        onPressed: onStart,
      ),
    );
  }
}

class _NewChatEmptyState extends StatelessWidget {
  const _NewChatEmptyState({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            GfSymbol(
              'users-round',
              size: 32,
              color: colors.baseContent.withValues(alpha: 0.30),
            ),
            const SizedBox(height: 10),
            Text(
              message,
              textAlign: TextAlign.center,
              style: TextStyle(
                color: colors.baseContent.withValues(alpha: 0.55),
                fontSize: 14,
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _NewChatUserRow extends StatelessWidget {
  const _NewChatUserRow({required this.user, required this.onTap});

  final UserConnectionPayload user;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final String name = privateDisplayName(
      context,
      user.id,
      user.username,
      user.nickname,
    );
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(radii.field),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            children: <Widget>[
              GfAvatar(
                src: resolveApiAssetUrl(user.avatarUrl),
                size: 40,
                ring: true,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '@${user.username}',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.55),
                        fontSize: 12,
                      ),
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
}

class _ChatEmptyState extends StatelessWidget {
  const _ChatEmptyState({required this.title, required this.description});

  final String title;
  final String description;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            GfSymbol(
              'message-circle',
              size: 40,
              color: colors.baseContent.withValues(alpha: 0.32),
            ),
            const SizedBox(height: 12),
            Text(
              title,
              style: TextStyle(
                color: colors.baseContent,
                fontSize: 16,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              description,
              textAlign: TextAlign.center,
              style: TextStyle(
                color: colors.baseContent.withValues(alpha: 0.55),
                fontSize: 14,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DatePill extends StatelessWidget {
  const _DatePill({super.key, required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    // 日期分隔按标题语义暴露(Web 用 <h2>),读屏不会把它当作一条消息。
    return Semantics(
      container: true,
      header: true,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
          decoration: BoxDecoration(
            color: colors.base300,
            borderRadius: BorderRadius.circular(999),
          ),
          child: Text(
            label,
            style: TextStyle(
              color: colors.baseContent.withValues(alpha: 0.55),
              fontSize: 11,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}

/// 对方头像的主页入口:44×44 命中区包住视觉头像,头像本身不位移、不缩放,
/// 命中区只向头像旁的空白扩展,点击进入 `/u/{peerId}`。
class _PeerAvatarButton extends StatelessWidget {
  const _PeerAvatarButton({
    super.key,
    required this.peerId,
    required this.label,
    required this.src,
    required this.size,
    this.ring = false,
    this.alignment = Alignment.center,
  });

  final int peerId;
  final String label;
  final String src;
  final double size;
  final bool ring;
  final Alignment alignment;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      // 独立语义节点:头像标签不会与同行的消息文本/标题合并成一个按钮。
      container: true,
      button: true,
      label: label,
      child: InkWell(
        onTap: () => context.push('/u/$peerId'),
        customBorder: const CircleBorder(),
        child: SizedBox(
          width: 44,
          height: 44,
          child: Align(
            alignment: alignment,
            child: GfAvatar(src: src, size: size, ring: ring),
          ),
        ),
      ),
    );
  }
}

/// Keep the message subtree mounted while revealing the selection rail. Native
/// checkboxes retain their checked-state animation and keyboard interaction.
class _MessageSelectionControl extends StatelessWidget {
  const _MessageSelectionControl({
    required this.visible,
    required this.selected,
    required this.onChanged,
  });

  final bool visible;
  final bool selected;
  final VoidCallback onChanged;

  @override
  Widget build(BuildContext context) => IgnorePointer(
    ignoring: !visible,
    child: ExcludeFocus(
      excluding: !visible,
      child: ExcludeSemantics(
        excluding: !visible,
        child: TweenAnimationBuilder<double>(
          tween: Tween(end: visible ? 1 : 0),
          duration: GfMotion.duration(context, GfMotion.layout),
          curve: GfMotion.layoutCurve,
          child: SizedBox(
            width: 44,
            height: 48,
            child: Checkbox(
              shape: const CircleBorder(),
              value: selected,
              onChanged: (_) => onChanged(),
            ),
          ),
          builder: (context, progress, child) => Offstage(
            offstage: progress == 0,
            child: ClipRect(
              child: Align(
                alignment: AlignmentDirectional.centerEnd,
                widthFactor: progress,
                child: Opacity(opacity: progress, child: child),
              ),
            ),
          ),
        ),
      ),
    ),
  );
}

class _MessageRow extends ConsumerWidget {
  const _MessageRow({
    this.bubbleKey,
    required this.message,
    required this.peerId,
    required this.peerProfileLabel,
    required this.peerAvatar,
    required this.viewerAvatar,
    this.onLongPress,
    this.onSwipeReply,
    this.replyToMessageId,
    this.onQuoteTap,
    this.showTime = true,
  });

  final GlobalKey? bubbleKey;
  final ChatMessagePayload message;
  final int peerId;
  final String peerProfileLabel;
  final String peerAvatar;
  final String viewerAvatar;
  final VoidCallback? onLongPress;
  final VoidCallback? onSwipeReply;
  final int? replyToMessageId;
  final VoidCallback? onQuoteTap;

  /// 由 [buildChatTimeline] 决定:只有分组首条消息显示时刻。
  final bool showTime;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ChatMessageRow(
      mine: message.isSelf,
      avatar: message.isSelf
          ? GfAvatar(src: resolveApiAssetUrl(viewerAvatar), size: 32)
          : _PeerAvatarButton(
              key: Key('chat-peer-avatar-${message.id}'),
              peerId: peerId,
              label: peerProfileLabel,
              src: resolveApiAssetUrl(peerAvatar),
              size: 32,
              alignment: Alignment.topLeft,
            ),
      child: ChatMessageBubble(
        bubbleKey: bubbleKey,
        text: message.content,
        msgType: message.msgType,
        mine: message.isSelf,
        time: showTime ? formatChatClock(message.createdAt) : null,
        maxWidthFactor: 0.74,
        selectable: message.msgType != 4 && message.forwarded == null,
        onLongPress: onLongPress,
        onSwipeReply: onSwipeReply,
        replyToMessageId: replyToMessageId,
        onQuoteTap: onQuoteTap,
        content: message.forwarded == null
            ? null
            : ForwardedMessageCard(bundle: message.forwarded!),
      ),
    );
  }
}

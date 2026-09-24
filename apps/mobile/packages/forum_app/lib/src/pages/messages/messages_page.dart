import '../../private_notes.dart';
import '../../widgets/root_surface.dart';
import '../../messages/chat_outbox.dart';
import '../../messages/visible_chat_reads.dart';
import '../../navigation/route_visibility.dart';
import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import '../../asset_url.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../format.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import '../../widgets/sticker_message_span.dart';

/// 私信(IM)页(web messages.index 的移动端形态):
/// 会话列表 + 消息游标分页 + 15s 轮询 + 已读回执 + 离线缓存 + 发起新会话。
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

class _MessagesPageState extends ConsumerState<MessagesPage> {
  AsyncValue<List<ChatItemPayload>> _conversations = const AsyncValue.loading();
  List<UserConnectionPayload> _suggestedUsers = const [];
  String _viewerAvatar = '';
  final TextEditingController _conversationSearch = TextEditingController();
  Timer? _pollTimer;
  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();
  late final GfTabScrollRegistry _tabScrollRegistry;
  bool _pollingConfigured = false;
  ChatItemPayload? _targetConversation;

  @override
  void initState() {
    super.initState();
    _tabScrollRegistry = ref.read(tabScrollRegistryProvider);
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
    _syncPolling(TickerMode.valuesOf(context).enabled);
  }

  void _syncPolling(bool shouldPoll) {
    final bool wasConfigured = _pollingConfigured;
    _pollingConfigured = true;
    if (shouldPoll == (_pollTimer != null)) return;

    _pollTimer?.cancel();
    _pollTimer = null;
    if (!shouldPoll) return;
    if (wasConfigured) _load(silent: true);
    _pollTimer = Timer.periodic(
      const Duration(seconds: 15),
      (_) => _load(silent: true),
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
    _conversationSearch.dispose();
    _tabScrollRegistry.unregister(
      GfShellDestination.messages,
      _scrollToTopController,
    );
    super.dispose();
  }

  Future<void> _load({bool silent = false}) async {
    // 记录发起时的缓存世代;401/登出/换账号后世代自增,返回时丢弃旧会话数据。
    final int epoch = ref.read(offlineCacheEpochProvider);
    try {
      final props = await ref.read(pageRepositoryProvider).fetch('/messages');
      final MessagesPageProps? parsed = parsePageProps<MessagesPageProps>(
        props,
      );
      final List<ChatItemPayload> items = parsed?.conversations ?? [];
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() {
          _conversations = AsyncValue.data(items);
          _suggestedUsers = parsed?.suggestedUsers ?? const [];
          _viewerAvatar = resolveApiAssetUrl(props.layout.viewer.avatarUrl);
          _targetConversation = _targetConversationFor(items);
        });
      }
      // 会话列表在单事务中批量写入离线缓存(断网可读);仅当前世代允许写入,
      // 避免 401/登出后旧会话在途响应把上一账号数据写回刚清空的缓存。
      if (epoch == ref.read(offlineCacheEpochProvider)) {
        await ref.read(offlineChatCacheProvider).putConversations(items);
      }
    } catch (e, st) {
      // 网络失败:回退离线缓存的会话列表。
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      // 无会话令牌(如 401 后进程被杀重启)时不得回退上一账号残留缓存。
      if (!await hasSessionToken(ref.read(tokenStorageProvider))) {
        if (!silent && mounted) {
          setState(() => _conversations = AsyncValue.error(e, st));
        }
        return;
      }
      try {
        final cached = await ref
            .read(offlineChatCacheProvider)
            .getConversations();
        // 读缓存期间会话可能已切换,再次校验世代再更新 UI。
        if (epoch != ref.read(offlineCacheEpochProvider)) return;
        if (cached.isNotEmpty) {
          setState(() {
            _conversations = AsyncValue.data(cached);
            _targetConversation = _targetConversationFor(cached);
          });
          return;
        }
      } catch (_) {
        // 缓存不可用时继续走错误态。
      }
      if (!silent && mounted) {
        setState(() => _conversations = AsyncValue.error(e, st));
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
    await Navigator.of(context, rootNavigator: true).push(
      MaterialPageRoute<void>(
        builder: (_) =>
            _ConversationPage(conv: conv, viewerAvatar: _viewerAvatar),
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
        peerUsername: selected.nickname.isEmpty
            ? selected.username
            : selected.nickname,
        peerAvatar: resolveApiAssetUrl(selected.avatarUrl),
        lastMsg: '',
        lastMsgTime: '',
        unreadCount: 0,
        convId: 0,
        peerUrl: selected.url,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ChatItemPayload? targetConversation = _targetConversation;
    if (targetConversation != null) {
      return _ConversationPage(
        key: ValueKey<int>(targetConversation.peerId),
        conv: targetConversation,
        viewerAvatar: _viewerAvatar,
      );
    }
    return RootSurface(
      title: l10n.messagesTitle,
      actionLabel: l10n.messagesNew,
      actionSymbol: 'message-circle',
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
      body: (top, bottom) => _conversations.when(
        loading: () => const GfLoading(),
        error: (e, _) =>
            GfErrorRetry(message: resolveErrorMessage(l10n, e), onRetry: _load),
        data: (items) => GfScrollToTop(
          semanticLabel: l10n.commonBackToTop,
          showButton: false,
          controller: _scrollToTopController,
          builder: (_, controller) => _ConversationList(
            controller: controller,
            padding: EdgeInsets.only(top: top, bottom: bottom),
            items: items,
            query: _conversationSearch.text,
            emptyMessage: l10n.messagesEmpty,
            emptyDescription: l10n.messagesEmptyDescription,
            actionLabel: l10n.messagesNew,
            onStart: _startNewChat,
            onOpen: _openConversation,
          ),
        ),
      ),
    );
  }
}

/// 单会话聊天页。
class _ConversationPage extends ConsumerStatefulWidget {
  const _ConversationPage({
    super.key,
    required this.conv,
    required this.viewerAvatar,
  });

  final ChatItemPayload conv;
  final String viewerAvatar;

  @override
  ConsumerState<_ConversationPage> createState() => _ConversationPageState();
}

class _ConversationPageState extends ConsumerState<_ConversationPage>
    with WidgetsBindingObserver {
  final List<ChatMessagePayload> _messages = [];
  final TextEditingController _input = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  bool _loading = true;
  bool _loadingOlder = false;
  bool _historyReady = false;
  bool _hasMoreBefore = false;
  Timer? _pollTimer;
  late int _convId;
  int _latestId = 0;
  int _nextBeforeId = 0;
  bool _pollingConfigured = false;
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

  bool get _sessionCurrent =>
      mounted && _sessionEpoch == ref.read(offlineCacheEpochProvider);
  bool get _canObserve =>
      _sessionCurrent &&
      _foreground &&
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
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _visibleReads.suspend();
    _syncPolling(_foreground && TickerMode.valuesOf(context).enabled);
    if (_foreground) _visibleReads.changed(restartDwell: true);
  }

  @override
  void didChangeMetrics() => _visibleReads.changed(restartDwell: true);

  @override
  void initState() {
    super.initState();
    _convId = widget.conv.convId > 0
        ? widget.conv.convId
        : ref.read(chatOutboxProvider(widget.conv.peerId)).conversationId;
    _sessionEpoch = ref.read(offlineCacheEpochProvider);
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
    _load();
    _scrollController.addListener(_onScroll);
    // 表情包库未就绪时拉一次(会话级缓存),完成后刷新气泡分段渲染。
    _ensureStickers();
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
    _syncPolling(
      _foreground && _sessionCurrent && TickerMode.valuesOf(context).enabled,
    );
    _visibleReads.changed(restartDwell: true);
  }

  void _syncPolling(bool shouldPoll) {
    final bool wasConfigured = _pollingConfigured;
    _pollingConfigured = true;
    if (shouldPoll == (_pollTimer != null)) return;

    _pollTimer?.cancel();
    _pollTimer = null;
    if (!shouldPoll) return;
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
    _pollTimer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    routeVisibilityChanges.removeListener(_visibilityChanged);
    _visibleReads.dispose();
    _input.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    _visibleReads.changed(restartDwell: true);
    if (_unseenNewMessages &&
        _scrollController.hasClients &&
        _scrollController.position.extentAfter < 24) {
      setState(() => _unseenNewMessages = false);
    }
    if (_scrollController.hasClients &&
        _scrollController.position.pixels < 80) {
      _loadOlder();
    }
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
    if (!_sessionCurrent) return;
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
    if (!silent && !_historyReady && mounted) {
      setState(() => _loading = true);
    }
    // 记录发起时的缓存世代;401/登出/换账号后世代自增,返回时丢弃旧会话数据。
    final int epoch = ref.read(offlineCacheEpochProvider);
    try {
      final bool initial = _latestId == 0;
      final ChatMessagesResponse resp = await ref
          .read(chatRepositoryProvider)
          .getMessages(convId: _convId, afterId: _latestId);
      final bool pinnedToBottom =
          !_scrollController.hasClients ||
          _scrollController.position.extentAfter < 80;
      final Set<int> seenIds = _messages
          .map((ChatMessagePayload message) => message.id)
          .toSet();
      final List<ChatMessagePayload> newMessages = resp.list
          .where((ChatMessagePayload message) => seenIds.add(message.id))
          .toList();
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() {
          if (newMessages.isNotEmpty) {
            _messages.addAll(newMessages);
            _messages.sort((a, b) => a.id.compareTo(b.id));
            if (!initial &&
                (!pinnedToBottom || !_canObserve) &&
                newMessages.any((m) => !m.isSelf && m.isRead == 0)) {
              _unseenNewMessages = true;
            }
          }
          if (resp.latestId > _latestId) _latestId = resp.latestId;
          // Only server history establishes a safe lower bound for new sends.
          // Cached history may omit a newer, identical self-authored message.
          _historyReady = true;
          // An afterId response describes its newer window, not the oldest
          // loaded cursor. Preserve older history pagination across live pulls.
          if (initial) {
            _hasMoreBefore = resp.hasMoreBefore;
            _nextBeforeId = resp.nextBeforeId;
          }
          _loading = false;
        });
        ref.read(chatOutboxProvider(widget.conv.peerId)).reconcile(_messages);
        if (initial || pinnedToBottom) _scrollToBottom();
      }
      if (newMessages.isNotEmpty &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        // 只持久化真正新增的消息，避免轮询重复写缓存和重复上报已读。
        await ref
            .read(offlineChatCacheProvider)
            .putMessages(_convId, newMessages);
      }
    } catch (_) {
      // 网络失败:回退离线缓存消息。
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      // 无会话令牌(如 401 后进程被杀重启)时不得回退上一账号残留缓存。
      if (!await hasSessionToken(ref.read(tokenStorageProvider))) {
        if (mounted && (!silent || !_historyReady)) {
          setState(() => _loading = false);
        }
        return;
      }
      try {
        final cached = await ref
            .read(offlineChatCacheProvider)
            .getMessages(_convId);
        // 读缓存期间会话可能已切换,再次校验世代再更新 UI。
        if (epoch != ref.read(offlineCacheEpochProvider)) return;
        if (cached.isNotEmpty) {
          setState(() {
            _messages
              ..clear()
              ..addAll(cached);
            _loading = false;
          });
          return;
        }
      } catch (_) {
        // 缓存不可用。
      }
      if (mounted && (!silent || !_historyReady)) {
        setState(() => _loading = false);
      }
    }
  }

  Future<void> _loadOlder() async {
    if (!_sessionCurrent ||
        _loadingOlder ||
        !_hasMoreBefore ||
        _nextBeforeId <= 0) {
      return;
    }
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _loadingOlder = true);
    try {
      final ChatMessagesResponse resp = await ref
          .read(chatRepositoryProvider)
          .getMessages(convId: _convId, beforeId: _nextBeforeId);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
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
      if (mounted) setState(() => _loadingOlder = false);
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

  Future<void> _send(String value) async {
    if (!_historyReady) return;
    final text = value.trim();
    if (text.isEmpty) return;
    final outbox = ref.read(chatOutboxProvider(widget.conv.peerId));
    final message = outbox.enqueue(text, _latestId);
    _scrollToBottom();
    await _sendPending(message);
  }

  Future<void> _sendPending(PendingMessage message) async {
    if (!_historyReady) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    final convId = await ref
        .read(chatOutboxProvider(widget.conv.peerId))
        .send(message);
    if (!mounted ||
        epoch != ref.read(offlineCacheEpochProvider) ||
        convId == null) {
      return;
    }
    if (_convId <= 0 && convId > 0) _convId = convId;
    await _load(silent: true);
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final outbox = ref.watch(chatOutboxProvider(widget.conv.peerId));
    ref.listen(offlineCacheEpochProvider, (_, epoch) {
      if (epoch == _sessionEpoch) return;
      _visibleReads.suspend();
      _pollTimer?.cancel();
      _pollTimer = null;
      setState(() {
        _messages.clear();
        _bubbleKeys.clear();
        _loading = false;
        _historyReady = false;
        _unseenNewMessages = false;
      });
    });
    _visibleReads.changed();

    return Scaffold(
      appBar: GfAppBar(
        title: Row(
          children: <Widget>[
            GfAvatar(
              src: resolveApiAssetUrl(widget.conv.peerAvatar),
              size: 36,
              ring: true,
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    privateDisplayName(
                      context,
                      widget.conv.peerId,
                      '',
                      widget.conv.peerUsername,
                    ),
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
          Expanded(
            child: NotificationListener<ScrollStartNotification>(
              onNotification: _onUserScroll,
              child: Stack(
                children: [
                  Positioned.fill(
                    child: ColoredBox(
                      key: _viewportKey,
                      color: colors.base100,
                      child: _loading
                          ? const GfLoading()
                          : _messages.isEmpty && outbox.items.isEmpty
                          ? _ChatEmptyState(
                              title: l10n.messagesStartChat,
                              description: l10n.messagesFirstMessageTo(
                                privateDisplayName(
                                  context,
                                  widget.conv.peerId,
                                  '',
                                  widget.conv.peerUsername,
                                ),
                              ),
                            )
                          : ListView.builder(
                              controller: _scrollController,
                              padding: const EdgeInsets.fromLTRB(
                                12,
                                12,
                                12,
                                18,
                              ),
                              itemCount:
                                  _messages.length +
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
                                if (messageIndex >= _messages.length) {
                                  final pending = outbox
                                      .items[messageIndex - _messages.length];
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
                                        GfMessageBubble(
                                          text: pending.content,
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
                                            icon: const Icon(
                                              Icons.error_outline,
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
                                final ChatMessagePayload message =
                                    _messages[messageIndex];
                                final bool startsDay =
                                    messageIndex == 0 ||
                                    formatDate(
                                          _messages[messageIndex - 1].createdAt,
                                        ) !=
                                        formatDate(message.createdAt);
                                return Column(
                                  children: <Widget>[
                                    if (startsDay)
                                      _DatePill(
                                        date: formatDate(message.createdAt),
                                      ),
                                    _MessageRow(
                                      bubbleKey: _bubbleKeys.putIfAbsent(
                                        message.id,
                                        GlobalKey.new,
                                      ),
                                      message: message,
                                      peerAvatar: widget.conv.peerAvatar,
                                      viewerAvatar: widget.viewerAvatar,
                                    ),
                                  ],
                                );
                              },
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
                        child: const Icon(Icons.arrow_downward),
                      ),
                    ),
                ],
              ),
            ),
          ),
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
                            onPressed: _canObserve ? _visibleReads.retry : null,
                            child: Text(l10n.commonRetry),
                          ),
                      ],
                    ),
                  ),
                if (!_historyReady && !_loading)
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    child: Row(
                      children: [
                        Expanded(child: Text(l10n.commonLoadFailed)),
                        TextButton.icon(
                          onPressed: _load,
                          icon: const Icon(Icons.refresh, size: 18),
                          label: Text(l10n.commonRetry),
                        ),
                      ],
                    ),
                  ),
                GfChatInput(
                  controller: _input,
                  hintText: l10n.messagesInputHint,
                  sendLabel: l10n.commonSend,
                  canSend: _historyReady,
                  onSend: _send,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ConversationList extends StatelessWidget {
  const _ConversationList({
    required this.controller,
    this.padding = EdgeInsets.zero,
    required this.items,
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
          item.lastMsg.toLowerCase().contains(normalized);
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
        return GfConversationRow(
          avatarUrl: resolveApiAssetUrl(conversation.peerAvatar),
          name: privateDisplayName(
            context,
            conversation.peerId,
            '',
            conversation.peerUsername,
          ),
          lastMessage: conversation.lastMsg.isEmpty
              ? l10n.messagesNoMessagesYet
              : stickerPreviewLabel(conversation.lastMsg),
          time: formatChatTime(conversation.lastMsgTime, l10n: l10n),
          unreadCount: conversation.unreadCount,
          onTap: () => onOpen(conversation),
        );
      },
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
          Padding(
            padding: const EdgeInsets.only(top: 8, bottom: 4),
            child: Container(
              width: 36,
              height: 4,
              decoration: BoxDecoration(
                color: colors.baseContent.withValues(alpha: 0.18),
                borderRadius: BorderRadius.circular(999),
              ),
            ),
          ),
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
                    icon: Icons.close,
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
      icon: Icons.chat_bubble_outline,
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
            Icon(
              Icons.person_search_outlined,
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
            Icon(
              Icons.chat_bubble_outline,
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
  const _DatePill({required this.date});

  final String date;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        decoration: BoxDecoration(
          color: colors.base300,
          borderRadius: BorderRadius.circular(999),
        ),
        child: Text(
          date,
          style: TextStyle(
            color: colors.baseContent.withValues(alpha: 0.55),
            fontSize: 11,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}

class _MessageRow extends ConsumerWidget {
  const _MessageRow({
    this.bubbleKey,
    required this.message,
    required this.peerAvatar,
    required this.viewerAvatar,
  });

  final GlobalKey? bubbleKey;
  final ChatMessagePayload message;
  final String peerAvatar;
  final String viewerAvatar;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final InlineSpan? contentSpan = buildStickerMessageSpan(
      message.content,
      ref.read(stickerLibraryProvider).urlByName,
    );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Row(
        mainAxisAlignment: message.isSelf
            ? MainAxisAlignment.end
            : MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          if (!message.isSelf) ...<Widget>[
            GfAvatar(src: resolveApiAssetUrl(peerAvatar), size: 32),
            const SizedBox(width: 8),
          ],
          Flexible(
            child: GfMessageBubble(
              bubbleKey: bubbleKey,
              text: message.content,
              contentSpan: contentSpan,
              mine: message.isSelf,
              time: formatChatTime(
                message.createdAt,
                l10n: AppLocalizations.of(context),
              ),
              maxWidthFactor: 0.74,
            ),
          ),
          if (message.isSelf) ...<Widget>[
            const SizedBox(width: 8),
            GfAvatar(
              src: resolveApiAssetUrl(viewerAvatar),
              size: 32,
              ring: true,
            ),
          ],
        ],
      ),
    );
  }
}

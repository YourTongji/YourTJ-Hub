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
import '../../widgets/status_views.dart';

const InputDecoration _compactSearchDecoration = InputDecoration(
  constraints: BoxConstraints(minHeight: 44),
  contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 10),
  prefixIconConstraints: BoxConstraints(minWidth: 40, minHeight: 40),
);

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
    _tabScrollRegistry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.messages, _scrollToTopController);
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
    try {
      final props = await ref.read(pageRepositoryProvider).fetch('/messages');
      final MessagesPageProps? parsed = parsePageProps<MessagesPageProps>(
        props,
      );
      final List<ChatItemPayload> items = parsed?.conversations ?? [];
      if (mounted) {
        setState(() {
          _conversations = AsyncValue.data(items);
          _suggestedUsers = parsed?.suggestedUsers ?? const [];
          _viewerAvatar = resolveApiAssetUrl(props.layout.viewer.avatarUrl);
          _targetConversation = _targetConversationFor(items);
        });
      }
      // 会话列表在单事务中批量写入离线缓存(断网可读)。
      await ref.read(offlineChatCacheProvider).putConversations(items);
    } catch (e, st) {
      // 网络失败:回退离线缓存的会话列表。
      if (mounted) {
        try {
          final cached = await ref
              .read(offlineChatCacheProvider)
              .getConversations();
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
    await Navigator.of(context).push(
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
    final GfColors colors = GfTheme.colorsOf(context);
    final ChatItemPayload? targetConversation = _targetConversation;
    if (targetConversation != null) {
      return _ConversationPage(
        key: ValueKey<int>(targetConversation.peerId),
        conv: targetConversation,
        viewerAvatar: _viewerAvatar,
      );
    }
    return Scaffold(
      backgroundColor: colors.base100,
      appBar: GfAppBar(
        title: Text(
          l10n.messagesTitle,
          style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w700),
        ),
        actions: [
          GfIconButton(
            icon: Icons.add_comment_outlined,
            tooltip: l10n.messagesNew,
            size: 44,
            onPressed: _startNewChat,
          ),
        ],
      ),
      body: _conversations.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (items) {
          return GfScrollToTop(
            semanticLabel: l10n.commonBackToTop,
            controller: _scrollToTopController,
            builder: (_, ScrollController controller) => Column(
              children: <Widget>[
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: colors.base100,
                    border: Border(bottom: BorderSide(color: colors.line)),
                  ),
                  child: GfInput(
                    controller: _conversationSearch,
                    hintText: l10n.messagesSearchConversations,
                    prefixIcon: const Icon(Icons.search, size: 18),
                    decoration: _compactSearchDecoration,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                Expanded(
                  child: _ConversationList(
                    controller: controller,
                    items: items,
                    query: _conversationSearch.text,
                    emptyMessage: l10n.messagesEmpty,
                    emptyDescription: l10n.messagesEmptyDescription,
                    actionLabel: l10n.messagesNew,
                    onStart: _startNewChat,
                    onOpen: _openConversation,
                  ),
                ),
              ],
            ),
          );
        },
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

class _ConversationPageState extends ConsumerState<_ConversationPage> {
  final List<ChatMessagePayload> _messages = [];
  final TextEditingController _input = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  bool _loading = true;
  bool _loadingOlder = false;
  bool _hasMoreBefore = false;
  Timer? _pollTimer;
  late int _convId;
  int _latestId = 0;
  int _nextBeforeId = 0;
  bool _pollingConfigured = false;

  @override
  void initState() {
    super.initState();
    _convId = widget.conv.convId;
    _load();
    _scrollController.addListener(_onScroll);
    // 打开会话即上报已读回执(清服务端未读数)。
    _markRead();
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
  void didUpdateWidget(covariant _ConversationPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    final int nextConvId = widget.conv.convId;
    if (_convId > 0 || nextConvId <= 0) return;

    _convId = nextConvId;
    _loading = true;
    unawaited(_load(silent: true));
    unawaited(_markRead());
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _input.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.hasClients &&
        _scrollController.position.pixels < 80) {
      _loadOlder();
    }
  }

  /// 已读回执:markRead + 缓存置空未读(服务端清除,列表轮询会反映)。
  Future<void> _markRead() async {
    if (_convId <= 0) return;
    try {
      await ref.read(chatRepositoryProvider).markRead(convId: _convId);
    } catch (_) {
      // 回执失败静默(下次轮询重试)。
    }
  }

  Future<void> _load({bool silent = false}) async {
    if (_convId <= 0) {
      if (mounted) setState(() => _loading = false);
      return;
    }
    try {
      final bool initial = _latestId == 0;
      final bool pinnedToBottom =
          !_scrollController.hasClients ||
          _scrollController.position.maxScrollExtent -
                  _scrollController.position.pixels <
              80;
      final ChatMessagesResponse resp = await ref
          .read(chatRepositoryProvider)
          .getMessages(convId: _convId, afterId: _latestId);
      final Set<int> seenIds = _messages
          .map((ChatMessagePayload message) => message.id)
          .toSet();
      final List<ChatMessagePayload> newMessages = resp.list
          .where((ChatMessagePayload message) => seenIds.add(message.id))
          .toList();
      if (mounted) {
        setState(() {
          if (newMessages.isNotEmpty) {
            _messages.addAll(newMessages);
            _messages.sort((a, b) => a.id.compareTo(b.id));
          }
          if (resp.list.isNotEmpty) _latestId = resp.latestId;
          _hasMoreBefore = resp.hasMoreBefore;
          _nextBeforeId = resp.nextBeforeId;
          _loading = false;
        });
        if (initial || pinnedToBottom) _scrollToBottom();
      }
      if (newMessages.isNotEmpty) {
        // 只持久化真正新增的消息，避免轮询重复写缓存和重复上报已读。
        await ref
            .read(offlineChatCacheProvider)
            .putMessages(_convId, newMessages);
        await _markRead();
      }
    } catch (_) {
      // 网络失败:回退离线缓存消息。
      if (mounted) {
        try {
          final cached = await ref
              .read(offlineChatCacheProvider)
              .getMessages(_convId);
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
      }
      if (mounted && !silent) setState(() => _loading = false);
    }
  }

  Future<void> _loadOlder() async {
    if (_loadingOlder || !_hasMoreBefore || _nextBeforeId <= 0) return;
    setState(() => _loadingOlder = true);
    final double previousExtent = _scrollController.hasClients
        ? _scrollController.position.maxScrollExtent
        : 0;
    try {
      final ChatMessagesResponse resp = await ref
          .read(chatRepositoryProvider)
          .getMessages(convId: _convId, beforeId: _nextBeforeId);
      if (!mounted) return;
      setState(() {
        final Set<int> existing = _messages.map((m) => m.id).toSet();
        _messages.addAll(resp.list.where((m) => !existing.contains(m.id)));
        _messages.sort((a, b) => a.id.compareTo(b.id));
        _hasMoreBefore = resp.hasMoreBefore;
        _nextBeforeId = resp.nextBeforeId;
      });
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!_scrollController.hasClients) return;
        final double addedExtent =
            _scrollController.position.maxScrollExtent - previousExtent;
        _scrollController.jumpTo(
          addedExtent.clamp(0, _scrollController.position.maxScrollExtent),
        );
      });
    } catch (_) {
      // 历史消息加载失败保持当前列表，允许下一次滚动重试。
    } finally {
      if (mounted) setState(() => _loadingOlder = false);
    }
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!_scrollController.hasClients) return;
      _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
    });
  }

  Future<void> _send(String value) async {
    final String text = value.trim();
    if (text.isEmpty) return;
    try {
      final int convId = await ref
          .read(chatRepositoryProvider)
          .sendMessage(peerId: widget.conv.peerId, content: text);
      if (_convId <= 0 && convId > 0) _convId = convId;
      await _load(silent: true);
    } catch (e) {
      if (mounted) {
        final l10n = AppLocalizations.of(context);
        showGfToast(context, l10n.messagesSendFailed('$e'), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);

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
                    widget.conv.peerUsername,
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
            child: ColoredBox(
              color: colors.base100,
              child: _loading
                  ? const GfLoading()
                  : _messages.isEmpty
                  ? _ChatEmptyState(
                      title: l10n.messagesStartChat,
                      description: l10n.messagesFirstMessageTo(
                        widget.conv.peerUsername,
                      ),
                    )
                  : ListView.builder(
                      controller: _scrollController,
                      padding: const EdgeInsets.fromLTRB(12, 12, 12, 18),
                      itemCount: _messages.length + (_loadingOlder ? 1 : 0),
                      itemBuilder: (BuildContext context, int index) {
                        if (_loadingOlder && index == 0) {
                          return const Padding(
                            padding: EdgeInsets.only(bottom: 10),
                            child: GfLoadingIndicator(small: true),
                          );
                        }
                        final int messageIndex =
                            index - (_loadingOlder ? 1 : 0);
                        final ChatMessagePayload message =
                            _messages[messageIndex];
                        final bool startsDay =
                            messageIndex == 0 ||
                            formatDate(_messages[messageIndex - 1].createdAt) !=
                                formatDate(message.createdAt);
                        return Column(
                          children: <Widget>[
                            if (startsDay)
                              _DatePill(date: formatDate(message.createdAt)),
                            _MessageRow(
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
          SafeArea(
            top: false,
            child: GfChatInput(
              controller: _input,
              hintText: l10n.messagesInputHint,
              sendLabel: l10n.commonSend,
              onSend: _send,
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
    required this.items,
    required this.query,
    required this.emptyMessage,
    required this.emptyDescription,
    required this.actionLabel,
    required this.onStart,
    required this.onOpen,
  });

  final ScrollController controller;
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
      controller: controller,
      itemCount: filtered.length,
      separatorBuilder: (_, _) => const GfDivider(),
      itemBuilder: (BuildContext context, int index) {
        final ChatItemPayload conversation = filtered[index];
        return GfConversationRow(
          avatarUrl: resolveApiAssetUrl(conversation.peerAvatar),
          name: conversation.peerUsername,
          lastMessage: conversation.lastMsg.isEmpty
              ? l10n.messagesNoMessagesYet
              : conversation.lastMsg,
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
          SizedBox(
            height: 48,
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
                    size: 32,
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
            child: GfInput(
              controller: _search,
              hintText: widget.searchHint,
              prefixIcon: const Icon(Icons.search, size: 18),
              decoration: _compactSearchDecoration,
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
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 28),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: colors.info.withValues(alpha: 0.10),
                borderRadius: BorderRadius.circular(radii.box * 2),
              ),
              child: Icon(
                Icons.chat_bubble_outline,
                size: 28,
                color: colors.primary,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              title,
              textAlign: TextAlign.center,
              style: TextStyle(
                color: colors.baseContent,
                fontSize: 17,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              description,
              textAlign: TextAlign.center,
              style: TextStyle(
                color: colors.baseContent.withValues(alpha: 0.55),
                fontSize: 14,
                height: 1.45,
              ),
            ),
            const SizedBox(height: 18),
            GfButton(
              label: actionLabel,
              icon: const Icon(Icons.add_comment_outlined, size: 17),
              onPressed: onStart,
            ),
          ],
        ),
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
    final String name = user.nickname.isEmpty ? user.username : user.nickname;
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

class _MessageRow extends StatelessWidget {
  const _MessageRow({
    required this.message,
    required this.peerAvatar,
    required this.viewerAvatar,
  });

  final ChatMessagePayload message;
  final String peerAvatar;
  final String viewerAvatar;

  @override
  Widget build(BuildContext context) {
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
              text: message.content,
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

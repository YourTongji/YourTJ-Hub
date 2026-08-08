import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import '../../asset_url.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../format.dart';
import '../../widgets/status_views.dart';

/// 私信(IM)页(web messages.index 的移动端形态):
/// 会话列表 + 消息游标分页 + 15s 轮询 + 已读回执 + 离线缓存 + 发起新会话。
class MessagesPage extends ConsumerStatefulWidget {
  const MessagesPage({super.key});

  @override
  ConsumerState<MessagesPage> createState() => _MessagesPageState();
}

class _MessagesPageState extends ConsumerState<MessagesPage> {
  AsyncValue<List<ChatItemPayload>> _conversations = const AsyncValue.loading();
  List<UserConnectionPayload> _suggestedUsers = const [];
  Timer? _pollTimer;

  @override
  void initState() {
    super.initState();
    _load();
    // 前台 15s 轮询(与后端无 WebSocket 的现状一致)。
    _pollTimer = Timer.periodic(
      const Duration(seconds: 15),
      (_) => _load(silent: true),
    );
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
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
        });
      }
      // 会话列表写入离线缓存(断网可读)。
      final cache = ref.read(offlineChatCacheProvider);
      for (final c in items) {
        await cache.putConversation(c);
      }
    } catch (e, st) {
      // 网络失败:回退离线缓存的会话列表。
      if (mounted) {
        try {
          final cached = await ref
              .read(offlineChatCacheProvider)
              .getConversations();
          if (cached.isNotEmpty) {
            setState(() => _conversations = AsyncValue.data(cached));
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

  Future<void> _openConversation(ChatItemPayload conv) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(builder: (_) => _ConversationPage(conv: conv)),
    );
    // 返回后刷新会话列表未读数。
    if (mounted) _load(silent: true);
  }

  /// 发起新会话:弹可联系用户列表(web startChat 语义),选中后发消息进入会话。
  Future<void> _startNewChat() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final UserConnectionPayload?
    selected = await showGfBottomSheet<UserConnectionPayload>(
      context,
      builder: (ctx) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                l10n.messagesNew,
                style: GfTheme.typographyOf(ctx).title2,
              ),
            ),
            if (_suggestedUsers.isEmpty)
              Padding(
                padding: const EdgeInsets.all(24),
                child: Center(
                  child: Text(
                    l10n.messagesNoContactableUsers,
                    style: TextStyle(color: GfTheme.colorsOf(ctx).iconMuted),
                  ),
                ),
              ),
            for (final user in _suggestedUsers)
              GfSettingRow(
                leading: GfAvatar(
                  src: resolveApiAssetUrl(user.avatarUrl),
                  size: 36,
                ),
                title: user.nickname.isEmpty ? user.username : user.nickname,
                description: '@${user.username}',
                onTap: () => Navigator.pop(ctx, user),
              ),
          ],
        ),
      ),
    );
    if (selected == null || !mounted) return;
    // 已有会话则直接打开,否则先发一条空消息建立会话(web startChat 语义)。
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
    try {
      final int convId = await ref
          .read(chatRepositoryProvider)
          .sendMessage(peerId: selected.id, content: '');
      if (!mounted) return;
      await _openConversation(
        ChatItemPayload(
          id: convId,
          peerId: selected.id,
          peerUsername: selected.nickname.isEmpty
              ? selected.username
              : selected.nickname,
          peerAvatar: resolveApiAssetUrl(selected.avatarUrl),
          lastMsg: '',
          lastMsgTime: '',
          unreadCount: 0,
          convId: convId,
          peerUrl: selected.url,
        ),
      );
    } catch (_) {
      if (mounted) {
        showGfToast(context, l10n.messagesSendFailed(''), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(
        title: Text(l10n.messagesTitle),
        actions: [
          GfIconButton(
            icon: Icons.edit_outlined,
            tooltip: l10n.messagesNew,
            onPressed: _startNewChat,
          ),
        ],
      ),
      body: _conversations.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (items) {
          if (items.isEmpty) return GfEmpty(message: l10n.messagesEmpty);
          return ListView.separated(
            itemCount: items.length,
            separatorBuilder: (_, _) => const GfDivider(),
            itemBuilder: (context, i) {
              final c = items[i];
              return GfConversationRow(
                avatarUrl: c.peerAvatar,
                name: c.peerUsername,
                lastMessage: c.lastMsg,
                time: formatChatTime(c.lastMsgTime, l10n: l10n),
                unreadCount: c.unreadCount,
                onTap: () => _openConversation(c),
              );
            },
          );
        },
      ),
    );
  }
}

/// 单会话聊天页。
class _ConversationPage extends ConsumerStatefulWidget {
  const _ConversationPage({required this.conv});

  final ChatItemPayload conv;

  @override
  ConsumerState<_ConversationPage> createState() => _ConversationPageState();
}

class _ConversationPageState extends ConsumerState<_ConversationPage> {
  final List<ChatMessagePayload> _messages = [];
  final TextEditingController _input = TextEditingController();
  bool _loading = true;
  Timer? _pollTimer;
  int _latestId = 0;

  @override
  void initState() {
    super.initState();
    _load();
    // 打开会话即上报已读回执(清服务端未读数)。
    _markRead();
    _pollTimer = Timer.periodic(
      const Duration(seconds: 15),
      (_) => _load(silent: true),
    );
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _input.dispose();
    super.dispose();
  }

  /// 已读回执:markRead + 缓存置空未读(服务端清除,列表轮询会反映)。
  Future<void> _markRead() async {
    try {
      await ref
          .read(chatRepositoryProvider)
          .markRead(convId: widget.conv.convId);
    } catch (_) {
      // 回执失败静默(下次轮询重试)。
    }
  }

  Future<void> _load({bool silent = false}) async {
    try {
      final resp = await ref
          .read(chatRepositoryProvider)
          .getMessages(convId: widget.conv.convId, afterId: _latestId);
      if (mounted) {
        setState(() {
          if (resp.list.isNotEmpty) {
            _messages.addAll(resp.list);
            _latestId = resp.latestId;
          }
          _loading = false;
        });
      }
      // 新消息写入离线缓存。
      final cache = ref.read(offlineChatCacheProvider);
      await cache.putMessages(widget.conv.convId, resp.list);
      // 收到新消息后再次上报已读回执(消息已展示即已读)。
      if (resp.list.isNotEmpty) {
        await _markRead();
      }
    } catch (_) {
      // 网络失败:回退离线缓存消息。
      if (mounted) {
        try {
          final cached = await ref
              .read(offlineChatCacheProvider)
              .getMessages(widget.conv.convId);
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

  Future<void> _send() async {
    final String text = _input.text.trim();
    if (text.isEmpty) return;
    _input.clear();
    try {
      await ref
          .read(chatRepositoryProvider)
          .sendMessage(peerId: widget.conv.peerId, content: text);
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

    return Scaffold(
      appBar: GfAppBar(title: Text(widget.conv.peerUsername)),
      body: Column(
        children: [
          Expanded(
            child: _loading
                ? const GfLoading()
                : _messages.isEmpty
                ? GfEmpty(message: l10n.messagesEmptyDetail)
                : ListView.builder(
                    reverse: true,
                    padding: const EdgeInsets.all(12),
                    itemCount: _messages.length,
                    itemBuilder: (context, i) {
                      final m = _messages[_messages.length - 1 - i];
                      return GfMessageBubble(
                        text: m.content,
                        mine: m.isSelf,
                        time: formatChatTime(m.createdAt),
                      );
                    },
                  ),
          ),
          SafeArea(
            top: false,
            child: GfChatInput(
              controller: _input,
              hintText: l10n.messagesInputHint,
              onSend: (_) => _send(),
            ),
          ),
        ],
      ),
    );
  }
}

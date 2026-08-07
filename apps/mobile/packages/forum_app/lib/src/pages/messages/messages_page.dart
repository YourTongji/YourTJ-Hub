import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

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
    final UserConnectionPayload? selected =
        await showModalBottomSheet<UserConnectionPayload>(
          context: context,
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
                        style: TextStyle(
                          color: GfTheme.colorsOf(ctx).iconMuted,
                        ),
                      ),
                    ),
                  ),
                for (final user in _suggestedUsers)
                  ListTile(
                    leading: CircleAvatar(
                      radius: 18,
                      backgroundImage: user.avatarUrl.isEmpty
                          ? null
                          : NetworkImage(user.avatarUrl),
                      child: user.avatarUrl.isEmpty
                          ? const Icon(Icons.person, size: 18)
                          : null,
                    ),
                    title: Text(
                      user.nickname.isEmpty ? user.username : user.nickname,
                    ),
                    subtitle: Text('@${user.username}'),
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
          peerAvatar: selected.avatarUrl,
          lastMsg: '',
          lastMsgTime: '',
          unreadCount: 0,
          convId: convId,
          peerUrl: selected.url,
        ),
      );
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.messagesSendFailed(''))));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.messagesTitle),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit_outlined),
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
            separatorBuilder: (_, _) => const Divider(height: 1),
            itemBuilder: (context, i) {
              final c = items[i];
              return ListTile(
                leading: CircleAvatar(
                  radius: 20,
                  backgroundImage: c.peerAvatar.isEmpty
                      ? null
                      : NetworkImage(c.peerAvatar),
                  child: c.peerAvatar.isEmpty
                      ? const Icon(Icons.person, size: 20)
                      : null,
                ),
                title: Text(c.peerUsername),
                subtitle: Text(
                  c.lastMsg,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Text(
                      formatChatTime(c.lastMsgTime, l10n: l10n),
                      style: GfTheme.typographyOf(context).meta.copyWith(
                        color: GfTheme.colorsOf(context).iconMuted,
                      ),
                    ),
                    if (c.unreadCount > 0) ...[
                      const SizedBox(height: 4),
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 6,
                          vertical: 1,
                        ),
                        decoration: BoxDecoration(
                          color: GfTheme.colorsOf(context).error,
                          borderRadius: BorderRadius.all(Radius.circular(10)),
                        ),
                        child: Text(
                          '${c.unreadCount}',
                          style: GfTheme.typographyOf(context).meta.copyWith(
                            color: GfTheme.colorsOf(context).neutralContent,
                          ),
                        ),
                      ),
                    ],
                  ],
                ),
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
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.messagesSendFailed('$e'))));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: AppBar(title: Text(widget.conv.peerUsername)),
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
                      return _MessageBubble(message: m, isMine: m.isSelf);
                    },
                  ),
          ),
          SafeArea(
            top: false,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: colors.base100,
                border: Border(top: BorderSide(color: colors.line, width: 0.5)),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _input,
                      minLines: 1,
                      maxLines: 4,
                      decoration: InputDecoration(
                        hintText: l10n.messagesInputHint,
                        isDense: true,
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(20),
                        ),
                        contentPadding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 10,
                        ),
                      ),
                      onSubmitted: (_) => _send(),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton(
                    icon: const Icon(Icons.send),
                    color: colors.primary,
                    onPressed: _send,
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _MessageBubble extends StatelessWidget {
  const _MessageBubble({required this.message, required this.isMine});

  final ChatMessagePayload message;
  final bool isMine;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Align(
      alignment: isMine ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        constraints: const BoxConstraints(maxWidth: 280),
        decoration: BoxDecoration(
          color: isMine ? colors.primary : colors.base300,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Text(
          message.content,
          style: GfTheme.typographyOf(context).body.copyWith(
            color: isMine ? colors.primaryContent : colors.baseContent,
          ),
        ),
      ),
    );
  }
}

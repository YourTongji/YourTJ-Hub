import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';

/// 私信(IM)页(web messages.index 的移动端形态):
/// 会话列表 + 消息游标分页 + 15s 轮询 + 已读回执 + 离线缓存。
class MessagesPage extends ConsumerStatefulWidget {
  const MessagesPage({super.key});

  @override
  ConsumerState<MessagesPage> createState() => _MessagesPageState();
}

class _MessagesPageState extends ConsumerState<MessagesPage> {
  AsyncValue<List<ChatItemPayload>> _conversations = const AsyncValue.loading();
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
        setState(() => _conversations = AsyncValue.data(items));
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

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.messagesTitle)),
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
                      _timeLabel(c.lastMsgTime),
                      style: const TextStyle(fontSize: 11, color: Colors.grey),
                    ),
                    if (c.unreadCount > 0) ...[
                      const SizedBox(height: 4),
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 6,
                          vertical: 1,
                        ),
                        decoration: const BoxDecoration(
                          color: Colors.red,
                          borderRadius: BorderRadius.all(Radius.circular(10)),
                        ),
                        child: Text(
                          '${c.unreadCount}',
                          style: const TextStyle(
                            fontSize: 11,
                            color: Colors.white,
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

  String _timeLabel(String iso) {
    final DateTime t = DateTime.tryParse(iso)?.toLocal() ?? DateTime.now();
    final DateTime now = DateTime.now();
    if (t.year == now.year && t.month == now.month && t.day == now.day) {
      return '${t.hour.toString().padLeft(2, '0')}:${t.minute.toString().padLeft(2, '0')}';
    }
    return '${t.month}/${t.day}';
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
          style: TextStyle(
            fontSize: 14,
            color: isMine ? colors.primaryContent : colors.baseContent,
          ),
        ),
      ),
    );
  }
}

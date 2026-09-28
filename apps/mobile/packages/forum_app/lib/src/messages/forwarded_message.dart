import '../asset_url.dart';
import 'chat_message_row.dart';
import 'chat_message_bubble.dart';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../format.dart';
import 'chat_reply.dart';

class ForwardedMessageCard extends ConsumerWidget {
  const ForwardedMessageCard({super.key, required this.bundle});
  final ChatForwardBundle bundle;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Semantics(
      button: true,
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () {
          final epoch = ref.read(offlineCacheEpochProvider);
          Navigator.of(context, rootNavigator: true).push(
            MaterialPageRoute<void>(
              builder: (_) =>
                  ForwardedMessagesPage(bundle: bundle, ownerEpoch: epoch),
            ),
          );
        },
        child: ConstrainedBox(
          constraints: const BoxConstraints(minWidth: 160, maxWidth: 280),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                l10n.messageForwardHistory,
                style: const TextStyle(
                  fontWeight: FontWeight.w700,
                  fontSize: 16,
                ),
              ),
              const SizedBox(height: 8),
              for (final item in bundle.messages.take(3))
                Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: Text(
                    '${item.senderName}: ${item.forwarded != null ? l10n.messageForwardHistory : chatReplyExcerpt(item.content, maxLength: 80)}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 13),
                  ),
                ),
              const Divider(height: 16),
              Row(
                children: [
                  Expanded(
                    child: Text(
                      l10n.messageForwardCount(bundle.messages.length),
                      style: const TextStyle(fontSize: 12),
                    ),
                  ),
                  GfSymbol(
                    'chevron-right',
                    size: 16,
                    color: DefaultTextStyle.of(context).style.color,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class ForwardedMessagesPage extends ConsumerWidget {
  const ForwardedMessagesPage({
    super.key,
    required this.bundle,
    required this.ownerEpoch,
  });
  final int ownerEpoch;
  final ChatForwardBundle bundle;
  @override
  Widget build(BuildContext context, WidgetRef ref) =>
      ref.watch(offlineCacheEpochProvider) != ownerEpoch
      ? const SizedBox.shrink()
      : Scaffold(
          appBar: GfAppBar(
            title: Text(AppLocalizations.of(context).messageForwardHistory),
          ),
          body: SafeArea(
            top: false,
            child: ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: bundle.messages.length,
              itemBuilder: (context, index) {
                final entry = bundle.messages[index];
                return ChatMessageRow(
                  mine: false,
                  senderName: entry.senderName,
                  avatar: SizedBox(
                    width: 44,
                    height: 44,
                    child: Align(
                      alignment: Alignment.topLeft,
                      child: GfAvatar(
                        src: resolveApiAssetUrl(entry.avatarUrl),
                        size: 32,
                      ),
                    ),
                  ),
                  child: ChatMessageBubble(
                    text: entry.content,
                    mine: false,
                    time: formatDateTime(entry.createdAt),
                    maxWidthFactor: 0.74,
                    content: entry.forwarded == null
                        ? null
                        : ForwardedMessageCard(bundle: entry.forwarded!),
                  ),
                );
              },
            ),
          ),
        );
}

import 'dart:async';

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
import '../widgets/stickers/sticker_library_state.dart';
import '../widgets/stickers/sticker_strings.dart';

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

class ForwardedMessagesPage extends ConsumerStatefulWidget {
  const ForwardedMessagesPage({
    super.key,
    required this.bundle,
    required this.ownerEpoch,
  });
  final int ownerEpoch;
  final ChatForwardBundle bundle;
  @override
  ConsumerState<ForwardedMessagesPage> createState() =>
      _ForwardedMessagesPageState();
}

class _ForwardedMessagesPageState extends ConsumerState<ForwardedMessagesPage> {
  final Map<int, GlobalKey> _bubbleKeys = {};

  bool get _sessionCurrent =>
      mounted && ref.read(offlineCacheEpochProvider) == widget.ownerEpoch;

  Future<void> _showEntryActions(int index, ChatForwardEntry entry) async {
    if (!_sessionCurrent) return;
    if (entry.msgType == 4 || entry.forwarded != null) return;
    final stickers = parseStickerSegments(
      entry.content,
      ref.read(stickerLibraryProvider).urlByName,
    ).whereType<StickerImageSegment>().map((s) => s.name).toList();
    if (stickers.isEmpty) return;
    final collection = ref.read(stickerCollectionProvider);
    if (!collection.active) return;
    final box = _bubbleKeys[index]?.currentContext?.findRenderObject();
    final overlay = Navigator.of(
      context,
      rootNavigator: true,
    ).overlay?.context.findRenderObject();
    if (box is! RenderBox || overlay is! RenderBox) return;
    final origin = box.localToGlobal(Offset.zero, ancestor: overlay);
    final action = await showMenu<String>(
      context: context,
      useRootNavigator: true,
      position: RelativeRect.fromRect(
        origin & box.size,
        Offset.zero & overlay.size,
      ),
      items: [
        PopupMenuItem(
          value: 'collect',
          child: Text(StickerStrings(context).collect),
        ),
      ],
    );
    if (!mounted || !_sessionCurrent || action == null) return;
    switch (action) {
      case 'collect':
        final strings = StickerStrings(context);
        try {
          for (final name in stickers) {
            await ref.read(stickerCollectionProvider).save(stickerName: name);
          }
          if (mounted) showGfToast(context, strings.saved);
        } catch (error) {
          if (mounted) {
            showGfToast(context, strings.failure(error), error: true);
          }
        }
    }
  }

  @override
  Widget build(BuildContext context) =>
      ref.watch(offlineCacheEpochProvider) != widget.ownerEpoch
      ? const SizedBox.shrink()
      : Scaffold(
          appBar: GfAppBar(
            title: Text(AppLocalizations.of(context).messageForwardHistory),
          ),
          body: SafeArea(
            top: false,
            child: ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: widget.bundle.messages.length,
              itemBuilder: (context, index) {
                final entry = widget.bundle.messages[index];
                final nested = entry.msgType == 4 || entry.forwarded != null;
                final hasStickers =
                    !nested &&
                    parseStickerSegments(
                      entry.content,
                      ref.read(stickerLibraryProvider).urlByName,
                    ).any((segment) => segment is StickerImageSegment);
                final canCollectStickers =
                    hasStickers && ref.read(stickerCollectionProvider).active;
                final key = _bubbleKeys.putIfAbsent(index, GlobalKey.new);
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
                    bubbleKey: key,
                    text: entry.content,
                    mine: false,
                    time: formatDateTime(entry.createdAt),
                    maxWidthFactor: 0.74,
                    selectable: !nested,
                    onLongPress: canCollectStickers
                        ? () => unawaited(_showEntryActions(index, entry))
                        : null,
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

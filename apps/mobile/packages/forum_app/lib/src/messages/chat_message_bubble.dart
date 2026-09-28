import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../link_navigation.dart';
import '../providers.dart';
import '../server_messages.dart';
import '../widgets/stickers/resolved_sticker_content.dart';
import '../widgets/sticker_message_span.dart';
import 'message_content.dart';
import 'swipe_reply.dart';

/// Shared chat body for the live conversation and read-only forwarded history.
class ChatMessageBubble extends ConsumerWidget {
  const ChatMessageBubble({
    super.key,
    this.bubbleKey,
    required this.text,
    required this.mine,
    this.time,
    this.maxWidthFactor = 0.88,
    this.onLongPress,
    this.onSwipeReply,
    this.content,
  });

  final GlobalKey? bubbleKey;
  final String text;
  final bool mine;
  final String? time;
  final double maxWidthFactor;
  final VoidCallback? onLongPress;
  final VoidCallback? onSwipeReply;
  final Widget? content;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ResolvedStickerContent(
      content: text,
      errorAlignment: mine ? CrossAxisAlignment.end : CrossAxisAlignment.start,
      builder: (stickers) => GfMessageBubble(
        bubbleKey: bubbleKey,
        interactionWrapper: (child) => SwipeReply(
          label: AppLocalizations.of(context).messageReply,
          onReply: onSwipeReply,
          onActions: onLongPress,
          actionsLabel: AppLocalizations.of(context).messageActions,
          child: child,
        ),
        text: text,
        showBubble: content != null || !isStickerOnlyMessage(text, stickers),
        selectable: true,
        onLongPress: onLongPress,
        copyMessageLabel: AppLocalizations.of(context).messagesCopyAll,
        content:
            content ??
            MessageContent(
              text: text,
              stickers: stickers,
              deferStickerLongPress: onLongPress != null,
              onOpenLink: (url) async {
                try {
                  await LinkNavigation.open(
                    context,
                    url,
                    baseUrl: ref.read(apiClientProvider).baseUrl,
                  );
                } catch (error) {
                  if (context.mounted) {
                    showGfToast(
                      context,
                      resolveErrorMessage(AppLocalizations.of(context), error),
                      error: true,
                    );
                  }
                }
              },
            ),
        mine: mine,
        time: time,
        maxWidthFactor: maxWidthFactor,
      ),
    );
  }
}

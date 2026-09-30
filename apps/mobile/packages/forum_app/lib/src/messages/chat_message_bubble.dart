import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../app_config.dart';
import '../link_navigation.dart';
import '../providers.dart';
import '../server_messages.dart';
import '../widgets/stickers/resolved_sticker_content.dart';
import '../widgets/sticker_message_span.dart';
import 'message_content.dart';
import 'swipe_reply.dart';
import 'chat_reply.dart';
import 'chat_image.dart';

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
    this.replyToMessageId,
    this.onQuoteTap,
    this.content,
    this.selectable = true,
    this.msgType = 1,
  });

  final GlobalKey? bubbleKey;
  final String text;
  final bool mine;
  final String? time;
  final double maxWidthFactor;
  final VoidCallback? onLongPress;
  final VoidCallback? onSwipeReply;
  final int? replyToMessageId;
  final VoidCallback? onQuoteTap;
  final Widget? content;
  final bool selectable;
  final int msgType;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final image =
        msgType == 2 &&
        isChatImageUrl(
          text,
          baseUrl: ref.watch(apiClientProvider).baseUrl,
          assetOrigins: AppConfig.chatImageOrigins.split(','),
        );
    final quote = content == null && !image ? parseChatReplyQuote(text) : null;
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
        showBubble:
            content != null ||
            (!image && !isStickerOnlyMessage(text, stickers)),
        selectable: selectable && !image,
        onLongPress: onLongPress,
        copyMessageLabel: AppLocalizations.of(context).messagesCopyAll,
        content:
            content ??
            (image
                ? ChatImage(url: text)
                : _MessageContentWithQuote(
                    text: quote?.body ?? text,
                    quote: quote,
                    replyToMessageId: replyToMessageId,
                    onQuoteTap: onQuoteTap,
                    stickers: stickers,
                    mine: mine,
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
                            resolveErrorMessage(
                              AppLocalizations.of(context),
                              error,
                            ),
                            error: true,
                          );
                        }
                      }
                    },
                    deferStickerLongPress: onLongPress != null,
                  )),
        mine: mine,
        time: time,
        maxWidthFactor: maxWidthFactor,
      ),
    );
  }
}

class _MessageContentWithQuote extends StatelessWidget {
  const _MessageContentWithQuote({
    required this.text,
    required this.quote,
    required this.replyToMessageId,
    required this.onQuoteTap,
    required this.stickers,
    required this.mine,
    required this.onOpenLink,
    required this.deferStickerLongPress,
  });

  final String text;
  final ChatReplyQuote? quote;
  final int? replyToMessageId;
  final VoidCallback? onQuoteTap;
  final Map<String, String> stickers;
  final bool mine;
  final ValueChanged<String> onOpenLink;
  final bool deferStickerLongPress;

  @override
  Widget build(BuildContext context) {
    Widget body() => MessageContent(
      text: text,
      stickers: stickers,
      onOpenLink: onOpenLink,
      deferStickerLongPress: deferStickerLongPress,
    );
    final reply = quote;
    if (reply == null) return body();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        _ChatReplyQuoteBlock(
          quote: reply,
          mine: mine,
          onTap: replyToMessageId == null ? null : onQuoteTap,
        ),
        if (text.isNotEmpty) ...[const SizedBox(height: 8), body()],
      ],
    );
  }
}

class _ChatReplyQuoteBlock extends StatelessWidget {
  const _ChatReplyQuoteBlock({
    required this.quote,
    required this.mine,
    this.onTap,
  });

  final ChatReplyQuote quote;
  final bool mine;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final foreground = mine
        ? colors.messageOutgoingContent
        : colors.baseContent;
    final fill = mine ? colors.messageOutgoingContent : colors.baseContent;
    final label = AppLocalizations.of(context).messagesJumpToQuotedMessage;
    final excerpt = localizedChatReplyExcerpt(
      quote.excerpt,
      imageLabel: AppLocalizations.of(context).messagesImage,
    );
    return Semantics(
      button: onTap != null,
      enabled: onTap != null ? true : null,
      label: onTap == null ? null : '$label: ${quote.sender} $excerpt',
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        onLongPress: () {},
        child: SelectionContainer.disabled(
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 6),
            decoration: BoxDecoration(
              color: fill.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(4),
            ),
            child: IntrinsicHeight(
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Container(
                    width: 1,
                    color:
                        (mine
                                ? colors.messageOutgoingContent
                                : colors.iconMuted)
                            .withValues(alpha: 0.72),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (quote.sender.isNotEmpty)
                          Text(
                            quote.sender,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: foreground.withValues(alpha: 0.76),
                              fontSize: 12,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        if (quote.excerpt.isNotEmpty)
                          Text(
                            excerpt,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(color: foreground, fontSize: 13),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

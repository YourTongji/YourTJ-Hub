import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../markdown_view.dart';
import '../sticker_message_span.dart';
import 'resolved_sticker_content.dart';
import 'sticker_image.dart';
import 'sticker_strings.dart';

/// A bounded, live view of the draft; persisted and submitted text stays intact.
class StickerDraftPreview extends StatelessWidget {
  const StickerDraftPreview({
    super.key,
    required this.content,
    this.markdown = false,
  });

  final String content;
  final bool markdown;

  @override
  Widget build(BuildContext context) {
    if (!stickerTokenPattern.hasMatch(content)) return const SizedBox.shrink();
    final colors = GfTheme.colorsOf(context);
    return Container(
      key: const Key('sticker-draft-preview'),
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(16),
      ),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxHeight: 96),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              StickerStrings(context).preview,
              style: Theme.of(
                context,
              ).textTheme.labelSmall?.copyWith(color: colors.iconMuted),
            ),
            Flexible(
              child: SingleChildScrollView(
                reverse: true,
                child: markdown
                    ? GfMarkdownView(
                        data: content,
                        stickerSize: StickerImage.thumbnailSize,
                      )
                    : ResolvedStickerContent(
                        content: content,
                        builder: (urls) => Text.rich(
                          buildStickerMessageSpan(
                                content,
                                urls,
                                stickerSize: StickerImage.thumbnailSize,
                              ) ??
                              TextSpan(text: content),
                          style: TextStyle(
                            fontSize: 16,
                            color: colors.baseContent,
                          ),
                        ),
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../asset_url.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/share/share_image_preview.dart';

/// Share only the public author identity; viewer-local notes never enter PNGs.
Future<void> showPostShareImage(
  BuildContext context, {
  required PostPayload post,
  required String topicTitle,
  required String baseUrl,
  bool topicLink = false,
}) async {
  final l10n = AppLocalizations.of(context);
  await showShareImagePreview(
    context,
    fileName: 'yourtj-post-${post.topicId}-${post.postNo}.png',
    cardBuilder: (theme) {
      final colors = theme.colors;
      final postUrl = Uri.parse(baseUrl)
          .replace(
            path: topicLink
                ? '/p/post/${post.topicId}'
                : '/p/post/${post.topicId}/${post.postNo}',
            query: null,
            fragment: null,
          )
          .toString();
      // Share cards mirror the reading view's author identity
      // (anonymous label / nickname / username) so a post looks the same
      // exported as it does in the app. They deliberately stay clear of
      // privateDisplayName(): it bakes the viewer's local private notes
      // into the label, which must never leak into an exported PNG.
      final author = post.isAnonymous
          ? l10n.courseCopyAuthorAnonymousLabel
          : (post.author.nickname?.trim().isNotEmpty == true
                ? post.author.nickname!.trim()
                : post.author.username.trim().isNotEmpty
                ? post.author.username.trim()
                : l10n.courseCopyAuthorAnonymousLabel);
      return ShareImageCard(
        theme: theme,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            GfText(
              topicTitle,
              style: TextStyle(
                color: colors.baseContent,
                fontSize: 22,
                height: 1.3,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                if (post.isAnonymous)
                  CircleAvatar(
                    radius: 18,
                    backgroundColor: colors.base200,
                    child: GfSymbol(
                      'user-round',
                      size: 18,
                      color: colors.iconMuted,
                    ),
                  )
                else
                  ClipOval(
                    child: ShareImageNetworkImage(
                      resolveApiAssetUrl(post.author.avatarUrl),
                      width: 36,
                      height: 36,
                      fit: BoxFit.cover,
                      errorBuilder: (_) => CircleAvatar(
                        radius: 18,
                        backgroundColor: colors.base200,
                        child: GfSymbol(
                          'user-round',
                          size: 18,
                          color: colors.iconMuted,
                        ),
                      ),
                    ),
                  ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      GfText(
                        author,
                        style: TextStyle(
                          color: colors.baseContent,
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      GfText(
                        '#${post.postNo} · ${timeAgo(post.createdAt, l10n: l10n)}',
                        style: TextStyle(color: colors.iconMuted, fontSize: 12),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            GfMarkdownView(
              data: post.content,
              mentions: post.mentions,
              screenshot: true,
              colors: colors,
            ),
            const SizedBox(height: 12),
            GfText(
              postUrl,
              style: TextStyle(color: colors.primary, fontSize: 11),
            ),
          ],
        ),
      );
    },
  );
}

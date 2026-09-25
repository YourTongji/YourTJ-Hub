import 'package:flutter/material.dart';
import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';

/// A flat content timeline entry. The author belongs to the displayed content,
/// never to the person who bookmarked/liked it. Missing authors stay anonymous.
class GfContentRow extends StatelessWidget {
  const GfContentRow({
    super.key,
    required this.author,
    required this.avatarUrl,
    required this.time,
    required this.text,
    this.title = '',
    this.contextLabel = '',
    this.contextIcon,
    this.thumbnailUrl = '',
    this.onTap,
    this.onAuthorTap,
    this.footer,
  });
  final String author, avatarUrl, time, text, title, contextLabel, thumbnailUrl;
  final Widget? footer;
  final IconData? contextIcon;
  final VoidCallback? onTap, onAuthorTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final muted = TextStyle(fontSize: 13, color: colors.iconMuted);
    return Material(
      color: colors.base100,
      child: InkWell(
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          decoration: BoxDecoration(
            border: Border(bottom: BorderSide(color: colors.line, width: .5)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (contextLabel.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(left: 56, bottom: 8),
                  child: Row(
                    children: [
                      if (contextIcon != null) ...[
                        Icon(contextIcon, size: 14, color: colors.iconMuted),
                        const SizedBox(width: 6),
                      ],
                      Expanded(
                        child: Text(
                          contextLabel,
                          style: muted,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ],
                  ),
                ),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Semantics(
                    button: onAuthorTap != null,
                    label: author,
                    child: InkWell(
                      onTap: onAuthorTap,
                      customBorder: const CircleBorder(),
                      child: GfAvatar(src: avatarUrl, size: 44),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text.rich(
                          TextSpan(
                            children: [
                              if (author.isNotEmpty)
                                TextSpan(
                                  text: author,
                                  style: TextStyle(
                                    fontWeight: FontWeight.w700,
                                    color: colors.baseContent,
                                  ),
                                ),
                              if (time.isNotEmpty)
                                TextSpan(
                                  text:
                                      '${author.isNotEmpty ? ' · ' : ''}$time',
                                  style: muted,
                                ),
                            ],
                          ),
                          style: const TextStyle(fontSize: 15),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        if (title.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 4),
                            child: Text(
                              title,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 16,
                                height: 1.35,
                                fontWeight: FontWeight.w600,
                                color: colors.baseContent,
                              ),
                            ),
                          ),
                        if (text.isNotEmpty || thumbnailUrl.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 4),
                            child: Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Expanded(
                                  child: Text(
                                    text,
                                    maxLines: 3,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                      fontSize: 15,
                                      height: 1.4,
                                      color: colors.baseContent.withValues(
                                        alpha: .8,
                                      ),
                                    ),
                                  ),
                                ),
                                if (thumbnailUrl.isNotEmpty) ...[
                                  const SizedBox(width: 10),
                                  ClipRRect(
                                    borderRadius: BorderRadius.circular(10),
                                    child: Image.network(
                                      thumbnailUrl,
                                      width: 64,
                                      height: 64,
                                      fit: BoxFit.cover,
                                      excludeFromSemantics: true,
                                      cacheWidth:
                                          (64 *
                                                  MediaQuery.devicePixelRatioOf(
                                                    context,
                                                  ))
                                              .round(),
                                      errorBuilder: (_, _, _) =>
                                          const SizedBox.shrink(),
                                    ),
                                  ),
                                ],
                              ],
                            ),
                          ),
                        ?footer,
                      ],
                    ),
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

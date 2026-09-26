import 'package:flutter/material.dart';
import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';
import '../gf_symbol.dart';

/// A flat preview with one compact identity/action line and an aligned body.
/// When [contextLabel] is present, [author] identifies the activity actor;
/// otherwise it identifies the content author. Missing authors stay anonymous.
class GfContentRow extends StatelessWidget {
  const GfContentRow({
    super.key,
    required this.author,
    required this.avatarUrl,
    required this.time,
    required this.text,
    this.title = '',
    this.contextLabel = '',
    this.contextSymbol,
    this.thumbnailUrl = '',
    this.onTap,
    this.onAuthorTap,
    this.footer,
  });
  final String author, avatarUrl, time, text, title, contextLabel, thumbnailUrl;
  final Widget? footer;
  final String? contextSymbol;
  final VoidCallback? onTap, onAuthorTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final muted = TextStyle(fontSize: 13, height: 1.4, color: colors.iconMuted);
    return Material(
      color: colors.base100,
      child: InkWell(
        onTap: onTap,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: EdgeInsets.fromLTRB(16, 12, 16, footer == null ? 12 : 4),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Semantics(
                    button: onAuthorTap != null,
                    label: onAuthorTap == null ? null : author,
                    child: InkWell(
                      onTap: onAuthorTap,
                      customBorder: const CircleBorder(),
                      child: SizedBox.square(
                        dimension: 44,
                        child: Center(
                          child: GfAvatar(src: avatarUrl, size: 40),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Wrap(
                          spacing: 6,
                          runSpacing: 2,
                          crossAxisAlignment: WrapCrossAlignment.center,
                          children: [
                            if (author.isNotEmpty)
                              Text(
                                author,
                                style: TextStyle(
                                  fontSize: 15,
                                  height: 1.4,
                                  fontWeight: FontWeight.w700,
                                  color: colors.baseContent,
                                ),
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                              ),
                            if (contextLabel.isNotEmpty)
                              Text.rich(
                                TextSpan(
                                  children: [
                                    if (contextSymbol != null)
                                      WidgetSpan(
                                        alignment: PlaceholderAlignment.middle,
                                        child: Padding(
                                          padding:
                                              const EdgeInsetsDirectional.only(
                                                end: 4,
                                              ),
                                          child: GfSymbol(
                                            contextSymbol!,
                                            size: 13,
                                            color: colors.iconMuted,
                                          ),
                                        ),
                                      ),
                                    TextSpan(text: contextLabel),
                                  ],
                                ),
                                style: muted,
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                              ),
                            if (time.isNotEmpty) Text(time, style: muted),
                          ],
                        ),
                        if (title.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 5),
                            child: Text(
                              title,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 16,
                                height: 1.4,
                                fontWeight: FontWeight.w600,
                                color: colors.baseContent,
                              ),
                            ),
                          ),
                        if (text.isNotEmpty || thumbnailUrl.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 5),
                            child: Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Expanded(
                                  child: Text(
                                    text,
                                    maxLines: 3,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                      fontSize: 16,
                                      height: 1.45,
                                      color: title.isEmpty
                                          ? colors.baseContent
                                          : colors.iconMuted,
                                    ),
                                  ),
                                ),
                                if (thumbnailUrl.isNotEmpty) ...[
                                  const SizedBox(width: 10),
                                  ClipRRect(
                                    borderRadius: BorderRadius.circular(12),
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
                        if (footer != null)
                          Padding(
                            padding: const EdgeInsets.only(top: 2),
                            child: footer!,
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsetsDirectional.only(start: 70, end: 16),
              child: SizedBox(
                height: .5,
                child: ColoredBox(color: colors.line),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

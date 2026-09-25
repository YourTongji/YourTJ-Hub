import 'package:flutter/material.dart';
import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';

/// A person in a relationship list, with identity and bio on one reading axis.
class GfConnectionRow extends StatelessWidget {
  const GfConnectionRow({
    super.key,
    required this.name,
    required this.username,
    required this.avatarUrl,
    required this.bio,
    this.action,
    this.onTap,
  });
  final String name, username, avatarUrl, bio;
  final Widget? action;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final identity = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          name,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            fontSize: 16,
            fontWeight: FontWeight.w700,
            color: colors.baseContent,
          ),
        ),
        Text(
          '@$username',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(fontSize: 15, color: colors.iconMuted),
        ),
      ],
    );
    return Material(
      color: colors.base100,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              GfAvatar(src: avatarUrl, size: 48),
              const SizedBox(width: 12),
              Expanded(
                child: LayoutBuilder(
                  builder: (context, constraints) {
                    final stackAction =
                        constraints.maxWidth < 240 ||
                        MediaQuery.textScalerOf(context).scale(15) > 20;
                    return Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (stackAction)
                          identity
                        else
                          Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Expanded(child: identity),
                              if (action != null) ...[
                                const SizedBox(width: 12),
                                action!,
                              ],
                            ],
                          ),
                        if (bio.trim().isNotEmpty) ...[
                          const SizedBox(height: 4),
                          Text(
                            bio.trim(),
                            maxLines: 3,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              fontSize: 15,
                              height: 1.4,
                              color: colors.baseContent,
                            ),
                          ),
                        ],
                        if (stackAction && action != null) ...[
                          const SizedBox(height: 8),
                          action!,
                        ],
                      ],
                    );
                  },
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Compact relationship action; callers own mutation, rollback and auth.
class GfFollowButton extends StatelessWidget {
  const GfFollowButton({
    super.key,
    required this.following,
    required this.label,
    required this.onPressed,
    this.busy = false,
  });
  final bool following, busy;
  final String label;
  final VoidCallback? onPressed;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return OutlinedButton(
      onPressed: busy ? null : onPressed,
      style: OutlinedButton.styleFrom(
        minimumSize: const Size(96, 44),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        shape: const StadiumBorder(),
        backgroundColor: following ? colors.base100 : colors.baseContent,
        foregroundColor: following ? colors.baseContent : colors.base100,
        side: BorderSide(color: following ? colors.line : colors.baseContent),
        textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (busy) ...[
            SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                color: following ? colors.baseContent : colors.base100,
              ),
            ),
            const SizedBox(width: 8),
          ],
          Flexible(child: Text(label, textAlign: TextAlign.center)),
        ],
      ),
    );
  }
}

import 'package:flutter/material.dart';

import 'gf_avatar.dart';

/// Stacked participant avatars, mirroring web `AvatarStack.vue`:
/// - `sm`: h-6 (24px) avatars with `-space-x-2` (8px overlap)
/// - `md`: h-8 (32px) avatars with `-space-x-3` (12px overlap)
///
/// Each avatar carries a 2px base-100 ring so overlaps read as separate
/// circles (web `ring-2 ring-base-100`).
class GfAvatarStack extends StatelessWidget {
  const GfAvatarStack({
    super.key,
    required this.avatarUrls,
    this.size = GfAvatarStackSize.sm,
    this.max = 4,
  });

  final List<String> avatarUrls;

  /// Stack density; `sm` is used in topic rows, `md` in larger cards.
  final GfAvatarStackSize size;

  /// Maximum avatars rendered (web shows at most a handful).
  final int max;

  @override
  Widget build(BuildContext context) {
    final List<String> urls = avatarUrls.take(max).toList();
    final double overlap = size == GfAvatarStackSize.sm ? 8 : 12;
    final double avatarSize = size == GfAvatarStackSize.sm ? 24 : 32;

    return SizedBox(
      height: avatarSize,
      width: urls.isEmpty
          ? 0
          : avatarSize + (urls.length - 1) * (avatarSize - overlap),
      child: Stack(
        clipBehavior: Clip.none,
        children: <Widget>[
          for (int i = 0; i < urls.length; i++)
            Positioned(
              left: i * (avatarSize - overlap),
              child: GfAvatar(src: urls[i], size: avatarSize, ring: true),
            ),
        ],
      ),
    );
  }
}

enum GfAvatarStackSize { sm, md }

import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'atoms/gf_avatar.dart';
import 'gf_card.dart';
import 'gf_chip.dart';
import 'gf_topic_row.dart';
import 'gf_image_viewer.dart';

/// Mobile topic-feed card aligned with the web `TopicFeedPreview` surface.
class GfTopicCard extends StatefulWidget {
  const GfTopicCard({
    super.key,
    required this.title,
    required this.description,
    required this.authorName,
    required this.authorAvatarUrl,
    required this.categories,
    required this.imageUrls,
    required this.activityText,
    required this.replyCount,
    required this.viewCount,
    this.onTap,
    this.onLike,
    this.onBookmark,
    this.likeTooltip,
    this.bookmarkTooltip,
    this.bookmarkedTooltip,
    this.liked = false,
    this.bookmarked = false,
    this.imageAspectRatio,
    this.pinned = false,
    this.unseen = false,
    this.hot = false,
  });

  final String title;
  final String description;
  final String authorName;
  final String authorAvatarUrl;
  final List<GfTopicCategory> categories;
  final List<String> imageUrls;
  final String activityText;
  final int replyCount;
  final int viewCount;
  final VoidCallback? onTap;
  final Future<bool> Function(bool target)? onLike;
  final Future<bool> Function(bool target)? onBookmark;
  final String? likeTooltip;
  final String? bookmarkTooltip;
  final String? bookmarkedTooltip;
  /// Controlled by the owning list so recycling never resets server state.
  final bool liked;
  final bool bookmarked;

  /// Optional known first-image ratio; otherwise decoded from the image stream.
  final double? imageAspectRatio;
  final bool pinned;
  final bool unseen;
  final bool hot;

  @override
  State<GfTopicCard> createState() => _GfTopicCardState();
}

class _GfTopicCardState extends State<GfTopicCard>
    with TickerProviderStateMixin {
  ImageStream? _stream;
  ImageStreamListener? _listener;
  double _ratio = 1.5;
  String? _observedUrl;
  bool get _liked => widget.liked;
  bool get _bookmarked => widget.bookmarked;
  bool _likeBusy = false;
  bool _bookmarkBusy = false;
  late final AnimationController _likeAnimation;
  late final AnimationController _bookmarkAnimation;

  @override
  void initState() {
    super.initState();
    _likeAnimation = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 720),
    );
    _bookmarkAnimation = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 360),
    );
  }

  @override
  void dispose() {
    _likeAnimation.dispose();
    _bookmarkAnimation.dispose();
    if (_listener != null) _stream?.removeListener(_listener!);
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _observeImage();
  }

  @override
  void didUpdateWidget(GfTopicCard oldWidget) {
    super.didUpdateWidget(oldWidget);
    _observeImage();
  }

  Future<void> _toggleLike() async {
    final callback = widget.onLike;
    if (callback == null || _likeBusy) return;
    final target = !_liked;
    setState(() => _likeBusy = true);
    try {
      if (await callback(target) && mounted) {
        if (target && !MediaQuery.disableAnimationsOf(context)) {
          _likeAnimation.forward(from: 0);
        }
      }
    } finally {
      if (mounted) setState(() => _likeBusy = false);
    }
  }

  Future<void> _toggleBookmark() async {
    final callback = widget.onBookmark;
    if (callback == null || _bookmarkBusy) return;
    final target = !_bookmarked;
    setState(() => _bookmarkBusy = true);
    try {
      if (await callback(target) && mounted) {
        if (target && !MediaQuery.disableAnimationsOf(context)) {
          _bookmarkAnimation.forward(from: 0);
        }
      }
    } finally {
      if (mounted) setState(() => _bookmarkBusy = false);
    }
  }

  void _observeImage() {
    final url = widget.imageUrls.where((url) => url.isNotEmpty).firstOrNull;
    if (url == _observedUrl) return;
    if (_listener != null) _stream?.removeListener(_listener!);
    _observedUrl = url;
    _ratio = 1.5;
    if (url == null) return;
    _stream = NetworkImage(url).resolve(createLocalImageConfiguration(context));
    _listener = ImageStreamListener((info, synchronousCall) {
      try {
        if (!mounted || info.image.height == 0) return;
        final ratio = info.image.width / info.image.height;
        if (synchronousCall) {
          _ratio = ratio;
        } else {
          setState(() => _ratio = ratio);
        }
      } finally {
        info.dispose();
      }
    }, onError: (Object error, StackTrace? stack) {});
    _stream!.addListener(_listener!);
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final allImages = widget.imageUrls.where((url) => url.isNotEmpty).toList();
    final images = allImages.take(3).toList();
    final ratio = widget.imageAspectRatio ?? _ratio;
    final portrait = ratio < 1;
    final singleImage = images.length == 1 && portrait;

    Widget photo(int index, {double? width, double height = 104}) =>
        GestureDetector(
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute<void>(
              builder: (_) =>
                  GfImageViewer(images: allImages, initialIndex: index),
            ),
          ),
          child: _TopicImage(
            url: images[index],
            width: width,
            height: height,
            fit: portrait ? BoxFit.cover : BoxFit.contain,
          ),
        );

    final Widget textContent = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        _AuthorMeta(
          name: widget.authorName,
          avatarUrl: widget.authorAvatarUrl,
          activityText: widget.activityText,
          categories: widget.categories,
          hot: widget.hot,
          pinned: widget.pinned,
        ),
        const SizedBox(height: 4),
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          widget.title,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: colors.baseContent,
                            fontSize: 17,
                            height: 1.45,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      if (widget.unseen) ...<Widget>[
                        const SizedBox(width: 6),
                        Container(
                          width: 8,
                          height: 8,
                          margin: const EdgeInsets.only(top: 7),
                          decoration: BoxDecoration(
                            color: colors.primary,
                            shape: BoxShape.circle,
                          ),
                        ),
                      ],
                    ],
                  ),
                  if (widget.description.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 4),
                    Text(
                      widget.description,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.85),
                        fontSize: 17,
                        height: 1.4,
                      ),
                    ),
                  ],
                ],
              ),
            ),
            if (singleImage) ...<Widget>[
              const SizedBox(width: 12),
              photo(0, width: 108, height: 132),
            ],
          ],
        ),
        if (images.isNotEmpty && !singleImage) ...<Widget>[
          const SizedBox(height: 4),
          LayoutBuilder(
            builder: (context, constraints) {
              if (images.length == 1) {
                return photo(
                  0,
                  width: constraints.maxWidth,
                  height: (constraints.maxWidth / ratio).clamp(96.0, 240.0),
                );
              }
              if (!portrait && images.length > 2) {
                return SizedBox(
                  height: 190,
                  child: Stack(
                    children: [
                      for (int i = images.length - 1; i >= 0; i--)
                        Positioned(
                          left: i * 18,
                          right: (2 - i) * 18,
                          top: i * 12,
                          bottom: (2 - i) * 12,
                          child: photo(i, height: 166),
                        ),
                      Positioned(
                        right: 10,
                        bottom: 8,
                        child: _ImageCount(count: allImages.length),
                      ),
                    ],
                  ),
                );
              }
              return Row(
                children: [
                  for (int i = 0; i < images.length; i++) ...[
                    if (i > 0) const SizedBox(width: 6),
                    Expanded(
                      child: Stack(
                        children: [
                          photo(
                            i,
                            width: double.infinity,
                            height: portrait
                                ? 132
                                : ((constraints.maxWidth - 6) / 2 / ratio)
                                      .clamp(72.0, 160.0),
                          ),
                          if (i == images.length - 1 && allImages.length > 3)
                            Positioned(
                              right: 6,
                              bottom: 6,
                              child: _ImageCount(count: allImages.length),
                            ),
                        ],
                      ),
                    ),
                  ],
                ],
              );
            },
          ),
        ],
        const SizedBox(height: 4),
        Wrap(
          alignment: WrapAlignment.spaceBetween,
          crossAxisAlignment: WrapCrossAlignment.center,
          spacing: 8,
          runSpacing: 4,
          children: <Widget>[
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _Metric(
                  icon: Icons.chat_bubble_outline,
                  value: '${widget.replyCount}',
                ),
                const SizedBox(width: 6),
                _Metric(
                  icon: Icons.visibility_outlined,
                  value: '${widget.viewCount}',
                ),
              ],
            ),
            if (widget.onLike != null || widget.onBookmark != null)
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (widget.onLike != null)
                    _LikeAction(
                      liked: _liked,
                      animation: _likeAnimation,
                      activeColor: colors.error,
                      inactiveColor: colors.iconMuted,
                      tooltip: widget.likeTooltip,
                      onPressed: _likeBusy ? null : _toggleLike,
                    ),
                  if (widget.onBookmark != null)
                    _BookmarkAction(
                      bookmarked: _bookmarked,
                      animation: _bookmarkAnimation,
                      activeColor: colors.primary,
                      inactiveColor: colors.iconMuted,
                      tooltip: _bookmarked
                          ? widget.bookmarkedTooltip
                          : widget.bookmarkTooltip,
                      onPressed: _bookmarkBusy ? null : _toggleBookmark,
                    ),
                ],
              ),
          ],
        ),
      ],
    );

    return GfCard(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      onTap: widget.onTap,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          GfAvatar(src: widget.authorAvatarUrl, size: 36),
          const SizedBox(width: 10),
          Expanded(child: textContent),
        ],
      ),
    );
  }
}

class _LikeAction extends StatelessWidget {
  const _LikeAction({
    required this.liked,
    required this.animation,
    required this.activeColor,
    required this.inactiveColor,
    required this.tooltip,
    required this.onPressed,
  });

  final bool liked;
  final Animation<double> animation;
  final Color activeColor;
  final Color inactiveColor;
  final String? tooltip;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: animation,
      builder: (context, child) {
        final double progress = animation.value;
        final double scale = liked && progress > 0 ? _likeScale(progress) : 1;
        final double colorProgress = liked && progress > 0
            ? Curves.easeOut.transform((progress * 2).clamp(0.0, 1.0))
            : 1;
        final Color iconColor = liked
            ? Color.lerp(inactiveColor, activeColor, colorProgress)!
            : inactiveColor;

        return SizedBox(
          width: 36,
          height: 36,
          child: Stack(
            clipBehavior: Clip.none,
            alignment: Alignment.center,
            children: [
              if (liked && progress > 0)
                Positioned.fill(
                  child: IgnorePointer(
                    child: CustomPaint(
                      painter: _LikeBurstPainter(
                        progress: progress,
                        color: activeColor,
                      ),
                    ),
                  ),
                ),
              Transform.scale(
                scale: scale,
                child: IconButton(
                  tooltip: tooltip,
                  onPressed: onPressed,
                  icon: Icon(
                    liked ? Icons.favorite : Icons.favorite_border,
                    size: 18,
                    color: iconColor,
                  ),
                  padding: EdgeInsets.zero,
                  visualDensity: VisualDensity.compact,
                  constraints: const BoxConstraints.tightFor(
                    width: 32,
                    height: 32,
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

double _likeScale(double progress) {
  if (progress < 0.28) {
    return 0.55 + (1.38 - 0.55) * Curves.easeOut.transform(progress / 0.28);
  }
  if (progress < 0.55) {
    return 1.38 -
        (1.38 - 0.88) * Curves.easeInOut.transform((progress - 0.28) / 0.27);
  }
  return 0.88 + (1 - 0.88) * Curves.easeOut.transform((progress - 0.55) / 0.45);
}

class _BookmarkAction extends StatelessWidget {
  const _BookmarkAction({
    required this.bookmarked,
    required this.animation,
    required this.activeColor,
    required this.inactiveColor,
    required this.tooltip,
    required this.onPressed,
  });

  final bool bookmarked;
  final Animation<double> animation;
  final Color activeColor;
  final Color inactiveColor;
  final String? tooltip;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: animation,
      builder: (context, child) {
        final double bounce = bookmarked && animation.value > 0
            ? math.sin(animation.value * math.pi) * 0.18
            : 0;
        return SizedBox(
          width: 36,
          height: 36,
          child: Transform.scale(
            scale: 1 + bounce,
            child: IconButton(
              tooltip: tooltip,
              onPressed: onPressed,
              icon: Icon(
                bookmarked ? Icons.bookmark : Icons.bookmark_border,
                size: 18,
                color: bookmarked ? activeColor : inactiveColor,
              ),
              padding: EdgeInsets.zero,
              visualDensity: VisualDensity.compact,
              constraints: const BoxConstraints.tightFor(width: 32, height: 32),
            ),
          ),
        );
      },
    );
  }
}

class _LikeBurstPainter extends CustomPainter {
  const _LikeBurstPainter({required this.progress, required this.color});

  final double progress;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final Offset center = size.center(Offset.zero);
    final double appear = Curves.easeOut.transform(
      (progress / 0.22).clamp(0.0, 1.0),
    );
    final double fade =
        1 - Curves.easeIn.transform(((progress - 0.56) / 0.44).clamp(0.0, 1.0));
    if (appear == 0 || fade == 0) return;

    final double radius = 5 + 18 * Curves.easeOut.transform(progress);
    final Color burstColor = color.withValues(alpha: 0.86 * appear * fade);
    final Paint rayPaint = Paint()
      ..color = burstColor
      ..strokeWidth = 1.6
      ..strokeCap = StrokeCap.round;
    final Paint dotPaint = Paint()..color = burstColor;

    for (int i = 0; i < 8; i++) {
      final double angle = -math.pi / 2 + i * math.pi / 4;
      final Offset direction = Offset(math.cos(angle), math.sin(angle));
      canvas.drawLine(
        center + direction * (radius * 0.62),
        center + direction * radius,
        rayPaint,
      );
      final double dotRadius = i.isEven ? 2.0 : 1.45;
      canvas.drawCircle(
        center + direction * (radius + 4 + (i.isEven ? 2 : 0)),
        dotRadius,
        dotPaint,
      );
    }

    final Paint ringPaint = Paint()
      ..color = color.withValues(alpha: 0.32 * appear * fade)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.2;
    canvas.drawCircle(center, 9 + radius * 0.42, ringPaint);
  }

  @override
  bool shouldRepaint(_LikeBurstPainter oldDelegate) =>
      oldDelegate.progress != progress || oldDelegate.color != color;
}

class _AuthorMeta extends StatelessWidget {
  const _AuthorMeta({
    required this.name,
    required this.avatarUrl,
    required this.activityText,
    required this.categories,
    required this.hot,
    required this.pinned,
  });

  final String name;
  final String avatarUrl;
  final String activityText;
  final List<GfTopicCategory> categories;
  final bool hot;
  final bool pinned;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Row(
                children: <Widget>[
                  Flexible(
                    child: Text(
                      name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent,
                        fontSize: 16,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Flexible(
                    child: Text(
                      activityText,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.55),
                        fontSize: 13,
                      ),
                    ),
                  ),
                ],
              ),
              if (categories.isNotEmpty || hot) ...<Widget>[
                const SizedBox(height: 4),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: <Widget>[
                    for (final GfTopicCategory category in categories)
                      GfChip(label: category.name, color: category.color),
                    if (hot)
                      Container(
                        height: 20,
                        padding: const EdgeInsets.symmetric(horizontal: 7),
                        decoration: BoxDecoration(
                          color: colors.warning.withValues(alpha: 0.1),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: <Widget>[
                            Icon(
                              Icons.auto_awesome,
                              size: 12,
                              color: colors.warning,
                            ),
                            const SizedBox(width: 3),
                            Text(
                              'hot',
                              style: TextStyle(
                                color: colors.warning,
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ],
            ],
          ),
        ),
        if (pinned) Icon(Icons.push_pin, size: 16, color: colors.error),
      ],
    );
  }
}

class _TopicImage extends StatelessWidget {
  const _TopicImage({
    required this.url,
    this.width,
    required this.height,
    this.fit = BoxFit.cover,
  });

  final String url;
  final double? width;
  final double height;
  final BoxFit fit;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: SizedBox(
        width: width,
        height: height,
        child: Image.network(
          url,
          fit: fit,
          errorBuilder:
              (BuildContext context, Object error, StackTrace? stack) {
                return ColoredBox(
                  color: colors.base200,
                  child: Icon(Icons.image_outlined, color: colors.iconMuted),
                );
              },
        ),
      ),
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.icon, required this.value});

  final IconData icon;
  final String value;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 16, color: colors.iconMuted),
          const SizedBox(width: 5),
          Text(
            value,
            style: TextStyle(
              color: colors.baseContent.withValues(alpha: 0.55),
              fontSize: 13,
            ),
          ),
        ],
      ),
    );
  }
}

class _ImageCount extends StatelessWidget {
  const _ImageCount({required this.count});
  final int count;
  @override
  Widget build(BuildContext context) => DecoratedBox(
    decoration: BoxDecoration(
      color: Colors.black.withValues(alpha: 0.65),
      borderRadius: BorderRadius.circular(20),
    ),
    child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(
            Icons.photo_library_outlined,
            size: 12,
            color: Colors.white,
          ),
          const SizedBox(width: 4),
          Text(
            '$count',
            style: const TextStyle(color: Colors.white, fontSize: 12),
          ),
        ],
      ),
    ),
  );
}

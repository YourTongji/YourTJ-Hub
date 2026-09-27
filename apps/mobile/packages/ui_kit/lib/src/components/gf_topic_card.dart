import 'dart:developer' as developer;
import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'atoms/gf_avatar.dart';
import 'gf_action_feedback.dart';
import 'gf_card.dart';
import 'gf_chip.dart';
import 'gf_image_viewer.dart';
import 'gf_symbol.dart';
import 'gf_topic_row.dart';

class GfTopicImageVariant {
  const GfTopicImageVariant({
    required this.url,
    required this.width,
    required this.height,
  });

  final String url;
  final int width;
  final int height;
}

class GfTopicImageMetadata {
  const GfTopicImageMetadata({
    required this.url,
    required this.width,
    required this.height,
    this.variants = const <GfTopicImageVariant>[],
  });

  final String url;
  final int width;
  final int height;
  final List<GfTopicImageVariant> variants;
}

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
    this.likeCount = 0,
    this.onTap,
    this.onAuthorTap,
    this.imageSemanticLabelBuilder,
    this.onSaveImage,
    this.saveImageLabel = 'Save image',
    this.onShareImage,
    this.shareImageLabel = 'Share image',
    this.onLike,
    this.onBookmark,
    this.onFirstMediaFrame,
    this.likeTooltip,
    this.bookmarkTooltip,
    this.bookmarkedTooltip,
    this.liked = false,
    this.bookmarked = false,
    this.imageMetadata = const <GfTopicImageMetadata>[],
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
  final int likeCount;
  final VoidCallback? onTap;
  final VoidCallback? onAuthorTap;
  final String Function(int index, int count)? imageSemanticLabelBuilder;
  final Future<void> Function(String imageUrl)? onSaveImage;
  final String saveImageLabel;
  final Future<void> Function(String imageUrl)? onShareImage;
  final String shareImageLabel;
  final Future<bool> Function(bool target)? onLike;
  final Future<bool> Function(bool target)? onBookmark;
  final VoidCallback? onFirstMediaFrame;
  final String? likeTooltip;
  final String? bookmarkTooltip;
  final String? bookmarkedTooltip;

  /// Controlled by the owning list so recycling never resets server state.
  final bool liked;
  final bool bookmarked;
  final List<GfTopicImageMetadata> imageMetadata;

  /// Optional first-image ratio; otherwise uses intrinsic metadata or a legacy fallback.
  final double? imageAspectRatio;
  final bool pinned;
  final bool unseen;
  final bool hot;

  @override
  State<GfTopicCard> createState() => _GfTopicCardState();
}

class _GfTopicCardState extends State<GfTopicCard> {
  static bool _firstMediaFrameRecorded = false;

  bool get _liked => widget.liked;
  bool get _bookmarked => widget.bookmarked;
  bool _likeBusy = false;
  bool _bookmarkBusy = false;

  static void _recordFirstMediaFrame(VoidCallback? onFirstMediaFrame) {
    if (!_firstMediaFrameRecorded) {
      _firstMediaFrameRecorded = true;
      developer.Timeline.instantSync('startup.feed_first_media_frame_ready');
    }
    onFirstMediaFrame?.call();
  }

  Future<void> _toggleLike() async {
    final callback = widget.onLike;
    if (callback == null || _likeBusy) return;
    final target = !_liked;
    setState(() => _likeBusy = true);
    try {
      await callback(target);
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
      await callback(target);
    } finally {
      if (mounted) setState(() => _bookmarkBusy = false);
    }
  }

  void _openImage(List<String> images, int index) {
    Navigator.of(context, rootNavigator: true).push(
      MaterialPageRoute<void>(
        // The viewer owns its Scaffold and SafeArea; back its translucent
        // surface with black, matching the Markdown lightbox route.
        builder: (_) => ColoredBox(
          color: Colors.black,
          child: GfImageViewer(
            images: images,
            initialIndex: index,
            onSaveImage: widget.onSaveImage,
            saveImageLabel: widget.saveImageLabel,
            onShareImage: widget.onShareImage,
            shareImageLabel: widget.shareImageLabel,
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final allImages = widget.imageUrls.where((url) => url.isNotEmpty).toList();
    final images = allImages.take(3).toList();
    final imageMetadata = <String, GfTopicImageMetadata>{
      for (final metadata in widget.imageMetadata) metadata.url: metadata,
    };
    final firstImageMetadata = images.isEmpty
        ? null
        : imageMetadata[images.first];
    final intrinsicRatio =
        firstImageMetadata != null &&
            firstImageMetadata.width > 0 &&
            firstImageMetadata.height > 0
        ? firstImageMetadata.width / firstImageMetadata.height
        : null;
    final ratio = widget.imageAspectRatio ?? intrinsicRatio ?? 1.5;
    final portrait = ratio < 1;
    final singleImage = images.length == 1 && portrait;

    Widget photo(int index, {double? width, double height = 104}) => Semantics(
      button: true,
      label:
          widget.imageSemanticLabelBuilder?.call(index + 1, allImages.length) ??
          'View image ${index + 1} of ${allImages.length}',
      child: InkWell(
        onTap: () => _openImage(allImages, index),
        child: _TopicImage(
          url: images[index],
          metadata: imageMetadata[images[index]],
          onFirstMediaFrame: widget.onFirstMediaFrame,
          width: width,
          height: height,
          fit: portrait ? BoxFit.cover : BoxFit.contain,
        ),
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
          onAuthorTap: widget.onAuthorTap,
        ),
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
                            height: 1.35,
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
                    const SizedBox(height: 2),
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
                        child: IgnorePointer(
                          child: _ImageCount(count: allImages.length),
                        ),
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
                              child: IgnorePointer(
                                child: _ImageCount(count: allImages.length),
                              ),
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
        const SizedBox(height: 2),
        Wrap(
          alignment: WrapAlignment.spaceBetween,
          crossAxisAlignment: WrapCrossAlignment.center,
          spacing: 8,
          runSpacing: 4,
          children: <Widget>[
            _Metric(icon: 'message-circle', value: '${widget.replyCount}'),
            _Metric(icon: 'eye', value: '${widget.viewCount}'),
            if (widget.onLike != null)
              _LikeAction(
                count: widget.likeCount,
                liked: _liked,
                activeColor: colors.error,
                inactiveColor: colors.iconMuted,
                tooltip: widget.likeTooltip,
                onPressed: _likeBusy ? null : _toggleLike,
              )
            else
              _Metric(icon: 'heart', value: '${widget.likeCount}'),
            if (widget.onBookmark != null)
              _BookmarkAction(
                bookmarked: _bookmarked,
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
    );

    return GfCard(
      padding: const EdgeInsets.fromLTRB(12, 0, 12, 4),
      onTap: widget.onTap,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Semantics(
              button: widget.onAuthorTap != null,
              label: widget.authorName,
              child: InkWell(
                onTap: widget.onAuthorTap,
                borderRadius: BorderRadius.circular(24),
                child: SizedBox(
                  width: 44,
                  height: 44,
                  child: Center(
                    child: GfAvatar(src: widget.authorAvatarUrl, size: 36),
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(child: textContent),
        ],
      ),
    );
  }
}

class _LikeAction extends StatelessWidget {
  const _LikeAction({
    required this.count,
    required this.liked,
    required this.activeColor,
    required this.inactiveColor,
    required this.tooltip,
    required this.onPressed,
  });
  final int count;
  final bool liked;
  final Color activeColor;
  final Color inactiveColor;
  final String? tooltip;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final color = liked ? activeColor : inactiveColor;
    return GfActionFeedback(
      active: liked,
      onPressed: onPressed,
      child: GfSymbol(liked ? 'heart-filled' : 'heart', size: 18, color: color),
      builder: (activate, visual) => IconButton(
        tooltip: tooltip,
        onPressed: activate,
        padding: const EdgeInsets.symmetric(horizontal: 8),
        constraints: const BoxConstraints(minWidth: 44, minHeight: 44),
        icon: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            visual,
            const SizedBox(width: 5),
            Text('$count', style: TextStyle(color: color, fontSize: 13)),
          ],
        ),
      ),
    );
  }
}

class _BookmarkAction extends StatelessWidget {
  const _BookmarkAction({
    required this.bookmarked,
    required this.activeColor,
    required this.inactiveColor,
    required this.tooltip,
    required this.onPressed,
  });
  final bool bookmarked;
  final Color activeColor;
  final Color inactiveColor;
  final String? tooltip;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) => GfActionFeedback(
    active: bookmarked,
    onPressed: onPressed,
    child: GfSymbol(
      bookmarked ? 'bookmark-filled' : 'bookmark',
      size: 18,
      color: bookmarked ? activeColor : inactiveColor,
    ),
    builder: (activate, visual) => SizedBox(
      width: 44,
      height: 44,
      child: IconButton(
        tooltip: tooltip,
        onPressed: activate,
        icon: visual,
        padding: EdgeInsets.zero,
        constraints: const BoxConstraints.tightFor(width: 44, height: 44),
      ),
    ),
  );
}

class _AuthorMeta extends StatelessWidget {
  const _AuthorMeta({
    required this.name,
    required this.avatarUrl,
    required this.activityText,
    required this.categories,
    required this.hot,
    required this.pinned,
    this.onAuthorTap,
  });

  final String name;
  final String avatarUrl;
  final String activityText;
  final List<GfTopicCategory> categories;
  final bool hot;
  final bool pinned;
  final VoidCallback? onAuthorTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    Widget metadataSlot(Widget child) => ConstrainedBox(
      constraints: const BoxConstraints(minHeight: 44),
      child: Align(
        alignment: Alignment.centerLeft,
        widthFactor: 1,
        heightFactor: 1,
        child: child,
      ),
    );
    final dateStyle = DefaultTextStyle.of(context).style.copyWith(
      color: colors.baseContent.withValues(alpha: 0.55),
      fontSize: 13,
    );
    final markers = <Widget>[
      for (final category in categories)
        metadataSlot(
          GfChip(
            label: category.name,
            color: category.color,
            onTap: category.onTap,
          ),
        ),
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
              GfSymbol('sparkles', size: 12, color: colors.warning),
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
    ];
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Expanded(
          child: LayoutBuilder(
            builder: (context, constraints) {
              const minimumNameWidth = 64.0;
              const minimumMarkerWidth = 64.0;
              const gap = 8.0;
              final datePainter = TextPainter(
                text: TextSpan(text: activityText, style: dateStyle),
                textDirection: Directionality.of(context),
                textScaler: MediaQuery.textScalerOf(context),
                maxLines: 1,
              )..layout();
              final dateWidth = math.min(
                datePainter.width.ceilToDouble(),
                math.max(
                  0.0,
                  constraints.maxWidth -
                      minimumNameWidth -
                      (markers.isEmpty ? gap : minimumMarkerWidth + gap * 2),
                ),
              );
              datePainter.dispose();
              // Give the name the remaining width so it ellipsizes before any
              // metadata can form a second 44px row. A crowded category group
              // scrolls independently without moving the author or timestamp.
              final markerWidth = math.max(
                0.0,
                constraints.maxWidth -
                    minimumNameWidth -
                    (activityText.isEmpty ? 0 : dateWidth + gap) -
                    gap,
              );
              return Row(
                children: <Widget>[
                  Flexible(
                    child: InkWell(
                      onTap: onAuthorTap,
                      child: ConstrainedBox(
                        constraints: BoxConstraints(
                          minWidth: onAuthorTap == null ? 0 : 44,
                          minHeight: 44,
                        ),
                        child: Align(
                          alignment: Alignment.centerLeft,
                          widthFactor: 1,
                          heightFactor: 1,
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
                      ),
                    ),
                  ),
                  if (activityText.isNotEmpty) ...[
                    const SizedBox(width: 8),
                    SizedBox(
                      width: dateWidth,
                      child: metadataSlot(
                        Text(
                          activityText,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: dateStyle,
                        ),
                      ),
                    ),
                  ],
                  if (markers.isNotEmpty) ...[
                    const SizedBox(width: 8),
                    ConstrainedBox(
                      constraints: BoxConstraints(maxWidth: markerWidth),
                      child: SingleChildScrollView(
                        scrollDirection: Axis.horizontal,
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            for (var i = 0; i < markers.length; i++) ...[
                              if (i > 0) const SizedBox(width: 8),
                              markers[i],
                            ],
                          ],
                        ),
                      ),
                    ),
                  ],
                ],
              );
            },
          ),
        ),
        if (pinned)
          Padding(
            padding: const EdgeInsets.only(top: 14),
            child: Semantics(
              label: 'pinned',
              child: GfSymbol('pin-filled', size: 16, color: colors.error),
            ),
          ),
      ],
    );
  }
}

class _TopicImage extends StatelessWidget {
  const _TopicImage({
    required this.url,
    this.metadata,
    this.onFirstMediaFrame,
    this.width,
    required this.height,
    this.fit = BoxFit.cover,
  });

  final String url;
  final GfTopicImageMetadata? metadata;
  final VoidCallback? onFirstMediaFrame;
  final double? width;
  final double height;
  final BoxFit fit;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final logicalWidth = width != null && width!.isFinite
              ? width!
              : constraints.maxWidth.isFinite
              ? constraints.maxWidth
              : MediaQuery.sizeOf(context).width;
          final int pixelWidth =
              (logicalWidth * MediaQuery.devicePixelRatioOf(context))
                  .round()
                  .clamp(1, 1000000)
                  .toInt();
          final source = _closestImageSource(metadata, pixelWidth);
          final int? cacheHeight = source == null
              ? null
              : (pixelWidth * source.height / source.width)
                    .round()
                    .clamp(1, 1000000)
                    .toInt();
          return SizedBox(
            width: width,
            height: height,
            child: Image(
              image: _feedImageProvider(
                source?.url ?? url,
                pixelWidth,
                cacheHeight,
              ),
              fit: fit,
              frameBuilder: (context, child, frame, wasSynchronouslyLoaded) {
                if (frame != null) {
                  _GfTopicCardState._recordFirstMediaFrame(onFirstMediaFrame);
                }
                return child;
              },
              errorBuilder:
                  (BuildContext context, Object error, StackTrace? stack) {
                    return ColoredBox(
                      color: colors.base200,
                      child: GfSymbol('image-off', color: colors.iconMuted),
                    );
                  },
            ),
          );
        },
      ),
    );
  }
}

GfTopicImageVariant? _closestImageSource(
  GfTopicImageMetadata? metadata,
  int targetWidth,
) {
  if (metadata == null || metadata.width < 1 || metadata.height < 1) {
    return null;
  }
  final candidates = <GfTopicImageVariant>[
    ...metadata.variants.where(
      (variant) =>
          variant.url.isNotEmpty &&
          variant.width > 0 &&
          variant.height > 0 &&
          variant.width <= metadata.width &&
          variant.height <= metadata.height,
    ),
    GfTopicImageVariant(
      url: metadata.url,
      width: metadata.width,
      height: metadata.height,
    ),
  ];
  var closest = candidates.first;
  for (final candidate in candidates.skip(1)) {
    final distance = (candidate.width - targetWidth).abs();
    final closestDistance = (closest.width - targetWidth).abs();
    if (distance < closestDistance ||
        (distance == closestDistance && candidate.width > closest.width)) {
      closest = candidate;
    }
  }
  return closest;
}

ResizeImage _feedImageProvider(String url, int pixelWidth, int? pixelHeight) {
  return ResizeImage(NetworkImage(url), width: pixelWidth, height: pixelHeight);
}

class _Metric extends StatelessWidget {
  const _Metric({required this.icon, required this.value});

  final String icon;
  final String value;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          GfSymbol(icon, size: 18, color: colors.iconMuted),
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
          const GfSymbol('images', size: 12, color: Colors.white),
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

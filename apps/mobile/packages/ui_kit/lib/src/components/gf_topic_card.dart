import 'atoms/gf_decorated_name.dart';
import 'dart:async';
import 'dart:developer' as developer;
import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import '../twemoji/twemoji_text.dart';
import 'atoms/gf_avatar.dart';
import 'gf_action_feedback.dart';
import 'gf_card.dart';
import 'gf_chip.dart';
import 'gf_image_viewer.dart';
import 'gf_symbol.dart';
import 'gf_media_image.dart';
import 'gf_motion.dart';
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
    this.avatarFrameColors = const [],
    this.avatarFrameAssetUrl = '',
    this.avatarFrameScale = 1.25,
    this.nicknameColors = const [],
    required this.authorAvatarUrl,
    required this.categories,
    required this.imageUrls,
    required this.activityText,
    required this.replyCount,
    required this.viewCount,
    this.likeCount = 0,
    this.onTap,
    this.onAuthorTap,
    this.onAvatarTap,
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
    this.feedContext,
    this.descriptionSpan,
    this.titleSpan,
  });

  final String title;
  final String description;

  /// Prebuilt rich title: an untitled moment's excerpt lands in the title slot
  /// and keeps its sticker images. When present it owns the text and [title]
  /// is ignored. Inline images are not tappable, like the excerpt.
  final InlineSpan? titleSpan;

  /// Prebuilt rich excerpt (page-level sticker resolution): when present it
  /// owns the text and [description] stays empty. Sticker images inside it are
  /// not tappable so the card's own tap target stays the only one.
  final InlineSpan? descriptionSpan;
  final String avatarFrameAssetUrl;
  final double avatarFrameScale;
  final String authorName;
  final List<Color> avatarFrameColors, nicknameColors;
  final String authorAvatarUrl;
  final List<GfTopicCategory> categories;
  final List<String> imageUrls;
  final String activityText;
  final int replyCount;
  final int viewCount;
  final int likeCount;
  final VoidCallback? onTap;
  final VoidCallback? onAuthorTap;
  final VoidCallback? onAvatarTap;
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

  /// Personalized-feed reason, stamped as a seal at the foot of the avatar
  /// column and joined to the avatar by a trail; tapping it names the reason.
  final GfTopicContext? feedContext;

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

  /// Sticker spans carry their own image widget; the card owns taps, so inline
  /// content must not compete for them.
  Widget _excerpt(GfColors colors) {
    final TextStyle style = TextStyle(
      color: colors.baseContent.withValues(alpha: 0.85),
      fontSize: 17,
      height: 1.4,
    );
    final InlineSpan? span = widget.descriptionSpan;
    if (span == null) {
      return GfText(
        widget.description,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: style,
      );
    }
    return IgnorePointer(
      child: GfText.rich(
        span,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: style,
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
          nicknameColors: widget.nicknameColors,
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
                      // 无标题瞬间（title 为空串）不渲染标题，未读圆点保持右对齐。
                      if (widget.titleSpan case final InlineSpan span)
                        Expanded(
                          child: IgnorePointer(
                            child: GfText.rich(
                              span,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: GfTheme.typographyOf(context).title3,
                            ),
                          ),
                        )
                      else if (widget.title.isNotEmpty)
                        Expanded(
                          child: GfText(
                            widget.title,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: GfTheme.typographyOf(context).title3,
                          ),
                        )
                      else
                        const Spacer(),
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
                  if (widget.description.isNotEmpty ||
                      widget.descriptionSpan != null) ...<Widget>[
                    const SizedBox(height: 2),
                    _excerpt(colors),
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
      ],
    );

    final GfTopicContext? feedContext = widget.feedContext;
    final Widget footer = Wrap(
      alignment: WrapAlignment.spaceBetween,
      crossAxisAlignment: WrapCrossAlignment.center,
      spacing: 8,
      runSpacing: 4,
      children: <Widget>[
        _Metric(
          icon: 'message-circle',
          value: '${widget.replyCount}',
          color: widget.feedContext?.newReplies == true ? colors.primary : null,
        ),
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
    );

    final Widget byline = Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Semantics(
            button: (widget.onAvatarTap ?? widget.onAuthorTap) != null,
            label: widget.authorName,
            child: InkWell(
              onTap: widget.onAvatarTap ?? widget.onAuthorTap,
              borderRadius: BorderRadius.circular(GfRadius.xl),
              child: SizedBox(
                width: 44,
                height: 44,
                child: Center(
                  child: GfAvatar(
                    frameColors: widget.avatarFrameColors,
                    frameAssetUrl: widget.avatarFrameAssetUrl,
                    frameScale: widget.avatarFrameScale,
                    src: widget.authorAvatarUrl,
                    size: 36,
                  ),
                ),
              ),
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(child: textContent),
      ],
    );
    final Widget footerLine = Padding(
      padding: const EdgeInsetsDirectional.only(start: 54),
      child: footer,
    );
    return GfCard(
      padding: const EdgeInsets.fromLTRB(12, 0, 12, 4),
      onTap: widget.onTap,
      child: feedContext == null
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[byline, footerLine],
            )
          : Stack(
              children: <Widget>[
                Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[
                    byline,
                    // A line as tall as its tallest control, so the seal's
                    // slot centers on it.
                    ConstrainedBox(
                      constraints: const BoxConstraints(
                        minHeight: _FeedReasonState._line,
                      ),
                      child: Row(
                        children: <Widget>[Expanded(child: footerLine)],
                      ),
                    ),
                  ],
                ),
                PositionedDirectional(
                  start: 0,
                  top: 0,
                  bottom: 0,
                  width: 44,
                  child: _FeedReason(feedContext: feedContext),
                ),
              ],
            ),
    );
  }
}

/// The avatar column, read top to bottom: who wrote this, then why it is here.
/// A hairline trail runs on the avatar's axis down to a seal on the footer
/// line, which is otherwise empty space, so the card keeps its height and
/// every reason lands in one place, clear of the trailing bookmark.
///
/// The seal's ring marks it as a control. A tap traces the trail from the
/// avatar to the seal, then the seal unrolls its label along the footer line
/// as a pill of the same fill, ring and height; both retract together.
class _FeedReason extends StatefulWidget {
  const _FeedReason({required this.feedContext});

  final GfTopicContext feedContext;

  @override
  State<_FeedReason> createState() => _FeedReasonState();
}

class _FeedReasonState extends State<_FeedReason>
    with SingleTickerProviderStateMixin {
  static const double _seal = 28;
  static const double _gap = 6;
  // The like control's padded tap target sets the footer line's height.
  static const double _line = kMinInteractiveDimension;
  // The avatar's 36px disc ends 44px down its column (see the byline).
  static const double _trailTop = 44 + _gap;
  static const double _trailBottom = (_line + _seal) / 2 + _gap;
  static const Duration _hold = Duration(seconds: 2);

  late final AnimationController _trace = AnimationController(
    vsync: this,
    duration: GfMotion.overlay,
    reverseDuration: GfMotion.layout,
  );
  late final Animation<double> _progress = CurvedAnimation(
    parent: _trace,
    curve: GfMotion.enterCurve,
    reverseCurve: GfMotion.exitCurve,
  );
  Timer? _retract;
  bool _pressed = false;

  void _triggered() {
    final reduced = GfMotion.reducedOf(context);
    _retract?.cancel();
    reduced ? _trace.value = 1 : _trace.forward();
    _retract = Timer(_hold, () {
      if (!mounted) return;
      reduced ? _trace.value = 0 : _trace.reverse();
    });
  }

  void _press(bool pressed) {
    if (_pressed != pressed) setState(() => _pressed = pressed);
  }

  @override
  void dispose() {
    _retract?.cancel();
    _trace.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool newReplies = widget.feedContext.newReplies;
    final Color rest = colors.line;
    final Color traced = newReplies ? colors.primary : colors.iconMuted;
    final Color fill = newReplies
        ? Color.alphaBlend(
            colors.primary.withValues(alpha: 0.1),
            colors.base100,
          )
        : colors.base200;
    final TextDirection direction = Directionality.of(context);
    // The pill wears the seal's fill and its fully traced ring.
    final ShapeBorder pill = StadiumBorder(side: BorderSide(color: traced));

    final Widget seal = Tooltip(
      message: widget.feedContext.label,
      triggerMode: TooltipTriggerMode.tap,
      onTriggered: _triggered,
      showDuration: _hold,
      constraints: const BoxConstraints(minHeight: _seal),
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: ShapeDecoration(color: fill, shape: pill),
      textStyle: TextStyle(
        color: newReplies ? colors.primary : colors.textSecondary,
        fontSize: 12,
        height: 1.35,
        fontWeight: FontWeight.w500,
      ),
      // Unrolls from the seal along the footer line instead of floating over
      // the content above it.
      positionDelegate: (place) => Offset(
        direction == TextDirection.ltr
            ? place.target.dx + _seal / 2 + _gap
            : place.target.dx - _seal / 2 - _gap - place.tooltipSize.width,
        place.target.dy - place.tooltipSize.height / 2,
      ),
      child: Listener(
        onPointerDown: (_) => _press(true),
        onPointerUp: (_) => _press(false),
        onPointerCancel: (_) => _press(false),
        child: ColoredBox(
          color: Colors.transparent,
          child: Center(
            child: AnimatedScale(
              scale: _pressed ? 0.96 : 1,
              duration: GfMotion.press,
              child: AnimatedBuilder(
                animation: _progress,
                builder: (context, child) => Container(
                  width: _seal,
                  height: _seal,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: fill,
                    shape: BoxShape.circle,
                    // The ring takes the trace's color as it arrives.
                    border: Border.all(
                      color: Color.lerp(
                        rest,
                        traced,
                        ((_progress.value - 0.6) / 0.4).clamp(0, 1),
                      )!,
                    ),
                  ),
                  child: child,
                ),
                child: widget.feedContext.mark(context, size: 16),
              ),
            ),
          ),
        ),
      ),
    );

    return LayoutBuilder(
      builder: (context, constraints) {
        final bool trail =
            constraints.maxHeight - _trailTop - _trailBottom >= 8;
        return Stack(
          children: <Widget>[
            if (trail)
              Positioned(
                left: 0,
                right: 0,
                top: _trailTop,
                bottom: _trailBottom,
                child: CustomPaint(
                  key: const Key('gf-topic-card-reason-trail'),
                  painter: _TrailPainter(
                    progress: _progress,
                    rest: rest,
                    traced: traced,
                  ),
                ),
              ),
            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              height: _line,
              child: seal,
            ),
          ],
        );
      },
    );
  }
}

class _TrailPainter extends CustomPainter {
  _TrailPainter({
    required this.progress,
    required this.rest,
    required this.traced,
  }) : super(repaint: progress);

  final Animation<double> progress;
  final Color rest;
  final Color traced;

  @override
  void paint(Canvas canvas, Size size) {
    final double x = size.width / 2;
    final paint = Paint()
      ..strokeWidth = 1.5
      ..strokeCap = StrokeCap.round
      ..color = rest;
    canvas.drawLine(Offset(x, 0), Offset(x, size.height), paint);
    final double t = progress.value;
    if (t > 0) {
      canvas.drawLine(
        Offset(x, 0),
        Offset(x, size.height * t),
        paint..color = traced,
      );
    }
  }

  @override
  bool shouldRepaint(_TrailPainter old) =>
      old.progress != progress || old.rest != rest || old.traced != traced;
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
            GfText('$count', style: TextStyle(color: color, fontSize: 13)),
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
    this.nicknameColors = const [],
    required this.avatarUrl,
    required this.activityText,
    required this.categories,
    required this.hot,
    required this.pinned,
    this.onAuthorTap,
  });

  final String name;
  final List<Color> nicknameColors;
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
      color: colors.textSecondary,
      fontSize: 13,
      height: 18 / 13,
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
          padding: const EdgeInsets.symmetric(horizontal: 6),
          decoration: BoxDecoration(
            color: colors.toneSoft(GfTone.warning),
            borderRadius: BorderRadius.circular(GfRadius.xs),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              GfSymbol(
                'sparkles',
                size: 12,
                color: colors.toneOnSoft(GfTone.warning),
              ),
              const SizedBox(width: 3),
              GfText(
                'hot',
                style: TextStyle(
                  color: colors.toneOnSoft(GfTone.warning),
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
                          child: GfDecoratedName(
                            name,
                            colors: nicknameColors,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: GfTheme.typographyOf(
                              context,
                            ).bodyStrong.copyWith(height: 20 / 16),
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
                        GfText(
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
      borderRadius: BorderRadius.circular(GfRadius.md),
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
          final image = _feedImageProvider(
            context,
            source?.url ?? url,
            pixelWidth,
            cacheHeight,
          );
          return SizedBox(
            width: width,
            height: height,
            child: Stack(
              fit: StackFit.expand,
              children: [
                ImageFiltered(
                  imageFilter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
                  child: Image(
                    image: image,
                    fit: BoxFit.cover,
                    gaplessPlayback: false,
                    errorBuilder: (_, _, _) => const SizedBox.shrink(),
                  ),
                ),
                ColoredBox(color: colors.base100.withValues(alpha: .46)),
                Image(
                  image: image,
                  fit: fit,
                  frameBuilder:
                      (context, child, frame, wasSynchronouslyLoaded) {
                        if (frame != null) {
                          _GfTopicCardState._recordFirstMediaFrame(
                            onFirstMediaFrame,
                          );
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
              ],
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

ImageProvider<Object> _feedImageProvider(
  BuildContext context,
  String url,
  int pixelWidth,
  int? pixelHeight,
) {
  return GfMediaScope.imageProvider(
    context,
    url,
    width: pixelWidth,
    height: pixelHeight,
  );
}

class _Metric extends StatelessWidget {
  const _Metric({required this.icon, required this.value, this.color});

  final String icon;
  final String value;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          GfSymbol(icon, size: 18, color: color ?? colors.textSecondary),
          const SizedBox(width: GfSpace.xs),
          GfText(
            value,
            style: TextStyle(
              color: color ?? colors.textSecondary,
              fontSize: 13,
              height: 18 / 13,
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
      borderRadius: BorderRadius.circular(GfRadius.lg),
    ),
    child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const GfSymbol('images', size: 12, color: Colors.white),
          const SizedBox(width: 4),
          GfText(
            '$count',
            style: const TextStyle(color: Colors.white, fontSize: 12),
          ),
        ],
      ),
    ),
  );
}

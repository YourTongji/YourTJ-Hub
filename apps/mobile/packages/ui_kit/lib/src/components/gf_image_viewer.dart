import 'dart:math' as math;

import 'package:extended_image/extended_image.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';

import '../theme/gf_theme.dart';
import 'atoms/gf_loading_indicator.dart';
import 'gf_glass_icon_button.dart';
import 'gf_motion.dart';
import 'gf_media_image.dart';
import 'gf_symbol.dart';
import 'surfaces/gf_context_menu.dart';

/// Shows the shared image action surface used by inline images and the
/// full-screen viewer. The host decides what saving means on each platform.
Future<bool> showGfImageSaveMenu(
  BuildContext context, {
  required String saveImageLabel,
  Offset? globalPosition,
}) async {
  return await showGfActionMenu<bool>(
        context,
        sourceRect: gfMenuSourceRectOf(context, globalPosition: globalPosition),
        actions: [
          GfContextAction(
            value: true,
            label: saveImageLabel,
            symbol: 'download',
          ),
        ],
      ) ??
      false;
}

/// A transparent media route keeps the source visible during drag-to-dismiss.
/// Callers retain ownership of account guards and origin policies in [builder].
PageRoute<void> gfImageViewerRoute(
  BuildContext context, {
  required WidgetBuilder builder,
}) => PageRouteBuilder<void>(
  opaque: false,
  transitionDuration: GfMotion.duration(context, GfMotion.overlay),
  reverseTransitionDuration: GfMotion.duration(context, GfMotion.content),
  pageBuilder: (context, _, _) =>
      HeroMode(enabled: !GfMotion.reducedOf(context), child: builder(context)),
  transitionsBuilder: (_, animation, _, child) =>
      GfFadeTransition(animation: animation, offset: Offset.zero, child: child),
);

/// Only fly back to a mounted thumbnail fully inside the viewport and its
/// ancestor clips. Cached, scrolled-out sources instead use the route fade.
bool gfImageSourceIsVisible(GlobalKey key) {
  final context = key.currentContext;
  if (context == null || !context.mounted) return false;
  final source = context.findRenderObject();
  if (source is! RenderBox || !source.attached || !source.hasSize) return false;
  final bounds = MatrixUtils.transformRect(
    source.getTransformTo(null),
    Offset.zero & source.size,
  );
  if (bounds.isEmpty || !bounds.isFinite) return false;
  final view = View.of(context);
  var visible = Offset.zero & (view.physicalSize / view.devicePixelRatio);
  RenderObject child = source;
  for (
    var ancestor = source.parent;
    ancestor != null;
    ancestor = ancestor.parent
  ) {
    if (ancestor is RenderOffstage && ancestor.offstage) return false;
    final clip = ancestor.describeApproximatePaintClip(child);
    if (clip != null) {
      visible = visible.intersect(
        MatrixUtils.transformRect(ancestor.getTransformTo(null), clip),
      );
    }
    child = ancestor;
  }
  return visible.contains(bounds.topLeft) &&
      visible.contains(bounds.bottomRight - const Offset(.01, .01));
}

/// Full-screen mobile image viewer with swipe, pinch, double-tap zoom,
/// vertical drag-to-dismiss, and tap-to-toggle chrome.
class GfImageViewer extends StatefulWidget {
  const GfImageViewer({
    super.key,
    required this.images,
    this.initialIndex = 0,
    this.heroTag,
    this.canReturnToSource,
    this.onPageChanged,
    this.enableActualSize = true,
    this.onSaveImage,
    this.saveImageLabel = 'Save image',
    this.onShareImage,
    this.shareImageLabel = 'Share image',
  });

  /// Image URLs to display.
  final List<String> images;
  final int initialIndex;

  /// Optional gallery identity; per-image Hero tags are derived from it.
  final Object? heroTag;

  /// Re-evaluated when closing, after a source may scroll, disappear or change.
  final bool Function(int index)? canReturnToSource;

  /// Keeps a still-current source gallery aligned with the focused image.
  final ValueChanged<int>? onPageChanged;

  /// Retained for the existing original-size viewing requirement.
  final bool enableActualSize;

  /// Saves the currently focused image. The host app owns the platform-specific
  /// implementation (gallery on mobile, file download on web).
  final Future<void> Function(String imageUrl)? onSaveImage;

  /// Label shown in the long-press action sheet.
  final String saveImageLabel;

  /// Shares the currently focused image through the host app's platform
  /// integration.
  final Future<void> Function(String imageUrl)? onShareImage;

  /// Accessibility label shown for the share action.
  final String shareImageLabel;

  @override
  State<GfImageViewer> createState() => _GfImageViewerState();
}

enum _ImageDragAxis { undecided, horizontal, vertical }

class _GfImageViewerState extends State<GfImageViewer>
    with SingleTickerProviderStateMixin {
  static const double _thumbnailItemExtent = 64;
  static const double _verticalDismissDominanceRatio = 1.5;
  static const double _horizontalTakeoverDominanceRatio = 1.05;

  late final ExtendedPageController _pageController;
  final ScrollController _thumbnailController = ScrollController();
  late final AnimationController _doubleTapController;
  final _slidePageKey = GlobalKey<ExtendedImageSlidePageState>();
  Duration? _initialSlideResetDuration;
  late int _currentIndex;
  bool _actualSize = false;
  bool _chromeVisible = true;
  bool _isSharing = false;
  ExtendedImageGestureState? _doubleTapState;
  Offset? _doubleTapPosition;
  double _doubleTapStartScale = 1.0;
  double _doubleTapTargetScale = 1.0;
  final Map<int, Size> _imageSizes = <int, Size>{};
  final Map<int, GlobalKey<ExtendedImageGestureState>> _gestureKeys =
      <int, GlobalKey<ExtendedImageGestureState>>{};
  final Set<int> _activePointers = <int>{};
  bool _multiTouch = false;
  bool _pointerMoved = false;
  Offset? _pointerStart;
  Offset? _lastSwipePosition;
  Duration? _pointerStartTime;
  double _gestureTouchSlop = kTouchSlop;
  double _pageAtPointerDown = 0;
  VelocityTracker? _dismissVelocityTracker;
  _ImageDragAxis _imageDragAxis = _ImageDragAxis.undecided;
  bool _trackDismissGesture = false;
  bool _slideResetInterrupted = false;
  bool _cancelingSlide = false;
  Offset? _lastTapPosition;
  Duration? _lastTapTime;

  @override
  void initState() {
    super.initState();
    assert(widget.images.isNotEmpty);
    _currentIndex = widget.initialIndex.clamp(0, widget.images.length - 1);
    _pageController = ExtendedPageController(initialPage: _currentIndex);
    _doubleTapController = AnimationController(
      vsync: this,
      duration: GfMotion.overlay,
    )..addListener(_applyDoubleTapScale);
    _doubleTapController.addStatusListener(_finishDoubleTapScale);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _centerThumbnail(_currentIndex, animate: false);
    });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final duration = GfMotion.duration(context, GfMotion.content);
    _initialSlideResetDuration ??= duration;
    // extended_image 9.1.0 recreates a SingleTicker controller when the widget
    // duration changes. Update its public controller instead, retaining the
    // page, zoom state and the package's existing animation listeners.
    final reset = _slidePageKey.currentState?.backAnimationController;
    if (reset != null) {
      reset.duration = duration;
      if (GfMotion.reducedOf(context) && reset.isAnimating) reset.value = 1;
    }
    if (GfMotion.reducedOf(context) && _doubleTapController.isAnimating) {
      _doubleTapController.value = 1;
    }
  }

  @override
  void dispose() {
    _doubleTapController.dispose();
    _pageController.dispose();
    _thumbnailController.dispose();
    super.dispose();
  }

  GlobalKey<ExtendedImageGestureState> _gestureKeyFor(int index) =>
      _gestureKeys.putIfAbsent(index, GlobalKey<ExtendedImageGestureState>.new);

  void _centerThumbnail(int index, {required bool animate}) {
    if (widget.images.length < 2 || !_thumbnailController.hasClients) return;

    final ScrollPosition position = _thumbnailController.position;
    final double target = (index * _thumbnailItemExtent)
        .clamp(0.0, position.maxScrollExtent)
        .toDouble();
    if (!animate ||
        MediaQuery.disableAnimationsOf(context) ||
        (position.pixels - target).abs() < 0.5) {
      _thumbnailController.jumpTo(target);
    } else {
      _thumbnailController.animateTo(
        target,
        duration: GfMotion.overlay,
        curve: GfMotion.enterCurve,
      );
    }
  }

  void _selectThumbnail(int index) {
    _doubleTapController.stop();
    _doubleTapState = null;
    _doubleTapPosition = null;
    _gestureKeys[_currentIndex]?.currentState?.reset();
    _gestureKeys[index]?.currentState?.reset();
    setState(() => _actualSize = false);

    if (index == _currentIndex) {
      _centerThumbnail(index, animate: true);
    } else if (GfMotion.reducedOf(context) ||
        (index - _currentIndex).abs() > 1) {
      _pageController.jumpToPage(index);
    } else {
      _pageController.animateToPage(
        index,
        duration: GfMotion.overlay,
        curve: GfMotion.enterCurve,
      );
    }
  }

  Widget _buildThumbnailRail(
    BuildContext context,
    GfColors colors,
    bool reduceMotion,
  ) => Positioned(
    left: 0,
    right: 0,
    bottom: 0,
    child: IgnorePointer(
      key: const Key('gf-image-viewer-thumbnail-interaction'),
      ignoring: !_chromeVisible,
      child: ExcludeSemantics(
        key: const Key('gf-image-viewer-thumbnail-semantics'),
        excluding: !_chromeVisible,
        child: AnimatedOpacity(
          key: const Key('gf-image-viewer-thumbnail-opacity'),
          opacity: _chromeVisible ? 1 : 0,
          duration: reduceMotion ? Duration.zero : GfMotion.content,
          child: SizedBox(
            height: 76,
            child: DecoratedBox(
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: <Color>[
                    Colors.transparent,
                    Colors.black.withValues(alpha: 0.48),
                  ],
                ),
              ),
              child: LayoutBuilder(
                builder: (BuildContext context, BoxConstraints constraints) {
                  final double sidePadding = math.max(
                    0.0,
                    (constraints.maxWidth - _thumbnailItemExtent) / 2,
                  );
                  return AnimatedBuilder(
                    animation: _thumbnailController,
                    builder: (BuildContext context, Widget? child) {
                      final double offset = _thumbnailController.hasClients
                          ? _thumbnailController.offset
                          : _currentIndex * _thumbnailItemExtent;
                      return Semantics(
                        container: true,
                        label: 'Image thumbnails',
                        explicitChildNodes: true,
                        child: ListView.builder(
                          key: const Key('gf-image-viewer-thumbnail-rail'),
                          controller: _thumbnailController,
                          scrollDirection: Axis.horizontal,
                          physics: const BouncingScrollPhysics(),
                          padding: EdgeInsets.symmetric(
                            horizontal: sidePadding,
                          ),
                          itemExtent: _thumbnailItemExtent,
                          itemCount: widget.images.length,
                          itemBuilder: (BuildContext context, int index) {
                            final double distance =
                                (index - offset / _thumbnailItemExtent).abs();
                            final double scale = (1 - distance * 0.09)
                                .clamp(0.68, 1.0)
                                .toDouble();
                            final double opacity = (1 - distance * 0.18)
                                .clamp(0.38, 1.0)
                                .toDouble();
                            final double darkness = (distance * 0.11)
                                .clamp(0.0, 0.32)
                                .toDouble();
                            final bool selected = index == _currentIndex;
                            final double pixelRatio =
                                MediaQuery.devicePixelRatioOf(context);
                            final thumbnail = GfMediaScope.imageProvider(
                              context,
                              widget.images[index],
                              policy: ResizeImagePolicy.fit,
                              width: (48 * pixelRatio).round(),
                              height: (60 * pixelRatio).round(),
                            );
                            return Semantics(
                              key: ValueKey(
                                'gf-image-viewer-thumbnail-semantic-$index',
                              ),
                              button: true,
                              selected: selected,
                              label:
                                  'Image ${index + 1} of ${widget.images.length}',
                              child: GestureDetector(
                                key: ValueKey(
                                  'gf-image-viewer-thumbnail-$index',
                                ),
                                behavior: HitTestBehavior.opaque,
                                onTap: () => _selectThumbnail(index),
                                child: SizedBox(
                                  width: _thumbnailItemExtent,
                                  height: 76,
                                  child: Center(
                                    child: Transform.translate(
                                      offset: Offset(
                                        0,
                                        distance < 0.01
                                            ? -2
                                            : math.min(4.0, distance * 1.2),
                                      ),
                                      child: Transform.scale(
                                        key: ValueKey(
                                          'gf-image-viewer-thumbnail-visual-$index',
                                        ),
                                        scale: scale,
                                        child: Opacity(
                                          opacity: opacity,
                                          child: DecoratedBox(
                                            decoration: BoxDecoration(
                                              border: Border.all(
                                                color: Colors.white.withValues(
                                                  alpha: selected ? 1 : 0.42,
                                                ),
                                                width: selected ? 2 : 1,
                                              ),
                                              borderRadius:
                                                  BorderRadius.circular(8),
                                            ),
                                            child: Padding(
                                              padding: EdgeInsets.all(
                                                selected ? 2 : 1,
                                              ),
                                              child: ClipRRect(
                                                borderRadius:
                                                    BorderRadius.circular(5),
                                                child: ColorFiltered(
                                                  colorFilter: ColorFilter.mode(
                                                    Colors.black.withValues(
                                                      alpha: darkness,
                                                    ),
                                                    BlendMode.darken,
                                                  ),
                                                  child: Image(
                                                    image: thumbnail,
                                                    width: 44,
                                                    height: 56,
                                                    fit: BoxFit.cover,
                                                    gaplessPlayback: false,
                                                    excludeFromSemantics: true,
                                                    errorBuilder: (_, _, _) =>
                                                        ColoredBox(
                                                          color: colors.base200,
                                                          child: Center(
                                                            child: GfSymbol(
                                                              'image-off',
                                                              size: 18,
                                                              color: colors
                                                                  .iconMuted,
                                                            ),
                                                          ),
                                                        ),
                                                  ),
                                                ),
                                              ),
                                            ),
                                          ),
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            );
                          },
                        ),
                      );
                    },
                  );
                },
              ),
            ),
          ),
        ),
      ),
    ),
  );

  void _finishDoubleTapScale(AnimationStatus status) {
    if (status != AnimationStatus.completed) return;
    final ExtendedImageGestureState? state = _doubleTapState;
    if (state != null) state.handleScaleEnd(ScaleEndDetails());
    _doubleTapState = null;
    _doubleTapPosition = null;
  }

  void _applyDoubleTapScale() {
    final ExtendedImageGestureState? state = _doubleTapState;
    final Offset? position = _doubleTapPosition;
    if (!mounted || state == null || position == null) return;

    final double progress = GfMotion.enterCurve.transform(
      _doubleTapController.value,
    );
    final double scale =
        _doubleTapStartScale +
        (_doubleTapTargetScale - _doubleTapStartScale) * progress;
    state.handleScaleUpdate(
      ScaleUpdateDetails(
        focalPoint: position,
        scale: scale / _doubleTapStartScale,
        focalPointDelta: Offset.zero,
        pointerCount: 2,
      ),
    );
  }

  void _handleDoubleTap(ExtendedImageGestureState state, int index) {
    final GestureDetails? details = state.gestureDetails;
    final GestureConfig? config = state.imageGestureConfig;
    if (details == null || config == null) return;

    final double currentScale = details.totalScale ?? config.minScale;
    final double targetScale = currentScale > config.minScale + 0.1
        ? config.minScale
        : _smartDoubleTapScale(_imageSizes[index], config);
    final Offset position =
        state.pointerDownPosition ??
        Offset(
          MediaQuery.sizeOf(context).width / 2,
          MediaQuery.sizeOf(context).height / 2,
        );

    _doubleTapController.stop();
    _doubleTapState = state;
    _doubleTapPosition = position;
    _doubleTapStartScale = currentScale;
    _doubleTapTargetScale = targetScale;
    state.handleScaleStart(ScaleStartDetails(focalPoint: position));
    if (GfMotion.reducedOf(context)) {
      _doubleTapController.value = 1;
      _applyDoubleTapScale();
      _finishDoubleTapScale(AnimationStatus.completed);
    } else {
      _doubleTapController.forward(from: 0);
    }
  }

  double _smartDoubleTapScale(Size? imageSize, GestureConfig config) {
    if (imageSize == null || imageSize.width <= 0 || imageSize.height <= 0) {
      return math
          .min(config.maxScale, 2.0)
          .clamp(config.minScale, config.maxScale);
    }

    final Size viewport = MediaQuery.sizeOf(context);
    final double imageAspect = imageSize.width / imageSize.height;
    final double viewportAspect = viewport.width / viewport.height;
    final double displayedWidth;
    final double displayedHeight;
    if (imageAspect > viewportAspect) {
      displayedWidth = viewport.width;
      displayedHeight = viewport.width / imageAspect;
    } else {
      displayedHeight = viewport.height;
      displayedWidth = viewport.height * imageAspect;
    }

    final double widthScale = viewport.width / displayedWidth;
    final double heightScale = viewport.height / displayedHeight;
    final double target = imageAspect < 0.8
        ? widthScale
        : imageAspect > 1.25
        ? heightScale
        : math.max(widthScale, heightScale);
    return target.clamp(1.5, config.maxScale).toDouble();
  }

  Future<void> _showImageActions(
    BuildContext context,
    Offset globalPosition,
  ) async {
    final Future<void> Function(String imageUrl)? onSaveImage =
        widget.onSaveImage;
    if (onSaveImage == null) return;

    final bool save = await showGfImageSaveMenu(
      context,
      saveImageLabel: widget.saveImageLabel,
      globalPosition: globalPosition,
    );

    if (!save || !mounted) return;
    await onSaveImage(widget.images[_currentIndex]);
  }

  Future<void> _shareCurrentImage() async {
    final Future<void> Function(String imageUrl)? onShareImage =
        widget.onShareImage;
    if (_isSharing || onShareImage == null) return;
    setState(() => _isSharing = true);
    try {
      await onShareImage(widget.images[_currentIndex]);
    } finally {
      if (mounted) setState(() => _isSharing = false);
    }
  }

  void _toggleChrome() => setState(() => _chromeVisible = !_chromeVisible);

  bool get _imageAtMinimumScale =>
      (_gestureKeys[_currentIndex]?.currentState?.gestureDetails?.totalScale ??
          1) <=
      1.01;

  void _interruptSlideReset() {
    final AnimationController? reset =
        _slidePageKey.currentState?.backAnimationController;
    if (reset?.isAnimating ?? false) {
      reset!.stop();
      _slideResetInterrupted = true;
    }
  }

  void _finishInterruptedSlideReset() {
    if (!_slideResetInterrupted) return;
    _slideResetInterrupted = false;
    _cancelSlide();
  }

  void _cancelSlide() {
    // extended_image evaluates slideEndHandler synchronously. A cancelled drag
    // must take its return-animation path even beyond the dismissal threshold.
    _cancelingSlide = true;
    try {
      _slidePageKey.currentState?.endSlide(ScaleEndDetails());
    } finally {
      _cancelingSlide = false;
    }
  }

  void _lockHorizontalSwipe() {
    final ExtendedImageSlidePageState? slide = _slidePageKey.currentState;
    if (_imageDragAxis == _ImageDragAxis.vertical || _slideResetInterrupted) {
      final Offset offset = slide?.offset ?? Offset.zero;
      if (offset != Offset.zero) slide?.slide(-offset);
    }
    _slideResetInterrupted = false;
    _imageDragAxis = _ImageDragAxis.horizontal;
  }

  bool _shouldAcceptPageDrag(Map<int, VelocityTracker> velocityTrackers) {
    if (!_trackDismissGesture ||
        _activePointers.length != 1 ||
        velocityTrackers.length != 1) {
      return false;
    }
    if (_imageDragAxis == _ImageDragAxis.horizontal) return true;

    final Offset movement = _lastSwipePosition! - _pointerStart!;
    if (movement.dx.abs() < _gestureTouchSlop ||
        movement.dx.abs() <=
            movement.dy.abs() * _horizontalTakeoverDominanceRatio) {
      return false;
    }
    if (_imageDragAxis == _ImageDragAxis.vertical) _lockHorizontalSwipe();
    return true;
  }

  void _settleFastHorizontalSwipe() {
    if (_imageDragAxis != _ImageDragAxis.horizontal ||
        !_pageController.hasClients) {
      return;
    }
    final double velocityX =
        _dismissVelocityTracker?.getVelocity().pixelsPerSecond.dx ?? 0;
    if (velocityX.abs() < 650) return;

    final double page = _pageController.page ?? _currentIndex.toDouble();
    final int direction = velocityX < 0 ? 1 : -1;
    final int releasePage = page.round();
    final int targetPage = releasePage == _pageAtPointerDown.round()
        ? releasePage + direction
        : releasePage;
    final int boundedTarget = targetPage.clamp(0, widget.images.length - 1);
    final Duration duration = GfMotion.duration(context, GfMotion.selection);

    // A quick new fling can interrupt PageView before it reaches its previous
    // snap target. Preserve one-page intent instead of restarting that target.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_pageController.hasClients) return;
      if (duration == Duration.zero) {
        _pageController.jumpToPage(boundedTarget);
      } else {
        _pageController.animateToPage(
          boundedTarget,
          duration: duration,
          curve: GfMotion.enterCurve,
        );
      }
    });
  }

  void _pointerDown(PointerDownEvent event) {
    _activePointers.add(event.pointer);
    if (_activePointers.length > 1) {
      _multiTouch = true;
      _trackDismissGesture = false;
      if (_imageDragAxis == _ImageDragAxis.vertical) {
        _cancelSlide();
      } else {
        _finishInterruptedSlideReset();
      }
      _imageDragAxis = _ImageDragAxis.undecided;
      _dismissVelocityTracker = null;
    } else {
      _interruptSlideReset();
      _pointerStart = event.position;
      _pointerStartTime = event.timeStamp;
      _gestureTouchSlop = computeHitSlop(
        event.kind,
        MediaQuery.gestureSettingsOf(context),
      );
      _pageAtPointerDown = _pageController.hasClients
          ? _pageController.page ?? _currentIndex.toDouble()
          : _currentIndex.toDouble();
      _pointerMoved = false;
      _lastSwipePosition = event.position;
      _imageDragAxis = _ImageDragAxis.undecided;
      _trackDismissGesture = _imageAtMinimumScale;
      _dismissVelocityTracker = _trackDismissGesture
          ? (VelocityTracker.withKind(event.kind)
              ..addPosition(event.timeStamp, event.position))
          : null;
    }
  }

  void _pointerMove(PointerMoveEvent event) {
    _dismissVelocityTracker?.addPosition(event.timeStamp, event.position);
    if (_pointerStart != null &&
        (event.position - _pointerStart!).distance > kTouchSlop) {
      _pointerMoved = true;
      _lastTapTime = null;
      _lastTapPosition = null;
    }

    if (!_trackDismissGesture || _activePointers.length != 1) return;
    final Offset delta = event.position - _pointerStart!;
    if (_imageDragAxis == _ImageDragAxis.undecided) {
      if (delta.distance < _gestureTouchSlop) return;

      if (delta.dx.abs() > delta.dy.abs()) {
        _lockHorizontalSwipe();
        return;
      }
      if (delta.dy.abs() < _gestureTouchSlop * _verticalDismissDominanceRatio ||
          delta.dy.abs() < delta.dx.abs() * _verticalDismissDominanceRatio) {
        return;
      }

      _imageDragAxis = _ImageDragAxis.vertical;
      _slideResetInterrupted = false;
      _slidePageKey.currentState?.slide(Offset(0, delta.dy));
    } else if (_imageDragAxis == _ImageDragAxis.vertical) {
      if (delta.dx.abs() >= _gestureTouchSlop &&
          delta.dx.abs() > delta.dy.abs() * _horizontalTakeoverDominanceRatio) {
        _lockHorizontalSwipe();
      } else {
        final double dy = event.position.dy - _lastSwipePosition!.dy;
        _slidePageKey.currentState?.slide(Offset(0, dy));
      }
    }
    _lastSwipePosition = event.position;
  }

  void _pointerUp(PointerUpEvent event) {
    _dismissVelocityTracker?.addPosition(event.timeStamp, event.position);
    if (_imageDragAxis == _ImageDragAxis.vertical) {
      _slidePageKey.currentState?.endSlide(
        ScaleEndDetails(
          velocity: _dismissVelocityTracker?.getVelocity() ?? Velocity.zero,
        ),
      );
    } else {
      _settleFastHorizontalSwipe();
    }
    _activePointers.remove(event.pointer);
    if (_activePointers.isNotEmpty) return;

    final bool tapped =
        !_multiTouch &&
        !_pointerMoved &&
        _pointerStart != null &&
        event.timeStamp - _pointerStartTime! <= kDoubleTapTimeout;
    if (tapped) {
      final bool doubleTap =
          _lastTapTime != null &&
          event.timeStamp - _lastTapTime! <= kDoubleTapTimeout &&
          (event.position - _lastTapPosition!).distance <= kDoubleTapSlop;
      _toggleChrome();
      _lastTapTime = doubleTap ? null : event.timeStamp;
      _lastTapPosition = doubleTap ? null : event.position;
    }

    _finishInterruptedSlideReset();

    _multiTouch = false;
    _pointerMoved = false;
    _pointerStart = null;
    _lastSwipePosition = null;
    _pointerStartTime = null;
    _dismissVelocityTracker = null;
    _trackDismissGesture = false;
    _imageDragAxis = _ImageDragAxis.undecided;
  }

  void _pointerCancel(PointerCancelEvent event) {
    if (_imageDragAxis == _ImageDragAxis.vertical) {
      _cancelSlide();
    }
    _activePointers.remove(event.pointer);
    if (_activePointers.isEmpty) {
      _finishInterruptedSlideReset();
      _multiTouch = false;
      _pointerMoved = false;
      _pointerStart = null;
      _lastSwipePosition = null;
      _pointerStartTime = null;
      _dismissVelocityTracker = null;
      _trackDismissGesture = false;
      _imageDragAxis = _ImageDragAxis.undecided;
      _lastTapTime = null;
      _lastTapPosition = null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool reduceMotion = MediaQuery.disableAnimationsOf(context);

    return Scaffold(
      backgroundColor: Colors.transparent,
      body: ExtendedImageSlidePage(
        key: _slidePageKey,
        slideAxis: SlideAxis.vertical,
        slideType: SlideType.wholePage,
        resetPageDuration: _initialSlideResetDuration!,
        slideEndHandler:
            (
              Offset offset, {
              ExtendedImageSlidePageState? state,
              ScaleEndDetails? details,
            }) {
              if (_cancelingSlide) return false;
              final double pageHeight =
                  state?.pageSize.height ?? MediaQuery.sizeOf(context).height;
              final double velocity =
                  details?.velocity.pixelsPerSecond.dy.abs() ?? 0;
              return offset.dy.abs() >= pageHeight / 10 || velocity >= 420;
            },
        slidePageBackgroundHandler: (Offset offset, Size size) =>
            defaultSlidePageBackgroundHandler(
              offset: offset,
              pageSize: size,
              color: const Color(0xEB000000),
              pageGestureAxis: SlideAxis.vertical,
            ),
        child: SafeArea(
          child: Stack(
            fit: StackFit.expand,
            children: <Widget>[
              Listener(
                key: const Key('gf-image-viewer-page-swipe-area'),
                behavior: HitTestBehavior.opaque,
                onPointerDown: _pointerDown,
                onPointerMove: _pointerMove,
                onPointerUp: _pointerUp,
                onPointerCancel: _pointerCancel,
                child: ExtendedImageGesturePageView.builder(
                  controller: _pageController,
                  itemCount: widget.images.length,
                  physics: const BouncingScrollPhysics(),
                  shouldAccpetHorizontalOrVerticalDrag: _shouldAcceptPageDrag,
                  canScrollPage: (GestureDetails? details) =>
                      (details?.totalScale ?? 1) <= 1.01,
                  onPageChanged: (int index) {
                    _doubleTapController.stop();
                    _doubleTapState = null;
                    _doubleTapPosition = null;
                    _gestureKeys[_currentIndex]?.currentState?.reset();
                    _gestureKeys[index]?.currentState?.reset();
                    setState(() {
                      _currentIndex = index;
                      _actualSize = false;
                    });
                    _centerThumbnail(index, animate: true);
                    widget.onPageChanged?.call(index);
                  },
                  itemBuilder: (BuildContext context, int index) {
                    Widget image = GestureDetector(
                      onLongPressStart: widget.onSaveImage == null
                          ? null
                          : (details) => _showImageActions(
                              context,
                              details.globalPosition,
                            ),
                      child: ExtendedImage(
                        image: GfMediaScope.imageProvider(
                          context,
                          widget.images[index],
                        ),
                        fit: _actualSize ? BoxFit.none : BoxFit.contain,
                        gaplessPlayback: false,
                        enableSlideOutPage: false,
                        mode: ExtendedImageMode.gesture,
                        extendedImageGestureKey: _gestureKeyFor(index),
                        initGestureConfigHandler: (ExtendedImageState state) =>
                            GestureConfig(
                              minScale: 1.0,
                              animationMinScale: 0.85,
                              maxScale: 4.0,
                              animationMaxScale: 4.5,
                              inertialSpeed: 500.0,
                              inPageView: widget.images.length > 1,
                              initialScale: 1.0,
                            ),
                        onDoubleTap: (ExtendedImageGestureState state) =>
                            _handleDoubleTap(state, index),
                        loadStateChanged: (ExtendedImageState state) {
                          switch (state.extendedImageLoadState) {
                            case LoadState.loading:
                              return const Center(child: GfProgressIndicator());
                            case LoadState.completed:
                              final image = state.extendedImageInfo?.image;
                              if (image != null) {
                                _imageSizes[index] = Size(
                                  image.width.toDouble(),
                                  image.height.toDouble(),
                                );
                              }
                              return null;
                            case LoadState.failed:
                              return Center(
                                child: GfSymbol(
                                  'image-off',
                                  color: colors.iconMuted,
                                  size: 48,
                                ),
                              );
                          }
                        },
                      ),
                    );
                    if (widget.heroTag != null) {
                      final hero = Hero(
                        tag: (widget.heroTag!, index),
                        child: image,
                      );
                      final animation = ModalRoute.of(context)!.animation!;
                      image = AnimatedBuilder(
                        animation: animation,
                        child: hero,
                        builder: (context, child) => HeroMode(
                          enabled:
                              !GfMotion.reducedOf(context) &&
                              (animation.status != AnimationStatus.reverse ||
                                  (widget.canReturnToSource?.call(index) ??
                                      true)),
                          child: child!,
                        ),
                      );
                    }
                    return image;
                  },
                ),
              ),
              Positioned(
                left: 12,
                top: 12,
                right: 12,
                child: IgnorePointer(
                  ignoring: !_chromeVisible,
                  child: ExcludeSemantics(
                    excluding: !_chromeVisible,
                    child: AnimatedOpacity(
                      key: const Key('gf-image-viewer-header-opacity'),
                      opacity: _chromeVisible ? 1 : 0,
                      duration: reduceMotion ? Duration.zero : GfMotion.content,
                      child: Row(
                        children: <Widget>[
                          Text(
                            '${_currentIndex + 1} / ${widget.images.length}',
                            style: TextStyle(
                              color: Colors.white.withValues(alpha: 0.82),
                              fontSize: 13,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          const Spacer(),
                          if (widget.enableActualSize)
                            GfGlassIconButton(
                              symbol: _actualSize ? 'minimize' : 'maximize',
                              tooltip: _actualSize
                                  ? 'Fit preview'
                                  : 'Original size',
                              onPressed: () => setState(() {
                                _actualSize = !_actualSize;
                              }),
                            ),
                          if (widget.onShareImage != null) ...<Widget>[
                            const SizedBox(width: 8),
                            GfGlassIconButton(
                              symbol: _isSharing ? 'clock' : 'share-2',
                              tooltip: widget.shareImageLabel,
                              onPressed: _isSharing ? null : _shareCurrentImage,
                            ),
                          ],
                          const SizedBox(width: 8),
                          GfGlassIconButton(
                            symbol: 'x',
                            tooltip: MaterialLocalizations.of(
                              context,
                            ).closeButtonTooltip,
                            onPressed: () => Navigator.of(context).maybePop(),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
              if (widget.images.length > 1)
                _buildThumbnailRail(context, colors, reduceMotion),
            ],
          ),
        ),
      ),
    );
  }
}

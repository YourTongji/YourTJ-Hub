import 'dart:math' as math;

import 'package:extended_image/extended_image.dart';
import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';

/// Shows the shared image action surface used by inline images and the
/// full-screen viewer. The host decides what saving means on each platform.
Future<bool> showGfImageSaveSheet(
  BuildContext context, {
  required String saveImageLabel,
}) async {
  final GfColors colors = GfTheme.colorsOf(context);
  final GfBorders borders = GfTheme.bordersOf(context);
  return await showModalBottomSheet<bool>(
        context: context,
        // Keep the action sheet above the persistent mobile shell as well as
        // the viewer's branch Navigator.
        useRootNavigator: true,
        showDragHandle: true,
        backgroundColor: colors.base100,
        shape: RoundedRectangleBorder(
          borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
          side: BorderSide(color: colors.line, width: borders.width),
        ),
        builder: (BuildContext sheetContext) => SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
            child: ListTile(
              minVerticalPadding: 14,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
              ),
              leading: Icon(Icons.save_alt_rounded, color: colors.primary),
              title: Text(saveImageLabel),
              onTap: () => Navigator.of(sheetContext).pop(true),
            ),
          ),
        ),
      ) ??
      false;
}

/// Full-screen image viewer mirroring web `MarkdownImageViewer.vue`:
/// dark backdrop, swipe to switch images, pinch to zoom, tap to toggle
/// the viewer chrome, and prev/next controls for multi-image sets. The top
/// zoom control toggles actual-size rendering.
class GfImageViewer extends StatefulWidget {
  const GfImageViewer({
    super.key,
    required this.images,
    this.initialIndex = 0,
    this.enableActualSize = true,
    this.onSaveImage,
    this.saveImageLabel = 'Save image',
    this.onShareImage,
    this.shareImageLabel = 'Share image',
  });

  /// Image URLs to display.
  final List<String> images;

  final int initialIndex;

  /// Whether the actual-size toggle is offered.
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

class _GfImageViewerState extends State<GfImageViewer>
    with SingleTickerProviderStateMixin {
  late final PageController _pageController;
  late final AnimationController _doubleTapController;
  late int _currentIndex;
  bool _actualSize = false;
  bool _isSharing = false;
  ExtendedImageGestureState? _doubleTapState;
  Offset? _doubleTapPosition;
  double _doubleTapStartScale = 1.0;
  double _doubleTapTargetScale = 1.0;
  final Map<int, Size> _imageSizes = <int, Size>{};

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex.clamp(0, widget.images.length - 1);
    _pageController = PageController(initialPage: _currentIndex);
    _doubleTapController =
        AnimationController(
            vsync: this,
            duration: const Duration(milliseconds: 260),
          )
          ..addListener(_applyDoubleTapScale)
          ..addStatusListener(_finishDoubleTapScale);
  }

  @override
  void dispose() {
    _doubleTapController.dispose();
    _pageController.dispose();
    super.dispose();
  }

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

    final double progress = Curves.easeOutCubic.transform(
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
    final bool zoomed = currentScale > config.minScale + 0.1;
    final double targetScale = zoomed
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
    _doubleTapController.forward(from: 0);
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

  void _showPrevious() {
    if (widget.images.length < 2) return;
    final int next = _currentIndex <= 0
        ? widget.images.length - 1
        : _currentIndex - 1;
    _pageController.animateToPage(
      next,
      duration: GfMotion.standardDuration,
      curve: GfMotion.standardEase,
    );
  }

  void _showNext() {
    if (widget.images.length < 2) return;
    final int next = _currentIndex >= widget.images.length - 1
        ? 0
        : _currentIndex + 1;
    _pageController.animateToPage(
      next,
      duration: GfMotion.standardDuration,
      curve: GfMotion.standardEase,
    );
  }

  Future<void> _showImageActions(BuildContext context) async {
    final Future<void> Function(String imageUrl)? onSaveImage =
        widget.onSaveImage;
    if (onSaveImage == null) return;

    final bool save = await showGfImageSaveSheet(
      context,
      saveImageLabel: widget.saveImageLabel,
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

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Scaffold(
      backgroundColor: const Color(0xFF000000).withValues(alpha: 0.62),
      body: SafeArea(
        child: Stack(
          children: <Widget>[
            Positioned.fill(
              child: PageView.builder(
                controller: _pageController,
                itemCount: widget.images.length,
                onPageChanged: (int index) {
                  _doubleTapController.stop();
                  _doubleTapState = null;
                  _doubleTapPosition = null;
                  setState(() {
                    _currentIndex = index;
                    _actualSize = false;
                  });
                },
                itemBuilder: (BuildContext context, int index) {
                  return GestureDetector(
                    onLongPress: widget.onSaveImage == null
                        ? null
                        : () => _showImageActions(context),
                    child: ExtendedImage.network(
                      widget.images[index],
                      fit: _actualSize ? BoxFit.none : BoxFit.contain,
                      gaplessPlayback: true,
                      mode: ExtendedImageMode.gesture,
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
                            return const Center(
                              child: CircularProgressIndicator(),
                            );
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
                              child: Icon(
                                Icons.broken_image_outlined,
                                color: colors.iconMuted,
                                size: 48,
                              ),
                            );
                        }
                      },
                    ),
                  );
                },
              ),
            ),
            // Counter badge ("1 / 3") when multiple images.
            if (widget.images.length > 1)
              Positioned(
                left: 12,
                top: 12,
                child: _ViewerBadge(
                  child: Text(
                    '${_currentIndex + 1} / ${widget.images.length}',
                    style: TextStyle(
                      color: colors.baseContent.withValues(alpha: 0.72),
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ),
            // Top-right controls.
            Positioned(
              right: 12,
              top: 12,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  if (widget.enableActualSize)
                    _ViewerIconButton(
                      icon: _actualSize
                          ? Icons.zoom_out_map
                          : Icons.zoom_in_map,
                      tooltip: _actualSize ? 'Fit preview' : 'Original size',
                      onPressed: () => setState(() {
                        _actualSize = !_actualSize;
                      }),
                    ),
                  if (widget.onShareImage != null) ...[
                    const SizedBox(width: 8),
                    _ViewerIconButton(
                      icon: _isSharing
                          ? Icons.hourglass_top_rounded
                          : Icons.share_outlined,
                      tooltip: widget.shareImageLabel,
                      onPressed: _shareCurrentImage,
                    ),
                  ],
                  const SizedBox(width: 8),
                  _ViewerIconButton(
                    icon: Icons.close,
                    tooltip: 'Close',
                    onPressed: () => Navigator.of(context).maybePop(),
                  ),
                ],
              ),
            ),
            // Side navigation.
            if (widget.images.length > 1)
              Positioned(
                left: 8,
                top: 0,
                bottom: 0,
                child: Center(
                  child: _ViewerIconButton(
                    icon: Icons.chevron_left,
                    tooltip: 'Previous',
                    onPressed: _showPrevious,
                  ),
                ),
              ),
            if (widget.images.length > 1)
              Positioned(
                right: 8,
                top: 0,
                bottom: 0,
                child: Center(
                  child: _ViewerIconButton(
                    icon: Icons.chevron_right,
                    tooltip: 'Next',
                    onPressed: _showNext,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _ViewerBadge extends StatelessWidget {
  const _ViewerBadge({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: colors.base100.withValues(alpha: 0.82),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(
          color: colors.line.withValues(alpha: 0.7),
          width: borders.width,
        ),
      ),
      child: child,
    );
  }
}

class _ViewerIconButton extends StatelessWidget {
  const _ViewerIconButton({
    required this.icon,
    required this.tooltip,
    required this.onPressed,
  });

  final IconData icon;
  final String tooltip;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    return Tooltip(
      message: tooltip,
      child: Material(
        color: colors.base100.withValues(alpha: 0.86),
        shape: const CircleBorder(),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onPressed,
          customBorder: const CircleBorder(),
          child: Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              border: Border.all(
                color: colors.line.withValues(alpha: 0.76),
                width: borders.width,
              ),
            ),
            child: Icon(
              icon,
              size: 20,
              color: colors.baseContent.withValues(alpha: 0.78),
            ),
          ),
        ),
      ),
    );
  }
}

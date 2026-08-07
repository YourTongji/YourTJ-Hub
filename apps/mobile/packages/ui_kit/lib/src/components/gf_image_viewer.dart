import 'package:extended_image/extended_image.dart';
import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';

/// Full-screen image viewer mirroring web `MarkdownImageViewer.vue`:
/// dark backdrop, swipe to switch images, pinch to zoom, tap to toggle
/// actual size, and prev/next controls for multi-image sets.
class GfImageViewer extends StatefulWidget {
  const GfImageViewer({
    super.key,
    required this.images,
    this.initialIndex = 0,
    this.enableActualSize = true,
  });

  /// Image URLs to display.
  final List<String> images;

  final int initialIndex;

  /// Whether the actual-size toggle is offered.
  final bool enableActualSize;

  @override
  State<GfImageViewer> createState() => _GfImageViewerState();
}

class _GfImageViewerState extends State<GfImageViewer> {
  late final PageController _pageController;
  late int _currentIndex;
  bool _actualSize = false;

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex.clamp(0, widget.images.length - 1);
    _pageController = PageController(initialPage: _currentIndex);
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  void _showPrevious() {
    if (widget.images.length < 2) return;
    final int next =
        _currentIndex <= 0 ? widget.images.length - 1 : _currentIndex - 1;
    _pageController.animateToPage(
      next,
      duration: GfMotion.standardDuration,
      curve: GfMotion.standard,
    );
  }

  void _showNext() {
    if (widget.images.length < 2) return;
    final int next =
        _currentIndex >= widget.images.length - 1 ? 0 : _currentIndex + 1;
    _pageController.animateToPage(
      next,
      duration: GfMotion.standardDuration,
      curve: GfMotion.standard,
    );
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Scaffold(
      backgroundColor: Colors.black.withValues(alpha: 0.62),
      body: SafeArea(
        child: Stack(
          children: <Widget>[
            Positioned.fill(
              child: PageView.builder(
                controller: _pageController,
                itemCount: widget.images.length,
                onPageChanged: (int index) {
                  setState(() {
                    _currentIndex = index;
                    _actualSize = false;
                  });
                },
                itemBuilder: (BuildContext context, int index) {
                  return GestureDetector(
                    onTap: () => setState(() {
                      _actualSize = !_actualSize;
                    }),
                    child: ExtendedImage.network(
                      widget.images[index],
                      fit: _actualSize ? BoxFit.none : BoxFit.contain,
                      gaplessPlayback: true,
                      mode: ExtendedImageMode.gesture,
                      initGestureConfigHandler: (ExtendedImageState state) =>
                          GestureConfig(
                        minScale: 1.0,
                        maxScale: 4.0,
                        animationMaxScale: 4.0,
                        inPageView: widget.images.length > 1,
                        initialScale: 1.0,
                      ),
                      loadStateChanged: (ExtendedImageState state) {
                        switch (state.extendedImageLoadState) {
                          case LoadState.loading:
                            return const Center(
                              child: CircularProgressIndicator(),
                            );
                          case LoadState.completed:
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
            child: Icon(icon, size: 20, color: colors.baseContent.withValues(alpha: 0.78)),
          ),
        ),
      ),
    );
  }
}

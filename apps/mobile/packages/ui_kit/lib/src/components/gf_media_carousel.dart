import 'dart:ui' show ImageFilter;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_image_viewer.dart';
import 'gf_motion.dart';
import 'gf_symbol.dart';

/// Uncropped content gallery, shared by the publishing preview and topic body.
class GfMediaCarousel extends StatefulWidget {
  const GfMediaCarousel({
    super.key,
    required this.images,
    this.onSaveImage,
    this.saveImageLabel = 'Save image',
    this.onShareImage,
    this.shareImageLabel = 'Share image',
  });

  final List<String> images;
  final Future<void> Function(String imageUrl)? onSaveImage;
  final String saveImageLabel;
  final Future<void> Function(String imageUrl)? onShareImage;
  final String shareImageLabel;

  @override
  State<GfMediaCarousel> createState() => _GfMediaCarouselState();
}

class _GfMediaCarouselState extends State<GfMediaCarousel> {
  int _index = 0;
  final Object _heroTag = Object();

  @override
  void didUpdateWidget(GfMediaCarousel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!listEquals(oldWidget.images, widget.images)) _index = 0;
  }

  @override
  Widget build(BuildContext context) {
    if (widget.images.isEmpty) return const SizedBox.shrink();
    final GfColors colors = GfTheme.colorsOf(context);
    final double radius = GfTheme.radiiOf(context).box;
    final bool reduceMotion = MediaQuery.disableAnimationsOf(context);

    return LayoutBuilder(
      builder: (context, constraints) {
        final double height = constraints.maxWidth.clamp(200.0, 420.0);
        final double pixelRatio = MediaQuery.devicePixelRatioOf(context);

        return ClipRRect(
          borderRadius: BorderRadius.circular(radius),
          child: SizedBox(
            height: height,
            child: Stack(
              fit: StackFit.expand,
              children: <Widget>[
                PageView.builder(
                  key: ValueKey(Object.hashAll(widget.images)),
                  itemCount: widget.images.length,
                  onPageChanged: (index) => setState(() => _index = index),
                  itemBuilder: (context, index) {
                    final ResizeImage preview = ResizeImage(
                      NetworkImage(widget.images[index]),
                      policy: ResizeImagePolicy.fit,
                      width: (constraints.maxWidth * pixelRatio).round(),
                      height: (height * pixelRatio).round(),
                    );
                    return Stack(
                      fit: StackFit.expand,
                      children: <Widget>[
                        ImageFiltered(
                          imageFilter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
                          child: Image(
                            image: preview,
                            fit: BoxFit.cover,
                            gaplessPlayback: true,
                            errorBuilder: (_, _, _) => const SizedBox.shrink(),
                          ),
                        ),
                        ColoredBox(
                          color: colors.base100.withValues(alpha: 0.46),
                        ),
                        Semantics(
                          button: true,
                          label: '${index + 1} / ${widget.images.length}',
                          child: GestureDetector(
                            behavior: HitTestBehavior.opaque,
                            onTap: () =>
                                Navigator.of(
                                  context,
                                  rootNavigator: true,
                                ).push<void>(
                                  PageRouteBuilder<void>(
                                    opaque: false,
                                    barrierColor: Colors.transparent,
                                    transitionDuration: reduceMotion
                                        ? Duration.zero
                                        : GfMotion.overlay,
                                    reverseTransitionDuration: reduceMotion
                                        ? Duration.zero
                                        : GfMotion.overlay,
                                    pageBuilder: (_, _, _) => GfImageViewer(
                                      images: widget.images,
                                      initialIndex: index,
                                      heroTag: _heroTag,
                                      onSaveImage: widget.onSaveImage,
                                      saveImageLabel: widget.saveImageLabel,
                                      onShareImage: widget.onShareImage,
                                      shareImageLabel: widget.shareImageLabel,
                                    ),
                                    transitionsBuilder:
                                        (_, animation, _, child) =>
                                            GfFadeTransition(
                                              animation: animation,
                                              offset: Offset.zero,
                                              child: child,
                                            ),
                                  ),
                                ),
                            onLongPress: widget.onSaveImage == null
                                ? null
                                : () async {
                                    final bool save =
                                        await showGfImageSaveSheet(
                                          context,
                                          saveImageLabel: widget.saveImageLabel,
                                        );
                                    if (save && context.mounted) {
                                      await widget.onSaveImage!(
                                        widget.images[index],
                                      );
                                    }
                                  },
                            child: Hero(
                              tag: (_heroTag, index),
                              child: Image(
                                image: preview,
                                fit: BoxFit.contain,
                                gaplessPlayback: true,
                                errorBuilder: (_, _, _) => GfSymbol(
                                  'image-off',
                                  color: colors.iconMuted,
                                ),
                              ),
                            ),
                          ),
                        ),
                      ],
                    );
                  },
                ),
                if (widget.images.length > 1)
                  Positioned(
                    top: 10,
                    right: 10,
                    child: IgnorePointer(
                      child: ExcludeSemantics(
                        child: ClipRRect(
                          key: const Key('gf-media-carousel-counter-badge'),
                          borderRadius: BorderRadius.circular(999),
                          child: BackdropFilter(
                            filter: ImageFilter.blur(sigmaX: 8, sigmaY: 8),
                            child: DecoratedBox(
                              decoration: BoxDecoration(
                                color: Colors.black.withValues(alpha: 0.48),
                                borderRadius: BorderRadius.circular(999),
                                border: Border.all(
                                  color: Colors.white.withValues(alpha: 0.18),
                                ),
                              ),
                              child: SizedBox(
                                height: 22,
                                child: Center(
                                  widthFactor: 1,
                                  child: Padding(
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 8,
                                    ),
                                    child: Text(
                                      '${_index + 1} / ${widget.images.length}',
                                      style: const TextStyle(
                                        color: Colors.white,
                                        fontSize: 10,
                                        fontWeight: FontWeight.w600,
                                        height: 1,
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
                if (widget.images.length > 1)
                  Positioned(
                    left: 0,
                    right: 0,
                    bottom: 12,
                    child: IgnorePointer(
                      child: ExcludeSemantics(
                        child: Center(
                          child: DecoratedBox(
                            decoration: BoxDecoration(
                              color: Colors.black.withValues(alpha: 0.3),
                              borderRadius: BorderRadius.circular(999),
                            ),
                            child: Padding(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 9,
                                vertical: 7,
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: List<Widget>.generate(
                                  widget.images.length,
                                  (pill) => AnimatedContainer(
                                    duration: reduceMotion
                                        ? Duration.zero
                                        : GfMotion.selection,
                                    width: pill == _index ? 16 : 4,
                                    height: 4,
                                    margin: EdgeInsets.only(
                                      right: pill == widget.images.length - 1
                                          ? 0
                                          : 4,
                                    ),
                                    decoration: BoxDecoration(
                                      color: Colors.white.withValues(
                                        alpha: pill == _index ? 1 : 0.5,
                                      ),
                                      borderRadius: BorderRadius.circular(999),
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
              ],
            ),
          ),
        );
      },
    );
  }
}

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import '../theme/gf_theme.dart';
import 'gf_image_viewer.dart';

/// Uncropped content gallery, shared by the publishing preview and topic body.
class GfMediaCarousel extends StatefulWidget {
  const GfMediaCarousel({super.key, required this.images});
  final List<String> images;
  @override
  State<GfMediaCarousel> createState() => _GfMediaCarouselState();
}

class _GfMediaCarouselState extends State<GfMediaCarousel> {
  int _index = 0;
  @override
  void didUpdateWidget(GfMediaCarousel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!listEquals(oldWidget.images, widget.images)) _index = 0;
  }

  @override
  Widget build(BuildContext context) {
    if (widget.images.isEmpty) return const SizedBox.shrink();
    final colors = GfTheme.colorsOf(context);
    return LayoutBuilder(
      builder: (context, constraints) => Column(
        children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
            child: ColoredBox(
              color: colors.base200,
              child: SizedBox(
                height: constraints.maxWidth.clamp(200.0, 420.0),
                child: PageView.builder(
                  key: ValueKey(Object.hashAll(widget.images)),
                  itemCount: widget.images.length,
                  onPageChanged: (index) => setState(() => _index = index),
                  itemBuilder: (context, index) => Semantics(
                    button: true,
                    label: '${index + 1} / ${widget.images.length}',
                    child: GestureDetector(
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute<void>(
                          builder: (_) => GfImageViewer(
                            images: widget.images,
                            initialIndex: index,
                          ),
                        ),
                      ),
                      child: Image.network(
                        widget.images[index],
                        fit: BoxFit.contain,
                        errorBuilder: (_, _, _) => Icon(
                          Icons.broken_image_outlined,
                          color: colors.iconMuted,
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
          if (widget.images.length > 1)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 10),
              child: Text(
                '${(_index + 1).clamp(1, widget.images.length)} / ${widget.images.length}',
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: colors.iconMuted),
              ),
            ),
        ],
      ),
    );
  }
}

import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../asset_url.dart';
import 'sticker_library_state.dart';
import 'sticker_strings.dart';
import 'sticker_preview.dart';
import '../share/share_image_readiness.dart';

/// Shared expression renderer. Taps preview this sticker alone; holding exposes
/// collection unless the surrounding message owns the long-press menu.
class StickerImage extends StatefulWidget {
  /// Reading surfaces need enough room for captions in animated stickers.
  /// Pickers and bounded draft previews retain the smaller thumbnail size.
  static const double readingSize = 128;
  static const double thumbnailSize = 56;

  const StickerImage({
    super.key,
    required this.name,
    required this.url,
    this.label,
    this.size = thumbnailSize,
    this.collectible = true,
    this.deferLongPress = false,
    this.excludeSemantics = false,
  });
  final String name;
  final String url;
  final String? label;
  final double size;
  final bool collectible;

  /// Embedded copies (picker grids, library rows) sit inside an outer labeled
  /// control behind an [IgnorePointer]: the outer label owns accessibility, so
  /// this inner button node is excluded to avoid announcing two nested buttons.
  final bool excludeSemantics;

  /// A surrounding surface (a chat message bubble) owns the long press for its
  /// action menu, so this sticker must not consume it — not even for its own
  /// retry/collection actions, and not through the unavailable-state tooltip.
  final bool deferLongPress;
  @override
  State<StickerImage> createState() => _StickerImageState();
}

class _StickerImageState extends State<StickerImage> {
  int _revision = 0;
  bool _failed = false;

  Future<void> _actions() async {
    final strings = StickerStrings(context);
    final owner = widget.collectible
        ? ProviderScope.containerOf(
            context,
            listen: false,
          ).read(stickerCollectionProvider)
        : null;
    final action = await showGfBottomSheet<String>(
      context,
      builder: (context) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (widget.collectible)
            ListTile(
              leading: const GfSymbol('bookmark', size: 22),
              title: Text(strings.collect),
              onTap: () => Navigator.pop(context, 'save'),
            ),
          if (_failed)
            ListTile(
              leading: const GfSymbol('refresh-cw', size: 22),
              title: Text(strings.retry),
              onTap: () => Navigator.pop(context, 'retry'),
            ),
        ],
      ),
    );
    if (!mounted) return;
    if (action == 'retry') {
      await GfMediaScope.imageProvider(
        context,
        resolveApiAssetUrl(widget.url),
      ).evict();
      if (mounted) {
        setState(() {
          _revision++;
          _failed = false;
        });
      }
    } else if (action == 'save') {
      final collection = owner!;
      if (!collection.active) return;
      try {
        await collection.save(stickerName: widget.name);
        if (mounted && collection.active) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(strings.saved)));
        }
      } catch (error) {
        if (mounted && collection.active) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(strings.failure(error))));
        }
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final readiness = ShareImageReadiness.maybeOf(context);
    final readinessKey = resolveApiAssetUrl(widget.url);
    readiness?.begin(readinessKey);
    final bool showActions =
        !widget.deferLongPress && (widget.collectible || _failed);
    var hasFrame = false;
    final semantics = Semantics(
      label:
          widget.label ??
          (RegExp(r'^u_[0-9a-f]{48}$').hasMatch(widget.name)
              ? StickerStrings(context).custom
              : widget.name),
      button: true,
      hint: StickerStrings(context).viewLarger,
      onLongPress: showActions ? _actions : null,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: () => showStickerPreview(context, widget.url),
        onLongPress: showActions ? _actions : null,
        child: SizedBox.square(
          dimension: widget.size,
          child: GfNetworkImage(
            readinessKey,
            key: ValueKey(_revision),
            fit: BoxFit.contain,
            cacheWidth: readiness == null ? null : (widget.size * 2).round(),
            excludeFromSemantics: true,
            frameBuilder: (_, child, frame, _) {
              hasFrame = frame != null;
              if (frame != null) readiness?.finish(readinessKey);
              return child;
            },
            loadingBuilder: (context, child, progress) {
              if (readiness?.isFrozen(readinessKey) == true) {
                return const SizedBox(
                  height: 80,
                  child: Center(child: GfSymbol('image-off')),
                );
              }
              return progress == null && hasFrame
                  ? child
                  : Center(
                      child: SizedBox.square(
                        dimension: 16,
                        child: GfProgressIndicator(
                          strokeWidth: 1.5,
                          value: progress?.expectedTotalBytes == null
                              ? null
                              : progress!.cumulativeBytesLoaded /
                                    progress.expectedTotalBytes!,
                        ),
                      ),
                    );
            },
            errorBuilder: (context, _, _) {
              readiness?.finish(readinessKey);
              _failed = true;
              return Tooltip(
                message: StickerStrings(context).unavailable,
                // The default long-press trigger would out-prioritize the
                // bubble's action menu on a failed sticker; the message stays
                // available to screen readers either way.
                triggerMode: widget.deferLongPress
                    ? TooltipTriggerMode.manual
                    : null,
                child: GfSymbol(
                  'image-off',
                  size: 22,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              );
            },
          ),
        ),
      ),
    );
    if (!widget.excludeSemantics) return semantics;
    return ExcludeSemantics(child: semantics);
  }
}

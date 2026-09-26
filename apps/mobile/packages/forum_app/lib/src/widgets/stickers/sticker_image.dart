import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../asset_url.dart';
import 'sticker_library_state.dart';
import 'sticker_strings.dart';

/// Shared expression renderer. A sticker consumes taps without opening a
/// gallery or activating the surrounding post; holding exposes collection.
class StickerImage extends StatefulWidget {
  const StickerImage({
    super.key,
    required this.name,
    required this.url,
    this.label,
    this.size = 56,
    this.collectible = true,
  });
  final String name;
  final String url;
  final String? label;
  final double size;
  final bool collectible;
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
      await NetworkImage(resolveApiAssetUrl(widget.url)).evict();
      if (mounted) {
        setState(() {
          _revision++;
          _failed = false;
        });
      }
    } else if (action == 'save') {
      final collection = owner!;
      if (!collection.active || collection.busy) return;
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
  Widget build(BuildContext context) => Semantics(
    label:
        widget.label ??
        (RegExp(r'^u_[0-9a-f]{48}$').hasMatch(widget.name)
            ? StickerStrings(context).custom
            : widget.name),
    onLongPress: widget.collectible || _failed ? _actions : null,
    child: GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: () {},
      onLongPress: widget.collectible || _failed ? _actions : null,
      child: SizedBox.square(
        dimension: widget.size,
        child: Image.network(
          resolveApiAssetUrl(widget.url),
          key: ValueKey(_revision),
          fit: BoxFit.contain,
          excludeFromSemantics: true,
          loadingBuilder: (context, child, progress) => progress == null
              ? child
              : Center(
                  child: SizedBox.square(
                    dimension: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 1.5,
                      value: progress.expectedTotalBytes == null
                          ? null
                          : progress.cumulativeBytesLoaded /
                                progress.expectedTotalBytes!,
                    ),
                  ),
                ),
          errorBuilder: (context, _, _) {
            _failed = true;
            return Tooltip(
              message: StickerStrings(context).unavailable,
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
}

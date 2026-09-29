import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../asset_url.dart';
import 'sticker_library_state.dart';

/// A sticker opens alone: it never joins the surrounding attachment gallery.
Future<void> showStickerPreview(
  BuildContext context,
  String url, {
  StickerCollection? collection,
}) => Navigator.of(context, rootNavigator: true).push<void>(
  MaterialPageRoute<void>(
    builder: (_) {
      final viewer = Scaffold(
        backgroundColor: Colors.black,
        body: SafeArea(child: GfImageViewer(images: [resolveApiAssetUrl(url)])),
      );
      return collection == null
          ? viewer
          : StickerSessionSurface(collection: collection, child: viewer);
    },
  ),
);

/// Remove private library overlays with the account that opened them. The
/// callback also checks the captured collection before any server mutation.
class StickerSessionSurface extends ConsumerStatefulWidget {
  const StickerSessionSurface({
    super.key,
    required this.collection,
    required this.child,
  });
  final StickerCollection? collection;
  final Widget child;

  @override
  ConsumerState<StickerSessionSurface> createState() =>
      _StickerSessionSurfaceState();
}

class _StickerSessionSurfaceState extends ConsumerState<StickerSessionSurface> {
  bool _closing = false;

  @override
  Widget build(BuildContext context) {
    final owner = widget.collection;
    if (owner != null &&
        (!identical(ref.watch(stickerCollectionProvider), owner) ||
            !owner.active)) {
      if (!_closing) {
        _closing = true;
        final route = ModalRoute.of(context);
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (route?.isActive == true) route!.navigator?.removeRoute(route);
        });
      }
      return const SizedBox.shrink();
    }
    return widget.child;
  }
}

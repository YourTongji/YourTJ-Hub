import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

/// Tallest an embedded image may render inside an editor body: this fraction
/// of the viewport height. The reader caps at `min(360px, 70vh)`; the editor
/// keeps the same intent so a portrait photo cannot blow up the block height
/// and push the caret far out of reach (issue #1122).
const double kEditorEmbedImageMaxHeightFraction = 0.6;

/// Sizing for a body image embed (Markdown image / composer photo) shared by
/// every editor host.
///
/// Quill's inline embed layout bounds the width but leaves the height
/// unbounded, so an image otherwise renders at its full intrinsic height
/// (e.g. a 3:5 phone photo at full editor width). Images fill the editor width
/// when wider than it and shrink proportionally under the height cap; both
/// composer and editor render through this widget so a block keeps the same
/// height everywhere it appears.
class EditorEmbedImage extends StatelessWidget {
  const EditorEmbedImage({super.key, required this.url});

  final String url;

  @override
  Widget build(BuildContext context) {
    final MediaQueryData media = MediaQuery.of(context);
    return ConstrainedBox(
      constraints: BoxConstraints(
        maxHeight: media.size.height * kEditorEmbedImageMaxHeightFraction,
      ),
      // Centers a shrunk image (and small originals) while keeping the block
      // height tight to the image; the web prose centers images the same way.
      child: Align(
        alignment: Alignment.center,
        heightFactor: 1,
        child: GfNetworkImage(
          url,
          fit: BoxFit.contain,
          cacheWidth: (media.size.width * media.devicePixelRatio).round(),
          errorBuilder: (_, _, _) => const GfSymbol('image-off'),
        ),
      ),
    );
  }
}

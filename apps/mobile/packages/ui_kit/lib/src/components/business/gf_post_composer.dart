import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_button.dart';
import '../surfaces/gf_floating_surface.dart';

/// Floating reply composer mirroring web `PostComposer.vue` mobile form: a
/// `gf-floating-surface` panel (`w-[min(42rem,100vw-1.5rem)]`) with an
/// optional reply-target bar, the markdown editor, an image button and a
/// publish button. Callers place it in a [Stack]/[Overlay] at bottom-4.
class GfPostComposer extends StatelessWidget {
  const GfPostComposer({
    super.key,
    required this.controller,
    required this.onPublish,
    required this.publishLabel,
    required this.hintText,
    this.focusNode,
    this.targetName,
    this.targetLabel,
    this.onCloseTarget,
    this.onPickImage,
    this.imageTooltip,
    this.imageUrl,
    this.onRemoveImage,
    this.removeImageTooltip,
    this.uploading = false,
    this.publishing = false,
    this.canPublish = true,
    this.toolbar,
  }) : assert(onPickImage == null || imageTooltip != null),
       assert(onRemoveImage == null || removeImageTooltip != null);

  final TextEditingController controller;
  final FocusNode? focusNode;
  final VoidCallback onPublish;

  /// When replying to a user, the target name and localized label shown in the
  /// reference bar.
  final String? targetName;
  final String? targetLabel;
  final VoidCallback? onCloseTarget;

  /// Image upload action, localized accessibility labels, and optional preview.
  final VoidCallback? onPickImage;
  final String? imageTooltip;
  final String? imageUrl;
  final VoidCallback? onRemoveImage;
  final String? removeImageTooltip;
  final bool uploading;

  final bool publishing;
  final bool canPublish;
  final String publishLabel;
  final String hintText;

  /// Optional extra toolbar widgets (bold/italic/link/quote/code…).
  final Widget? toolbar;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    return GfFloatingSurface(
      padding: const EdgeInsets.all(12),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          if (targetName != null)
            Container(
              margin: const EdgeInsets.only(bottom: 8),
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
              decoration: BoxDecoration(
                color: colors.base200,
                borderRadius: BorderRadius.circular(radii.field),
              ),
              child: Row(
                children: <Widget>[
                  Icon(
                    Icons.reply,
                    size: 14,
                    color: colors.baseContent.withValues(alpha: 0.55),
                  ),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      targetLabel ?? targetName!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 13,
                        color: colors.baseContent.withValues(alpha: 0.75),
                      ),
                    ),
                  ),
                  if (onCloseTarget != null)
                    InkWell(
                      onTap: onCloseTarget,
                      child: Icon(
                        Icons.close,
                        size: 16,
                        color: colors.iconMuted,
                      ),
                    ),
                ],
              ),
            ),
          if (toolbar != null || onPickImage != null) ...<Widget>[
            Row(
              children: <Widget>[
                if (onPickImage != null)
                  IconButton(
                    icon: uploading
                        ? const SizedBox.square(
                            dimension: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Icon(
                            Icons.image_outlined,
                            size: 21,
                            color: colors.iconMuted,
                          ),
                    onPressed: uploading ? null : onPickImage,
                    tooltip: imageTooltip,
                    visualDensity: VisualDensity.compact,
                  ),
                if (toolbar != null) Expanded(child: toolbar!),
              ],
            ),
            const SizedBox(height: 6),
          ],
          if (imageUrl != null && imageUrl!.isNotEmpty) ...<Widget>[
            Align(
              alignment: Alignment.centerLeft,
              child: Stack(
                clipBehavior: Clip.none,
                children: <Widget>[
                  ClipRRect(
                    borderRadius: BorderRadius.circular(radii.field),
                    child: Image.network(
                      imageUrl!,
                      width: 88,
                      height: 72,
                      fit: BoxFit.cover,
                      errorBuilder: (_, _, _) => Container(
                        width: 88,
                        height: 72,
                        color: colors.base300,
                        alignment: Alignment.center,
                        child: Icon(
                          Icons.broken_image_outlined,
                          color: colors.iconMuted,
                        ),
                      ),
                    ),
                  ),
                  if (onRemoveImage != null)
                    Positioned(
                      right: -10,
                      top: -10,
                      child: IconButton.filled(
                        onPressed: onRemoveImage,
                        tooltip: removeImageTooltip,
                        icon: const Icon(Icons.close, size: 15),
                        style: IconButton.styleFrom(
                          minimumSize: const Size(32, 32),
                          backgroundColor: colors.baseContent,
                          foregroundColor: colors.base100,
                        ),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 8),
          ],
          Container(
            constraints: const BoxConstraints(minHeight: 76, maxHeight: 160),
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: colors.base100,
              borderRadius: BorderRadius.circular(radii.field),
              border: Border.all(color: colors.line, width: 1),
            ),
            child: TextField(
              controller: controller,
              focusNode: focusNode,
              maxLines: null,
              minLines: 2,
              expands: false,
              style: const TextStyle(fontSize: 16, height: 1.5),
              decoration: InputDecoration(
                hintText: hintText,
                border: InputBorder.none,
                isDense: true,
              ),
            ),
          ),
          const SizedBox(height: 10),
          Align(
            alignment: Alignment.centerRight,
            child: GfButton(
              label: publishLabel,
              size: GfButtonSize.medium,
              loading: publishing,
              onPressed: publishing || !canPublish ? null : onPublish,
            ),
          ),
        ],
      ),
    );
  }
}

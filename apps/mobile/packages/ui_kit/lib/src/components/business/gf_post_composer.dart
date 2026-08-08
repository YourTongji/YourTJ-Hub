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
    this.targetName,
    this.onCloseTarget,
    this.onPickImage,
    this.publishing = false,
    this.publishLabel = '发布',
    this.hintText = '写下你的回复…',
    this.toolbar,
  });

  final TextEditingController controller;
  final VoidCallback onPublish;

  /// When replying to a user, the target name shown in the reference bar.
  final String? targetName;
  final VoidCallback? onCloseTarget;

  /// Image upload button (web toolbar image button).
  final VoidCallback? onPickImage;

  final bool publishing;
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
                      '回复 $targetName',
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
          if (toolbar != null) ...<Widget>[toolbar!, const SizedBox(height: 8)],
          Container(
            constraints: const BoxConstraints(minHeight: 80, maxHeight: 160),
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            decoration: BoxDecoration(
              color: colors.base200,
              borderRadius: BorderRadius.circular(radii.field),
              border: Border.all(color: colors.line, width: 1),
            ),
            child: TextField(
              controller: controller,
              maxLines: null,
              minLines: 3,
              expands: false,
              style: const TextStyle(fontSize: 14, height: 1.5),
              decoration: InputDecoration(
                hintText: hintText,
                border: InputBorder.none,
                isDense: true,
              ),
            ),
          ),
          const SizedBox(height: 10),
          Row(
            children: <Widget>[
              if (onPickImage != null)
                IconButton(
                  icon: Icon(
                    Icons.image_outlined,
                    size: 20,
                    color: colors.iconMuted,
                  ),
                  onPressed: onPickImage,
                  tooltip: '图片',
                ),
              const Spacer(),
              GfButton(
                label: publishLabel,
                size: GfButtonSize.medium,
                loading: publishing,
                onPressed: publishing ? null : onPublish,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

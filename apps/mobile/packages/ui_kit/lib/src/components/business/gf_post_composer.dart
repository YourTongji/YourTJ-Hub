import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_button.dart';

/// Compact reply surface with a borderless growing editor and one action row.
/// Optional verification content appears only when supplied by the caller.
class GfPostComposer extends StatelessWidget {
  const GfPostComposer({
    super.key,
    required this.controller,
    required this.onPublish,
    required this.publishLabel,
    required this.hintText,
    this.focusNode,
    this.hideKeyboardLabel,
    this.onCollapse,
    this.collapseLabel,
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
  final String? hideKeyboardLabel;
  final VoidCallback? onCollapse;
  final String? collapseLabel;
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

    return Material(
      color: colors.base100,
      elevation: 4,
      shadowColor: colors.baseContent.withValues(alpha: 0.12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
        side: BorderSide(color: colors.line.withValues(alpha: 0.65)),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 8, 12, 8),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            if (targetName != null)
              Row(
                children: [
                  Icon(Icons.reply_rounded, size: 18, color: colors.iconMuted),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      targetLabel ?? targetName!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: GfTheme.typographyOf(
                        context,
                      ).caption.copyWith(color: colors.iconMuted),
                    ),
                  ),
                  if (onCloseTarget != null)
                    IconButton(
                      onPressed: onCloseTarget,
                      tooltip: MaterialLocalizations.of(
                        context,
                      ).closeButtonTooltip,
                      icon: Icon(
                        Icons.close_rounded,
                        size: 18,
                        color: colors.iconMuted,
                      ),
                    ),
                ],
              ),
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
            ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 160),
              child: TextField(
                controller: controller,
                focusNode: focusNode,
                keyboardType: TextInputType.multiline,
                textInputAction: TextInputAction.newline,
                maxLines: null,
                minLines: 2,
                style: GfTheme.typographyOf(context).body,
                decoration: InputDecoration(
                  hintText: hintText,
                  hintStyle: TextStyle(color: colors.iconMuted),
                  border: InputBorder.none,
                  enabledBorder: InputBorder.none,
                  focusedBorder: InputBorder.none,
                  disabledBorder: InputBorder.none,
                  filled: false,
                  isDense: true,
                  contentPadding: const EdgeInsets.symmetric(vertical: 8),
                ),
              ),
            ),
            if (toolbar != null)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 6),
                child: toolbar!,
              ),
            const SizedBox(height: 4),
            Row(
              children: [
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
                    constraints: const BoxConstraints(
                      minWidth: 44,
                      minHeight: 44,
                    ),
                  ),
                IconButton(
                  icon: Icon(
                    Icons.keyboard_hide_rounded,
                    size: 21,
                    color: colors.iconMuted,
                  ),
                  tooltip:
                      hideKeyboardLabel ??
                      MaterialLocalizations.of(context).closeButtonTooltip,
                  onPressed: () =>
                      FocusManager.instance.primaryFocus?.unfocus(),
                ),
                if (onCollapse != null)
                  IconButton(
                    onPressed: onCollapse,
                    tooltip:
                        collapseLabel ??
                        MaterialLocalizations.of(context).closeButtonTooltip,
                    icon: Icon(
                      Icons.keyboard_arrow_down_rounded,
                      size: 22,
                      color: colors.iconMuted,
                    ),
                  ),
                Expanded(
                  child: Align(
                    alignment: Alignment.centerRight,
                    child: GfButton(
                      label: publishLabel,
                      size: GfButtonSize.medium,
                      loading: publishing,
                      onPressed: publishing || !canPublish ? null : onPublish,
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

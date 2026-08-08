import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_divider.dart';

import '../surfaces/gf_floating_surface.dart';

/// Floating action in the topic controls bar (web TopicFloatingControls.vue).
class GfTopicAction {
  const GfTopicAction({
    required this.icon,
    required this.active,
    required this.activeColor,
    required this.onTap,
    this.acting = false,
    this.title,
  });

  final IconData icon;
  final bool active;

  /// Color of the icon when [active] (web activeClass per action type).
  final Color activeColor;

  final VoidCallback onTap;

  /// Shows a small spinner instead of the icon (web `Loader2 animate-spin`).
  final bool acting;

  final String? title;
}

/// Bottom floating controls for the topic page, mirroring web
/// TopicFloatingControls.vue: a pill `gf-floating-surface` with the floor
/// number button (`currentNo / maxNo`, primary, tabular), round action
/// buttons (36px) and the "join discussion" text button.
///
/// Positioning (bottom-4, z-90) is the caller's responsibility via [child]
/// placement in a [Stack]/[Overlay]; the widget itself is the pill content.
class GfFloatingControls extends StatelessWidget {
  const GfFloatingControls({
    super.key,
    required this.actions,
    required this.onOpenReply,
    this.currentNo,
    this.maxNo,
    this.onFloorTap,
    this.joinLabel = '参与讨论',
  });

  final List<GfTopicAction> actions;
  final VoidCallback onOpenReply;

  /// Current / max floor numbers; when null the floor button is hidden.
  final int? currentNo;
  final int? maxNo;
  final VoidCallback? onFloorTap;

  /// Label of the "join discussion" button (web `topic.joinDiscussion`).
  final String joinLabel;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return GfFloatingSurface(
      radius: 999,
      padding: const EdgeInsets.all(4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          if (currentNo != null && maxNo != null) ...<Widget>[
            InkWell(
              onTap: onFloorTap,
              borderRadius: BorderRadius.circular(999),
              child: Container(
                height: 36,
                padding: const EdgeInsets.symmetric(horizontal: 10),
                alignment: Alignment.center,
                child: Text(
                  '$currentNo / $maxNo',
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w800,
                    color: colors.primary,
                    fontFeatures: const <FontFeature>[
                      FontFeature.tabularFigures(),
                    ],
                  ),
                ),
              ),
            ),
            GfDivider(inset: 4, color: colors.line),
          ],
          for (final GfTopicAction action in actions)
            _RoundAction(action: action),
          GfDivider(inset: 4, color: colors.line),
          InkWell(
            onTap: onOpenReply,
            borderRadius: BorderRadius.circular(999),
            child: Container(
              height: 36,
              padding: const EdgeInsets.symmetric(horizontal: 12),
              alignment: Alignment.center,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  Icon(
                    Icons.chat_bubble_outline,
                    size: 16,
                    color: colors.baseContent.withValues(alpha: 0.75),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    joinLabel,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent.withValues(alpha: 0.75),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _RoundAction extends StatelessWidget {
  const _RoundAction({required this.action});

  final GfTopicAction action;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 2),
      child: Material(
        color: Colors.transparent,
        shape: const CircleBorder(),
        child: InkWell(
          onTap: action.onTap,
          customBorder: const CircleBorder(),
          child: Container(
            width: 36,
            height: 36,
            alignment: Alignment.center,
            child: action.acting
                ? SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: action.active
                          ? action.activeColor
                          : colors.baseContent.withValues(alpha: 0.75),
                    ),
                  )
                : Icon(
                    action.icon,
                    size: 16,
                    color: action.active
                        ? action.activeColor
                        : colors.baseContent.withValues(alpha: 0.75),
                  ),
          ),
        ),
      ),
    );
  }
}

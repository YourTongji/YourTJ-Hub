import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_divider.dart';
import '../gf_symbol.dart';

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
    this.symbol,
  });

  final IconData icon;
  final String? symbol;
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
  final VoidCallback? onOpenReply;

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
            Flexible(
              child: InkWell(
                onTap: onFloorTap,
                borderRadius: BorderRadius.circular(999),
                child: Container(
                  height: 36,
                  padding: const EdgeInsets.symmetric(horizontal: 10),
                  alignment: Alignment.center,
                  child: Text(
                    '$currentNo / $maxNo',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
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
            ),
            GfDivider(inset: 4, color: colors.line),
          ],
          for (final GfTopicAction action in actions)
            _RoundAction(action: action),
          if (onOpenReply != null) ...[
            GfDivider(inset: 4, color: colors.line),
            Flexible(
              flex: 3,
              child: _ReplyControl(label: joinLabel, onTap: onOpenReply!),
            ),
          ],
        ],
      ),
    );
  }
}

/// Keep a full accessible label when only the reply icon fits the dock.
class _ReplyControl extends StatelessWidget {
  const _ReplyControl({required this.label, required this.onTap});
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = GfTheme.colorsOf(context).baseContent.withValues(alpha: .75);
    final style = TextStyle(
      fontSize: 14,
      fontWeight: FontWeight.w600,
      color: color,
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        final painter = TextPainter(
          text: TextSpan(text: label, style: style),
          textDirection: Directionality.of(context),
          textScaler: MediaQuery.textScalerOf(context),
          maxLines: 1,
        )..layout();
        final showLabel = painter.width + 46 <= constraints.maxWidth;
        painter.dispose();
        return Tooltip(
          message: label,
          child: Semantics(
            label: label,
            button: true,
            child: InkWell(
              onTap: onTap,
              borderRadius: BorderRadius.circular(999),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: SizedBox(
                  height: 36,
                  child: ExcludeSemantics(
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        GfSymbol('corner-down-left', size: 16, color: color),
                        if (showLabel) ...[
                          const SizedBox(width: 6),
                          Text(label, style: style),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      },
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
                : action.symbol != null
                ? GfSymbol(
                    action.symbol!,
                    size: 18,
                    color: action.active
                        ? action.activeColor
                        : colors.iconMuted,
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

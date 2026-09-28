import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_divider.dart';
import '../atoms/gf_loading_indicator.dart';
import '../gf_action_feedback.dart';
import '../gf_symbol.dart';
import '../surfaces/gf_floating_surface.dart';

/// Gap a dock divider keeps on either side of the items it separates.
const double _dividerInset = 4;

/// Footprint of a dock divider inside a `Row`: the hairline is horizontal and
/// has no width along the main axis, so only the two [_dividerInset]s remain.
const double _dividerExtent = _dividerInset * 2;

/// Horizontal padding of the floor button around its label.
const EdgeInsets _floorPadding = EdgeInsets.symmetric(horizontal: 10);

/// Style a `Text` with [style] actually renders with: it is merged into the
/// ambient [DefaultTextStyle], so measurements have to do the same or they
/// are off by the inherited metrics (family, letter spacing).
TextStyle _renderedStyle(BuildContext context, TextStyle style) =>
    DefaultTextStyle.of(context).style.merge(style);

/// Natural single-line width of [text] in [style], laid out the way a `Text`
/// would under the ambient [MediaQuery.textScalerOf].
double _labelWidth(BuildContext context, String text, TextStyle style) {
  final TextPainter painter = TextPainter(
    text: TextSpan(text: text, style: _renderedStyle(context, style)),
    textDirection: Directionality.of(context),
    textScaler: MediaQuery.textScalerOf(context),
    maxLines: 1,
  )..layout();
  final double width = painter.width;
  painter.dispose();
  return width;
}

/// Floating action in the topic controls bar (web TopicFloatingControls.vue).
class GfTopicAction {
  const GfTopicAction({
    required this.symbol,
    required this.active,
    required this.activeColor,
    required this.onTap,
    this.acting = false,
    this.title,
  });

  final String symbol;
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
    final bool hasFloor = currentNo != null && maxNo != null;
    final VoidCallback? openReply = onOpenReply;
    final bool hasReply = openReply != null;
    final String floorLabel = hasFloor ? '$currentNo / $maxNo' : '';
    final TextStyle floorStyle = _renderedStyle(
      context,
      TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w800,
        color: colors.primary,
        fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
      ),
    );

    return GfFloatingSurface(
      radius: 999,
      padding: const EdgeInsets.all(4),
      child: LayoutBuilder(
        builder: (BuildContext context, BoxConstraints constraints) {
          // A `Flexible` child always receives a share of the whole free space,
          // whether or not it needs it: the floor number used to be squeezed
          // into "1 …" while the reply control left most of its share unused.
          // Size the floor number first and hand the remainder to the reply
          // control, which drops its label when it runs out of room, so the
          // number only ellipsizes when the pill cannot fit it next to the
          // controls that remain.
          final double reserved =
              actions.length * _RoundAction.extent +
              (hasFloor ? _dividerExtent : 0) +
              (hasReply ? _dividerExtent + _ReplyControl.minExtent : 0);
          final double naturalFloor = hasFloor
              ? _labelWidth(context, floorLabel, floorStyle) +
                    _floorPadding.horizontal
              : 0;
          final double floorExtent = math.min(
            naturalFloor,
            math.max(0, constraints.maxWidth - reserved),
          );

          return Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              if (hasFloor) ...<Widget>[
                SizedBox(
                  width: floorExtent,
                  child: InkWell(
                    onTap: onFloorTap,
                    borderRadius: BorderRadius.circular(999),
                    child: Container(
                      height: 36,
                      padding: _floorPadding,
                      alignment: Alignment.center,
                      child: Text(
                        floorLabel,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: floorStyle,
                      ),
                    ),
                  ),
                ),
                GfDivider(inset: _dividerInset, color: colors.line),
              ],
              for (final GfTopicAction action in actions)
                _RoundAction(action: action),
              if (openReply != null) ...<Widget>[
                GfDivider(inset: _dividerInset, color: colors.line),
                Flexible(
                  child: _ReplyControl(label: joinLabel, onTap: openReply),
                ),
              ],
            ],
          );
        },
      ),
    );
  }
}

/// Keep a full accessible label when only the reply icon fits the dock.
class _ReplyControl extends StatelessWidget {
  const _ReplyControl({required this.label, required this.onTap});
  final String label;
  final VoidCallback onTap;

  /// Symbol size, gap between symbol and label, and the horizontal padding of
  /// the control, also fixed by the layout below.
  static const double _symbolExtent = 16;
  static const double _labelGap = 6;
  static const double _hPadding = 12;

  /// Width the reply control needs once its label is dropped; while the
  /// control is present the dock keeps this much free before sizing the floor
  /// button.
  static const double minExtent = _symbolExtent + _hPadding * 2;

  /// Width the reply control needs with its label shown.
  static const double _labelExtent = _symbolExtent + _labelGap + _hPadding * 2;

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
        final showLabel =
            _labelWidth(context, label, style) + _labelExtent <=
            constraints.maxWidth;
        return Tooltip(
          message: label,
          child: Semantics(
            label: label,
            button: true,
            child: InkWell(
              onTap: onTap,
              borderRadius: BorderRadius.circular(999),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: _hPadding),
                child: SizedBox(
                  height: 36,
                  child: ExcludeSemantics(
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        GfSymbol(
                          'corner-down-left',
                          size: _symbolExtent,
                          color: color,
                        ),
                        if (showLabel) ...[
                          const SizedBox(width: _labelGap),
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

  /// Tap target and the gutter it keeps on either side in the dock row.
  static const double _tapExtent = 36;
  static const double _gutter = 2;

  /// Footprint of one action inside the dock row: the dock reserves this much
  /// before sizing the floor button.
  static const double extent = _tapExtent + _gutter * 2;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return GfActionFeedback(
      active: action.active,
      onPressed: action.onTap,
      child: GfSymbol(
        action.symbol,
        size: 18,
        color: action.active ? action.activeColor : colors.iconMuted,
      ),
      builder: (activate, visual) => Tooltip(
        message: action.title ?? '',
        excludeFromSemantics: true,
        child: Semantics(
          label: action.title,
          button: true,
          enabled: true,
          toggled: action.active,
          onTap: activate,
          excludeSemantics: true,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: _gutter),
            child: Material(
              color: Colors.transparent,
              shape: const CircleBorder(),
              child: InkWell(
                onTap: activate,
                customBorder: const CircleBorder(),
                child: Container(
                  width: _tapExtent,
                  height: _tapExtent,
                  alignment: Alignment.center,
                  child: action.acting
                      ? SizedBox(
                          width: 16,
                          height: 16,
                          child: GfProgressIndicator(
                            strokeWidth: 2,
                            color: action.active
                                ? action.activeColor
                                : colors.baseContent.withValues(alpha: 0.75),
                          ),
                        )
                      : visual,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';

/// Topic controls inside a verified native side pocket. The caller owns its
/// placement; short pockets scroll instead of reducing the 48px touch targets.
class TopicSideActions extends StatelessWidget {
  const TopicSideActions({
    super.key,
    required this.onBack,
    required this.actions,
    required this.currentNo,
    required this.maxNo,
    required this.onFloorTap,
    required this.onOpenReply,
  });

  final VoidCallback onBack;
  final List<GfTopicAction> actions;
  final int currentNo;
  final int? maxNo;
  final VoidCallback? onFloorTap;
  final VoidCallback? onOpenReply;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Align(
        alignment: Alignment.bottomCenter,
        child: SizedBox(
          key: const Key('topic-action-rail'),
          width: 56,
          child: Stack(
            children: [
              const Positioned.fill(
                child: GfNavigationSurface(
                  borderRadius: BorderRadius.all(Radius.circular(28)),
                ),
              ),
              Material(
                type: MaterialType.transparency,
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(28),
                  child: ListView(
                    shrinkWrap: true,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 4,
                      vertical: 8,
                    ),
                    children: [
                      GfIconButton(
                        symbol: 'chevron-left',
                        tooltip: l10n.commonBack,
                        size: 48,
                        onPressed: onBack,
                      ),
                      if (maxNo != null) ...[
                        Padding(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 10,
                            vertical: 6,
                          ),
                          child: GfDivider(color: colors.separator),
                        ),
                        GfIconButton(
                          symbol: 'list-ordered',
                          tooltip:
                              '${l10n.topicJumpToFloor} · $currentNo / $maxNo',
                          size: 48,
                          onPressed: onFloorTap,
                        ),
                      ],
                      for (final action in actions)
                        GfActionFeedback(
                          active: action.active,
                          onPressed: action.onTap,
                          child: GfSymbol(
                            action.symbol,
                            color: action.active
                                ? action.activeColor
                                : colors.iconMuted,
                          ),
                          builder: (activate, visual) => Semantics(
                            label: action.title,
                            button: true,
                            enabled: true,
                            toggled: action.active,
                            onTap: activate,
                            excludeSemantics: true,
                            child: Tooltip(
                              message: action.title ?? '',
                              excludeFromSemantics: true,
                              child: SizedBox.square(
                                dimension: 48,
                                child: TextButton(
                                  onPressed: activate,
                                  style: TextButton.styleFrom(
                                    padding: EdgeInsets.zero,
                                    shape: const CircleBorder(),
                                  ),
                                  child: visual,
                                ),
                              ),
                            ),
                          ),
                        ),
                      if (onOpenReply != null)
                        GfIconButton(
                          symbol: 'corner-down-left',
                          tooltip: l10n.topicJoinDiscussion,
                          size: 48,
                          color: colors.primary,
                          onPressed: onOpenReply,
                        ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

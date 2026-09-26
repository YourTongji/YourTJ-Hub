import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../format.dart';

Color userBadgeColor(UserBadgePayload badge) {
  const colors = <String, Color>{
    'blue': Color(0xFF1D4ED8),
    'emerald': Color(0xFF047857),
    'teal': Color(0xFF0F766E),
    'sky': Color(0xFF0369A1),
    'cyan': Color(0xFF0E7490),
    'rose': Color(0xFFBE123C),
    'violet': Color(0xFF6D28D9),
    'purple': Color(0xFF7E22CE),
    'fuchsia': Color(0xFFA21CAF),
    'indigo': Color(0xFF4338CA),
    'amber': Color(0xFFB45309),
    'orange': Color(0xFFC2410C),
    'yellow': Color(0xFFA16207),
    'slate': Color(0xFF334155),
  };
  return colors[badge.color] ??
      (badge.level == 'gold'
          ? colors['amber']!
          : badge.level == 'special'
          ? colors['indigo']!
          : colors['blue']!);
}

/// The parent control supplies the accessible name and the detail interaction.
class UserBadgeArtwork extends StatelessWidget {
  const UserBadgeArtwork(this.badge, {super.key, this.size = 32});
  final UserBadgePayload badge;
  final double size;

  @override
  Widget build(BuildContext context) => TooltipVisibility(
    visible: false,
    child: ExcludeSemantics(
      child: GfBadgeIcon(
        url: resolveApiAssetUrl(badge.iconUrl),
        label: badge.name,
        framed: false,
        size: size,
      ),
    ),
  );
}

Future<void> showUserBadgeDetails(
  BuildContext context,
  UserBadgePayload badge,
) {
  final l10n = AppLocalizations.of(context);
  final date = DateTime.tryParse(badge.grantedAt);
  return showBadgeDetails(
    context,
    title: badge.name,
    description: badge.description,
    color: userBadgeColor(badge),
    icon: UserBadgeArtwork(badge, size: 40),
    note: badge.reason.trim(),
    earned: date == null
        ? null
        : l10n.profileBadgeEarnedOn(formatDate(badge.grantedAt)),
  );
}

/// A bounded, scrollable native info card for taps; no navigation or data write.
Future<void> showBadgeDetails(
  BuildContext context, {
  required String title,
  required String description,
  required Color color,
  required Widget icon,
  String note = '',
  String? earned,
}) => showGfBottomSheet<void>(
  context,
  builder: (context) {
    final colors = GfTheme.colorsOf(context);
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(24, 4, 24, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Align(
              alignment: AlignmentDirectional.centerEnd,
              child: GfIconButton(
                symbol: 'x',
                tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
                onPressed: () => Navigator.of(context).pop(),
              ),
            ),
            ExcludeSemantics(
              child: GfBadgeMedallion(icon: icon, color: color, size: 80),
            ),
            const SizedBox(height: 16),
            Semantics(
              header: true,
              child: Text(
                title,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w700,
                  color: colors.baseContent,
                ),
              ),
            ),
            if (description.trim().isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                description.trim(),
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 15,
                  height: 1.5,
                  color: colors.iconMuted,
                ),
              ),
            ],
            if (note.isNotEmpty &&
                note != description.trim() &&
                note != title.trim()) ...[
              const SizedBox(height: 16),
              Text(
                note,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 14,
                  height: 1.5,
                  color: colors.baseContent,
                ),
              ),
            ],
            if (earned != null) ...[
              const SizedBox(height: 16),
              Text(
                earned,
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 13, color: colors.iconMuted),
              ),
            ],
          ],
        ),
      ),
    );
  },
);

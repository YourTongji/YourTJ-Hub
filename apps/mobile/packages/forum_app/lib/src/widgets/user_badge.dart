import 'dart:math' as math;

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

/// Match the Web avatar badge's bg-100 and ring-200 color for each badge hue.
const _wornBadgeColors = <String, (Color, Color)>{
  'blue': (Color(0xFFDBEAFE), Color(0xFFBFDBFE)),
  'emerald': (Color(0xFFD1FAE5), Color(0xFFA7F3D0)),
  'teal': (Color(0xFFCCFBF1), Color(0xFF99F6E4)),
  'sky': (Color(0xFFE0F2FE), Color(0xFFBAE6FD)),
  'cyan': (Color(0xFFCFFAFE), Color(0xFFA5F3FC)),
  'rose': (Color(0xFFFFE4E6), Color(0xFFFECDD3)),
  'violet': (Color(0xFFEDE9FE), Color(0xFFDDD6FE)),
  'purple': (Color(0xFFF3E8FF), Color(0xFFE9D5FF)),
  'fuchsia': (Color(0xFFFAE8FF), Color(0xFFF5D0FE)),
  'indigo': (Color(0xFFE0E7FF), Color(0xFFC7D2FE)),
  'amber': (Color(0xFFFEF3C7), Color(0xFFFDE68A)),
  'orange': (Color(0xFFFFEDD5), Color(0xFFFED7AA)),
  'yellow': (Color(0xFFFEF9C3), Color(0xFFFEF08A)),
  'slate': (Color(0xFFF1F5F9), Color(0xFFE2E8F0)),
};

class UserWornBadge extends StatelessWidget {
  const UserWornBadge(this.badge, {super.key, required this.avatarSize});

  final UserBadgePayload badge;
  final double avatarSize;

  @override
  Widget build(BuildContext context) {
    final size = math.max(14.0, avatarSize * .3);
    final (background, ring) = _wornBadgeColors[badge.color] ??
        _wornBadgeColors[switch (badge.level) {
          'gold' => 'amber',
          'special' => 'indigo',
          _ => 'blue',
        }]!;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: background,
        shape: BoxShape.circle,
        border: Border.all(color: ring, width: 2),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: .1),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: GfBadgeIcon(
        url: resolveApiAssetUrl(
          badge.iconUrl.isEmpty
              ? '/static/badges/contributor.svg'
              : badge.iconUrl,
        ),
        label: badge.name,
        size: size,
        framed: false,
      ),
    );
  }
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

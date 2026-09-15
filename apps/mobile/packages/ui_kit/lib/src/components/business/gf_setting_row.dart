import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';
import '../gf_icon_tile.dart';
import '../gf_symbol.dart';

/// Settings list row mirroring web SettingsPage.vue form rows: an optional
/// leading icon, a title + optional description, and a trailing widget.
/// Uses the Material [ListTile] primitives with the Gf `listTileTheme`
/// (contentPadding 16, minVerticalPadding 12).
class GfSettingRow extends StatelessWidget {
  const GfSettingRow({
    super.key,
    required this.title,
    this.description,
    this.subtitleWidget,
    this.icon,
    this.symbol,
    this.iconColor,
    this.leading,
    this.trailing,
    this.onTap,
  });

  final String title;
  final String? description;

  /// Arbitrary subtitle widget; takes precedence over [description].
  final Widget? subtitleWidget;

  final IconData? icon;
  final String? symbol;
  final Color? iconColor;

  /// Arbitrary leading widget (e.g. [GfAvatar]); takes precedence over
  /// [icon].
  final Widget? leading;

  final Widget? trailing;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return td.TCell(
      onTap: onTap,
      arrow: false,
      prefix:
          leading ??
          (symbol != null
              ? GfIconTile(symbol!, color: iconColor)
              : icon == null
              ? null
              : Icon(icon, size: 20, color: colors.iconMuted)),
      title: Text(
        title,
        maxLines: 3,
        softWrap: true,
        style: TextStyle(fontSize: 16, height: 1.35, color: colors.baseContent),
      ),
      subtitle:
          subtitleWidget ??
          (description == null
              ? null
              : Text(
                  description!,
                  style: TextStyle(
                    fontSize: 14,
                    height: 1.45,
                    color: colors.baseContent.withValues(alpha: 0.72),
                  ),
                )),
      trailing:
          trailing ??
          (onTap == null
              ? null
              : GfSymbol('chevron-right', size: 20, color: colors.iconMuted)),
    );
  }
}

/// Settings switch row mirroring the same layout with a [Switch] trailing
/// (web privacy/security toggles).
class GfSwitchRow extends StatelessWidget {
  const GfSwitchRow({
    super.key,
    required this.title,
    required this.value,
    required this.onChanged,
    this.description,
    this.subtitleWidget,
    this.icon,
    this.symbol,
    this.iconColor,
    this.leading,
  });

  final String title;
  final String? description;

  /// Arbitrary subtitle widget; takes precedence over [description].
  final Widget? subtitleWidget;

  final IconData? icon;
  final String? symbol;
  final Color? iconColor;

  /// Arbitrary leading widget; takes precedence over [icon].
  final Widget? leading;

  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return td.TCell(
      onTap: () => onChanged(!value),
      prefix:
          leading ??
          (symbol != null
              ? GfIconTile(symbol!, color: iconColor)
              : icon == null
              ? null
              : Icon(icon, size: 20, color: colors.iconMuted)),
      title: Text(
        title,
        maxLines: 3,
        softWrap: true,
        style: TextStyle(fontSize: 16, height: 1.35, color: colors.baseContent),
      ),
      subtitle:
          subtitleWidget ??
          (description == null
              ? null
              : Text(
                  description!,
                  style: TextStyle(
                    fontSize: 14,
                    height: 1.45,
                    color: colors.baseContent.withValues(alpha: 0.72),
                  ),
                )),
      trailing: td.TSwitch(value: value, onChanged: onChanged),
    );
  }
}

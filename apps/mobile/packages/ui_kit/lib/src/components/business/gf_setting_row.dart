import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

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
    this.leading,
    this.trailing,
    this.onTap,
  });

  final String title;
  final String? description;

  /// Arbitrary subtitle widget; takes precedence over [description].
  final Widget? subtitleWidget;

  final IconData? icon;

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
      arrow: onTap != null && trailing == null,
      prefix:
          leading ??
          (icon == null ? null : Icon(icon, size: 20, color: colors.iconMuted)),
      title: Text(title),
      subtitle:
          subtitleWidget ?? (description == null ? null : Text(description!)),
      trailing: trailing,
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
    this.leading,
  });

  final String title;
  final String? description;

  /// Arbitrary subtitle widget; takes precedence over [description].
  final Widget? subtitleWidget;

  final IconData? icon;

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
          (icon == null ? null : Icon(icon, size: 20, color: colors.iconMuted)),
      title: Text(title),
      subtitle:
          subtitleWidget ?? (description == null ? null : Text(description!)),
      trailing: td.TSwitch(value: value, onChanged: onChanged),
    );
  }
}

import 'package:flutter/material.dart';

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
    this.icon,
    this.trailing,
    this.onTap,
  });

  final String title;
  final String? description;
  final IconData? icon;
  final Widget? trailing;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return ListTile(
      onTap: onTap,
      leading: icon == null
          ? null
          : Icon(icon, size: 20, color: colors.iconMuted),
      title: Text(title),
      subtitle: description == null ? null : Text(description!),
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
    this.icon,
  });

  final String title;
  final String? description;
  final IconData? icon;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return ListTile(
      leading: icon == null
          ? null
          : Icon(icon, size: 20, color: colors.iconMuted),
      title: Text(title),
      subtitle: description == null ? null : Text(description!),
      trailing: Switch(value: value, onChanged: onChanged),
    );
  }
}

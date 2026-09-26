import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';

/// Shared navigation/value row. Root and detail settings use the same neutral
/// icon, type scale and padding; text grows instead of shrinking or clipping.
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

    final prefix =
        leading ??
        (symbol != null
            ? GfSymbol(
                symbol!,
                size: 22,
                color: iconColor ?? colors.baseContent,
              )
            : icon == null
            ? null
            : Icon(icon, size: 22, color: iconColor ?? colors.baseContent));
    final suffix =
        trailing ??
        (onTap == null
            ? null
            : GfSymbol('chevron-right', size: 18, color: colors.iconMuted));
    final subtitle =
        subtitleWidget ?? (description == null ? null : Text(description!));
    return InkWell(
      onTap: onTap,
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 56),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          child: Row(
            children: [
              if (prefix != null) ...[
                if (leading != null)
                  prefix
                else
                  SizedBox(width: 24, child: prefix),
                const SizedBox(width: 14),
              ],
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      softWrap: true,
                      style: TextStyle(
                        fontSize: 16,
                        height: 1.35,
                        color: colors.baseContent,
                      ),
                    ),
                    if (subtitle != null) ...[
                      const SizedBox(height: 4),
                      DefaultTextStyle(
                        style: TextStyle(
                          fontSize: 13,
                          height: 1.4,
                          color: colors.iconMuted,
                        ),
                        child: subtitle,
                      ),
                    ],
                  ],
                ),
              ),
              if (suffix != null) ...[const SizedBox(width: 12), suffix],
            ],
          ),
        ),
      ),
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
    return GfSettingRow(
      onTap: () => onChanged(!value),
      title: title,
      description: description,
      subtitleWidget: subtitleWidget,
      leading: leading,
      symbol: symbol,
      icon: icon,
      iconColor: iconColor,
      trailing: Switch.adaptive(value: value, onChanged: onChanged),
    );
  }
}

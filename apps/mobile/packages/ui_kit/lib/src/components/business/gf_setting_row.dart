import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../gf_symbol.dart';

/// Shared navigation/value row. Root and detail settings use the same neutral
/// icon, type scale and padding; text grows instead of shrinking or clipping.
class GfSettingRow extends StatelessWidget {
  const GfSettingRow({
    super.key,
    required this.title,
    this.description,
    this.subtitleWidget,
    this.symbol,
    this.iconColor,
    this.leading,
    this.trailing,
    this.value,
    this.destructive = false,
    this.onTap,
  });

  final String title;
  final String? description;

  /// Replaces [description] with custom subtitle content.
  final Widget? subtitleWidget;

  /// Neutral 22px ReIcon on the shared 24px column.
  final String? symbol;
  final Color? iconColor;

  /// Custom leading widget (avatar, tile); replaces [symbol].
  final Widget? leading;

  /// Custom trailing widget; replaces [value] and the chevron.
  final Widget? trailing;

  /// Current value shown before the chevron (language, cache size, ...).
  final String? value;

  /// Destructive rows (sign out, delete) use the error colour.
  final bool destructive;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);
    final Color titleColor = destructive ? colors.error : colors.textPrimary;

    final prefix =
        leading ??
        (symbol == null
            ? null
            : GfSymbol(
                symbol!,
                size: 22,
                color: iconColor ?? (destructive ? colors.error : titleColor),
              ));
    final Widget? chevron = onTap == null || destructive
        ? null
        : GfSymbol(
            'chevron-right',
            size: GfIconSize.sm,
            color: colors.textPlaceholder,
          );
    final Widget? suffix =
        trailing ??
        (value == null
            ? chevron
            : Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Flexible(
                    child: GfText(
                      value!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: typography.body.copyWith(
                        color: colors.textSecondary,
                      ),
                    ),
                  ),
                  if (chevron != null) ...[
                    const SizedBox(width: GfSpace.xs),
                    chevron,
                  ],
                ],
              ));
    final subtitle =
        subtitleWidget ?? (description == null ? null : GfText(description!));
    return InkWell(
      onTap: onTap,
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: GfControl.row),
        child: Padding(
          padding: const EdgeInsets.symmetric(
            horizontal: GfSpace.lg,
            vertical: GfSpace.md,
          ),
          child: Row(
            children: [
              if (prefix != null) ...[
                if (leading == null)
                  SizedBox(width: GfIconSize.lg, child: prefix)
                else
                  ConstrainedBox(
                    constraints: const BoxConstraints(minWidth: GfIconSize.lg),
                    child: Center(
                      widthFactor: 1,
                      heightFactor: 1,
                      child: prefix,
                    ),
                  ),
                const SizedBox(width: GfSpace.md),
              ],
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfText(
                      title,
                      softWrap: true,
                      style: typography.body.copyWith(color: titleColor),
                    ),
                    if (subtitle != null) ...[
                      const SizedBox(height: GfSpace.xxs),
                      DefaultTextStyle(
                        style: typography.caption.copyWith(
                          color: colors.textSecondary,
                        ),
                        child: subtitle,
                      ),
                    ],
                  ],
                ),
              ),
              if (suffix != null) ...[
                const SizedBox(width: GfSpace.md),
                if (value != null && trailing == null)
                  ConstrainedBox(
                    constraints: BoxConstraints(
                      maxWidth: MediaQuery.sizeOf(context).width * 0.45,
                    ),
                    child: suffix,
                  )
                else
                  suffix,
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// Setting row with a trailing native switch; tapping the row toggles it.
class GfSwitchRow extends StatelessWidget {
  const GfSwitchRow({
    super.key,
    required this.title,
    required this.value,
    required this.onChanged,
    this.enabled = true,
    this.description,
    this.subtitleWidget,
    this.symbol,
    this.iconColor,
    this.leading,
  });

  final String title;
  final String? description;

  /// Replaces [description] with custom subtitle content.
  final Widget? subtitleWidget;

  final String? symbol;
  final Color? iconColor;

  /// Custom leading widget; replaces [symbol].
  final Widget? leading;

  final bool value;
  final bool enabled;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return GfSettingRow(
      onTap: enabled ? () => onChanged(!value) : null,
      title: title,
      description: description,
      subtitleWidget: subtitleWidget,
      leading: leading,
      symbol: symbol,
      iconColor: iconColor,
      // The whole row is the target: the switch paints at its native size
      // but lays out at text height so switch rows match the 52px rhythm.
      trailing: SizedBox(
        width: 64,
        height: 24,
        child: OverflowBox(
          maxHeight: 48,
          alignment: Alignment.centerRight,
          child: Switch.adaptive(
            value: value,
            onChanged: enabled ? onChanged : null,
            activeTrackColor: GfTheme.colorsOf(context).primary,
            materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
          ),
        ),
      ),
    );
  }
}

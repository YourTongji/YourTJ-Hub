import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Page header, mirroring web `PageHeader.vue` / `.gf-page-header`
/// (components.css): mobile form is a stacked column with an 8px gap, a
/// `line/70` bottom border and `px-4 py-3` padding; the title is 20px w700
/// (web `text-xl font-bold`), the description 14px `base-content/55`.
class GfPageHeader extends StatelessWidget {
  const GfPageHeader({
    super.key,
    required this.title,
    this.description,
    this.badge,
    this.actions,
  });

  final String title;
  final String? description;

  /// Optional badge next to the title (e.g. category color dot).
  final Widget? badge;

  /// Optional trailing actions slot (e.g. search form, action buttons).
  final Widget? actions;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(
            color: colors.line.withValues(alpha: 0.7),
            width: 1,
          ),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: <Widget>[
              Flexible(
                child: Text(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.w700,
                    color: colors.baseContent,
                  ),
                ),
              ),
              if (badge != null) ...<Widget>[const SizedBox(width: 8), badge!],
            ],
          ),
          if (description != null) ...<Widget>[
            const SizedBox(height: 4),
            Text(
              description!,
              style: TextStyle(
                fontSize: 14,
                color: colors.baseContent.withValues(alpha: 0.55),
              ),
            ),
          ],
          if (actions != null) ...<Widget>[
            const SizedBox(height: 12),
            actions!,
          ],
        ],
      ),
    );
  }
}

/// Section header inside a settings-style panel, mirroring web
/// `SectionHeader.vue`: a bottom border, `px-4 py-3` padding, 14px w600
/// title and an optional 12px `base-content/55` description.
class GfSectionHeader extends StatelessWidget {
  const GfSectionHeader({
    super.key,
    required this.title,
    this.description,
    this.icon,
    this.actions,
  });

  final String title;
  final String? description;
  final IconData? icon;
  final Widget? actions;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: colors.line, width: 1)),
      ),
      child: Row(
        children: <Widget>[
          if (icon != null) ...<Widget>[
            Icon(
              icon,
              size: 16,
              color: colors.baseContent.withValues(alpha: 0.55),
            ),
            const SizedBox(width: 8),
          ],
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  title,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: colors.baseContent,
                  ),
                ),
                if (description != null) ...<Widget>[
                  const SizedBox(height: 2),
                  Text(
                    description!,
                    style: TextStyle(
                      fontSize: 12,
                      color: colors.baseContent.withValues(alpha: 0.55),
                    ),
                  ),
                ],
              ],
            ),
          ),
          if (actions != null) ...<Widget>[actions!],
        ],
      ),
    );
  }
}

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../gf_symbol.dart';

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
      padding: const EdgeInsets.fromLTRB(
        GfSpace.gutter,
        GfSpace.lg,
        GfSpace.gutter,
        GfSpace.md,
      ),
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(
            color: colors.separator,
            width: gfHairline(context),
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
                child: GfText(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: GfTheme.typographyOf(context).title1,
                ),
              ),
              if (badge != null) ...<Widget>[const SizedBox(width: 8), badge!],
            ],
          ),
          if (description != null) ...<Widget>[
            const SizedBox(height: GfSpace.xs),
            GfText(
              description!,
              style: GfTheme.typographyOf(
                context,
              ).small.copyWith(color: colors.textSecondary),
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
    this.symbol,
    this.actions,
  });

  final String title;
  final String? description;
  final String? symbol;
  final Widget? actions;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: GfSpace.gutter,
        vertical: GfSpace.md,
      ),
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(
            color: colors.separator,
            width: gfHairline(context),
          ),
        ),
      ),
      child: Row(
        children: <Widget>[
          if (symbol != null) ...<Widget>[
            GfSymbol(symbol!, size: 18, color: colors.textSecondary),
            const SizedBox(width: 8),
          ],
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                GfText(title, style: GfTheme.typographyOf(context).heading),
                if (description != null) ...<Widget>[
                  const SizedBox(height: 2),
                  GfText(
                    description!,
                    style: GfTheme.typographyOf(
                      context,
                    ).caption.copyWith(color: colors.textSecondary),
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

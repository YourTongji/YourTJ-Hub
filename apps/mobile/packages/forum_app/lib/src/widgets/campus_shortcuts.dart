import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';

/// Real campus destinations, shared by discovery and the campus landing page.
class CampusShortcuts extends StatelessWidget {
  const CampusShortcuts({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final entries = [
      ('graduation-cap', l10n.coursesTitle, '/courses', colors.primary),
      ('calendar-days', l10n.scheduleTitle, '/schedule', colors.success),
      ('book-open', l10n.wikiTitle, '/wiki', colors.warning),
    ];
    return LayoutBuilder(
      builder: (context, constraints) {
        final stacked =
            constraints.maxWidth < 320 ||
            MediaQuery.textScalerOf(context).scale(16) > 22;
        return Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            for (final (symbol, label, route, tone) in entries)
              SizedBox(
                width: stacked
                    ? constraints.maxWidth
                    : (constraints.maxWidth - 24) / 3,
                child: Semantics(
                  button: true,
                  child: Material(
                    color: colors.base100,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(20),
                      side: BorderSide(
                        color: colors.line.withValues(alpha: 0.7),
                      ),
                    ),
                    clipBehavior: Clip.antiAlias,
                    child: InkWell(
                      onTap: () => context.push(route),
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: stacked
                            ? Row(
                                children: [
                                  GfIconTile(symbol, color: tone, size: 40),
                                  const SizedBox(width: 16),
                                  Expanded(
                                    child: Text(
                                      label,
                                      style: TextStyle(
                                        fontSize: 16,
                                        color: colors.baseContent,
                                      ),
                                    ),
                                  ),
                                  const GfSymbol('chevron-right', size: 20),
                                ],
                              )
                            : Column(
                                children: [
                                  GfIconTile(symbol, color: tone, size: 48),
                                  const SizedBox(height: 16),
                                  Text(
                                    label,
                                    textAlign: TextAlign.center,
                                    style: TextStyle(
                                      fontSize: 16,
                                      fontWeight: FontWeight.w600,
                                      color: colors.baseContent,
                                    ),
                                  ),
                                ],
                              ),
                      ),
                    ),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }
}

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../atoms/gf_loading_indicator.dart';
import '../gf_symbol.dart';

/// A readable state with a quiet symbol, explanation and optional next step.
/// Scrolls on short viewports and with large accessibility text.
class GfEmpty extends StatelessWidget {
  const GfEmpty({
    super.key,
    required this.message,
    this.description,
    this.symbol,
    this.loading = false,
    this.action,
  });

  final String message;
  final String? description;

  /// Shared outline symbol.
  final String? symbol;
  final bool loading;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 360),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (loading)
                const GfLoadingIndicator()
              else
                Container(
                  width: 64,
                  height: 64,
                  decoration: BoxDecoration(
                    color: colors.fill,
                    borderRadius: BorderRadius.circular(GfRadius.lg),
                  ),
                  child: Center(
                    child: GfSymbol(
                      symbol ?? 'inbox',
                      size: 28,
                      color: colors.textSecondary,
                    ),
                  ),
                ),
              const SizedBox(height: GfSpace.xl),
              GfText(
                message,
                textAlign: TextAlign.center,
                style: GfTheme.typographyOf(context).title3,
              ),
              if (description != null) ...[
                const SizedBox(height: GfSpace.sm - 2),
                GfText(
                  description!,
                  textAlign: TextAlign.center,
                  style: GfTheme.typographyOf(
                    context,
                  ).small.copyWith(color: colors.textSecondary),
                ),
              ],
              if (action != null) ...[
                const SizedBox(height: GfSpace.xxl),
                action!,
              ],
            ],
          ),
        ),
      ),
    );
  }
}

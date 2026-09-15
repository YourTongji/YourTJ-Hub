import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_loading_indicator.dart';

/// A readable state with a quiet symbol, explanation and optional next step.
/// Scrolls on short viewports and with large accessibility text.
class GfEmpty extends StatelessWidget {
  const GfEmpty({
    super.key,
    required this.message,
    this.description,
    this.icon,
    this.loading = false,
    this.action,
  });

  final String message;
  final String? description;
  final IconData? icon;
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
                    color: colors.primary.withValues(alpha: 0.08),
                    borderRadius: BorderRadius.circular(20),
                  ),
                  child: Icon(
                    icon ?? Icons.inbox_outlined,
                    size: 28,
                    color: colors.primary,
                  ),
                ),
              const SizedBox(height: 24),
              Text(
                message,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 18,
                  height: 1.35,
                  fontWeight: FontWeight.w600,
                  color: colors.baseContent,
                ),
              ),
              if (description != null) ...[
                const SizedBox(height: 8),
                Text(
                  description!,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.5,
                    color: colors.baseContent.withValues(alpha: 0.72),
                  ),
                ),
              ],
              if (action != null) ...[const SizedBox(height: 24), action!],
            ],
          ),
        ),
      ),
    );
  }
}

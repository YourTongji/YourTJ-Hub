import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Empty state, mirroring web `EmptyState.vue`: min-height 224px
/// (`min-h-56`), a 32px icon at `base-content/35`, a 16px w600 title and an
/// optional 14px `base-content/55` description. The default icon is an inbox
/// (web `Inbox`), and `loading` spins the icon like web's `animate-spin`.
class GfEmpty extends StatelessWidget {
  const GfEmpty({
    super.key,
    required this.message,
    this.description,
    this.icon,
    this.loading = false,
  });

  /// Primary empty-state text (16px w600, web `text-base font-semibold`).
  final String message;

  /// Optional secondary description (14px `base-content/55`).
  final String? description;

  /// Override the default inbox icon.
  final IconData? icon;

  /// Whether to spin the icon (web `animate-spin` loading empty states).
  final bool loading;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            Icon(
              icon ?? Icons.inbox_outlined,
              size: 32,
              color: colors.baseContent.withValues(alpha: 0.35),
            ),
            const SizedBox(height: 8),
            Text(
              message,
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.w600,
                color: colors.baseContent,
              ),
            ),
            if (description != null) ...<Widget>[
              const SizedBox(height: 4),
              Text(
                description!,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 14,
                  color: colors.baseContent.withValues(alpha: 0.55),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

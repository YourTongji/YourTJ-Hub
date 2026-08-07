import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';

/// 加载中视图。
class GfLoading extends StatelessWidget {
  const GfLoading({super.key, this.message});

  final String? message;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const CircularProgressIndicator(strokeWidth: 2.5),
          if (message != null) ...[
            const SizedBox(height: 12),
            Text(
              message!,
              style: TextStyle(color: colors.iconMuted, fontSize: 13),
            ),
          ],
        ],
      ),
    );
  }
}

/// 错误 + 重试视图。
class GfErrorRetry extends StatelessWidget {
  const GfErrorRetry({super.key, required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.cloud_off_outlined, size: 40, color: colors.iconMuted),
            const SizedBox(height: 12),
            Text(
              message,
              textAlign: TextAlign.center,
              style: TextStyle(color: colors.baseContent, fontSize: 14),
            ),
            const SizedBox(height: 16),
            GfButton(
              label: AppLocalizations.of(context).commonRetry,
              variant: GfButtonVariant.outline,
              onPressed: onRetry,
            ),
          ],
        ),
      ),
    );
  }
}

/// 空态视图。
class GfEmpty extends StatelessWidget {
  const GfEmpty({super.key, required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.inbox_outlined, size: 40, color: colors.iconMuted),
            const SizedBox(height: 12),
            Text(
              message,
              style: TextStyle(color: colors.iconMuted, fontSize: 13),
            ),
          ],
        ),
      ),
    );
  }
}

/// 列表底部加载指示器。
class GfListFooter extends StatelessWidget {
  const GfListFooter({
    super.key,
    required this.loading,
    required this.hasMore,
    required this.onLoadMore,
  });

  final bool loading;
  final bool hasMore;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    if (loading) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Center(
          child: SizedBox(
            width: 22,
            height: 22,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
        ),
      );
    }
    if (!hasMore) {
      return Padding(
        padding: const EdgeInsets.all(16),
        child: Center(
          child: Text(
            '— ${AppLocalizations.of(context).commonEmpty} —',
            style: TextStyle(color: colors.iconMuted, fontSize: 12),
          ),
        ),
      );
    }
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Center(
        child: GfButton(
          label: AppLocalizations.of(context).commonLoadMore,
          variant: GfButtonVariant.ghost,
          onPressed: onLoadMore,
        ),
      ),
    );
  }
}

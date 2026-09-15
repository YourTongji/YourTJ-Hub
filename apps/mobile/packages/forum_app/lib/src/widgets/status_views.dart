import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';

/// 加载中视图。
class GfLoading extends StatelessWidget {
  const GfLoading({super.key, this.message});

  final String? message;

  @override
  Widget build(BuildContext context) {
    return Center(child: GfLoadingIndicator(message: message));
  }
}

/// 错误 + 重试视图。
class GfErrorRetry extends StatelessWidget {
  const GfErrorRetry({super.key, required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return GfEmpty(
      icon: Icons.cloud_off_outlined,
      message: message,
      action: GfButton(
        label: AppLocalizations.of(context).commonRetry,
        variant: GfButtonVariant.outline,
        onPressed: onRetry,
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
    this.error,
  });

  final bool loading;
  final bool hasMore;
  final VoidCallback onLoadMore;
  final String? error;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    if (loading) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Center(child: GfLoadingIndicator(small: true)),
      );
    }
    if (!hasMore) {
      return Padding(
        padding: const EdgeInsets.all(16),
        child: Center(
          child: Text(
            AppLocalizations.of(context).commonEndOfList,
            style: GfTheme.typographyOf(
              context,
            ).caption.copyWith(color: colors.iconMuted),
          ),
        ),
      );
    }
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          if (error != null)
            Text(
              error!,
              textAlign: TextAlign.center,
              style: TextStyle(color: colors.error),
            ),
          GfButton(
            label: error == null
                ? AppLocalizations.of(context).commonLoadMore
                : AppLocalizations.of(context).commonRetry,
            variant: GfButtonVariant.ghost,
            onPressed: onLoadMore,
          ),
        ],
      ),
    );
  }
}

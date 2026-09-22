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
class GfListFooter extends StatefulWidget {
  const GfListFooter({
    super.key,
    required this.loading,
    required this.hasMore,
    required this.onLoadMore,
    this.error,
    this.progressKey,
    this.autoLoad = true,
  });

  final bool loading;
  final bool hasMore;
  final VoidCallback onLoadMore;
  final String? error;

  /// Next cursor or loaded count. A failed/no-progress response is not retried
  /// automatically; the visible button remains available for an explicit retry.
  final Object? progressKey;
  final bool autoLoad;

  @override
  State<GfListFooter> createState() => _GfListFooterState();
}

class _GfListFooterState extends State<GfListFooter> {
  ScrollPosition? _position;
  bool _requested = false;
  bool _scheduled = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final next = Scrollable.maybeOf(context)?.position;
    if (!identical(next, _position)) {
      _position?.removeListener(_schedule);
      _position = next;
      _position?.addListener(_schedule);
    }
    _schedule();
  }

  @override
  void didUpdateWidget(covariant GfListFooter oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.progressKey != widget.progressKey) _requested = false;
    _schedule();
  }

  void _schedule() {
    if (_scheduled || !mounted) return;
    _scheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _scheduled = false;
      if (!mounted ||
          _requested ||
          !widget.autoLoad ||
          widget.loading ||
          !widget.hasMore ||
          widget.error != null ||
          _position == null ||
          !TickerMode.valuesOf(context).enabled) {
        return;
      }
      final box = context.findRenderObject();
      if (box is! RenderBox || !box.hasSize || !box.attached) return;
      final top = box.localToGlobal(Offset.zero).dy;
      if (top + box.size.height < 0 ||
          top > MediaQuery.sizeOf(context).height + 240) {
        return;
      }
      _requested = true;
      widget.onLoadMore();
    });
  }

  @override
  void dispose() {
    _position?.removeListener(_schedule);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    if (widget.loading) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Center(child: GfLoadingIndicator(small: true)),
      );
    }
    if (!widget.hasMore) {
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
          if (widget.error != null)
            Text(
              widget.error!,
              textAlign: TextAlign.center,
              style: TextStyle(color: colors.error),
            ),
          GfButton(
            label: widget.error == null
                ? AppLocalizations.of(context).commonLoadMore
                : AppLocalizations.of(context).commonRetry,
            variant: GfButtonVariant.ghost,
            onPressed: widget.onLoadMore,
          ),
        ],
      ),
    );
  }
}

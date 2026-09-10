import 'dart:async';

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

final _activeToasts = Expando<OverlayEntry>();

/// Shared top feedback banner, above dialogs/sheets and clear of system insets.
/// A new message replaces the previous one; failures stay visible longer.
void showGfToast(BuildContext context, String message, {bool error = false}) {
  final overlay = Overlay.of(context, rootOverlay: true);
  final previous = _activeToasts[overlay];
  if (previous != null) {
    previous.remove();
    previous.dispose();
  }
  final themes = InheritedTheme.capture(from: context, to: overlay.context);
  late final OverlayEntry entry;
  void dismiss() {
    if (_activeToasts[overlay] != entry) return;
    _activeToasts[overlay] = null;
    entry.remove();
    entry.dispose();
  }

  entry = OverlayEntry(
    builder: (_) => themes.wrap(
      _FeedbackBanner(message: message, error: error, onDismiss: dismiss),
    ),
  );
  _activeToasts[overlay] = entry;
  overlay.insert(entry);
}

class _FeedbackBanner extends StatefulWidget {
  const _FeedbackBanner({
    required this.message,
    required this.error,
    required this.onDismiss,
  });
  final String message;
  final bool error;
  final VoidCallback onDismiss;

  @override
  State<_FeedbackBanner> createState() => _FeedbackBannerState();
}

class _FeedbackBannerState extends State<_FeedbackBanner> {
  Timer? _timer;
  @override
  void initState() {
    super.initState();
    _timer = Timer(Duration(seconds: widget.error ? 7 : 4), widget.onDismiss);
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final accent = widget.error ? colors.error : colors.success;
    return Positioned(
      top: 0,
      left: 0,
      right: 0,
      child: SafeArea(
        bottom: false,
        minimum: const EdgeInsets.fromLTRB(12, 8, 12, 0),
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 560),
            child: Material(
              color: colors.base100,
              elevation: 4,
              shadowColor: Colors.black.withValues(alpha: .14),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
                side: BorderSide(color: accent.withValues(alpha: .25)),
              ),
              child: Semantics(
                liveRegion: true,
                child: Padding(
                  padding: const EdgeInsets.only(
                    left: 14,
                    top: 6,
                    bottom: 6,
                    right: 4,
                  ),
                  child: Row(
                    children: [
                      Icon(
                        widget.error
                            ? Icons.error_outline_rounded
                            : Icons.check_circle_outline_rounded,
                        color: accent,
                        size: 22,
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(
                          widget.message,
                          style: GfTheme.typographyOf(
                            context,
                          ).body.copyWith(color: colors.baseContent),
                        ),
                      ),
                      IconButton(
                        tooltip: MaterialLocalizations.of(
                          context,
                        ).closeButtonTooltip,
                        onPressed: widget.onDismiss,
                        icon: Icon(
                          Icons.close_rounded,
                          size: 18,
                          color: colors.baseContent.withValues(alpha: .55),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

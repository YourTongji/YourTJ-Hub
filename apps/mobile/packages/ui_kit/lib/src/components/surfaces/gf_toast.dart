import 'dart:async';

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';
import 'gf_liquid_surface.dart';
import '../gf_symbol.dart';

final _activeToasts = Expando<_ToastEntry>();

class _ToastMessage {
  const _ToastMessage(this.text, this.error);
  final String text;
  final bool error;
}

class _ToastEntry {
  _ToastEntry(_ToastMessage message) : message = ValueNotifier(message);
  final ValueNotifier<_ToastMessage> message;
  late final OverlayEntry entry;
}

/// One live feedback surface per overlay. Replacements keep the same route-free
/// host; closing finishes its exit before removal. A new message cancels closing.
void showGfToast(BuildContext context, String message, {bool error = false}) {
  final overlay = Overlay.of(context, rootOverlay: true);
  final next = _ToastMessage(message, error);
  final previous = _activeToasts[overlay];
  if (previous != null) {
    previous.message.value = next;
    return;
  }
  final handle = _ToastEntry(next);
  final themes = InheritedTheme.capture(from: context, to: overlay.context);
  void remove(_ToastMessage expected) {
    if (_activeToasts[overlay] != handle || handle.message.value != expected) {
      return;
    }
    _activeToasts[overlay] = null;
    handle.entry.remove();
    handle.entry.dispose();
    handle.message.dispose();
  }

  handle.entry = OverlayEntry(
    builder: (_) => themes.wrap(
      ValueListenableBuilder<_ToastMessage>(
        valueListenable: handle.message,
        builder: (_, message, _) =>
            _FeedbackBanner(message: message, onDismiss: remove),
      ),
    ),
  );
  _activeToasts[overlay] = handle;
  overlay.insert(handle.entry);
}

class _FeedbackBanner extends StatefulWidget {
  const _FeedbackBanner({required this.message, required this.onDismiss});
  final _ToastMessage message;
  final ValueChanged<_ToastMessage> onDismiss;

  @override
  State<_FeedbackBanner> createState() => _FeedbackBannerState();
}

class _FeedbackBannerState extends State<_FeedbackBanner>
    with SingleTickerProviderStateMixin {
  late final _controller = AnimationController(
    vsync: this,
    duration: GfMotion.layout,
    reverseDuration: GfMotion.selection,
  );
  Timer? _timer;
  bool _started = false;
  bool _closing = false;
  int _generation = 0;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_started) {
      _started = true;
      _restart();
    }
    if (GfMotion.reducedOf(context)) {
      _controller.value = _closing ? 0 : 1;
      if (_closing) {
        final generation = _generation;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && _closing && generation == _generation) {
            widget.onDismiss(widget.message);
          }
        });
      }
    }
  }

  @override
  void didUpdateWidget(_FeedbackBanner oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.message != widget.message) _restart();
  }

  void _restart() {
    _generation++;
    _closing = false;
    _timer?.cancel();
    _timer = Timer(Duration(seconds: widget.message.error ? 7 : 4), _dismiss);
    if (GfMotion.reducedOf(context)) {
      _controller.value = 1;
    } else {
      _controller.forward();
    }
  }

  Future<void> _dismiss() async {
    if (_closing) return;
    _timer?.cancel();
    final generation = _generation;
    setState(() => _closing = true);
    if (!GfMotion.reducedOf(context)) {
      try {
        await _controller.reverse().orCancel;
      } on TickerCanceled {
        return;
      }
    }
    if (mounted && generation == _generation) widget.onDismiss(widget.message);
  }

  @override
  void dispose() {
    _timer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final accent = widget.message.error ? colors.error : colors.success;
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
            child: IgnorePointer(
              ignoring: _closing,
              child: GfFadeTransition(
                animation: _controller,
                offset: const Offset(0, -GfMotion.rise),
                child: AnimatedSwitcher(
                  duration: GfMotion.duration(context, GfMotion.content),
                  switchInCurve: GfMotion.enterCurve,
                  switchOutCurve: GfMotion.enterCurve,
                  layoutBuilder: (current, previous) => Stack(
                    alignment: Alignment.topCenter,
                    children: [
                      for (final child in previous)
                        ExcludeSemantics(child: IgnorePointer(child: child)),
                      ?current,
                    ],
                  ),
                  child: GfLiquidSurface(
                    key: ObjectKey(widget.message),
                    radius: 28,
                    weight: GfGlassWeight.strong,
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
                            GfSymbol(
                              widget.message.error
                                  ? 'circle-alert'
                                  : 'circle-check',
                              color: accent,
                              size: 22,
                            ),
                            const SizedBox(width: 10),
                            Expanded(
                              child: Text(
                                widget.message.text,
                                style: GfTheme.typographyOf(
                                  context,
                                ).body.copyWith(color: colors.baseContent),
                              ),
                            ),
                            IconButton(
                              tooltip: MaterialLocalizations.of(
                                context,
                              ).closeButtonTooltip,
                              onPressed: _dismiss,
                              icon: GfSymbol(
                                'x',
                                size: 18,
                                color: colors.baseContent.withValues(
                                  alpha: .55,
                                ),
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
          ),
        ),
      ),
    );
  }
}

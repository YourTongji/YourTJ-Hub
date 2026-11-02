import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';

/// The input surface a composer intends to show below its editor.
enum GfInputAccessory {
  none,
  keyboard,
  panel,

  /// Keeps the space of the keyboard that just left, e.g. while a sheet
  /// briefly takes focus, so the editor holds still until it returns.
  reserved,
}

/// Sole owner of a composer's bottom edge: software keyboard, in-app panel or
/// the bottom safe area. The host `Scaffold` must not resize for the keyboard
/// (`resizeToAvoidBottomInset: false`) so `viewInsets.bottom` is consumed here
/// exactly once.
///
/// When the keyboard and panel trade places, the slot stays as tall as the
/// outgoing surface until the incoming one covers it, so the editor above never
/// drops and rises again. The panel adopts the last settled keyboard height for
/// the current orientation, which makes a like-for-like switch motionless. A
/// panel that needs more room than a short keyboard ([minPanelHeight]) grows
/// smoothly from the keyboard's edge instead of jumping.
class GfInputAccessorySlot extends StatefulWidget {
  const GfInputAccessorySlot({
    super.key,
    required this.accessory,
    required this.panelFocusNode,
    required this.panelBuilder,
    required this.availableHeight,
    this.minPanelHeight = 0,
    this.maxPanelWidth = 560,
  });

  final GfInputAccessory accessory;

  /// Takes focus while the panel is open, dismissing the software keyboard.
  /// A text field inside the panel (such as search) is its descendant.
  final FocusNode panelFocusNode;

  /// Built lazily on first open and retained, hidden, while the slot stays
  /// mounted, so tab, scroll and search state survive keyboard round trips.
  final WidgetBuilder panelBuilder;

  /// Height of the box hosting the composer; bounds the panel.
  final double availableHeight;

  /// Content height the panel needs to be useful, bottom safe area excluded.
  /// The panel is at least this tall, still within the [availableHeight] cap.
  final double minPanelHeight;
  final double maxPanelWidth;

  @override
  State<GfInputAccessorySlot> createState() => _GfInputAccessorySlotState();
}

class _GfInputAccessorySlotState extends State<GfInputAccessorySlot>
    with SingleTickerProviderStateMixin {
  /// Insets unchanged for this long belong to a settled keyboard.
  static const Duration _settle = Duration(milliseconds: 160);

  /// Upper bound for holding the panel when no software keyboard appears,
  /// e.g. with a hardware keyboard or a refused focus request.
  static const Duration _holdLimit = Duration(milliseconds: 600);

  /// Shorter insets are candidate strips or floating keyboards, not a size
  /// worth matching.
  static const double _minKeyboard = 120;

  late final AnimationController _reveal = AnimationController(
    vsync: this,
    duration: GfMotion.layout,
    reverseDuration: GfMotion.content,
    value: widget.accessory == GfInputAccessory.panel ? 1 : 0,
  )..addStatusListener(_revealStatusChanged);
  late final CurvedAnimation _revealCurve = CurvedAnimation(
    parent: _reveal,
    curve: GfMotion.enterCurve,
    reverseCurve: GfMotion.exitCurve,
  );
  final Map<Orientation, double> _settledKeyboard = <Orientation, double>{};
  double _inset = 0;
  bool _holding = false;

  /// Minimum height kept for a reserved keyboard, until one covers it again.
  double _reserve = 0;
  late bool _panelMounted = widget.accessory == GfInputAccessory.panel;
  Timer? _settleTimer;
  Timer? _holdTimer;
  Timer? _reserveTimer;

  @override
  void initState() {
    super.initState();
    widget.panelFocusNode.addListener(_panelFocusChanged);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final double inset = MediaQuery.viewInsetsOf(context).bottom;
    if (inset == _inset) return;
    _inset = inset;
    _settleTimer?.cancel();
    _settleTimer = Timer(_settle, _keyboardSettled);
    if (_holding && inset >= _panelHeight() - .5) _release(animate: false);
    if (widget.accessory != GfInputAccessory.reserved &&
        _reserve > 0 &&
        inset >= _reserve - .5) {
      _clearReserve();
    }
  }

  @override
  void didUpdateWidget(covariant GfInputAccessorySlot oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.panelFocusNode != widget.panelFocusNode) {
      oldWidget.panelFocusNode.removeListener(_panelFocusChanged);
      widget.panelFocusNode.addListener(_panelFocusChanged);
    }
    final GfInputAccessory from = oldWidget.accessory;
    final GfInputAccessory to = widget.accessory;
    if (from == to) return;
    if (to == GfInputAccessory.reserved) {
      _reserveTimer?.cancel();
      _reserve = math.max(_reserve, _inset);
      return;
    }
    if (from == GfInputAccessory.reserved) {
      // A returning keyboard fills the reserve; anything else releases it.
      if (to == GfInputAccessory.keyboard && _inset < _reserve - .5) {
        _reserveTimer = Timer(_holdLimit, () {
          if (mounted) setState(_clearReserve);
        });
      } else {
        _clearReserve();
      }
    }
    if (to == GfInputAccessory.panel) {
      _cancelHold();
      _panelMounted = true;
      final double height = _panelHeight();
      // Under a visible keyboard the panel is revealed by the keyboard itself.
      if (GfMotion.reducedOf(context) ||
          (_inset > 0 && _inset >= height - .5)) {
        _reveal.value = 1;
      } else if (_inset > 0) {
        // A panel taller than the keyboard rises from the keyboard's edge.
        _reveal.value = _revealAt(_inset / height);
        _reveal.forward();
      } else {
        _reveal.forward();
      }
    } else if (from == GfInputAccessory.panel &&
        to == GfInputAccessory.keyboard) {
      _reveal.stop();
      if (_inset >= _panelHeight() - .5) {
        _reveal.value = 0;
      } else {
        _holding = true;
        _holdTimer = Timer(_holdLimit, () => _release(animate: true));
      }
    } else if (from == GfInputAccessory.panel || _holding) {
      _release(animate: true);
    }
  }

  @override
  void dispose() {
    widget.panelFocusNode.removeListener(_panelFocusChanged);
    _settleTimer?.cancel();
    _holdTimer?.cancel();
    _reserveTimer?.cancel();
    _revealCurve.dispose();
    _reveal.dispose();
    super.dispose();
  }

  void _panelFocusChanged() {
    if (mounted) setState(() {});
  }

  void _revealStatusChanged(AnimationStatus status) {
    if (status == AnimationStatus.dismissed && mounted) setState(() {});
  }

  void _keyboardSettled() {
    if (!mounted) return;
    if (_inset >= _minKeyboard) {
      _settledKeyboard[MediaQuery.orientationOf(context)] = _inset;
    }
    if (_holding && _inset > 0) {
      _release(animate: true);
    } else {
      if (widget.accessory == GfInputAccessory.keyboard && _inset > 0) {
        _clearReserve();
      }
      setState(() {});
    }
  }

  void _clearReserve() {
    _reserve = 0;
    _reserveTimer?.cancel();
    _reserveTimer = null;
  }

  void _cancelHold() {
    _holding = false;
    _holdTimer?.cancel();
    _holdTimer = null;
  }

  void _release({required bool animate}) {
    if (!mounted) return;
    _cancelHold();
    if (animate && !GfMotion.reducedOf(context)) {
      _reveal.reverse();
    } else {
      _reveal.value = 0;
    }
    setState(() {});
  }

  /// The reveal value whose curved extent is [fraction] of the panel.
  double _revealAt(double fraction) {
    double low = 0, high = 1;
    for (int i = 0; i < 24; i++) {
      final double mid = (low + high) / 2;
      if (GfMotion.enterCurve.transform(mid) < fraction) {
        low = mid;
      } else {
        high = mid;
      }
    }
    return high;
  }

  /// Panel height measured from the screen bottom, safe area included, like
  /// the keyboard it replaces.
  double _panelHeight() {
    final Size size = MediaQuery.sizeOf(context);
    final EdgeInsets padding = MediaQuery.viewPaddingOf(context);
    final double fallback =
        ((size.height - padding.vertical) * .36).clamp(80.0, 300.0) +
        padding.bottom;
    final double preferred = math.max(
      _settledKeyboard[MediaQuery.orientationOf(context)] ?? fallback,
      widget.minPanelHeight + padding.bottom,
    );
    return math.min(preferred, math.max(80.0, widget.availableHeight * .55));
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final double safeBottom = MediaQuery.viewPaddingOf(context).bottom;
    final double floor = math.max(_inset, safeBottom);
    final bool open = widget.accessory == GfInputAccessory.panel;
    final bool visible = open || _holding || _reveal.value > 0;
    // A focused field inside the panel raises the keyboard below the panel.
    final bool searching =
        open &&
        widget.panelFocusNode.hasFocus &&
        !widget.panelFocusNode.hasPrimaryFocus;
    final double panelHeight = searching
        ? math.min(
            _panelHeight(),
            math.max(0.0, (widget.availableHeight - floor) * .3),
          )
        : _panelHeight();
    final Widget? panel = _panelMounted
        ? Offstage(
            offstage: !visible,
            child: TickerMode(
              enabled: visible,
              child: OverflowBox(
                alignment: Alignment.topCenter,
                minHeight: panelHeight,
                maxHeight: panelHeight,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    border: Border(top: BorderSide(color: colors.line)),
                  ),
                  child: Padding(
                    padding: EdgeInsets.only(
                      bottom: searching ? 0 : safeBottom,
                    ),
                    child: Center(
                      child: ConstrainedBox(
                        constraints: BoxConstraints(
                          maxWidth: widget.maxPanelWidth,
                        ),
                        child: widget.panelBuilder(context),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          )
        : null;
    return Focus(
      focusNode: widget.panelFocusNode,
      skipTraversal: true,
      child: AnimatedBuilder(
        animation: _revealCurve,
        child: panel,
        builder: (context, child) {
          final double extent = panelHeight * _revealCurve.value;
          return ColoredBox(
            key: const Key('input-accessory-slot'),
            color: colors.base100,
            child: SizedBox(
              width: double.infinity,
              height: searching
                  ? extent + floor
                  : math.max(math.max(floor, _reserve), visible ? extent : 0),
              child: ClipRect(child: child),
            ),
          );
        },
      ),
    );
  }
}

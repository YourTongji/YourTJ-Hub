import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/physics.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';

enum LogoRefreshPhase { idle, pulling, armed, refreshing, settling }

/// Pull-to-refresh that lives in the header instead of over the content.
///
/// The finger stretches the header and draws a sheen onto the centre logo;
/// past the threshold the sheen reaches the middle of the mark. Released
/// there, the header holds a little taller while the sheen keeps sweeping,
/// then folds back once the feed has refreshed. Anything shorter springs back
/// the way it came.
class LogoRefreshController extends ChangeNotifier {
  LogoRefreshController({required TickerProvider vsync})
    : _stretch = AnimationController.unbounded(vsync: vsync) {
    _stretch.addListener(_syncShift);
    _overscroll.addListener(_syncShift);
  }

  /// Finger travel that arms a refresh; matches the app's other pulls.
  static const double pullDistance = 72;
  static const double _maxStretch = 120;
  static const double _holdStretch = 30;
  static const Duration _minimumSweep = Duration(milliseconds: 760);
  static final SpringDescription _spring = SpringDescription.withDampingRatio(
    mass: 1,
    stiffness: 380,
    ratio: 1,
  );

  /// Header growth for a given finger travel: one-to-one at first, then
  /// increasingly resistant.
  static double stretchFor(double travel) =>
      _maxStretch * (1 - 1 / (travel / _maxStretch + 1));

  static final double armedStretch = stretchFor(pullDistance);

  final AnimationController _stretch;
  final ValueNotifier<double> _overscroll = ValueNotifier(0);
  final ValueNotifier<double> _shift = ValueNotifier(0);
  final ValueNotifier<LogoRefreshPhase> _phase = ValueNotifier(
    LogoRefreshPhase.idle,
  );

  /// Extra header height.
  ValueListenable<double> get stretch => _stretch;

  /// How far the content must move to stay under the header; whatever the
  /// scroll view already overscrolls by itself (iOS bounce) is subtracted.
  ValueListenable<double> get contentShift => _shift;

  ValueListenable<LogoRefreshPhase> get phase => _phase;

  bool get busy =>
      _phase.value == LogoRefreshPhase.refreshing ||
      _phase.value == LogoRefreshPhase.settling;

  set overscroll(double value) => _overscroll.value = math.max(0, value);

  void _syncShift() =>
      _shift.value = math.max(0, _stretch.value - _overscroll.value);

  void _springTo(double target, {required bool reduced}) {
    if (reduced) {
      _stretch.value = target;
      return;
    }
    unawaited(
      _stretch.animateWith(
        SpringSimulation(_spring, _stretch.value, target, _stretch.velocity),
      ),
    );
  }

  /// Follows the finger while the reader pulls from the top.
  void pull(double travel) {
    if (busy) return;
    _stretch.stop();
    _stretch.value = stretchFor(travel);
    final bool armed = travel >= pullDistance;
    final LogoRefreshPhase next = armed
        ? LogoRefreshPhase.armed
        : LogoRefreshPhase.pulling;
    if (next == _phase.value) return;
    if (armed) {
      HapticFeedback.mediumImpact();
    } else if (_phase.value == LogoRefreshPhase.armed) {
      HapticFeedback.selectionClick();
    }
    _phase.value = next;
  }

  /// Drops a pull whose gesture was lost, without refreshing.
  void cancel({required bool reduced}) {
    if (_disposed || busy) return;
    _overscroll.value = 0;
    _phase.value = LogoRefreshPhase.idle;
    _springTo(0, reduced: reduced);
  }

  /// Ends a pull: refreshes when armed, otherwise springs back.
  void release(Future<void> Function() onRefresh, {required bool reduced}) {
    if (busy) return;
    if (_phase.value != LogoRefreshPhase.armed) {
      _phase.value = LogoRefreshPhase.idle;
      _springTo(0, reduced: reduced);
      return;
    }
    unawaited(_run(onRefresh, reduced: reduced));
  }

  Future<void> _run(
    Future<void> Function() onRefresh, {
    required bool reduced,
  }) async {
    _phase.value = LogoRefreshPhase.refreshing;
    _springTo(_holdStretch, reduced: reduced);
    final Stopwatch elapsed = Stopwatch()..start();
    try {
      await onRefresh();
    } catch (_) {
      // The page reports its own failures; the header only has to settle.
    }
    // A fast network still shows one full sweep, so a refresh never flickers.
    final Duration rest = _minimumSweep - elapsed.elapsed;
    if (rest > Duration.zero && !reduced) await Future<void>.delayed(rest);
    if (_disposed) return;
    _phase.value = LogoRefreshPhase.settling;
    _springTo(0, reduced: reduced);
    if (!reduced) {
      await Future<void>.delayed(GfMotion.layout);
      if (_disposed) return;
    }
    if (_phase.value == LogoRefreshPhase.settling) {
      _phase.value = LogoRefreshPhase.idle;
    }
  }

  bool _disposed = false;

  @override
  void dispose() {
    _disposed = true;
    _stretch.dispose();
    _overscroll.dispose();
    _shift.dispose();
    _phase.dispose();
    super.dispose();
  }
}

/// Feeds a scroll view's top-edge pulls to a [LogoRefreshController]. It
/// measures finger travel rather than rubber-band displacement, so a pull
/// is as long on iOS as on Android, and it takes over the platform's top
/// overscroll effect, which the stretching header replaces.
class LogoPullRefresh extends StatefulWidget {
  const LogoPullRefresh({
    super.key,
    required this.controller,
    required this.onRefresh,
    required this.child,
    this.enabled = true,
  });

  final LogoRefreshController controller;
  final Future<void> Function() onRefresh;
  final Widget child;
  final bool enabled;

  @override
  State<LogoPullRefresh> createState() => _LogoPullRefreshState();
}

class _LogoPullRefreshState extends State<LogoPullRefresh> {
  bool _tracking = false;
  double _travel = 0;
  bool _reduced = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _reduced = GfMotion.reducedOf(context);
  }

  @override
  void dispose() {
    // The feed can unmount mid-pull, as when the cache is cleared; the
    // controller outlives it and must not stay stretched or armed. The header
    // it drives cannot rebuild while this tree is being torn down.
    if (_tracking) {
      final LogoRefreshController controller = widget.controller;
      final bool reduced = _reduced;
      WidgetsBinding.instance.addPostFrameCallback(
        (_) => controller.cancel(reduced: reduced),
      );
    }
    super.dispose();
  }

  void _release() {
    if (!_tracking) return;
    _tracking = false;
    _travel = 0;
    widget.controller.release(
      widget.onRefresh,
      reduced: GfMotion.reducedOf(context),
    );
  }

  bool _onScroll(ScrollNotification notification) {
    if (!widget.enabled ||
        notification.depth != 0 ||
        notification.metrics.axisDirection != AxisDirection.down) {
      return false;
    }
    final LogoRefreshController controller = widget.controller;
    controller.overscroll =
        notification.metrics.minScrollExtent - notification.metrics.pixels;
    if (notification is ScrollStartNotification) {
      _tracking =
          notification.dragDetails != null &&
          notification.metrics.extentBefore == 0 &&
          !controller.busy;
      _travel = 0;
    } else if (_tracking) {
      if (notification is ScrollEndNotification) {
        _release();
      } else {
        final DragUpdateDetails? details = switch (notification) {
          ScrollUpdateNotification() => notification.dragDetails,
          OverscrollNotification() => notification.dragDetails,
          _ => null,
        };
        if (details != null) {
          _travel = math.max(0, _travel + details.delta.dy);
          if (_travel == 0 && notification.metrics.extentBefore > 0) {
            // Scrolled into the feed instead: hand the gesture back.
            _release();
          } else {
            controller.pull(_travel);
          }
        } else if (notification is ScrollUpdateNotification ||
            notification is OverscrollNotification) {
          // iOS begins its ballistic return before emitting ScrollEnd.
          _release();
        }
      }
    }
    return false;
  }

  @override
  Widget build(BuildContext context) =>
      NotificationListener<OverscrollIndicatorNotification>(
        onNotification: (OverscrollIndicatorNotification notification) {
          if (widget.enabled && notification.leading) {
            notification.disallowIndicator();
          }
          return false;
        },
        child: NotificationListener<ScrollNotification>(
          onNotification: _onScroll,
          child: widget.child,
        ),
      );
}

/// The header mark for a [LogoRefreshController]: it grows a little with the
/// header, and a sheen the finger draws in keeps sweeping while refreshing.
class RefreshingLogo extends StatefulWidget {
  const RefreshingLogo({super.key, required this.controller, this.size = 32});

  final LogoRefreshController controller;
  final double size;

  @override
  State<RefreshingLogo> createState() => _RefreshingLogoState();
}

class _RefreshingLogoState extends State<RefreshingLogo>
    with TickerProviderStateMixin {
  // 0 and 1 park the sheen just off the mark; 0.5 crosses its centre.
  late final AnimationController _sweep = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1100),
  );
  late final AnimationController _pop = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 260),
  );
  late Listenable _motion = _listenables();

  Listenable _listenables() => Listenable.merge(<Listenable>[
    widget.controller.stretch,
    widget.controller.phase,
    _sweep,
    _pop,
  ]);

  @override
  void initState() {
    super.initState();
    widget.controller.phase.addListener(_onPhase);
  }

  @override
  void didUpdateWidget(RefreshingLogo oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller.phase.removeListener(_onPhase);
      widget.controller.phase.addListener(_onPhase);
      _motion = _listenables();
    }
  }

  void _onPhase() {
    final bool reduced = GfMotion.reducedOf(context);
    switch (widget.controller.phase.value) {
      case LogoRefreshPhase.armed:
        if (!reduced) _pop.forward(from: 0);
      case LogoRefreshPhase.refreshing:
        if (reduced) break;
        // Carry on from where the finger left the sheen, then loop.
        _sweep.value = _pulledSheen;
        unawaited(
          _sweep
              .animateTo(1, duration: _sweep.duration! * (1 - _sweep.value))
              .whenComplete(() {
                if (mounted &&
                    widget.controller.phase.value ==
                        LogoRefreshPhase.refreshing) {
                  _sweep.repeat();
                }
              }),
        );
      case LogoRefreshPhase.settling:
        // Let the current pass leave the mark instead of cutting it off.
        if (_sweep.isAnimating) {
          unawaited(
            _sweep
                .animateTo(1, duration: _sweep.duration! * (1 - _sweep.value))
                .whenComplete(() => _sweep.value = 0),
          );
        }
      case LogoRefreshPhase.idle:
      case LogoRefreshPhase.pulling:
        break;
    }
  }

  double get _pulledSheen =>
      0.5 *
      (widget.controller.stretch.value / LogoRefreshController.armedStretch)
          .clamp(0.0, 1.0);

  @override
  void dispose() {
    widget.controller.phase.removeListener(_onPhase);
    _sweep.dispose();
    _pop.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool reduced = GfMotion.reducedOf(context);
    // A luminance sheen: light across the dark mark, dark across the light.
    final Color sheen = colors.isDark
        ? Colors.black.withValues(alpha: 0.5)
        : Colors.white.withValues(alpha: 0.72);
    final Widget logo = GfLogo(size: widget.size);
    final String loading = AppLocalizations.of(context).commonLoading;
    return AnimatedBuilder(
      animation: _motion,
      child: logo,
      builder: (BuildContext context, Widget? logo) {
        final LogoRefreshPhase phase = widget.controller.phase.value;
        final bool sweeping =
            phase == LogoRefreshPhase.refreshing ||
            (phase == LogoRefreshPhase.settling && _sweep.value > 0);
        final double position = reduced
            ? (phase == LogoRefreshPhase.refreshing ? 0.5 : _pulledSheen)
            : sweeping
            ? _sweep.value
            : _pulledSheen;
        final double grow =
            0.1 *
            (widget.controller.stretch.value /
                    LogoRefreshController.armedStretch)
                .clamp(0.0, 1.3);
        final double pop = math.sin(math.pi * _pop.value) * 0.07;
        Widget mark = logo!;
        // A spring comes to rest within a hair of its target; skip the
        // layer once the sheen is effectively off the mark.
        if (position > 0.005 && position < 0.995) {
          // Map 0..1 so both ends sit fully outside the mark.
          const double band = 0.22;
          final double centre = -band + position * (1 + band * 2);
          mark = ShaderMask(
            blendMode: BlendMode.srcATop,
            shaderCallback: (Rect bounds) => LinearGradient(
              begin: Alignment.bottomLeft,
              end: Alignment.topRight,
              colors: <Color>[
                sheen.withValues(alpha: 0),
                sheen.withValues(alpha: sheen.a * 0.45),
                sheen,
                sheen.withValues(alpha: sheen.a * 0.45),
                sheen.withValues(alpha: 0),
              ],
              stops: <double>[
                centre - band,
                centre - band / 2,
                centre,
                centre + band / 2,
                centre + band,
              ],
            ).createShader(bounds),
            child: mark,
          );
        }
        return Semantics(
          liveRegion: true,
          label: phase == LogoRefreshPhase.refreshing ? loading : null,
          child: Transform.scale(scale: 1 + grow + pop, child: mark),
        );
      },
    );
  }
}

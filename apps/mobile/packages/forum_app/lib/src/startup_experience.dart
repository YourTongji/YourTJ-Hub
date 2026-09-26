import 'dart:async';

import 'package:animated_text_effects/animated_text_effects.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:ui_kit/ui_kit.dart';

/// Keeps the native launch mark continuous while the real app warms underneath.
class StartupExperience extends StatefulWidget {
  const StartupExperience({super.key, required this.child});

  final Widget child;

  @override
  State<StartupExperience> createState() => _StartupExperienceState();
}

class _StartupExperienceState extends State<StartupExperience>
    with TickerProviderStateMixin {
  static const _brandDuration = Duration(milliseconds: 800);
  static const _exitDuration = Duration(milliseconds: 200);
  static const _captionDelay = Duration(milliseconds: 450);
  static const _staticLogoAsset = 'assets/splash/brand_mark.svg';
  static const _staticLogoFallbackAsset = 'assets/splash/brand_mark.png';
  static const _logoAsset = 'assets/splash/logo_motion.webp';
  static const _lightCaptionColor = Color(0xFF536568);
  static const _darkCaptionColor = Color(0xFFAAB9BA);
  static const _lightCaptionShimmer = Color(0xFF008E8B);
  static const _darkCaptionShimmer = Color(0xFF75E3D7);
  static const _lightCaptionEffects = <ShimmerEffect>[
    ShimmerEffect(
      duration: Duration(milliseconds: 700),
      baseColor: _lightCaptionColor,
      highlightColor: _lightCaptionShimmer,
      width: 0.24,
    ),
  ];
  static const _darkCaptionEffects = <ShimmerEffect>[
    ShimmerEffect(
      duration: Duration(milliseconds: 700),
      baseColor: _darkCaptionColor,
      highlightColor: _darkCaptionShimmer,
      width: 0.24,
    ),
  ];

  late final AnimationController _brandController;
  late final AnimationController _exitController;
  late final Animation<double> _brandProgress;
  late final Animation<double> _exitProgress;
  late final Listenable _animation;
  Timer? _captionTimer;
  Timer? _startupTimeoutTimer;
  bool _initialized = false;
  bool _sequenceStarted = false;
  bool _reducedMotion = false;
  bool _captionVisible = false;
  bool _showStaticLogo = true;
  bool _leaving = false;
  bool _finished = false;

  @override
  void initState() {
    super.initState();
    _brandController = AnimationController(vsync: this);
    _exitController = AnimationController(vsync: this, duration: _exitDuration);
    _brandProgress = CurvedAnimation(
      parent: _brandController,
      curve: Curves.easeOutCubic,
    );
    _exitProgress = CurvedAnimation(
      parent: _exitController,
      curve: Curves.easeInOutCubic,
    );
    _animation = Listenable.merge([_brandController, _exitController]);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_initialized) return;
    _initialized = true;

    final mobile =
        !kIsWeb &&
        (defaultTargetPlatform == TargetPlatform.android ||
            defaultTargetPlatform == TargetPlatform.iOS);
    _reducedMotion = WidgetsBinding
        .instance
        .platformDispatcher
        .accessibilityFeatures
        .disableAnimations;
    if (!mobile) {
      _finished = true;
      return;
    }

    // Decode the animated mark while the native splash is still covering the
    // first Flutter frames so the handoff does not pause on its placeholder.
    unawaited(
      precacheImage(const AssetImage(_logoAsset), context, onError: (_, _) {}),
    );
    _brandController.duration = _brandDuration;
    if (_reducedMotion) {
      _captionVisible = true;
      _finished = true;
    }
  }

  void _startSequenceAfterVisibleLayout() {
    if (_sequenceStarted || _finished || _reducedMotion) return;
    _sequenceStarted = true;
    _startupTimeoutTimer = Timer(
      const Duration(milliseconds: 1200),
      _skipStartupExperience,
    );
    _captionTimer = Timer(_captionDelay, () {
      if (mounted && !_leaving) setState(() => _captionVisible = true);
    });
    unawaited(_runSequence());
  }

  Future<void> _runSequence() async {
    try {
      await Future<void>.delayed(const Duration(milliseconds: 100));
      if (!mounted) return;
      setState(() => _showStaticLogo = false);
      await _brandController.forward().orCancel;
      if (!mounted) return;
      _captionTimer?.cancel();
      setState(() => _leaving = true);
      await _exitController.forward().orCancel;
      if (mounted) {
        _startupTimeoutTimer?.cancel();
        setState(() => _finished = true);
      }
    } on TickerCanceled {
      // Disposal during a route teardown ends the decorative sequence.
    }
  }

  void _skipStartupExperience() {
    if (!mounted || _finished) return;
    _captionTimer?.cancel();
    _brandController.stop();
    _exitController.stop();
    setState(() {
      _leaving = true;
      _finished = true;
    });
  }

  @override
  void dispose() {
    _captionTimer?.cancel();
    _startupTimeoutTimer?.cancel();
    _brandController.dispose();
    _exitController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // This widget lives inside MaterialApp.builder so its brightness is the
    // same resolved theme used by the app and updates with system changes.
    final dark = Theme.of(context).brightness == Brightness.dark;
    final background = dark ? const Color(0xFF0B1112) : const Color(0xFFF7FBFA);
    final foreground = dark ? Brightness.light : Brightness.dark;
    final reveal = _finished
        ? const AlwaysStoppedAnimation<double>(1)
        : _exitProgress;

    return Directionality(
      textDirection: TextDirection.ltr,
      child: AbsorbPointer(
        absorbing: !_leaving && !_finished,
        child: Stack(
          fit: StackFit.expand,
          children: [
            FadeTransition(
              opacity: reveal,
              child: SlideTransition(
                position: Tween<Offset>(
                  begin: const Offset(0, 0.018),
                  end: Offset.zero,
                ).animate(reveal),
                child: TickerMode(
                  enabled: _leaving || _finished,
                  child: widget.child,
                ),
              ),
            ),
            if (!_finished)
              Positioned.fill(
                child: AnnotatedRegion<SystemUiOverlayStyle>(
                  value: SystemUiOverlayStyle(
                    statusBarColor: background,
                    statusBarIconBrightness: foreground,
                    statusBarBrightness: dark
                        ? Brightness.dark
                        : Brightness.light,
                    systemNavigationBarColor: background,
                    systemNavigationBarIconBrightness: foreground,
                    systemNavigationBarDividerColor: background,
                  ),
                  child: AnimatedBuilder(
                    animation: _animation,
                    builder: (context, _) =>
                        _buildBrandLayer(background: background, dark: dark),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildBrandLayer({required Color background, required bool dark}) {
    final exit = _exitProgress.value;
    final enter = _brandProgress.value;
    final opacity = 1 - exit;
    final centerFactor = 0.5 - 0.07 * enter;
    final glow = dark ? const Color(0xFF55AFA5) : const Color(0xFF69AAA2);
    final glowOpacity = 0.08 * enter * opacity;
    final logoScale = (0.96 + 0.04 * enter) * (1 - 0.06 * exit);

    return Opacity(
      opacity: opacity,
      child: AnimatedContainer(
        color: background,
        duration: const Duration(milliseconds: 180),
        curve: Curves.easeOut,
        child: LayoutBuilder(
          builder: (context, constraints) {
            if (constraints.maxWidth > 0 && constraints.maxHeight > 0) {
              _startSequenceAfterVisibleLayout();
            }
            const logoSize = 128.0;
            const glowSize = 240.0;
            final centerY = constraints.maxHeight * centerFactor;
            final logoTop = centerY - logoSize / 2;
            final glowTop = centerY - glowSize / 2;
            final glowLeft = (constraints.maxWidth - glowSize) / 2;
            final safeLogoTop = logoTop.clamp(0.0, constraints.maxHeight);

            return Stack(
              clipBehavior: Clip.none,
              children: [
                Positioned(
                  left: glowLeft,
                  top: glowTop,
                  width: glowSize,
                  height: glowSize,
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: glow.withValues(alpha: glowOpacity * 0.35),
                      boxShadow: [
                        BoxShadow(
                          color: glow.withValues(alpha: glowOpacity * 0.55),
                          blurRadius: 48,
                          spreadRadius: 10,
                        ),
                      ],
                    ),
                  ),
                ),
                Positioned(
                  left: 0,
                  right: 0,
                  top: safeLogoTop,
                  child: Transform.translate(
                    offset: Offset(0, -12 * exit),
                    child: Transform.scale(
                      scale: logoScale,
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          AnimatedSwitcher(
                            duration: const Duration(milliseconds: 120),
                            child: _showStaticLogo
                                ? _buildStaticLogo(logoSize)
                                : Image.asset(
                                    _logoAsset,
                                    width: logoSize,
                                    height: logoSize,
                                    fit: BoxFit.contain,
                                    gaplessPlayback: true,
                                    frameBuilder:
                                        (
                                          context,
                                          child,
                                          frame,
                                          wasSynchronouslyLoaded,
                                        ) {
                                          if (wasSynchronouslyLoaded ||
                                              frame != null) {
                                            return child;
                                          }
                                          return _buildStaticLogo(logoSize);
                                        },
                                    excludeFromSemantics: true,
                                  ),
                          ),
                          const SizedBox(height: 30),
                          AnimatedOpacity(
                            opacity: _captionVisible ? 1 : 0,
                            duration: const Duration(milliseconds: 170),
                            curve: Curves.easeOut,
                            child: SizedBox(
                              height: 24,
                              child: _buildCaption(dark: dark),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }

  Widget _buildStaticLogo(double size) => SvgPicture.asset(
    _staticLogoAsset,
    width: size,
    height: size,
    fit: BoxFit.contain,
    placeholderBuilder: (_) => Image.asset(
      _staticLogoFallbackAsset,
      width: size,
      height: size,
      fit: BoxFit.contain,
      excludeFromSemantics: true,
    ),
    excludeFromSemantics: true,
  );

  Widget _buildCaption({required bool dark}) {
    final baseColor = dark ? _darkCaptionColor : _lightCaptionColor;
    final textStyle = GfTheme.typographyOf(context).body.copyWith(
      color: baseColor,
      fontSize: 16,
      fontWeight: FontWeight.w500,
      height: 1.4,
      letterSpacing: 0.64,
      decoration: TextDecoration.none,
    );

    if (_reducedMotion || !_captionVisible) {
      return Text('未济非终，皆有可能', style: textStyle, textAlign: TextAlign.center);
    }

    return AnimatedText(
      '未济非终，皆有可能',
      effects: dark ? _darkCaptionEffects : _lightCaptionEffects,
      style: textStyle,
      textAlign: TextAlign.center,
      repeat: true,
      keepAlive: false,
    );
  }
}

import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../pages/publish/publish_type.dart';

/// A modal first level above the FAB. The route owns dismissal and focus, so
/// taps cannot reach the feed or retained destination underneath it.
Future<void> showComposeMenu(
  BuildContext context, {
  required double bottom,
  Future<void> Function(PublishType type)? onCompose,
}) async {
  final l10n = AppLocalizations.of(context);
  final source = context.findRenderObject();
  final window = MediaQuery.of(
    Navigator.of(context, rootNavigator: true).context,
  );
  final sourceRight = source is RenderBox && source.hasSize
      ? source.localToGlobal(Offset(source.size.width, 0)).dx
      : window.size.width;
  // The dialog belongs to the root navigator; anchor it to the reading column
  // that opened it, even when the window has gutters and a navigation rail.
  // SafeArea below already consumes the root window's right inset. Measure
  // only the remaining gutter to the source's safe content edge.
  final rightInset =
      (window.size.width -
              window.padding.right -
              sourceRight +
              MediaQuery.paddingOf(context).right)
          .clamp(0.0, window.size.width);
  final options = PublishType.values;
  final type = await showGeneralDialog<PublishType>(
    context: context,
    barrierDismissible: true,
    barrierLabel: l10n.commonClose,
    // [_FrostedScrim] draws the dimming; the barrier keeps dismissal.
    barrierColor: Colors.transparent,
    transitionDuration: GfMotion.duration(context, GfMotion.layout),
    transitionBuilder: (context, animation, secondary, child) => Stack(
      children: [
        Positioned.fill(child: _FrostedScrim(animation: animation)),
        child,
      ],
    ),
    pageBuilder: (context, animation, secondary) => SafeArea(
      child: Align(
        alignment: Alignment.bottomRight,
        child: Padding(
          padding: EdgeInsets.fromLTRB(24, 16, 16 + rightInset, bottom),
          child: ConstrainedBox(
            key: const ValueKey('compose-menu'),
            constraints: const BoxConstraints(maxWidth: 320),
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  for (final (index, option) in options.indexed)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: _RiseFromAction(
                        animation: animation,
                        // The item nearest the action arrives first.
                        order: options.length - 1 - index,
                        child: GfGlassButton(
                          key: ValueKey('compose-${option.value}'),
                          autofocus: option == PublishType.moment,
                          minSize: const Size(0, 52),
                          padding: const EdgeInsets.fromLTRB(20, 10, 12, 10),
                          onPressed: () => Navigator.pop(context, option),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Flexible(
                                child: GfText(
                                  option.label(l10n),
                                  style: GfTheme.typographyOf(
                                    context,
                                  ).bodyStrong,
                                ),
                              ),
                              const SizedBox(width: 20),
                              PublishTypeIcon(option, size: 32),
                            ],
                          ),
                        ),
                      ),
                    ),
                  // Sits exactly over the action that opened the menu, whose
                  // plus turns into this close mark.
                  GfGlassButton(
                    tone: GfGlassTone.prominent,
                    tooltip: l10n.commonClose,
                    onPressed: () => Navigator.pop(context),
                    child: _PlusToClose(animation: animation),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    ),
  );
  if (type != null && context.mounted) {
    if (onCompose != null) {
      await onCompose(type);
    } else {
      context.push('/publish?type=${type.value}');
    }
  }
}

/// Dims the page and, on glass, frosts it, so the menu's glass reads against
/// a softened backdrop instead of busy content.
class _FrostedScrim extends StatelessWidget {
  const _FrostedScrim({required this.animation});

  final Animation<double> animation;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final glass = GfNavigationSurface.usesGlass(context);
    return IgnorePointer(
      child: AnimatedBuilder(
        animation: animation,
        builder: (context, _) {
          final t = animation.value;
          final dim = ColoredBox(
            color: glass
                // The frost already separates the menu; dim less than a sheet.
                ? Colors.black.withValues(
                    alpha: (colors.isDark ? 0.5 : 0.2) * t,
                  )
                : colors.scrim.withValues(alpha: colors.scrim.a * t),
            child: const SizedBox.expand(),
          );
          if (!glass || t == 0) return dim;
          return BackdropFilter(
            filter: ImageFilter.blur(sigmaX: 8 * t, sigmaY: 8 * t),
            child: dim,
          );
        },
      ),
    );
  }
}

/// Menu items grow out of the action below them: each rises a little and
/// scales up from its trailing bottom corner, nearest first. They leave
/// together, softer than they arrived.
class _RiseFromAction extends StatelessWidget {
  const _RiseFromAction({
    required this.animation,
    required this.order,
    required this.child,
  });

  final Animation<double> animation;
  final int order;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final start = math.min(0.4, order * 0.12);
    final enter = Interval(
      start,
      math.min(1, start + 0.6),
      curve: GfMotion.enterCurve,
    );
    return AnimatedBuilder(
      animation: animation,
      builder: (context, child) {
        final leaving = animation.status == AnimationStatus.reverse;
        final t = leaving
            ? Curves.easeOutCubic.transform(animation.value)
            : enter.transform(animation.value);
        return Opacity(
          opacity: t.clamp(0.0, 1.0),
          child: Transform.translate(
            offset: Offset(0, (leaving ? 4 : 10) * (1 - t)),
            child: Transform.scale(
              scale: leaving ? 1 : 0.92 + 0.08 * t,
              alignment: AlignmentDirectional.bottomEnd.resolve(
                Directionality.of(context),
              ),
              child: child,
            ),
          ),
        );
      },
      child: child,
    );
  }
}

/// The opening action's plus turns a quarter of the way round into the close
/// mark, and back as the menu closes.
class _PlusToClose extends StatelessWidget {
  const _PlusToClose({required this.animation});

  final Animation<double> animation;

  @override
  Widget build(BuildContext context) {
    final color = GfTheme.colorsOf(context).primaryContent;
    return AnimatedBuilder(
      animation: animation,
      builder: (context, _) {
        final t = GfMotion.enterCurve.transform(animation.value.clamp(0, 1));
        return Stack(
          alignment: Alignment.center,
          children: [
            Opacity(
              opacity: 1 - t,
              child: Transform.rotate(
                angle: math.pi / 4 * t,
                child: GfSymbol('plus', size: 28, color: color),
              ),
            ),
            Opacity(
              opacity: t,
              child: Transform.rotate(
                angle: -math.pi / 4 * (1 - t),
                child: GfSymbol('x', size: 28, color: color),
              ),
            ),
          ],
        );
      },
    );
  }
}

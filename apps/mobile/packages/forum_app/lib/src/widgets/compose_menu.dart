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
  final window = MediaQuery.of(context);
  final sourceRight = source is RenderBox && source.hasSize
      ? source.localToGlobal(Offset(source.size.width, 0)).dx
      : window.size.width;
  // The dialog belongs to the root navigator; anchor it to the reading column
  // that opened it, even when the window has gutters and a navigation rail.
  final rightInset = (window.size.width - sourceRight).clamp(
    0.0,
    window.size.width,
  );
  final type = await showGeneralDialog<PublishType>(
    context: context,
    barrierDismissible: true,
    barrierLabel: l10n.commonClose,
    barrierColor: Theme.of(context).colorScheme.scrim,
    transitionDuration: MediaQuery.disableAnimationsOf(context)
        ? Duration.zero
        : const Duration(milliseconds: 180),
    transitionBuilder: (context, animation, secondary, child) => FadeTransition(
      opacity: animation,
      child: SlideTransition(
        position: Tween(
          begin: const Offset(0, .025),
          end: Offset.zero,
        ).animate(CurvedAnimation(parent: animation, curve: Curves.easeOut)),
        child: child,
      ),
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
                  for (final option in PublishType.values)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: Material(
                        color: GfTheme.colorsOf(context).base100,
                        borderRadius: BorderRadius.circular(18),
                        elevation: 2,
                        shadowColor: Colors.black.withValues(alpha: .15),
                        child: InkWell(
                          key: ValueKey('compose-${option.value}'),
                          autofocus: option == PublishType.moment,
                          borderRadius: BorderRadius.circular(18),
                          onTap: () => Navigator.pop(context, option),
                          child: Padding(
                            padding: const EdgeInsets.fromLTRB(20, 10, 12, 10),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Flexible(
                                  child: Text(
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
                    ),
                  FloatingActionButton(
                    heroTag: null,
                    tooltip: l10n.commonClose,
                    onPressed: () => Navigator.pop(context),
                    child: GfSymbol(
                      'x',
                      size: 28,
                      color: GfTheme.colorsOf(context).primaryContent,
                    ),
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

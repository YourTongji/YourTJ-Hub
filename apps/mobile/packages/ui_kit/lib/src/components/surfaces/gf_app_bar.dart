import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../gf_symbol.dart';

/// Compact, centered navigation shared with the unified mobile design.
///
/// The public surface deliberately mirrors the small subset of [AppBar] used
/// by the mobile app with repository-owned appearance and native navigation.
class GfAppBar extends StatelessWidget implements PreferredSizeWidget {
  const GfAppBar({
    super.key,
    required this.title,
    this.leading,
    this.actions = const <Widget>[],
    this.automaticallyImplyLeading = true,
    this.centerTitle = true,
    this.bottom,
    this.showDivider = true,
    this.backgroundColor,
    this.leadingWidth,
  });

  /// Hairline under the bar; off for pages whose first block is a cover or
  /// a grouped canvas.
  final bool showDivider;

  /// Defaults to the page canvas; grouped pages pass `canvasGrouped`.
  final Color? backgroundColor;

  /// Width reserved for [leading]; text actions such as Cancel need more
  /// than the default 56px.
  final double? leadingWidth;

  final Widget title;
  final Widget? leading;
  final List<Widget> actions;
  final bool automaticallyImplyLeading;
  final bool centerTitle;
  final PreferredSizeWidget? bottom;

  @override
  Size get preferredSize =>
      Size.fromHeight(56 + (bottom?.preferredSize.height ?? 0));

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool showDefaultBack =
        leading == null &&
        automaticallyImplyLeading &&
        Navigator.canPop(context);

    return AppBar(
      title: title,
      leading:
          leading ??
          (showDefaultBack
              ? IconButton(
                  tooltip: MaterialLocalizations.of(context).backButtonTooltip,
                  onPressed: () => Navigator.maybePop(context),
                  icon: const GfSymbol('chevron-left'),
                )
              : null),
      automaticallyImplyLeading: false,
      centerTitle: centerTitle,
      actions: actions,
      backgroundColor: backgroundColor ?? colors.canvas,
      titleTextStyle: GfTheme.typographyOf(context).title3,
      toolbarHeight: 56,
      leadingWidth: leadingWidth,
      bottom: bottom,
      shape: showDivider
          ? Border(
              bottom: BorderSide(
                color: colors.separator,
                width: gfHairline(context),
              ),
            )
          : null,
    );
  }
}

/// Text action for navigation bars: Cancel, Done, Save. [emphasized] marks
/// the confirming action with primary semibold text.
class GfNavTextAction extends StatelessWidget {
  const GfNavTextAction({
    super.key,
    required this.label,
    required this.onPressed,
    this.emphasized = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool emphasized;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return TextButton(
      onPressed: onPressed,
      style: TextButton.styleFrom(
        foregroundColor: emphasized ? colors.primary : colors.textPrimary,
        disabledForegroundColor: colors.textDisabled,
        padding: const EdgeInsets.symmetric(horizontal: GfSpace.md),
        minimumSize: const Size(GfControl.touch, GfControl.touch),
        textStyle: GfTheme.typographyOf(context).body.copyWith(
          fontWeight: emphasized ? FontWeight.w600 : FontWeight.w400,
        ),
      ),
      child: GfText(label, maxLines: 1),
    );
  }
}

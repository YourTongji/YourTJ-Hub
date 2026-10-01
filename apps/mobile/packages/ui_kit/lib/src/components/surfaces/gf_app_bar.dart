import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';
import 'gf_liquid_surface.dart';

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
  });

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

    Widget island(Widget child) => Center(
      child: GfLiquidSurface(
        radius: 24,
        elevated: false,
        pressable: true,
        child: child,
      ),
    );
    final back =
        leading ??
        (showDefaultBack
            ? IconButton(
                tooltip: MaterialLocalizations.of(context).backButtonTooltip,
                onPressed: () => Navigator.maybePop(context),
                icon: const GfSymbol('chevron-left'),
              )
            : null);
    return AppBar(
      title: title,
      titleSpacing: 8,
      leading: back == null
          ? null
          : Padding(
              padding: const EdgeInsets.only(left: 8),
              child: island(back),
            ),
      automaticallyImplyLeading: false,
      centerTitle: centerTitle,
      actions: actions.isEmpty
          ? null
          : [
              Padding(
                padding: const EdgeInsets.only(right: 8),
                child: island(
                  Row(mainAxisSize: MainAxisSize.min, children: actions),
                ),
              ),
            ],
      backgroundColor: colors.base100,
      surfaceTintColor: Colors.transparent,
      scrolledUnderElevation: 0,
      elevation: 0,
      titleTextStyle: Theme.of(context).textTheme.titleLarge?.copyWith(
        fontSize: 18,
        height: 26 / 18,
        fontWeight: FontWeight.w600,
      ),
      toolbarHeight: 56,
      bottom: bottom,
    );
  }
}

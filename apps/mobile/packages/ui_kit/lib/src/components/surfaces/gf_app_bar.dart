import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Compact, centered navigation shared with the unified mobile design.
///
/// The public surface deliberately mirrors the small subset of [AppBar] used
/// by the mobile app so pages do not depend on TDesign's pre-release API.
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

    return AppBar(
      title: title,
      leading: leading,
      automaticallyImplyLeading: showDefaultBack,
      centerTitle: centerTitle,
      actions: actions,
      backgroundColor: colors.base100,
      titleTextStyle: Theme.of(context).textTheme.titleLarge?.copyWith(
        fontSize: 18,
        height: 26 / 18,
        fontWeight: FontWeight.w700,
      ),
      toolbarHeight: 56,
      bottom: bottom,
      shape: Border(bottom: BorderSide(color: colors.line)),
    );
  }
}

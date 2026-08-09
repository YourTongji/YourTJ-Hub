import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

/// Application navigation bar backed by TDesign's [td.TNavBar].
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
    this.centerTitle = false,
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

    // Scaffold adds the status-bar inset to the app-bar slot but arbitrary
    // PreferredSizeWidget children do not consume it automatically. SafeArea
    // keeps TNavBar below the Dynamic Island/notch while preserving the 56px
    // content height used by Android and tests.
    return SafeArea(
      bottom: false,
      child: td.TNavBar(
        titleWidget: title,
        leading: leading == null
            ? null
            : <td.TNavBarItem>[
                td.TNavBarItem(customWidget: leading, onTap: () {}),
              ],
        actions: <td.TNavBarItem>[
          for (final Widget action in actions)
            td.TNavBarItem(customWidget: action, onTap: () {}),
        ],
        centerTitle: centerTitle,
        useDefaultBack: showDefaultBack,
        height: preferredSize.height,
        belowTitleWidget: bottom,
        backgroundColor: colors.base100,
        titleColor: colors.baseContent,
        backIconColor: colors.iconMuted,
        titleFontWeight: FontWeight.w700,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        boxShadow: <BoxShadow>[
          BoxShadow(color: colors.line, offset: const Offset(0, 1)),
        ],
      ),
    );
  }
}

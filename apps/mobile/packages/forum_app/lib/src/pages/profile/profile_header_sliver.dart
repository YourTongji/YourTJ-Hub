import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';

/// Shared collapsing cover, avatar and navigation for member/persona profiles.
class ProfileHeaderSliver extends StatelessWidget {
  const ProfileHeaderSliver({
    super.key,
    required this.avatarUrl,
    required this.title,
    this.coverUrl = '',
    this.avatarBadge,
    this.actions,
    this.actionHeightFor,
    this.navigationActions,
  });
  final String avatarUrl, coverUrl;
  final Widget title;
  final Widget? avatarBadge, actions;
  final double Function(BuildContext, double)? actionHeightFor;
  final List<Widget> Function(BuildContext, bool)? navigationActions;

  @override
  Widget build(BuildContext context) => SliverLayoutBuilder(
    builder: (context, constraints) {
      final colors = GfTheme.colorsOf(context);
      final l10n = AppLocalizations.of(context);
      final topInset = MediaQuery.paddingOf(context).top;
      final width = constraints.crossAxisExtent;
      final coverHeight = GfUserCard.coverHeightFor(width, topInset: topInset);
      bool expanded(BuildContext context) {
        final settings = context
            .dependOnInheritedWidgetOfExactType<FlexibleSpaceBarSettings>();
        return settings == null ||
            settings.currentExtent > settings.minExtent + 12;
      }

      final actionHeight =
          actionHeightFor?.call(context, width) ??
          GfUserCardHeader.actionHeightFor(MediaQuery.textScalerOf(context));
      return SliverAppBar(
        key: const Key('profile-cover-navigation'),
        pinned: true,
        expandedHeight: coverHeight + actionHeight - topInset,
        toolbarHeight: 56,
        elevation: 0,
        scrolledUnderElevation: 0,
        backgroundColor: colors.base100,
        foregroundColor: colors.baseContent,
        automaticallyImplyLeading: false,
        centerTitle: false,
        titleSpacing: Navigator.canPop(context) ? 0 : 16,
        titleTextStyle: TextStyle(
          color: colors.baseContent,
          fontSize: 18,
          fontWeight: FontWeight.w700,
        ),
        leadingWidth: 64,
        leading: Navigator.canPop(context)
            ? Builder(
                builder: (context) => Padding(
                  padding: const EdgeInsetsDirectional.only(
                    start: 12,
                    top: 6,
                    bottom: 6,
                  ),
                  child: expanded(context)
                      ? GfGlassIconButton(
                          symbol: 'arrow-left',
                          tooltip: l10n.commonBack,
                          onPressed: () => Navigator.maybePop(context),
                        )
                      : GfIconButton(
                          symbol: 'arrow-left',
                          tooltip: l10n.commonBack,
                          onPressed: () => Navigator.maybePop(context),
                        ),
                ),
              )
            : null,
        title: Builder(
          builder: (context) =>
              expanded(context) ? const SizedBox.shrink() : title,
        ),
        actions: [
          Builder(
            builder: (context) => Padding(
              padding: const EdgeInsetsDirectional.only(end: 12),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                spacing: 8,
                children:
                    navigationActions?.call(context, expanded(context)) ??
                    const <Widget>[],
              ),
            ),
          ),
        ],
        flexibleSpace: Builder(
          builder: (context) {
            final isExpanded = expanded(context);
            return AnnotatedRegion<SystemUiOverlayStyle>(
              value:
                  isExpanded || Theme.of(context).brightness == Brightness.dark
                  ? SystemUiOverlayStyle.light
                  : SystemUiOverlayStyle.dark,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  FlexibleSpaceBar(
                    collapseMode: CollapseMode.pin,
                    // Keep the cover, complete avatar and action band in the same
                    // sliver; a negative overflow into a later sliver is occluded.
                    background: IgnorePointer(
                      ignoring: !isExpanded,
                      child: ExcludeSemantics(
                        excluding: !isExpanded,
                        child: GfUserCardHeader(
                          coverHeight: coverHeight,
                          actionHeight: actionHeight,
                          coverUrl: resolveApiAssetUrl(coverUrl),
                          avatarUrl: resolveApiAssetUrl(avatarUrl),
                          avatarBadge: avatarBadge,
                          actions: actions,
                        ),
                      ),
                    ),
                  ),
                  if (isExpanded)
                    Positioned(
                      left: 0,
                      right: 0,
                      top: 0,
                      height: topInset + 64,
                      child: const IgnorePointer(
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            gradient: LinearGradient(
                              begin: Alignment.topCenter,
                              end: Alignment.bottomCenter,
                              colors: [
                                Color(0x99000000),
                                Color(0x88000000),
                                Colors.transparent,
                              ],
                              stops: [0, .6, 1],
                            ),
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            );
          },
        ),
      );
    },
  );
}

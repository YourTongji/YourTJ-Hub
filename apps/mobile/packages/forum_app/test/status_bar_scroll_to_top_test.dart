import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('status bar scroll skips refresh and pagination', (tester) async {
    const FakeViewPadding statusBar = FakeViewPadding(top: 59, bottom: 34);
    addTearDown(tester.view.reset);
    tester.view.physicalSize = const Size(390, 844);
    tester.view.padding = statusBar;
    tester.view.viewPadding = statusBar;
    final ScrollController scrollController = ScrollController();
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
      scrollController.dispose();
    });
    var refreshCalls = 0;
    var loadMoreCalls = 0;

    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: GfStatusBarScrollToTop(
          scrollController: scrollController,
          child: Scaffold(
            body: AppRefreshIndicator(
              onRefresh: () async {
                refreshCalls++;
              },
              child: CustomScrollView(
                controller: scrollController,
                physics: const AlwaysScrollableScrollPhysics(),
                slivers: [
                  SliverList(
                    delegate: SliverChildBuilderDelegate(
                      (_, index) => const SizedBox(height: 80),
                      childCount: 80,
                    ),
                  ),
                  SliverToBoxAdapter(
                    child: GfListFooter(
                      loading: false,
                      hasMore: true,
                      progressKey: 1,
                      onLoadMore: () => loadMoreCalls++,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );

    scrollController.jumpTo(1400);
    await tester.pump();
    final int loadMoreCallsBeforeTap = loadMoreCalls;
    expect(refreshCalls, 0);

    tester.state<ScaffoldState>(find.byType(Scaffold)).handleStatusBarTap();
    await tester.pumpAndSettle();

    expect(scrollController.offset, 0);
    expect(refreshCalls, 0);
    expect(loadMoreCalls, loadMoreCallsBeforeTap);
  }, variant: TargetPlatformVariant.only(TargetPlatform.iOS));
}

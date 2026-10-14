import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('status bar tap scrolls the visible main list to its top', (
    tester,
  ) async {
    _setStatusBarInsets(tester);
    final ScrollController scrollController = ScrollController();
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
      scrollController.dispose();
    });

    await tester.pumpWidget(
      MaterialApp(
        home: GfStatusBarScrollToTop(
          scrollController: scrollController,
          child: Scaffold(
            body: ListView.builder(
              controller: scrollController,
              itemExtent: 60,
              itemCount: 80,
              itemBuilder: (_, index) => Text('row $index'),
            ),
          ),
        ),
      ),
    );

    scrollController.jumpTo(800);
    _tapStatusBar(tester);
    await tester.pumpAndSettle();

    expect(scrollController.offset, 0);
    expect(tester.takeException(), isNull);
  }, variant: TargetPlatformVariant.only(TargetPlatform.iOS));

  testWidgets('only the currently visible tab scrolls to its top', (
    tester,
  ) async {
    _setStatusBarInsets(tester);
    final firstTab = GfScrollToTopController();
    final secondTab = GfScrollToTopController();
    final selectedTab = ValueNotifier<int>(0);
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
      firstTab.dispose();
      secondTab.dispose();
      selectedTab.dispose();
    });

    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: ValueListenableBuilder<int>(
          valueListenable: selectedTab,
          builder: (context, selected, _) => GfStatusBarScrollToTop(
            controller: selected == 0 ? firstTab : secondTab,
            child: Scaffold(
              body: Stack(
                fit: StackFit.expand,
                children: [
                  Offstage(
                    offstage: selected != 0,
                    child: _tabList(firstTab, 'first'),
                  ),
                  Offstage(
                    offstage: selected != 1,
                    child: _tabList(secondTab, 'second'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    firstTab.scrollController!.jumpTo(500);
    secondTab.scrollController!.jumpTo(500);
    _tapStatusBar(tester);
    await tester.pumpAndSettle();

    expect(firstTab.scrollController!.offset, 0);
    expect(secondTab.scrollController!.offset, 500);

    selectedTab.value = 1;
    await tester.pumpAndSettle();
    firstTab.scrollController!.jumpTo(300);
    _tapStatusBar(tester);
    await tester.pumpAndSettle();

    expect(firstTab.scrollController!.offset, 300);
    expect(secondTab.scrollController!.offset, 0);
  }, variant: TargetPlatformVariant.only(TargetPlatform.iOS));

  testWidgets('nested scroll reaches the top', (tester) async {
    _setStatusBarInsets(tester);
    final ScrollController outerController = ScrollController();
    final GlobalKey<NestedScrollViewState> nestedKey =
        GlobalKey<NestedScrollViewState>();
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
      outerController.dispose();
    });

    await tester.pumpWidget(
      MaterialApp(
        home: GfStatusBarScrollToTop(
          scrollController: outerController,
          child: Scaffold(
            body: NestedScrollView(
              key: nestedKey,
              controller: outerController,
              headerSliverBuilder: (context, innerBoxIsScrolled) => [
                const SliverToBoxAdapter(child: SizedBox(height: 240)),
              ],
              body: CustomScrollView(
                slivers: [
                  SliverList(
                    delegate: SliverChildBuilderDelegate(
                      (_, index) => SizedBox(
                        height: 60,
                        child: Text('nested row $index'),
                      ),
                      childCount: 80,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );

    final NestedScrollViewState nested = nestedKey.currentState!;
    nested.innerController.jumpTo(700);
    nested.outerController.jumpTo(100);
    _tapStatusBar(tester);
    await tester.pumpAndSettle();

    expect(nested.outerController.offset, 0);
  }, variant: TargetPlatformVariant.only(TargetPlatform.iOS));

  testWidgets('tapping at the top repeatedly is harmless', (tester) async {
    _setStatusBarInsets(tester);
    final ScrollController scrollController = ScrollController();
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
      scrollController.dispose();
    });
    await tester.pumpWidget(
      MaterialApp(
        home: GfStatusBarScrollToTop(
          scrollController: scrollController,
          child: Scaffold(
            body: ListView(
              controller: scrollController,
              children: const [SizedBox(height: 1800)],
            ),
          ),
        ),
      ),
    );

    _tapStatusBar(tester);
    await tester.pumpAndSettle();
    _tapStatusBar(tester);
    await tester.pumpAndSettle();

    expect(scrollController.offset, 0);
    expect(tester.takeException(), isNull);
  }, variant: TargetPlatformVariant.only(TargetPlatform.iOS));

  testWidgets('Android status-bar behavior is unchanged', (tester) async {
    final ScrollController ambientController = ScrollController();
    final ScrollController pageController = ScrollController();
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
      ambientController.dispose();
      pageController.dispose();
    });
    ScrollController? observedPrimary;

    await tester.pumpWidget(
      MaterialApp(
        home: PrimaryScrollController(
          controller: ambientController,
          child: GfStatusBarScrollToTop(
            scrollController: pageController,
            child: Builder(
              builder: (context) {
                observedPrimary = PrimaryScrollController.maybeOf(context);
                return Scaffold(
                  body: ListView.builder(
                    controller: pageController,
                    itemExtent: 60,
                    itemCount: 80,
                    itemBuilder: (_, index) => Text('row $index'),
                  ),
                );
              },
            ),
          ),
        ),
      ),
    );

    expect(observedPrimary, same(ambientController));
    pageController.jumpTo(600);
    _tapStatusBar(tester);
    await tester.pumpAndSettle();

    expect(pageController.offset, 600);
  }, variant: TargetPlatformVariant.only(TargetPlatform.android));
}

Widget _tabList(GfScrollToTopController controller, String label) =>
    GfScrollToTop(
      controller: controller,
      semanticLabel: 'Back to top',
      showButton: false,
      builder: (context, scrollController) => ListView.builder(
        controller: scrollController,
        itemExtent: 60,
        itemCount: 80,
        itemBuilder: (_, index) => Text('$label $index'),
      ),
    );

void _tapStatusBar(WidgetTester tester) =>
    tester.state<ScaffoldState>(find.byType(Scaffold)).handleStatusBarTap();

void _setStatusBarInsets(WidgetTester tester) {
  const FakeViewPadding statusBar = FakeViewPadding(top: 59, bottom: 34);
  addTearDown(tester.view.reset);
  tester.view.physicalSize = const Size(390, 844);
  tester.view.padding = statusBar;
  tester.view.viewPadding = statusBar;
}

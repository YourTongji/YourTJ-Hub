import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('GfAppBar delegates navigation chrome to TDesign', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          appBar: GfAppBar(
            title: const Text('首页'),
            actions: <Widget>[
              GfIconButton(icon: Icons.settings, onPressed: () {}),
            ],
          ),
        ),
      ),
    );

    expect(find.byType(td.TNavBar), findsOneWidget);
    expect(find.text('首页'), findsOneWidget);
    expect(find.byIcon(Icons.settings), findsOneWidget);

    final Finder buttonBox = find
        .ancestor(
          of: find.byIcon(Icons.settings),
          matching: find.byType(SizedBox),
        )
        .first;
    expect(tester.getSize(buttonBox), const Size.square(44));
  });

  testWidgets('GfAppBar keeps TDesign chrome below the status safe area', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: MediaQuery(
          data: const MediaQueryData(padding: EdgeInsets.only(top: 44)),
          child: const Scaffold(appBar: GfAppBar(title: Text('安全区'))),
        ),
      ),
    );

    expect(tester.getTopLeft(find.byType(td.TNavBar)).dy, 44);
  });

  testWidgets('GfBottomNavigation separates destinations and compose action', (
    tester,
  ) async {
    int selected = 0;
    int composeTaps = 0;
    await tester.pumpWidget(
      gfApp(
        GfBottomNavigation(
          currentIndex: selected,
          onSelected: (int value) => selected = value,
          onAction: () => composeTaps++,
          actionLabel: '发布',
          items: const <GfBottomNavigationItem>[
            GfBottomNavigationItem(
              label: '首页',
              icon: Icons.home_outlined,
              selectedIcon: Icons.home,
            ),
            GfBottomNavigationItem(
              label: '搜索',
              icon: Icons.search_outlined,
              selectedIcon: Icons.search,
            ),
            GfBottomNavigationItem(
              label: '消息',
              icon: Icons.forum_outlined,
              selectedIcon: Icons.forum,
              badge: true,
            ),
            GfBottomNavigationItem(
              label: '我的',
              icon: Icons.person_outline,
              selectedIcon: Icons.person,
            ),
          ],
        ),
      ),
    );

    expect(find.text('首页'), findsOneWidget);
    expect(find.text('搜索'), findsOneWidget);
    expect(find.text('消息'), findsOneWidget);
    expect(find.text('我的'), findsOneWidget);
    expect(find.text('发布'), findsOneWidget);

    await tester.tap(find.text('我的'));
    expect(selected, 3);
    await tester.tap(find.text('发布'));
    expect(composeTaps, 1);
  });

  testWidgets(
    'GfBottomNavigation rejects invalid destination counts at runtime',
    (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfBottomNavigation(
            currentIndex: 0,
            onSelected: (_) {},
            items: const <GfBottomNavigationItem>[
              GfBottomNavigationItem(
                label: '首页',
                icon: Icons.home_outlined,
                selectedIcon: Icons.home,
              ),
              GfBottomNavigationItem(
                label: '搜索',
                icon: Icons.search_outlined,
                selectedIcon: Icons.search,
              ),
              GfBottomNavigationItem(
                label: '消息',
                icon: Icons.forum_outlined,
                selectedIcon: Icons.forum,
              ),
            ],
          ),
        ),
      );

      final Object? error = tester.takeException();
      expect(error, isA<FlutterError>());
      expect('$error', contains('requires exactly four destinations'));
    },
  );

  testWidgets('GfScrollToTop appears after threshold and returns to start', (
    tester,
  ) async {
    final GfScrollToTopController controller = GfScrollToTopController();
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: GfScrollToTop(
            semanticLabel: '返回顶部',
            controller: controller,
            threshold: 100,
            builder: (BuildContext context, ScrollController scrollController) {
              return ListView.builder(
                controller: scrollController,
                itemCount: 30,
                itemExtent: 60,
                itemBuilder: (BuildContext context, int index) =>
                    Text('第 $index 项'),
              );
            },
          ),
        ),
      ),
    );

    expect(controller.isAttached, isTrue);
    expect(find.byTooltip('返回顶部'), findsNothing);
    await tester.drag(find.byType(ListView), const Offset(0, -500));
    await tester.pumpAndSettle();
    expect(find.byTooltip('返回顶部'), findsOneWidget);

    final Future<void> scroll = controller.scrollToTop();
    await tester.pumpAndSettle();
    await scroll;
    expect(find.text('第 0 项'), findsOneWidget);
    expect(find.byTooltip('返回顶部'), findsNothing);
  });
}

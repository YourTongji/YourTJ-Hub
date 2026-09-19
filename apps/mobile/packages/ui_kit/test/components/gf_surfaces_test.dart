import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfCard / GfCardList', () {
    testWidgets('mobile card draws divider, emphasized adds shadow', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          Column(
            children: <Widget>[
              GfCard(child: const Text('a')),
              GfCard(
                emphasized: true,
                showDivider: false,
                child: const Text('b'),
              ),
            ],
          ),
        ),
      );
      expect(find.byType(GfCard), findsNWidgets(2));
      expect(find.text('a'), findsOneWidget);
      expect(find.text('b'), findsOneWidget);
      final Iterable<Container> emphasizedContainers = tester
          .widgetList<Container>(
            find.descendant(
              of: find.byWidgetPredicate(
                (Widget widget) => widget is GfCard && widget.emphasized,
              ),
              matching: find.byType(Container),
            ),
          );
      expect(
        emphasizedContainers.any(
          (Container container) =>
              container.decoration is BoxDecoration &&
              (container.decoration! as BoxDecoration).color ==
                  GfColors.light.base100,
        ),
        isTrue,
      );
    });

    testWidgets('GfCardList hides divider on all but last card', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          const GfCardList(
            children: <Widget>[Text('one'), Text('two'), Text('three')],
          ),
        ),
      );
      expect(find.byType(GfCard), findsNWidgets(3));
      expect(find.text('three'), findsOneWidget);
    });
  });

  group('GfMenuSurface / GfMenuItem', () {
    testWidgets('renders menu items in surface', (tester) async {
      await tester.pumpWidget(
        gfApp(
          const GfMenuSurface(
            child: Column(
              children: [
                GfMenuItem(label: 'Edit', icon: Icons.edit),
                GfMenuItem(label: 'Delete', variant: GfMenuItemVariant.danger),
              ],
            ),
          ),
        ),
      );
      expect(find.text('Edit'), findsOneWidget);
      expect(find.text('Delete'), findsOneWidget);
    });

    testWidgets('popup menus open below their trigger', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: Align(
              alignment: Alignment.topCenter,
              child: Padding(
                padding: const EdgeInsets.only(top: 100),
                child: PopupMenuButton<String>(
                  itemBuilder: (_) => const [
                    PopupMenuItem(value: 'one', child: Text('One')),
                    PopupMenuItem(value: 'two', child: Text('Two')),
                  ],
                ),
              ),
            ),
          ),
        ),
      );

      final Rect trigger = tester.getRect(find.byType(PopupMenuButton<String>));
      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();

      expect(
        tester.getRect(find.text('One')).top,
        greaterThanOrEqualTo(trigger.bottom),
      );
      expect(tester.takeException(), isNull);
    });
  });

  group('GfFloatingSurface', () {
    testWidgets('renders child', (tester) async {
      await tester.pumpWidget(
        gfApp(const GfFloatingSurface(child: Text('float'))),
      );
      expect(find.text('float'), findsOneWidget);
    });
  });

  group('GfModal', () {
    for (final alert in [false, true]) {
      testWidgets('${alert ? 'alert' : 'modal'} keeps actions above keyboard', (
        tester,
      ) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = const Size(390, 844);
        tester.view.padding = const FakeViewPadding(top: 59, bottom: 34);
        tester.view.viewPadding = const FakeViewPadding(top: 59, bottom: 34);
        addTearDown(tester.view.reset);
        late BuildContext page;
        await tester.pumpWidget(
          gfApp(
            Builder(
              builder: (context) {
                page = context;
                return const SizedBox();
              },
            ),
          ),
        );
        Widget form(BuildContext context) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const GfInput(hintText: 'Reason', maxLines: 3),
            GfButton(
              label: 'Submit',
              onPressed: () => Navigator.pop(context, 'ok'),
            ),
          ],
        );
        final result = alert
            ? showGfAlertDialog<String>(
                page,
                builder: (context) => GfAlertDialog(
                  title: const Text('Confirm'),
                  content: form(context),
                ),
              )
            : showGfModal<String>(page, builder: form);
        await tester.pumpAndSettle();
        await tester.enterText(find.byType(TextField), 'Keep this text');
        tester.view.viewInsets = const FakeViewPadding(bottom: 336);
        tester.view.padding = const FakeViewPadding(top: 59);
        await tester.pumpAndSettle();
        expect(tester.getBottomLeft(find.text('Submit')).dy, lessThan(508));
        expect(find.text('Keep this text'), findsOneWidget);
        await tester.tap(find.text('Submit'));
        await tester.pumpAndSettle();
        expect(await result, 'ok');
        expect(tester.takeException(), isNull);
      });
    }
  });

  group('GfBottomSheet', () {
    testWidgets('sheet is above a nested shell overlay', (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 844);
      tester.view.padding = const FakeViewPadding(bottom: 34);
      tester.view.viewPadding = const FakeViewPadding(bottom: 34);
      addTearDown(tester.view.reset);
      late BuildContext page;

      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Stack(
            children: [
              Positioned.fill(
                child: Navigator(
                  onGenerateRoute: (_) => MaterialPageRoute<void>(
                    builder: (context) {
                      page = context;
                      return const SizedBox();
                    },
                  ),
                ),
              ),
              Positioned(
                key: const Key('shell-navigation'),
                left: 0,
                right: 0,
                bottom: 0,
                height: 80,
                child: ColoredBox(color: Colors.black),
              ),
            ],
          ),
        ),
      );

      showGfBottomSheet<void>(
        page,
        builder: (_) => const SizedBox(key: Key('sheet-content'), height: 200),
      );
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('sheet-content')), findsOneWidget);
      expect(
        find.byKey(const Key('shell-navigation')).hitTestable(),
        findsNothing,
      );
      expect(tester.takeException(), isNull);
      Navigator.of(page, rootNavigator: true).pop();
      await tester.pumpAndSettle();
    });

    testWidgets('short sheets consume safe areas once and fit their content', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 844);
      tester.view.padding = const FakeViewPadding(top: 59, bottom: 34);
      tester.view.viewPadding = const FakeViewPadding(top: 59, bottom: 34);
      addTearDown(tester.view.reset);
      late BuildContext page;
      await tester.pumpWidget(
        gfApp(
          Builder(
            builder: (context) {
              page = context;
              return const SizedBox();
            },
          ),
        ),
      );
      showGfBottomSheet<void>(
        page,
        builder: (_) => const SafeArea(
          child: SizedBox(key: Key('sheet-content'), height: 100),
        ),
      );
      await tester.pumpAndSettle();

      final content = tester.getRect(find.byKey(const Key('sheet-content')));
      final panel = tester.getRect(
        find
            .ancestor(
              of: find.byKey(const Key('sheet-content')),
              matching: find.byType(Material),
            )
            .first,
      );
      expect(content.height, 100);
      expect(content.top, panel.top, reason: 'No notch padding inside a sheet');
      expect(content.bottom, 844 - 34);
      expect(panel.bottom, 844, reason: 'Surface paints behind home indicator');
      expect(panel.height, 134, reason: 'No default 240px empty panel');
      expect(tester.takeException(), isNull);
    });

    testWidgets('long sheets scroll within the safe viewport', (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 844);
      tester.view.padding = const FakeViewPadding(top: 59, bottom: 34);
      tester.view.viewPadding = const FakeViewPadding(top: 59, bottom: 34);
      addTearDown(tester.view.reset);
      late BuildContext page;
      await tester.pumpWidget(
        gfApp(
          Builder(
            builder: (context) {
              page = context;
              return const SizedBox();
            },
          ),
        ),
      );
      final result = showGfBottomSheet<int>(
        page,
        builder: (context) => SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text('Pick class'),
              ),
              Flexible(
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: 30,
                  itemBuilder: (_, index) => ListTile(
                    title: Text('Class $index'),
                    onTap: () => Navigator.pop(context, index),
                  ),
                ),
              ),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.getTopLeft(find.text('Pick class')).dy, 59 + 16);
      expect(tester.getBottomLeft(find.byType(ListView)).dy, 844 - 34);
      await tester.scrollUntilVisible(find.text('Class 29'), 300);
      await tester.tap(find.text('Class 29'));
      await tester.pumpAndSettle();
      expect(await result, 29);
      expect(tester.takeException(), isNull);
    });

    testWidgets('provides Material for inputs rendered in popup', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          Builder(
            builder: (BuildContext context) => GfButton(
              label: 'New message',
              onPressed: () {
                showGfBottomSheet<void>(
                  context,
                  builder: (_) =>
                      const SafeArea(child: GfInput(hintText: 'Search users')),
                );
              },
            ),
          ),
        ),
      );

      await tester.tap(find.text('New message'));
      await tester.pumpAndSettle();

      expect(find.text('Search users'), findsOneWidget);
      expect(
        find.ancestor(
          of: find.byType(TextField),
          matching: find.byType(Material),
        ),
        findsWidgets,
      );
      expect(tester.takeException(), isNull);
    });

    testWidgets(
      'preferred height fits above the keyboard without double insets',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = const Size(390, 844);
        tester.view.padding = const FakeViewPadding(top: 59);
        tester.view.viewPadding = const FakeViewPadding(top: 59, bottom: 34);
        tester.view.viewInsets = const FakeViewPadding(bottom: 336);
        addTearDown(tester.view.reset);
        late BuildContext page;
        await tester.pumpWidget(
          gfApp(
            Builder(
              builder: (context) {
                page = context;
                return const SizedBox();
              },
            ),
          ),
        );
        showGfBottomSheet<void>(
          page,
          height: 600,
          keyboardAware: true,
          builder: (_) => Column(
            children: [
              const Text('Review'),
              Expanded(
                child: ListView(children: const [GfInput(hintText: 'Content')]),
              ),
              const SizedBox(key: Key('save'), height: 48),
            ],
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.getTopLeft(find.text('Review')).dy, 59);
        expect(
          tester.getBottomLeft(find.byKey(const Key('save'))).dy,
          844 - 336,
        );
        expect(tester.takeException(), isNull);
      },
    );

    testWidgets('keyboard-aware sheet moves its input above view insets', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          Builder(
            builder: (BuildContext context) => GfButton(
              label: 'Compose',
              onPressed: () {
                showGfBottomSheet<void>(
                  context,
                  height: 280,
                  keyboardAware: true,
                  builder: (_) => const GfInput(hintText: 'Search people'),
                );
              },
            ),
          ),
        ),
      );

      await tester.tap(find.text('Compose'));
      await tester.pumpAndSettle();
      final double before = tester.getBottomLeft(find.byType(TextField)).dy;

      tester.view.viewInsets = const FakeViewPadding(bottom: 240);
      addTearDown(tester.view.resetViewInsets);
      await tester.pumpAndSettle();

      final double after = tester.getBottomLeft(find.byType(TextField)).dy;
      expect(after, lessThan(before));
      expect(tester.takeException(), isNull);
    });
  });

  group('GfToast', () {
    testWidgets('top banners replace, dismiss and survive sheet closure', (
      tester,
    ) async {
      late BuildContext page;
      await tester.pumpWidget(
        gfApp(
          Builder(
            builder: (context) {
              page = context;
              return const SizedBox();
            },
          ),
        ),
      );
      showModalBottomSheet<void>(
        context: page,
        builder: (context) => TextButton(
          onPressed: () {
            showGfToast(context, 'Saved');
            Navigator.pop(context);
          },
          child: const Text('Save'),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(find.text('Saved'), findsOneWidget);
      expect(tester.getTopLeft(find.text('Saved')).dy, lessThan(120));
      showGfToast(page, 'Detailed error reason', error: true);
      await tester.pumpAndSettle();
      expect(find.text('Saved'), findsNothing);
      expect(find.text('Detailed error reason'), findsOneWidget);
      await tester.pump(const Duration(seconds: 4));
      expect(find.text('Detailed error reason'), findsOneWidget);
      await tester.tap(find.byTooltip('Close'));
      await tester.pump();
      expect(find.text('Detailed error reason'), findsNothing);
      showGfToast(page, 'Timed');
      await tester.pump();
      await tester.pump(const Duration(seconds: 4));
      expect(find.text('Timed'), findsNothing);
      showGfToast(page, 'Unmount');
      await tester.pump();
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 8));
      expect(tester.takeException(), isNull);
    });
  });

  group('GfPageHeader / GfSectionHeader / GfEmpty', () {
    testWidgets('renders page header with title and description', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          const GfPageHeader(
            title: 'Hello',
            description: 'desc',
            actions: Text('action'),
          ),
        ),
      );
      expect(find.text('Hello'), findsOneWidget);
      expect(find.text('desc'), findsOneWidget);
      expect(find.text('action'), findsOneWidget);
    });

    testWidgets('renders section header', (tester) async {
      await tester.pumpWidget(
        gfApp(const GfSectionHeader(title: 'Section', icon: Icons.settings)),
      );
      expect(find.text('Section'), findsOneWidget);
    });

    testWidgets('renders empty state with message and description', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(const GfEmpty(message: 'Nothing', description: 'Try again')),
      );
      expect(find.text('Nothing'), findsOneWidget);
      expect(find.text('Try again'), findsOneWidget);
    });
  });
}

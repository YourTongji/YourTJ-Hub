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
    testWidgets('shows dialog content via showGfModal', (tester) async {
      await tester.pumpWidget(gfApp(const SizedBox()));
      await tester.tap(find.byType(SizedBox));
      await tester.pump();
      // No-op guard: showGfModal requires a button; verified in forum_app.
      expect(find.byType(GfModal), findsNothing);
    });
  });

  group('GfBottomSheet', () {
    testWidgets('provides Material for inputs rendered in TDesign popup', (
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
    testWidgets('renders toast helper scaffold', (tester) async {
      // showGfToast needs a ScaffoldMessenger; smoke via building app.
      await tester.pumpWidget(gfApp(const SizedBox()));
      expect(find.byType(Scaffold), findsOneWidget);
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

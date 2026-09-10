import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfBadge', () {
    testWidgets('renders all five variants with token colors', (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            Column(
              children: [
                for (final variant in GfBadgeVariant.values)
                  GfBadge(label: 'badge', variant: variant),
              ],
            ),
            brightness: brightness,
          ),
        );
        expect(find.byType(GfBadge), findsNWidgets(5));
        expect(find.text('badge'), findsNWidgets(5));
      });
    });

    testWidgets('renders optional icon', (tester) async {
      await tester.pumpWidget(
        gfApp(GfBadge(label: 'x', icon: const Icon(Icons.star))),
      );
      expect(find.byIcon(Icons.star), findsOneWidget);
    });
  });

  group('GfStatusMessage', () {
    testWidgets('renders error and success variants', (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            Column(
              children: [
                const GfStatusMessage(message: 'boom'),
                const GfStatusMessage(
                  message: 'ok',
                  variant: GfStatusMessageVariant.success,
                ),
              ],
            ),
            brightness: brightness,
          ),
        );
        expect(find.text('boom'), findsOneWidget);
        expect(find.text('ok'), findsOneWidget);
      });
    });
  });

  group('GfSegmented', () {
    testWidgets('renders segments and reports selection', (tester) async {
      String? selected = 'a';
      await tester.pumpWidget(
        gfApp(
          StatefulBuilder(
            builder: (context, setState) => GfSegmented<String>(
              segments: const [('A', 'a'), ('B', 'b')],
              selected: selected!,
              onSelected: (value) => setState(() => selected = value),
            ),
          ),
        ),
      );

      expect(find.text('A'), findsOneWidget);
      expect(find.text('B'), findsOneWidget);

      await tester.tap(find.text('B'));
      await tester.pump();
      expect(selected, 'b');
    });

    testWidgets('active item uses primary text color', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfSegmented<String>(
            segments: const [('A', 'a'), ('B', 'b')],
            selected: 'a',
            onSelected: (_) {},
          ),
        ),
      );
      final Text a = tester.widget(find.text('A'));
      expect(
        a.style?.color,
        GfColors.light.primary,
        reason: 'active segment uses primary text',
      );
    });
  });

  group('GfAvatar', () {
    testWidgets('renders fallback icon for empty src', (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(const GfAvatar(src: '', size: 40), brightness: brightness),
        );
        expect(find.byIcon(Icons.person), findsOneWidget);
      });
    });

    testWidgets('renders stack with overlap', (tester) async {
      await tester.pumpWidget(
        gfApp(
          const GfAvatarStack(
            avatarUrls: ['a', 'b', 'c'],
            size: GfAvatarStackSize.sm,
          ),
        ),
      );
      expect(find.byType(GfAvatar), findsNWidgets(3));
    });
  });

  group('GfInput / GfTextarea', () {
    testWidgets('renders input with hint', (tester) async {
      await tester.pumpWidget(gfApp(const GfInput(hintText: 'Search...')));
      expect(find.text('Search...'), findsOneWidget);
      expect(
        tester.getSize(find.byType(GfInput)).height,
        greaterThanOrEqualTo(48),
      );
    });

    testWidgets('renders textarea with multiline', (tester) async {
      await tester.pumpWidget(gfApp(const GfTextarea(hintText: 'Body')));
      expect(find.byType(TextField), findsOneWidget);
    });
  });

  group('GfDivider', () {
    testWidgets('renders 1px line', (tester) async {
      await tester.pumpWidget(gfApp(const GfDivider()));
      final Container container = tester.widget(find.byType(Container));
      expect(container.color, isNotNull);
    });
  });

  group('GfAlert', () {
    testWidgets('renders child and icon', (tester) async {
      await tester.pumpWidget(
        gfApp(
          const GfAlert(
            icon: Icon(Icons.info_outline),
            child: Text('alert body'),
          ),
        ),
      );
      expect(find.text('alert body'), findsOneWidget);
      expect(find.byIcon(Icons.info_outline), findsOneWidget);
    });
  });

  group('GfTooltip / GfSkeleton', () {
    testWidgets('renders tooltip wrapper', (tester) async {
      await tester.pumpWidget(
        gfApp(const GfTooltip(message: 'tip', child: Text('target'))),
      );
      expect(find.text('target'), findsOneWidget);
    });

    testWidgets('renders skeleton box', (tester) async {
      await tester.pumpWidget(gfApp(const GfSkeleton(width: 80, height: 16)));
      expect(find.byType(GfSkeleton), findsOneWidget);
    });
  });
}

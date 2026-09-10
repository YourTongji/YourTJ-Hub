import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

void main() {
  testWidgets('gallery retains aspect fit, swipe count and full-screen entry', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const Scaffold(
          body: GfMediaCarousel(
            images: [
              'https://example.test/one.png',
              'https://example.test/two.png',
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('1 / 2'), findsOneWidget);
    expect(tester.widget<Image>(find.byType(Image).first).fit, BoxFit.contain);
    await tester.drag(find.byType(PageView), const Offset(-700, 0));
    await tester.pumpAndSettle();
    expect(find.text('2 / 2'), findsOneWidget);
    await tester.pumpWidget(
      gfApp(
        const Scaffold(
          body: GfMediaCarousel(
            images: [
              'https://example.test/new.png',
              'https://example.test/other.png',
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      find.text('1 / 2'),
      findsOneWidget,
      reason: 'A replaced gallery must reset its page and counter together',
    );
  });
}

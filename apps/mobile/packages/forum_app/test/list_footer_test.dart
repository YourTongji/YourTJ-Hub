import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets(
    'loads near the end once, never from an offscreen cached footer',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);
      var calls = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              controller: controller,
              child: Column(
                children: [
                  const SizedBox(height: 1600),
                  GfListFooter(
                    loading: false,
                    hasMore: true,
                    onLoadMore: () => calls++,
                  ),
                ],
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      expect(calls, 0);
      controller.jumpTo(controller.position.maxScrollExtent);
      await tester.pump();
      expect(calls, 1);
      controller.jumpTo(controller.offset - 1);
      await tester.pump();
      expect(calls, 1);
    },
  );
  testWidgets(
    'short lists fill the viewport, errors and exhausted lists do not auto-load',
    (tester) async {
      var calls = 0;
      Future<void> show({
        bool loading = false,
        bool more = true,
        String? error,
      }) async {
        await tester.pumpWidget(
          MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: ListView(
                children: [
                  GfListFooter(
                    key: UniqueKey(),
                    loading: loading,
                    hasMore: more,
                    error: error,
                    onLoadMore: () => calls++,
                  ),
                ],
              ),
            ),
          ),
        );
        await tester.pump();
      }

      await show(loading: true);
      expect(calls, 0);
      await show(more: false);
      expect(calls, 0);
      await show(error: 'network');
      expect(calls, 0);
      await show();
      expect(calls, 1);
    },
  );
}

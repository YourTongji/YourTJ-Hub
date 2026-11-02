import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/notifications/unread_display.dart';
import 'package:forum_app/src/notifications/unread_display_settings.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  for (final language in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets(
        'unread display setting remains usable in $language/${brightness.name} with large text',
        (tester) async {
          tester.view.devicePixelRatio = 1;
          tester.view.physicalSize = const Size(320, 780);
          addTearDown(tester.view.reset);
          var fail = true;
          Completer<NotificationPreferences>? pending;
          final controller = UnreadDisplayController(
            loadPreferences: () async => const NotificationPreferences(),
            savePreferences: (style) async {
              if (pending != null) await pending.future;
              if (fail) {
                throw const ApiException(
                  fallbackMessage: 'Failed',
                  messageCode: 'common.operation.failed',
                );
              }
              return NotificationPreferences(badgeStyle: style);
            },
            isCurrent: () => true,
          );
          final container = ProviderContainer(
            overrides: [
              unreadDisplayProvider.overrideWith((ref) => controller),
            ],
          );
          var disposed = false;
          void dispose() {
            if (!disposed) {
              disposed = true;
              container.dispose();
            }
          }

          addTearDown(dispose);
          await tester.pumpWidget(
            UncontrolledProviderScope(
              container: container,
              child: MaterialApp(
                theme: gfThemeData(brightness),
                locale: Locale(language),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(
                    context,
                  ).copyWith(textScaler: const TextScaler.linear(2)),
                  child: child!,
                ),
                home: Scaffold(
                  body: ListView(children: const [UnreadDisplaySettings()]),
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          final row = find.byKey(const Key('unread-style-switch'));
          expect(tester.widget<GfSwitchRow>(row).value, isTrue);
          pending = Completer();
          await tester.tap(find.byType(Switch));
          await tester.pumpAndSettle();
          expect(tester.widget<GfSwitchRow>(row).enabled, isFalse);
          expect(tester.widget<Switch>(find.byType(Switch)).onChanged, isNull);
          pending.complete(const NotificationPreferences());
          await tester.pumpAndSettle();
          expect(tester.widget<GfSwitchRow>(row).value, isTrue);
          expect(controller.state.error, isNotNull);
          expect(tester.takeException(), isNull);
          await tester.tap(
            find.byTooltip(
              MaterialLocalizations.of(tester.element(row)).closeButtonTooltip,
            ),
          );
          await tester.pumpAndSettle();
          fail = false;
          pending = null;
          await tester.tap(find.byType(Switch));
          await tester.pumpAndSettle();
          expect(tester.widget<GfSwitchRow>(row).value, isFalse);
          expect(controller.state.style, UnreadBadgeStyle.dot);
          expect(controller.state.error, isNull);
          expect(tester.takeException(), isNull);
          await tester.pumpWidget(const SizedBox());
          dispose();
        },
      );
    }
  }
}

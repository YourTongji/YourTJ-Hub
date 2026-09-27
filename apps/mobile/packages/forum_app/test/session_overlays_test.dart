import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/session_overlays.dart';
import 'package:forum_app/src/providers.dart';

void main() {
  for (final nested in [false, true]) {
    testWidgets(
      'session boundary cancels old forms in ${nested ? 'nested' : 'direct settings'} navigation',
      (tester) async {
        final registry = SessionOverlayRegistry();
        final container = ProviderContainer();
        addTearDown(container.dispose);
        final navigatorKey = GlobalKey<NavigatorState>();
        final submitted = <String>[];
        Future<bool?>? oldResult;
        Future<void> openForm(BuildContext context) async {
          oldResult = showDialog<bool>(
            context: context,
            useRootNavigator: !nested,
            builder: (context) => AlertDialog(
              title: const Text('Previous account form'),
              actions: [
                TextButton(
                  onPressed: () => Navigator.pop(context, true),
                  child: const Text('Save'),
                ),
              ],
            ),
          );
          if (await oldResult == true) submitted.add('old account write');
        }

        Widget page() => Scaffold(
          body: Builder(
            builder: (context) => TextButton(
              onPressed: () => openForm(context),
              child: const Text('Edit account'),
            ),
          ),
        );
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: SessionOverlayHost(
              registry: registry,
              child: MaterialApp(
                navigatorKey: nested ? null : navigatorKey,
                navigatorObservers: [registry.observer()],
                home: nested
                    ? Navigator(
                        key: navigatorKey,
                        observers: [registry.observer()],
                        onGenerateRoute: (_) =>
                            MaterialPageRoute<void>(builder: (_) => page()),
                      )
                    : page(),
              ),
            ),
          ),
        );
        await tester.tap(find.text('Edit account'));
        await tester.pumpAndSettle();
        container.read(offlineCacheEpochProvider.notifier).invalidate();
        // A new-session overlay opens before delayed cleanup runs. It must survive.
        final newResult = navigatorKey.currentState!.push(
          DialogRoute<void>(
            context: navigatorKey.currentContext!,
            builder: (_) => const AlertDialog(title: Text('New account form')),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.text('Previous account form'), findsNothing);
        expect(find.text('New account form'), findsOneWidget);
        expect(await oldResult, isNull);
        expect(submitted, isEmpty);
        navigatorKey.currentState!.pop();
        await newResult;
        await tester.pumpAndSettle();
      },
    );
  }

  testWidgets('a late validation result cannot submit a removed session form', (
    tester,
  ) async {
    final registry = SessionOverlayRegistry();
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final validated = Completer<void>();
    var writes = 0;
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: SessionOverlayHost(
          registry: registry,
          child: MaterialApp(
            navigatorObservers: [registry.observer()],
            home: Scaffold(
              body: Builder(
                builder: (context) => TextButton(
                  child: const Text('Edit'),
                  onPressed: () async {
                    final accepted = await showDialog<bool>(
                      context: context,
                      builder: (dialogContext) => AlertDialog(
                        actions: [
                          TextButton(
                            child: const Text('Save'),
                            onPressed: () async {
                              await validated.future;
                              if (dialogContext.mounted) {
                                Navigator.of(dialogContext).pop(true);
                              }
                            },
                          ),
                        ],
                      ),
                    );
                    if (accepted == true) writes++;
                  },
                ),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Edit'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    validated.complete();
    await tester.pumpAndSettle();
    expect(writes, 0);
    expect(find.byType(AlertDialog), findsNothing);
  });
}

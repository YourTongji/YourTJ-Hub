import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/reading_preferences.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues(<String, Object>{}));

  /// Zero debounce so `pumpEventQueue()` flushes the scheduled persist.
  ProviderContainer makeContainer() => ProviderContainer(
    overrides: <Override>[
      contentFontScaleProvider.overrideWith(
        () => ContentFontScaleNotifier(persistDelay: Duration.zero),
      ),
    ],
  );

  test('defaults to 100% when nothing is stored', () async {
    final container = makeContainer();
    addTearDown(container.dispose);

    expect(container.read(contentFontScaleProvider), 1);
    await pumpEventQueue();
    expect(container.read(contentFontScaleProvider), 1);
  });

  test('persists the reader preference and restores it', () async {
    final container = makeContainer();
    addTearDown(container.dispose);

    container.read(contentFontScaleProvider.notifier).setScale(1.3);
    expect(container.read(contentFontScaleProvider), 1.3);
    await pumpEventQueue();
    expect(
      (await SharedPreferences.getInstance()).getDouble(
        ContentFontScaleNotifier.prefsKey,
      ),
      1.3,
    );

    SharedPreferences.setMockInitialValues(<String, Object>{
      ContentFontScaleNotifier.prefsKey: 1.3,
    });
    final restored = makeContainer();
    addTearDown(restored.dispose);
    restored.read(contentFontScaleProvider);
    await pumpEventQueue();
    expect(restored.read(contentFontScaleProvider), 1.3);
  });

  test('clamps out-of-range values, including restored ones', () async {
    final container = makeContainer();
    addTearDown(container.dispose);
    final notifier = container.read(contentFontScaleProvider.notifier);

    notifier.setScale(.1);
    expect(container.read(contentFontScaleProvider), .8);
    notifier.setScale(9);
    expect(container.read(contentFontScaleProvider), 1.4);
    notifier.resetToDefault();
    expect(container.read(contentFontScaleProvider), 1);

    // Flush queued writes before replacing the mock store; otherwise the
    // earlier 100% write lands in the fresh store and hides the restore path.
    await pumpEventQueue();
    SharedPreferences.setMockInitialValues(<String, Object>{
      ContentFontScaleNotifier.prefsKey: 3,
    });
    final restored = makeContainer();
    addTearDown(restored.dispose);
    restored.read(contentFontScaleProvider);
    await pumpEventQueue();
    expect(restored.read(contentFontScaleProvider), 1.4);
  });

  test('the last choice wins even when writes are still in flight', () async {
    final container = makeContainer();
    addTearDown(container.dispose);
    final notifier = container.read(contentFontScaleProvider.notifier);

    notifier.setScale(1.4);
    notifier.setScale(.9);
    notifier.persistScale();
    await pumpEventQueue();
    expect(container.read(contentFontScaleProvider), .9);
    expect(
      (await SharedPreferences.getInstance()).getDouble(
        ContentFontScaleNotifier.prefsKey,
      ),
      .9,
    );
  });

  test('restore never overrides a choice made while it was in flight', () async {
    SharedPreferences.setMockInitialValues(<String, Object>{
      ContentFontScaleNotifier.prefsKey: .9,
    });
    final container = makeContainer();
    addTearDown(container.dispose);

    // Reading the provider starts the async restore; the user drags the
    // slider before it completes. The revision guard must keep the user's
    // 1.3 instead of overwriting it with the stored .9.
    final notifier = container.read(contentFontScaleProvider.notifier);
    notifier.setScale(1.3);
    await pumpEventQueue();
    expect(container.read(contentFontScaleProvider), 1.3);
  });

  test('dragging defers the write until the debounce fires', () async {
    final container = ProviderContainer(
      overrides: <Override>[
        contentFontScaleProvider.overrideWith(
          () => ContentFontScaleNotifier(
            persistDelay: const Duration(milliseconds: 50),
          ),
        ),
      ],
    );
    addTearDown(container.dispose);
    final notifier = container.read(contentFontScaleProvider.notifier);

    notifier.setScale(1.1);
    notifier.setScale(1.2);
    // Nothing has been written yet: only the in-session state moves.
    expect(
      (await SharedPreferences.getInstance()).getDouble(
        ContentFontScaleNotifier.prefsKey,
      ),
      isNull,
    );

    await Future<void>.delayed(const Duration(milliseconds: 100));
    await pumpEventQueue();
    expect(
      (await SharedPreferences.getInstance()).getDouble(
        ContentFontScaleNotifier.prefsKey,
      ),
      1.2,
    );
  });

  test('persistScale flushes immediately and cancels the pending write', () async {
    final container = ProviderContainer(
      overrides: <Override>[
        contentFontScaleProvider.overrideWith(
          () => ContentFontScaleNotifier(
            persistDelay: const Duration(days: 1),
          ),
        ),
      ],
    );
    addTearDown(container.dispose);
    final notifier = container.read(contentFontScaleProvider.notifier);

    notifier.setScale(1.3);
    notifier.persistScale();
    await pumpEventQueue();
    expect(
      (await SharedPreferences.getInstance()).getDouble(
        ContentFontScaleNotifier.prefsKey,
      ),
      1.3,
    );
  });
}

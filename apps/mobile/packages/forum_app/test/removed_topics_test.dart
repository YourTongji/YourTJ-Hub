import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/providers.dart';

void main() {
  test('topic tombstones deduplicate and reset at session/site boundaries', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    container.read(removedTopicsProvider.notifier).add(100);
    container.read(removedTopicsProvider.notifier).add(100);
    expect(container.read(removedTopicsProvider), {100});
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    expect(container.read(removedTopicsProvider), isEmpty);
  });
}

import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/local/writing_store.dart';

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  const draft = LocalDraft(
    key: 'new-3',
    title: '',
    content: '写到一半',
    contentType: 3,
    topicId: 0,
    categories: [],
    images: ['https://example.com/a.png'],
    updatedAt: 1,
  );
  test(
    'incomplete draft survives a store restart and is isolated by site and account',
    () async {
      final scope = writingScope('https://forum.test', 1);
      await WritingStore().save(scope, draft);
      expect(
        (await WritingStore().drafts(scope)).single.toJson(),
        draft.toJson(),
      );
      expect(
        await WritingStore().drafts(writingScope('https://forum.test', 2)),
        isEmpty,
      );
      expect(
        await WritingStore().drafts(writingScope('https://other.test', 1)),
        isEmpty,
      );
      expect(
        await WritingStore().drafts(writingScope('https://forum.test', 0)),
        isEmpty,
      );
    },
  );
  test('successful delete runs after pending save', () async {
    final store = WritingStore();
    final saving = store.save('site:1', draft);
    final deleting = store.delete('site:1', draft.key);
    await Future.wait([saving, deleting]);
    expect(await store.drafts('site:1'), isEmpty);
  });
  test('account closure clears only that account local writing', () async {
    final store = WritingStore();
    await store.save('site:1', draft);
    await store.remember('site:1', 'private query');
    await store.save('site:2', draft);
    await store.clearAccount('site:1');
    expect(await store.drafts('site:1'), isEmpty);
    expect(await store.history('site:1'), isEmpty);
    expect(await store.drafts('site:2'), hasLength(1));
  });
  test('history is unique, bounded, scoped and clearable', () async {
    final store = WritingStore();
    for (var i = 0; i < 12; i++) {
      await store.remember('site:1', 'query $i');
    }
    await store.remember('site:1', ' query 5 ');
    final history = await WritingStore().history('site:1');
    expect(history.length, 10);
    expect(history.first, 'query 5');
    expect(history.where((q) => q == 'query 5').length, 1);
    expect(await store.history('site:2'), isEmpty);
    await store.clearHistory('site:1');
    expect(await store.history('site:1'), isEmpty);
  });
}

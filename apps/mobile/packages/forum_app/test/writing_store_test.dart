import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_platform_interface.dart';
import 'package:forum_app/src/local/writing_store.dart';

// Android 平台的 SharedPreferences.remove 对缺失 key 返回 false（报告的是
// key 是否存在，而非写入成败）；默认 InMemory mock 恒返回 true 复现不了 #704，
// 需要存在性精确的 fake store 才能把回归测试写红。
class _ExistenceAccurateStore extends SharedPreferencesStorePlatform {
  final Map<String, Object> _data = <String, Object>{};

  @override
  Future<bool> remove(String key) async => _data.remove(key) != null;

  @override
  Future<bool> setValue(String valueType, String key, Object value) async {
    _data[key] = value;
    return true;
  }

  @override
  Future<bool> clear() async {
    _data.clear();
    return true;
  }

  @override
  Future<Map<String, Object>> getAll() async => Map<String, Object>.from(_data);
}

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

  group('missing-key cleanup is idempotent (#704)', () {
    setUp(() {
      // 外层 setUp 先装好 InMemory mock，这里换成存在性精确的 store 并清掉
      // 已缓存的 SharedPreferences 单例，使 getInstance 重新读 fake 数据。
      SharedPreferencesStorePlatform.instance = _ExistenceAccurateStore();
      SharedPreferences.resetStatic();
    });
    test('saving an empty draft that was never persisted succeeds', () async {
      const emptyDraft = LocalDraft(
        key: 'new-3',
        title: '',
        content: '',
        contentType: 3,
        topicId: 0,
        categories: [],
        images: [],
        updatedAt: 1,
      );
      await expectLater(WritingStore().save('site:1', emptyDraft), completes);
      expect(await WritingStore().drafts('site:1'), isEmpty);
    });
    test('deleting a draft that was never persisted succeeds', () async {
      await expectLater(
        WritingStore().delete('site:1', 'never-saved'),
        completes,
      );
    });
    test(
      'clearing history and account data without records succeeds',
      () async {
        await expectLater(WritingStore().clearHistory('site:1'), completes);
        await expectLater(WritingStore().clearAccount('site:1'), completes);
      },
    );
  });
}

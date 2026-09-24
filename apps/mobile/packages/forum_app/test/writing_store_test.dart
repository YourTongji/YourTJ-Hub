import 'dart:async';
import 'dart:convert';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_bridge.dart';
import 'pages_smoke_test.dart' show NoopOfflineCache;
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

class _DelayedStore extends _ExistenceAccurateStore {
  final pending = Completer<void>();
  bool first = true;
  @override
  Future<bool> setValue(String valueType, String key, Object value) async {
    if (first) {
      first = false;
      await pending.future;
    }
    return super.setValue(valueType, key, value);
  }
}

class _FailingRemovalStore extends _ExistenceAccurateStore {
  @override
  Future<bool> remove(String key) async => false;
}

class _FailOnceStore extends _ExistenceAccurateStore {
  bool failNext = true;
  @override
  Future<bool> setValue(String valueType, String key, Object value) async {
    if (failNext) {
      failNext = false;
      return false;
    }
    return super.setValue(valueType, key, value);
  }
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
    'undo restores all metadata only while the deleted identity is absent',
    () async {
      final store = WritingStore();
      await store.save('site:1', draft);
      await store.delete('site:1', draft.key);
      expect(await store.restoreIfAbsent('site:1', draft), isTrue);
      expect((await store.drafts('site:1')).single.toJson(), draft.toJson());
      final newer = LocalDraft.fromJson({
        ...draft.toJson(),
        'content': 'new body',
        'updatedAt': 2,
      });
      await store.save('site:1', newer);
      expect(await store.restoreIfAbsent('site:1', draft), isFalse);
      expect((await store.drafts('site:1')).single.content, 'new body');
      expect(await store.drafts('site:2'), isEmpty);
      await expectLater(
        store.restoreIfAbsent('site:0', draft),
        throwsStateError,
      );
    },
  );
  test(
    'undo waits for an editor save and never overwrites that newer copy',
    () async {
      final platform = _DelayedStore();
      SharedPreferencesStorePlatform.instance = platform;
      SharedPreferences.resetStatic();
      final store = WritingStore();
      final newer = LocalDraft.fromJson({
        ...draft.toJson(),
        'content': 'new body',
      });
      final save = store.save('site:1', newer);
      final undo = store.restoreIfAbsent('site:1', draft);
      platform.pending.complete();
      await save;
      expect(await undo, isFalse);
      expect((await store.drafts('site:1')).single.content, 'new body');
    },
  );
  test(
    'queued undo fails after session invalidation without restoring old content',
    () async {
      final platform = _DelayedStore();
      SharedPreferencesStorePlatform.instance = platform;
      SharedPreferences.resetStatic();
      final store = WritingStore();
      final save = store.save('site:2', draft);
      var current = true;
      final undo = store.restoreIfAbsent(
        'site:1',
        draft,
        isCurrent: () => current,
      );
      final failed = expectLater(undo, throwsStateError);
      current = false;
      platform.pending.complete();
      await save;
      await failed;
      expect(await store.drafts('site:1'), isEmpty);
      expect(await store.drafts('site:2'), hasLength(1));
    },
  );
  test(
    'failed platform restoration remains retryable despite preferences cache',
    () async {
      SharedPreferencesStorePlatform.instance = _FailOnceStore();
      SharedPreferences.resetStatic();
      final store = WritingStore();
      await expectLater(
        store.restoreIfAbsent('site:1', draft),
        throwsStateError,
      );
      expect(await store.restoreIfAbsent('site:1', draft), isTrue);
      SharedPreferences.resetStatic();
      expect(
        (await WritingStore().drafts('site:1')).single.toJson(),
        draft.toJson(),
      );
    },
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
  test('legacy v1 slots survive and new identities do not collide', () async {
    final scope = writingScope('https://forum.test', 1);
    final legacy = draft.toJson()
      ..remove('version')
      ..remove('kind')
      ..remove('replyToPostId')
      ..remove('replyTargetName')
      ..remove('replyMentionPrefix');
    SharedPreferences.setMockInitialValues({
      'yourtj:writing:v1:$scope:draft:new-3': jsonEncode(legacy),
    });
    final loaded = (await WritingStore().drafts(scope)).single;
    expect(loaded.key, 'new-3');
    expect(loaded.kind, DraftKind.newTopic);
    expect(loaded.content, draft.content);
    final keys = List.generate(50, (_) => newTopicDraftKey()).toSet();
    expect(keys, hasLength(50));
    expect(keys, isNot(contains('new-3')));
    expect({
      topicDraftKey(42, published: true),
      topicDraftKey(42, published: false),
      replyDraftKey(42),
    }, hasLength(3));
  });

  test(
    'a queued mutation is rejected after its account session changes',
    () async {
      final platform = _DelayedStore();
      SharedPreferencesStorePlatform.instance = platform;
      SharedPreferences.resetStatic();
      final store = WritingStore();
      final first = store.save('site:1', draft);
      var current = true;
      final stale = store.save('site:2', draft, isCurrent: () => current);
      final rejected = expectLater(stale, throwsStateError);
      current = false;
      platform.pending.complete();
      await first;
      await rejected;
      expect(await store.drafts('site:2'), isEmpty);
      expect(await store.drafts('site:1'), hasLength(1));
    },
  );

  test('cache clearing preserves account-scoped writing', () async {
    final store = WritingStore();
    await store.save('site:1', draft);
    await clearOfflineCache(
      NoopOfflineCache(),
      NoopOfflineCache(),
      ScheduleWidgetBridge(),
    );
    expect((await store.drafts('site:1')).single.content, draft.content);
  });

  test('a failed deletion of an existing recovery copy is reported', () async {
    SharedPreferencesStorePlatform.instance = _FailingRemovalStore();
    SharedPreferences.resetStatic();
    final store = WritingStore();
    await store.save('site:1', draft);
    await expectLater(store.delete('site:1', draft.key), throwsStateError);
  });

  test('a contextual reply title alone does not create a draft', () async {
    final store = WritingStore();
    await store.save(
      'site:1',
      const LocalDraft(
        key: 'reply-100',
        kind: DraftKind.reply,
        title: 'Topic context',
        content: '',
        contentType: 2,
        topicId: 100,
        categories: [],
        images: [],
        updatedAt: 1,
      ),
    );
    expect(await store.drafts('site:1'), isEmpty);
  });

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

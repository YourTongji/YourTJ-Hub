import 'dart:async';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/messages/chat_drafts.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_platform_interface.dart';

const peer = ChatItemPayload(
  id: 0,
  peerId: 2,
  peerUsername: 'Bob',
  peerAvatar: '',
  convId: 0,
  lastMsg: '',
  lastMsgTime: '',
  unreadCount: 0,
  peerUrl: '/u/2',
);
const input = TextEditingValue(
  text: '还没写完的私信',
  selection: TextSelection(baseOffset: 2, extentOffset: 5),
);

class _FailingPlatform extends SharedPreferencesStorePlatform {
  final data = <String, Object>{};
  bool fail = false;
  @override
  Future<Map<String, Object>> getAll() async => Map.of(data);
  @override
  Future<bool> setValue(String valueType, String key, Object value) async {
    if (fail) return false;
    data[key] = value;
    return true;
  }

  @override
  Future<bool> remove(String key) async {
    if (fail) return false;
    return data.remove(key) != null;
  }

  @override
  Future<bool> clear() async {
    data.clear();
    return true;
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => SharedPreferences.setMockInitialValues({}));
  ChatDrafts registry({
    String scope = 'site:1',
    ChatDraftStore? store,
    Future<String> Function()? owner,
    bool Function()? current,
  }) {
    final drafts = ChatDrafts(
      store: store ?? ChatDraftStore(),
      resolveScope: owner ?? () async => scope,
      isCurrent: current ?? () => true,
    );
    addTearDown(drafts.dispose);
    return drafts;
  }

  test(
    'restart restores text and selection only for the same account and site',
    () async {
      final first = registry();
      first.update(
        peer,
        input.copyWith(composing: const TextRange(start: 0, end: 2)),
      );
      expect(await first.flush(), isTrue);
      final reopened = registry();
      await reopened.flush();
      expect(reopened.forPeer(2)!.value, input);
      for (final scope in ['site:2', 'other:1']) {
        final other = registry(scope: scope);
        await other.flush();
        expect(other.items, isEmpty);
      }
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getKeys().single, 'yourtj:writing:v1:site:1:chat:2');
      expect(await WritingStore().drafts('site:1'), isEmpty);
    },
  );

  test(
    'typing before owner resolution wins over the earlier stored draft',
    () async {
      final previous = registry();
      previous.update(peer, input);
      await previous.flush();
      final owner = Completer<String>();
      final drafts = registry(owner: () => owner.future);
      const newest = TextEditingValue(
        text: '新输入',
        selection: TextSelection.collapsed(offset: 1),
      );
      drafts.update(peer, newest);
      final saving = drafts.flush();
      owner.complete('site:1');
      expect(await saving, isTrue);
      expect(drafts.forPeer(2)!.value, newest);
      expect((await ChatDraftStore().read('site:1')).single.value, newest);
    },
  );

  test(
    'session change while owner is unresolved cannot persist or reveal old text',
    () async {
      final owner = Completer<String>();
      var current = true;
      final drafts = registry(
        owner: () => owner.future,
        current: () => current,
      );
      drafts.update(peer, input);
      final saving = drafts.flush();
      current = false;
      owner.complete('site:1');
      expect(await saving, isFalse);
      expect(await ChatDraftStore().read('site:1'), isEmpty);
      final next = registry(scope: 'site:2');
      await next.flush();
      expect(next.items, isEmpty);
    },
  );

  test(
    'acknowledgement clears only the submitted revision and saves conversation identity',
    () async {
      final drafts = registry();
      drafts.update(peer, input);
      final sent = drafts.forPeer(2)!.revision;
      const next = TextEditingValue(
        text: '下一条',
        selection: TextSelection.collapsed(offset: 2),
      );
      drafts.update(peer, next);
      drafts.acknowledge(2, sent, 42);
      await drafts.flush();
      expect(drafts.forPeer(2)!.value, next);
      expect((await ChatDraftStore().read('site:1')).single.convId, 42);
      drafts.acknowledge(2, drafts.forPeer(2)!.revision, 42);
      await drafts.flush();
      expect(drafts.items, isEmpty);
      expect(await ChatDraftStore().read('site:1'), isEmpty);
    },
  );

  test(
    'caret changes do not turn an acknowledged message into a new draft',
    () async {
      final drafts = registry();
      drafts.update(peer, input);
      final sent = drafts.forPeer(2)!.revision;
      drafts.update(
        peer,
        input.copyWith(selection: const TextSelection.collapsed(offset: 0)),
      );
      drafts.acknowledge(2, sent, 42);
      await drafts.flush();
      expect(drafts.items, isEmpty);
    },
  );

  test(
    'failed save and failed acknowledgement cleanup stay dirty and retry platform truth',
    () async {
      final platform = _FailingPlatform();
      SharedPreferencesStorePlatform.instance = platform;
      SharedPreferences.resetStatic();
      final drafts = registry();
      drafts.update(peer, input);
      expect(await drafts.flush(), isTrue);
      platform.fail = true;
      drafts.update(peer, const TextEditingValue(text: '最新修改'));
      expect(await drafts.flush(), isFalse);
      expect(drafts.error, isNotNull);
      expect(drafts.isDirty(2), isTrue);
      expect((await ChatDraftStore().read('site:1')).single.value, input);
      platform.fail = false;
      expect(await drafts.flush(), isTrue);
      expect(drafts.error, isNull);
      platform.fail = true;
      drafts.acknowledge(2, drafts.forPeer(2)!.revision, 42);
      expect(await drafts.flush(), isFalse);
      expect(drafts.isDirty(2), isTrue);
      expect(platform.data, isNotEmpty);
      platform.fail = false;
      expect(await drafts.flush(), isTrue);
      expect(platform.data, isEmpty);
    },
  );

  test(
    'clearing one conversation and closing one account preserves other owners',
    () async {
      final drafts = registry();
      drafts.update(peer, input);
      await drafts.flush();
      final other = registry(scope: 'site:2');
      other.update(peer, input);
      await other.flush();
      await WritingStore().clearAccount('site:1');
      expect(await ChatDraftStore().read('site:1'), isEmpty);
      expect(await ChatDraftStore().read('site:2'), hasLength(1));
      other.update(peer, TextEditingValue.empty);
      expect(await other.flush(), isTrue);
      expect(await ChatDraftStore().read('site:2'), isEmpty);
    },
  );

  test(
    'guest or invalidated writes never enter the persistent store',
    () async {
      final drafts = registry(scope: 'site:0');
      drafts.update(peer, input);
      expect(await drafts.flush(), isFalse);
      expect(drafts.error, isNotNull);
      final draft = drafts.forPeer(2)!;
      await expectLater(
        ChatDraftStore().write('site:1', draft, isCurrent: () => false),
        throwsStateError,
      );
      expect((await SharedPreferences.getInstance()).getKeys(), isEmpty);
    },
  );
}

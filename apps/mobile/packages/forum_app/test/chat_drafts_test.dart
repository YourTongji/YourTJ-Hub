import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_secure_storage/test/test_flutter_secure_storage_platform.dart';
import 'package:flutter_secure_storage_platform_interface/flutter_secure_storage_platform_interface.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/messages/chat_drafts.dart';
import 'package:forum_app/src/providers.dart';
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

class _FailingSecurePlatform extends TestFlutterSecureStoragePlatform {
  _FailingSecurePlatform() : super({});
  bool fail = false;
  bool failReadAll = false;
  Map<String, String>? lastOptions;
  @override
  Future<Map<String, String>> readAll({required Map<String, String> options}) {
    if (failReadAll) throw PlatformException(code: 'corrupt_ciphertext');
    return super.readAll(options: options);
  }

  @override
  Future<void> write({
    required String key,
    required String value,
    required Map<String, String> options,
  }) async {
    lastOptions = options;
    if (fail) throw PlatformException(code: 'locked');
    await super.write(key: key, value: value, options: options);
  }

  @override
  Future<void> delete({
    required String key,
    required Map<String, String> options,
  }) async {
    if (fail) throw PlatformException(code: 'locked');
    await super.delete(key: key, options: options);
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });
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

  test('legacy plaintext migrates only after a verified secure copy', () async {
    final prefs = await SharedPreferences.getInstance();
    const key = 'yourtj:writing:v1:site:1:chat:2';
    final legacy = ChatDraft(
      peerId: 2,
      peerName: 'Bob',
      peerAvatar: '',
      convId: 0,
      value: input,
      updatedAt: 1,
      revision: 0,
    );
    await prefs.setString(key, jsonEncode(legacy.toJson()));
    expect((await ChatDraftStore().read('site:1')).single.value, input);
    expect(prefs.containsKey(key), isFalse);
    expect(await const FlutterSecureStorage().read(key: key), isNotNull);
  });

  test('new chat drafts never write plaintext preferences', () async {
    final drafts = registry();
    drafts.update(peer, input);
    await drafts.flush();
    expect((await SharedPreferences.getInstance()).getKeys(), isEmpty);
    expect((await const FlutterSecureStorage().readAll()).length, 1);
  });

  test('iOS storage uses a non-syncing device-bound Keychain service', () {
    final options = ChatDraftStore().secureStorage.iOptions.toMap();
    expect(options['accessibility'], 'first_unlock_this_device');
    expect(options['synchronizable'], 'false');
    expect(options['accountName'], 'yourtj_chat_drafts');
    expect(
      ChatDraftStore().secureStorage.aOptions.toMap(),
      const FlutterSecureStorage().aOptions.toMap(),
      reason:
          'Android must not mutate the shared native token-storage configuration',
    );
  });

  test(
    'failed secure migration retains plaintext and retry preserves other owners',
    () async {
      final prefs = await SharedPreferences.getInstance();
      final draft = ChatDraft(
        peerId: 2,
        peerName: 'Bob',
        peerAvatar: '',
        convId: 0,
        value: input,
        updatedAt: 1,
        revision: 0,
      );
      for (final scope in ['site:1', 'site:2']) {
        await prefs.setString(
          'yourtj:writing:v1:$scope:chat:2',
          jsonEncode(draft.toJson()),
        );
      }
      final secure = _FailingSecurePlatform()..fail = true;
      FlutterSecureStoragePlatform.instance = secure;
      final store = ChatDraftStore();
      await expectLater(
        store.read('site:1'),
        throwsA(isA<PlatformException>()),
      );
      expect(prefs.getKeys(), hasLength(2));
      secure.fail = false;
      expect((await store.read('site:1')).single.value, input);
      expect(prefs.getKeys(), isEmpty);
      expect((await store.read('site:2')).single.value, input);
      secure.data['session_token'] = 'unrelated credential';
      await store.clearAccount('site:1');
      expect(await store.read('site:1'), isEmpty);
      expect(await store.read('site:2'), hasLength(1));
      expect(secure.data['session_token'], 'unrelated credential');
    },
  );

  test(
    'failed legacy deletion cannot resurrect an acknowledged draft',
    () async {
      final platform = _FailingPlatform();
      SharedPreferencesStorePlatform.instance = platform;
      SharedPreferences.resetStatic();
      final prefs = await SharedPreferences.getInstance();
      const key = 'yourtj:writing:v1:site:1:chat:2';
      final old = ChatDraft(
        peerId: 2,
        peerName: 'Bob',
        peerAvatar: '',
        convId: 0,
        value: input,
        updatedAt: 1,
        revision: 0,
      );
      await prefs.setString(key, jsonEncode(old.toJson()));
      platform.fail = true;
      final store = ChatDraftStore();
      final empty = ChatDraft.fromJson({
        ...old.toJson(),
        'text': '',
        'base': 0,
        'extent': 0,
      }, 1);
      await expectLater(
        store.write('site:1', empty, isCurrent: () => true),
        throwsStateError,
      );
      expect(await const FlutterSecureStorage().read(key: key), '');
      platform.fail = false;
      expect(await store.read('site:1'), isEmpty);
      expect(prefs.containsKey(key), isFalse);
      expect(await const FlutterSecureStorage().read(key: key), isNull);
    },
  );

  test(
    'unreadable secure storage preserves legacy and in-memory text for retry',
    () async {
      final prefs = await SharedPreferences.getInstance();
      const key = 'yourtj:writing:v1:site:1:chat:2';
      final old = ChatDraft(
        peerId: 2,
        peerName: 'Bob',
        peerAvatar: '',
        convId: 0,
        value: input,
        updatedAt: 1,
        revision: 0,
      );
      await prefs.setString(key, jsonEncode(old.toJson()));
      final secure = _FailingSecurePlatform()..failReadAll = true;
      FlutterSecureStoragePlatform.instance = secure;
      final drafts = registry();
      drafts.update(peer, const TextEditingValue(text: '新输入'));
      expect(await drafts.flush(), isFalse);
      expect(drafts.error, isNotNull);
      expect(drafts.forPeer(2)!.value.text, '新输入');
      expect(prefs.containsKey(key), isTrue);
      secure.failReadAll = false;
      expect(await drafts.flush(), isTrue);
      expect((await ChatDraftStore().read('site:1')).single.value.text, '新输入');
      expect(prefs.containsKey(key), isFalse);
    },
  );

  test('same-origin login recreates a guest draft registry', () async {
    int? account;
    final container = ProviderContainer(
      overrides: [
        currentUserProvider.overrideWith(
          (ref) async => account == null
              ? null
              : CurrentUser(id: account, username: 'Alice'),
        ),
      ],
    );
    addTearDown(container.dispose);
    final saved = registry(
      scope: writingScope(
        Uri.parse(container.read(apiClientProvider).baseUrl).origin,
        1,
      ),
    );
    saved.update(peer, input);
    await saved.flush();
    final guest = container.read(chatDraftsProvider);
    expect(await guest.flush(), isFalse);
    account = 1;
    container.invalidate(currentUserProvider);
    await container.read(writingScopeProvider.future);
    final signedIn = container.read(chatDraftsProvider);
    expect(identical(signedIn, guest), isFalse);
    expect(await signedIn.flush(), isTrue);
    expect(signedIn.error, isNull);
    expect(signedIn.forPeer(2)!.value, input);
    expect(guest.current, isFalse);
  });

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
      expect(prefs.getKeys(), isEmpty);
      expect(
        (await const FlutterSecureStorage().readAll()).keys.single,
        'yourtj:writing:v1:site:1:chat:2',
      );
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
      final platform = _FailingSecurePlatform();
      FlutterSecureStoragePlatform.instance = platform;
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
      await ChatDraftStore().clearAccount('site:1');
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

import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

/// 内存版 TokenStorage 测试替身(无真实存储)。
class _MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

GfApiClient _offlineClient() => GfApiClient(
  dio: Dio(BaseOptions(baseUrl: 'http://test')),
  tokenStorage: _MemoryTokenStorage(),
);

const List<StickerItemPayload> _smileOnly = <StickerItemPayload>[
  StickerItemPayload(name: 'smile', url: '/s.png'),
];

/// 前 [failures] 次抛错、之后成功的替身。
class _FlakyRepository extends StickerRepository {
  _FlakyRepository({this.failures = 1}) : super(_offlineClient());

  final int failures;
  int calls = 0;

  @override
  Future<List<StickerItemPayload>> list() async {
    calls++;
    if (calls <= failures) throw StateError('boom');
    return _smileOnly;
  }
}

/// 用 Completer 手动放行的替身(测并发去重)。
class _GatedRepository extends StickerRepository {
  _GatedRepository() : super(_offlineClient());

  final List<Completer<List<StickerItemPayload>>> gates =
      <Completer<List<StickerItemPayload>>>[];

  @override
  Future<List<StickerItemPayload>> list() {
    final Completer<List<StickerItemPayload>> completer =
        Completer<List<StickerItemPayload>>();
    gates.add(completer);
    return completer.future;
  }
}

/// 固定返回列表的替身(含空 name/url 噪声条目)。
class _NoisyRepository extends StickerRepository {
  _NoisyRepository() : super(_offlineClient());

  @override
  Future<List<StickerItemPayload>> list() async => const <StickerItemPayload>[
    StickerItemPayload(name: 'smile', url: '/s.png'),
    StickerItemPayload(name: '', url: '/empty-name.png'),
    StickerItemPayload(name: 'no-url', url: ''),
  ];
}

class _ResolvingRepository extends StickerRepository {
  _ResolvingRepository() : super(_offlineClient());
  final List<List<String>> batches = [];
  int failures = 0;
  @override
  Future<List<StickerItemPayload>> list() async => [];
  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async {
    batches.add([...names]);
    if (failures-- > 0) throw StateError('offline');
    return [
      for (final name in names)
        StickerItemPayload(name: name, url: '/$name.png', isOfficial: false),
    ];
  }
}

void main() {
  group('StickerLibrary.resolveContent', () {
    test(
      'coalesces many mounted messages into a bounded shared request',
      () async {
        final repository = _ResolvingRepository();
        final library = StickerLibrary(repository);
        await Future.wait([
          for (var i = 0; i < 60; i++)
            library.resolveContent('[:sticker:personal_$i:]'),
        ]);
        expect(repository.batches, hasLength(1));
        expect(repository.batches.single, hasLength(60));
        expect(library.urlByName, hasLength(60));
      },
    );

    test('long content remains within the resolver batch limit', () async {
      final repository = _ResolvingRepository();
      final library = StickerLibrary(repository);
      await library.resolveContent(
        [for (var i = 0; i < 205; i++) '[:sticker:personal_$i:]'].join(' '),
      );
      expect(repository.batches.map((batch) => batch.length), [100, 100, 5]);
      expect(library.urlByName, hasLength(205));
    });

    test('a failed resolver batch remains retryable', () async {
      final repository = _ResolvingRepository()..failures = 1;
      final library = StickerLibrary(repository);
      await expectLater(
        library.resolveContent('[:sticker:personal:]'),
        throwsStateError,
      );
      await library.resolveContent('[:sticker:personal:]');
      expect(repository.batches, hasLength(2));
      expect(library.urlByName['personal'], '/personal.png');
    });
  });

  group('StickerLibrary.load', () {
    test('首次拉取成功并缓存,isLoaded 翻转', () async {
      final _FlakyRepository repository = _FlakyRepository(failures: 0);
      final StickerLibrary library = StickerLibrary(repository);
      expect(library.isLoaded, isFalse);
      final List<StickerItemPayload> items = await library.load();
      expect(items.single.name, 'smile');
      expect(repository.calls, 1);
      expect(library.isLoaded, isTrue);
      // 缓存生效:再次 load 不发新请求。
      await library.load();
      expect(repository.calls, 1);
    });

    test('失败不缓存,下次 load 重试成功', () async {
      final _FlakyRepository repository = _FlakyRepository();
      final StickerLibrary library = StickerLibrary(repository);
      await expectLater(library.load(), throwsStateError);
      expect(repository.calls, 1);
      expect(library.isLoaded, isFalse);
      final List<StickerItemPayload> items = await library.load();
      expect(items.single.url, '/s.png');
      expect(repository.calls, 2);
    });

    test('并发调用共享同一在途请求,不重复拉取', () async {
      final _GatedRepository repository = _GatedRepository();
      final StickerLibrary library = StickerLibrary(repository);
      final Future<List<StickerItemPayload>> first = library.load();
      final Future<List<StickerItemPayload>> second = library.load();
      expect(repository.gates, hasLength(1));
      repository.gates.single.complete(_smileOnly);
      final List<List<StickerItemPayload>> items = await Future.wait(
        <Future<List<StickerItemPayload>>>[first, second],
      );
      expect(items[0].single.name, 'smile');
      expect(items[1].single.name, 'smile');
      await library.load();
      expect(repository.gates, hasLength(1));
    });

    test('disposed session ignores a late load and future writes', () async {
      final repository = _GatedRepository();
      final library = StickerLibrary(repository);
      final pending = library.load();
      library.dispose();
      repository.gates.single.complete(_smileOnly);
      await pending;
      library.remember(_smileOnly);
      expect(library.urlByName, isEmpty);
      expect(await library.load(), isEmpty);
      expect(repository.gates, hasLength(1));
    });
  });

  group('StickerLibrary.urlByName', () {
    test('未加载时返回空映射', () {
      final StickerLibrary library = StickerLibrary(_FlakyRepository());
      expect(library.urlByName, isEmpty);
    });

    test('加载后给出 token→url 映射,跳过空 name/url 条目', () async {
      final StickerLibrary library = StickerLibrary(_NoisyRepository());
      await library.load();
      expect(library.urlByName, <String, String>{'smile': '/s.png'});
    });
  });
}

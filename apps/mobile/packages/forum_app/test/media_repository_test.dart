import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/storage/media_repository.dart';

class _Http implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  Future<ResponseBody> Function(RequestOptions)? handle;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? stream,
    Future<void>? cancel,
  ) async {
    requests.add(options);
    return handle?.call(options) ?? response();
  }

  @override
  void close({bool force = false}) {}
}

class _CountingDirectory implements Directory {
  _CountingDirectory(this.directory);
  final Directory directory;
  int listings = 0;
  @override
  String get path => directory.path;
  @override
  Future<Directory> create({bool recursive = false}) =>
      directory.create(recursive: recursive);
  @override
  Stream<FileSystemEntity> list({
    bool recursive = false,
    bool followLinks = true,
  }) {
    listings++;
    return directory.list(recursive: recursive, followLinks: followLinks);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

ResponseBody response({
  String cache = 'public, max-age=60',
  List<int> bytes = const [1, 2, 3],
  int status = 200,
  Map<String, List<String>> extra = const {},
}) => ResponseBody.fromBytes(
  bytes,
  status,
  headers: {
    'content-type': ['image/gif'],
    'cache-control': [cache],
    ...extra,
  },
);
const _origin = 'https://example.test';
const _url = '$_origin/image.gif';
void main() {
  late Directory dir;
  late _Http http;
  late Dio dio;
  late DateTime now;
  late MediaRepository cache;
  setUp(() async {
    dir = await Directory.systemTemp.createTemp('yourtj-media-test-');
    http = _Http();
    dio = Dio()..httpClientAdapter = http;
    now = DateTime.utc(2026, 9, 29);
    cache = MediaRepository(
      directory: () async => dir,
      dio: dio,
      now: () => now,
    );
  });
  tearDown(() async {
    cache.dispose();
    await dir.delete(recursive: true);
  });
  Future<Uint8List> load({String url = _url, String scope = 'site:42:zh'}) =>
      cache.load(url, scopeKey: scope, apiOrigin: _origin);
  test(
    'chat origin policy rejects tracking redirects before requesting them',
    () async {
      http.handle = (_) async => response(
        status: 302,
        extra: {
          'location': ['https://tracker.example/pixel.gif'],
        },
      );
      await expectLater(
        cache.load(
          _url,
          scopeKey: 'site:42:zh',
          apiOrigin: _origin,
          allowedOrigins: {_origin},
        ),
        throwsFormatException,
      );
      expect(http.requests.map((r) => r.uri.toString()), [_url]);
    },
  );

  test('chat origin policy permits explicitly trusted CDN redirects', () async {
    const cdn = 'https://cdn.example.test';
    http.handle = (request) async => request.uri.toString() == _url
        ? response(
            status: 302,
            extra: {
              'location': ['$cdn/image.gif'],
            },
          )
        : response();
    expect(
      await cache.load(
        _url,
        scopeKey: 'site:42:zh',
        apiOrigin: _origin,
        allowedOrigins: {_origin, cdn},
      ),
      [1, 2, 3],
    );
    expect(http.requests.map((r) => r.uri.toString()), [
      _url,
      '$cdn/image.gif',
    ]);
  });
  test(
    'public fresh image survives repository restart and scopes stay separate',
    () async {
      expect(await load(), [1, 2, 3]);
      expect(await load(), [1, 2, 3]);
      expect(http.requests.length, 1);
      cache.dispose();
      cache = MediaRepository(
        directory: () async => dir,
        dio: dio,
        now: () => now,
      );
      expect(await load(), [1, 2, 3]);
      expect(http.requests.length, 1);
      await load(scope: 'site:99:zh');
      await load(scope: 'site:42:de');
      expect(http.requests.length, 3);
    },
  );
  for (final policy in [
    'private, max-age=60',
    'no-store',
    'public, no-cache',
    'public, max-age=0',
    'max-age=60',
    'public, max-age=60, must-revalidate',
  ]) {
    test('HTTP policy $policy is respected', () async {
      http.handle = (_) async => response(cache: policy);
      await load();
      if (policy.endsWith('must-revalidate')) {
        now = now.add(const Duration(minutes: 2));
        http.handle = (_) async => throw StateError('offline');
        await expectLater(load(), throwsA(isA<DioException>()));
      } else {
        expect(await cache.usageBytes(), 0);
        await load();
        expect(http.requests.length, 2);
      }
    });
  }
  test(
    'signed URLs, cross-origin URLs and cookies are never persisted',
    () async {
      for (final url in [
        '$_url?signature=secret',
        'https://cdn.example.test/image.gif',
        'https://user:secret@example.test/image.gif',
      ]) {
        await load(url: url);
        expect(await cache.usageBytes(), 0);
      }
      http.handle = (_) async => response(
        extra: {
          'set-cookie': ['s=private'],
        },
      );
      await load();
      expect(await cache.usageBytes(), 0);
    },
  );
  test(
    'expired public image revalidates with ETag and a 304 reuses bytes',
    () async {
      http.handle = (_) async => response(
        extra: {
          'etag': ['"v1"'],
        },
      );
      await load();
      now = now.add(const Duration(minutes: 2));
      http.handle = (_) async => response(status: 304, bytes: []);
      expect(await load(), [1, 2, 3]);
      expect(http.requests.last.headers['If-None-Match'], '"v1"');
    },
  );
  test('clear fences a delayed response and never refills disk', () async {
    final pending = Completer<ResponseBody>();
    http.handle = (_) => pending.future;
    final read = load();
    final expectation = expectLater(read, throwsA(anything));
    while (http.requests.isEmpty) {
      await Future<void>.delayed(Duration.zero);
    }
    await cache.clear();
    pending.complete(response());
    await expectation;
    expect(await cache.usageBytes(), 0);
  });
  test('global budget evicts older scopes and idle files expire', () async {
    cache.dispose();
    cache = MediaRepository(
      directory: () => Future.value(dir),
      dio: dio,
      now: () => now,
      budgetBytes: 6000,
    );
    http.handle = (_) async => response(bytes: List.filled(3000, 1));
    await load(scope: 'one');
    now = now.add(const Duration(seconds: 1));
    await load(scope: 'two');
    expect(await cache.usageBytes(), lessThanOrEqualTo(6000));
    await load(scope: 'one');
    expect(http.requests.length, 3);
    now = now.add(const Duration(days: 15));
    expect(await cache.usageBytes(), 0);
  });
  test(
    'large eviction scans the directory a bounded number of times',
    () async {
      String key(int id) => id.toRadixString(16).padLeft(64, '0');
      final entries = <String, Map<String, dynamic>>{};
      for (var id = 0; id < 128; id++) {
        entries[key(id)] = {
          'bytes': 32,
          'accessed': now.millisecondsSinceEpoch + id,
          'freshUntil': now
              .add(const Duration(hours: 1))
              .millisecondsSinceEpoch,
          'cacheControl': 'public, max-age=3600',
          'etag': '"表情-$id"',
        };
        await File('${dir.path}/${key(id)}').writeAsBytes(List.filled(32, id));
      }
      // Missing/mis-sized entries and interrupted writes must not skew either
      // physical accounting or the retained LRU suffix.
      entries[key(128)] = {...entries[key(0)]!, 'bytes': 4096};
      await File('${dir.path}/${key(128)}').writeAsBytes([1]);
      entries[key(129)] = {...entries[key(0)]!};
      await File(
        '${dir.path}/interrupted.tmp',
      ).writeAsBytes(List.filled(4096, 0));
      await File(
        '${dir.path}/index.json',
      ).writeAsString(jsonEncode({'version': 1, 'entries': entries}));
      final directory = _CountingDirectory(dir);
      cache.dispose();
      const budget = 18000;
      cache = MediaRepository(
        directory: () async => directory,
        dio: dio,
        now: () => now,
        budgetBytes: budget,
      );
      final usage = await cache.usageBytes();
      final files = await dir.list().cast<File>().toList();
      var actual = 0;
      for (final file in files) {
        actual += await file.length();
      }
      final retained =
          (jsonDecode(await File('${dir.path}/index.json').readAsString())
                  as Map)['entries']
              as Map;
      expect(usage, actual);
      expect(usage, lessThanOrEqualTo(budget * .8));
      expect(usage, greaterThan(budget * .8 - 500));
      expect(retained.length, inExclusiveRange(0, 128));
      expect(retained.keys, [
        for (var id = 128 - retained.length; id < 128; id++) key(id),
      ]);
      expect(await File('${dir.path}/${key(128)}').exists(), isFalse);
      expect(await File('${dir.path}/interrupted.tmp').exists(), isFalse);
      expect(directory.listings, lessThanOrEqualTo(8));
    },
  );
  test('eviction uses actual file sizes between sweeps', () async {
    cache.dispose();
    cache = MediaRepository(
      directory: () async => dir,
      dio: dio,
      now: () => now,
      budgetBytes: 4000,
    );
    http.handle = (_) async => response(bytes: List.filled(500, 1));
    await load(url: '$_origin/first');
    now = now.add(const Duration(seconds: 1));
    await load(url: '$_origin/second');
    final index =
        (jsonDecode(await File('${dir.path}/index.json').readAsString())
                as Map)['entries']
            as Map;
    final older = File('${dir.path}/${index.keys.first}');
    final newer = File('${dir.path}/${index.keys.last}');
    // An external file change must not make incremental accounting subtract
    // stale index metadata and unnecessarily evict the newer valid image.
    await older.writeAsBytes(List.filled(2500, 1));
    final temporary = File('${dir.path}/interrupted.tmp');
    await temporary.writeAsBytes(List.filled(600, 0));
    now = now.add(const Duration(seconds: 1));
    await load(url: '$_origin/third');
    expect(await older.exists(), isFalse);
    expect(await newer.exists(), isTrue);
    expect(await temporary.exists(), isTrue);
    var actual = 0;
    await for (final file in dir.list().cast<File>()) {
      actual += await file.length();
    }
    expect(actual, lessThanOrEqualTo(4000));
    expect(await cache.usageBytes(), actual - 600);
    expect(await temporary.exists(), isFalse);
  });
  test('oversized downloads fail without persistence', () async {
    cache.dispose();
    cache = MediaRepository(
      directory: () => Future.value(dir),
      dio: dio,
      maxDownloadBytes: 2,
    );
    await expectLater(load(), throwsA(anything));
    expect(await cache.usageBytes(), 0);
  });
  test('concurrent readers coalesce and at most three downloads run', () async {
    final responses = <Completer<ResponseBody>>[];
    http.handle = (_) {
      final pending = Completer<ResponseBody>();
      responses.add(pending);
      return pending.future;
    };
    final reads = [
      load(),
      load(),
      load(url: '$_origin/2'),
      load(url: '$_origin/3'),
      load(url: '$_origin/4'),
    ];
    while (responses.length < 3) {
      await Future<void>.delayed(Duration.zero);
    }
    expect(http.requests.length, 3);
    responses[0].complete(response());
    while (responses.length < 4) {
      await Future<void>.delayed(Duration.zero);
    }
    for (final pending in responses.skip(1)) {
      pending.complete(response());
    }
    await Future.wait(reads);
    expect(http.requests.length, 4);
  });
  test(
    'pending identity is memory only and query values stay distinct',
    () async {
      await load(scope: 'pending:42');
      expect(await cache.usageBytes(), 0);
      await load(url: '$_url?size=100');
      await load(url: '$_url?size=200');
      expect(http.requests.map((r) => r.uri.query), [
        '',
        'size=100',
        'size=200',
      ]);
      expect(await cache.usageBytes(), 0);
    },
  );
  test(
    'redirect origin and every hop HTTP policy can forbid storage',
    () async {
      for (final location in [
        'https://cdn.example.test/image.gif',
        '$_origin/target',
      ]) {
        http.handle = (request) async => request.uri.toString() == _url
            ? response(
                status: 302,
                cache: 'no-store',
                extra: {
                  'location': [location],
                },
              )
            : response();
        await load();
        expect(await cache.usageBytes(), 0);
      }
      expect(
        http.requests.every(
          (r) =>
              !r.headers.containsKey('Authorization') &&
              !r.headers.containsKey('Cookie'),
        ),
        isTrue,
      );
    },
  );
  test(
    'new no-store headers remove expired public bytes even if body fails',
    () async {
      await load();
      now = now.add(const Duration(minutes: 2));
      http.handle = (_) async => ResponseBody(
        Stream.error(StateError('body interrupted')),
        200,
        headers: {
          'content-type': ['image/gif'],
          'cache-control': ['no-store'],
        },
      );
      await expectLater(load(), throwsA(anything));
      expect(await cache.usageBytes(), 0);
    },
  );
  test('Age and Vary prevent invalid reuse and decoded keys expire', () async {
    for (final extra in [
      {
        'age': ['61'],
      },
      {
        'vary': ['Authorization'],
      },
      {
        'vary': ['*'],
      },
    ]) {
      http.handle = (_) async => response(extra: extra);
      await load();
      expect(await cache.usageBytes(), 0);
    }
    http.handle = (_) async => response();
    await load();
    expect(
      cache.decodedIdentity(_url, scopeKey: 'site:42:zh', apiOrigin: _origin),
      isNotNull,
    );
    now = now.add(const Duration(minutes: 2));
    expect(
      cache.decodedIdentity(_url, scopeKey: 'site:42:zh', apiOrigin: _origin),
      isNull,
    );
  });
  test(
    'retiring legacy image bytes never deletes neighboring attachments',
    () async {
      cache.dispose();
      final legacy = Directory('${dir.path}/legacy-cacheimage');
      await legacy.create();
      await File('${legacy.path}/old-image').writeAsBytes([1, 2, 3]);
      final attachment = File('${dir.path}/draft-attachment');
      await attachment.writeAsBytes([4, 5]);
      cache = MediaRepository(
        directory: () async => Directory('${dir.path}/managed'),
        legacyDirectory: () async => legacy,
        dio: dio,
      );
      expect(await cache.usageBytes(), 0);
      expect(await legacy.exists(), isFalse);
      expect(await attachment.readAsBytes(), [4, 5]);
      await cache.clear();
      expect(await attachment.exists(), isTrue);
    },
  );
  test(
    'media requests look like a browser fetch and send no Referer',
    () async {
      await load();
      final headers = http.requests.single.headers;
      expect(headers['User-Agent'], contains('Mozilla/5.0'));
      expect(headers['User-Agent'], contains('YourTJ-Hub/1.0'));
      expect(headers['Accept'], 'image/*,*/*;q=0.8');
      expect(headers.containsKey('Referer'), isFalse);
    },
  );
  test(
    'invalid decoded bytes can be discarded without touching a newer generation',
    () async {
      await load();
      final old = cache.generation;
      await cache.discard(
        _url,
        scopeKey: 'site:42:zh',
        apiOrigin: _origin,
        generation: old,
      );
      expect(await cache.usageBytes(), 0);
      cache.invalidate();
      await load();
      await cache.discard(
        _url,
        scopeKey: 'site:42:zh',
        apiOrigin: _origin,
        generation: old,
      );
      expect(await cache.usageBytes(), greaterThan(0));
    },
  );
}

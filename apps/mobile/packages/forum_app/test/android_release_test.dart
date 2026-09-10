import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/updates/android_release.dart';

Map<String, dynamic> release({int build = 12, String abi = 'arm64-v8a'}) => {
  'draft': false,
  'prerelease': false,
  'tag_name': 'mobile-v1.2.0',
  'assets': [
    <String, dynamic>{
      'name': 'YourTJ-1.2.0+$build-$abi.apk',
      'state': 'uploaded',
      'size': 12,
      'digest': 'sha256:${'ab' * 32}',
      'browser_download_url':
          'https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-v1.2.0/YourTJ-1.2.0%2B$build-$abi.apk',
    },
  ],
};

void main() {
  test(
    'selects newest build compatible with device, excluding server releases',
    () {
      final selected = AndroidRelease.latest(
        [
          release(build: 12),
          release(build: 13),
          release(build: 20, abi: 'x86_64'),
          {...release(build: 99), 'tag_name': 'v9.0.0'},
        ],
        ['arm64-v8a', 'armeabi-v7a'],
        11,
      );
      expect(selected?.buildNumber, 13);
      expect(AndroidRelease.latest([release()], ['x86_64'], 1), isNull);
      expect(AndroidRelease.latest([release()], ['arm64-v8a'], 12), isNull);
    },
  );

  test('rejects untrusted asset metadata and pre-releases', () {
    final mutations = <Map<String, dynamic>>[
      {'digest': null},
      {'digest': 'sha256:bad'},
      {'size': -1},
      {'size': 1024 * 1024 * 1024},
      {'state': 'new'},
      {'browser_download_url': 'https://github.com.evil.test/file.apk'},
      {
        'browser_download_url':
            'https://github.com/another/repo/releases/download/mobile-v1.2.0/YourTJ-1.2.0%2B12-arm64-v8a.apk',
      },
      {
        'browser_download_url':
            'http://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-v1.2.0/YourTJ-1.2.0%2B12-arm64-v8a.apk',
      },
    ];
    for (final mutation in mutations) {
      final item = release();
      (item['assets'] as List).first.addAll(mutation);
      expect(
        AndroidRelease.latest([item], ['arm64-v8a'], 1),
        isNull,
        reason: '$mutation',
      );
    }
    expect(
      AndroidRelease.latest(
        [
          {...release(), 'draft': true},
        ],
        ['arm64-v8a'],
        1,
      ),
      isNull,
    );
    expect(
      AndroidRelease.latest(
        [
          {...release(), 'prerelease': true},
        ],
        ['arm64-v8a'],
        1,
      ),
      isNull,
    );
  });

  test(
    'corrupt mirror fails over; only verified APK reaches final path',
    () async {
      final directory = await Directory.systemTemp.createTemp(
        'yourtj-update-test-',
      );
      final bytes = Uint8List.fromList(List.generate(12, (i) => i));
      final adapter = _Adapter(bytes);
      final client = AndroidReleaseClient(
        dio: Dio()..httpClientAdapter = adapter,
      );
      final parsed = AndroidRelease.latest([release()], ['arm64-v8a'], 1)!;
      final update = AndroidRelease(
        version: parsed.version,
        buildNumber: parsed.buildNumber,
        url: parsed.url,
        size: bytes.length,
        sha256Digest: sha256.convert(bytes).toString(),
      );
      final cancel = CancelToken();
      try {
        final file = await client.download(
          update,
          directory,
          cancel: cancel,
          onProgress: (_, _) {},
        );
        expect(await file.readAsBytes(), bytes);
        expect(adapter.downloads, contains('ghfast.top'));
        expect(adapter.downloads.length, greaterThan(1));
        expect(
          directory.listSync().where((f) => f.path.endsWith('.part')),
          isEmpty,
        );
        expect(
          adapter.requests.every(
            (r) =>
                !r.headers.containsKey('Authorization') &&
                !r.headers.containsKey('Cookie'),
          ),
          isTrue,
        );
      } finally {
        cancel.cancel();
        await directory.delete(recursive: true);
      }
    },
  );
  test('all corrupt sources fail closed and remove partial files', () async {
    final directory = await Directory.systemTemp.createTemp(
      'yourtj-update-fail-',
    );
    final adapter = _Adapter(Uint8List(12));
    final client = AndroidReleaseClient(
      dio: Dio()..httpClientAdapter = adapter,
    );
    final cancel = CancelToken();
    try {
      await expectLater(
        client.download(
          AndroidRelease.latest([release()], ['arm64-v8a'], 1)!,
          directory,
          cancel: cancel,
          onProgress: (_, _) {},
        ),
        throwsStateError,
      );
      expect(directory.listSync(), isEmpty);
      expect(
        adapter.downloads.toSet(),
        containsAll([
          'github.com',
          ...releaseMirrors.map((m) => Uri.parse(m).host),
        ]),
      );
    } finally {
      cancel.cancel();
      await directory.delete(recursive: true);
    }
  });

  test('cancelled update never starts an APK download', () async {
    final directory = await Directory.systemTemp.createTemp(
      'yourtj-update-cancel-',
    );
    final adapter = _Adapter(Uint8List(12));
    final client = AndroidReleaseClient(
      dio: Dio()..httpClientAdapter = adapter,
    );
    try {
      await expectLater(
        client.download(
          AndroidRelease.latest([release()], ['arm64-v8a'], 1)!,
          directory,
          cancel: CancelToken()..cancel(),
          onProgress: (_, _) {},
        ),
        throwsA(isA<DioException>()),
      );
      expect(adapter.downloads, isEmpty);
      expect(directory.listSync(), isEmpty);
    } finally {
      await directory.delete(recursive: true);
    }
  });
}

class _Adapter implements HttpClientAdapter {
  _Adapter(this.bytes);
  final Uint8List bytes;
  final requests = <RequestOptions>[];
  final downloads = <String>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    if (options.headers['Range'] != null) {
      if (options.uri.host != 'ghfast.top') {
        await Future<void>.delayed(const Duration(milliseconds: 30));
      }
      return ResponseBody.fromBytes(
        bytes,
        206,
        headers: {
          'content-range': ['bytes 0-11/12'],
          'content-length': ['12'],
        },
      );
    }
    downloads.add(options.uri.host);
    return ResponseBody.fromBytes(
      options.uri.host == 'ghfast.top' ? Uint8List(12) : bytes,
      200,
      headers: {
        'content-length': ['12'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

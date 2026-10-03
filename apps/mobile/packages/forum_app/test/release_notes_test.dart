import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/updates/android_release.dart';
import 'package:forum_app/src/updates/release_notes.dart';
import 'package:forum_app/src/updates/ios_store_release.dart';
import 'package:shared_preferences/shared_preferences.dart';

Map<String, dynamic> note(
  String id,
  String title, {
  List<String> platforms = const ['android'],
  bool required = false,
  String kind = 'feature',
}) => {
  'id': id,
  'title': title,
  'summary': '$title summary',
  'platforms': platforms,
  'kind': kind,
  if (required) 'required': true,
};

Map<String, dynamic> release(
  int build, {
  List<String> channels = const ['android'],
  List<dynamic> highlights = const [],
  List<dynamic> breaking = const [],
  List<dynamic> actions = const [],
}) => {
  'version': '1.0.${build - 1}',
  'buildNumber': build,
  'channels': channels,
  'highlights': highlights,
  'breaking': breaking,
  'requiredActions': actions,
  'testflightNotes': const [],
};

Map<String, dynamic> catalog() => {
  'schemaVersion': 1,
  'historyCoverage': {
    'source': 'github-release-receipts',
    'publishedAt': '2026-10-03T00:00:00Z',
    'byChannel': {
      'android': {
        'completeFromBuild': 4,
        'throughBuild': 7,
        'coveredBuilds': [4, 5, 6, 7],
      },
      'ios-app-store': {
        'completeFromBuild': 5,
        'throughBuild': 6,
        'coveredBuilds': [6],
      },
      'ios-testflight': {
        'completeFromBuild': 6,
        'throughBuild': 7,
        'coveredBuilds': [7],
      },
    },
  },
  'releases': [
    release(4),
    release(
      5,
      breaking: [note('shared-update', 'Older title', required: true)],
    ),
    release(
      6,
      channels: const ['android', 'ios-app-store'],
      highlights: [
        note('shared-update', 'Current title'),
        note('ios-only', 'iOS', platforms: ['ios-app-store']),
      ],
    ),
    {
      ...release(
        7,
        channels: const ['android', 'ios-testflight'],
        breaking: [
          note(
            'important-change',
            'Important',
            required: true,
            kind: 'security',
          ),
        ],
      ),
      'testflightNotes': [
        {'id': 'testing-focus', 'text': 'Verify account recovery in the beta.'},
      ],
    },
  ],
};

void main() {
  test(
    'iOS public release lookup is pinned to the Chinese App Store listing',
    () async {
      final adapter = _Adapter(
        (_) => ResponseBody.fromString(
          jsonEncode({
            'resultCount': 1,
            'results': [
              {
                'bundleId': appStoreBundleId,
                'trackId': int.parse(appStoreAppId),
                'version': '2.3.4',
                'trackViewUrl':
                    'https://apps.apple.com/cn/app/yourtj/id$appStoreAppId',
              },
            ],
          }),
          200,
          headers: {
            'content-type': ['application/json'],
          },
        ),
      );
      final listing = await IosStoreListing.lookup(
        dio: Dio()..httpClientAdapter = adapter,
      );
      expect(listing?.version, '2.3.4');
      expect(adapter.requests.single.uri.queryParameters['country'], 'cn');
    },
  );

  test(
    'parses the shared publisher fixture and channel-only TestFlight notes',
    () {
      final fixture = jsonDecode(
        File('test/fixtures/mobile_release_catalog.json').readAsStringSync(),
      );
      final decoded = ReleaseNoteCatalog.decode(fixture)!;
      final release43 = decoded.releases.singleWhere(
        (release) => release.buildNumber == 43,
      );
      expect(release43.channels, {'android', 'ios-testflight'});
      final betaNotes = decoded.forRange(
        installedBuild: 42,
        targetBuild: 43,
        platform: 'ios-testflight',
      );
      expect(
        betaNotes.single.summary,
        'Verify the course widget after changing a course.',
      );
    },
  );

  test(
    'aggregates the exclusive to inclusive build range with channel filtering and required dedup',
    () {
      final decoded = ReleaseNoteCatalog.decode(catalog())!;
      final notes = decoded.forRange(
        installedBuild: 4,
        targetBuild: 7,
        platform: 'android',
      );
      expect(notes.map((entry) => entry.id), [
        'important-change',
        'shared-update',
      ]);
      expect(notes.last.title, 'Current title');
      expect(notes.last.required, isTrue);
      expect(notes.last.summary, 'Current title summary');
      expect(
        decoded
            .forRange(
              installedBuild: 4,
              targetBuild: 7,
              platform: 'ios-app-store',
            )
            .map((entry) => entry.id),
        ['ios-only'],
      );
      expect(
        decoded.hasCompleteRange(
          installedBuild: 4,
          targetBuild: 7,
          channel: 'android',
        ),
        isTrue,
      );
      expect(
        decoded.hasCompleteRange(
          installedBuild: 4,
          targetBuild: 8,
          channel: 'android',
        ),
        isFalse,
      );
      expect(
        decoded.hasCompleteRange(
          installedBuild: 3,
          targetBuild: 7,
          channel: 'android',
        ),
        isFalse,
      );
    },
  );

  test('rejects claimed coverage without a matching release receipt', () {
    final invalid = catalog();
    ((invalid['historyCoverage'] as Map)['byChannel'] as Map)['android'] = {
      'completeFromBuild': 4,
      'throughBuild': 7,
      'coveredBuilds': [4, 5, 6, 8],
    };
    expect(ReleaseNoteCatalog.decode(invalid), isNull);
  });

  test('incomplete history falls back to only the target build notes', () {
    final incomplete = catalog();
    final android =
        ((incomplete['historyCoverage'] as Map)['byChannel'] as Map)['android']
            as Map;
    android['completeFromBuild'] = 6;
    android['coveredBuilds'] = [6, 7];
    final decoded = ReleaseNoteCatalog.decode(incomplete)!;
    expect(
      decoded
          .notesForUpdate(
            installedBuild: 4,
            targetBuild: 7,
            platform: 'android',
            channel: 'android',
          )
          .map((entry) => entry.id),
      ['important-change'],
    );
  });

  test(
    'generic iOS notes match both iOS channels in history and update ranges',
    () {
      final fixture = catalog();
      final releases = fixture['releases'] as List;
      final appStoreRelease =
          releases.singleWhere((release) => release['buildNumber'] == 6)
              as Map<String, dynamic>;
      appStoreRelease['highlights'] = [
        ...appStoreRelease['highlights'] as List,
        note('generic-ios-store', 'Generic iOS store note', platforms: ['ios']),
      ];
      final testFlightRelease =
          releases.singleWhere((release) => release['buildNumber'] == 7)
              as Map<String, dynamic>;
      testFlightRelease['highlights'] = [
        ...testFlightRelease['highlights'] as List,
        note(
          'generic-ios-testflight',
          'Generic iOS TestFlight note',
          platforms: ['ios'],
        ),
      ];
      final decoded = ReleaseNoteCatalog.decode(fixture)!;

      expect(
        decoded
            .forRange(
              installedBuild: 5,
              targetBuild: 6,
              platform: 'ios-app-store',
            )
            .map((entry) => entry.id),
        contains('generic-ios-store'),
      );
      expect(
        decoded
            .notesForUpdate(
              installedBuild: 5,
              targetBuild: 6,
              platform: 'ios-app-store',
              channel: 'ios-app-store',
            )
            .map((entry) => entry.id),
        contains('generic-ios-store'),
      );
      expect(
        decoded
            .forRange(
              installedBuild: 6,
              targetBuild: 7,
              platform: 'ios-testflight',
            )
            .map((entry) => entry.id),
        contains('generic-ios-testflight'),
      );
      expect(
        decoded
            .notesForUpdate(
              installedBuild: 6,
              targetBuild: 7,
              platform: 'ios-testflight',
              channel: 'ios-testflight',
            )
            .map((entry) => entry.id),
        contains('generic-ios-testflight'),
      );
    },
  );

  test(
    'aggregates a one-release upgrade without including installed notes',
    () {
      final fixture = catalog()
        ..remove('historyCoverage')
        ..['releases'] = [
          release(6, highlights: [note('target', 'Target')]),
          release(7, highlights: [note('future', 'Future')]),
        ];
      final decoded = ReleaseNoteCatalog.decode(fixture)!;
      expect(
        decoded
            .forRange(installedBuild: 5, targetBuild: 6, platform: 'android')
            .map((entry) => entry.id),
        ['target'],
      );
    },
  );

  test(
    'keeps last successful full-URL cache when sources fail and drops an ETag without a body',
    () async {
      SharedPreferences.setMockInitialValues({
        'mobile.releaseNotes.${Uri.encodeComponent(mobileReleaseNotesUrl)}.json':
            jsonEncode(catalog()),
        'mobile.releaseNotes.${Uri.encodeComponent(mobileReleaseNotesUrl)}.etag':
            'stale-tag',
        'mobile.releaseNotes.${Uri.encodeComponent(githubReleaseNotesUrl)}.etag':
            'orphan-tag',
      });
      final adapter = _Adapter((request) => ResponseBody.fromString('', 503));
      final dio = Dio()..httpClientAdapter = adapter;
      final client = AndroidReleaseClient(dio: dio)
        ..notesPreferences = await SharedPreferences.getInstance();
      final result = await client.loadHistory();
      expect(result?.releases.length, 4);
      expect(adapter.requests.first.headers['If-None-Match'], 'stale-tag');
      expect(
        adapter.requests.last.headers.containsKey('If-None-Match'),
        isFalse,
      );
    },
  );

  test(
    '304 and invalid JSON preserve validated cache; missing cache ETag is removed',
    () async {
      final key =
          'mobile.releaseNotes.${Uri.encodeComponent(mobileReleaseNotesUrl)}';
      SharedPreferences.setMockInitialValues({
        '$key.json': jsonEncode(catalog()),
        '$key.etag': 'v1',
      });
      final adapter = _Adapter((request) {
        if (request.uri.host == 'status.yourtj.de') {
          return ResponseBody.fromString('', 304);
        }
        return ResponseBody.fromString('{invalid', 200);
      });
      final client = AndroidReleaseClient(
        dio: Dio()..httpClientAdapter = adapter,
      )..notesPreferences = await SharedPreferences.getInstance();
      expect((await client.loadHistory())?.releases.length, 4);
      expect(adapter.requests.first.headers['If-None-Match'], 'v1');
    },
  );

  test(
    'untrusted redirect is rejected and notes failure does not affect APK choice',
    () async {
      SharedPreferences.setMockInitialValues({});
      final adapter = _Adapter((request) {
        if (request.uri.host == 'github.com') {
          return ResponseBody.fromString(
            '',
            302,
            headers: {
              'location': ['https://attacker.example/notes.json'],
            },
          );
        }
        return ResponseBody.fromString('', 503);
      });
      final client = AndroidReleaseClient(
        dio: Dio()..httpClientAdapter = adapter,
      )..notesPreferences = await SharedPreferences.getInstance();
      final selected = AndroidRelease.latest(
        [releaseFixture()],
        ['arm64-v8a'],
        1,
      )!;
      final decorated = await client.withNotes(selected, 1);
      expect(decorated.buildNumber, selected.buildNumber);
      expect(decorated.notes, isEmpty);
      expect(
        adapter.requests.any(
          (request) => request.uri.host == 'attacker.example',
        ),
        isFalse,
      );
    },
  );

  test(
    'Android split-ABI version codes resolve to catalog build numbers',
    () async {
      SharedPreferences.setMockInitialValues({
        'mobile.releaseNotes.${Uri.encodeComponent(mobileReleaseNotesUrl)}.json':
            jsonEncode(catalog()),
      });
      final client = AndroidReleaseClient(
        dio: Dio()
          ..httpClientAdapter = _Adapter(
            (request) => ResponseBody.fromString('', 503),
          ),
      )..notesPreferences = await SharedPreferences.getInstance();
      final selected = AndroidRelease.latest(
        [releaseFixture(versionCode: 2007)],
        ['arm64-v8a'],
        2004,
      )!;
      final decorated = await client.withNotes(selected, 2004, refresh: false);
      expect(decorated.buildNumber, 2007);
      expect(decorated.notes.map((entry) => entry.id), [
        'important-change',
        'shared-update',
      ]);
      expect(decorated.hasCompleteHistory, isTrue);
    },
  );

  test(
    'follows only HTTPS GitHub release asset redirects without credentials',
    () async {
      SharedPreferences.setMockInitialValues({});
      final body = jsonEncode(catalog());
      final adapter = _Adapter((request) {
        if (request.uri.host == 'status.yourtj.de') {
          return ResponseBody.fromString('', 503);
        }
        if (request.uri.host == 'github.com') {
          return ResponseBody.fromString(
            '',
            302,
            headers: {
              'location': [
                'https://release-assets.githubusercontent.com/mobile-notes?sig=x',
              ],
            },
          );
        }
        return ResponseBody.fromString(
          body,
          200,
          headers: {
            'content-type': ['application/json'],
          },
        );
      });
      final client = AndroidReleaseClient(
        dio: Dio()..httpClientAdapter = adapter,
      )..notesPreferences = await SharedPreferences.getInstance();
      expect((await client.loadHistory())?.releases.length, 4);
      expect(
        adapter.requests.last.uri.host,
        'release-assets.githubusercontent.com',
      );
      expect(
        adapter.requests.every(
          (request) =>
              !request.headers.containsKey('Authorization') &&
              !request.headers.containsKey('Cookie'),
        ),
        isTrue,
      );
    },
  );
}

Map<String, dynamic> releaseFixture({int versionCode = 12}) => {
  'draft': false,
  'prerelease': false,
  'tag_name': 'mobile-v1.2.0',
  'assets': [
    {
      'name': 'YourTJ-1.2.0+$versionCode-arm64-v8a.apk',
      'state': 'uploaded',
      'size': 12,
      'digest': 'sha256:${'ab' * 32}',
      'browser_download_url':
          'https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-v1.2.0/YourTJ-1.2.0%2B$versionCode-arm64-v8a.apk',
    },
  ],
};

class _Adapter implements HttpClientAdapter {
  _Adapter(this.reply);
  final ResponseBody Function(RequestOptions) reply;
  final requests = <RequestOptions>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    return reply(options);
  }

  @override
  void close({bool force = false}) {}
}

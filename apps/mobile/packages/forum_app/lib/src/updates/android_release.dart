import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'release_notes.dart';

const releaseRepository = 'YourTongji/YourTJ-Hub';
const releaseMirrors = [
  'https://ghfast.top/',
  'https://ghproxy.net/',
  'https://gh-proxy.com/',
];
const mobileReleaseNotesUrl = String.fromEnvironment(
  'YOURTJ_MOBILE_RELEASES_URL',
  defaultValue: 'https://status.yourtj.de/mobile/releases.json',
);
const githubReleaseNotesUrl =
    'https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-notes/releases.json';

class AndroidRelease {
  const AndroidRelease({
    required this.version,
    required this.buildNumber,
    required this.url,
    required this.size,
    required this.sha256Digest,
    this.notes = const [],
    this.hasCompleteHistory = false,
    this.catalogBuildNumber,
  });

  final String version;
  final int buildNumber;
  final Uri url;
  final int size;
  final String sha256Digest;
  final List<ReleaseNote> notes;
  final bool hasCompleteHistory;
  final int? catalogBuildNumber;

  /// Metadata is accepted only from GitHub's HTTPS API, never from APK mirrors.
  /// GitHub computes the digest itself when the signed asset is uploaded.
  static AndroidRelease? latest(
    List<dynamic> releases,
    List<String> supportedAbis,
    int installedBuild,
  ) {
    AndroidRelease? latest;
    final tagPattern = RegExp(r'^mobile-v(\d+\.\d+\.\d+)$');
    for (final item in releases.whereType<Map>()) {
      if (item['draft'] != false || item['prerelease'] != false) continue;
      final tag = item['tag_name'];
      if (tag is! String) continue;
      final match = tagPattern.firstMatch(tag);
      if (match == null || item['assets'] is! List) continue;
      final version = match[1]!;
      AndroidRelease? preferred;
      for (final abi in supportedAbis) {
        for (final asset in (item['assets'] as List).whereType<Map>()) {
          final name = asset['name'];
          if (name is! String || asset['state'] != 'uploaded') continue;
          final nameMatch = RegExp(
            '^YourTJ-${RegExp.escape(version)}\\+([1-9][0-9]*)-${RegExp.escape(abi)}\\.apk\$',
          ).firstMatch(name);
          if (nameMatch == null) continue;
          final number = int.tryParse(nameMatch[1]!);
          final size = asset['size'];
          final digest = asset['digest'];
          final url = asset['browser_download_url'] is String
              ? Uri.tryParse(asset['browser_download_url'] as String)
              : null;
          if (number == null ||
              number <= installedBuild ||
              number > 2100000000 ||
              size is! int ||
              size <= 0 ||
              size > 512 * 1024 * 1024 ||
              digest is! String ||
              !RegExp(r'^sha256:[a-fA-F0-9]{64}$').hasMatch(digest) ||
              url == null ||
              url.scheme != 'https' ||
              url.host != 'github.com' ||
              url.userInfo.isNotEmpty ||
              url.hasQuery ||
              url.hasFragment ||
              url.port != 443 ||
              url.pathSegments.join('/') !=
                  '$releaseRepository/releases/download/$tag/$name') {
            continue;
          }
          preferred = AndroidRelease(
            version: version,
            buildNumber: number,
            url: url,
            size: size,
            sha256Digest: digest.substring(7).toLowerCase(),
            catalogBuildNumber: switch (abi) {
              'armeabi-v7a' => number - 1000,
              'arm64-v8a' => number - 2000,
              'x86_64' => number - 4000,
              _ => null,
            },
          );
          break;
        }
        if (preferred != null) break;
      }
      if (preferred != null &&
          (latest == null || preferred.buildNumber > latest.buildNumber)) {
        latest = preferred;
      }
    }
    return latest;
  }
}

class AndroidReleaseClient {
  AndroidReleaseClient({Dio? dio})
    : _dio =
          dio ??
          Dio(
            BaseOptions(
              connectTimeout: const Duration(seconds: 8),
              receiveTimeout: const Duration(seconds: 15),
              headers: {'User-Agent': 'YourTJ-Mobile-Updater'},
            ),
          ),
      _notesDio = Dio(
        BaseOptions(
          connectTimeout: const Duration(seconds: 4),
          receiveTimeout: const Duration(seconds: 8),
          headers: const {
            'Accept': 'application/json',
            'User-Agent': 'YourTJ-Mobile-Updater',
          },
        ),
      ) {
    if (dio != null) _notesDio.httpClientAdapter = dio.httpClientAdapter;
  }
  // An isolated client: forum cookies, bearer tokens and user IDs never reach mirrors.
  final Dio _dio;
  final Dio _notesDio;
  final Map<String, ReleaseNoteCatalog> _notesByUrl = {};
  final Map<String, String> _etagsByUrl = {};
  final Set<String> _notesValidatedThisLoad = {};
  SharedPreferences? notesPreferences;

  Future<AndroidRelease?> check(List<String> abis, int installedBuild) async {
    final response = await _dio.get<dynamic>(
      'https://api.github.com/repos/$releaseRepository/releases',
      queryParameters: {'per_page': 100},
      options: Options(headers: {'Accept': 'application/vnd.github+json'}),
    );
    if (response.data is! List) {
      throw const FormatException('Invalid release list');
    }
    return AndroidRelease.latest(response.data as List, abis, installedBuild);
  }

  /// [installedBuild] is the base build from native BuildConfig, before ABI offsets.
  Future<AndroidRelease> withNotes(
    AndroidRelease release,
    int installedBuild, {
    bool refresh = true,
  }) async {
    try {
      final catalog = await loadHistory(refresh: refresh);
      if (catalog == null) return release;
      final targetBuild = release.catalogBuildNumber;
      if (targetBuild == null || targetBuild <= 0) return release;
      final notes = catalog.notesForUpdate(
        installedBuild: installedBuild,
        targetBuild: targetBuild,
        platform: 'android',
        channel: 'android',
      );
      return AndroidRelease(
        version: release.version,
        buildNumber: release.buildNumber,
        url: release.url,
        size: release.size,
        sha256Digest: release.sha256Digest,
        catalogBuildNumber: targetBuild,
        notes: notes,
        hasCompleteHistory: catalog.hasCompleteRange(
          installedBuild: installedBuild,
          targetBuild: targetBuild,
          channel: 'android',
        ),
      );
    } catch (_) {
      // Notes are optional display data and cannot block a verified APK update.
      return release;
    }
  }

  Future<ReleaseNoteCatalog?> loadHistory({bool refresh = true}) async {
    final urls = [
      Uri.tryParse(mobileReleaseNotesUrl),
      Uri.tryParse(githubReleaseNotesUrl),
    ];
    for (final uri in urls) {
      if (uri == null ||
          uri.scheme != 'https' ||
          uri.host.isEmpty ||
          uri.userInfo.isNotEmpty ||
          uri.hasQuery ||
          uri.hasFragment ||
          uri.port != 443) {
        continue;
      }
      final cached = await _readNotesCache(uri);
      if (cached != null) {
        return refresh ? await _loadNotes() ?? cached : cached;
      }
    }
    return refresh ? _loadNotes() : null;
  }

  Future<ReleaseNoteCatalog?> _loadNotes() async {
    _notesValidatedThisLoad.clear();
    final configured = Uri.tryParse(mobileReleaseNotesUrl);
    final urls = [configured, Uri.tryParse(githubReleaseNotesUrl)];
    ReleaseNoteCatalog? lastCache;
    for (final uri in urls) {
      if (uri == null ||
          uri.scheme != 'https' ||
          uri.host.isEmpty ||
          uri.userInfo.isNotEmpty ||
          uri.hasQuery ||
          uri.hasFragment ||
          uri.port != 443) {
        continue;
      }
      final cached = await _loadNotesUrl(uri);
      if (cached != null && _notesValidatedThisLoad.contains(uri.toString())) {
        return cached;
      }
      lastCache ??= _notesByUrl[uri.toString()];
    }
    return lastCache;
  }

  Future<ReleaseNoteCatalog?> _loadNotesUrl(Uri uri) async {
    final url = uri.toString();
    final prefs = notesPreferences;
    await _readNotesCache(uri);
    final cacheKey = 'mobile.releaseNotes.${Uri.encodeComponent(url)}';
    final headers = <String, dynamic>{'Accept': 'application/json'};
    if (_etagsByUrl[url] case final etag?) headers['If-None-Match'] = etag;
    try {
      final cancel = CancelToken();
      Future<Response<List<int>>> request(
        String address,
        Map<String, dynamic> requestHeaders,
      ) => _notesDio.get<List<int>>(
        address,
        cancelToken: cancel,
        onReceiveProgress: (received, _) {
          if (received > 1024 * 1024) {
            cancel.cancel('Release notes exceed 1 MiB');
          }
        },
        options: _notesOptions(requestHeaders),
      );
      var response = await request(url, headers);
      for (
        var redirects = 0;
        response.statusCode != 200 &&
            response.statusCode != 304 &&
            redirects < 2;
        redirects++
      ) {
        final location = response.headers.value('location');
        final target = location == null
            ? null
            : uri.resolveUri(Uri.tryParse(location) ?? Uri()).normalizePath();
        if (target == null ||
            target.scheme != 'https' ||
            target.port != 443 ||
            !const {
              'github.com',
              'release-assets.githubusercontent.com',
              'objects.githubusercontent.com',
            }.contains(target.host) ||
            target.userInfo.isNotEmpty ||
            target.fragment.isNotEmpty) {
          return _notesByUrl[url];
        }
        response = await request(target.toString(), {
          'Accept': 'application/json',
        });
      }
      if (response.statusCode != 200 && response.statusCode != 304) {
        return _notesByUrl[url];
      }
      if (response.statusCode == 304) {
        if (_notesByUrl[url] != null) _notesValidatedThisLoad.add(url);
        return _notesByUrl[url];
      }
      final bytes = response.data;
      if (bytes == null || bytes.length > 1024 * 1024) {
        throw const FormatException('Notes too large');
      }
      final decoded = jsonDecode(utf8.decode(bytes));
      final catalog = ReleaseNoteCatalog.decode(decoded);
      if (catalog == null) throw const FormatException('Invalid notes');
      final etag = response.headers.value('etag');
      if (etag == null) {
        _etagsByUrl.remove(url);
      } else {
        _etagsByUrl[url] = etag;
      }
      _notesByUrl[url] = catalog;
      _notesValidatedThisLoad.add(url);
      final encoded = jsonEncode(decoded);
      if (encoded.length <= 1024 * 1024 && prefs != null) {
        await prefs.setString('$cacheKey.json', encoded);
        if (etag != null) {
          await prefs.setString('$cacheKey.etag', etag);
        } else {
          await prefs.remove('$cacheKey.etag');
        }
      }
      return catalog;
    } catch (_) {
      return _notesByUrl[url];
    }
  }

  Options _notesOptions(Map<String, dynamic> headers) => Options(
    headers: headers,
    receiveTimeout: const Duration(seconds: 8),
    responseType: ResponseType.bytes,
    followRedirects: false,
    maxRedirects: 0,
    validateStatus: (status) => status != null && status >= 200 && status < 400,
  );

  Future<ReleaseNoteCatalog?> _readNotesCache(Uri uri) async {
    final url = uri.toString();
    if (_notesByUrl.containsKey(url)) return _notesByUrl[url];
    final prefs = notesPreferences;
    if (prefs == null) return null;
    final cacheKey = 'mobile.releaseNotes.${Uri.encodeComponent(url)}';
    final encoded = prefs.getString('$cacheKey.json');
    if (encoded == null) {
      await prefs.remove('$cacheKey.etag');
      return null;
    }
    try {
      if (encoded.length > 1024 * 1024) {
        throw const FormatException('Notes too large');
      }
      final catalog = ReleaseNoteCatalog.fromJson(encoded);
      _notesByUrl[url] = catalog;
      final etag = prefs.getString('$cacheKey.etag');
      if (etag != null) _etagsByUrl[url] = etag;
      return catalog;
    } catch (_) {
      await prefs.remove('$cacheKey.json');
      await prefs.remove('$cacheKey.etag');
      return null;
    }
  }

  Future<Duration?> _probe(
    Uri uri,
    AndroidRelease release,
    CancelToken cancel,
  ) async {
    final token = CancelToken();
    cancel.whenCancel.then((_) => token.cancel());
    final watch = Stopwatch()..start();
    final deadline = Timer(const Duration(seconds: 8), () => token.cancel());
    try {
      final response = await _dio.get<ResponseBody>(
        uri.toString(),
        cancelToken: token,
        options: Options(
          responseType: ResponseType.stream,
          headers: {'Range': 'bytes=0-65535'},
          receiveTimeout: const Duration(seconds: 5),
          validateStatus: (status) => status == 206,
          followRedirects: true,
          maxRedirects: 5,
        ),
      );
      final range = response.headers.value('content-range') ?? '';
      final expected = release.size < 65536 ? release.size : 65536;
      if (range != 'bytes 0-${expected - 1}/${release.size}') return null;
      var received = 0;
      await for (final chunk in response.data!.stream.timeout(
        const Duration(seconds: 5),
      )) {
        received += chunk.length;
        if (received > expected) return null;
      }
      return received == expected ? watch.elapsed : null;
    } catch (_) {
      return null;
    } finally {
      deadline.cancel();
      token.cancel();
    }
  }

  Future<File> download(
    AndroidRelease release,
    Directory directory, {
    required CancelToken cancel,
    required void Function(int received, int total) onProgress,
  }) async {
    await directory.create(recursive: true);
    final sources = [
      release.url,
      for (final prefix in releaseMirrors) Uri.parse('$prefix${release.url}'),
    ];
    final measurements = await Future.wait([
      for (final source in sources) _probe(source, release, cancel),
    ]);
    if (cancel.isCancelled) {
      throw DioException(
        requestOptions: RequestOptions(),
        type: DioExceptionType.cancel,
      );
    }
    final ranked = <(Uri, Duration)>[
      for (var i = 0; i < sources.length; i++)
        if (measurements[i] != null) (sources[i], measurements[i]!),
    ]..sort((a, b) => a.$2.compareTo(b.$2));
    final candidates = [for (final entry in ranked) entry.$1];
    if (!candidates.contains(release.url)) candidates.add(release.url);
    final target = File('${directory.path}/yourtj-${release.buildNumber}.apk');
    final partial = File('${target.path}.part');
    for (final source in candidates) {
      final transfer = CancelToken();
      final deadline = Timer(
        const Duration(minutes: 10),
        () => transfer.cancel(),
      );
      cancel.whenCancel.then((_) => transfer.cancel());
      try {
        if (cancel.isCancelled) break;
        await _dio.download(
          source.toString(),
          partial.path,
          cancelToken: transfer,
          options: Options(followRedirects: true, maxRedirects: 5),
          onReceiveProgress: (received, total) {
            if (received > release.size ||
                (total > 0 && total != release.size)) {
              transfer.cancel('Unexpected asset size');
            } else {
              onProgress(received, release.size);
            }
          },
        );
        if (await partial.length() != release.size ||
            (await sha256.bind(partial.openRead()).first).toString() !=
                release.sha256Digest) {
          throw const FormatException('APK integrity check failed');
        }
        if (cancel.isCancelled) break;
        return await partial.rename(target.path);
      } catch (_) {
        if (cancel.isCancelled) break;
        // A slow or corrupt public mirror never prevents trying the next source.
      } finally {
        deadline.cancel();
        transfer.cancel();
        if (await partial.exists()) await partial.delete();
      }
    }
    throw StateError(
      cancel.isCancelled
          ? 'Update cancelled'
          : 'No verified APK could be downloaded',
    );
  }
}

import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';

/// Disposable public image bytes only. Credentials, uploads and local attachments
/// never enter this directory. Public unauthenticated media is keyed by
/// `(apiOrigin, url)` so account sign-in and UI language switches reuse the
/// same disk bytes without duplication.
class MediaRepository extends ChangeNotifier {
  MediaRepository({
    Future<Directory> Function()? directory,
    Future<Directory?> Function()? legacyDirectory,
    Dio? dio,
    DateTime Function()? now,
    this.budgetBytes = 192 * 1024 * 1024,
    this.maxDownloadBytes = 32 * 1024 * 1024,
    this.maxExternalPersistBytes = 8 * 1024 * 1024,
    this.maxConcurrentDownloads = 6,
  }) : _directory =
           directory ??
           (() async => Directory(
             '${(await getApplicationCacheDirectory()).path}/yourtj_media_v1',
           )),
       _legacyDirectory =
           legacyDirectory ??
           (directory == null
               ? (() async => Directory(
                   '${(await getTemporaryDirectory()).path}/cacheimage',
                 ))
               : (() async => null)),
       _ownsDio = dio == null,
       _dio =
           dio ??
           Dio(
             BaseOptions(
               connectTimeout: const Duration(seconds: 15),
               receiveTimeout: const Duration(seconds: 30),
             ),
           ),
       _now = now ?? DateTime.now;

  final Future<Directory> Function() _directory;
  final Future<Directory?> Function() _legacyDirectory;
  final Dio _dio;
  final bool _ownsDio;
  final DateTime Function() _now;
  final int budgetBytes;
  final int maxDownloadBytes;
  final int maxExternalPersistBytes;
  final int maxConcurrentDownloads;

  /// Remote image hosts routinely gate on the request looking like a browser
  /// fetch; Dart's default `Dart/x.y (dart:io)` user agent is rejected by some
  /// hosts that serve the same URL fine in a web view. Deliberately no Referer:
  /// an empty one is usually the permissive branch, a wrong one trips hotlink
  /// protection.
  static const String _browserLikeUserAgent =
      'Mozilla/5.0 (Linux; Android 14; YourTJ-Hub/1.0) AppleWebKit/537.36 '
      '(KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36';
  static const String _mediaAcceptHeader = 'image/*,*/*;q=0.8';
  static const int _maxMemoryIdentities = 500;
  static const Duration _indexFlushDelay = Duration(seconds: 30);

  static const Set<String> _signedQueryParams = {
    'signature',
    'sig',
    'sign',
    'x-amz-signature',
    'x-amz-credential',
    'x-amz-security-token',
    'x-goog-signature',
    'ossaccesskeyid',
    'q-signature',
    'q-sign-algorithm',
    'token',
    'access_token',
    'auth_key',
  };

  final Map<String, Map<String, dynamic>> _entries = {};
  final Map<String, ({Object token, DateTime expiresAt})> _memoryIdentities = {};
  final Map<String, Future<Uint8List>> _pending = {};
  final Set<CancelToken> _tokens = {};
  final List<Completer<void>> _waiters = [];
  Future<void> _tail = Future.value();
  Directory? _dir;
  Timer? _indexFlushTimer;
  bool _indexDirty = false;
  int _generation = 0;
  int _activeDownloads = 0;
  int _suspensions = 0;
  bool get isSuspended => _suspensions > 0;
  bool _disposed = false;
  DateTime? _lastSweep;
  int get generation => _generation;
  bool isCurrent(int generation) => !_disposed && generation == _generation;

  String _cacheKey(String apiOrigin, String url) =>
      sha256.convert(utf8.encode(jsonEncode([apiOrigin, url]))).toString();

  void _guard(int generation) {
    if (!isCurrent(generation) || isSuspended) {
      throw StateError('Media request invalidated');
    }
  }

  Future<T> _locked<T>(Future<T> Function() work) {
    final result = _tail.then((_) => work());
    _tail = result.then<void>((_) {}, onError: (Object _, StackTrace _) {});
    return result;
  }

  Future<void> _open() async {
    if (_dir != null) return;
    final dir = await _directory();
    await dir.create(recursive: true);
    final index = File('${dir.path}/index.json');
    if (await index.exists()) {
      try {
        final json =
            jsonDecode(await index.readAsString()) as Map<String, dynamic>;
        if (json['version'] != 1) {
          throw const FormatException('Unknown media schema');
        }
        for (final entry in (json['entries'] as Map<String, dynamic>).entries) {
          final value = entry.value as Map<String, dynamic>;
          if (!RegExp(r'^[0-9a-f]{64}$').hasMatch(entry.key) ||
              value['bytes'] is! int ||
              value['accessed'] is! int ||
              value['freshUntil'] is! int ||
              value['cacheControl'] is! String) {
            throw const FormatException('Invalid media index');
          }
          _entries[entry.key] = value;
        }
      } catch (_) {
        _entries.clear(); // Only disposable data may be rebuilt.
      }
    }
    _dir = dir;
    // Retire exactly ExtendedImage's old directory; never sweep the surrounding
    // temporary directory, where unsent uploads and picker files may live.
    final legacy = await _legacyDirectory();
    if (legacy != null && await legacy.exists()) {
      await legacy.delete(recursive: true);
    }
    await _sweep();
  }

  File _file(String key) => File('${_dir!.path}/$key');
  /// Returns whether an index entry existed, so callers can skip an index
  /// rewrite when only an orphan file was removed.
  Future<bool> _delete(String key) async {
    final existed = _entries.remove(key) != null;
    _memoryIdentities.remove(key);
    final file = _file(key);
    if (await file.exists()) await file.delete();
    return existed;
  }

  void _scheduleIndexFlush() {
    _indexDirty = true;
    if (_disposed || _indexFlushTimer != null) return;
    _indexFlushTimer = Timer(_indexFlushDelay, () {
      _indexFlushTimer = null;
      if (_disposed || !_indexDirty || _dir == null) return;
      unawaited(
        _locked(() async {
          if (_disposed || !_indexDirty || _dir == null) return;
          await _saveIndex();
        }).catchError((Object _) {}),
      );
    });
  }

  Future<void> _saveIndex() async {
    _indexFlushTimer?.cancel();
    _indexFlushTimer = null;
    _indexDirty = false;
    final index = File('${_dir!.path}/index.json');
    if (_entries.isEmpty) {
      if (await index.exists()) await index.delete();
      return;
    }
    final temporary = File('${index.path}.tmp');
    await temporary.writeAsString(
      jsonEncode({'version': 1, 'entries': _entries}),
      flush: true,
    );
    await temporary.rename(index.path);
  }

  Future<int> _diskBytes() async {
    var total = 0;
    await for (final file in _dir!.list(followLinks: false)) {
      if (file is File) total += await file.length();
    }
    final legacy = await _legacyDirectory();
    if (legacy != null && await legacy.exists()) {
      await for (final file in legacy.list(
        recursive: true,
        followLinks: false,
      )) {
        if (file is File) total += await file.length();
      }
    }
    return total;
  }

  Future<void> _sweep({bool force = false}) async {
    if (!force &&
        _lastSweep != null &&
        _now().difference(_lastSweep!) < const Duration(hours: 1)) {
      return;
    }
    _lastSweep = _now();
    final cutoff = _now()
        .subtract(const Duration(days: 14))
        .millisecondsSinceEpoch;
    for (final key in _entries.keys.toList()) {
      final file = _file(key);
      if ((_entries[key]!['accessed'] as int) < cutoff ||
          !await file.exists() ||
          await file.length() != _entries[key]!['bytes']) {
        await _delete(key);
      }
    }
    await for (final file in _dir!.list(followLinks: false)) {
      final name = file.uri.pathSegments.last;
      if (file is File && name != 'index.json' && !_entries.containsKey(name)) {
        await file.delete();
      }
    }
    await _saveIndex();
    if (await _diskBytes() > budgetBytes) {
      await _trim((budgetBytes * .8).floor());
    }
  }

  Future<void> _trim(int target) async {
    // Measure actual files once, including unindexed/temporary and legacy bytes.
    // All repository disk mutations share this lock, so each successful removal
    // can update the total without rescanning the directory for every entry.
    var remaining = await _diskBytes();
    final index = File('${_dir!.path}/index.json');
    final storedIndexBytes = await index.exists() ? await index.length() : 0;
    var indexBytes = _entries.isEmpty
        ? 0
        : utf8.encode(jsonEncode({'version': 1, 'entries': _entries})).length;
    remaining += indexBytes - storedIndexBytes;
    final oldest = _entries.keys.toList()
      ..sort(
        (a, b) => (_entries[a]!['accessed'] as int).compareTo(
          _entries[b]!['accessed'] as int,
        ),
      );
    for (final key in oldest) {
      if (remaining <= target) break;
      final entry = _entries[key]!;
      final file = _file(key);
      final removedBytes = await file.exists() ? await file.length() : 0;
      await _delete(key);
      // Removing one JSON member also removes one comma, except that an empty
      // index is deleted entirely. Count UTF-8, including non-ASCII validators.
      final removedIndexBytes = _entries.isEmpty
          ? indexBytes
          : utf8.encode(jsonEncode({key: entry})).length - 2 + 1;
      indexBytes -= removedIndexBytes;
      remaining -= removedBytes + removedIndexBytes;
    }
    // A crash mid-batch leaves only missing-file index entries, which _sweep
    // already repairs. Commit once instead of rewriting the full map per file.
    await _saveIndex();
  }

  Future<int> usageBytes() => _locked(() async {
    await _open();
    await _sweep(force: true);
    return _diskBytes();
  });

  /// Synchronous fence precedes any queued disk deletion or scope transition.
  void invalidate() {
    if (_disposed) return;
    _generation++;
    _memoryIdentities.clear();
    for (final token in _tokens.toList()) {
      token.cancel('Media scope changed');
    }
    notifyListeners();
  }

  /// Begin a wider clear transaction before its journal or other stores await.
  /// Nested users must pair this with resume, including failed transactions.
  void suspend() {
    _suspensions++;
    invalidate();
  }

  /// Deliberately no notification: clearing must not itself start a new fetch.
  void resume() {
    if (_suspensions > 0) _suspensions--;
  }

  Future<void> clear() async {
    suspend();
    try {
      await _locked(() async {
        _indexFlushTimer?.cancel();
        _indexFlushTimer = null;
        _indexDirty = false;
        await _open();
        _entries.clear();
        _memoryIdentities.clear();
        await for (final file in _dir!.list(followLinks: false)) {
          await file.delete(recursive: true);
        }
        final legacy = await _legacyDirectory();
        if (legacy != null && await legacy.exists()) {
          await legacy.delete(recursive: true);
        }
      });
    } finally {
      resume();
    }
  }

  void _rememberMemoryIdentity(String key, Duration ttl) {
    if (ttl <= Duration.zero) {
      _memoryIdentities.remove(key);
      return;
    }
    _memoryIdentities.remove(key);
    if (_memoryIdentities.length >= _maxMemoryIdentities) {
      _memoryIdentities.remove(_memoryIdentities.keys.first);
    }
    _memoryIdentities[key] = (
      token: (key, _generation, Object()),
      expiresAt: _now().add(ttl),
    );
  }

  /// Fresh public bytes or memory-only unauthenticated bytes may reuse a
  /// decoded frame without another disk read. Private, credentialed, or
  /// expired entries deliberately have no reusable identity.
  Object? decodedIdentity(
    String url, {
    required String scopeKey,
    required String apiOrigin,
  }) {
    final key = _cacheKey(apiOrigin, url);
    final now = _now();
    final entry = _entries[key];
    if (entry != null &&
        (entry['freshUntil'] as int) > now.millisecondsSinceEpoch) {
      return (key, entry['freshUntil'], entry['etag']);
    }
    final mem = _memoryIdentities[key];
    if (mem != null) {
      if (mem.expiresAt.isAfter(now)) {
        return mem.token;
      }
      _memoryIdentities.remove(key);
    }
    return null;
  }

  /// Returns fresh persisted bytes from disk without triggering a network call.
  Future<Uint8List?> readCachedBytes(
    String url, {
    required String apiOrigin,
  }) async {
    if (_disposed || isSuspended) return null;
    final generation = _generation;
    final key = _cacheKey(apiOrigin, url);
    File? file;
    int? expectedBytes;
    await _locked(() async {
      if (!isCurrent(generation) || isSuspended) return;
      await _open();
      final entry = _entries[key];
      if (entry == null) return;
      if ((entry['freshUntil'] as int) <= _now().millisecondsSinceEpoch) return;
      file = _file(key);
      expectedBytes = entry['bytes'] as int;
    });
    if (file == null || !isCurrent(generation) || isSuspended) return null;
    try {
      final bytes = await file!.readAsBytes();
      if (bytes.isEmpty ||
          (expectedBytes != null && bytes.length != expectedBytes)) {
        return null;
      }
      return bytes;
    } on FileSystemException {
      return null;
    }
  }

  /// Decode failures evict only their own generation, so retry can refetch a
  /// malformed cached image without deleting a newer account's response.
  Future<void> discard(
    String url, {
    required String scopeKey,
    required String apiOrigin,
    required int generation,
  }) => _locked(() async {
    if (!isCurrent(generation) || isSuspended) return;
    await _open();
    final key = _cacheKey(apiOrigin, url);
    await _delete(key);
    await _saveIndex();
  });

  Future<Uint8List> load(
    String url, {
    required String scopeKey,
    required String apiOrigin,
    Set<String>? allowedOrigins,
    Future<String?> Function()? readAccessToken,
  }) {
    final generation = _generation;
    final key = _cacheKey(apiOrigin, url);
    final policyKey = allowedOrigins == null
        ? ''
        : (allowedOrigins.toList()..sort()).join(',');
    final resolvedScope = !scopeKey.startsWith('pending:');
    final flightKey = '$generation:$key:$policyKey:${resolvedScope ? 1 : 0}';
    return _pending.putIfAbsent(flightKey, () {
      final result = _load(
        url,
        apiOrigin,
        key,
        generation,
        resolvedScope,
        allowedOrigins,
        readAccessToken,
      );
      result.then<void>(
        (_) {
          _pending.remove(flightKey);
        },
        onError: (Object _, StackTrace _) {
          _pending.remove(flightKey);
        },
      );
      return result;
    });
  }

  bool _hasSignedQueryParams(Uri uri) => uri.queryParameters.keys.any(
    (key) => _signedQueryParams.contains(key.toLowerCase()),
  );

  bool _persistableAddress(Uri uri) =>
      uri.scheme == 'https' &&
      uri.host.isNotEmpty &&
      uri.userInfo.isEmpty &&
      !uri.hasFragment &&
      !_hasSignedQueryParams(uri);

  /// Only this site's own uploads may be retried with the session: pending
  /// content images answer 404 to anonymous readers (issue #975).
  bool _privateUploadAddress(Uri uri, Uri origin) =>
      uri.origin == origin.origin &&
      uri.userInfo.isEmpty &&
      !uri.hasQuery &&
      !uri.hasFragment &&
      uri.path.startsWith('/file/img/');

  /// Ignore transport-only `Vary: Accept-Encoding` because Dio decompresses
  /// responses before storing raw image bytes. Reject any other Vary field.
  bool _hasUnsafeVary(Headers headers) {
    final values = headers['vary'];
    if (values == null || values.isEmpty) return false;
    for (final raw in values) {
      for (final part in raw.split(',')) {
        final token = part.trim().toLowerCase();
        if (token.isEmpty || token == 'accept-encoding') continue;
        return true;
      }
    }
    return false;
  }

  bool _restrictedRedirect(Headers headers) {
    final control = (headers['cache-control']?.join(',') ?? '').toLowerCase();
    return control.split(',').any((part) {
          final directive = part.trim().split('=').first.trim();
          return const ['private', 'no-store', 'no-cache'].contains(directive);
        }) ||
        headers['set-cookie'] != null ||
        _hasUnsafeVary(headers);
  }

  Future<Uint8List> _load(
    String url,
    String apiOrigin,
    String key,
    int generation,
    bool resolvedScope,
    Set<String>? allowedOrigins,
    Future<String?> Function()? readAccessToken,
  ) async {
    _guard(generation);
    var uri = Uri.parse(url);
    final origin = Uri.parse(apiOrigin);
    if (!['https', 'http'].contains(uri.scheme)) {
      throw const FormatException('Unsupported image URL');
    }
    void checkOrigin() {
      if (allowedOrigins != null &&
          (uri.userInfo.isNotEmpty || !allowedOrigins.contains(uri.origin))) {
        throw const FormatException('Untrusted image origin');
      }
    }

    checkOrigin();
    var persistable = _persistableAddress(uri);
    var sameOriginChain = uri.origin == origin.origin;
    // Userinfo must never become implicit HTTP Basic authentication.
    uri = uri.replace(userInfo: '');
    Map<String, dynamic>? previous;
    File? candidateFile;
    int? expectedBytes;
    var isFreshOnDisk = false;
    await _locked(() async {
      await _open();
      _guard(generation);
      await _sweep();
      final entry = _entries[key];
      if (entry != null) {
        if (!persistable) {
          await _delete(key);
          await _saveIndex();
          return;
        }
        previous = Map<String, dynamic>.of(entry);
        candidateFile = _file(key);
        expectedBytes = entry['bytes'] as int;
        final nowMs = _now().millisecondsSinceEpoch;
        if ((entry['freshUntil'] as int) > nowMs) {
          isFreshOnDisk = true;
          if (nowMs - (entry['accessed'] as int) >= 60 * 1000) {
            entry['accessed'] = nowMs;
            _scheduleIndexFlush();
          }
        }
      }
    });
    _guard(generation);
    Uint8List? local;
    if (candidateFile != null) {
      try {
        final bytes = await candidateFile!.readAsBytes();
        if (expectedBytes != null && bytes.length != expectedBytes) {
          throw const FileSystemException('Cached media size mismatch');
        }
        local = bytes;
      } on FileSystemException {
        await _locked(() async {
          if (!isCurrent(generation) || isSuspended) return;
          await _delete(key);
          await _saveIndex();
        });
        previous = null;
        isFreshOnDisk = false;
      }
    }
    _guard(generation);
    if (local != null && isFreshOnDisk) {
      return local;
    }
    if (_activeDownloads >= maxConcurrentDownloads) {
      final waiter = Completer<void>();
      _waiters.add(waiter);
      await waiter.future;
    } else {
      _activeDownloads++;
    }
    final token = CancelToken();
    _tokens.add(token);
    final deadline = Timer(
      const Duration(seconds: 60),
      () => token.cancel('Media request deadline'),
    );
    try {
      _guard(generation);
      Response<ResponseBody>? response;
      var redirected = false;
      var usedCredentials = false;
      for (var redirect = 0; redirect <= 5; redirect++) {
        checkOrigin();
        response = await _dio.getUri<ResponseBody>(
          uri,
          cancelToken: token,
          options: Options(
            responseType: ResponseType.stream,
            followRedirects: false,
            validateStatus: (_) => true,
            headers: {
              'User-Agent': _browserLikeUserAgent,
              'Accept': _mediaAcceptHeader,
              if (redirect == 0 &&
                  local != null &&
                  previous != null &&
                  previous!['etag'] is String)
                'If-None-Match': previous!['etag'],
              if (redirect == 0 &&
                  local != null &&
                  previous != null &&
                  previous!['lastModified'] is String)
                'If-Modified-Since': previous!['lastModified'],
            },
          ),
        );
        _guard(generation);
        if ([301, 302, 303, 307, 308].contains(response.statusCode)) {
          await response.data?.stream.listen((_) {}).cancel();
          final location = response.headers.value('location');
          if (location == null || redirect == 5) {
            throw StateError('Invalid media redirect');
          }
          persistable = persistable && !_restrictedRedirect(response.headers);
          if (!persistable) {
            await _locked(() async {
              _guard(generation);
              await _delete(key);
              await _saveIndex();
            });
          }
          redirected = true;
          uri = uri.resolve(location);
          persistable = persistable && _persistableAddress(uri);
          sameOriginChain = sameOriginChain && uri.origin == origin.origin;
          if (!['https', 'http'].contains(uri.scheme)) {
            throw const FormatException('Unsupported media redirect');
          }
          uri = uri.replace(userInfo: '');
          continue;
        }
        break;
      }
      // Public images never carry credentials. A 404 from this site's own
      // upload path may be the author's or a reviewer's pending image: retry
      // once with the session, never across redirects, and never persist it.
      if (response!.statusCode == 404 &&
          !redirected &&
          resolvedScope &&
          readAccessToken != null &&
          _privateUploadAddress(uri, origin)) {
        final accessToken = await readAccessToken();
        _guard(generation);
        if (accessToken != null && accessToken.isNotEmpty) {
          await response.data?.stream.listen((_) {}).cancel();
          persistable = false;
          usedCredentials = true;
          response = await _dio.getUri<ResponseBody>(
            uri,
            cancelToken: token,
            options: Options(
              responseType: ResponseType.stream,
              followRedirects: false,
              validateStatus: (_) => true,
              headers: {
                'User-Agent': _browserLikeUserAgent,
                'Accept': _mediaAcceptHeader,
                'Authorization': 'Bearer $accessToken',
              },
            ),
          );
          _guard(generation);
        }
      }
      final status = response.statusCode;
      if (status != 200 && !(status == 304 && local != null)) {
        await response.data?.stream.listen((_) {}).cancel();
        await _locked(() async {
          _guard(generation);
          await _delete(key);
          await _saveIndex();
        });
        throw StateError('Image request failed ($status)');
      }
      final headers = response.headers;
      final control =
          headers['cache-control']?.join(',') ??
          (status == 304
              ? (previous == null ? null : previous!['cacheControl'] as String?)
              : null) ??
          '';
      final directives = control
          .toLowerCase()
          .split(',')
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toList();
      final directiveNames = directives
          .map((d) => d.split('=').first.trim())
          .toList();
      final ageDirectives = directives
          .where((d) => d.split('=').first.trim() == 'max-age')
          .toList();
      final maxAgeText = ageDirectives.length == 1
          ? ageDirectives.single
                .split('=')
                .skip(1)
                .join('=')
                .trim()
                .replaceAll('"', '')
          : '';
      final hasMaxAgeDirective = ageDirectives.isNotEmpty;
      final maxAge = RegExp(r'^\d+$').hasMatch(maxAgeText)
          ? int.tryParse(maxAgeText) ?? 0
          : 0;
      var age = int.tryParse(headers.value('age') ?? '0') ?? maxAge;
      if (age < 0) age = maxAge;
      try {
        final dateAge = _now()
            .difference(HttpDate.parse(headers.value('date') ?? ''))
            .inSeconds;
        if (dateAge > age) age = dateAge;
      } catch (_) {}

      final hasPublic = directiveNames.contains('public');
      final hasPrivate = directiveNames.contains('private');
      final hasNoStore = directiveNames.contains('no-store');
      final hasNoCache = directiveNames.contains('no-cache');
      final hasSetCookie = headers['set-cookie'] != null;
      final hasUnsafeVary = _hasUnsafeVary(headers);
      final etag =
          headers.value('etag') ??
          (status == 304 && previous != null
              ? previous!['etag'] as String?
              : null);
      final lastModified =
          headers.value('last-modified') ??
          (status == 304 && previous != null
              ? previous!['lastModified'] as String?
              : null);
      final hasValidator =
          (etag != null && etag.isNotEmpty) ||
          (lastModified != null && lastModified.isNotEmpty);

      final isImageResponse =
          status == 304 ||
          (headers.value('content-type') ?? '').toLowerCase().startsWith(
            'image/',
          );
      final freshSeconds = hasNoCache
          ? 0
          : (maxAge - age).clamp(0, 365 * 24 * 60 * 60);
      final freshUntil = _now().add(Duration(seconds: freshSeconds));

      // Same-origin freshness requires explicit `public`; external HTTPS CDNs
      // often omit `public` while sending `max-age`, which is safe because no
      // credentials or cookies are ever attached to public media requests.
      // RFC 9111 `no-cache` with a validator (ETag / Last-Modified) may be
      // stored on disk with freshUntil = 0 for 304 revalidation (e.g. Wiki).
      final freshPersistable =
          !hasNoCache && maxAge > age && (hasPublic || !sameOriginChain);
      final revalidatePersistable = hasNoCache && hasValidator;
      final persist =
          persistable &&
          !usedCredentials &&
          !hasPrivate &&
          !hasNoStore &&
          !hasSetCookie &&
          !hasUnsafeVary &&
          isImageResponse &&
          (freshPersistable || revalidatePersistable);

      // A changed restriction revokes the previous public entry immediately,
      // even if reading the replacement subsequently fails or exceeds limits.
      // Skip the rewrite when no entry existed: the index already matches.
      if (!persist) {
        await _locked(() async {
          _guard(generation);
          if (await _delete(key)) {
            await _saveIndex();
          }
        });
      }
      final bytes = BytesBuilder(copy: false);
      if (status == 304) {
        await response.data?.stream.listen((_) {}).cancel();
        bytes.add(local!);
      } else {
        final declared = int.tryParse(headers.value('content-length') ?? '');
        if (declared != null && declared > maxDownloadBytes) {
          token.cancel('Image too large');
          throw StateError('Image exceeds download limit');
        }
        await for (final chunk in response.data!.stream) {
          _guard(generation);
          if (bytes.length + chunk.length > maxDownloadBytes) {
            token.cancel('Image too large');
            throw StateError('Image exceeds download limit');
          }
          bytes.add(chunk);
        }
      }
      final result = bytes.takeBytes();
      if (result.isEmpty) throw StateError('Empty image');
      _guard(generation);
      final maxPersistSize = sameOriginChain
          ? budgetBytes
          : math.min(budgetBytes, maxExternalPersistBytes);
      var persistedOnDisk = false;
      await _locked(() async {
        _guard(generation);
        final removed = await _delete(key);
        final entry = <String, dynamic>{
          'bytes': result.length,
          'accessed': _now().millisecondsSinceEpoch,
          'freshUntil': freshUntil.millisecondsSinceEpoch,
          'cacheControl': control,
          'etag': etag,
          'lastModified': ?lastModified,
        };
        int indexSize(Map<String, Map<String, dynamic>> entries) =>
            utf8.encode(jsonEncode({'version': 1, 'entries': entries})).length;
        if (persist &&
            result.length <= maxPersistSize &&
            result.length + indexSize({key: entry}) <= budgetBytes) {
          // Include the atomic replacement index in the peak write budget.
          // An old index can coexist with index.json.tmp until rename finishes.
          final reserve = result.length + indexSize({..._entries, key: entry});
          if (await _diskBytes() + reserve > budgetBytes ||
              _entries.length >= 5000) {
            await _trim(
              ((budgetBytes * .8).floor() - reserve).clamp(0, budgetBytes),
            );
          }
          final temporary = File('${_file(key).path}.tmp');
          await temporary.writeAsBytes(result, flush: true);
          if (!isCurrent(generation)) {
            await temporary.delete();
            _guard(generation);
          }
          await temporary.rename(_file(key).path);
          _entries[key] = entry;
          persistedOnDisk = true;
          await _saveIndex();
          if (await _diskBytes() > budgetBytes) {
            await _trim((budgetBytes * .8).floor());
          }
        } else if (removed) {
          await _saveIndex();
        }
      });
      _guard(generation);
      // Non-credentialed images that are not fresh on disk (signed URLs,
      // external images without Cache-Control, or revalidated no-cache assets)
      // stay memory-only in Flutter's ImageCache within the current generation,
      // unless explicitly private, restricted, same-origin no-store, or already
      // expired per HTTP Age >= max-age.
      final freshOnDiskNow = persistedOnDisk && freshSeconds > 0;
      final explicitlyExpired = hasMaxAgeDirective && maxAge <= age;
      final memoryEligible =
          !freshOnDiskNow &&
          !usedCredentials &&
          !hasPrivate &&
          !hasSetCookie &&
          !hasUnsafeVary &&
          !(hasNoStore && sameOriginChain) &&
          !explicitlyExpired &&
          isImageResponse;
      if (memoryEligible) {
        final Duration memoryTtl;
        if (hasNoCache) {
          memoryTtl = const Duration(seconds: 30);
        } else if (maxAge > age) {
          memoryTtl = Duration(seconds: (maxAge - age).clamp(1, 300));
        } else {
          memoryTtl = const Duration(minutes: 5);
        }
        _rememberMemoryIdentity(key, memoryTtl);
      } else {
        _memoryIdentities.remove(key);
      }
      return result;
    } finally {
      deadline.cancel();
      _tokens.remove(token);
      if (_waiters.isNotEmpty) {
        _waiters.removeAt(0).complete();
      } else {
        _activeDownloads--;
      }
    }
  }

  @override
  void dispose() {
    _indexFlushTimer?.cancel();
    _indexFlushTimer = null;
    invalidate();
    _disposed = true;
    if (_ownsDio) _dio.close(force: true);
    super.dispose();
  }
}

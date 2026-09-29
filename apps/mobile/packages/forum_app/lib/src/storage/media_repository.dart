import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';

/// Disposable public image bytes only. Credentials, uploads and local attachments
/// never enter this directory. All accounts share one byte budget and index.
class MediaRepository extends ChangeNotifier {
  MediaRepository({
    Future<Directory> Function()? directory,
    Future<Directory?> Function()? legacyDirectory,
    Dio? dio,
    DateTime Function()? now,
    this.budgetBytes = 192 * 1024 * 1024,
    this.maxDownloadBytes = 32 * 1024 * 1024,
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
  final Map<String, Map<String, dynamic>> _entries = {};
  final Map<String, Future<Uint8List>> _pending = {};
  final Set<CancelToken> _tokens = {};
  final List<Completer<void>> _waiters = [];
  Future<void> _tail = Future.value();
  Directory? _dir;
  int _generation = 0;
  int _activeDownloads = 0;
  int _suspensions = 0;
  bool get isSuspended => _suspensions > 0;
  bool _disposed = false;
  DateTime? _lastSweep;
  int get generation => _generation;
  bool isCurrent(int generation) => !_disposed && generation == _generation;

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
  Future<void> _delete(String key) async {
    _entries.remove(key);
    final file = _file(key);
    if (await file.exists()) await file.delete();
  }

  Future<void> _saveIndex() async {
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
        await _open();
        _entries.clear();
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

  /// Called only after load/revalidation, before consulting decoded ImageCache.
  /// Missing/private/expired entries deliberately have no reusable identity.
  Object? decodedIdentity(
    String url, {
    required String scopeKey,
    required String apiOrigin,
  }) {
    final key = sha256
        .convert(utf8.encode(jsonEncode([apiOrigin, scopeKey, url])))
        .toString();
    final entry = _entries[key];
    if (entry == null ||
        (entry['freshUntil'] as int) <= _now().millisecondsSinceEpoch) {
      return null;
    }
    return (key, entry['freshUntil'], entry['etag']);
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
    final key = sha256
        .convert(utf8.encode(jsonEncode([apiOrigin, scopeKey, url])))
        .toString();
    await _delete(key);
    await _saveIndex();
  });

  Future<Uint8List> load(
    String url, {
    required String scopeKey,
    required String apiOrigin,
  }) {
    final generation = _generation;
    final key = sha256
        .convert(utf8.encode(jsonEncode([apiOrigin, scopeKey, url])))
        .toString();
    final flightKey = '$generation:$key';
    return _pending.putIfAbsent(flightKey, () {
      final result = _load(
        url,
        apiOrigin,
        key,
        generation,
        !scopeKey.startsWith('pending:'),
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

  bool _publicAddress(Uri uri, Uri origin) =>
      uri.scheme == 'https' &&
      uri.origin == origin.origin &&
      uri.userInfo.isEmpty &&
      !uri.hasQuery &&
      !uri.hasFragment;

  bool _restricted(Headers headers) {
    final control = (headers['cache-control']?.join(',') ?? '').toLowerCase();
    return control.split(',').any((part) {
          final directive = part.trim().split('=').first.trim();
          return const ['private', 'no-store', 'no-cache'].contains(directive);
        }) ||
        headers['set-cookie'] != null ||
        (headers['vary']?.join(',') ?? '').trim().isNotEmpty;
  }

  Future<Uint8List> _load(
    String url,
    String apiOrigin,
    String key,
    int generation,
    bool resolvedScope,
  ) async {
    _guard(generation);
    var uri = Uri.parse(url);
    final origin = Uri.parse(apiOrigin);
    if (!['https', 'http'].contains(uri.scheme)) {
      throw const FormatException('Unsupported image URL');
    }
    var publicAddress = resolvedScope && _publicAddress(uri, origin);
    // Userinfo must never become implicit HTTP Basic authentication.
    uri = uri.replace(userInfo: '');
    Map<String, dynamic>? previous;
    Uint8List? local;
    await _locked(() async {
      await _open();
      _guard(generation);
      await _sweep();
      previous = _entries[key];
      if (previous != null) {
        try {
          local = await _file(key).readAsBytes();
        } on FileSystemException {
          await _delete(key);
          await _saveIndex();
          previous = null;
          return;
        }
        if (publicAddress &&
            (previous!['freshUntil'] as int) > _now().millisecondsSinceEpoch) {
          previous!['accessed'] = _now().millisecondsSinceEpoch;
          await _saveIndex();
        } else if (!publicAddress) {
          await _delete(key);
          await _saveIndex();
          local = null;
        }
      }
    });
    _guard(generation);
    if (local != null &&
        (previous!['freshUntil'] as int) > _now().millisecondsSinceEpoch) {
      return local!;
    }
    if (_activeDownloads >= 3) {
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
      for (var redirect = 0; redirect <= 5; redirect++) {
        response = await _dio.getUri<ResponseBody>(
          uri,
          cancelToken: token,
          options: Options(
            responseType: ResponseType.stream,
            followRedirects: false,
            validateStatus: (_) => true,
            headers: {
              if (redirect == 0 &&
                  local != null &&
                  previous != null &&
                  previous!['etag'] is String)
                'If-None-Match': previous!['etag'],
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
          publicAddress = publicAddress && !_restricted(response.headers);
          if (!publicAddress) {
            await _locked(() async {
              _guard(generation);
              await _delete(key);
              await _saveIndex();
            });
          }
          uri = uri.resolve(location);
          publicAddress = publicAddress && _publicAddress(uri, origin);
          if (!['https', 'http'].contains(uri.scheme)) {
            throw const FormatException('Unsupported media redirect');
          }
          uri = uri.replace(userInfo: '');
          continue;
        }
        break;
      }
      final status = response!.statusCode;
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
      final freshUntil = _now().add(
        Duration(seconds: (maxAge - age).clamp(0, 365 * 24 * 60 * 60)),
      );
      final persist =
          publicAddress &&
          directives.contains('public') &&
          maxAge > age &&
          !directives.any(
            (d) =>
                d.startsWith('no-store') ||
                d.startsWith('private') ||
                d.startsWith('no-cache'),
          ) &&
          headers['set-cookie'] == null &&
          (headers['vary']?.join(',') ?? '').trim().isEmpty &&
          (status == 304 ||
              (headers.value('content-type') ?? '').toLowerCase().startsWith(
                'image/',
              ));
      // A changed restriction revokes the previous public entry immediately,
      // even if reading the replacement subsequently fails or exceeds limits.
      if (!persist) {
        await _locked(() async {
          _guard(generation);
          await _delete(key);
          await _saveIndex();
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
      await _locked(() async {
        _guard(generation);
        await _delete(key);
        await _saveIndex();
        final entry = {
          'bytes': result.length,
          'accessed': _now().millisecondsSinceEpoch,
          'freshUntil': freshUntil.millisecondsSinceEpoch,
          'cacheControl': control,
          'etag':
              headers.value('etag') ??
              (status == 304
                  ? (previous == null ? null : previous!['etag'])
                  : null),
        };
        int indexSize(Map<String, Map<String, dynamic>> entries) =>
            utf8.encode(jsonEncode({'version': 1, 'entries': entries})).length;
        if (persist && result.length + indexSize({key: entry}) <= budgetBytes) {
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
          await _saveIndex();
          if (await _diskBytes() > budgetBytes) {
            await _trim((budgetBytes * .8).floor());
          }
        }
      });
      _guard(generation);
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
    invalidate();
    _disposed = true;
    if (_ownsDio) _dio.close(force: true);
    super.dispose();
  }
}

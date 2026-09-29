import 'dart:io';
import 'dart:math';

import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:path_provider/path_provider.dart';
import 'package:sqlite3/sqlite3.dart';

const _keys = FlutterSecureStorage(
  iOptions: IOSOptions(
    accountName: 'yourtj_database_keys',
    accessibility: KeychainAccessibility.first_unlock_this_device,
    synchronizable: false,
  ),
);

/// Durable local files are separate from OS-reclaimable media. The native iOS
/// bridge explicitly excludes this directory (and future WAL files) from backup.
Future<Directory> privateStorageDirectory() async {
  final support = await getApplicationSupportDirectory();
  final directory = Directory('${support.path}/yourtj_private');
  await directory.create(recursive: true);
  if (Platform.isIOS) {
    await const MethodChannel(
      'yourtj/storage',
    ).invokeMethod<void>('excludeFromBackup', {'path': directory.path});
  }
  return directory;
}

Future<int> privateDatabaseBytes(String name) async {
  final directory = await privateStorageDirectory();
  var bytes = 0;
  for (final suffix in ['', '-wal', '-shm', '.migrating']) {
    final file = File('${directory.path}/$name.sqlite$suffix');
    if (await file.exists()) bytes += await file.length();
  }
  return bytes;
}

/// No plaintext fallback, including release builds. User work with a missing
/// key remains untouched; only explicitly disposable data can be rebuilt.
QueryExecutor openPrivateDatabase({
  required String name,
  bool disposable = false,
  String? legacyName,
}) => LazyDatabase(() async {
  final directory = await privateStorageDirectory();
  final file = File('${directory.path}/$name.sqlite');
  final keyName = 'yourtj.database.$name.key.v1';
  var key = await _keys.read(key: keyName);
  if (key == null) {
    if (await file.exists()) {
      if (!disposable) {
        throw StateError('Local work encryption key unavailable');
      }
      for (final suffix in ['', '-wal', '-shm', '.migrating']) {
        final stale = File('${file.path}$suffix');
        if (await stale.exists()) await stale.delete();
      }
    }
    final random = Random.secure();
    key = List.generate(
      32,
      (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0'),
    ).join();
    await _keys.write(key: keyName, value: key);
    if (await _keys.read(key: keyName) != key) {
      throw StateError('Local encryption key could not be saved');
    }
  }
  if (!RegExp(r'^[a-f0-9]{64}$').hasMatch(key)) {
    throw StateError('Invalid local encryption key');
  }
  final secret = key;
  final legacy = legacyName == null
      ? null
      : File(
          '${(await getApplicationDocumentsDirectory()).path}/$legacyName.sqlite',
        );
  final path = file.path;
  final legacyPath = legacy?.path;
  return NativeDatabase.createInBackground(
    file,
    isolateSetup: () async {
      if (legacyPath == null) return;
      final old = File(legacyPath);
      final target = File(path);
      if (!await old.exists()) return;
      if (await target.exists()) {
        // A crash after atomic rename may leave plaintext cleanup unfinished.
        final installed = sqlite3.open(path);
        try {
          requireCipher(installed);
          installed.execute("PRAGMA key = '$secret'");
          if (installed.select('PRAGMA integrity_check').single.values.single !=
              'ok') {
            throw StateError('Local cache migration verification failed');
          }
        } finally {
          installed.close();
        }
        for (final suffix in ['', '-wal', '-shm']) {
          final leftover = File('$legacyPath$suffix');
          if (await leftover.exists()) await leftover.delete();
        }
        return;
      }
      final temporary = File('$path.migrating');
      if (await temporary.exists()) await temporary.delete();
      final source = sqlite3.open(legacyPath);
      try {
        source.execute("VACUUM INTO '${temporary.path.replaceAll("'", "''")}'");
      } finally {
        source.close();
      }
      final copy = sqlite3.open(temporary.path);
      try {
        requireCipher(copy);
        copy.execute("PRAGMA rekey = '$secret'");
      } finally {
        copy.close();
      }
      final verify = sqlite3.open(temporary.path);
      try {
        requireCipher(verify);
        verify.execute("PRAGMA key = '$secret'");
        if (verify.select('PRAGMA integrity_check').single.values.single !=
            'ok') {
          throw StateError('Local cache migration verification failed');
        }
      } finally {
        verify.close();
      }
      await temporary.rename(path);
      // A verified encrypted copy is authoritative before removing plaintext.
      for (final suffix in ['', '-wal', '-shm']) {
        final oldFile = File('$legacyPath$suffix');
        if (await oldFile.exists()) await oldFile.delete();
      }
    },
    setup: (raw) {
      requireCipher(raw);
      raw.execute("PRAGMA key = '$secret'");
      final vacuumMode = raw.select('PRAGMA auto_vacuum').single.values.single;
      raw.execute('PRAGMA auto_vacuum = INCREMENTAL');
      // NONE -> INCREMENTAL needs a rewrite for existing databases. A pragma
      // alone leaves legacy files unable to return freed pages to the OS.
      if (vacuumMode == 0 &&
          raw.select('PRAGMA page_count').single.values.single as int > 0) {
        raw.execute('VACUUM');
      }
      raw.execute('PRAGMA journal_mode = WAL');
      raw.execute('PRAGMA wal_autocheckpoint = 256');
      raw.execute('PRAGMA journal_size_limit = 2097152');
      raw.execute('PRAGMA synchronous = FULL');
      raw.execute('PRAGMA secure_delete = ON');
    },
  );
});

void requireCipher(Database database) {
  if (database.select('PRAGMA cipher').isEmpty) {
    throw StateError(
      'Encrypted SQLite is unavailable; refusing plaintext storage',
    );
  }
}

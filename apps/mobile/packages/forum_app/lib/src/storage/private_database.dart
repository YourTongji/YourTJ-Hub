import 'dart:io';
import 'dart:isolate';
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
      await _deleteDatabaseFiles(file.path);
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
  // Drift's isolateSetup runs before its reply port is published. An exception
  // there can strand the opener. Await a separate worker so preflight/migration
  // failures reach StorageGate and every opened SQLite handle is already closed.
  await Isolate.run(() async {
    try {
      _verifyEncryptedDatabase(path, secret);
      await _migrateLegacyDatabase(path, legacyPath, secret);
    } on SqliteException catch (error) {
      // BUSY, IOERR, FULL, permissions and unavailable cipher support are not
      // proof of corruption. Never erase user work for any opener failure.
      if (!disposable ||
          (error.resultCode != SqlError.SQLITE_CORRUPT &&
              error.resultCode != SqlError.SQLITE_NOTADB)) {
        rethrow;
      }
      await _deleteDatabaseFiles(path);
      // A corrupt plaintext source must not trigger the same failed migration
      // forever or resurrect obsolete cache rows on the next launch.
      if (legacyPath != null) await _deleteDatabaseFiles(legacyPath);
    }
  });
  return NativeDatabase.createInBackground(
    file,
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

Future<void> _deleteDatabaseFiles(String path) async {
  // Remove the main file last: an interrupted cleanup must not leave orphan
  // WAL/journal pages next to a newly created database on the next launch.
  for (final suffix in [
    '-wal',
    '-shm',
    '-journal',
    '.migrating',
    '.migrating-wal',
    '.migrating-shm',
    '.migrating-journal',
    '',
  ]) {
    final file = File('$path$suffix');
    if (await file.exists()) await file.delete();
  }
}

void _verifyEncryptedDatabase(String path, String secret) {
  if (!File(path).existsSync()) return;
  final database = sqlite3.open(path);
  try {
    requireCipher(database);
    database.execute("PRAGMA key = '$secret'");
    final result = database.select('PRAGMA quick_check');
    if (result.length != 1 || result.single.values.single != 'ok') {
      throw SqliteException(
        extendedResultCode: SqlError.SQLITE_CORRUPT,
        message: 'Local database integrity verification failed',
      );
    }
  } finally {
    database.close();
  }
}

Future<void> _migrateLegacyDatabase(
  String path,
  String? legacyPath,
  String secret,
) async {
  if (legacyPath == null) return;
  final old = File(legacyPath);
  final target = File(path);
  if (!await old.exists()) return;
  if (await target.exists()) {
    // A verified installed encrypted copy wins after an interrupted rename.
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
    if (verify.select('PRAGMA integrity_check').single.values.single != 'ok') {
      throw SqliteException(
        extendedResultCode: SqlError.SQLITE_CORRUPT,
        message: 'Local cache migration verification failed',
      );
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
}

void requireCipher(Database database) {
  if (database.select('PRAGMA cipher').isEmpty) {
    throw StateError(
      'Encrypted SQLite is unavailable; refusing plaintext storage',
    );
  }
}

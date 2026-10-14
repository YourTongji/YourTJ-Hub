import 'dart:async';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

import 'temp_directory_helpers.dart';

/// Stands in for a directory another process still holds open. Windows reports
/// that as a sharing violation (errno 32) from the first [failures] deletions.
class _LockedDirectory implements Directory {
  _LockedDirectory(this.path, {required this.failures, this.errorCode = 32});

  int failures;
  final int errorCode;
  int attempts = 0;

  @override
  final String path;

  @override
  Uri get uri => Directory(path).uri;

  @override
  Future<Directory> delete({bool recursive = false}) async {
    attempts++;
    if (failures > 0) {
      failures--;
      throw PathAccessException(
        path,
        OSError('The process cannot access the file.', errorCode),
        'Deletion failed',
      );
    }
    return Directory(path);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Collects everything the cleanup prints while [body] runs, so a reported
/// failure can be told apart from a silent one.
Future<List<String>> _capturePrint(Future<void> Function() body) async {
  final lines = <String>[];
  await runZoned(
    body,
    zoneSpecification: ZoneSpecification(
      print: (_, _, _, line) => lines.add(line),
    ),
  );
  return lines;
}

void main() {
  const noDelay = [Duration.zero, Duration.zero, Duration.zero];

  test('a temporary directory is deleted with everything in it', () async {
    final directory = await Directory.systemTemp.createTemp('temp-dir-helper-');
    await File('${directory.path}/cache.sqlite').writeAsString('fixture');
    await deleteTempDirectory(directory);
    expect(await directory.exists(), isFalse);
  });

  test('a lock that clears shortly after the delete is retried', () async {
    final directory = _LockedDirectory('/locked', failures: 2);
    final printed = await _capturePrint(
      () => deleteTempDirectory(directory, backoff: noDelay),
    );
    expect(directory.attempts, 3);
    expect(printed, isEmpty);
  });

  test('a lock that outlives the budget is reported, not thrown', () async {
    final directory = _LockedDirectory('/locked', failures: 100);
    final printed = await _capturePrint(
      () => deleteTempDirectory(directory, backoff: noDelay),
    );
    expect(directory.attempts, noDelay.length + 1);
    expect(printed, hasLength(1));
    expect(printed.single, contains('leaving /locked'));
    expect(printed.single, contains('errno = 32'));
  });

  test('a failure other than a sharing violation is reported once', () async {
    final directory = _LockedDirectory('/denied', failures: 100, errorCode: 5);
    final printed = await _capturePrint(
      () => deleteTempDirectory(directory, backoff: noDelay),
    );
    expect(directory.attempts, 1);
    expect(printed, hasLength(1));
    expect(printed.single, contains('leaving /denied'));
    expect(printed.single, contains('errno = 5'));
  });

  test('a directory that is already gone is cleaned up silently', () async {
    final directory = await Directory.systemTemp.createTemp('temp-dir-gone-');
    await directory.delete();
    final printed = await _capturePrint(
      () => deleteTempDirectory(directory, backoff: noDelay),
    );
    expect(printed, isEmpty);
  });
}

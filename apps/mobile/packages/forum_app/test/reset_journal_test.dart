import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/storage/reset_journal.dart';

class _UnreadableDirectory implements Directory {
  _UnreadableDirectory(this.path);
  @override
  final String path;
  @override
  Uri get uri => Directory(path).uri;
  @override
  Stream<FileSystemEntity> list({
    bool recursive = false,
    bool followLinks = true,
  }) => Stream.error(
    FileSystemException('Fixture directory lookup failed', path),
  );
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void main() {
  late Directory directory;
  late ResetJournal journal;
  setUp(() async {
    directory = await Directory.systemTemp.createTemp('yourtj-reset-journal-');
    journal = ResetJournal(directory: () async => directory);
  });
  tearDown(() async => directory.delete(recursive: true));

  test('journal lookup errors cannot report a completed reset', () async {
    final journal = ResetJournal(
      directory: () async => _UnreadableDirectory('/unreadable-fixture'),
    );
    await expectLater(journal.isPending(), throwsA(isA<FileSystemException>()));
  });
  test(
    'pending marker survives a fresh journal until explicitly finished',
    () async {
      expect(await journal.isPending(), isFalse);
      await journal.begin();
      final reopened = ResetJournal(directory: () async => directory);
      expect(await reopened.isPending(), isTrue);
      await reopened.finish();
      expect(await journal.isPending(), isFalse);
    },
  );
  test(
    'an empty marker is pending, including with a trailing directory separator',
    () async {
      await File('${directory.path}/reset.intent').writeAsString('');
      final reopened = ResetJournal(
        directory: () async => Directory('${directory.path}/'),
      );
      expect(await reopened.isPending(), isTrue);
      await reopened.finish();
      expect(await reopened.isPending(), isFalse);
    },
  );
  test(
    'a malformed directory marker remains pending when removal fails',
    () async {
      final marker = Directory('${directory.path}/reset.intent');
      await marker.create();
      await File('${marker.path}/keep').writeAsString('blocked');
      expect(await journal.isPending(), isTrue);
      await expectLater(journal.finish(), throwsA(isA<FileSystemException>()));
      expect(await journal.isPending(), isTrue);
    },
  );
  test(
    'a dangling link marker is pending without following its target',
    () async {
      await Link(
        '${directory.path}/reset.intent',
      ).create('${directory.path}/missing');
      expect(await journal.isPending(), isTrue);
      await expectLater(journal.finish(), throwsA(isA<FileSystemException>()));
      expect(await journal.isPending(), isTrue);
    },
    skip: Platform.isWindows,
  );
}

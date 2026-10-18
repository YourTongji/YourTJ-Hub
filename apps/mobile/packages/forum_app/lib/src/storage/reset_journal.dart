import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'private_database.dart';

final resetJournalProvider = Provider<ResetJournal>((ref) => ResetJournal());

const _markerName = 'reset.intent';

/// Contains only a reset intent, never credentials or user content. It lives in
/// backup-excluded application support outside disposable databases and prefs.
/// Any existing marker is pending, including an interrupted or malformed write.
class ResetJournal {
  ResetJournal({Future<Directory> Function()? directory})
    : _directory = directory ?? privateStorageDirectory;

  final Future<Directory> Function() _directory;
  Future<File> _file() async =>
      File.fromUri((await _directory()).uri.resolve(_markerName));

  Future<bool> isPending() async {
    final directory = await _directory();
    // type/exists convert lookup failures into "not found". Listing preserves
    // those errors, so unreadable intent cannot reopen business routes.
    return directory.list(followLinks: false).any((entry) {
      // URI segments avoid comparing Windows paths with mixed separators.
      final name = entry.uri.pathSegments.lastWhere(
        (segment) => segment.isNotEmpty,
        orElse: () => '',
      );
      return name == _markerName;
    });
  }

  Future<void> begin() async {
    final file = await _file();
    await file.writeAsString('reset-v1', flush: true);
    if (await file.readAsString() != 'reset-v1') {
      throw StateError('Local reset intent could not be saved');
    }
  }

  Future<void> finish() async {
    final file = await _file();
    if (await isPending()) await file.delete();
    if (await isPending()) {
      throw StateError('Local reset intent could not be removed');
    }
  }
}

import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'private_database.dart';

final resetJournalProvider = Provider<ResetJournal>((ref) => ResetJournal());

/// Contains only a reset intent, never credentials or user content. It lives in
/// backup-excluded application support outside disposable databases and prefs.
/// Any existing marker is pending, including an interrupted or malformed write.
class ResetJournal {
  ResetJournal({Future<Directory> Function()? directory})
    : _directory = directory ?? privateStorageDirectory;

  final Future<Directory> Function() _directory;
  Future<File> _file() async =>
      File.fromUri((await _directory()).uri.resolve('reset.intent'));

  Future<bool> isPending() async {
    final directory = await _directory();
    final path = File.fromUri(directory.uri.resolve('reset.intent')).path;
    // type/exists convert lookup failures into "not found". Listing preserves
    // those errors, so unreadable intent cannot reopen business routes.
    return directory
        .list(followLinks: false)
        .any((entry) => entry.path == path);
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

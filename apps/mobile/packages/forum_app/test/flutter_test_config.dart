import 'dart:async';

import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/storage/user_work_database.dart';

Future<void> testExecutable(FutureOr<void> Function() testMain) async {
  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    final database = UserWorkDatabase(NativeDatabase.memory());
    // Open outside testWidgets' FakeAsync so drift's opener cannot wait on a
    // fake timer while the global teardown waits for an unfinished transaction.
    await database.customSelect('SELECT 1').get();
    UserWorkDatabase.setInstanceForTesting(database);
  });
  // Test teardowns run last-registered-first, so this close runs after every
  // file's teardown — including one that deletes a temporary directory. A file
  // that swaps in a file-backed instance should close it in its own teardown
  // before deleting that directory: Windows cannot delete an open file, so the
  // delete would only leave the directory behind as a printed report (#1138).
  tearDown(() async {
    await UserWorkDatabase.instance.close();
    UserWorkDatabase.setInstanceForTesting(null);
  });
  await testMain();
}

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
  tearDown(() async {
    await UserWorkDatabase.instance.close();
    UserWorkDatabase.setInstanceForTesting(null);
  });
  await testMain();
}

import 'dart:convert';
import 'dart:io';
import 'dart:isolate';
import 'dart:math';

import 'package:auth/auth.dart';
import 'package:gal/gal.dart';
import 'package:share_plus/share_plus.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:sqlite3/sqlite3.dart';

import 'package:forum_app/src/storage/private_database.dart';

void main() async {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  runApp(const MaterialApp(home: Scaffold(body: Text('YourTJ native tests'))));
  await binding.waitUntilFirstFrameRasterized;

  testWidgets('HarmonyOS native storage, encrypted SQLite and system bridges', (
    tester,
  ) async {
    expect(Platform.operatingSystem, 'ohos');
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: Text('YourTJ native bridge tests')),
      ),
    );
    const secure = NativeSecureStorage();
    final name = 'test.harmony.${DateTime.now().microsecondsSinceEpoch}';
    final directory = await Directory(
      '${(await getTemporaryDirectory()).path}/$name',
    ).create();
    final preferences = await SharedPreferences.getInstance();
    final key = List.generate(
      32,
      (_) => Random.secure().nextInt(256).toRadixString(16).padLeft(2, '0'),
    ).join();
    try {
      await secure.write(key: name, value: key);
      expect(await secure.read(key: name), key);
      await preferences.setString(name, 'native preference');
      await preferences.reload();
      expect(preferences.getString(name), 'native preference');
      final path = '${directory.path}/encrypted.sqlite';
      await Isolate.run(() {
        final database = sqlite3.open(path);
        try {
          requireCipher(database);
          database.execute("PRAGMA key = '$key'");
          database.execute('CREATE TABLE sample (value TEXT NOT NULL)');
          database.execute('INSERT INTO sample VALUES (?)', ['sample only']);
        } finally {
          database.close();
        }
        final reopened = sqlite3.open(path);
        try {
          reopened.execute("PRAGMA key = '$key'");
          if (reopened.select('SELECT value FROM sample').single['value'] !=
              'sample only') {
            throw StateError('Encrypted SQLite did not preserve the sample');
          }
        } finally {
          reopened.close();
        }
        final wrongKey = sqlite3.open(path);
        try {
          wrongKey.execute("PRAGMA key = '${'0' * 64}'");
          var rejected = false;
          try {
            wrongKey.select('SELECT value FROM sample');
          } on SqliteException {
            rejected = true;
          }
          if (!rejected) {
            throw StateError('Encrypted SQLite accepted an incorrect key');
          }
        } finally {
          wrongKey.close();
        }
      });
      expect(
        String.fromCharCodes((await File(path).readAsBytes()).take(16)),
        isNot('SQLite format 3\u0000'),
      );
      expect(await Connectivity().checkConnectivity(), isNotEmpty);
      final profile = await const MethodChannel(
        'yourtj/startup',
      ).invokeMapMethod<String, String>('getDeviceProfile');
      expect(profile?['device'], 'HarmonyOS');
      await const MethodChannel('yourtj/push').invokeMethod<void>('stop');
    } finally {
      await secure.delete(key: name);
      await preferences.remove(name);
      await directory.delete(recursive: true);
    }
    expect(await secure.containsKey(key: name), isFalse);
  });

  testWidgets(
    'HarmonyOS system dialogs report cancellation without leftover files',
    (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: Text('YourTJ system dialog tests')),
        ),
      );
      debugPrint('YOURTJ_TEST_SHARE_READY');
      final shared = await SharePlus.instance.share(
        ShareParams(
          text: 'YourTJ synthetic share test',
          title: 'YourTJ native test',
        ),
      );
      expect(shared.status, ShareResultStatus.dismissed);
      final cache = await getTemporaryDirectory();
      Future<Set<String>> temporarySaves() async => {
        await for (final file in cache.list())
          if (file.path.split('/').last.startsWith('yourtj-save-')) file.path,
      };
      final before = await temporarySaves();
      debugPrint('YOURTJ_TEST_GALLERY_READY');
      await expectLater(
        Gal.putImageBytes(
          base64Decode(
            'iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAEElEQVR4nGP8zwACTGCSAQANHQEDgslx/wAAAABJRU5ErkJggg==',
          ),
          name: 'YourTJ synthetic test',
        ).timeout(const Duration(seconds: 20)),
        throwsA(
          isA<GalException>().having(
            (error) => error.type,
            'type',
            GalExceptionType.accessDenied,
          ),
        ),
      );
      expect(await temporarySaves(), before);
    },
  );
}

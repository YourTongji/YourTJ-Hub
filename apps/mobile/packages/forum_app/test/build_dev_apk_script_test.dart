import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/app_config.dart';

void main() {
  test('dev APK script passes the dev preview AppConfig defines', () async {
    final forumAppDirectory = Directory.current;
    final script = File('../../scripts/build_dev_apk.sh').absolute;
    expect(script.existsSync(), isTrue);

    final temporaryDirectory = await Directory.systemTemp.createTemp(
      'build-dev-apk-test-',
    );
    addTearDown(() => temporaryDirectory.delete(recursive: true));

    final capturedArguments = File('${temporaryDirectory.path}/arguments');
    final fakeFlutter = File('${temporaryDirectory.path}/flutter');
    await fakeFlutter.writeAsString('''#!/usr/bin/env bash
printf '%s\\n' "\$@" > "\$YOURTJ_CAPTURED_FLUTTER_ARGUMENTS"
if [[ "\$YOURTJ_CREATE_TEST_APK" == true ]]; then
  mkdir -p build/app/outputs/flutter-apk
  : > build/app/outputs/flutter-apk/app-arm64-v8a-debug.apk
fi
''');
    final chmod = await Process.run('chmod', ['+x', fakeFlutter.path]);
    expect(chmod.exitCode, 0);

    final expectedApk = File(
      'build/app/outputs/flutter-apk/app-arm64-v8a-debug.apk',
    );
    final createTestApk = !expectedApk.existsSync();
    final buildDirectoryExisted = Directory('build').existsSync();
    if (createTestApk) {
      addTearDown(() async {
        if (expectedApk.existsSync()) await expectedApk.delete();
        if (!buildDirectoryExisted) {
          final generatedBuildDirectory = Directory('build');
          if (generatedBuildDirectory.existsSync()) {
            await generatedBuildDirectory.delete(recursive: true);
          }
        }
      });
    }

    final process = await Process.run(
      '/bin/bash',
      [script.path],
      environment: {
        ...Platform.environment,
        'PATH': '${temporaryDirectory.path}:${Platform.environment['PATH']}',
        'YOURTJ_CAPTURED_FLUTTER_ARGUMENTS': capturedArguments.path,
        'YOURTJ_CREATE_TEST_APK': createTestApk.toString(),
        'YOURTJ_DEV_CLIENT_ID': 'yourtj-mobile',
      },
      workingDirectory: forumAppDirectory.path,
    );
    expect(process.exitCode, 0, reason: '${process.stderr}');
    expect(process.stdout, contains('app-arm64-v8a-debug.apk'));
    expect(
      script.readAsStringSync(),
      contains('build/app/outputs/flutter-apk/app-*-debug.apk'),
    );

    final defines = <String, String>{};
    for (final argument in await capturedArguments.readAsLines()) {
      const prefix = '--dart-define=';
      if (!argument.startsWith(prefix)) continue;
      final entry = argument.substring(prefix.length);
      final separator = entry.indexOf('=');
      if (separator < 0) continue;
      defines[entry.substring(0, separator)] = entry.substring(separator + 1);
    }

    expect(defines['YOURTJ_ENVIRONMENT'], 'dev');
    expect(defines['YOURTJ_PREVIEW'], 'true');
    expect(defines['YOURTJ_API_BASE_URL'], 'https://dev.yourtj.de');
    expect(defines['YOURTJ_OIDC_ISSUER'], 'https://dev.yourtj.de/api/oauth');
    expect(defines['YOURTJ_OIDC_CLIENT_ID'], 'yourtj-mobile');

    AppConfig.validateValues(
      environment: defines['YOURTJ_ENVIRONMENT']!,
      preview: defines['YOURTJ_PREVIEW'] == 'true',
      apiBaseUrl: defines['YOURTJ_API_BASE_URL']!,
      oidcIssuer: defines['YOURTJ_OIDC_ISSUER']!,
    );

    final androidBuild = File(
      'android/app/build.gradle.kts',
    ).readAsStringSync();
    expect(
      androidBuild,
      contains('val devIdentity = dartDefines["YOURTJ_ENVIRONMENT"] == "dev"'),
    );
    expect(
      androidBuild,
      contains(
        'applicationId = if (devIdentity) "tj.yourtj.forum_app.dev" else "tj.yourtj.forum_app"',
      ),
    );
  });
}

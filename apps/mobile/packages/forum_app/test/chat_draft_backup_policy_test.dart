import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Android cloud backup and device transfer exclude private drafts', () {
    final manifest = File(
      'android/app/src/main/AndroidManifest.xml',
    ).readAsStringSync();
    for (final attribute in ['fullBackupContent', 'dataExtractionRules']) {
      final resource = RegExp(
        'android:$attribute="@xml/([^"]+)"',
      ).firstMatch(manifest)?.group(1);
      expect(resource, isNotNull, reason: '$attribute must define exclusions');
      final rules = File(
        'android/app/src/main/res/xml/$resource.xml',
      ).readAsStringSync();
      final sections = attribute == 'fullBackupContent'
          ? ['full-backup-content']
          : ['cloud-backup', 'device-transfer'];
      for (final section in sections) {
        final content = RegExp(
          '<$section(?:\\s[^>]*)?>([\\s\\S]*?)</$section>',
        ).firstMatch(rules)?.group(1);
        expect(content, isNotNull, reason: '$section rules must be explicit');
        for (final file in [
          'FlutterSharedPreferences.xml',
          'FlutterSecureStorage.xml',
          'FlutterSecureKeyStorage.xml',
        ]) {
          expect(
            content,
            contains('<exclude domain="sharedpref" path="$file"'),
            reason: '$section must exclude both migrated and legacy drafts',
          );
        }
      }
    }
  });
}

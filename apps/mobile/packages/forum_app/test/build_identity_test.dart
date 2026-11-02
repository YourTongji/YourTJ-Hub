import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/updates/update_host.dart';
import 'package:forum_app/src/updates/release_notes.dart';

void main() {
  test('iOS date number never becomes internal release history counter', () {
    expect(
      installedCatalogBuild({'buildNumber': 202611030001}, configuredBuild: 42),
      42,
    );
    expect(installedCatalogBuild({'buildNumber': 33}, configuredBuild: 0), 33);
    expect(installedCatalogBuild({}, configuredBuild: 0), isNull);
  });
  test('public display ID and native catalog ordering stay independent', () {
    final note = ReleaseNoteVersion.parse({
      'version': '1.0.18',
      'buildNumber': 42,
      'displayBuild': '202611030001',
      'channels': ['ios-app-store'],
      'highlights': [],
      'breaking': [],
      'requiredActions': [],
      'testflightNotes': [],
    });
    expect(note?.displayBuild, '202611030001');
    expect(note?.buildNumber, 42);
  });
}

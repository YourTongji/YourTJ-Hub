import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/admin/admin_navigation.dart';

void main() {
  final policy = AdminNavigation(Uri.parse('https://forum.example'));
  test('exports require exact first-party origin and numeric task path', () {
    expect(
      policy.isExport(
        Uri.parse('https://forum.example/api/admin/data/export/download/12'),
      ),
      isTrue,
    );
    for (final url in [
      'https://forum.example.evil/api/admin/data/export/download/12',
      'http://forum.example/api/admin/data/export/download/12',
      'https://forum.example:444/api/admin/data/export/download/12',
      'https://user@forum.example/api/admin/data/export/download/12',
      'https://forum.example/api/admin/data/export/download/../users',
      'https://forum.example/api/admin/data/export/download/0',
      'https://forum.example/api/admin/data/export/download/12?redirect=https://evil.test',
    ]) {
      expect(policy.isExport(Uri.parse(url)), isFalse, reason: url);
    }
  });
  test('cleartext handoff is restricted to local development hosts', () {
    expect(policy.isSecureOrigin, isTrue);
    expect(
      AdminNavigation(Uri.parse('http://10.0.2.2:5234')).isSecureOrigin,
      isTrue,
    );
    expect(
      AdminNavigation(Uri.parse('http://forum.example')).isSecureOrigin,
      isFalse,
    );
    expect(
      AdminNavigation(
        Uri.parse('https://user:password@forum.example'),
      ).isSecureOrigin,
      isFalse,
    );
  });
}

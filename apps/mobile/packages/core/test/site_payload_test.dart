import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final legacy in [false, true]) {
    test('site payload ignores server-only markup (legacy: $legacy)', () {
      final publicSite = <String, dynamic>{
        'name': 'YourTJ',
        'description': 'Forum',
        'logo': '/logo.png',
        'favicon': '/favicon.png',
        'brandType': 'text',
        'brandText': 'YourTJ',
        'brandImage': '',
      };
      final payload = SitePayload.fromJson({
        ...publicSite,
        if (legacy) 'externalLinks': '<meta name="legacy-head"/>',
      });

      expect(payload.toJson(), publicSite);
    });
  }
}

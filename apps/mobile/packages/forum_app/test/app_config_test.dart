import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/app_config.dart';

void main() {
  test('dev config requires the preview flag and official dev endpoints', () {
    expect(
      () => AppConfig.validateValues(
        environment: 'dev',
        preview: true,
        apiBaseUrl: 'https://dev.yourtj.de',
        oidcIssuer: 'https://dev.yourtj.de/api/oauth',
      ),
      returnsNormally,
    );
    expect(
      () => AppConfig.validateValues(
        environment: 'dev',
        preview: false,
        apiBaseUrl: 'https://dev.yourtj.de',
        oidcIssuer: 'https://dev.yourtj.de/api/oauth',
      ),
      throwsStateError,
    );
    expect(
      () => AppConfig.validateValues(
        environment: 'dev',
        preview: true,
        apiBaseUrl: 'https://example.com',
        oidcIssuer: 'https://example.com/api/oauth',
      ),
      throwsStateError,
    );
  });

  test('production config cannot connect to the dev API', () {
    expect(
      () => AppConfig.validateValues(
        environment: 'production',
        preview: false,
        apiBaseUrl: 'https://dev.yourtj.de',
        oidcIssuer: 'https://dev.yourtj.de/api/oauth',
      ),
      throwsStateError,
    );
  });
}

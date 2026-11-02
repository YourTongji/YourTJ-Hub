import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Map<String, dynamic> payload() => {
    'initialMode': 'login',
    'redirectUrl': '/',
    'githubUrl': '/api/auth/github',
    'googleReady': false,
  };

  test('GitHub readiness defaults to false for older page payloads', () {
    final options = LoginPageProps.fromJson(payload());
    expect(options.githubReady, isFalse);
    expect(options.toJson()['githubReady'], isFalse);
  });

  for (final ready in [false, true]) {
    test('GitHub readiness $ready survives JSON and copyWith', () {
      final options = LoginPageProps.fromJson({
        ...payload(),
        'githubReady': ready,
      });
      expect(options.githubReady, ready);
      expect(options.toJson()['githubReady'], ready);
      expect(LoginPageProps.fromJson(options.toJson()), options);
      expect(options.copyWith().githubReady, ready);
      final changed = options.copyWith(githubReady: !ready);
      expect(changed.githubReady, !ready);
      expect(changed, isNot(options));
    });
  }
}

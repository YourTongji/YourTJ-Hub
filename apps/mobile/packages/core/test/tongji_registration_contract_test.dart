import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('school registration mirrors shared response fixtures', () {
    Map<String, dynamic> fixture(String file) =>
        (jsonDecode(
                  File(
                    '../../../../packages/api-contract/fixtures/$file',
                  ).readAsStringSync(),
                )
                as Map<String, dynamic>)['result']
            as Map<String, dynamic>;
    final status = TongjiRegistrationStatusPayload.fromJson(
      fixture('tongji-registration-status-success.json'),
    );
    expect(status.csrfToken, 'opaque-csrf-proof');
    expect(status.email, '2356789@tongji.edu.cn');
    final result = TongjiRegistrationResult.fromJson(
      fixture('tongji-registration-success.json'),
    );
    expect(result.redirect, '/campus');
    expect(
      const TongjiRegistrationRequest(
        username: 'chosen_user',
        password: 'Password123',
        csrfToken: 'proof',
      ).toJson(),
      {
        'username': 'chosen_user',
        'password': 'Password123',
        'csrfToken': 'proof',
      },
    );
  });
}

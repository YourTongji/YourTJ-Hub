import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('connection mirror preserves viewer relationship fields', () {
    final user = UserConnectionPayload.fromJson({
      'id': 42,
      'username': 'alice',
      'nickname': 'Alice',
      'avatarUrl': '',
      'bio': 'Hello',
      'url': '/u/42',
      'isFollowing': true,
      'isSelf': false,
    });
    expect(user.toJson()['isFollowing'], true);
    expect(user.toJson()['isSelf'], false);
  });
  test('legacy relationship state remains unknown', () {
    final user = UserConnectionPayload.fromJson({
      'id': 42,
      'username': 'alice',
      'nickname': 'Alice',
      'avatarUrl': '',
      'bio': '',
      'url': '/u/42',
    });
    expect(user.isFollowing, isNull);
    expect(user.isSelf, false);
  });
}

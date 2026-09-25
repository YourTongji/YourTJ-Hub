import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final legacy = <String, dynamic>{
    'id': 1,
    'action': 5,
    'subjectType': 'Post',
    'subjectId': 42,
    'contentPreview': 'reply',
    'url': '/p/post/10/2',
    'label': 'comment',
    'createdAt': '2026-01-01T00:00:00Z',
  };
  test('activity viewer state distinguishes absent from explicit false', () {
    final old = UserActivityPayload.fromJson(legacy);
    expect(old.liked, isNull);
    expect(old.bookmarked, isNull);
    final current = UserActivityPayload.fromJson({
      ...legacy,
      'liked': true,
      'bookmarked': false,
      'likeCount': 7,
    });
    expect(current.liked, isTrue);
    expect(current.bookmarked, isFalse);
    expect(current.likeCount, 7);
    expect(UserActivityPayload.fromJson(current.toJson()), current);
  });
}

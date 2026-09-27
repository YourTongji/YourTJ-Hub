import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'notification shared fixture carries current avatar and plain reply preview',
    () {
      final json =
          jsonDecode(
                File(
                  '../../../../packages/api-contract/fixtures/notifications-like-preview-success.json',
                ).readAsStringSync(),
              )
              as Map<String, dynamic>;
      final item = NotificationListResponse.fromJson(
        json['result'] as Map<String, dynamic>,
      ).items.single;
      expect(item.actor.avatarUrl, '/static/pic/3.webp');
      expect(item.content, 'Readable reply');
      expect(item.payload.postNo, 8);
    },
  );

  final legacy = {
    'id': 1,
    'topicId': 9,
    'title': 'Original topic',
    'url': '/p/post/9',
    'likedAt': '2026-09-25',
  };
  final author = {
    'id': 77,
    'username': 'author',
    'nickname': 'Author',
    'avatarUrl': '/static/pic/1.webp',
  };
  test(
    'like previews deserialize author and preserve older-server fallback',
    () {
      final old = UserLikePayload.fromJson(legacy);
      expect(old.author, isNull);
      expect(old.excerpt, isNull);
      expect(old.thumbnailUrl, isNull);
      final row = UserLikePayload.fromJson({
        ...legacy,
        'author': author,
        'excerpt': 'Original body',
        'thumbnailUrl': '/file/img/1.png',
      });
      expect(row.author!.id, 77);
      expect(row.excerpt, 'Original body');
      expect(row.thumbnailUrl, '/file/img/1.png');
    },
  );
  test(
    'bookmark preview keeps anonymous authors absent and reply target intact',
    () {
      final raw = {
        ...legacy,
        'type': 'post',
        'postId': 42,
        'postNo': 7,
        'bookmarkedAt': '2026-09-25',
        'excerpt': 'Reply body',
      };
      final row = UserBookmarkPayload.fromJson(raw);
      expect(row.author, isNull);
      expect(row.postNo, 7);
      expect(row.excerpt, 'Reply body');
      expect(
        UserBookmarkPayload.fromJson({...raw, 'author': author}).author!.id,
        77,
      );
    },
  );
}

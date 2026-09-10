// 用户主页 payload 解析与生产实捕形态的一致性测试。
//
// 回归背景（2026-09-06 生产实捕 f.yourtj.de/u/1、/u/2）：后端 nil 切片把
// activityTabs / badges / user.badges 序列化为 JSON null（违反 payload.ts
// 非空数组契约），旧镜像的非空 required List 解析抛 TypeError，被
// parsePageProps 的 catch(_) 吞掉 → Profile 页显示「页面数据解析失败」。
// 本测试锁定：镜像对已部署后端的 null 数组必须容错为空列表。
library;

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

/// 与生产实捕同形的最小 payload：三处数组显式为 null（旧后端形态）。
Map<String, dynamic> prodShapeUserProfileJson() => <String, dynamic>{
  'user': <String, dynamic>{
    'userId': 2,
    'username': 'yzxoimoe',
    'nickname': 'yzxoi',
    'avatarUrl': '',
    'profileCoverUrl': '',
    'bio': '',
    'signature': '',
    'websiteName': '',
    'website': '',
    'prestige': 10,
    'externalInformation': <String, dynamic>{},
    'isAdmin': true,
    'topicCount': 1,
    'replyCount': 2,
    'likeReceivedCount': 3,
    'likeGivenCount': 0,
    'followerCount': 0,
    'followingCount': 0,
    'collectionCount': 0,
    'isOnline': false,
    'isFollowing': false,
    'isSelf': true,
    'badges': null,
    'lastActiveTime': '2026-09-06T07:17:12Z',
    'createdAt': '2026-08-06T07:35:59Z',
  },
  'section': 'summary',
  'activityTab': 'timeline',
  'tabs': <Map<String, dynamic>>[
    <String, dynamic>{'key': 'summary', 'url': '/u/2', 'active': true},
  ],
  'activityTabs': null,
  'pagination': <String, dynamic>{
    'page': 1,
    'nextPage': 0,
    'hasNext': false,
    'nextUrl': '',
  },
  'badges': null,
  'topics': <Object>[],
  'activities': <Object>[],
  'likes': <Object>[],
  'bookmarks': <Object>[],
  'following': <Object>[],
  'followers': <Object>[],
  'isOwnProfile': true,
  'canMessage': false,
  'canFollow': false,
  'messageUrl': '',
  'settingsUrl': '/settings',
};

void main() {
  group('UserProfileProps · 生产 null 数组容错', () {
    test('activityTabs/badges/user.badges 为 null 时解析为空列表而非抛错', () {
      final UserProfileProps props = UserProfileProps.fromJson(
        prodShapeUserProfileJson(),
      );
      expect(props.activityTabs, isEmpty);
      expect(props.badges, isEmpty);
      expect(props.user.badges, isEmpty);
      expect(props.user.username, 'yzxoimoe');
      expect(props.tabs, hasLength(1));
    });

    test('正常数组形态不受容错影响', () {
      final json = prodShapeUserProfileJson()
        ..['activityTabs'] = <Map<String, dynamic>>[
          <String, dynamic>{
            'key': 'timeline',
            'url': '/u/2/activity',
            'active': true,
          },
        ]
        ..['badges'] = <Map<String, dynamic>>[
          <String, dynamic>{
            'code': 'first_post',
            'type': 'system',
            'grantMode': 'auto',
            'name': '初次发帖',
            'description': '发布了第一篇主题',
            'iconType': 'asset',
            'iconKey': '',
            'iconUrl': '/static/badges/first-post.svg',
            'color': 'blue',
            'level': 'common',
            'isEnabled': true,
            'isWearable': true,
            'sortOrder': 10,
            'source': 'auto',
            'reason': '',
            'grantedAt': '2026-08-06T07:35:59Z',
          },
        ];
      final UserProfileProps props = UserProfileProps.fromJson(json);
      expect(props.activityTabs.single.key, 'timeline');
      expect(props.badges.single.code, 'first_post');
    });
  });
}

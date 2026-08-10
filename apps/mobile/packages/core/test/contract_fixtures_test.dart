import 'dart:convert';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('首页 PagePayload (home.index)', () {
    final page = PagePayload.fromJson({
      'component': 'home.index',
      'props': {
        'sort': 'hot',
        'tabs': [
          {'key': 'hot', 'label': '热门', 'url': '/?sort=hot', 'active': true},
          {'key': 'latest', 'label': '最新', 'url': '/?sort=latest', 'active': false},
        ],
        'topics': [
          {
            'id': 1,
            'title': '测试话题',
            'description': '描述',
            'firstImageUrl': '/uploads/a.webp',
            'url': '/p/post/1',
            'author': {
              'id': 10,
              'username': 'alice',
              'nickname': '爱丽丝',
              'avatarUrl': '/static/1.webp',
            },
            'participants': [
              {'id': 11, 'username': 'bob', 'avatarUrl': '/static/2.webp'},
            ],
            'categories': [
              {'id': 3, 'name': '闲聊', 'url': '/c/chat/3', 'color': '#f00'},
            ],
            'replyCount': 5,
            'viewCount': 100,
            'pinWeight': 0,
            'processStatus': 0,
            'activityText': '5 分钟前',
            'lastUpdateTime': '2026-08-07T10:00:00+08:00',
          },
        ],
        'pagination': {
          'page': 1,
          'nextPage': 2,
          'hasNext': true,
          'nextUrl': '/?sort=hot&page=2',
        },
        'announcement': {
          'enabled': true,
          'html': '<p>公告</p>',
          'publishedAt': '2026-08-01T00:00:00+08:00',
        },
      },
      'meta': {
        'title': 'yourtj',
        'description': '同济大学校园论坛',
        'canonical': 'https://yourtj.example/',
      },
      'layout': _layout(),
      'url': '/',
      'version': '1.0',
    });

    test('解析组件与 meta', () {
      expect(page.component, 'home.index');
      expect(page.meta.title, 'yourtj');
      expect(page.meta.description, '同济大学校园论坛');
      expect(page.url, '/');
      expect(page.version, '1.0');
    });

    test('parsePageProps 解析 HomeProps', () {
      final props = parsePageProps<HomeProps>(page);
      expect(props, isNotNull);
      expect(props!.sort, 'hot');
      expect(props.tabs, hasLength(2));
      expect(props.tabs.first.active, isTrue);
      expect(props.topics, hasLength(1));
      expect(props.topics.first.author.username, 'alice');
      expect(props.topics.first.author.nickname, '爱丽丝');
      expect(props.topics.first.categories.first.color, '#f00');
      expect(props.pagination.hasNext, isTrue);
      expect(props.announcement.enabled, isTrue);
      expect(props.announcement.html, '<p>公告</p>');
    });
  });

  group('话题详情 PagePayload (topic.detail)', () {
    final page = PagePayload.fromJson({
      'component': 'topic.detail',
      'props': {
        'topic': {
          'id': 1,
          'title': '详情标题',
          'description': '详情描述',
          'url': '/p/post/1',
          'topicStatus': 0,
          'processStatus': 0,
          'author': {'id': 10, 'username': 'alice', 'avatarUrl': '/static/1.webp'},
          'participants': [],
          'categories': [
            {'id': 3, 'name': '闲聊', 'url': '/c/chat/3', 'color': '#f00'},
          ],
          'replyCount': 2,
          'maxPostNo': 3,
          'viewCount': 42,
          'likeCount': 7,
          'isLiked': false,
          'isBookmarked': false,
          'isWatched': false,
          'createdAt': '2026-08-01T00:00:00+08:00',
          'updatedAt': '2026-08-07T10:00:00+08:00',
        },
        'postStream': {
          'posts': [
            {
              'id': 100,
              'topicId': 1,
              'postNo': 1,
              'content': '# 正文',
              'renderedContent': '<h1>正文</h1>',
              'processStatus': 0,
              'isHidden': false,
              'canModerate': false,
              'author': {'id': 10, 'username': 'alice', 'avatarUrl': '/static/1.webp'},
              'createdAt': '2026-08-01T00:00:00+08:00',
              'isOwnPost': true,
              'likeCount': 3,
              'isLiked': true,
              'isBookmarked': false,
            },
          ],
          'replyTargets': [],
          'hasBefore': false,
          'hasAfter': true,
          'total': 3,
          'maxPostNo': 3,
        },
        'hotTopics': [],
        'permissions': {
          'isOwnTopic': true,
          'canPost': true,
          'canModerateTopic': false,
        },
      },
      'meta': {'title': '详情标题 - yourtj'},
      'layout': _layout(),
      'url': '/p/post/1',
      'version': '1.0',
    });

    test('parsePageProps 解析 TopicDetailProps', () {
      final props = parsePageProps<TopicDetailProps>(page);
      expect(props, isNotNull);
      expect(props!.topic.title, '详情标题');
      expect(props.topic.likeCount, 7);
      expect(props.postStream.posts, hasLength(1));
      expect(props.postStream.posts.first.content, '# 正文');
      expect(props.postStream.posts.first.isOwnPost, isTrue);
      expect(props.postStream.posts.first.isLiked, isTrue);
      expect(props.postStream.afterPostNo, isNull);
      expect(props.postStream.hasAfter, isTrue);
      expect(props.permissions.canPost, isTrue);
      expect(props.permissions.isOwnTopic, isTrue);
    });
  });

  group('用户卡片 UserCardPayload', () {
    final card = UserCardPayload.fromJson({
      'userId': 10,
      'username': 'alice',
      'nickname': '爱丽丝',
      'avatarUrl': '/static/1.webp',
      'profileCoverUrl': '/uploads/cover.webp',
      'bio': '你好',
      'signature': '签名',
      'websiteName': '',
      'website': '',
      'prestige': 3,
      'externalInformation': {'github': {'link': 'https://github.com/alice'}},
      'isAdmin': false,
      'topicCount': 5,
      'replyCount': 20,
      'likeReceivedCount': 10,
      'likeGivenCount': 8,
      'followerCount': 2,
      'followingCount': 1,
      'collectionCount': 0,
      'isOnline': true,
      'isFollowing': false,
      'isSelf': true,
      'badges': [
        {
          'code': 'early',
          'type': 'system',
          'grantMode': 'auto',
          'name': '早期用户',
          'description': '描述',
          'iconType': 'emoji',
          'iconKey': 'star',
          'iconUrl': '',
          'color': '#ffd700',
          'level': 'gold',
          'isEnabled': true,
          'isWearable': true,
          'sortOrder': 1,
          'source': 'auto',
          'reason': '注册',
          'grantedAt': '2026-01-01T00:00:00+08:00',
        },
      ],
      'wornBadge': null,
      'lastActiveTime': '2026-08-07T09:00:00+08:00',
      'createdAt': '2026-01-01T00:00:00+08:00',
    });

    test('字段逐一对齐', () {
      expect(card.userId, 10);
      expect(card.username, 'alice');
      expect(card.nickname, '爱丽丝');
      expect(card.prestige, 3);
      expect(card.externalInformation['github']?.link, 'https://github.com/alice');
      expect(card.isSelf, isTrue);
      expect(card.badges, hasLength(1));
      expect(card.badges.first.code, 'early');
      expect(card.badges.first.level, 'gold');
      expect(card.wornBadge, isNull);
      expect(card.lastActiveTime, '2026-08-07T09:00:00+08:00');
    });
  });

  group('通知 NotificationListResponse', () {
    final response = NotificationListResponse.fromJson({
      'items': [
        {
          'id': 900,
          'eventType': 'comment',
          'isRead': false,
          'createdAt': '2026-08-07T10:00:00+08:00',
          'title': '有人回复了你',
          'content': '回复内容预览',
          'actor': {'id': 11, 'username': 'bob', 'avatarUrl': '/static/2.webp'},
          'topic': {'id': 1, 'title': '测试话题', 'url': '/p/post/1'},
          'payload': {
            'templateKey': 'notifications.templates.comment',
            'templateParams': {'preview': '回复内容预览'},
            'actorId': 11,
            'actorName': 'bob',
            'topicId': 1,
            'postId': 100,
            'topicTitle': '测试话题',
          },
        },
      ],
      'nextCursor': 20,
      'hasNext': true,
      'unreadCount': 3,
    });

    test('解析通知列表与模板参数', () {
      expect(response.items, hasLength(1));
      final item = response.items.first;
      expect(item.id, 900);
      expect(item.eventType, 'comment');
      expect(item.isRead, isFalse);
      expect(item.actor.username, 'bob');
      expect(item.topic?.url, '/p/post/1');
      expect(item.payload.actorId, 11);
      expect(item.payload.templateKey, 'notifications.templates.comment');
      expect(item.payload.templateParams?.preview, '回复内容预览');
      expect(response.nextCursor, 20);
      expect(response.hasNext, isTrue);
      expect(response.unreadCount, 3);
    });
  });

  group('私信 ChatMessagesResponse', () {
    final response = ChatMessagesResponse.fromJson({
      'list': [
        {
          'id': 500,
          'senderId': 11,
          'content': '在吗',
          'msgType': 1,
          'isRead': 1,
          'createdAt': '2026-08-07T10:00:00+08:00',
          'isSelf': false,
        },
      ],
      'hasMoreBefore': false,
      'hasMoreAfter': true,
      'nextBeforeId': 499,
      'latestId': 500,
    });

    test('解析消息列表与游标', () {
      expect(response.list, hasLength(1));
      final message = response.list.first;
      expect(message.senderId, 11);
      expect(message.content, '在吗');
      expect(message.msgType, 1);
      expect(message.isRead, 1);
      expect(message.isSelf, isFalse);
      expect(response.hasMoreAfter, isTrue);
      expect(response.nextBeforeId, 499);
      expect(response.latestId, 500);
    });
  });

  group('搜索 SearchPageProps', () {
    final props = SearchPageProps.fromJson({
      'query': 'flutter',
      'scope': 'all',
      'topics': [
        {
          'id': 2,
          'title': 'Flutter 讨论',
          'description': '',
          'url': '/p/post/2',
          'author': {'id': 12, 'username': 'carol', 'avatarUrl': '/static/3.webp'},
          'participants': [],
          'categories': [],
          'replyCount': 0,
          'viewCount': 1,
          'pinWeight': 0,
          'processStatus': 0,
          'activityText': '刚刚',
          'lastUpdateTime': '2026-08-07T11:00:00+08:00',
        },
      ],
      'users': [
        {
          'id': 13,
          'username': 'dart',
          'nickname': '小狐狸',
          'avatarUrl': '/static/4.webp',
          'bio': '喜欢 Flutter',
        },
      ],
      'categories': [
        {'id': 5, 'name': '技术', 'slug': 'tech', 'icon': 'code', 'color': '#00f', 'desc': '技术讨论'},
      ],
      'total': 1,
      'usersTotal': 1,
      'categoriesTotal': 1,
      'totalPages': 1,
      'pagination': {'page': 1, 'nextPage': 1, 'hasNext': false, 'nextUrl': ''},
    });

    test('解析搜索三类结果', () {
      expect(props.query, 'flutter');
      expect(props.topics, hasLength(1));
      expect(props.topics.first.title, 'Flutter 讨论');
      expect(props.users, hasLength(1));
      expect(props.users.first.nickname, '小狐狸');
      expect(props.categories, hasLength(1));
      expect(props.categories.first.slug, 'tech');
      expect(props.total, 1);
      expect(props.totalPages, 1);
      expect(props.pagination.hasNext, isFalse);
    });
  });

  group('会话 UserSessionPayload', () {
    // 与 packages/api-contract/fixtures/sessions-list-success.json 对齐。
    final sessions = (jsonDecode('''
{
  "result": [
    {
      "id": 42,
      "ipMasked": "127.0.0.*",
      "userAgent": "contract-test",
      "createdAt": 1754496000000,
      "expiresAt": 1757088000000,
      "isCurrent": true
    }
  ],
  "code": 0
}
''') as Map<String, dynamic>)['result'] as List<dynamic>;

    test('解析会话列表条目', () {
      final session =
          UserSessionPayload.fromJson(sessions.single as Map<String, dynamic>);
      expect(session.id, 42);
      expect(session.ipMasked, '127.0.0.*');
      expect(session.userAgent, 'contract-test');
      expect(session.createdAt, 1754496000000);
      expect(session.expiresAt, 1757088000000);
      expect(session.isCurrent, isTrue);
    });
  });

  group('GfResponse 响应包装', () {
    test('code == 0 成功', () {
      final response = GfResponse<int>.fromJson(
        {'code': 0, 'result': 42},
        (json) => json as int,
      );
      expect(response.isSuccess, isTrue);
      expect(response.result, 42);
    });

    test('code != 0 业务失败保留 messageCode 与 params', () {
      final response = GfResponse<int>.fromJson(
        {
          'code': 1,
          'messageCode': 'auth.login.invalidCredentials',
          'params': {'retryAfterSeconds': 60},
        },
        (json) => json as int,
      );
      expect(response.isSuccess, isFalse);
      expect(response.messageCode, 'auth.login.invalidCredentials');
      expect(response.params?['retryAfterSeconds'], 60);
      expect(response.result, isNull);
    });
  });
}

Map<String, dynamic> _layout() {
  return {
    'site': {
      'name': 'yourtj',
      'description': '同济大学校园论坛',
      'logo': '/static/logo.webp',
      'favicon': '/static/favicon.ico',
      'brandType': 'text',
      'brandText': 'yourtj',
      'brandImage': '',
    },
    'viewer': {
      'id': 10,
      'username': 'alice',
      'email': 'alice@example.com',
      'avatarUrl': '/static/1.webp',
      'isAuthenticated': true,
      'canAccessAdmin': false,
      'isModerator': false,
      'requiresEmailVerification': false,
      'adminPermissions': [],
    },
    'sidebar': {
      'categories': [
        {'id': 3, 'label': '闲聊', 'url': '/c/chat/3', 'color': '#f00'},
      ],
      'activeKey': 'home',
    },
    'footer': {
      'links': [
        {'name': '关于', 'url': '/links'},
      ],
      'primary': ['yourtj'],
    },
    'unread': {
      'notifications': true,
      'messages': false,
    },
    'theme': {
      'enabled': true,
      'current': 'gf-light',
      'themeColor': '#4f46e5',
    },
  };
}

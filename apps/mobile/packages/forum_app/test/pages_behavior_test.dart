import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/drafts/drafts_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/pages/search/search_page.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'package:forum_app/src/widgets/skeletons.dart';

import 'fixtures/page_fixtures.dart';

/// 内存 TokenStorage(与 pages_smoke_test 同款)。
class MemTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

/// no-op 离线缓存。
class NoopCache implements OfflineTopicCache, OfflineChatCache {
  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {}

  @override
  Future<PagePayload?> get(int topicId) async => null;

  @override
  Future<void> putConversations(List<ChatItemPayload> conversations) async {}

  @override
  Future<List<ChatItemPayload>> getConversations() async => const [];

  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {}

  @override
  Future<List<ChatMessagePayload>> getMessages(int convId) async => const [];

  @override
  Future<void> clear() async {}

  @override
  Future<void> close() async {}
}

class RecordingChatCache extends NoopCache {
  int putMessageCalls = 0;
  final List<List<int>> storedMessageIds = <List<int>>[];

  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {
    putMessageCalls++;
    storedMessageIds.add(
      messages.map((ChatMessagePayload message) => message.id).toList(),
    );
  }
}

/// 记录 clear 调用次数的离线缓存(登出/登录清缓存断言)。
class RecordingCache implements OfflineTopicCache, OfflineChatCache {
  int clears = 0;

  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {}

  @override
  Future<PagePayload?> get(int topicId) async => null;

  @override
  Future<void> putConversation(ChatItemPayload conv) async {}

  @override
  Future<List<ChatItemPayload>> getConversations() async => const [];

  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {}

  @override
  Future<List<ChatMessagePayload>> getMessages(int convId) async => const [];

  @override
  Future<void> clear() async {
    clears++;
  }

  @override
  Future<void> close() async {}
}

class RecordingChatRepository extends ChatRepository {
  RecordingChatRepository(super.client);

  final List<(int, String)> sent = <(int, String)>[];

  @override
  Future<int> sendMessage({
    required int peerId,
    required String content,
    int msgType = 1,
  }) async {
    sent.add((peerId, content));
    return 9;
  }

  @override
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int limit = 30,
  }) async {
    return const ChatMessagesResponse(
      list: <ChatMessagePayload>[],
      hasMoreBefore: false,
      hasMoreAfter: false,
      nextBeforeId: 0,
      latestId: 0,
    );
  }

  @override
  Future<bool> markRead({required int convId}) async => true;
}

ChatMessagePayload makeChatMessage(int id) {
  return ChatMessagePayload(
    id: id,
    senderId: 2,
    content: '消息 $id',
    msgType: 1,
    isRead: 0,
    createdAt: '2026-08-07T10:${(id % 60).toString().padLeft(2, '0')}:00+08:00',
    isSelf: false,
  );
}

class PollingChatRepository extends ChatRepository {
  PollingChatRepository(
    super.client, {
    required this.messages,
    this.hasMoreBefore = false,
    this.failOlder = false,
  });

  final List<ChatMessagePayload> messages;
  final bool hasMoreBefore;
  final bool failOlder;
  int afterCalls = 0;
  int beforeCalls = 0;
  int markReadCalls = 0;

  @override
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int limit = 30,
  }) async {
    if (beforeId > 0) {
      beforeCalls++;
      if (failOlder) throw StateError('older messages unavailable');
      return const ChatMessagesResponse(
        list: <ChatMessagePayload>[],
        hasMoreBefore: false,
        hasMoreAfter: false,
        nextBeforeId: 0,
        latestId: 0,
      );
    }
    afterCalls++;
    return ChatMessagesResponse(
      list: messages,
      hasMoreBefore: hasMoreBefore,
      hasMoreAfter: false,
      nextBeforeId: hasMoreBefore ? messages.first.id : 0,
      latestId: messages.isEmpty ? 0 : messages.last.id,
    );
  }

  @override
  Future<bool> markRead({required int convId}) async {
    markReadCalls++;
    return true;
  }
}

class IncomingChatRepository extends RecordingChatRepository {
  IncomingChatRepository(super.client);

  final List<int> fetchedConvIds = <int>[];

  @override
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int limit = 30,
  }) async {
    fetchedConvIds.add(convId);
    return const ChatMessagesResponse(
      list: <ChatMessagePayload>[
        ChatMessagePayload(
          id: 41,
          senderId: 2,
          content: '刚刚发来的消息',
          msgType: 1,
          isRead: 0,
          createdAt: '2026-08-10T12:00:00+08:00',
          isSelf: false,
        ),
      ],
      hasMoreBefore: false,
      hasMoreAfter: false,
      nextBeforeId: 0,
      latestId: 41,
    );
  }
}

/// 记录 fetch 调用次数的 PageRepository。
class CountingPageRepository extends PageRepository {
  CountingPageRepository(super.client);

  int fetchCalls = 0;

  @override
  Future<PagePayload> fetch(String path) async {
    fetchCalls++;
    if (path == '/' || path.startsWith('/?sort=')) {
      return parsePayload(homePayloadJson());
    }
    if (path == '/settings') {
      return parsePayload(settingsPayloadJson());
    }
    if (path == '/messages') {
      return parsePayload(messagesPayloadJson());
    }
    if (path == '/u/1') {
      return parsePayload(userProfilePayloadJson());
    }
    if (path == '/u/2') {
      return parsePayload(peerProfilePayloadJson());
    }
    if (path.startsWith('/p/post/')) {
      return parsePayload(topicDetailPayloadJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

class CountingMessagesPageRepository extends PageRepository {
  CountingMessagesPageRepository(super.client);

  int fetchCalls = 0;

  @override
  Future<PagePayload> fetch(String path) async {
    if (path != '/messages') {
      throw UnimplementedError('unexpected page path: $path');
    }
    fetchCalls++;
    return parsePayload(messagesPayloadJson());
  }
}

/// Returns richer, long-form fixtures for the redesigned content pages.
class RedesignPageRepository extends PageRepository {
  RedesignPageRepository(super.client, {this.profilePayload});

  final Map<String, dynamic>? profilePayload;

  @override
  Future<PagePayload> fetch(String path) async {
    if (path.startsWith('/p/post/')) {
      return parsePayload(redesignedTopicPayloadJson());
    }
    if (path.startsWith('/u/')) {
      return parsePayload(profilePayload ?? redesignedProfilePayloadJson());
    }
    if (path == '/' || path.startsWith('/?sort=')) {
      return parsePayload(homePayloadJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

class RecordingFollowTopicRepository extends TopicRepository {
  RecordingFollowTopicRepository(super.client);

  final List<int> userIds = <int>[];
  final List<bool> currentStates = <bool>[];

  @override
  Future<bool> followUser({
    required int userId,
    required bool isFollowing,
  }) async {
    userIds.add(userId);
    currentStates.add(isFollowing);
    return true;
  }
}

class PagedTopicPageRepository extends PageRepository {
  PagedTopicPageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
    if (path.startsWith('/p/post/')) {
      return parsePayload(pagedTopicPayloadJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

class AnchoredTopicPageRepository extends PageRepository {
  AnchoredTopicPageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
    if (path.startsWith('/p/post/')) {
      return parsePayload(anchoredTopicPayloadJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

/// Holds page loading open until a test explicitly resolves it.
class DelayedPageRepository extends PageRepository {
  DelayedPageRepository(super.client);

  final Completer<PagePayload> response = Completer<PagePayload>();

  @override
  Future<PagePayload> fetch(String path) => response.future;

  void complete(Map<String, dynamic> payload) {
    if (!response.isCompleted) response.complete(parsePayload(payload));
  }
}

Map<String, dynamic> redesignedTopicPayloadJson() {
  final Map<String, dynamic> json = topicDetailPayloadJson();
  final Map<String, dynamic> props = json['props'] as Map<String, dynamic>;
  final Map<String, dynamic> stream =
      props['postStream'] as Map<String, dynamic>;
  final List<dynamic> sourcePosts = stream['posts'] as List<dynamic>;
  final List<Map<String, dynamic>> posts = <Map<String, dynamic>>[
    Map<String, dynamic>.from(sourcePosts.first as Map<dynamic, dynamic>),
    for (int floor = 2; floor <= 14; floor++)
      <String, dynamic>{
        'id': 9000 + floor,
        'topicId': 100,
        'postNo': floor,
        'content': '第 $floor 楼回复：简洁而有力的移动端内容层级。',
        'renderedContent': '',
        'processStatus': 0,
        'isHidden': false,
        'canModerate': false,
        'author': <String, dynamic>{
          'id': floor,
          'username': 'user$floor',
          'nickname': '用户 $floor',
          'avatarUrl': '',
        },
        'createdAt':
            '2025-01-15T09:${floor.toString().padLeft(2, '0')}:00+08:00',
        'isOwnPost': false,
        'likeCount': floor,
        'isLiked': false,
        'isBookmarked': false,
      },
  ];
  stream
    ..['posts'] = posts
    ..['total'] = posts.length
    ..['maxPostNo'] = posts.length
    ..['hasAfter'] = false;
  final Map<String, dynamic> topic = props['topic'] as Map<String, dynamic>;
  topic
    ..['replyCount'] = posts.length - 1
    ..['maxPostNo'] = posts.length;
  return json;
}

Map<String, dynamic> redesignedProfilePayloadJson() {
  final Map<String, dynamic> json = userProfilePayloadJson();
  final Map<String, dynamic> props = json['props'] as Map<String, dynamic>;
  props['topics'] = <Object>[
    <String, dynamic>{
      'id': 101,
      'title': 'Alice 的移动端设计主题',
      'description': '对齐 Web 设计语言，同时保留移动端的信息效率。',
      'url': '/p/post/101',
      'author': <String, dynamic>{
        'id': 1,
        'username': 'alice',
        'avatarUrl': '',
      },
      'participants': <Object>[],
      'categories': <Object>[],
      'replyCount': 8,
      'viewCount': 128,
      'pinWeight': 0,
      'processStatus': 0,
      'activityText': '2025-01-15T09:30:00+08:00',
      'lastUpdateTime': '2025-01-15T09:30:00+08:00',
    },
  ];
  props['following'] = <Object>[
    <String, dynamic>{
      'id': 2,
      'username': 'bob',
      'nickname': 'Bob',
      'avatarUrl': '',
      'bio': '建筑与设计',
      'url': '/u/2',
    },
  ];
  return json;
}

Map<String, dynamic> followableProfilePayloadJson() {
  final Map<String, dynamic> json = redesignedProfilePayloadJson();
  final Map<String, dynamic> props = json['props'] as Map<String, dynamic>;
  final Map<String, dynamic> user = props['user'] as Map<String, dynamic>;
  user
    ..['userId'] = 2
    ..['isSelf'] = false
    ..['isFollowing'] = false;
  props
    ..['isOwnProfile'] = false
    ..['canFollow'] = true
    ..['canMessage'] = false
    ..['settingsUrl'] = '';
  return json;
}

PostPayload makePostPayload(int id, int postNo, String content) {
  return PostPayload(
    id: id,
    topicId: 100,
    postNo: postNo,
    content: content,
    renderedContent: '',
    processStatus: 0,
    isHidden: false,
    canModerate: false,
    author: UserBriefPayload(
      id: postNo,
      username: 'user$postNo',
      avatarUrl: '',
    ),
    createdAt: '2025-01-15T09:${postNo.toString().padLeft(2, '0')}:00+08:00',
    isOwnPost: false,
    likeCount: postNo,
    isLiked: false,
    isBookmarked: false,
  );
}

Map<String, dynamic> makePostJson(int id, int postNo, String content) {
  final PostPayload post = makePostPayload(id, postNo, content);
  return <String, dynamic>{...post.toJson(), 'author': post.author.toJson()};
}

Map<String, dynamic> pagedTopicPayloadJson() {
  final Map<String, dynamic> json = topicDetailPayloadJson();
  final Map<String, dynamic> props = json['props'] as Map<String, dynamic>;
  final Map<String, dynamic> stream =
      props['postStream'] as Map<String, dynamic>;
  final List<dynamic> initialPosts = stream['posts'] as List<dynamic>;
  stream
    ..['posts'] = <Object>[
      Map<String, dynamic>.from(initialPosts.first as Map<dynamic, dynamic>),
      makePostJson(9002, 2, '二楼内容'),
    ]
    ..['afterPostNo'] = 2
    ..['hasAfter'] = true
    ..['total'] = 5
    ..['maxPostNo'] = 5;
  final Map<String, dynamic> topic = props['topic'] as Map<String, dynamic>;
  topic
    ..['replyCount'] = 4
    ..['maxPostNo'] = 5;
  return json;
}

Map<String, dynamic> anchoredTopicPayloadJson() {
  final Map<String, dynamic> json = topicDetailPayloadJson();
  final Map<String, dynamic> props = json['props'] as Map<String, dynamic>;
  final Map<String, dynamic> topic = props['topic'] as Map<String, dynamic>;
  final Map<String, dynamic> stream =
      props['postStream'] as Map<String, dynamic>;
  topic['description'] = '服务端提供的主帖摘要';
  stream
    ..['posts'] = <Object>[makePostJson(9002, 2, '锚定窗口中的二楼回复')]
    ..['hasBefore'] = true
    ..['hasAfter'] = false
    ..['total'] = 3
    ..['maxPostNo'] = 3;
  return json;
}

/// 记录 search 调用 page 的 TopicRepository。

class ActivatingConversationPageRepository extends CountingPageRepository {
  ActivatingConversationPageRepository(super.client);

  int messagesFetches = 0;

  @override
  Future<PagePayload> fetch(String path) async {
    if (path != '/messages') return super.fetch(path);

    fetchCalls++;
    messagesFetches++;
    final Map<String, dynamic> payload = messagesPayloadJson();
    final Map<String, dynamic> props = payload['props'] as Map<String, dynamic>;
    final List<dynamic> conversations = props['conversations'] as List<dynamic>;
    if (messagesFetches == 1) {
      conversations.clear();
    } else {
      final Map<String, dynamic> conversation =
          conversations.first as Map<String, dynamic>;
      conversation
        ..['convId'] = 9
        ..['lastMsg'] = '刚刚发来的消息';
    }
    return parsePayload(payload);
  }
}

class ErroringProfilePageRepository extends CountingPageRepository {
  ErroringProfilePageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
    if (path.startsWith('/u/')) {
      fetchCalls++;
      throw StateError('profile request failed');
    }
    return super.fetch(path);
  }
}

/// 记录 search 调用 page 的 TopicRepository。
class PagingTopicRepository extends TopicRepository {
  PagingTopicRepository(super.client);

  final List<int> searchPages = [];

  @override
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
  }) async {
    searchPages.add(page);
    if (page <= 1) {
      return SearchPageProps(
        query: query,
        scope: scope,
        topics: [makeTopic(1, '结果-第一页')],
        users: const [],
        categories: const [],
        total: 2,
        usersTotal: 0,
        categoriesTotal: 0,
        totalPages: 2,
        pagination: const PaginationPayload(
          page: 1,
          nextPage: 2,
          hasNext: true,
          nextUrl: '/search?page=2',
        ),
      );
    }
    return SearchPageProps(
      query: query,
      scope: scope,
      topics: [makeTopic(2, '结果-第二页')],
      users: const [],
      categories: const [],
      total: 2,
      usersTotal: 0,
      categoriesTotal: 0,
      totalPages: 2,
      pagination: const PaginationPayload(
        page: 2,
        nextPage: 0,
        hasNext: false,
        nextUrl: '',
      ),
    );
  }
}

class AggregateSearchRepository extends PagingTopicRepository {
  AggregateSearchRepository(super.client);

  final List<String> scopes = <String>[];

  @override
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
  }) async {
    scopes.add(scope);
    return SearchPageProps(
      query: query,
      scope: scope,
      topics: <TopicPayload>[makeTopic(1, '聚合帖子结果')],
      users: const <UserSearchPayload>[
        UserSearchPayload(
          id: 2,
          username: 'bob',
          nickname: 'Bob',
          avatarUrl: '',
          bio: '校园开发者',
        ),
      ],
      categories: const <CategorySearchPayload>[
        CategorySearchPayload(
          id: 3,
          name: '开发',
          slug: 'dev',
          icon: '#',
          color: '#2563eb',
          desc: '技术交流',
        ),
      ],
      total: 1,
      usersTotal: 1,
      categoriesTotal: 1,
      totalPages: 1,
      pagination: const PaginationPayload(
        page: 1,
        nextPage: 0,
        hasNext: false,
        nextUrl: '',
      ),
    );
  }
}

class WindowTopicRepository extends TopicRepository {
  WindowTopicRepository(super.client);

  final List<int?> cursors = <int?>[];

  @override
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) async {
    cursors.add(afterPostNo);
    if (afterPostNo == 2) {
      return PostWindowPayload(
        posts: <PostPayload>[
          makePostPayload(9003, 3, '三楼内容'),
          makePostPayload(9004, 4, '四楼内容'),
        ],
        replyTargets: const <ReplyTargetPayload>[],
        afterPostNo: 4,
        hasBefore: false,
        hasAfter: true,
        total: 5,
        maxPostNo: 5,
      );
    }
    if (afterPostNo == 4) {
      return PostWindowPayload(
        posts: <PostPayload>[
          makePostPayload(9004, 4, '四楼内容'),
          makePostPayload(9005, 5, '五楼内容'),
        ],
        replyTargets: const <ReplyTargetPayload>[],
        afterPostNo: 5,
        hasBefore: false,
        hasAfter: false,
        total: 5,
        maxPostNo: 5,
      );
    }
    throw StateError('unexpected afterPostNo: $afterPostNo');
  }
}

class EmptyWindowTopicRepository extends TopicRepository {
  EmptyWindowTopicRepository(super.client);

  final List<int?> cursors = <int?>[];

  @override
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) async {
    cursors.add(afterPostNo);
    return PostWindowPayload(
      posts: const <PostPayload>[],
      replyTargets: const <ReplyTargetPayload>[],
      afterPostNo: afterPostNo,
      hasBefore: false,
      hasAfter: true,
      total: 5,
      maxPostNo: 5,
    );
  }
}

class StalledWindowTopicRepository extends TopicRepository {
  StalledWindowTopicRepository(super.client);

  final List<int?> cursors = <int?>[];

  @override
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) async {
    cursors.add(afterPostNo);
    return PostWindowPayload(
      posts: <PostPayload>[makePostPayload(9003, 3, '停滞游标返回的内容')],
      replyTargets: const <ReplyTargetPayload>[],
      afterPostNo: afterPostNo,
      hasBefore: false,
      hasAfter: true,
      total: 5,
      maxPostNo: 5,
    );
  }
}

class DraftsPageRepository extends PageRepository {
  DraftsPageRepository(super.client, {required this.editUrl});

  final String editUrl;

  @override
  Future<PagePayload> fetch(String path) async {
    if (path != '/drafts') {
      throw UnimplementedError('unexpected page path: $path');
    }
    return parsePayload(draftsPayloadJson(editUrl: editUrl));
  }
}

/// 记录 filter 的 NotificationRepository。
class FilteringNotificationRepository extends NotificationRepository {
  FilteringNotificationRepository(super.client);

  final List<String> filters = [];

  @override
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
  }) async {
    filters.add(filter);
    final NotificationPayload n = NotificationPayload(
      id: 1,
      eventType: 'reply',
      isRead: filter == 'all',
      createdAt: '2026-08-07T10:00:00+08:00',
      title: filter == 'all' ? '全部通知' : '未读通知',
      content: '内容',
      actor: const NotificationActorPayload(id: 1, username: 'alice'),
      payload: const NotificationInnerPayload(actorId: 1),
    );
    return NotificationListResponse(
      items: [n],
      nextCursor: 0,
      hasNext: false,
      unreadCount: 1,
    );
  }
}

/// 记录 logout 的 AuthRepository。
class LogoutAuthRepository extends AuthRepository {
  LogoutAuthRepository(super.client);

  int logoutCalls = 0;

  @override
  Future<bool> logout() async {
    logoutCalls++;
    return true;
  }
}

class RecordingPostRepository extends PostRepository {
  RecordingPostRepository(super.client);

  int? lastTopicId;
  String? lastContent;
  int? lastReplyToPostId;

  @override
  Future<CreatePostResult> createPost({
    required int topicId,
    required String content,
    int replyToPostId = 0,
    String? captchaId,
    String? captchaCode,
  }) async {
    lastTopicId = topicId;
    lastContent = content;
    lastReplyToPostId = replyToPostId;
    return const CreatePostResult(id: 9999, postNo: 15, renderedContent: '');
  }
}

class EmptySessionsUserRepository extends UserRepository {
  EmptySessionsUserRepository(super.client);

  int sessionCalls = 0;

  @override
  Future<List<UserSessionPayload>> listSessions() async {
    sessionCalls++;
    return const <UserSessionPayload>[];
  }
}

/// 构造最小 TopicPayload。
TopicPayload makeTopic(int id, String title) {
  return TopicPayload(
    id: id,
    title: title,
    description: '摘要',
    url: '/p/post/$id',
    author: const UserBriefPayload(id: 1, username: 'alice', avatarUrl: ''),
    participants: const [],
    categories: const [],
    replyCount: 0,
    viewCount: 10,
    pinWeight: 0,
    processStatus: 0,
    activityText: '2026-08-07T10:00:00+08:00',
    lastUpdateTime: '2026-08-07T10:00:00+08:00',
  );
}

/// 构造最小 settings payload(登录态用户 + 空会话)。
Map<String, dynamic> settingsPayloadJson() {
  return {
    'component': PageComponent.settings,
    'props': {
      'user': {
        'id': 1,
        'username': 'alice',
        'email': 'alice@example.com',
        'nickname': 'Alice',
        'locale': 'zh',
        'avatarUrl': '',
        'profileCoverUrl': '',
        'bio': '',
        'signature': '',
        'websiteName': '',
        'website': '',
        'prestige': 0,
        'createdAt': '2026-01-01T00:00:00+08:00',
        'externalInformation': <String, Object>{},
        'wornBadgeCode': '',
        'badges': <Object>[],
        'wearableBadges': <Object>[],
      },
      'stats': {
        'topicCount': 0,
        'replyCount': 0,
        'followerCount': 0,
        'followingCount': 0,
        'likeReceivedCount': 0,
        'likeGivenCount': 0,
        'collectionCount': 0,
        'createdAt': '2026-01-01T00:00:00+08:00',
      },
      'tabs': <Object>[],
    },
    'meta': {'title': '设置'},
    'layout': minimalLayoutJson(),
    'url': '/settings',
    'version': '1.0',
  };
}

Map<String, dynamic> draftsPayloadJson({required String editUrl}) {
  return <String, dynamic>{
    'component': PageComponent.drafts,
    'props': <String, dynamic>{
      'total': 1,
      'drafts': <Object>[
        <String, dynamic>{
          'id': 42,
          'title': '受控草稿',
          'description': '草稿摘要',
          'editUrl': editUrl,
          'replyCount': 0,
          'viewCount': 3,
          'processStatus': 0,
          'updatedAt': '2026-08-07T10:00:00+08:00',
          'createdAt': '2026-08-06T10:00:00+08:00',
          'categories': <Object>[],
        },
      ],
      'pagination': <String, dynamic>{
        'page': 1,
        'nextPage': 0,
        'hasNext': false,
        'nextUrl': '',
      },
    },
    'meta': <String, dynamic>{'title': '草稿箱'},
    'layout': minimalLayoutJson(),
    'url': '/drafts',
    'version': '1.0',
  };
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  Future<ProviderContainer> makeContainer({
    required PageRepository pageRepo,
    TopicRepository? topicRepo,
    PostRepository? postRepo,
    FilteringNotificationRepository? notifRepo,
    LogoutAuthRepository? authRepo,
    ChatRepository? chatRepo,
    UserRepository? userRepo,
    OfflineTopicCache? topicCache,
    OfflineChatCache? chatCache,
    int? currentUserId,
  }) async {
    final storage = MemTokenStorage()..write('token');
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(storage),
        currentUserProvider.overrideWith(
          (ref) async => currentUserId == null
              ? null
              : CurrentUser(id: currentUserId, username: 'alice'),
        ),
        pageRepositoryProvider.overrideWithValue(pageRepo),
        topicRepositoryProvider.overrideWithValue(
          topicRepo ?? PagingTopicRepository(client),
        ),
        postRepositoryProvider.overrideWithValue(
          postRepo ?? PostRepository(client),
        ),
        notificationRepositoryProvider.overrideWithValue(
          notifRepo ?? FilteringNotificationRepository(client),
        ),
        authRepositoryProvider.overrideWithValue(
          authRepo ?? LogoutAuthRepository(client),
        ),
        if (chatRepo != null)
          chatRepositoryProvider.overrideWithValue(chatRepo),
        userRepositoryProvider.overrideWithValue(
          userRepo ?? EmptySessionsUserRepository(client),
        ),
        offlineTopicCacheProvider.overrideWithValue(
          topicCache ?? NoopCache(),
        ),
        offlineChatCacheProvider.overrideWithValue(chatCache ?? NoopCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  Widget app(ProviderContainer container, Widget home) {
    return UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('zh'),
        home: home,
      ),
    );
  }

  GoRouter profileRouter({required String initialLocation}) {
    return GoRouter(
      initialLocation: initialLocation,
      routes: <RouteBase>[
        GoRoute(path: '/profile', builder: (_, _) => const ProfilePage()),
        GoRoute(
          path: '/u/:userId',
          builder: (_, GoRouterState state) =>
              ProfilePage(userId: int.parse(state.pathParameters['userId']!)),
        ),
        GoRoute(
          path: '/messages',
          builder: (_, GoRouterState state) => MessagesPage(
            targetUserId: int.tryParse(
              state.uri.queryParameters['userId'] ?? '',
            ),
            targetUsername: state.uri.queryParameters['username'] ?? '',
            targetAvatarUrl: state.uri.queryParameters['avatar'] ?? '',
          ),
        ),
        GoRoute(
          path: '/settings',
          builder: (_, _) =>
              const Scaffold(body: Center(child: Text('settings-page'))),
        ),
        GoRoute(
          path: '/login',
          builder: (_, _) =>
              const Scaffold(body: Center(child: Text('login-page'))),
        ),
      ],
    );
  }

  Widget routerApp(ProviderContainer container, GoRouter router) {
    return UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        routerConfig: router,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('zh'),
      ),
    );
  }

  group('首页下拉刷新', () {
    testWidgets('下拉触发重载第一页(fetch 再次调用)', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const HomePage()));
      await tester.pumpAndSettle();

      expect(find.text('移动端测试话题'), findsOneWidget);
      final int callsBefore = pageRepo.fetchCalls;
      expect(callsBefore, greaterThanOrEqualTo(1));

      // 下拉手势触发 RefreshIndicator。
      await tester.fling(find.byType(GfTopicList), const Offset(0, 400), 1200);
      await tester.pumpAndSettle();

      expect(
        pageRepo.fetchCalls,
        greaterThan(callsBefore),
        reason: '下拉刷新应再次调用 fetch 重载第一页',
      );
    });
  });

  group('首页话题布局', () {
    testWidgets('卡片和列表可通过胶囊切换且记住选择', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const HomePage()));
      await tester.pumpAndSettle();

      expect(find.byType(GfTopicCard), findsOneWidget);
      expect(find.byType(GfTopicRow), findsNothing);
      expect(find.text('新建话题'), findsNothing);
      final Finder feedSwitch = find.byType(GfPillSwitch<GfTopicFeedMode>);
      expect(tester.getSize(feedSwitch).height, 32);
      expect(
        tester.getCenter(feedSwitch).dy,
        closeTo(tester.getCenter(find.byType(GfTabBar)).dy, 1),
      );
      final Finder brandLogo = find.byType(Image);
      expect(brandLogo, findsOneWidget);
      expect(tester.getTopLeft(brandLogo).dx, inInclusiveRange(0, 24));
      expect(tester.getSize(brandLogo), const Size(128, 34));

      await tester.tap(find.text('列表'));
      await tester.pumpAndSettle();

      expect(find.byType(GfTopicCard), findsNothing);
      expect(find.byType(GfTopicRow), findsOneWidget);
      final SharedPreferences preferences =
          await SharedPreferences.getInstance();
      expect(preferences.getString('goose:home-feed-mode'), 'list');
    });
  });

  group('消息轮询', () {
    testWidgets('隐藏分支暂停轮询，重新可见后立即刷新', (tester) async {
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final CountingMessagesPageRepository pageRepo =
          CountingMessagesPageRepository(client);
      final ProviderContainer container = await makeContainer(
        pageRepo: pageRepo,
      );
      bool active = false;
      late StateSetter setTickerMode;

      await tester.pumpWidget(
        app(
          container,
          StatefulBuilder(
            builder: (BuildContext context, StateSetter setState) {
              setTickerMode = setState;
              return TickerMode(enabled: active, child: const MessagesPage());
            },
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(pageRepo.fetchCalls, 1);

      await tester.pump(const Duration(seconds: 16));
      expect(pageRepo.fetchCalls, 1);

      setTickerMode(() => active = true);
      await tester.pumpAndSettle();
      expect(pageRepo.fetchCalls, 2);

      await tester.pump(const Duration(seconds: 15));
      await tester.pumpAndSettle();
      expect(pageRepo.fetchCalls, 3);

      await tester.pumpWidget(const SizedBox.shrink());
    });
  });

  group('登录表单', () {
    testWidgets('忘记密码入口不被遮挡并可进入重置模式', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const LoginPage()));
      await tester.pumpAndSettle();

      final Finder forgotPassword = find.text('忘记密码？');
      expect(forgotPassword, findsOneWidget);
      expect(forgotPassword.hitTestable(), findsOneWidget);

      await tester.tap(forgotPassword);
      await tester.pumpAndSettle();

      expect(find.text('重置密码'), findsOneWidget);
      expect(find.text('邮箱'), findsOneWidget);
      expect(find.text('返回登录'), findsOneWidget);
    });

    testWidgets('从登录页返回 pop 回上一页', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      final GoRouter router = GoRouter(
        initialLocation: '/host',
        routes: <RouteBase>[
          GoRoute(
            path: '/host',
            builder: (BuildContext context, GoRouterState state) => Scaffold(
              body: Center(
                child: FilledButton(
                  onPressed: () => context.push('/login'),
                  child: const Text('打开登录'),
                ),
              ),
            ),
          ),
          GoRoute(path: '/login', builder: (_, _) => const LoginPage()),
        ],
      );

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.tap(find.text('打开登录'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/login');

      await tester.tap(find.byTooltip('返回'));
      await tester.pumpAndSettle();

      expect(router.state.uri.path, '/host');
      expect(find.text('打开登录'), findsOneWidget);
    });
  });

  group('话题回复', () {
    testWidgets('点击帖子回复后展开编辑器并自动聚焦', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
      await tester.pumpAndSettle();

      expect(find.byType(GfPostComposer), findsNothing);
      await tester.tap(find.byTooltip('回复').first);
      await tester.pump();

      final Finder composer = find.byType(GfPostComposer);
      final Finder replyField = find.descendant(
        of: composer,
        matching: find.byType(TextField),
      );
      expect(composer, findsOneWidget);
      expect(replyField, findsOneWidget);
      final TextField textField = tester.widget<TextField>(replyField);
      expect(textField.focusNode, isNotNull);
      expect(textField.focusNode!.hasFocus, isTrue);
      expect(textField.controller!.text, '@bob ');

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    });
  });

  group('搜索分页', () {
    testWidgets('加载更多调用 page+1 并追加结果', (tester) async {
      final topicRepo = PagingTopicRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(
        pageRepo: pageRepo,
        topicRepo: topicRepo,
      );
      await tester.pumpWidget(app(container, const SearchPage()));
      await tester.pumpAndSettle();

      // 输入关键词并搜索。
      await tester.enterText(find.byType(TextField), 'flutter');
      await tester.tap(find.byIcon(Icons.search));
      await tester.pumpAndSettle();

      expect(find.text('结果-第一页'), findsOneWidget);
      expect(topicRepo.searchPages, [1]);

      // 点击"加载更多"触发 page+1。
      await tester.tap(find.text('加载更多'));
      await tester.pumpAndSettle();

      expect(topicRepo.searchPages, [1, 2], reason: '加载更多应调用 search(page+1)');
      expect(find.text('结果-第一页'), findsOneWidget);
      expect(find.text('结果-第二页'), findsOneWidget, reason: '第二页结果应追加到列表');
    });

    testWidgets('全部范围同时展示帖子、用户与分类，并使用复数 scope', (tester) async {
      final topicRepo = AggregateSearchRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(
        pageRepo: pageRepo,
        topicRepo: topicRepo,
      );
      await tester.pumpWidget(app(container, const SearchPage()));
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextField), '同济');
      await tester.tap(find.byIcon(Icons.search));
      await tester.pumpAndSettle();

      expect(topicRepo.scopes, <String>['']);
      expect(find.text('聚合帖子结果'), findsOneWidget);
      expect(find.text('Bob'), findsOneWidget);
      expect(find.text('@bob'), findsOneWidget);
      expect(find.text('开发'), findsOneWidget);

      await tester.tap(find.text('用户').first);
      await tester.pumpAndSettle();
      expect(topicRepo.scopes.last, 'users');
    });
  });

  group('私信列表', () {
    testWidgets('新私信弹层可渲染并搜索用户', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('新私信'));
      await tester.pumpAndSettle();

      expect(find.text('新私信'), findsOneWidget);
      expect(find.text('Bob'), findsOneWidget);
      expect(find.text('@bob'), findsOneWidget);
      expect(tester.takeException(), isNull);

      await tester.enterText(find.byType(TextField).last, 'bob');
      await tester.pumpAndSettle();
      expect(find.text('Bob'), findsOneWidget);

      await tester.tap(find.byIcon(Icons.close));
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets('选择新用户不发空消息，首条真实消息才建立会话', (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final chatRepo = RecordingChatRepository(client);
      final pageRepo = CountingPageRepository(client);
      final container = await makeContainer(
        pageRepo: pageRepo,
        chatRepo: chatRepo,
      );
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('新私信'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Dave'));
      await tester.pumpAndSettle();

      expect(chatRepo.sent, isEmpty, reason: '选联系人时不应向后端发送空消息');
      expect(find.text('开始聊天'), findsOneWidget);
      expect(find.text('给 Dave 发出第一条消息。'), findsOneWidget);

      await tester.enterText(find.byType(TextField), '你好');
      await tester.pump();
      await tester.tap(find.text('发送'));
      await tester.pumpAndSettle();
      expect(chatRepo.sent, <(int, String)>[(4, '你好')]);

      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets('会话搜索同时过滤用户名和消息预览', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();

      expect(find.text('bob'), findsOneWidget);
      expect(find.text('carol'), findsOneWidget);
      await tester.enterText(find.byType(TextField), '樱花');
      await tester.pumpAndSettle();

      expect(find.text('bob'), findsNothing);
      expect(find.text('carol'), findsOneWidget);

      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets('重复轮询不重复写缓存或上报已读', (tester) async {
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final PollingChatRepository chatRepo = PollingChatRepository(
        client,
        messages: <ChatMessagePayload>[makeChatMessage(101)],
      );
      final RecordingChatCache chatCache = RecordingChatCache();
      final ProviderContainer container = await makeContainer(
        pageRepo: CountingPageRepository(client),
        chatRepo: chatRepo,
        chatCache: chatCache,
      );
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();

      expect(chatRepo.afterCalls, 1);
      expect(chatRepo.markReadCalls, 2);
      expect(chatCache.putMessageCalls, 1);
      expect(chatCache.storedMessageIds, <List<int>>[
        <int>[101],
      ]);
      expect(find.byIcon(Icons.more_horiz), findsNothing);

      await tester.pump(const Duration(seconds: 15));
      await tester.pumpAndSettle();

      expect(chatRepo.afterCalls, 2);
      expect(chatRepo.markReadCalls, 2);
      expect(chatCache.putMessageCalls, 1);

      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets('历史消息加载失败后清理 loading 状态并允许重试', (tester) async {
      tester.view.physicalSize = const Size(390, 700);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final PollingChatRepository chatRepo = PollingChatRepository(
        client,
        messages: List<ChatMessagePayload>.generate(
          40,
          (int index) => makeChatMessage(100 + index),
        ),
        hasMoreBefore: true,
        failOlder: true,
      );
      final ProviderContainer container = await makeContainer(
        pageRepo: CountingPageRepository(client),
        chatRepo: chatRepo,
      );
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();
      final Finder messageList = find.byType(ListView).last;

      await tester.fling(messageList, const Offset(0, 3000), 3000);
      await tester.pumpAndSettle();
      final int callsAfterFailure = chatRepo.beforeCalls;
      expect(callsAfterFailure, greaterThanOrEqualTo(1));
      expect(tester.takeException(), isNull);
      expect(find.byType(GfLoadingIndicator), findsNothing);

      await tester.drag(messageList, const Offset(0, -160));
      await tester.pumpAndSettle();
      await tester.drag(messageList, const Offset(0, 240));
      await tester.pumpAndSettle();
      expect(chatRepo.beforeCalls, greaterThan(callsAfterFailure));
      expect(tester.takeException(), isNull);

      await tester.pumpWidget(const SizedBox.shrink());
    });
  });

  group('话题分页', () {
    testWidgets('锚定窗口不把第一条回复误作主帖', (tester) async {
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final ProviderContainer container = await makeContainer(
        pageRepo: AnchoredTopicPageRepository(client),
      );
      await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
      await tester.pumpAndSettle();

      expect(find.text('服务端提供的主帖摘要'), findsOneWidget);
      expect(find.text('锚定窗口中的二楼回复'), findsOneWidget);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    });
    testWidgets('加载更多用服务端 cursor 续拉并去重追加', (tester) async {
      tester.view.physicalSize = const Size(1080, 2400);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);

      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final WindowTopicRepository topicRepo = WindowTopicRepository(client);
      final ProviderContainer container = await makeContainer(
        pageRepo: PagedTopicPageRepository(client),
        topicRepo: topicRepo,
      );
      await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
      await tester.pumpAndSettle();

      expect(find.text('二楼内容'), findsOneWidget);
      await tester.ensureVisible(find.text('加载更多'));
      await tester.tap(find.text('加载更多'));
      await tester.pumpAndSettle();

      expect(topicRepo.cursors, <int?>[2]);
      expect(find.text('三楼内容'), findsOneWidget);
      expect(find.text('四楼内容'), findsOneWidget);

      await tester.ensureVisible(find.text('加载更多'));
      await tester.tap(find.text('加载更多'));
      await tester.pumpAndSettle();

      expect(topicRepo.cursors, <int?>[2, 4]);
      expect(find.text('四楼内容'), findsOneWidget);
      expect(find.text('五楼内容'), findsOneWidget);
      expect(find.text('加载更多'), findsNothing);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    });
  });

  testWidgets('服务端返回空窗口时立即终止加载更多', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    final GfApiClient client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemTokenStorage(),
      baseUrl: 'http://fake.local',
    );
    final EmptyWindowTopicRepository topicRepo = EmptyWindowTopicRepository(
      client,
    );
    final ProviderContainer container = await makeContainer(
      pageRepo: PagedTopicPageRepository(client),
      topicRepo: topicRepo,
    );
    await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('加载更多'));
    await tester.tap(find.text('加载更多'));
    await tester.pumpAndSettle();

    expect(topicRepo.cursors, <int?>[2]);
    expect(find.text('加载更多'), findsNothing);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('服务端游标未前进时追加本次内容后终止加载更多', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    final GfApiClient client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemTokenStorage(),
      baseUrl: 'http://fake.local',
    );
    final StalledWindowTopicRepository topicRepo = StalledWindowTopicRepository(
      client,
    );
    final ProviderContainer container = await makeContainer(
      pageRepo: PagedTopicPageRepository(client),
      topicRepo: topicRepo,
    );
    await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('加载更多'));
    await tester.tap(find.text('加载更多'));
    await tester.pumpAndSettle();

    expect(topicRepo.cursors, <int?>[2]);
    expect(find.text('停滞游标返回的内容'), findsOneWidget);
    expect(find.text('加载更多'), findsNothing);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  group('草稿导航', () {
    testWidgets('草稿条目忽略服务端外部 URL 并进入内部编辑路由', (tester) async {
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final ProviderContainer container = await makeContainer(
        pageRepo: DraftsPageRepository(
          client,
          editUrl: 'https://unexpected.invalid/publish?id=999',
        ),
      );
      final GoRouter router = GoRouter(
        initialLocation: '/drafts',
        routes: <RouteBase>[
          GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
          GoRoute(
            path: '/publish',
            builder: (BuildContext context, GoRouterState state) => Scaffold(
              body: Center(
                child: Text('publish-${state.uri.queryParameters['id']}'),
              ),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('受控草稿'));
      await tester.pumpAndSettle();

      expect(router.state.uri.toString(), '/publish?id=42');
      expect(find.text('publish-42'), findsOneWidget);
    });
  });

  group('个人主页发私信', () {
    test('messageUrl 正确解码 Go QueryEscape 的空格', () {
      final Uri uri = Uri.parse(
        '/messages?userId=4&username=Bob+Smith&avatar=',
      );

      expect(uri.queryParameters['username'], 'Bob Smith');
    });

    testWidgets('按 messageUrl 打开目标会话且返回原用户主页', (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final pageRepo = CountingPageRepository(client);
      final chatRepo = RecordingChatRepository(client);
      final container = await makeContainer(
        pageRepo: pageRepo,
        chatRepo: chatRepo,
      );
      final GoRouter router = profileRouter(initialLocation: '/u/2');

      tester.view.physicalSize = const Size(1080, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(routerApp(container, router));
      await tester.pumpAndSettle();

      expect(find.text('Bob'), findsOneWidget);
      await tester.tap(find.text('新私信'));
      await tester.pumpAndSettle();

      expect(router.state.uri.path, '/messages');
      expect(router.state.uri.queryParameters['userId'], '2');
      expect(find.text('bob'), findsOneWidget);
      expect(find.text('开始聊天'), findsOneWidget);
      expect(router.canPop(), isTrue);

      router.pop();
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/u/2');
      expect(find.text('Bob'), findsOneWidget);

      await tester.pumpWidget(const SizedBox.shrink());
    });
  });

  testWidgets('轮询发现新会话后立即加载对方消息', (tester) async {
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemTokenStorage(),
      baseUrl: 'http://fake.local',
    );
    final pageRepo = ActivatingConversationPageRepository(client);
    final chatRepo = IncomingChatRepository(client);
    final container = await makeContainer(
      pageRepo: pageRepo,
      chatRepo: chatRepo,
    );

    await tester.pumpWidget(
      app(
        container,
        const MessagesPage(targetUserId: 2, targetUsername: 'Bob'),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('开始聊天'), findsOneWidget);
    expect(chatRepo.fetchedConvIds, isEmpty);

    await tester.pump(const Duration(seconds: 15));
    await tester.pumpAndSettle();

    expect(pageRepo.messagesFetches, greaterThanOrEqualTo(2));
    expect(chatRepo.fetchedConvIds, contains(9));
    expect(find.text('刚刚发来的消息'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
  });

  group('个人主页错误态', () {
    testWidgets('未登录时仍展示登录与设置入口', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      final GoRouter router = profileRouter(initialLocation: '/profile');

      await tester.pumpWidget(routerApp(container, router));
      await tester.pumpAndSettle();

      expect(find.text('未登录'), findsOneWidget);
      expect(find.text('登录'), findsOneWidget);
      expect(find.text('设置'), findsOneWidget);
    });

    testWidgets('未登录时登录入口可导航', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      final GoRouter router = profileRouter(initialLocation: '/profile');

      await tester.pumpWidget(routerApp(container, router));
      await tester.pumpAndSettle();

      await tester.tap(find.text('登录'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/login');
      expect(find.text('login-page'), findsOneWidget);
    });

    testWidgets('资料请求失败时设置入口仍可导航', (tester) async {
      final pageRepo = ErroringProfilePageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(
        pageRepo: pageRepo,
        currentUserId: 1,
      );
      final GoRouter router = profileRouter(initialLocation: '/profile');

      await tester.pumpWidget(routerApp(container, router));
      await tester.pumpAndSettle();

      expect(find.textContaining('profile request failed'), findsOneWidget);
      expect(find.text('登录'), findsNothing);
      expect(find.text('设置'), findsOneWidget);

      await tester.tap(find.text('设置'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/settings');
      expect(find.text('settings-page'), findsOneWidget);
    });
  });

  group('通知筛选', () {
    testWidgets('all/unread 切换触发重新请求(filter 参数变化)', (tester) async {
      final notifRepo = FilteringNotificationRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(
        pageRepo: pageRepo,
        notifRepo: notifRepo,
      );
      await tester.pumpWidget(app(container, const NotificationsPage()));
      await tester.pumpAndSettle();

      expect(notifRepo.filters, ['all']);
      expect(find.text('全部通知'), findsOneWidget);

      // 切到"未读"。
      await tester.tap(find.text('未读'));
      await tester.pumpAndSettle();

      expect(notifRepo.filters, [
        'all',
        'unread',
      ], reason: '切换 tab 应以 filter=unread 重新请求');
      expect(find.text('未读通知'), findsOneWidget);
    });
  });

  group('设置页加载与导航', () {
    testWidgets('首次加载显示设置骨架', (tester) async {
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final DelayedPageRepository pageRepo = DelayedPageRepository(client);
      final ProviderContainer container = await makeContainer(
        pageRepo: pageRepo,
        userRepo: EmptySessionsUserRepository(client),
      );

      await tester.pumpWidget(app(container, const SettingsPage()));
      await tester.pump();
      expect(find.byType(GfSettingsSkeleton), findsOneWidget);

      pageRepo.complete(settingsPayloadJson());
      await tester.pumpAndSettle();
      expect(find.byType(GfSettingsSkeleton), findsNothing);
      expect(find.text('个人资料'), findsOneWidget);
    });

    testWidgets('下拉刷新再次请求设置数据', (tester) async {
      final CountingPageRepository pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final ProviderContainer container = await makeContainer(
        pageRepo: pageRepo,
      );
      await tester.pumpWidget(app(container, const SettingsPage()));
      await tester.pumpAndSettle();
      final int callsBefore = pageRepo.fetchCalls;

      await tester.fling(
        find.byType(ListView).first,
        const Offset(0, 400),
        1200,
      );
      await tester.pumpAndSettle();

      expect(pageRepo.fetchCalls, greaterThan(callsBefore));
    });

    testWidgets('从设置页返回 pop 回上一页', (tester) async {
      final CountingPageRepository pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final ProviderContainer container = await makeContainer(
        pageRepo: pageRepo,
      );
      final GoRouter router = GoRouter(
        initialLocation: '/host',
        routes: <RouteBase>[
          GoRoute(
            path: '/host',
            builder: (BuildContext context, GoRouterState state) => Scaffold(
              body: Center(
                child: FilledButton(
                  onPressed: () => context.push('/settings'),
                  child: const Text('打开设置'),
                ),
              ),
            ),
          ),
          GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
        ],
      );

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.tap(find.text('打开设置'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/settings');

      await tester.tap(find.byTooltip('返回'));
      await tester.pumpAndSettle();

      expect(router.state.uri.path, '/host');
      expect(find.text('打开设置'), findsOneWidget);
    });
  });

  group('设置登出', () {
    testWidgets('登出调用 authRepository.logout、清 token 并跳转登录页', (tester) async {
      final authRepo = LogoutAuthRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final storage = MemTokenStorage()..write('token');
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: storage,
        baseUrl: 'http://fake.local',
      );
      final topicCache = RecordingCache();
      final chatCache = RecordingCache();
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          pageRepositoryProvider.overrideWithValue(pageRepo),
          topicRepositoryProvider.overrideWithValue(
            PagingTopicRepository(client),
          ),
          notificationRepositoryProvider.overrideWithValue(
            FilteringNotificationRepository(client),
          ),
          authRepositoryProvider.overrideWithValue(authRepo),
          offlineTopicCacheProvider.overrideWithValue(topicCache),
          offlineChatCacheProvider.overrideWithValue(chatCache),
        ],
      );
      addTearDown(container.dispose);

      final router = GoRouter(
        initialLocation: '/settings',
        routes: [
          GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
          GoRoute(
            path: '/login',
            builder: (_, _) =>
                const Scaffold(body: Center(child: Text('login-page'))),
          ),
        ],
      );

      // 设置更大视口,保证安全 tab 内"退出登录"按钮无需滚动即可见。
      tester.view.physicalSize = const Size(1080, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // 切到 Security tab 找登出按钮。
      await tester.tap(find.text('安全'));
      await tester.pumpAndSettle();
      expect(find.text('退出登录'), findsOneWidget);

      // 点击登出 → 确认对话框 → 确认。
      await tester.tap(find.text('退出登录'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('退出登录').last);
      await tester.pumpAndSettle();

      expect(authRepo.logoutCalls, 1, reason: '登出应调用 authRepository.logout');
      expect(await storage.read(), isNull, reason: '登出应清空本地 token');
      expect(
        topicCache.clears + chatCache.clears,
        2,
        reason: '登出应清空话题与私信离线缓存,防止跨账号数据泄漏',
      );
      expect(router.state.uri.path, '/login', reason: '登出后应跳转登录页');
      expect(find.text('login-page'), findsOneWidget);
    });
  });

  group('核心页面移动端交互', () {
    testWidgets('长话题出现回顶按钮，打开回复编辑器后隐藏', (tester) async {
      tester.view.physicalSize = const Size(390, 700);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);

      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final container = await makeContainer(
        pageRepo: RedesignPageRepository(client),
      );
      await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
      await tester.pumpAndSettle();

      expect(find.byType(GfPostComposer), findsNothing);
      expect(find.byTooltip('返回顶部'), findsNothing);
      final Finder topicList = find.byWidgetPredicate(
        (Widget widget) =>
            widget is CustomScrollView &&
            widget.physics is AlwaysScrollableScrollPhysics,
        description: 'topic page primary scroll view',
      );
      await tester.drag(topicList, const Offset(0, -900));
      await tester.pumpAndSettle();
      expect(find.byTooltip('返回顶部'), findsOneWidget);

      await tester.tap(find.text('参与讨论'));
      await tester.pumpAndSettle();
      expect(find.byType(GfPostComposer), findsOneWidget);
      expect(find.byTooltip('添加图片'), findsOneWidget);
      expect(find.byTooltip('返回顶部'), findsNothing);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    });

    testWidgets('取消定向回复后重新参与讨论不会沿用旧回复目标', (tester) async {
      tester.view.physicalSize = const Size(390, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);

      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final RecordingPostRepository postRepo = RecordingPostRepository(client);
      final ProviderContainer container = await makeContainer(
        pageRepo: RedesignPageRepository(client),
        postRepo: postRepo,
      );
      await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('回复').first);
      await tester.pumpAndSettle();
      expect(find.text('回复 用户 2'), findsOneWidget);
      expect(find.text('@user2 '), findsOneWidget);

      await tester.tap(find.byTooltip('取消'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('参与讨论'));
      await tester.pumpAndSettle();
      expect(find.text('回复 用户 2'), findsNothing);
      expect(find.text('@user2 '), findsNothing);

      await tester.enterText(find.byType(TextField).last, '普通回复');
      await tester.pump();
      final Finder sendButton = find.descendant(
        of: find.byType(GfPostComposer),
        matching: find.widgetWithText(GfButton, '发送'),
      );
      expect(sendButton, findsOneWidget);
      expect(tester.widget<GfButton>(sendButton).onPressed, isNotNull);
      await tester.ensureVisible(sendButton);
      await tester.tap(sendButton);
      await tester.pumpAndSettle();

      expect(postRepo.lastTopicId, 100);
      expect(postRepo.lastContent, '普通回复');
      expect(postRepo.lastReplyToPostId, 0);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 3100));
    });

    testWidgets('个人主页提供明确返回并从标签内容导航', (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final container = await makeContainer(
        pageRepo: RedesignPageRepository(client),
      );
      late final GoRouter router;
      router = GoRouter(
        initialLocation: '/profile-host',
        routes: <RouteBase>[
          GoRoute(
            path: '/profile-host',
            builder: (BuildContext context, GoRouterState state) => Scaffold(
              body: Center(
                child: FilledButton(
                  onPressed: () => context.push('/u/1'),
                  child: const Text('打开个人主页'),
                ),
              ),
            ),
          ),
          GoRoute(
            path: '/u/:id',
            builder: (BuildContext context, GoRouterState state) {
              final int id = int.parse(state.pathParameters['id']!);
              return id == 1
                  ? const ProfilePage(userId: 1)
                  : Scaffold(body: Center(child: Text('user-$id')));
            },
          ),
          GoRoute(
            path: '/p/:id',
            builder: (BuildContext context, GoRouterState state) => Scaffold(
              body: Center(child: Text("topic-${state.pathParameters['id']}")),
            ),
          ),
        ],
      );

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.tap(find.text('打开个人主页'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/u/1');
      expect(router.canPop(), isTrue);

      // TDesign default back occupies the standard 44dp top-left target.
      await tester.tapAt(const Offset(28, 28));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/profile-host');

      await tester.tap(find.text('打开个人主页'));
      await tester.pumpAndSettle();
      await tester.tap(
        find.descendant(of: find.byType(GfTabBar), matching: find.text('主题')),
      );
      await tester.pumpAndSettle();
      expect(find.text('Alice 的移动端设计主题'), findsOneWidget);
      await tester.tap(find.text('Alice 的移动端设计主题'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/p/101');

      router.pop();
      await tester.pumpAndSettle();
      await tester.tap(
        find.descendant(of: find.byType(GfTabBar), matching: find.text('关注')),
      );
      await tester.pumpAndSettle();
      expect(find.text('Bob'), findsOneWidget);
      await tester.tap(find.text('Bob'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/u/2');
    });

    testWidgets('关注按钮按操作前状态映射关注与取消关注', (tester) async {
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final RecordingFollowTopicRepository topicRepo =
          RecordingFollowTopicRepository(client);
      final ProviderContainer container = await makeContainer(
        pageRepo: RedesignPageRepository(
          client,
          profilePayload: followableProfilePayloadJson(),
        ),
        topicRepo: topicRepo,
      );

      await tester.pumpWidget(app(container, const ProfilePage(userId: 2)));
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(GfButton, '关注'));
      await tester.pump();
      expect(topicRepo.userIds, <int>[2]);
      expect(topicRepo.currentStates, <bool>[false]);
      expect(find.widgetWithText(GfButton, '已关注'), findsOneWidget);

      await tester.tap(find.widgetWithText(GfButton, '已关注'));
      await tester.pump();
      expect(topicRepo.userIds, <int>[2, 2]);
      expect(topicRepo.currentStates, <bool>[false, true]);
      expect(find.widgetWithText(GfButton, '关注'), findsOneWidget);
    });
  });

  group('结构化加载态', () {
    testWidgets('首页等待数据时显示信息流骨架', (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final repo = DelayedPageRepository(client);
      final container = await makeContainer(pageRepo: repo);
      await tester.pumpWidget(app(container, const HomePage()));
      await tester.pump();
      expect(find.byType(GfTopicFeedSkeleton), findsOneWidget);

      repo.complete(homePayloadJson());
      await tester.pumpAndSettle();
      expect(find.byType(GfTopicFeedSkeleton), findsNothing);
      expect(find.text('移动端测试话题'), findsOneWidget);
    });

    testWidgets('话题等待数据时显示详情骨架', (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final repo = DelayedPageRepository(client);
      final container = await makeContainer(pageRepo: repo);
      await tester.pumpWidget(app(container, const TopicPage(topicId: 100)));
      await tester.pump();
      expect(find.byType(GfTopicDetailSkeleton), findsOneWidget);

      repo.complete(topicDetailPayloadJson());
      await tester.pumpAndSettle();
      expect(find.byType(GfTopicDetailSkeleton), findsNothing);
      expect(find.text('移动端测试话题'), findsWidgets);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    });

    testWidgets('个人主页等待数据时显示身份卡骨架', (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final repo = DelayedPageRepository(client);
      final container = await makeContainer(pageRepo: repo);
      await tester.pumpWidget(app(container, const ProfilePage(userId: 1)));
      await tester.pump();
      expect(find.byType(GfProfileSkeleton), findsOneWidget);

      repo.complete(userProfilePayloadJson());
      await tester.pumpAndSettle();
      expect(find.byType(GfProfileSkeleton), findsNothing);
      expect(find.text('Alice'), findsOneWidget);
    });
  });

  group('离线缓存清理(登出/换账号数据泄漏)', () {
    testWidgets('进入登录页即清空上一账号的离线缓存(换账号场景)', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final topicCache = RecordingCache();
      final chatCache = RecordingCache();
      final container = await makeContainer(
        pageRepo: pageRepo,
        topicCache: topicCache,
        chatCache: chatCache,
      );

      // 模拟上一账号遗留的缓存:非空缓存应被清空。
      await tester.pumpWidget(app(container, const LoginPage()));
      await tester.pumpAndSettle();

      expect(
        topicCache.clears + chatCache.clears,
        2,
        reason: '进入登录页应清空话题与私信离线缓存,换账号后不能读到上一账号数据',
      );
    });
  });
}

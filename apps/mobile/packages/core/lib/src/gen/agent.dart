import 'package:freezed_annotation/freezed_annotation.dart';

import 'search.dart';
import 'topic.dart';

part 'agent.freezed.dart';
part 'agent.g.dart';

@freezed
abstract class AgentMeResult with _$AgentMeResult {
  const factory AgentMeResult({
    required int agentId,
    required String username,
    required String nickname,
    required String avatarUrl,
    required String tokenPrefix,
    required int enabled,
    required int createdAt,
    required int updatedAt,
  }) = _AgentMeResult;

  factory AgentMeResult.fromJson(Map<String, dynamic> json) =>
      _$AgentMeResultFromJson(json);
}

@freezed
abstract class AgentTopicItem with _$AgentTopicItem {
  const factory AgentTopicItem({
    required int id,
    required String title,
    required String excerpt,
    required List<int> categoryIds,
    required int userId,
    required int status,
    required int processStatus,
    required int replyCount,
    required int viewCount,
    required int postCount,
    int? lastPostedAt,
    required int createdAt,
    required int updatedAt,
  }) = _AgentTopicItem;

  factory AgentTopicItem.fromJson(Map<String, dynamic> json) =>
      _$AgentTopicItemFromJson(json);
}

@freezed
abstract class AgentTopicListResult with _$AgentTopicListResult {
  const factory AgentTopicListResult({
    required List<AgentTopicItem> list,
    required int page,
    required int pageSize,
    required bool hasNext,
  }) = _AgentTopicListResult;

  factory AgentTopicListResult.fromJson(Map<String, dynamic> json) =>
      _$AgentTopicListResultFromJson(json);
}

@freezed
abstract class AgentWriteTopicRequest with _$AgentWriteTopicRequest {
  const factory AgentWriteTopicRequest({
    required String title,
    required String content,
    required List<int> categoryId,
  }) = _AgentWriteTopicRequest;

  factory AgentWriteTopicRequest.fromJson(Map<String, dynamic> json) =>
      _$AgentWriteTopicRequestFromJson(json);
}

@freezed
abstract class AgentCreatePostRequest with _$AgentCreatePostRequest {
  const factory AgentCreatePostRequest({
    required String content,
    int? replyToPostId,
  }) = _AgentCreatePostRequest;

  factory AgentCreatePostRequest.fromJson(Map<String, dynamic> json) =>
      _$AgentCreatePostRequestFromJson(json);
}

@freezed
abstract class AgentCreatePostResult with _$AgentCreatePostResult {
  const factory AgentCreatePostResult({
    required int id,
    required int postNo,
    required String renderedContent,
  }) = _AgentCreatePostResult;

  factory AgentCreatePostResult.fromJson(Map<String, dynamic> json) =>
      _$AgentCreatePostResultFromJson(json);
}

typedef AgentPostListResult = PostWindowPayload;
typedef AgentSearchResult = SearchPageProps;

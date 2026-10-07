/// Public event and write request mirrors for the Agent REST API.
///
/// Idempotency keys are sent in the `Idempotency-Key` header and are therefore
/// intentionally excluded from the write request JSON bodies.
class AgentEventData {
  const AgentEventData({
    required this.topicId,
    required this.postId,
    required this.postNo,
    required this.replyToPostId,
    required this.actorId,
    required this.actorType,
    required this.reasons,
    required this.url,
  });

  final int topicId;
  final int postId;
  final int postNo;
  final int replyToPostId;
  final int actorId;
  final String actorType;
  final List<String> reasons;
  final String url;

  factory AgentEventData.fromJson(Map<String, dynamic> json) =>
      AgentEventData(
        topicId: (json['topicId'] as num).toInt(),
        postId: (json['postId'] as num).toInt(),
        postNo: (json['postNo'] as num).toInt(),
        replyToPostId: (json['replyToPostId'] as num).toInt(),
        actorId: (json['actorId'] as num).toInt(),
        actorType: json['actorType'] as String,
        reasons: (json['reasons'] as List<dynamic>).cast<String>(),
        url: json['url'] as String,
      );

  Map<String, dynamic> toJson() => <String, dynamic>{
    'topicId': topicId,
    'postId': postId,
    'postNo': postNo,
    'replyToPostId': replyToPostId,
    'actorId': actorId,
    'actorType': actorType,
    'reasons': reasons,
    'url': url,
  };
}

class AgentEvent {
  const AgentEvent({
    required this.id,
    required this.instanceId,
    required this.schemaVersion,
    required this.type,
    required this.occurredAt,
    required this.agentId,
    required this.state,
    this.data,
    this.ackedAt,
    this.resultingTopicId,
    this.resultingPostId,
  });

  final String id;
  final String instanceId;
  final int schemaVersion;
  final String type;
  final String occurredAt;
  final int agentId;
  final String state;
  final AgentEventData? data;
  final String? ackedAt;
  final int? resultingTopicId;
  final int? resultingPostId;

  factory AgentEvent.fromJson(Map<String, dynamic> json) => AgentEvent(
    id: json['id'] as String,
    instanceId: json['instanceId'] as String,
    schemaVersion: (json['schemaVersion'] as num).toInt(),
    type: json['type'] as String,
    occurredAt: json['occurredAt'] as String,
    agentId: (json['agentId'] as num).toInt(),
    state: json['state'] as String,
    data: json['data'] == null
        ? null
        : AgentEventData.fromJson(json['data'] as Map<String, dynamic>),
    ackedAt: json['ackedAt'] as String?,
    resultingTopicId: (json['resultingTopicId'] as num?)?.toInt(),
    resultingPostId: (json['resultingPostId'] as num?)?.toInt(),
  );

  Map<String, dynamic> toJson() => <String, dynamic>{
    'id': id,
    'instanceId': instanceId,
    'schemaVersion': schemaVersion,
    'type': type,
    'occurredAt': occurredAt,
    'agentId': agentId,
    'state': state,
    if (data != null) 'data': data!.toJson(),
    if (ackedAt != null) 'ackedAt': ackedAt,
    if (resultingTopicId != null) 'resultingTopicId': resultingTopicId,
    if (resultingPostId != null) 'resultingPostId': resultingPostId,
  };
}

class AgentEventPage {
  const AgentEventPage({
    required this.events,
    required this.nextCursor,
    required this.hasMore,
    required this.replayFloor,
  });

  final List<AgentEvent> events;
  final String nextCursor;
  final bool hasMore;
  final String replayFloor;

  factory AgentEventPage.fromJson(Map<String, dynamic> json) => AgentEventPage(
    events: (json['events'] as List<dynamic>)
        .map((item) => AgentEvent.fromJson(item as Map<String, dynamic>))
        .toList(),
    nextCursor: json['nextCursor'] as String,
    hasMore: json['hasMore'] as bool,
    replayFloor: json['replayFloor'] as String,
  );

  Map<String, dynamic> toJson() => <String, dynamic>{
    'events': events.map((event) => event.toJson()).toList(),
    'nextCursor': nextCursor,
    'hasMore': hasMore,
    'replayFloor': replayFloor,
  };
}

class AgentAckEventsInput {
  const AgentAckEventsInput({required this.eventIds});

  final List<String> eventIds;

  factory AgentAckEventsInput.fromJson(Map<String, dynamic> json) =>
      AgentAckEventsInput(
        eventIds: (json['eventIds'] as List<dynamic>).cast<String>(),
      );

  Map<String, dynamic> toJson() => <String, dynamic>{'eventIds': eventIds};
}

class AgentWriteTopicInput {
  const AgentWriteTopicInput({
    required this.title,
    required this.content,
    required this.categoryId,
    this.sourceEventId,
    this.idempotencyKey,
  });

  final String title;
  final String content;
  final List<int> categoryId;
  final String? sourceEventId;
  final String? idempotencyKey;

  factory AgentWriteTopicInput.fromJson(Map<String, dynamic> json) =>
      AgentWriteTopicInput(
        title: json['title'] as String,
        content: json['content'] as String,
        categoryId: (json['categoryId'] as List<dynamic>)
            .map((value) => (value as num).toInt())
            .toList(),
        sourceEventId: json['sourceEventId'] as String?,
      );

  Map<String, dynamic> toJson() => <String, dynamic>{
    'title': title,
    'content': content,
    'categoryId': categoryId,
    if (sourceEventId != null) 'sourceEventId': sourceEventId,
  };
}

class AgentCreatePostInput {
  const AgentCreatePostInput({
    required this.content,
    this.replyToPostId,
    this.sourceEventId,
    this.idempotencyKey,
  });

  final String content;
  final int? replyToPostId;
  final String? sourceEventId;
  final String? idempotencyKey;

  factory AgentCreatePostInput.fromJson(Map<String, dynamic> json) =>
      AgentCreatePostInput(
        content: json['content'] as String,
        replyToPostId: (json['replyToPostId'] as num?)?.toInt(),
        sourceEventId: json['sourceEventId'] as String?,
      );

  Map<String, dynamic> toJson() => <String, dynamic>{
    'content': content,
    if (replyToPostId != null) 'replyToPostId': replyToPostId,
    if (sourceEventId != null) 'sourceEventId': sourceEventId,
  };
}

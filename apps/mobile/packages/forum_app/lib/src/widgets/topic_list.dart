import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:core/core.dart';

import '../../l10n/app_localizations.dart';
import '../format.dart';
import 'status_views.dart';
import '../asset_url.dart';

enum GfTopicFeedMode { list, card }

/// 话题列表(无限分页),行复用 ui_kit [GfTopicRow]。
class GfTopicList extends StatelessWidget {
  const GfTopicList({
    super.key,
    required this.loading,
    required this.topics,
    this.controller,
    this.padding = EdgeInsets.zero,
    this.header,
    this.feedMode = GfTopicFeedMode.list,
    this.onLikeTopic,
    this.onBookmarkTopic,
    this.onReturnFromTopic,
    required this.hasMore,
    required this.onLoadMore,
  });

  final bool loading;
  final EdgeInsets padding;
  final Widget? header;
  final ScrollController? controller;
  final List<TopicPayload> topics;
  final GfTopicFeedMode feedMode;
  final Future<bool> Function(TopicPayload topic, bool target)? onLikeTopic;
  final Future<bool> Function(TopicPayload topic, bool target)? onBookmarkTopic;
  final VoidCallback? onReturnFromTopic;
  final bool hasMore;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    if (loading && topics.isEmpty) return const GfLoading();
    if (topics.isEmpty) {
      return CustomScrollView(
        controller: controller,
        physics: const AlwaysScrollableScrollPhysics(),
        slivers: [
          SliverPadding(
            padding: EdgeInsets.only(top: padding.top),
            sliver: SliverToBoxAdapter(
              child: header ?? const SizedBox.shrink(),
            ),
          ),
          SliverFillRemaining(
            hasScrollBody: false,
            child: Padding(
              padding: EdgeInsets.fromLTRB(
                padding.left,
                0,
                padding.right,
                padding.bottom,
              ),
              child: GfEmpty(message: AppLocalizations.of(context).topicEmpty),
            ),
          ),
        ],
      );
    }
    return ListView.separated(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: padding,
      itemCount: topics.length + 1 + (header == null ? 0 : 1),
      separatorBuilder: (_, _) => const SizedBox.shrink(),
      itemBuilder: (context, index) {
        if (header != null) {
          if (index == 0) return header!;
          index -= 1;
        }
        if (index == topics.length) {
          return GfListFooter(
            loading: loading,
            hasMore: hasMore,
            onLoadMore: onLoadMore,
          );
        }
        final TopicPayload topic = topics[index];
        return feedMode == GfTopicFeedMode.card
            ? _topicCard(
                context,
                topic,
                onReturn: onReturnFromTopic,
                onLike: onLikeTopic == null || topic.liked == null
                    ? null
                    : (target) => onLikeTopic!(topic, target),
                onBookmark: onBookmarkTopic == null || topic.bookmarked == null
                    ? null
                    : (target) => onBookmarkTopic!(topic, target),
              )
            : _topicRow(
                context,
                topic,
                isLast: index == topics.length - 1,
                onReturn: onReturnFromTopic,
              );
      },
    );
  }
}

/// 把后端话题 payload 映射为 [GfTopicRow](对齐 web TopicRow.vue 语义)。
Widget _topicRow(
  BuildContext context,
  TopicPayload topic, {
  required bool isLast,
  VoidCallback? onReturn,
}) {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final List<GfTopicCategory> categories = <GfTopicCategory>[
    for (final CategoryBriefPayload cat in topic.categories)
      GfTopicCategory(name: cat.name, color: colorFromHex(cat.color)),
  ];

  final List<String> participantAvatarUrls = <String>[
    for (final UserBriefPayload participant in topic.participants)
      resolveApiAssetUrl(participant.avatarUrl),
  ];

  return GfTopicRow(
    title: topic.title,
    description: topic.description,
    categories: categories,
    participantAvatarUrls: participantAvatarUrls,
    activityText: timeAgo(
      topic.activityText.isNotEmpty ? topic.activityText : topic.lastUpdateTime,
      l10n: l10n,
    ),
    replyCount: topic.replyCount,
    viewCount: topic.viewCount,
    hot: topic.viewCount > 500,
    pinned: topic.pinWeight > 0,
    unseen: topic.unseen == true,
    showDivider: !isLast,
    onTap: () async {
      await context.push('/p/${topic.id}');
      onReturn?.call();
    },
  );
}

Widget _topicCard(
  BuildContext context,
  TopicPayload topic, {
  VoidCallback? onReturn,
  Future<bool> Function(bool target)? onLike,
  Future<bool> Function(bool target)? onBookmark,
}) {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final String nickname = topic.author.nickname?.trim() ?? '';
  final List<String> images = <String>[
    for (final String image in topic.images ?? const <String>[])
      resolveApiAssetUrl(image),
  ];
  if (images.isEmpty && (topic.firstImageUrl?.isNotEmpty ?? false)) {
    images.add(resolveApiAssetUrl(topic.firstImageUrl!));
  }

  return GfTopicCard(
    key: ValueKey<int>(topic.id),
    title: topic.title,
    description: topic.description,
    authorName: nickname.isNotEmpty ? nickname : topic.author.username,
    authorAvatarUrl: resolveApiAssetUrl(topic.author.avatarUrl),
    categories: <GfTopicCategory>[
      for (final CategoryBriefPayload category in topic.categories)
        GfTopicCategory(
          name: category.name,
          color: colorFromHex(category.color),
        ),
    ],
    imageUrls: images,
    activityText: timeAgo(
      topic.activityText.isNotEmpty ? topic.activityText : topic.lastUpdateTime,
      l10n: l10n,
    ),
    replyCount: topic.replyCount,
    viewCount: topic.viewCount,
    likeCount: topic.likeCount,
    liked: topic.liked ?? false,
    bookmarked: topic.bookmarked ?? false,
    onLike: onLike,
    onBookmark: onBookmark,
    likeTooltip: l10n.topicLike,
    bookmarkTooltip: l10n.topicBookmark,
    bookmarkedTooltip: l10n.topicBookmarked,
    hot: topic.viewCount > 500,
    pinned: topic.pinWeight > 0,
    unseen: topic.unseen == true,
    onTap: () async {
      await context.push('/p/${topic.id}');
      onReturn?.call();
    },
  );
}

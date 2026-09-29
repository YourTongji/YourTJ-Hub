import '../private_notes.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:core/core.dart';

import '../../l10n/app_localizations.dart';
import '../format.dart';
import 'status_views.dart';
import '../asset_url.dart';
import '../images/image_save.dart';

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
    this.collapsePinned = false,
    this.onLikeTopic,
    this.onBookmarkTopic,
    this.onFirstMediaFrame,
    this.onReturnFromTopic,
    required this.hasMore,
    required this.onLoadMore,
    this.loadMoreError,
    this.hiddenCategoryId,
    this.onCategorySelected,
  });

  final bool loading;
  final EdgeInsets padding;
  final Widget? header;
  final ScrollController? controller;
  final List<TopicPayload> topics;
  final GfTopicFeedMode feedMode;

  /// Home list mode may group pins; ordered streams (Following) keep server order.
  final bool collapsePinned;
  final Future<bool> Function(TopicPayload topic, bool target)? onLikeTopic;
  final Future<bool> Function(TopicPayload topic, bool target)? onBookmarkTopic;
  final VoidCallback? onFirstMediaFrame;
  final VoidCallback? onReturnFromTopic;
  final bool hasMore;
  final VoidCallback onLoadMore;
  final String? loadMoreError;
  final int? hiddenCategoryId;
  final ValueChanged<int>? onCategorySelected;

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
    final pinned = collapsePinned && feedMode == GfTopicFeedMode.list
        ? topics.where((topic) => topic.pinWeight > 0).toList()
        : <TopicPayload>[];
    final visibleTopics = pinned.isEmpty
        ? topics
        : topics.where((topic) => topic.pinWeight <= 0).toList();
    return ListView.separated(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: padding,
      itemCount:
          visibleTopics.length +
          1 +
          (header == null ? 0 : 1) +
          (pinned.isEmpty ? 0 : 1),
      separatorBuilder: (_, _) => const SizedBox.shrink(),
      itemBuilder: (context, index) {
        if (header != null) {
          if (index == 0) return header!;
          index -= 1;
        }
        if (pinned.isNotEmpty) {
          if (index == 0) {
            return _PinnedTopicGroup(
              key: const ValueKey('pinned-topic-group'),
              topics: pinned,
              onCategorySelected: onCategorySelected,
              hiddenCategoryId: hiddenCategoryId,
              onReturn: onReturnFromTopic,
            );
          }
          index -= 1;
        }
        if (index == visibleTopics.length) {
          return GfListFooter(
            progressKey: topics.length,
            loading: loading,
            error: loadMoreError,
            hasMore: hasMore,
            onLoadMore: onLoadMore,
          );
        }
        final TopicPayload topic = visibleTopics[index];
        return feedMode == GfTopicFeedMode.card
            ? buildTopicFeedCard(
                context,
                topic,
                hiddenCategoryId: hiddenCategoryId,
                onCategorySelected: onCategorySelected,
                onReturn: onReturnFromTopic,
                onFirstMediaFrame: onFirstMediaFrame,
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
                isLast: index == visibleTopics.length - 1,
                onCategorySelected: onCategorySelected,
                hiddenCategoryId: hiddenCategoryId,
                onReturn: onReturnFromTopic,
              );
      },
    );
  }
}

class _PinnedTopicGroup extends StatefulWidget {
  const _PinnedTopicGroup({
    super.key,
    required this.topics,
    this.hiddenCategoryId,
    this.onCategorySelected,
    this.onReturn,
  });
  final List<TopicPayload> topics;
  final int? hiddenCategoryId;
  final ValueChanged<int>? onCategorySelected;
  final VoidCallback? onReturn;

  @override
  State<_PinnedTopicGroup> createState() => _PinnedTopicGroupState();
}

class _PinnedTopicGroupState extends State<_PinnedTopicGroup>
    with AutomaticKeepAliveClientMixin {
  bool _expanded = false;

  // Retain the disclosure when its lazy list item leaves the viewport.
  @override
  bool get wantKeepAlive => true;

  @override
  Widget build(BuildContext context) {
    super.build(context);
    final colors = GfTheme.colorsOf(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Semantics(
          button: true,
          expanded: _expanded,
          child: InkWell(
            onTap: () => setState(() => _expanded = !_expanded),
            child: ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 48),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 8,
                ),
                child: Row(
                  children: [
                    GfSymbol('pin-filled', size: 16, color: colors.error),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        AppLocalizations.of(
                          context,
                        ).homePinnedTopics(widget.topics.length),
                        style: TextStyle(
                          fontSize: 14,
                          color: colors.baseContent,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    GfSymbol(
                      _expanded ? 'chevron-up' : 'chevron-down',
                      size: 18,
                      color: colors.baseContent.withValues(alpha: 0.6),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
        if (_expanded)
          for (final topic in widget.topics)
            _topicRow(
              context,
              topic,
              isLast: false,
              onCategorySelected: widget.onCategorySelected,
              hiddenCategoryId: widget.hiddenCategoryId,
              onReturn: widget.onReturn,
            ),
      ],
    );
  }
}

/// 把后端话题 payload 映射为 [GfTopicRow](对齐 web TopicRow.vue 语义)。
Widget _topicRow(
  BuildContext context,
  TopicPayload topic, {
  required bool isLast,
  int? hiddenCategoryId,
  ValueChanged<int>? onCategorySelected,
  VoidCallback? onReturn,
}) {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final List<GfTopicCategory> categories = <GfTopicCategory>[
    for (final CategoryBriefPayload cat in topic.categories)
      if (cat.id != hiddenCategoryId)
        GfTopicCategory(
          name: cat.name,
          color: colorFromHex(cat.color),
          onTap: onCategorySelected == null
              ? null
              : () => onCategorySelected(cat.id),
        ),
  ];

  final List<String> participantAvatarUrls = <String>[
    for (final UserBriefPayload participant in topic.participants)
      resolveApiAssetUrl(participant.avatarUrl),
  ];

  // 无标题瞬间与 Web TopicRow/SSR 一致：标题为空时摘要在标题位展示（不再重复一行），
  // 摘要也为空时使用同族颜文字，保证列表行始终有可识别文案。
  final String rowTitle = topic.title.isNotEmpty
      ? topic.title
      : (topic.description.trim().isNotEmpty ? topic.description : '(｀・ω・´)');
  return GfTopicRow(
    title: rowTitle,
    description: topic.title.isEmpty ? '' : topic.description,
    categories: categories,
    participantAvatarUrls: participantAvatarUrls,
    activityText: timeAgo(
      topic.activityText.isNotEmpty ? topic.activityText : topic.lastUpdateTime,
      l10n: l10n,
    ),
    replyCount: topic.replyCount,
    viewCount: topic.viewCount,
    pinned: topic.pinWeight > 0,
    pinnedLabel: l10n.topicPinned,
    contentType: switch (topic.contentType) {
      1 => GfTopicContentType.question,
      2 => GfTopicContentType.moment,
      3 => GfTopicContentType.article,
      _ => null,
    },
    contentTypeLabel: switch (topic.contentType) {
      1 => l10n.publishQuestion,
      2 => l10n.publishMoment,
      3 => l10n.publishArticle,
      _ => null,
    },
    unseen: topic.unseen == true,
    showDivider: !isLast,
    onTap: () async {
      await context.push('/p/${topic.id}');
      onReturn?.call();
    },
  );
}

/// Shared author-led topic card for Home and profile topic streams.
Widget buildTopicFeedCard(
  BuildContext context,
  TopicPayload topic, {
  VoidCallback? onReturn,
  VoidCallback? onFirstMediaFrame,
  Future<bool> Function(bool target)? onLike,
  Future<bool> Function(bool target)? onBookmark,
  int? hiddenCategoryId,
  ValueChanged<int>? onCategorySelected,
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
    authorName: privateDisplayName(
      context,
      topic.author.id,
      topic.author.username,
      nickname,
    ),
    authorAvatarUrl: resolveApiAssetUrl(topic.author.avatarUrl),
    onAuthorTap: topic.author.id > 0
        ? () => context.push('/u/${topic.author.id}')
        : null,
    imageSemanticLabelBuilder: l10n.imageViewPosition,
    onSaveImage: (url) => saveImageFromUrl(context, url),
    saveImageLabel: l10n.imageSave,
    onShareImage: (url) => shareImageFromUrl(context, url),
    shareImageLabel: l10n.topicShare,
    categories: <GfTopicCategory>[
      for (final CategoryBriefPayload category in topic.categories)
        if (category.id != hiddenCategoryId)
          GfTopicCategory(
            name: category.name,
            color: colorFromHex(category.color),
            onTap: onCategorySelected == null
                ? null
                : () => onCategorySelected(category.id),
          ),
    ],
    imageUrls: images,
    onFirstMediaFrame: onFirstMediaFrame,
    imageMetadata: <GfTopicImageMetadata>[
      for (final TopicImageMetadataPayload metadata
          in topic.imageMetadata ?? const <TopicImageMetadataPayload>[])
        GfTopicImageMetadata(
          url: resolveApiAssetUrl(metadata.url),
          width: metadata.width,
          height: metadata.height,
          variants: <GfTopicImageVariant>[
            for (final TopicImageVariantPayload variant in metadata.variants)
              GfTopicImageVariant(
                url: resolveApiAssetUrl(variant.url),
                width: variant.width,
                height: variant.height,
              ),
          ],
        ),
    ],
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
    pinned: topic.pinWeight > 0,
    unseen: topic.unseen == true,
    onTap: () async {
      await context.push('/p/${topic.id}');
      onReturn?.call();
    },
  );
}

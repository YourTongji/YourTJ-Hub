import 'user_decoration.dart';
import 'feed_visibility.dart';
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
import 'sticker_message_span.dart';

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
    this.stickerLibrary,
  });

  final bool loading;
  final EdgeInsets padding;
  final Widget? header;
  final ScrollController? controller;
  final List<TopicPayload> topics;
  final GfTopicFeedMode feedMode;

  /// Home may group pins (a summary in list mode, a title digest in card
  /// mode); ordered streams (Following) keep server order.
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

  /// Page-level sticker resolution for feed excerpts. Rows only receive the
  /// resolved name→url map, so scrolling never triggers a lookup per item;
  /// null keeps excerpts plain.
  final StickerLibrary? stickerLibrary;

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
    final pinned = collapsePinned
        ? topics.where((topic) => topic.pinWeight > 0).toList()
        : <TopicPayload>[];
    Widget card(TopicPayload topic, Map<String, String> stickerUrlByName) =>
        buildTopicFeedCard(
          context,
          topic,
          stickerUrlByName: stickerUrlByName,
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
        );
    final visibleTopics = pinned.isEmpty
        ? topics
        : topics.where((topic) => topic.pinWeight <= 0).toList();
    Widget feed(Map<String, String> stickerUrlByName) => ListView.separated(
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
            return _PinnedTopicStrip(
              key: const ValueKey('pinned-topic-strip'),
              stickerUrlByName: stickerUrlByName,
              topics: pinned,
              // Under a header (the announcement strip) keep one 6-pixel gap.
              topMargin: header == null ? 6 : 0,
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
            ? card(topic, stickerUrlByName)
            : _topicRow(
                context,
                topic,
                stickerUrlByName: stickerUrlByName,
                isLast: index == visibleTopics.length - 1,
                onCategorySelected: onCategorySelected,
                hiddenCategoryId: hiddenCategoryId,
                onReturn: onReturnFromTopic,
              );
      },
    );
    final StickerLibrary? library = stickerLibrary;
    return library == null
        ? feed(const <String, String>{})
        : TopicStickerExcerpts(
            library: library,
            topics: topics,
            builder: feed,
          );
  }
}

/// 卡片摘要（17px 文本）里的贴纸图片尺寸；列表行是小字视图，改用 [名字] 文本。
const double _cardStickerSize = 22;

/// One resolution for the whole page: the visible topics' descriptions are
/// sent together, rows only read the resolved map. Unknown or disabled tokens
/// vanish from the excerpt instead of surfacing as markup.
class TopicStickerExcerpts extends StatefulWidget {
  const TopicStickerExcerpts({
    super.key,
    required this.library,
    required this.topics,
    required this.builder,
  });

  final StickerLibrary library;
  final List<TopicPayload> topics;
  final Widget Function(Map<String, String> urlByName) builder;

  @override
  State<TopicStickerExcerpts> createState() => TopicStickerExcerptsState();
}

class TopicStickerExcerptsState extends State<TopicStickerExcerpts> {
  String? _requested;

  @override
  void initState() {
    super.initState();
    widget.library.addListener(_catalogChanged);
    _resolve();
  }

  @override
  void didUpdateWidget(TopicStickerExcerpts oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.library != widget.library) {
      oldWidget.library.removeListener(_catalogChanged);
      widget.library.addListener(_catalogChanged);
    }
    _resolve();
  }

  @override
  void dispose() {
    widget.library.removeListener(_catalogChanged);
    super.dispose();
  }

  void _catalogChanged() => setState(() {});

  void _resolve() {
    final String sources = [
      for (final TopicPayload topic in widget.topics)
        '${topic.title}\n${topic.description}',
    ].join('\n');
    if (sources == _requested) return;
    _requested = sources;
    // A failed lookup leaves the excerpt readable without its stickers.
    widget.library.resolveContent(sources).catchError((Object _) {});
  }

  @override
  Widget build(BuildContext context) =>
      widget.builder(widget.library.urlByName);
}

/// Feed excerpt text plus its resolved span. Tokens always travel through the
/// span, so resolved ones render as images and unresolved ones disappear.
(String, InlineSpan?) _feedExcerpt(
  String text,
  Map<String, String> urlByName,
  double stickerSize,
) {
  if (!containsStickerToken(text)) return (text, null);
  return (
    '',
    buildStickerMessageSpan(
      text,
      urlByName,
      stickerSize: stickerSize,
      collectible: false,
      deferLongPress: true,
      dropUnknownStickers: true,
    ),
  );
}

/// Collapsed pins, in card and list feeds alike, fold into one line beside
/// the announcement bar: pin, "Pinned" badge and the first pinned title. A
/// single pin opens directly; several unfold into avatar-led title rows
/// inside the same strip.
class _PinnedTopicStrip extends StatefulWidget {
  const _PinnedTopicStrip({
    super.key,
    required this.topics,
    required this.topMargin,
    this.stickerUrlByName = const <String, String>{},
    this.onReturn,
  });

  final List<TopicPayload> topics;
  final Map<String, String> stickerUrlByName;
  final double topMargin;
  final VoidCallback? onReturn;

  @override
  State<_PinnedTopicStrip> createState() => _PinnedTopicStripState();
}

class _PinnedTopicStripState extends State<_PinnedTopicStrip>
    with AutomaticKeepAliveClientMixin {
  bool _expanded = false;

  // Retain the disclosure when its lazy list item leaves the viewport.
  @override
  bool get wantKeepAlive => true;

  // 无标题瞬间的置顶标题其实是摘要，可能带 [:sticker:name:]；与列表行同样走共享分段。
  Widget pinnedTitle(BuildContext context, TopicPayload topic, GfColors colors,
      {double fontSize = 12.5}) {
    final TextStyle style = TextStyle(
      fontSize: fontSize,
      fontWeight: FontWeight.w500,
      color: colors.baseContent.withValues(alpha: 0.88),
    );
    final String title = stickerLabelText(
      topicDisplayTitle(topic),
      widget.stickerUrlByName,
    );
    return GfText(title, maxLines: 1, overflow: TextOverflow.ellipsis, style: style);
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final Color muted = colors.baseContent.withValues(alpha: 0.45);
    final List<TopicPayload> topics = widget.topics;
    final bool single = topics.length == 1;
    final bool expanded = _expanded && !single;
    // Large text keeps the line for the title; the pin glyph still names it.
    final TextScaler scaler = MediaQuery.textScalerOf(context);
    final bool showTime = scaler.scale(12) <= 16;
    final bool showBadge = scaler.scale(10) <= 15;

    final Widget lead = Row(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        Container(
          width: 22,
          height: 22,
          decoration: BoxDecoration(
            color: colors.error.withValues(alpha: 0.12),
            borderRadius: BorderRadius.circular(6),
          ),
          alignment: Alignment.center,
          child: Semantics(
            label: showBadge ? null : l10n.topicPinned,
            child: GfSymbol('pin-filled', size: 12, color: colors.error),
          ),
        ),
        if (showBadge) ...<Widget>[
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
            decoration: BoxDecoration(
              color: colors.error.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(4),
            ),
            child: GfText(
              l10n.topicPinned,
              style: TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.w700,
                color: colors.error,
                letterSpacing: 0.2,
              ),
            ),
          ),
        ],
      ],
    );

    final Widget line = Semantics(
      button: true,
      expanded: single ? null : expanded,
      label: single ? null : l10n.homePinnedTopics(topics.length),
      child: InkWell(
        key: const ValueKey('pinned-strip-line'),
        onTap: single
            ? () => _openTopic(context, topics.first, widget.onReturn)
            : () => setState(() => _expanded = !_expanded),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
          child: Row(
            children: <Widget>[
              lead,
              const SizedBox(width: 8),
              Expanded(
                child: expanded
                    ? ExcludeSemantics(
                        child: GfText(
                          l10n.announcementCollapseAction,
                          textAlign: TextAlign.end,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(fontSize: 11, color: muted),
                        ),
                      )
                    : pinnedTitle(context, topics.first, colors, fontSize: 12),
              ),
              if (!single) ...<Widget>[
                if (!expanded) ...<Widget>[
                  const SizedBox(width: 6),
                  ExcludeSemantics(
                    child: GfText(
                      '${topics.length}',
                      style: TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.w600,
                        color: muted,
                      ),
                    ),
                  ),
                ],
                const SizedBox(width: 2),
                AnimatedRotation(
                  turns: expanded ? .5 : 0,
                  duration: GfMotion.duration(context, GfMotion.selection),
                  curve: GfMotion.layoutCurve,
                  child: GfSymbol('chevron-down', size: 13, color: muted),
                ),
              ],
            ],
          ),
        ),
      ),
    );

    Widget titleRow(TopicPayload topic) => Semantics(
      button: true,
      child: InkWell(
        key: ValueKey<String>('pinned-strip-${topic.id}'),
        onTap: () => _openTopic(context, topic, widget.onReturn),
        child: Padding(
          // The author's avatar sits under the pin tile; titles start under
          // the badge.
          padding: const EdgeInsetsDirectional.fromSTEB(11, 8, 12, 8),
          child: Row(
            children: <Widget>[
              ExcludeSemantics(
                child: GfAvatar(
                  frameColors: topic.author.publicUid != null
                      ? const <Color>[]
                      : decorationColors(
                          context,
                          topic.author.decoration?.avatarFrame,
                        ),
                  frameAssetUrl: topic.author.publicUid != null
                      ? ''
                      : decorationFrameAssetUrl(
                          topic.author.decoration?.avatarFrame,
                        ),
                  frameScale:
                      topic.author.decoration?.avatarFrame?.assetScale ?? 1.25,
                  src: resolveApiAssetUrl(topic.author.avatarUrl),
                  size: 20,
                ),
              ),
              const SizedBox(width: 9),
              Expanded(child: pinnedTitle(context, topic, colors)),
              if (topic.unseen == true) ...<Widget>[
                const SizedBox(width: 6),
                Container(
                  width: 5,
                  height: 5,
                  decoration: BoxDecoration(
                    color: colors.primary,
                    shape: BoxShape.circle,
                  ),
                ),
              ],
              if (showTime) ...<Widget>[
                const SizedBox(width: 10),
                GfText(
                  timeAgo(
                    topic.activityText.isNotEmpty
                        ? topic.activityText
                        : topic.lastUpdateTime,
                    l10n: l10n,
                  ),
                  style: TextStyle(fontSize: 11, color: muted),
                ),
              ],
            ],
          ),
        ),
      ),
    );

    return Container(
      margin: EdgeInsets.fromLTRB(16, widget.topMargin, 16, 6),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colors.line),
      ),
      child: Material(
        type: MaterialType.transparency,
        child: AnimatedSize(
          duration: GfMotion.duration(context, GfMotion.layout),
          curve: GfMotion.layoutCurve,
          alignment: Alignment.topCenter,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              line,
              if (expanded) ...<Widget>[
                for (final TopicPayload topic in topics) titleRow(topic),
                const SizedBox(height: 4),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

Future<void> _openTopic(
  BuildContext context,
  TopicPayload topic,
  VoidCallback? onReturn,
) async {
  FeedTelemetry.instance.select(topic);
  await context.push('/p/${topic.id}');
  FeedTelemetry.instance.endDetail();
  onReturn?.call();
}

/// 无标题瞬间与 Web TopicRow/SSR 一致：标题为空时摘要在标题位展示（不再重复一行），
/// 摘要也为空时使用同族颜文字，保证列表行始终有可识别文案。
String topicDisplayTitle(TopicPayload topic) => topic.title.isNotEmpty
    ? topic.title
    : (topic.description.trim().isNotEmpty ? topic.description : '(｀・ω・´)');

/// 把后端话题 payload 映射为 [GfTopicRow](对齐 web TopicRow.vue 语义)。
Widget _topicRow(
  BuildContext context,
  TopicPayload topic, {
  required bool isLast,
  required Map<String, String> stickerUrlByName,
  int? hiddenCategoryId,
  ValueChanged<int>? onCategorySelected,
  VoidCallback? onReturn,
}) {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final String displayTitle = topic.processStatus == 2
      ? "${l10n.contentReviewPending} · ${topicDisplayTitle(topic)}"
      : topicDisplayTitle(topic);
  // 列表行是小字视图：贴纸显示 [名字] 纯文本（卡片视图才渲染图片）。
  // 无标题瞬间把摘要放在标题位，因此两个槽位都走同一规则。
  final String title = stickerLabelText(displayTitle, stickerUrlByName);
  final String description = topic.title.isEmpty
      ? ''
      : stickerLabelText(topic.description, stickerUrlByName);
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

  return FeedVisibility(
    topic: topic,
    child: GfTopicRow(
      feedContext: _feedContext(l10n, topic),
      title: title,
      description: description,
      categories: categories,
      participantAvatarUrls: participantAvatarUrls,
      participantFrameScales: [
        for (final participant in topic.participants)
          participant.decoration?.avatarFrame?.assetScale ?? 1.25,
      ],
      participantFrameAssetUrls: [
        for (final participant in topic.participants)
          decorationFrameAssetUrl(participant.decoration?.avatarFrame),
      ],
      participantFrameColors: [
        for (final participant in topic.participants)
          decorationColors(context, participant.decoration?.avatarFrame),
      ],
      activityText: timeAgo(
        topic.activityText.isNotEmpty
            ? topic.activityText
            : topic.lastUpdateTime,
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
        FeedTelemetry.instance.select(topic);
        await context.push('/p/${topic.id}');
        FeedTelemetry.instance.endDetail();
        onReturn?.call();
      },
    ),
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
  Map<String, String> stickerUrlByName = const <String, String>{},
}) {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final String nickname = topic.author.nickname?.trim() ?? '';
  final (String description, InlineSpan? descriptionSpan) = _feedExcerpt(
    topic.description,
    stickerUrlByName,
    _cardStickerSize,
  );
  final List<String> images = <String>[
    for (final String image in topic.images ?? const <String>[])
      resolveApiAssetUrl(image),
  ];
  if (images.isEmpty && (topic.firstImageUrl?.isNotEmpty ?? false)) {
    images.add(resolveApiAssetUrl(topic.firstImageUrl!));
  }

  return FeedVisibility(
    topic: topic,
    child: GfTopicCard(
      feedContext: _feedContext(l10n, topic),
      avatarFrameColors: decorationColors(
        context,
        topic.author.decoration?.avatarFrame,
      ),
      avatarFrameAssetUrl: decorationFrameAssetUrl(
        topic.author.decoration?.avatarFrame,
      ),
      avatarFrameScale:
          topic.author.decoration?.avatarFrame?.assetScale ?? 1.25,
      nicknameColors: decorationColors(
        context,
        topic.author.decoration?.nicknameStyle,
      ),
      key: ValueKey<int>(topic.id),
      title: topic.processStatus == 2
          ? "${l10n.contentReviewPending} · ${topic.title}"
          : topic.title,
      description: description,
      descriptionSpan: descriptionSpan,
      authorName: topic.author.publicUid != null
          ? '${topic.author.nickname ?? topic.author.username} · ${l10n.anonymousTag}'
          : privateDisplayName(
              context,
              topic.author.id,
              topic.author.username,
              nickname,
            ),
      authorAvatarUrl: resolveApiAssetUrl(topic.author.avatarUrl),
      onAuthorTap: topic.author.publicUid != null
          ? () => context.push('/a/${topic.author.publicUid}')
          : topic.author.id > 0
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
        topic.activityText.isNotEmpty
            ? topic.activityText
            : topic.lastUpdateTime,
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
        FeedTelemetry.instance.select(topic);
        await context.push('/p/${topic.id}');
        FeedTelemetry.instance.endDetail();
        onReturn?.call();
      },
    ),
  );
}

/// For You reasons borrow the mark of the element that justifies them. Every
/// reason the server sends is labelled, `recent` fallback included, so no For
/// You card goes without its seal.
GfTopicContext? _feedContext(AppLocalizations l10n, TopicPayload topic) =>
    switch (topic.feedReason) {
      'following' => GfTopicContext(
        label: l10n.feedReasonFollowing,
        symbol: 'user-round-check',
      ),
      'category' => GfTopicContext(
        label: l10n.feedReasonCategory,
        // Search uses the same glyph for its categories scope.
        symbol: 'folder',
      ),
      'newreply' => GfTopicContext(
        label: l10n.feedReasonNewReply,
        symbol: 'message-circle',
        newReplies: true,
      ),
      'recent' => GfTopicContext(
        label: l10n.feedReasonRecent,
        // Profiles and wiki pages use the same glyph for time.
        symbol: 'clock',
      ),
      _ => null,
    };

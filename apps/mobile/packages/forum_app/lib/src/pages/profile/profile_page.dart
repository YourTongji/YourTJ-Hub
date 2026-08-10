import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../current_user.dart';
import '../../format.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';
import '../../widgets/skeletons.dart';

Color _userBadgeColor(UserBadgePayload badge) {
  const Map<String, Color> colors = <String, Color>{
    'blue': Color(0xFF1D4ED8),
    'emerald': Color(0xFF047857),
    'teal': Color(0xFF0F766E),
    'sky': Color(0xFF0369A1),
    'cyan': Color(0xFF0E7490),
    'rose': Color(0xFFBE123C),
    'violet': Color(0xFF6D28D9),
    'purple': Color(0xFF7E22CE),
    'fuchsia': Color(0xFFA21CAF),
    'indigo': Color(0xFF4338CA),
    'amber': Color(0xFFB45309),
    'orange': Color(0xFFC2410C),
    'yellow': Color(0xFFA16207),
    'slate': Color(0xFF334155),
  };
  return colors[badge.color] ??
      (badge.level == 'gold'
          ? colors['amber']!
          : badge.level == 'special'
          ? colors['indigo']!
          : colors['blue']!);
}

/// User profile aligned with the web identity card while keeping mobile
/// navigation, actions and content streams clear and thumb-friendly.
class ProfilePage extends ConsumerStatefulWidget {
  const ProfilePage({super.key, this.userId});

  final int? userId;

  @override
  ConsumerState<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends ConsumerState<ProfilePage> {
  AsyncValue<UserProfileProps> _page = const AsyncValue.loading();
  int _tabIndex = 0;
  bool _following = false;
  bool _loginRequired = false;

  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();
  GfTabScrollRegistry? _tabScrollRegistry;

  bool get _isShellProfile => widget.userId == null;

  @override
  void initState() {
    super.initState();
    if (_isShellProfile) {
      final GfTabScrollRegistry registry = ref.read(tabScrollRegistryProvider);
      registry.register(GfShellDestination.profile, _scrollToTopController);
      _tabScrollRegistry = registry;
    }
    _load();
  }

  @override
  void dispose() {
    _tabScrollRegistry?.unregister(
      GfShellDestination.profile,
      _scrollToTopController,
    );
    super.dispose();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent && mounted) {
      setState(() {
        _page = const AsyncValue.loading();
        _loginRequired = false;
      });
    }
    try {
      final int? currentId = (await ref.read(currentUserProvider.future))?.id;
      final int uid = widget.userId ?? currentId ?? 0;
      if (uid == 0) {
        if (!mounted) return;
        setState(() {
          _loginRequired = true;
          _page = AsyncValue.error(
            AppLocalizations.of(context).profileNotLoggedIn,
            StackTrace.current,
          );
        });
        return;
      }

      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .userProfile(uid);
      final UserProfileProps? props = parsePageProps<UserProfileProps>(payload);
      if (!mounted) return;
      setState(() {
        _loginRequired = false;
        _page = props == null
            ? AsyncValue.error(
                AppLocalizations.of(context).commonParseFailed,
                StackTrace.current,
              )
            : AsyncValue.data(props);
        _following = props?.user.isFollowing ?? false;
        if (props != null && props.activityTabs.isNotEmpty) {
          _tabIndex = _tabIndex.clamp(0, props.activityTabs.length - 1);
        }
      });
    } catch (e, st) {
      if (mounted) {
        setState(() {
          _loginRequired = false;
          _page = AsyncValue.error(e, st);
        });
      }
    }
  }

  Future<void> _toggleFollow(UserCardPayload user) async {
    if (user.isSelf) return;
    final bool wasFollowing = _following;
    final bool target = !wasFollowing;
    setState(() => _following = target);
    try {
      await ref
          .read(topicRepositoryProvider)
          .followUser(userId: user.userId, isFollowing: wasFollowing);
    } catch (_) {
      if (mounted) setState(() => _following = wasFollowing);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(
        title: Text(l10n.profileTitle),
        automaticallyImplyLeading: !_isShellProfile,
        actions: _isShellProfile
            ? <Widget>[
                GfIconButton(
                  icon: Icons.notifications_outlined,
                  tooltip: l10n.notificationsTitle,
                  size: 44,
                  onPressed: () => context.push('/notifications'),
                ),
              ]
            : const <Widget>[],
      ),
      body: _page.when(
        loading: () => const GfProfileSkeleton(),
        error: (e, _) => _isShellProfile
            ? _ProfileErrorBody(
                message: '$e',
                onRetry: _load,
                showLogin: _loginRequired,
                l10n: l10n,
              )
            : GfErrorRetry(message: '$e', onRetry: _load),
        data: (UserProfileProps props) {
          return GfScrollToTop(
            semanticLabel: l10n.commonBackToTop,
            controller: _isShellProfile ? _scrollToTopController : null,
            threshold: 360,
            builder: (BuildContext context, ScrollController controller) {
              return RefreshIndicator(
                onRefresh: () => _load(silent: true),
                child: CustomScrollView(
                  controller: controller,
                  physics: const AlwaysScrollableScrollPhysics(),
                  slivers: <Widget>[
                    SliverToBoxAdapter(child: _profileCard(props)),
                    const SliverToBoxAdapter(child: GfDivider()),
                    if (props.activityTabs.isNotEmpty)
                      SliverToBoxAdapter(
                        child: _ProfileTabs(
                          tabs: props.activityTabs,
                          index: _tabIndex,
                          onChanged: (int index) {
                            setState(() => _tabIndex = index);
                          },
                        ),
                      ),
                    const SliverToBoxAdapter(child: GfDivider()),
                    _ProfileBody(props: props, index: _tabIndex),
                    if (props.isOwnProfile || props.user.isSelf) ...<Widget>[
                      const SliverToBoxAdapter(child: SizedBox(height: 12)),
                      SliverToBoxAdapter(child: _AccountShortcuts(l10n: l10n)),
                    ],
                    const SliverToBoxAdapter(child: SizedBox(height: 32)),
                  ],
                ),
              );
            },
          );
        },
      ),
    );
  }

  Widget _profileCard(UserProfileProps props) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final UserCardPayload user = props.user;
    final Map<String, GfUserBadge> badges = <String, GfUserBadge>{};
    for (final UserBadgePayload badge in <UserBadgePayload>[
      ...user.badges,
      ...props.badges,
    ]) {
      badges.putIfAbsent(
        badge.code.isEmpty ? badge.name : badge.code,
        () => GfUserBadge(label: badge.name, color: _userBadgeColor(badge)),
      );
    }
    if (user.isAdmin) {
      badges['admin'] = const GfUserBadge(
        label: 'Admin',
        color: Color(0xFFB45309),
      );
    }

    final List<Widget> actions = <Widget>[];
    if (user.isSelf || props.isOwnProfile) {
      actions.add(
        GfButton(
          icon: const Icon(Icons.edit_outlined, size: 18),
          label: l10n.settingsEditProfile,
          variant: GfButtonVariant.outline,
          size: GfButtonSize.small,
          onPressed: () => context.push('/settings'),
        ),
      );
    } else {
      if (props.canFollow && !user.isAdmin) {
        actions.add(
          GfButton(
            label: _following ? l10n.profileFollowing : l10n.profileFollow,
            variant: _following
                ? GfButtonVariant.outline
                : GfButtonVariant.primary,
            size: GfButtonSize.small,
            onPressed: () => _toggleFollow(user),
          ),
        );
      }
      if (props.canMessage && props.messageUrl.trim().isNotEmpty) {
        actions.add(
          GfButton(
            icon: const Icon(Icons.mail_outline_rounded, size: 18),
            label: l10n.messagesNew,
            variant: GfButtonVariant.outline,
            size: GfButtonSize.small,
            onPressed: () => context.push(props.messageUrl),
          ),
        );
      }
    }

    return GfUserCard(
      coverUrl: resolveApiAssetUrl(user.profileCoverUrl),
      avatarUrl: resolveApiAssetUrl(user.avatarUrl),
      name: user.nickname.isEmpty ? user.username : user.nickname,
      username: user.username,
      bio: user.bio,
      signature: user.signature,
      coloredBadges: badges.values.toList(growable: false),
      stats: <(String, String)>[
        (l10n.profileTopics, formatNumber(user.topicCount)),
        (l10n.profileReplies, formatNumber(user.replyCount)),
        (l10n.profileLikes, formatNumber(user.likeReceivedCount)),
        (l10n.profileFollowers, formatNumber(user.followerCount)),
        (l10n.profileFollowingCount, formatNumber(user.followingCount)),
      ],
      actions: actions.isEmpty
          ? null
          : Wrap(spacing: 8, runSpacing: 8, children: actions),
    );
  }
}

class _ProfileErrorBody extends StatelessWidget {
  const _ProfileErrorBody({
    required this.message,
    required this.onRetry,
    required this.showLogin,
    required this.l10n,
  });

  final String message;
  final VoidCallback onRetry;
  final bool showLogin;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.only(top: 12, bottom: 32),
      children: <Widget>[
        GfErrorRetry(message: message, onRetry: onRetry),
        if (showLogin) ...<Widget>[
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: GfButton(
              icon: const Icon(Icons.login, size: 18),
              label: l10n.loginModeLogin,
              onPressed: () => context.push('/login'),
            ),
          ),
          const SizedBox(height: 12),
        ],
        _AccountShortcuts(l10n: l10n),
      ],
    );
  }
}

class _ProfileTabs extends StatelessWidget {
  const _ProfileTabs({
    required this.tabs,
    required this.index,
    required this.onChanged,
  });

  final List<TabItemPayload> tabs;
  final int index;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 48,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8),
        child: GfTabBar(
          tabs: <GfTab>[
            for (int i = 0; i < tabs.length; i++)
              GfTab(label: tabs[i].label ?? tabs[i].key, value: i),
          ],
          selected: index.clamp(0, tabs.length - 1),
          onSelected: (Object value) => onChanged(value as int),
        ),
      ),
    );
  }
}

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({required this.props, required this.index});

  static final RegExp _topicRoutePattern = RegExp(r'/p/(?:post/)?(\d+)');
  static final RegExp _userRoutePattern = RegExp(r'/u/(\d+)');

  final UserProfileProps props;
  final int index;

  String get _selectedKey {
    if (props.activityTabs.isEmpty) return 'activity';
    final int safeIndex = index.clamp(0, props.activityTabs.length - 1);
    return props.activityTabs[safeIndex].key;
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return switch (_selectedKey) {
      'topics' => _topicRows(context, l10n),
      'likes' => _likeRows(context, l10n),
      'bookmarks' => _bookmarkRows(context, l10n),
      'following' => _connectionRows(
        context,
        props.following,
        l10n.profileEmptyFollowing,
      ),
      'followers' => _connectionRows(
        context,
        props.followers,
        l10n.profileEmptyFollowers,
      ),
      _ => _activityRows(context, l10n),
    };
  }

  Widget _empty(IconData icon, String message) {
    return SliverToBoxAdapter(
      child: GfEmpty(icon: icon, message: message),
    );
  }

  Widget _activityRows(BuildContext context, AppLocalizations l10n) {
    if (props.activities.isEmpty) {
      return _empty(Icons.auto_awesome_outlined, l10n.profileEmptyActivity);
    }
    return SliverList.builder(
      itemCount: props.activities.length,
      itemBuilder: (BuildContext context, int index) {
        final UserActivityPayload activity = props.activities[index];
        final String? route = _activityRoute(activity);
        return GfSettingRow(
          icon: Icons.bolt_outlined,
          title: activity.contentPreview,
          description: timeAgo(activity.createdAt, l10n: l10n),
          onTap: route == null ? null : () => context.push(route),
        );
      },
    );
  }

  Widget _topicRows(BuildContext context, AppLocalizations l10n) {
    if (props.topics.isEmpty) {
      return _empty(Icons.article_outlined, l10n.profileEmptyTopics);
    }
    return SliverList.builder(
      itemCount: props.topics.length,
      itemBuilder: (BuildContext context, int index) => _topicRow(
        context,
        props.topics[index],
        showDivider: index < props.topics.length - 1,
      ),
    );
  }

  Widget _likeRows(BuildContext context, AppLocalizations l10n) {
    if (props.likes.isEmpty) {
      return _empty(Icons.favorite_border, l10n.profileEmptyLikes);
    }
    return SliverList.builder(
      itemCount: props.likes.length,
      itemBuilder: (BuildContext context, int index) {
        final UserLikePayload like = props.likes[index];
        return GfSettingRow(
          icon: Icons.favorite_border,
          title: like.title,
          description: timeAgo(like.likedAt, l10n: l10n),
          onTap: () => context.push('/p/${like.topicId}'),
        );
      },
    );
  }

  Widget _bookmarkRows(BuildContext context, AppLocalizations l10n) {
    if (props.bookmarks.isEmpty) {
      return _empty(Icons.bookmark_border, l10n.profileEmptyBookmarks);
    }
    return SliverList.builder(
      itemCount: props.bookmarks.length,
      itemBuilder: (BuildContext context, int index) {
        final UserBookmarkPayload bookmark = props.bookmarks[index];
        return GfSettingRow(
          icon: Icons.bookmark_border,
          title: bookmark.title,
          description: bookmark.excerpt?.isNotEmpty == true
              ? bookmark.excerpt
              : timeAgo(bookmark.bookmarkedAt, l10n: l10n),
          onTap: () => context.push('/p/${bookmark.topicId}'),
        );
      },
    );
  }

  Widget _connectionRows(
    BuildContext context,
    List<UserConnectionPayload> users,
    String emptyMessage,
  ) {
    if (users.isEmpty) return _empty(Icons.people_outline, emptyMessage);
    return SliverList.builder(
      itemCount: users.length,
      itemBuilder: (BuildContext context, int index) {
        final UserConnectionPayload user = users[index];
        return GfSettingRow(
          leading: GfAvatar(src: resolveApiAssetUrl(user.avatarUrl), size: 36),
          title: user.nickname.isEmpty ? user.username : user.nickname,
          description: user.bio.isEmpty ? '@${user.username}' : user.bio,
          onTap: () => context.push('/u/${user.id}'),
        );
      },
    );
  }

  Widget _topicRow(
    BuildContext context,
    TopicPayload topic, {
    required bool showDivider,
  }) {
    return GfTopicRow(
      title: topic.title,
      description: topic.description,
      categories: <GfTopicCategory>[
        for (final CategoryBriefPayload category in topic.categories)
          GfTopicCategory(
            name: category.name,
            color: colorFromHex(category.color),
          ),
      ],
      participantAvatarUrls: <String>[
        for (final UserBriefPayload participant in topic.participants)
          resolveApiAssetUrl(participant.avatarUrl),
      ],
      activityText: timeAgo(
        topic.activityText.isNotEmpty
            ? topic.activityText
            : topic.lastUpdateTime,
        l10n: AppLocalizations.of(context),
      ),
      replyCount: topic.replyCount,
      viewCount: topic.viewCount,
      hot: topic.viewCount > 500,
      pinned: topic.pinWeight > 0,
      unseen: topic.unseen == true,
      showDivider: showDivider,
      onTap: () => context.push('/p/${topic.id}'),
    );
  }

  String? _activityRoute(UserActivityPayload activity) {
    if (activity.subjectType == 'topic' && activity.subjectId > 0) {
      return '/p/${activity.subjectId}';
    }
    final RegExpMatch? topicMatch = _topicRoutePattern.firstMatch(activity.url);
    if (topicMatch != null) return '/p/${topicMatch.group(1)}';
    final RegExpMatch? userMatch = _userRoutePattern.firstMatch(activity.url);
    if (userMatch != null) return '/u/${userMatch.group(1)}';
    return null;
  }
}

class _AccountShortcuts extends StatelessWidget {
  const _AccountShortcuts({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: GfCardList(
        children: <Widget>[
          GfSettingRow(
            icon: Icons.settings_outlined,
            title: l10n.settingsTitle,
            onTap: () => context.push('/settings'),
          ),
          GfSettingRow(
            icon: Icons.notifications_outlined,
            title: l10n.notificationsTitle,
            onTap: () => context.push('/notifications'),
          ),
          GfSettingRow(
            icon: Icons.description_outlined,
            title: l10n.draftsTitle,
            onTap: () => context.push('/drafts'),
          ),
        ],
      ),
    );
  }
}

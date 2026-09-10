import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../current_user.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../profile_links.dart';
import '../../server_messages.dart';
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
  const ProfilePage({super.key, this.userId, this.initialStream = 'timeline'});

  final int? userId;
  final String initialStream;

  @override
  ConsumerState<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends ConsumerState<ProfilePage> {
  AsyncValue<UserProfileProps> _page = const AsyncValue.loading();
  int _tabIndex = 0;
  int _request = 0;
  bool _loadingMore = false;
  bool _streamLoading = false;
  Object? _streamError;
  UserProfileProps? _headerProps;
  double _minimumScrollOffset = 0;
  String _stream = 'timeline';
  bool _following = false;
  bool _followBusy = false;
  bool _loginRequired = false;
  bool _canAccessAdmin = false;
  bool _canModerate = false;
  bool _canManageCourses = false;

  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();

  bool get _isShellProfile => widget.userId == null;

  @override
  void initState() {
    super.initState();
    _stream = widget.initialStream;
    _tabIndex = _stream == 'bookmarks' ? 3 : 0;

    _load();
  }

  Future<void> _load({
    bool silent = false,
    String? nextUrl,
    bool streamChange = false,
  }) async {
    final request = ++_request;
    final previous = _page.valueOrNull;
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

      final path = nextUrl ?? _streamPath(uid, _stream);
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch(path);
      var props = parsePageProps<UserProfileProps>(payload);
      if (!mounted || request != _request) return;
      if (nextUrl != null && previous != null && props != null) {
        props = props.copyWith(
          topics: [...previous.topics, ...props.topics],
          activities: [...previous.activities, ...props.activities],
          likes: [...previous.likes, ...props.likes],
          bookmarks: [...previous.bookmarks, ...props.bookmarks],
          following: [...previous.following, ...props.following],
          followers: [...previous.followers, ...props.followers],
        );
      }
      if (props == null) {
        throw FormatException(AppLocalizations.of(context).commonParseFailed);
      }
      final loaded = props;
      setState(() {
        _loginRequired = false;
        _page = AsyncValue.data(loaded);
        if (!streamChange && nextUrl == null) _headerProps = loaded;
        _streamLoading = false;
        _streamError = null;
        _following = loaded.user.isFollowing;
        _canAccessAdmin = payload.layout.viewer.canAccessAdmin;
        _canModerate = payload.layout.viewer.isModerator;
        _canManageCourses = payload.layout.viewer.canManageCourses;
      });
    } catch (e, st) {
      if (mounted && request == _request) {
        setState(() {
          _loginRequired = false;
          if (streamChange && previous != null) {
            _page = AsyncValue.data(previous);
            _streamLoading = false;
            _streamError = e;
          } else {
            _page = AsyncValue.error(e, st);
          }
        });
      }
    }
  }

  String _streamPath(int uid, String key) => switch (key) {
    'bookmarks' || 'badges' => '/u/$uid/$key',
    'timeline' => '/u/$uid/activity',
    _ => '/u/$uid/activity/$key',
  };

  List<TabItemPayload> _tabs(UserProfileProps props, AppLocalizations l10n) => [
    TabItemPayload(
      key: 'timeline',
      label: l10n.profileActivity,
      url: '',
      active: false,
    ),
    TabItemPayload(
      key: 'topics',
      label: l10n.profileTopics,
      url: '',
      active: false,
    ),
    TabItemPayload(
      key: 'likes',
      label: l10n.profileLikes,
      url: '',
      active: false,
    ),
    if (props.isOwnProfile)
      TabItemPayload(
        key: 'bookmarks',
        label: l10n.profileBookmarks,
        url: '',
        active: false,
      ),
    TabItemPayload(
      key: 'following',
      label: l10n.profileFollowingCount,
      url: '',
      active: false,
    ),
    TabItemPayload(
      key: 'followers',
      label: l10n.profileFollowers,
      url: '',
      active: false,
    ),
    TabItemPayload(
      key: 'badges',
      label: l10n.profileBadges,
      url: '',
      active: false,
    ),
  ];

  Future<void> _loadMore(UserProfileProps props) async {
    if (_loadingMore || !props.pagination.hasNext) return;
    final uri = Uri.tryParse(props.pagination.nextUrl);
    // SSR pagination stays in the current user's profile; reject foreign URLs.
    if (uri == null ||
        uri.hasScheme ||
        uri.hasAuthority ||
        !uri.path.startsWith('/u/${props.user.userId}/')) {
      return;
    }
    setState(() => _loadingMore = true);
    await _load(silent: true, nextUrl: uri.toString());
    if (mounted) setState(() => _loadingMore = false);
  }

  Future<void> _toggleFollow(UserCardPayload user) async {
    if (user.isSelf || _followBusy) return;
    final bool wasFollowing = _following;
    final bool target = !wasFollowing;
    setState(() {
      _following = target;
      _followBusy = true;
    });
    try {
      await ref
          .read(topicRepositoryProvider)
          .followUser(userId: user.userId, isFollowing: wasFollowing);
    } catch (error) {
      if (mounted) {
        setState(() => _following = wasFollowing);
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _followBusy = false);
    }
  }

  Future<void> _openProfileTool(String route) async {
    await context.push(route);
    if (mounted) await _load(silent: true);
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(
        title: Text(l10n.profileTitle),
        automaticallyImplyLeading: true,
        actions: _isShellProfile || _page.valueOrNull?.isOwnProfile == true
            ? <Widget>[
                PopupMenuButton<String>(
                  tooltip: l10n.profileMore,
                  onSelected: _openProfileTool,
                  itemBuilder: (_) => [
                    PopupMenuItem(
                      value: '/drafts',
                      child: Text(l10n.draftsTitle),
                    ),
                    PopupMenuItem(
                      value: '/my-course-reviews',
                      child: Text(l10n.myCourseReviewsTitle),
                    ),
                    PopupMenuItem(
                      value: '/my-content',
                      child: Text(l10n.profileContent),
                    ),
                    PopupMenuItem(
                      value: '/recycle-bin',
                      child: Text(l10n.profileTrash),
                    ),
                    if (_canModerate)
                      PopupMenuItem(
                        value: '/moderation',
                        child: Text(l10n.profileModeration),
                      ),
                    if (_canAccessAdmin)
                      PopupMenuItem(
                        value: '/admin',
                        child: Text(l10n.profileAdmin),
                      ),
                    if (_canManageCourses) ...[
                      PopupMenuItem(
                        value: '/moderation/courses',
                        child: Text(l10n.coursesManagement),
                      ),
                      PopupMenuItem(
                        value: '/moderation/course-reviews',
                        child: Text(l10n.coursesReviewModeration),
                      ),
                    ],
                    const PopupMenuDivider(),
                    PopupMenuItem(
                      value: '/settings/account',
                      child: Text(l10n.profileSecurity),
                    ),
                    PopupMenuItem(
                      value: '/settings',
                      child: Text(l10n.settingsTitle),
                    ),
                  ],
                ),
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
                message: resolveErrorMessage(l10n, e),
                onRetry: _load,
                showLogin: _loginRequired,
                l10n: l10n,
              )
            : GfErrorRetry(
                message: resolveErrorMessage(l10n, e),
                onRetry: _load,
              ),
        data: (UserProfileProps props) {
          final tabs = _tabs(props, l10n);
          return GfScrollToTop(
            semanticLabel: l10n.commonBackToTop,
            controller: _isShellProfile ? _scrollToTopController : null,
            threshold: 360,
            builder: (BuildContext context, ScrollController controller) {
              return AppRefreshIndicator(
                onRefresh: () => _load(silent: true),
                child: CustomScrollView(
                  controller: controller,
                  physics: const AlwaysScrollableScrollPhysics(),
                  slivers: <Widget>[
                    SliverToBoxAdapter(
                      child: _profileCard(_headerProps ?? props),
                    ),
                    if (props.isOwnProfile)
                      SliverToBoxAdapter(
                        child: ListTile(
                          leading: GfSymbol(
                            'star',
                            color: GfTheme.colorsOf(context).primary,
                          ),
                          title: Text(l10n.myCourseReviewsTitle),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () => context.push('/my-course-reviews'),
                        ),
                      ),
                    const SliverToBoxAdapter(child: GfDivider()),
                    if (tabs.isNotEmpty)
                      SliverLayoutBuilder(
                        builder: (context, constraints) => SliverToBoxAdapter(
                          child: _ProfileTabs(
                            tabs: tabs,
                            index: _tabIndex,
                            onChanged: (int index) {
                              if (_stream == tabs[index].key) return;
                              // Retain header collapse, not an arbitrary offset
                              // into the previous stream that could hide status.
                              final retainedOffset = math.min(
                                math.max(0.0, controller.offset),
                                constraints.precedingScrollExtent,
                              );
                              controller.jumpTo(retainedOffset);
                              setState(() {
                                _minimumScrollOffset = retainedOffset;
                                _tabIndex = index;
                                _stream = tabs[index].key;
                                _streamLoading = true;
                                _streamError = null;
                              });
                              _load(silent: true, streamChange: true);
                            },
                          ),
                        ),
                      ),
                    const SliverToBoxAdapter(child: GfDivider()),
                    if (_streamLoading)
                      const SliverToBoxAdapter(
                        child: Padding(
                          padding: EdgeInsets.all(24),
                          child: Center(child: GfLoadingIndicator(small: true)),
                        ),
                      )
                    else if (_streamError != null)
                      SliverToBoxAdapter(
                        child: GfErrorRetry(
                          message: resolveErrorMessage(l10n, _streamError!),
                          onRetry: () {
                            setState(() {
                              _streamLoading = true;
                              _streamError = null;
                            });
                            _load(silent: true, streamChange: true);
                          },
                        ),
                      )
                    else
                      _ProfileBody(props: props, selectedKey: _stream),
                    if (!_streamLoading &&
                        _streamError == null &&
                        props.pagination.hasNext)
                      SliverToBoxAdapter(
                        child: Padding(
                          padding: const EdgeInsets.all(16),
                          child: GfButton(
                            label: l10n.commonLoadMore,
                            loading: _loadingMore,
                            onPressed: () => _loadMore(props),
                          ),
                        ),
                      ),

                    // Short/empty streams must not clamp the shared header back
                    // into view. Reserve only the retained header-collapse offset.
                    SliverLayoutBuilder(
                      builder: (context, constraints) => SliverToBoxAdapter(
                        child: SizedBox(
                          height: math.max(
                            32,
                            _minimumScrollOffset +
                                constraints.viewportMainAxisExtent -
                                constraints.precedingScrollExtent,
                          ),
                        ),
                      ),
                    ),
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
    if (user.isAdmin) {
      badges['admin'] = GfUserBadge(
        label: l10n.profileRoleAdmin,
        color: Color(0xFFB45309),
      );
    }

    final List<Widget> actions = <Widget>[];
    if (user.isSelf || props.isOwnProfile) {
      actions.add(
        GfButton(
          icon: const GfSymbol('square-pen', size: 18),
          label: l10n.settingsEditProfile,
          variant: GfButtonVariant.outline,
          size: GfButtonSize.small,
          onPressed: () => _openProfileTool('/settings/profile'),
        ),
      );
    } else {
      if (props.canFollow) {
        actions.add(
          GfButton(
            label: _following ? l10n.profileFollowing : l10n.profileFollow,
            icon: GfSymbol(
              _following ? 'user-round-check' : 'user-round-plus',
              size: 18,
            ),
            loading: _followBusy,
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
            icon: const GfSymbol('mail', size: 18),
            label: l10n.messagesNew,
            variant: GfButtonVariant.outline,
            size: GfButtonSize.small,
            onPressed: () => context.push(props.messageUrl),
          ),
        );
      }
    }

    final links = publicProfileLinks(user);
    return GfUserCard(
      coverUrl: resolveApiAssetUrl(user.profileCoverUrl),
      avatarUrl: resolveApiAssetUrl(user.avatarUrl),
      avatarBadge: user.wornBadge == null
          ? null
          : GfBadgeIcon(
              url: resolveApiAssetUrl(
                user.wornBadge!.iconUrl.isEmpty
                    ? '/static/badges/contributor.svg'
                    : user.wornBadge!.iconUrl,
              ),
              label: user.wornBadge!.name,
            ),
      name: user.nickname.isEmpty ? user.username : user.nickname,
      username: user.username,
      bio: user.bio,
      signature: user.signature,
      details: links.isEmpty
          ? null
          : Wrap(
              spacing: 4,
              runSpacing: 4,
              children: [
                for (final (label, uri, provider) in links)
                  TextButton.icon(
                    style: TextButton.styleFrom(
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                    ),
                    icon: GfSocialIcon(provider, size: 20),
                    label: Text(
                      label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    onPressed: () async {
                      try {
                        if (!await launchUrl(
                          uri,
                          mode: LaunchMode.externalApplication,
                        )) {
                          throw StateError('Could not open profile link');
                        }
                      } catch (error) {
                        if (mounted) {
                          showGfToast(
                            context,
                            resolveErrorMessage(l10n, error),
                            error: true,
                          );
                        }
                      }
                    },
                  ),
              ],
            ),
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
    final colors = GfTheme.colorsOf(context);
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: Row(
        children: [
          for (int i = 0; i < tabs.length; i++)
            Tooltip(
              message: tabs[i].label ?? tabs[i].key,
              child: Semantics(
                selected: i == index,
                button: true,
                label: tabs[i].label ?? tabs[i].key,
                child: InkWell(
                  onTap: () => onChanged(i),
                  borderRadius: BorderRadius.circular(24),
                  child: Container(
                    constraints: const BoxConstraints(
                      minWidth: 48,
                      minHeight: 48,
                    ),
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    decoration: BoxDecoration(
                      border: Border(
                        bottom: BorderSide(
                          color: i == index
                              ? colors.primary
                              : Colors.transparent,
                          width: 3,
                        ),
                      ),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        GfSymbol(
                          switch (tabs[i].key) {
                            'topics' => 'file-text',
                            'likes' => 'heart',
                            'bookmarks' => 'bookmark',
                            'following' => 'user-round-plus',
                            'followers' => 'users-round',
                            'badges' => 'award',
                            _ => 'activity',
                          },
                          size: 22,
                          color: i == index ? colors.primary : colors.iconMuted,
                        ),
                        if (i == index) ...[
                          const SizedBox(width: 8),
                          Text(
                            tabs[i].label ?? tabs[i].key,
                            style: TextStyle(
                              fontWeight: FontWeight.w600,
                              color: colors.primary,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({required this.props, required this.selectedKey});

  static final RegExp _topicRoutePattern = RegExp(r'/p/(?:post/)?(\d+)');
  static final RegExp _userRoutePattern = RegExp(r'/u/(\d+)');

  final UserProfileProps props;
  final String selectedKey;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return switch (selectedKey) {
      'badges' =>
        props.badges.isEmpty
            ? _empty(Icons.workspace_premium_outlined, l10n.profileNoBadges)
            : SliverPadding(
                padding: const EdgeInsets.all(16),
                sliver: SliverList.separated(
                  itemCount: props.badges.length,
                  separatorBuilder: (_, _) => const SizedBox(height: 10),
                  itemBuilder: (_, index) {
                    final badge = props.badges[index];
                    return GfAchievementCard(
                      title: badge.name,
                      description: badge.description,
                      color: _userBadgeColor(badge),
                      icon: GfBadgeIcon(
                        url: resolveApiAssetUrl(badge.iconUrl),
                        label: badge.name,
                        framed: false,
                        size: 28,
                      ),
                    );
                  },
                ),
              ),
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
        final action = switch (activity.action) {
          1 => 'signup',
          2 => 'post',
          3 => 'like',
          4 => 'follow',
          5 => 'comment',
          _ => activity.label,
        };
        return GfActivityCard(
          symbol: switch (action) {
            'signup' => 'user-round',
            'post' => 'square-pen',
            'like' => 'heart',
            'follow' => 'user-round-plus',
            'comment' => 'message-circle',
            _ => 'activity',
          },
          color: switch (action) {
            'like' => const Color(0xFFE11D48),
            'follow' => const Color(0xFF7C3AED),
            'comment' => const Color(0xFF059669),
            _ => GfTheme.colorsOf(context).primary,
          },
          title: [
            switch (action) {
              'signup' => l10n.profileActionSignup,
              'post' => l10n.profileActionPost,
              'like' => l10n.profileActionLike,
              'follow' => l10n.profileActionFollow,
              'comment' => l10n.profileActionComment,
              _ => activity.label,
            },
            activity.contentPreview,
          ].where((s) => s.isNotEmpty).join(' · '),
          time: timeAgo(activity.createdAt, l10n: l10n),
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

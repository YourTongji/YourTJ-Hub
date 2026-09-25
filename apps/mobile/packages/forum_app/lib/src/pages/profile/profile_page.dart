import '../../private_notes.dart';
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
import '../../widgets/topic_list.dart';

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
  const ProfilePage({super.key, this.userId, this.initialStream = 'timeline'})
    : connectionsOnly = false;

  const ProfilePage.connections({
    super.key,
    this.userId,
    this.initialStream = 'following',
  }) : connectionsOnly = true;

  final bool connectionsOnly;

  final int? userId;
  final String initialStream;

  @override
  ConsumerState<ProfilePage> createState() => _ProfilePageState();
}

class _ProfileStreamState {
  UserProfileProps? props;
  bool loading = false;
  bool loadingMore = false;
  Object? error;
  Object? paginationError;
  int request = 0;
  double? offset;
}

class _ProfilePageState extends ConsumerState<ProfilePage> {
  final _streams = <String, _ProfileStreamState>{};
  _ProfileStreamState get _active =>
      _streams.putIfAbsent(_stream, _ProfileStreamState.new);
  AsyncValue<UserProfileProps> get _page {
    final props = _active.props ?? _headerProps;
    if (props != null) return AsyncValue.data(props);
    if (_active.error != null) {
      return AsyncValue.error(_active.error!, StackTrace.current);
    }
    return const AsyncValue.loading();
  }

  bool get _loadingMore => _active.loadingMore;
  bool get _streamLoading => _active.loading && _active.props == null;
  Object? get _streamError => _active.error;
  int _followRevision = 0;
  UserProfileProps? _headerProps;
  double _minimumScrollOffset = 0;
  String _stream = 'timeline';
  bool _following = false;
  bool _followBusy = false;
  final _connectionFollowing = <int, bool>{};
  final _connectionBusy = <int>{};
  int _connectionEpoch = 0;
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

    _load();
  }

  @override
  void didUpdateWidget(covariant ProfilePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.userId != widget.userId ||
        oldWidget.initialStream != widget.initialStream ||
        oldWidget.connectionsOnly != widget.connectionsOnly) {
      _stream = widget.initialStream;
      _resetStreams();
      _load();
    }
  }

  void _resetStreams() {
    _streams.clear();
    _connectionFollowing.clear();
    _connectionBusy.clear();
    _connectionEpoch++;
    _headerProps = null;
    _minimumScrollOffset = 0;
    _followBusy = false;
    _following = false;
    _canAccessAdmin = false;
    _canModerate = false;
    _canManageCourses = false;
    _followRevision++;
  }

  Future<void> _load({String? nextUrl, bool streamChange = false}) async {
    final key = _stream;
    final state = _active;
    final request = ++state.request;
    final epoch = ref.read(offlineCacheEpochProvider);
    final followRevision = _followRevision;
    final previous = state.props;
    bool current() =>
        mounted &&
        request == state.request &&
        identical(_streams[key], state) &&
        ref.read(offlineCacheEpochProvider) == epoch;
    setState(() {
      state.loading = nextUrl == null;
      state.loadingMore = nextUrl != null;
      state.error = null;
      state.paginationError = null;
      _loginRequired = false;
    });
    try {
      final int? currentId = (await ref.read(currentUserProvider.future))?.id;
      if (!mounted || !current()) return;
      final int uid = widget.userId ?? currentId ?? 0;
      if (uid == 0) {
        setState(() {
          _loginRequired = true;
          state.error = AppLocalizations.of(context).profileNotLoggedIn;
        });
        return;
      }
      final path = nextUrl ?? _streamPath(uid, key);
      final payload = await ref.read(pageRepositoryProvider).fetch(path);
      if (!mounted || !current()) return;
      var props = parsePageProps<UserProfileProps>(payload);
      if (props == null) {
        throw FormatException(AppLocalizations.of(context).commonParseFailed);
      }
      if (nextUrl != null && previous != null) {
        props = props.copyWith(
          topics: _merge(previous.topics, props.topics, (item) => item.id),
          activities: _merge(
            previous.activities,
            props.activities,
            (item) => item.id,
          ),
          likes: _merge(previous.likes, props.likes, (item) => item.id),
          bookmarks: _merge(
            previous.bookmarks,
            props.bookmarks,
            (item) => item.id,
          ),
          following: _merge(
            previous.following,
            props.following,
            (item) => item.id,
          ),
          followers: _merge(
            previous.followers,
            props.followers,
            (item) => item.id,
          ),
        );
      }
      final loaded = props;
      setState(() {
        state.props = loaded;
        // Inactive streams may finish, but cannot replace the visible identity
        // or undo a follow action started after this read.
        if (key == _stream && nextUrl == null && !streamChange) {
          _headerProps = loaded;
          if (!_followBusy && followRevision == _followRevision) {
            _following = loaded.user.isFollowing;
          }
          _canAccessAdmin = payload.layout.viewer.canAccessAdmin;
          _canModerate = payload.layout.viewer.isModerator;
          _canManageCourses = payload.layout.viewer.canManageCourses;
        }
      });
    } catch (error) {
      if (mounted && current()) {
        setState(() {
          if (nextUrl != null) {
            state.paginationError = error;
          } else {
            state.error = error;
          }
        });
      }
    } finally {
      if (mounted && current()) {
        setState(() {
          state.loading = false;
          state.loadingMore = false;
        });
      }
    }
  }

  List<T> _merge<T>(List<T> previous, List<T> next, int Function(T) id) => {
    for (final item in previous) id(item): item,
    for (final item in next) id(item): item,
  }.values.toList();

  void _selectStream(
    String key,
    ScrollController controller,
    double headerExtent,
  ) {
    if (key == _stream) return;
    _active.offset = controller.offset;
    final state = _streams.putIfAbsent(key, _ProfileStreamState.new);
    final offset =
        state.offset ??
        math.min(math.max(0.0, controller.offset), headerExtent);
    setState(() {
      _stream = key;
      _minimumScrollOffset = offset;
    });
    // Restore after the new slivers have laid out, preserving deep offsets for
    // visited streams and only the collapsed header for a first visit.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _stream == key && controller.hasClients) {
        controller.jumpTo(offset);
      }
    });
    if (state.props == null && !state.loading && state.error == null) {
      _load(streamChange: true);
    }
  }

  String _streamPath(int uid, String key) => switch (key) {
    'bookmarks' || 'badges' || 'following' || 'followers' => '/u/$uid/$key',
    'timeline' => '/u/$uid/activity',
    _ => '/u/$uid/activity/$key',
  };

  List<TabItemPayload> _tabs(UserProfileProps props, AppLocalizations l10n) => [
    for (final (key, label)
        in widget.connectionsOnly
            ? [
                ('following', l10n.profileFollowingCount),
                ('followers', l10n.profileFollowers),
              ]
            : [
                ('timeline', l10n.profileActivity),
                ('topics', l10n.profileTopics),
                ('likes', l10n.profileLikedPosts),
                if (props.isOwnProfile) ('bookmarks', l10n.profileBookmarks),
                ('badges', l10n.profileBadges),
              ])
      TabItemPayload(key: key, label: label, url: '', active: key == _stream),
  ];

  Future<void> _loadMore(UserProfileProps props) async {
    if (_active.loading || _loadingMore || !props.pagination.hasNext) return;
    final uri = Uri.tryParse(props.pagination.nextUrl);
    // Follow only relative pagination URLs for this exact user and stream.
    if (uri == null ||
        uri.hasScheme ||
        uri.hasAuthority ||
        uri.path != _streamPath(props.user.userId, _stream)) {
      return;
    }
    await _load(nextUrl: uri.toString());
  }

  Future<void> _toggleFollow(UserCardPayload user) async {
    if (user.isSelf || _followBusy) return;
    final revision = ++_followRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    bool current() =>
        mounted &&
        revision == _followRevision &&
        ref.read(offlineCacheEpochProvider) == epoch;
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
      if (mounted && current()) {
        setState(() => _following = wasFollowing);
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (current()) {
        setState(() {
          // Reads begun during the mutation also precede its settled truth.
          _followRevision++;
          _followBusy = false;
        });
      }
    }
  }

  Future<void> _toggleConnection(UserConnectionPayload user) async {
    if (user.isSelf || _connectionBusy.contains(user.id)) return;
    if (ref.read(currentUserProvider).valueOrNull == null) {
      await context.push('/login');
      return;
    }
    final previous = _connectionFollowing[user.id] ?? user.isFollowing;
    if (previous == null) return;
    final epoch = _connectionEpoch;
    final session = ref.read(offlineCacheEpochProvider);
    bool current() =>
        mounted &&
        epoch == _connectionEpoch &&
        session == ref.read(offlineCacheEpochProvider);
    setState(() {
      _connectionFollowing[user.id] = !previous;
      _connectionBusy.add(user.id);
    });
    try {
      await ref
          .read(topicRepositoryProvider)
          .followUser(userId: user.id, isFollowing: previous);
    } catch (error) {
      if (mounted && current()) {
        setState(() => _connectionFollowing[user.id] = previous);
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (current()) setState(() => _connectionBusy.remove(user.id));
    }
  }

  Future<void> _openProfileTool(String route) async {
    await context.push(route);
    if (mounted) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (previous, next) {
      if (previous == next) return;
      setState(_resetStreams);
      _load();
    });
    return Scaffold(
      appBar: GfAppBar(
        centerTitle: false,
        title: _page.valueOrNull == null
            ? Text(l10n.profileTitle)
            : Builder(
                builder: (context) {
                  final user = (_headerProps ?? _page.valueOrNull!).user;
                  return Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        privateDisplayName(
                          context,
                          user.userId,
                          user.username,
                          user.nickname,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      if (MediaQuery.textScalerOf(context).scale(13) <= 18)
                        Text(
                          widget.connectionsOnly
                              ? '@${user.username}'
                              : '${formatNumber(user.topicCount)} ${l10n.profileTopics}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 13,
                            height: 1.2,
                            fontWeight: FontWeight.w400,
                            color: GfTheme.colorsOf(context).iconMuted,
                          ),
                        ),
                    ],
                  );
                },
              ),
        automaticallyImplyLeading: true,
        actions:
            !widget.connectionsOnly &&
                (_isShellProfile || _page.valueOrNull?.isOwnProfile == true)
            ? <Widget>[
                PopupMenuButton<String>(
                  tooltip: l10n.profileMore,
                  useRootNavigator: true,
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
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: _page.when(
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
                    onRefresh: () => _load(),
                    child: CustomScrollView(
                      controller: controller,
                      physics: const AlwaysScrollableScrollPhysics(),
                      slivers: <Widget>[
                        if (!widget.connectionsOnly)
                          SliverToBoxAdapter(
                            child: _profileCard(_headerProps ?? props),
                          ),
                        const SliverToBoxAdapter(child: GfDivider()),
                        if (tabs.isNotEmpty)
                          SliverLayoutBuilder(
                            builder: (context, constraints) =>
                                SliverPersistentHeader(
                                  pinned: true,
                                  delegate: _ProfileTabsHeader(
                                    height: math.max(
                                      52,
                                      MediaQuery.textScalerOf(
                                                context,
                                              ).scale(16) *
                                              1.4 +
                                          24,
                                    ),
                                    child: _ProfileTabs(
                                      tabs: tabs,
                                      index: tabs.indexWhere(
                                        (tab) => tab.key == _stream,
                                      ),
                                      onChanged: (index) => _selectStream(
                                        tabs[index].key,
                                        controller,
                                        constraints.precedingScrollExtent,
                                      ),
                                    ),
                                  ),
                                ),
                          ),
                        if (_streamLoading)
                          const SliverToBoxAdapter(
                            child: Padding(
                              padding: EdgeInsets.all(24),
                              child: Center(
                                child: GfLoadingIndicator(small: true),
                              ),
                            ),
                          )
                        else if (_streamError != null)
                          SliverToBoxAdapter(
                            child: GfErrorRetry(
                              message: resolveErrorMessage(l10n, _streamError!),
                              onRetry: () {
                                _load(streamChange: true);
                              },
                            ),
                          ),
                        if (!_streamLoading && _active.props != null)
                          // Key each stream so recycled SliverList geometry cannot
                          // shift its restored scroll offset.
                          _ProfileBody(
                            key: ValueKey(_stream),
                            props: props,
                            selectedKey: _stream,
                            following: _connectionFollowing,
                            busy: _connectionBusy,
                            onFollow: _toggleConnection,
                          ),
                        if (!_streamLoading &&
                            _streamError == null &&
                            props.pagination.hasNext)
                          SliverToBoxAdapter(
                            child: GfListFooter(
                              key: ValueKey(_stream),
                              error: _active.paginationError == null
                                  ? null
                                  : resolveErrorMessage(
                                      l10n,
                                      _active.paginationError!,
                                    ),
                              progressKey: (_stream, props.pagination.nextUrl),
                              hasMore: props.pagination.hasNext,
                              loading: _loadingMore,
                              onLoadMore: () => _loadMore(props),
                            ),
                          ),

                        // Keep short streams from clamping a restored offset.
                        // First visits retain only the header-collapse offset.
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
        ),
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

    for (final badge in (user.displayBadges ?? user.badges.take(5))) {
      badges['earned:${badge.code}'] = GfUserBadge(
        label: badge.name,
        color: _userBadgeColor(badge),
      );
    }
    final List<Widget> actions = <Widget>[];
    if (user.isSelf || props.isOwnProfile) {
      actions.add(
        OutlinedButton(
          style: OutlinedButton.styleFrom(
            minimumSize: const Size(96, 44),
            shape: const StadiumBorder(),
            foregroundColor: GfTheme.colorsOf(context).baseContent,
            side: BorderSide(color: GfTheme.colorsOf(context).line),
            textStyle: const TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w700,
            ),
          ),
          onPressed: () => _openProfileTool('/settings/profile'),
          child: Text(l10n.settingsEditProfile, textAlign: TextAlign.center),
        ),
      );
    } else {
      actions.add(
        PrivateNoteButton(userId: user.userId, username: user.username),
      );
      if (props.canFollow) {
        actions.add(
          GfFollowButton(
            following: _following,
            label: _following ? l10n.profileFollowing : l10n.profileFollow,
            busy: _followBusy,
            onPressed: () => _toggleFollow(user),
          ),
        );
      }
      if (props.canMessage && props.messageUrl.trim().isNotEmpty) {
        actions.add(
          IconButton.outlined(
            icon: const GfSymbol('mail', size: 20),
            tooltip: l10n.messagesNew,
            constraints: const BoxConstraints(minWidth: 44, minHeight: 44),
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
      name: privateDisplayName(
        context,
        user.userId,
        user.username,
        user.nickname,
      ),
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
        (l10n.profileFollowingCount, formatNumber(user.followingCount)),
        (l10n.profileFollowers, formatNumber(user.followerCount)),
        (l10n.profileTopics, formatNumber(user.topicCount)),
        (l10n.profileReplies, formatNumber(user.replyCount)),
        (l10n.profileLikes, formatNumber(user.likeReceivedCount)),
      ],
      statActions: {
        0: () => context.push('/u/${user.userId}/following'),
        1: () => context.push('/u/${user.userId}/followers'),
      },
      actions: actions.isEmpty
          ? null
          : Wrap(
              alignment: WrapAlignment.end,
              spacing: 8,
              runSpacing: 8,
              children: actions,
            ),
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

class _ProfileTabsHeader extends SliverPersistentHeaderDelegate {
  const _ProfileTabsHeader({required this.height, required this.child});
  final double height;
  final Widget child;
  @override
  double get minExtent => height;
  @override
  double get maxExtent => height;
  @override
  Widget build(
    BuildContext context,
    double shrinkOffset,
    bool overlapsContent,
  ) => Material(
    color: GfTheme.colorsOf(context).base100,
    child: Column(
      children: [
        Expanded(child: child),
        const GfDivider(),
      ],
    ),
  );
  @override
  bool shouldRebuild(covariant _ProfileTabsHeader oldDelegate) => true;
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
    return LayoutBuilder(
      builder: (context, constraints) => SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          children: [
            for (int i = 0; i < tabs.length; i++)
              Tooltip(
                message: tabs[i].label ?? tabs[i].key,
                excludeFromSemantics: true,
                child: Semantics(
                  selected: i == index,
                  button: true,
                  label: tabs[i].label ?? tabs[i].key,
                  child: InkWell(
                    onTap: () => onChanged(i),
                    child: Container(
                      alignment: Alignment.center,
                      constraints: BoxConstraints(
                        minWidth: math.max(
                          72,
                          constraints.maxWidth / tabs.length,
                        ),
                        minHeight: 48,
                      ),
                      height: math.max(
                        52,
                        MediaQuery.textScalerOf(context).scale(16) * 1.4 + 24,
                      ),
                      child: Stack(
                        alignment: Alignment.center,
                        children: [
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 20),
                            child: ExcludeSemantics(
                              child: Text(
                                tabs[i].label ?? tabs[i].key,
                                style: TextStyle(
                                  fontSize: 16,
                                  fontWeight: FontWeight.w600,
                                  color: i == index
                                      ? colors.baseContent
                                      : colors.iconMuted,
                                ),
                              ),
                            ),
                          ),
                          if (i == index)
                            Positioned(
                              bottom: 0,
                              width: 40,
                              height: 3,
                              child: DecoratedBox(
                                decoration: BoxDecoration(
                                  color: colors.primary,
                                  borderRadius: BorderRadius.circular(3),
                                ),
                              ),
                            ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({
    super.key,
    required this.props,
    required this.selectedKey,
    required this.following,
    required this.busy,
    required this.onFollow,
  });

  final UserProfileProps props;
  final String selectedKey;
  final Map<int, bool> following;
  final Set<int> busy;
  final ValueChanged<UserConnectionPayload> onFollow;

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
        return GfContentRow(
          author: privateDisplayName(
            context,
            props.user.userId,
            props.user.username,
            props.user.nickname,
          ),
          avatarUrl: resolveApiAssetUrl(props.user.avatarUrl),
          contextIcon: switch (action) {
            'signup' => Icons.person_outline,
            'post' => Icons.edit_outlined,
            'like' => Icons.favorite,
            'follow' => Icons.person_add_outlined,
            'comment' => Icons.chat_bubble_outline,
            _ => Icons.timeline,
          },
          contextLabel: switch (action) {
            'signup' => l10n.profileActionSignup,
            'post' => l10n.profileActionPost,
            'like' => l10n.profileActionLike,
            'follow' => l10n.profileActionFollow,
            'comment' => l10n.profileActionComment,
            _ => activity.label,
          },
          text: activity.contentPreview,
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
      itemBuilder: (BuildContext context, int index) =>
          buildTopicFeedCard(context, props.topics[index]),
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
        return _contentRow(
          context,
          author: like.author,
          title: like.title,
          excerpt: like.excerpt ?? '',
          thumbnail: like.thumbnailUrl ?? '',
          time: like.likedAt,
          route: '/p/${like.topicId}',
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
        return _contentRow(
          context,
          author: bookmark.author,
          title: bookmark.title,
          excerpt: bookmark.excerpt ?? '',
          thumbnail: bookmark.thumbnailUrl ?? '',
          time: bookmark.bookmarkedAt,
          route: bookmark.postNo != null && bookmark.postNo! > 0
              ? '/p/${bookmark.topicId}?postNo=${bookmark.postNo}'
              : '/p/${bookmark.topicId}',
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
        return GfConnectionRow(
          avatarUrl: resolveApiAssetUrl(user.avatarUrl),
          name: privateDisplayName(
            context,
            user.id,
            user.username,
            user.nickname,
          ),
          username: user.username,
          bio: user.bio,
          action: user.isSelf || user.isFollowing == null
              ? null
              : GfFollowButton(
                  following: following[user.id] ?? user.isFollowing!,
                  label: (following[user.id] ?? user.isFollowing!)
                      ? AppLocalizations.of(context).profileFollowing
                      : AppLocalizations.of(context).profileFollow,
                  busy: busy.contains(user.id),
                  onPressed: () => onFollow(user),
                ),
          onTap: () => context.push('/u/${user.id}'),
        );
      },
    );
  }

  Widget _contentRow(
    BuildContext context, {
    required UserBriefPayload? author,
    required String title,
    required String excerpt,
    required String thumbnail,
    required String time,
    required String route,
  }) => GfContentRow(
    author: author == null
        ? ''
        : privateDisplayName(
            context,
            author.id,
            author.username,
            author.nickname,
          ),
    avatarUrl: resolveApiAssetUrl(author?.avatarUrl ?? ''),
    title: title,
    text: excerpt,
    thumbnailUrl: resolveApiAssetUrl(thumbnail),
    time: timeAgo(time, l10n: AppLocalizations.of(context)),
    onAuthorTap: author == null || author.id <= 0
        ? null
        : () => context.push('/u/${author.id}'),
    onTap: () => context.push(route),
  );

  String? _activityRoute(UserActivityPayload activity) {
    final route = profileActivityRoute(activity.url);
    if (route != null) return route;
    if (activity.subjectType == 'topic' && activity.subjectId > 0) {
      return '/p/${activity.subjectId}';
    }
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

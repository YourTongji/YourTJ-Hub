import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../current_user.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';

/// 用户主页(web ProfilePage.vue 的移动端形态):用户卡片 + 活动/主题/点赞/收藏 tab。
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

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      // 未传 userId 时展示当前登录用户;未登录时提示登录。
      final int? currentId = (await ref.read(currentUserProvider.future))?.id;
      final int uid = widget.userId ?? currentId ?? 0;
      if (uid == 0) {
        setState(
          () => _page = AsyncValue.error(
            AppLocalizations.of(context).profileNotLoggedIn,
            StackTrace.current,
          ),
        );
        return;
      }
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .userProfile(uid);
      final UserProfileProps? props = parsePageProps<UserProfileProps>(payload);
      setState(() {
        _page = props == null
            ? AsyncValue.error(
                AppLocalizations.of(context).commonParseFailed,
                StackTrace.current,
              )
            : AsyncValue.data(props);
        _following = props?.user.isFollowing ?? false;
      });
    } catch (e, st) {
      setState(() => _page = AsyncValue.error(e, st));
    }
  }

  Future<void> _toggleFollow(UserCardPayload user) async {
    if (user.isSelf) return;
    final bool target = !_following;
    setState(() => _following = target);
    try {
      await ref
          .read(topicRepositoryProvider)
          .followUser(userId: user.userId, isFollowing: _following);
    } catch (_) {
      // 回滚。
      if (mounted) setState(() => _following = !target);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.profileTitle)),
      body: Column(
        children: [
          Expanded(
            child: _page.when(
              loading: () => const GfLoading(),
              error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
              data: (props) {
                return Column(
                  children: [
                    _ProfileHeader(
                      user: props.user,
                      badges: props.badges,
                      following: _following,
                      onFollow: () => _toggleFollow(props.user),
                    ),
                    _ProfileTabs(
                      props: props,
                      index: _tabIndex,
                      onChanged: (i) => setState(() => _tabIndex = i),
                    ),
                    const Divider(height: 1),
                    Expanded(
                      child: RefreshIndicator(
                        onRefresh: () => _load(silent: true),
                        child: _ProfileBody(props: props, index: _tabIndex),
                      ),
                    ),
                  ],
                );
              },
            ),
          ),
          // 我的入口:设置 / 通知 / 草稿(可达性修复,任何状态可见)。
          SafeArea(
            top: false,
            child: Container(
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surface,
                border: Border(
                  top: BorderSide(
                    color: Theme.of(context).dividerColor,
                    width: 0.5,
                  ),
                ),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  TextButton.icon(
                    icon: const Icon(Icons.settings_outlined, size: 18),
                    label: Text(l10n.settingsTitle),
                    onPressed: () => context.go('/settings'),
                  ),
                  TextButton.icon(
                    icon: const Icon(Icons.notifications_outlined, size: 18),
                    label: Text(l10n.notificationsTitle),
                    onPressed: () => context.go('/notifications'),
                  ),
                  TextButton.icon(
                    icon: const Icon(Icons.description_outlined, size: 18),
                    label: Text(l10n.draftsTitle),
                    onPressed: () => context.go('/drafts'),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ProfileHeader extends StatelessWidget {
  const _ProfileHeader({
    required this.user,
    required this.badges,
    required this.following,
    required this.onFollow,
  });

  final UserCardPayload user;
  final List<UserBadgePayload> badges;
  final bool following;
  final VoidCallback onFollow;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          CircleAvatar(
            radius: 30,
            backgroundImage: NetworkImage(user.avatarUrl),
            onBackgroundImageError: (_, _) {},
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        user.nickname.isEmpty ? user.username : user.nickname,
                        style: const TextStyle(
                          fontSize: 17,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    if (!user.isSelf && !user.isAdmin)
                      GfButton(
                        label: following
                            ? l10n.profileFollowing
                            : l10n.profileFollow,
                        variant: following
                            ? GfButtonVariant.outline
                            : GfButtonVariant.primary,
                        size: GfButtonSize.small,
                        onPressed: onFollow,
                      ),
                  ],
                ),
                if (user.nickname.isNotEmpty && user.nickname != user.username)
                  Text(
                    '@${user.username}',
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                if (user.bio.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text(user.bio, style: const TextStyle(fontSize: 13)),
                ],
                const SizedBox(height: 8),
                Row(
                  children: [
                    _StatItem(
                      label: l10n.profileTopics,
                      value: user.topicCount,
                    ),
                    _StatItem(
                      label: l10n.profileReplies,
                      value: user.replyCount,
                    ),
                    _StatItem(
                      label: l10n.profileLikes,
                      value: user.likeReceivedCount,
                    ),
                    _StatItem(
                      label: l10n.profileFollowers,
                      value: user.followerCount,
                    ),
                    _StatItem(
                      label: l10n.profileFollowingCount,
                      value: user.followingCount,
                    ),
                  ],
                ),
                // 徽章展示区(对齐 web profile badges,数据来自 UserProfileProps)。
                if (badges.isNotEmpty) ...[
                  const SizedBox(height: 10),
                  Wrap(
                    spacing: 6,
                    runSpacing: 6,
                    children: [
                      for (final b in badges)
                        GfChip(label: b.name, color: colorFromHex(b.color)),
                    ],
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _StatItem extends StatelessWidget {
  const _StatItem({required this.label, required this.value});

  final String label;
  final int value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(right: 14),
      child: Column(
        children: [
          Text(
            formatNumber(value),
            style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
          ),
          Text(label, style: const TextStyle(fontSize: 11, color: Colors.grey)),
        ],
      ),
    );
  }
}

class _ProfileTabs extends StatelessWidget {
  const _ProfileTabs({
    required this.props,
    required this.index,
    required this.onChanged,
  });

  final UserProfileProps props;
  final int index;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 42,
      alignment: Alignment.centerLeft,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: GfTabBar(
        tabs: <GfTab>[
          for (int i = 0; i < props.activityTabs.length; i++)
            GfTab(
              label: props.activityTabs[i].label ?? props.activityTabs[i].key,
              value: i,
            ),
        ],
        selected: index,
        onSelected: (Object value) => onChanged(value as int),
      ),
    );
  }
}

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({required this.props, required this.index});

  final UserProfileProps props;
  final int index;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final List<Widget> contents = <Widget>[
      if (props.activities.isEmpty)
        GfEmpty(message: l10n.profileEmptyActivity)
      else
        ListView.builder(
          itemCount: props.activities.length,
          itemBuilder: (context, i) => ListTile(
            leading: const Icon(Icons.bolt_outlined, size: 18),
            title: Text(
              props.activities[i].contentPreview,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: Text(
              timeAgo(props.activities[i].createdAt, l10n: l10n),
              style: const TextStyle(fontSize: 11, color: Colors.grey),
            ),
          ),
        ),
      if (props.topics.isEmpty)
        GfEmpty(message: l10n.profileEmptyTopics)
      else
        ListView.builder(
          itemCount: props.topics.length,
          itemBuilder: (context, i) => ListTile(
            title: Text(
              props.topics[i].title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: Text(
              l10n.topicReplies(props.topics[i].replyCount),
              style: const TextStyle(fontSize: 11, color: Colors.grey),
            ),
          ),
        ),
      if (props.likes.isEmpty)
        GfEmpty(message: l10n.profileEmptyLikes)
      else
        ListView.builder(
          itemCount: props.likes.length,
          itemBuilder: (context, i) => ListTile(
            title: Text(
              props.likes[i].title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: Text(
              timeAgo(props.likes[i].likedAt, l10n: l10n),
              style: const TextStyle(fontSize: 11, color: Colors.grey),
            ),
          ),
        ),
      if (props.bookmarks.isEmpty)
        GfEmpty(message: l10n.profileEmptyBookmarks)
      else
        ListView.builder(
          itemCount: props.bookmarks.length,
          itemBuilder: (context, i) => ListTile(
            title: Text(
              props.bookmarks[i].title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: Text(
              timeAgo(props.bookmarks[i].bookmarkedAt, l10n: l10n),
              style: const TextStyle(fontSize: 11, color: Colors.grey),
            ),
          ),
        ),
      if (props.following.isEmpty)
        GfEmpty(message: l10n.profileEmptyFollowing)
      else
        ListView.builder(
          itemCount: props.following.length,
          itemBuilder: (context, i) => ListTile(
            leading: CircleAvatar(
              radius: 14,
              backgroundImage: NetworkImage(props.following[i].avatarUrl),
            ),
            title: Text(
              props.following[i].nickname.isEmpty
                  ? props.following[i].username
                  : props.following[i].nickname,
            ),
          ),
        ),
      if (props.followers.isEmpty)
        GfEmpty(message: l10n.profileEmptyFollowers)
      else
        ListView.builder(
          itemCount: props.followers.length,
          itemBuilder: (context, i) => ListTile(
            leading: CircleAvatar(
              radius: 14,
              backgroundImage: NetworkImage(props.followers[i].avatarUrl),
            ),
            title: Text(
              props.followers[i].nickname.isEmpty
                  ? props.followers[i].username
                  : props.followers[i].nickname,
            ),
          ),
        ),
    ];

    final int idx = index.clamp(0, contents.length - 1);
    return contents[idx];
  }
}

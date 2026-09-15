import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../current_user.dart';
import '../format.dart';
import '../providers.dart';

final accountLayoutProvider = FutureProvider.autoDispose<LayoutPayload>((
  ref,
) async {
  ref.watch(currentUserProvider);
  return (await ref.watch(pageRepositoryProvider).home()).layout;
});

// Keyed by the server viewer ID; a session change discards cached identity.
final accountCardProvider = FutureProvider.autoDispose
    .family<UserCardPayload, int>((ref, id) {
      ref.watch(currentUserProvider);
      return ref.watch(userRepositoryProvider).getUserCard(id);
    });

class AccountAvatar extends ConsumerWidget {
  const AccountAvatar({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final layout = ref.watch(accountLayoutProvider);
    final viewer = layout.isLoading ? null : layout.asData?.value.viewer;
    return GfAvatar(src: resolveApiAssetUrl(viewer?.avatarUrl ?? ''), size: 30);
  }
}

class AccountDrawer extends ConsumerWidget {
  const AccountDrawer({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final layout = ref.watch(accountLayoutProvider);
    final viewer = layout.isLoading ? null : layout.asData?.value.viewer;
    final signedIn = viewer?.isAuthenticated == true;
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final card = signedIn ? ref.watch(accountCardProvider(viewer!.id)) : null;
    final user = card?.isLoading == true ? null : card?.asData?.value;
    void open(String path) {
      Navigator.of(context).pop();
      context.push(path);
    }

    Widget connection(String label, int? count, String stream) => TextButton(
      style: TextButton.styleFrom(
        foregroundColor: colors.baseContent,
        padding: EdgeInsets.zero,
        minimumSize: const Size(0, 44),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      ),
      onPressed: () => open('/profile?stream=$stream'),
      child: Text.rich(
        TextSpan(
          children: [
            TextSpan(
              text: count == null ? '—' : formatNumber(count),
              style: const TextStyle(fontWeight: FontWeight.w700),
            ),
            TextSpan(
              text: ' $label',
              style: TextStyle(color: colors.iconMuted),
            ),
          ],
        ),
        style: type.small.copyWith(fontSize: 15),
      ),
    );
    Widget entry(String icon, String title, String path) => ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 32, vertical: 6),
      minLeadingWidth: 28,
      horizontalTitleGap: 24,
      leading: GfSymbol(icon, size: 28, color: colors.baseContent),
      title: Text(title, style: type.bodyStrong.copyWith(fontSize: 20)),
      onTap: () => open(path),
    );
    return Drawer(
      width: (MediaQuery.sizeOf(context).width * .84).clamp(0.0, 400.0),
      backgroundColor: GfTheme.colorsOf(context).base100,
      shape: const RoundedRectangleBorder(),
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.symmetric(vertical: 16),
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(32, 12, 28, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  InkWell(
                    borderRadius: BorderRadius.circular(30),
                    onTap: () => open(signedIn ? '/profile' : '/login'),
                    child: GfAvatar(
                      src: resolveApiAssetUrl(
                        user?.avatarUrl ?? viewer?.avatarUrl ?? '',
                      ),
                      size: 56,
                    ),
                  ),
                  const SizedBox(height: 16),
                  Text(
                    signedIn
                        ? (user?.nickname.isNotEmpty == true
                              ? user!.nickname
                              : viewer!.username)
                        : 'YourTJ',
                    style: type.title1,
                  ),
                  if (signedIn)
                    Text(
                      '@${viewer!.username}',
                      style: type.body.copyWith(color: colors.iconMuted),
                    ),
                  if (signedIn) ...[
                    const SizedBox(height: 12),
                    Wrap(
                      spacing: 18,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        connection(
                          l10n.accountFollowing,
                          user?.followingCount,
                          'following',
                        ),
                        connection(
                          l10n.accountFollowers,
                          user?.followerCount,
                          'followers',
                        ),
                        if (card?.hasError == true)
                          IconButton(
                            tooltip: l10n.commonRetry,
                            onPressed: () =>
                                ref.invalidate(accountCardProvider(viewer!.id)),
                            icon: const GfSymbol('refresh-cw', size: 18),
                          ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            if (!signedIn) entry('user-round', l10n.loginModeLogin, '/login'),
            if (signedIn) ...[
              entry('user-round', l10n.settingsSectionProfile, '/profile'),
              entry(
                'bookmark',
                l10n.profileBookmarks,
                '/profile?stream=bookmarks',
              ),
              entry('file-text', l10n.draftsTitle, '/drafts'),
              entry('archive', l10n.accountContent, '/my-content'),
              entry('trash-2', l10n.profileTrash, '/recycle-bin'),
              entry(
                'graduation-cap',
                l10n.myCourseReviewsTitle,
                '/my-course-reviews',
              ),
              if (viewer!.canAccessAdmin)
                entry('shield-check', l10n.profileAdmin, '/admin'),
              if (viewer.isModerator || viewer.canAccessAdmin)
                entry('shield-check', l10n.profileModeration, '/moderation'),
              if (viewer.canManageCourses) ...[
                entry(
                  'graduation-cap',
                  l10n.coursesManagement,
                  '/moderation/courses',
                ),
                entry(
                  'shield-check',
                  l10n.coursesReviewModeration,
                  '/moderation/course-reviews',
                ),
              ],
            ],
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 16, horizontal: 32),
              child: Divider(),
            ),
            entry('settings', l10n.settingsTitle, '/settings'),
            entry('book-open', l10n.siteInfoTitle, '/about'),
          ],
        ),
      ),
    );
  }
}

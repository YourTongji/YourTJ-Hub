import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../current_user.dart';
import '../providers.dart';

final accountLayoutProvider = FutureProvider.autoDispose<LayoutPayload>((
  ref,
) async {
  ref.watch(currentUserProvider);
  return (await ref.watch(pageRepositoryProvider).home()).layout;
});

class AccountAvatar extends ConsumerWidget {
  const AccountAvatar({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final viewer = ref.watch(accountLayoutProvider).valueOrNull?.viewer;
    return GfAvatar(src: resolveApiAssetUrl(viewer?.avatarUrl ?? ''), size: 30);
  }
}

class AccountDrawer extends ConsumerWidget {
  const AccountDrawer({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final viewer = ref.watch(accountLayoutProvider).valueOrNull?.viewer;
    final signedIn = viewer?.isAuthenticated == true;
    Widget entry(String icon, String title, String path) => ListTile(
      leading: GfSymbol(icon),
      title: Text(title),
      onTap: () {
        Navigator.of(context).pop();
        context.push(path);
      },
    );
    return Drawer(
      width: MediaQuery.sizeOf(context).width * .84,
      backgroundColor: GfTheme.colorsOf(context).base100,
      shape: const RoundedRectangleBorder(),
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.symmetric(vertical: 16),
          children: [
            Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  GfAvatar(
                    src: resolveApiAssetUrl(viewer?.avatarUrl ?? ''),
                    size: 52,
                  ),
                  const SizedBox(height: 16),
                  Text(
                    signedIn ? viewer!.username : 'YourTJ',
                    style: GfTheme.typographyOf(context).display,
                  ),
                  if (signedIn)
                    Text(
                      '@${viewer!.username}',
                      style: GfTheme.typographyOf(context).small,
                    ),
                ],
              ),
            ),
            if (!signedIn) entry('user-round', l10n.loginModeLogin, '/login'),
            if (signedIn) ...[
              entry('user-round', l10n.navProfile, '/profile'),
              entry(
                'bookmark',
                l10n.profileBookmarks,
                '/profile?stream=bookmarks',
              ),
              ExpansionTile(
                leading: const GfSymbol('archive'),
                title: Text(l10n.profileContent),
                children: [
                  entry('file-text', l10n.draftsTitle, '/drafts'),
                  entry('archive', l10n.profileContent, '/my-content'),
                  entry('trash-2', l10n.profileTrash, '/recycle-bin'),
                ],
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
              padding: EdgeInsets.symmetric(vertical: 16, horizontal: 20),
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

import 'dart:math' as math;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../current_user.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../navigation/tab_swipe_surface.dart';
import '../../widgets/app_refresh_indicator.dart';
import '../../widgets/topic_list.dart';
import '../../widgets/status_views.dart';
import '../settings/anonymous_identity_page.dart';
import 'profile_header_sliver.dart';
import 'profile_tabs.dart';
import 'profile_edit_button.dart';

class AnonymousProfilePage extends ConsumerStatefulWidget {
  const AnonymousProfilePage({super.key, required this.uid});
  final String uid;
  @override
  ConsumerState<AnonymousProfilePage> createState() =>
      _AnonymousProfilePageState();
}

class _AnonymousProfilePageState extends ConsumerState<AnonymousProfilePage> {
  Map<String, dynamic>? data;
  Object? error;
  int page = 1, revision = 0, tab = 0;
  bool loading = false;
  @override
  void initState() {
    super.initState();
    load();
  }

  @override
  void didUpdateWidget(covariant AnonymousProfilePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.uid != widget.uid) {
      data = null;
      tab = 0;
      load();
    }
  }

  Future<void> load({bool append = false}) async {
    final request = ++revision;
    final epoch = ref.read(offlineCacheEpochProvider);
    final nextPage = append ? page + 1 : 1;
    setState(() {
      loading = true;
      error = null;
    });
    try {
      final next = await ref
          .read(apiClientProvider)
          .get<Map<String, dynamic>>(
            '/a/${widget.uid}',
            queryParameters: {'page': nextPage},
            headers: {GfApiClient.pageRequestHeader: 'true'},
            parser: (j) =>
                (j as Map<String, dynamic>)['props'] as Map<String, dynamic>,
          );
      if (!mounted ||
          request != revision ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        if (append && data != null && next['showContent'] != false) {
          for (final key in ['topics', 'replies']) {
            final previous = data![key] as List;
            final ids = previous.map((row) => (row as Map)['id']).toSet();
            next[key] = [
              ...previous,
              ...(next[key] as List).where(
                (row) => !ids.contains((row as Map)['id']),
              ),
            ];
          }
        }
        data = next;
        page = nextPage;
      });
    } catch (e) {
      if (mounted &&
          request == revision &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => error = e);
      }
    } finally {
      if (mounted &&
          request == revision &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => loading = false);
      }
    }
  }

  Future<void> manage() async {
    final epoch = ref.read(offlineCacheEpochProvider);
    await showAnonymousIdentitySheet(context);
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    await load();
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (previous, next) {
      if (previous == next) return;
      setState(() {
        data = null;
        tab = 0;
      });
      load();
    });
    final signedIn = ref.watch(currentUserProvider).valueOrNull != null;
    final ownState = signedIn ? ref.watch(anonymousIdentityProvider) : null;
    final ownPersona = ownState?.valueOrNull?.persona;
    // A signed-in viewer's line waits for the private ownership read, so the
    // owner never briefly sees the wording meant for other members.
    final ownResolved = !signedIn || ownState!.hasValue || ownState.hasError;
    final d = data;
    if (d == null) {
      return Scaffold(
        appBar: GfAppBar(title: Text(l.anonymousIdentity)),
        body: error == null
            ? const Center(child: CircularProgressIndicator())
            : GfErrorRetry(
                message: resolveErrorMessage(l, error!),
                onRetry: load,
              ),
      );
    }
    final publicPersona = AnonymousPersona.fromJson(
      d['persona'] as Map<String, dynamic>,
    );
    final isOwn = ownPersona?.publicUid == widget.uid;
    final persona = isOwn ? ownPersona! : publicPersona;
    final topics = (d['topics'] as List)
        .map((raw) => TopicPayload.fromJson(raw as Map<String, dynamic>))
        .toList();
    final replies = d['replies'] as List;
    final showContent = d['showContent'] != false;
    final tabs = [
      TabItemPayload(
        key: 'topics',
        label: l.profileTopics,
        url: '',
        active: tab == 0,
      ),
      TabItemPayload(
        key: 'replies',
        label: l.profileReplies,
        url: '',
        active: tab == 1,
      ),
    ];
    final hasMore =
        showContent &&
        page * 20 < (d[tab == 0 ? 'topicCount' : 'replyCount'] as num);
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: TabSwipeSurface(
            index: tab,
            length: tabs.length,
            onChanged: (value) => setState(() => tab = value),
            child: AppRefreshIndicator(
              edgeOffset: MediaQuery.paddingOf(context).top + 56,
              onRefresh: load,
              child: CustomScrollView(
                physics: const AlwaysScrollableScrollPhysics(),
                slivers: [
                  ProfileHeaderSliver(
                    avatarUrl: persona.avatarUrl,
                    title: Text(
                      persona.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    actions: isOwn
                        ? ProfileEditButton(
                            label: l.anonymousManage,
                            onPressed: manage,
                          )
                        : null,
                    actionHeightFor: (context, width) => isOwn
                        ? GfUserCardHeader.actionHeightFor(
                            MediaQuery.textScalerOf(context),
                          )
                        : GfUserCardHeader.minimumActionHeight,
                  ),
                  SliverToBoxAdapter(
                    child: GfUserCard(
                      showHeader: false,
                      avatarUrl: resolveApiAssetUrl(persona.avatarUrl),
                      name: persona.name,
                      username: '',
                      showUsername: false,
                      nameBadges: [
                        GfBadge(
                          label: isOwn ? l.anonymousTagOwn : l.anonymousTag,
                          variant: isOwn
                              ? GfBadgeVariant.info
                              : GfBadgeVariant.muted,
                        ),
                      ],
                      bio: !ownResolved
                          ? null
                          : isOwn
                          ? l.anonymousProfileOwnerHint
                          : signedIn
                          ? l.anonymousProfileMemberHint
                          : l.anonymousProfileGuestHint,
                      stats: showContent
                          ? [
                              (
                                l.profileTopics,
                                formatNumber((d['topicCount'] as num).toInt()),
                              ),
                              (
                                l.profileReplies,
                                formatNumber((d['replyCount'] as num).toInt()),
                              ),
                            ]
                          : [],
                      statActions: {
                        0: () => setState(() => tab = 0),
                        1: () => setState(() => tab = 1),
                      },
                    ),
                  ),
                  if (showContent) ...[
                    const SliverToBoxAdapter(child: GfDivider()),
                    SliverPersistentHeader(
                      pinned: true,
                      delegate: ProfileTabsHeader(
                        height: math.max(
                          52,
                          MediaQuery.textScalerOf(context).scale(16) * 1.4 + 24,
                        ),
                        child: ProfileTabs(
                          tabs: tabs,
                          index: tab,
                          onChanged: (value) => setState(() => tab = value),
                        ),
                      ),
                    ),
                    if (tab == 0 && topics.isNotEmpty)
                      SliverList.builder(
                        itemCount: topics.length,
                        itemBuilder: (context, index) => buildTopicFeedCard(
                          context,
                          topics[index],
                          onReturn: load,
                        ),
                      )
                    else if (tab == 1 && replies.isNotEmpty)
                      SliverList.builder(
                        itemCount: replies.length,
                        itemBuilder: (context, index) {
                          final reply = replies[index] as Map;
                          return GfContentRow(
                            author: persona.name,
                            avatarUrl: resolveApiAssetUrl(persona.avatarUrl),
                            title: l.profileReplies,
                            text: reply['excerpt'] as String,
                            time: '',
                            onTap: () => context.push(reply['url'] as String),
                          );
                        },
                      )
                    else
                      SliverToBoxAdapter(
                        child: Padding(
                          padding: const EdgeInsets.symmetric(vertical: 32),
                          child: GfEmpty(
                            message: tab == 0
                                ? l.profileEmptyTopics
                                : l.commonEmpty,
                          ),
                        ),
                      ),
                    SliverToBoxAdapter(
                      child: GfListFooter(
                        progressKey: page * 2 + tab,
                        loading: loading,
                        hasMore: hasMore,
                        error: error == null
                            ? null
                            : resolveErrorMessage(l, error!),
                        onLoadMore: () {
                          if (!loading) load(append: true);
                        },
                      ),
                    ),
                  ] else
                    SliverToBoxAdapter(
                      child: Padding(
                        padding: const EdgeInsets.symmetric(vertical: 32),
                        child: isOwn
                            ? GfEmpty(
                                key: const Key('anonymous-hidden-own'),
                                message: l.anonymousProfileContentHiddenOwn,
                                description: l
                                    .anonymousProfileContentHiddenOwnDescription,
                                action: GfButton(
                                  label: l.anonymousManage,
                                  variant: GfButtonVariant.secondary,
                                  onPressed: manage,
                                ),
                              )
                            : GfEmpty(message: l.anonymousProfileContentHidden),
                      ),
                    ),
                  SliverToBoxAdapter(
                    child: SizedBox(
                      height: MediaQuery.paddingOf(context).bottom + 16,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../local/writing_store.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import '../../widgets/compose_menu.dart';

/// 草稿列表页(web drafts.index 的移动端形态)。
///
/// 数据通道:/drafts 页面级接口(DraftsPageProps,需登录)。
class DraftsPage extends ConsumerStatefulWidget {
  const DraftsPage({super.key});

  @override
  ConsumerState<DraftsPage> createState() => _DraftsPageState();
}

class _DraftsPageState extends ConsumerState<DraftsPage> {
  AsyncValue<DraftsPageProps> _page = const AsyncValue.loading();

  List<LocalDraft> _localDrafts = [];
  String? _owner;
  int _loadSequence = 0;

  Future<void> _loadLocal() async {
    final epoch = ref.read(offlineCacheEpochProvider);
    try {
      final owner = await ref.read(writingScopeProvider.future);
      final drafts = await ref.read(writingStoreProvider).drafts(owner);
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() {
          _owner = owner;
          _localDrafts = drafts;
        });
      }
    } catch (_) {
      /* Cloud drafts remain available. */
    }
  }

  Future<void> _openLocal(LocalDraft draft) async {
    await context.push(
      Uri(
        path: '/publish',
        queryParameters: {
          'local': draft.key,
          'type': '${draft.contentType}',
          if (draft.topicId > 0) 'id': '${draft.topicId}',
        },
      ).toString(),
    );
    if (mounted) _load(silent: true);
  }

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool silent = false}) async {
    final sequence = ++_loadSequence;
    final epoch = ref.read(offlineCacheEpochProvider);
    _loadLocal();
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/drafts');
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final DraftsPageProps? props = parsePageProps<DraftsPageProps>(payload);
      setState(() {
        _page = props == null
            ? AsyncValue.error(
                AppLocalizations.of(context).commonParseFailed,
                StackTrace.current,
              )
            : AsyncValue.data(props);
      });
    } catch (e, st) {
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() => _page = AsyncValue.error(e, st));
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      _loadSequence++;
      setState(() {
        _owner = null;
        _localDrafts = [];
        _page = const AsyncValue.loading();
      });
    });
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.draftsTitle)),
      body: Column(
        children: [
          if (_localDrafts.isNotEmpty)
            Flexible(
              child: ListView(
                shrinkWrap: true,
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
                children: [
                  Text(
                    l10n.draftLocalSection,
                    style: GfTheme.typographyOf(context).title2,
                  ),
                  for (final draft in _localDrafts)
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      leading: const Icon(Icons.edit_note_outlined),
                      title: Text(
                        draft.title.isEmpty
                            ? (draft.content.isEmpty
                                  ? l10n.topicNoTitle
                                  : draft.content)
                            : draft.title,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      subtitle: Text(l10n.draftLocalOnly),
                      onTap: () => _openLocal(draft),
                      trailing: IconButton(
                        tooltip: l10n.draftDeleteLocal,
                        icon: const Icon(Icons.delete_outline),
                        onPressed: () async {
                          final owner = _owner;
                          if (owner == null) return;
                          final confirmed = await showDialog<bool>(
                            context: context,
                            builder: (context) => AlertDialog(
                              title: Text(l10n.draftDeleteLocal),
                              actions: [
                                TextButton(
                                  onPressed: () =>
                                      Navigator.pop(context, false),
                                  child: Text(l10n.commonCancel),
                                ),
                                TextButton(
                                  onPressed: () => Navigator.pop(context, true),
                                  child: Text(l10n.draftDeleteLocal),
                                ),
                              ],
                            ),
                          );
                          if (confirmed != true ||
                              !mounted ||
                              owner != _owner) {
                            return;
                          }
                          try {
                            await ref
                                .read(writingStoreProvider)
                                .delete(owner, draft.key);
                            await _loadLocal();
                          } catch (error) {
                            if (mounted && context.mounted) {
                              showGfToast(
                                context,
                                resolveErrorMessage(l10n, error),
                                error: true,
                              );
                            }
                          }
                        },
                      ),
                    ),
                ],
              ),
            ),
          if (_localDrafts.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text(
                  l10n.draftCloudSection,
                  style: GfTheme.typographyOf(context).title2,
                ),
              ),
            ),
          Expanded(
            child: _page.when(
              loading: () => const GfLoading(),
              error: (e, _) => GfErrorRetry(
                message: resolveErrorMessage(l10n, e),
                onRetry: _load,
              ),
              data: (props) => GfScrollToTop(
                semanticLabel: l10n.commonBackToTop,
                threshold: 360,
                builder: (BuildContext context, ScrollController controller) {
                  return AppRefreshIndicator(
                    onRefresh: () => _load(silent: true),
                    child: props.drafts.isEmpty
                        ? CustomScrollView(
                            controller: controller,
                            physics: const AlwaysScrollableScrollPhysics(),
                            slivers: <Widget>[
                              SliverFillRemaining(
                                hasScrollBody: false,
                                child: GfEmpty(
                                  icon: Icons.edit_note_outlined,
                                  message: l10n.draftsEmpty,
                                  description: l10n.draftsEmptyDescription,
                                  action: GfButton(
                                    icon: const GfSymbol('plus', size: 20),
                                    label: l10n.navPublish,
                                    onPressed: () =>
                                        showComposeMenu(context, bottom: 16),
                                  ),
                                ),
                              ),
                            ],
                          )
                        : ListView.separated(
                            controller: controller,
                            physics: const AlwaysScrollableScrollPhysics(),
                            itemCount: props.drafts.length,
                            separatorBuilder: (_, _) => const GfDivider(),
                            itemBuilder: (context, i) {
                              final draft = props.drafts[i];
                              final meta = [
                                l10n.draftsMetaCreated(
                                  formatDateTime(draft.createdAt),
                                ),
                                l10n.draftsMetaViews(draft.viewCount),
                                l10n.draftsMetaReplies(draft.replyCount),
                              ].join(' · ');
                              return GfDraftRow(
                                editLabel: l10n.commonEdit,
                                blockedLabel: l10n.draftsBlocked,
                                title: draft.title.isEmpty
                                    ? l10n.topicNoTitle
                                    : draft.title,
                                description: draft.description,
                                categories: <GfTopicCategory>[
                                  for (final cat in draft.categories)
                                    GfTopicCategory(
                                      name: cat.name,
                                      color: colorFromHex(cat.color),
                                    ),
                                ],
                                blocked: draft.processStatus == 1,
                                meta: meta,
                                updatedTime: formatDate(draft.updatedAt),
                                onTap: () =>
                                    context.push('/publish?id=${draft.id}'),
                              );
                            },
                          ),
                  );
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}

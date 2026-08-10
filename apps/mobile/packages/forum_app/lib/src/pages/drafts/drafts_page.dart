import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';

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

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/drafts');
      if (!mounted) return;
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
      if (!mounted) return;
      setState(() => _page = AsyncValue.error(e, st));
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.draftsTitle)),
      body: _page.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (props) => GfScrollToTop(
          semanticLabel: l10n.commonBackToTop,
          threshold: 360,
          builder: (BuildContext context, ScrollController controller) {
            return RefreshIndicator(
              onRefresh: () => _load(silent: true),
              child: props.drafts.isEmpty
                  ? CustomScrollView(
                      controller: controller,
                      physics: const AlwaysScrollableScrollPhysics(),
                      slivers: <Widget>[
                        SliverFillRemaining(
                          hasScrollBody: false,
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: <Widget>[
                              GfEmpty(
                                icon: Icons.edit_note_outlined,
                                message: l10n.draftsEmpty,
                              ),
                              GfButton(
                                icon: const Icon(Icons.add_rounded, size: 18),
                                label: l10n.navPublish,
                                onPressed: () => context.push('/publish'),
                              ),
                              const SizedBox(height: 48),
                            ],
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
                          onTap: () => context.push('/publish?id=${draft.id}'),
                        );
                      },
                    ),
            );
          },
        ),
      ),
    );
  }
}

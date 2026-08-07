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
      setState(() => _page = AsyncValue.error(e, st));
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.draftsTitle)),
      body: _page.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (props) {
          if (props.drafts.isEmpty) return GfEmpty(message: l10n.draftsEmpty);
          return RefreshIndicator(
            onRefresh: () => _load(silent: true),
            child: ListView.separated(
              itemCount: props.drafts.length,
              separatorBuilder: (_, _) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final draft = props.drafts[i];
                final colors = GfTheme.colorsOf(context);
                return InkWell(
                  onTap: () {
                    // 跳转编辑:web 草稿 editUrl 形如 /publish?topicId=xxx。
                    context.push('/publish?topicId=${draft.id}');
                  },
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 12,
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Expanded(
                              child: Text(
                                draft.title.isEmpty
                                    ? l10n.topicNoTitle
                                    : draft.title,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: GfTheme.typographyOf(context).heading,
                              ),
                            ),
                            // blocked badge(web processStatus===1)。
                            if (draft.processStatus == 1) ...[
                              const SizedBox(width: 8),
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 8,
                                  vertical: 2,
                                ),
                                decoration: BoxDecoration(
                                  color: colors.error.withValues(alpha: 0.1),
                                  borderRadius: BorderRadius.circular(4),
                                ),
                                child: Text(
                                  l10n.draftsBlocked,
                                  style: GfTheme.typographyOf(context).meta.copyWith(color: colors.error, fontWeight: FontWeight.w600),
                                ),
                              ),
                            ],
                          ],
                        ),
                        if (draft.description.isNotEmpty) ...[
                          const SizedBox(height: 4),
                          Text(
                            draft.description,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: GfTheme.typographyOf(context).small,
                          ),
                        ],
                        const SizedBox(height: 6),
                        Wrap(
                          spacing: 8,
                          runSpacing: 4,
                          crossAxisAlignment: WrapCrossAlignment.center,
                          children: [
                            for (final cat in draft.categories)
                              GfChip(
                                label: cat.name,
                                color: colorFromHex(cat.color),
                              ),
                            Text(
                              l10n.draftsMetaCreated(
                                formatDateTime(draft.createdAt),
                              ),
                              style: GfTheme.typographyOf(context).meta,
                            ),
                            Text(
                              l10n.draftsMetaViews(draft.viewCount),
                              style: GfTheme.typographyOf(context).meta,
                            ),
                            Text(
                              l10n.draftsMetaReplies(draft.replyCount),
                              style: GfTheme.typographyOf(context).meta,
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                );
              },
            ),
          );
        },
      ),
    );
  }
}

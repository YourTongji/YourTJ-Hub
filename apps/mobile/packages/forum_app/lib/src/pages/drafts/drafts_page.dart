import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
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

  Future<void> _load() async {
    setState(() => _page = const AsyncValue.loading());
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
          return ListView.separated(
            itemCount: props.drafts.length,
            separatorBuilder: (_, _) => const Divider(height: 1),
            itemBuilder: (context, i) {
              final draft = props.drafts[i];
              return ListTile(
                leading: const Icon(Icons.description_outlined, size: 20),
                title: Text(
                  draft.title.isEmpty ? l10n.topicNoTitle : draft.title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  draft.description,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: () {
                  // 跳转编辑:web 草稿 editUrl 形如 /publish?topicId=xxx。
                  context.push('/publish?topicId=${draft.id}');
                },
              );
            },
          );
        },
      ),
    );
  }
}

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import 'wiki_page.dart' show encodeWikiPath;

/// wiki 首页(web WikiHome.vue 的移动端形态):
/// 命名空间概览卡片 + 最近更新列表,均以 wiki 路径推入详情页。
class WikiHomePage extends ConsumerStatefulWidget {
  const WikiHomePage({super.key});

  @override
  ConsumerState<WikiHomePage> createState() => _WikiHomePageState();
}

class _WikiHomePageState extends ConsumerState<WikiHomePage> {
  AsyncValue<WikiHomeData> _data = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _data = const AsyncValue.loading());
    try {
      final WikiHomeData data = await ref.read(wikiRepositoryProvider).home();
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _data = AsyncValue.data(data));
    } catch (e, st) {
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _data = AsyncValue.error(e, st));
    }
  }

  void _openPage(String path) {
    context.push('/wiki/${encodeWikiPath(path)}');
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.wikiTitle)),
      body: _data.when(
        loading: () => const _WikiHomeSkeleton(),
        error: (Object e, StackTrace _) =>
            GfErrorRetry(message: resolveErrorMessage(l10n, e), onRetry: _load),
        data: (WikiHomeData data) {
          if (data.namespaces.isEmpty && data.recent.isEmpty) {
            return GfEmpty(message: l10n.commonEmpty);
          }
          final List<Widget> children = <Widget>[
            Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.wikiExploreTitle,
                    style: GfTheme.typographyOf(context).display,
                  ),
                  const SizedBox(height: 16),
                  InkWell(
                    onTap: () => context.push('/wiki/search'),
                    child: Container(
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(
                        color: GfTheme.colorsOf(context).base200,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Row(
                        children: [
                          const GfSymbol('search', size: 20),
                          const SizedBox(width: 12),
                          Text(l10n.wikiSearchHint),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ];
          if (data.namespaces.isNotEmpty) {
            children.add(_SectionLabel(l10n.wikiNamespaces));
            for (int i = 0; i < data.namespaces.length; i++) {
              final WikiNamespace namespace = data.namespaces[i];
              children.add(
                _NamespaceCard(
                  namespace: namespace,
                  showDivider: i < data.namespaces.length - 1,
                  onTap: namespace.firstPagePath.isEmpty
                      ? null
                      : () => _openPage(namespace.firstPagePath),
                ),
              );
            }
          }
          if (data.recent.isNotEmpty) {
            children.add(_SectionLabel(l10n.wikiRecent));
            for (int i = 0; i < data.recent.length; i++) {
              final WikiRecentPage page = data.recent[i];
              children.add(
                _RecentRow(
                  page: page,
                  showDivider: i < data.recent.length - 1,
                  onTap: page.path.isEmpty ? null : () => _openPage(page.path),
                ),
              );
            }
          }
          return ListView(
            padding: const EdgeInsets.only(bottom: 24),
            children: children,
          );
        },
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 20, 16, 8),
      child: Text(
        text,
        style: GfTheme.typographyOf(context).small.copyWith(
          color: colors.baseContent.withValues(alpha: 0.55),
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _NamespaceCard extends StatelessWidget {
  const _NamespaceCard({
    required this.namespace,
    required this.onTap,
    required this.showDivider,
  });

  final WikiNamespace namespace;
  final VoidCallback? onTap;
  final bool showDivider;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);
    final Color muted = colors.baseContent.withValues(alpha: 0.55);
    final Color faint = colors.baseContent.withValues(alpha: 0.45);
    return GfCard(
      onTap: onTap,
      showDivider: showDivider,
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            children: <Widget>[
              Expanded(
                child: Text(
                  namespace.name,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: typography.heading,
                ),
              ),
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: colors.base200,
                  borderRadius: BorderRadius.circular(
                    GfTheme.radiiOf(context).selector,
                  ),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: <Widget>[
                    Icon(Icons.description_outlined, size: 12, color: muted),
                    const SizedBox(width: 3),
                    Text(
                      '${namespace.pageCount}',
                      style: typography.caption.copyWith(
                        color: muted,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          if (namespace.description.isNotEmpty) ...<Widget>[
            const SizedBox(height: 6),
            Text(
              namespace.description,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: typography.small.copyWith(color: muted),
            ),
          ],
          if (namespace.updatedAt.isNotEmpty) ...<Widget>[
            const SizedBox(height: 8),
            Row(
              children: <Widget>[
                Icon(Icons.schedule, size: 12, color: faint),
                const SizedBox(width: 4),
                Text(
                  formatDateTime(namespace.updatedAt),
                  style: typography.caption.copyWith(color: faint),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _RecentRow extends StatelessWidget {
  const _RecentRow({
    required this.page,
    required this.onTap,
    required this.showDivider,
  });

  final WikiRecentPage page;
  final VoidCallback? onTap;
  final bool showDivider;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);
    final Color muted = colors.baseContent.withValues(alpha: 0.55);
    final Color faint = colors.baseContent.withValues(alpha: 0.45);
    return GfCard(
      onTap: onTap,
      showDivider: showDivider,
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(
            page.title,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: typography.bodyStrong,
          ),
          const SizedBox(height: 4),
          Row(
            children: <Widget>[
              Expanded(
                child: Text(
                  page.path,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: typography.small.copyWith(color: muted),
                ),
              ),
              if (page.updatedAt.isNotEmpty) ...<Widget>[
                const SizedBox(width: 8),
                Text(
                  timeAgo(page.updatedAt, l10n: AppLocalizations.of(context)),
                  style: typography.caption.copyWith(color: faint),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }
}

class _WikiHomeSkeleton extends StatelessWidget {
  const _WikiHomeSkeleton();

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: ListView(
        padding: const EdgeInsets.symmetric(vertical: 16),
        children: const <Widget>[
          _SkeletonSection(),
          _NamespaceCardSkeleton(),
          _NamespaceCardSkeleton(),
          _NamespaceCardSkeleton(),
          _SkeletonSection(),
          _RecentRowSkeleton(),
          _RecentRowSkeleton(),
          _RecentRowSkeleton(),
        ],
      ),
    );
  }
}

class _SkeletonSection extends StatelessWidget {
  const _SkeletonSection();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.fromLTRB(16, 20, 16, 12),
      child: Align(
        alignment: Alignment.centerLeft,
        child: GfSkeleton(width: 96, height: 14, radius: 5),
      ),
    );
  }
}

class _NamespaceCardSkeleton extends StatelessWidget {
  const _NamespaceCardSkeleton();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.fromLTRB(16, 6, 16, 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          GfSkeleton(width: 148, height: 17, radius: 5),
          SizedBox(height: 8),
          GfSkeleton(height: 13, radius: 5),
          SizedBox(height: 8),
          GfSkeleton(width: 180, height: 12, radius: 5),
        ],
      ),
    );
  }
}

class _RecentRowSkeleton extends StatelessWidget {
  const _RecentRowSkeleton();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.fromLTRB(16, 8, 16, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          GfSkeleton(width: 220, height: 16, radius: 5),
          SizedBox(height: 8),
          GfSkeleton(width: 150, height: 12, radius: 5),
        ],
      ),
    );
  }
}

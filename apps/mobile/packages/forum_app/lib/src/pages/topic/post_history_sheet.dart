import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/status_views.dart';

class PostHistorySheet extends ConsumerStatefulWidget {
  const PostHistorySheet({super.key, required this.postId});
  final int postId;
  @override
  ConsumerState<PostHistorySheet> createState() => _PostHistorySheetState();
}

class _PostHistorySheetState extends ConsumerState<PostHistorySheet> {
  final _versions = <PostRevision>[];
  bool _loading = false;
  bool _hasMore = true;
  int _before = 0;
  String? _error;
  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    if (_loading || !_hasMore) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final page = await ref
          .read(postRepositoryProvider)
          .revisions(postId: widget.postId, beforeVersion: _before);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (page.postId != widget.postId) {
        throw StateError('Unexpected post history');
      }
      setState(() {
        final seen = _versions.map((v) => v.version).toSet();
        _versions.addAll(page.versions.where((v) => seen.add(v.version)));
        _versions.sort((a, b) => b.version.compareTo(a.version));
        _hasMore =
            page.hasMore &&
            page.beforeVersion > 0 &&
            (_before == 0 || page.beforeVersion < _before);
        _before = page.beforeVersion;
      });
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (_, _) {
      if (mounted) Navigator.pop(context);
    });
    return SafeArea(
      child: SizedBox(
        height: MediaQuery.sizeOf(context).height * .8,
        child: Column(
          children: [
            ListTile(
              title: Text(
                l10n.topicHistory,
                style: GfTheme.typographyOf(context).title3,
              ),
              trailing: IconButton(
                tooltip: l10n.commonClose,
                icon: const Icon(Icons.close),
                onPressed: () => Navigator.pop(context),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  for (final version in _versions)
                    ExpansionTile(
                      key: ValueKey(version.version),
                      title: Text(
                        'v${version.version} · ${version.editor.nickname ?? version.editor.username}',
                      ),
                      subtitle: Text(version.createdAt),
                      childrenPadding: const EdgeInsets.all(12),
                      children: [
                        version.content.isEmpty
                            ? Text(l10n.topicHistoryUnavailable)
                            : GfMarkdownView(
                                data: version.content,
                                selectable: true,
                              ),
                      ],
                    ),
                  if (_error != null)
                    GfErrorRetry(message: _error!, onRetry: _load),
                  if (_loading)
                    const Padding(
                      padding: EdgeInsets.all(16),
                      child: Center(child: CircularProgressIndicator()),
                    ),
                  if (!_loading && _error == null && _versions.isEmpty)
                    GfEmpty(message: l10n.topicHistoryEmpty),
                  if (!_loading && _error == null && _hasMore)
                    TextButton(
                      onPressed: _load,
                      child: Text(l10n.commonLoadMore),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

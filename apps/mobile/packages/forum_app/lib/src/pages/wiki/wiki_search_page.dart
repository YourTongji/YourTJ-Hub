import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import 'wiki_page.dart';

class WikiSearchPage extends ConsumerStatefulWidget {
  const WikiSearchPage({super.key});
  @override
  ConsumerState<WikiSearchPage> createState() => _WikiSearchPageState();
}

class _WikiSearchPageState extends ConsumerState<WikiSearchPage> {
  final _query = TextEditingController();
  Timer? _debounce;
  int _request = 0;
  AsyncValue<WikiSearchResult>? _result;
  @override
  void dispose() {
    _debounce?.cancel();
    _query.dispose();
    super.dispose();
  }

  void _changed(String value) {
    _debounce?.cancel();
    _request++;
    if (value.trim().isEmpty) {
      setState(() => _result = null);
      return;
    }
    setState(() => _result = const AsyncValue.loading());
    _debounce = Timer(const Duration(milliseconds: 300), _search);
  }

  Future<void> _search() async {
    _debounce?.cancel();
    final request = ++_request;
    if (_query.text.trim().isEmpty) {
      setState(() => _result = null);
      return;
    }
    try {
      final result = await ref
          .read(wikiRepositoryProvider)
          .search(_query.text.trim());
      if (mounted && request == _request) {
        setState(() => _result = AsyncValue.data(result));
      }
    } catch (e, st) {
      if (mounted && request == _request) {
        setState(() => _result = AsyncValue.error(e, st));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final type = GfTheme.typographyOf(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.wikiTitle)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: GfSearchField(
              controller: _query,
              hintText: l10n.commonSearch,
              clearLabel: l10n.courseCopyClearSearch,
              autofocus: true,
              maxLength: 100,
              onChanged: _changed,
              onSubmitted: (_) => _search(),
            ),
          ),
          Expanded(
            child:
                _result?.when(
                  loading: () => const GfLoading(),
                  error: (e, _) => GfErrorRetry(
                    message: resolveErrorMessage(l10n, e),
                    onRetry: _search,
                  ),
                  data: (result) => result.searchUnavailable
                      ? GfErrorRetry(
                          message: l10n.wikiSearchUnavailable,
                          onRetry: _search,
                        )
                      : result.items.isEmpty
                      ? GfEmpty(message: l10n.commonEmpty)
                      : ListView.separated(
                          itemCount: result.items.length,
                          separatorBuilder: (_, _) => const Divider(),
                          itemBuilder: (_, index) {
                            final item = result.items[index];
                            return ListTile(
                              contentPadding: const EdgeInsets.symmetric(
                                horizontal: 16,
                                vertical: 12,
                              ),
                              title: Text(item.title, style: type.heading),
                              subtitle: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    '${item.namespace} · ${item.heading}',
                                    style: type.caption,
                                  ),
                                  const SizedBox(height: 8),
                                  Text(
                                    item.snippet.replaceAll(
                                      RegExp('<[^>]*>'),
                                      '',
                                    ),
                                    style: type.body,
                                  ),
                                ],
                              ),
                              onTap: () => context.push(
                                '/wiki/${encodeWikiPath(item.path)}${item.anchors.isEmpty ? '' : '#${Uri.encodeComponent(item.anchors.first)}'}',
                              ),
                            );
                          },
                        ),
                ) ??
                GfEmpty(message: l10n.wikiSearchHint),
          ),
        ],
      ),
    );
  }
}

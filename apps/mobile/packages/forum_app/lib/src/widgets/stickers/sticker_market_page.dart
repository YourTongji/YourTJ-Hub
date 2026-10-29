import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../asset_url.dart';
import '../../navigation/reading_window.dart';
import 'sticker_image.dart';
import 'sticker_library_state.dart';
import 'sticker_peek.dart';
import 'sticker_preview.dart';
import 'sticker_strings.dart';

/// Opens the market above the current route; packs added there appear in the
/// picker on every signed-in device.
Future<void> openStickerMarket(BuildContext context) =>
    Navigator.of(context, rootNavigator: true).push(
      MaterialPageRoute<void>(
        builder: (_) => ReadingPageWindow(child: const StickerMarketPage()),
      ),
    );

/// Adds or removes [pack] and reports the outcome; returns whether it changed.
Future<bool> toggleStickerPack(
  BuildContext context,
  StickerCollection collection,
  StickerMarketPackPayload pack,
) async {
  final strings = StickerStrings(context);
  final adding = !collection.hasPack(pack.id);
  try {
    if (adding) {
      await collection.addPack(pack);
    } else {
      await collection.removePack(pack);
    }
    if (context.mounted && collection.active) {
      showGfToast(
        context,
        adding ? strings.packAdded(pack.name) : strings.packRemoved(pack.name),
      );
    }
    return true;
  } catch (error) {
    if (context.mounted && collection.active) {
      showGfToast(context, strings.failure(error), error: true);
    }
    return false;
  }
}

/// Published packs by category (issue #1429). Browsing needs no account;
/// adding a pack is a server write that waits for confirmation.
class StickerMarketPage extends ConsumerStatefulWidget {
  const StickerMarketPage({super.key});

  @override
  ConsumerState<StickerMarketPage> createState() => _StickerMarketPageState();
}

class _StickerMarketPageState extends ConsumerState<StickerMarketPage> {
  static const _pageSize = 24;
  List<StickerMarketTopicPayload> _topics = const [];
  List<StickerMarketPackPayload> _packs = const [];
  String _topic = '';
  int _page = 0;
  int _total = 0;
  bool _loading = false;
  Object? _error;
  int _request = 0;

  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      if (!mounted) return;
      ref.read(stickerCollectionProvider).loadPacks(refresh: true);
      _load();
    });
  }

  Future<void> _load({String? topic, bool more = false}) async {
    final nextTopic = topic ?? _topic;
    final page = more ? _page + 1 : 1;
    final request = ++_request;
    setState(() {
      _loading = true;
      _error = null;
      if (!more) _topic = nextTopic;
    });
    try {
      final result = await ref
          .read(stickerCollectionProvider)
          .repository
          .market(topic: nextTopic, page: page, pageSize: _pageSize);
      if (!mounted || request != _request) return;
      setState(() {
        _topics = result.topics;
        _packs = more ? [..._packs, ...result.packs] : result.packs;
        _total = result.total;
        _page = page;
      });
    } catch (error) {
      if (mounted && request == _request) setState(() => _error = error);
    } finally {
      if (mounted && request == _request) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final strings = StickerStrings(context);
    final collection = ref.watch(stickerCollectionProvider);
    final colors = GfTheme.colorsOf(context);
    final hasMore = _packs.length < _total;
    return StickerSessionSurface(
      collection: collection,
      child: Scaffold(
        appBar: GfAppBar(title: GfText(strings.market)),
        body: SafeArea(
          top: false,
          child: RefreshIndicator(
            onRefresh: () => _load(),
            child: CustomScrollView(
              physics: const AlwaysScrollableScrollPhysics(),
              slivers: [
                if (_topics.isNotEmpty)
                  SliverToBoxAdapter(
                    child: SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
                      child: Row(
                        children: [
                          for (final topic in [null, ..._topics])
                            Padding(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 3,
                              ),
                              child: ChoiceChip(
                                label: GfText(topic?.label ?? strings.all),
                                selected: _topic == (topic?.id ?? ''),
                                onSelected: (_) =>
                                    _load(topic: topic?.id ?? ''),
                              ),
                            ),
                        ],
                      ),
                    ),
                  ),
                if (_loading && _packs.isEmpty)
                  const SliverToBoxAdapter(
                    child: Padding(
                      padding: EdgeInsets.all(32),
                      child: Center(child: GfProgressIndicator()),
                    ),
                  )
                else if (_error != null && _packs.isEmpty)
                  SliverToBoxAdapter(
                    child: _MarketStatus(
                      message: strings.failure(_error, loading: true),
                      action: strings.retry,
                      onAction: () => _load(),
                    ),
                  )
                else if (_packs.isEmpty)
                  SliverToBoxAdapter(
                    child: _MarketStatus(message: strings.marketEmpty),
                  )
                else
                  SliverList.builder(
                    itemCount: _packs.length,
                    itemBuilder: (context, index) {
                      final pack = _packs[index];
                      final added = collection.hasPack(pack.id);
                      return ListTile(
                        contentPadding: const EdgeInsets.symmetric(
                          horizontal: 16,
                        ),
                        leading: _PackCover(url: pack.coverUrl),
                        title: GfText(
                          pack.name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        subtitle: GfText(
                          strings.packCount(pack.stickerCount),
                          style: TextStyle(
                            fontSize: 13,
                            color: colors.textSecondary,
                          ),
                        ),
                        trailing: Semantics(
                          label:
                              '${added ? strings.removePackFull : strings.addPackFull} ${pack.name}',
                          excludeSemantics: true,
                          button: true,
                          child: GfButton(
                            label: added ? strings.remove : strings.addPack,
                            size: GfButtonSize.small,
                            variant: added
                                ? GfButtonVariant.secondary
                                : GfButtonVariant.tonal,
                            onPressed: collection.busy
                                ? null
                                : () => toggleStickerPack(
                                    context,
                                    collection,
                                    pack,
                                  ),
                          ),
                        ),
                        onTap: () => Navigator.of(context).push(
                          MaterialPageRoute<void>(
                            builder: (_) => ReadingPageWindow(
                              child: StickerMarketPackPage(pack: pack),
                            ),
                          ),
                        ),
                      );
                    },
                  ),
                if (_packs.isNotEmpty && (hasMore || _error != null))
                  SliverToBoxAdapter(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Center(
                        child: _loading
                            ? const GfProgressIndicator()
                            : TextButton(
                                onPressed: () => _load(more: hasMore),
                                child: GfText(
                                  _error != null
                                      ? strings.retry
                                      : strings.loadMore,
                                ),
                              ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// One pack before adding it: hold a sticker to preview, tap to view larger.
class StickerMarketPackPage extends ConsumerStatefulWidget {
  const StickerMarketPackPage({super.key, required this.pack});
  final StickerMarketPackPayload pack;

  @override
  ConsumerState<StickerMarketPackPage> createState() =>
      _StickerMarketPackPageState();
}

class _StickerMarketPackPageState extends ConsumerState<StickerMarketPackPage> {
  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      if (mounted) {
        ref.read(stickerCollectionProvider).loadPackStickers(widget.pack.id);
      }
    });
  }

  Future<void> _collect(
    StickerCollection collection,
    StickerItemPayload item,
  ) async {
    final strings = StickerStrings(context);
    try {
      await collection.save(stickerName: item.name);
      if (mounted && collection.active) showGfToast(context, strings.saved);
    } catch (error) {
      if (mounted && collection.active) {
        showGfToast(context, strings.failure(error), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final strings = StickerStrings(context);
    final collection = ref.watch(stickerCollectionProvider);
    final pack = widget.pack;
    final items = collection.packStickers[pack.id];
    final error = collection.packStickerErrors[pack.id];
    final added = collection.hasPack(pack.id);
    return StickerSessionSurface(
      collection: collection,
      child: Scaffold(
        appBar: GfAppBar(title: GfText(pack.name)),
        body: SafeArea(
          top: false,
          child: Column(
            children: [
              Expanded(
                child: items == null && error == null
                    ? const Center(child: GfProgressIndicator())
                    : error != null && items == null
                    ? _MarketStatus(
                        message: strings.failure(error, loading: true),
                        action: strings.retry,
                        onAction: () =>
                            collection.loadPackStickers(pack.id, refresh: true),
                      )
                    : StickerPeekScope(
                        labelOf: strings.displayLabel,
                        collectLabel: strings.collectShort,
                        canCollect: (item) => !collection.mine.any(
                          (value) => value.name == item.name,
                        ),
                        onCollect: (item) => _collect(collection, item),
                        child: GridView.builder(
                          padding: const EdgeInsets.all(12),
                          gridDelegate:
                              const SliverGridDelegateWithMaxCrossAxisExtent(
                                maxCrossAxisExtent: 90,
                                mainAxisExtent: 88,
                                mainAxisSpacing: 4,
                                crossAxisSpacing: 4,
                              ),
                          itemCount: items!.length,
                          itemBuilder: (context, index) {
                            final item = items[index];
                            return StickerPeekCell(
                              item: item,
                              child: Semantics(
                                button: true,
                                label: strings.displayLabel(item),
                                hint: strings.viewLarger,
                                child: InkWell(
                                  borderRadius: BorderRadius.circular(
                                    GfRadius.md,
                                  ),
                                  onTap: () => showStickerPreview(
                                    context,
                                    item.url,
                                    collection: collection,
                                  ),
                                  child: Column(
                                    children: [
                                      IgnorePointer(
                                        child: StickerImage(
                                          name: item.name,
                                          url: item.url,
                                          label: strings.displayLabel(item),
                                          collectible: false,
                                          excludeSemantics: true,
                                        ),
                                      ),
                                      GfText(
                                        strings.displayLabel(item),
                                        maxLines: 1,
                                        overflow: TextOverflow.ellipsis,
                                        style: Theme.of(
                                          context,
                                        ).textTheme.labelSmall,
                                      ),
                                    ],
                                  ),
                                ),
                              ),
                            );
                          },
                        ),
                      ),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
                child: GfButton(
                  label: added ? strings.removePackFull : strings.addPackFull,
                  variant: added
                      ? GfButtonVariant.secondary
                      : GfButtonVariant.primary,
                  expanded: true,
                  loading: collection.busy,
                  onPressed: collection.busy
                      ? null
                      : () => toggleStickerPack(context, collection, pack),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _PackCover extends StatelessWidget {
  const _PackCover({required this.url});
  final String url;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Container(
      width: 48,
      height: 48,
      decoration: BoxDecoration(
        color: colors.fill,
        borderRadius: BorderRadius.circular(GfRadius.sm),
      ),
      clipBehavior: Clip.antiAlias,
      child: url.isEmpty
          ? null
          : GfNetworkImage(
              resolveApiAssetUrl(url),
              fit: BoxFit.contain,
              cacheWidth: 96,
              excludeFromSemantics: true,
            ),
    );
  }
}

class _MarketStatus extends StatelessWidget {
  const _MarketStatus({required this.message, this.action, this.onAction});
  final String message;
  final String? action;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.all(32),
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        GfText(message, textAlign: TextAlign.center),
        if (action != null)
          TextButton(onPressed: onAction, child: GfText(action!)),
      ],
    ),
  );
}

import '../../navigation/reading_window.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../link_navigation.dart';
import '../../providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'sticker_image.dart';
import 'sticker_library_page.dart';
import 'sticker_library_state.dart';
import 'sticker_market_page.dart';
import 'sticker_peek.dart';
import 'sticker_strings.dart';
import 'sticker_preview.dart';

/// Inserts into the remembered selection without sending or discarding IME text.
void insertStickerText(
  TextEditingController controller,
  String token, {
  TextSelection? selection,
}) {
  final text = controller.text;
  final range = selection ?? controller.selection;
  final start = (range.isValid ? range.start : text.length).clamp(
    0,
    text.length,
  );
  final end = (range.isValid ? range.end : text.length).clamp(
    start,
    text.length,
  );
  controller.value = TextEditingValue(
    text: text.replaceRange(start, end, token),
    selection: TextSelection.collapsed(offset: start + token.length),
  );
}

/// One picker for chat, replies and publishing; selection inserts, never sends.
class StickerPicker extends ConsumerStatefulWidget {
  const StickerPicker({super.key, required this.onInsert});
  final ValueChanged<String> onInsert;

  /// Grid geometry shared with [comfortableHeight]: about five stickers per
  /// row on a phone, each with a 56 px image and its own touch target.
  static const double cellExtent = 76;
  static const double rowExtent = 64;
  static const double _gap = 4;
  static const double _headerHeight = 44;
  static const double _filterHeight = 40;
  static const double _gridPadding = 8;
  static const double _gridTopPadding = 4;

  /// Content height that shows the header, the pack strip and three full
  /// rows (296 px). Composers open the panel at least this tall (safe area
  /// excluded), so a panel matched to a short keyboard is not reduced to one
  /// row; common keyboards are at least this tall, so the swap stays still.
  static const double comfortableHeight =
      _headerHeight +
      _filterHeight +
      3 * rowExtent +
      2 * _gap +
      _gridTopPadding +
      _gridPadding;

  @override
  ConsumerState<StickerPicker> createState() => _StickerPickerState();
}

class _StickerPickerState extends ConsumerState<StickerPicker> {
  int _tab = 2;
  String? _pack;
  int? _marketPack;
  String _query = '';
  // Search starts as an icon; opening it swaps the tab row for a field that
  // searches every loaded sticker, whatever tab was showing.
  bool _searching = false;
  final TextEditingController _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  void _openSearch() {
    setState(() => _searching = true);
    // Search covers every added pack, not only the ones opened so far.
    final state = ref.read(stickerCollectionProvider);
    for (final pack in state.packs) {
      if (!state.packStickers.containsKey(pack.id)) {
        state.loadPackStickers(pack.id);
      }
    }
  }

  void _closeSearch() {
    _search.clear();
    setState(() {
      _searching = false;
      _query = '';
    });
    // Hand focus back to the panel so the keyboard the field raised goes away
    // while the panel stays open.
    final panel = Focus.maybeOf(context);
    if (panel != null) {
      panel.requestFocus();
    } else {
      FocusManager.instance.primaryFocus?.unfocus();
    }
  }

  void _openLibrary() => Navigator.of(context, rootNavigator: true).push(
    MaterialPageRoute<void>(
      builder: (_) => ReadingPageWindow(child: const StickerLibraryPage()),
    ),
  );

  /// Every loaded sticker once, in the order a person would look for it.
  List<StickerItemPayload> _searchPool(StickerCollection state) {
    final seen = <String>{};
    return [
      for (final item in [
        ...state.recent,
        ...state.mine,
        ...state.official,
        for (final pack in state.packs) ...?state.packStickers[pack.id],
      ])
        if (seen.add(item.name)) item,
    ];
  }

  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      if (!mounted) return;
      final collection = ref.read(stickerCollectionProvider);
      if (collection.recent.isNotEmpty) setState(() => _tab = 0);
      collection.loadOfficial(refresh: true);
      collection.loadMine(refresh: true);
      collection.loadPacks(refresh: true);
    });
  }

  void _showMarketPack(StickerCollection collection, int id) {
    setState(() => _marketPack = id);
    collection.loadPackStickers(id);
  }

  Future<void> _collect(StickerItemPayload item) async {
    final collection = ref.read(stickerCollectionProvider);
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

  Future<void> _actions(StickerItemPayload item) async {
    final collection = ref.read(stickerCollectionProvider);
    final strings = StickerStrings(context);
    final saved = collection.mine.any((value) => value.name == item.name);
    final action = await showGfActionSheet<String>(
      context,
      surfaceBuilder: (_, child) =>
          StickerSessionSurface(collection: collection, child: child),
      items: [
        GfPopupMenuItem(
          value: 'preview',
          child: Row(
            children: [
              GfSymbol('maximize', size: 22),
              const SizedBox(width: 12),
              Expanded(child: GfText(strings.viewLarger)),
            ],
          ),
        ),
        GfPopupMenuItem(
          value: 'save',
          enabled: !saved && !collection.busy,
          child: Row(
            children: [
              GfSymbol(saved ? 'bookmark-filled' : 'bookmark', size: 22),
              const SizedBox(width: 12),
              Expanded(child: GfText(saved ? strings.saved : strings.collect)),
            ],
          ),
        ),
      ],
    );
    if (!mounted || !collection.active) return;
    if (action == 'preview') {
      await showStickerPreview(context, item.url, collection: collection);
      return;
    }
    if (action == 'save') await _collect(item);
  }

  Future<void> _openLicenses() async {
    try {
      await LinkNavigation.open(
        context,
        'https://github.com/YourTongji/YourTJ-Hub/blob/dev/apps/gooseforum/app/console/stickerpresets/preset_stickers/NOTICE.md',
        baseUrl: ref.read(apiClientProvider).baseUrl,
      );
    } catch (_) {
      if (mounted) {
        showGfToast(
          context,
          StickerStrings(context).operationFailed,
          error: true,
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final strings = StickerStrings(context);
    final state = ref.watch(stickerCollectionProvider);
    final colors = GfTheme.colorsOf(context);
    // The packs tab follows the account's subscriptions; a removed pack falls
    // back to the first remaining one.
    final marketPack = state.packs.any((pack) => pack.id == _marketPack)
        ? _marketPack
        : state.packs.firstOrNull?.id;
    if (_tab == 3 && marketPack != null && marketPack != _marketPack) {
      _marketPack = marketPack;
      Future.microtask(() => state.loadPackStickers(marketPack));
    }
    final query = _query.trim().toLowerCase();
    final searching = _searching && query.isNotEmpty;
    final source = searching
        ? _searchPool(state)
        : switch (_tab) {
            1 => state.mine,
            2 => state.official,
            3 => state.packStickers[marketPack] ?? const <StickerItemPayload>[],
            _ => state.recent,
          };
    final items = source
        .where(
          (item) =>
              item.isAvailable &&
              (searching || _tab != 2 || _pack == null || item.pack == _pack) &&
              (query.isEmpty ||
                  strings.displayLabel(item).toLowerCase().contains(query) ||
                  item.name.toLowerCase().contains(query)),
        )
        .toList();
    final error = searching
        ? null
        : switch (_tab) {
            1 => state.mineError,
            2 => state.officialError,
            3 =>
              marketPack == null
                  ? state.packsError
                  : state.packStickerErrors[marketPack],
            _ => null,
          };
    final loading =
        !searching &&
        switch (_tab) {
          1 => state.loadingMine,
          2 => state.loadingOfficial,
          3 =>
            state.loadingPacks ||
                state.loadingPackStickers.contains(marketPack),
          _ => false,
        };
    final officialPacks = state.official
        .map((e) => e.pack)
        .where((e) => e.isNotEmpty)
        .toSet()
        .toList();
    final Widget? filters = _searching
        ? null
        : _tab == 2 && officialPacks.length > 1
        ? _filterStrip([
            for (final pack in [null, ...officialPacks])
              _filterChip(
                label: pack ?? strings.all,
                selected: _pack == pack,
                onSelected: () => setState(() => _pack = pack),
              ),
          ])
        : _tab == 3 && state.packs.isNotEmpty
        ? _filterStrip([
            for (final pack in state.packs)
              _filterChip(
                label: pack.name,
                cover: pack.coverUrl,
                selected: marketPack == pack.id,
                onSelected: () => _showMarketPack(state, pack.id),
              ),
            ActionChip(
              avatar: const GfSymbol('plus', size: 16),
              label: GfText(strings.market),
              tooltip: strings.browseMarket,
              visualDensity: VisualDensity.compact,
              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
              onPressed: () => openStickerMarket(context),
            ),
          ])
        : null;
    final duration = GfMotion.duration(context, GfMotion.content);
    return Column(
      children: [
        SizedBox(
          height: StickerPicker._headerHeight,
          child: AnimatedSwitcher(
            duration: duration,
            switchInCurve: GfMotion.enterCurve,
            switchOutCurve: GfMotion.exitCurve,
            transitionBuilder: (child, animation) => FadeTransition(
              opacity: animation,
              child: SlideTransition(
                position: Tween<Offset>(
                  begin: const Offset(.04, 0),
                  end: Offset.zero,
                ).animate(animation),
                child: child,
              ),
            ),
            child: _searching
                ? _searchBar(strings)
                : _tabBar(strings, colors, state, marketPack),
          ),
        ),
        AnimatedSize(
          duration: duration,
          curve: GfMotion.layoutCurve,
          alignment: Alignment.topCenter,
          child: filters ?? const SizedBox(width: double.infinity),
        ),
        Expanded(
          child: StickerPeekScope(
            labelOf: strings.displayLabel,
            collectLabel: strings.collectShort,
            canCollect: (item) =>
                !state.mine.any((value) => value.name == item.name),
            onCollect: _collect,
            child: CustomScrollView(
              // Each tab, pack and search starts at its first row.
              key: ValueKey((searching, _tab, marketPack, _pack)),
              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.manual,
              slivers: [
                if (loading && source.isEmpty)
                  const SliverToBoxAdapter(
                    child: Padding(
                      padding: EdgeInsets.all(24),
                      child: Center(child: GfProgressIndicator()),
                    ),
                  )
                else if (error != null)
                  SliverToBoxAdapter(
                    child: _status(
                      strings.failure(error, loading: true),
                      action: strings.retry,
                      onAction: () => switch (_tab) {
                        1 => state.loadMine(refresh: true),
                        3 when marketPack != null => state.loadPackStickers(
                          marketPack,
                          refresh: true,
                        ),
                        3 => state.loadPacks(refresh: true),
                        _ => state.loadOfficial(refresh: true),
                      },
                    ),
                  )
                else if (items.isEmpty)
                  SliverToBoxAdapter(
                    child: query.isNotEmpty
                        ? _status(strings.noResults)
                        : _searching
                        ? const SizedBox.shrink()
                        : _status(
                            switch (_tab) {
                              1 => strings.emptyMine,
                              2 => strings.emptyOfficial,
                              3 => strings.emptyPacks,
                              _ => strings.emptyRecent,
                            },
                            action: switch (_tab) {
                              0 => strings.official,
                              1 => strings.add,
                              3 when state.packs.isEmpty =>
                                strings.browseMarket,
                              _ => null,
                            },
                            onAction: _tab == 0
                                ? () => setState(() => _tab = 2)
                                : _tab == 3
                                ? () => openStickerMarket(context)
                                : _openLibrary,
                          ),
                  )
                else
                  SliverPadding(
                    padding: const EdgeInsets.fromLTRB(
                      StickerPicker._gridPadding,
                      StickerPicker._gridTopPadding,
                      StickerPicker._gridPadding,
                      StickerPicker._gridPadding,
                    ),
                    sliver: SliverGrid.builder(
                      gridDelegate:
                          const SliverGridDelegateWithMaxCrossAxisExtent(
                            maxCrossAxisExtent: StickerPicker.cellExtent,
                            mainAxisExtent: StickerPicker.rowExtent,
                            mainAxisSpacing: StickerPicker._gap,
                            crossAxisSpacing: StickerPicker._gap,
                          ),
                      itemCount: items.length,
                      itemBuilder: (context, index) =>
                          _cell(items[index], strings, state),
                    ),
                  ),
                if (!_searching && items.isNotEmpty)
                  SliverPadding(
                    padding: const EdgeInsets.fromLTRB(16, 0, 16, 4),
                    sliver: SliverToBoxAdapter(
                      child: GfText(
                        strings.pickerHint,
                        textAlign: TextAlign.center,
                        style: TextStyle(fontSize: 12, color: colors.iconMuted),
                      ),
                    ),
                  ),
                if (_tab == 2 && !_searching)
                  SliverToBoxAdapter(
                    child: Align(
                      child: TextButton.icon(
                        onPressed: _openLicenses,
                        icon: const GfSymbol('external-link', size: 14),
                        label: GfText(strings.licenses),
                        style: TextButton.styleFrom(
                          textStyle: const TextStyle(fontSize: 12),
                          foregroundColor: colors.iconMuted,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _tabBar(
    StickerStrings strings,
    GfColors colors,
    StickerCollection state,
    int? marketPack,
  ) {
    final typography = Theme.of(context).textTheme.labelLarge;
    return Row(
      key: const ValueKey('sticker-tabs'),
      children: [
        const SizedBox(width: 4),
        for (final (index, title) in [
          strings.recent,
          strings.mine,
          strings.official,
          strings.packs,
        ].indexed)
          Expanded(
            child: Semantics(
              selected: _tab == index,
              child: TextButton(
                onPressed: () {
                  setState(() => _tab = index);
                  if (index == 3 && marketPack != null) {
                    state.loadPackStickers(marketPack);
                  }
                },
                style: TextButton.styleFrom(
                  minimumSize: const Size(0, 40),
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  foregroundColor: _tab == index
                      ? colors.primary
                      : colors.iconMuted,
                  // Weight marks the selection without relying on color alone.
                  textStyle: typography?.copyWith(
                    fontWeight: _tab == index
                        ? FontWeight.w600
                        : FontWeight.w400,
                  ),
                ),
                child: GfText(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ),
          ),
        IconButton(
          tooltip: strings.searchAll,
          icon: const GfSymbol('search', size: 20),
          color: colors.iconMuted,
          onPressed: _openSearch,
        ),
        IconButton(
          tooltip: strings.manageLibrary,
          icon: const GfSymbol('settings', size: 20),
          color: colors.iconMuted,
          onPressed: _openLibrary,
        ),
        const SizedBox(width: 4),
      ],
    );
  }

  Widget _searchBar(StickerStrings strings) => Padding(
    key: const ValueKey('sticker-search'),
    padding: const EdgeInsetsDirectional.only(start: 12, end: 4),
    child: Row(
      children: [
        Expanded(
          child: GfSearchField(
            controller: _search,
            autofocus: true,
            hintText: strings.searchAll,
            clearLabel: MaterialLocalizations.of(context).deleteButtonTooltip,
            onChanged: (value) => setState(() => _query = value),
          ),
        ),
        TextButton(onPressed: _closeSearch, child: GfText(strings.closeSearch)),
      ],
    ),
  );

  Widget _filterStrip(List<Widget> chips) => SizedBox(
    height: StickerPicker._filterHeight,
    child: ListView.separated(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      itemCount: chips.length,
      separatorBuilder: (_, _) => const SizedBox(width: 6),
      itemBuilder: (_, index) => chips[index],
    ),
  );

  Widget _filterChip({
    required String label,
    required bool selected,
    required VoidCallback onSelected,
    String? cover,
  }) => ChoiceChip(
    label: GfText(label, maxLines: 1, overflow: TextOverflow.ellipsis),
    // A pack's cover makes the strip scannable before reading its name.
    avatar: cover == null || cover.isEmpty
        ? null
        : IgnorePointer(
            child: StickerImage(
              name: label,
              url: cover,
              size: 20,
              collectible: false,
              excludeSemantics: true,
            ),
          ),
    showCheckmark: false,
    selected: selected,
    visualDensity: VisualDensity.compact,
    materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
    onSelected: (_) => onSelected(),
  );

  Widget _cell(
    StickerItemPayload item,
    StickerStrings strings,
    StickerCollection state,
  ) {
    // Holding opens the preview bubble. Screen readers, keyboards and pointers
    // keep the preview/save menu. The label lives in semantics and the bubble,
    // so the grid spends its height on stickers.
    final label = strings.displayLabel(item);
    return StickerPeekCell(
      item: item,
      onSecondaryTap: () => _actions(item),
      child: CallbackShortcuts(
        bindings: {
          const SingleActivator(LogicalKeyboardKey.contextMenu): () =>
              _actions(item),
          const SingleActivator(LogicalKeyboardKey.f10, shift: true): () =>
              _actions(item),
        },
        child: Semantics(
          button: true,
          label: label,
          hint: strings.pickerHint,
          onLongPress: () => _actions(item),
          child: InkWell(
            borderRadius: BorderRadius.circular(GfRadius.md),
            onTap: () {
              widget.onInsert(item.token);
              state.used(item);
            },
            child: Center(
              child: IgnorePointer(
                child: StickerImage(
                  name: item.name,
                  url: item.url,
                  label: label,
                  collectible: false,
                  excludeSemantics: true,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _status(String message, {String? action, VoidCallback? onAction}) =>
      Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          children: [
            GfText(message, textAlign: TextAlign.center),
            if (action != null)
              TextButton(onPressed: onAction, child: GfText(action)),
          ],
        ),
      );
}

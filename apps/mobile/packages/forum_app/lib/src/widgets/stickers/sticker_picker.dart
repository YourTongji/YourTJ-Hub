import 'package:flutter/material.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../link_navigation.dart';
import '../../providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'sticker_image.dart';
import 'sticker_library_page.dart';
import 'sticker_library_state.dart';
import 'sticker_strings.dart';

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

Future<void> showStickerPicker(
  BuildContext context, {
  required ValueChanged<String> onInsert,
}) async {
  FocusManager.instance.primaryFocus?.unfocus();
  await showGfBottomSheet<void>(
    context,
    keyboardAware: true,
    height: (MediaQuery.sizeOf(context).height * .58).clamp(240.0, 520.0),
    builder: (_) => StickerPicker(onInsert: onInsert),
  );
}

/// One picker for chat, replies and publishing; selection inserts, never sends.
class StickerPicker extends ConsumerStatefulWidget {
  const StickerPicker({super.key, required this.onInsert});
  final ValueChanged<String> onInsert;
  @override
  ConsumerState<StickerPicker> createState() => _StickerPickerState();
}

class _StickerPickerState extends ConsumerState<StickerPicker> {
  int _tab = 2;
  String? _pack;
  String _query = '';
  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      if (!mounted) return;
      final collection = ref.read(stickerCollectionProvider);
      if (collection.recent.isNotEmpty) setState(() => _tab = 0);
      collection.loadOfficial(refresh: true);
      collection.loadMine(refresh: true);
    });
  }

  Future<void> _collect(StickerItemPayload item) async {
    final collection = ref.read(stickerCollectionProvider);
    final strings = StickerStrings(context);
    final saved = collection.mine.any((value) => value.name == item.name);
    final confirmed = await showGfBottomSheet<bool>(
      context,
      builder: (context) => ListTile(
        leading: GfSymbol(saved ? 'bookmark-filled' : 'bookmark', size: 22),
        title: Text(saved ? strings.saved : strings.collect),
        enabled: !saved && !collection.busy,
        onTap: () => Navigator.pop(context, true),
      ),
    );
    if (confirmed != true || !mounted || !collection.active) return;
    try {
      await collection.save(stickerName: item.name);
      if (mounted && collection.active) showGfToast(context, strings.saved);
    } catch (error) {
      if (mounted && collection.active) {
        showGfToast(context, strings.failure(error), error: true);
      }
    }
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
    final source = switch (_tab) {
      1 => state.mine,
      2 => state.official,
      _ => state.recent,
    };
    final items = source
        .where(
          (item) =>
              item.isEnabled &&
              (_tab != 2 || _pack == null || item.pack == _pack) &&
              (strings
                      .displayLabel(item)
                      .toLowerCase()
                      .contains(_query.toLowerCase()) ||
                  item.name.toLowerCase().contains(_query.toLowerCase())),
        )
        .toList();
    final error = _tab == 1
        ? state.mineError
        : _tab == 2
        ? state.officialError
        : null;
    final loading = _tab == 1
        ? state.loadingMine
        : _tab == 2
        ? state.loadingOfficial
        : false;
    final packs = state.official
        .map((e) => e.pack)
        .where((e) => e.isNotEmpty)
        .toSet()
        .toList();
    return Column(
      children: [
        Row(
          children: [
            const SizedBox(width: 12),
            for (final (index, title) in [
              strings.recent,
              strings.mine,
              strings.official,
            ].indexed)
              Expanded(
                child: Semantics(
                  selected: _tab == index,
                  child: TextButton(
                    onPressed: () => setState(() => _tab = index),
                    style: TextButton.styleFrom(
                      foregroundColor: _tab == index
                          ? colors.primary
                          : colors.iconMuted,
                    ),
                    child: Text(
                      title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ),
              ),
            IconButton(
              tooltip: strings.manage,
              icon: const GfSymbol('sliders-horizontal', size: 22),
              onPressed: () => Navigator.of(context, rootNavigator: true).push(
                MaterialPageRoute<void>(
                  builder: (_) => const StickerLibraryPage(),
                ),
              ),
            ),
          ],
        ),
        // Search, packs and attribution scroll with the grid on short windows.
        // Only the three destinations remain fixed, leaving a useful viewport
        // when the keyboard or enlarged text consumes the available height.
        Expanded(
          child: CustomScrollView(
            slivers: [
              SliverPadding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                sliver: SliverToBoxAdapter(
                  child: GfSearchField(
                    hintText: strings.search,
                    clearLabel: MaterialLocalizations.of(
                      context,
                    ).deleteButtonTooltip,
                    onChanged: (value) => setState(() => _query = value),
                  ),
                ),
              ),
              if (_tab == 2 && packs.isNotEmpty)
                SliverToBoxAdapter(
                  child: SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    child: Row(
                      children: [
                        for (final pack in [null, ...packs])
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 3),
                            child: ChoiceChip(
                              label: Text(pack ?? strings.all),
                              selected: _pack == pack,
                              onSelected: (_) => setState(() => _pack = pack),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              if (loading && source.isEmpty)
                const SliverToBoxAdapter(
                  child: Padding(
                    padding: EdgeInsets.all(24),
                    child: Center(child: CircularProgressIndicator()),
                  ),
                )
              else if (error != null)
                SliverToBoxAdapter(
                  child: _status(
                    strings.failure(error, loading: true),
                    action: strings.retry,
                    onAction: () => _tab == 1
                        ? state.loadMine(refresh: true)
                        : state.loadOfficial(refresh: true),
                  ),
                )
              else if (items.isEmpty)
                SliverToBoxAdapter(
                  child: _status(
                    _query.isNotEmpty
                        ? strings.noResults
                        : switch (_tab) {
                            1 => strings.emptyMine,
                            2 => strings.emptyOfficial,
                            _ => strings.emptyRecent,
                          },
                    action: _tab == 0
                        ? strings.official
                        : _tab == 1
                        ? strings.add
                        : null,
                    onAction: _tab == 0
                        ? () => setState(() => _tab = 2)
                        : () => Navigator.of(context, rootNavigator: true).push(
                            MaterialPageRoute<void>(
                              builder: (_) => const StickerLibraryPage(),
                            ),
                          ),
                  ),
                )
              else
                SliverPadding(
                  padding: const EdgeInsets.all(12),
                  sliver: SliverGrid.builder(
                    gridDelegate:
                        const SliverGridDelegateWithMaxCrossAxisExtent(
                          maxCrossAxisExtent: 90,
                          mainAxisExtent: 88,
                          mainAxisSpacing: 4,
                          crossAxisSpacing: 4,
                        ),
                    itemCount: items.length,
                    itemBuilder: (context, index) {
                      final item = items[index];
                      return Semantics(
                        button: true,
                        label: strings.displayLabel(item),
                        child: InkWell(
                          borderRadius: BorderRadius.circular(12),
                          onLongPress: _tab == 1 ? null : () => _collect(item),
                          onTap: () {
                            widget.onInsert(item.token);
                            state.used(item);
                          },
                          child: Column(
                            children: [
                              IgnorePointer(
                                child: StickerImage(
                                  name: item.name,
                                  url: item.url,
                                  label: strings.displayLabel(item),
                                  collectible: false,
                                ),
                              ),
                              Text(
                                strings.displayLabel(item),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: Theme.of(context).textTheme.labelSmall,
                              ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
                ),
              if (_tab == 2)
                SliverToBoxAdapter(
                  child: Align(
                    child: TextButton.icon(
                      onPressed: _openLicenses,
                      icon: const GfSymbol('external-link', size: 14),
                      label: Text(strings.licenses),
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
      ],
    );
  }

  Widget _status(String message, {String? action, VoidCallback? onAction}) =>
      Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          children: [
            Text(message, textAlign: TextAlign.center),
            if (action != null)
              TextButton(onPressed: onAction, child: Text(action)),
          ],
        ),
      );
}

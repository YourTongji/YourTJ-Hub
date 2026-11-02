import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart' as quill;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'sticker_image.dart';
import 'sticker_library_state.dart';
import 'sticker_strings.dart';

/// Candidates for a `[:sticker:` fragment typed by hand (MADR 0106
/// supplement): a horizontal strip above the input that filters the viewer's
/// stickers, then the official ones, then subscribed packs, by name and
/// label. Tapping one replaces the fragment with the complete token, which
/// the composer then shows as the image. Hidden while no fragment is open.
class StickerSuggestionBar extends ConsumerStatefulWidget {
  const StickerSuggestionBar({
    super.key,
    required this.listenable,
    required this.prefix,
    required this.replace,
    this.margin = const EdgeInsets.only(bottom: 6),
  });

  /// A plain-text composer: the fragment ends at a collapsed caret.
  factory StickerSuggestionBar.text({
    Key? key,
    required TextEditingController controller,
    EdgeInsetsGeometry margin = const EdgeInsets.only(bottom: 6),
  }) => StickerSuggestionBar(
    key: key,
    margin: margin,
    listenable: controller,
    prefix: () {
      final value = controller.value;
      final selection = value.selection;
      if (!selection.isValid || !selection.isCollapsed) return null;
      return value.text.substring(
        0,
        selection.baseOffset.clamp(0, value.text.length),
      );
    },
    replace: (length, token) {
      final value = controller.value;
      final end = value.selection.baseOffset;
      final start = (end - length).clamp(0, end);
      controller.value = TextEditingValue(
        text: value.text.replaceRange(start, end, token),
        selection: TextSelection.collapsed(offset: start + token.length),
      );
    },
  );

  /// A rich (Quill) editor: the pick goes in as a sticker embed. No
  /// candidates inside inline code or code blocks, where tokens stay text.
  factory StickerSuggestionBar.quill({
    Key? key,
    required quill.QuillController controller,
    EdgeInsetsGeometry margin = const EdgeInsets.only(bottom: 6),
  }) => StickerSuggestionBar(
    key: key,
    margin: margin,
    listenable: controller,
    prefix: () {
      final selection = controller.selection;
      if (!selection.isValid || !selection.isCollapsed) return null;
      final attributes = controller.getSelectionStyle().attributes;
      if (attributes.containsKey(quill.Attribute.inlineCode.key) ||
          attributes.containsKey(quill.Attribute.codeBlock.key)) {
        return null;
      }
      final plain = controller.document.toPlainText();
      final end = selection.baseOffset.clamp(0, plain.length);
      // A fragment is at most the prefix plus 64 name characters.
      return plain.substring((end - 80).clamp(0, end), end);
    },
    replace: (length, token) {
      final end = controller.selection.baseOffset;
      final start = (end - length).clamp(0, end);
      final name = stickerTokenPattern.firstMatch(token)!.group(1)!;
      controller.replaceText(
        start,
        end - start,
        StickerEmbed(name),
        TextSelection.collapsed(offset: start + 1),
      );
    },
  );

  final Listenable listenable;

  /// Text before the caret, or null while text is selected.
  final String? Function() prefix;

  /// Replaces the [length] characters before the caret with [token].
  final void Function(int length, String token) replace;

  /// Space around the strip; only applied while it is visible.
  final EdgeInsetsGeometry margin;

  static const double cellExtent = 52;
  static const double imageSize = 44;
  static const int limit = 16;

  @override
  ConsumerState<StickerSuggestionBar> createState() =>
      _StickerSuggestionBarState();
}

class _StickerSuggestionBarState extends ConsumerState<StickerSuggestionBar> {
  /// A new account/site session brings a new collection to load once.
  StickerCollection? _requestedFor;

  void _requestSources(StickerCollection collection) {
    if (identical(_requestedFor, collection)) return;
    _requestedFor = collection;
    // Loading notifies the collection; never from inside this build.
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (!mounted || !collection.active) return;
      await Future.wait([
        collection.loadOfficial(),
        collection.loadMine(),
        collection.loadPacks(),
      ]);
      for (final pack in collection.packs) {
        if (!mounted || !collection.active) return;
        await collection.loadPackStickers(pack.id);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final collection = ref.watch(stickerCollectionProvider);
    return ListenableBuilder(
      listenable: widget.listenable,
      builder: (context, _) {
        final prefix = widget.prefix();
        final query = prefix == null ? null : extractStickerQuery(prefix);
        if (query == null) return const SizedBox.shrink();
        _requestSources(collection);
        return ListenableBuilder(
          listenable: collection,
          builder: (context, _) => _strip(context, collection, query),
        );
      },
    );
  }

  Widget _strip(
    BuildContext context,
    StickerCollection collection,
    StickerQuery query,
  ) {
    final strings = StickerStrings(context);
    final colors = GfTheme.colorsOf(context);
    final sources = <StickerItemPayload>[
      ...collection.mine,
      ...collection.officialInPackOrder,
      for (final pack in collection.packs) ...?collection.packStickers[pack.id],
    ].where((item) => item.isAvailable && item.url.isNotEmpty);
    final candidates = rankStickerSuggestions(
      sources,
      query.query,
      name: (item) => item.name,
      label: strings.displayLabel,
      limit: StickerSuggestionBar.limit,
    );
    final loading =
        collection.loadingOfficial ||
        collection.loadingMine ||
        collection.loadingPacks;
    return Semantics(
      container: true,
      label: strings.suggestions,
      child: Container(
        key: const Key('sticker-suggestion-bar'),
        height: StickerSuggestionBar.cellExtent + 8,
        margin: widget.margin,
        decoration: BoxDecoration(
          color: colors.base200,
          borderRadius: BorderRadius.circular(GfRadius.lg),
        ),
        child: candidates.isEmpty
            ? Center(
                child: loading
                    ? const SizedBox.square(
                        dimension: 16,
                        child: GfProgressIndicator(strokeWidth: 1.5),
                      )
                    : GfText(
                        strings.noMatches,
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: colors.iconMuted,
                        ),
                      ),
              )
            : ListView.separated(
                scrollDirection: Axis.horizontal,
                padding: const EdgeInsets.all(4),
                itemCount: candidates.length,
                separatorBuilder: (_, _) => const SizedBox(width: 2),
                itemBuilder: (context, index) {
                  final item = candidates[index];
                  return Semantics(
                    button: true,
                    label: strings.displayLabel(item),
                    excludeSemantics: true,
                    child: InkWell(
                      key: ValueKey('sticker-suggestion-${item.name}'),
                      borderRadius: BorderRadius.circular(GfRadius.md),
                      onTap: () {
                        collection.library.remember([item]);
                        widget.replace(query.length, item.token);
                      },
                      child: SizedBox.square(
                        dimension: StickerSuggestionBar.cellExtent,
                        child: Center(
                          child: IgnorePointer(
                            child: StickerImage(
                              name: item.name,
                              url: item.url,
                              size: StickerSuggestionBar.imageSize,
                              collectible: false,
                              excludeSemantics: true,
                            ),
                          ),
                        ),
                      ),
                    ),
                  );
                },
              ),
      ),
    );
  }
}

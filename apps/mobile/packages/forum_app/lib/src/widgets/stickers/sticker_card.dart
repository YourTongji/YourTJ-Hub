import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../asset_url.dart';
import '../../providers.dart';
import 'sticker_library_state.dart';
import 'sticker_preview.dart';
import 'sticker_strings.dart';

/// Tapping a sticker in content opens this compact card instead of the photo
/// viewer: an enlarged sticker, its name and one collection action. The card
/// closes with the account that opened it, like library-launched previews.
///
/// Guests can browse public content, but the sticker library is account-owned:
/// any authenticated request from a guest (my-stickers / my-sticker-save)
/// returns 401, which the API client treats as a session boundary and tears
/// the whole shell down to the login page. So for guests the card skips the
/// library fetch entirely and offers a "sign in to save" action instead.
Future<void> showStickerCard(
  BuildContext context, {
  required String name,
  required String url,
  String? label,
  bool collectible = true,
}) async {
  final container = ProviderScope.containerOf(context, listen: false);
  final collection = collectible
      ? container.read(stickerCollectionProvider)
      : null;
  // Distinguish "no token" (guest) from "storage unavailable": the latter
  // keeps the signed-in path so a transient secure-storage failure cannot
  // demote a signed-in user to the guest action. The read is bounded: the
  // card must always open (widget tests without a storage override never
  // complete the secure-storage plugin call), and a stalled plugin must not
  // block a reader either.
  bool signedIn;
  try {
    final token = await container
        .read(tokenStorageProvider)
        .read()
        .timeout(const Duration(milliseconds: 500));
    signedIn = token != null && token.isNotEmpty;
  } catch (_) {
    signedIn = true;
  }
  if (!context.mounted) return;
  // The callback runs after the dialog closes, when the caller element may
  // already be defunct, so resolve the router now, while the caller context
  // is alive, and let the callback pop via the dialog's own context instead
  // of touching the caller's.
  final router = signedIn ? null : GoRouter.of(context);
  return showGfModal<void>(
    context,
    builder: (dialogContext) {
      final card = StickerCard(
        name: name,
        url: url,
        label: label,
        collection: collection,
        signedIn: signedIn,
        onSignIn: signedIn
            ? null
            : () {
                Navigator.of(dialogContext, rootNavigator: true).maybePop();
                router?.go('/login');
              },
      );
      return collection == null
          ? card
          : StickerSessionSurface(collection: collection, child: card);
    },
  );
}

class StickerCard extends StatefulWidget {
  const StickerCard({
    super.key,
    required this.name,
    required this.url,
    this.label,
    this.collection,
    this.signedIn = true,
    this.onSignIn,
  });

  /// Enlarged display size; small enough to keep the card light.
  static const double stageSize = 168;

  final String name;
  final String url;
  final String? label;

  /// Null hides the collection action (read-only excerpts).
  final StickerCollection? collection;

  /// False for guests: the library is not fetched and the collection action
  /// becomes a sign-in prompt ([onSignIn]).
  final bool signedIn;

  /// Where the guest prompt navigates; provided by [showStickerCard].
  final VoidCallback? onSignIn;

  @override
  State<StickerCard> createState() => _StickerCardState();
}

class _StickerCardState extends State<StickerCard> {
  bool _saving = false;
  Object? _error;

  @override
  void initState() {
    super.initState();
    final collection = widget.collection;
    if (collection == null || !widget.signedIn) return;
    collection.addListener(_changed);
    // Membership decides between "save" and "saved"; a cached list is reused.
    Future.microtask(() {
      if (mounted && collection.active) collection.loadMine();
    });
  }

  @override
  void dispose() {
    widget.collection?.removeListener(_changed);
    super.dispose();
  }

  void _changed() {
    if (mounted) setState(() {});
  }

  bool get _saved =>
      widget.collection?.mine.any((item) => item.name == widget.name) ?? false;

  Future<void> _save() async {
    final collection = widget.collection!;
    if (!collection.active || _saving) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await collection.save(stickerName: widget.name);
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  String _label(StickerStrings strings) {
    if (widget.label != null) return widget.label!;
    final collection = widget.collection;
    for (final item in [...?collection?.mine, ...?collection?.library.items]) {
      if (item.name == widget.name) return strings.displayLabel(item);
    }
    return strings.fallbackLabel(widget.name);
  }

  @override
  Widget build(BuildContext context) {
    final strings = StickerStrings(context);
    final colors = GfTheme.colorsOf(context);
    final typography = GfTheme.typographyOf(context);
    final label = _label(strings);
    final collection = widget.collection;
    final saved = _saved;
    return SizedBox(
      width: 240,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Semantics(
            image: true,
            label: label,
            child: SizedBox.square(
              dimension: StickerCard.stageSize,
              child: GfNetworkImage(
                resolveApiAssetUrl(widget.url),
                fit: BoxFit.contain,
                excludeFromSemantics: true,
                errorBuilder: (context, _, _) => Center(
                  child: GfSymbol(
                    'image-off',
                    size: 28,
                    color: colors.iconMuted,
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(height: GfSpace.lg),
          GfText(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
            style: typography.heading,
          ),
          if (collection != null) ...[
            const SizedBox(height: GfSpace.xl),
            AnimatedSwitcher(
              duration: GfMotion.duration(context, GfMotion.content),
              child: !widget.signedIn
                  ? GfButton(
                      key: const ValueKey('sign-in'),
                      label: strings.signInToSave,
                      onPressed: widget.onSignIn,
                      variant: GfButtonVariant.tonal,
                      icon: const GfSymbol('bookmark', size: 18),
                      expanded: true,
                    )
                  : saved
                  ? GfButton(
                      key: const ValueKey('saved'),
                      label: strings.inMine,
                      onPressed: null,
                      variant: GfButtonVariant.secondary,
                      icon: const GfSymbol('bookmark-filled', size: 18),
                      expanded: true,
                    )
                  : GfButton(
                      key: const ValueKey('save'),
                      label: strings.collect,
                      onPressed: collection.active ? _save : null,
                      loading: _saving,
                      variant: GfButtonVariant.tonal,
                      icon: const GfSymbol('bookmark', size: 18),
                      expanded: true,
                    ),
            ),
            AnimatedSize(
              duration: GfMotion.duration(context, GfMotion.content),
              alignment: Alignment.topCenter,
              child: _error == null
                  ? const SizedBox(width: double.infinity)
                  : Padding(
                      padding: const EdgeInsets.only(top: GfSpace.sm),
                      child: Semantics(
                        liveRegion: true,
                        child: GfText(
                          strings.failure(_error),
                          textAlign: TextAlign.center,
                          style: typography.caption.copyWith(
                            color: colors.error,
                          ),
                        ),
                      ),
                    ),
            ),
          ],
        ],
      ),
    );
  }
}

import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../providers.dart';
import 'sticker_strings.dart';

/// Resolves shared personal assets which are intentionally absent from the
/// official picker; stale site/account responses never update a new session.
class ResolvedStickerContent extends ConsumerStatefulWidget {
  const ResolvedStickerContent({
    super.key,
    required this.content,
    required this.builder,
    this.errorAlignment = CrossAxisAlignment.start,
    this.resolve = true,
  });
  final String content;
  final Widget Function(Map<String, String>) builder;
  final CrossAxisAlignment errorAlignment;

  /// Share captures freeze the already-known token map instead of starting
  /// late network resolution after export begins.
  final bool resolve;
  @override
  ConsumerState<ResolvedStickerContent> createState() =>
      _ResolvedStickerContentState();
}

class _ResolvedStickerContentState
    extends ConsumerState<ResolvedStickerContent> {
  StickerLibrary? _library;
  Set<String>? _names;
  bool _failed = false;
  Map<String, String>? _frozenUrls;
  @override
  Widget build(BuildContext context) {
    final library = ref.watch(stickerLibraryProvider);
    if (!widget.resolve) {
      return widget.builder(
        _frozenUrls ??= Map<String, String>.unmodifiable(library.urlByName),
      );
    }
    final names = stickerTokenPattern
        .allMatches(widget.content)
        .map((match) => match.group(1)!)
        .toSet();
    // Ordinary draft edits do not change the assets that need resolution.
    // Keep unavailable names stable until the tokens change or the user retries.
    if (_library != library || !setEquals(_names, names)) {
      _library = library;
      _names = names;
      _failed = false;
      final content = widget.content;
      library.resolveContent(content).catchError((Object _) {
        if (mounted &&
            identical(_library, library) &&
            identical(_names, names)) {
          setState(() => _failed = true);
        }
      });
    }
    return ListenableBuilder(
      listenable: library,
      builder: (context, _) {
        final content = widget.builder(library.urlByName);
        if (!_failed) return content;
        final strings = StickerStrings(context);
        return Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: widget.errorAlignment,
          children: [
            content,
            Tooltip(
              message: strings.failed,
              child: TextButton.icon(
                onPressed: () => setState(() => _names = null),
                icon: const GfSymbol('refresh-cw', size: 16),
                label: Text(strings.retry),
              ),
            ),
          ],
        );
      },
    );
  }
}

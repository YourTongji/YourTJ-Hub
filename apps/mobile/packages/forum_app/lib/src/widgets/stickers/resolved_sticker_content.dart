import 'package:core/core.dart';
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
  });
  final String content;
  final Widget Function(Map<String, String>) builder;
  final CrossAxisAlignment errorAlignment;
  @override
  ConsumerState<ResolvedStickerContent> createState() =>
      _ResolvedStickerContentState();
}

class _ResolvedStickerContentState
    extends ConsumerState<ResolvedStickerContent> {
  StickerLibrary? _library;
  String? _content;
  bool _failed = false;
  @override
  Widget build(BuildContext context) {
    final library = ref.watch(stickerLibraryProvider);
    if (_library != library || _content != widget.content) {
      _library = library;
      _content = widget.content;
      _failed = false;
      final content = widget.content;
      library.resolveContent(content).catchError((Object _) {
        if (mounted && identical(_library, library) && _content == content) {
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
                onPressed: () => setState(() => _content = null),
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

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../providers.dart';
import '../stickers/sticker_image.dart';

/// Rich-editor sticker embeds (MADR 0106 supplement): the published inline
/// image at 2.5× the surrounding font size, centred on the text line. A name
/// that does not resolve shows its token as text, as the published post does.
class StickerEmbedBuilder extends EmbedBuilder {
  const StickerEmbedBuilder();

  @override
  String get key => stickerEmbedType;

  @override
  bool get expanded => false;

  @override
  WidgetSpan buildWidgetSpan(Widget widget) =>
      WidgetSpan(alignment: PlaceholderAlignment.middle, child: widget);

  // Copying a sticker copies its token, which pastes back as the sticker.
  @override
  String toPlainText(Embed node) => stickerEmbedToken(node);

  @override
  Widget build(BuildContext context, EmbedContext embedContext) {
    return _EditorSticker(
      name: embedContext.node.value.data as String,
      textStyle: embedContext.textStyle,
    );
  }
}

class _EditorSticker extends ConsumerStatefulWidget {
  const _EditorSticker({required this.name, required this.textStyle});

  final String name;
  final TextStyle textStyle;

  @override
  ConsumerState<_EditorSticker> createState() => _EditorStickerState();
}

class _EditorStickerState extends ConsumerState<_EditorSticker> {
  StickerLibrary? _requested;

  @override
  Widget build(BuildContext context) {
    final library = ref.watch(stickerLibraryProvider);
    return ListenableBuilder(
      listenable: library,
      builder: (context, _) {
        final url = library.urlByName[widget.name];
        final token = '[:sticker:${widget.name}:]';
        if (url == null || url.isEmpty) {
          if (!identical(_requested, library)) {
            _requested = library;
            library.resolveContent(token).catchError((Object _) {});
          }
          return Text(token, style: widget.textStyle);
        }
        return IgnorePointer(
          child: StickerImage(
            name: widget.name,
            url: url,
            size: MediaQuery.textScalerOf(context).scale(
                  widget.textStyle.fontSize ?? 16,
                ) *
                StickerImage.inlineScale,
            collectible: false,
            excludeSemantics: true,
          ),
        );
      },
    );
  }
}

import 'package:flutter/material.dart';

import 'package:core/core.dart';

import 'stickers/sticker_image.dart';

/// Only resolved stickers and surrounding whitespace can omit a chat bubble.
/// Unknown/disabled tokens remain text and need a readable message surface.
bool isStickerOnlyMessage(String content, Map<String, String> urlByName) {
  final segments = parseStickerSegments(content, urlByName);
  return segments.any((segment) => segment is StickerImageSegment) &&
      segments.every(
        (segment) => switch (segment) {
          StickerTextSegment(:final text) => text.trim().isEmpty,
          StickerImageSegment(:final url) => url.isNotEmpty,
        },
      );
}

/// 私信气泡的表情包分段渲染:把 `[:sticker:name:]` token 组装成 Text.rich
/// 可用的 [InlineSpan](文本段保留 + 已知表情内联图片)。
///
/// 无 token、或 token 全部未知/停用时返回 null,调用方回退纯文本路径
/// (对齐 web 消息气泡的安全分段渲染语义:未识别 token 保持原文)。
InlineSpan? buildStickerMessageSpan(
  String content,
  Map<String, String> urlByName,
) {
  if (!containsStickerToken(content)) return null;
  final List<StickerMessageSegment> segments = parseStickerSegments(
    content,
    urlByName,
  );
  if (segments.length == 1 && segments.single is StickerTextSegment) {
    return null;
  }
  return TextSpan(
    children: <InlineSpan>[
      for (final StickerMessageSegment segment in segments)
        switch (segment) {
          StickerTextSegment(:final text) => TextSpan(text: text),
          StickerImageSegment(:final name, :final url) => WidgetSpan(
            alignment: PlaceholderAlignment.middle,
            child: StickerImage(name: name, url: url),
          ),
        },
    ],
  );
}

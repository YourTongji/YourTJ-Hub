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
///
/// [dropUnknownStickers] 面向只读摘要(列表行/卡片):未识别 token 直接省略而
/// 不是原文展示,省略后无内容可渲染时返回 null 表示该摘要应整段隐藏。
/// [collectible] 关闭表情收藏入口;列表行由外层负责点击与长按。
InlineSpan? buildStickerMessageSpan(
  String content,
  Map<String, String> urlByName, {
  bool deferLongPress = false,
  bool collectible = true,
  bool dropUnknownStickers = false,
  double stickerSize = StickerImage.readingSize,
}) {
  if (!containsStickerToken(content)) return null;
  String strip(String value) =>
      dropUnknownStickers ? value.replaceAll(stickerTokenPattern, '') : value;
  final List<StickerMessageSegment> segments = parseStickerSegments(
    content,
    urlByName,
  );
  if (segments.length == 1 && segments.single is StickerTextSegment) {
    // Token-free content keeps the caller's plain path; an excerpt whose tokens
    // were all unresolved has nothing left to show.
    if (!dropUnknownStickers) return null;
    final String remaining = strip(
      (segments.single as StickerTextSegment).text,
    );
    return remaining.trim().isEmpty ? null : TextSpan(text: remaining);
  }
  return TextSpan(
    children: <InlineSpan>[
      for (final StickerMessageSegment segment in segments)
        switch (segment) {
          StickerTextSegment(:final text) => TextSpan(text: strip(text)),
          StickerImageSegment(:final name, :final url) => WidgetSpan(
            alignment: PlaceholderAlignment.middle,
            child: StickerImage(
              name: name,
              url: url,
              size: stickerSize,
              collectible: collectible,
              deferLongPress: deferLongPress,
            ),
          ),
        },
    ],
  );
}

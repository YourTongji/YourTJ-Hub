import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import 'package:core/core.dart';

import '../asset_url.dart';

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
            child: _StickerBubbleImage(
              name: name,
              url: resolveApiAssetUrl(url),
            ),
          ),
        },
    ],
  );
}

/// 气泡内联表情图(web 气泡 `h-14 w-14` = 56):上限 56,竖图按原始宽高比
/// 缩到 ≤56 高、横图 ≤56 宽;加载完成前占位等宽方块避免行高跳动。
class _StickerBubbleImage extends StatefulWidget {
  const _StickerBubbleImage({required this.name, required this.url});

  final String name;
  final String url;

  @override
  State<_StickerBubbleImage> createState() => _StickerBubbleImageState();
}

class _StickerBubbleImageState extends State<_StickerBubbleImage> {
  static const double boxSize = 56;

  ImageStream? _imageStream;
  ImageStreamListener? _imageListener;
  ImageInfo? _imageInfo;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _resolveImage();
  }

  @override
  void didUpdateWidget(covariant _StickerBubbleImage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url != widget.url) _resolveImage();
  }

  void _resolveImage() {
    final ImageStream newStream = NetworkImage(
      widget.url,
    ).resolve(createLocalImageConfiguration(context));
    if (newStream.key == _imageStream?.key) return;
    final ImageStreamListener? oldListener = _imageListener;
    if (oldListener != null) _imageStream?.removeListener(oldListener);
    _imageStream = newStream;
    _imageListener = ImageStreamListener(_handleImage, onError: (_, _) {});
    newStream.addListener(_imageListener!);
  }

  void _handleImage(ImageInfo info, bool synchronousCall) {
    final ImageInfo? previous = _imageInfo;
    _imageInfo = info;
    previous?.dispose();
    if (synchronousCall) return;
    setState(() {});
  }

  @override
  void dispose() {
    final ImageStreamListener? listener = _imageListener;
    if (listener != null) _imageStream?.removeListener(listener);
    _imageInfo?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final ui.Image? image = _imageInfo?.image;
    final Widget content;
    if (image == null) {
      content = SizedBox(width: boxSize, height: boxSize);
    } else {
      final double ratio = image.height / image.width;
      double width = boxSize;
      double height = boxSize * ratio;
      if (height > boxSize) {
        height = boxSize;
        width = boxSize / ratio;
      }
      content = SizedBox(
        width: width,
        height: height,
        child: RawImage(image: image, fit: BoxFit.contain),
      );
    }
    // 语义标注(表情名)对齐 web img alt,占位与已加载状态一致。
    return Semantics(label: widget.name, child: content);
  }
}

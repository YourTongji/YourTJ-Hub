import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';

import 'image_gallery_saver.dart'
    if (dart.library.io) 'image_gallery_saver_io.dart';

/// Downloads an image with the same URL used by the viewer and hands it to
/// the platform's native save surface. Mobile stores it in the photo gallery;
/// Web uses share_plus' browser download fallback.
Future<void> saveImageFromUrl(BuildContext context, String imageUrl) async {
  final AppLocalizations l10n = AppLocalizations.of(context);
  try {
    final Response<List<int>> response = await Dio().get<List<int>>(
      imageUrl,
      options: Options(responseType: ResponseType.bytes),
    );
    final List<int>? data = response.data;
    if (data == null || data.isEmpty) throw StateError('empty image response');

    final Uint8List bytes = Uint8List.fromList(data);
    final String fileName = _fileNameFor(imageUrl);
    final bool isMobile =
        !kIsWeb &&
        (defaultTargetPlatform == TargetPlatform.android ||
            defaultTargetPlatform == TargetPlatform.iOS);

    if (isMobile) {
      if (await saveImageToGallery(bytes, fileName)) {
        if (context.mounted) showGfToast(context, l10n.imageSaved);
      } else if (context.mounted) {
        showGfToast(context, l10n.imageSaveFailed, error: true);
      }
      return;
    }

    final ShareResult result = await SharePlus.instance.share(
      ShareParams(
        files: <XFile>[
          XFile.fromData(bytes, name: fileName, mimeType: _mimeType(fileName)),
        ],
        fileNameOverrides: <String>[fileName],
        downloadFallbackEnabled: true,
      ),
    );
    // On Web, share_plus reports `unavailable` after successfully using its
    // browser download fallback, so a thrown error is the only failure signal.
    if (context.mounted &&
        (kIsWeb || result.status != ShareResultStatus.dismissed)) {
      showGfToast(context, l10n.imageSaved);
    }
  } catch (_) {
    if (context.mounted) {
      showGfToast(context, l10n.imageSaveFailed, error: true);
    }
  }
}

/// Shares an image file through the host platform's system share surface.
/// Web falls back to the browser download behavior exposed by share_plus.
Future<void> shareImageFromUrl(BuildContext context, String imageUrl) async {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final RenderBox? box = context.findRenderObject() as RenderBox?;
  try {
    final Response<List<int>> response = await Dio().get<List<int>>(
      imageUrl,
      options: Options(responseType: ResponseType.bytes),
    );
    final List<int>? data = response.data;
    if (data == null || data.isEmpty) throw StateError('empty image response');

    final String fileName = _fileNameFor(imageUrl);
    await SharePlus.instance.share(
      ShareParams(
        files: <XFile>[
          XFile.fromData(
            Uint8List.fromList(data),
            name: fileName,
            mimeType: _mimeType(fileName),
          ),
        ],
        fileNameOverrides: <String>[fileName],
        sharePositionOrigin: box == null
            ? null
            : box.localToGlobal(Offset.zero) & box.size,
        downloadFallbackEnabled: true,
      ),
    );
  } catch (_) {
    if (context.mounted) {
      showGfToast(context, l10n.imageSaveFailed, error: true);
    }
  }
}

String _fileNameFor(String imageUrl) {
  final Uri? uri = Uri.tryParse(imageUrl);
  final String candidate = uri?.pathSegments.lastOrNull ?? '';
  final String clean = candidate.split('?').first.trim();
  final String safe = clean.replaceAll(RegExp(r'[^A-Za-z0-9._-]'), '_');
  if (safe.isEmpty || !safe.contains('.')) {
    return 'yourtj-image-${DateTime.now().millisecondsSinceEpoch}.jpg';
  }
  return safe;
}

String _mimeType(String fileName) {
  final String extension = fileName.split('.').last.toLowerCase();
  return switch (extension) {
    'gif' => 'image/gif',
    'png' => 'image/png',
    'webp' => 'image/webp',
    'bmp' => 'image/bmp',
    'avif' => 'image/avif',
    _ => 'image/jpeg',
  };
}

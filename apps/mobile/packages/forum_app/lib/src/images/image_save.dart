import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../providers.dart';

import 'image_gallery_saver.dart'
    if (dart.library.io) 'image_gallery_saver_io.dart';

/// Downloads an image with the same URL used by the viewer and hands it to
/// the platform's native save surface. Mobile stores it in the photo gallery;
/// Web uses share_plus' browser download fallback.
Future<void> saveImageFromUrl(BuildContext context, String imageUrl) async {
  final AppLocalizations l10n = AppLocalizations.of(context);
  try {
    final Uint8List bytes = await _downloadImageBytes(context, imageUrl);
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
    final Uint8List bytes = await _downloadImageBytes(context, imageUrl);
    final String fileName = _fileNameFor(imageUrl);
    await SharePlus.instance.share(
      ShareParams(
        files: <XFile>[
          XFile.fromData(bytes, name: fileName, mimeType: _mimeType(fileName)),
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

/// Downloads an image using the same origin and session as the mobile app.
///
/// Image URLs are normally public, but attachment providers can still return
/// a session-protected response. A fresh bare Dio instance used to lose the
/// current bearer token and made saving fail even though the viewer had loaded
/// the image successfully. Only same-origin requests receive the token; CDN
/// URLs must never receive the forum session credential.
Future<Uint8List> _downloadImageBytes(
  BuildContext context,
  String imageUrl,
) async {
  final String resolvedUrl = resolveApiAssetUrl(imageUrl);
  final Uri? imageUri = Uri.tryParse(resolvedUrl);
  if (imageUri == null ||
      (imageUri.scheme != 'http' && imageUri.scheme != 'https')) {
    throw FormatException('unsupported image URL');
  }

  final Dio dio = Dio(
    BaseOptions(
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 30),
      sendTimeout: const Duration(seconds: 10),
      followRedirects: false,
    ),
  );
  final String? token = await _sessionToken(context);
  final Uri apiOrigin = Uri.parse(resolveApiAssetUrl('/'));

  DioException? lastError;
  for (int attempt = 0; attempt < 3; attempt++) {
    if (attempt > 0) {
      await Future<void>.delayed(Duration(milliseconds: 250 * attempt));
    }
    try {
      Uri requestUri = imageUri;
      for (int redirect = 0; redirect < 4; redirect++) {
        final Map<String, dynamic> headers = <String, dynamic>{
          'Accept':
              'image/avif,image/webp,image/apng,image/svg+xml,image/*;q=0.8',
        };
        if (token != null &&
            token.isNotEmpty &&
            _sameOrigin(requestUri, apiOrigin)) {
          headers['Authorization'] = 'Bearer $token';
        }
        final Response<List<int>> response = await dio.getUri<List<int>>(
          requestUri,
          options: Options(
            responseType: ResponseType.bytes,
            headers: headers,
            followRedirects: false,
            validateStatus: (int? status) =>
                status != null && status >= 200 && status < 400,
          ),
        );
        final int status = response.statusCode ?? 0;
        if (status >= 300 && status < 400) {
          final String? location = response.headers.value('location');
          if (location == null || location.isEmpty) {
            throw StateError('image redirect has no location');
          }
          requestUri = requestUri.resolve(location);
          continue;
        }
        final List<int>? data = response.data;
        if (data == null || data.isEmpty) {
          throw StateError('empty image response');
        }
        final String contentType =
            response.headers.value(Headers.contentTypeHeader)?.toLowerCase() ??
            '';
        if (contentType.startsWith('text/') ||
            contentType.contains('application/json')) {
          throw StateError('image endpoint returned a non-image response');
        }
        return Uint8List.fromList(data);
      }
      throw StateError('too many image redirects');
    } on DioException catch (error) {
      lastError = error;
      if (!_isRetryable(error)) rethrow;
    }
  }
  throw lastError ?? StateError('image download failed');
}

Future<String?> _sessionToken(BuildContext context) async {
  try {
    return await ProviderScope.containerOf(
      context,
      listen: false,
    ).read(tokenStorageProvider).read();
  } catch (_) {
    return null;
  }
}

bool _sameOrigin(Uri left, Uri right) =>
    left.scheme.toLowerCase() == right.scheme.toLowerCase() &&
    left.host.toLowerCase() == right.host.toLowerCase() &&
    _effectivePort(left) == _effectivePort(right);

int _effectivePort(Uri uri) {
  if (uri.hasPort) return uri.port;
  return uri.scheme.toLowerCase() == 'https' ? 443 : 80;
}

bool _isRetryable(DioException error) {
  if (error.type == DioExceptionType.connectionError ||
      error.type == DioExceptionType.connectionTimeout ||
      error.type == DioExceptionType.receiveTimeout ||
      error.type == DioExceptionType.sendTimeout) {
    return true;
  }
  final int? status = error.response?.statusCode;
  return status != null && (status == 408 || status == 429 || status >= 500);
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

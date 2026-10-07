import 'dart:convert';

import 'package:dio/dio.dart';

const appStoreAppId = '6809457637';
const appStoreBundleId = 'tj.yourtj.forumApp';

class IosStoreListing {
  const IosStoreListing({required this.version, required this.url});
  final String version;
  final Uri url;

  bool isNewerThan(String installedVersion) =>
      compareVersions(version, installedVersion) > 0;

  static int compareVersions(String left, String right) {
    final a = left.split('.').map((part) => int.tryParse(part) ?? 0).toList();
    final b = right.split('.').map((part) => int.tryParse(part) ?? 0).toList();
    final length = a.length > b.length ? a.length : b.length;
    for (var i = 0; i < length; i++) {
      final comparison = (i < a.length ? a[i] : 0).compareTo(
        i < b.length ? b[i] : 0,
      );
      if (comparison != 0) return comparison;
    }
    return 0;
  }

  static IosStoreListing? parse(Object? value) {
    if (value is! Map ||
        value['resultCount'] != 1 ||
        value['results'] is! List) {
      return null;
    }
    final results = value['results'] as List;
    if (results.length != 1 || results.single is! Map) return null;
    final item = results.single as Map;
    if (item['bundleId'] != appStoreBundleId ||
        item['trackId']?.toString() != appStoreAppId) {
      return null;
    }
    final version = item['version'];
    final url = Uri.tryParse(
      item['trackViewUrl'] is String ? item['trackViewUrl'] as String : '',
    );
    if (version is! String ||
        !RegExp(r'^\d+(?:\.\d+){1,3}$').hasMatch(version) ||
        url == null ||
        url.scheme != 'https' ||
        url.host != 'apps.apple.com' ||
        url.userInfo.isNotEmpty ||
        url.port != 443 ||
        url.fragment.isNotEmpty) {
      return null;
    }
    return IosStoreListing(version: version, url: url);
  }

  static Future<IosStoreListing?> lookup({Dio? dio}) async {
    final client =
        dio ??
        Dio(
          BaseOptions(
            connectTimeout: const Duration(seconds: 4),
            receiveTimeout: const Duration(seconds: 4),
            headers: const {'Accept': 'application/json'},
          ),
        );
    final cancel = CancelToken();
    try {
      final response = await client.get<List<int>>(
        'https://itunes.apple.com/lookup',
        queryParameters: const {'id': appStoreAppId, 'country': 'cn'},
        cancelToken: cancel,
        onReceiveProgress: (received, _) {
          if (received > 256 * 1024) {
            cancel.cancel('Store lookup response too large');
          }
        },
        options: Options(
          responseType: ResponseType.bytes,
          followRedirects: false,
          maxRedirects: 0,
        ),
      );
      final bytes = response.data;
      if (bytes == null || bytes.length > 256 * 1024) return null;
      return parse(jsonDecode(utf8.decode(bytes)));
    } catch (_) {
      return null;
    }
  }
}

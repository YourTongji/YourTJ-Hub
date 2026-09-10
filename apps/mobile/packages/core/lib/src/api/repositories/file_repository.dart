import 'package:dio/dio.dart';

import '../api_error.dart';
import '../gf_api_client.dart';

/// 图片上传接口。
class FileRepository {
  FileRepository(this._client);

  final GfApiClient _client;

  /// 上传通用图片(发帖插图/封面等),成功返回图片 URL。
  Future<String> uploadImage({
    required List<int> bytes,
    required String filename,
  }) {
    return _client.postMultipart<String>(
      '/file/img-upload',
      formData: FormData.fromMap({
        'file': MultipartFile.fromBytes(bytes, filename: filename),
      }),
      parser: (json) {
        if (json is String) return json;
        final url = (json as Map<String, dynamic>)['url'];
        if (url is String && url.isNotEmpty) return url;
        throw const ApiException(fallbackMessage: 'Image upload failed');
      },
    );
  }

  /// 上传头像,成功返回 avatarUrl。
  Future<String> uploadAvatar({
    required List<int> bytes,
    required String filename,
  }) {
    return _client.postMultipart<String>(
      '/api/upload-avatar',
      formData: FormData.fromMap({
        'avatar': MultipartFile.fromBytes(bytes, filename: filename),
      }),
      parser: (json) {
        if (json is String) return json;
        final avatarUrl = (json as Map<String, dynamic>)['avatarUrl'];
        if (avatarUrl is String && avatarUrl.isNotEmpty) return avatarUrl;
        throw const ApiException(fallbackMessage: 'Avatar upload failed');
      },
    );
  }
}

import 'package:dio/dio.dart';

import '../../gen/campus.dart';
import '../gf_api_client.dart';

/// Request-scoped campus data. Never writes school records or credentials to disk.
class CampusRepository {
  CampusRepository(this._client);
  final GfApiClient _client;

  Future<CampusStatus> status({CancelToken? cancelToken}) => _client.get(
    '/api/campus/status',
    cancelToken: cancelToken,
    parser: (v) => CampusStatus.fromJson(v as Map<String, dynamic>),
  );

  Future<CampusDataset> dataset(String key, {CancelToken? cancelToken}) =>
      _client.get(
        '/api/campus/data/${Uri.encodeComponent(key)}',
        cancelToken: cancelToken,
        parser: (v) => CampusDataset.fromJson(v as Map<String, dynamic>),
      );

  Future<CampusMessageDetail> message(String id, {CancelToken? cancelToken}) =>
      _client.get(
        '/api/campus/messages/${Uri.encodeComponent(id)}',
        cancelToken: cancelToken,
        parser: (v) => CampusMessageDetail.fromJson(v as Map<String, dynamic>),
      );

  Future<Uri> start(String mode, {CancelToken? cancelToken}) => _client.post(
    '/api/campus/tongji/start',
    body: {'mode': mode},
    cancelToken: cancelToken,
    parser: (v) => Uri.parse((v as Map<String, dynamic>)['url'] as String),
  );

  Future<void> confirm({CancelToken? cancelToken}) async {
    await _client.post<Object?>(
      '/api/campus/tongji/confirm',
      cancelToken: cancelToken,
    );
  }

  Future<void> unbind(String revision, {CancelToken? cancelToken}) async {
    await _client.post<Object?>(
      '/api/campus/tongji/unbind',
      body: {'revision': revision},
      cancelToken: cancelToken,
    );
  }
}

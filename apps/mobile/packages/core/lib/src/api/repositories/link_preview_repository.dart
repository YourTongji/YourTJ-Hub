import '../../gen/link_preview.dart';
import '../../markdown/link_preview_candidate.dart';
import '../gf_api_client.dart';

class LinkPreviewRepository {
  LinkPreviewRepository(this._client);

  final GfApiClient _client;

  Future<List<LinkPreviewPayload>> resolve(List<String> urls) {
    return _client.post<List<LinkPreviewPayload>>(
      '/api/link-previews/resolve',
      body: <String, Object>{
        'urls': urls.take(maxLinkPreviewCandidates).toList(growable: false),
      },
      parser: (Object? json) => (json as List<dynamic>? ?? const <dynamic>[])
          .map(
            (dynamic item) => LinkPreviewPayload.fromJson(
              Map<String, dynamic>.from(item as Map<dynamic, dynamic>),
            ),
          )
          .toList(growable: false),
    );
  }
}

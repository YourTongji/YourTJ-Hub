import '../gf_api_client.dart';
import '../../gen/anonymous_identity.dart';

class AnonymousIdentityRepository {
  AnonymousIdentityRepository(this.client);
  final GfApiClient client;
  Future<AnonymousIdentityState> state() => client.get(
    '/api/forum/anonymous/state',
    headers: {'Cache-Control': 'no-store'},
    parser: (j) => AnonymousIdentityState.fromJson(j as Map<String, dynamic>),
  );
  Future<AnonymousNameBatch> generate(String day, String requestKey) =>
      client.post(
        '/api/forum/anonymous/batches',
        body: {'day': day, 'requestKey': requestKey},
        parser: (j) => AnonymousNameBatch.fromJson(j as Map<String, dynamic>),
      );
  Future<AnonymousPersona> confirm(String batchId, int index) => client.post(
    '/api/forum/anonymous/confirm',
    body: {'batchId': batchId, 'index': index},
    parser: (j) => AnonymousPersona.fromJson(j as Map<String, dynamic>),
  );
  Future<void> setDisabled(bool disabled) async {
    await client.post<Object?>(
      '/api/forum/anonymous/disable',
      body: {'disabled': disabled},
    );
  }

  Future<void> setProfileContent(bool showContent) async {
    await client.post<Object?>(
      '/api/forum/anonymous/privacy',
      body: {'showContent': showContent},
    );
  }

  Future<void> govern(int postId, bool disabled, String reason) async {
    await client.post<Object?>(
      '/api/forum/anonymous/govern',
      body: {'postId': postId, 'disabled': disabled, 'reason': reason},
    );
  }

  Future<AnonymousReveal> reveal(String publicUid, String reason) =>
      client.post(
        '/api/forum/anonymous/reveal',
        body: {'publicUid': publicUid, 'reason': reason},
        parser: (j) => AnonymousReveal.fromJson(j as Map<String, dynamic>),
      );
}

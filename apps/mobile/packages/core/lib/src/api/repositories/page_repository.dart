import '../../gen/page.dart';
import '../gf_api_client.dart';

/// 页面级数据通道:请求头 X-Goose-Page: true 时,
/// 页面路由(/, /p/post/:id, /c/:slug/:id, /u/:userId 等)直接返回 PagePayload JSON。
class PageRepository {
  PageRepository(this._client);

  final GfApiClient _client;

  static const _headers = {GfApiClient.pageRequestHeader: 'true'};

  Future<PagePayload> fetch(String path) async {
    return _client.get<PagePayload>(
      path,
      headers: _headers,
      parser: (json) => PagePayload.fromJson(json as Map<String, dynamic>),
    );
  }

  /// 首页。sort: hot | latest | ...(与 web 端一致)。
  Future<PagePayload> home({String sort = ''}) {
    return fetch(sort.isEmpty ? '/' : '/?sort=$sort');
  }

  /// 话题详情页。
  Future<PagePayload> topicDetail(int topicId, {int? postNo}) {
    return fetch(postNo == null ? '/p/post/$topicId' : '/p/post/$topicId/$postNo');
  }

  /// 分类页。
  Future<PagePayload> category(String slug, int id, {String sort = ''}) {
    return fetch(sort.isEmpty ? '/c/$slug/$id' : '/c/$slug/$id/l/$sort');
  }

  /// 用户主页。
  Future<PagePayload> userProfile(int userId, {String section = ''}) {
    return fetch(section.isEmpty ? '/u/$userId' : '/u/$userId/$section');
  }
}

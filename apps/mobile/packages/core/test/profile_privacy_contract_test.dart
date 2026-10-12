// issue #1124: profile privacy mirrors. Settings use the shared contract
// fixtures; profile visibility tolerates servers that predate the field.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'user_profile_props_test.dart' show prodShapeUserProfileJson;

Map<String, dynamic> _fixture(String name) =>
    jsonDecode(
          File(
            '../../../../packages/api-contract/fixtures/$name.json',
          ).readAsStringSync(),
        )
        as Map<String, dynamic>;

class _NoToken implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _Adapter implements HttpClientAdapter {
  _Adapter(this.result);
  final Map<String, dynamic> result;
  final List<RequestOptions> requests = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    return ResponseBody.fromString(
      jsonEncode(result),
      200,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  test('settings parse the shared default and updated fixtures', () {
    final defaults = UserPrivacySettingsPayload.fromJson(
      _fixture('privacy-settings-default')['result'] as Map<String, dynamic>,
    );
    for (final key in profilePrivacyAudiences.keys) {
      expect(defaults.audienceOf(key), 'everyone', reason: key);
    }
    final updated = UserPrivacySettingsPayload.fromJson(
      _fixture('privacy-settings-updated')['result'] as Map<String, dynamic>,
    );
    expect(updated.audienceOf('profileTopics'), 'self');
    expect(updated.audienceOf('followersList'), 'mutual');
  });

  test('allowed audiences mirror the server', () {
    expect(profilePrivacyAudiences['profileLikes'], [
      'everyone',
      'followers',
      'self',
    ]);
    expect(
      profilePrivacyAudiences['followingList'],
      isNot(contains('members')),
    );
    expect(profilePrivacyAudiences['profileTopics'], hasLength(5));
  });

  test('profiles from servers without privacy show every section', () {
    final props = UserProfileProps.fromJson(prodShapeUserProfileJson());
    expect(props.visibility, const UserProfileVisibilityPayload());
    expect(props.visibility.profileLikes.visible, isTrue);
  });

  test('hidden sections keep their reason; owners get the audience', () {
    final props = UserProfileProps.fromJson(
      prodShapeUserProfileJson()
        ..['visibility'] = {
          'profileTopics': {'visible': true, 'audience': 'self'},
          'profileActivity': {'visible': true},
          'profileLikes': {'visible': false, 'reason': 'followers'},
          'followingList': {'visible': false, 'reason': 'signin'},
          'followersList': {'visible': false, 'reason': 'private'},
        },
    );
    expect(props.visibility.profileTopics.audience, 'self');
    expect(props.visibility.profileLikes.visible, isFalse);
    expect(props.visibility.profileLikes.reason, 'followers');
    expect(props.visibility.followingList.reason, 'signin');
    final roundTrip = UserProfileProps.fromJson(
      jsonDecode(jsonEncode(props.toJson())) as Map<String, dynamic>,
    );
    expect(roundTrip, props);
    expect(props.copyWith(settingsUrl: '/x').visibility, props.visibility);
  });

  test('repository reads and patches one section', () async {
    final dio = Dio();
    final adapter = _Adapter(_fixture('privacy-settings-updated'));
    dio.httpClientAdapter = adapter;
    final repository = UserRepository(
      GfApiClient(dio: dio, tokenStorage: _NoToken(), baseUrl: 'http://test'),
    );
    final read = await repository.getPrivacySettings();
    expect(read.audienceOf('profileTopics'), 'self');
    final saved = await repository.updatePrivacySetting(
      'followersList',
      'mutual',
    );
    expect(saved.audienceOf('followersList'), 'mutual');
    expect(adapter.requests.first.method, 'GET');
    expect(adapter.requests.last.method, 'PATCH');
    expect(adapter.requests.last.path, '/api/user/privacy-settings');
    expect(adapter.requests.last.data, {'followersList': 'mutual'});
  });
}

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

Map<String, dynamic> layoutJson() => {
  'site': {
    'name': '',
    'description': '',
    'logo': '',
    'favicon': '',
    'brandType': '',
    'brandText': '',
    'brandImage': '',
  },
  'viewer': {
    'id': 0,
    'username': '',
    'email': '',
    'avatarUrl': '',
    'isAuthenticated': false,
    'canAccessAdmin': false,
    'isModerator': false,
    'requiresEmailVerification': false,
  },
  'sidebar': {'categories': [], 'activeKey': ''},
  'footer': {'links': [], 'primary': []},
  'unread': {'notifications': false, 'messages': false},
  'theme': {'enabled': false, 'current': 'gf-light', 'themeColor': ''},
};

void main() {
  test('layout preserves server posting limit', () {
    final input = layoutJson()..['posting'] = {'maxTitleLength': 42};
    final payload = LayoutPayload.fromJson(input);
    expect(payload.posting?.maxTitleLength, 42);
    expect(payload.posting?.toJson(), {'maxTitleLength': 42});
  });

  test('older layout without posting limits remains readable', () {
    expect(LayoutPayload.fromJson(layoutJson()).posting, isNull);
  });
}

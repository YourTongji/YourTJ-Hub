import 'dart:convert';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'api_client_test.dart' show MockAdapter, ResponseData;
import 'push_repository_test.dart' show Tokens;

Map<String, dynamic> style(String kind) => {
  'code': 'test-$kind',
  'name': 'Test',
  'kind': kind,
  'lightColors': ['#123456', '#654321'],
  'darkColors': ['#abcdef', '#fedcba'],
  'isEnabled': true,
  'sortOrder': 1,
};
void main() {
  test(
    'frame scale is bounded, defaults for old payloads and survives caching',
    () {
      for (final value in [0.5, 1, 1.4, 2]) {
        final parsed = DecorationStyle.fromJson({
          ...style('avatarFrame'),
          'assetScale': value,
        });
        expect(parsed.toJson()['assetScale'], value.toDouble());
        expect(
          DecorationStyle.fromJson(
            jsonDecode(jsonEncode(parsed.toJson())),
          ).assetScale,
          value.toDouble(),
        );
      }
      for (final value in [
        null,
        0,
        -1,
        0.49,
        2.01,
        double.nan,
        double.infinity,
        '1.4',
      ]) {
        final parsed = DecorationStyle.fromJson({
          ...style('avatarFrame'),
          'assetScale': value,
        });
        expect(parsed.toJson()['assetScale'], 1.25);
      }
    },
  );
  test('only safe local PNG and WebP paths are accepted as frame art', () {
    const path = '/static/decorations/custom-frame.png';
    final data = {...style('avatarFrame'), 'assetUrl': path};
    final parsed = DecorationStyle.parse(data, 'avatarFrame')!;
    expect(parsed.assetUrl, path);
    expect(parsed.toJson()['assetUrl'], path);
    for (final invalid in [
      'https://outside.test$path',
      '$path?v=1',
      '/static/decorations/../custom.png',
      '/static/decorations/custom.svg',
    ]) {
      expect(
        DecorationStyle.fromJson({...data, 'assetUrl': invalid}).assetUrl,
        isNull,
      );
    }
    expect(
      DecorationStyle.fromJson({...data, 'kind': 'nicknameStyle'}).assetUrl,
      isNull,
    );
  });
  test(
    'one-stop palettes remain one stop and out-of-contract palettes fall back',
    () {
      for (final kind in ['avatarFrame', 'nicknameStyle']) {
        final single = {
          ...style(kind),
          'lightColors': ['#123456'],
          'darkColors': ['#abcdef'],
        };
        final parsed = DecorationStyle.parse(single, kind)!;
        expect(parsed.lightColors, ['#123456']);
        expect(parsed.toJson()['darkColors'], ['#abcdef']);
        for (final invalid in [
          <String>[],
          ['#123456', '#654321', '#000000'],
        ]) {
          expect(
            DecorationStyle.parse({...single, 'lightColors': invalid}, kind),
            isNull,
          );
        }
      }
    },
  );
  test(
    'old and unknown appearance payloads fall back while cache serialization preserves equipment',
    () {
      final old = UserBriefPayload.fromJson({
        'id': 1,
        'username': 'u',
        'avatarUrl': '',
      });
      expect(old.decoration, isNull);
      expect(
        UserBriefPayload.fromJson({
          ...old.toJson(),
          'decoration': 'unknown',
        }).decoration,
        isNull,
      );
      final appearance = UserDecoration.fromJson({
        'avatarFrame': style('avatarFrame'),
        'nicknameStyle': style('nicknameStyle'),
      });
      final decoded = UserDecoration.fromJson(appearance.toJson());
      expect(decoded.avatarFrame!.code, 'test-avatarFrame');
      final brief = old.copyWith(decoration: appearance);
      expect(
        UserBriefPayload.fromJson(
          jsonDecode(jsonEncode(brief.toJson())),
        ).decoration!.avatarFrame!.code,
        'test-avatarFrame',
      );
      expect(decoded.nicknameStyle!.darkColors, ['#abcdef', '#fedcba']);
      for (final bad in [
        null,
        'bad',
        style('unknown'),
        {
          ...style('nicknameStyle'),
          'lightColors': ['url(x)', 'red'],
        },
        {...style('nicknameStyle'), 'isEnabled': false},
      ]) {
        expect(
          UserDecoration.fromJson({'nicknameStyle': bad}).nicknameStyle,
          isNull,
        );
      }
    },
  );
  test(
    'equip sends both independent slots and explicit empty removal',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio();
      addTearDown(dio.close);
      dio.httpClientAdapter = MockAdapter((request) async {
        requests.add(request);
        return ResponseData(200, {
          'code': 0,
          'result': {
            'catalog': [style('avatarFrame')],
            'ownedCodes': [],
            'equipped': {
              'avatarFrame': '',
              'nicknameStyle': 'test-nicknameStyle',
            },
          },
        });
      });
      final repo = UserRepository(
        GfApiClient(dio: dio, tokenStorage: Tokens()),
      );
      expect((await repo.getDecorations()).ownedCodes, isEmpty);
      await repo.equipDecorations(
        avatarFrame: '',
        nicknameStyle: 'test-nicknameStyle',
      );
      expect(requests[0].path, '/api/user/decorations');
      expect(requests[1].path, '/api/user/decorations/equip');
      expect(requests[1].data, {
        'avatarFrame': '',
        'nicknameStyle': 'test-nicknameStyle',
      });
    },
  );
}

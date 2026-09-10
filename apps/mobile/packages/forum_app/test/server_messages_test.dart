import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/l10n/app_localizations_en.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/server_messages.dart';

void main() {
  final AppLocalizationsZh zh = AppLocalizationsZh();
  final AppLocalizationsEn en = AppLocalizationsEn();

  group('resolveErrorMessage', () {
    test('publishing limit is actionable in both locales', () {
      const error = ApiException(
        fallbackMessage: 'Request failed',
        messageCode: 'topic.dailyLimit',
      );
      expect(resolveErrorMessage(zh, error), contains('明天'));
      expect(resolveErrorMessage(en, error), contains('tomorrow'));
    });
    test('upload limits interpolate the server parameters', () {
      const error = ApiException(
        fallbackMessage: 'Request failed',
        messageCode: 'upload.dailyLimit.avatar',
        params: {'count': 9, 'fileCount': 2},
      );
      expect(resolveErrorMessage(en, error), contains('9 files'));
      expect(resolveErrorMessage(en, error), contains('2 slots'));
      expect(resolveErrorMessage(en, error), isNot(contains('{')));
    });

    group('已知 messageCode 命中本地化目录', () {
      test('zh: topic.notFound 返回中文目录文案', () {
        const error = ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'topic.notFound',
        );
        expect(resolveErrorMessage(zh, error), '内容不存在');
      });

      test('en: topic.notFound 返回英文目录文案', () {
        const error = ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'topic.notFound',
        );
        expect(resolveErrorMessage(en, error), 'Topic not found');
      });
    });

    group('未知 messageCode(目录未命中)', () {
      test('zh: 回退 commonLoadFailed,不暴露英文 fallbackMessage', () {
        const error = ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'some.unknown.code',
        );
        expect(resolveErrorMessage(zh, error), '加载失败');
      });

      test('en: 回退 commonLoadFailed,不暴露 fallbackMessage', () {
        const error = ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'some.unknown.code',
        );
        expect(resolveErrorMessage(en, error), 'Failed to load');
      });

      test('zh: 原始错误码绝不展示给用户', () {
        const error = ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'topic.notFound',
        );
        expect(
          resolveErrorMessage(zh, error),
          isNot(contains('topic.notFound')),
        );
      });
    });

    group('缺失 messageCode', () {
      test('zh: 无 messageCode 回退 commonLoadFailed', () {
        const error = ApiException(fallbackMessage: 'Failed to load');
        expect(resolveErrorMessage(zh, error), '加载失败');
      });

      test('en: 无 messageCode 回退 commonLoadFailed', () {
        const error = ApiException(fallbackMessage: 'Failed to load');
        expect(resolveErrorMessage(en, error), 'Failed to load');
      });

      test('zh: 空白 messageCode 视为缺失,回退 commonLoadFailed', () {
        const error = ApiException(
          fallbackMessage: 'Failed to load',
          messageCode: '   ',
        );
        expect(resolveErrorMessage(zh, error), '加载失败');
      });
    });

    group('非 ApiException', () {
      test('页面自身业务错误文案保留原样', () {
        expect(resolveErrorMessage(zh, '草稿已保存'), '草稿已保存');
        expect(resolveErrorMessage(zh, 404), '404');
      });
    });
  });
}

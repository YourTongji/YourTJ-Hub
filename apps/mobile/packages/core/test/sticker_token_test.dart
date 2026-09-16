import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('expandStickerTokens', () {
    const Map<String, String> urlByName = <String, String>{
      'smile': '/file/img/stickers/smile.png',
      '哭哭': '/file/img/stickers/cry.png',
    };

    for (final source in <String>[
      '`[:sticker:smile:]`',
      '```\n[:sticker:smile:]\n```',
      '    [:sticker:smile:]',
      '[[:sticker:smile:]](/target)',
      '[**[:sticker:smile:]**](/target)',
      '[link](https://example.test/[:sticker:smile:])',
      '![image](https://example.test/[:sticker:smile:])',
      '<https://example.test/[:sticker:smile:]>',
      '[ref]: https://example.test/[:sticker:smile:]\n\n[link][ref]',
      '<div>[:sticker:smile:]</div>',
      'before <span title="[:sticker:smile:]">text</span> after',
      r'$[:sticker:smile:]$',
      r'$$[:sticker:smile:]$$',
      '\$\$\n\n[:sticker:smile:]\n\n\$\$',
      r'\begin{align}[:sticker:smile:]\end{align}',
      r'\([:sticker:smile:]\)',
      r'\[[:sticker:smile:]\]',
      r'\[:sticker:smile:]',
    ]) {
      test('preserves Markdown context: $source', () {
        expect(expandStickerTokens(source, urlByName), source);
      });
    }
    test('escapes Markdown destination punctuation', () {
      expect(
        expandStickerTokens('[:sticker:smile:]', {'smile': '/a)b (c.png'}),
        '![sticker:smile](/a%29b%20%28c.png)',
      );
    });

    test('无 token 快路径原样返回同一字符串', () {
      const String content = '普通文本 #标题 ![img](/a.png) [:sticker 没有token';
      expect(expandStickerTokens(content, urlByName), same(content));
    });

    test('基本展开为标准图片语法', () {
      expect(
        expandStickerTokens('你好 [:sticker:smile:]!', urlByName),
        '你好 ![sticker:smile](/file/img/stickers/smile.png)!',
      );
    });

    test('未知/停用 token 保持原文', () {
      const String content = '前 [:sticker:unknown:] 后';
      expect(expandStickerTokens(content, urlByName), same(content));
    });

    test('多 token 混合未知,已知展开未知保留', () {
      expect(
        expandStickerTokens(
          '[:sticker:smile:][:sticker:ghost:][:sticker:哭哭:]',
          urlByName,
        ),
        '![sticker:smile](/file/img/stickers/smile.png)[:sticker:ghost:]'
        '![sticker:哭哭](/file/img/stickers/cry.png)',
      );
    });

    test('相邻 token 全部展开', () {
      expect(
        expandStickerTokens('[:sticker:smile:][:sticker:smile:]', urlByName),
        '![sticker:smile](/file/img/stickers/smile.png)'
        '![sticker:smile](/file/img/stickers/smile.png)',
      );
    });

    test('中文/连字符/下划线 name', () {
      const Map<String, String> urls = <String, String>{'a-b_中文': '/a.png'};
      expect(
        expandStickerTokens('[:sticker:a-b_中文:]', urls),
        '![sticker:a-b_中文](/a.png)',
      );
    });

    test('非法形态不误伤:空名/空格/冒号/方括号/缺尾', () {
      const String content =
          '[:sticker::] [:sticker:a b:] [:sticker:a:b:] '
          '[:sticker:a[b]:] [:sticker:abc]';
      expect(expandStickerTokens(content, urlByName), same(content));
    });

    test('name 超 64 字符不匹配', () {
      final String longName = 'a' * 65;
      final String content = '[:sticker:$longName:]';
      expect(expandStickerTokens(content, urlByName), same(content));
      final String maxName = 'a' * 64;
      expect(
        expandStickerTokens('[:sticker:$maxName:]', <String, String>{
          maxName: '/a.png',
        }),
        '![sticker:$maxName](/a.png)',
      );
    });

    test('嵌套 token 展开内层不越界', () {
      expect(
        expandStickerTokens('[:sticker:[:sticker:smile:]:]', urlByName),
        '[:sticker:![sticker:smile](/file/img/stickers/smile.png):]',
      );
    });
  });

  group('parseStickerSegments', () {
    const Map<String, String> urlByName = <String, String>{'smile': '/s.png'};

    test('无 token 快路径单文本段', () {
      final List<StickerMessageSegment> segments = parseStickerSegments(
        '普通消息',
        urlByName,
      );
      expect(segments, hasLength(1));
      expect((segments.single as StickerTextSegment).text, '普通消息');
    });

    test('基本分段:文本/图片/文本', () {
      final List<StickerMessageSegment> segments = parseStickerSegments(
        '前 [:sticker:smile:] 后',
        urlByName,
      );
      expect(segments, hasLength(3));
      expect((segments[0] as StickerTextSegment).text, '前 ');
      expect((segments[1] as StickerImageSegment).url, '/s.png');
      expect((segments[1] as StickerImageSegment).name, 'smile');
      expect((segments[2] as StickerTextSegment).text, ' 后');
    });

    test('未知 token 保持原文,不产生图片段', () {
      final List<StickerMessageSegment> segments = parseStickerSegments(
        '前 [:sticker:ghost:]',
        urlByName,
      );
      expect(segments, hasLength(1));
      expect(
        (segments.single as StickerTextSegment).text,
        '前 [:sticker:ghost:]',
      );
    });

    test('整条消息只有未知 token 时返回单文本段原文', () {
      final List<StickerMessageSegment> segments = parseStickerSegments(
        '[:sticker:ghost:]',
        urlByName,
      );
      expect(segments, hasLength(1));
      expect((segments.single as StickerTextSegment).text, '[:sticker:ghost:]');
    });

    test('混合已知/未知/纯文本', () {
      final List<StickerMessageSegment> segments = parseStickerSegments(
        '[:sticker:smile:]a[:sticker:ghost:]b[:sticker:smile:]',
        urlByName,
      );
      final List<bool> isImage = segments
          .map((StickerMessageSegment s) => s is StickerImageSegment)
          .toList();
      expect(isImage, <bool>[true, false, true]);
    });
  });

  group('stickerPreviewLabel', () {
    test('token 缩写为 [name],无 token 原样', () {
      expect(stickerPreviewLabel('[:sticker:smile:] 早'), '[smile] 早');
      expect(stickerPreviewLabel('普通消息'), '普通消息');
      // 预览层不区分启停:未知 token 同样缩写。
      expect(stickerPreviewLabel('[:sticker:ghost:]'), '[ghost]');
    });
  });
}

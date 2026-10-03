import 'dart:ui' show Color;

import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/widgets/rich_content/gf_html_content.dart';
import 'package:forum_app/src/widgets/rich_content/gf_table.dart';

void main() {
  group('gfPlainTextFromHtml', () {
    test('strips tags and decodes the common entities', () {
      expect(
        gfPlainTextFromHtml('<p>1 &lt; 2 &amp; 3 &gt; 2</p>'),
        '1 < 2 & 3 > 2',
      );
      expect(
        gfPlainTextFromHtml('<p>&quot;it&#39;s&quot; &nbsp; ok</p>'),
        '"it\'s" ok',
      );
    });

    test('collapses block boundaries and line breaks into single spaces', () {
      expect(
        gfPlainTextFromHtml(
          '<h2>课程内容</h2><p>很好</p><ul><li>a</li><li>b</li></ul>',
        ),
        '课程内容 很好 a b',
      );
      expect(gfPlainTextFromHtml('a<br>b<br/>c'), 'a b c');
    });

    test('drops script and style blocks entirely', () {
      expect(
        gfPlainTextFromHtml(
          '<p>a</p><script>alert("x")</script><style>.y { color: red; }</style><p>b</p>',
        ),
        'a b',
      );
    });

    test('collapses runs of whitespace and trims the result', () {
      expect(gfPlainTextFromHtml('<p>a</p>\n\n  <p>  b </p>\t'), 'a b');
    });
  });

  group('gfCssColor', () {
    test('omits the alpha channel for opaque colors', () {
      expect(gfCssColor(const Color(0xFF112233)), '#112233');
    });

    test('puts alpha last, following the CSS4 #rrggbbaa order', () {
      // flutter_widget_from_html reads 8-digit hex as #rrggbbaa; getting this
      // wrong shifts every channel.
      expect(gfCssColor(const Color(0x80112233)), '#11223380');
    });
  });
}

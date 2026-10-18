import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/ascii_hostname.dart';

void main() {
  // 期望值与服务端同源：golang.org/x/net/idna 的 Lookup.ToASCII 结果
  //（apps/gooseforum/app/bundles/domainpolicy 用它校验白名单条目）。
  test('encodes IDN labels to the same punycode the server stores', () {
    expect(asciiHostname('中国.com'), 'xn--fiqs8s.com');
    expect(asciiHostname('日本語.jp'), 'xn--wgv71a119e.jp');
    expect(asciiHostname('bücher.example'), 'xn--bcher-kva.example');
    expect(asciiHostname('münchen.de'), 'xn--mnchen-3ya.de');
    expect(punycodeLabel('中文'), 'fiq228c');
    expect(punycodeLabel('例え'), 'r8jz45g');
    expect(punycodeLabel('ñandú'), 'and-6ma2c');
  });

  test('decodes the percent-encoded hostname Dart returns for IDN URLs', () {
    // Uri.parse('https://中国.com/') 的 host 是百分号编码形式，不是 Unicode。
    final String dartHost = Uri.parse('https://中国.com/路径').host;
    expect(dartHost, contains('%'));
    expect(asciiHostname(dartHost), 'xn--fiqs8s.com');
  });

  test(
    'normalizes case, whitespace, root dots and keeps ASCII hosts intact',
    () {
      expect(asciiHostname('  GitHub.COM. '), 'github.com');
      expect(asciiHostname('mp.weixin.qq.com'), 'mp.weixin.qq.com');
      expect(asciiHostname(''), '');
      // 已经是 punycode 的输入保持原样（幂等）。
      expect(asciiHostname('xn--fiqs8s.com'), 'xn--fiqs8s.com');
    },
  );

  test('keeps malformed input comparable instead of throwing', () {
    expect(asciiHostname('bad%zzhost.example'), 'bad%zzhost.example');
  });
}

import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/link_navigation.dart';

void main() {
  tearDown(() {
    LinkNavigation.clearSessionTrust();
    LinkNavigation.applyTrustedDomains(const <String>[]);
  });

  test('session trust uses the Public Suffix List registrable domain', () {
    expect(
      LinkNavigation.trustDomainFor('docs.team.example.co.uk'),
      'example.co.uk',
    );

    LinkNavigation.trustForSession('docs.team.example.co.uk');
    expect(LinkNavigation.isTrusted('status.example.co.uk'), isTrue);
    expect(LinkNavigation.isTrusted('example.com'), isFalse);
  });

  test('session trust normalizes IDN hosts written from raw Uri.host', () {
    // 生产写入路径传 Uri.host（IDN 的百分号编码形式），读取路径传 asciiHostname
    // 归一后的 punycode；两端必须落在同一个键上，否则「不再询问」对 IDN 失效。
    LinkNavigation.trustForSession(Uri.parse('https://中国.com/x').host);
    expect(LinkNavigation.isTrusted('xn--fiqs8s.com'), isTrue);
    expect(LinkNavigation.isTrusted('apps.xn--fiqs8s.com'), isTrue);
    expect(LinkNavigation.isTrusted('xn--fiqs8s.org'), isFalse);
  });

  test('private suffix tenants do not share trust', () {
    LinkNavigation.trustForSession('alice.github.io');
    expect(LinkNavigation.isTrusted('alice.github.io'), isTrue);
    expect(LinkNavigation.isTrusted('bob.github.io'), isFalse);
  });

  test('policy whitelist entries match the host and its subdomains only', () {
    // 大小写/空白/根点归一化；单标签条目（localhost）被丢弃（issue #1200）。
    LinkNavigation.applyTrustedDomains(<String>[
      '  GitHub.com. ',
      'localhost',
      'github.com',
    ]);
    expect(LinkNavigation.isPolicyTrusted('github.com'), isTrue);
    expect(LinkNavigation.isPolicyTrusted('gist.github.com'), isTrue);
    // 点边界比较：伪装后缀与同级域名都不继承白名单。
    expect(LinkNavigation.isPolicyTrusted('github.com.evil.test'), isFalse);
    expect(LinkNavigation.isPolicyTrusted('notgithub.com'), isFalse);
    expect(LinkNavigation.isPolicyTrusted('localhost'), isFalse);
  });

  test('mislabeledLinkTargets flags URL labels pointing elsewhere', () {
    // 与 Web 的 linkRisk 等价：标签是站外 URL 且 registrable domain 与目标不同。
    expect(
      mislabeledLinkTargets(
        '[https://bank.example/account](https://github.com/x)',
      ),
      <String>{'https://github.com/x'},
    );
    // 同注册域的标签（含子域）不是误导性标签。
    expect(
      mislabeledLinkTargets(
        '[https://docs.example.com](https://example.com/y)',
      ),
      isEmpty,
    );
    // 普通文字标签、相对目标、自动链接都不成卡。
    expect(
      mislabeledLinkTargets(
        '[文档](https://github.com/x) [https://bank.example](/relative) <https://github.com/z>',
      ),
      isEmpty,
    );
  });

  test('mislabeledLinkTargetsInHtml flags mislabeled HTML anchors', () {
    // Wiki 与课评正文是服务端 HTML(GfHtmlContent),判定与 Markdown 链路等价:
    // 标签是站外 URL 且 registrable domain 与目标不同(issue #1200 review)。
    expect(
      mislabeledLinkTargetsInHtml(
        '<p><a href="https://github.com/x">https://bank.example/account</a></p>',
        base: Uri.parse('http://fake.local'),
      ),
      <String>{'https://github.com/x'},
    );
    // 同注册域的标签(含子域)不是误导性标签。
    expect(
      mislabeledLinkTargetsInHtml(
        '<p><a href="https://example.com/y">https://docs.example.com</a></p>',
        base: Uri.parse('http://fake.local'),
      ),
      isEmpty,
    );
    // 普通文字标签、相对目标、非 http(s) 标签都不成卡。
    expect(
      mislabeledLinkTargetsInHtml(
        '<a href="https://github.com/x">文档</a>'
        '<a href="/relative">https://bank.example</a>'
        '<a href="https://github.com/y">mailto:me@example.com</a>',
        base: Uri.parse('http://fake.local'),
      ),
      isEmpty,
    );
  });

  test('mislabeledLinkTargetsInHtml resolves scheme-less targets like fwfh', () {
    // fwfh 先把无 scheme 的 href 按 HtmlWidget 的 baseUrl 解析成绝对 URL 才交给
    // 点击回调,集合键必须用同一形式,否则协议相对目标查不到表(issue #1200)。
    // 集合同时收 href 原样与按 base 解析后的形式:绝对 href 由 fwfh 原样交给
    // 点击回调,而 Wiki 侧会经 Uri.toString() 归一化(issue #1200 review)。
    expect(
      mislabeledLinkTargetsInHtml(
        '<a href="//github.com/x">https://bank.example</a>',
        base: Uri.parse('http://fake.local'),
      ),
      <String>{'//github.com/x', 'http://github.com/x'},
    );
    expect(
      mislabeledLinkTargetsInHtml(
        '<a href="https://GitHub.com/x">https://bank.example</a>',
        base: Uri.parse('http://fake.local'),
      ),
      <String>{'https://GitHub.com/x', 'https://github.com/x'},
    );
  });

  test('IDN policy entries match the percent-encoded host Dart parses', () {
    // 服务端把 IDN 条目存成 punycode（idna.Lookup.ToASCII），而 Dart 的
    // Uri.host 给的是百分号编码形式（issue #1200 review）。
    LinkNavigation.applyTrustedDomains(<String>['xn--fiqs8s.com']); // 中国.com
    expect(
      LinkNavigation.isPolicyTrusted(Uri.parse('https://中国.com/x').host),
      isTrue,
    );
    expect(LinkNavigation.isPolicyTrusted('xn--fiqs8s.com'), isTrue);
    expect(LinkNavigation.isPolicyTrusted('www.xn--fiqs8s.com'), isTrue);
    expect(LinkNavigation.isPolicyTrusted('xn--fiqs8s.com.evil.test'), isFalse);
  });

  test(
    'normalizeTrustedDomains lowercases, dedupes and drops unusable entries',
    () {
      expect(
        normalizeTrustedDomains(<String>[
          'GitHub.com',
          'github.com',
          '',
          '   ',
          'localhost',
          '中国.com',
        ]),
        <String>{'github.com', 'xn--fiqs8s.com'},
      );
    },
  );
}

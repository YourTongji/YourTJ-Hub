import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('collects links whose visible text is an absolute http(s) URL', () {
    final List<UrlLabeledLink> links = urlLabeledMarkdownLinks(
      '[https://bank.example/account](https://github.com/x) 与 [文档](https://docs.example)',
    );
    expect(links, <UrlLabeledLink>[
      const UrlLabeledLink(
        href: 'https://github.com/x',
        label: 'https://bank.example/account',
      ),
    ]);
  });

  test('covers reference-style links and titled targets', () {
    final List<UrlLabeledLink> byReference = urlLabeledMarkdownLinks(
      '[https://bank.example][ref]\n\n[ref]: https://github.com/y',
    );
    expect(byReference.single.href, 'https://github.com/y');
    expect(byReference.single.label, 'https://bank.example');

    final List<UrlLabeledLink> titled = urlLabeledMarkdownLinks(
      '[https://bank.example](https://github.com/y "标题")',
    );
    expect(titled.single.href, 'https://github.com/y');
    expect(titled.single.label, 'https://bank.example');
  });

  test('keeps the href verbatim so callers can match the click callback', () {
    // 相对目标也照原样返回（调用方按目标主机判定）；点击回调拿到的是同一个字符串。
    final List<UrlLabeledLink> links = urlLabeledMarkdownLinks(
      '[https://bank.example](/relative/path)',
    );
    expect(links.single.href, '/relative/path');
  });

  test('ignores plain labels, non-http labels and link-free text', () {
    expect(urlLabeledMarkdownLinks('[文档](https://docs.example)'), isEmpty);
    expect(
      urlLabeledMarkdownLinks(
        '[mailto:hi@bank.example](mailto:hi@bank.example)',
      ),
      isEmpty,
    );
    expect(urlLabeledMarkdownLinks('正文没有链接'), isEmpty);
  });

  test('autolinks keep the URL as both label and target', () {
    // 自动链接的可见文本就是目标本身，调用方的 registrable domain 比较会判为
    // 一致（与 Web 一致：不是「标签与目标不符」）。
    final List<UrlLabeledLink> links = urlLabeledMarkdownLinks(
      '<https://docs.example>',
    );
    expect(links.single.href, 'https://docs.example');
    expect(links.single.label, 'https://docs.example');
  });
}

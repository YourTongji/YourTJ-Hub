import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final converter = MarkdownConverter();

  // 语义断言辅助:md → delta → md 后,原文关键标记仍然存在。
  void expectRoundTrip(
    String input,
    List<String> preserved, {
    List<String>? lost,
  }) {
    final delta = converter.mdToDelta(input);
    final output = converter.deltaToMarkdown(delta);
    for (final marker in preserved) {
      expect(
        output,
        contains(marker),
        reason: 'round-trip 后应保留: $marker\n输入:\n$input\n输出:\n$output',
      );
    }
    for (final marker in lost ?? const <String>[]) {
      expect(
        output,
        isNot(contains(marker)),
        reason: 'round-trip 后应丢弃: $marker\n输入:\n$input\n输出:\n$output',
      );
    }
  }

  group('markdown 往返转换', () {
    test('标题层级', () {
      expectRoundTrip('# 一级\n\n## 二级\n\n### 三级', ['# 一级', '## 二级', '### 三级']);
    });

    test('粗体与斜体', () {
      // markdown_quill 会规范斜体为 _ 语法;语义保留即可。
      expectRoundTrip('**粗体** 和 *斜体* 以及 ***粗斜体***', ['**粗体**', '斜体', '粗斜体']);
    });

    test('行内代码', () {
      expectRoundTrip('这是 `inline code` 示例', ['`inline code`']);
    });

    test('嵌套列表', () {
      const input = '- 一级\n  - 二级\n    - 三级\n- 另一个';
      final delta = converter.mdToDelta(input);
      final output = converter.deltaToMarkdown(delta);
      // markdown_quill 会保留嵌套层级(用缩进或子列表);断言条目文本存在且有序。
      expect(output, contains('一级'));
      expect(output, contains('二级'));
      expect(output, contains('三级'));
      expect(output, contains('另一个'));
      expect(output.indexOf('一级'), lessThan(output.indexOf('二级')));
      expect(output.indexOf('二级'), lessThan(output.indexOf('三级')));
    });

    test('有序列表', () {
      expectRoundTrip('1. 第一\n2. 第二', ['第一', '第二']);
    });

    test('引用块', () {
      expectRoundTrip('> 引用内容\n> 第二行', ['引用内容', '第二行']);
    });

    test('引用内标题(降级为文本)', () {
      // markdown_quill 限制:blockquote 内 header 会降级;语义保留文本即可。
      expectRoundTrip('> # 引用标题', ['引用标题'], lost: ['#']);
    });

    test('表格(只读保留)', () {
      const input = '| 语法 | 描述 |\n| --- | --- |\n| 表格 | 支持 |';
      final delta = converter.mdToDelta(input);
      final output = converter.deltaToMarkdown(delta);
      expect(output, contains('语法'));
      expect(output, contains('描述'));
      expect(output, contains('表格'));
      expect(output, contains('支持'));
    });

    test('任务列表', () {
      const input = '- [x] 已完成\n- [ ] 未完成';
      final delta = converter.mdToDelta(input);
      final output = converter.deltaToMarkdown(delta);
      expect(output, contains('[x]'));
      expect(output, contains('[ ]'));
      expect(output, contains('已完成'));
      expect(output, contains('未完成'));
    });

    test('围栏代码块', () {
      const input = '```dart\nvoid main() {}\n```';
      final delta = converter.mdToDelta(input);
      final output = converter.deltaToMarkdown(delta);
      expect(output, contains('void main() {}'));
      expect(output, contains('dart'), reason: '代码块语言标注保留');
    });

    test('图片(保留 URL,alt 丢失)', () {
      const input = '![alt 文本](https://example.com/a.webp)';
      final delta = converter.mdToDelta(input);
      final output = converter.deltaToMarkdown(delta);
      expect(output, contains('https://example.com/a.webp'));
      expect(output, isNot(contains('alt 文本')), reason: '图片 alt 不保留');
    });

    test('链接', () {
      expectRoundTrip('[链接文本](https://example.com)', [
        '[链接文本](https://example.com)',
      ]);
    });

    test('分割线', () {
      // markdown_quill 输出 `- - -`,仍是合法分割线。
      expectRoundTrip('上面\n\n---\n\n下面', ['- - -']);
    });

    test('空文档与纯文本', () {
      expect(converter.deltaToMarkdown(converter.mdToDelta('')), isEmpty);
      expectRoundTrip('普通段落文本', ['普通段落文本']);
    });

    test('mdToDocument 生成可编辑 Document', () {
      final document = converter.mdToDocument('# 标题\n\n正文');
      expect(document.toDelta().length, greaterThan(0));
      // Document → markdown 往返
      final output = converter.documentToMarkdown(document);
      expect(output, contains('标题'));
      expect(output, contains('正文'));
    });

    test('真实帖子风格:混合排版(标题+粗体+列表+代码+引用+图片+链接)', () {
      // 模拟 web 端真实帖子内容(论坛常见混合排版)。
      const input =
          '# 同济大学图书馆攻略\n\n'
          '**开放时间**:周一至周日 8:00-22:00\n\n'
          '## 楼层分布\n\n'
          '- 一楼:大厅与自助借还\n'
          '- 二楼:中文图书\n'
          '- 三楼:期刊与自习区\n\n'
          '> 期末期间座位紧张,建议提前预约\n\n'
          '```bash\n'
          'ssh user@lib.tongji.edu.cn\n'
          '```\n\n'
          '详细规则见[图书馆官网](https://www.lib.tongji.edu.cn)\n\n'
          '![图书馆照片](https://example.com/lib.webp)';
      final delta = converter.mdToDelta(input);
      final output = converter.deltaToMarkdown(delta);
      for (final marker in [
        '同济大学图书馆攻略',
        '开放时间',
        '楼层分布',
        '一楼',
        '二楼',
        '三楼',
        '期末期间座位紧张',
        'ssh user@lib.tongji.edu.cn',
        '图书馆官网',
        'https://www.lib.tongji.edu.cn',
        'https://example.com/lib.webp',
      ]) {
        expect(
          output,
          contains(marker),
          reason: '真实帖子 round-trip 应保留: $marker',
        );
      }
    });

    test('真实帖子风格:长帖正文(多段落+列表+行内代码)', () {
      // 模拟移动端发帖常见长文(纯文本+结构化混排)。
      const input =
          '分享一下我的实习面经。\n\n'
          '## 一面\n\n'
          '问了一些基础题,比如 `HashMap` 和 `ConcurrentHashMap` 的区别。\n\n'
          '## 二面\n\n'
          '主要考察项目深挖,追问了缓存一致性问题。\n\n'
          '## 总结\n\n'
          '1. 简历上的项目要能讲清楚\n'
          '2. 算法题多刷\n'
          '3. 保持自信\n\n'
          '祝大家都能拿到满意的 offer!';
      final delta = converter.mdToDelta(input);
      final output = converter.deltaToMarkdown(delta);
      for (final marker in [
        '实习面经',
        '一面',
        '二面',
        '总结',
        'HashMap',
        'ConcurrentHashMap',
        '缓存一致性',
        '简历上的项目要能讲清楚',
        '保持自信',
        'offer',
      ]) {
        expect(output, contains(marker), reason: '长帖 round-trip 应保留: $marker');
      }
    });
  });
}

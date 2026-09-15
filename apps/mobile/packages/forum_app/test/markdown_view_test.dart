import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/markdown_view.dart';

/// 内存版 TokenStorage(贴纸库 fake 的离线 client 用)。
class _MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

/// 固定返回 smile 表情的贴纸库替身(不发真实网络)。
class _FakeStickerRepository extends StickerRepository {
  _FakeStickerRepository()
    : super(
        GfApiClient(
          dio: Dio(BaseOptions(baseUrl: 'http://test')),
          tokenStorage: _MemoryTokenStorage(),
        ),
      );

  @override
  Future<List<StickerItemPayload>> list() async => const <StickerItemPayload>[
    StickerItemPayload(name: 'smile', url: '/file/img/stickers/smile.png'),
  ];
}

Widget _wrap(Widget child) => MaterialApp(
  theme: gfThemeData(Brightness.light),
  home: Scaffold(body: Column(children: [child])),
);

void main() {
  testWidgets('reading paragraphs and inline code use a legible type scale', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: const Scaffold(body: GfMarkdownView(data: 'Body with `code`')),
      ),
    );
    final config = tester
        .widget<MarkdownWidget>(find.byType(MarkdownWidget))
        .config!;
    expect(config.p.textStyle.fontSize, 18);
    expect(config.code.style.fontSize, 16);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('short replies have compact block spacing and readable text', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: _wrap(
          const GfMarkdownView(data: 'First paragraph\n\nSecond paragraph'),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      tester.getSize(find.byType(GfMarkdownView)).height,
      lessThanOrEqualTo(72),
    );
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('embedded markdown does not repeat device safe-area spacing', (
    tester,
  ) async {
    Future<double> measure(EdgeInsets insets) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            home: MediaQuery(
              data: MediaQueryData(padding: insets),
              child: const Scaffold(
                body: Column(children: [GfMarkdownView(data: 'A short reply')]),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      return tester.getSize(find.byType(GfMarkdownView)).height;
    }

    final baseline = await measure(EdgeInsets.zero);
    final withInsets = await measure(
      const EdgeInsets.only(top: 62, bottom: 34),
    );
    expect(withInsets, baseline);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('父级重建复用 markdown 解析树，内容变化时才重建', (tester) async {
    late StateSetter rebuild;
    String markdown = '**第一版**';

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: StatefulBuilder(
              builder: (BuildContext context, StateSetter setState) {
                rebuild = setState;
                return GfMarkdownView(data: markdown);
              },
            ),
          ),
        ),
      ),
    );

    final MarkdownWidget first = tester.widget<MarkdownWidget>(
      find.byType(MarkdownWidget),
    );

    rebuild(() {});
    await tester.pump();

    final MarkdownWidget afterParentRebuild = tester.widget<MarkdownWidget>(
      find.byType(MarkdownWidget),
    );
    expect(identical(first, afterParentRebuild), isTrue);

    rebuild(() => markdown = '**第二版**');
    await tester.pump();

    final MarkdownWidget afterContentChange = tester.widget<MarkdownWidget>(
      find.byType(MarkdownWidget),
    );
    expect(identical(first, afterContentChange), isFalse);
    expect(find.text('第二版'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('表情包 token 在贴纸库就绪后展开为图片', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: <Override>[
          stickerLibraryProvider.overrideWithValue(
            StickerLibrary(_FakeStickerRepository()),
          ),
        ],
        child: _wrap(const GfMarkdownView(data: '你好 [:sticker:smile:]')),
      ),
    );

    // 首帧:贴纸库尚未就绪,token 保持原文(未加载映射为空,未知语义保持原文)。
    expect(
      tester.widget<MarkdownWidget>(find.byType(MarkdownWidget)).data,
      '你好 [:sticker:smile:]',
    );

    // 库拉取完成后异步刷新:token 重写为标准图片语法并渲染图片组件。
    await tester.pump();
    expect(
      tester.widget<MarkdownWidget>(find.byType(MarkdownWidget)).data,
      '你好 ![sticker:smile](/file/img/stickers/smile.png)',
    );
    expect(find.byType(Image), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('未知表情 token 始终保持原文', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: <Override>[
          stickerLibraryProvider.overrideWithValue(
            StickerLibrary(_FakeStickerRepository()),
          ),
        ],
        child: _wrap(const GfMarkdownView(data: '前 [:sticker:ghost:] 后')),
      ),
    );
    await tester.pump();
    await tester.pump();
    expect(
      tester.widget<MarkdownWidget>(find.byType(MarkdownWidget)).data,
      '前 [:sticker:ghost:] 后',
    );
    expect(find.byType(Image), findsNothing);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });
}

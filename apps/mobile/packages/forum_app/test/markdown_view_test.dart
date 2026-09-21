import 'package:dio/dio.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
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

class _FakeLinkPreviewRepository extends LinkPreviewRepository {
  _FakeLinkPreviewRepository(this.previews)
    : super(
        GfApiClient(
          dio: Dio(BaseOptions(baseUrl: 'http://test')),
          tokenStorage: _MemoryTokenStorage(),
        ),
      );

  final List<LinkPreviewPayload> previews;
  List<String>? requestedUrls;

  @override
  Future<List<LinkPreviewPayload>> resolve(List<String> urls) async {
    requestedUrls = List<String>.of(urls);
    return previews;
  }
}

Widget _wrap(Widget child) => MaterialApp(
  theme: gfThemeData(Brightness.light),
  home: Scaffold(body: Column(children: [child])),
);

TapGestureRecognizer? _linkRecognizer(InlineSpan span) {
  if (span is TextSpan && span.recognizer is TapGestureRecognizer) {
    return span.recognizer! as TapGestureRecognizer;
  }
  if (span is! TextSpan) return null;
  for (final InlineSpan child in span.children ?? const <InlineSpan>[]) {
    final TapGestureRecognizer? recognizer = _linkRecognizer(child);
    if (recognizer != null) return recognizer;
  }
  return null;
}

void main() {
  testWidgets(
    'standalone URL resolves once and renders a responsive preview card',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(320, 640));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      const String url = 'https://example.com/article';
      final repository = _FakeLinkPreviewRepository(const <LinkPreviewPayload>[
        LinkPreviewPayload(
          requestedUrl: url,
          kind: 'external',
          status: 'ready',
          url: url,
          displayHost: 'example.com',
          siteName: 'Example',
          title: 'Example title that remains readable on a narrow screen',
          description: 'A short description.',
        ),
      ]);

      await tester.pumpWidget(
        ProviderScope(
          overrides: <Override>[
            linkPreviewRepositoryProvider.overrideWithValue(repository),
          ],
          child: MaterialApp(
            theme: gfThemeData(Brightness.dark),
            home: MediaQuery(
              data: const MediaQueryData(
                size: Size(320, 640),
                textScaler: TextScaler.linear(2),
              ),
              child: const Scaffold(body: GfMarkdownView(data: url)),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(repository.requestedUrls, <String>[url]);
      expect(find.textContaining('Example title'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );

  testWidgets('campus cards localize fallback copy when the server sends no name', (
    tester,
  ) async {
    // 校园网卡片由服务端按部署配置本地渲染：配置没给名字时 title / description
    // 都为空，兜底文案必须由客户端 l10n 出（服务端不留任何中文）。修复前这里
    // 会走到 `preview.title!`，直接抛 null。
    const String url = 'https://agent.tongji.edu.cn/chat';
    final repository = _FakeLinkPreviewRepository(const <LinkPreviewPayload>[
      LinkPreviewPayload(
        requestedUrl: url,
        kind: 'external',
        status: 'ready',
        url: url,
        displayHost: 'agent.tongji.edu.cn',
        siteName: 'agent.tongji.edu.cn',
        campus: true,
      ),
    ]);
    await tester.pumpWidget(
      ProviderScope(
        overrides: <Override>[
          linkPreviewRepositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const Scaffold(body: GfMarkdownView(data: url)),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(repository.requestedUrls, <String>[url]);
    expect(find.text('校园网'), findsOneWidget);
    expect(find.text('需校园网络访问'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('configured campus names win over the localized fallback', (
    tester,
  ) async {
    const String url = 'https://1.tongji.edu.cn/';
    final repository = _FakeLinkPreviewRepository(const <LinkPreviewPayload>[
      LinkPreviewPayload(
        requestedUrl: url,
        kind: 'external',
        status: 'ready',
        url: url,
        displayHost: '1.tongji.edu.cn',
        siteName: '1.tongji.edu.cn',
        title: '同济大学教学管理系统',
        campus: true,
      ),
    ]);
    await tester.pumpWidget(
      ProviderScope(
        overrides: <Override>[
          linkPreviewRepositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const Scaffold(body: GfMarkdownView(data: url)),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // 服务端给了名字就用名字，客户端不得覆盖；描述仍补本地化提示。
    expect(find.text('同济大学教学管理系统'), findsOneWidget);
    expect(find.text('校园网'), findsNothing);
    expect(find.text('需校园网络访问'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('failed preview keeps the original link readable', (
    tester,
  ) async {
    const String url = 'https://example.com/unavailable';
    final repository = _FakeLinkPreviewRepository(const <LinkPreviewPayload>[
      LinkPreviewPayload(
        requestedUrl: url,
        kind: 'external',
        status: 'unavailable',
      ),
    ]);
    await tester.pumpWidget(
      ProviderScope(
        overrides: <Override>[
          linkPreviewRepositoryProvider.overrideWithValue(repository),
        ],
        child: _wrap(const GfMarkdownView(data: url)),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining(url), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('offscreen previews wait until they approach the viewport', (
    tester,
  ) async {
    const String url = 'https://example.com/deferred';
    final repository = _FakeLinkPreviewRepository(const <LinkPreviewPayload>[
      LinkPreviewPayload(
        requestedUrl: url,
        kind: 'external',
        status: 'ready',
        url: url,
        displayHost: 'example.com',
        title: 'Deferred preview',
      ),
    ]);
    final ScrollController controller = ScrollController();
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: <Override>[
          linkPreviewRepositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              controller: controller,
              child: const Column(
                children: <Widget>[
                  SizedBox(height: 1200),
                  GfMarkdownView(data: url),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(repository.requestedUrls, isNull);

    controller.jumpTo(controller.position.maxScrollExtent);
    await tester.pumpAndSettle();
    expect(repository.requestedUrls, <String>[url]);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('ordinary external Markdown links use the confirmation dialog', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(320, 640));
    tester.platformDispatcher.textScaleFactorTestValue = 2;
    addTearDown(() async {
      tester.platformDispatcher.clearTextScaleFactorTestValue();
      await tester.binding.setSurfaceSize(null);
    });
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          locale: Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: GfMarkdownView(data: '[Docs](https://example.com)'),
          ),
        ),
      ),
    );
    final RichText link = tester.widget<RichText>(
      find.byWidgetPredicate(
        (Widget widget) =>
            widget is RichText && widget.text.toPlainText().contains('Docs'),
      ),
    );
    _linkRecognizer(link.text)?.onTap?.call();
    await tester.pumpAndSettle();
    expect(find.text('Leaving YourTJ'), findsOneWidget);
    expect(find.text('https://example.com'), findsOneWidget);
    await tester.ensureVisible(find.text('Cancel'));
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('userinfo links are rejected without opening the guard', (
    tester,
  ) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          locale: Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: GfMarkdownView(data: '[Docs](https://user@example.com)'),
          ),
        ),
      ),
    );
    final RichText link = tester.widget<RichText>(
      find.byWidgetPredicate(
        (Widget widget) =>
            widget is RichText && widget.text.toPlainText().contains('Docs'),
      ),
    );
    _linkRecognizer(link.text)?.onTap?.call();
    await tester.pumpAndSettle();
    expect(find.text('Leaving YourTJ'), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('reading paragraphs and inline code use a legible type scale', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: const Scaffold(body: GfMarkdownView(data: 'Body with `code`')),
        ),
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

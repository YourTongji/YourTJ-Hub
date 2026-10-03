import 'package:dio/dio.dart';
import 'package:go_router/go_router.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:forum_app/src/widgets/stickers/sticker_image.dart';
import 'package:forum_app/src/widgets/stickers/sticker_draft_preview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/reading_preferences.dart';
import 'package:forum_app/src/widgets/rich_content/gf_code_block.dart';
import 'package:forum_app/src/widgets/rich_content/gf_table.dart';
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
  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async => [];
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

/// Flattens every [TextSpan] of a rendered rich text so style assertions can
/// look at the actual token spans instead of the outer wrapper.
List<TextSpan> _textSpans(InlineSpan span) {
  final spans = <TextSpan>[];
  void walk(InlineSpan node) {
    if (node is! TextSpan) return;
    spans.add(node);
    for (final child in node.children ?? const <InlineSpan>[]) {
      walk(child);
    }
  }

  walk(span);
  return spans;
}

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
  for (final markdown in [false, true]) {
    testWidgets('draft stickers remain compact markdown=$markdown', (
      tester,
    ) async {
      final library = StickerLibrary(_FakeStickerRepository());
      addTearDown(library.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [stickerLibraryProvider.overrideWithValue(library)],
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: StickerDraftPreview(
                content: '[:sticker:smile:]',
                markdown: markdown,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.getSize(find.byType(StickerImage)), const Size(56, 56));
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    });
  }

  testWidgets('large stickers wrap within narrow quoted prose', (tester) async {
    await tester.binding.setSurfaceSize(const Size(320, 700));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final library = StickerLibrary(_FakeStickerRepository());
    addTearDown(library.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [stickerLibraryProvider.overrideWithValue(library)],
        child: _wrap(
          const GfMarkdownView(
            data:
                '> 看看这些表情 [:sticker:smile:][:sticker:smile:][:sticker:smile:]',
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final stickers = find.byType(StickerImage);
    expect(stickers, findsNWidgets(3));
    final boxes = [
      for (final element in stickers.evaluate())
        tester.getRect(find.byWidget(element.widget)),
    ];
    expect(boxes.last.top, greaterThan(boxes.first.top));
    for (final box in boxes) {
      expect(box.size, const Size(128, 128));
      expect(box.left, greaterThanOrEqualTo(0));
      expect(box.right, lessThanOrEqualTo(320));
    }
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('server mention mapping opens native user page', (tester) async {
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => const Scaffold(
            body: GfMarkdownView(
              data: '😀 @alice_smith `@alice_smith`',
              mentions: [
                PostMention(
                  username: 'alice_smith',
                  userId: 42,
                  start: 3,
                  end: 15,
                ),
              ],
            ),
          ),
        ),
        GoRoute(
          path: '/u/:id',
          builder: (_, state) =>
              Scaffold(body: Text('profile ${state.pathParameters['id']}')),
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          theme: gfThemeData(Brightness.light),
          routerConfig: router,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    final links = tester
        .widgetList<RichText>(find.byType(RichText))
        .map((text) => _linkRecognizer(text.text))
        .whereType<TapGestureRecognizer>()
        .toList();
    expect(links, hasLength(1));
    links.single.onTap!();
    await tester.pumpAndSettle();
    expect(find.text('profile 42'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets(
    'wide preview covers contain the image while phone thumbnails crop',
    (tester) async {
      for (final width in [320.0, 700.0]) {
        await tester.binding.setSurfaceSize(Size(width, 640));
        await tester.pumpWidget(
          ProviderScope(
            child: _wrap(
              const GfLinkPreviewCard(
                preview: LinkPreviewPayload(
                  requestedUrl: 'https://example.com',
                  kind: 'external',
                  status: 'ready',
                  url: 'https://example.com',
                  displayHost: 'example.com',
                  title: 'A wide image stays complete',
                  imageUrl: 'https://image.test/cover.png',
                ),
              ),
            ),
          ),
        );
        final cover = tester.widget<Image>(find.byType(Image).first);
        expect(cover.fit, width >= 640 ? BoxFit.contain : BoxFit.cover);
        await tester.pumpWidget(const SizedBox.shrink());
      }
      await tester.binding.setSurfaceSize(null);
    },
  );

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

  testWidgets(
    'campus cards localize fallback copy when the server sends no name',
    (tester) async {
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
    },
  );

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
    // Reading text comes from the shared rich-content profile: the body is the
    // design baseline (17), headings are relative ratios, code is one step
    // smaller. The reader preference multiplies these values at 100% by default.
    final profile = GfRichContentTypography.standard(
      typography: GfTypography.standard(GfColors.light.baseContent),
      colors: GfColors.light,
    );
    expect(config.p.textStyle.fontSize, profile.body.fontSize);
    expect(config.p.textStyle.fontSize, 17);
    expect(config.code.style.fontSize, profile.inlineCode.fontSize);
    expect(config.code.style.fontSize, 16);
    expect(config.h1.style.fontSize, profile.h1.fontSize);
    expect(config.h2.style.fontSize, profile.h2.fontSize);
    expect(config.h3.style.fontSize, profile.h3.fontSize);
    expect(config.h4.style.fontSize, profile.h4.fontSize);
    expect(config.table.bodyStyle!.fontSize, profile.tableBody.fontSize);
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
      startsWith('你好 ![sticker:smile](gf-sticker-render:'),
    );
    expect(find.byType(Image), findsOneWidget);
    expect(find.byType(StickerImage), findsOneWidget);
    expect(tester.getSize(find.byType(StickerImage)), const Size(128, 128));
    expect(
      tester.widget<StickerImage>(find.byType(StickerImage)).url,
      '/file/img/stickers/smile.png',
    );
    await tester.tap(find.byType(StickerImage));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.byType(GfImageViewer), findsOneWidget);
    expect(tester.widget<GfImageViewer>(find.byType(GfImageViewer)).images, [
      endsWith('/file/img/stickers/smile.png'),
    ]);

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
  testWidgets(
    'ordinary sticker-prefixed image alt never grants sticker behavior',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            stickerLibraryProvider.overrideWithValue(
              StickerLibrary(_FakeStickerRepository()),
            ),
          ],
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(
              body: GfMarkdownView(
                data:
                    '[:sticker:smile:]\n\n![sticker:smile](/file/img/stickers/smile.png)\n\n![sticker:unknown](/file/img/photo.png)',
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(StickerImage), findsOneWidget);
      final photos = find.byWidgetPredicate(
        (widget) => widget is Image && widget.image is ResizeImage,
      );
      expect(photos, findsNWidgets(2));
      await tester.tap(photos.last);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
      final viewer = tester.widget<GfImageViewer>(find.byType(GfImageViewer));
      expect(viewer.images, hasLength(2));
      expect(viewer.images.first, endsWith('/file/img/stickers/smile.png'));
      expect(viewer.images.last, endsWith('/file/img/photo.png'));
      expect(viewer.initialIndex, 1);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );
  testWidgets(
    'ordinary photo viewer excludes stickers including supplied gallery entries',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            stickerLibraryProvider.overrideWithValue(
              StickerLibrary(_FakeStickerRepository()),
            ),
          ],
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(
              body: GfMarkdownView(
                data: '[:sticker:smile:]\n\n![photo](/file/img/photo.png)',
                images: ['/file/img/stickers/smile.png', '/file/img/photo.png'],
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      final photo = find.byWidgetPredicate(
        (widget) => widget is Image && widget.image is ResizeImage,
      );
      expect(photo, findsOneWidget);
      await tester.tap(photo);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
      final viewer = tester.widget<GfImageViewer>(find.byType(GfImageViewer));
      expect(viewer.images, hasLength(1));
      expect(viewer.images.single, endsWith('/file/img/photo.png'));
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );

  testWidgets('fenced code is highlighted, scrolls on its own and copies', (
    tester,
  ) async {
    const code = 'final int answer = 42;\nfinal int doubled = answer * 2;';
    final clipboard = <MethodCall>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (MethodCall call) async {
        clipboard.add(call);
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );

    await tester.pumpWidget(
      ProviderScope(
        child: _wrap(const GfMarkdownView(data: '```dart\n$code\n```')),
      ),
    );
    await tester.pumpAndSettle();

    final block = find.byType(GfCodeBlock);
    expect(block, findsOneWidget);
    expect(
      find.descendant(of: block, matching: find.byType(GfHorizontalScrollView)),
      findsOneWidget,
    );
    // Token colors come from the highlight theme, not from one flat body color.
    final rendered = tester.widget<RichText>(
      find.byWidgetPredicate(
        (Widget widget) =>
            widget is RichText &&
            widget.text.toPlainText().contains('final int answer'),
      ),
    );
    final tokenColors = _textSpans(rendered.text)
        .map((span) => span.style?.color)
        .whereType<Color>()
        .toSet();
    expect(tokenColors.length, greaterThan(1));
    expect(find.text('DART'), findsOneWidget);

    await tester.tap(
      find.descendant(of: block, matching: find.byType(IconButton)),
    );
    await tester.pumpAndSettle();
    final copied = clipboard.where(
      (call) => call.method == 'Clipboard.setData',
    );
    expect(copied, hasLength(1));
    expect((copied.single.arguments as Map<Object?, Object?>)['text'], code);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('an unknown fenced language still renders as plain code', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: _wrap(const GfMarkdownView(data: '```klingon\nplain body\n```')),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.byType(GfCodeBlock), findsOneWidget);
    expect(find.text('KLINGON'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (Widget widget) =>
            widget is RichText &&
            widget.text.toPlainText().contains('plain body'),
      ),
      findsOneWidget,
    );
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('wide tables scroll inside their own viewport', (tester) async {
    const table =
        '| a | b | c | d | e | f | g | h |\n'
        '| --- | --- | --- | --- | --- | --- | --- | --- |\n'
        '| 1111111111 | 2222222222 | 3333333333 | 4444444444 '
        '| 5555555555 | 6666666666 | 7777777777 | 8888888888 |';
    await tester.pumpWidget(
      ProviderScope(
        child: _wrap(
          const SizedBox(width: 320, child: GfMarkdownView(data: table)),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    final viewport = find.byType(GfTableViewport);
    expect(viewport, findsOneWidget);
    expect(
      find.descendant(
        of: viewport,
        matching: find.byType(GfHorizontalScrollView),
      ),
      findsOneWidget,
    );
    // The table scrolls in place: the viewport itself never exceeds the width
    // the page gave it.
    expect(tester.getSize(viewport).width, lessThanOrEqualTo(320));
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('reader preference scales rich content on top of the baseline', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Consumer(
            builder: (context, ref, _) => Scaffold(
              body: Column(
                children: <Widget>[
                  GfMarkdownView(
                    data: 'Body',
                    key: ValueKey(ref.watch(contentFontScaleProvider)),
                  ),
                  TextButton(
                    onPressed: () => ref
                        .read(contentFontScaleProvider.notifier)
                        .setScale(1.4),
                    child: const Text('scale'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('scale'));
    await tester.pumpAndSettle();

    final config = tester
        .widget<MarkdownWidget>(find.byType(MarkdownWidget))
        .config!;
    expect(config.p.textStyle.fontSize, closeTo(17 * 1.4, .001));
    // Headings grow with the body but keep less of their extra emphasis.
    final double emphasis = GfRichContentTypography.headingEmphasisFor(
      renderedScale: 1.4,
      width: 800,
    );
    expect(
      config.h1.style.fontSize,
      closeTo(17 * (1 + .45 * emphasis) * 1.4, .001),
    );
    expect(config.h1.style.fontSize, lessThan(17 * 1.45 * 1.4));
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });
}

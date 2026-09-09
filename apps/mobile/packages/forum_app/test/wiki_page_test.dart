import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/wiki/wiki_home_page.dart';
import 'package:forum_app/src/pages/wiki/wiki_page.dart';
import 'package:forum_app/src/providers.dart';

import 'fixtures/page_fixtures.dart';

/// 测试用内存 TokenStorage。
class _MemoryTokenStorage implements TokenStorage {
  @override
  Future<String?> read() async => null;

  @override
  Future<void> write(String token) async {}

  @override
  Future<void> clear() async {}
}

/// 首页假仓库:固定返回一组 namespace + recent。
class _FakeWikiRepository extends WikiRepository {
  _FakeWikiRepository(super.client, {required this.data});

  final WikiHomeData data;
  int homeCalls = 0;

  @override
  Future<WikiHomeData> home() async {
    homeCalls++;
    return data;
  }
}

/// 页面通道假仓库:按请求路径返回固定 PagePayload;可配置前 N 次失败。
class _FakePageRepository extends PageRepository {
  _FakePageRepository(
    super.client, {
    required this.payloads,
    this.failuresBeforeSuccess = 0,
  });

  final Map<String, PagePayload> payloads;
  final List<String> fetchedPaths = <String>[];
  int failuresBeforeSuccess;

  @override
  Future<PagePayload> fetch(String path) async {
    fetchedPaths.add(path);
    if (failuresBeforeSuccess > 0) {
      failuresBeforeSuccess--;
      throw const NetworkException(fallbackMessage: 'network unavailable');
    }
    return payloads[path]!;
  }
}

Map<String, dynamic> _pageJson(Map<String, dynamic> page) => <String, dynamic>{
  'component': 'wiki.detail',
  'props': <String, dynamic>{
    'page': page,
    'contributors': <Object>[],
    'hotTopics': <Object>[],
  },
  'meta': <String, dynamic>{'title': page['title']},
  'layout': minimalLayoutJson(),
  'url': '/wiki/${page['path']}',
  'version': '1.0',
};

PagePayload _detailPayload({
  required String path,
  required String title,
  bool canEdit = true,
  String editUrl = 'https://github.com/yourtj/wiki/edit/main/guide/start.md',
  String? content,
}) {
  return PagePayload.fromJson(
    _pageJson(<String, dynamic>{
      'id': 11,
      'topicId': 0,
      'namespace': 'guide',
      'path': path,
      'title': title,
      'content':
          content ??
          '<h2 id="intro">介绍</h2><p>欢迎阅读 wiki 正文。</p><h3 id="details">细节说明</h3>',
      'toc': <Map<String, dynamic>>[
        <String, dynamic>{'level': 2, 'id': 'intro', 'text': '介绍'},
        <String, dynamic>{'level': 3, 'id': 'details', 'text': '细节说明'},
      ],
      'updatedAt': '2026-08-10T15:00:00+08:00',
      'likeCount': 5,
      'viewCount': 42,
      'postCount': 0,
      'liked': false,
      'bookmarked': false,
      'watched': false,
      'canEdit': canEdit,
      'publishedRevisionNo': 0,
      if (canEdit) 'editUrl': editUrl,
      'historyUrl': '',
    }),
  );
}

WikiHomeData _homeData() => WikiHomeData.fromJson(<String, dynamic>{
  'namespaces': <Map<String, dynamic>>[
    <String, dynamic>{
      'name': '指南',
      'description': '社区使用指南',
      'sortOrder': 10,
      'pageCount': 3,
      'updatedAt': '2026-08-10T15:00:00+08:00',
      'firstPagePath': 'guide/getting-started',
    },
  ],
  'recent': <Map<String, dynamic>>[
    <String, dynamic>{
      'pageId': 1001,
      'path': 'guide/content',
      'title': '内容规范',
      'updatedAt': '2026-08-10T14:00:00+08:00',
    },
  ],
});

GfApiClient _client() => GfApiClient(
  dio: Dio(),
  tokenStorage: _MemoryTokenStorage(),
  baseUrl: 'http://fake.local',
);

/// 点击正文链接文字字形处:fwfh 的块级 RichText 占满整行宽度,链接文字
/// 靠左,tap 中心点落在文字右侧空白处不会命中识别器,须点文字起始处。
Future<void> _tapLinkText(WidgetTester tester, Finder finder) async {
  final Rect rect = tester.getRect(finder);
  await tester.tapAt(Offset(rect.left + 12, rect.center.dy));
}

Future<GoRouter> _pumpApp(
  WidgetTester tester, {
  required _FakePageRepository pages,
  _FakeWikiRepository? wiki,
  String initialLocation = '/',
}) async {
  final _FakeWikiRepository wikiRepo =
      wiki ??
      _FakeWikiRepository(
        _client(),
        data: WikiHomeData.fromJson(<String, dynamic>{
          'namespaces': <Object>[],
          'recent': <Object>[],
        }),
      );
  final GoRouter router = GoRouter(
    initialLocation: initialLocation,
    routes: <RouteBase>[
      GoRoute(path: '/', builder: (_, _) => const WikiHomePage()),
      GoRoute(
        // 与生产 router.dart 对齐:参数名 wikiPath、锚点经 state.uri.fragment 传入。
        path: '/wiki/:wikiPath(.*)',
        builder: (BuildContext context, GoRouterState state) => WikiPage(
          wikiPath: state.pathParameters['wikiPath'] ?? '',
          initialAnchor: state.uri.fragment,
        ),
      ),
    ],
  );

  await tester.pumpWidget(
    ProviderScope(
      overrides: <Override>[
        pageRepositoryProvider.overrideWithValue(pages),
        wikiRepositoryProvider.overrideWithValue(wikiRepo),
      ],
      child: MaterialApp.router(
        theme: gfThemeData(Brightness.light),
        routerConfig: router,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('zh'),
      ),
    ),
  );
  await tester.pumpAndSettle();

  return router;
}

void main() {
  testWidgets('wiki 首页渲染命名空间卡片与最近更新并可进入详情', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/content': _detailPayload(
          path: 'guide/content',
          title: '内容规范',
        ),
      },
    );
    final _FakeWikiRepository wiki = _FakeWikiRepository(
      _client(),
      data: _homeData(),
    );
    final GoRouter router = await _pumpApp(tester, pages: pages, wiki: wiki);

    // 首页结构:命名空间区 + 最近更新区。
    expect(wiki.homeCalls, 1);
    expect(find.text('命名空间'), findsOneWidget);
    expect(find.text('指南'), findsOneWidget);
    expect(find.text('社区使用指南'), findsOneWidget);
    expect(find.text('最近更新'), findsOneWidget);
    expect(find.text('内容规范'), findsOneWidget);
    expect(find.text('guide/content'), findsOneWidget);

    // 点最近更新条目 → 以 wiki 路径推入详情页并走页面通道。
    await tester.tap(find.text('内容规范'));
    await tester.pumpAndSettle();

    expect(pages.fetchedPaths, <String>['/wiki/guide/content']);
    expect(find.text('内容规范'), findsOneWidget);
    expect(router.state.uri.path, '/wiki/guide/content');
  });

  testWidgets('wiki 详情渲染标题、HTML 正文与 meta 脚注', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
        ),
      },
    );
    await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    expect(pages.fetchedPaths, <String>['/wiki/guide/getting-started']);
    expect(find.text('快速开始'), findsOneWidget);
    // 正文是服务端渲染 HTML(标题 + 段落)。
    expect(
      find.textContaining('欢迎阅读 wiki 正文', findRichText: true),
      findsOneWidget,
    );
    expect(find.textContaining('介绍', findRichText: true), findsWidgets);
    // meta 脚注:更新时间 + 浏览数 + 点赞数。
    expect(find.text('2026-08-10 15:00'), findsOneWidget);
    expect(find.text('42 次浏览'), findsOneWidget);
    expect(find.text('5'), findsOneWidget);
  });

  testWidgets('目录 sheet 列出 toc 条目', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
        ),
      },
    );
    await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    await tester.tap(find.text('目录'));
    await tester.pumpAndSettle();

    expect(find.text('目录'), findsWidgets);
    expect(find.text('介绍'), findsOneWidget);
    expect(find.text('细节说明'), findsOneWidget);

    // 点击遮罩关闭 sheet。
    await tester.tapAt(const Offset(400, 60));
    await tester.pumpAndSettle();
    expect(find.text('细节说明'), findsNothing);
  });

  testWidgets('wiki 详情加载失败展示错误与重试', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
        ),
      },
      failuresBeforeSuccess: 1,
    );
    await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    expect(find.text('加载失败'), findsOneWidget);

    await tester.tap(find.text('重试'));
    await tester.pumpAndSettle();

    expect(pages.fetchedPaths, hasLength(2));
    expect(find.text('快速开始'), findsOneWidget);
    expect(find.text('加载失败'), findsNothing);
  });

  testWidgets('canEdit=false 且无 editUrl 时不显示编辑入口', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
          canEdit: false,
          editUrl: '',
        ),
      },
    );
    await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    expect(find.text('在 GitHub 编辑'), findsNothing);
    expect(find.text('目录'), findsOneWidget);
  });

  testWidgets('canEdit=true 且有 editUrl 时显示 GitHub 编辑入口', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
          canEdit: true,
          editUrl: 'https://github.com/yourtj/wiki/edit/main/guide/start.md',
        ),
      },
    );
    await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    expect(find.text('在 GitHub 编辑'), findsOneWidget);
  });
  testWidgets('正文内相对站内链接:中文路径单次编码跳转(issue #560)', (tester) async {
    const String encodedTarget =
        '/wiki/guide/%E9%80%89%E8%AF%BE%E6%8C%87%E5%8D%97';
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
          content: '<p><a href="$encodedTarget">选课指南</a></p>',
        ),
        encodedTarget: _detailPayload(path: 'guide/选课指南', title: '选课指南'),
      },
    );
    final GoRouter router = await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    // 服务端渲染的 href 处于 percent-encoded 态(relative_urls.go 契约)。
    await _tapLinkText(tester, find.textContaining('选课指南', findRichText: true));
    await tester.pumpAndSettle();

    // 页面通道按段编码后单次编码请求;go_router 单次解码还原目标页。
    expect(pages.fetchedPaths, <String>[
      '/wiki/guide/getting-started',
      encodedTarget,
    ]);
    expect(router.state.uri.path, encodedTarget);
    expect(find.text('选课指南'), findsOneWidget);
  });

  testWidgets('正文内相对站内链接:纯 ASCII 路径不回归', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
          content: '<p><a href="/wiki/guide/faq">常见问题</a></p>',
        ),
        '/wiki/guide/faq': _detailPayload(path: 'guide/faq', title: '常见问题'),
      },
    );
    final GoRouter router = await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    await _tapLinkText(tester, find.textContaining('常见问题', findRichText: true));
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/wiki/guide/faq');
    expect(router.state.uri.fragment, isEmpty);
    expect(pages.fetchedPaths, <String>[
      '/wiki/guide/getting-started',
      '/wiki/guide/faq',
    ]);
  });

  testWidgets('正文内绝对站内链接:中文路径同样单次编码跳转', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
          content:
              '<p><a href="http://fake.local/wiki/guide/%E9%80%89%E8%AF%BE%E6%8C%87%E5%8D%97">选课指南</a></p>',
        ),
        '/wiki/guide/%E9%80%89%E8%AF%BE%E6%8C%87%E5%8D%97': _detailPayload(
          path: 'guide/选课指南',
          title: '选课指南',
        ),
      },
    );
    final GoRouter router = await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    await _tapLinkText(tester, find.textContaining('选课指南', findRichText: true));
    await tester.pumpAndSettle();

    expect(
      router.state.uri.path,
      '/wiki/guide/%E9%80%89%E8%AF%BE%E6%8C%87%E5%8D%97',
    );
    expect(
      pages.fetchedPaths.last,
      '/wiki/guide/%E9%80%89%E8%AF%BE%E6%8C%87%E5%8D%97',
    );
  });

  testWidgets('正文内跨页锚点链接:锚点经 initialAnchor 传入不丢', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
          content: '<p><a href="/wiki/guide/details#intro">跳到细节</a></p>',
        ),
        '/wiki/guide/details': _detailPayload(
          path: 'guide/details',
          title: '细节',
          content: '<h2 id="intro">介绍</h2><p>细节正文。</p>',
        ),
      },
    );
    final GoRouter router = await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    await _tapLinkText(tester, find.textContaining('跳到细节', findRichText: true));
    await tester.pumpAndSettle();

    // 锚点不卷入路径(%23),路由 fragment 收到锚点。
    expect(router.state.uri.path, '/wiki/guide/details');
    expect(router.state.uri.fragment, 'intro');
    expect(pages.fetchedPaths, <String>[
      '/wiki/guide/getting-started',
      '/wiki/guide/details',
    ]);
  });

  testWidgets('正文内页内锚点链接:不触发页面跳转', (tester) async {
    final _FakePageRepository pages = _FakePageRepository(
      _client(),
      payloads: <String, PagePayload>{
        '/wiki/guide/getting-started': _detailPayload(
          path: 'guide/getting-started',
          title: '快速开始',
          content:
              '<h2 id="intro">介绍</h2><p>欢迎阅读 wiki 正文。</p>'
              '<p><a href="#intro">回看介绍</a></p>',
        ),
      },
    );
    await _pumpApp(
      tester,
      pages: pages,
      initialLocation: '/wiki/guide/getting-started',
    );

    await _tapLinkText(tester, find.textContaining('回看介绍', findRichText: true));
    await tester.pumpAndSettle();

    // 页内锚点交还 fwfh 内部滚动,不产生新的页面请求。
    expect(pages.fetchedPaths, <String>['/wiki/guide/getting-started']);
  });
}

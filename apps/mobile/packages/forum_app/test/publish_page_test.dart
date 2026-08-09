import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/publish/publish_page.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/providers.dart';

class _MemoryTokenStorage implements TokenStorage {
  String? _token = 'token';

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

class _PublishPageRepository extends PageRepository {
  _PublishPageRepository(super.client, this.payload);

  final PagePayload payload;
  final List<String> paths = <String>[];

  @override
  Future<PagePayload> fetch(String path) async {
    paths.add(path);
    return payload;
  }
}

class _CountingMarkdownConverter extends MarkdownConverter {
  int documentToMarkdownCalls = 0;

  @override
  String documentToMarkdown(Document document) {
    documentToMarkdownCalls++;
    return super.documentToMarkdown(document);
  }
}

class _RecordingTopicRepository extends TopicRepository {
  _RecordingTopicRepository(super.client, {this.resultId = 99});

  final int resultId;
  final List<
    ({
      int topicId,
      String title,
      String content,
      List<int> categoryIds,
      int topicStatus,
    })
  >
  writes =
      <
        ({
          int topicId,
          String title,
          String content,
          List<int> categoryIds,
          int topicStatus,
        })
      >[];

  @override
  Future<int> writeTopic({
    required int topicId,
    required String title,
    required String content,
    required List<int> categoryIds,
    required int topicStatus,
    String? captchaId,
    String? captchaCode,
  }) async {
    writes.add((
      topicId: topicId,
      title: title,
      content: content,
      categoryIds: List<int>.of(categoryIds),
      topicStatus: topicStatus,
    ));
    return resultId;
  }
}

PagePayload _publishPayload({required bool editing}) {
  return PagePayload.fromJson(<String, dynamic>{
    'component': PageComponent.publish,
    'props': <String, dynamic>{
      'topicId': editing ? 42 : 0,
      'isEditing': editing,
      'categories': <Object>[
        <String, Object>{'id': 1, 'name': '校园', 'color': '#2563eb'},
        <String, Object>{'id': 2, 'name': '开发', 'color': '#16a34a'},
        <String, Object>{'id': 3, 'name': '生活', 'color': '#d97706'},
        <String, Object>{'id': 4, 'name': '闲聊', 'color': '#7c3aed'},
      ],
      'topic': <String, dynamic>{
        'title': editing ? '原始标题' : '',
        'content': editing ? '## 预览标题\n\n**正文内容**' : '',
        'categoryIds': editing ? <int>[2] : null,
        'topicStatus': editing ? 1 : 0,
      },
    },
    'meta': <String, dynamic>{'title': editing ? '编辑话题' : '发布话题'},
    'layout': <String, dynamic>{
      'site': <String, dynamic>{
        'name': 'yourtj',
        'description': '',
        'logo': '',
        'favicon': '',
        'brandType': 'text',
        'brandText': 'yourtj',
        'brandImage': '',
      },
      'viewer': <String, dynamic>{
        'id': 1,
        'username': 'alice',
        'email': '',
        'avatarUrl': '',
        'isAuthenticated': true,
        'canAccessAdmin': false,
        'isModerator': false,
        'requiresEmailVerification': false,
      },
      'sidebar': <String, dynamic>{'categories': <Object>[], 'activeKey': ''},
      'footer': <String, dynamic>{'links': <Object>[], 'primary': <Object>[]},
      'unread': <String, dynamic>{'notifications': false, 'messages': false},
      'theme': <String, dynamic>{
        'enabled': false,
        'current': 'light',
        'themeColor': '#2563eb',
      },
    },
    'url': editing ? '/publish?id=42' : '/publish',
    'version': '1.0',
  });
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  Future<
    ({
      GoRouter router,
      _PublishPageRepository pageRepository,
      _RecordingTopicRepository topicRepository,
    })
  >
  pumpPublishPage(
    WidgetTester tester, {
    required bool editing,
    String editQueryKey = 'topicId',
    int resultId = 99,
    MarkdownConverter? markdownConverter,
  }) async {
    final _MemoryTokenStorage storage = _MemoryTokenStorage();
    final GfApiClient client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final _PublishPageRepository pageRepository = _PublishPageRepository(
      client,
      _publishPayload(editing: editing),
    );
    final _RecordingTopicRepository topicRepository = _RecordingTopicRepository(
      client,
      resultId: resultId,
    );
    final GoRouter router = GoRouter(
      initialLocation: editing ? '/publish?$editQueryKey=42' : '/publish',
      routes: <RouteBase>[
        GoRoute(
          path: '/publish',
          builder: (BuildContext context, GoRouterState state) => PublishPage(
            topicId: publishTopicIdFromUri(state.uri),
            markdownConverter: markdownConverter,
          ),
        ),
        GoRoute(
          path: '/p/:id',
          builder: (BuildContext context, GoRouterState state) => Scaffold(
            body: Center(child: Text('topic-${state.pathParameters['id']}')),
          ),
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: <Override>[
          tokenStorageProvider.overrideWithValue(storage),
          pageRepositoryProvider.overrideWithValue(pageRepository),
          topicRepositoryProvider.overrideWithValue(topicRepository),
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

    return (
      router: router,
      pageRepository: pageRepository,
      topicRepository: topicRepository,
    );
  }

  testWidgets('编辑模式加载完整载荷，窄屏可在编辑与实时预览间切换', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    final result = await pumpPublishPage(tester, editing: true);

    expect(result.pageRepository.paths, <String>['/publish?id=42']);
    expect(find.text('原始标题'), findsOneWidget);
    expect(find.byKey(const Key('publish-editor')), findsOneWidget);
    expect(find.byKey(const Key('publish-preview')), findsNothing);

    await tester.tap(find.text('预览'));
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.byKey(const Key('publish-editor')), findsNothing);
    expect(find.byKey(const Key('publish-preview')), findsOneWidget);
    expect(find.text('预览标题'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('宽屏实时预览在输入停止 200ms 后更新', (tester) async {
    tester.view.physicalSize = const Size(1000, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    await pumpPublishPage(tester, editing: true);
    final Finder previewText = find.descendant(
      of: find.byKey(const Key('publish-preview')),
      matching: find.text('节流后的正文'),
    );
    final QuillController controller = tester
        .widget<QuillEditor>(find.byType(QuillEditor))
        .controller;

    controller.replaceText(
      0,
      controller.document.length - 1,
      '节流后的正文',
      const TextSelection.collapsed(offset: 6),
    );
    await tester.pump(const Duration(milliseconds: 199));
    expect(previewText, findsNothing);

    await tester.pump(const Duration(milliseconds: 1));
    expect(previewText, findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('仅移动编辑器选区不会重新转换实时预览', (tester) async {
    tester.view.physicalSize = const Size(1000, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final _CountingMarkdownConverter converter = _CountingMarkdownConverter();

    await pumpPublishPage(tester, editing: true, markdownConverter: converter);
    final QuillController controller = tester
        .widget<QuillEditor>(find.byType(QuillEditor))
        .controller;
    final int callsBeforeSelection = converter.documentToMarkdownCalls;

    controller.updateSelection(
      const TextSelection.collapsed(offset: 1),
      ChangeSource.local,
    );
    await tester.pump(const Duration(milliseconds: 250));

    expect(converter.documentToMarkdownCalls, callsBeforeSelection);

    controller.replaceText(
      0,
      0,
      '补充',
      const TextSelection.collapsed(offset: 2),
    );
    await tester.pump(const Duration(milliseconds: 250));

    expect(
      converter.documentToMarkdownCalls,
      greaterThan(callsBeforeSelection),
    );

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('服务端草稿 editUrl 的 id 参数进入编辑模式', (tester) async {
    final result = await pumpPublishPage(
      tester,
      editing: true,
      editQueryKey: 'id',
    );

    expect(result.router.state.uri.queryParameters, <String, String>{
      'id': '42',
    });
    expect(result.pageRepository.paths, <String>['/publish?id=42']);
    expect(find.text('原始标题'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('发布提交编辑载荷并替换到话题详情路由', (tester) async {
    final result = await pumpPublishPage(tester, editing: true, resultId: 99);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();

    expect(result.topicRepository.writes, hasLength(1));
    final write = result.topicRepository.writes.single;
    expect(write.topicId, 42);
    expect(write.title, '原始标题');
    expect(write.content, contains('预览标题'));
    expect(write.categoryIds, <int>[2]);
    expect(write.topicStatus, 1);
    expect(result.router.state.uri.path, '/p/99');
    expect(find.text('topic-99'), findsOneWidget);

    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('保存草稿停留编辑页并显示成功反馈', (tester) async {
    final result = await pumpPublishPage(tester, editing: true, resultId: 55);

    await tester.ensureVisible(find.byKey(const Key('publish-save-draft')));
    await tester.tap(find.byKey(const Key('publish-save-draft')));
    await tester.pumpAndSettle();

    expect(result.topicRepository.writes, hasLength(1));
    expect(result.topicRepository.writes.single.topicStatus, 0);
    expect(result.router.state.uri.path, '/publish');
    expect(result.router.state.uri.queryParameters['topicId'], '42');
    expect(find.text('已保存为草稿'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('无分类保存草稿和发布都要求选择分类', (tester) async {
    final result = await pumpPublishPage(tester, editing: false, resultId: 55);
    await tester.enterText(find.byType(TextField).first, '无分类草稿');
    final QuillController controller = tester
        .widget<QuillEditor>(find.byType(QuillEditor))
        .controller;
    controller.replaceText(
      0,
      controller.document.length - 1,
      '草稿正文',
      const TextSelection.collapsed(offset: 4),
    );
    await tester.pump(const Duration(milliseconds: 250));

    await tester.ensureVisible(find.byKey(const Key('publish-save-draft')));
    await tester.tap(find.byKey(const Key('publish-save-draft')));
    await tester.pump();

    expect(find.text('请至少选择一个分类'), findsOneWidget);
    expect(result.topicRepository.writes, isEmpty);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pump();
    expect(find.text('请至少选择一个分类'), findsOneWidget);
    expect(result.topicRepository.writes, isEmpty);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('空表单在提交动作附近显示明确校验错误', (tester) async {
    await pumpPublishPage(tester, editing: false);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pump();

    expect(find.text('标题不能为空'), findsOneWidget);
    expect(find.byType(GfStatusMessage), findsOneWidget);
  });
}

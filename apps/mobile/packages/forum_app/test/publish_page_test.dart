import 'dart:convert';
import 'dart:ui' show SemanticsAction, Tristate;
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/current_user.dart';

import 'package:core/core.dart';
import 'package:image/image.dart' as img;
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_quill/quill_delta.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/publish/embed_image_move.dart';
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
  bool offline = false;
  final List<String> paths = <String>[];

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    paths.add(path);
    if (offline) throw const NetworkException(fallbackMessage: 'offline');
    return payload;
  }
}

class _FailingWritingStore extends WritingStore {
  bool fail = true;
  @override
  Future<void> save(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) async {
    if (fail) throw StateError('disk unavailable');
    await super.save(scope, draft, isCurrent: isCurrent);
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

class _RejectingMarkdownConverter extends MarkdownConverter {
  @override
  Document mdToDocument(String markdown) {
    if (markdown.isNotEmpty) {
      throw const FormatException('Unsupported document');
    }
    return super.mdToDocument(markdown);
  }
}

class _RecordingTopicRepository extends TopicRepository {
  _RecordingTopicRepository(
    super.client, {
    this.resultId = 99,
    this.requireCaptcha = false,
  });
  final bool requireCaptcha;

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
    int contentType = 3,
    List<String>? images,
    String? captchaId,
    String? captchaCode,
  }) async {
    if (requireCaptcha && (captchaId != 'challenge' || captchaCode != 'ABCD')) {
      throw const ApiException(
        fallbackMessage: 'Captcha required',
        messageCode: 'common.captchaRequired',
      );
    }
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

class _CaptchaAuthRepository extends AuthRepository {
  _CaptchaAuthRepository(super.client);
  @override
  Future<CaptchaPayload> getCaptcha() async => CaptchaPayload(
    captchaId: 'challenge',
    captchaImg: base64Encode(img.encodePng(img.Image(width: 2, height: 2))),
  );
}

PagePayload _publishPayload({
  required bool editing,
  int contentType = 0,
  int? topicStatus,
  List<int>? categoryIds,
  String? content,
  bool viewerAuthenticated = true,
}) {
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
        'content': editing ? (content ?? '## 预览标题\n\n**正文内容**') : '',
        'categoryIds': categoryIds ?? (editing ? <int>[2] : null),
        'topicStatus': topicStatus ?? (editing ? 1 : 0),
        'contentType': contentType,
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
        'id': viewerAuthenticated ? 1 : 0,
        'username': viewerAuthenticated ? 'alice' : '',
        'email': '',
        'avatarUrl': '',
        'isAuthenticated': viewerAuthenticated,
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
  setUp(() => SharedPreferences.setMockInitialValues({}));

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
    String? localDraftKey,
    int contentType = 0,
    int? topicStatus,
    List<int>? categoryIds,
    Locale locale = const Locale('zh'),
    String? content,
    int resultId = 99,
    MarkdownConverter? markdownConverter,
    bool requireCaptcha = false,
    bool offline = false,
    int userId = 1,
    bool viewerAuthenticated = true,
    WritingStore? localStore,
  }) async {
    final _MemoryTokenStorage storage = _MemoryTokenStorage();
    final GfApiClient client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final _PublishPageRepository pageRepository = _PublishPageRepository(
      client,
      _publishPayload(
        editing: editing,
        contentType: contentType,
        topicStatus: topicStatus,
        categoryIds: categoryIds,
        content: content,
        viewerAuthenticated: viewerAuthenticated,
      ),
    );
    pageRepository.offline = offline;
    final _RecordingTopicRepository topicRepository = _RecordingTopicRepository(
      client,
      resultId: resultId,
      requireCaptcha: requireCaptcha,
    );
    final GoRouter router = GoRouter(
      initialLocation: editing ? '/publish?$editQueryKey=42' : '/publish',
      routes: <RouteBase>[
        GoRoute(
          path: '/',
          builder: (BuildContext context, GoRouterState state) =>
              const Scaffold(body: Center(child: Text('home'))),
        ),
        GoRoute(
          path: '/publish',
          builder: (BuildContext context, GoRouterState state) => PublishPage(
            topicId: publishTopicIdFromUri(state.uri),
            localDraftKey: localDraftKey,
            initialContentType: contentType == 0 ? 3 : contentType,
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
          apiClientProvider.overrideWithValue(client),
          currentUserProvider.overrideWith(
            (ref) async =>
                userId == 0 ? null : CurrentUser(id: userId, username: 'alice'),
          ),
          if (localStore != null)
            writingStoreProvider.overrideWithValue(localStore),
          authRepositoryProvider.overrideWithValue(
            _CaptchaAuthRepository(client),
          ),
          pageRepositoryProvider.overrideWithValue(pageRepository),
          topicRepositoryProvider.overrideWithValue(topicRepository),
        ],
        child: MaterialApp.router(
          theme: gfThemeData(Brightness.light),
          routerConfig: router,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: locale,
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

  for (final published in [true, false]) {
    testWidgets(
      'editing uses a distinct ${published ? 'published topic' : 'server draft'} identity',
      (tester) async {
        await pumpPublishPage(
          tester,
          editing: true,
          topicStatus: published ? 1 : 0,
        );
        await tester.enterText(find.byType(TextField).first, '本机修改');
        await tester.pump(const Duration(milliseconds: 800));
        await tester.pumpAndSettle();
        final draft = (await WritingStore().drafts(
          writingScope('http://fake.local', 1),
        )).single;
        expect(draft.key, published ? 'topic-edit-42' : 'server-draft-42');
        expect(
          draft.kind,
          published ? DraftKind.topicEdit : DraftKind.serverDraft,
        );
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(milliseconds: 600));
      },
    );
  }

  testWidgets('offline recovery preserves the server-draft identity', (
    tester,
  ) async {
    final scope = writingScope('http://fake.local', 1);
    await WritingStore().save(
      scope,
      const LocalDraft(
        key: 'server-draft-42',
        kind: DraftKind.serverDraft,
        title: '离线草稿',
        content: '草稿正文',
        contentType: 3,
        topicId: 42,
        categories: [],
        images: [],
        updatedAt: 1,
      ),
    );
    await pumpPublishPage(
      tester,
      editing: true,
      offline: true,
      localDraftKey: 'server-draft-42',
    );
    await tester.enterText(find.byType(TextField).first, '离线继续修改');
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    final draft = (await WritingStore().drafts(scope)).single;
    expect(draft.kind, DraftKind.serverDraft);
    expect(draft.key, 'server-draft-42');
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('cloud draft URL finds its modern recovery copy while offline', (
    tester,
  ) async {
    final scope = writingScope('http://fake.local', 1);
    await WritingStore().save(
      scope,
      const LocalDraft(
        key: 'server-draft-42',
        kind: DraftKind.serverDraft,
        title: '离线草稿',
        content: '草稿正文',
        contentType: 3,
        topicId: 42,
        categories: [],
        images: [],
        updatedAt: 1,
      ),
    );
    await pumpPublishPage(tester, editing: true, offline: true);
    expect(find.byType(TextField), findsWidgets);
    await tester.enterText(find.byType(TextField).first, '离线继续修改');
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    final draft = (await WritingStore().drafts(scope)).single;
    expect(draft.kind, DraftKind.serverDraft);
    expect(draft.key, 'server-draft-42');
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('new topics keep independent recovery drafts', (tester) async {
    await pumpPublishPage(tester, editing: false, contentType: 3);
    tester
        .widget<QuillEditor>(find.byType(QuillEditor))
        .controller
        .replaceText(0, 0, '第一篇', const TextSelection.collapsed(offset: 3));
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    await pumpPublishPage(tester, editing: false, contentType: 3);
    final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
    expect(editor.controller.document.toPlainText().trim(), isEmpty);
    editor.controller.replaceText(
      0,
      0,
      '第二篇',
      const TextSelection.collapsed(offset: 3),
    );
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    final drafts = await WritingStore().drafts(
      writingScope('http://fake.local', 1),
    );
    expect(
      drafts.map((d) => d.content.trim()),
      unorderedEquals(['第一篇', '第二篇']),
    );
    expect(drafts.map((d) => d.key).toSet(), hasLength(2));
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
  });

  testWidgets(
    'unfinished article autosaves locally and restores after reopening',
    (tester) async {
      await pumpPublishPage(tester, editing: false, contentType: 3);
      final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
      editor.controller.replaceText(
        0,
        0,
        '只写到这里',
        const TextSelection.collapsed(offset: 5),
      );
      await tester.pump(const Duration(milliseconds: 800));
      await tester.pumpAndSettle();
      expect(find.text('已保存到本机'), findsOneWidget);
      final scope = writingScope('http://fake.local', 1);
      final saved = (await WritingStore().drafts(scope)).single;
      expect(saved.title, isEmpty);
      expect(saved.categories, isEmpty);
      expect(saved.content.trim(), '只写到这里');
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      await pumpPublishPage(
        tester,
        editing: false,
        contentType: 3,
        localDraftKey: saved.key,
      );
      expect(
        tester
            .widget<QuillEditor>(find.byType(QuillEditor))
            .controller
            .document
            .toPlainText()
            .trim(),
        '只写到这里',
      );
      expect(find.text('已恢复上次未完成的内容'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );

  testWidgets('autosave failure keeps text and exposes a working retry', (
    tester,
  ) async {
    final store = _FailingWritingStore();
    await pumpPublishPage(tester, editing: false, localStore: store);
    final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
    editor.controller.replaceText(
      0,
      0,
      '请保留',
      const TextSelection.collapsed(offset: 3),
    );
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    expect(find.text('本机保存失败，请重试'), findsOneWidget);
    expect(editor.controller.document.toPlainText().trim(), '请保留');
    store.fail = false;
    await tester.tap(find.text('重试'));
    await tester.pumpAndSettle();
    expect(find.text('已保存到本机'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('offline editor restores only the signed-in account draft', (
    tester,
  ) async {
    final store = WritingStore();
    await store.save(
      writingScope('http://fake.local', 1),
      const LocalDraft(
        key: 'new-3',
        title: 'A 私有草稿',
        content: '离线继续写',
        contentType: 3,
        topicId: 0,
        categories: [],
        images: [],
        updatedAt: 1,
      ),
    );
    await pumpPublishPage(
      tester,
      editing: false,
      offline: true,
      localDraftKey: 'new-3',
    );
    expect(find.text('A 私有草稿'), findsWidgets);
    expect(
      tester
          .widget<QuillEditor>(find.byType(QuillEditor))
          .controller
          .document
          .toPlainText()
          .trim(),
      '离线继续写',
    );
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    await pumpPublishPage(
      tester,
      editing: false,
      userId: 2,
      offline: true,
      localDraftKey: 'new-3',
    );
    expect(find.text('A 私有草稿'), findsNothing);
    expect(find.byType(QuillEditor), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('server draft acknowledgement removes the local recovery copy', (
    tester,
  ) async {
    await pumpPublishPage(tester, editing: true);
    await tester.enterText(find.byType(TextField).first, '云端保存后的标题');
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    final scope = writingScope('http://fake.local', 1);
    expect(await WritingStore().drafts(scope), hasLength(1));
    await tester.tap(find.byKey(const Key('publish-save-draft')));
    await tester.pumpAndSettle();
    expect(await WritingStore().drafts(scope), isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
    expect(await WritingStore().drafts(scope), isEmpty);
  });

  testWidgets(
    'writing tools stay above keyboard and format the selected text',
    (tester) async {
      await pumpPublishPage(tester, editing: false, contentType: 3);
      final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
      editor.controller.replaceText(
        0,
        0,
        'Selected words',
        const TextSelection(baseOffset: 0, extentOffset: 8),
      );
      await tester.tap(find.byType(QuillEditor));
      editor.controller.updateSelection(
        const TextSelection(baseOffset: 0, extentOffset: 8),
        ChangeSource.local,
      );
      tester.view.viewInsets = const FakeViewPadding(bottom: 250);
      addTearDown(tester.view.resetViewInsets);
      await tester.pumpAndSettle();
      final tools = find.byKey(const Key('publish-writing-tools'));
      final keyboardTop =
          tester.view.physicalSize.height / tester.view.devicePixelRatio -
          250 / tester.view.devicePixelRatio;
      expect(
        tester.getBottomLeft(tools).dy,
        lessThanOrEqualTo(keyboardTop + 0.01),
      );
      await tester.tap(find.text('文字格式'));
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('粗体'));
      await tester.pump();
      expect(
        editor.controller.getSelectionStyle().attributes.containsKey(
          Attribute.bold.key,
        ),
        isTrue,
      );
      expect(editor.controller.document.toPlainText(), 'Selected words\n');
      expect(tester.takeException(), isNull);
      await tester.tap(find.byKey(const Key('publish-appbar-submit')));
      await tester.pumpAndSettle();
      expect(tools, findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );
  testWidgets('article input nodes survive toolbar and autosave rebuilds', (
    tester,
  ) async {
    await pumpPublishPage(tester, editing: false, contentType: 3);
    final original = tester.widget<QuillEditor>(find.byType(QuillEditor));
    original.controller.replaceText(
      0,
      0,
      'Keep editing',
      const TextSelection.collapsed(offset: 5),
    );
    original.focusNode.requestFocus();
    await tester.pump();
    await tester.tap(find.text('文字格式'));
    await tester.pumpAndSettle();
    final expanded = tester.widget<QuillEditor>(find.byType(QuillEditor));
    expect(identical(expanded.focusNode, original.focusNode), isTrue);
    expect(
      identical(expanded.scrollController, original.scrollController),
      isTrue,
    );
    expect(expanded.focusNode.hasFocus, isTrue);
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    expect(
      tester.widget<QuillEditor>(find.byType(QuillEditor)).focusNode.hasFocus,
      isTrue,
    );
    expect(original.controller.document.toPlainText(), 'Keep editing\n');
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 800));
  });

  testWidgets(
    'article preview return restores the active body selection and focus',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await pumpPublishPage(tester, editing: false, contentType: 3);
      final original = tester.widget<QuillEditor>(find.byType(QuillEditor));
      original.controller.replaceText(
        0,
        0,
        'Selected words',
        const TextSelection(baseOffset: 0, extentOffset: 8),
      );
      original.focusNode.requestFocus();
      await tester.pumpAndSettle();
      final selection = original.controller.selection;
      final before = original.controller.document.toDelta();
      await tester.tap(find.byKey(const Key('publish-appbar-submit')));
      await tester.pumpAndSettle();
      expect(find.byType(QuillEditor), findsNothing);
      await tester.tap(find.byTooltip('返回'));
      await tester.pumpAndSettle();
      final restored = tester.widget<QuillEditor>(find.byType(QuillEditor));
      expect(restored.focusNode.hasFocus, isTrue);
      expect(restored.controller.selection, selection);
      expect(restored.controller.document.toDelta(), before);
      expect(identical(restored.controller, original.controller), isTrue);
      expect(restored.controller.hasUndo, isTrue);
      restored.controller.undo();
      expect(restored.controller.document.toPlainText(), '\n');
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 800));
    },
  );

  testWidgets(
    'article preview restores reading position without opening keyboard',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await pumpPublishPage(
        tester,
        editing: true,
        contentType: 3,
        content: List.generate(50, (index) => 'Paragraph $index').join('\n\n'),
      );
      final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
      final scroll = tester
          .widget<SingleChildScrollView>(
            find
                .ancestor(
                  of: find.byKey(const Key('publish-editor')),
                  matching: find.byType(SingleChildScrollView),
                )
                .first,
          )
          .controller!;
      editor.focusNode.requestFocus();
      tester.view.viewInsets = const FakeViewPadding(bottom: 250);
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('收起键盘'));
      tester.view.resetViewInsets();
      await tester.pumpAndSettle();
      scroll.jumpTo(500);
      await tester.pumpAndSettle();
      expect(editor.focusNode.hasFocus, isFalse);
      final before = editor.controller.document.toDelta();
      await tester.tap(find.byKey(const Key('publish-appbar-submit')));
      await tester.pumpAndSettle();
      scroll.jumpTo(900);
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('返回'));
      await tester.pumpAndSettle();
      expect(scroll.offset, closeTo(500, 1));
      expect(editor.focusNode.hasFocus, isFalse);
      expect(editor.controller.document.toDelta(), before);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 800));
    },
  );

  for (final language in ['zh', 'en', 'ja', 'de']) {
    for (final width in [320.0, 1024.0]) {
      testWidgets('article toolbar at 200% text in $language on $width', (
        tester,
      ) async {
        tester.view.physicalSize = Size(width, 900);
        tester.view.devicePixelRatio = 1;
        tester.platformDispatcher.textScaleFactorTestValue = 2;
        addTearDown(tester.view.reset);
        addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
        await pumpPublishPage(
          tester,
          editing: false,
          contentType: 3,
          locale: Locale(language),
        );
        final l10n = AppLocalizations.of(
          tester.element(find.byType(PublishPage)),
        );
        await tester.tap(find.text(l10n.publishFormatting));
        await tester.pumpAndSettle();
        expect(find.byTooltip(l10n.publishUndo), findsOneWidget);
        expect(find.byTooltip(l10n.publishToolBold), findsOneWidget);
        await tester.tap(find.byKey(const Key('publish-appbar-submit')));
        await tester.pumpAndSettle();
        await tester.tap(find.byTooltip(l10n.commonBack));
        await tester.pumpAndSettle();
        expect(find.byType(QuillEditor), findsOneWidget);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(milliseconds: 800));
      });
    }
  }

  testWidgets('article toolbar follows undo history and selected formatting', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await pumpPublishPage(tester, editing: false, contentType: 3);
    final l10n = AppLocalizations.of(tester.element(find.byType(PublishPage)));
    await tester.tap(find.text('文字格式'));
    await tester.pumpAndSettle();
    GfIconButton button(String label) => tester.widget<GfIconButton>(
      find.ancestor(
        of: find.byTooltip(label),
        matching: find.byType(GfIconButton),
      ),
    );
    expect(button('撤销').onPressed, isNull);
    expect(button('重做').onPressed, isNull);
    final undoSemantics = tester
        .getSemantics(find.byTooltip(l10n.publishUndo))
        .getSemanticsData();
    expect(undoSemantics.label, l10n.publishUndo);
    expect(undoSemantics.flagsCollection.isEnabled, Tristate.isFalse);
    expect(undoSemantics.hasAction(SemanticsAction.tap), isFalse);
    final controller = tester
        .widget<QuillEditor>(find.byType(QuillEditor))
        .controller;
    controller.replaceText(
      0,
      0,
      'Bold\nPlain',
      const TextSelection(baseOffset: 0, extentOffset: 4),
    );
    controller.formatSelection(Attribute.bold);
    await tester.pump();
    expect(button('撤销').onPressed, isNotNull);
    final boldSemantics = tester
        .getSemantics(find.byTooltip(l10n.publishToolBold))
        .getSemanticsData();
    expect(boldSemantics.label, l10n.publishToolBold);
    expect(boldSemantics.flagsCollection.isButton, isTrue);
    expect(boldSemantics.flagsCollection.isEnabled, Tristate.isTrue);
    expect(boldSemantics.hasAction(SemanticsAction.tap), isTrue);
    expect(boldSemantics.flagsCollection.isToggled, Tristate.isTrue);
    controller.updateSelection(
      const TextSelection.collapsed(offset: 7),
      ChangeSource.local,
    );
    await tester.pump();
    expect(
      tester
          .getSemantics(find.byTooltip(l10n.publishToolBold))
          .getSemanticsData()
          .flagsCollection
          .isToggled,
      Tristate.isFalse,
    );
    controller.undo();
    await tester.pump();
    expect(button('重做').onPressed, isNotNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 800));
    semantics.dispose();
  });

  testWidgets('heading level picker applies h1/h2/h3 from the toolbar', (
    tester,
  ) async {
    await pumpPublishPage(tester, editing: false, contentType: 3);
    final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
    editor.controller.replaceText(
      0,
      0,
      'Selected words',
      const TextSelection(baseOffset: 0, extentOffset: 8),
    );
    await tester.tap(find.byType(QuillEditor));
    editor.controller.updateSelection(
      const TextSelection(baseOffset: 0, extentOffset: 8),
      ChangeSource.local,
    );
    tester.view.viewInsets = const FakeViewPadding(bottom: 250);
    addTearDown(tester.view.resetViewInsets);
    await tester.pumpAndSettle();
    await tester.tap(find.text('文字格式'));
    await tester.pumpAndSettle();

    int? headerLevel() =>
        editor.controller
                .getSelectionStyle()
                .attributes[Attribute.header.key]
                ?.value
            as int?;

    // 轻点保持默认行为:应用二级标题。
    await tester.tap(find.byTooltip('标题 · 长按选级别'));
    await tester.pump();
    expect(headerLevel(), 2);

    // 长按打开级别菜单,提供 H1-H3 三个选项。
    await tester.longPress(find.byTooltip('标题 · 长按选级别'));
    await tester.pumpAndSettle();
    expect(find.text('一级标题'), findsOneWidget);
    expect(find.text('二级标题'), findsOneWidget);
    expect(find.text('三级标题'), findsOneWidget);
    await tester.tap(find.text('一级标题'));
    await tester.pumpAndSettle();
    expect(headerLevel(), 1);

    // 再次长按可切换到三级标题。
    await tester.longPress(find.byTooltip('标题 · 长按选级别'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('三级标题'));
    await tester.pumpAndSettle();
    expect(headerLevel(), 3);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  for (final type in [2, 3]) {
    testWidgets('dismiss keyboard preserves type $type draft', (tester) async {
      await pumpPublishPage(tester, editing: false, contentType: type);
      final editor = type == 3
          ? find.byType(QuillEditor)
          : find.byKey(const Key('publish-editor'));
      await tester.ensureVisible(editor);
      await tester.tap(editor);
      await tester.pump();
      final inputFocus = type == 3
          ? tester.widget<QuillEditor>(editor).focusNode
          : tester.widget<TextField>(editor).focusNode;
      final rich = type == 3
          ? tester.widget<QuillEditor>(editor).controller
          : null;
      final simple = type == 2
          ? tester.widget<TextField>(editor).controller
          : null;
      if (rich != null) rich.replaceText(0, 0, 'Unsent article', null);
      if (simple != null) simple.text = 'Unsent moment';
      final draft = rich?.document.toPlainText() ?? simple!.text;
      final editingFocus = inputFocus ?? FocusManager.instance.primaryFocus!;
      expect(editingFocus.hasFocus, isTrue);
      tester.view.viewInsets = const FakeViewPadding(bottom: 250);
      addTearDown(tester.view.resetViewInsets);
      await tester.pump();
      expect(find.byTooltip('收起键盘'), findsOneWidget);
      await tester.tap(find.byTooltip('收起键盘'));
      await tester.pump();
      expect(editingFocus.hasFocus, isFalse);
      expect(rich?.document.toPlainText() ?? simple!.text, draft);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    });
  }

  testWidgets(
    'an unreadable existing gallery blocks editing instead of clearing photos',
    (tester) async {
      final result = await pumpPublishPage(
        tester,
        editing: true,
        contentType: 2,
      );
      expect(find.byKey(const Key('publish-editor')), findsNothing);
      expect(
        tester
            .widget<GfButton>(find.byKey(const Key('publish-appbar-submit')))
            .onPressed,
        isNull,
      );
      expect(result.topicRepository.writes, isEmpty);
    },
  );

  testWidgets('a document parse failure can be retried and disposed safely', (
    tester,
  ) async {
    await pumpPublishPage(
      tester,
      editing: true,
      markdownConverter: _RejectingMarkdownConverter(),
    );
    expect(
      tester
          .widget<GfButton>(find.byKey(const Key('publish-appbar-submit')))
          .onPressed,
      isNull,
    );
    await tester.tap(find.text('重试'));
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
    expect(tester.takeException(), isNull);
  });

  testWidgets('编辑模式加载完整载荷，窄屏可在编辑与实时预览间切换', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    final result = await pumpPublishPage(tester, editing: true);

    expect(result.pageRepository.paths, <String>['/publish?id=42']);
    expect(
      tester.widget<TextField>(find.byType(TextField).first).controller!.text,
      '原始标题',
    );
    expect(find.byKey(const Key('publish-editor')), findsOneWidget);
    expect(find.byKey(const Key('publish-preview')), findsNothing);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
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

  testWidgets(
    'required publishing captcha loads a challenge and retries without losing the draft',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final result = await pumpPublishPage(
        tester,
        editing: true,
        requireCaptcha: true,
      );
      await tester.tap(find.byKey(const Key('publish-appbar-submit')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('publish-appbar-submit')));
      await tester.pumpAndSettle();
      final captcha = find.byWidgetPredicate(
        (w) => w is TextField && w.decoration?.labelText == '验证码',
      );
      expect(captcha, findsOneWidget);
      expect(find.byType(GfCaptchaImage), findsOneWidget);
      expect(find.text('原始标题'), findsOneWidget);
      await tester.enterText(captcha, 'ABCD');
      await tester.tap(find.byKey(const Key('publish-appbar-submit')));
      await tester.pumpAndSettle();
      expect(result.topicRepository.writes.single.title, '原始标题');
      expect(result.router.state.uri.path, '/p/99');
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    },
  );

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
    expect(
      tester.widget<TextField>(find.byType(TextField).first).controller!.text,
      '原始标题',
    );

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('发布提交编辑载荷并替换到话题详情路由', (tester) async {
    final result = await pumpPublishPage(tester, editing: true, resultId: 99);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();

    expect(result.topicRepository.writes, isEmpty);
    expect(find.text('选择分区与标签'), findsOneWidget);
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

  testWidgets('无分类内容可保存在本机但发布要求选择分类', (tester) async {
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
    await tester.pumpAndSettle();

    expect(find.text('已保存到本机'), findsOneWidget);
    expect(result.topicRepository.writes, isEmpty);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();
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

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pump();
    expect(find.text('标题不能为空'), findsOneWidget);
    expect(find.byType(GfStatusMessage), findsOneWidget);
  });

  testWidgets('预览页只保留右上角发布按钮，保存草稿移入 AppBar', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await pumpPublishPage(tester, editing: true);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.byKey(const Key('publish-footer-submit')), findsNothing);
    expect(find.byKey(const Key('publish-appbar-submit')), findsOneWidget);
    final Finder saveDraft = find.byKey(const Key('publish-save-draft'));
    expect(saveDraft, findsOneWidget);
    expect(
      find.descendant(of: find.byType(GfAppBar), matching: saveDraft),
      findsOneWidget,
    );

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('瞬间/提问编辑态只保留顶部画廊图片入口', (tester) async {
    for (final type in [1, 2]) {
      await pumpPublishPage(tester, editing: false, contentType: type);
      expect(find.byTooltip('添加图片'), findsNothing);
      expect(find.text('先选图片，再记录这一刻'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 600));
    }
  });

  testWidgets('文章类型保留底部工具栏图片入口', (tester) async {
    await pumpPublishPage(tester, editing: false, contentType: 3);
    expect(find.byTooltip('添加图片'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('宽屏预览同样只有右上角发布与保存草稿', (tester) async {
    tester.view.physicalSize = const Size(1000, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await pumpPublishPage(tester, editing: true);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.byKey(const Key('publish-footer-submit')), findsNothing);
    expect(find.byKey(const Key('publish-save-draft')), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(GfAppBar),
        matching: find.byKey(const Key('publish-save-draft')),
      ),
      findsOneWidget,
    );
    expect(find.byKey(const Key('publish-appbar-submit')), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('预览步 AppBar 保存草稿写回草稿并停留在预览步', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final result = await pumpPublishPage(tester, editing: true, resultId: 55);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('publish-preview')), findsOneWidget);

    await tester.tap(find.byKey(const Key('publish-save-draft')));
    await tester.pumpAndSettle();

    expect(result.topicRepository.writes, hasLength(1));
    expect(result.topicRepository.writes.single.topicStatus, 0);
    expect(result.topicRepository.writes.single.categoryIds, <int>[2]);
    expect(result.router.state.uri.path, '/publish');
    expect(result.router.state.uri.queryParameters['topicId'], '42');
    expect(find.byKey(const Key('publish-preview')), findsOneWidget);
    expect(find.text('已保存为草稿'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('预览步返回回到编辑步而不是离开页面', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final result = await pumpPublishPage(tester, editing: true);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('publish-preview')), findsOneWidget);

    await tester.tap(find.byTooltip('返回'));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('publish-editor')), findsOneWidget);
    expect(find.byKey(const Key('publish-preview')), findsNothing);
    expect(result.router.state.uri.path, '/publish');

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('编辑态有未保存改动时返回先确认，可留在编辑步', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await pumpPublishPage(tester, editing: true);

    await tester.enterText(find.byType(TextField).first, '改动后的标题');
    await tester.pump(const Duration(milliseconds: 250));

    await tester.tap(find.byTooltip('返回'));
    await tester.pumpAndSettle();

    expect(find.text('保留这次创作？'), findsOneWidget);
    expect(find.text('可以保存到本机后离开，或放弃本次修改。'), findsOneWidget);

    await tester.tap(find.text('继续编辑'));
    await tester.pumpAndSettle();

    expect(find.text('保留这次创作？'), findsNothing);
    expect(find.byKey(const Key('publish-editor')), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('预览步无分类仍可保存本机草稿但发布要求分类', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final result = await pumpPublishPage(
      tester,
      editing: true,
      categoryIds: const <int>[],
      resultId: 55,
    );

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('publish-preview')), findsOneWidget);
    expect(result.topicRepository.writes, isEmpty);

    await tester.tap(find.byKey(const Key('publish-save-draft')));
    await tester.pumpAndSettle();
    expect(find.text('已保存到本机'), findsOneWidget);
    expect(result.topicRepository.writes, isEmpty);

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();
    expect(find.text('请至少选择一个分类'), findsOneWidget);
    expect(result.topicRepository.writes, isEmpty);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('德语窄屏预览步 AppBar 的存草稿与发布按钮不溢出', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await pumpPublishPage(tester, editing: true, locale: const Locale('de'));

    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('publish-preview')), findsOneWidget);
    expect(find.byTooltip('Entwurf speichern'), findsOneWidget);
    expect(find.text('Veröffentlichen'), findsOneWidget);
    // The overflow assertion is implicit: an unhandled RenderFlex overflow is
    // reported as a failure by the test binding (and prints the offending row).

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  for (final contentType in [1, 2, 3]) {
    testWidgets(
      'type $contentType stays usable with large German text on 320px',
      (tester) async {
        tester.view.physicalSize = const Size(320, 700);
        tester.view.devicePixelRatio = 1;
        tester.platformDispatcher.textScaleFactorTestValue = 1.6;
        addTearDown(tester.view.reset);
        addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
        await pumpPublishPage(
          tester,
          editing: false,
          contentType: contentType,
          locale: const Locale('de'),
        );
        await tester.tap(find.byKey(const Key('publish-appbar-submit')));
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('publish-preview')), findsOneWidget);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(milliseconds: 600));
      },
    );
  }

  testWidgets('正文图片支持长按拖拽到其他段落', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    await pumpPublishPage(
      tester,
      editing: true,
      contentType: 3,
      content: '第一段\n\n![image](u1)\n\n第三段\n',
    );
    expect(find.text('长按正文图片，可拖动到任意段落位置'), findsOneWidget);

    QuillController controllerOfEditor() =>
        tester.widget<QuillEditor>(find.byType(QuillEditor)).controller;

    String flatOf(Document document) => document
        .toDelta()
        .toList()
        .map((Operation op) => op.data is String ? op.data as String : '\uFFFC')
        .join();

    final QuillController controller = controllerOfEditor();
    final String before = flatOf(controller.document);
    expect(before.indexOf('\uFFFC'), greaterThan(before.indexOf('第一段')));
    expect(before.indexOf('\uFFFC'), lessThan(before.indexOf('第三段')));

    final Finder draggable = find.descendant(
      of: find.byType(QuillEditor),
      matching: find.byType(LongPressDraggable<ComposerImageDragPayload>),
    );
    expect(draggable, findsOneWidget);

    final TestGesture gesture = await tester.startGesture(
      tester.getCenter(draggable),
    );
    await tester.pump(const Duration(milliseconds: 600));
    await gesture.moveBy(const Offset(0, 140));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    final String after = flatOf(controllerOfEditor().document);
    expect(after, isNot(before));
    // Drop semantics: the image lands directly below the dropped-on
    // paragraph — dragging onto 第三段 moves the image after it.
    expect(after.indexOf('\uFFFC'), greaterThan(after.indexOf('第三段')));

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('拖拽自动滚动挂载页面滚动控制器', (tester) async {
    await pumpPublishPage(
      tester,
      editing: true,
      contentType: 3,
      content: '第一段\n\n![image](u1)\n\n第三段\n',
    );

    // Edge auto-scroll must drive the page scroll view; a detached
    // controller makes every autoscroll tick a no-op.
    final Finder pageScroll = find.ancestor(
      of: find.byKey(const Key('publish-editor')),
      matching: find.byType(SingleChildScrollView),
    );
    final SingleChildScrollView view = tester.widget<SingleChildScrollView>(
      pageScroll.first,
    );
    expect(view.controller, isNotNull);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('访客会话不显示本机保存状态且可正常离开（#705 回归）', (tester) async {
    await pumpPublishPage(
      tester,
      editing: false,
      userId: 0,
      viewerAuthenticated: false,
    );
    final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
    editor.controller.replaceText(
      0,
      0,
      'guest draft',
      const TextSelection.collapsed(offset: 11),
    );
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
    // 无账号会话没有本机持久化：不得出现悬挂的“正在保存…”或任何保存失败状态。
    expect(find.text('正在保存…'), findsNothing);
    expect(find.text('本机保存失败，请重试'), findsNothing);
    expect(find.text('已保存到本机'), findsNothing);

    // “放弃修改”离开：guest 无本地草稿可删，删除路径被跳过，可直接离开。
    await tester.tap(find.byTooltip('返回'));
    await tester.pumpAndSettle();
    expect(find.text('保留这次创作？'), findsOneWidget);
    await tester.tap(find.text('放弃修改'));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('publish-editor')), findsNothing);
    expect(find.text('home'), findsOneWidget);

    // 与本文件其他用例一致的收尾：让残留的自动保存/预览去抖定时器走完。
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 800));
  });
}

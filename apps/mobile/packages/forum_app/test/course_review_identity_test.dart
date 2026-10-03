import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/asset_url.dart';
import 'package:forum_app/src/pages/courses/review_form_sheet.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  Future<BuildContext> pumpSheet(
    WidgetTester tester, {
    required Size size,
    required double textScale,
    List<Override> overrides = const <Override>[],
  }) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = size;
    addTearDown(tester.view.reset);
    late BuildContext page;
    await tester.pumpWidget(
      ProviderScope(
        overrides: overrides,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: TextScaler.linear(textScale)),
            child: child!,
          ),
          home: Scaffold(
            body: Builder(
              builder: (context) {
                page = context;
                return const SizedBox();
              },
            ),
          ),
        ),
      ),
    );
    showGfBottomSheet<void>(
      page,
      height: 600,
      keyboardAware: true,
      builder: (_) => CourseReviewFormSheet(
        pageContext: page,
        offerings: const <CourseOfferingPayload>[],
        repository: CourseRepository(
          GfApiClient(
            dio: Dio(),
            tokenStorage: MemoryTokenStorage(),
            baseUrl: 'http://fake.local',
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return page;
  }

  testWidgets('anonymous identity shows the full label at 320dp/200%', (
    tester,
  ) async {
    await pumpSheet(tester, size: const Size(320, 568), textScale: 2);

    // 主文案与说明分行完整展示，不再被截成「匿名发布（…」。
    expect(find.text('匿名发布'), findsOneWidget);
    expect(find.text('对公众隐藏身份'), findsOneWidget);
    expect(
      tester.renderObject<RenderParagraph>(find.text('匿名发布')).didExceedMaxLines,
      isFalse,
    );
    expect(
      tester
          .renderObject<RenderParagraph>(find.text('对公众隐藏身份'))
          .didExceedMaxLines,
      isFalse,
    );
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('non-anonymous identity previews the publisher avatar and name', (
    tester,
  ) async {
    await pumpSheet(
      tester,
      size: const Size(390, 844),
      textScale: 1,
      overrides: <Override>[
        reviewPublisherProvider.overrideWith(
          (ref) => (name: '张三', avatarUrl: '/static/pic/9.webp'),
        ),
      ],
    );

    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();

    expect(find.text('以 张三 发布'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is GfNetworkImage &&
            widget.url == resolveApiAssetUrl('/static/pic/9.webp'),
      ),
      findsOneWidget,
    );
    expect(find.text('对公众隐藏身份'), findsNothing);
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('compact identity keeps the primary label at 200% keyboard', (
    tester,
  ) async {
    await pumpSheet(tester, size: const Size(320, 568), textScale: 2);
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    await tester.pumpAndSettle();

    // 键盘把面板压到 360 以下时退回单行横向滚动：主文案仍完整存在。
    expect(find.text('匿名发布'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
}

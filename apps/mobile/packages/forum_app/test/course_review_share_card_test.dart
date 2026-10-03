import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/asset_url.dart';
import 'package:forum_app/src/widgets/share/course_review_share_card.dart';
import 'package:forum_app/src/widgets/share/share_image_preview.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';
// The app uses this transitive dependency; disable its batching timer in widget tests.
// ignore: depend_on_referenced_packages
import 'package:visibility_detector/visibility_detector.dart';

CourseDetailPayload _course() => const CourseDetailPayload(
  id: 42,
  primaryCode: '100001',
  name: '高等数学(A)上',
  department: '数学科学学院',
  creditX10: 50,
  teacherName: '张三',
  ratingAvg: 4.5,
  reviewCount: 12,
);

ReviewPayload _review({
  String kind = 'anonymous',
  String label = '匿名同学',
  String? avatarUrl,
  int? rating = 4,
  int id = 7,
  String content = '课程内容充实，**作业量**适中。',
  String contentHtml = '<p>课程内容充实，<strong>作业量</strong>适中。</p>',
}) => ReviewPayload(
  id: id,
  offeringId: 901,
  rating: rating,
  content: content,
  contentHtml: contentHtml,
  author: ReviewAuthorPayload(kind: kind, label: label, avatarUrl: avatarUrl),
  viewer: const ReviewViewerPayload(
    canEdit: false,
    canDelete: false,
    isHelpful: false,
  ),
  helpfulCount: 2,
  createdAt: '2026-09-30T08:00:00+08:00',
  updatedAt: '2026-09-30T08:00:00+08:00',
);

final Uint8List _mockPng = Uint8List.fromList(
  img.encodePng(img.Image(width: 4, height: 4, numChannels: 4)),
);

Widget _app(
  Widget card, {
  ShareImageTheme theme = const ShareImageTheme(
    'paper',
    'Paper',
    GfColors.light,
  ),
}) => ProviderScope(
  child: GfMediaScope(
    identity: 'course-review-share-card-test',
    factory:
        (
          url, {
          int? width,
          int? height,
          Set<String>? allowedOrigins,
          ResizeImagePolicy policy = ResizeImagePolicy.exact,
        }) => MemoryImage(_mockPng),
    child: MaterialApp(
      theme: gfThemeData(Brightness.light),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('zh'),
      home: Scaffold(
        body: SingleChildScrollView(
          child: ShareImageCard(
            theme: theme,
            footerTrailing: const Text('yourtj.de'),
            child: card,
          ),
        ),
      ),
    ),
  ),
);

/// WCAG 相对亮度对比度（用于校验分享卡在 5 个主题下的文字可读性）。
double _contrast(Color a, Color b) {
  final double la = a.computeLuminance();
  final double lb = b.computeLuminance();
  final double hi = la > lb ? la : lb;
  final double lo = la > lb ? lb : la;
  return (hi + 0.05) / (lo + 0.05);
}

void main() {
  VisibilityDetectorController.instance.updateInterval = Duration.zero;

  Widget card({
    ShareImageTheme theme = const ShareImageTheme(
      'paper',
      'Paper',
      GfColors.light,
    ),
    CourseDetailPayload? course,
    ReviewPayload? review,
    String offeringLabel = '25秋 · 01班 · 四平路校区',
  }) => _app(
    CourseReviewShareCard(
      theme: theme,
      course: course ?? _course(),
      review: review ?? _review(),
      offeringLabel: offeringLabel,
    ),
    theme: theme,
  );

  testWidgets('review share card reads identity, stats, body, signature', (
    tester,
  ) async {
    await tester.pumpWidget(card());
    await tester.pumpAndSettle();
    final AppLocalizations l10n = AppLocalizations.of(
      tester.element(find.byType(CourseReviewShareCard)),
    );

    // 课程身份。
    expect(find.text('高等数学(A)上'), findsOneWidget);
    expect(find.text('张三 · 数学科学学院 · 100001'), findsOneWidget);
    // 数据条：本条评分（含星级）｜课程评分｜课评数，一行读完。
    final Text mine = tester.widget<Text>(
      find.byKey(const ValueKey<String>('share-review-rating')),
    );
    final Text average = tester.widget<Text>(
      find.byKey(const ValueKey<String>('share-score')),
    );
    final Text count = tester.widget<Text>(
      find.byKey(const ValueKey<String>('share-review-count')),
    );
    expect(mine.data, '4.0');
    expect(average.data, '4.5');
    expect(average.style!.fontSize, CourseReviewShareCard.scoreSize);
    expect(count.data, '12');
    expect(find.text(l10n.courseShareThisReview), findsOneWidget);
    expect(find.text(l10n.courseCopyRatingTitle), findsOneWidget);
    expect(find.text(l10n.courseDetailReviews), findsOneWidget);
    expect(
      find.byKey(const ValueKey<String>('share-review-stars')),
      findsOneWidget,
    );
    final double statsRow = tester.getRect(find.text('4.0')).top;
    expect(tester.getRect(find.text('4.5')).top, closeTo(statsRow, .5));
    expect(tester.getRect(find.text('12')).top, closeTo(statsRow, .5));
    // 署名：作者 + 开课信息 · 日期。
    expect(find.text(l10n.courseCopyAuthorAnonymousLabel), findsOneWidget);
    expect(find.text('25秋 · 01班 · 四平路校区 · 2026-09-30'), findsOneWidget);
    expect(find.text('yourtj.de'), findsOneWidget);
    // 读序：身份 → 数据条 → 正文 → 署名。
    final Rect stats = tester.getRect(
      find.byKey(const ValueKey<String>('share-stats')),
    );
    final Rect body = tester.getRect(
      find.textContaining('课程内容充实', findRichText: true).last,
    );
    expect(
      stats.top,
      greaterThan(tester.getRect(find.text('张三 · 数学科学学院 · 100001')).bottom),
    );
    expect(body.top, greaterThan(stats.bottom));
    expect(
      tester.getRect(find.byType(GfBeamAvatar)).top,
      greaterThan(body.bottom),
    );
    // 匿名评价用 Web 同 seed 的生成头像。
    final GfBeamAvatar beam = tester.widget<GfBeamAvatar>(
      find.byType(GfBeamAvatar),
    );
    expect(beam.seed, '匿名同学-7');
    expect(
      tester.getSize(find.byType(GfBeamAvatar)),
      const Size.square(CourseReviewShareCard.avatarSize),
    );
    // 正文为长图阅读排版。
    final RichText bodyText = tester.widget<RichText>(
      find.textContaining('课程内容充实', findRichText: true).last,
    );
    expect(bodyText.text.style?.fontSize, CourseReviewShareCard.bodyFontSize);
    expect(bodyText.text.style?.height, CourseReviewShareCard.bodyLineHeight);
    expect(tester.takeException(), isNull);
  });

  testWidgets('no share text is truncated with an ellipsis', (tester) async {
    await tester.pumpWidget(
      card(
        course: _course().copyWith(
          name: '跨学科计算方法与城市系统设计专题研讨课程（荣誉课·全英文）',
          department: '马克思主义学院与人文学院联合教学中心',
          teacherName: '王小莉、欧阳明远、司马长风',
        ),
        review: _review(kind: 'member', label: '一位名字特别特别长的同学想把名字全部写出来'),
        offeringLabel: '25秋 · 17班 · 54009917 · 四平路校区 · 马克思主义学院 · 王小莉',
      ),
    );
    await tester.pumpAndSettle();

    // 必要信息全部完整出现，且没有任何一段文字被省略号截断或限行。
    expect(find.text('跨学科计算方法与城市系统设计专题研讨课程（荣誉课·全英文）'), findsOneWidget);
    expect(find.text('一位名字特别特别长的同学想把名字全部写出来'), findsOneWidget);
    for (final Text text in tester.widgetList<Text>(
      find.descendant(
        of: find.byType(CourseReviewShareCard),
        matching: find.byType(Text),
      ),
    )) {
      expect(text.maxLines, isNull, reason: text.data);
      expect(text.overflow, isNot(TextOverflow.ellipsis), reason: text.data);
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('review stars keep all five slots visible', (tester) async {
    await tester.pumpWidget(card(review: _review(rating: 3)));
    await tester.pumpAndSettle();

    final Iterable<GfSymbol> stars = tester.widgetList<GfSymbol>(
      find.descendant(
        of: find.byKey(const ValueKey<String>('share-review-stars')),
        matching: find.byType(GfSymbol),
      ),
    );
    expect(stars.map((s) => s.name).toList(), <String>[
      'star-filled',
      'star-filled',
      'star-filled',
      'star',
      'star',
    ]);
    expect(tester.takeException(), isNull);
  });

  testWidgets('share body headings use the narrow mobile scale', (
    tester,
  ) async {
    await tester.pumpWidget(
      card(review: _review(contentHtml: '<h2>课程内容</h2><p>统一的 ppt 内容。</p>')),
    );
    await tester.pumpAndSettle();

    final RichText heading = tester.widget<RichText>(
      find.textContaining('课程内容', findRichText: true).last,
    );
    double? size;
    heading.text.visitChildren((span) {
      size ??= span.style?.fontSize;
      return size == null;
    });
    expect(
      size ?? heading.text.style?.fontSize,
      CourseReviewShareCard.headingSizes[1],
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('member avatar loads through the tracked share image', (
    tester,
  ) async {
    await tester.pumpWidget(
      card(
        review: _review(
          kind: 'member',
          label: 'alice',
          avatarUrl: '/static/pic/9.webp',
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('alice'), findsOneWidget);
    final Finder tracked = find.byWidgetPredicate(
      (widget) =>
          widget is ShareImageNetworkImage &&
          widget.url == resolveApiAssetUrl('/static/pic/9.webp'),
    );
    expect(tracked, findsOneWidget);
    expect(
      tester.getSize(tracked),
      const Size.square(CourseReviewShareCard.avatarSize),
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('share card resolves body images against the API base', (
    tester,
  ) async {
    await tester.pumpWidget(
      card(
        review: _review(
          contentHtml: '<p>题图如下</p><img src="/file/img/answer.png" alt="题图">',
        ),
      ),
    );
    await tester.pumpAndSettle();

    // 相对 src 必须解析为绝对地址后再进导出卡（否则预览/导出缺图）。
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is ShareImageNetworkImage &&
            widget.url == resolveApiAssetUrl('/file/img/answer.png'),
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('share text keeps AA contrast in every theme', (tester) async {
    final AppLocalizations l10n = AppLocalizationsZh();
    for (final ShareImageTheme theme in ShareImageTheme.all(l10n)) {
      await tester.pumpWidget(card(theme: theme));
      await tester.pumpAndSettle();

      // 卡片底色上的次要文字。
      final Color surface = theme.colors.base100;
      for (final String key in <String>[
        'share-course-identity',
        'share-author-meta',
      ]) {
        final Text text = tester.widget<Text>(
          find.byKey(ValueKey<String>(key)),
        );
        expect(
          _contrast(Color.alphaBlend(text.style!.color!, surface), surface),
          greaterThanOrEqualTo(4.5),
          reason: '$key contrast failed for theme ${theme.id}',
        );
      }
      // 数据条底色上的数字与标签。
      final Color panel = theme.brightness == Brightness.dark
          ? theme.colors.base300
          : theme.colors.base200;
      for (final Finder finder in <Finder>[
        find.byKey(const ValueKey<String>('share-score')),
        find.text(l10n.courseShareThisReview),
        find.text(l10n.courseCopyRatingTitle),
      ]) {
        final Text text = tester.widget<Text>(finder);
        expect(
          _contrast(Color.alphaBlend(text.style!.color!, panel), panel),
          greaterThanOrEqualTo(4.5),
          reason: 'stats contrast failed for theme ${theme.id}: $finder',
        );
      }
      expect(tester.takeException(), isNull);
    }
  });

  testWidgets('share card hides review stars without a review rating', (
    tester,
  ) async {
    await tester.pumpWidget(
      card(
        course: _course().copyWith(
          teacherName: null,
          ratingAvg: null,
          reviewCount: 0,
        ),
        review: _review(rating: null),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.byKey(const ValueKey<String>('share-review-stars')),
      findsNothing,
    );
    // 无本条评分：数据条只剩课程评分与课评数，均分缺失显示占位。
    expect(
      find.byKey(const ValueKey<String>('share-review-rating')),
      findsNothing,
    );
    expect(find.text('—'), findsOneWidget);
    expect(find.text('0'), findsOneWidget);
    expect(find.text('数学科学学院 · 100001'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}

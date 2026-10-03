import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../format.dart';
import '../../pages/courses/course_common.dart';
import '../rich_content/gf_html_content.dart';
import 'share_image_preview.dart';

/// 课评分享长图（375 逻辑宽，预览与导出同一棵树）：一张浮在主题底色上的卡片。
///
/// Design read：给在 QQ/微信转发课评的同学看的静态长图，三秒内读懂「哪门课、
/// 打几分、怎么说」。质感只来自排版与材质：底色、柔和投影、发丝线、等宽数字、
/// 精确留白；不用装饰圆环、渐变、发光或彩条。
/// 1. 课程身份：课程名 + 「教师 · 院系 · 课号」；
/// 2. 数据条：本条评分（含星级）｜课程评分｜课评数，一行读完；
/// 3. 正文：16/1.7 阅读排版，标题级差 16–19；
/// 4. 署名：发丝线之后，头像 + 作者 + 「开课信息 · 日期」。
///
/// 所有文字都允许换行，不做省略号截断。
class CourseReviewShareCard extends StatelessWidget {
  const CourseReviewShareCard({
    super.key,
    required this.theme,
    required this.course,
    required this.review,
    required this.offeringLabel,
  });

  static const double titleSize = 22;
  static const double scoreSize = 24;
  static const double avatarSize = 36;
  static const double bodyFontSize = 16;
  static const double bodyLineHeight = 1.7;
  static const double cardRadius = 20;

  /// 正文标题级差：h1–h6 = 19/18/17/16/16/16，在 16 号正文上只差 0–3 号。
  static const List<double> headingSizes = <double>[19, 18, 17, 16, 16, 16];

  final ShareImageTheme theme;
  final CourseDetailPayload course;
  final ReviewPayload review;
  final String offeringLabel;

  /// 次要文字色：iconMuted 在淡彩主题底上不足 4.5:1，统一用降透明度的正文色。
  static Color mutedOf(GfColors colors) =>
      colors.baseContent.withValues(alpha: 0.66);

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = theme.colors;
    final GfTypography type = GfTheme.typographyOf(context);
    final Color muted = mutedOf(colors);
    final bool dark = theme.brightness == Brightness.dark;
    // 深色主题的 baseContent 近白，拿来做投影会变成光晕；改用黑色。
    final Color shadow = dark ? const Color(0xFF000000) : colors.baseContent;

    final Widget card = DecoratedBox(
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(cardRadius),
        border: Border.all(color: colors.line.withValues(alpha: 0.8)),
        // 两层投影：贴边的 1px 接触影 + 柔和的环境影，色相取正文色。
        boxShadow: <BoxShadow>[
          BoxShadow(
            color: shadow.withValues(alpha: dark ? 0.4 : 0.05),
            blurRadius: 2,
            offset: const Offset(0, 1),
          ),
          BoxShadow(
            color: shadow.withValues(alpha: dark ? 0.5 : 0.07),
            blurRadius: 24,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 22, 20, 18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            _identity(colors, type, muted),
            const SizedBox(height: 16),
            _stats(l10n, copy, colors, type, muted),
            const SizedBox(height: 18),
            _body(context, colors),
            const SizedBox(height: 18),
            Container(height: 1, color: colors.line),
            const SizedBox(height: 14),
            _signature(l10n, copy, colors, type, muted),
          ],
        ),
      ),
    );

    // 手帐感点缀：只落在外边距与卡片边缘，不压文字、不改布局；全部取主题 token。
    final Color accent = theme.accent;
    return Stack(
      clipBehavior: Clip.none,
      children: <Widget>[
        // 卡片后方：右上角探出的网点，左下角一枚细圆环。
        Positioned(
          top: -14,
          right: -14,
          child: _decor(
            const Size(36, 22),
            _DotGridPainter(accent.withValues(alpha: 0.45)),
          ),
        ),
        Positioned(
          left: -20,
          bottom: -16,
          child: _decor(
            const Size(52, 52),
            _RingPainter(accent.withValues(alpha: 0.30)),
          ),
        ),
        card,
        // 卡片前方：左上角斜贴的纸胶带，右上角两颗四角闪光。
        Positioned(
          top: -10,
          left: 18,
          child: Transform.rotate(
            angle: -0.12,
            child: _decor(
              const Size(72, 22),
              // 深色底上低透明度会像污渍，提高一档。
              _TapePainter(accent.withValues(alpha: dark ? 0.36 : 0.28)),
            ),
          ),
        ),
        Positioned(
          top: -18,
          right: 28,
          child: _decor(const Size(13, 13), _SparklePainter(colors.warning)),
        ),
        Positioned(
          top: -8,
          right: 18,
          child: _decor(
            const Size(7, 7),
            _SparklePainter(colors.warning.withValues(alpha: 0.55)),
          ),
        ),
      ],
    );
  }

  static Widget _decor(Size size, CustomPainter painter) => IgnorePointer(
    child: ExcludeSemantics(
      child: CustomPaint(size: size, painter: painter),
    ),
  );

  Widget _identity(GfColors colors, GfTypography type, Color muted) {
    final String meta = <String>[
      course.teacherName?.trim() ?? '',
      course.department.trim(),
      course.primaryCode,
    ].where((String part) => part.isNotEmpty).join(' · ');
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          course.name,
          key: const ValueKey<String>('share-course-name'),
          style: type.title1.copyWith(
            fontSize: titleSize,
            height: 1.3,
            letterSpacing: -0.3,
            fontWeight: FontWeight.w800,
            color: colors.baseContent,
          ),
        ),
        const SizedBox(height: 6),
        Text(
          meta,
          key: const ValueKey<String>('share-course-identity'),
          style: type.caption.copyWith(fontSize: 13, height: 1.4, color: muted),
        ),
      ],
    );
  }

  /// 数据条：三列等高，发丝线分隔；数字 24/w800 等宽，标签 11 号。
  Widget _stats(
    AppLocalizations l10n,
    CourseCopy copy,
    GfColors colors,
    GfTypography type,
    Color muted,
  ) {
    final int? rating = review.rating;
    final bool hasReviewScore = rating != null && rating > 0;
    const List<FontFeature> tabular = <FontFeature>[
      FontFeature.tabularFigures(),
    ];
    final TextStyle number = type.title1.copyWith(
      fontSize: scoreSize,
      height: 1.1,
      letterSpacing: -0.4,
      fontWeight: FontWeight.w800,
      color: colors.baseContent,
      fontFeatures: tabular,
    );
    final TextStyle label = type.meta.copyWith(
      fontSize: 11,
      height: 1.3,
      fontWeight: FontWeight.w600,
      color: muted,
    );

    // 数字行（三列顶部对齐，星级跟在本条评分数字后）+ 标签行。
    Widget cell({
      required Key key,
      required String value,
      required String caption,
      Widget? trailing,
    }) => Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        FittedBox(
          fit: BoxFit.scaleDown,
          alignment: AlignmentDirectional.topStart,
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              Text(value, key: key, style: number),
              if (trailing != null) ...<Widget>[
                const SizedBox(width: 6),
                trailing,
              ],
            ],
          ),
        ),
        const SizedBox(height: 4),
        Text(caption, style: label),
      ],
    );

    final List<(int, Widget)> cells = <(int, Widget)>[
      if (hasReviewScore)
        (
          9,
          cell(
            key: const ValueKey<String>('share-review-rating'),
            value: formatRating(rating.toDouble()),
            caption: l10n.courseShareThisReview,
            trailing: Row(
              key: const ValueKey<String>('share-review-stars'),
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                // 未点亮的星用描边星 + 次要色，五格刻度在任何主题下都看得见。
                for (int i = 0; i < 5; i++)
                  GfSymbol(
                    i < rating ? 'star-filled' : 'star',
                    size: 12,
                    color: i < rating ? colors.warning : muted,
                  ),
              ],
            ),
          ),
        ),
      (
        5,
        cell(
          key: const ValueKey<String>('share-score'),
          value: formatRating(course.ratingAvg),
          caption: copy.ratingTitle,
        ),
      ),
      (
        4,
        cell(
          key: const ValueKey<String>('share-review-count'),
          value: '${course.reviewCount ?? 0}',
          caption: l10n.courseDetailReviews,
        ),
      ),
    ];

    return Container(
      key: const ValueKey<String>('share-stats'),
      padding: const EdgeInsets.symmetric(vertical: 12),
      // 数据条与卡片同一套材质：填充 + 发丝描边（深色主题填充取更亮的 base300，
      // 否则 base200 比卡片还暗、像挖了个洞）。
      decoration: BoxDecoration(
        color: theme.brightness == Brightness.dark
            ? colors.base300
            : colors.base200,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colors.line.withValues(alpha: 0.7)),
      ),
      child: IntrinsicHeight(
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            for (int i = 0; i < cells.length; i++) ...<Widget>[
              if (i > 0)
                VerticalDivider(
                  width: 1,
                  thickness: 1,
                  indent: 4,
                  endIndent: 4,
                  color: colors.line,
                ),
              Expanded(
                flex: cells[i].$1,
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 10),
                  child: cells[i].$2,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _body(BuildContext context, GfColors colors) {
    // 与阅读卡片同源：服务端 contentHtml 已归一化历史标题并折叠软换行。
    return GfHtmlContent(
      html: review.contentHtml,
      profile: _readingProfile(context, colors),
      colors: colors,
      customWidgetBuilder: (element) {
        if (element.localName != 'img') return null;
        final String src = (element.attributes['src'] ?? '').trim();
        if (src.isEmpty) return null;
        // 与阅读卡片同一外壳（居中、圆角、hairline），导出不漂移。
        return Container(
          margin: const EdgeInsets.symmetric(vertical: 8),
          alignment: Alignment.center,
          child: Container(
            decoration: BoxDecoration(
              border: Border.all(color: colors.line),
              borderRadius: BorderRadius.circular(12),
            ),
            clipBehavior: Clip.antiAlias,
            child: ShareImageNetworkImage(
              resolveApiAssetUrl(src),
              fit: BoxFit.contain,
              errorBuilder: (_) => const GfSymbol('image-off', size: 20),
            ),
          ),
        );
      },
    );
  }

  Widget _signature(
    AppLocalizations l10n,
    CourseCopy copy,
    GfColors colors,
    GfTypography type,
    Color muted,
  ) {
    final String meta = <String>[
      offeringLabel.trim(),
      formatDate(review.createdAt),
    ].where((String part) => part.isNotEmpty).join(' · ');
    return Row(
      children: <Widget>[
        DecoratedBox(
          // 头像加发丝圆环，与正文图片的描边语言一致。
          position: DecorationPosition.foreground,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            border: Border.all(color: colors.line),
          ),
          child: reviewAvatar(
            review,
            size: avatarSize,
            imageBuilder: (String url, Widget fallback) =>
                ShareImageNetworkImage(
                  url,
                  width: avatarSize,
                  height: avatarSize,
                  fit: BoxFit.cover,
                  cacheWidth: 108,
                  errorBuilder: (_) => fallback,
                ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Text(
                reviewAuthorLabel(review, copy),
                style: type.bodyStrong.copyWith(
                  fontSize: 14,
                  height: 1.35,
                  fontWeight: FontWeight.w700,
                  color: colors.baseContent,
                ),
              ),
              if (meta.isNotEmpty)
                Text(
                  meta,
                  key: const ValueKey<String>('share-author-meta'),
                  style: type.meta.copyWith(
                    fontSize: 12,
                    height: 1.4,
                    fontWeight: FontWeight.w400,
                    color: muted,
                    fontFeatures: const <FontFeature>[
                      FontFeature.tabularFigures(),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }

  /// 长图正文阅读排版：16/1.7 正文，段距 6；标题级差收窄（见 [headingSizes]），
  /// 标题上 12、下 6，贴近它引出的段落（接近律）。
  static GfRichContentTypography _readingProfile(
    BuildContext context,
    GfColors colors,
  ) {
    final GfRichContentTypography base = GfRichContentTypography.standard(
      typography: GfTheme.typographyOf(context),
      colors: colors,
      bodySize: bodyFontSize,
    );
    TextStyle heading(TextStyle style, int level, FontWeight weight) =>
        style.copyWith(
          fontSize: headingSizes[level],
          fontWeight: weight,
          height: 1.45,
        );
    return GfRichContentTypography(
      body: base.body.copyWith(height: bodyLineHeight),
      h1: heading(base.h1, 0, FontWeight.w700),
      h2: heading(base.h2, 1, FontWeight.w700),
      h3: heading(base.h3, 2, FontWeight.w700),
      h4: heading(base.h4, 3, FontWeight.w700),
      h5: heading(base.h5, 4, FontWeight.w600),
      h6: heading(base.h6, 5, FontWeight.w600),
      inlineCode: base.inlineCode,
      code: base.code,
      tableHeader: base.tableHeader,
      tableBody: base.tableBody,
      quote: base.quote.copyWith(height: bodyLineHeight),
      // 段距 10（行距 27.2 的 37%）：相邻的无标题段落也分得开。
      paragraphSpacing: 10,
      blockSpacing: 6,
      listIndent: base.listIndent,
      listSpacing: 6,
      quoteSide: base.quoteSide,
      quotePadding: base.quotePadding,
      codePadding: base.codePadding,
      tableCellPadding: base.tableCellPadding,
    );
  }
}

/// 半色调网点：4×3 小圆点阵。
class _DotGridPainter extends CustomPainter {
  const _DotGridPainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final Paint paint = Paint()..color = color;
    const int columns = 5;
    const int rows = 3;
    final double dx = size.width / columns;
    final double dy = size.height / rows;
    for (int r = 0; r < rows; r++) {
      for (int c = 0; c < columns; c++) {
        canvas.drawCircle(Offset(dx * (c + .5), dy * (r + .5)), 1.6, paint);
      }
    }
  }

  @override
  bool shouldRepaint(_DotGridPainter oldDelegate) => oldDelegate.color != color;
}

/// 细圆环。
class _RingPainter extends CustomPainter {
  const _RingPainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.drawCircle(
      size.center(Offset.zero),
      size.shortestSide / 2 - 1,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.5,
    );
  }

  @override
  bool shouldRepaint(_RingPainter oldDelegate) => oldDelegate.color != color;
}

/// 纸胶带：半透明色条 + 细斜纹，两端锯齿撕口。
class _TapePainter extends CustomPainter {
  const _TapePainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    const double tooth = 3;
    final Path path = Path()..moveTo(0, 0);
    path.lineTo(size.width, 0);
    // 右端锯齿
    for (double y = 0; y < size.height; y += tooth * 2) {
      path
        ..lineTo(size.width - tooth, y + tooth)
        ..lineTo(size.width, (y + tooth * 2).clamp(0, size.height));
    }
    path.lineTo(0, size.height);
    // 左端锯齿
    for (double y = size.height; y > 0; y -= tooth * 2) {
      path
        ..lineTo(tooth, y - tooth)
        ..lineTo(0, (y - tooth * 2).clamp(0, size.height));
    }
    path.close();
    canvas.drawPath(path, Paint()..color = color);
    canvas.save();
    canvas.clipPath(path);
    final Paint stripe = Paint()
      ..color = color.withValues(alpha: color.a * 0.9)
      ..strokeWidth = 2;
    for (double x = -size.height; x < size.width; x += 7) {
      canvas.drawLine(
        Offset(x, size.height),
        Offset(x + size.height, 0),
        stripe,
      );
    }
    canvas.restore();
  }

  @override
  bool shouldRepaint(_TapePainter oldDelegate) => oldDelegate.color != color;
}

/// 四角闪光（✦）：四段向内收的二次曲线。
class _SparklePainter extends CustomPainter {
  const _SparklePainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final double cx = size.width / 2;
    final double cy = size.height / 2;
    final Path path = Path()
      ..moveTo(cx, 0)
      ..quadraticBezierTo(cx, cy, size.width, cy)
      ..quadraticBezierTo(cx, cy, cx, size.height)
      ..quadraticBezierTo(cx, cy, 0, cy)
      ..quadraticBezierTo(cx, cy, cx, 0)
      ..close();
    canvas.drawPath(path, Paint()..color = color);
  }

  @override
  bool shouldRepaint(_SparklePainter oldDelegate) => oldDelegate.color != color;
}

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../asset_url.dart';
import '../../server_messages.dart';
import '../../../l10n/app_localizations.dart';

/// 课程域页面文案与纯格式化工具。
///
/// All course UI copy uses the same generated four-language catalog as the app.
class CourseCopy {
  CourseCopy(this.l10n);

  final AppLocalizations l10n;

  // ---- 目录 ----
  String get creditUnit => l10n.courseCopyCreditUnit;
  String get noTeacher => l10n.courseCopyNoTeacher;
  String get catalogEmptyTitle => l10n.courseCopyCatalogEmptyTitle;
  String get catalogEmptyDescription => l10n.courseCopyCatalogEmptyDescription;
  String get noFilterResults => l10n.courseCopyNoFilterResults;
  String get noFilterResultsDescription =>
      l10n.courseCopyNoFilterResultsDescription;
  String get clearSearch => l10n.courseCopyClearSearch;
  String get done => l10n.courseCopyDone;
  String get noOptions => l10n.courseCopyNoOptions;
  String selectedCount(int count) => l10n.courseCopySelectedCount(count);
  String get instructorInputHint => l10n.courseCopyInstructorInputHint;
  String get instructorAdd => l10n.courseCopyInstructorAdd;
  String get instructorEmptyHint => l10n.courseCopyInstructorEmptyHint;

  // ---- 详情头部 ----
  String get aliasesLabel => l10n.courseCopyAliasesLabel;
  String get legacyNamesLabel => l10n.courseCopyLegacyNamesLabel;
  String get reviewScopeTeam => l10n.courseCopyReviewScopeTeam;
  String get reviewScopeCourse => l10n.courseCopyReviewScopeCourse;
  String get teamInstructorsPrefix => l10n.courseCopyTeamInstructorsPrefix;
  String teamInstructorsSuffix(int count) =>
      l10n.courseCopyTeamInstructorsSuffix(count);

  // ---- 评分卡 ----
  String get ratingTitle => l10n.courseCopyRatingTitle;
  String get ratingOutOf => l10n.courseCopyRatingOutOf;
  String get noRatingQuiet => l10n.courseCopyNoRatingQuiet;

  // ---- 开课班级 / 聚焦 ----
  String get offeringsEmpty => l10n.courseCopyOfferingsEmpty;
  String get offeringFocusLabel => l10n.courseCopyOfferingFocusLabel;
  String get offeringFocusClear => l10n.courseCopyOfferingFocusClear;

  // ---- AI 总结 ----
  String get summaryGenerated => l10n.courseCopySummaryGenerated;
  String get summaryKeywords => l10n.courseCopySummaryKeywords;
  String get summaryPros => l10n.courseCopySummaryPros;
  String get summaryCons => l10n.courseCopySummaryCons;
  String get summaryRepresentativeReviews =>
      l10n.courseCopySummaryRepresentativeReviews;
  String get summarySentimentPositive =>
      l10n.courseCopySummarySentimentPositive;
  String get summarySentimentNeutral => l10n.courseCopySummarySentimentNeutral;
  String get summarySentimentNegative =>
      l10n.courseCopySummarySentimentNegative;
  String get summaryRefresh => l10n.courseCopySummaryRefresh;
  String get summaryExpand => l10n.courseCopySummaryExpand;
  String get summaryCollapse => l10n.courseCopySummaryCollapse;
  String get summaryDisclaimer => l10n.courseCopySummaryDisclaimer;
  String get summaryInsufficient => l10n.courseCopySummaryInsufficient;
  String get summaryLoadFailed => l10n.courseCopySummaryLoadFailed;

  String summaryConsensus(String level) => switch (level) {
    'strong_recommend' => l10n.courseCopySummaryConsensusStrongRecommend,
    'recommend' => l10n.courseCopySummaryConsensusRecommend,
    'neutral' => l10n.courseCopySummaryConsensusNeutral,
    'cautious' => l10n.courseCopySummaryConsensusCautious,
    'not_recommend' => l10n.courseCopySummaryConsensusNotRecommend,
    _ => level,
  };

  String summaryConsensusText(String level) => switch (level) {
    'strong_recommend' => l10n.courseCopySummaryConsensusTextStrongRecommend,
    'recommend' => l10n.courseCopySummaryConsensusTextRecommend,
    'neutral' => l10n.courseCopySummaryConsensusTextNeutral,
    'cautious' => l10n.courseCopySummaryConsensusTextCautious,
    'not_recommend' => l10n.courseCopySummaryConsensusTextNotRecommend,
    _ => level,
  };

  String summarySentiment(String sentiment) => switch (sentiment) {
    'positive' => summarySentimentPositive,
    'negative' => summarySentimentNegative,
    _ => summarySentimentNeutral,
  };

  // ---- 写评 / 课评操作 ----
  String get writeReviewTitle => l10n.courseCopyWriteReviewTitle;
  String get editReviewTitle => l10n.courseCopyEditReviewTitle;
  String get selectOffering => l10n.courseCopySelectOffering;
  String get ratingLabel => l10n.courseCopyRatingLabel;
  String get contentLabel => l10n.courseCopyContentLabel;
  String get contentPlaceholder => l10n.courseCopyContentPlaceholder;
  String get ratingRequired => l10n.courseCopyRatingRequired;
  String get contentRequired => l10n.courseCopyContentRequired;
  String get anonymousLabel => l10n.courseCopyAnonymousLabel;
  // 写评身份区：主文案与说明分开，避免 320dp + 大字号下被截断成
  // 「匿名发布（…」；非匿名时展示当前发布身份。
  String get anonymousTitle => l10n.courseCopyAnonymousTitle;
  String get anonymousHint => l10n.courseCopyAnonymousHint;
  String publishAs(String name) => l10n.courseCopyPublishAs(name);
  String get publishPublic => l10n.courseCopyPublishPublic;
  String get submitSuccess => l10n.courseCopySubmitSuccess;
  String get updateSuccess => l10n.courseCopyUpdateSuccess;
  String get delete => l10n.courseCopyDelete;
  String get deleteReviewTitle => l10n.courseCopyDeleteReviewTitle;
  String get confirmDeleteReview => l10n.courseCopyConfirmDeleteReview;
  String get reviewDeleted => l10n.courseCopyReviewDeleted;
  String get operationFailed => l10n.courseCopyOperationFailed;
  String get reviewsLoadFailed => l10n.courseCopyReviewsLoadFailed;
  String get authorAnonymousLabel => l10n.courseCopyAuthorAnonymousLabel;
  String get authorLegacyLabel => l10n.courseCopyAuthorLegacyLabel;

  // ---- 相关课程 / 沿革 ----
  String get relatedTeacherCoursesTitle =>
      l10n.courseCopyRelatedTeacherCoursesTitle;
  String get relatedOtherTeachersTitle =>
      l10n.courseCopyRelatedOtherTeachersTitle;
  String get relatedEmpty => l10n.courseCopyRelatedEmpty;
  String get relationEquivalent => l10n.courseCopyRelationEquivalent;
  String get relationRenamed => l10n.courseCopyRelationRenamed;
  String get relationSplit => l10n.courseCopyRelationSplit;
  String get relationMerged => l10n.courseCopyRelationMerged;
  String get relationRelated => l10n.courseCopyRelationRelated;
  String relationLabel(String type) => switch (type) {
    'EQUIVALENT' => relationEquivalent,
    'RENAMED_FROM' => relationRenamed,
    'SPLIT_FROM' => relationSplit,
    'MERGED_FROM' => relationMerged,
    'RELATED' => relationRelated,
    _ => type,
  };
}

/// 学期代码缩写（对齐 web `shortTerm`）：`2025-2026-2` → `25春`；
/// 1=秋、2=春；非标准码原样返回。
String shortTerm(String term, {String locale = 'zh'}) {
  final RegExpMatch? m = RegExp(
    r'^(\d{4})-(\d{4})-([12])$',
  ).firstMatch(term.trim());
  if (m == null) return term;
  final autumn = m.group(3) == '1';
  final season = switch (locale.split('_').first) {
    'en' => autumn ? ' Fall' : ' Spring',
    'de' => autumn ? ' WiSe' : ' SoSe',
    _ => autumn ? '秋' : '春',
  };
  return '${m.group(1)!.substring(2)}$season';
}

/// 学期排序键：标准码按 startYear*10+semester；非标准码 -1。
int termSortKey(String term) {
  final RegExpMatch? m = RegExp(
    r'^(\d{4})-(\d{4})-([12])$',
  ).firstMatch(term.trim());
  return m == null ? -1 : int.parse(m.group(1)!) * 10 + int.parse(m.group(3)!);
}

/// 最近学期按时间降序（最新在前），非标准码（如「其他」）恒置末尾。
List<String> sortedRecentTerms(List<String>? terms) {
  final List<String> list = [...?terms];
  list.sort((a, b) {
    final int ka = termSortKey(a);
    final int kb = termSortKey(b);
    if (ka < 0 && kb < 0) return 0;
    if (ka < 0) return 1;
    if (kb < 0) return -1;
    return kb - ka;
  });
  return list;
}

/// 学分展示（对齐 web `formatCredit`）：25 → 2.5，50 → 5（去掉 `.0`）。
String formatCreditText(int creditX10) {
  if (creditX10 <= 0) return '';
  final String value = (creditX10 / 10).toStringAsFixed(1);
  return value.replaceFirst(RegExp(r'\.0$'), '');
}

/// 均分展示：恒一位小数（4 → 4.0，对齐 web ratingAvg.toFixed(1)）。
String formatRating(double? ratingAvg) {
  if (ratingAvg == null || ratingAvg <= 0) return '—';
  return ratingAvg.toStringAsFixed(1);
}

/// 评分环进度渐变，逐像素对齐 web `RatingSummaryCard.vue`：SVG
/// `linearGradient(x1=0 y1=100% → x2=100% y2=0)` 映射在圆的包围盒上，整个 SVG
/// 再 `-rotate-90`，视觉上即「右下 [start] → 左上 [end]」的线性渐变。
/// 线性渐变没有角度回绕，弧起点（正上方）是两色的平滑中间调，不会出现
/// SweepGradient 首尾相接处黄/蓝硬接缝。
LinearGradient ratingRingGradient({required Color start, required Color end}) {
  return LinearGradient(
    begin: Alignment.bottomRight,
    end: Alignment.topLeft,
    colors: <Color>[start, end],
  );
}

/// Keep server-provided business reasons localized; never expose transport internals.
String courseReviewError(AppLocalizations l10n, Object error) {
  if (error is NetworkException) return l10n.courseReviewNetworkError;
  if (error is ApiException) {
    final reason = resolveErrorMessage(l10n, error);
    if (reason != l10n.commonLoadFailed) return reason;
  }
  return l10n.courseReviewUnknownError;
}

/// 开课信息收敛（better-layout：只保留读者决策需要且课程页头部未展示的信息）：
/// - `cardMeta`：课评卡片单行元信息 = 学期 · 班次 · 教师（日期由卡片拼短格式）；
///   班号/校区/院系属于课程头部或下方开课记录已展示的信息，不在卡片重复。
/// - `chip`：分享卡用的「学期 · 班级」开课信息 chip。
({String cardMeta, String chip}) offeringMetaParts(
  CourseOfferingPayload? offering, {
  required String locale,
  required int fallbackId,
}) {
  if (offering == null) {
    final String fallback = '#$fallbackId';
    return (cardMeta: fallback, chip: fallback);
  }
  final String term = shortTerm(offering.termCode, locale: locale);
  final String className = offering.className?.trim() ?? '';
  final String classLabel = <String>[
    className,
    offering.classCode?.trim() ?? '',
  ].where((String part) => part.isNotEmpty).toSet().join(' · ');
  final String teachers = <String>[
    ...?offering.instructors?.map((String name) => name.trim()),
  ].where((String part) => part.isNotEmpty).toSet().join('、');
  final String cardMeta = <String>[
    term,
    className,
    teachers,
  ].where((String part) => part.isNotEmpty).join(' · ');
  final String chip = <String>[
    term,
    classLabel,
  ].where((String part) => part.isNotEmpty).join(' · ');
  return (
    cardMeta: cardMeta.isEmpty ? '#$fallbackId' : cardMeta,
    chip: chip.isEmpty ? '#$fallbackId' : chip,
  );
}

/// 课评作者展示名：member 用公开 label，匿名/历史用本地化占位；调用方不再
/// 自行推断身份（服务端 avatarUrl omitempty 语义同理）。
String reviewAuthorLabel(ReviewPayload review, CourseCopy copy) {
  if (review.author.kind == 'member') return review.author.label;
  if (review.author.kind == 'legacy') return copy.authorLegacyLabel;
  return copy.authorAnonymousLabel;
}

/// 课评头像：member 用服务端回填的真实头像，其余（匿名 / 历史）用与 Web 同 seed
/// 的生成头像。
///
/// seed 与 Web `reviewAvatarSrc` 完全一致（`'<label>-<reviewId>'`），因此同一条
/// 评价在两端得到同一个笑脸；匿名/历史评价的 `avatarUrl` 由服务端 omitempty，
/// 客户端不做任何身份推断。
///
/// 用 [GfNetworkImage] 而不是 `GfAvatar`：`td.TAvatar` 的 errorBuilder 固定为
/// `SizedBox.shrink()`，无法回落到生成头像。
///
/// [imageBuilder] 让分享卡等截图场景换成可登记的 ShareImageNetworkImage，
/// 同时保持头像选择规则（member 网络头像 / 失败回落 / 匿名 beam seed）单点。
Widget reviewAvatar(
  ReviewPayload review, {
  double size = 40,
  Widget Function(String url, Widget fallback)? imageBuilder,
}) {
  final ReviewAuthorPayload author = review.author;
  final String rawUrl = author.kind == 'member'
      ? (author.avatarUrl?.trim() ?? '')
      : '';
  // 服务端 avatarUrl 是相对路径（如 `/static/pic/9.webp`），
  // Image.network/GfNetworkImage 需要绝对 URL；与 post_actions 等共用同一解析点。
  final String url = rawUrl.isEmpty ? '' : resolveApiAssetUrl(rawUrl);
  final Widget fallback = GfBeamAvatar(
    seed: '${author.label}-${review.id}',
    size: size,
  );
  if (url.isEmpty) return fallback;
  final Widget image = imageBuilder != null
      ? imageBuilder(url, fallback)
      : GfNetworkImage(
          url,
          width: size,
          height: size,
          fit: BoxFit.cover,
          semanticLabel: author.label,
          errorBuilder: (_, _, _) => fallback,
        );
  return SizedBox.square(
    dimension: size,
    child: ClipOval(child: image),
  );
}

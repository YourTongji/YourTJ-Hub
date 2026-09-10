import 'package:core/core.dart';
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

/// Keep server-provided business reasons localized; never expose transport internals.
String courseReviewError(AppLocalizations l10n, Object error) {
  if (error is NetworkException) return l10n.courseReviewNetworkError;
  if (error is ApiException) {
    final reason = resolveErrorMessage(l10n, error);
    if (reason != l10n.commonLoadFailed) return reason;
  }
  return l10n.courseReviewUnknownError;
}

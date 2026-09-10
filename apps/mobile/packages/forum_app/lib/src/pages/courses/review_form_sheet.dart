import 'package:flutter/material.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import 'course_common.dart';

// ---- 写评 / 编辑表单 sheet ----

class CourseReviewFormSheet extends StatefulWidget {
  const CourseReviewFormSheet({
    super.key,
    required this.pageContext,
    required this.repository,
    required this.offerings,
    this.editing,
    this.initialOfferingId,
  });

  final BuildContext pageContext;
  final CourseRepository repository;

  /// 开课实例（编辑模式不可换班，沿用原 offering）。
  final List<CourseOfferingPayload> offerings;

  /// null = 写新评价。
  final ReviewPayload? editing;

  final int? initialOfferingId;

  @override
  State<CourseReviewFormSheet> createState() => CourseReviewFormSheetState();
}

class CourseReviewFormSheetState extends State<CourseReviewFormSheet> {
  final TextEditingController _contentController = TextEditingController();
  late int? _offeringId;
  late int _rating;
  late bool _anonymous;
  bool _submitting = false;

  late final CourseRepository _repo = widget.repository;

  @override
  void initState() {
    super.initState();
    final ReviewPayload? editing = widget.editing;
    if (editing != null) {
      _offeringId = editing.offeringId;
      _rating = editing.rating ?? 0;
      _contentController.text = editing.content;
      _anonymous = editing.author.kind == 'anonymous';
    } else {
      _offeringId = widget.initialOfferingId;
      _rating = 0;
      _anonymous = true;
    }
  }

  @override
  void dispose() {
    _contentController.dispose();
    super.dispose();
  }

  void _toast(String message, {bool error = false}) {
    if (!mounted) return;
    showGfToast(widget.pageContext, message, error: error);
  }

  Future<void> _submit() async {
    final CourseCopy copy = CourseCopy(AppLocalizations.of(context));
    if (_rating < 1) {
      _toast(copy.ratingRequired, error: true);
      return;
    }
    final String content = _contentController.text.trim();
    if (content.isEmpty) {
      _toast(copy.contentRequired, error: true);
      return;
    }
    final int? offeringId = _offeringId;
    if (offeringId == null) {
      _toast(copy.operationFailed, error: true);
      return;
    }
    setState(() => _submitting = true);
    try {
      final ReviewPayload? result;
      final ReviewPayload? editing = widget.editing;
      if (editing != null) {
        result = await _repo.updateReview(
          editing.id,
          UpdateCourseReviewInput(
            rating: _rating,
            content: content,
            isAnonymous: _anonymous,
          ),
        );
      } else {
        result = await _repo.createReview(
          CreateCourseReviewInput(
            offeringId: offeringId,
            rating: _rating,
            content: content,
            isAnonymous: _anonymous,
          ),
        );
      }
      if (!mounted) return;
      Navigator.pop(context, result);
    } catch (e) {
      if (!mounted) return;
      setState(() => _submitting = false);
      if (e is! UnauthorizedException) {
        _toast(courseReviewError(AppLocalizations.of(context), e), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final bool editing = widget.editing != null;

    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            Text(
              editing ? copy.editReviewTitle : copy.writeReviewTitle,
              style: type.heading.copyWith(fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 10),
            // 开课实例选择（编辑模式只读展示）。
            Expanded(
              child: ListView(
                shrinkWrap: true,
                children: <Widget>[
                  Text(
                    copy.selectOffering,
                    style: type.caption.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.7),
                    ),
                  ),
                  const SizedBox(height: 6),
                  for (final CourseOfferingPayload offering in widget.offerings)
                    _offeringOption(offering, copy, colors, type),
                  const SizedBox(height: 12),
                  Text(
                    copy.ratingLabel,
                    style: type.caption.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.7),
                    ),
                  ),
                  const SizedBox(height: 4),
                  Row(
                    children: <Widget>[
                      for (int star = 1; star <= 5; star++)
                        Padding(
                          padding: const EdgeInsets.only(right: 4),
                          child: InkWell(
                            borderRadius: BorderRadius.circular(999),
                            onTap: _submitting
                                ? null
                                : () => setState(() => _rating = star),
                            child: Icon(
                              star <= _rating ? Icons.star : Icons.star_border,
                              size: 30,
                              color: star <= _rating
                                  ? colors.warning
                                  : colors.baseContent.withValues(alpha: 0.25),
                            ),
                          ),
                        ),
                      if (_rating > 0) ...<Widget>[
                        const SizedBox(width: 8),
                        Text(
                          '$_rating.0',
                          style: type.small.copyWith(
                            fontWeight: FontWeight.w600,
                            color: colors.baseContent.withValues(alpha: 0.7),
                          ),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 12),
                  Text(
                    copy.contentLabel,
                    style: type.caption.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.7),
                    ),
                  ),
                  const SizedBox(height: 6),
                  GfTextarea(
                    controller: _contentController,
                    hintText: copy.contentPlaceholder,
                    maxLength: 2000,
                    enabled: !_submitting,
                  ),
                  const SizedBox(height: 4),
                  Row(
                    children: <Widget>[
                      Icon(
                        _anonymous
                            ? Icons.visibility_off_outlined
                            : Icons.visibility_outlined,
                        size: 16,
                        color: colors.baseContent.withValues(alpha: 0.5),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          copy.anonymousLabel,
                          style: type.small.copyWith(
                            color: colors.baseContent.withValues(alpha: 0.7),
                          ),
                        ),
                      ),
                      Switch(
                        value: _anonymous,
                        onChanged: _submitting
                            ? null
                            : (bool value) =>
                                  setState(() => _anonymous = value),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(height: 8),
            Row(
              children: <Widget>[
                Expanded(
                  child: GfButton(
                    label: l10n.commonCancel,
                    variant: GfButtonVariant.ghost,
                    onPressed: _submitting
                        ? null
                        : () => Navigator.pop(context),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: GfButton(
                    label: editing ? l10n.commonSave : l10n.reviewSubmit,
                    loading: _submitting,
                    onPressed: _submit,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _offeringOption(
    CourseOfferingPayload offering,
    CourseCopy copy,
    GfColors colors,
    GfTypography type,
  ) {
    final String classLabel = (offering.className?.isNotEmpty ?? false)
        ? offering.className!
        : (offering.classCode ?? '');
    final String detailLine = <String>[
      offering.campus ?? '',
      offering.instructors?.join('、') ?? '',
    ].where((String s) => s.isNotEmpty).join(' · ');
    final bool selected = _offeringId == offering.id;
    return InkWell(
      onTap: _submitting || widget.editing != null
          ? null
          : () => setState(() => _offeringId = offering.id),
      child: Container(
        margin: const EdgeInsets.only(bottom: 6),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? colors.info.withValues(alpha: 0.1) : colors.base100,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(
            color: selected
                ? colors.primary.withValues(alpha: 0.4)
                : colors.line.withValues(alpha: 0.6),
          ),
        ),
        child: Row(
          children: <Widget>[
            Icon(
              selected
                  ? Icons.radio_button_checked
                  : Icons.radio_button_unchecked,
              size: 18,
              color: selected
                  ? colors.primary
                  : colors.baseContent.withValues(alpha: 0.3),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    <String>[
                      shortTerm(
                        offering.termCode,
                        locale: AppLocalizations.of(context).localeName,
                      ),
                      classLabel,
                    ].join(' · '),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: type.small.copyWith(
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                  if (detailLine.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 1),
                    Text(
                      detailLine,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: type.meta.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.55),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

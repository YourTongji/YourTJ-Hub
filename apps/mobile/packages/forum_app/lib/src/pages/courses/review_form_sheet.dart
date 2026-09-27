import 'package:flutter/material.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import 'course_common.dart';
import '../../widgets/confirm_discard_edit.dart';

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
  bool _allowPop = false, _closing = false;
  late final (int?, int, String, bool) _initial;

  bool get _dirty =>
      _initial != (_offeringId, _rating, _contentController.text, _anonymous);

  void _finish([ReviewPayload? result]) {
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) Navigator.pop(context, result);
    });
  }

  Future<void> _close() async {
    if (_submitting || _closing) return;
    _closing = true;
    final leave = !_dirty || await confirmDiscardEdit(context);
    _closing = false;
    if (mounted && leave) _finish();
  }

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
    _initial = (_offeringId, _rating, _contentController.text, _anonymous);
    _contentController.addListener(() => setState(() {}));
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
    if (_submitting) return;
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
      _finish(result);
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

    return PopScope(
      canPop: _allowPop || (!_submitting && !_dirty),
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) => SingleChildScrollView(
            child: SizedBox(
              // Keep a usable editing area when keyboard and large text leave
              // too little room for the fixed heading/actions. The outer scroll
              // makes every control reachable without reparenting the editor.
              height:
                  constraints.maxHeight <
                      MediaQuery.textScalerOf(context).scale(280)
                  ? MediaQuery.textScalerOf(context).scale(280)
                  : constraints.maxHeight,
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
                          for (final CourseOfferingPayload offering
                              in widget.offerings)
                            _offeringOption(offering, copy, colors, type),
                          const SizedBox(height: 12),
                          Text(
                            copy.ratingLabel,
                            style: type.caption.copyWith(
                              color: colors.baseContent.withValues(alpha: 0.7),
                            ),
                          ),
                          const SizedBox(height: 4),
                          Wrap(
                            crossAxisAlignment: WrapCrossAlignment.center,
                            children: <Widget>[
                              for (int star = 1; star <= 5; star++)
                                Semantics(
                                  key: ValueKey('review-rating-$star'),
                                  label: '${copy.ratingLabel}: $star / 5',
                                  button: true,
                                  selected: _rating == star,
                                  enabled: !_submitting,
                                  onTap: _submitting
                                      ? null
                                      : () => setState(() => _rating = star),
                                  child: InkWell(
                                    excludeFromSemantics: true,
                                    borderRadius: BorderRadius.circular(24),
                                    onTap: _submitting
                                        ? null
                                        : () => setState(() => _rating = star),
                                    child: SizedBox(
                                      width: 48,
                                      height: 48,
                                      child: Center(
                                        child: GfSymbol(
                                          star <= _rating
                                              ? 'star-filled'
                                              : 'star',
                                          size: 28,
                                          color: star <= _rating
                                              ? colors.warning
                                              : colors.iconMuted,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              if (_rating > 0)
                                Padding(
                                  padding: const EdgeInsets.only(left: 8),
                                  child: Text(
                                    '$_rating.0',
                                    style: type.small.copyWith(
                                      color: colors.iconMuted,
                                    ),
                                  ),
                                ),
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
                              GfSymbol(
                                _anonymous ? 'eye-off' : 'eye',
                                size: 16,
                                color: colors.baseContent.withValues(
                                  alpha: 0.5,
                                ),
                              ),
                              const SizedBox(width: 8),
                              Expanded(
                                child: Text(
                                  copy.anonymousLabel,
                                  style: type.small.copyWith(
                                    color: colors.baseContent.withValues(
                                      alpha: 0.7,
                                    ),
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
                            onPressed: _submitting ? null : _close,
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: GfButton(
                            label: editing
                                ? l10n.commonSave
                                : l10n.reviewSubmit,
                            loading: _submitting,
                            onPressed: _submit,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ),
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
    final classParts = <String>[
      offering.className?.trim() ?? '',
      offering.classCode?.trim() ?? '',
    ].where((part) => part.isNotEmpty).toSet();
    final String classLabel = classParts.join(' · ');
    final String detailLine = <String>[
      offering.campus?.trim() ?? '',
      offering.faculty?.trim() ?? '',
      ...?offering.instructors?.map((name) => name.trim()),
    ].where((part) => part.isNotEmpty).toSet().join(' · ');
    final bool selected = _offeringId == offering.id;
    return InkWell(
      onTap: _submitting || widget.editing != null
          ? null
          : () => setState(() => _offeringId = offering.id),
      child: Container(
        margin: const EdgeInsets.only(bottom: 6),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        decoration: BoxDecoration(
          color: colors.base200,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: selected
                ? colors.primary.withValues(alpha: 0.4)
                : colors.line.withValues(alpha: 0.6),
          ),
        ),
        child: Row(
          children: <Widget>[
            GfSymbol(
              selected ? 'circle-dot' : 'circle',
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
                      (offering.termName?.trim().isNotEmpty ?? false)
                          ? offering.termName!.trim()
                          : shortTerm(
                              offering.termCode,
                              locale: AppLocalizations.of(context).localeName,
                            ),
                      classLabel,
                    ].join(' · '),
                    style: type.small.copyWith(
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                  if (detailLine.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 1),
                    Text(
                      detailLine,
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

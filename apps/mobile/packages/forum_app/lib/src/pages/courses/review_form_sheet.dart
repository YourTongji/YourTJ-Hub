import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import 'course_common.dart';
import '../../widgets/confirm_discard_edit.dart';
import '../../widgets/editor/course_review_templates.dart';
import '../../widgets/editor/rich_markdown_editor.dart';

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
  final FocusNode _editorFocusNode = FocusNode(
    debugLabel: 'course-review-body',
  );
  final MarkdownConverter _converter = MarkdownConverter();
  late QuillController _editorController;
  late StreamSubscription<DocChange> _editorChanges;
  String _content = '';
  late int? _offeringId;
  late int _rating;
  late bool _anonymous;
  bool _submitting = false;
  bool _allowPop = false, _closing = false;
  late final (int?, int, String, bool) _initial;

  bool get _dirty => _initial != (_offeringId, _rating, _content, _anonymous);

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
      _content = editing.content;
      _anonymous = editing.author.kind == 'anonymous';
    } else {
      _offeringId = widget.initialOfferingId;
      _rating = 0;
      _anonymous = true;
    }
    _initial = (_offeringId, _rating, _content, _anonymous);
    _editorController = _createEditor(_content);
  }

  QuillController _createEditor(String markdown) {
    final controller = QuillController(
      document: _converter.mdToDocument(markdown),
      selection: const TextSelection.collapsed(offset: 0),
    );
    _editorChanges = controller.document.changes.listen(
      (_) => _editorChanged(),
    );
    return controller;
  }

  void _editorChanged() {
    final String next = _converter
        .documentToMarkdown(_editorController.document)
        .trim();
    if (next == _content || !mounted) return;
    setState(() => _content = next);
  }

  @override
  void dispose() {
    unawaited(_editorChanges.cancel());
    _editorController.dispose();
    _editorFocusNode.dispose();
    super.dispose();
  }

  void _toast(String message, {bool error = false}) {
    if (!mounted) return;
    showGfToast(widget.pageContext, message, error: error);
  }

  Future<void> _submit() async {
    if (_submitting) return;
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    if (_rating < 1) {
      _toast(copy.ratingRequired, error: true);
      return;
    }
    final String content = _content.trim();
    final int contentLength = content.runes.length;
    if (content.isEmpty) {
      _toast(copy.contentRequired, error: true);
      return;
    }
    if (contentLength > 2000) {
      _toast(l10n.courseReviewContentLimitError, error: true);
      return;
    }
    final int? offeringId = _offeringId;
    if (offeringId == null) {
      _toast(copy.operationFailed, error: true);
      return;
    }
    setState(() {
      _submitting = true;
      _editorController.readOnly = true;
    });
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
      setState(() {
        _submitting = false;
        _editorController.readOnly = false;
      });
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
                          Row(
                            children: <Widget>[
                              Expanded(
                                child: Text(
                                  copy.contentLabel,
                                  style: type.caption.copyWith(
                                    color: colors.baseContent.withValues(
                                      alpha: 0.7,
                                    ),
                                  ),
                                ),
                              ),
                              Text(
                                '${_content.runes.length}/2000',
                                style: type.meta.copyWith(
                                  color: _content.runes.length > 2000
                                      ? colors.error
                                      : colors.iconMuted,
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 6),
                          Align(
                            alignment: Alignment.centerLeft,
                            child: TextButton.icon(
                              key: const Key('course-review-templates'),
                              onPressed: _submitting ? null : _chooseTemplate,
                              icon: const GfSymbol('file-text', size: 18),
                              label: Text(l10n.courseReviewTemplates),
                            ),
                          ),
                          RichMarkdownEditor(
                            controller: _editorController,
                            focusNode: _editorFocusNode,
                            placeholder: copy.contentPlaceholder,
                            enabled: !_submitting,
                            onHeading: () => _chooseHeading(l10n),
                            onInsertLink: _insertLink,
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

  Future<void> _chooseTemplate() async {
    final l10n = AppLocalizations.of(context);
    final selected = await showGfBottomSheet<CourseReviewTemplate>(
      context,
      keyboardAware: true,
      builder: (sheetContext) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: <Widget>[
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
              child: Text(
                l10n.courseReviewTemplates,
                style: GfTheme.typographyOf(context).heading,
              ),
            ),
            for (final template in courseReviewTemplates)
              ListTile(
                leading: GfSymbol(_templateIcon(template.id), size: 20),
                title: Text(_templateName(l10n, template.id)),
                subtitle: Text(_templateDescription(l10n, template.id)),
                onTap: () => Navigator.pop(sheetContext, template),
              ),
          ],
        ),
      ),
    );
    if (selected == null || !mounted) return;
    if (_content.trim().isNotEmpty) {
      final replace = await showDialog<bool>(
        context: context,
        builder: (dialogContext) => AlertDialog(
          title: Text(l10n.courseReviewTemplateReplaceTitle),
          content: Text(l10n.courseReviewTemplateReplaceBody),
          actions: <Widget>[
            TextButton(
              onPressed: () => Navigator.pop(dialogContext, false),
              child: Text(l10n.courseReviewTemplateKeepEditing),
            ),
            TextButton(
              onPressed: () => Navigator.pop(dialogContext, true),
              child: Text(l10n.courseReviewTemplateApply),
            ),
          ],
        ),
      );
      if (replace != true || !mounted) return;
    }
    final cancellation = _editorChanges.cancel();
    if (!mounted) return;
    final previous = _editorController;
    setState(() {
      _content = selected.content.trim();
      _editorController = _createEditor(selected.content);
    });
    await cancellation;
    WidgetsBinding.instance.addPostFrameCallback((_) => previous.dispose());
  }

  String _templateIcon(String id) => switch (id) {
    'comprehensive' => 'book-open',
    'quick' => 'star',
    'teacher-focused' => 'graduation-cap',
    'exam-focused' => 'shield-check',
    'workload' => 'clock',
    _ => 'file-text',
  };

  String _templateName(AppLocalizations l10n, String id) => switch (id) {
    'comprehensive' => l10n.courseReviewTemplateComprehensiveName,
    'quick' => l10n.courseReviewTemplateQuickName,
    'teacher-focused' => l10n.courseReviewTemplateTeacherFocusedName,
    'exam-focused' => l10n.courseReviewTemplateExamFocusedName,
    'workload' => l10n.courseReviewTemplateWorkloadName,
    _ => l10n.courseReviewTemplateBlankName,
  };

  String _templateDescription(AppLocalizations l10n, String id) => switch (id) {
    'comprehensive' => l10n.courseReviewTemplateComprehensiveDescription,
    'quick' => l10n.courseReviewTemplateQuickDescription,
    'teacher-focused' => l10n.courseReviewTemplateTeacherFocusedDescription,
    'exam-focused' => l10n.courseReviewTemplateExamFocusedDescription,
    'workload' => l10n.courseReviewTemplateWorkloadDescription,
    _ => l10n.courseReviewTemplateBlankDescription,
  };

  Future<void> _chooseHeading(AppLocalizations l10n) async {
    final level =
        _editorController
                .getSelectionStyle()
                .attributes[Attribute.header.key]
                ?.value
            as int?;
    final selected = await showGfBottomSheet<Attribute>(
      context,
      keyboardAware: true,
      builder: (sheetContext) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: <Widget>[
            for (final (attribute, label, value) in <(Attribute, String, int)>[
              (Attribute.h1, l10n.publishHeadingLevel1, 1),
              (Attribute.h2, l10n.publishHeadingLevel2, 2),
              (Attribute.h3, l10n.publishHeadingLevel3, 3),
            ])
              ListTile(
                title: Text(label),
                trailing: level == value
                    ? const GfSymbol('check', size: 20)
                    : null,
                onTap: () => Navigator.pop(sheetContext, attribute),
              ),
          ],
        ),
      ),
    );
    if (selected != null && mounted) {
      _editorController.formatSelection(selected);
    }
  }

  Future<void> _insertLink() async {
    final input = TextEditingController();
    final l10n = AppLocalizations.of(context);
    final url = await showDialog<String>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.publishToolLink),
        content: TextField(
          controller: input,
          autofocus: true,
          keyboardType: TextInputType.url,
          decoration: InputDecoration(hintText: 'https://'),
        ),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, input.text.trim()),
            child: Text(l10n.commonSave),
          ),
        ],
      ),
    );
    input.dispose();
    if (url == null || !mounted) return;
    final uri = Uri.tryParse(url);
    if (uri == null ||
        !['https', 'http', 'mailto'].contains(uri.scheme) ||
        (uri.scheme != 'mailto' && uri.host.isEmpty)) {
      showGfToast(context, l10n.publishLinkInvalid, error: true);
      return;
    }
    final selection = _editorController.selection;
    if (selection.isCollapsed || !selection.isValid) {
      final start = selection.isValid ? selection.start : 0;
      _editorController.replaceText(
        start,
        0,
        url,
        TextSelection(baseOffset: start, extentOffset: start + url.length),
      );
    }
    _editorController.formatSelection(LinkAttribute(url));
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

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../current_user.dart';
import '../../images/image_upload.dart';
import '../../providers.dart';
import 'course_common.dart';
import '../../widgets/confirm_discard_edit.dart';
import '../../widgets/editor/course_review_templates.dart';
import '../../widgets/editor/rich_markdown_editor.dart';

// ---- 写评 / 编辑表单 sheet ----

/// 写评身份（昵称 + 头像），供 sheet 展示「以谁的身份发布」。
typedef ReviewPublisher = ({String name, String avatarUrl});

/// 当前发布身份：用户卡接口一次拉取；离线、未登录或失败时返回 null，
/// sheet 降级为「公开身份」占位，绝不伪造昵称。
final reviewPublisherProvider = FutureProvider.autoDispose<ReviewPublisher?>((
  ref,
) async {
  try {
    final CurrentUser? user = await ref.watch(currentUserProvider.future);
    if (user == null) return null;
    final UserCardPayload card = await ref
        .watch(userRepositoryProvider)
        .getUserCard(user.id);
    final String nickname = card.nickname.trim();
    return (
      name: nickname.isNotEmpty ? nickname : card.username,
      avatarUrl: card.avatarUrl,
    );
  } catch (_) {
    return null;
  }
});

class CourseReviewFormSheet extends ConsumerStatefulWidget {
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
  ConsumerState<CourseReviewFormSheet> createState() =>
      CourseReviewFormSheetState();
}

class CourseReviewFormSheetState extends ConsumerState<CourseReviewFormSheet> {
  /// 正文区域至少要有这么高才放得下常驻工具栏 + 一行正文（工具栏 48 + 正文 48）。
  static const double _editorRegionWithToolbar = 96;

  /// 元信息完整展示需要的最小面板高度：标题 48 + 班次 40 + 身份 48 + 星级 48
  /// + 间距 14 + 动作 56 + 正文下限 96 ≈ 350；低于它退回单行横向滚动，
  /// 否则正文会被挤到负空间（320×568 + 200% 字号 + 键盘只剩 268px）。
  static const double _metaFullMinHeight = 360;

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
  bool _uploadingImage = false;
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

  Future<void> _insertImage() async {
    if (_submitting || _uploadingImage) return;
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _uploadingImage = true);
    try {
      final String? url = await pickAndUploadImage(ref: ref);
      if (!mounted ||
          epoch != ref.read(offlineCacheEpochProvider) ||
          url == null) {
        return;
      }
      // 上传完成后按当前光标插入：选图期间用户若继续编辑，图片落在最新
      // 光标处。Document.insert 返回 Delta，光标直接取插入点后一位即可。
      final TextSelection selection = _editorController.selection;
      final int caret = selection.isValid
          ? selection.end
          : _editorController.document.length - 1;
      final int at = caret.clamp(0, _editorController.document.length - 1);
      _editorController.document.insert(at, BlockEmbed.image(url));
      _editorController.updateSelection(
        TextSelection.collapsed(offset: at + 1),
        ChangeSource.local,
      );
      _editorFocusNode.requestFocus();
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        _toast(
          courseReviewError(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _uploadingImage = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    // 编辑器优先布局：标题 / 紧凑元信息（班次+匿名、星级）/ 常驻工具栏 +
    // 正文内部滚动 / 常驻动作。
    // 面板高度由 showGfBottomSheet 收敛到键盘之上，正文在 Expanded 里滚动，
    // 因此工具栏和底部动作不会随行数漂移。
    return PopScope(
      canPop: _allowPop || (!_submitting && !_dirty),
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
          child: LayoutBuilder(
            builder: (context, sheetConstraints) => Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                _titleRow(l10n, copy, type),
                const SizedBox(height: 8),
                // 面板够高时元信息两行全可见；被键盘 + 大字压到 300px 以下
                // 时退回单行横向滚动（必填星级与匿名开关排在最前）。
                _metaRow(
                  copy,
                  colors,
                  type,
                  full: sheetConstraints.maxHeight >= _metaFullMinHeight,
                ),
                const SizedBox(height: 6),
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, constraints) => RichMarkdownEditor(
                      controller: _editorController,
                      focusNode: _editorFocusNode,
                      placeholder: copy.contentPlaceholder,
                      enabled: !_submitting,
                      // 编辑器自身的块级样式走阅读态同一档位。
                      fill: true,
                      profile: GfRichContentTypography.of(
                        context,
                        compact: true,
                      ),
                      // 200% 字号 + 键盘在 320px 宽屏上只留几十像素时，工具栏
                      // 会和正文抢同一点高度并溢出；此时收起工具栏，让正文与
                      // 固定动作行都可用（正常字号下工具栏常驻）。
                      showToolbar:
                          constraints.maxHeight >= _editorRegionWithToolbar,
                      onInsertImage: _insertImage,
                      insertingImage: _uploadingImage,
                      onHeading: () => _chooseHeading(l10n),
                      onInsertLink: _insertLink,
                    ),
                  ),
                ),
                const SizedBox(height: 6),
                _actionRow(l10n, colors),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// 标题行：标题 + 模板入口 +（键盘弹起时）收起键盘。
  Widget _titleRow(AppLocalizations l10n, CourseCopy copy, GfTypography type) {
    final bool editing = widget.editing != null;
    return Row(
      children: <Widget>[
        Expanded(
          child: Text(
            editing ? copy.editReviewTitle : copy.writeReviewTitle,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: type.heading.copyWith(fontWeight: FontWeight.w700),
          ),
        ),
        ConstrainedBox(
          // 德/日文案在 200% 字号下比整行还宽；限定最多半行并单行省略，
          // 标题行永远不会横向溢出。
          constraints: BoxConstraints(
            maxWidth: MediaQuery.sizeOf(context).width * 0.45,
          ),
          child: TextButton.icon(
            key: const Key('course-review-templates'),
            onPressed: _submitting ? null : _chooseTemplate,
            icon: const GfSymbol('file-text', size: 18),
            label: Text(
              l10n.courseReviewTemplates,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ),
        if (MediaQuery.viewInsetsOf(context).bottom > 0)
          GfIconButton(
            symbol: 'keyboard-hide',
            tooltip: l10n.commonHideKeyboard,
            size: 44,
            onPressed: () => FocusManager.instance.primaryFocus?.unfocus(),
          ),
      ],
    );
  }

  /// 元信息分层（better-layout：组内 6–8dp、跨组 ≥2×）：
  /// 班次 chip（可选）→ 发布身份行（头像 + 文案 + 匿名开关）→ 星级。
  /// 面板够高（≥360px）时全部直接可见；被键盘 + 大字压到 360px 以下时
  /// 退回单行横向滚动（必填星级优先），身份行只保留主文案避免正文被挤成负空间。
  /// 编辑态且没有班次数据（如「我的课评」入口）时不渲染只读班次 chip，
  /// 避免显示一个不可点击的「选择班次」占位。
  bool get _showOfferingChip =>
      widget.editing == null || _offeringById(_offeringId) != null;

  Widget _metaRow(
    CourseCopy copy,
    GfColors colors,
    GfTypography type, {
    required bool full,
  }) {
    final Widget identity = _identityRow(copy, colors, type, compact: !full);
    if (!full) {
      return SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            // 星级本身就是数值的可视表达，不再重复一行「4.0」数值：
            // 星级行（240dp）与班次 chip 共行时才有足够宽度（390dp/100%）。
            ..._starTargets(copy, colors, type),
            const SizedBox(width: 12),
            identity,
            if (_showOfferingChip) ...<Widget>[
              const SizedBox(width: 12),
              _offeringChip(copy, colors, type),
            ],
          ],
        ),
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        // 班次 chip 与星级共行：chip 贴左、星级贴右（better-layout：同一行内
        // 的控件对齐到共享边）。不用 Wrap：Wrap 会在「chip 恰好多出几像素」时
        // 静默折行，出现半空的第二行；这里显式分行——常规宽度（390dp，内容
        // 358dp）永远同一行，chip 文本按需省略号收窄；只有窄屏（内容 < 336dp：
        // 240dp 星级 + 可读 chip 放不下）或大字号才降级为上下两行。
        LayoutBuilder(
          builder: (context, constraints) {
            final Widget stars = Row(
              mainAxisSize: MainAxisSize.min,
              children: _starTargets(copy, colors, type),
            );
            final bool stack =
                constraints.maxWidth < 336 ||
                MediaQuery.textScalerOf(context).scale(1) > 1.3;
            if (stack) {
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  if (_showOfferingChip) ...<Widget>[
                    Align(
                      alignment: Alignment.centerLeft,
                      child: _offeringChip(copy, colors, type),
                    ),
                    const SizedBox(height: 8),
                  ],
                  Align(alignment: Alignment.centerRight, child: stars),
                ],
              );
            }
            return Row(
              // chip 缺席（编辑态无班次数据）时星级仍贴右：与有 chip 时
              // 的共享边一致，也与下方身份行的左对齐形成对照。
              mainAxisAlignment: _showOfferingChip
                  ? MainAxisAlignment.spaceBetween
                  : MainAxisAlignment.end,
              children: <Widget>[
                if (_showOfferingChip)
                  Flexible(child: _offeringChip(copy, colors, type)),
                stars,
              ],
            );
          },
        ),
        const SizedBox(height: 8),
        identity,
      ],
    );
  }

  /// 五颗星的 48px 命名命中区。
  List<Widget> _starTargets(
    CourseCopy copy,
    GfColors colors,
    GfTypography type,
  ) {
    return <Widget>[
      for (int star = 1; star <= 5; star++)
        Semantics(
          key: ValueKey<String>('review-rating-$star'),
          label: '${copy.ratingLabel}: $star / 5',
          button: true,
          selected: _rating == star,
          enabled: !_submitting,
          onTap: _submitting ? null : () => setState(() => _rating = star),
          child: InkWell(
            excludeFromSemantics: true,
            borderRadius: BorderRadius.circular(24),
            onTap: _submitting ? null : () => setState(() => _rating = star),
            child: SizedBox(
              width: 48,
              height: 48,
              child: Center(
                child: GfSymbol(
                  star <= _rating ? 'star-filled' : 'star',
                  size: 28,
                  color: star <= _rating ? colors.warning : colors.iconMuted,
                ),
              ),
            ),
          ),
        ),
    ];
  }

  /// 发布身份行：头像 + 「匿名发布 / 以 xx 发布」主文案（匿名时附次级说明）
  /// + 匿名开关。文案用 Expanded 换行而不是省略号截断；紧凑模式只保留主文案
  /// 并让整行横向滚动，避免把正文挤到负空间。
  Widget _identityRow(
    CourseCopy copy,
    GfColors colors,
    GfTypography type, {
    required bool compact,
  }) {
    final ReviewPublisher? publisher = ref
        .watch(reviewPublisherProvider)
        .valueOrNull;
    final String name = publisher?.name ?? '';
    final String primary = _anonymous
        ? copy.anonymousTitle
        : (name.isEmpty ? copy.publishPublic : copy.publishAs(name));
    final Widget avatar = _publisherAvatar(
      anonymous: _anonymous,
      url: _anonymous ? '' : (publisher?.avatarUrl ?? ''),
      colors: colors,
    );
    final Widget toggle = Switch(
      value: _anonymous,
      onChanged: _submitting
          ? null
          : (bool value) => setState(() => _anonymous = value),
    );
    if (compact) {
      return Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          avatar,
          const SizedBox(width: 8),
          Text(
            primary,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: type.small.copyWith(
              color: colors.baseContent.withValues(alpha: 0.8),
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(width: 8),
          toggle,
        ],
      );
    }
    return Row(
      children: <Widget>[
        avatar,
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Text(
                primary,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: type.small.copyWith(
                  color: colors.baseContent,
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (_anonymous) ...<Widget>[
                const SizedBox(height: 2),
                Text(
                  copy.anonymousHint,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: type.meta.copyWith(
                    color: colors.baseContent.withValues(alpha: 0.55),
                  ),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(width: 8),
        toggle,
      ],
    );
  }

  /// 发布者头像：有服务端头像时解析相对路径加载，失败或匿名时回落身份占位。
  Widget _publisherAvatar({
    required bool anonymous,
    required String url,
    required GfColors colors,
    double size = 32,
  }) {
    final Widget fallback = DecoratedBox(
      decoration: BoxDecoration(color: colors.base200, shape: BoxShape.circle),
      child: SizedBox.square(
        dimension: size,
        child: Center(
          child: GfSymbol(
            anonymous ? 'eye-off' : 'user-round',
            size: size * 0.55,
            color: colors.iconMuted,
          ),
        ),
      ),
    );
    final String resolved = url.trim().isEmpty
        ? ''
        : resolveApiAssetUrl(url.trim());
    if (anonymous || resolved.isEmpty) return fallback;
    return ClipOval(
      child: GfNetworkImage(
        resolved,
        width: size,
        height: size,
        fit: BoxFit.cover,
        errorBuilder: (_, _, _) => fallback,
      ),
    );
  }

  /// 班次选择 chip：写评态点击打开既有班次列表（复用 `_offeringOption`），
  /// 编辑态只读展示（服务端不允许换班）。
  Widget _offeringChip(CourseCopy copy, GfColors colors, GfTypography type) {
    final bool readOnly = widget.editing != null;
    final CourseOfferingPayload? offering = _offeringById(_offeringId);
    final String label = offering == null
        ? copy.selectOffering
        : _offeringSummary(offering);
    return InkWell(
      key: const Key('course-review-offering'),
      borderRadius: BorderRadius.circular(16),
      onTap: readOnly || _submitting ? null : _pickOffering,
      child: Container(
        // 紧凑内边距：与星级共行时 chip 必须窄到单行放得下（390dp/100%）。
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        decoration: BoxDecoration(
          color: colors.base200,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: colors.line.withValues(alpha: 0.6)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            Flexible(
              child: ConstrainedBox(
                // 单独一行时允许用满屏宽；横向滚动分支也能拿到有限上限。
                constraints: BoxConstraints(
                  maxWidth: MediaQuery.sizeOf(context).width,
                ),
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: type.small.copyWith(
                    fontWeight: FontWeight.w600,
                    color: colors.baseContent,
                  ),
                ),
              ),
            ),
            if (!readOnly) ...<Widget>[
              const SizedBox(width: 4),
              GfSymbol('chevron-down', size: 16, color: colors.iconMuted),
            ],
          ],
        ),
      ),
    );
  }

  /// 常驻动作行：字数计数 + 取消 + 发布/保存。
  Widget _actionRow(AppLocalizations l10n, GfColors colors) {
    final int length = _content.runes.length;
    final bool editing = widget.editing != null;
    // 计数只在常规字号下与按钮同排：≥150% 字号时德/日按钮
    //（Abbrechen / Bewertung senden）单行就要 300px 以上，再挤进计数器只会
    // 把按钮压成 4-5 行、把面板顶出屏幕。此时计数让位（长度上限仍在提交时
    // 校验），只保留两个动作。
    final bool showCounter = MediaQuery.textScalerOf(context).scale(1) < 1.5;
    // 按钮保持单行（不换行、不横向溢出），整行按需横向滚动：动作行高度稳定
    // 在 48/56px，键盘弹出后取消/提交一定落在键盘上方。常规字号下宽度足够，
    // 计数贴左、取消/提交贴右，与设计稿同排。
    return LayoutBuilder(
      builder: (context, constraints) => SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: ConstrainedBox(
          constraints: BoxConstraints(minWidth: constraints.maxWidth),
          child: Row(
            mainAxisAlignment: showCounter
                ? MainAxisAlignment.spaceBetween
                : MainAxisAlignment.end,
            children: <Widget>[
              if (showCounter)
                Text(
                  '$length/2000',
                  maxLines: 1,
                  style: GfTheme.typographyOf(context).meta.copyWith(
                    color: length > 2000 ? colors.error : colors.iconMuted,
                  ),
                ),
              Row(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  GfButton(
                    label: l10n.commonCancel,
                    variant: GfButtonVariant.ghost,
                    onPressed: _submitting ? null : _close,
                  ),
                  const SizedBox(width: 8),
                  GfButton(
                    label: editing ? l10n.commonSave : l10n.reviewSubmit,
                    loading: _submitting,
                    onPressed: _submit,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  CourseOfferingPayload? _offeringById(int? id) {
    if (id == null) return null;
    for (final CourseOfferingPayload offering in widget.offerings) {
      if (offering.id == id) return offering;
    }
    return null;
  }

  /// 班次 chip 的单行摘要：短学期码 + 班级名（完整学期名与班级代码留在
  /// 班次列表里）。紧凑形式让单行元信息尽量少横向滚动。
  String _offeringSummary(CourseOfferingPayload offering) {
    final String term = shortTerm(
      offering.termCode,
      locale: AppLocalizations.of(context).localeName,
    );
    final String classLabel =
        <String>[
          offering.className?.trim() ?? '',
          offering.classCode?.trim() ?? '',
        ].where((String part) => part.isNotEmpty).firstOrNull ??
        '';
    return <String>[
      term,
      classLabel,
    ].where((String part) => part.isNotEmpty).join(' · ');
  }

  /// 打开班次列表（复用既有 `_offeringOption` 渲染）。
  Future<void> _pickOffering() async {
    if (widget.editing != null) return;
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final CourseOfferingPayload? selected =
        await showGfBottomSheet<CourseOfferingPayload>(
          context,
          keyboardAware: true,
          builder: (sheetContext) => SafeArea(
            child: ListView(
              shrinkWrap: true,
              children: <Widget>[
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    copy.selectOffering,
                    style: GfTheme.typographyOf(context).heading,
                  ),
                ),
                for (final CourseOfferingPayload offering in widget.offerings)
                  _offeringOption(
                    offering,
                    copy,
                    colors,
                    type,
                    onTap: () => Navigator.pop(sheetContext, offering),
                  ),
              ],
            ),
          ),
        );
    if (selected == null || !mounted) return;
    setState(() => _offeringId = selected.id);
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
    // 原地替换：controller 实例、撤销栈、焦点与正文滚动位置都保持不变。
    // 传 Delta 而不是 Document：flutter_quill 11.5.1 的
    // `replaceText`/`Document.replace` 断言只接受 String/Embeddable/Delta。
    // 替换整篇（`document.length`）与旧的 `mdToDocument(content)` 结果完全一致；
    // 用 `length - 1` 会留下一个多余的空段落。`_content` 由 `_editorChanged`
    // 监听文档变化回填，不再手动赋值。
    final Document document = _converter.mdToDocument(selected.content);
    _editorController.replaceText(
      0,
      _editorController.document.length,
      document.toDelta(),
      TextSelection.collapsed(offset: _templateCaretOffset(document)),
    );
    _editorFocusNode.requestFocus();
  }

  /// 模板插入后的光标位置。
  ///
  /// 模板首行是标题时落在标题之后的第一个可填写行；转换器会把 Markdown 的空行
  /// 收进块级间距（`MarkdownToDelta` 不为空行产出独立段落，publish 编辑器同理），
  /// 所以「模板全是标题」时退化为首行标题末尾——用户从这里回车即可开始写正文。
  /// 首行不是标题时落在文档开头。
  int _templateCaretOffset(Document document) {
    final List<Line> lines = <Line>[
      for (final Node node in document.root.children) node as Line,
    ];
    if (lines.isEmpty || !lines.first.style.containsKey(Attribute.header.key)) {
      return 0;
    }
    int offset = 0;
    for (final Line line in lines) {
      if (!line.style.containsKey(Attribute.header.key)) return offset;
      offset += line.length;
    }
    return document.length - 1;
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
    GfTypography type, {
    required VoidCallback onTap,
  }) {
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
      onTap: onTap,
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

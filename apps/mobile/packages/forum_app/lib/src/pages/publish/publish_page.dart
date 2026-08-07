import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';

/// 发布/编辑话题页(web PublishPage.vue 的移动端形态)。
///
/// 富文本编辑器用 flutter_quill,保存时经 core 的 MarkdownConverter
/// 转为 markdown 提交(与 web 端 ProseMirror ↔ markdown 契约一致)。
/// 图片:image_picker 选图 → POST /file/img-upload 上传 → 插入 image embed。
/// 分类:从 /publish 页面数据通道加载真实分类列表。
class PublishPage extends ConsumerStatefulWidget {
  const PublishPage({
    super.key,
    this.topicId,
    this.editTitle,
    this.editContent,
    this.editCategoryIds,
  });

  /// 编辑模式时的话题 id;发布模式为 null(新建)。
  final int? topicId;

  final String? editTitle;
  final String? editContent;
  final List<int>? editCategoryIds;

  @override
  ConsumerState<PublishPage> createState() => _PublishPageState();
}

class _PublishPageState extends ConsumerState<PublishPage> {
  final TextEditingController _title = TextEditingController();
  final List<int> _categoryIds = [];
  final MarkdownConverter _converter = MarkdownConverter();
  late final QuillController _quill;
  bool _submitting = false;
  bool _uploading = false;
  String _error = '';
  String _message = '';

  final ImagePicker _imagePicker = ImagePicker();

  final List<PublishCategoryPayload> _categories = [];

  @override
  void initState() {
    super.initState();
    _quill = QuillController(
      document: _converter.mdToDocument(widget.editContent ?? ''),
      selection: const TextSelection.collapsed(offset: 0),
    );
    if (widget.editTitle != null) _title.text = widget.editTitle!;
    if (widget.editCategoryIds != null) {
      _categoryIds.addAll(widget.editCategoryIds!);
    }
    _loadCategories();
  }

  /// 从 /publish 页面数据通道加载真实分类列表(web PublishPageProps.categories)。
  Future<void> _loadCategories() async {
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/publish');
      final PublishPageProps? props = parsePageProps<PublishPageProps>(payload);
      if (props != null && mounted) {
        setState(() => _categories.addAll(props.categories));
      }
    } catch (_) {
      // 分类加载失败静默(编辑模式可用传入的分类)。
    }
  }

  @override
  void dispose() {
    _title.dispose();
    _quill.dispose();
    super.dispose();
  }

  /// 选择图片上传并插入编辑器。
  Future<void> _pickAndInsertImage() async {
    if (_uploading) return;
    final XFile? picked = await _imagePicker.pickImage(
      source: ImageSource.gallery,
      maxWidth: 2048,
      imageQuality: 85,
    );
    if (picked == null) return;

    setState(() => _uploading = true);
    try {
      final bytes = await picked.readAsBytes();
      final String url = await ref
          .read(fileRepositoryProvider)
          .uploadImage(bytes: bytes, filename: picked.name);
      if (!mounted) return;
      final int index = _quill.selection.baseOffset;
      _quill.document.insert(index < 0 ? 0 : index, BlockEmbed.image(url));
    } catch (e) {
      if (mounted) {
        final l10n = AppLocalizations.of(context);
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.publishImageFailed('$e'))));
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _submit({required int topicStatus}) async {
    final String title = _title.text.trim();
    final String content = _converter
        .documentToMarkdown(_quill.document)
        .trim();
    final l10n = AppLocalizations.of(context);

    if (title.isEmpty) {
      setState(() => _error = l10n.publishTitleRequired);
      return;
    }
    if (content.isEmpty) {
      setState(() => _error = l10n.publishContentRequired);
      return;
    }

    setState(() {
      _submitting = true;
      _error = '';
      _message = '';
    });
    try {
      final int id = await ref
          .read(topicRepositoryProvider)
          .writeTopic(
            topicId: widget.topicId ?? 0,
            title: title,
            content: content,
            // 未选择分类时由后端按默认规则处理(不再硬编码 [1])。
            categoryIds: _categoryIds,
            topicStatus: topicStatus,
          );
      if (!mounted) return;
      setState(
        () => _message = topicStatus == 1
            ? l10n.publishSuccess
            : l10n.publishSavedDraft,
      );
      // 跳转到话题详情。
      if (topicStatus == 1 && id > 0) {
        Navigator.of(context).pop();
      }
    } on ApiException catch (e) {
      setState(() => _error = e.messageKey);
    } catch (e) {
      setState(() => _error = l10n.publishFailed('$e'));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.topicId == null ? l10n.publishTitle : l10n.publishEditTitle,
        ),
        actions: [
          GfButton(
            label: l10n.publishPublish,
            variant: GfButtonVariant.primary,
            size: GfButtonSize.small,
            loading: _submitting,
            onPressed: _submitting ? null : () => _submit(topicStatus: 1),
          ),
          const SizedBox(width: 12),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            controller: _title,
            maxLength: 100,
            decoration: InputDecoration(
              labelText: l10n.publishTitleField,
              hintText: l10n.publishTitleHint,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          // 分类选择(数据来自 /publish 数据通道)。
          if (_categories.isNotEmpty) ...[
            Wrap(
              spacing: 8,
              children: [
                for (final cat in _categories)
                  FilterChip(
                    label: Text(cat.name),
                    selected: _categoryIds.contains(cat.id),
                    onSelected: (sel) {
                      setState(() {
                        if (sel) {
                          _categoryIds.add(cat.id);
                        } else {
                          _categoryIds.remove(cat.id);
                        }
                      });
                    },
                  ),
              ],
            ),
            const SizedBox(height: 12),
          ],
          // 富文本编辑器工具栏(精简版)。
          _buildToolbar(),
          Container(
            constraints: const BoxConstraints(minHeight: 200),
            decoration: BoxDecoration(
              border: Border.all(color: colors.line),
              borderRadius: BorderRadius.circular(8),
            ),
            padding: const EdgeInsets.all(8),
            child: QuillEditor.basic(
              controller: _quill,
              config: QuillEditorConfig(
                placeholder: l10n.publishBodyPlaceholder,
              ),
            ),
          ),
          const SizedBox(height: 8),
          if (_error.isNotEmpty)
            Text(_error, style: GfTheme.typographyOf(context).small.copyWith(color: colors.error)),
          if (_message.isNotEmpty)
            Text(
              _message,
              style: GfTheme.typographyOf(context).small.copyWith(color: colors.success),
            ),
          const SizedBox(height: 8),
          Row(
            children: [
              Expanded(
                child: GfButton(
                  label: l10n.publishSaveDraft,
                  variant: GfButtonVariant.outline,
                  loading: _submitting,
                  onPressed: _submitting ? null : () => _submit(topicStatus: 0),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildToolbar() {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: [
          _toolButton(
            Icons.format_bold,
            () => _quill.formatSelection(Attribute.bold),
          ),
          _toolButton(
            Icons.format_italic,
            () => _quill.formatSelection(Attribute.italic),
          ),
          _toolButton(
            Icons.format_strikethrough,
            () => _quill.formatSelection(Attribute.strikeThrough),
          ),
          _toolButton(
            Icons.format_quote,
            () => _quill.formatSelection(Attribute.blockQuote),
          ),
          _toolButton(
            Icons.code,
            () => _quill.formatSelection(Attribute.inlineCode),
          ),
          _toolButton(
            Icons.format_list_bulleted,
            () => _quill.formatSelection(Attribute.ul),
          ),
          _toolButton(
            Icons.format_list_numbered,
            () => _quill.formatSelection(Attribute.ol),
          ),
          _toolButton(Icons.image, _uploading ? null : _pickAndInsertImage),
        ],
      ),
    );
  }

  Widget _toolButton(IconData icon, VoidCallback? onTap) {
    return IconButton(icon: Icon(icon, size: 20), onPressed: onTap);
  }
}

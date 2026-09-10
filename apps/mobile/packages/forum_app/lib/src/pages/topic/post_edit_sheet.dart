import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../images/image_upload.dart';
import '../../providers.dart';
import '../../server_messages.dart';

class PostEditSheet extends ConsumerStatefulWidget {
  const PostEditSheet({super.key, required this.post});
  final PostPayload post;
  @override
  ConsumerState<PostEditSheet> createState() => _PostEditSheetState();
}

class _PostEditSheetState extends ConsumerState<PostEditSheet> {
  late final TextEditingController _text = TextEditingController(
    text: widget.post.content,
  );
  bool _busy = false;
  bool _uploading = false;
  bool _allowPop = false;
  String? _error;
  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  Future<void> _close() async {
    if (_busy || _uploading) return;
    final l10n = AppLocalizations.of(context);
    if (_text.text != widget.post.content) {
      final discard = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(l10n.publishLeaveTitle),
          content: Text(l10n.publishLeaveBody),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(l10n.commonCancel),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(l10n.publishDiscard),
            ),
          ],
        ),
      );
      if (!mounted || discard != true) return;
    }
    _pop(false);
  }

  void _pop(bool saved) {
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) Navigator.pop(context, saved);
    });
  }

  Future<void> _save() async {
    if (_busy || _uploading || _text.text.trim().isEmpty) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref
          .read(postRepositoryProvider)
          .updatePost(postId: widget.post.id, content: _text.text.trim());
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      _pop(true);
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _image() async {
    if (_busy || _uploading) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _uploading = true);
    try {
      final url = await pickAndUploadImage(ref: ref);
      if (!mounted ||
          epoch != ref.read(offlineCacheEpochProvider) ||
          url == null) {
        return;
      }
      final selection = _text.selection;
      final start = selection.isValid ? selection.start : _text.text.length;
      final end = selection.isValid ? selection.end : start;
      final image = '\n![]($url)\n';
      _text.value = TextEditingValue(
        text: _text.text.replaceRange(start, end, image),
        selection: TextSelection.collapsed(offset: start + image.length),
      );
    } catch (error) {
      if (mounted) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (_, _) => _pop(false));
    return PopScope(
      canPop: _allowPop,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: SafeArea(
        child: Padding(
          padding: EdgeInsets.fromLTRB(
            16,
            8,
            16,
            16 + MediaQuery.viewInsetsOf(context).bottom,
          ),
          child: SizedBox(
            height: MediaQuery.sizeOf(context).height * .55,
            child: Column(
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        l10n.topicEditReply,
                        style: GfTheme.typographyOf(context).title3,
                      ),
                    ),
                    IconButton(
                      tooltip: l10n.commonClose,
                      onPressed: _busy || _uploading ? null : _close,
                      icon: const Icon(Icons.close),
                    ),
                  ],
                ),
                Expanded(
                  child: TextField(
                    key: const Key('post-edit-content'),
                    controller: _text,
                    enabled: !_busy,
                    expands: true,
                    maxLines: null,
                    minLines: null,
                    textAlignVertical: TextAlignVertical.top,
                    decoration: InputDecoration(
                      hintText: l10n.topicReplyHint,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                ),
                if (_error != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(
                      _error!,
                      style: TextStyle(color: GfTheme.colorsOf(context).error),
                    ),
                  ),
                Row(
                  children: [
                    IconButton(
                      tooltip: l10n.publishToolImage,
                      onPressed: _busy || _uploading ? null : _image,
                      icon: const Icon(Icons.add_photo_alternate_outlined),
                    ),
                    const Spacer(),
                    GfButton(
                      label: l10n.commonSave,
                      loading: _busy || _uploading,
                      onPressed: _save,
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

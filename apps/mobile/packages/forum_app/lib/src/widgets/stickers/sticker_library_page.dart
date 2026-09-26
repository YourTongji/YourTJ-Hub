import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../providers.dart';
import 'sticker_image.dart';
import 'sticker_library_state.dart';
import 'sticker_strings.dart';

/// The same personal library is reachable from settings and the input panel.
class StickerLibraryPage extends ConsumerWidget {
  const StickerLibraryPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Page-local retries contain private files and uploaded URLs. Replacing the
    // account/site collection must dispose those drafts and pending UI writes.
    final collection = ref.watch(stickerCollectionProvider);
    return _StickerLibrarySession(key: ObjectKey(collection));
  }
}

class _StickerLibrarySession extends ConsumerStatefulWidget {
  const _StickerLibrarySession({super.key});

  @override
  ConsumerState<_StickerLibrarySession> createState() =>
      _StickerLibraryPageState();
}

class _StickerLibraryPageState extends ConsumerState<_StickerLibrarySession> {
  final Set<String> _selected = {};
  bool _selecting = false;
  bool _uploading = false;
  XFile? _retryFile;
  String? _retryUploadUrl;
  String? _uploadError;

  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      if (mounted) ref.read(stickerCollectionProvider).loadMine(refresh: true);
    });
  }

  Future<void> _run(Future<void> Function() action) async {
    final collection = ref.read(stickerCollectionProvider);
    try {
      await action();
    } catch (error) {
      if (!mounted || !collection.active) return;
      final strings = StickerStrings(context);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(strings.failure(error))));
    }
    if (mounted && collection.active) {
      setState(() => _selected.retainAll(collection.mine.map((e) => e.name)));
    }
  }

  Future<void> _upload({bool retry = false}) async {
    if (_uploading) return;
    final collection = ref.read(stickerCollectionProvider);
    final strings = StickerStrings(context);
    if (collection.mine.length >= 200) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(strings.limit)));
      return;
    }
    final files = ref.read(fileRepositoryProvider);
    setState(() {
      _uploading = true;
      _uploadError = null;
    });
    try {
      final file = retry
          ? _retryFile
          : await openFile(
              acceptedTypeGroups: const [
                XTypeGroup(
                  label: 'Images and GIFs',
                  extensions: ['png', 'jpg', 'jpeg', 'webp', 'gif'],
                  uniformTypeIdentifiers: ['public.image'],
                ),
              ],
            );
      if (file == null || !mounted || !collection.active) return;
      if (!retry) _retryUploadUrl = null;
      _retryFile = file;
      if (await file.length() > 4 * 1024 * 1024) {
        if (mounted) {
          setState(() {
            _uploadError = strings.tooLarge;
            _retryFile = null;
          });
        }
        return;
      }
      final bytes = await file.readAsBytes();
      if (!mounted || !collection.active) return;
      final url =
          _retryUploadUrl ??
          await files.uploadImage(bytes: bytes, filename: file.name);
      _retryUploadUrl = url;
      if (!mounted || !collection.active) return;
      await collection.save(
        fileName: url,
        displayName: String.fromCharCodes(
          file.name.replaceFirst(RegExp(r'\.[^.]+$'), '').runes.take(64),
        ),
      );
      _retryFile = null;
      _retryUploadUrl = null;
    } catch (error) {
      if (mounted && collection.active) {
        setState(() => _uploadError = strings.failure(error));
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _rename(StickerItemPayload item) async {
    final collection = ref.read(stickerCollectionProvider);
    await showGfAlertDialog<void>(
      context,
      barrierDismissible: true,
      builder: (_) => _StickerRenameDialog(item: item, collection: collection),
    );
  }

  @override
  Widget build(BuildContext context) {
    final strings = StickerStrings(context);
    final collection = ref.watch(stickerCollectionProvider);
    final busy = collection.busy || _uploading;
    return Scaffold(
      appBar: GfAppBar(
        title: Text(strings.title),
        actions: [
          TextButton(
            onPressed: collection.mine.isEmpty || busy
                ? null
                : () => setState(() {
                    _selecting = !_selecting;
                    _selected.clear();
                  }),
            child: Text(_selecting ? strings.done : strings.select),
          ),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    strings.reorderHint,
                    style: TextStyle(
                      fontSize: 13,
                      height: 1.5,
                      color: GfTheme.colorsOf(context).iconMuted,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    strings.uploadHint,
                    style: TextStyle(
                      fontSize: 13,
                      height: 1.5,
                      color: GfTheme.colorsOf(context).iconMuted,
                    ),
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    onPressed:
                        busy || stickerApiUnsupported(collection.mineError)
                        ? null
                        : _upload,
                    icon: const GfSymbol('plus', size: 20),
                    label: Text(strings.add),
                  ),
                  if (_uploading) const LinearProgressIndicator(),
                  if (_uploadError != null)
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            _uploadError!,
                            style: TextStyle(
                              color: Theme.of(context).colorScheme.error,
                            ),
                          ),
                        ),
                        if (_retryFile != null)
                          TextButton(
                            onPressed: busy ? null : () => _upload(retry: true),
                            child: Text(strings.retry),
                          ),
                      ],
                    ),
                ],
              ),
            ),
            if (collection.busy) const LinearProgressIndicator(minHeight: 2),
            Expanded(
              child: collection.loadingMine && !collection.mineLoaded
                  ? const Center(child: CircularProgressIndicator())
                  : collection.mineError != null
                  ? _status(
                      stickerApiUnsupported(collection.mineError)
                          ? strings.unsupported
                          : strings.failed,
                      retry: () => collection.loadMine(refresh: true),
                    )
                  : collection.mine.isEmpty
                  ? _status(strings.emptyMine)
                  : RefreshIndicator(
                      onRefresh: () => collection.loadMine(refresh: true),
                      child: ReorderableListView.builder(
                        padding: const EdgeInsets.fromLTRB(8, 0, 8, 24),
                        buildDefaultDragHandles: false,
                        itemCount: collection.mine.length,
                        onReorderItem: (from, to) {
                          if (!busy && !_selecting) {
                            _run(() => collection.reorder(from, to));
                          }
                        },
                        itemBuilder: (context, index) {
                          final item = collection.mine[index];
                          return ListTile(
                            key: ValueKey(item.name),
                            leading: IgnorePointer(
                              child: Opacity(
                                opacity: item.isEnabled ? 1 : 0.4,
                                child: StickerImage(
                                  name: item.name,
                                  url: item.url,
                                  label: strings.displayLabel(item),
                                  collectible: false,
                                  size: 48,
                                ),
                              ),
                            ),
                            title: Text(
                              strings.displayLabel(item),
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                            ),
                            subtitle: item.isEnabled
                                ? null
                                : Text(
                                    strings.unavailable,
                                    style: TextStyle(
                                      fontSize: 13,
                                      color: GfTheme.colorsOf(
                                        context,
                                      ).iconMuted,
                                    ),
                                  ),
                            onTap: busy || (!_selecting && !item.isEnabled)
                                ? null
                                : () {
                                    if (_selecting) {
                                      setState(
                                        () => _selected.contains(item.name)
                                            ? _selected.remove(item.name)
                                            : _selected.add(item.name),
                                      );
                                    } else {
                                      _rename(item);
                                    }
                                  },
                            trailing: _selecting
                                ? Checkbox(
                                    value: _selected.contains(item.name),
                                    onChanged: busy
                                        ? null
                                        : (value) => setState(
                                            () => value!
                                                ? _selected.add(item.name)
                                                : _selected.remove(item.name),
                                          ),
                                  )
                                : ReorderableDragStartListener(
                                    index: index,
                                    enabled: !busy,
                                    child: const SizedBox(
                                      width: 48,
                                      height: 48,
                                      child: Center(
                                        child: GfSymbol(
                                          'grip-vertical',
                                          size: 20,
                                        ),
                                      ),
                                    ),
                                  ),
                          );
                        },
                      ),
                    ),
            ),
            if (_selecting)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
                child: SizedBox(
                  width: double.infinity,
                  child: FilledButton.tonalIcon(
                    onPressed: _selected.isEmpty || busy
                        ? null
                        : () => _run(() => collection.remove({..._selected})),
                    icon: const GfSymbol('trash-2', size: 20),
                    label: Text('${strings.remove} (${_selected.length})'),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _status(String message, {VoidCallback? retry}) => GfEmpty(
    message: message == StickerStrings(context).emptyMine
        ? StickerStrings(context).mine
        : StickerStrings(context).unavailable,
    description: message,
    symbol: 'smile',
    action: retry == null
        ? null
        : GfButton(
            label: StickerStrings(context).retry,
            variant: GfButtonVariant.outline,
            onPressed: retry,
          ),
  );
}

/// Keep a failed rename editable, and discard its private label with its owner.
class _StickerRenameDialog extends ConsumerStatefulWidget {
  const _StickerRenameDialog({required this.item, required this.collection});
  final StickerItemPayload item;
  final StickerCollection collection;

  @override
  ConsumerState<_StickerRenameDialog> createState() =>
      _StickerRenameDialogState();
}

class _StickerRenameDialogState extends ConsumerState<_StickerRenameDialog> {
  TextEditingController? _controller;
  bool _saving = false;
  bool _closing = false;
  String? _error;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _controller ??= TextEditingController(
      text: StickerStrings(context).displayLabel(widget.item),
    );
  }

  @override
  void dispose() {
    _controller?.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_saving || !widget.collection.active) return;
    final strings = StickerStrings(context);
    final name = _controller!.text.trim();
    if (name == strings.displayLabel(widget.item)) {
      Navigator.pop(context);
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.collection.save(
        stickerName: widget.item.name,
        displayName: name,
      );
      if (mounted && widget.collection.active) Navigator.pop(context);
    } catch (error) {
      if (mounted && widget.collection.active) {
        setState(() => _error = strings.failure(error));
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!identical(ref.watch(stickerCollectionProvider), widget.collection) ||
        !widget.collection.active) {
      if (!_closing) {
        _closing = true;
        final route = ModalRoute.of(context);
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (route?.isActive == true) route!.navigator?.removeRoute(route);
        });
      }
      return const SizedBox.shrink();
    }
    final strings = StickerStrings(context);
    return PopScope(
      canPop: !_saving,
      child: GfAlertDialog(
        title: Text(strings.rename),
        content: GfInput(
          controller: _controller,
          autofocus: true,
          enabled: !_saving,
          maxLength: 64,
          labelText: strings.name,
          textInputAction: TextInputAction.done,
          onSubmitted: (_) => _save(),
          decoration: InputDecoration(errorText: _error, errorMaxLines: 4),
        ),
        actions: [
          GfButton(
            label: MaterialLocalizations.of(context).cancelButtonLabel,
            variant: GfButtonVariant.ghost,
            onPressed: _saving ? null : () => Navigator.pop(context),
          ),
          GfButton(
            label: strings.done,
            loading: _saving,
            onPressed: _saving ? null : _save,
          ),
        ],
      ),
    );
  }
}

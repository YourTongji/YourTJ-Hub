import 'package:flutter/foundation.dart';
import 'package:image_picker/image_picker.dart';

enum ComposerUploadStatus { queued, uploading, failed }

class ComposerUpload {
  ComposerUpload(this.id, this.file);
  final int id;
  final XFile file;
  ComposerUploadStatus status = ComposerUploadStatus.queued;
  Object? error;
  String? url;
}

/// Foreground-only, ordered uploads. A failed head waits for retry/removal, so
/// later photos never overtake it. Temporary picker files are never persisted.
class ComposerUploadQueue extends ChangeNotifier {
  ComposerUploadQueue({
    required this.upload,
    required this.onUploaded,
    required this.isCurrent,
  });

  final Future<String> Function(XFile file) upload;
  final void Function(String url) onUploaded;
  final bool Function() isCurrent;
  final List<ComposerUpload> _items = [];
  List<ComposerUpload> get items => List.unmodifiable(_items);
  bool get hasPending => _items.isNotEmpty;
  bool _running = false, _disposed = false, _active = true;
  int _nextId = 0;

  void add(List<XFile> files) {
    if (_disposed || !isCurrent()) return;
    _items.addAll(files.map((file) => ComposerUpload(_nextId++, file)));
    notifyListeners();
    _drain();
  }

  void retry(int id) {
    if (_disposed || !isCurrent()) return;
    final item = _items.where((item) => item.id == id).firstOrNull;
    if (item == null || item.status != ComposerUploadStatus.failed) return;
    item.status = ComposerUploadStatus.queued;
    item.error = null;
    notifyListeners();
    _drain();
  }

  void remove(int id) {
    if (_disposed) return;
    _items.removeWhere((item) => item.id == id);
    notifyListeners();
    _drain();
  }

  void setActive(bool active) {
    _active = active;
    if (active) _drain();
  }

  Future<void> _drain() async {
    if (_running || _disposed || !_active || !isCurrent()) return;
    _running = true;
    try {
      while (!_disposed && _active && isCurrent() && _items.isNotEmpty) {
        final item = _items.first;
        if (item.status == ComposerUploadStatus.failed) break;
        item.status = ComposerUploadStatus.uploading;
        notifyListeners();
        try {
          final url = item.url ?? await upload(item.file);
          if (_disposed || !isCurrent()) return;
          if (!_items.contains(item)) continue;
          item.url = url;
          onUploaded(url);
          _items.remove(item);
        } catch (error) {
          if (_disposed || !isCurrent()) return;
          if (!_items.contains(item)) continue;
          item.status = ComposerUploadStatus.failed;
          item.error = error;
        }
        notifyListeners();
      }
    } finally {
      _running = false;
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _items.clear();
    super.dispose();
  }
}

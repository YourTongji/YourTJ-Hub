import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/images/composer_upload_queue.dart';
import 'package:image_picker/image_picker.dart';

void main() {
  late ComposerUploadQueue queue;
  late List<Completer<String>> requests;
  late List<String> filenames, inserted;
  bool current = true;
  XFile photo(String name) => XFile.fromData(Uint8List(1), name: name, path: name);
  Future<void> tick() => Future<void>.delayed(Duration.zero);
  setUp(() {
    current = true;
    requests = [];
    filenames = [];
    inserted = [];
    queue = ComposerUploadQueue(
      isCurrent: () => current,
      upload: (file) {
        filenames.add(file.name);
        final result = Completer<String>();
        requests.add(result);
        return result.future;
      },
      onUploaded: inserted.add,
    );
  });
  tearDown(() => queue.dispose());

  test(
    'failed head waits for explicit retry and preserves selection order',
    () async {
      queue.add([photo('a'), photo('b')]);
      expect(filenames, ['a']);
      requests[0].completeError(StateError('offline'));
      await tick();
      expect(queue.items.first.status, ComposerUploadStatus.failed);
      expect(filenames, ['a']);
      queue.retry(queue.items.first.id);
      requests[1].complete('a-url');
      await tick();
      expect(filenames, ['a', 'a', 'b']);
      requests[2].complete('b-url');
      await tick();
      expect(inserted, ['a-url', 'b-url']);
      expect(queue.hasPending, isFalse);
    },
  );

  test('removing an active photo suppresses its late result', () async {
    queue.add([photo('a'), photo('b')]);
    queue.remove(queue.items.first.id);
    requests.first.complete('removed-url');
    await tick();
    expect(inserted, isEmpty);
    expect(filenames, ['a', 'b']);
    requests.last.complete('kept-url');
    await tick();
    expect(inserted, ['kept-url']);
  });

  test(
    'removing a failed head resumes the next photo without repicking',
    () async {
      queue.add([photo('a'), photo('b')]);
      requests.first.completeError(StateError('offline'));
      await tick();
      queue.remove(queue.items.first.id);
      expect(filenames, ['a', 'b']);
      requests.last.complete('b-url');
      await tick();
      expect(inserted, ['b-url']);
    },
  );

  test(
    'identity invalidation fences completion and stops later uploads',
    () async {
      queue.add([photo('a'), photo('b')]);
      current = false;
      requests.first.complete('old-url');
      await tick();
      queue.retry(queue.items.first.id);
      queue.add([photo('c')]);
      expect(filenames, ['a']);
      expect(inserted, isEmpty);
    },
  );

  test(
    'backgrounding starts no further upload until foreground resumes',
    () async {
      queue.add([photo('a'), photo('b')]);
      queue.setActive(false);
      requests.first.complete('a-url');
      await tick();
      expect(inserted, ['a-url']);
      expect(filenames, ['a']);
      queue.setActive(true);
      expect(filenames, ['a', 'b']);
      requests.last.complete('b-url');
      await tick();
    },
  );
}

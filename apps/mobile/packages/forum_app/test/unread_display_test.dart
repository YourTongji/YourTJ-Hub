import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/notifications/unread_display.dart';

void main() {
  test(
    'one initial read is shared and a newer status wins over its completion',
    () async {
      final read = Completer<NotificationPreferences>();
      var requests = 0;
      final controller = UnreadDisplayController(
        loadPreferences: () {
          requests++;
          return read.future;
        },
        savePreferences: (style) async =>
            NotificationPreferences(badgeStyle: style),
        isCurrent: () => true,
      );
      addTearDown(controller.dispose);
      final pending = controller.load();
      expect(identical(pending, controller.load()), isTrue);
      controller.acceptRemote(
        UnreadBadgeStyle.dot,
        expectedRevision: controller.revision,
      );
      read.complete(const NotificationPreferences());
      expect(await pending, isTrue);
      expect(controller.state.style, UnreadBadgeStyle.dot);
      expect(requests, 1);
    },
  );

  test(
    'a slow poll cannot undo a local choice during or after its save',
    () async {
      final saved = Completer<NotificationPreferences>();
      var writes = 0;
      final controller = UnreadDisplayController(
        loadPreferences: () async => const NotificationPreferences(),
        savePreferences: (style) {
          writes++;
          return saved.future;
        },
        isCurrent: () => true,
      );
      addTearDown(controller.dispose);
      await controller.load();
      final pollRevision = controller.revision;
      final pending = controller.update(UnreadBadgeStyle.dot);
      expect(controller.state.saving, isTrue);
      controller.acceptRemote(
        UnreadBadgeStyle.number,
        expectedRevision: pollRevision,
      );
      expect(controller.state.style, UnreadBadgeStyle.dot);
      expect(await controller.update(UnreadBadgeStyle.number), isFalse);
      saved.complete(
        const NotificationPreferences(badgeStyle: UnreadBadgeStyle.dot),
      );
      expect(await pending, isTrue);
      controller.acceptRemote(
        UnreadBadgeStyle.number,
        expectedRevision: pollRevision,
      );
      expect(controller.state.style, UnreadBadgeStyle.dot);
      expect(controller.state.saving, isFalse);
      expect(await controller.update(UnreadBadgeStyle.dot), isTrue);
      expect(writes, 1);
    },
  );

  test(
    'a failed save rolls back the style and can be retried explicitly',
    () async {
      var fails = true;
      final failure = StateError('offline');
      final controller = UnreadDisplayController(
        loadPreferences: () async => const NotificationPreferences(),
        savePreferences: (style) async {
          if (fails) throw failure;
          return NotificationPreferences(badgeStyle: style);
        },
        isCurrent: () => true,
      );
      addTearDown(controller.dispose);
      await controller.load();
      expect(await controller.update(UnreadBadgeStyle.dot), isFalse);
      expect(controller.state.style, UnreadBadgeStyle.number);
      expect(controller.state.error, same(failure));
      expect(controller.state.saving, isFalse);
      fails = false;
      expect(await controller.update(UnreadBadgeStyle.dot), isTrue);
      expect(controller.state.style, UnreadBadgeStyle.dot);
      expect(controller.state.error, isNull);
    },
  );

  test(
    'account invalidation rejects old reads and save acknowledgements',
    () async {
      var current = true;
      final save = Completer<NotificationPreferences>();
      final controller = UnreadDisplayController(
        loadPreferences: () async => const NotificationPreferences(),
        savePreferences: (_) => save.future,
        isCurrent: () => current,
      );
      addTearDown(controller.dispose);
      await controller.load();
      final pending = controller.update(UnreadBadgeStyle.dot);
      current = false;
      final revision = controller.revision;
      controller.acceptRemote(
        UnreadBadgeStyle.number,
        expectedRevision: revision,
      );
      save.complete(const NotificationPreferences());
      expect(await pending, isFalse);
      expect(controller.revision, revision);
      expect(controller.state.style, UnreadBadgeStyle.dot);
    },
  );

  test(
    'a failed initial read is visible and a retry loads the preference',
    () async {
      var fails = true;
      final controller = UnreadDisplayController(
        loadPreferences: () async {
          if (fails) throw StateError('offline');
          return const NotificationPreferences(
            badgeStyle: UnreadBadgeStyle.dot,
          );
        },
        savePreferences: (style) async =>
            NotificationPreferences(badgeStyle: style),
        isCurrent: () => true,
      );
      addTearDown(controller.dispose);
      expect(await controller.load(), isFalse);
      expect(controller.state.loaded, isFalse);
      expect(controller.state.error, isNotNull);
      expect(await controller.update(UnreadBadgeStyle.dot), isFalse);
      fails = false;
      expect(await controller.load(), isTrue);
      expect(controller.state.style, UnreadBadgeStyle.dot);
    },
  );
}

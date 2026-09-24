import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/realtime/foreground_realtime.dart';

void main() {
  test(
    'one foreground stream resyncs on hello and invalidates REST views',
    () async {
      final stream = StreamController<ForumSseFrame>();
      final connected = Completer<void>();
      var connectCount = 0;
      var resyncCount = 0;
      var chatCount = 0;
      var fallbackCount = 0;
      final health = <bool>[];
      final coordinator = ForegroundRealtimeCoordinator(
        readToken: () async => 'session',
        connect: (token, cancel, onActivity) async {
          expect(token, 'session');
          connectCount++;
          connected.complete();
          return ForumRealtimeConnection(stream.stream, cancel);
        },
        onResync: () => resyncCount++,
        onEvent: (frame) {
          if (frame.event == 'chat.changed') chatCount++;
        },
        onFallbackTick: () => fallbackCount++,
        onHealthChanged: health.add,
      );
      coordinator.start();
      coordinator.start();
      await connected.future;
      stream.add(
        const ForumSseFrame(
          event: 'hello',
          data:
              '{"version":1,"heartbeatSeconds":15,"resync":true,"capabilities":{"visibleRead":true}}',
        ),
      );
      stream.add(
        const ForumSseFrame(
          event: 'chat.changed',
          data: '{"convId":4,"change":"message"}',
        ),
      );
      await Future<void>.delayed(Duration.zero);
      expect(connectCount, 1);
      expect(resyncCount, 1);
      expect(chatCount, 1);
      expect(health, [true]);
      expect(fallbackCount, 0);
      coordinator.stop();
      expect(health, [true, false]);
      await stream.close();
    },
  );

  test(
    'unsupported endpoint falls back, and stopped pending connection is discarded',
    () async {
      final fallback = Completer<void>();
      var calls = 0;
      final coordinator = ForegroundRealtimeCoordinator(
        readToken: () async => 'session',
        connect: (token, cancel, onActivity) async {
          calls++;
          throw const ForumRealtimeHttpException(404);
        },
        onResync: () {},
        onEvent: (_) {},
        onFallbackTick: fallback.complete,
        onHealthChanged: (_) {},
      );
      coordinator.start();
      await fallback.future;
      coordinator.stop();
      expect(calls, 1);

      final pending = Completer<ForumRealtimeConnection>();
      CancelToken? issued;
      final second = ForegroundRealtimeCoordinator(
        readToken: () async => 'session',
        connect: (token, cancel, onActivity) {
          issued = cancel;
          return pending.future;
        },
        onResync: () {},
        onEvent: (_) {},
        onFallbackTick: () {},
        onHealthChanged: (_) {},
      );
      second.start();
      await Future<void>.delayed(Duration.zero);
      second.stop();
      expect(issued!.isCancelled, isTrue);
      pending.complete(
        ForumRealtimeConnection(const Stream<ForumSseFrame>.empty(), issued!),
      );
      await Future<void>.delayed(Duration.zero);
    },
  );

  test(
    'stream completion starts one fallback and retires its callbacks',
    () async {
      final stream = StreamController<ForumSseFrame>();
      final connected = Completer<void>();
      var fallbackCount = 0;
      final coordinator = ForegroundRealtimeCoordinator(
        readToken: () async => 'session',
        connect: (token, cancel, onActivity) async {
          connected.complete();
          return ForumRealtimeConnection(stream.stream, cancel);
        },
        onResync: () {},
        onEvent: (_) {},
        onFallbackTick: () => fallbackCount++,
        onHealthChanged: (_) {},
      );
      coordinator.start();
      await connected.future;
      stream.add(
        const ForumSseFrame(
          event: 'hello',
          data:
              '{"version":1,"heartbeatSeconds":15,"resync":true,"capabilities":{"visibleRead":true}}',
        ),
      );
      await stream.close();
      await Future<void>.delayed(Duration.zero);
      expect(fallbackCount, 1);
      coordinator.stop();
    },
  );
}

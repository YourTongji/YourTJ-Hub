import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'api_client_test.dart' show MockAdapter, ResponseData;

class Storage implements TokenStorage {
  @override
  Future<String?> read() async => 'test-session';
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

void main() {
  test(
    'calendar export parses the shared contract and uses the forum session',
    () async {
      final dio = Dio();
      final cancel = CancelToken();
      dio.httpClientAdapter = MockAdapter((r) async {
        expect(r.path, '/api/campus/calendar-export');
        expect(r.headers['Authorization'], 'Bearer test-session');
        return ResponseData(200, {
          'code': 0,
          'result': {
            'filename': 'yourtj-courses-2026-09-14.ics',
            'content': 'BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n',
            'eventCount': 2,
          },
        });
      });
      final result = await CampusRepository(
        GfApiClient(dio: dio, tokenStorage: Storage()),
      ).exportCalendar(cancelToken: cancel);
      expect(result.filename, endsWith('.ics'));
      expect(result.eventCount, 2);
      expect(result.content, startsWith('BEGIN:VCALENDAR\r\n'));
    },
  );
  test(
    'campus mutations use session headers and exact revision bodies',
    () async {
      final dio = Dio();
      final calls = <RequestOptions>[];
      dio.httpClientAdapter = MockAdapter((r) async {
        calls.add(r);
        expect(r.headers['Authorization'], 'Bearer test-session');
        return ResponseData(200, {
          'code': 0,
          'result': r.path.endsWith('/start')
              ? {'url': 'https://api.tongji.edu.cn/authorize'}
              : null,
        });
      });
      final repository = CampusRepository(
        GfApiClient(dio: dio, tokenStorage: Storage()),
      );
      await repository.start('reauthorize');
      await repository.confirm();
      await repository.unbind('revision-2');
      expect(calls.map((r) => r.path), [
        '/api/campus/tongji/start',
        '/api/campus/tongji/confirm',
        '/api/campus/tongji/unbind',
      ]);
      expect(calls.first.data, {'mode': 'reauthorize'});
      expect(calls.last.data, {'revision': 'revision-2'});
    },
  );
  test(
    'campus detail preserves large IDs and GET cancellation reaches transport',
    () async {
      final started = Completer<void>();
      final waiting = Completer<ResponseData>();
      final dio = Dio();
      dio.httpClientAdapter = MockAdapter((r) async {
        expect(r.path, '/api/campus/messages/90071992547409931');
        started.complete();
        return waiting.future;
      });
      final repository = CampusRepository(
        GfApiClient(dio: dio, tokenStorage: Storage()),
      );
      final cancel = CancelToken();
      final response = repository.message(
        '90071992547409931',
        cancelToken: cancel,
      );
      final assertion = expectLater(response, throwsA(isA<NetworkException>()));
      await started.future;
      cancel.cancel();
      await assertion;
      waiting.complete(ResponseData(200, {'code': 0, 'result': null}));
    },
  );
}

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'api_client_test.dart' show MockAdapter, ResponseData;
import 'push_repository_test.dart' show Tokens;

void main() {
  test(
    'unknown legacy counts stay unknown while preference responses validate their enum',
    () {
      final status = UnreadStatusPayload.fromJson({
        'notifications': true,
        'messages': false,
      });
      expect(status.notificationCount, isNull);
      expect(status.messageCount, isNull);
      expect(
        UnreadBadgeStyle.parse(status.badgeStyle),
        UnreadBadgeStyle.number,
      );
      expect(
        NotificationPreferences.fromJson({'badgeStyle': 'dot'}).badgeStyle,
        UnreadBadgeStyle.dot,
      );
      expect(
        () => NotificationPreferences.fromJson({'badgeStyle': 'unknown'}),
        throwsFormatException,
      );
      expect(() => NotificationPreferences.fromJson({}), throwsFormatException);
    },
  );
  test(
    'preference reads and writes freeze the submitting session and retain exact unread counts',
    () async {
      final dio = Dio();
      addTearDown(dio.close);
      final tokens = Tokens();
      var requests = 0;
      dio.httpClientAdapter = MockAdapter((request) async {
        requests++;
        expect(request.headers['Authorization'], 'Bearer session');
        if (request.path.endsWith('/unread-status')) {
          return ResponseData(200, {
            'code': 0,
            'result': {
              'notifications': true,
              'messages': true,
              'notificationCount': 4,
              'messageCount': 12,
              'badgeStyle': 'dot',
            },
          });
        }
        expect(request.path, '/api/forum/notification-preferences');
        if (request.method == 'PUT') {
          expect(request.data, {'badgeStyle': 'dot'});
        }
        return ResponseData(200, {
          'code': 0,
          'result': {'badgeStyle': request.method == 'PUT' ? 'dot' : 'number'},
        });
      });
      final repository = await NotificationRepository(
        GfApiClient(dio: dio, tokenStorage: tokens),
      ).forSession();
      tokens.value = 'another-account';
      expect(
        (await repository.preferences()).badgeStyle,
        UnreadBadgeStyle.number,
      );
      expect(
        (await repository.updatePreferences(UnreadBadgeStyle.dot)).badgeStyle,
        UnreadBadgeStyle.dot,
      );
      final status = await repository.getUnreadStatus();
      expect(status.notificationCount, 4);
      expect(status.messageCount, 12);
      expect(requests, 3);
    },
  );
}

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/widgets/moderation_blocked_dialog.dart';

void main() {
  test('only AI moderation block codes trigger the blocked dialog', () {
    for (final code in <String>[
      'content.aiModeration.blocked',
      'content.aiModeration.externalImageBlocked',
    ]) {
      expect(
        isModerationBlocked(
          ApiException(fallbackMessage: 'blocked', messageCode: code),
        ),
        isTrue,
      );
    }
    for (final code in <String?>[
      'content.sensitive.blocked',
      'common.rateLimited',
      null,
    ]) {
      expect(
        isModerationBlocked(
          ApiException(fallbackMessage: 'other', messageCode: code),
        ),
        isFalse,
      );
    }
    expect(isModerationBlocked(Exception('network')), isFalse);
  });
}

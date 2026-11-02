import 'package:forum_app/src/push/push_service.dart';
import 'local_journey_test.dart' as journey;

// Font acceptance reads the real embedded console after the full native writing
// journey. OS push delivery is separate; a denied baseline avoids system prompts.
class _PermissionDeniedPushDriver extends PushDriver {
  @override
  Future<bool> permission({required bool request}) async => false;
}

void main() =>
    journey.registerLocalJourney(pushDriver: _PermissionDeniedPushDriver());

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'course workspace discovery requires an authenticated course permission',
    () {
      const viewer = ViewerPayload(
        id: 1,
        username: 'member',
        email: '',
        avatarUrl: '',
        isAuthenticated: true,
        canAccessAdmin: true,
        isModerator: true,
        requiresEmailVerification: false,
      );
      expect(viewer.canManageCourses, isFalse);
      for (final permissions in <List<int>>[
        [],
        [1],
        [2, 3, 4, 5],
      ]) {
        expect(
          viewer.copyWith(adminPermissions: permissions).canManageCourses,
          isFalse,
        );
      }
      for (final permissions in [
        [0],
        [6],
        [1, 6],
      ]) {
        final manager = viewer.copyWith(adminPermissions: permissions);
        expect(manager.canManageCourses, isTrue);
        expect(
          manager.copyWith(isAuthenticated: false).canManageCourses,
          isFalse,
        );
      }
    },
  );
}

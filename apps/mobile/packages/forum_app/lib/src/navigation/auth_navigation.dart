import 'package:core/core.dart';

import '../providers.dart';

/// A return location is navigation context only, never an instruction to write.
/// Keep this allowlist aligned with the native routes in router.dart.
String? safeAuthReturnTo(String? raw) {
  try {
    return _parseAuthReturnTo(raw);
  } on FormatException {
    return null;
  }
}

String? _parseAuthReturnTo(String? raw) {
  if (raw == null || raw.isEmpty || raw.length > 2048 || raw != raw.trim()) {
    return null;
  }
  if (!raw.startsWith('/') ||
      raw.startsWith('//') ||
      raw.contains('\\') ||
      RegExp(r'[\x00-\x20\x7f]').hasMatch(raw)) {
    return null;
  }
  final uri = Uri.tryParse(raw);
  if (uri == null || uri.hasScheme || uri.hasAuthority) return null;
  final parts = uri.pathSegments;
  if (parts.any(
    (part) =>
        part == '.' ||
        part == '..' ||
        part.contains('/') ||
        part.contains('\\') ||
        RegExp(r'[\x00-\x1f\x7f]').hasMatch(part),
  )) {
    return null;
  }
  bool positive(String value) =>
      RegExp(r'^[1-9][0-9]*$').hasMatch(value) &&
      (int.tryParse(value) ?? 0) > 0;
  const sections = {
    'profile',
    'account',
    'privacy',
    'binding',
    'security',
    'appearance',
    'notifications',
  };
  const staticPaths = {
    '/',
    '/campus',
    '/campus/explore',
    '/campus/official',
    '/notifications',
    '/messages',
    '/chat',
    '/search',
    '/publish',
    '/settings',
    '/my-course-reviews',
    '/my-content',
    '/recycle-bin',
    '/profile',
    '/moderation',
    '/moderation/courses',
    '/moderation/course-reviews',
    '/admin',
    '/about',
    '/terms',
    '/privacy',
    '/drafts',
    '/schedule',
    '/courses',
    '/wiki',
    '/wiki/search',
  };
  final dynamicPath =
      (parts.length == 2 &&
          {'p', 'u', 'courses'}.contains(parts.first) &&
          positive(parts[1])) ||
      (parts.length == 3 &&
          parts.first == 'c' &&
          parts[1].isNotEmpty &&
          positive(parts[2])) ||
      (parts.length == 2 &&
          parts.first == 'settings' &&
          sections.contains(parts[1])) ||
      (parts.length >= 2 &&
          parts.first == 'wiki' &&
          parts.every((p) => p.isNotEmpty));
  if (!staticPaths.contains(uri.path) && !dynamicPath) return null;
  final allowedQuery = switch (parts.firstOrNull) {
    'p' => const {'postNo'},
    'chat' || 'messages' => const {'userId', 'username', 'avatar'},
    'publish' => const {'topicId', 'id', 'local', 'type', 'contentType'},
    'courses' => const {'q', 'reviewId', 'offeringId'},
    'profile' => const {'stream'},
    'search' || 'wiki' => const {'q'},
    null || '' => const {'sort'},
    _ => const <String>{},
  };
  for (final entry in uri.queryParametersAll.entries) {
    if (!allowedQuery.contains(entry.key) || entry.value.length != 1) {
      return null;
    }
    final value = entry.value.single;
    if (value.length > 1024 || RegExp(r'[\x00-\x1f\x7f]').hasMatch(value)) {
      return null;
    }
    if ({
          'postNo',
          'userId',
          'topicId',
          'id',
          'reviewId',
          'offeringId',
        }.contains(entry.key) &&
        !positive(value)) {
      return null;
    }
  }
  if (uri.path == '/chat' && !positive(uri.queryParameters['userId'] ?? '')) {
    return null;
  }
  if (uri.hasFragment &&
      (parts.firstOrNull != 'wiki' ||
          RegExp(
            r'[\x00-\x1f\x7f]',
          ).hasMatch(Uri.decodeComponent(uri.fragment)))) {
    return null;
  }
  return uri.toString();
}

String authLoginLocation({String? returnTo}) => Uri(
  path: '/login',
  queryParameters: {'returnTo': safeAuthReturnTo(returnTo) ?? '/'},
).toString();

bool requiresNativeSession(Uri uri) => const {
  '/notifications',
  '/messages',
  '/chat',
  '/publish',
  '/drafts',
  '/my-course-reviews',
  '/my-content',
  '/recycle-bin',
  '/admin',
  '/moderation',
  '/moderation/courses',
  '/moderation/course-reviews',
  '/settings/profile',
  '/settings/account',
  '/settings/privacy',
  '/settings/binding',
  '/settings/security',
  '/settings/notifications',
}.contains(uri.path);

/// Called before building a protected page, so a guest never starts its fetch or
/// mutation workflow. The old route is captured before awaiting secure storage.
Future<String?> authNavigationRedirect({
  required Uri requested,
  required TokenStorage tokenStorage,
  String? previousLocation,
}) async {
  if (requested.path == '/login') {
    String? candidate;
    try {
      final targets = requested.queryParametersAll['returnTo'];
      candidate = targets == null
          ? previousLocation
          : targets.length == 1
          ? targets.single
          : null;
    } on FormatException {
      // A deep link may contain invalid UTF-8 in the outer query itself.
      candidate = null;
    }
    final canonical = authLoginLocation(returnTo: candidate);
    return requested.toString() == canonical ? null : canonical;
  }
  if (requiresNativeSession(requested) &&
      !await hasSessionToken(tokenStorage)) {
    return authLoginLocation(returnTo: requested.toString());
  }
  return null;
}

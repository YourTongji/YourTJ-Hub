/// Notifications may navigate only to notification-owned in-app routes.
String? pushRoute(String? value) {
  if (value == null) return null;
  final uri = Uri.tryParse(value);
  if (uri == null || uri.hasScheme || uri.hasAuthority || uri.hasFragment) {
    return null;
  }
  final rawPath = value
      .split('?')
      .first
      .replaceAll(RegExp('%2e', caseSensitive: false), '.');
  if (RegExp(r'(^|/)\.\.?(/|$)').hasMatch(rawPath)) return null;
  bool positive(String? value) =>
      value != null && RegExp(r'^[1-9][0-9]*$').hasMatch(value);
  final allowed = <String>{'recipientId'};
  if (RegExp(r'^/p/[1-9][0-9]*$').hasMatch(uri.path)) {
    allowed.add('postNo');
  } else if (uri.path == '/chat') {
    allowed.add('userId');
    if (!positive(uri.queryParameters['userId'])) return null;
  } else if (uri.path.startsWith('/wiki/')) {
    if (uri.pathSegments
        .skip(1)
        .any(
          (part) =>
              part.isEmpty ||
              part == '.' ||
              part == '..' ||
              part.contains('/') ||
              part.contains('\\') ||
              part.runes.any((char) => char < 32),
        )) {
      return null;
    }
  } else if (!RegExp(r'^/u/[1-9][0-9]*$').hasMatch(uri.path) &&
      uri.path != '/notifications' &&
      uri.path != '/my-content') {
    return null;
  }
  if (uri.queryParametersAll.entries.any(
    (entry) =>
        !allowed.contains(entry.key) ||
        entry.value.length != 1 ||
        !positive(entry.value.single),
  )) {
    return null;
  }
  return value;
}

/// Only the configured first-party origin may stay in the authenticated view.
/// Export matching is deliberately narrower than general navigation: credentials
/// must never accompany a download selected by an arbitrary URL or redirect.
class AdminNavigation {
  AdminNavigation(this.origin);
  final Uri origin;

  bool get isSecureOrigin =>
      origin.userInfo.isEmpty &&
      (origin.scheme == 'https' ||
          (origin.scheme == 'http' &&
              const [
                'localhost',
                '127.0.0.1',
                '::1',
                '10.0.2.2',
              ].contains(origin.host)));

  bool isSameOrigin(Uri uri) =>
      uri.userInfo.isEmpty &&
      uri.scheme == origin.scheme &&
      uri.host == origin.host &&
      uri.port == origin.port;

  bool isExport(Uri uri) =>
      isSameOrigin(uri) &&
      uri.query.isEmpty &&
      uri.fragment.isEmpty &&
      RegExp(
        r'^/api/admin/data/export/download/[1-9][0-9]*$',
      ).hasMatch(uri.path);
}

import 'package:core/core.dart';

const profileSocialProviders = {
  'github': ('GitHub', 'https://github.com/'),
  'twitter': ('X / Twitter', 'https://twitter.com/'),
  'linkedIn': ('LinkedIn', 'https://www.linkedin.com/in/'),
  'weibo': ('Weibo', 'https://weibo.com/'),
  'bilibili': ('Bilibili', 'https://space.bilibili.com/'),
  'zhihu': ('Zhihu', 'https://www.zhihu.com/people/'),
};

String? normalizeProfileLink(String value, {String? prefix}) {
  final raw = value.trim();
  if (raw.isEmpty) return '';
  final uri = Uri.tryParse(raw);
  if (uri == null) return null;
  if (!uri.hasScheme && prefix != null && !raw.startsWith('//')) {
    return '$prefix${Uri.encodeComponent(raw)}';
  }
  return ['https', 'http'].contains(uri.scheme) &&
          uri.host.isNotEmpty &&
          uri.userInfo.isEmpty
      ? uri.toString()
      : null;
}

/// Display only explicit public HTTP(S) destinations from the server. Unknown
/// providers retain their label; editing them must not erase their stored URL.
List<(String, Uri, String?)> publicProfileLinks(UserCardPayload user) {
  final links = <(String, Uri, String?)>[];
  void add(String label, String? raw, [String? provider]) {
    final url = normalizeProfileLink(raw ?? '');
    if (url == null || url.isEmpty) return;
    final uri = Uri.parse(url);
    links.add((label.trim().isEmpty ? uri.host : label, uri, provider));
  }

  add(user.websiteName, user.website);
  for (final entry in user.externalInformation.entries) {
    add(
      profileSocialProviders[entry.key]?.$1 ?? entry.key,
      entry.value.link,
      entry.key,
    );
  }
  return links;
}

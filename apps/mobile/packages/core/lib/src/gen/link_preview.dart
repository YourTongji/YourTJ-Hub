/// Link-preview contract mirror for `/api/link-previews/resolve`.
///
/// OpenAPI remains the source of truth; Dart generation is not enabled for
/// this package yet, so this file follows the other hand-maintained mirrors.
class LinkPreviewPayload {
  const LinkPreviewPayload({
    required this.requestedUrl,
    required this.kind,
    required this.status,
    this.url,
    this.displayHost,
    this.registrableDomain,
    this.siteName,
    this.title,
    this.description,
    this.imageUrl,
    this.faviconUrl,
    this.fetchedAt,
    this.campus = false,
    this.platform,
    this.resourceType,
    this.author,
  });

  final String requestedUrl;
  final String kind;
  final String status;
  final String? url;
  final String? displayHost;
  final String? registrableDomain;
  final String? siteName;
  final String? title;
  final String? description;
  final String? imageUrl;
  final String? faviconUrl;
  final DateTime? fetchedAt;

  /// Rendered from the deployment's local campus configuration without any
  /// outbound request. `title` / `description` may be absent in that case, so
  /// clients supply their own localized fallback copy.
  final bool campus;

  /// Strictly recognized external platform resource (`bilibili` /
  /// `apple_music`). A platform identifier, not a display name: the localized
  /// label comes from the client, while [siteName] stays the page's own label.
  final String? platform;

  /// Resource kind inside the platform (`video` / `song` / `album`). Mirrors
  /// the OpenAPI enum; only present together with [platform].
  final String? resourceType;

  /// Uploader or artist published by the platform source. Optional, and never
  /// synthesized by the server.
  final String? author;

  bool get isReady =>
      status == 'ready' &&
      (url?.isNotEmpty ?? false) &&
      ((title?.isNotEmpty ?? false) || campus);

  factory LinkPreviewPayload.fromJson(Map<String, dynamic> json) {
    final String? fetchedAt = json['fetchedAt'] as String?;
    return LinkPreviewPayload(
      requestedUrl: json['requestedUrl'] as String? ?? '',
      kind: json['kind'] as String? ?? 'unknown',
      status: json['status'] as String? ?? 'unavailable',
      url: json['url'] as String?,
      displayHost: json['displayHost'] as String?,
      registrableDomain: json['registrableDomain'] as String?,
      siteName: json['siteName'] as String?,
      title: json['title'] as String?,
      description: json['description'] as String?,
      imageUrl: json['imageUrl'] as String?,
      faviconUrl: json['faviconUrl'] as String?,
      fetchedAt: fetchedAt == null ? null : DateTime.tryParse(fetchedAt),
      campus: json['campus'] as bool? ?? false,
      platform: json['platform'] as String?,
      resourceType: json['resourceType'] as String?,
      author: json['author'] as String?,
    );
  }
}

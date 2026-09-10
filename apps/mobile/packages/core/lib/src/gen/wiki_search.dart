/// Mirror of the existing WikiSearchResult contract. Search is grouped by page.
class WikiSearchResult {
  const WikiSearchResult({
    required this.query,
    required this.items,
    required this.total,
    required this.searchUnavailable,
  });
  final String query;
  final List<WikiSearchItem> items;
  final int total;
  final bool searchUnavailable;
  factory WikiSearchResult.fromJson(Map<String, dynamic> json) =>
      WikiSearchResult(
        query: json['query'] as String? ?? '',
        items: (json['items'] as List? ?? [])
            .map(
              (e) =>
                  WikiSearchItem.fromJson(Map<String, dynamic>.from(e as Map)),
            )
            .toList(),
        total: (json['total'] as num?)?.toInt() ?? 0,
        searchUnavailable: json['searchUnavailable'] == true,
      );
}

class WikiSearchItem {
  const WikiSearchItem({
    required this.path,
    required this.title,
    required this.heading,
    required this.snippet,
    required this.anchors,
    required this.namespace,
    required this.titleHit,
    required this.score,
    required this.hitType,
  });
  final String path, title, heading, snippet, namespace;
  final List<String> anchors;
  final bool titleHit;
  final double score;
  final String hitType;
  factory WikiSearchItem.fromJson(Map<String, dynamic> json) => WikiSearchItem(
    titleHit: json['titleHit'] == true,
    score: (json['score'] as num?)?.toDouble() ?? 0,
    hitType: json['hitType'] as String? ?? '',
    path: json['path'] as String? ?? '',
    title: json['title'] as String? ?? '',
    heading: json['heading'] as String? ?? '',
    snippet: json['snippet'] as String? ?? '',
    namespace: json['namespace'] as String? ?? '',
    anchors: (json['anchors'] as List? ?? []).whereType<String>().toList(),
  );
}

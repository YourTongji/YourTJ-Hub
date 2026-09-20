import 'package:tldts/data/trie.dart' as public_suffix_data;

class _SuffixMatch {
  const _SuffixMatch(this.index);

  final int index;
}

/// Returns the eTLD+1 using the Public Suffix List bundled by `tldts`.
///
/// The package's public parser currently assumes a covariant map shape that
/// fails under sound Dart for some private suffixes. Walking its generated,
/// pinned trie here keeps the policy data authoritative without copying the
/// list or falling back to unsafe label splitting.
String registrableDomain(String hostname) {
  final String normalized = hostname.toLowerCase().replaceFirst(
    RegExp(r'\.$'),
    '',
  );
  if (normalized.isEmpty ||
      normalized.contains(':') ||
      RegExp(r'^\d+(?:\.\d+){3}$').hasMatch(normalized)) {
    return normalized;
  }
  final List<String> labels = normalized.split('.');
  final _SuffixMatch? exception = _lookup(
    labels,
    public_suffix_data.exceptions,
  );
  final int suffixStart;
  if (exception != null) {
    suffixStart = exception.index + 1;
  } else {
    final _SuffixMatch? rule = _lookup(labels, public_suffix_data.rules);
    suffixStart = rule?.index ?? labels.length - 1;
  }
  if (suffixStart <= 0) return normalized;
  return labels.sublist(suffixStart - 1).join('.');
}

_SuffixMatch? _lookup(List<String> labels, List<dynamic>? trie) {
  _SuffixMatch? match;
  List<dynamic>? node = trie;
  int index = labels.length - 1;
  while (node != null) {
    final int flags = node[0] as int;
    if ((flags & 3) != 0) match = _SuffixMatch(index + 1);
    if (index < 0) break;
    final Map<dynamic, dynamic> children = node[1] as Map<dynamic, dynamic>;
    final Object? child = children[labels[index]] ?? children['*'];
    node = child is List<dynamic> ? child : null;
    index--;
  }
  return match;
}

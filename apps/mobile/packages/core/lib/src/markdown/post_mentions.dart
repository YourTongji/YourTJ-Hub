import '../gen/topic.dart';

/// Render only server-accepted occurrences. Never guess identities or parse a
/// second mention grammar. The persisted/editor Markdown remains unchanged.
String expandPostMentions(String source, List<PostMention> mentions) {
  final ordered = List<PostMention>.of(mentions)
    ..sort((a, b) => a.start.compareTo(b.start));
  final out = StringBuffer();
  var cursor = 0;
  for (final mention in ordered) {
    if (mention.userId <= 0 ||
        mention.start < cursor ||
        mention.end > source.length ||
        mention.end <= mention.start ||
        !RegExp(r'^[a-zA-Z0-9_-]{1,64}$').hasMatch(mention.username) ||
        source.substring(mention.start, mention.end) !=
            '@${mention.username}') {
      continue;
    }
    out.write(source.substring(cursor, mention.start));
    final label = mention.username.replaceAll('_', r'\_');
    out.write('[@$label](/u/${mention.userId})');
    cursor = mention.end;
  }
  out.write(source.substring(cursor));
  return out.toString();
}

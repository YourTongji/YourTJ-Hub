/// Match the catalog's canonical term to PK calendar labels. This follows the
/// server's course.NormalizeTermLabel; unknown labels never match a term.
String? canonicalPlannerTerm(String label) {
  final match = RegExp(
    r'(\d{4}-\d{4})[^0-9一二三四五六七八九十]*([0-9一二三四五六七八九十]+)[^0-9一二三四五六七八九十]*$',
  ).firstMatch(label.trim());
  if (match == null) return null;
  final raw = match[2]!;
  const digits = {
    '一': 1,
    '二': 2,
    '三': 3,
    '四': 4,
    '五': 5,
    '六': 6,
    '七': 7,
    '八': 8,
    '九': 9,
  };
  final ordinal =
      int.tryParse(raw) ??
      digits[raw] ??
      (raw == '十'
          ? 10
          : raw.startsWith('十') && raw.length == 2
          ? 10 + (digits[raw[1]] ?? 0)
          : raw.endsWith('十') && raw.length == 2
          ? (digits[raw[0]] ?? 0) * 10
          : raw.length == 3 &&
                raw[1] == '十' &&
                digits.containsKey(raw[0]) &&
                digits.containsKey(raw[2])
          ? digits[raw[0]]! * 10 + digits[raw[2]]!
          : null);
  return ordinal == null ? null : '${match[1]}-$ordinal';
}

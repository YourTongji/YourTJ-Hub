/// 主机名归一化为可直接与服务端比较的 ASCII（punycode）形式（issue #1200）。
///
/// `Uri.host` 不做 IDNA 转换：`https://中国.com/` 的主机是百分号编码的
/// `%E4%B8%AD%E5%9B%BD.com`，而服务端把白名单条目存成 `xn--fiqs8s.com`
/// （idna.Lookup.ToASCII），浏览器也总是给 punycode 形式。两端要命中同一份
/// 清单，客户端必须先还原百分号编码再对非 ASCII 标签做 RFC 3492 编码。
///
/// 与 UTS46 的映射细节不同：这里只做 punycode 编码，不做兼容字符规范化与
/// Unicode 组合形式（NFC）归一，覆盖中文、日文与预组合拉丁重音等常见 IDN。
/// 已知限制：链接主机名写成分解形式（如 `cafe` + U+0301）而清单条目是预组合
/// 形式时，两端仍可能不相等——服务端与浏览器都会先做 UTS46/NFC 归一。该形态
/// 在正文里极罕见，且服务端会拒绝无法规范化的条目，因此不会出现「服务端接受、
/// 客户端永不命中」的清单项。
library;

const int _base = 36;
const int _tmin = 1;
const int _tmax = 26;
const int _skew = 38;
const int _damp = 700;
const int _initialBias = 72;
const int _initialN = 128;

int _adapt(int delta, int numPoints, bool firstTime) {
  delta = firstTime ? delta ~/ _damp : delta ~/ 2;
  delta += delta ~/ numPoints;
  int k = 0;
  while (delta > ((_base - _tmin) * _tmax) ~/ 2) {
    delta ~/= _base - _tmin;
    k += _base;
  }
  return k + (((_base - _tmin + 1) * delta) ~/ (delta + _skew));
}

String _digitToChar(int digit) => digit < 26
    ? String.fromCharCode(0x61 + digit)
    : String.fromCharCode(0x30 + digit - 26);

/// RFC 3492 编码：把含非 ASCII 字符的单个标签转成 punycode 主体（不含前缀）。
String punycodeLabel(String label) {
  final List<int> codePoints = label.runes.toList();
  final StringBuffer output = StringBuffer();
  int n = _initialN;
  int delta = 0;
  int bias = _initialBias;
  int basicCount = 0;
  for (final int codePoint in codePoints) {
    if (codePoint < 0x80) {
      output.writeCharCode(codePoint);
      basicCount++;
    }
  }
  int handled = basicCount;
  if (basicCount > 0) output.write('-');

  while (handled < codePoints.length) {
    int m = 0x10FFFF;
    for (final int codePoint in codePoints) {
      if (codePoint >= n && codePoint < m) m = codePoint;
    }
    delta += (m - n) * (handled + 1);
    n = m;
    for (final int codePoint in codePoints) {
      if (codePoint < n) delta++;
      if (codePoint != n) continue;
      int q = delta;
      for (int k = _base; ; k += _base) {
        final int t = k <= bias
            ? _tmin
            : (k >= bias + _tmax ? _tmax : k - bias);
        if (q < t) break;
        output.write(_digitToChar(t + (q - t) % (_base - t)));
        q = (q - t) ~/ (_base - t);
      }
      output.write(_digitToChar(q));
      bias = _adapt(delta, handled + 1, handled == basicCount);
      delta = 0;
      handled++;
    }
    delta++;
    n++;
  }
  return output.toString();
}

/// 百分号解码（只在含 `%` 时尝试）：Dart 对非 ASCII 主机名保留百分号转义。
String decodeHostPercentEscapes(String hostname) {
  if (!hostname.contains('%')) return hostname;
  try {
    return Uri.decodeComponent(hostname);
  } catch (_) {
    // 非法转义（如落单的 %）：按原样比较，宁可多一次确认也不误豁免。
    return hostname;
  }
}

/// 归一化主机名：去空白、小写、去根点、百分号解码，再对非 ASCII 标签做
/// punycode 编码。全 ASCII 输入原样返回（仅大小写/根点归一化）。
String asciiHostname(String hostname) {
  final String decoded = decodeHostPercentEscapes(
    hostname.trim().toLowerCase(),
  );
  final String normalized = decoded.replaceFirst(RegExp(r'\.+$'), '');
  if (normalized.isEmpty) return normalized;

  bool converted = false;
  final List<String> labels = <String>[];
  for (final String label in normalized.split('.')) {
    final bool ascii = label.codeUnits.every((int unit) => unit < 0x80);
    if (ascii || label.isEmpty) {
      labels.add(label);
      continue;
    }
    labels.add('xn--${punycodeLabel(label)}');
    converted = true;
  }
  return converted ? labels.join('.') : normalized;
}

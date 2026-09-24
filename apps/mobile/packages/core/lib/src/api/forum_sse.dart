import 'dart:async';
import 'dart:convert';

/// One complete Server-Sent Events frame. An incomplete frame at EOF is lost
/// by design; the next connection must reconcile its REST snapshot.
class ForumSseFrame {
  const ForumSseFrame({required this.event, required this.data});

  final String event;
  final String data;
}

/// Decode bounded SSE frames. The UTF-8 decoder and line parser both keep
/// state across arbitrary network chunk boundaries, including split CRLF and
/// multibyte characters. Comments count as transport activity but emit no
/// application event.
Stream<ForumSseFrame> decodeForumSse(
  Stream<List<int>> bytes, {
  void Function()? onActivity,
}) async* {
  final parser = _ForumSseParser();
  await for (final chunk in utf8.decoder.bind(bytes)) {
    onActivity?.call();
    for (final frame in parser.add(chunk)) {
      yield frame;
    }
  }
}

class _ForumSseParser {
  static const int _maxLine = 16 * 1024;
  static const int _maxFrame = 64 * 1024;

  final StringBuffer _line = StringBuffer();
  final List<String> _data = [];
  String _event = '';
  int _frameLength = 0;
  bool _afterCr = false;

  Iterable<ForumSseFrame> add(String chunk) sync* {
    for (final rune in chunk.runes) {
      if (_afterCr) {
        _afterCr = false;
        if (rune == 10) continue;
      }
      if (rune == 13 || rune == 10) {
        final frame = _finishLine();
        if (frame != null) yield frame;
        if (rune == 13) _afterCr = true;
      } else {
        _line.writeCharCode(rune);
        if (_line.length > _maxLine) {
          throw const FormatException('SSE line exceeds limit');
        }
      }
    }
  }

  ForumSseFrame? _finishLine() {
    final line = _line.toString();
    _line.clear();
    if (line.isEmpty) {
      if (_data.isEmpty) {
        _event = '';
        _frameLength = 0;
        return null;
      }
      final frame = ForumSseFrame(
        event: _event.isEmpty ? 'message' : _event,
        data: _data.join('\n'),
      );
      _data.clear();
      _event = '';
      _frameLength = 0;
      return frame;
    }
    if (line.startsWith(':')) return null;
    final colon = line.indexOf(':');
    final field = colon < 0 ? line : line.substring(0, colon);
    var value = colon < 0 ? '' : line.substring(colon + 1);
    if (value.startsWith(' ')) value = value.substring(1);
    if (field == 'event') {
      _event = value;
      _frameLength += value.length;
    } else if (field == 'data') {
      _data.add(value);
      _frameLength += value.length + 1;
    }
    if (_frameLength > _maxFrame) {
      throw const FormatException('SSE frame exceeds limit');
    }
    return null;
  }
}

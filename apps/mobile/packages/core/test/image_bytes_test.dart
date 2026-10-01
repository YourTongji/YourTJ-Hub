import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

/// 构造 `RIFF....WEBP` 头（前 4 字节 RIFF，8..12 为 WEBP）。
List<int> _webpBytes() {
  return <int>[
    0x52, 0x49, 0x46, 0x46, // RIFF
    0x00, 0x00, 0x00, 0x00, // size placeholder
    0x57, 0x45, 0x42, 0x50, // WEBP
    0x00, 0x00, 0x00, 0x00,
  ];
}

void main() {
  group('imageExtensionForBytes', () {
    final cases = <String, (List<int>, String?)>{
      'jpeg': ([0xFF, 0xD8, 0xFF, 0xE0], '.jpg'),
      'png': ([0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A], '.png'),
      'gif87a': ('GIF87a'.codeUnits, '.gif'),
      'gif89a': ('GIF89a'.codeUnits, '.gif'),
      'webp': (_webpBytes(), '.webp'),
      'bmp': ([
        0x42,
        0x4D,
        0x3A,
        0x00,
        0x00,
        0x00, // file size
        0x00,
        0x00,
        0x00,
        0x00, // reserved1/2 (spec-fixed zero)
        0x36,
        0x00,
        0x00,
        0x00, // pixel data offset
      ], '.bmp'),
      'bm-prefixed plain text': ('BM ${'x' * 20}'.codeUnits, null),
      'unknown': ([0x00, 0x01, 0x02, 0x03], null),
      'empty': (<int>[], null),
      'plain text': ('hello world'.codeUnits, null),
      'riff but not webp': ([
        0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x41, 0x56, 0x49, 0x20,
      ], null),
    };
    cases.forEach((name, entry) {
      test('$name bytes -> ${entry.$2}', () {
        expect(imageExtensionForBytes(entry.$1), entry.$2);
      });
    });

    test('accepts Uint8List and truncates nothing at the magic boundary', () {
      final bytes = Uint8List.fromList([0xFF, 0xD8, 0xFF]);
      expect(imageExtensionForBytes(bytes), '.jpg');
    });
  });

  group('alignImageFileNameWithBytes', () {
    test('replaces a mismatched mixed-case extension, keeps basename', () {
      expect(
        alignImageFileNameWithBytes('scaled_IMG.PNG', [0xFF, 0xD8, 0xFF]),
        'scaled_IMG.jpg',
      );
    });

    test('extension-only replacement for .png name with jpeg bytes', () {
      expect(
        alignImageFileNameWithBytes('photo.png', [0xFF, 0xD8, 0xFF]),
        'photo.jpg',
      );
    });

    test('appends an extension when the name has none', () {
      expect(
        alignImageFileNameWithBytes('photo', [0xFF, 0xD8, 0xFF]),
        'photo.jpg',
      );
    });

    test('keeps an already-matching .jpeg name', () {
      expect(
        alignImageFileNameWithBytes('photo.jpeg', [0xFF, 0xD8, 0xFF]),
        'photo.jpeg',
      );
    });

    test('keeps unknown bytes and empty bytes unchanged', () {
      expect(alignImageFileNameWithBytes('photo.png', [0x00, 0x01]), 'photo.png');
      expect(alignImageFileNameWithBytes('scaled_IMG.PNG', <int>[]), 'scaled_IMG.PNG');
    });

    test('replaces only the last extension segment', () {
      expect(
        alignImageFileNameWithBytes('a.b.c', [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A]),
        'a.b.png',
      );
    });

    test('preserves directory prefixes with / and \\', () {
      expect(
        alignImageFileNameWithBytes('dir/sub/photo.PNG', [0xFF, 0xD8, 0xFF]),
        'dir/sub/photo.jpg',
      );
      expect(
        alignImageFileNameWithBytes(r'C:\tmp\photo', [
          0x42,
          0x4D,
          0x3A,
          0x00,
          0x00,
          0x00,
          0x00,
          0x00,
          0x00,
          0x00,
          0x36,
          0x00,
          0x00,
          0x00,
        ]),
        r'C:\tmp\photo.bmp',
      );
    });

    test('does not create a doubled suffix for a dotted directory name', () {
      expect(
        alignImageFileNameWithBytes('v1.2/photo', [0xFF, 0xD8, 0xFF]),
        'v1.2/photo.jpg',
      );
    });
  });
}

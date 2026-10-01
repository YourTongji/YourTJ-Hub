import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/images/image_sniff.dart';

final Uint8List _png = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAACklEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg==',
);

/// Minimal headers; the sniffer only reads magic bytes, not full images.
final Uint8List _jpeg = Uint8List.fromList([
  0xFF,
  0xD8,
  0xFF,
  0xE0,
  0x00,
  0x10,
  0x4A,
  0x46,
  0x49,
  0x46,
]);
final Uint8List _gif = Uint8List.fromList([
  0x47,
  0x49,
  0x46,
  0x38,
  0x39,
  0x61,
  0x01,
  0x00,
]);
final Uint8List _bmp = Uint8List.fromList([0x42, 0x4D, 0x3A, 0x00]);
final Uint8List _webp = Uint8List.fromList([
  0x52,
  0x49,
  0x46,
  0x46,
  0x1A,
  0x00,
  0x00,
  0x00,
  0x57,
  0x45,
  0x42,
  0x50,
]);

void main() {
  group('sniffImageFormat', () {
    test('recognizes each supported image family by magic bytes', () {
      expect(sniffImageFormat(_jpeg), 'jpeg');
      expect(sniffImageFormat(_png), 'png');
      expect(sniffImageFormat(_gif), 'gif');
      expect(sniffImageFormat(_bmp), 'bmp');
      expect(sniffImageFormat(_webp), 'webp');
    });

    test('returns null for empty or unrecognized bytes', () {
      expect(sniffImageFormat(const []), isNull);
      expect(sniffImageFormat(Uint8List.fromList([1, 2, 3])), isNull);
      expect(sniffImageFormat(Uint8List.fromList([0x89, 0x50])), isNull);
    });
  });

  group('normalizeImageFilename', () {
    test('renames a png filename carrying re-encoded jpeg bytes (issue #969)', () {
      expect(
        normalizeImageFilename('scaled_photo.png', _jpeg),
        'scaled_photo.jpg',
      );
    });

    test('keeps the filename when the extension already matches the bytes', () {
      expect(normalizeImageFilename('photo.png', _png), 'photo.png');
      expect(normalizeImageFilename('photo.jpeg', _jpeg), 'photo.jpeg');
      expect(normalizeImageFilename('photo.PNG', _png), 'photo.PNG');
    });

    test('renames mismatched extensions to the sniffed format', () {
      expect(normalizeImageFilename('archive.tar.png', _gif), 'archive.tar.gif');
      expect(normalizeImageFilename('shot.png', _webp), 'shot.webp');
      expect(normalizeImageFilename('scan.png', _bmp), 'scan.bmp');
    });

    test('leaves unfixable filenames untouched', () {
      // No extension to rewrite: the server rejects these as before.
      expect(normalizeImageFilename('noext', _jpeg), 'noext');
      expect(normalizeImageFilename('', _jpeg), '');
      expect(normalizeImageFilename('.png', _jpeg), '.png');
      // Unknown bytes: never guess an extension.
      expect(normalizeImageFilename('photo.png', Uint8List.fromList([1])), 'photo.png');
    });
  });
}

import 'dart:async';
import 'dart:convert';
import 'dart:ui' as ui;
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:ui_kit/ui_kit.dart';

final _gif = base64Decode(
  'R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7',
);
void main() {
  final binding = TestWidgetsFlutterBinding.ensureInitialized();
  tearDown(() {
    binding.imageCache.clear();
    binding.imageCache.clearLiveImages();
  });
  for (final public in [false, true]) {
    testWidgets('decoded image cache respects reusable=$public', (
      tester,
    ) async {
      var loads = 0;
      final provider = GfBytesImage(
        identity: 'a:zh:0',
        url: 'https://example.test/image.gif',
        isCurrent: () => true,
        load: () async {
          loads++;
          return GfMediaData(_gif, cacheIdentity: public ? 'fresh-v1' : null);
        },
      );
      final key = await provider.obtainKey(ImageConfiguration.empty);
      await tester.pumpWidget(
        Directionality(
          textDirection: TextDirection.ltr,
          child: Image(image: provider),
        ),
      );
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(tester.takeException(), isNull);
      final raw = tester.widget<RawImage>(find.byType(RawImage));
      expect(raw.image, isNotNull);
      expect(binding.imageCache.containsKey(key), public);
      final oldLoads = loads;
      await tester.pumpWidget(const SizedBox());
      await tester.pumpWidget(
        Directionality(
          textDirection: TextDirection.ltr,
          child: Image(image: provider),
        ),
      );
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(loads, greaterThan(oldLoads));
      expect(tester.takeException(), isNull);
    });
  }
  test('late image bytes cannot resolve after a clear boundary', () async {
    var active = true;
    final bytes = Completer<GfMediaData>();
    final provider = GfBytesImage(
      identity: 'old',
      url: 'https://example.test/image.gif',
      isCurrent: () => active,
      load: () => bytes.future,
    );
    final key = provider.obtainKey(ImageConfiguration.empty);

    active = false;
    bytes.complete(GfMediaData(_gif, cacheIdentity: 'fresh'));
    final invalidated = await key;
    expect(invalidated.bytes, isNull);
    expect(invalidated.retain, isFalse);
  });
  testWidgets('the image codec keeps every GIF frame', (tester) async {
    final gif = base64Decode(
      'R0lGODlhAQABAIAAAAAAAP///yH/C05FVFNDQVBFMi4wAwEAAAAh+QQAAQAAACwAAAAAAQABAAACAkQBACH5BAABAAAALAAAAAABAAEAAAICTAEAOw==',
    );
    final provider = GfBytesImage(
      identity: 'gif',
      url: 'https://example.test/animated.gif',
      isCurrent: () => true,
      load: () async => GfMediaData(gif),
    );
    final key = await provider.obtainKey(ImageConfiguration.empty);
    final frames = Completer<int>();
    final completer = provider.loadImage(key, (buffer, {getTargetSize}) async {
      final codec = await ui.instantiateImageCodecWithSize(
        buffer,
        getTargetSize: getTargetSize,
      );
      frames.complete(codec.frameCount);
      return codec;
    });
    final listener = ImageStreamListener((image, _) => image.dispose());
    completer.addListener(listener);
    await tester.runAsync(() => frames.future);
    expect(await frames.future, 2);
    completer.removeListener(listener);
  });
  testWidgets('SVG bytes render even when the URL extension lies', (
    tester,
  ) async {
    var loads = 0;
    final svg = Uint8List.fromList(
      utf8.encode(
        '<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10">'
        '<rect width="10" height="10" fill="#336699"/></svg>',
      ),
    );
    // Rendering is read back from the widget tree: a raster decode of SVG bytes
    // fails, so a painted frame with the vector's size proves the vector path.
    Future<ui.Image> render(GfBytesImage provider) async {
      await tester.pumpWidget(
        Directionality(
          textDirection: TextDirection.ltr,
          child: Image(image: provider),
        ),
      );
      ui.Image? image;
      // Real rasterization is asynchronous in a widget test: poll the painted
      // frame instead of betting on one fixed delay.
      for (var attempt = 0; attempt < 40 && image == null; attempt++) {
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 50));
        });
        await tester.pump();
        final Finder painted = find.byType(RawImage);
        if (painted.evaluate().isNotEmpty) {
          image = tester.widget<RawImage>(painted).image;
        }
      }
      expect(tester.takeException(), isNull);
      expect(image, isNotNull);
      return image!;
    }

    // A .png name with SVG bytes must still take the vector path.
    final image = await render(
      GfBytesImage(
        identity: 'vector',
        url: 'https://example.test/vector.png',
        isCurrent: () => true,
        load: () async {
          loads++;
          return GfMediaData(svg);
        },
      ),
    );
    expect(loads, 1);
    expect(image.width, 20);
    expect(image.height, 20);

    // Explicit cache targets are honoured and clamped to 4x.
    final clampedImage = await render(
      GfBytesImage(
        identity: 'vector-clamped',
        url: 'https://example.test/vector.png',
        width: 10000,
        height: 10000,
        isCurrent: () => true,
        load: () async {
          loads++;
          return GfMediaData(svg);
        },
      ),
    );
    expect(clampedImage.width, 40);
    expect(clampedImage.height, 40);
    expect(loads, 2);
  });

  test('isSvgDocument only accepts document-like vector heads', () {
    const svg = '<svg xmlns="http://www.w3.org/2000/svg"></svg>';
    expect(isSvgDocument(Uint8List.fromList(utf8.encode(svg))), isTrue);
    expect(
      isSvgDocument(Uint8List.fromList(utf8.encode('  \n\t$svg'))),
      isTrue,
    );
    expect(
      isSvgDocument(
        Uint8List.fromList(utf8.encode('<?xml version="1.0"?>$svg')),
      ),
      isTrue,
    );
    expect(
      isSvgDocument(Uint8List.fromList(utf8.encode('<SVG viewBox="0 0 1 1"/>'))),
      isTrue,
    );
    expect(
      isSvgDocument(
        Uint8List.fromList(<int>[0xEF, 0xBB, 0xBF, ...utf8.encode(svg)]),
      ),
      isTrue,
    );
    // Bearers that must never be mistaken for vectors.
    expect(isSvgDocument(_gif), isFalse);
    expect(isSvgDocument(Uint8List(0)), isFalse);
    expect(
      isSvgDocument(
        Uint8List.fromList(utf8.encode('<!DOCTYPE html><html></html>')),
      ),
      isFalse,
    );
  });
  testWidgets(
    'network SVG uses the same byte loader without a second parsed cache',
    (tester) async {
      svg.cache.clear();
      var loads = 0;
      ImageProvider<Object> factory(
        String url, {
        int? width,
        int? height,
        ResizeImagePolicy policy = ResizeImagePolicy.exact,
        Set<String>? allowedOrigins,
      }) => GfBytesImage(
        identity: 'svg',
        url: url,
        isCurrent: () => true,
        load: () async {
          loads++;
          return GfMediaData(
            Uint8List.fromList(
              utf8.encode(
                '<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="blue"/></svg>',
              ),
            ),
          );
        },
      );
      await tester.pumpWidget(
        MaterialApp(
          home: GfMediaScope(
            identity: 'svg',
            factory: factory,
            child: const GfNetworkSvg(
              'https://example.test/badge.svg',
              fallback: SizedBox(),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 40));
      });
      await tester.pump();
      expect(loads, 1);
      final picture = tester.widget<SvgPicture>(find.byType(SvgPicture));
      expect(picture.bytesLoader, isA<SvgBytesLoader>());
      expect(svg.cache.count, 0);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );
  testWidgets(
    'avatar display and prefetch use the host and matching dimensions',
    (tester) async {
      final providers = <GfBytesImage>[];
      ImageProvider<Object> factory(
        String url, {
        int? width,
        int? height,
        ResizeImagePolicy policy = ResizeImagePolicy.exact,
        Set<String>? allowedOrigins,
      }) {
        final provider = GfBytesImage(
          identity: 'host',
          url: url,
          width: width,
          height: height,
          policy: policy,
          isCurrent: () => true,
          load: () async => GfMediaData(_gif, cacheIdentity: 'fresh'),
        );
        providers.add(provider);
        return provider;
      }

      await tester.pumpWidget(
        MaterialApp(
          home: GfMediaScope(
            identity: 'host',
            factory: factory,
            child: Builder(
              builder: (context) {
                GfAvatar.imageProviderFor(
                  'https://example.test/avatar.gif',
                  context: context,
                  size: 36,
                  devicePixelRatio: MediaQuery.devicePixelRatioOf(context),
                );
                return const GfAvatar(
                  src: 'https://example.test/avatar.gif',
                  size: 36,
                );
              },
            ),
          ),
        ),
      );
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(providers, hasLength(2));
      expect(providers.first, providers.last);
      expect(providers.first.policy, ResizeImagePolicy.fit);
      expect(tester.takeException(), isNull);
    },
  );
}

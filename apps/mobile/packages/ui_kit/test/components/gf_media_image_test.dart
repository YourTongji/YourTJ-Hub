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

import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

final _gif = base64Decode(
  'R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7',
);

ImageProvider<Object> _factory({
  required Future<GfMediaData> Function() load,
  int? width,
  int? height,
}) => GfBytesImage(
  identity: 'host',
  url: 'https://example.test/image.gif',
  width: width,
  height: height,
  isCurrent: () => true,
  load: load,
);

void main() {
  testWidgets('loading failures reach the host fallback and retry reloads', (
    tester,
  ) async {
    var loads = 0;
    Object? reported;
    VoidCallback? retry;
    await tester.pumpWidget(
      MaterialApp(
        home: GfMediaScope(
          identity: 'host',
          factory:
              (
                url, {
                width,
                height,
                policy = ResizeImagePolicy.exact,
                allowedOrigins,
              }) => _factory(
                width: width,
                height: height,
                load: () async {
                  loads++;
                  if (loads == 1) {
                    throw StateError('Image request failed (403)');
                  }
                  return GfMediaData(_gif, cacheIdentity: 'fresh');
                },
              ),
          imageErrorBuilder: (context, error, retryCallback, url) {
            reported = error;
            retry = retryCallback;
            return Text('failed:$url');
          },
          child: const GfNetworkImage('https://example.test/image.gif'),
        ),
      ),
    );
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump();
    expect(reported, isA<StateError>());
    expect(find.text('failed:https://example.test/image.gif'), findsOneWidget);

    final before = loads;
    retry!();
    await tester.pump();
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump();
    expect(loads, greaterThan(before));
    expect(find.byType(RawImage), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('a caller errorBuilder still outranks the host fallback', (
    tester,
  ) async {
    final fallbacks = <Object>[];
    await tester.pumpWidget(
      MaterialApp(
        home: GfMediaScope(
          identity: 'host',
          factory:
              (
                url, {
                width,
                height,
                policy = ResizeImagePolicy.exact,
                allowedOrigins,
              }) => _factory(
                load: () async => throw StateError('Image request failed (500)'),
              ),
          imageErrorBuilder: (context, error, retry, url) {
            fallbacks.add(error);
            return const Text('host');
          },
          child: GfNetworkImage(
            'https://example.test/image.gif',
            errorBuilder: (_, _, _) => const Text('caller'),
          ),
        ),
      ),
    );
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump();
    expect(find.text('caller'), findsOneWidget);
    expect(find.text('host'), findsNothing);
    expect(fallbacks, isEmpty);
  });

  testWidgets('onImageError observes a failure without owning the fallback', (
    tester,
  ) async {
    final observed = <Object>[];
    await tester.pumpWidget(
      MaterialApp(
        home: GfMediaScope(
          identity: 'host',
          factory:
              (
                url, {
                width,
                height,
                policy = ResizeImagePolicy.exact,
                allowedOrigins,
              }) => _factory(
                load: () async => throw StateError('Image request failed (500)'),
              ),
          imageErrorBuilder: (context, error, retry, url) =>
              const Text('host'),
          child: GfNetworkImage(
            'https://example.test/image.gif',
            onImageError: observed.add,
          ),
        ),
      ),
    );
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump();
    expect(observed.single, isA<StateError>());
    expect(find.text('host'), findsOneWidget);
  });

  testWidgets('without a host hook the existing silent failure stays', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: GfMediaScope(
          identity: 'host',
          factory:
              (
                url, {
                width,
                height,
                policy = ResizeImagePolicy.exact,
                allowedOrigins,
              }) => _factory(
                load: () async => throw StateError('Image request failed (500)'),
              ),
          child: const GfNetworkImage('https://example.test/image.gif'),
        ),
      ),
    );
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump();
    // Without any fallback Flutter paints its own debug error widget, so the
    // guarantee under test is narrower: no actionable control appears.
    expect(find.byType(TextButton), findsNothing);
    final Object? reported = tester.takeException();
    expect(reported, isA<StateError>());
    expect((reported! as StateError).message, 'Image request failed (500)');
  });
}

import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

final _gif = base64Decode(
  'R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7',
);

void main() {
  final binding = TestWidgetsFlutterBinding.ensureInitialized();
  tearDown(() {
    binding.imageCache.clear();
    binding.imageCache.clearLiveImages();
  });

  testWidgets(
    'a fresh avatar does not flash its placeholder on the next page',
    (tester) async {
      var loads = 0;
      FutureOr<GfMediaData> load() {
        final data = GfMediaData(_gif, cacheIdentity: 'avatar-fresh');
        if (loads > 0) return data;
        loads++;
        return Future<GfMediaData>.value(data);
      }

      Widget page(Widget child) => gfApp(
        GfMediaScope(
          identity: 'site',
          factory:
              (
                url, {
                width,
                height,
                policy = ResizeImagePolicy.exact,
                allowedOrigins,
              }) => GfBytesImage(
                identity: 'site',
                url: url,
                width: width,
                height: height,
                policy: policy,
                isCurrent: () => true,
                load: load,
              ),
          child: child,
        ),
      );
      const avatar = GfAvatar(
        src: 'https://example.test/file/img/avatars/1/a.gif',
        size: 40,
      );

      await tester.pumpWidget(page(avatar));
      expect(find.byType(GfSymbol), findsOneWidget);

      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 50)),
      );
      await tester.pump();
      expect(find.byType(GfSymbol), findsNothing);
      expect(tester.takeException(), isNull);
      expect(loads, 1);

      await tester.pumpWidget(page(const SizedBox.shrink()));
      await tester.pumpWidget(page(avatar));
      expect(find.byType(GfSymbol), findsNothing);
      expect(tester.widget<RawImage>(find.byType(RawImage)).image, isNotNull);
    },
  );
}

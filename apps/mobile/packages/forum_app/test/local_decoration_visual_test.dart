import 'dart:io';
import 'dart:ui' as ui;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/decoration_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'decoration_page_test.dart' show Decorations;
import 'test_font_helpers.dart' show loadTestFonts;

void main() {
  testWidgets(
    'administrator-configured sample art renders in the real decoration page',
    (tester) async {
      await loadTestFonts(tester);
      final png = File(
        '../../../../apps/gooseforum/resource/static/decorations/tea-garden-v1.png',
      ).readAsBytesSync();
      tester.view.physicalSize = const Size(860, 1800);
      tester.view.devicePixelRatio = 2;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      const catalog = [
        DecorationStyle(
          code: 'frame-garden',
          name: '花园示例框',
          kind: 'avatarFrame',
          lightColors: ['#B45309', '#D97706'],
          darkColors: ['#FCD34D', '#FDBA74'],
          assetUrl: '/static/decorations/tea-garden-v1.png',
        ),
        DecorationStyle(
          code: 'frame-sample',
          name: '环形示例框',
          kind: 'avatarFrame',
          lightColors: ['#059669', '#0D9488'],
          darkColors: ['#34D399', '#2DD4BF'],
        ),
        DecorationStyle(
          code: 'name-garden',
          name: '绿色昵称示例',
          kind: 'nicknameStyle',
          lightColors: ['#065F46', '#115E59'],
          darkColors: ['#6EE7B7', '#99F6E4'],
        ),
        DecorationStyle(
          code: 'name-sky',
          name: '蓝色昵称示例',
          kind: 'nicknameStyle',
          lightColors: ['#1E40AF', '#3730A3'],
          darkColors: ['#BFDBFE', '#DDD6FE'],
        ),
      ];
      for (final brightness in [Brightness.light, Brightness.dark]) {
        final repo = Decorations()
          ..state = const DecorationState(
            catalog: catalog,
            ownedCodes: ['frame-garden', 'name-garden'],
            avatarFrame: 'frame-garden',
            nicknameStyle: 'name-garden',
          );
        final container = ProviderContainer(
          overrides: [userRepositoryProvider.overrideWithValue(repo)],
        );
        final theme = gfThemeData(brightness);
        final key = GlobalKey();
        final actual = theme.copyWith(
          textTheme: theme.textTheme.apply(fontFamily: 'NotoSansCJK'),
          primaryTextTheme: theme.primaryTextTheme.apply(
            fontFamily: 'NotoSansCJK',
          ),
          listTileTheme: theme.listTileTheme.copyWith(
            titleTextStyle: theme.listTileTheme.titleTextStyle!.copyWith(
              fontFamily: 'NotoSansCJK',
            ),
            subtitleTextStyle: theme.listTileTheme.subtitleTextStyle!.copyWith(
              fontFamily: 'NotoSansCJK',
            ),
          ),
        );
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              key: ValueKey(brightness),
              theme: actual,
              locale: const Locale('zh'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: GfMediaScope(
                identity: brightness,
                factory:
                    (
                      url, {
                      width,
                      height,
                      policy = ResizeImagePolicy.exact,
                      allowedOrigins,
                    }) {
                      expect(
                        Uri.parse(url).path,
                        '/static/decorations/tea-garden-v1.png',
                      );
                      return ResizeImage(
                        MemoryImage(png),
                        width: width,
                        height: height,
                        policy: policy,
                      );
                    },
                child: RepaintBoundary(
                  key: key,
                  child: const DecorationPage(name: '同济校园 · 茶铃'),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.runAsync(
          () => Future<void>.delayed(const Duration(milliseconds: 60)),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        expect(tester.widget<RawImage>(find.byType(RawImage)).image, isNotNull);
        final image = (await tester.runAsync(
          () =>
              (key.currentContext!.findRenderObject() as RenderRepaintBoundary)
                  .toImage(pixelRatio: 2),
        ))!;
        final bytes = (await tester.runAsync(
          () => image.toByteData(format: ui.ImageByteFormat.png),
        ))!;
        await tester.runAsync(
          () => File(
            '/tmp/yourtj-1156-mobile-${brightness.name}.png',
          ).writeAsBytes(bytes.buffer.asUint8List()),
        );
        image.dispose();
        container.dispose();
      }
    },
  );
}

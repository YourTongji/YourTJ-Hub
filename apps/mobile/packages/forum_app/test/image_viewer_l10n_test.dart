import 'dart:async';
import 'dart:ui' show Tristate;
import 'package:forum_app/src/widgets/stickers/sticker_preview.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/markdown_view.dart';

void main() {
  for (final language in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets(
        'markdown viewer localizes controls and positions $language $brightness',
        (tester) async {
          final semantics = tester.ensureSemantics();
          try {
            final provider = MemoryImage(
              img.encodePng(img.Image(width: 100, height: 160)),
            );
            await tester.pumpWidget(
              ProviderScope(
                child: GfMediaScope(
                  identity: provider,
                  factory:
                      (
                        _, {
                        int? width,
                        int? height,
                        Set<String>? allowedOrigins,
                        ResizeImagePolicy policy = ResizeImagePolicy.exact,
                      }) => provider,
                  child: MaterialApp(
                    theme: gfThemeData(brightness),
                    locale: Locale(language),
                    localizationsDelegates:
                        AppLocalizations.localizationsDelegates,
                    supportedLocales: AppLocalizations.supportedLocales,
                    home: const Scaffold(
                      body: GfMarkdownView(
                        data:
                            '![one](https://dev.yourtj.de/file/img/a.png)\n\n![two](https://dev.yourtj.de/file/img/b.png)',
                      ),
                    ),
                  ),
                ),
              ),
            );
            await tester.runAsync(
              () => precacheImage(
                provider,
                tester.element(find.byType(GfMarkdownView)),
              ),
            );
            await tester.pumpAndSettle();
            await tester.tap(find.byType(Image).first);
            await tester.pumpAndSettle();
            expect(find.byType(GfImageViewer), findsOneWidget);
            final l = AppLocalizations.of(
              tester.element(find.byType(GfImageViewer)),
            );
            expect(find.byTooltip(l.imageViewerOriginalSize), findsOneWidget);
            expect(
              find.bySemanticsLabel(l.imageViewerThumbnails),
              findsOneWidget,
            );
            expect(
              find.bySemanticsLabel(l.imageViewPosition(1, 2)),
              findsOneWidget,
            );
            if (language == 'zh') {
              expect(find.byTooltip('原始尺寸'), findsOneWidget);
              expect(find.bySemanticsLabel('图片缩略图'), findsOneWidget);
              expect(find.byTooltip('Original size'), findsNothing);
            }
            await tester.tap(find.byTooltip(l.imageViewerOriginalSize));
            await tester.pumpAndSettle();
            expect(find.byTooltip(l.imageViewerFitPreview), findsOneWidget);
            final second = find.byKey(const Key('gf-image-viewer-thumbnail-1'));
            await tester.tap(second);
            await tester.pumpAndSettle();
            expect(find.text('2 / 2'), findsOneWidget);
            expect(
              tester
                  .getSemantics(
                    find.byKey(
                      const Key('gf-image-viewer-thumbnail-semantic-1'),
                    ),
                  )
                  .flagsCollection
                  .isSelected,
              Tristate.isTrue,
            );
            expect(
              tester
                  .getSemantics(
                    find.byKey(
                      const Key('gf-image-viewer-thumbnail-semantic-0'),
                    ),
                  )
                  .flagsCollection
                  .isSelected,
              Tristate.isFalse,
            );
            expect(find.byTooltip(l.imageViewerOriginalSize), findsOneWidget);
            await tester.tap(
              find.byTooltip(
                MaterialLocalizations.of(
                  tester.element(find.byType(GfImageViewer)),
                ).closeButtonTooltip,
              ),
            );
            await tester.pumpAndSettle();
            expect(find.byType(GfImageViewer), findsNothing);
            unawaited(
              showStickerPreview(
                tester.element(find.byType(GfMarkdownView)),
                'https://dev.yourtj.de/file/img/a.png',
              ),
            );
            await tester.pumpAndSettle();
            expect(find.byTooltip(l.imageViewerOriginalSize), findsOneWidget);
            expect(
              find.bySemanticsLabel(l.imageViewerThumbnails),
              findsNothing,
            );
            await tester.tap(
              find.byTooltip(
                MaterialLocalizations.of(
                  tester.element(find.byType(GfImageViewer)),
                ).closeButtonTooltip,
              ),
            );
            await tester.pumpAndSettle();
            await tester.pumpWidget(const SizedBox.shrink());
            await tester.pump(const Duration(milliseconds: 600));
            await tester.pumpAndSettle();
          } finally {
            semantics.dispose();
          }
        },
      );
    }
  }
}

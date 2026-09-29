import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfTopicRow', () {
    const GfTopicCategory category = GfTopicCategory(
      name: '校园生活',
      color: Color(0xFF00BC7D),
    );

    Widget buildRow({
      VoidCallback? onTap,
      bool pinned = false,
      bool unseen = false,
      String title = '同济大学樱花大道拍照攻略',
      Brightness brightness = Brightness.light,
    }) {
      return gfApp(
        GfTopicRow(
          pinnedLabel: 'pinned',
          title: title,
          description: '三月末的樱花大道,适合清晨人少时去…',
          categories: const <GfTopicCategory>[category],
          participantAvatarUrls: const <String>[
            'https://example.com/a.png',
            'https://example.com/b.png',
          ],
          activityText: '3 小时前',
          replyCount: 42,
          onTap: onTap,
          pinned: pinned,
          unseen: unseen,
        ),
        brightness: brightness,
      );
    }

    testWidgets('renders title, description, chip, meta in both themes', (
      tester,
    ) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(buildRow(brightness: brightness));
        expect(find.text('同济大学樱花大道拍照攻略'), findsOneWidget);
        expect(find.text('三月末的樱花大道,适合清晨人少时去…'), findsOneWidget);
        expect(find.text('校园生活'), findsOneWidget);
        expect(find.text('3 小时前'), findsOneWidget);
        expect(find.text('42'), findsOneWidget);
        expect(
          find.byWidgetPredicate(
            (widget) => widget is GfSymbol && widget.name == 'message-circle',
          ),
          findsOneWidget,
        );
      });
    });

    testWidgets('hides an empty title while keeping description and meta', (
      tester,
    ) async {
      await tester.pumpWidget(buildRow(title: ''));
      expect(find.text('同济大学樱花大道拍照攻略'), findsNothing);
      // 空标题不占位：不渲染空 Text（标题字号 15）。
      expect(
        find.byWidgetPredicate(
          (Widget w) => w is Text && w.data == '' && w.style?.fontSize == 15,
        ),
        findsNothing,
      );
      expect(find.text('三月末的樱花大道,适合清晨人少时去…'), findsOneWidget);
      expect(find.text('校园生活'), findsOneWidget);
      expect(find.text('3 小时前'), findsOneWidget);
      expect(find.text('42'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('shows pin mark and unseen dot when flagged', (tester) async {
      await tester.pumpWidget(buildRow(pinned: true, unseen: true));
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'pin-filled',
        ),
        findsOneWidget,
      );
      expect(
        find.byWidgetPredicate(
          (Widget w) =>
              w is Container &&
              w.decoration is BoxDecoration &&
              (w.decoration as BoxDecoration).shape == BoxShape.circle &&
              (w.decoration as BoxDecoration).color == GfColors.light.primary,
        ),
        findsOneWidget,
      );
    });

    testWidgets('shows the hot badge when flagged', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfTopicRow(
            pinnedLabel: 'pinned',
            title: 'hot topic',
            description: '',
            categories: const <GfTopicCategory>[],
            participantAvatarUrls: const <String>[],
            activityText: '1 小时前',
            replyCount: 999,
            hot: true,
          ),
        ),
      );
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'flame',
        ),
        findsOneWidget,
      );
      expect(find.text('hot'), findsOneWidget);
    });

    testWidgets(
      'compact row preserves category targets and separate gestures',
      (tester) async {
        var rowTaps = 0;
        var categoryTaps = 0;
        await tester.pumpWidget(
          gfApp(
            SingleChildScrollView(
              child: SizedBox(
                width: 402,
                child: GfTopicRow(
                  pinnedLabel: 'pinned',
                  title: '校园短标题',
                  description: '摘要仍然清晰可读',
                  categories: [
                    GfTopicCategory(
                      name: '校园生活',
                      color: Colors.green,
                      onTap: () => categoryTaps++,
                    ),
                  ],
                  participantAvatarUrls: const ['', ''],
                  activityText: '3 小时前',
                  replyCount: 42,
                  onTap: () => rowTaps++,
                ),
              ),
            ),
          ),
        );
        expect(
          tester.getSize(find.byType(GfTopicRow)).height,
          lessThanOrEqualTo(104),
        );
        final chip = find.byType(GfChip);
        expect(tester.getSize(chip).height, greaterThanOrEqualTo(44));
        expect(tester.getSize(chip).width, greaterThanOrEqualTo(44));
        await tester.tap(find.text('校园生活'));
        expect(categoryTaps, 1);
        expect(rowTaps, 0);
        await tester.tap(find.text('校园短标题'));
        expect(rowTaps, 1);
      },
    );

    testWidgets('divider does not add a blank footer to each row', (
      tester,
    ) async {
      Future<double> height(bool divider) async {
        await tester.pumpWidget(
          gfApp(
            SingleChildScrollView(
              child: SizedBox(
                width: 402,
                child: GfTopicRow(
                  pinnedLabel: 'pinned',
                  title: 'A readable topic',
                  description: 'A short excerpt',
                  categories: const [],
                  participantAvatarUrls: const [],
                  activityText: '1 h',
                  replyCount: 2,
                  showDivider: divider,
                ),
              ),
            ),
          ),
        );
        return tester.getSize(find.byType(GfTopicRow)).height;
      }

      final withDivider = await height(true);
      expect(await height(false), withDivider);
    });

    testWidgets('long metadata remains usable at narrow widths and large text', (
      tester,
    ) async {
      for (final width in [320.0, 390.0, 600.0, 768.0, 1024.0]) {
        tester.view.physicalSize = Size(width, 1200);
        tester.view.devicePixelRatio = 1;
        for (final scale in [1.0, 2.0]) {
          await tester.pumpWidget(
            MaterialApp(
              theme: gfThemeData(Brightness.dark),
              home: MediaQuery(
                data: MediaQueryData(
                  size: Size(width, 1200),
                  textScaler: TextScaler.linear(scale),
                ),
                child: Scaffold(
                  body: SingleChildScrollView(
                    child: GfTopicRow(
                      pinnedLabel: 'pinned',
                      title:
                          'Ein sehr langer Titel für die gemeinsame Diskussion auf dem Campus',
                      description:
                          'A longer readable excerpt should not overflow the reading column.',
                      categories: [
                        GfTopicCategory(
                          name: 'Studium und Campusleben',
                          color: Colors.green,
                          onTap: () {},
                        ),
                        GfTopicCategory(
                          name: '第二个完整分类名称',
                          color: Colors.blue,
                          onTap: () {},
                        ),
                      ],
                      participantAvatarUrls: const ['', '', '', ''],
                      activityText: 'vor mehreren Monaten',
                      replyCount: 123456,
                      contentType: GfTopicContentType.question,
                      contentTypeLabel: 'Frage',
                      pinned: true,
                      unseen: true,
                      hot: true,
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.pump();
          expect(tester.takeException(), isNull, reason: '$width @ $scale');
          expect(
            tester.getSize(find.byType(GfChip).first).height,
            greaterThanOrEqualTo(44),
          );
        }
      }
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
    });

    testWidgets('type badge text maintains contrast in both themes', (
      tester,
    ) async {
      for (final brightness in Brightness.values) {
        for (final type in GfTopicContentType.values) {
          await tester.pumpWidget(
            gfApp(
              GfTopicRow(
                pinnedLabel: 'pinned',
                title: 'Topic',
                description: '',
                categories: const [],
                participantAvatarUrls: const [],
                activityText: '1 h',
                replyCount: 2,
                contentType: type,
                contentTypeLabel: 'Type',
              ),
              brightness: brightness,
            ),
          );
          await tester.pumpAndSettle();
          final color = tester.widget<Text>(find.text('Type')).style!.color!;
          final background = Color.alphaBlend(
            color.withValues(alpha: 0.12),
            GfColors.forBrightness(brightness).base100,
          );
          final light = color.computeLuminance();
          final dark = background.computeLuminance();
          final contrast = light > dark
              ? (light + 0.05) / (dark + 0.05)
              : (dark + 0.05) / (light + 0.05);
          expect(
            contrast,
            greaterThanOrEqualTo(4.5),
            reason: '$type $brightness',
          );
        }
      }
    });

    testWidgets('row tap fires onTap', (tester) async {
      int taps = 0;
      await tester.pumpWidget(buildRow(onTap: () => taps++));
      await tester.tap(find.text('同济大学樱花大道拍照攻略'));
      expect(taps, 1);
    });
  });

  group('GfTopicCard', () {
    testWidgets('renders web-aligned author, content and metrics', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 360,
            child: GfTopicCard(
              title: '校园卡片话题',
              description: '卡片摘要内容',
              authorName: 'Alice',
              authorAvatarUrl: '',
              categories: const <GfTopicCategory>[
                GfTopicCategory(name: '校园生活', color: Color(0xFF00BC7D)),
              ],
              imageUrls: const <String>[],
              activityText: '3 小时前',
              replyCount: 42,
              viewCount: 128,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.text('Alice'), findsOneWidget);
      expect(find.text('校园卡片话题'), findsOneWidget);
      expect(find.text('卡片摘要内容'), findsOneWidget);
      expect(find.text('校园生活'), findsOneWidget);
      expect(find.text('42'), findsOneWidget);
      expect(find.text('128'), findsOneWidget);
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'message-circle',
        ),
        findsOneWidget,
      );
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'eye',
        ),
        findsOneWidget,
      );
    });
  });
}

import 'dart:ui' show SemanticsAction;

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

const _destinations = <String, String>{
  'github': 'https://github.com/alice',
  'twitter': 'https://x.com/alice',
  'linkedIn': 'https://www.linkedin.com/in/alice',
  'weibo': 'https://weibo.com/alice',
  'bilibili': 'https://space.bilibili.com/123',
  'zhihu': 'https://www.zhihu.com/people/alice',
  'mastodon': 'https://social.example/@alice',
};

class _MemoryTokenStorage implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _SocialProfiles extends PageRepository {
  _SocialProfiles()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _MemoryTokenStorage(),
          baseUrl: 'https://forum.example',
        ),
      );

  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async {
    final json = userProfilePayloadJson();
    final props = json['props'] as Map<String, dynamic>;
    final user = props['user'] as Map<String, dynamic>;
    user['websiteName'] = 'Personal site';
    user['website'] = 'https://example.test/about';
    user['externalInformation'] = {
      for (final entry in _destinations.entries)
        entry.key: {'link': entry.value},
      'unsafe': {'link': 'javascript:alert(1)'},
    };
    return parsePayload(json);
  }
}

Future<void> _pumpProfile(
  WidgetTester tester, {
  Brightness brightness = Brightness.light,
  double scale = 1,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        currentUserProvider.overrideWith((_) async => null),
        pageRepositoryProvider.overrideWithValue(_SocialProfiles()),
      ],
      child: MaterialApp(
        theme: gfThemeData(brightness),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
        home: const ProfilePage(userId: 1),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  for (final brightness in Brightness.values) {
    testWidgets(
      'profile links use named icon targets on a narrow scaled ${brightness.name} screen',
      (tester) async {
        tester.view.physicalSize = const Size(320, 900);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final semantics = tester.ensureSemantics();
        try {
          await _pumpProfile(tester, brightness: brightness, scale: 2);
          for (final label in [
            'Personal site',
            'GitHub',
            'X / Twitter',
            'LinkedIn',
            'Weibo',
            'Bilibili',
            'Zhihu',
            'mastodon',
          ]) {
            final target = find.byTooltip(label);
            expect(target, findsOneWidget);
            expect(find.text(label), findsNothing);
            await tester.ensureVisible(target);
            await tester.pumpAndSettle();
            final bounds = tester.getRect(target);
            expect(bounds.width, greaterThanOrEqualTo(44));
            expect(bounds.height, greaterThanOrEqualTo(44));
            expect(bounds.left, greaterThanOrEqualTo(0));
            expect(bounds.right, lessThanOrEqualTo(320));
            final node = tester.getSemantics(target);
            expect(node.label, label);
            expect(node.flagsCollection.isButton, isTrue);
            expect(
              node.getSemanticsData().hasAction(SemanticsAction.tap),
              isTrue,
            );
          }
          expect(find.byType(GfSocialIcon), findsNWidgets(8));
          expect(find.byTooltip('unsafe'), findsNothing);
          expect(tester.takeException(), isNull);
        } finally {
          semantics.dispose();
        }
      },
    );
  }

  testWidgets('social icons retain their external browser destinations', (
    tester,
  ) async {
    final opened = <Map<Object?, Object?>>[];
    const channel = MethodChannel('plugins.flutter.io/url_launcher');
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
      call,
    ) async {
      if (call.method == 'launch') {
        opened.add(Map<Object?, Object?>.from(call.arguments as Map));
      }
      return true;
    });
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        channel,
        null,
      ),
    );
    await _pumpProfile(tester);
    for (final label in ['GitHub', 'X / Twitter']) {
      await tester.ensureVisible(find.byTooltip(label));
      await tester.tap(find.byTooltip(label));
      await tester.pumpAndSettle();
    }
    expect(opened.map((call) => call['url']), [
      _destinations['github'],
      _destinations['twitter'],
    ]);
    for (final call in opened) {
      expect(call['useSafariVC'], isFalse);
      expect(call['useWebView'], isFalse);
    }
    expect(find.byType(ProfilePage), findsOneWidget);
  });
}

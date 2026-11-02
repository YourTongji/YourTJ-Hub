import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/notifications/unread_display.dart';
import 'package:forum_app/src/notifications/unread_display_settings.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/fixtures/campus_fixtures.dart' show CampusMemoryTokenStorage;

class _Surface extends ConsumerWidget {
  const _Surface(this.status);
  final UnreadStatusPayload status;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final numbers =
        ref.watch(unreadDisplayProvider).style == UnreadBadgeStyle.number;
    final l = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(title: GfText(l.unreadStyleTitle)),
      body: ListView(
        children: [
          const UnreadDisplaySettings(),
          const GfDivider(),
          GfConversationRow(
            avatarUrl: '',
            name: 'Peer',
            lastMessage: 'Hello',
            time: '',
            unreadCount: status.messageCount!,
            unreadLabel: l.notificationsUnread,
            showUnreadCount: numbers,
          ),
        ],
      ),
      bottomNavigationBar: GfBottomNavigation(
        currentIndex: 0,
        showLabels: false,
        onSelected: (_) {},
        items: [
          GfBottomNavigationItem(
            label: l.navHome,
            symbol: 'house',
            selectedSymbol: 'house',
          ),
          GfBottomNavigationItem(
            label: l.navCampus,
            symbol: 'graduation-cap',
            selectedSymbol: 'graduation-cap',
          ),
          GfBottomNavigationItem(
            label: l.notificationsTitle,
            symbol: 'bell',
            selectedSymbol: 'bell',
            badge: status.notifications,
            badgeCount: numbers ? status.notificationCount : null,
            badgeSemanticLabel: l.notificationsUnread,
          ),
          GfBottomNavigationItem(
            label: l.messagesTitle,
            symbol: 'message-circle',
            selectedSymbol: 'message-circle',
            badge: status.messages,
            badgeCount: numbers ? status.messageCount : null,
            badgeSemanticLabel: l.notificationsUnread,
          ),
        ],
      ),
    );
  }
}

/// Real HTTP client and native shared controls; only loopback synthetic data.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  testWidgets(
    'account unread style changes every native indicator without changing unread facts',
    (tester) async {
      AppConfig.validate();
      expect(AppConfig.isDev, isTrue);
      final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      var style = 'number', writes = 0, countReads = 0;
      final unexpected = <String>[];
      server.listen((request) async {
        if (request.headers.value(HttpHeaders.authorizationHeader) !=
            'Bearer synthetic-owner') {
          unexpected.add('wrong owner');
        }
        request.response.headers.contentType = ContentType.json;
        Object? result;
        if (request.uri.path == '/api/forum/unread-status' &&
            request.method == 'GET') {
          countReads++;
          result = {
            'notifications': true,
            'messages': true,
            'notificationCount': 149,
            'messageCount': 12,
            'badgeStyle': style,
          };
        } else if (request.uri.path == '/api/forum/notification-preferences' &&
            request.method == 'GET') {
          result = {'badgeStyle': style};
        } else if (request.uri.path == '/api/forum/notification-preferences' &&
            request.method == 'PUT') {
          final body =
              jsonDecode(await utf8.decoder.bind(request).join())
                  as Map<String, dynamic>;
          if (body.keys.length != 1 ||
              !['number', 'dot'].contains(body['badgeStyle'])) {
            unexpected.add('invalid preference body');
          }
          style = body['badgeStyle'] as String;
          writes++;
          result = {'badgeStyle': style};
        } else {
          unexpected.add('${request.method} ${request.uri.path}');
          request.response.statusCode = 404;
        }
        request.response.write(jsonEncode({'code': 0, 'result': result}));
        await request.response.close();
      });
      try {
        for (final language in ['zh', 'en', 'ja', 'de']) {
          for (final brightness in Brightness.values) {
            style = 'number';
            final token = CampusMemoryTokenStorage()..value = 'synthetic-owner';
            final dio = Dio();
            final client = GfApiClient(
              dio: dio,
              tokenStorage: token,
              baseUrl: 'http://127.0.0.1:${server.port}',
            );
            final repository = await NotificationRepository(
              client,
            ).forSession();
            final status = await repository.getUnreadStatus();
            final container = ProviderContainer(
              overrides: [
                apiClientProvider.overrideWithValue(client),
                tokenStorageProvider.overrideWithValue(token),
              ],
            );
            final display = container.read(unreadDisplayProvider.notifier);
            display.acceptRemote(
              UnreadBadgeStyle.parse(status.badgeStyle),
              expectedRevision: display.revision,
            );
            await tester.pumpWidget(
              UncontrolledProviderScope(
                container: container,
                child: MaterialApp(
                  theme: gfThemeData(brightness),
                  locale: Locale(language),
                  localizationsDelegates:
                      AppLocalizations.localizationsDelegates,
                  supportedLocales: AppLocalizations.supportedLocales,
                  builder: (context, child) => MediaQuery(
                    data: MediaQuery.of(context).copyWith(
                      size: Size(320, MediaQuery.sizeOf(context).height),
                      textScaler: const TextScaler.linear(2),
                    ),
                    child: Center(child: SizedBox(width: 320, child: child)),
                  ),
                  home: _Surface(status),
                ),
              ),
            );
            await tester.pumpAndSettle();
            // The dock paints a resting layer and a magnified lens layer.
            // Assert each user surface instead of counting both paint copies.
            final dock = find.byKey(
              const ValueKey<String>('gf-bottom-navigation-resting-layer'),
            );
            expect(
              find.descendant(of: dock, matching: find.text('99+')),
              findsOneWidget,
            );
            expect(
              find.descendant(of: dock, matching: find.text('12')),
              findsOneWidget,
            );
            expect(
              find.descendant(
                of: find.byType(GfConversationRow),
                matching: find.text('12'),
              ),
              findsOneWidget,
            );
            if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
              await tester.pump(const Duration(milliseconds: 300));
              await binding.takeScreenshot(
                'unread-number-$language-${brightness.name}',
              );
            }
            final beforeReads = countReads;
            await tester.tap(find.byType(Switch));
            await tester.pumpAndSettle();
            expect(
              container.read(unreadDisplayProvider).style,
              UnreadBadgeStyle.dot,
            );
            expect(find.text('99+'), findsNothing);
            expect(find.text('12'), findsNothing);
            expect(
              countReads,
              beforeReads,
              reason: 'changing style does not request counts again',
            );
            expect(
              (await repository.preferences()).badgeStyle,
              UnreadBadgeStyle.dot,
            );
            final after = await repository.getUnreadStatus();
            expect(after.notifications, isTrue);
            expect(after.messages, isTrue);
            expect(after.notificationCount, 149);
            expect(after.messageCount, 12);
            expect(tester.takeException(), isNull);
            if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
              await tester.pump(const Duration(milliseconds: 300));
              await binding.takeScreenshot(
                'unread-dot-$language-${brightness.name}',
              );
            }
            await tester.pumpWidget(const SizedBox());
            await tester.pumpAndSettle();
            container.dispose();
            dio.close();
          }
        }
        expect(writes, 8);
        expect(countReads, 16);
        expect(unexpected, isEmpty);
      } finally {
        await server.close(force: true);
      }
    },
  );
}

import 'dart:async';
import 'dart:typed_data';
import 'package:image/image.dart' as img;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/account_closure_dialog.dart';
import 'package:forum_app/src/pages/settings/password_edit_page.dart';
import 'package:forum_app/src/pages/settings/profile_edit_dialog.dart';
import 'package:forum_app/src/pages/settings/session_device_label.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_behavior_test.dart' show settingsPayloadJson;

Future<void> openEditor(
  WidgetTester tester,
  Widget page, {
  double scale = 1,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: gfThemeData(Brightness.light),
      locale: const Locale('en'),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      builder: (context, child) => MediaQuery(
        data: MediaQuery.of(
          context,
        ).copyWith(textScaler: TextScaler.linear(scale)),
        child: child!,
      ),
      home: Builder(
        builder: (context) => Scaffold(
          body: TextButton(
            onPressed: () => Navigator.of(
              context,
            ).push(MaterialPageRoute<void>(builder: (_) => page)),
            child: const Text('Open'),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open'));
  await tester.pumpAndSettle();
}

void main() {
  SettingsUserPayload user() => SettingsUserPayload.fromJson(
    (settingsPayloadJson()['props'] as Map)['user'] as Map<String, dynamic>,
  );
  testWidgets(
    'profile waits for acknowledgement and prevents duplicate saves',
    (tester) async {
      final pending = Completer<void>();
      var attempts = 0;
      SettingsUserPayload? submitted;
      await openEditor(
        tester,
        ProfileEditPage(
          user: user(),
          onSave: (value) {
            attempts++;
            submitted = value;
            return pending.future;
          },
        ),
      );
      await tester.enterText(
        find.byKey(const ValueKey('profile-nickname')),
        'New name',
      );
      await tester.pump();
      await tester.tap(find.byKey(const Key('profile-save')));
      await tester.pump();
      expect(attempts, 1);
      expect(find.byType(ProfileEditPage), findsOneWidget);
      await tester.tap(find.byKey(const Key('profile-save')));
      await tester.pump();
      expect(attempts, 1);
      expect(submitted?.nickname, 'New name');
      pending.complete();
      await tester.pumpAndSettle();
      expect(find.byType(ProfileEditPage), findsNothing);
    },
  );
  testWidgets('profile cover keeps the public profile safe area geometry', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(393, 852);
    tester.view.devicePixelRatio = 1;
    tester.view.viewPadding = const FakeViewPadding(top: 62);
    tester.view.padding = const FakeViewPadding(top: 62);
    addTearDown(tester.view.reset);
    await openEditor(
      tester,
      ProfileEditPage(user: user(), onSave: (_) async {}),
    );
    final coverSize = tester.getSize(
      find.byKey(const Key('profile-cover-preview')),
    );
    expect(coverSize.width, 393);
    expect(
      coverSize.height,
      GfUserCard.coverHeightFor(coverSize.width, topInset: 62),
    );
    expect(coverSize.height, greaterThan(coverSize.width / 3));
    final cover = tester.getRect(
      find.byKey(const Key('profile-cover-preview')),
    );
    final nickname = tester.getRect(
      find.byKey(const ValueKey('profile-nickname')),
    );
    expect(nickname.top - cover.bottom, 56);
    expect(tester.takeException(), isNull);
  });
  testWidgets('profile back preserves work unless discard is confirmed', (
    tester,
  ) async {
    await openEditor(
      tester,
      ProfileEditPage(user: user(), onSave: (_) async {}),
    );
    await tester.enterText(
      find.byKey(const ValueKey('profile-nickname')),
      'Unsaved name',
    );
    await tester.tap(find.byTooltip('Cancel'));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(tester.element(find.byType(ProfileEditPage)));
    await tester.tap(find.text(l.settingsKeepEditing));
    await tester.pumpAndSettle();
    expect(find.text('Unsaved name'), findsOneWidget);
    await tester.tap(find.byTooltip('Cancel'));
    await tester.pumpAndSettle();
    await tester.tap(find.text(l.settingsDiscardChanges));
    await tester.pumpAndSettle();
    expect(find.byType(ProfileEditPage), findsNothing);
  });
  testWidgets('profile photos preview locally and discard makes no writes', (
    tester,
  ) async {
    final bytes = Uint8List.fromList(
      img.encodePng(img.Image(width: 4, height: 4)),
    );
    var writes = 0;
    await openEditor(
      tester,
      ProfileEditPage(
        user: user(),
        onSave: (_) async {
          writes++;
        },
        onPickImage: (_) async => bytes,
        onSaveAvatar: (_) async {
          writes++;
          return '/new-avatar.webp';
        },
        onSaveCover: (_) async {
          writes++;
          return '/new-cover.webp';
        },
      ),
    );
    await tester.tap(find.byKey(const Key('profile-edit-cover')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('profile-edit-avatar')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('profile-local-cover')), findsOneWidget);
    expect(find.byKey(const Key('profile-local-avatar')), findsOneWidget);
    expect(writes, 0);
    await tester.tap(find.byTooltip('Cancel'));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(tester.element(find.byType(ProfileEditPage)));
    await tester.tap(find.text(l.settingsDiscardChanges));
    await tester.pumpAndSettle();
    expect(writes, 0);
    expect(find.byType(ProfileEditPage), findsNothing);
  });

  testWidgets(
    'partial profile save retains remaining image and resumes failed step',
    (tester) async {
      final bytes = Uint8List.fromList(
        img.encodePng(img.Image(width: 4, height: 4)),
      );
      var infos = 0, covers = 0, avatars = 0;
      final original = user().copyWith(
        externalInformation: {
          ...user().externalInformation,
          'future-provider': const ExternalLinkPayload(
            link: 'https://example.org/custom',
          ),
        },
      );
      SettingsUserPayload? saved;
      await openEditor(
        tester,
        ProfileEditPage(
          user: original,
          onSave: (updated) async {
            infos++;
            saved = updated;
          },
          onPickImage: (_) async => bytes,
          onSaveCover: (_) async {
            covers++;
            return '/saved-cover.webp';
          },
          onSaveAvatar: (_) async {
            avatars++;
            if (avatars == 1) {
              throw const ApiException(
                fallbackMessage: 'Avatar temporarily unavailable',
              );
            }
            return '/saved-avatar.webp';
          },
        ),
      );
      await tester.enterText(
        find.byKey(const ValueKey('profile-nickname')),
        'Edited profile',
      );
      await tester.tap(find.byKey(const Key('profile-edit-cover')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('profile-edit-avatar')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('profile-save')));
      await tester.pumpAndSettle();
      expect((infos, covers, avatars), (1, 1, 1));
      expect(
        saved!.externalInformation['future-provider']!.link,
        'https://example.org/custom',
      );
      expect(find.byKey(const Key('profile-partial-save')), findsOneWidget);
      expect(find.byKey(const Key('profile-save-error')), findsOneWidget);
      expect(find.byKey(const Key('profile-local-avatar')), findsOneWidget);
      expect(find.text('Edited profile'), findsOneWidget);
      await tester.tap(find.byKey(const Key('profile-save')));
      await tester.pumpAndSettle();
      expect((infos, covers, avatars), (1, 1, 2));
      expect(find.byType(ProfileEditPage), findsNothing);
    },
  );

  testWidgets('removing the cover is staged until explicit save', (
    tester,
  ) async {
    final coverWrites = <Uint8List?>[];
    await openEditor(
      tester,
      ProfileEditPage(
        user: user().copyWith(
          profileCoverUrl: 'https://example.org/cover.webp',
        ),
        onSave: (_) async => fail('unchanged text should not be submitted'),
        onSaveCover: (bytes) async {
          coverWrites.add(bytes);
          return '';
        },
      ),
    );
    await tester.tap(find.byKey(const Key('profile-remove-cover')));
    await tester.pumpAndSettle();
    expect(coverWrites, isEmpty);
    expect(find.byKey(const Key('profile-remove-cover')), findsNothing);
    await tester.tap(find.byKey(const Key('profile-save')));
    await tester.pumpAndSettle();
    expect(coverWrites, [null]);
    expect(find.byType(ProfileEditPage), findsNothing);
  });

  testWidgets('invalid profile links block every save and preserve input', (
    tester,
  ) async {
    var writes = 0;
    await openEditor(
      tester,
      ProfileEditPage(
        user: user(),
        onSave: (_) async {
          writes++;
        },
      ),
    );
    final website = find.byKey(const ValueKey('profile-website'));
    await tester.ensureVisible(website);
    await tester.enterText(website, 'javascript:alert(1)');
    await tester.pump();
    await tester.tap(find.byKey(const Key('profile-save')));
    await tester.pumpAndSettle();
    expect(writes, 0);
    expect(
      tester.widget<TextFormField>(website).controller!.text,
      'javascript:alert(1)',
    );
    expect(find.byType(ProfileEditPage), findsOneWidget);
  });

  testWidgets(
    'failed profile text save retains image drafts and performs no image write',
    (tester) async {
      final bytes = Uint8List.fromList(
        img.encodePng(img.Image(width: 4, height: 4)),
      );
      var images = 0;
      await openEditor(
        tester,
        ProfileEditPage(
          user: user(),
          onSave: (_) async => throw const ApiException(
            fallbackMessage: 'Unable to save profile',
          ),
          onPickImage: (_) async => bytes,
          onSaveAvatar: (_) async {
            images++;
            return '/avatar.webp';
          },
        ),
      );
      await tester.enterText(
        find.byKey(const ValueKey('profile-nickname')),
        'Preserved name',
      );
      await tester.tap(find.byKey(const Key('profile-edit-avatar')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('profile-save')));
      await tester.pumpAndSettle();
      expect(images, 0);
      expect(find.text('Preserved name'), findsOneWidget);
      expect(find.byKey(const Key('profile-local-avatar')), findsOneWidget);
      expect(find.byKey(const Key('profile-save-error')), findsOneWidget);
    },
  );

  testWidgets('password validation and server failure retain both fields', (
    tester,
  ) async {
    var attempts = 0;
    await openEditor(
      tester,
      PasswordEditPage(
        onSave: (_, _) async {
          attempts++;
          throw StateError('offline');
        },
      ),
    );
    final fields = tester
        .widgetList<TextField>(find.byType(TextField))
        .toList();
    expect(fields.first.autofillHints, contains(AutofillHints.password));
    expect(fields.last.autofillHints, contains(AutofillHints.newPassword));
    for (final field in fields) {
      expect(field.autocorrect, isFalse);
      expect(field.enableSuggestions, isFalse);
      expect(field.obscureText, isTrue);
    }
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(attempts, 0);
    expect(find.byType(PasswordEditPage), findsOneWidget);
    await tester.enterText(
      find.byKey(const Key('settings-old-password')),
      'old password',
    );
    await tester.enterText(
      find.byKey(const Key('settings-new-password')),
      'new password',
    );
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(attempts, 1);
    expect(find.byKey(const Key('password-save-error')), findsOneWidget);
    expect(
      tester
          .widget<TextFormField>(find.byKey(const Key('settings-old-password')))
          .controller
          ?.text,
      'old password',
    );
    expect(
      tester
          .widget<TextFormField>(find.byKey(const Key('settings-new-password')))
          .controller
          ?.text,
      'new password',
    );
  });
  testWidgets('profile editor fits a narrow window with enlarged text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await openEditor(
      tester,
      ProfileEditPage(user: user(), onSave: (_) async {}),
      scale: 2,
    );
    expect(find.byType(AlertDialog), findsNothing);
    expect(tester.takeException(), isNull);
    await tester.drag(find.byType(ListView), const Offset(0, -1500));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });
  testWidgets('account closure stays explicit and fits enlarged text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await openEditor(tester, const AccountClosureDialog(), scale: 2);
    final field = tester.widget<TextField>(find.byType(TextField));
    expect(field.autofillHints, contains(AutofillHints.password));
    expect(field.obscureText, isTrue);
    expect(field.enableSuggestions, isFalse);
    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.byType(TextField));
    await tester.enterText(find.byType(TextField), ' secret ');
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();
    expect(find.byType(AccountClosureDialog), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
  testWidgets('session labels never expose an unrecognized user agent', (
    tester,
  ) async {
    await openEditor(tester, const Scaffold());
    final l = AppLocalizations.of(tester.element(find.byType(Scaffold).last));
    expect(
      sessionDeviceLabel(
        'Mozilla/5.0 (iPhone; CPU iPhone OS 26) Safari/605.1',
        l,
        isCurrent: false,
      ),
      'iPhone · Safari',
    );
    expect(
      sessionDeviceLabel(
        'Dart/3.12 some long private diagnostic',
        l,
        isCurrent: true,
      ),
      l.settingsDeviceCurrent,
    );
    expect(
      sessionDeviceLabel('unknown client', l, isCurrent: false),
      l.settingsDeviceUnknown,
    );
  });
}

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/images/image_save.dart';
// ignore: depend_on_referenced_packages
import 'package:share_plus_platform_interface/share_plus_platform_interface.dart';

class _ShareSpy extends SharePlatform {
  ShareResultStatus status = ShareResultStatus.success;
  Object? error;
  final delivered = <({String name, String? mimeType, List<int> bytes})>[];

  @override
  Future<ShareResult> share(ShareParams params) async {
    if (error != null) throw error!;
    final file = params.files!.single;
    delivered.add((
      name: params.fileNameOverrides!.single,
      mimeType: file.mimeType,
      bytes: await file.readAsBytes(),
    ));
    return ShareResult('test-target', status);
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('save and share helpers deliver generated PNG bytes', (
    tester,
  ) async {
    final previousPlatform = SharePlatform.instance;
    final spy = _ShareSpy();
    SharePlatform.instance = spy;
    debugDefaultTargetPlatformOverride = TargetPlatform.macOS;
    try {
      late BuildContext context;
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (value) {
              context = value;
              return const Scaffold(body: SizedBox());
            },
          ),
        ),
      );
      final bytes = [137, 80, 78, 71];
      await saveImageBytes(context, Uint8List.fromList(bytes), 'review.png');
      await shareImageBytes(context, Uint8List.fromList(bytes), 'review.png');
      expect(spy.delivered, hasLength(2));
      for (final file in spy.delivered) {
        expect(file.name, 'review.png');
        expect(file.mimeType, 'image/png');
        expect(file.bytes, bytes);
      }
      spy.status = ShareResultStatus.dismissed;
      await saveImageBytes(context, Uint8List.fromList(bytes), 'review.png');
      spy.error = StateError('share unavailable');
      await shareImageBytes(context, Uint8List.fromList(bytes), 'review.png');
      await tester.pump(const Duration(milliseconds: 300));
      expect(find.text(AppLocalizations.of(context).shareImageFailed), findsOneWidget);
    } finally {
      SharePlatform.instance = previousPlatform;
      debugDefaultTargetPlatformOverride = null;
    }
  });

  testWidgets('gallery permission denial has a dedicated localized message', (
    tester,
  ) async {
    const channel = MethodChannel('gal');
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
      call,
    ) async {
      return switch (call.method) {
        'hasAccess' || 'requestAccess' => false,
        _ => null,
      };
    });
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    try {
      late BuildContext context;
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (value) {
              context = value;
              return const Scaffold(body: SizedBox());
            },
          ),
        ),
      );
      await saveImageBytes(context, Uint8List(4), 'review.png');
      await tester.pump(const Duration(milliseconds: 300));
      expect(
        find.text(AppLocalizations.of(context).imagePermissionDenied),
        findsOneWidget,
      );
    } finally {
      debugDefaultTargetPlatformOverride = null;
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        channel,
        null,
      );
    }
  });
}

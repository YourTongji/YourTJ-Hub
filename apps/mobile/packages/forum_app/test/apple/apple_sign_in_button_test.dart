import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/apple/apple_sign_in_button.dart';
import 'package:ui_kit/ui_kit.dart';

/// The native button cannot restyle itself after creation, so every visual
/// input has to travel in the platform view creation params.
Map<Object?, Object?> creationParamsOf(MethodCall call) {
  final args = call.arguments as Map<Object?, Object?>;
  final params = args['params']! as Uint8List;
  return const StandardMessageCodec().decodeMessage(
        ByteData.sublistView(params),
      )!
      as Map<Object?, Object?>;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const platformViews = MethodChannel('flutter/platform_views');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  late List<Map<Object?, Object?>> creations;

  setUp(() {
    creations = [];
    messenger.setMockMethodCallHandler(platformViews, (call) async {
      if (call.method == 'create') creations.add(creationParamsOf(call));
      return null;
    });
  });
  tearDown(() => messenger.setMockMethodCallHandler(platformViews, null));

  Widget host(Brightness brightness, List<Widget> children) => MaterialApp(
    theme: gfThemeData(brightness),
    home: Scaffold(body: Column(children: children)),
  );

  Widget siblingProvider() => OutlinedButton.icon(
    onPressed: () {},
    icon: const Icon(Icons.login),
    label: const Text('使用 Google 登录'),
  );

  testWidgets('native button receives the theme and the sibling pill radius', (
    tester,
  ) async {
    await tester.pumpWidget(
      host(Brightness.dark, [
        AppleSignInButton(onPressed: () {}),
        siblingProvider(),
      ]),
    );
    await tester.pump();

    expect(creations, hasLength(1));
    expect(creations.single['dark'], isTrue);
    expect(creations.single['height'], AppleSignInButton.height);
    expect(creations.single['radius'], AppleSignInButton.radius);
    // The sibling login rows are stadium pills occupying the same row height,
    // so the Apple button draws the same pill (Apple allows cornerRadius only).
    final sibling = tester.getSize(find.byType(OutlinedButton));
    expect(sibling.height, AppleSignInButton.height);
    expect(AppleSignInButton.radius, sibling.height / 2);
  });

  testWidgets('switching to the light theme recreates the platform view', (
    tester,
  ) async {
    await tester.pumpWidget(
      host(Brightness.dark, [AppleSignInButton(onPressed: () {})]),
    );
    await tester.pump();
    expect(creations.single['dark'], isTrue);

    await tester.pumpWidget(
      host(Brightness.light, [AppleSignInButton(onPressed: () {})]),
    );
    await tester.pumpAndSettle();

    expect(creations, hasLength(2));
    expect(creations.last['dark'], isFalse);
    expect(creations.last['height'], AppleSignInButton.height);
    expect(creations.last['radius'], AppleSignInButton.radius);
  });

  testWidgets('a disabled button dims and swallows taps', (tester) async {
    await tester.pumpWidget(
      host(Brightness.light, [const AppleSignInButton(onPressed: null)]),
    );
    await tester.pump();

    Finder inside(Type type) => find.descendant(
      of: find.byType(AppleSignInButton),
      matching: find.byType(type),
    );
    expect(
      tester.widget<AbsorbPointer>(inside(AbsorbPointer)).absorbing,
      isTrue,
    );
    expect(tester.widget<Opacity>(inside(Opacity)).opacity, 0.5);
  });
}

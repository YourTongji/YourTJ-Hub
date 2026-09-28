import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Apple's system button supplies the official logo, localization and contrast.
/// Apple allows adjusting only the frame and the corner radius of the control,
/// so the login rows' pill silhouette is mirrored through [radius].
///
/// The button's language follows the app bundle's declared localizations
/// (`ios/Runner/Info.plist` `CFBundleLocalizations`), not Flutter's locale;
/// the two only diverge when the in-app language differs from the device or
/// per-app language. See docs/product/mobile-experience.md.
class AppleSignInButton extends StatefulWidget {
  const AppleSignInButton({super.key, required this.onPressed});

  /// Height of the login row, shared with the sibling `OutlinedButton.icon`
  /// providers.
  static const double height = 48;

  /// Pill radius matching the siblings' `StadiumBorder` silhouette.
  static const double radius = height / 2;

  final VoidCallback? onPressed;

  @override
  State<AppleSignInButton> createState() => _AppleSignInButtonState();
}

class _AppleSignInButtonState extends State<AppleSignInButton> {
  MethodChannel? _channel;
  @override
  void dispose() {
    _channel?.setMethodCallHandler(null);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return AbsorbPointer(
      absorbing: widget.onPressed == null,
      child: Opacity(
        opacity: widget.onPressed == null ? 0.5 : 1,
        child: SizedBox(
          height: AppleSignInButton.height,
          child: UiKitView(
            // Creation params are read once by the native view, so a change
            // (here: the theme) has to recreate the platform view.
            key: ValueKey((
              dark: dark,
              height: AppleSignInButton.height,
              radius: AppleSignInButton.radius,
            )),
            viewType: 'yourtj/apple-sign-in-button',
            creationParams: {
              'dark': dark,
              // The engine calls the platform view factory with a zero frame
              // and resizes the view later, so the native side caps the radius
              // against this row height instead.
              'height': AppleSignInButton.height,
              'radius': AppleSignInButton.radius,
            },
            creationParamsCodec: const StandardMessageCodec(),
            onPlatformViewCreated: (id) {
              _channel?.setMethodCallHandler(null);
              _channel = MethodChannel('yourtj/apple-button/$id')
                ..setMethodCallHandler((call) async {
                  if (mounted && call.method == 'pressed') {
                    widget.onPressed?.call();
                  }
                });
            },
          ),
        ),
      ),
    );
  }
}

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Apple's system button supplies the official logo, localization and contrast.
class AppleSignInButton extends StatefulWidget {
  const AppleSignInButton({super.key, required this.onPressed});
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
  Widget build(BuildContext context) => AbsorbPointer(
    absorbing: widget.onPressed == null,
    child: Opacity(
      opacity: widget.onPressed == null ? 0.5 : 1,
      child: SizedBox(
        height: 48,
        child: UiKitView(
          key: ValueKey(Theme.of(context).brightness),
          viewType: 'yourtj/apple-sign-in-button',
          creationParams: {
            'dark': Theme.of(context).brightness == Brightness.dark,
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

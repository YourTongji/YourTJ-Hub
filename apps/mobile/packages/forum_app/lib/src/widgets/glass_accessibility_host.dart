import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';

/// Flutter exposes reduced motion/contrast, but not iOS Reduce Transparency.
/// Keep this preference live even while a menu or sheet is already presented.
class GlassAccessibilityHost extends StatefulWidget {
  const GlassAccessibilityHost({super.key, required this.child});
  final Widget child;

  @override
  State<GlassAccessibilityHost> createState() => _GlassAccessibilityHostState();
}

class _GlassAccessibilityHostState extends State<GlassAccessibilityHost>
    with WidgetsBindingObserver {
  static const _channel = MethodChannel('yourtj/accessibility');
  bool _reduceTransparency = true;
  int _generation = 0;
  bool get _native => !kIsWeb && defaultTargetPlatform == TargetPlatform.iOS;

  @override
  void initState() {
    super.initState();
    if (_native) {
      WidgetsBinding.instance.addObserver(this);
      _channel.setMethodCallHandler((call) async {
        if (call.method == 'reduceTransparencyChanged' &&
            call.arguments is bool) {
          _generation++;
          if (mounted) {
            setState(() => _reduceTransparency = call.arguments as bool);
          }
        }
      });
      unawaited(_refresh());
    } else {
      _reduceTransparency = false;
    }
  }

  Future<void> _refresh() async {
    final generation = ++_generation;
    try {
      final value = await _channel.invokeMethod<bool>('reduceTransparency');
      if (mounted && generation == _generation) {
        setState(() => _reduceTransparency = value ?? true);
      }
    } on PlatformException {
      // Retain the last known preference on a transient channel failure.
    } on MissingPluginException {
      // Older hosts and tests remain readable until the bridge is available.
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) unawaited(_refresh());
  }

  @override
  void dispose() {
    if (_native) {
      WidgetsBinding.instance.removeObserver(this);
      _channel.setMethodCallHandler(null);
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => GfGlassSettings(
    reduceTransparency: _reduceTransparency,
    child: widget.child,
  );
}

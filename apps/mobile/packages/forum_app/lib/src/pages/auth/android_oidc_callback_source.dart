import 'package:flutter/services.dart';

import 'android_oidc_coordinator.dart';

/// Callback source backed by the MainActivity MethodChannel/EventChannel
/// bridge. Android filters the intent shape first; the coordinator and auth
/// controller still perform the authoritative state and redirect checks.
class MainActivityOidcCallbackSource implements OidcCallbackSource {
  MainActivityOidcCallbackSource({
    MethodChannel? methodChannel,
    EventChannel? eventChannel,
  }) : _methodChannel = methodChannel ?? const MethodChannel('yourtj/oidc'),
       _eventChannel =
           eventChannel ?? const EventChannel('yourtj/oidc/callbacks');

  final MethodChannel _methodChannel;
  final EventChannel _eventChannel;

  /// HarmonyOS opens the browser without logging the authorization URI.
  Future<bool> openAuthorization(Uri uri) async {
    if (uri.scheme != 'https' || uri.host.isEmpty || uri.userInfo.isNotEmpty) {
      throw ArgumentError('Authorization requires an HTTPS origin');
    }
    return await _methodChannel.invokeMethod<bool>('openAuthorization', {
          'url': uri.toString(),
        }) ??
        false;
  }

  @override
  Stream<Uri> get callbacks => _eventChannel
      .receiveBroadcastStream()
      .map((event) => event is String ? Uri.tryParse(event) : null)
      .where((uri) => uri != null)
      .cast<Uri>();

  @override
  Future<Uri?> getInitialCallback() async {
    final String? raw = await _methodChannel.invokeMethod<String>(
      'getInitialCallback',
    );
    if (raw == null || raw.isEmpty) return null;
    return Uri.tryParse(raw);
  }
}

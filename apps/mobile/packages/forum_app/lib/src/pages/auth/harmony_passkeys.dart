import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:passkeys/exceptions.dart';

/// HarmonyOS uses the system authenticator; signed client data stays opaque.
abstract final class HarmonyPasskeys {
  static const _channel = MethodChannel('yourtj/passkeys');
  static bool get isPlatform => !kIsWeb && defaultTargetPlatform.name == 'ohos';

  static Future<bool> isAvailable() async {
    try {
      return await _channel.invokeMethod<bool>('isAvailable') ?? false;
    } on MissingPluginException {
      return false;
    } on PlatformException catch (error) {
      if (error.code == 'deviceNotSupported') return false;
      rethrow;
    }
  }

  static Future<Map<String, dynamic>> authenticate(
    Map<String, dynamic> options,
  ) => _request('authenticate', options);

  static Future<Map<String, dynamic>> register(Map<String, dynamic> options) =>
      _request('register', options);

  static Future<Map<String, dynamic>> _request(
    String method,
    Map<String, dynamic> options,
  ) async {
    String? encoded;
    try {
      encoded = await _channel.invokeMethod<String>(
        method,
        jsonEncode(options),
      );
    } on PlatformException catch (error) {
      if (error.code == 'cancelled') throw PasskeyAuthCancelledException();
      if (error.code == 'deviceNotSupported') {
        throw DeviceNotSupportedException();
      }
      rethrow;
    }
    if (encoded == null || encoded.length > 65536) {
      throw const FormatException('Invalid authenticator response');
    }
    final decoded = jsonDecode(encoded);
    if (decoded is! Map<String, dynamic> ||
        decoded['type'] != 'public-key' ||
        decoded['response'] is! Map<String, dynamic>) {
      throw const FormatException('Invalid authenticator response');
    }
    final response = Map<String, dynamic>.from(decoded['response'] as Map);
    response['clientDataJSON'] = _base64Url(response.remove('clientDataJson'));
    for (final field
        in method == 'register'
            ? ['attestationObject']
            : ['authenticatorData', 'signature']) {
      response[field] = _base64Url(response[field]);
    }
    if (response['userHandle'] == '') response['userHandle'] = null;
    if (response['userHandle'] != null) {
      response['userHandle'] = _base64Url(response['userHandle']);
    }
    final extensions = decoded['clientExtensionResults'];
    if (extensions != null && extensions != 'NULL' && extensions is! Map) {
      throw const FormatException('Invalid authenticator extensions');
    }
    return {
      ...decoded,
      'id': _base64Url(decoded['id']),
      'rawId': _base64Url(decoded['rawId']),
      'response': response,
      'clientExtensionResults': extensions is Map
          ? extensions
          : <String, dynamic>{},
    };
  }

  static String _base64Url(Object? value) {
    if (value is! String || value.isEmpty) {
      throw const FormatException('Invalid authenticator binary field');
    }
    return base64Url
        .encode(base64.decode(base64.normalize(value)))
        .replaceAll('=', '');
  }
}

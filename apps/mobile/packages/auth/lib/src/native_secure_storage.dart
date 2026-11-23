import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Retains the installed storage format on existing platforms. HarmonyOS uses
/// the app's Asset Store backed encryption bridge, with no plaintext fallback.
class NativeSecureStorage extends FlutterSecureStorage {
  const NativeSecureStorage({
    super.iOptions,
    super.aOptions,
    super.lOptions,
    super.wOptions,
    super.webOptions,
    super.mOptions,
  });

  static const channel = MethodChannel('yourtj/secure_storage');
  @visibleForTesting
  bool get isHarmonyPlatform => !kIsWeb && Platform.operatingSystem == 'ohos';

  @override
  Future<String?> read({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) => isHarmonyPlatform
      ? channel.invokeMethod<String>('read', {'key': key})
      : super.read(
          key: key,
          iOptions: iOptions,
          aOptions: aOptions,
          lOptions: lOptions,
          webOptions: webOptions,
          mOptions: mOptions,
          wOptions: wOptions,
        );

  @override
  Future<void> write({
    required String key,
    required String? value,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) => isHarmonyPlatform
      ? channel.invokeMethod<void>(value == null ? 'delete' : 'write', {
          'key': key,
          'value': ?value,
        })
      : super.write(
          key: key,
          value: value,
          iOptions: iOptions,
          aOptions: aOptions,
          lOptions: lOptions,
          webOptions: webOptions,
          mOptions: mOptions,
          wOptions: wOptions,
        );

  @override
  Future<void> delete({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) => isHarmonyPlatform
      ? channel.invokeMethod<void>('delete', {'key': key})
      : super.delete(
          key: key,
          iOptions: iOptions,
          aOptions: aOptions,
          lOptions: lOptions,
          webOptions: webOptions,
          mOptions: mOptions,
          wOptions: wOptions,
        );

  @override
  Future<bool> containsKey({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) async => isHarmonyPlatform
      ? await channel.invokeMethod<bool>('containsKey', {'key': key}) ?? false
      : super.containsKey(
          key: key,
          iOptions: iOptions,
          aOptions: aOptions,
          lOptions: lOptions,
          webOptions: webOptions,
          mOptions: mOptions,
          wOptions: wOptions,
        );

  @override
  Future<Map<String, String>> readAll({
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) async => isHarmonyPlatform
      ? await channel.invokeMapMethod<String, String>('readAll') ?? {}
      : super.readAll(
          iOptions: iOptions,
          aOptions: aOptions,
          lOptions: lOptions,
          webOptions: webOptions,
          mOptions: mOptions,
          wOptions: wOptions,
        );

  @override
  Future<void> deleteAll({
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) => isHarmonyPlatform
      ? channel.invokeMethod<void>('deleteAll')
      : super.deleteAll(
          iOptions: iOptions,
          aOptions: aOptions,
          lOptions: lOptions,
          webOptions: webOptions,
          mOptions: mOptions,
          wOptions: wOptions,
        );
}

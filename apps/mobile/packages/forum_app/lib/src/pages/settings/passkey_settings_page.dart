import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:passkeys/authenticator.dart';
import 'package:passkeys/types.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../auth/passkey_error_message.dart';

class PasskeySettingsPage extends ConsumerStatefulWidget {
  const PasskeySettingsPage({super.key});

  @override
  ConsumerState<PasskeySettingsPage> createState() =>
      _PasskeySettingsPageState();
}

class _PasskeySettingsPageState extends ConsumerState<PasskeySettingsPage> {
  bool _loading = true;
  bool _enabled = false;
  bool _totpEnabled = false;
  bool _busy = false;
  String? _error;
  List<PasskeyCredentialSummary> _credentials = const [];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final auth = ref.read(authRepositoryProvider);
      final status = await auth.getPasskeyStatus();
      if (!mounted) return;
      _enabled = status.enabled;
      if (!status.enabled) {
        setState(() => _loading = false);
        return;
      }
      final totp = await ref.read(userRepositoryProvider).getTotpStatus();
      final credentials = await auth.listPasskeyCredentials();
      if (!mounted) return;
      setState(() {
        _totpEnabled = totp.enabled;
        _credentials = credentials;
        _loading = false;
      });
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = resolvePasskeyErrorMessage(
            AppLocalizations.of(context),
            error,
            fallback: AppLocalizations.of(context).commonLoadFailed,
          );
          _loading = false;
        });
      }
    }
  }

  Future<({String password, String totpCode, String name})?> _prompt({
    required String title,
    required bool includeName,
    String initialName = '',
  }) async {
    final l10n = AppLocalizations.of(context);
    final password = TextEditingController();
    final code = TextEditingController();
    final name = TextEditingController(text: initialName);
    try {
      return await showDialog<({String password, String totpCode, String name})>(
        context: context,
        builder: (context) => AlertDialog(
          title: GfText(title),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (includeName) ...[
                  TextField(
                    controller: name,
                    maxLength: 80,
                    textCapitalization: TextCapitalization.words,
                    decoration: InputDecoration(labelText: l10n.settingsPasskeyName),
                  ),
                  const SizedBox(height: 8),
                ],
                TextField(
                  controller: password,
                  obscureText: true,
                  autofillHints: const [AutofillHints.password],
                  decoration: InputDecoration(labelText: l10n.settingsPasskeyPassword),
                ),
                if (_totpEnabled) ...[
                  const SizedBox(height: 8),
                  TextField(
                    controller: code,
                    keyboardType: TextInputType.number,
                    autofillHints: const [AutofillHints.oneTimeCode],
                    decoration: InputDecoration(labelText: l10n.settingsTotpCode),
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: GfText(l10n.commonCancel),
            ),
            FilledButton(
              onPressed: () {
                if (password.text.isEmpty ||
                    (_totpEnabled && code.text.trim().isEmpty) ||
                    (includeName && name.text.trim().isEmpty)) {
                  return;
                }
                Navigator.pop(
                  context,
                  (
                    password: password.text,
                    totpCode: code.text.trim(),
                    name: name.text.trim(),
                  ),
                );
              },
              child: GfText(l10n.settingsTotpNext),
            ),
          ],
        ),
      );
    } finally {
      password.dispose();
      code.dispose();
      name.dispose();
    }
  }

  Future<String> _proof(String password, String code) async {
    final auth = ref.read(authRepositoryProvider);
    final key = await auth.getLoginPublicKey();
    final encrypted = RsaEncryptor().encryptPassword(
      publicKeyPem: key.publicKey,
      password: password,
      serverTs: key.serverTs,
    );
    final result = await auth.reauthenticatePasskeyManagement(
      encryptedPassword: encrypted,
      totpCode: code,
    );
    return result.proof;
  }

  String _paddedBase64Url(String value) =>
      value + ('=' * ((4 - value.length % 4) % 4));

  RegisterRequestType _registrationRequest(Map<String, dynamic> options) {
    final rp = Map<String, dynamic>.from(options['rp'] as Map);
    final user = Map<String, dynamic>.from(options['user'] as Map);
    final rawSelection = options['authenticatorSelection'];
    return RegisterRequestType(
      challenge: options['challenge'] as String,
      relyingParty: RelyingPartyType.fromJson(rp),
      user: UserType.fromJson(<String, dynamic>{
        ...user,
        'id': _paddedBase64Url(user['id'] as String),
      }),
      excludeCredentials: (options['excludeCredentials'] as List<dynamic>? ??
              const <dynamic>[])
          .map((value) => CredentialType.fromJson(
                Map<String, dynamic>.from(value as Map),
              ))
          .toList(growable: false),
      authSelectionType: rawSelection is Map
          ? AuthenticatorSelectionType.fromJson(
              Map<String, dynamic>.from(rawSelection),
            )
          : null,
      pubKeyCredParams: (options['pubKeyCredParams'] as List<dynamic>? ??
              const <dynamic>[])
          .map((value) => PubKeyCredParamType.fromJson(
                Map<String, dynamic>.from(value as Map),
              ))
          .toList(growable: false),
      timeout: (options['timeout'] as num?)?.toInt(),
      attestation: options['attestation'] as String?,
    );
  }

  Future<void> _addPasskey() async {
    if (_busy) return;
    final l10n = AppLocalizations.of(context);
    final input = await _prompt(
      title: l10n.settingsPasskeyAdd,
      includeName: true,
    );
    if (input == null || !mounted) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final authenticator = PasskeyAuthenticator();
      final availability = authenticator.getAvailability();
      final canAuthenticate = kIsWeb
          ? (await availability.web()).hasPasskeySupport
          : switch (defaultTargetPlatform) {
              TargetPlatform.android =>
                (await availability.android()).hasPasskeySupport,
              TargetPlatform.iOS =>
                (await availability.iOS()).hasPasskeySupport,
              _ => false,
            };
      if (!canAuthenticate) {
        throw StateError(l10n.authPasskeyUnavailable);
      }
      final auth = ref.read(authRepositoryProvider);
      final proof = await _proof(input.password, input.totpCode);
      final begin = await auth.beginPasskeyRegistration(proof: proof);
      final credential = await authenticator.register(
        _registrationRequest(begin.publicKey),
      );
      await auth.finishPasskeyRegistration(
        ceremonyId: begin.ceremonyId,
        name: input.name,
        credential: credential.toJson(),
      );
      await _load();
      if (mounted) showGfToast(context, l10n.settingsPasskeyAdded);
    } catch (error) {
      if (mounted) {
        setState(() => _error = resolvePasskeyErrorMessage(
              l10n,
              error,
              fallback: l10n.commonLoadFailed,
            ));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _renamePasskey(PasskeyCredentialSummary credential) async {
    final l10n = AppLocalizations.of(context);
    final input = await _prompt(
      title: l10n.settingsPasskeyRename,
      includeName: true,
      initialName: credential.name,
    );
    if (input == null || !mounted) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final proof = await _proof(input.password, input.totpCode);
      await ref.read(authRepositoryProvider).renamePasskey(
            proof: proof,
            credentialId: credential.id,
            name: input.name,
          );
      await _load();
      if (mounted) showGfToast(context, l10n.settingsInfoSaved);
    } catch (error) {
      if (mounted) {
        setState(() => _error = resolvePasskeyErrorMessage(
              l10n,
              error,
              fallback: l10n.commonLoadFailed,
            ));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _revokePasskey(PasskeyCredentialSummary credential) async {
    final l10n = AppLocalizations.of(context);
    final input = await _prompt(
      title: l10n.settingsPasskeyRevoke,
      includeName: false,
    );
    if (input == null || !mounted) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final proof = await _proof(input.password, input.totpCode);
      await ref.read(authRepositoryProvider).revokePasskey(
            proof: proof,
            credentialId: credential.id,
          );
      await _load();
      if (mounted) showGfToast(context, l10n.settingsPasskeyRevoked);
    } catch (error) {
      if (mounted) {
        setState(() => _error = resolvePasskeyErrorMessage(
              l10n,
              error,
              fallback: l10n.commonLoadFailed,
            ));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: GfText(l10n.settingsPasskeyTitle)),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : !_enabled
              ? Center(child: GfText(l10n.settingsPasskeyUnavailable))
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView(
                    padding: const EdgeInsets.all(16),
                    children: [
                      GfText(l10n.settingsPasskeyDescription),
                      const SizedBox(height: 4),
                      GfText(
                        l10n.settingsPasskeyRevokedHint,
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                      if (_totpEnabled) ...[
                        const SizedBox(height: 8),
                        GfText(l10n.settingsPasskeyReauthHint),
                      ],
                      if (_error != null) ...[
                        const SizedBox(height: 12),
                        GfStatusMessage(message: _error!),
                      ],
                      const SizedBox(height: 12),
                      if (_credentials.isEmpty)
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 24),
                          child: GfEmpty(message: l10n.settingsPasskeyEmpty),
                        ),
                      for (final credential in _credentials)
                        Card(
                          child: ListTile(
                            leading: const GfSymbol('key-round', size: 20),
                            title: GfText(credential.name),
                            subtitle: GfText(
                              '${l10n.settingsPasskeyAddedAt}: ${MaterialLocalizations.of(context).formatMediumDate(credential.createdAt.toLocal())}'
                              '\n${l10n.settingsPasskeyLastUsed}: ${credential.lastUsedAt == null ? l10n.settingsPasskeyNeverUsed : MaterialLocalizations.of(context).formatMediumDate(credential.lastUsedAt!.toLocal())}'
                              '\n${credential.backupEligible ? l10n.settingsPasskeySyncCapable : l10n.settingsPasskeyDeviceBound}'
                              '${credential.backupEligible && credential.backupState ? ' · ${l10n.settingsPasskeyBackedUp}' : ''}',
                            ),
                            trailing: PopupMenuButton<String>(
                              enabled: !_busy,
                              onSelected: (value) => value == 'rename'
                                  ? _renamePasskey(credential)
                                  : _revokePasskey(credential),
                              itemBuilder: (_) => [
                                PopupMenuItem(
                                  value: 'rename',
                                  child: GfText(l10n.settingsPasskeyRename),
                                ),
                                PopupMenuItem(
                                  value: 'revoke',
                                  child: GfText(l10n.settingsPasskeyRevoke),
                                ),
                              ],
                            ),
                          ),
                        ),
                      const SizedBox(height: 12),
                      FilledButton.icon(
                        onPressed: _busy ? null : _addPasskey,
                        icon: _busy
                            ? const SizedBox.square(
                                dimension: 18,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              )
                            : const Icon(Icons.add),
                        label: GfText(l10n.settingsPasskeyAdd),
                      ),
                    ],
                  ),
                ),
    );
  }
}

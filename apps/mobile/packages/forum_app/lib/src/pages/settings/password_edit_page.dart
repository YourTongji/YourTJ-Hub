import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../server_messages.dart';
import '../../widgets/confirm_discard_edit.dart';

class PasswordEditPage extends StatefulWidget {
  const PasswordEditPage({super.key, required this.onSave});
  final Future<void> Function(String oldPassword, String newPassword) onSave;
  @override
  State<PasswordEditPage> createState() => _PasswordEditPageState();
}

class _PasswordEditPageState extends State<PasswordEditPage> {
  final _form = GlobalKey<FormState>();
  final _old = TextEditingController();
  final _next = TextEditingController();
  bool _saving = false, _allowPop = false, _closing = false;
  String? _error;
  @override
  void initState() {
    super.initState();
    _old.addListener(_edited);
    _next.addListener(_edited);
  }

  void _edited() => setState(() {});

  @override
  void dispose() {
    _old.dispose();
    _next.dispose();
    super.dispose();
  }

  void _pop(bool saved) {
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) Navigator.pop(context, saved);
    });
  }

  Future<void> _close() async {
    if (_saving || _closing) return;
    _closing = true;
    final leave =
        (_old.text.isEmpty && _next.text.isEmpty) ||
        await confirmDiscardEdit(context);
    _closing = false;
    if (mounted && leave) _pop(false);
  }

  Future<void> _save() async {
    if (_saving || !_form.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.onSave(_old.text, _next.text);
      if (mounted) _pop(true);
    } catch (error) {
      if (mounted) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    return PopScope(
      canPop:
          _allowPop || (!_saving && _old.text.isEmpty && _next.text.isEmpty),
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: Scaffold(
        appBar: GfAppBar(
          title: Text(l.settingsChangePassword),
          leading: IconButton(
            tooltip: l.commonBack,
            onPressed: _saving ? null : _close,
            icon: const GfSymbol('chevron-left'),
          ),
        ),
        body: SafeArea(
          top: false,
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: Form(
                key: _form,
                child: AutofillGroup(
                  child: ListView(
                    padding: const EdgeInsets.all(20),
                    children: [
                      TextFormField(
                        key: const Key('settings-old-password'),
                        controller: _old,
                        enabled: !_saving,
                        obscureText: true,
                        autocorrect: false,
                        enableSuggestions: false,
                        autofillHints: const [AutofillHints.password],
                        textInputAction: TextInputAction.next,
                        decoration: InputDecoration(
                          labelText: l.settingsCurrentPassword,
                        ),
                        validator: (value) => value == null || value.isEmpty
                            ? l.settingsFillComplete
                            : null,
                      ),
                      const SizedBox(height: 20),
                      TextFormField(
                        key: const Key('settings-new-password'),
                        controller: _next,
                        enabled: !_saving,
                        obscureText: true,
                        autocorrect: false,
                        enableSuggestions: false,
                        autofillHints: const [AutofillHints.newPassword],
                        textInputAction: TextInputAction.done,
                        onFieldSubmitted: (_) => _save(),
                        decoration: InputDecoration(
                          labelText: l.authNewPassword,
                        ),
                        validator: (value) => value == null || value.isEmpty
                            ? l.settingsFillComplete
                            : null,
                      ),
                      if (_error != null)
                        Padding(
                          padding: const EdgeInsets.only(top: 16),
                          child: Semantics(
                            liveRegion: true,
                            child: Text(
                              _error!,
                              key: const Key('password-save-error'),
                              style: TextStyle(
                                color: GfTheme.colorsOf(context).error,
                              ),
                            ),
                          ),
                        ),
                      const SizedBox(height: 24),
                      GfButton(
                        label: l.commonSave,
                        loading: _saving,
                        onPressed: _save,
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

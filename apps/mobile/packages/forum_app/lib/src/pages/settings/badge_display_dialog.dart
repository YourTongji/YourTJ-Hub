import 'package:core/core.dart';
import 'package:flutter/material.dart';
import '../../../l10n/app_localizations.dart';
import '../../server_messages.dart';

class BadgeDisplayDialog extends StatefulWidget {
  const BadgeDisplayDialog({
    super.key,
    required this.badges,
    required this.selected,
    required this.onSave,
  });
  final List<UserBadgePayload> badges;
  final List<UserBadgePayload> selected;
  final Future<void> Function(List<String>) onSave;
  @override
  State<BadgeDisplayDialog> createState() => _BadgeDisplayDialogState();
}

class _BadgeDisplayDialogState extends State<BadgeDisplayDialog> {
  late final List<String> _codes = widget.selected.map((b) => b.code).toList();
  bool _saving = false;
  String? _error;
  void _move(int index, int delta) {
    final next = index + delta;
    if (_saving || next < 0 || next >= _codes.length) return;
    setState(() {
      final code = _codes.removeAt(index);
      _codes.insert(next, code);
    });
  }

  Future<void> _save() async {
    if (_saving) return;
    final l10n = AppLocalizations.of(context);
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.onSave(List.of(_codes));
      if (mounted) Navigator.pop(context, true);
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = resolveErrorMessage(l10n, error);
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return PopScope(
      canPop: !_saving,
      child: AlertDialog(
        title: Text(l10n.badgeDisplayTitle),
        scrollable: true,
        content: SizedBox(
          width: 400,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(l10n.badgeDisplayHint),
              for (final badge in widget.badges)
                CheckboxListTile(
                  key: Key('badge-select-${badge.code}'),
                  contentPadding: EdgeInsets.zero,
                  title: Text(badge.name),
                  value: _codes.contains(badge.code),
                  onChanged:
                      _saving ||
                          (!_codes.contains(badge.code) && _codes.length >= 5)
                      ? null
                      : (checked) => setState(() {
                          if (checked == true) {
                            _codes.add(badge.code);
                          } else {
                            _codes.remove(badge.code);
                          }
                        }),
                ),
              for (final (index, code) in _codes.indexed)
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        '${index + 1}. ${widget.badges.firstWhere((b) => b.code == code).name}',
                      ),
                    ),
                    IconButton(
                      tooltip: l10n.badgeDisplayUp,
                      onPressed: _saving || index == 0
                          ? null
                          : () => _move(index, -1),
                      icon: const Icon(Icons.arrow_upward),
                    ),
                    IconButton(
                      tooltip: l10n.badgeDisplayDown,
                      onPressed: _saving || index == _codes.length - 1
                          ? null
                          : () => _move(index, 1),
                      icon: const Icon(Icons.arrow_downward),
                    ),
                  ],
                ),
              if (_error != null)
                Text(
                  _error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: _saving ? null : () => Navigator.pop(context),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: _saving ? null : _save,
            child: Text(l10n.commonSave),
          ),
        ],
      ),
    );
  }
}
